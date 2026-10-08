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

// communityInfoCacheTTL is deliberately short: community_info is tiny and
// read on every ask as standing prompt context, so a brief cache avoids a
// Postgres round trip per request without going stale for long.
const communityInfoCacheTTL = 5 * time.Minute

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

// GetAll returns cached SearchAll results for a query, or false on a miss.
// It uses its own "searchall:" prefix so entries can never collide with the
// older "search:" members-only shape.
func (c *SearchCache) GetAll(ctx context.Context, query string, dest any) (bool, error) {
	val, err := c.client.Get(ctx, searchAllKey(query)).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("redis get: %w", err)
	}
	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return false, fmt.Errorf("unmarshaling cached searchall result: %w", err)
	}
	return true, nil
}

// SetAll caches SearchAll results for a query with the same TTL as Search.
func (c *SearchCache) SetAll(ctx context.Context, query string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshaling searchall result for cache: %w", err)
	}
	if err := c.client.Set(ctx, searchAllKey(query), data, searchCacheTTL).Err(); err != nil {
		return fmt.Errorf("redis set: %w", err)
	}
	return nil
}

func searchAllKey(query string) string {
	return "searchall:" + NormalizeQuery(query)
}

func communityInfoKey() string {
	return "communityinfo:all"
}

// GetCommunityInfo returns the cached standing community_info list, or false
// on a miss. The key is fixed — the list is the same for every query.
func (c *SearchCache) GetCommunityInfo(ctx context.Context, dest any) (bool, error) {
	val, err := c.client.Get(ctx, communityInfoKey()).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("redis get: %w", err)
	}
	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return false, fmt.Errorf("unmarshaling cached community info: %w", err)
	}
	return true, nil
}

// SetCommunityInfo caches the full community_info list briefly so every ask
// doesn't hit Postgres.
func (c *SearchCache) SetCommunityInfo(ctx context.Context, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshaling community info for cache: %w", err)
	}
	if err := c.client.Set(ctx, communityInfoKey(), data, communityInfoCacheTTL).Err(); err != nil {
		return fmt.Errorf("redis set: %w", err)
	}
	return nil
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
