package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/cache"
	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/llm"
	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/search"
)

type askRequest struct {
	Query string `json:"query"`
}

type askResponse struct {
	Query    string          `json:"query"`
	Relevant bool            `json:"relevant"`
	Answer   string          `json:"answer"`
	Results  []search.Result `json:"results"`
}

// askHandler handles POST /api/ask: retrieve first with the same service
// /api/search uses, then answer from cache or from the model. The raw
// results ride along so the frontend can still render member cards under
// the paragraph.
func askHandler(svc *search.Service, asker *llm.Client, answers *cache.AnswerCache) http.HandlerFunc {
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

		var hit llm.AskResult
		if ok, err := answers.Get(r.Context(), req.Query, &hit); err == nil && ok {
			writeJSON(w, http.StatusOK, askResponse{
				Query: req.Query, Relevant: hit.Relevant, Answer: hit.Answer, Results: results,
			})
			return
		}

		answer, err := asker.Ask(r.Context(), req.Query, results)
		if err != nil {
			// Surface the failure instead of inventing a "not relevant":
			// a missing API key must not read as "your query is off-topic".
			// A failed call is also never cached.
			writeError(w, http.StatusBadGateway, "answer generation failed")
			return
		}

		// Cache write failures are swallowed: the answer is still correct,
		// just slower next time (same as search).
		_ = answers.Set(r.Context(), req.Query, answer)

		writeJSON(w, http.StatusOK, askResponse{
			Query: req.Query, Relevant: answer.Relevant, Answer: answer.Answer, Results: results,
		})
	}
}
