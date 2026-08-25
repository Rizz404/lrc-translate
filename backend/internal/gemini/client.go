// Package gemini is a client for the Google AI Studio (Gemini) generateContent
// API, used as an alternative MT backend to LibreTranslate (see
// internal/libretranslate). Unlike LibreTranslate's plain NMT, this is an
// LLM steered by a prompt — the point of using it here is that it can be
// told to translate like a song lyric (natural/idiomatic) instead of
// word-for-word, which is what LibreTranslate/Argos Translate can't do.
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
)

// Client talks to the Gemini generateContent API.
type Client struct {
	apiKey string
	model  string
	http   *http.Client
}

// New creates a Client. model may be empty to use a sane default.
func New(apiKey, model string) *Client {
	if model == "" {
		model = "gemini-2.5-flash"
	}
	return &Client{
		apiKey: apiKey,
		model:  model,
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

// langNames maps common ISO codes to a display name for the prompt — Gemini
// understands bare codes fine too, but a name reads more naturally in an
// instruction and avoids any ambiguity (e.g. "id" vs "auto").
var langNames = map[string]string{
	"ja": "Japanese",
	"en": "English",
	"id": "Indonesian",
	"ko": "Korean",
	"zh": "Chinese",
	"es": "Spanish",
	"fr": "French",
	"de": "German",
	"pt": "Portuguese",
	"ru": "Russian",
	"ar": "Arabic",
	"hi": "Hindi",
	"th": "Thai",
	"vi": "Vietnamese",
}

func langName(code string) string {
	if name, ok := langNames[code]; ok {
		return name
	}
	return code
}

// Translate sends one line of song lyrics through Gemini, retrying with
// backoff on 429 (rate limited) and 5xx responses.
func (c *Client) Translate(ctx context.Context, text, sourceLang, targetLang string) (string, error) {
	prompt := buildPrompt(text, sourceLang, targetLang)

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
		result, retryable, err := c.doTranslate(ctx, body)
		if err == nil {
			return result, nil
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

func buildPrompt(text, sourceLang, targetLang string) string {
	var from string
	if sourceLang == "" || sourceLang == "auto" {
		from = "the source language (detect it automatically)"
	} else {
		from = langName(sourceLang)
	}
	to := langName(targetLang)

	return fmt.Sprintf(
		"You are translating one line from a song's lyrics, from %s to %s. "+
			"This line is one of many sent as separate, independent requests "+
			"for the same song, so you won't see the other lines — follow the "+
			"rules below exactly as given so every line stays consistent with "+
			"the rest of the song even without that context.\n\n"+
			"Rules:\n"+
			"1. Register: song lyrics address people informally. Use the "+
			"casual/informal form of \"I\" and \"you\" in %s, never a formal or "+
			"polite register (e.g. in Indonesian use \"aku\"/\"kamu\", never "+
			"\"saya\"/\"Anda\"; in French use \"tu\", never \"vous\") — unless "+
			"the source line is itself unmistakably formal or reverent in "+
			"tone. Never mix formal and informal address across lines.\n"+
			"2. Translate the FULL line. Every word must end up in %s. Never "+
			"leave part or all of the line untranslated, and never output the "+
			"source text unchanged — the only exception is a proper noun "+
			"(a name, a place) with no natural equivalent.\n"+
			"3. Prioritize meaning over literal wording: render what the line "+
			"actually means and how a native %s speaker would naturally say it "+
			"in a song, not a stiff word-for-word mapping of the source "+
			"grammar.\n"+
			"4. Keep it roughly as concise as the original line, so it still "+
			"reads like a lyric.\n\n"+
			"Output ONLY the translated line itself, with no quotation marks, "+
			"romanization, explanation, or extra commentary.\n\nLine: %q",
		from, to, to, to, to, text,
	)
}

func (c *Client) doTranslate(ctx context.Context, body []byte) (result string, retryable bool, err error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", c.model)
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
		translated := strings.TrimSpace(parsed.Candidates[0].Content.Parts[0].Text)
		translated = strings.Trim(translated, "\"“”")
		return translated, false, nil

	case resp.StatusCode == http.StatusTooManyRequests:
		return "", true, fmt.Errorf("gemini: rate limited (429)")

	case resp.StatusCode >= 500:
		return "", true, fmt.Errorf("gemini: server error (%d)", resp.StatusCode)

	default:
		return "", false, fmt.Errorf("gemini: request rejected (%d): %s", resp.StatusCode, string(respBody))
	}
}
