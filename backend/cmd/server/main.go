package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/api"
	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/cache"
	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/config"
	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/llm"
	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/search"
	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/session"
	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/store"
	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/store/queries"
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
	answerCache := cache.NewAnswerCache(redisClient)
	sessionStore := session.NewStore(redisClient)
	rateLimiter := cache.NewRateLimiter(redisClient, 60, time.Minute) // 60 req/min per IP
	llmClient := llm.NewClient(cfg.LLMAPIKey, cfg.LLMModel)
	if cfg.LLMAPIKey == "" {
		// Startup keeps going: search is unaffected, only /api/ask fails.
		log.Printf("warning: LLM_API_KEY is unset — POST /api/ask will fail until it is set")
	}

	router := api.NewRouter(api.Deps{
		SearchService: searchService,
		AnswerCache:   answerCache,
		SessionStore:  sessionStore,
		LLMClient:     llmClient,
		RateLimiter:   rateLimiter,
		AllowedOrigin: cfg.AllowedOrigin,
	})

	addr := ":" + cfg.Port
	log.Printf("listening on %s (env=%s)", addr, cfg.Env)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
