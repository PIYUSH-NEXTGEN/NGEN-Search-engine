package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/cache"
	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/config"
	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/llm"
	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/search"
)

type Deps struct {
	SearchService *search.Service
	AnswerCache   *cache.AnswerCache
	LLMClient     *llm.Client
	RateLimiter   *cache.RateLimiter
	AllowedOrigin string // CORS: the single origin allowed to call this API
}

func NewRouter(deps Deps) http.Handler {
	r := chi.NewRouter()

	// A missing AllowedOrigins entry is not treated as "allow all" by
	// go-chi/cors — an empty string would match no browser origin at all, so
	// fall back to the dev default instead of silently breaking CORS.
	allowedOrigin := deps.AllowedOrigin
	if allowedOrigin == "" {
		allowedOrigin = config.DefaultAllowedOrigin
	}

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(10 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{allowedOrigin}, // set ALLOWED_ORIGIN per environment
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	r.Route("/api", func(api chi.Router) {
		api.Use(rateLimitMiddleware(deps.RateLimiter))
		api.Get("/search", searchHandler(deps.SearchService))
		// /api/ask sits inside the same route block, so it inherits the
		// rate limiter mounted above.
		api.Post("/ask", askHandler(deps.SearchService, deps.LLMClient, deps.AnswerCache))
	})

	return r
}
