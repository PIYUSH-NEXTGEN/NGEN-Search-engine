package api

import (
	"encoding/json"
	"net/http"

	"github.com/PIYUSH-NEXTGEN/NGEN-Search-engine/internal/search"
)

type searchResponse struct {
	Query   string          `json:"query"`
	Results []search.Result `json:"results"`
}

// searchHandler handles GET /api/search?q=...
// This is Phase 1: pure retrieval, no LLM involved yet.
func searchHandler(svc *search.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		if q == "" {
			writeError(w, http.StatusBadRequest, "query parameter 'q' is required")
			return
		}

		results, err := svc.Search(r.Context(), q)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "search failed")
			return
		}

		writeJSON(w, http.StatusOK, searchResponse{Query: q, Results: results})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
