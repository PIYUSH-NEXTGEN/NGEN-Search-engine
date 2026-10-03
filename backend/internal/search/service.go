// Package search orchestrates the retrieval flow: normalize the query,
// check Redis, fall back to Postgres full-text search, populate the cache.
// This package knows nothing about HTTP or the LLM layer — it's a plain
// "query in, ranked members out" service, easy to test on its own.
package search

import (
	"context"
	"fmt"

	"github.com/yourname/community-search/internal/cache"
	"github.com/yourname/community-search/internal/store/queries"
)

type Result struct {
	ID       string  `json:"id"`
	FullName string  `json:"full_name"`
	Headline string  `json:"headline,omitempty"`
	Bio      string  `json:"bio,omitempty"`
	Location string  `json:"location,omitempty"`
	Rank     float32 `json:"rank"`
}

type Service struct {
	queries *queries.Queries
	cache   *cache.SearchCache
}

func NewService(q *queries.Queries, c *cache.SearchCache) *Service {
	return &Service{queries: q, cache: c}
}

const defaultResultLimit = 20

// Search returns ranked members matching the query, serving from Redis
// when available and falling back to Postgres full-text search otherwise.
func (s *Service) Search(ctx context.Context, query string) ([]Result, error) {
	if query == "" {
		return nil, fmt.Errorf("query must not be empty")
	}

	var cached []Result
	if hit, err := s.cache.Get(ctx, query, &cached); err == nil && hit {
		return cached, nil
	}

	rows, err := s.queries.SearchMembers(ctx, query, defaultResultLimit)
	if err != nil {
		return nil, fmt.Errorf("searching members: %w", err)
	}

	results := make([]Result, 0, len(rows))
	for _, r := range rows {
		results = append(results, Result{
			ID:       r.ID.String(),
			FullName: r.FullName,
			Headline: derefStr(r.Headline),
			Bio:      derefStr(r.Bio),
			Location: derefStr(r.Location),
			Rank:     r.Rank,
		})
	}

	// Cache write failures shouldn't fail the request — search still works
	// without caching, just slower. Log this in a real app.
	_ = s.cache.Set(ctx, query, results)

	return results, nil
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
