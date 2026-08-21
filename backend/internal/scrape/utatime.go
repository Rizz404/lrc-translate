// utatime.go adds site-aware handling for utatime.com (https://www.utatime.com).
//
// Its lyric pages have an unusually clean, purpose-built structure for our
// use case: a "#Original" tab with the original lyrics and a "#Translations"
// tab with per-language subtabs, each rendering one <span class="line-text">
// per line (blank lines are their own numbered "<br/>"-only entry) — already
// line-aligned by the site itself, no heuristic guessing needed on their
// side. See parseUtatimeHTML.
//
// URL discovery is the hard part. utatime.com's own search
// (SearchConfig.endpoint = /music-api/site/v1/search) and its sitemap paths
// sit behind a Cloudflare bot challenge that a plain HTTP client can't pass,
// and during development this repo's own probing traffic got rate-limited
// (transient 403s) after roughly a dozen requests in a couple of minutes —
// so this package deliberately tries only a couple of guessed URLs and
// leaves the rest to the "paste the URL yourself" fallback already in
// scrape_handler.go, rather than hammering the site with more attempts.
package scrape

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func isUtatimeHost(host string) bool {
	h := strings.ToLower(host)
	return h == "utatime.com" || h == "www.utatime.com"
}

var (
	nonSlugCharsRe  = regexp.MustCompile(`[^a-z0-9\s-]`)
	whitespaceRunRe = regexp.MustCompile(`[\s-]+`)
)

// slugify approximates WordPress's default sanitize_title(): lowercase,
// drop anything but letters/digits/spaces/hyphens, collapse whitespace runs
// into single hyphens. It only works for latin-script input — utatime.com's
// own artist slugs are often a Japanese-reading romanization (e.g. artist
// "TUYU" -> slug "tsuyu") that no generic transform of our stored metadata
// can reliably reproduce, which is exactly why this is a "best guess, may
// well fail" helper and not a real search.
func slugify(s string) string {
	s = strings.ToLower(s)
	s = nonSlugCharsRe.ReplaceAllString(s, "")
	s = whitespaceRunRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// guessUtatimeURLs returns a short, ordered list of candidate lyric page
// URLs for the given artist/title. Deliberately short (see package doc) —
// this is a best-effort guess, not a crawl.
func guessUtatimeURLs(artist, title string) []string {
	a, t := slugify(artist), slugify(title)
	if a == "" || t == "" {
		return nil
	}
	return []string{
		fmt.Sprintf("https://www.utatime.com/global/lyrics/%s/%s/", a, t),
	}
}

// TryAutoDiscoverUtatime guesses at a utatime.com URL for artist/title,
// fetches the first guess that resolves to a real lyrics page, and returns
// its resolved URL plus extracted translation text. Callers should treat any
// error here as "auto-discovery failed" and fall back to asking the user for
// the page URL directly — not as a reason to retry more guesses.
func TryAutoDiscoverUtatime(ctx context.Context, artist, title string) (resolvedURL, translation string, err error) {
	candidates := guessUtatimeURLs(artist, title)
	if len(candidates) == 0 {
		return "", "", fmt.Errorf("artist/title has no usable characters to guess a URL from")
	}

	var lastErr error
	for _, candidate := range candidates {
		target, parseErr := url.Parse(candidate)
		if parseErr != nil {
			lastErr = parseErr
			continue
		}

		allowed, robotsErr := checkRobots(ctx, target)
		if robotsErr == nil && !allowed {
			lastErr = ErrDisallowedByRobots
			continue
		}

		html, fetchErr := fetch(ctx, candidate)
		if fetchErr != nil {
			lastErr = fetchErr
			continue
		}

		tl, parseHTMLErr := parseUtatimeHTML(html)
		if parseHTMLErr != nil {
			lastErr = parseHTMLErr
			continue
		}

		return candidate, tl, nil
	}

	return "", "", fmt.Errorf("couldn't find this song on utatime.com automatically (%v) — paste the page URL instead", lastErr)
}

// parseUtatimeHTML extracts the translation lines from a utatime.com lyric
// page: it picks the English subtab under #Translations if present,
// otherwise the first language subtab available, and reads one line per
// ".olyrictext .line-text" span. goquery's .Text() already turns a
// "<br/>"-only span into an empty string, which lines up with align.go's
// convention of blank lines marking block boundaries.
func parseUtatimeHTML(html string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", fmt.Errorf("parse HTML: %w", err)
	}

	var target *goquery.Selection
	doc.Find(`#Translations .subcontents`).EachWithBreak(func(_ int, sel *goquery.Selection) bool {
		id, _ := sel.Attr("id")
		if target == nil {
			target = sel // remember the first as a fallback
		}
		if strings.EqualFold(id, "English") {
			target = sel
			return false // English found, stop looking
		}
		return true
	})
	if target == nil {
		return "", fmt.Errorf("no translation tab found on this utatime.com page")
	}

	var lines []string
	target.Find(".olyrictext .line-text").Each(func(_ int, sel *goquery.Selection) {
		lines = append(lines, strings.TrimSpace(sel.Text()))
	})
	if len(lines) == 0 {
		return "", fmt.Errorf("translation tab found but contained no text")
	}

	return strings.Join(lines, "\n"), nil
}
