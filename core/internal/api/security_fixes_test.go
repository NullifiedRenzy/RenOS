package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIsBlockedCommand(t *testing.T) {
	tests := []struct {
		cmd     string
		blocked bool
	}{
		{"rm -rf /", true},
		{"rm -rf /*", true},
		{"mkfs /dev/sda", true},
		{":(){:|:&};:", true},
		{"dd if=/dev/zero of=/dev/sda", true},
		{"chmod -R 777 /", true},
		{"ls -la", false},
		{"cat /etc/passwd", false},
		{"echo hello", false},
		{"docker ps", false},
		{"systemctl status nginx", false},
		{"RM -RF /", true}, // case insensitive
		{"  rm -rf /  ", true},
		{"rm -rf /tmp/data", false}, // /tmp is safe
		{"rm -rf /var/tmp/data", false},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			got := isBlockedCommand(tt.cmd)
			if got != tt.blocked {
				t.Errorf("isBlockedCommand(%q) = %v, want %v", tt.cmd, got, tt.blocked)
			}
		})
	}
}

func TestSanitizeErrorMessage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"connection refused on 127.0.0.1:5432", "internal error"},
		{"pq: relation does not exist", "internal error"},
		{"no such host: db.internal", "internal error"},
		{"permission denied on /opt/renos", "internal error"},
		{"i/o timeout connecting to redis", "internal error"},
		{"command is required", "command is required"},
		{"device not found", "device not found"},
		{"invalid request body", "invalid request body"},
		{"failed to read file /home/user/secret.key", "internal error"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeErrorMessage(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeErrorMessage(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestValidateFilePath(t *testing.T) {
	tests := []struct {
		path  string
		valid bool
	}{
		{"data/file.txt", true},
		{"../etc/passwd", false},
		{"data/../secrets", false},
		{"data\x00file.txt", false},
		{"/absolute/path", true},
		{"./relative/path", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := validateFilePath(tt.path)
			if got != tt.valid {
				t.Errorf("validateFilePath(%q) = %v, want %v", tt.path, got, tt.valid)
			}
		})
	}
}

func TestWriteErrorSanitizesInternalDetails(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "pq: relation does not exist")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if strings.Contains(body, "pq:") {
		t.Error("error response should not contain internal DB error details")
	}
	if !strings.Contains(body, "internal error") {
		t.Error("error response should contain sanitized message")
	}
}

func TestWriteErrorPreservesSafeMessages(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "name is required")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "name is required") {
		t.Error("error response should preserve safe messages")
	}
}

func TestRateLimiterMaxSize(t *testing.T) {
	rl := &rateLimiter{
		clients:  make(map[string]*clientLimiter),
		rate:     100,
		burst:    100,
		cleanupT: time.Hour, // long cleanup so it doesn't interfere
		maxSize:  100,
	}
	// Don't start cleanup goroutine for this test

	// Fill to max
	for i := 0; i < rl.maxSize; i++ {
		key := "client-" + strings.Repeat("x", i/26) + string(rune('A'+i%26))
		rl.Allow(key)
	}

	// Verify at capacity
	rl.mu.Lock()
	size := len(rl.clients)
	rl.mu.Unlock()
	if size != rl.maxSize {
		t.Errorf("expected %d clients, got %d", rl.maxSize, size)
	}

	// Adding one more should evict oldest
	rl.Allow("new-client")
	rl.mu.Lock()
	size = len(rl.clients)
	rl.mu.Unlock()
	if size != rl.maxSize {
		t.Errorf("expected %d clients after eviction, got %d", rl.maxSize, size)
	}
}

func TestWebSocketOriginRejectsEmpty(t *testing.T) {
	// Empty origin should be rejected
	if terminalUpgrader.CheckOrigin(httptest.NewRequest("GET", "/", nil)) {
		t.Error("empty origin should be rejected")
	}
}

func TestWebSocketOriginRejectsRenosLocal(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://evil.renos.local")
	if terminalUpgrader.CheckOrigin(req) {
		t.Error("wildcard .renos.local origin should be rejected")
	}
}

func TestWebSocketOriginAllowsLocalhost(t *testing.T) {
	tests := []string{
		"http://localhost:3000",
		"https://localhost:8443",
		"https://localhost",
	}
	for _, origin := range tests {
		t.Run(origin, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Origin", origin)
			if !terminalUpgrader.CheckOrigin(req) {
				t.Errorf("localhost origin %s should be allowed", origin)
			}
		})
	}
}

func TestWebSocketOriginRejectsNonLocalhost(t *testing.T) {
	tests := []string{
		"https://evil.com",
		"http://attacker.local",
		"https://localhost.evil.com",
	}
	for _, origin := range tests {
		t.Run(origin, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Origin", origin)
			if terminalUpgrader.CheckOrigin(req) {
				t.Errorf("non-localhost origin %s should be rejected", origin)
			}
		})
	}
}
