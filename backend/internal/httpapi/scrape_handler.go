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

	var req ScrapeTrackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rawText, err := scrape.Scrape(c.Request.Context(), req.SourceURL)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, scrape.ErrDisallowedByRobots) {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	source := appdb.ScrapeSource{
		TrackID:   trackID,
		SourceURL: req.SourceURL,
		RawText:   rawText,
		FetchedAt: time.Now(),
	}
	if err := s.db.Create(&source).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ScrapeTrackResponse{
		ScrapeSourceID: source.ID,
		SourceURL:      source.SourceURL,
		RawText:        source.RawText,
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
	scrapedLines := strings.Split(source.RawText, "\n")

	aligned := align.Align(originalTexts, scrapedLines)

	resp := AlignTrackResponse{Lines: []LineDTO{}}
	for i, translation := range aligned {
		if translation == "" {
			continue
		}
		line := track.Lines[i]
		line.Translation = translation
		line.Method = appdb.MethodScrape
		line.NeedsReview = true
		if err := s.db.Save(&line).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save aligned line: " + err.Error()})
			return
		}
		resp.Lines = append(resp.Lines, toLineDTO(line))
	}

	c.JSON(http.StatusOK, resp)
}
