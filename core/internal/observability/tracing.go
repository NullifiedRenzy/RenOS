package observability

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	Tracer trace.Tracer
	tp     *sdktrace.TracerProvider
)

func init() {
	Tracer = otel.Tracer("renos-core")
}

// InitTracing initializes the OpenTelemetry tracer provider based on config.
func InitTracing(cfg *ObservabilityConfig) func() {
	if !cfg.Tracing.Enabled {
		Tracer = otel.Tracer(cfg.Tracing.ServiceName)
		return func() {}
	}

	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(cfg.Tracing.ServiceName),
		semconv.DeploymentEnvironment(cfg.Tracing.Environment),
	)

	var exporter sdktrace.SpanExporter
	var err error

	switch cfg.Tracing.Exporter {
	case "otlp", "otlphttp":
		exporter, err = otlptracehttp.New(context.Background(),
			otlptracehttp.WithEndpointURL(cfg.Tracing.Endpoint),
		)
	case "zipkin":
		exporter, err = zipkin.New(cfg.Tracing.Endpoint)
	case "stdout":
		exporter, err = stdouttrace.New(
			stdouttrace.WithPrettyPrint(),
		)
	default:
		slog.Warn("unknown tracing exporter, falling back to stdout", "exporter", cfg.Tracing.Exporter)
		exporter, err = stdouttrace.New()
	}

	if err != nil {
		slog.Error("failed to create tracing exporter", "error", err, "exporter", cfg.Tracing.Exporter)
		Tracer = otel.Tracer(cfg.Tracing.ServiceName)
		return func() {}
	}

	tp = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.Tracing.SampleRate)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	Tracer = tp.Tracer(cfg.Tracing.ServiceName)

	slog.Info("tracing initialized",
		"exporter", cfg.Tracing.Exporter,
		"sample_rate", cfg.Tracing.SampleRate,
	)

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			slog.Error("failed to shutdown tracer provider", "error", err)
		}
	}
}

// TracingMiddleware adds OpenTelemetry tracing to HTTP requests.
func TracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if Tracer == nil {
			next.ServeHTTP(w, r)
			return
		}

		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		spanName := r.Method + " " + r.URL.Path
		ctx, span := Tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPRequestMethodKey.String(r.Method),
				semconv.URLFullKey.String(r.URL.String()),
				semconv.HTTPRouteKey.String(r.URL.Path),
				semconv.UserAgentOriginalKey.String(r.UserAgent()),
			),
		)
		defer span.End()

		// Inject trace context into response headers
		otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(w.Header()))

		// Add trace ID to context for logging
		spanCtx := trace.SpanContextFromContext(ctx)
		if spanCtx.HasTraceID() {
			ctx = context.WithValue(ctx, traceIDKey, spanCtx.TraceID().String())
			ctx = context.WithValue(ctx, spanIDKey, spanCtx.SpanID().String())
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type contextKey string

const traceIDKey contextKey = "trace_id"
const spanIDKey contextKey = "span_id"

// GetTraceID extracts the trace ID from context.
func GetTraceID(ctx context.Context) string {
	if v, ok := ctx.Value(traceIDKey).(string); ok {
		return v
	}
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasTraceID() {
		return spanCtx.TraceID().String()
	}
	return ""
}

// GetSpanID extracts the span ID from context.
func GetSpanID(ctx context.Context) string {
	if v, ok := ctx.Value(spanIDKey).(string); ok {
		return v
	}
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.HasSpanID() {
		return spanCtx.SpanID().String()
	}
	return ""
}

// StartDBSpan creates a span for a database operation.
func StartDBSpan(ctx context.Context, operation, query string) (context.Context, trace.Span) {
	ctx, span := Tracer.Start(ctx, "db."+operation,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			semconv.DBSystemPostgreSQL,
			attribute.String("db.operation", operation),
			attribute.String("db.query", truncate(query, 200)),
		),
	)
	return ctx, span
}

// StartNATSSpan creates a span for a NATS publish operation.
func StartNATSSpan(ctx context.Context, subject string) (context.Context, trace.Span) {
	ctx, span := Tracer.Start(ctx, "nats.publish",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "nats"),
			attribute.String("messaging.destination", subject),
		),
	)
	return ctx, span
}

// StartWSSpan creates a span for a WebSocket message.
func StartWSSpan(ctx context.Context, msgType string) (context.Context, trace.Span) {
	ctx, span := Tracer.Start(ctx, "ws."+msgType,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			attribute.String("websocket.message_type", msgType),
		),
	)
	return ctx, span
}

// StartWorkflowSpan creates a span for a workflow execution.
func StartWorkflowSpan(ctx context.Context, workflowID, stepType string) (context.Context, trace.Span) {
	ctx, span := Tracer.Start(ctx, "workflow."+stepType,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("workflow.id", workflowID),
			attribute.String("workflow.step_type", stepType),
		),
	)
	return ctx, span
}

// RecordSpanError records an error on a span.
func RecordSpanError(span trace.Span, err error) {
	if err != nil && span != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
