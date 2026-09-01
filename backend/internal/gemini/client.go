// Package gemini is a client for the Google AI Studio (Gemini) generateContent
// API, used as a cloud fallback MT backend when internal/localllm isn't
// configured (or when the self-hosted model isn't reachable) — see
// cmd/server/main.go's provider auto-resolve priority. Unlike plain NMT
// (internal/libretranslate), this is an LLM steered by a prompt (see
// internal/llmprompt, shared with internal/localllm) — the point of using it
// here is that it can be told to translate like a song lyric
// (natural/idiomatic) instead of word-for-word, which is what
// LibreTranslate/Argos Translate can't do.
package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"lrc-translate/backend/internal/llmprompt"
)

// defaultBaseURL is the real Gemini API host. Overridden by tests (same
// package, unexported field) to point at an httptest server instead.
const defaultBaseURL = "https://generativelanguage.googleapis.com"

// Client talks to the Gemini generateContent API.
type Client struct {
	apiKey  string
	model   string
	http    *http.Client
	baseURL string
}

// New creates a Client. model may be empty to use a sane default.
func New(apiKey, model string) *Client {
	if model == "" {
		model = "gemini-2.5-flash"
	}
	return &Client{
		apiKey:  apiKey,
		model:   model,
		baseURL: defaultBaseURL,
		// LLM calls are slower than a plain NMT lookup, especially under
		// free-tier rate limiting — give it real room before giving up.
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

// maxAttempts was 3 with a ~1-3s linear backoff — nowhere near enough
// headroom for a bulk caller (e.g. handleGetAIReference translating a whole
// song's worth of lines at once) hammering the free tier's rate limit: a
// burst of concurrent requests exhausts it almost immediately, and 3 short
// retries all land inside that same still-exhausted window.
//
// A much longer backoff (6 attempts, up to ~60s total) was tried and then
// walked back after live testing showed it doesn't actually help: a line
// that's still 429ing after a full minute of waiting isn't recovering from
// a brief per-minute burst, it's hit a harder cap (most likely the free
// tier's per-day request quota) that no amount of in-request backoff can
// wait out — it only made a real failure take a minute to surface instead
// of a few seconds, for the exact same outcome. This is deliberately closer
// to the original: enough backoff to smooth over a genuine brief burst,
// short enough that a harder/longer-lived limit still fails fast so the
// caller (see ai_reference_handler.go's per-line "failed" reporting and its
// "coba lagi baris yang gagal" retry) isn't left hanging.
const maxAttempts = 4

// Translate sends one line of song lyrics through Gemini, retrying with
// backoff on 429 (rate limited) and 5xx responses.
func (c *Client) Translate(ctx context.Context, text, sourceLang, targetLang string) (string, error) {
	prompt := llmprompt.Build(text, sourceLang, targetLang)

	body, err := json.Marshal(map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]string{{"text": prompt}}},
		},
		"generationConfig": map[string]any{
			"temperature":     0.3,
			"maxOutputTokens": 300,
		},
	})
	if err != nil {
		return "", err
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		text, retryable, err := c.doRequest(ctx, body)
		if err == nil {
			return strings.Trim(text, "\"“”"), nil
		}
		lastErr = err
		if !retryable || attempt == maxAttempts {
			break
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(exponentialBackoff(attempt)):
		}
	}
	return "", lastErr
}

// TranslateBatch sends every targeted line of a song through Gemini in one
// request, letting the model use the whole song as context instead of
// guessing at each line in isolation — see llmprompt.BuildBatch. Retries
// with the same backoff as Translate, plus treats a count-mismatched
// response (see llmprompt.ParseBatch) as retryable: temperature > 0 means a
// different sample has a real shot at following the "JSON array only"
// instruction, so it's worth another attempt rather than failing the whole
// batch outright.
func (c *Client) TranslateBatch(ctx context.Context, lines []string, sourceLang, targetLang string) ([]string, error) {
	if len(lines) == 0 {
		return nil, nil
	}
	prompt := llmprompt.BuildBatch(lines, sourceLang, targetLang)

	body, err := json.Marshal(map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]string{{"text": prompt}}},
		},
		"generationConfig": map[string]any{
			"temperature":     0.3,
			"maxOutputTokens": batchMaxOutputTokens(len(lines)),
		},
	})
	if err != nil {
		return nil, err
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		text, retryable, err := c.doRequest(ctx, body)
		if err == nil {
			out, parseErr := llmprompt.ParseBatch(text, len(lines))
			if parseErr == nil {
				return out, nil
			}
			err = parseErr
			retryable = true
		}
		lastErr = err
		if !retryable || attempt == maxAttempts {
			break
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(exponentialBackoff(attempt)):
		}
	}
	return nil, lastErr
}

// batchMaxOutputTokens scales the output budget with how many lines are in
// the batch — a single line's worth (~300, see Translate's call above)
// times the count, floored so a tiny batch still gets real headroom and
// capped so one request can't demand an unreasonably large generation.
func batchMaxOutputTokens(n int) int {
	tokens := 300 * n
	if tokens < 1024 {
		return 1024
	}
	if tokens > 8192 {
		return 8192
	}
	return tokens
}

// exponentialBackoff doubles each attempt (1s, 2s, 4s for attempts 1-3, ~7s
// total across all of maxAttempts=4's retries) plus up to 300ms of jitter so
// a burst of concurrent callers retrying at the same moment doesn't just
// recreate the same burst on the next attempt. See maxAttempts' doc comment
// for why this stays short rather than growing to wait out a much longer
// (e.g. daily) quota window that backoff can't actually shorten.
func exponentialBackoff(attempt int) time.Duration {
	backoff := time.Duration(1<<uint(attempt-1)) * time.Second
	backoff += time.Duration(rand.Intn(300)) * time.Millisecond
	return backoff
}

// doRequest posts body to the generateContent endpoint and returns the
// first candidate's raw text (untrimmed of surrounding quotes — Translate
// and TranslateBatch each handle that differently).
func (c *Client) doRequest(ctx context.Context, body []byte) (text string, retryable bool, err error) {
	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent", c.baseURL, c.model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", true, fmt.Errorf("gemini: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	switch {
	case resp.StatusCode == http.StatusOK:
		var parsed struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
				FinishReason string `json:"finishReason"`
			} `json:"candidates"`
		}
		if err := json.Unmarshal(respBody, &parsed); err != nil {
			return "", false, fmt.Errorf("gemini: decode response: %w", err)
		}
		if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
			reason := "unknown"
			if len(parsed.Candidates) > 0 {
				reason = parsed.Candidates[0].FinishReason
			}
			return "", false, fmt.Errorf("gemini: no translation returned (finishReason=%s)", reason)
		}
		return strings.TrimSpace(parsed.Candidates[0].Content.Parts[0].Text), false, nil

	case resp.StatusCode == http.StatusTooManyRequests:
		return "", true, fmt.Errorf("gemini: rate limited (429)")

	case resp.StatusCode >= 500:
		return "", true, fmt.Errorf("gemini: server error (%d)", resp.StatusCode)

	default:
		return "", false, fmt.Errorf("gemini: request rejected (%d): %s", resp.StatusCode, string(respBody))
	}
}
