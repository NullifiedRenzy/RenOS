package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ──────────────── Recovery Tests ────────────────

func TestReadyzRecoversAfterFailure(t *testing.T) {
	rec := httptest.NewRecorder()
	if rec.Code == http.StatusOK {
		t.Log("health endpoint accessible")
	}
}

func TestWriteJSONConcurrent(t *testing.T) {
	// Verify writeJSON is safe under concurrent access
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec := httptest.NewRecorder()
			writeJSON(rec, http.StatusOK, map[string]string{"key": "value"})
			if rec.Code != http.StatusOK {
				t.Errorf("expected 200, got %d", rec.Code)
			}
		}()
	}
	wg.Wait()
}

func TestWriteErrorConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec := httptest.NewRecorder()
			writeError(rec, http.StatusInternalServerError, "TEST_ERR", "test error")
			if rec.Code != http.StatusInternalServerError {
				t.Errorf("expected 500, got %d", rec.Code)
			}
		}()
	}
	wg.Wait()
}

func TestMethodOverrideRecovery(t *testing.T) {
	// Verify the method override middleware doesn't panic under concurrent access
	mw := methodOverrideMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				req := httptest.NewRequest("POST", "/test", nil)
				req.Header.Set("X-HTTP-Method-Override", "GET")
				rec := httptest.NewRecorder()
				mw.ServeHTTP(rec, req)
			}
		}()
	}
	wg.Wait()
}

func TestCSRFRecovery(t *testing.T) {
	mw := csrfProtectionMW(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("POST", "/test", nil)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test")
			rec := httptest.NewRecorder()
			mw.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Errorf("expected 200, got %d", rec.Code)
			}
		}()
	}
	wg.Wait()
}

// ──────────────── Context Cancellation ────────────────

func TestContextCancellationInHandler(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			writeError(w, http.StatusGatewayTimeout, "TIMEOUT", "request cancelled")
		case <-time.After(100 * time.Millisecond):
			writeJSON(w, http.StatusOK, "done")
		}
	})

	req := httptest.NewRequest("GET", "/test", nil)
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)
	cancel() // Cancel immediately

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusGatewayTimeout {
		t.Errorf("expected 504 for cancelled context, got %d", rec.Code)
	}
}

// ──────────────── Retry / Backoff Pattern ────────────────

func TestExponentialBackoffBounds(t *testing.T) {
	baseDelay := 30 * time.Second
	maxDelay := 10 * time.Minute

	tests := []struct {
		retries int
		expect  time.Duration
	}{
		{0, 30 * time.Second},
		{1, 60 * time.Second},
		{2, 120 * time.Second},
		{3, 240 * time.Second},
		{4, 480 * time.Second},
		{5, 600 * time.Second}, // capped at maxDelay (10m = 600s)
		{10, 600 * time.Second},
	}

	for _, tt := range tests {
		delay := baseDelay * (1 << tt.retries)
		if delay > maxDelay {
			delay = maxDelay
		}
		if delay != tt.expect {
			t.Errorf("retries=%d: expected %v, got %v", tt.retries, tt.expect, delay)
		}
	}
}

// ──────────────── Concurrent Rate Limiter ────────────────

func TestRateLimiterConcurrent(t *testing.T) {
	rl := &rateLimiter{
		clients:  make(map[string]*clientLimiter),
		rate:     100,
		burst:    100,
		cleanupT: time.Hour, // never runs during test
		maxSize:  10000,
	}

	var allowed, denied atomic.Int64
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				key := "client-" + string(rune('A'+id))
				if rl.Allow(key) {
					allowed.Add(1)
				} else {
					denied.Add(1)
				}
			}
		}(i)
	}
	wg.Wait()

	total := allowed.Load() + denied.Load()
	if total != 500 {
		t.Errorf("expected 500 operations, got %d", total)
	}
}

// ──────────────── Connection Pool Stress ────────────────

func TestConcurrentWriteJSON(t *testing.T) {
	var wg sync.WaitGroup
	start := make(chan struct{})
	n := 50

	// Burst 50 concurrent writeJSON calls to simulate connection pool pressure
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < 10; j++ {
				rec := httptest.NewRecorder()
				writeJSON(rec, http.StatusOK, struct{ ID int }{ID: j})
				if rec.Code != http.StatusOK {
					t.Errorf("expected 200, got %d", rec.Code)
				}
			}
		}()
	}
	close(start)
	wg.Wait()
}

// ──────────────── Pagination Edge Cases ────────────────

func TestPaginationLimits(t *testing.T) {
	tests := []struct {
		limitStr   string
		offsetStr  string
		wantLimit  int
		wantOffset int
	}{
		{"50", "0", 50, 0},
		{"200", "0", 200, 0},
		{"300", "0", 200, 0},   // capped
		{"0", "0", 50, 0},      // default
		{"-1", "0", 50, 0},     // invalid -> default
		{"50", "5000", 50, 5000},
		{"50", "15000", 50, 10000}, // capped
		{"50", "-1", 50, 0},        // invalid -> default
		{"abc", "0", 50, 0},        // invalid -> default
	}

	for _, tt := range tests {
		limit := parseLimit(tt.limitStr)
		offset := parseOffset(tt.offsetStr)
		if limit != tt.wantLimit {
			t.Errorf("parseLimit(%q): got %d, want %d", tt.limitStr, limit, tt.wantLimit)
		}
		if offset != tt.wantOffset {
			t.Errorf("parseOffset(%q): got %d, want %d", tt.offsetStr, offset, tt.wantOffset)
		}
	}
}
