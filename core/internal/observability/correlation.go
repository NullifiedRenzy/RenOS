package observability

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/trace"
)

type logContextKey string

const (
	ctxRequestID  logContextKey = "request_id"
	ctxUserID     logContextKey = "user_id"
	ctxDeviceID   logContextKey = "device_id"
	ctxWorkflowID logContextKey = "workflow_id"
	ctxSessionID  logContextKey = "session_id"
)

// LogCorrelationMiddleware enriches the context and slog logger with request-scoped
// correlation IDs (trace_id, span_id, request_id, user_id, device_id, etc.).
func LogCorrelationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Extract known correlation IDs from headers
		requestID := r.Header.Get("X-Request-ID")
		userID := r.Header.Get("X-User-ID")
		deviceID := r.Header.Get("X-Device-ID")
		workflowID := r.Header.Get("X-Workflow-ID")
		sessionID := r.Header.Get("X-Session-ID")

		if requestID == "" {
			requestID = r.Header.Get("X-Correlation-ID")
		}

		// Store in context
		if requestID != "" {
			ctx = context.WithValue(ctx, ctxRequestID, requestID)
		}
		if userID != "" {
			ctx = context.WithValue(ctx, ctxUserID, userID)
		}
		if deviceID != "" {
			ctx = context.WithValue(ctx, ctxDeviceID, deviceID)
		}
		if workflowID != "" {
			ctx = context.WithValue(ctx, ctxWorkflowID, workflowID)
		}
		if sessionID != "" {
			ctx = context.WithValue(ctx, ctxSessionID, sessionID)
		}

		// Extract trace/span from OpenTelemetry
		spanCtx := trace.SpanContextFromContext(ctx)
		traceID := ""
		spanID := ""
		if spanCtx.HasTraceID() {
			traceID = spanCtx.TraceID().String()
			ctx = context.WithValue(ctx, traceIDKey, traceID)
		}
		if spanCtx.HasSpanID() {
			spanID = spanCtx.SpanID().String()
			ctx = context.WithValue(ctx, spanIDKey, spanID)
		}

		// Build correlation args for the logger
		var args []any
		if traceID != "" {
			args = append(args, slog.String("trace_id", traceID))
		}
		if spanID != "" {
			args = append(args, slog.String("span_id", spanID))
		}
		if requestID != "" {
			args = append(args, slog.String("request_id", requestID))
		}
		if userID != "" {
			args = append(args, slog.String("user_id", userID))
		}
		if deviceID != "" {
			args = append(args, slog.String("device_id", deviceID))
		}
		if workflowID != "" {
			args = append(args, slog.String("workflow_id", workflowID))
		}
		if sessionID != "" {
			args = append(args, slog.String("session_id", sessionID))
		}

		// Attach correlated logger to context
		if len(args) > 0 {
			ctx = context.WithValue(ctx, logContextKey("logger"), slog.With(args...))
		}

		start := time.Now()
		next.ServeHTTP(w, r.WithContext(ctx))

		// Log request completion
		logArgs := append([]any{
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Duration("duration", time.Since(start)),
		}, args...)
		slog.Info("request completed", logArgs...)
	})
}

// LogWithContext returns a slog logger with stored correlation IDs from the context.
func LogWithContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(logContextKey("logger")).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// WithUserID adds a user ID to the context for correlation.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ctxUserID, userID)
}

// WithDeviceID adds a device ID to the context for correlation.
func WithDeviceID(ctx context.Context, deviceID string) context.Context {
	return context.WithValue(ctx, ctxDeviceID, deviceID)
}

// WithWorkflowID adds a workflow ID to the context for correlation.
func WithWorkflowID(ctx context.Context, workflowID string) context.Context {
	return context.WithValue(ctx, ctxWorkflowID, workflowID)
}

// WithSessionID adds a session ID to the context for correlation.
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, ctxSessionID, sessionID)
}

// GetRequestID retrieves the request ID from context.
func GetRequestID(ctx context.Context) string {
	if v, ok := ctx.Value(ctxRequestID).(string); ok {
		return v
	}
	return ""
}

// GetUserID retrieves the user ID from context.
func GetUserID(ctx context.Context) string {
	if v, ok := ctx.Value(ctxUserID).(string); ok {
		return v
	}
	return ""
}

// GetDeviceID retrieves the device ID from context.
func GetDeviceID(ctx context.Context) string {
	if v, ok := ctx.Value(ctxDeviceID).(string); ok {
		return v
	}
	return ""
}

// GetWorkflowID retrieves the workflow ID from context.
func GetWorkflowID(ctx context.Context) string {
	if v, ok := ctx.Value(ctxWorkflowID).(string); ok {
		return v
	}
	return ""
}
