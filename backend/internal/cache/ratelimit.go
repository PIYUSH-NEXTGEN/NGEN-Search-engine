package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	client *redis.Client
	limit  int64
	window time.Duration
}

// NewRateLimiter creates a fixed-window limiter: at most `limit` requests
// per `window` for a given key (typically an IP address or session ID).
func NewRateLimiter(client *redis.Client, limit int64, window time.Duration) *RateLimiter {
	return &RateLimiter{client: client, limit: limit, window: window}
}

// Allow increments the counter for key and reports whether the caller is
// still within the limit for the current window.
func (r *RateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	rlKey := "ratelimit:" + key

	count, err := r.client.Incr(ctx, rlKey).Result()
	if err != nil {
		return false, fmt.Errorf("redis incr: %w", err)
	}

	if count == 1 {
		// first request in this window — set the expiry
		if err := r.client.Expire(ctx, rlKey, r.window).Err(); err != nil {
			return false, fmt.Errorf("redis expire: %w", err)
		}
	}

	return count <= r.limit, nil
}
