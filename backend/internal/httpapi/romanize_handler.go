package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// POST /api/tracks/:id/romanize
// Runs the kagome+gojp/kana pipeline over every non-empty line and persists
// the result. Cheap enough (and idempotent) to just always recompute for all
// lines rather than tracking which ones already have a romanization — except
// lines whose Romanized already came from a scrape source (e.g. utatime.com,
// see handleAlignTrack) or a manual edit (see handleUpdateLine): both are
// trusted over this pipeline, so they're left untouched and counted in
// SkippedCount instead.
func (s *Server) handleRomanizeTrack(c *gin.Context) {
	track, err := s.loadTrack(c.Param("id"))
	if err != nil {
		respondTrackLookupError(c, err)
		return
	}

	resp := RomanizeResponse{Lines: []LineDTO{}}
	for _, line := range track.Lines {
		if line.Original == "" {
			continue
		}
		if line.RomanizedSource == "scrape" || line.RomanizedSource == "manual" {
			resp.SkippedCount++
			continue
		}
		line.Romanized = s.romanizer.ToRomaji(line.Original)
		line.RomanizedSource = "internal"
		if err := s.db.Save(&line).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		resp.Lines = append(resp.Lines, toLineDTO(line))
	}

	c.JSON(http.StatusOK, resp)
}
