package search

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/store/queries"
)

func TestSearchAllMembersOnlyMatch(t *testing.T) {
	q := &fakeQuerier{
		members: map[string][]queries.SearchMembersRow{
			"ml": {{
				ID: uuid.New(), FullName: "Asha Rao",
				Headline: strp("ML engineer"), Bio: strp("Builds recommenders"),
				Location: strp("Bengaluru"), Rank: 0.9,
			}},
		},
	}
	c := &fakeCacher{}
	svc := &Service{queries: q, cache: c}

	got, err := svc.SearchAll(context.Background(), "ml")
	if err != nil {
		t.Fatalf("SearchAll: %v", err)
	}
	if len(got.Members) != 1 || got.Members[0].FullName != "Asha Rao" {
		t.Errorf("members = %+v, want the one matching member", got.Members)
	}
	if q.gotMembersLimit != membersLimit {
		t.Errorf("members limit = %d, want %d", q.gotMembersLimit, membersLimit)
	}
	if c.setAllQuery != "ml" || c.setAllValue == nil || len(c.setAllValue.Members) != 1 {
		t.Errorf("SearchAll result not cached: query=%q value=%+v", c.setAllQuery, c.setAllValue)
	}
}

// TestSearchAllNeverSearchesCommunityInfo pins the single-mechanism rule: a
// query that would match community_info rows ("mission" appears in the
// Our Mission body) must return an empty member group with no error, because
// SearchAll only queries members — standing context flows to the prompt
// exclusively through StandingInfo.
func TestSearchAllNeverSearchesCommunityInfo(t *testing.T) {
	q := &fakeQuerier{} // no rows at all, member or info
	svc := &Service{queries: q, cache: &fakeCacher{}}

	got, err := svc.SearchAll(context.Background(), "mission")
	if err != nil {
		t.Fatalf("SearchAll: %v", err)
	}
	if len(got.Members) != 0 {
		t.Errorf("members = %+v, want empty", got.Members)
	}
	if q.listCalls != 0 {
		t.Errorf("ListCommunityInfo called %d times from SearchAll, want 0 (standing context is ask-only)", q.listCalls)
	}
}

func TestSearchAllNoMatchesReturnsEmptyGroups(t *testing.T) {
	svc := &Service{queries: &fakeQuerier{}, cache: &fakeCacher{}}

	got, err := svc.SearchAll(context.Background(), "zzz-no-such-thing")
	if err != nil {
		t.Fatalf("SearchAll: %v", err)
	}
	if got.Members == nil {
		t.Errorf("members group must be a non-nil empty slice, got %+v", got)
	}
	if len(got.Members) != 0 {
		t.Errorf("members = %+v, want empty", got.Members)
	}
}
