package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	appdb "lrc-translate/backend/internal/db"
)

// POST /api/tracks/:id/reset
// Wipes Translation/Romanized/RomanizedSource/Method (plus the manual-edit
// SuggestedTranslation/SuggestedMethod snapshot and NeedsReview flag) on
// every line back to pristine — i.e. undoes all translation, romanization,
// scrape-alignment, and manual-edit progress in one shot. Original text and
// Timestamp/TimeMs are left untouched: this resets *translation progress*,
// not the synced lyrics themselves (there's no separate endpoint to reset
// those — re-import the track for that).
func (s *Server) handleResetTrack(c *gin.Context) {
	track, err := s.loadTrack(c.Param("id"))
	if err != nil {
		respondTrackLookupError(c, err)
		return
	}

	resp := ResetTrackResponse{Lines: make([]LineDTO, len(track.Lines))}
	for i, line := range track.Lines {
		line.Translation = ""
		line.Romanized = ""
		line.RomanizedSource = ""
		line.Method = appdb.MethodNone
		line.SuggestedTranslation = ""
		line.SuggestedMethod = ""
		line.NeedsReview = false

		if err := s.db.Save(&line).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		resp.Lines[i] = toLineDTO(line)
	}

	c.JSON(http.StatusOK, resp)
}
