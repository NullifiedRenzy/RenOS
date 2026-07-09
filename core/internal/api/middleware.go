package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/renos-project/renos/core/internal/auth"
	"github.com/renos-project/renos/core/internal/db"
)

type contextKey string

const claimsKey contextKey = "auth_claims"

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

func recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered",
					"request_id", middleware.GetReqID(r.Context()),
					"panic", rec,
				)
				writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

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

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

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
	errNotEnrolled   = &apiError{status: http.StatusUnauthorized, code: "UNAUTHORIZED", message: "Device not enrolled"}
	errInvalidSecret = &apiError{status: http.StatusUnauthorized, code: "UNAUTHORIZED", message: "Invalid device secret"}
	errInternal      = &apiError{status: http.StatusInternalServerError, code: "INTERNAL_ERROR", message: "Internal error"}
)

type apiError struct {
	status  int
	code    string
	message string
}

func (e *apiError) Error() string {
	return e.message
}
