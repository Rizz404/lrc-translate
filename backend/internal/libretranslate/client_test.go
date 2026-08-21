package libretranslate

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestTranslate_SucceedsFirstTry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"translatedText": "halo dunia"})
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	got, err := c.Translate(context.Background(), "hello world", "en", "id")
	if err != nil {
		t.Fatalf("Translate returned error: %v", err)
	}
	if got != "halo dunia" {
		t.Errorf("got %q, want %q", got, "halo dunia")
	}
}

func TestTranslate_RetriesOn429ThenSucceeds(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&attempts, 1) < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"rate limited"}`))
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"translatedText": "ok"})
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	got, err := c.Translate(context.Background(), "hi", "en", "id")
	if err != nil {
		t.Fatalf("Translate returned error: %v", err)
	}
	if got != "ok" {
		t.Errorf("got %q, want %q", got, "ok")
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts (2 retries), got %d", attempts)
	}
}

func TestTranslate_RetriesOn5xxThenSucceeds(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&attempts, 1) < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"translatedText": "ok"})
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	got, err := c.Translate(context.Background(), "hi", "en", "id")
	if err != nil {
		t.Fatalf("Translate returned error: %v", err)
	}
	if got != "ok" {
		t.Errorf("got %q, want %q", got, "ok")
	}
}

func TestTranslate_DoesNotRetryOn400(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Visit https://portal.libretranslate.com to get an API key"}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	_, err := c.Translate(context.Background(), "hi", "en", "id")
	if err == nil {
		t.Fatal("expected an error for 400 response, got nil")
	}
	if attempts != 1 {
		t.Errorf("expected exactly 1 attempt (no retry on 400), got %d", attempts)
	}
}

func TestTranslate_GivesUpAfterMaxAttempts(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := New(srv.URL, "")
	_, err := c.Translate(context.Background(), "hi", "en", "id")
	if err == nil {
		t.Fatal("expected an error after exhausting retries, got nil")
	}
	if attempts != maxAttempts {
		t.Errorf("expected %d attempts, got %d", maxAttempts, attempts)
	}
}
