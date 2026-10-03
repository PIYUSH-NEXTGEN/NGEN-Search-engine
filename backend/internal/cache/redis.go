// Package cache wraps Redis: search-result caching, rate limiting, and
// (in later phases) generated-answer caching and popular-query tracking.
package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func NewClient(ctx context.Context, addr string, db int) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   db,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("connecting to redis: %w", err)
	}

	return client, nil
}
