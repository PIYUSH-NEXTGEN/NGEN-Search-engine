package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/llm"
	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/search"
	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/session"
)

// fakeSearcher returns fixed groups and standing info so the handler test
// never touches Postgres or Redis.
type fakeSearcher struct {
	all      search.AllResults
	standing []search.CommunityInfo
	err      error
}

func (f *fakeSearcher) SearchAll(_ context.Context, _ string) (search.AllResults, error) {
	return f.all, f.err
}

func (f *fakeSearcher) StandingInfo(_ context.Context) ([]search.CommunityInfo, error) {
	return f.standing, nil
}

// fakeAsker stands in for the LLM client: no tokens, deterministic answer,
// and it records what retrieval it was given.
type fakeAsker struct {
	gotResults search.AllResults
	gotInfo    []llm.Info
	gotHistory []llm.Turn
	askCalls   int
	answer     llm.AskResult
}

func (f *fakeAsker) Ask(_ context.Context, _ string, results search.AllResults, info []llm.Info, history []llm.Turn) (llm.AskResult, error) {
	f.askCalls++
	f.gotResults = results
	f.gotInfo = info
	f.gotHistory = history
	if f.answer != (llm.AskResult{}) {
		return f.answer, nil
	}
	return llm.AskResult{Relevant: true, Answer: "Asha Rao works on machine learning."}, nil
}

type fakeAnswerCache struct{}

func (fakeAnswerCache) Get(_ context.Context, _ string, _ any) (bool, error) {
	return false, nil
}

func (fakeAnswerCache) Set(_ context.Context, _ string, _ any) error { return nil }

type fakeSessions struct{}

func (fakeSessions) Get(_ context.Context, _ string) (*session.Session, bool, error) {
	return nil, false, nil
}

func (fakeSessions) Save(_ context.Context, _ string, _ *session.Session) error { return nil }

func TestAskHandlerReturnsMembersGroupOnly(t *testing.T) {
	svc := &fakeSearcher{
		all: search.AllResults{
			Members: []search.Result{{ID: "m1", FullName: "Asha Rao"}},
		},
		standing: []search.CommunityInfo{{Slug: "about", Title: "About", Body: "A community."}},
	}
	asker := &fakeAsker{}

	h := askHandler(svc, asker, fakeAnswerCache{}, fakeSessions{})
	req := httptest.NewRequest(http.MethodPost, "/api/ask", strings.NewReader(`{"query":"machine learning"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	raw, ok := body["results"]
	if !ok {
		t.Fatalf("response missing \"results\" key (body: %s)", rec.Body.String())
	}
	if arr, ok := raw.([]any); !ok || len(arr) != 1 {
		t.Errorf("results = %v, want one-element array", raw)
	}
	// "results" stays the members array so the current frontend works.
	if arr, _ := body["results"].([]any); len(arr) == 1 {
		if m, _ := arr[0].(map[string]any); m["full_name"] != "Asha Rao" {
			t.Errorf("results[0] = %v, want the member", arr[0])
		}
	}
	// Standing context reaches the LLM through StandingInfo only, so the
	// response must not carry a search-derived "info" group anymore.
	if _, ok := body["info"]; ok {
		t.Errorf("response must not carry an \"info\" key; standing context is the single info path (body: %s)", rec.Body.String())
	}
	// The fake LLM saw the member group plus standing info, no history.
	if _, ok := body["projects"]; ok {
		t.Errorf("response must not carry a %q key; projects were removed", "projects")
	}
	if len(asker.gotResults.Members) != 1 {
		t.Errorf("asker got results %+v, want one member", asker.gotResults)
	}
	if len(asker.gotInfo) != 1 || asker.gotInfo[0].Slug != "about" {
		t.Errorf("asker got info %+v, want the about row", asker.gotInfo)
	}
	if len(asker.gotHistory) != 0 {
		t.Errorf("asker got history %+v, want none for a fresh ask", asker.gotHistory)
	}
}

// TestAskHandlerPassesInfoWithNoMembers covers the "how do I join" case:
// full-text search matches no members, but the standing community info must
// still reach the model so it can answer from the rules and invite data.
func TestAskHandlerPassesInfoWithNoMembers(t *testing.T) {
	svc := &fakeSearcher{
		all: search.AllResults{}, // zero members — only standing info feeds the prompt
		standing: []search.CommunityInfo{
			{Slug: "how-to-join", Title: "How to Join", Body: "Join on Discord: https://discord.gg/AUz7KqDrnf"},
			{Slug: "rules", Title: "Community Rules", Body: "No spamming."},
		},
	}
	asker := &fakeAsker{answer: llm.AskResult{Relevant: true, Answer: "Join on Discord: https://discord.gg/AUz7KqDrnf"}}

	h := askHandler(svc, asker, fakeAnswerCache{}, fakeSessions{})
	req := httptest.NewRequest(http.MethodPost, "/api/ask", strings.NewReader(`{"query":"how do I join"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	if asker.askCalls != 1 {
		t.Fatalf("asker was called %d times, want 1 — a cache hit must not swallow this ask", asker.askCalls)
	}
	if len(asker.gotInfo) != 2 {
		t.Fatalf("asker got info %+v, want both standing rows even with no members", asker.gotInfo)
	}
	if asker.gotInfo[0].Slug != "how-to-join" || !strings.Contains(asker.gotInfo[0].Body, "https://discord.gg/AUz7KqDrnf") {
		t.Errorf("asker got info %+v, want the how-to-join row with the invite link", asker.gotInfo[0])
	}
	if len(asker.gotResults.Members) != 0 {
		t.Errorf("asker got members %+v, want none for this query", asker.gotResults.Members)
	}
}
