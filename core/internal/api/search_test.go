package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/renos-project/renos/core/internal/db"
)

func TestSearchHandlerEmptyQuery(t *testing.T) {
	// Empty query must return an empty result set without touching the DB.
	h := searchHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=", nil)
	rr := httptest.NewRecorder()
	h(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var resp struct {
		Data db.SearchResponse `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.Data.Total != 0 {
		t.Errorf("total = %d, want 0", resp.Data.Total)
	}
	if resp.Data.Results == nil {
		t.Error("results must be non-nil empty map")
	}
}

func TestSearchHandlerInvalidLimit(t *testing.T) {
	h := searchHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=test&limit=abc", nil)
	rr := httptest.NewRecorder()
	h(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "limit must be an integer") {
		t.Errorf("unexpected body: %s", rr.Body.String())
	}
}

func TestSearchHandlerMissingQuery(t *testing.T) {
	h := searchHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search", nil)
	rr := httptest.NewRecorder()
	h(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var resp struct {
		Data db.SearchResponse `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.Data.Total != 0 {
		t.Errorf("total = %d, want 0", resp.Data.Total)
	}
}

func TestSearchResultSerialization(t *testing.T) {
	res := db.SearchResult{
		Type:     "devices",
		ID:       "abc-123",
		Title:    "webserver-01",
		Subtitle: "linux · online",
		Href:     "/devices/abc-123",
	}
	raw, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	// Secret values must never be included in search payloads.
	if strings.Contains(string(raw), "secret") {
		t.Errorf("search result should not serialize secrets: %s", raw)
	}
}
