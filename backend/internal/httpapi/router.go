// Package httpapi wires Gin routes to the app's handlers.
package httpapi

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"lrc-translate/backend/internal/libretranslate"
	"lrc-translate/backend/internal/lrclib"
	"lrc-translate/backend/internal/romanize"
)

// Server holds the dependencies shared by all HTTP handlers.
type Server struct {
	db         *gorm.DB
	lrclib     *lrclib.Client
	translator *libretranslate.Client
	romanizer  *romanize.Romanizer
}

// NewServer builds a Server with its dependencies.
func NewServer(db *gorm.DB, lrclibClient *lrclib.Client, translator *libretranslate.Client, romanizer *romanize.Romanizer) *Server {
	return &Server{db: db, lrclib: lrclibClient, translator: translator, romanizer: romanizer}
}

// NewRouter builds the Gin engine with CORS (restricted to allowedOrigin) and
// all routes registered. When staticDir is non-empty, it also serves the
// built frontend (SPA) from that directory, falling back to index.html for
// any unmatched non-/api route so that client-side routing works on refresh
// — this is what lets a single container serve both the API and the UI.
func NewRouter(s *Server, allowedOrigin, staticDir string) *gin.Engine {
	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", allowedOrigin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	api := r.Group("/api")
	{
		api.GET("/health", s.handleHealth)
		api.GET("/search", s.handleSearch)

		api.GET("/tracks", s.handleListTracks)
		api.POST("/tracks/import", s.handleImportTrack)
		api.GET("/tracks/:id", s.handleGetTrack)
		api.PUT("/tracks/:id", s.handleUpdateTrack)
		api.DELETE("/tracks/:id", s.handleDeleteTrack)

		api.PUT("/tracks/:id/lines/:lineId", s.handleUpdateLine)
		api.POST("/tracks/:id/lines/:lineId/revert", s.handleRevertLine)
		api.POST("/tracks/:id/reset", s.handleResetTrack)

		api.POST("/tracks/:id/translate", s.handleTranslateTrack)
		api.POST("/tracks/:id/translate/clear", s.handleClearTranslation)
		api.POST("/tracks/:id/romanize", s.handleRomanizeTrack)

		api.POST("/tracks/:id/scrape", s.handleScrapeTrack)
		api.POST("/tracks/:id/align", s.handleAlignTrack)
	}

	if staticDir != "" {
		fileServer := http.FileServer(http.Dir(staticDir))
		r.NoRoute(func(c *gin.Context) {
			reqPath := c.Request.URL.Path
			if strings.HasPrefix(reqPath, "/api/") {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			// Serve the file directly if it exists (JS/CSS/assets/etc.),
			// otherwise fall back to index.html so react-router can handle
			// the route client-side.
			fullPath := filepath.Join(staticDir, filepath.Clean(reqPath))
			if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
			c.File(filepath.Join(staticDir, "index.html"))
		})
	}

	return r
}
