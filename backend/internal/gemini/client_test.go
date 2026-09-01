package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// geminiResponse builds a generateContent-shaped response body.
func geminiResponse(text, finishReason string) map[string]any {
	return map[string]any{
		"candidates": []map[string]any{
			{
				"content":      map[string]any{"parts": []map[string]string{{"text": text}}},
				"finishReason": finishReason,
			},
		},
	}
}

func TestTranslate_SucceedsFirstTry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(geminiResponse("halo dunia", "STOP"))
	}))
	defer srv.Close()

	c := New("test-key", "test-model")
	c.http = srv.Client()
	c.baseURL = srv.URL

	got, err := c.Translate(context.Background(), "hello world", "en", "id")
	if err != nil {
		t.Fatalf("Translate returned error: %v", err)
	}
	if got != "halo dunia" {
		t.Errorf("got %q, want %q", got, "halo dunia")
	}
}

func TestTranslateBatch_SucceedsFirstTry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(geminiResponse(`["halo", "dunia"]`, "STOP"))
	}))
	defer srv.Close()

	c := New("test-key", "test-model")
	c.http = srv.Client()
	c.baseURL = srv.URL

	got, err := c.TranslateBatch(context.Background(), []string{"hello", "world"}, "en", "id")
	if err != nil {
		t.Fatalf("TranslateBatch returned error: %v", err)
	}
	want := []string{"halo", "dunia"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestTranslateBatch_RetriesOnCountMismatchThenSucceeds(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&attempts, 1) < 2 {
			json.NewEncoder(w).Encode(geminiResponse(`["only one"]`, "STOP"))
			return
		}
		json.NewEncoder(w).Encode(geminiResponse(`["satu", "dua"]`, "STOP"))
	}))
	defer srv.Close()

	c := New("test-key", "test-model")
	c.http = srv.Client()
	c.baseURL = srv.URL

	got, err := c.TranslateBatch(context.Background(), []string{"one", "two"}, "en", "id")
	if err != nil {
		t.Fatalf("TranslateBatch returned error: %v", err)
	}
	if len(got) != 2 || got[0] != "satu" || got[1] != "dua" {
		t.Errorf("got %v, want [satu dua]", got)
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestTranslateBatch_EmptyInputReturnsNilWithoutRequest(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		json.NewEncoder(w).Encode(geminiResponse("[]", "STOP"))
	}))
	defer srv.Close()

	c := New("test-key", "test-model")
	c.http = srv.Client()
	c.baseURL = srv.URL

	got, err := c.TranslateBatch(context.Background(), nil, "en", "id")
	if err != nil {
		t.Fatalf("TranslateBatch returned error: %v", err)
	}
	if got != nil {
		t.Errorf("got %v, want nil", got)
	}
	if called {
		t.Error("expected no HTTP request for an empty batch")
	}
}
