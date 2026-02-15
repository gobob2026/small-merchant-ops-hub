package main

import (
	"log"

	"small-merchant-ops-hub-server/internal/cache"
	"small-merchant-ops-hub-server/internal/config"
	"small-merchant-ops-hub-server/internal/db"
	httpapi "small-merchant-ops-hub-server/internal/http"
)

func main() {
	cfg := config.LoadFromEnv()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid config: %v", err)
	}

	database, err := db.Open(cfg)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}

	cacheStore, err := cache.New(cfg)
	if err != nil {
		log.Fatalf("create cache: %v", err)
	}
	defer func() {
		if err := cacheStore.Close(); err != nil {
			log.Printf("close cache: %v", err)
		}
	}()

	router := httpapi.NewRouter(database, cacheStore, cfg)
	addr := ":" + cfg.Port
	log.Printf("starting server on %s (env=%s, db=%s, cache=%s)", addr, cfg.Env, cfg.DatabaseDriver(), cfg.CacheMode)
	if err := router.Run(addr); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
