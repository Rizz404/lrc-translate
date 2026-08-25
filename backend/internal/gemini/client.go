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

const maxAttempts = 3

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

		backoff := time.Duration(attempt) * time.Second
		backoff += time.Duration(rand.Intn(300)) * time.Millisecond // jitter
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(backoff):
		}
	}
	return "", lastErr
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
			"Produce a natural, idiomatic translation that reads like part of "+
			"a song — not a stiff, literal, word-for-word translation. Preserve "+
			"the emotional meaning and tone, and keep it roughly as concise as "+
			"the original line. Output ONLY the translated line itself, with no "+
			"quotation marks, explanation, or extra commentary.\n\nLine: %q",
		from, to, text,
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
