// Package search orchestrates the retrieval flow: normalize the query,
// check Redis, fall back to Postgres full-text search, populate the cache.
// This package knows nothing about HTTP or the LLM layer — it's plain
// "query in, ranked groups out", easy to test on its own.
//
// Scores from different tables are never merged into one ranked list: each
// group is ranked internally by its own ts_rank, so SearchAll returns the
// groups side by side instead of interleaving them.
package search

import (
	"context"
	"fmt"

	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/cache"
	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/store/queries"
)

type Result struct {
	ID       string  `json:"id"`
	FullName string  `json:"full_name"`
	Headline string  `json:"headline,omitempty"`
	Bio      string  `json:"bio,omitempty"`
	Location string  `json:"location,omitempty"`
	Rank     float32 `json:"rank"`
}

// AllResults is the grouped outcome of SearchAll. Members is ranked by its
// own table's ts_rank; any further groups must be ranked independently and
// never merged into one list, because ranks from different tables aren't
// comparable.
type AllResults struct {
	Members []Result `json:"members"`
}

// CommunityInfo is the standing context loaded on every ask: the full
// community_info table (it's tiny), regardless of what search matched.
type CommunityInfo struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

type storeQuerier interface {
	SearchMembers(ctx context.Context, query string, resultLimit int32) ([]queries.SearchMembersRow, error)
	ListCommunityInfo(ctx context.Context) ([]queries.CommunityInfo, error)
}

type searchCacher interface {
	Get(ctx context.Context, query string, dest any) (bool, error)
	Set(ctx context.Context, query string, value any) error
	GetAll(ctx context.Context, query string, dest any) (bool, error)
	SetAll(ctx context.Context, query string, value any) error
	GetCommunityInfo(ctx context.Context, dest any) (bool, error)
	SetCommunityInfo(ctx context.Context, value any) error
}

type Service struct {
	queries storeQuerier
	cache   searchCacher
}

func NewService(q *queries.Queries, c *cache.SearchCache) *Service {
	var cc searchCacher
	if c != nil {
		cc = c
	}
	return &Service{queries: q, cache: cc}
}

const defaultResultLimit = 20

// membersLimit caps SearchAll's members group. There is deliberately no
// per-query community-info limit: SearchAll never touches community_info —
// StandingInfo feeds every prompt the same full list.
const membersLimit = 5

// Search returns ranked members matching the query, serving from Redis
// when available and falling back to Postgres full-text search otherwise.
// It stays members-only so GET /api/search keeps its exact shape.
func (s *Service) Search(ctx context.Context, query string) ([]Result, error) {
	if query == "" {
		return nil, fmt.Errorf("query must not be empty")
	}

	if s.cache != nil {
		var cached []Result
		if hit, err := s.cache.Get(ctx, query, &cached); err == nil && hit {
			return cached, nil
		}
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
	if s.cache != nil {
		_ = s.cache.Set(ctx, query, results)
	}

	return results, nil
}

// SearchAll runs the members full-text lookup and returns the top matches.
// Community info is intentionally not searched here: a per-query slice of
// community_info would be a second path to data StandingInfo already provides
// in full, so members are the only group retrieved. Results are cached under
// a dedicated "searchall:" prefix so they can't collide with the members-only
// entries.
func (s *Service) SearchAll(ctx context.Context, query string) (AllResults, error) {
	if query == "" {
		return AllResults{}, fmt.Errorf("query must not be empty")
	}

	if s.cache != nil {
		var cached AllResults
		if hit, err := s.cache.GetAll(ctx, query, &cached); err == nil && hit {
			return cached, nil
		}
	}

	memberRows, err := s.queries.SearchMembers(ctx, query, membersLimit)
	if err != nil {
		return AllResults{}, fmt.Errorf("searching members: %w", err)
	}

	members := make([]Result, 0, len(memberRows))
	for _, r := range memberRows {
		members = append(members, Result{
			ID:       r.ID.String(),
			FullName: r.FullName,
			Headline: derefStr(r.Headline),
			Bio:      derefStr(r.Bio),
			Location: derefStr(r.Location),
			Rank:     r.Rank,
		})
	}

	out := AllResults{Members: members}
	if out.Members == nil {
		out.Members = []Result{}
	}

	if s.cache != nil {
		_ = s.cache.SetAll(ctx, query, out)
	}

	return out, nil
}

// StandingInfo loads every community_info row (ordered by slug) as the
// standing prompt context, cached briefly so it doesn't hit Postgres on
// every ask. The table is tiny (about, mission, what-you-can-do), so the
// full list — not a query-filtered slice — goes into every prompt.
func (s *Service) StandingInfo(ctx context.Context) ([]CommunityInfo, error) {
	if s.cache != nil {
		var cached []CommunityInfo
		if hit, err := s.cache.GetCommunityInfo(ctx, &cached); err == nil && hit {
			return cached, nil
		}
	}

	rows, err := s.queries.ListCommunityInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing community info: %w", err)
	}

	out := make([]CommunityInfo, 0, len(rows))
	for _, r := range rows {
		out = append(out, CommunityInfo{Slug: r.Slug, Title: r.Title, Body: r.Body})
	}
	if out == nil {
		out = []CommunityInfo{}
	}

	if s.cache != nil {
		_ = s.cache.SetCommunityInfo(ctx, out)
	}

	return out, nil
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
