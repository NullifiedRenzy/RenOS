package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSecurityHeaders(t *testing.T) {
	handler := securityHeadersMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	tests := []struct {
		header string
		value  string
	}{
		{"X-Content-Type-Options", "nosniff"},
		{"X-Frame-Options", "DENY"},
		{"X-XSS-Protection", "0"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
		{"Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()"},
		{"Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload"},
	}

	for _, tt := range tests {
		got := rec.Header().Get(tt.header)
		if got != tt.value {
			t.Errorf("header %s = %q, want %q", tt.header, got, tt.value)
		}
	}
}

func TestMethodOverride(t *testing.T) {
	var captured string
	handler := methodOverrideMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Method
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		method   string
		header   string
		value    string
		expected string
	}{
		{"POST", "X-HTTP-Method-Override", "GET", "GET"},
		{"POST", "X-HTTP-Method-Override", "HEAD", "HEAD"},
		{"POST", "X-HTTP-Method-Override", "OPTIONS", "OPTIONS"},
		{"POST", "X-HTTP-Method-Override", "DELETE", "POST"}, // blocked: DELETE not allowed
		{"POST", "X-HTTP-Method-Override", "PUT", "POST"},    // blocked: PUT not allowed
		{"GET", "X-HTTP-Method-Override", "DELETE", "GET"},   // not POST, override ignored
		{"POST", "", "", "POST"},                              // no override
	}

	for _, tt := range tests {
		t.Run(tt.method+"_"+tt.value, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.header != "" && tt.value != "" {
				req.Header.Set(tt.header, tt.value)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if captured != tt.expected {
				t.Errorf("method override %s + %s=%s: got %s, want %s", tt.method, tt.header, tt.value, captured, tt.expected)
			}
		})
	}
}

func TestTimingHeader(t *testing.T) {
	handler := timingHeaderMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	timing := rec.Header().Get("X-Response-Time")
	if timing == "" {
		t.Error("expected X-Response-Time header")
	}
}

func TestMemoryStats(t *testing.T) {
	stats := getMemoryStats()
	if stats.NumCPU == 0 {
		t.Error("expected NumCPU > 0")
	}
	if stats.Goroutines == 0 {
		t.Error("expected Goroutines > 0")
	}
}

func TestConnectionTracker(t *testing.T) {
	ct := newConnectionTracker()
	current, peak, total := ct.stats()
	if current != 0 || peak != 0 || total != 0 {
		t.Errorf("expected all zeros, got %d/%d/%d", current, peak, total)
	}

	ct.begin()
	current, peak, total = ct.stats()
	if current != 1 || peak != 1 || total != 1 {
		t.Errorf("expected 1/1/1, got %d/%d/%d", current, peak, total)
	}

	ct.end()
	current, _, total = ct.stats()
	if current != 0 || total != 1 {
		t.Errorf("expected 0/1, got %d/%d", current, total)
	}
}

func TestExtractClientIP(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		remote   string
		expected string
	}{
		{"XFF", map[string]string{"X-Forwarded-For": "1.2.3.4, 5.6.7.8"}, "10.0.0.1:1234", "1.2.3.4"},
		{"XRI", map[string]string{"X-Real-IP": "2.3.4.5"}, "10.0.0.1:1234", "2.3.4.5"},
		{"CF", map[string]string{"CF-Connecting-IP": "3.4.5.6"}, "10.0.0.1:1234", "3.4.5.6"},
		{"Remote", map[string]string{}, "10.0.0.1:1234", "10.0.0.1"},
		{"RemoteNoPort", map[string]string{}, "10.0.0.1", "10.0.0.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			req.RemoteAddr = tt.remote
			got := extractClientIP(req)
			if got != tt.expected {
				t.Errorf("extractClientIP() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSlowRequestLogger(t *testing.T) {
	logger := newSlowRequestLogger(10 * time.Millisecond)
	handler := logger.middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(20 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if logger.count != 1 {
		t.Errorf("expected 1 slow request, got %d", logger.count)
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	handler := requestIDMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "test-123")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-ID") != "test-123" {
		t.Errorf("expected X-Request-ID to be forwarded")
	}
}

func TestCompressionHintMiddleware(t *testing.T) {
	handler := compressionHintMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Vary") != "Accept-Encoding" {
		t.Errorf("expected Vary: Accept-Encoding")
	}
}

func TestServerStatsHandler(t *testing.T) {
	handler := serverStatsHandler()
	req := httptest.NewRequest("GET", "/stats", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestMemoryStatsHandler(t *testing.T) {
	handler := memoryStatsHandler()
	req := httptest.NewRequest("GET", "/stats/memory", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestGCStatsHandler(t *testing.T) {
	handler := gcStatsHandler()
	req := httptest.NewRequest("GET", "/stats/gc", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestConnectionStatsHandler(t *testing.T) {
	handler := connectionStatsHandler()
	req := httptest.NewRequest("GET", "/stats/connections", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestSlowRequestStatsHandler(t *testing.T) {
	handler := slowRequestStatsHandler()
	req := httptest.NewRequest("GET", "/stats/slow-requests", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestDebugRequestHandler(t *testing.T) {
	handler := debugRequestHandler()
	req := httptest.NewRequest("POST", "/debug/request", nil)
	req.Header.Set("User-Agent", "test-agent")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}
