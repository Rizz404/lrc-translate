// utatime.go adds site-aware handling for utatime.com (https://www.utatime.com).
//
// Its lyric pages have an unusually clean, purpose-built structure for our
// use case: a "#Original" tab with the original lyrics and a "#Translations"
// tab with per-language subtabs, each rendering one <span class="line-text">
// per line (blank lines are their own numbered "<br/>"-only entry) — already
// line-aligned by the site itself, no heuristic guessing needed on their
// side. See parseUtatimeHTML.
//
// URL discovery used to be the hard part: naively guessing a WordPress-style
// slug from our own stored artist/title metadata often misses, because
// utatime.com's slugs are frequently a Japanese-reading romanization we have
// no reliable way to reproduce (e.g. artist "TUYU" -> slug "tsuyu") — see
// slugify. The fix is to not guess at all: utatime.com's own site search API
// (/music-api/site/v1/search, the same endpoint the search box at
// https://www.utatime.com/global/#search calls) turns out to be reachable
// with a plain HTTP client and returns a JSON payload with an HTML fragment
// of results — no Cloudflare bot challenge in the way, as earlier probing
// during development had assumed. So this package now searches for
// "<artist> <title>" and takes the first song result, rather than guessing.
//
// One wrinkle: search results link to the Japanese-first page
// (https://www.utatime.com/lyrics/<artist>/<title>/), which has no
// #Translations/#Romaji tabs — those only exist on the "global" edition at
// https://www.utatime.com/global/lyrics/<artist>/<title>/. See
// toUtatimeGlobalURL, which rewrites one into the other.
//
// This still isn't hammered: one search request plus one page fetch per
// discovery call. The old guessed-slug URL is kept as a fallback candidate
// (tried after the search result, or alone if the search itself errors) in
// case the search endpoint is ever unreachable — not because it's expected
// to out-guess a real search.
package scrape

import (
	"context"
	"encoding/json"
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

// utatimeSearchEndpoint is utatime.com's own site search API — the same one
// the search box at https://www.utatime.com/global/#search calls.
const utatimeSearchEndpoint = "https://www.utatime.com/music-api/site/v1/search"

// utatimeSearchResponse is the shape of utatimeSearchEndpoint's JSON body:
// not structured search results, just a rendered HTML fragment (the same
// markup the site's own JS drops into the page), which parseUtatimeSearchHTML
// picks apart.
type utatimeSearchResponse struct {
	HTML string `json:"html"`
}

// toUtatimeGlobalURL rewrites a utatime.com lyric URL to use its "/global/"
// path segment. Search results link to the Japanese-first page (e.g.
// https://www.utatime.com/lyrics/kessoku-band/seiza-ni-naretara/), which has
// no #Translations/#Romaji tabs; the same lyrics live with those tabs added
// at https://www.utatime.com/global/lyrics/kessoku-band/seiza-ni-naretara/.
// URLs that already have a "/global/" segment (or don't match the expected
// "/lyrics/" shape at all) are returned unchanged.
func toUtatimeGlobalURL(rawURL string) string {
	if strings.Contains(rawURL, "/global/") {
		return rawURL
	}
	const marker = "/lyrics/"
	i := strings.Index(rawURL, marker)
	if i < 0 {
		return rawURL
	}
	return rawURL[:i] + "/global" + rawURL[i:]
}

// parseUtatimeSearchHTML picks the first song result out of
// utatimeSearchResponse.HTML and returns its lyric page URL (rewritten to
// the "/global/" edition — see toUtatimeGlobalURL). utatime.com's search
// results page groups results into sections (songs, tie-ins, lyricists,
// ...); only "曲" (songs) results carry the "search-result-lyric" class,
// which is what distinguishes an actual lyric page link from e.g. an anime
// series or lyricist page that happened to match the query text.
func parseUtatimeSearchHTML(fragmentHTML string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(fragmentHTML))
	if err != nil {
		return "", fmt.Errorf("parse search result HTML: %w", err)
	}

	href, ok := doc.Find("a.search-result-lyric").First().Attr("href")
	if !ok || href == "" {
		return "", fmt.Errorf("no song result found")
	}

	return toUtatimeGlobalURL(href), nil
}

