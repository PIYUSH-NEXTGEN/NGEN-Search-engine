package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/cache"
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

// askHandler handles POST /api/ask: retrieve first with the same service
// /api/search uses, then answer from cache or from the model. Prior turns
// from the caller's session go into the prompt so follow-up questions have
// context. The raw results ride along so the frontend can still render
// member cards under the paragraph.
func askHandler(svc *search.Service, asker *llm.Client, answers *cache.AnswerCache, sessions *session.Store) http.HandlerFunc {
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

		results, err := svc.Search(r.Context(), req.Query)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "search failed")
			return
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
			answer, err = asker.Ask(r.Context(), req.Query, results, history)
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
			Results: results, SessionID: sessionID,
		})
	}
}
