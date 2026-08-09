package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/renos-project/renos/core/internal/db"
)

// searchHandler returns aggregate search results across the platform.
// GET /api/v1/search?q=<query>&limit=<n>
func searchHandler(search *db.SearchRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if q == "" {
			writeJSON(w, http.StatusOK, &db.SearchResponse{
				Query:   "",
				Total:   0,
				Results: map[string][]db.SearchResult{},
			})
			return
		}
		limit := 10
		if l := r.URL.Query().Get("limit"); l != "" {
			n, err := strconv.Atoi(l)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid_request", "limit must be an integer")
				return
			}
			limit = n
		}

		resp, err := search.Search(r.Context(), q, limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "search failed: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp)
	}
}
