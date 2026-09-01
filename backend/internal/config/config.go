// Package config loads application configuration from environment variables.
package config

import (
	"os"
)

// Config holds all runtime configuration for the backend.
type Config struct {
	Port              string
	DBDriver          string // "sqlite" | "postgres" | "mysql"
	DBDSN             string // sqlite file path, or postgres/mysql DSN
	LRCLIBBaseURL     string
	LibreTranslateURL string
	LibreTranslateKey string
	AllowedOrigin     string
	StaticDir         string // if set, backend also serves the built frontend (SPA) from this dir

	// TranslateProvider selects which MT backend handleTranslateTrack uses:
	// "localllm" (self-hosted OpenAI-compatible server, e.g. LM Studio —
	// needs LocalLLMURL, free/unlimited since it runs on your own hardware),
	// "gemini" (Google AI Studio — needs GeminiAPIKey, cloud LLM with a
	// free-tier rate limit/daily quota), or "libretranslate" (self-hosted/
	// free-but-literal plain NMT, see docker-compose.yml). localllm and
	// gemini both give much more natural/idiomatic output than libretranslate
	// since they're LLMs that can be steered by a prompt (e.g. "translate
	// like a song lyric, not word-for-word" — see internal/llmprompt).
	//
	// KISS priority (2026-08-31): left unset ("") it's resolved in main.go
	// to auto-prefer localllm when LocalLLMURL is set, else gemini when
	// GeminiAPIKey is present, else libretranslate — see main.go's
	// translator-construction switch. Set explicitly to force one provider
	// regardless of what's configured (e.g. to recover from a Gemini
	// daily-quota exhaustion, or a local server being unreachable, by forcing
	// libretranslate + restart).
	TranslateProvider string
	GeminiAPIKey      string
	GeminiModel       string

	// LocalLLMURL, if set, points at a self-hosted OpenAI-compatible chat
	// completions server (LM Studio, Ollama's OpenAI shim, etc.) — usually
	// reached over a tunnel (e.g. ngrok) since it isn't publicly routable on
	// its own. See internal/localllm and TranslateProvider's doc comment
	// above for how this fits into provider selection.
	LocalLLMURL    string
	LocalLLMAPIKey string // only needed if the server requires an Authorization: Bearer token (e.g. LM Studio's "require API key" setting)
	LocalLLMModel  string
}

// Load reads configuration from environment variables, applying sane defaults
// for local development. It does not fail on missing optional values.
func Load() Config {
	return Config{
		Port:              getEnv("PORT", "8080"),
		DBDriver:          getEnv("DB_DRIVER", "sqlite"),
		DBDSN:             getEnv("DB_DSN", "data/db.sqlite"),
		LRCLIBBaseURL:     getEnv("LRCLIB_BASE_URL", "https://lrclib.net/api"),
		LibreTranslateURL: getEnv("LIBRETRANSLATE_URL", "https://libretranslate.com"),
		LibreTranslateKey: getEnv("LIBRETRANSLATE_API_KEY", ""),
		AllowedOrigin:     getEnv("ALLOWED_ORIGIN", "http://localhost:5173"),
		StaticDir:         getEnv("STATIC_DIR", ""),
		TranslateProvider: getEnv("TRANSLATE_PROVIDER", ""),
		GeminiAPIKey:      getEnv("GEMINI_API_KEY", ""),
		GeminiModel:       getEnv("GEMINI_MODEL", "gemini-2.5-flash"),
		LocalLLMURL:       getEnv("LOCAL_LLM_URL", ""),
		LocalLLMAPIKey:    getEnv("LOCAL_LLM_API_KEY", ""),
		LocalLLMModel:     getEnv("LOCAL_LLM_MODEL", "Qwen3.6 35B A3B Uncensored HauhauCS Aggressive Q5 K P (modified)"),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
