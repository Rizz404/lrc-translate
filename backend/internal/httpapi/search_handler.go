package httpapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// GET /api/search?title=&artist=
func (s *Server) handleSearch(c *gin.Context) {
	title := normalizeQuery(c.Query("title"))
	artist := normalizeQuery(c.Query("artist"))

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

// normalizeQuery trims leading/trailing whitespace and collapses any run of
// internal whitespace down to a single space. Without this, a query like
// "naruto  shippuden" (stray double space) or " naruto" (leading space) gets
// forwarded to LRCLIB as-is and comes back with zero results even though a
// clean "naruto shippuden"/"naruto" query would match — strings.Fields+Join
// is the idiomatic way to do this in one pass (Fields already splits on any
// whitespace run and drops empties).
func normalizeQuery(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
