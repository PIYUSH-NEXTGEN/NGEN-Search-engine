package search

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/store/queries"
)

func TestSearchAllEmptyQuery(t *testing.T) {
	svc := &Service{queries: &fakeQuerier{}, cache: &fakeCacher{}}
	if _, err := svc.SearchAll(context.Background(), ""); err == nil {
		t.Error("want error for empty query, got nil")
	}
}

func TestStandingInfoLoadsAll(t *testing.T) {
	q := &fakeQuerier{
		standing: []queries.CommunityInfo{
			{ID: 3, Slug: "what-you-can-do", Title: "What you can do", Body: "Find people."},
			{ID: 1, Slug: "about", Title: "About", Body: "A community."},
			{ID: 2, Slug: "mission", Title: "Mission", Body: "Connect."},
		},
	}
	c := &fakeCacher{}
	svc := &Service{queries: q, cache: c}

	got, err := svc.StandingInfo(context.Background())
	if err != nil {
		t.Fatalf("StandingInfo: %v", err)
	}
	// Order comes from ListCommunityInfo's ORDER BY slug at the SQL level;
	// the fake returns rows as seeded, so assert content rather than order.
	seen := map[string]bool{}
	for _, ci := range got {
		seen[ci.Slug] = true
	}
	for _, want := range []string{"about", "mission", "what-you-can-do"} {
		if !seen[want] {
			t.Errorf("standing info missing slug %q (got %+v)", want, got)
		}
	}
	if !c.gotStanding || c.setStandingVal == nil || len(*c.setStandingVal) != 3 {
		t.Errorf("standing list not cached: checked=%v written=%v", c.gotStanding, c.setStandingVal)
	}
}

func TestStandingInfoError(t *testing.T) {
	q := &fakeQuerier{standingErr: errors.New("db down")}
	svc := &Service{queries: q, cache: &fakeCacher{}}
	if _, err := svc.StandingInfo(context.Background()); err == nil {
		t.Error("want error when ListCommunityInfo fails, got nil")
	}
}

func TestStandingInfoCacheHitSkipsPostgres(t *testing.T) {
	cached := []CommunityInfo{{Slug: "about", Title: "About", Body: "From cache."}}
	q := &fakeQuerier{standingErr: errors.New("must not be reached")}
	c := &fakeCacher{standingHit: cached}
	svc := &Service{queries: q, cache: c}

	got, err := svc.StandingInfo(context.Background())
	if err != nil {
		t.Fatalf("StandingInfo: %v", err)
	}
	if len(got) != 1 || got[0].Body != "From cache." {
		t.Errorf("got %+v, want the cached row", got)
	}
	if q.listCalls != 0 {
		t.Errorf("ListCommunityInfo called %d times on a cache hit, want 0", q.listCalls)
	}
}

func TestStandingInfoRedisErrorFallsBackToPostgres(t *testing.T) {
	q := &fakeQuerier{standing: []queries.CommunityInfo{
		{ID: 1, Slug: "about", Title: "About", Body: "From the database."},
	}}
	c := &fakeCacher{standingCacheErr: errors.New("redis down")}
	svc := &Service{queries: q, cache: c}

	got, err := svc.StandingInfo(context.Background())
	if err != nil {
		t.Fatalf("StandingInfo with a Redis failure must fall back to Postgres, got: %v", err)
	}
	if len(got) != 1 || got[0].Body != "From the database." {
		t.Errorf("got %+v, want the Postgres row", got)
	}
	if q.listCalls != 1 {
		t.Errorf("ListCommunityInfo called %d times, want 1 (the fallback)", q.listCalls)
	}
}

// errQuerier fails the member lookup on demand, proving SearchAll surfaces
// the failing query in the error rather than returning partial groups.
type errQuerier struct {
	fakeQuerier
	failOn string
}

func (e *errQuerier) SearchMembers(ctx context.Context, q string, l int32) ([]queries.SearchMembersRow, error) {
	if e.failOn == "members" {
		return nil, errors.New("members down")
	}
	return e.fakeQuerier.SearchMembers(ctx, q, l)
}

func TestSearchAllSurfacesTableErrors(t *testing.T) {
	svc := &Service{queries: &errQuerier{failOn: "members"}, cache: &fakeCacher{}}
	_, err := svc.SearchAll(context.Background(), "ml")
	if err == nil || !strings.Contains(err.Error(), "members") {
		t.Errorf("error = %v, want it to name the failing table", err)
	}
}
