// Command server runs the lrc-translate backend HTTP API.
package main

import (
	"log"

	"github.com/joho/godotenv"

	"lrc-translate/backend/internal/config"
	"lrc-translate/backend/internal/db"
	"lrc-translate/backend/internal/gemini"
	"lrc-translate/backend/internal/httpapi"
	"lrc-translate/backend/internal/libretranslate"
	"lrc-translate/backend/internal/lrclib"
	"lrc-translate/backend/internal/romanize"
)

func main() {
	// Load backend/.env into the process environment for local dev (see
	// .env.example) — config.Load() only ever reads real environment
	// variables (os.Getenv), so without this step a .env file sitting right
	// next to the binary was silently ignored, e.g. TRANSLATE_PROVIDER=gemini
	// in .env never took effect and every MT call quietly fell back to
	// config.Load()'s "libretranslate" default instead. godotenv.Load()
	// never overrides a variable that's already set in the real environment
	// (that always wins), and a missing .env — the normal case in
	// production/Docker, where real env vars are injected directly — is not
	// an error worth failing startup over, so its return value is ignored.
	_ = godotenv.Load()

	cfg := config.Load()

	gdb, err := db.Open(cfg.DBDriver, cfg.DBDSN)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	lrclibClient := lrclib.New(cfg.LRCLIBBaseURL)

	var translator httpapi.Translator
	switch cfg.TranslateProvider {
	case "gemini":
		if cfg.GeminiAPIKey == "" {
			log.Fatalf("TRANSLATE_PROVIDER=gemini but GEMINI_API_KEY is not set")
		}
		translator = gemini.New(cfg.GeminiAPIKey, cfg.GeminiModel)
	case "libretranslate", "":
		translator = libretranslate.New(cfg.LibreTranslateURL, cfg.LibreTranslateKey)
	default:
		log.Fatalf("unknown TRANSLATE_PROVIDER %q (expected \"libretranslate\" or \"gemini\")", cfg.TranslateProvider)
	}

	log.Println("loading Japanese romanization dictionary…")
	romanizer, err := romanize.New()
	if err != nil {
		log.Fatalf("failed to init romanizer: %v", err)
	}

	server := httpapi.NewServer(gdb, lrclibClient, translator, cfg.TranslateProvider, romanizer)
	router := httpapi.NewRouter(server, cfg.AllowedOrigin, cfg.StaticDir)

	log.Printf("listening on :%s (db driver=%s dsn=%s)", cfg.Port, cfg.DBDriver, cfg.DBDSN)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
