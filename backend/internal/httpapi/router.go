// Package httpapi wires Gin routes to the app's handlers.
package httpapi

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"lrc-translate/backend/internal/lrclib"
)

// Server holds the dependencies shared by all HTTP handlers.
type Server struct {
	db     *gorm.DB
	lrclib *lrclib.Client
}

// NewServer builds a Server with its dependencies.
func NewServer(db *gorm.DB, lrclibClient *lrclib.Client) *Server {
	return &Server{db: db, lrclib: lrclibClient}
}

// NewRouter builds the Gin engine with CORS (restricted to allowedOrigin) and
// all routes registered.
func NewRouter(s *Server, allowedOrigin string) *gin.Engine {
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
	}

	return r
}
