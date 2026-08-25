// Package config loads application configuration from environment variables.
package config

import (
	"os"
)

// Config holds all runtime configuration for the backend.
type Config struct {
	Port               string
	DBDriver           string // "sqlite" | "postgres" | "mysql"
	DBDSN              string // sqlite file path, or postgres/mysql DSN
	LRCLIBBaseURL      string
	LibreTranslateURL  string
	LibreTranslateKey  string
	AllowedOrigin      string
	StaticDir          string // if set, backend also serves the built frontend (SPA) from this dir

	// TranslateProvider selects which MT backend handleTranslateTrack uses:
	// "libretranslate" (self-hosted/free-but-literal, see docker-compose.yml)
	// or "gemini" (Google AI Studio — needs GeminiAPIKey, gives much more
	// natural/idiomatic output since it's an LLM that can be steered by a
	// prompt, e.g. "translate like a song lyric, not word-for-word").
	TranslateProvider string
	GeminiAPIKey      string
	GeminiModel       string
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
		TranslateProvider: getEnv("TRANSLATE_PROVIDER", "libretranslate"),
		GeminiAPIKey:      getEnv("GEMINI_API_KEY", ""),
		GeminiModel:       getEnv("GEMINI_MODEL", "gemini-2.5-flash"),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
