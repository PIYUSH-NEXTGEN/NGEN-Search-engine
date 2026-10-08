package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Answers cost a model call to produce, so they get their own longer-lived
// key space on top of the 15-minute search cache.
const answerCacheTTL = 30 * time.Minute

type AnswerCache struct {
	client *redis.Client
}

func NewAnswerCache(client *redis.Client) *AnswerCache {
	return &AnswerCache{client: client}
}

// answerKey uses its own "answer2:" prefix, bumped when the answer layer
// started grounding on the full standing community info — entries written
// under the old "answer:" key predate that change and must never be served.
func answerKey(query string) string {
	return "answer2:" + NormalizeQuery(query)
}

// Get returns the cached answer for a query, or (nil, false) on a cache miss.
// NormalizeQuery comes from search_cache.go, so both caches key the same way.
func (c *AnswerCache) Get(ctx context.Context, query string, dest any) (bool, error) {
	val, err := c.client.Get(ctx, answerKey(query)).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("redis get: %w", err)
	}
	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return false, fmt.Errorf("unmarshaling cached answer: %w", err)
	}
	return true, nil
}

// Set caches an answer for a query with a fixed TTL.
func (c *AnswerCache) Set(ctx context.Context, query string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshaling answer for cache: %w", err)
	}
	if err := c.client.Set(ctx, answerKey(query), data, answerCacheTTL).Err(); err != nil {
		return fmt.Errorf("redis set: %w", err)
	}
	return nil
}
