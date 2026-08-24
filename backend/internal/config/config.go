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
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
