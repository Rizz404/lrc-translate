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
		http:    &http.Client{Timeout: 10 * time.Second},
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/translate", bytes.NewReader(body))
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", true, fmt.Errorf("libretranslate: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	switch {
	case resp.StatusCode == http.StatusOK:
		var parsed struct {
			TranslatedText string `json:"translatedText"`
		}
		if err := json.Unmarshal(respBody, &parsed); err != nil {
			return "", false, fmt.Errorf("libretranslate: decode response: %w", err)
		}
		return parsed.TranslatedText, false, nil

	case resp.StatusCode == http.StatusTooManyRequests:
		return "", true, fmt.Errorf("libretranslate: rate limited (429)")

	case resp.StatusCode >= 500:
		return "", true, fmt.Errorf("libretranslate: server error (%d)", resp.StatusCode)

	default:
		return "", false, fmt.Errorf("libretranslate: request rejected (%d): %s", resp.StatusCode, string(respBody))
	}
}
