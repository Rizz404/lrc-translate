// Package libretranslate is a client for the LibreTranslate MT API
// (https://libretranslate.com or a self-hosted instance — see
// docker-compose.yml). The same client code works against either, only the
// base URL/API key differ.
package libretranslate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"
)

// Client talks to a LibreTranslate instance.
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// New creates a Client. apiKey may be empty for instances that don't require one.
func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		// Self-hosted LibreTranslate is CPU-bound neural MT with no GPU —
		// under concurrent load (or a cold model not yet resident in a
		// given gunicorn worker) a single translate call can take well
		// over 10s. 10s was tuned for a lightly-loaded/dedicated instance;
		// on a shared host running several other containers, raise this
		// further if you still see "context deadline exceeded" errors.
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

const maxAttempts = 3

// Translate sends one line of text through the MT API, retrying with backoff
// on 429 (rate limited) and 5xx (server error) responses. Other errors
// (4xx besides 429, network failures after retries) are returned as-is so
// the caller can surface a clear message to the user.
func (c *Client) Translate(ctx context.Context, text, sourceLang, targetLang string) (string, error) {
	if sourceLang == "" {
		sourceLang = "auto"
	}

	body, err := json.Marshal(map[string]string{
		"q":       text,
		"source":  sourceLang,
		"target":  targetLang,
		"format":  "text",
		"api_key": c.apiKey,
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

func (c *Client) doTranslate(ctx context.Context, body []byte) (result string, retryable bool, err error) {
	respBody, retryable, err := c.doRequest(ctx, body)
	if err != nil {
		return "", retryable, err
	}

	var parsed struct {
		TranslatedText string `json:"translatedText"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", false, fmt.Errorf("libretranslate: decode response: %w", err)
	}
	return parsed.TranslatedText, false, nil
}

// TranslateBatch sends every targeted line through LibreTranslate in one
// request, using its native support for a "q" array (LibreTranslate
// translates each element independently — this is a plain NMT engine, not
// an LLM, so it can't use surrounding lines as context the way
// internal/gemini/internal/localllm's TranslateBatch can). This exists
// mainly so Translator's callers (see httpapi.handleTranslateTrack) don't
// need provider-specific branching: one HTTP round trip for the whole
// batch instead of one per line either way.
func (c *Client) TranslateBatch(ctx context.Context, lines []string, sourceLang, targetLang string) ([]string, error) {
	if len(lines) == 0 {
		return nil, nil
	}
	if sourceLang == "" {
		sourceLang = "auto"
	}

	body, err := json.Marshal(map[string]any{
		"q":       lines,
		"source":  sourceLang,
		"target":  targetLang,
		"format":  "text",
		"api_key": c.apiKey,
	})
	if err != nil {
		return nil, err
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		respBody, retryable, err := c.doRequest(ctx, body)
		if err == nil {
			var parsed struct {
				TranslatedText []string `json:"translatedText"`
			}
			if err := json.Unmarshal(respBody, &parsed); err != nil {
				return nil, fmt.Errorf("libretranslate: decode batch response: %w", err)
			}
			if len(parsed.TranslatedText) != len(lines) {
				return nil, fmt.Errorf("libretranslate: expected %d translated lines back, got %d", len(lines), len(parsed.TranslatedText))
			}
			return parsed.TranslatedText, nil
		}
		lastErr = err
		if !retryable || attempt == maxAttempts {
			break
		}

		backoff := time.Duration(attempt) * time.Second
		backoff += time.Duration(rand.Intn(300)) * time.Millisecond // jitter
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
	}
	return nil, lastErr
}

// doRequest posts body to the /translate endpoint and returns the raw
// response body for the caller (Translate/TranslateBatch) to decode — they
// expect translatedText as a string vs. an array respectively, depending on
// whether "q" was sent as a single string or an array.
func (c *Client) doRequest(ctx context.Context, body []byte) (respBody []byte, retryable bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/translate", bytes.NewReader(body))
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, true, fmt.Errorf("libretranslate: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ = io.ReadAll(resp.Body)

	switch {
	case resp.StatusCode == http.StatusOK:
		return respBody, false, nil

	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, true, fmt.Errorf("libretranslate: rate limited (429)")

	case resp.StatusCode >= 500:
		return nil, true, fmt.Errorf("libretranslate: server error (%d)", resp.StatusCode)

	default:
		return nil, false, fmt.Errorf("libretranslate: request rejected (%d): %s", resp.StatusCode, string(respBody))
	}
}
