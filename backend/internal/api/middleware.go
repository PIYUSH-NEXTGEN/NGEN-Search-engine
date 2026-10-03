package api

import (
	"encoding/json"
	"net"
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
			// RemoteAddr to the real client address when forwarding headers are
			// present. Strip the TCP port so the key is the IP alone: without a
			// proxy, RemoteAddr keeps the ephemeral source port, which would
			// give every connection its own rate-limit bucket.
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				// Not in "host:port" form — use it verbatim rather than
				// dropping the request on the floor.
				ip = r.RemoteAddr
			}

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
