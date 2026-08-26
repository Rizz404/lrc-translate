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

	// KISS priority (2026-08-26): Gemini (LLM, natural/idiomatic output) is
	// preferred over LibreTranslate (literal MT) whenever a key is actually
	// available; scrape+align is a separate, manual, last-resort feature
	// entirely outside this switch (see scrape_handler.go). This is a
	// startup-time default only, not a runtime fallback — if Gemini starts
	// failing mid-session (e.g. daily quota exhausted, see
	// docs/backend/fixes-2026-08-25-scrape-alignment.md point 8), recovery
	// is TRANSLATE_PROVIDER=libretranslate + restart, not an automatic retry
	// with the other provider.
	var translator httpapi.Translator
	// resolvedProvider is the *actual* provider picked, as opposed to
	// cfg.TranslateProvider which can be "" (auto). Passed to NewServer
	// below instead of the raw config value so the translation cache stays
	// correctly namespaced per real provider even when auto-resolved (see
	// Server.translatorID's doc comment in router.go) — otherwise two
	// servers auto-resolving to different providers (e.g. one with
	// GEMINI_API_KEY set, one without) would both use the "" cache
	// namespace and could serve each other's stale results.
	var resolvedProvider string
	switch cfg.TranslateProvider {
	case "gemini":
		if cfg.GeminiAPIKey == "" {
			log.Fatalf("TRANSLATE_PROVIDER=gemini but GEMINI_API_KEY is not set")
		}
		translator = gemini.New(cfg.GeminiAPIKey, cfg.GeminiModel)
		resolvedProvider = "gemini"
	case "libretranslate":
		translator = libretranslate.New(cfg.LibreTranslateURL, cfg.LibreTranslateKey)
		resolvedProvider = "libretranslate"
	case "":
		// Auto-resolve: prefer gemini when a key is present (doesn't fail
		// startup like the explicit "gemini" case above would without one),
		// otherwise fall back to libretranslate — same as before this
		// priority change when no key is configured.
		if cfg.GeminiAPIKey != "" {
			translator = gemini.New(cfg.GeminiAPIKey, cfg.GeminiModel)
			resolvedProvider = "gemini"
		} else {
			translator = libretranslate.New(cfg.LibreTranslateURL, cfg.LibreTranslateKey)
			resolvedProvider = "libretranslate"
		}
	default:
		log.Fatalf("unknown TRANSLATE_PROVIDER %q (expected \"libretranslate\" or \"gemini\")", cfg.TranslateProvider)
	}
	if cfg.TranslateProvider == "" {
		log.Printf("translate provider: %s (auto-resolved, TRANSLATE_PROVIDER unset)", resolvedProvider)
	} else {
		log.Printf("translate provider: %s", resolvedProvider)
	}

	log.Println("loading Japanese romanization dictionary…")
	romanizer, err := romanize.New()
	if err != nil {
		log.Fatalf("failed to init romanizer: %v", err)
	}

	server := httpapi.NewServer(gdb, lrclibClient, translator, resolvedProvider, romanizer)
	router := httpapi.NewRouter(server, cfg.AllowedOrigin, cfg.StaticDir)

	log.Printf("listening on :%s (db driver=%s dsn=%s)", cfg.Port, cfg.DBDriver, cfg.DBDSN)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
