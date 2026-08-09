package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/renos-project/renos/core/internal/config"
)

func TestCommandRoutesWithoutAuthReturn401(t *testing.T) {
	deps := &Dependencies{
		Cfg: &config.Config{JWTSecret: "test-secret"},
	}
	router := NewRouter(deps)

	tests := []struct {
		name string
		method string
		path   string
		body   string
	}{
		{"request command", http.MethodPost, "/api/v1/devices/aa04f763-9402-4e07-876a-9e35044df404/commands", `{"command":"whoami"}`},
		{"list commands", http.MethodGet, "/api/v1/devices/aa04f763-9402-4e07-876a-9e35044df404/commands", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			// POST without auth headers hits CSRF first (403), GET hits auth (401)
			if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusForbidden {
				t.Errorf("expected 401 or 403, got %d", rec.Code)
			}

			var body map[string]interface{}
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			errObj, ok := body["error"].(map[string]interface{})
			if !ok {
				t.Fatal("expected error object in response")
			}
			code := errObj["code"].(string)
			if code != "UNAUTHORIZED" && code != "CSRF_REJECTED" {
				t.Errorf("expected UNAUTHORIZED or CSRF_REJECTED, got '%s'", code)
			}
		})
	}
}

func TestAgentCommandRoutesRequireDeviceSecret(t *testing.T) {
	deps := &Dependencies{
		Cfg: &config.Config{JWTSecret: "test-secret"},
	}
	router := NewRouter(deps)

	tests := []struct {
		name string
		method string
		path   string
		body   string
	}{
		{"pending commands", http.MethodPost, "/api/v1/commands/pending", `{"device_id":"","device_secret":""}`},
		{"submit result", http.MethodPost, "/api/v1/commands/nonexistent/result", `{"device_id":"","device_secret":"","exit_code":0,"output":"","error":""}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest && rec.Code != http.StatusNotFound {
				t.Errorf("expected 400 or 404, got %d", rec.Code)
			}
		})
	}
}

func TestPendingCommandsHandlerWithMissingFields(t *testing.T) {
	handler := pendingCommandsHandler(nil, nil)

	tests := []struct {
		name string
		body string
		want int
	}{
		{"empty body", `{}`, http.StatusBadRequest},
		{"missing device_secret", `{"device_id":"abc"}`, http.StatusBadRequest},
		{"missing device_id", `{"device_secret":"abc"}`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/commands/pending", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			handler(rec, req)

			if rec.Code != tt.want {
				t.Errorf("expected %d, got %d", tt.want, rec.Code)
			}
		})
	}
}

func TestSubmitResultHandlerWithMissingFields(t *testing.T) {
	handler := submitResultHandler(nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/commands/test-id/result", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestRequestCommandHandlerWithEmptyBody(t *testing.T) {
	handler := requestCommandHandler(nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/test-id/commands", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestRequestCommandHandlerWithMissingCommand(t *testing.T) {
	handler := requestCommandHandler(nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/test-id/commands", strings.NewReader(`{"command":""}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestListCommandsHandlerRoutesRegistered(t *testing.T) {
	deps := &Dependencies{
		Cfg: &config.Config{JWTSecret: "test-secret"},
	}
	router := NewRouter(deps)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/nonexistent/commands", nil)
	req.Header.Set("Authorization", "Bearer test")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Errorf("route should be registered, got 404")
	}
}

func TestGetCommandHandlerRouteExists(t *testing.T) {
	deps := &Dependencies{
		Cfg: &config.Config{JWTSecret: "test-secret"},
	}
	router := NewRouter(deps)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/nonexistent/commands/cmd-1", nil)
	req.Header.Set("Authorization", "Bearer test")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusNotFound {
		t.Errorf("command detail route should be registered, got 404")
	}
}
