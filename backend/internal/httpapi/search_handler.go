package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GET /api/search?title=&artist=
func (s *Server) handleSearch(c *gin.Context) {
	title := c.Query("title")
	artist := c.Query("artist")

	if title == "" && artist == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provide at least one of title or artist"})
		return
	}

	tracks, err := s.lrclib.Search(c.Request.Context(), title, artist)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to query LRCLIB: " + err.Error()})
		return
	}

	results := make([]SearchResultDTO, 0, len(tracks))
	for _, t := range tracks {
		results = append(results, SearchResultDTO{
			LrclibID:        t.ID,
			Title:           t.TrackName,
			Artist:          t.ArtistName,
			Album:           t.AlbumName,
			DurationMs:      int64(t.Duration * 1000),
			Instrumental:    t.Instrumental,
			HasSyncedLyrics: t.SyncedLyrics != "",
		})
	}

	c.JSON(http.StatusOK, results)
}
