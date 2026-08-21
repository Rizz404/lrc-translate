package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	appdb "lrc-translate/backend/internal/db"
)

// POST /api/tracks/:id/lines/:lineId/revert
// Restores Translation/Method from the SuggestedTranslation/SuggestedMethod
// snapshot taken just before the last manual edit (see handleUpdateLine).
// A no-op if the line was never manually edited (nothing to revert to).
func (s *Server) handleRevertLine(c *gin.Context) {
	var line appdb.Line
	if err := s.db.Where("id = ? AND track_id = ?", c.Param("lineId"), c.Param("id")).First(&line).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "line not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if line.SuggestedMethod == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "no earlier suggestion to revert to"})
		return
	}

	line.Translation = line.SuggestedTranslation
	line.Method = line.SuggestedMethod
	line.SuggestedTranslation = ""
	line.SuggestedMethod = ""

	if err := s.db.Save(&line).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, toLineDTO(line))
}
