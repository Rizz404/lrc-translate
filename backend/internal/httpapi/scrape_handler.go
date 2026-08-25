package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"lrc-translate/backend/internal/align"
	appdb "lrc-translate/backend/internal/db"
	"lrc-translate/backend/internal/scrape"
)

// POST /api/tracks/:id/scrape { source_url }
// Cabang C, stage 1: fetch the user-supplied page and store its extracted
// raw text. Does not touch any Line yet.
func (s *Server) handleScrapeTrack(c *gin.Context) {
	trackID := c.Param("id")
	var track appdb.Track
	if err := s.db.First(&track, "id = ?", trackID).Error; err != nil {
		respondTrackLookupError(c, err)
		return
	}

	// A body is optional here (empty/omitted means "auto-discover"), unlike
	// other POST handlers, so only attempt to bind one if it was sent.
	var req ScrapeTrackRequest
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	var rawText, rawRomanized, language, resolvedURL string
	var err error
	autoDiscovered := req.SourceURL == ""

	if autoDiscovered {
		resolvedURL, rawText, rawRomanized, language, err = scrape.TryAutoDiscoverUtatime(c.Request.Context(), track.Artist, track.Title)
	} else {
		resolvedURL = req.SourceURL
		rawText, rawRomanized, language, err = scrape.Scrape(c.Request.Context(), resolvedURL)
	}
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, scrape.ErrDisallowedByRobots) {
			status = http.StatusForbidden
		}
		if autoDiscovered {
			// Auto-discovery failing isn't a server error — it's an
			// expected outcome the frontend should react to by prompting
			// for a manual URL (see plan-extended.md M3 decision on
			// utatime.com discovery).
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	source := appdb.ScrapeSource{
		TrackID:      trackID,
		SourceURL:    resolvedURL,
		RawText:      rawText,
		Language:     language,
		RawRomanized: rawRomanized,
		FetchedAt:    time.Now(),
	}
	if err := s.db.Create(&source).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ScrapeTrackResponse{
		ScrapeSourceID: source.ID,
		ResolvedURL:    source.SourceURL,
		RawText:        source.RawText,
		Language:       source.Language,
		RawRomanized:   source.RawRomanized,
		AutoDiscovered: autoDiscovered,
	})
}

// POST /api/tracks/:id/align { scrape_source_id }
// Cabang C, stage 2: run the alignment heuristic between the track's
// original lines and a previously-scraped source's raw text, writing
// results onto Lines with method="scrape" and needs_review=true (always —
// this heuristic is never treated as verified, see internal/align).
func (s *Server) handleAlignTrack(c *gin.Context) {
	track, err := s.loadTrack(c.Param("id"))
	if err != nil {
		respondTrackLookupError(c, err)
		return
	}

	var req AlignTrackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var source appdb.ScrapeSource
	if err := s.db.Where("id = ? AND track_id = ?", req.ScrapeSourceID, track.ID).First(&source).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "scrape source not found for this track"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Fill in/override the source's language when the caller supplies one —
	// the only way a non-utatime.com source (ScrapeTrackResponse.Language
	// empty) ever gets one, see AlignTrackRequest.Language.
	if req.Language != "" && req.Language != source.Language {
		source.Language = req.Language
		if err := s.db.Save(&source).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	originalTexts := make([]string, len(track.Lines))
	for i, l := range track.Lines {
		originalTexts[i] = l.Original
	}

	alignedTranslation, translationContexts := align.AlignWithContext(originalTexts, strings.Split(source.RawText, "\n"))

	// Romanization is optional (only utatime.com sources have it — see
	// internal/scrape/utatime.go) and aligned independently: its line
	// breakdown doesn't necessarily match the translation's.
	var alignedRomanized []string
	var romanizedContexts []align.Context
	if source.RawRomanized != "" {
		alignedRomanized, romanizedContexts = align.AlignWithContext(originalTexts, strings.Split(source.RawRomanized, "\n"))
	}

	resp := AlignTrackResponse{Lines: []LineDTO{}}
	for i, translation := range alignedTranslation {
		var romanized string
		if alignedRomanized != nil {
			romanized = alignedRomanized[i]
		}
		if translation == "" && romanized == "" {
			continue
		}

		line := track.Lines[i]
		if translation != "" {
			line.Translation = translation
			line.TranslationLang = source.Language
			line.Method = appdb.MethodScrape
		}
		if romanized != "" {
			// UtaTime's own romanization tends to be more accurate than
			// this app's kagome/gojp-kana pipeline, so it replaces
			// whatever was there (including a prior auto-generated one).
			// RomanizedSource="scrape" also protects it from later being
			// clobbered by the internal pipeline — see handleRomanizeTrack.
			line.Romanized = romanized
			line.RomanizedSource = "scrape"
		}
		line.NeedsReview = true

		// Snapshot this run's raw-scraped neighborhood for the editor's
		// side-by-side review UI — see db.Line.ScrapeContext. Replaces
		// whatever was there before wholesale (rather than merging field by
		// field) so it can never show a neighborhood left over from a
		// different scrape source than the one this line's text actually
		// came from.
		ctxs := LineScrapeContextsDTO{
			Translation: toScrapeContextDTO(translationContexts[i]),
		}
		if romanizedContexts != nil {
			ctxs.Romanized = toScrapeContextDTO(romanizedContexts[i])
		}
		if ctxs.Translation != nil || ctxs.Romanized != nil {
			if encoded, err := json.Marshal(ctxs); err == nil {
				line.ScrapeContext = string(encoded)
			}
		}

		if err := s.db.Save(&line).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save aligned line: " + err.Error()})
			return
		}
		resp.Lines = append(resp.Lines, toLineDTO(line))
	}

	c.JSON(http.StatusOK, resp)
}

// toScrapeContextDTO converts an align.Context into its JSON-storable DTO
// shape, or nil when nothing was actually matched (a zero-value Context —
// e.g. an instrumental gap, or scrapedRaw too short to cover this line).
func toScrapeContextDTO(ctx align.Context) *LineScrapeContextDTO {
	if ctx.Matched == "" && ctx.Prev == "" && ctx.Next == "" {
		return nil
	}
	return &LineScrapeContextDTO{Prev: ctx.Prev, Matched: ctx.Matched, Next: ctx.Next}
}
