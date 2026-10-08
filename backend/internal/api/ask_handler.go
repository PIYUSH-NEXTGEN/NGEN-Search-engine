package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/llm"
	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/search"
	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/session"
)

type askRequest struct {
	Query string `json:"query"`
	// SessionID is optional: send back the one a previous response handed
	// out to turn this request into a follow-up ("tell me more about her").
	SessionID string `json:"session_id"`
}

type askResponse struct {
	Query     string          `json:"query"`
	Relevant  bool            `json:"relevant"`
	Answer    string          `json:"answer"`
	Results   []search.Result `json:"results"`
	SessionID string          `json:"session_id"`
}

// searcher is the retrieval surface askHandler needs: grouped search plus the
// standing community_info context. search.Service satisfies it; tests use a
// fake.
type searcher interface {
	SearchAll(ctx context.Context, query string) (search.AllResults, error)
	StandingInfo(ctx context.Context) ([]search.CommunityInfo, error)
}

// asker generates one grounded answer. llm.Client satisfies it; tests use a
// fake.
type asker interface {
	Ask(ctx context.Context, query string, results search.AllResults, info []llm.Info, history []llm.Turn) (llm.AskResult, error)
}

// answerCacher stores context-free answers. cache.AnswerCache satisfies it;
// tests use a fake.
type answerCacher interface {
	Get(ctx context.Context, query string, dest any) (bool, error)
	Set(ctx context.Context, query string, value any) error
}

// sessionStorer keeps follow-up turns. session.Store satisfies it; tests use
// a fake.
type sessionStorer interface {
	Get(ctx context.Context, sessionID string) (*session.Session, bool, error)
	Save(ctx context.Context, sessionID string, sess *session.Session) error
}

// askHandler handles POST /api/ask: retrieve first with the same service
// /api/search uses, then answer from cache or from the model. Prior turns
// from the caller's session go into the prompt so follow-up questions have
// context. The member group rides along so the frontend can still render
// member cards under the paragraph — "results" stays the members array so
// the current frontend keeps working untouched.
func askHandler(svc searcher, asker asker, answers answerCacher, sessions sessionStorer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req askRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "body must be JSON with a 'query' field")
			return
		}
		if strings.TrimSpace(req.Query) == "" {
			writeError(w, http.StatusBadRequest, "field 'query' is required")
			return
		}

		all, err := svc.SearchAll(r.Context(), req.Query)
		if err != nil {
			log.Printf("ask: search all: %v", err)
			writeError(w, http.StatusInternalServerError, "search failed")
			return
		}
		if all.Members == nil {
			all.Members = []search.Result{}
		}

		// Standing community info goes into every prompt so general
		// questions ("what is this community about?") work even when
		// full-text search matched nothing. A failed load degrades to an
		// empty list rather than failing the ask — the matched groups may
		// still answer the question.
		standing, err := svc.StandingInfo(r.Context())
		if err != nil {
			log.Printf("ask: standing info: %v", err)
			standing = []search.CommunityInfo{}
		}
		// Map into the llm package's own row shape so the answer layer
		// doesn't depend on the search package's model.
		info := make([]llm.Info, 0, len(standing))
		for _, row := range standing {
			info = append(info, llm.Info{Slug: row.Slug, Title: row.Title, Body: row.Body})
		}

		// Reuse the caller's session when they send one; otherwise mint an id
		// so they can carry this conversation into the next request.
		sessionID := strings.TrimSpace(req.SessionID)
		if sessionID == "" {
			sessionID = uuid.NewString()
		}
		sess := session.Session{}
		var history []llm.Turn
		if loaded, ok, err := sessions.Get(r.Context(), sessionID); err == nil && ok {
			sess = *loaded
			history = sess.Turns
		}

		// A cached answer was written without this conversation's context, so
		// only context-free asks may read it — and only they may write it,
		// otherwise a context-aware answer would be replayed out of context.
		cacheable := len(history) == 0

		var answer llm.AskResult
		answered := false
		if cacheable {
			var hit llm.AskResult
			if ok, err := answers.Get(r.Context(), req.Query, &hit); err == nil && ok {
				answer, answered = hit, true
			}
		}
		if !answered {
			answer, err = asker.Ask(r.Context(), req.Query, all, info, history)
			if err != nil {
				// Surface the failure instead of inventing a "not relevant":
				// a missing API key must not read as "your query is off-topic".
				// A failed call is also never cached. The detail goes to the
				// server log — the response stays generic.
				log.Printf("ask: %v", err)
				writeError(w, http.StatusBadGateway, "answer generation failed")
				return
			}
			if cacheable {
				// Cache write failures are swallowed: the answer is still
				// correct, just slower next time (same as search).
				_ = answers.Set(r.Context(), req.Query, answer)
			}
		}

		// Record the turn so the next question can lean on it. A failed write
		// only costs the follow-up its context, so swallow it like the other
		// cache writes rather than failing a successful answer.
		sess.AddTurn(req.Query, answer)
		_ = sessions.Save(r.Context(), sessionID, &sess)

		writeJSON(w, http.StatusOK, askResponse{
			Query: req.Query, Relevant: answer.Relevant, Answer: answer.Answer,
			Results: all.Members, SessionID: sessionID,
		})
	}
}
