package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/renos-project/renos/core/internal/config"
)

func TestHealthEndpointReturns200(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/api/v1/healthz", healthHandler(nil, nil, nil))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%s'", body["status"])
	}

	if body["version"] != "0.1.0" {
		t.Errorf("expected version '0.1.0', got '%s'", body["version"])
	}
}

func TestHealthEndpointContentType(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/api/v1/healthz", healthHandler(nil, nil, nil))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", ct)
	}
}

func TestReadyEndpointReportsDegradedWhenDepsDown(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/api/v1/readyz", readyHandler(nil, nil, nil))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/readyz", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503 when deps are down, got %d", rec.Code)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if body["status"] != "degraded" {
		t.Errorf("expected status 'degraded', got '%s'", body["status"])
	}
}

func TestReadyEndpointReportsDegradedWhenNATSDown(t *testing.T) {
	t.Skip("requires real NATS connection — integration test")
}

func TestCheckDatabaseWithNilPool(t *testing.T) {
	result := checkDatabase(nil)

	if result.Status != "error" {
		t.Errorf("expected status 'error' for nil pool, got '%s'", result.Status)
	}
}

func TestRouterServesHealthAndReady(t *testing.T) {
	deps := &Dependencies{
		Cfg: &config.Config{JWTSecret: "test-secret"},
	}
	router := NewRouter(deps)

	tt := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{"health endpoint", "/api/v1/healthz", http.StatusOK},
		{"ready endpoint", "/api/v1/readyz", http.StatusServiceUnavailable},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("path %s: expected %d, got %d", tc.path, tc.wantStatus, rec.Code)
			}
		})
	}
}

func TestHealthEndpointWithoutDB(t *testing.T) {
	deps := &Dependencies{
		Cfg: &config.Config{JWTSecret: "test-secret"},
	}
	router := NewRouter(deps)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("health should return 200 even without deps, got %d", rec.Code)
	}
}
