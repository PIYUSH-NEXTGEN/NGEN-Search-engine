package api

import (
	"encoding/json"
	"net/http"

	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/cache"
)

// rateLimitMiddleware limits requests per client IP so a single caller
// can't hammer the search/LLM endpoints. Swap the key source (e.g. a
// session cookie) once auth exists.
func rateLimitMiddleware(limiter *cache.RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// RealIP runs before this middleware (see router.go) and rewrites
			// RemoteAddr to the real client address, so RemoteAddr alone is
			// the rate-limit key — a request ID is not a client address.
			ip := r.RemoteAddr

			allowed, err := limiter.Allow(r.Context(), ip)
			if err != nil {
				// fail open: don't block search traffic because Redis hiccuped
				next.ServeHTTP(w, r)
				return
			}
			if !allowed {
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "rate limit exceeded, please slow down",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
