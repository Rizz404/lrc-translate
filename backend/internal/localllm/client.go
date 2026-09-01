// Package localllm is a client for a self-hosted OpenAI-compatible chat
// completions server (LM Studio, Ollama's OpenAI shim, etc.) — typically run
// on hardware the user owns and reached over a tunnel (e.g. ngrok) since it
// isn't publicly routable on its own. It's the preferred MT backend over
// internal/gemini when configured (see cmd/server/main.go's provider
// auto-resolve priority): same LLM-steered prompt (internal/llmprompt) for
// natural/lyric-like output, but free and not subject to a cloud provider's
// rate limit or daily quota.
package localllm

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

// Client talks to an OpenAI-compatible /v1/chat/completions endpoint.
type Client struct {
	baseURL string
	apiKey  string
	model   string
	http    *http.Client
}

// New creates a Client. apiKey may be empty for a server that doesn't
// require one (LM Studio's "require API key" setting is off by default —
// only pass one if that setting is turned on).
func New(baseURL, apiKey, model string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		// Local inference is CPU/GPU-bound on whatever's hosting it — no
		// cloud-scale batching — and a "thinking"/reasoning model (see
		// maxTokens below) can take a good while to work through its hidden
		// reasoning before ever emitting the actual translated line. Give it
		// real room before giving up.
		http: &http.Client{Timeout: 120 * time.Second},
	}
}

const maxAttempts = 3

// maxTokens is deliberately generous: a reasoning model (e.g. the Qwen3
// family) burns an outsized, unpredictable chunk of the completion budget on
// hidden chain-of-thought before it ever writes the actual answer — hand
// testing against LM Studio saw ~220 completion tokens spent translating a
// three-word phrase, almost all of it reasoning. A tight budget risks the
// response getting cut off (finishReason "length") with an empty answer, see
// doTranslate's handling of that case.
const maxTokens = 2048

// Translate sends one line of song lyrics through the local model, retrying
// with backoff on 429 (rate limited), 5xx, and a truncated-empty-answer
// response.
func (c *Client) Translate(ctx context.Context, text, sourceLang, targetLang string) (string, error) {
	prompt := llmprompt.Build(text, sourceLang, targetLang)

	body, err := json.Marshal(map[string]any{
		"model": c.model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.3,
		"max_tokens":  maxTokens,
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return "", true, fmt.Errorf("localllm: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	switch {
	case resp.StatusCode == http.StatusOK:
		var parsed struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
		}
		if err := json.Unmarshal(respBody, &parsed); err != nil {
			return "", false, fmt.Errorf("localllm: decode response: %w", err)
		}
		if len(parsed.Choices) == 0 {
			return "", false, fmt.Errorf("localllm: no choices returned")
		}

		translated := stripThinking(parsed.Choices[0].Message.Content)
		translated = strings.Trim(translated, "\"“”")
		if translated == "" {
			// Most likely maxTokens was exhausted by hidden reasoning before
			// any answer was written (see maxTokens' doc comment) — retrying
			// gives the model (temperature > 0, so not fully deterministic)
			// another shot at a shorter chain-of-thought.
			return "", true, fmt.Errorf("localllm: empty answer (finishReason=%s), likely truncated by max_tokens before the model finished reasoning", parsed.Choices[0].FinishReason)
		}
		return translated, false, nil

	case resp.StatusCode == http.StatusTooManyRequests:
		return "", true, fmt.Errorf("localllm: rate limited (429)")

	case resp.StatusCode >= 500:
		return "", true, fmt.Errorf("localllm: server error (%d)", resp.StatusCode)

	default:
		return "", false, fmt.Errorf("localllm: request rejected (%d): %s", resp.StatusCode, string(respBody))
	}
}

// stripThinking removes a leading <think>...</think> block from a reasoning
// model's answer. The LM Studio setup this was built against returns
// reasoning separately (message.reasoning_content, ignored above since it's
// never the translated line), but some other OpenAI-compatible backends or
// chat templates inline it into message.content instead — this is a
// defensive no-op against the former and a real fix against the latter.
func stripThinking(s string) string {
	for {
		start := strings.Index(s, "<think>")
		if start < 0 {
			break
		}
		rest := s[start+len("<think>"):]
		end := strings.Index(rest, "</think>")
		if end < 0 {
			// Unterminated block (truncated mid-thought) — drop everything
			// from the tag onward, there's no answer left to salvage.
			s = s[:start]
			break
		}
		s = s[:start] + rest[end+len("</think>"):]
	}
	return strings.TrimSpace(s)
}