// searchUtatimeLyricURL queries utatime.com's own search for "<artist>
// <title>" and returns the resolved lyric page URL of the first song
// result. Callers should treat any error as "no result", not as a reason to
// retry with a different query — see the package doc comment on why this
// package doesn't hammer the endpoint.
func searchUtatimeLyricURL(ctx context.Context, artist, title string) (string, error) {
	query := strings.TrimSpace(artist + " " + title)
	if query == "" {
		return "", fmt.Errorf("artist/title is empty, nothing to search for")
	}

	endpoint := utatimeSearchEndpoint + "?q=" + url.QueryEscape(query)
	target, parseErr := url.Parse(endpoint)
	if parseErr != nil {
		return "", parseErr
	}

	if allowed, robotsErr := checkRobots(ctx, target); robotsErr == nil && !allowed {
		return "", ErrDisallowedByRobots
	}

	body, err := fetch(ctx, endpoint)
	if err != nil {
		return "", err
	}

	var parsed utatimeSearchResponse
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return "", fmt.Errorf("parse search response: %w", err)
	}

	return parseUtatimeSearchHTML(parsed.HTML)
}

// TryAutoDiscoverUtatime searches utatime.com for artist/title (see
// searchUtatimeLyricURL), falling back to a guessed slug URL if the search
// itself fails, fetches the first candidate that resolves to a real lyrics
// page, and returns its resolved URL plus extracted translation/romanized
// text. Callers should treat any error here as "auto-discovery failed" and
// fall back to asking the user for the page URL directly — not as a reason
// to retry more guesses.
func TryAutoDiscoverUtatime(ctx context.Context, artist, title string) (resolvedURL, translation, romanized string, err error) {
	var candidates []string
	searchURL, searchErr := searchUtatimeLyricURL(ctx, artist, title)
	if searchErr == nil {
		candidates = append(candidates, searchURL)
	}
	candidates = append(candidates, guessUtatimeURLs(artist, title)...)

	if len(candidates) == 0 {
		return "", "", "", fmt.Errorf("artist/title has no usable characters to guess a URL from")
	}

	lastErr := searchErr
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

		tl, rj, parseHTMLErr := parseUtatimeHTML(html)
		if parseHTMLErr != nil {
			lastErr = parseHTMLErr
			continue
		}

		return candidate, tl, rj, nil
	}

	return "", "", "", fmt.Errorf("couldn't find this song on utatime.com automatically (%v) — paste the page URL instead", lastErr)
}

// parseUtatimeHTML extracts translation and romanization from a utatime.com
// lyric page.
//
// Translation: picks the English subtab under #Translations if present,
// otherwise the first language subtab available (this mirrors picking the
// top entry of the page's own "Add Translation" dropdown, since English
// (UtaTime)'s own official translation is listed first when available).
// Missing entirely is an error — there's nothing useful to align without it.
//
// Romanization: read from #Romaji. UtaTime's own romanization is generally
// more accurate than this app's own kagome/gojp-kana pipeline (word
// grouping, sokuon handling, etc.), so when present it's used to replace —
// not just supplement — the auto-generated one for aligned lines (see
// scrape_handler.go). Missing is not an error; the app's own romanizer
// still covers that case.
//
// Both read one line per ".olyrictext .line-text" span; goquery's .Text()
// already turns a "<br/>"-only span into an empty string, which lines up
// with align.go's convention of blank lines marking block boundaries.
func parseUtatimeHTML(html string) (translation, romanized string, err error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", "", fmt.Errorf("parse HTML: %w", err)
	}

	var translationTab *goquery.Selection
	doc.Find(`#Translations .subcontents`).EachWithBreak(func(_ int, sel *goquery.Selection) bool {
		id, _ := sel.Attr("id")
		if translationTab == nil {
			translationTab = sel // remember the first as a fallback
		}
		if strings.EqualFold(id, "English") {
			translationTab = sel
			return false // English found, stop looking
		}
		return true
	})
	if translationTab == nil {
		return "", "", fmt.Errorf("no translation tab found on this utatime.com page")
	}

	translationLines := extractOlyrictextLines(translationTab)
	if len(translationLines) == 0 {
		return "", "", fmt.Errorf("translation tab found but contained no text")
	}

	romanized = strings.Join(extractOlyrictextLines(doc.Find("#Romaji").First()), "\n")

	return strings.Join(translationLines, "\n"), romanized, nil
}

func extractOlyrictextLines(scope *goquery.Selection) []string {
	var lines []string
	scope.Find(".olyrictext .line-text").Each(func(_ int, sel *goquery.Selection) {
		lines = append(lines, strings.TrimSpace(sel.Text()))
	})
	return lines
}
