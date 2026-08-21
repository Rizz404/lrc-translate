package scrape

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestExtractText_StripsChromeAndDeduplicates(t *testing.T) {
	html := `
	<html>
	<head><style>.x{color:red}</style><script>alert(1)</script></head>
	<body>
		<nav>Home | About | Contact</nav>
		<header>Site Header</header>
		<main>
			<p>Sora ga aoi</p>
			<p>The sky is blue</p>
			<div><p>Nested paragraph should not duplicate its parent div's text</p></div>
		</main>
		<footer>Copyright 2026</footer>
	</body>
	</html>`

	got, err := extractText(html)
	if err != nil {
		t.Fatalf("extractText returned error: %v", err)
	}

	if strings.Contains(got, "Site Header") || strings.Contains(got, "Home | About") || strings.Contains(got, "Copyright") {
		t.Errorf("expected nav/header/footer chrome to be stripped, got: %q", got)
	}
	if !strings.Contains(got, "Sora ga aoi") || !strings.Contains(got, "The sky is blue") {
		t.Errorf("expected content lines present, got: %q", got)
	}

	// The wrapping <div> around the nested <p> has no direct text of its
	// own, so it should not duplicate the paragraph's text.
	count := strings.Count(got, "Nested paragraph should not duplicate its parent div's text")
	if count != 1 {
		t.Errorf("expected nested paragraph text exactly once, got %d times in: %q", count, got)
	}
}

func TestExtractText_ErrorsOnEmptyPage(t *testing.T) {
	_, err := extractText(`<html><body><nav>only chrome</nav></body></html>`)
	if err == nil {
		t.Fatal("expected error for a page with no extractable content")
	}
}

func TestCheckRobots_DisallowsBlockedPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			w.Write([]byte("User-agent: *\nDisallow: /private/\n"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	target := mustParseURL(t, srv.URL+"/private/page.html")
	allowed, err := checkRobots(context.Background(), target)
	if err != nil {
		t.Fatalf("checkRobots returned error: %v", err)
	}
	if allowed {
		t.Error("expected /private/ to be disallowed")
	}

	target2 := mustParseURL(t, srv.URL+"/public/page.html")
	allowed2, err := checkRobots(context.Background(), target2)
	if err != nil {
		t.Fatalf("checkRobots returned error: %v", err)
	}
	if !allowed2 {
		t.Error("expected /public/ to be allowed")
	}
}

func TestCheckRobots_NoRobotsTxtMeansAllowed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	target := mustParseURL(t, srv.URL+"/anything")
	allowed, err := checkRobots(context.Background(), target)
	if err != nil {
		t.Fatalf("checkRobots returned error: %v", err)
	}
	if !allowed {
		t.Error("expected missing robots.txt to mean allowed")
	}
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}
	return u
}
