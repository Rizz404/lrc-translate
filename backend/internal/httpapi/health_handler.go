package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// handleHealth also reports the active translate provider (s.translatorID,
// e.g. "localllm"/"gemini"/"libretranslate") and its actual model id
// (s.translatorModel, empty for libretranslate) so the frontend can tailor
// its translate-in-progress messaging — a self-hosted LLM (see
// internal/localllm) genuinely runs much longer than cloud NMT, and a user
// staring at a spinner with no idea *why* it's slow is far more likely to
// assume it's hung. translate_model is included specifically so the UI
// shows the real configured model instead of the bare provider id —
// "localllm" on its own is an internal implementation detail, not something
// that reads well surfaced directly to whoever's actually using the app.
// See TranslatePanel.tsx.
func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":             "ok",
		"translate_provider": s.translatorID,
		"translate_model":    s.translatorModel,
	})
}
