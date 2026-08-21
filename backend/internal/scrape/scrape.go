// Package scrape fetches a user-provided web page and extracts its readable
// text, for Cabang C (scraping + alignment).
//
// Design note (see plan-extended.md): the original plan sketched an
// auto-discovery scraper against a fixed target like a lyrics wiki. During
// implementation, the obvious candidate databases turned out to be
// unusable: lyricstranslate.com's robots.txt explicitly disallows AI/bot
// user-agents (including Claude), and Fandom sits behind bot-blocking that
// makes even fetching robots.txt fail. Rather than working around either of
// those, this package scrapes whatever URL the *user* pastes in — content
// they already found and chose to use — and still performs its own
// robots.txt courtesy check against that URL before fetching, so it
// respects whatever policy the target site (any site) declares.
//
// utatime.com (see utatime.go) is the one exception with a site-aware
// parser and best-effort URL auto-discovery.
package scrape

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/temoto/robotstxt"
)

// userAgent identifies this tool honestly to servers it talks to, rather
// than spoofing a browser.
const userAgent = "lrc-translate-bot/0.1 (+personal lyric-translation tool; fetches only URLs a user explicitly supplies)"

var httpClient = &http.Client{Timeout: 15 * time.Second}

// ErrDisallowedByRobots is returned when the target site's robots.txt
// disallows this bot from fetching the requested path.
var ErrDisallowedByRobots = fmt.Errorf("scraping disallowed by the site's robots.txt")

// Scrape fetches rawURL (after checking robots.txt) and returns its
// extracted translation text, suitable as input to align.Align. Recognizes
// utatime.com URLs and uses its site-aware parser (utatime.go) instead of
// the generic readability heuristic.
func Scrape(ctx context.Context, rawURL string) (string, error) {
	target, err := url.Parse(rawURL)
	if err != nil || (target.Scheme != "http" && target.Scheme != "https") {
		return "", fmt.Errorf("invalid URL %q", rawURL)
	}

	allowed, err := checkRobots(ctx, target)
	if err != nil {
		// Fail open on robots.txt fetch errors (e.g. no robots.txt at all,
		// which is a valid and common case meaning "everything allowed") —
		// only a fetched-and-parsed explicit disallow blocks us.
		allowed = true
	}
	if !allowed {
		return "", ErrDisallowedByRobots
	}

	html, err := fetch(ctx, target.String())
	if err != nil {
		return "", err
	}

	if isUtatimeHost(target.Host) {
		return parseUtatimeHTML(html)
	}
	return extractText(html)
}

func checkRobots(ctx context.Context, target *url.URL) (bool, error) {
	robotsURL := fmt.Sprintf("%s://%s/robots.txt", target.Scheme, target.Host)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, robotsURL, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// No robots.txt (404) or inaccessible -> treat as "no restrictions
		// declared" rather than blocking the user's request.
		return true, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	robots, err := robotstxt.FromBytes(body)
	if err != nil {
		return false, err
	}

	return robots.TestAgent(target.Path, userAgent), nil
}

func fetch(ctx context.Context, rawURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch %s: unexpected status %d", rawURL, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// noiseTagsSelector removes elements that are never lyric/translation
// content: scripts, styles, navigation chrome, and common ad/comment
// containers.
const noiseTagsSelector = "script, style, nav, header, footer, aside, form, noscript, iframe, svg"

var whitespaceRe = regexp.MustCompile(`[ \t]+`)

// extractText pulls readable text out of raw HTML using a generic
// readability-style heuristic: strip non-content chrome, then read text
// from block-level elements (p, li, br-separated div/td content), one
// output line per element, deduplicating immediate repeats and dropping
// very short boilerplate fragments.
func extractText(html string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", fmt.Errorf("parse HTML: %w", err)
	}

	doc.Find(noiseTagsSelector).Remove()

	var lines []string
	seenLast := ""

	doc.Find("p, li, td, div, span").Each(func(_ int, sel *goquery.Selection) {
		// Only take elements whose direct text (not just descendant text)
		// is non-trivial, to avoid emitting the same paragraph once per
		// wrapping container.
		if sel.Children().Length() > 0 && !hasDirectText(sel) {
			return
		}

		text := strings.TrimSpace(whitespaceRe.ReplaceAllString(sel.Text(), " "))
		if text == "" || len([]rune(text)) < 2 {
			return
		}
		if text == seenLast {
			return // collapse immediate duplicates (nested containers, etc.)
		}

		lines = append(lines, text)
		seenLast = text
	})

	if len(lines) == 0 {
		return "", fmt.Errorf("no readable text found on page")
	}

	return strings.Join(lines, "\n"), nil
}

// hasDirectText reports whether sel has non-whitespace text as a direct
// child text node (not just nested inside child elements).
func hasDirectText(sel *goquery.Selection) bool {
	direct := false
	sel.Contents().Each(func(_ int, c *goquery.Selection) {
		if goquery.NodeName(c) == "#text" && strings.TrimSpace(c.Text()) != "" {
			direct = true
		}
	})
	return direct
}
