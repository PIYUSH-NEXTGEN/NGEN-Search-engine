package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const searchCacheTTL = 15 * time.Minute

type SearchCache struct {
	client *redis.Client
}

func NewSearchCache(client *redis.Client) *SearchCache {
	return &SearchCache{client: client}
}

// NormalizeQuery is used everywhere a raw user query becomes a cache key,
// so "Machine Learning", " machine learning ", and "MACHINE LEARNING"
// all hit the same cache entry.
func NormalizeQuery(q string) string {
	return strings.ToLower(strings.TrimSpace(q))
}

func searchKey(query string) string {
	return "search:" + NormalizeQuery(query)
}

// Get returns cached results for a query, or (nil, false) on a cache miss.
func (c *SearchCache) Get(ctx context.Context, query string, dest any) (bool, error) {
	val, err := c.client.Get(ctx, searchKey(query)).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("redis get: %w", err)
	}
	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return false, fmt.Errorf("unmarshaling cached search result: %w", err)
	}
	return true, nil
}

// Set caches results for a query with a fixed TTL.
func (c *SearchCache) Set(ctx context.Context, query string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshaling search result for cache: %w", err)
	}
	if err := c.client.Set(ctx, searchKey(query), data, searchCacheTTL).Err(); err != nil {
		return fmt.Errorf("redis set: %w", err)
	}
	return nil
}
