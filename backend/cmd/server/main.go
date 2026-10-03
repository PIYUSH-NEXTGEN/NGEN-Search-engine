package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/yourname/community-search/internal/api"
	"github.com/yourname/community-search/internal/cache"
	"github.com/yourname/community-search/internal/config"
	"github.com/yourname/community-search/internal/search"
	"github.com/yourname/community-search/internal/store"
	"github.com/yourname/community-search/internal/store/queries"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	pool, err := store.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database error: %v", err)
	}
	defer pool.Close()

	redisClient, err := cache.NewClient(ctx, cfg.RedisAddr, cfg.RedisDB)
	if err != nil {
		log.Fatalf("redis error: %v", err)
	}
	defer redisClient.Close()

	q := queries.New(pool)
	searchCache := cache.NewSearchCache(redisClient)
	searchService := search.NewService(q, searchCache)
	rateLimiter := cache.NewRateLimiter(redisClient, 60, time.Minute) // 60 req/min per IP

	router := api.NewRouter(api.Deps{
		SearchService: searchService,
		RateLimiter:   rateLimiter,
	})

	addr := ":" + cfg.Port
	log.Printf("listening on %s (env=%s)", addr, cfg.Env)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
