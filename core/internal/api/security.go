package api

import (
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"
)

// --- Security Headers Middleware ---

func securityHeadersMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "0")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self' ws: wss:; worker-src 'self' blob:; frame-ancestors 'none'")
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		next.ServeHTTP(w, r)
	})
}

// --- Request ID propagation ---

func requestIDMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get("X-Request-ID")
		if rid == "" {
			rid = r.Header.Get("X-Trace-ID")
		}
		if rid != "" {
			w.Header().Set("X-Request-ID", rid)
		}
		next.ServeHTTP(w, r)
	})
}

// --- Slow request logging ---

type slowRequestLogger struct {
	threshold time.Duration
	mu        sync.Mutex
	count     int64
}

func newSlowRequestLogger(threshold time.Duration) *slowRequestLogger {
	return &slowRequestLogger{threshold: threshold}
}

func (s *slowRequestLogger) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &statusWriter{ResponseWriter: w, status: 200}

		defer func() {
			duration := time.Since(start)
			if duration > s.threshold {
				s.mu.Lock()
				s.count++
				s.mu.Unlock()
				slog.Warn("slow request",
					"method", r.Method,
					"path", r.URL.Path,
					"duration_ms", duration.Milliseconds(),
					"threshold_ms", s.threshold.Milliseconds(),
					"request_id", r.Header.Get("X-Request-ID"),
				)
			}
			_ = ww
		}()

		next.ServeHTTP(ww, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}

func (sw *statusWriter) Unwrap() http.ResponseWriter {
	return sw.ResponseWriter
}

// --- Request timing header ---

func timingHeaderMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &timingWriter{ResponseWriter: w, start: start}
		next.ServeHTTP(ww, r)
	})
}

type timingWriter struct {
	http.ResponseWriter
	start time.Time
}

func (tw *timingWriter) WriteHeader(code int) {
	elapsed := time.Since(tw.start)
	tw.ResponseWriter.Header().Set("X-Response-Time", elapsed.String())
	tw.ResponseWriter.WriteHeader(code)
}

func (tw *timingWriter) Unwrap() http.ResponseWriter {
	return tw.ResponseWriter
}

// --- Method override (for clients that can't send PUT/DELETE) ---

func methodOverrideMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if m := r.Header.Get("X-HTTP-Method-Override"); m != "" {
				method := strings.ToUpper(m)
				if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
					r.Method = method
				}
			} else if m := r.URL.Query().Get("_method"); m != "" {
				method := strings.ToUpper(m)
				if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
					r.Method = method
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// --- CSRF protection ---
// Requires Origin, X-Requested-With, or Authorization header on state-changing requests.

func csrfProtectionMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			ct := r.Header.Get("Content-Type")
			if strings.HasPrefix(ct, "application/x-www-form-urlencoded") || strings.HasPrefix(ct, "multipart/form-data") {
				writeError(w, http.StatusUnsupportedMediaType, "INVALID_CONTENT_TYPE", "Use application/json for API requests")
				return
			}
			// Agent-facing endpoints use device_secret in body, not Authorization header.
			// Check for device auth patterns to skip CSRF.
			if r.Header.Get("Authorization") == "" && r.Header.Get("Origin") == "" && r.Header.Get("X-Requested-With") == "" {
				// Allow requests with device_id in body (agent auth pattern)
				if !isDeviceAuthRequest(r) {
					writeError(w, http.StatusForbidden, "CSRF_REJECTED", "Missing Origin, X-Requested-With, or Authorization header")
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// isDeviceAuthRequest checks if the request is for a known agent-facing endpoint.
func isDeviceAuthRequest(r *http.Request) bool {
	path := r.URL.Path
	// Agent endpoints that use device_secret in body (no Authorization header)
	// These POST endpoints are in the agent-facing route group
	agentSuffixes := []string{
		"/devices/enroll", "/devices/connect", "/devices/heartbeat",
		"/commands/pending",
		"/container-actions/pending",
		"/service-actions/pending",
		"/file-actions/pending",
		"/file-transfers/pending",
	}
	for _, suffix := range agentSuffixes {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	// Agent result/sync endpoints follow patterns like:
	// /commands/{cmdId}/result, /container-actions/{id}/result,
	// /devices/{id}/containers, /devices/{id}/docker-images, etc.
	// /file-transfers/{id}/chunks/{num}, /file-transfers/{id}/assemble
	if strings.Contains(path, "/commands/") && strings.HasSuffix(path, "/result") {
		return true
	}
	if strings.Contains(path, "/container-actions/") && strings.HasSuffix(path, "/result") {
		return true
	}
	if strings.Contains(path, "/service-actions/") && strings.HasSuffix(path, "/result") {
		return true
	}
	if strings.Contains(path, "/file-actions/") && strings.HasSuffix(path, "/result") {
		return true
	}
	if strings.Contains(path, "/file-transfers/") && (strings.Contains(path, "/chunks/") || strings.HasSuffix(path, "/assemble")) {
		return true
	}
	// Sync endpoints under /devices/{id}/ (agent reports state)
	syncEndpoints := []string{
		"/containers", "/docker-images", "/docker-volumes", "/docker-networks",
		"/systemd-services", "/network-interfaces", "/storage-devices",
		"/virtual-machines", "/updates", "/plugins", "/compose-stacks",
		"/container-metrics", "/proxmox",
	}
	for _, ep := range syncEndpoints {
		if strings.HasSuffix(path, ep) {
			return true
		}
	}
	return false
}

// --- Memory stats for health ---

type MemoryStats struct {
	HeapAlloc   uint64 `json:"heap_alloc"`
	HeapInuse   uint64 `json:"heap_inuse"`
	HeapObjects uint64 `json:"heap_objects"`
	NumGC       uint32 `json:"num_gc"`
	Goroutines  int    `json:"goroutines"`
	NumCPU      int    `json:"num_cpu"`
}

func getMemoryStats() MemoryStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return MemoryStats{
		HeapAlloc:   m.HeapAlloc,
		HeapInuse:   m.HeapInuse,
		HeapObjects: m.HeapObjects,
		NumGC:       m.NumGC,
		Goroutines:  runtime.NumGoroutine(),
		NumCPU:      runtime.NumCPU(),
	}
}

// --- IP extraction ---

func extractClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	if fwd := r.Header.Get("CF-Connecting-IP"); fwd != "" {
		return fwd
	}
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}
	return host
}

// --- Concurrent connection tracker ---

type connectionTracker struct {
	mu       sync.Mutex
	current  int64
	peak     int64
	total    int64
}

func newConnectionTracker() *connectionTracker {
	return &connectionTracker{}
}

func (ct *connectionTracker) begin() {
	ct.mu.Lock()
	ct.current++
	ct.total++
	if ct.current > ct.peak {
		ct.peak = ct.current
	}
	ct.mu.Unlock()
}

func (ct *connectionTracker) end() {
	ct.mu.Lock()
	ct.current--
	ct.mu.Unlock()
}

func (ct *connectionTracker) stats() (current, peak, total int64) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	return ct.current, ct.peak, ct.total
}

func (ct *connectionTracker) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct.begin()
		defer ct.end()
		next.ServeHTTP(w, r)
	})
}

// --- Global instances ---

var (
	slowLogger  = newSlowRequestLogger(5 * time.Second)
	connTracker = newConnectionTracker()
)

func getSlowRequestCount() int64 {
	slowLogger.mu.Lock()
	defer slowLogger.mu.Unlock()
	return slowLogger.count
}
