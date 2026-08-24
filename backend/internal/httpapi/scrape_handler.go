package httpapi

import (
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

	var rawText, rawRomanized, resolvedURL string
	var err error
	autoDiscovered := req.SourceURL == ""

	if autoDiscovered {
		resolvedURL, rawText, rawRomanized, err = scrape.TryAutoDiscoverUtatime(c.Request.Context(), track.Artist, track.Title)
	} else {
		resolvedURL = req.SourceURL
		rawText, rawRomanized, err = scrape.Scrape(c.Request.Context(), resolvedURL)
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

	originalTexts := make([]string, len(track.Lines))
	for i, l := range track.Lines {
		originalTexts[i] = l.Original
	}

	alignedTranslation := align.Align(originalTexts, strings.Split(source.RawText, "\n"))

	// Romanization is optional (only utatime.com sources have it — see
	// internal/scrape/utatime.go) and aligned independently: its line
	// breakdown doesn't necessarily match the translation's.
	var alignedRomanized []string
	if source.RawRomanized != "" {
		alignedRomanized = align.Align(originalTexts, strings.Split(source.RawRomanized, "\n"))
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
		if err := s.db.Save(&line).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save aligned line: " + err.Error()})
			return
		}
		resp.Lines = append(resp.Lines, toLineDTO(line))
	}

	c.JSON(http.StatusOK, resp)
}
