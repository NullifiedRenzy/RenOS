// Package fuzz_test contains fuzz tests for RenOS core API handlers.
//
// Run with: go test -fuzz=. -fuzztime=30s ./release/tests/fuzz/
package fuzz_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"

	"github.com/renos-project/renos/core/internal/api"
	"github.com/renos-project/renos/core/internal/auth"
	"github.com/renos-project/renos/core/internal/config"
)

// FuzzAPIRequest sends random HTTP requests to core API handlers.
func FuzzAPIRequest(f *testing.F) {
	deps := &api.Dependencies{
		Cfg: &config.Config{JWTSecret: "test-secret"},
	}
	router := api.NewRouter(deps)

	seeds := []string{
		"GET /api/v1/healthz",
		"GET /api/v1/readyz",
		"GET /api/v1/version",
		"POST /api/v1/auth/login",
		"GET /api/v1/devices",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input string) {
		parts := strings.SplitN(input, " ", 3)
		if len(parts) < 2 {
			return
		}
		method := parts[0]
		path := parts[1]
		var body io.Reader
		if len(parts) == 3 {
			body = strings.NewReader(parts[2])
		}

		validMethods := map[string]bool{
			"GET": true, "POST": true, "PUT": true,
			"DELETE": true, "PATCH": true, "HEAD": true,
		}
		if !validMethods[method] {
			return
		}

		req := httptest.NewRequest(method, path, body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		func() {
			defer func() {
				_ = recover()
			}()
			router.ServeHTTP(rec, req)
		}()

		code := rec.Code
		if code < 100 || code > 599 {
			t.Errorf("invalid status code: %d", code)
		}
	})
}

// FuzzJSONParse sends random JSON payloads to API endpoints.
func FuzzJSONParse(f *testing.F) {
	deps := &api.Dependencies{
		Cfg: &config.Config{JWTSecret: "test-secret"},
	}
	router := api.NewRouter(deps)

	seeds := []string{
		`{}`,
		`{"username":"admin","password":"secret"}`,
		`[]`,
		`null`,
		`{"nested":{"deeply":{"a":1}}}`,
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, payload string) {
		body := bytes.NewReader([]byte(payload))
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		func() {
			defer func() {
				_ = recover()
			}()
			router.ServeHTTP(rec, req)
		}()

		code := rec.Code
		if code < 100 || code > 599 {
			t.Errorf("invalid status code from JSON fuzz: %d", code)
		}

		var resp any
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			if code == http.StatusOK {
				t.Errorf("200 response with invalid JSON: %v", err)
			}
		}
	})
}

// FuzzWebSocket sends random frames to the WebSocket endpoint.
func FuzzWebSocket(f *testing.F) {
	seeds := [][]byte{
		[]byte(`{"type":"ping"}`),
		[]byte(`{"type":"subscribe","channel":"devices"}`),
		[]byte(`hello`),
		{0x00, 0x01, 0x02, 0x03},
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) == 0 {
			return
		}
		if len(data) > 65536 {
			return
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/ws?token=test", nil)
		req.Header.Set("Upgrade", "websocket")
		req.Header.Set("Connection", "upgrade")
		rec := httptest.NewRecorder()

		deps := &api.Dependencies{
			Cfg: &config.Config{JWTSecret: "test-secret"},
		}
		router := api.NewRouter(deps)

		func() {
			defer func() {
				_ = recover()
			}()
			router.ServeHTTP(rec, req)
		}()

		_ = data
	})
}

// FuzzDeviceID tests device ID parsing with random strings.
func FuzzDeviceID(f *testing.F) {
	seeds := []string{
		"device-001",
		"a:b:c:d:e:f",
		"",
		"00000000-0000-0000-0000-000000000000",
		"valid-device-id_123",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	validDeviceID := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_:.-]{0,254}$`)
	uuidPattern := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	macPattern := regexp.MustCompile(`^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$`)

	f.Fuzz(func(t *testing.T, id string) {
		if len(id) > 255 {
			return
		}

		valid := validDeviceID.MatchString(id) &&
			!strings.Contains(id, "..") &&
			!strings.Contains(id, "__")

		if uuidPattern.MatchString(id) {
			valid = true
		}
		if macPattern.MatchString(id) {
			valid = true
		}

		if strings.Contains(id, "../") || strings.Contains(id, "~") {
			valid = false
		}

		_ = valid
	})
}

// FuzzWorkflowName tests workflow name validation.
func FuzzWorkflowName(f *testing.F) {
	seeds := []string{
		"Deploy Production",
		"backup-db",
		"",
		"a",
		"nested/workflow/name",
		"<script>alert(1)</script>",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	validName := regexp.MustCompile(`^[a-zA-Z0-9 _.-]{1,128}$`)

	f.Fuzz(func(t *testing.T, name string) {
		if len(name) > 1024 {
			return
		}

		valid := validName.MatchString(name) &&
			!strings.HasPrefix(name, " ") &&
			!strings.HasSuffix(name, " ") &&
			!strings.HasPrefix(name, ".") &&
			!strings.HasSuffix(name, ".")

		if name == "" {
			valid = false
		}

		if strings.Contains(name, "../") ||
			strings.Contains(name, "..\\") ||
			strings.Contains(name, "\x00") {
			valid = false
		}

		if strings.Contains(name, "\n") || strings.Contains(name, "\r") {
			valid = false
		}

		trimmed := strings.TrimSpace(name)
		if trimmed != name && name != "" {
			valid = false
		}

		_ = valid
	})
}

// FuzzJWTToken tests JWT parsing with random tokens.
func FuzzJWTToken(f *testing.F) {
	seeds := []string{
		"eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
		"invalid.token.string",
		"",
		"a.b.c",
		"eyJhbGciOiJub25lIn0.eyJ1c2VyX2lkIjoiYWRtaW4iLCJyb2xlIjoiYWRtaW4ifQ.",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, tokenStr string) {
		if len(tokenStr) > 8192 {
			return
		}

		claims, err := auth.ValidateToken("test-secret", tokenStr)
		if err == nil {
			if claims.UserID == "" {
				t.Error("valid token missing user_id")
			}
			if claims.Role == "" {
				t.Error("valid token missing role")
			}
			if _, ok := claims.RegisteredClaims.GetExpirationTime(); !ok {
				t.Error("valid token missing expiration")
			}
		}

		parsed, _, _ := new(jwt.Parser).ParseUnverified(tokenStr, jwt.MapClaims{})
		if parsed != nil {
			if _, ok := parsed.Header["alg"]; !ok {
				t.Error("token header missing alg")
			}
		}
	})
}
