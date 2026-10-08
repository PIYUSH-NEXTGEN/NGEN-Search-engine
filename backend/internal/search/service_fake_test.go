package search

import (
	"context"

	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/store/queries"
)

// fakeQuerier is the storeQuerier behind fakes: each test seeds exactly the
// rows it needs, so matching behavior is deterministic with no DB. It has no
// SearchCommunityInfo method on purpose — SearchAll must never query
// community_info, so the interface (and this fake) can't offer it.
type fakeQuerier struct {
	members map[string][]queries.SearchMembersRow

	standing    []queries.CommunityInfo
	standingErr error

	// listCalls counts ListCommunityInfo invocations, so cache tests can
	// prove Postgres was (or wasn't) reached.
	listCalls int

	// gotMembersLimit records the cap SearchAll asked for.
	gotMembersLimit int32
}

func (f *fakeQuerier) SearchMembers(_ context.Context, query string, limit int32) ([]queries.SearchMembersRow, error) {
	f.gotMembersLimit = limit
	return f.members[query], nil
}

func (f *fakeQuerier) ListCommunityInfo(_ context.Context) ([]queries.CommunityInfo, error) {
	f.listCalls++
	return f.standing, f.standingErr
}

// fakeCacher records SearchAll cache traffic; by default it never reports a
// hit, so tests exercise the DB path and assert what got written. The
// standing fields flip that for the community-info list: set standingHit to
// simulate a Redis hit, or standingCacheErr to simulate a Redis failure.
type fakeCacher struct {
	setAllQuery string
	setAllValue *AllResults

	gotStanding    bool
	setStandingVal *[]CommunityInfo

	standingHit      []CommunityInfo
	standingCacheErr error
}

func (f *fakeCacher) Get(_ context.Context, _ string, _ any) (bool, error) {
	return false, nil
}

func (f *fakeCacher) Set(_ context.Context, _ string, _ any) error { return nil }

func (f *fakeCacher) GetAll(_ context.Context, _ string, _ any) (bool, error) {
	return false, nil
}

func (f *fakeCacher) SetAll(_ context.Context, query string, value any) error {
	f.setAllQuery = query
	if v, ok := value.(AllResults); ok {
		f.setAllValue = &v
	}
	return nil
}

func (f *fakeCacher) GetCommunityInfo(_ context.Context, dest any) (bool, error) {
	f.gotStanding = true
	if f.standingCacheErr != nil {
		return false, f.standingCacheErr
	}
	if f.standingHit == nil {
		return false, nil
	}
	if d, ok := dest.(*[]CommunityInfo); ok {
		*d = f.standingHit
	}
	return true, nil
}

func (f *fakeCacher) SetCommunityInfo(_ context.Context, value any) error {
	if v, ok := value.([]CommunityInfo); ok {
		f.setStandingVal = &v
	}
	return nil
}

func strp(s string) *string { return &s }
