// Command server runs the lrc-translate backend HTTP API.
package main

import (
	"log"

	"lrc-translate/backend/internal/config"
	"lrc-translate/backend/internal/db"
	"lrc-translate/backend/internal/httpapi"
	"lrc-translate/backend/internal/lrclib"
)

func main() {
	cfg := config.Load()

	gdb, err := db.Open(cfg.DBDriver, cfg.DBDSN)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	lrclibClient := lrclib.New(cfg.LRCLIBBaseURL)
	server := httpapi.NewServer(gdb, lrclibClient)
	router := httpapi.NewRouter(server, cfg.AllowedOrigin)

	log.Printf("listening on :%s (db driver=%s dsn=%s)", cfg.Port, cfg.DBDriver, cfg.DBDSN)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
