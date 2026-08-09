package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/renos-project/renos/core/internal/auth"
	"github.com/renos-project/renos/core/internal/config"
	"github.com/renos-project/renos/core/internal/crashreport"
	"github.com/renos-project/renos/core/internal/db"
	"golang.org/x/time/rate"
)

type contextKey string

const claimsKey contextKey = "auth_claims"

// --- Rate limiter ---

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type rateLimiter struct {
	mu       sync.Mutex
	clients  map[string]*clientLimiter
	rate     rate.Limit
	burst    int
	cleanupT time.Duration
	maxSize  int
}

func newRateLimiter(r rate.Limit, burst int) *rateLimiter {
	rl := &rateLimiter{
		clients:  make(map[string]*clientLimiter),
		rate:     r,
		burst:    burst,
		cleanupT: 10 * time.Minute,
		maxSize:  10000,
	}
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) cleanup() {
	for {
		time.Sleep(rl.cleanupT)
		rl.mu.Lock()
		for key, cl := range rl.clients {
			if time.Since(cl.lastSeen) > rl.cleanupT {
				delete(rl.clients, key)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *rateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cl, ok := rl.clients[key]
	if !ok {
		// Evict oldest entry if at capacity
		if len(rl.clients) >= rl.maxSize {
			var oldestKey string
			var oldestTime time.Time
			for k, v := range rl.clients {
				if oldestKey == "" || v.lastSeen.Before(oldestTime) {
					oldestKey = k
					oldestTime = v.lastSeen
				}
			}
			if oldestKey != "" {
				delete(rl.clients, oldestKey)
			}
		}
		cl = &clientLimiter{limiter: rate.NewLimiter(rl.rate, rl.burst)}
		rl.clients[key] = cl
	}
	cl.lastSeen = time.Now()
	return cl.limiter.Allow()
}

var (
	loginLimiter     = newRateLimiter(5, 10)
	apiLimiter       = newRateLimiter(50, 100)
	wsConnectLimiter = newRateLimiter(2, 5)
)

func InitRateLimiters(cfg *config.Config) {
	r := func(v, def int) int {
		if v > 0 {
			return v
		}
		return def
	}
	b := func(v, def int) int {
		if v > 0 {
			return v
		}
		return def
	}
	loginLimiter = newRateLimiter(rate.Limit(r(cfg.RateLimitLogin, 5)), b(cfg.RateLimitLoginBurst, 10))
	apiLimiter = newRateLimiter(rate.Limit(r(cfg.RateLimitAPI, 50)), b(cfg.RateLimitAPIBurst, 100))
	wsConnectLimiter = newRateLimiter(rate.Limit(r(cfg.RateLimitWSConnect, 2)), b(cfg.RateLimitWSConnectBurst, 5))
}

func rateLimitMW(limiter *rateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.RemoteAddr
			if !limiter.Allow(key) {
				writeError(w, http.StatusTooManyRequests, "RATE_LIMITED", "Too many requests")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// --- Role-based access control ---

func writePermissionDenied(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	json.NewEncoder(w).Encode(map[string]string{
		"error": "permission_denied",
	})
}

// requireOneOfRoles checks that the authenticated user has one of the given roles.
// Logs permission.denied via the audit repository when access is rejected.
func requireOneOfRoles(audits *db.AuditRepository, roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(claimsKey).(*auth.Claims)
			if !ok || claims == nil {
				writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
				return
			}
			for _, role := range roles {
				if strings.EqualFold(claims.Role, role) {
					next.ServeHTTP(w, r)
					return
				}
			}
			if audits != nil {
				audits.Create(r.Context(), "system", "rbac", "permission.denied", "user", claims.UserID)
			}
			writePermissionDenied(w)
		})
	}
}

// roleRequired is kept for backward compatibility during migration — delegates to requireOneOfRoles.
func roleRequired(audits *db.AuditRepository, roles ...string) func(http.Handler) http.Handler {
	return requireOneOfRoles(audits, roles...)
}

// --- CORS ---

func corsMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowedOrigins := []string{"http://localhost:3000", "http://localhost:8443", "https://localhost:8443"}

		allowed := false
		for _, o := range allowedOrigins {
			if origin == o {
				allowed = true
				break
			}
		}
		if origin == "" {
			allowed = true // Allow direct requests (CLI, curl)
		}

		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// --- Request body size limit ---

var maxBodySize int64 = 1 << 20 // 1 MB default

func bodyLimitMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
		next.ServeHTTP(w, r)
	})
}

// --- JWT auth middleware ---

func authRequired(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authorization header")
				return
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid authorization header format")
				return
			}

			claims, err := auth.ValidateToken(jwtSecret, parts[1])
			if err != nil {
				writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// --- Request logger ---

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		defer func() {
			slog.Info("request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		}()

		next.ServeHTTP(ww, r)
	})
}

// --- Panic recovery ---

func recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered",
					"request_id", middleware.GetReqID(r.Context()),
					"panic", rec,
				)
				buf := make([]byte, 1<<16)
				n := runtime.Stack(buf, false)
				crashreport.Capture(rec, buf[:n])
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// --- Standardized response helpers ---

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": data,
	})
}

// writeJSONResponse writes a top-level JSON payload without the standard
// {"data": ...} wrapper. Used by endpoints that expose a fixed schema such as
// onboarding status and the health dashboard.
func writeJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": sanitizeErrorMessage(message),
		},
	})
}

// sanitizeErrorMessage removes internal details from error messages
// to prevent information disclosure to clients.
func sanitizeErrorMessage(msg string) string {
	// Strip common internal error prefixes
	sensitivePatterns := []string{
		"pq:", "pgx:", "connection refused", "no such host",
		"permission denied", "i/o timeout", "EOF",
		"/home/", "/opt/", "/var/", "/etc/",
	}
	lower := strings.ToLower(msg)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(lower, pattern) {
			return "internal error"
		}
	}
	return msg
}

// --- Device secret auth ---

type deviceAuthRequest struct {
	DeviceID     string `json:"device_id"`
	DeviceSecret string `json:"device_secret"`
}

func verifyDeviceSecret(devices *db.DeviceRepository, req deviceAuthRequest) (*db.Device, *apiError) {
	if req.DeviceID == "" || req.DeviceSecret == "" {
		return nil, errMissingFields
	}
	device, err := devices.FindByID(context.Background(), req.DeviceID)
	if err != nil {
		return nil, errDeviceNotFound
	}
	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(device.Metadata), &metadata); err != nil {
		return nil, errInternal
	}
	hash, ok := metadata["device_secret_hash"].(string)
	if !ok || hash == "" {
		return nil, errNotEnrolled
	}
	if !auth.VerifyPassword(hash, req.DeviceSecret) {
		return nil, errInvalidSecret
	}
	return device, nil
}

var (
	errMissingFields  = &apiError{status: http.StatusBadRequest, code: "INVALID_REQUEST", message: "device_id and device_secret required"}
	errDeviceNotFound = &apiError{status: http.StatusUnauthorized, code: "UNAUTHORIZED", message: "Device not found"}
	errNotEnrolled    = &apiError{status: http.StatusUnauthorized, code: "UNAUTHORIZED", message: "Device not enrolled"}
	errInvalidSecret  = &apiError{status: http.StatusUnauthorized, code: "UNAUTHORIZED", message: "Invalid device secret"}
	errInternal       = &apiError{status: http.StatusInternalServerError, code: "INTERNAL_ERROR", message: "Internal error"}
)

type apiError struct {
	status  int
	code    string
	message string
}

func (e *apiError) Error() string {
	return e.message
}
