// Package httpapi wires Gin routes to the app's handlers.
package httpapi

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"lrc-translate/backend/internal/lrclib"
	"lrc-translate/backend/internal/romanize"
)

// Translator is implemented by any MT backend (see internal/libretranslate,
// internal/gemini, internal/localllm) — lets Server swap providers via
// config without the rest of the package caring which one is active.
type Translator interface {
	// Translate translates one line in isolation — used by
	// handleGetAIReference (ai_reference_handler.go), which needs a
	// same-language reference for individual lines and caches per line
	// (translateOneCached), not a whole-track batch.
	Translate(ctx context.Context, text, sourceLang, targetLang string) (string, error)
	// TranslateBatch translates every line together in one call, in order —
	// used by handleTranslateTrack so an LLM backend (gemini/localllm) can
	// use the whole song as context instead of guessing at each line blind
	// to its neighbors (see llmprompt.BuildBatch). LibreTranslate has no
	// such context to use, but still implements this against its own native
	// batch "q" array support, so callers don't need provider-specific
	// branching.
	TranslateBatch(ctx context.Context, lines []string, sourceLang, targetLang string) ([]string, error)
}

// Server holds the dependencies shared by all HTTP handlers.
type Server struct {
	db         *gorm.DB
	lrclib     *lrclib.Client
	translator Translator
	// translatorID is e.g. "libretranslate"/"gemini"/"localllm" — namespaces
	// the translation cache (see translationCacheKey) so switching providers
	// doesn't serve stale results from a different engine, and is exposed
	// via GET /api/health so the frontend knows what's active.
	translatorID string
	// translatorIsLLM is true for gemini/localllm (steered by
	// internal/llmprompt), false for libretranslate (plain NMT) — lets
	// handleTranslateTrack tag a line db.MethodAI vs db.MethodMT. See
	// resolvedIsLLM in cmd/server/main.go for how this is decided.
	translatorIsLLM bool
	romanizer       *romanize.Romanizer
}

// NewServer builds a Server with its dependencies. translatorID/
// translatorIsLLM describe the Translator passed in — see their doc
// comments on Server.
func NewServer(db *gorm.DB, lrclibClient *lrclib.Client, translator Translator, translatorID string, translatorIsLLM bool, romanizer *romanize.Romanizer) *Server {
	return &Server{
		db:              db,
		lrclib:          lrclibClient,
		translator:      translator,
		translatorID:    translatorID,
		translatorIsLLM: translatorIsLLM,
		romanizer:       romanizer,
	}
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
		// KISS 2026-08-26: route disabled, not deleted — see
		// docs/backend/fixes-2026-08-25-scrape-alignment.md point 6 for what
		// this endpoint does. With translate now prioritizing Gemini/
		// LibreTranslate over scrape (main.go), and the frontend's
		// "Bandingkan dengan AI" trigger commented out (EditorPage.tsx),
		// this handler (ai_reference_handler.go, left intact) has no caller
		// left; keeping it registered would leave a dead-but-reachable API
		// surface. Uncomment this line to re-enable.
		// api.POST("/tracks/:id/ai-reference", s.handleGetAIReference)
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
