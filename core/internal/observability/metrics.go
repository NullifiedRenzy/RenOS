package observability

import (
	"bufio"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metric collectors for the application.
type Metrics struct {
	// API metrics
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPRequestsActive  prometheus.Gauge
	HTTPResponseSize    *prometheus.HistogramVec
	HTTPErrorTotal      *prometheus.CounterVec

	// Database metrics
	DBQueriesTotal     *prometheus.CounterVec
	DBQueryDuration    *prometheus.HistogramVec
	DBPoolActive       prometheus.Gauge
	DBPoolIdle         prometheus.Gauge
	DBPoolWaitCount    prometheus.Counter
	DBPoolWaitDuration prometheus.Histogram

	// NATS metrics
	NATSPublishTotal   *prometheus.CounterVec
	NATSSubscribeTotal *prometheus.CounterVec
	NATSReconnects     prometheus.Counter
	NATSDisconnects    prometheus.Counter

	// Redis metrics
	RedisCommandDuration *prometheus.HistogramVec
	RedisErrorsTotal     *prometheus.CounterVec

	// Agent metrics
	AgentsOnline          prometheus.Gauge
	AgentsTotal           prometheus.Gauge
	AgentHeartbeatLatency *prometheus.HistogramVec
	AgentCommandLatency   *prometheus.HistogramVec

	// Terminal metrics
	TerminalActiveSessions prometheus.Gauge
	TerminalBytesTotal     *prometheus.CounterVec

	// RenAI metrics
	AIPromptCount      *prometheus.CounterVec
	AILatency          *prometheus.HistogramVec
	AITokensTotal      *prometheus.CounterVec
	AIProviderFailures *prometheus.CounterVec

	// Plugin metrics
	PluginExecutionsTotal   *prometheus.CounterVec
	PluginExecutionDuration *prometheus.HistogramVec
	PluginWorkersActive     prometheus.Gauge

	// Workflow metrics
	WorkflowExecutionsTotal *prometheus.CounterVec
	WorkflowDuration        *prometheus.HistogramVec
	WorkflowStepsTotal      *prometheus.CounterVec

	// Circuit breaker metrics
	CircuitBreakerTrips *prometheus.CounterVec

	// SLO metrics
	SLOAvailability *prometheus.GaugeVec
	SLOErrorBudget  *prometheus.GaugeVec
	SLOLatency      *prometheus.GaugeVec

	registry *prometheus.Registry
	once     sync.Once
}

var metrics *Metrics

// InitMetrics creates and registers all Prometheus metric collectors.
func InitMetrics() *Metrics {
	m := &Metrics{
		registry: prometheus.NewRegistry(),
	}

	// Register Go runtime and process metrics
	m.registry.MustRegister(collectors.NewGoCollector())
	m.registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	// ── API metrics ──
	m.HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "renos_http_requests_total",
			Help: "Total number of HTTP requests by method, path, and status",
		},
		[]string{"method", "path", "status"},
	)
	m.HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "renos_http_request_duration_ms",
			Help:    "HTTP request latency in milliseconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
	m.HTTPRequestsActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "renos_http_requests_active",
			Help: "Number of active HTTP requests",
		},
	)
	m.HTTPResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "renos_http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: []float64{256, 1024, 4096, 16384, 65536, 262144, 1048576},
		},
		[]string{"method", "path"},
	)
	m.HTTPErrorTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "renos_http_errors_total",
			Help: "Total number of HTTP errors by code",
		},
		[]string{"status", "path"},
	)

	// ── Database metrics ──
	m.DBQueriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "renos_db_queries_total",
			Help: "Total number of database queries by operation",
		},
		[]string{"operation"},
	)
	m.DBQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "renos_db_query_duration_ms",
			Help:    "Database query latency in milliseconds",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 5000},
		},
		[]string{"operation"},
	)
	m.DBPoolActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "renos_db_pool_active_connections",
			Help: "Number of active database connections",
		},
	)
	m.DBPoolIdle = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "renos_db_pool_idle_connections",
			Help: "Number of idle database connections",
		},
	)
	m.DBPoolWaitCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "renos_db_pool_wait_count_total",
			Help: "Total number of waits for a database connection",
		},
	)
	m.DBPoolWaitDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "renos_db_pool_wait_duration_ms",
			Help:    "Time waited for a database connection in milliseconds",
			Buckets: []float64{1, 5, 10, 25, 50, 100},
		},
	)

	// ── NATS metrics ──
	m.NATSPublishTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "renos_nats_publish_total",
			Help: "Total number of NATS publish operations by subject",
		},
		[]string{"subject"},
	)
	m.NATSSubscribeTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "renos_nats_subscribe_total",
			Help: "Total number of NATS messages received by subject",
		},
		[]string{"subject"},
	)
	m.NATSReconnects = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "renos_nats_reconnects_total",
			Help: "Total number of NATS reconnections",
		},
	)
	m.NATSDisconnects = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "renos_nats_disconnects_total",
			Help: "Total number of NATS disconnections",
		},
	)

	// ── Redis metrics ──
	m.RedisCommandDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "renos_redis_command_duration_ms",
			Help:    "Redis command latency in milliseconds",
			Buckets: []float64{0.5, 1, 2, 5, 10, 25, 50, 100},
		},
		[]string{"command"},
	)
	m.RedisErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "renos_redis_errors_total",
			Help: "Total number of Redis errors by command",
		},
		[]string{"command"},
	)

	// ── Agent metrics ──
	m.AgentsOnline = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "renos_agents_online",
			Help: "Number of currently connected agents",
		},
	)
	m.AgentsTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "renos_agents_total",
			Help: "Total number of registered agents",
		},
	)
	m.AgentHeartbeatLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "renos_agent_heartbeat_latency_ms",
			Help:    "Agent heartbeat latency in milliseconds",
			Buckets: []float64{10, 50, 100, 250, 500, 1000, 5000},
		},
		[]string{"agent_id"},
	)
	m.AgentCommandLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "renos_agent_command_latency_ms",
			Help:    "Agent command round-trip latency in milliseconds",
			Buckets: []float64{50, 100, 250, 500, 1000, 2500, 5000, 10000},
		},
		[]string{"agent_id", "command_type"},
	)

	// ── Terminal metrics ──
	m.TerminalActiveSessions = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "renos_terminal_active_sessions",
			Help: "Number of active terminal sessions",
		},
	)
	m.TerminalBytesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "renos_terminal_bytes_total",
			Help: "Total bytes transferred through terminal sessions",
		},
		[]string{"direction"},
	)

	// ── RenAI metrics ──
	m.AIPromptCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "renos_ai_prompts_total",
			Help: "Total number of AI prompts by provider",
		},
		[]string{"provider", "model"},
	)
	m.AILatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "renos_ai_latency_ms",
			Help:    "AI request latency in milliseconds",
			Buckets: []float64{100, 500, 1000, 2500, 5000, 10000, 30000, 60000},
		},
		[]string{"provider"},
	)
	m.AITokensTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "renos_ai_tokens_total",
			Help: "Total number of tokens processed by AI",
		},
		[]string{"provider", "type"},
	)
	m.AIProviderFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "renos_ai_provider_failures_total",
			Help: "Total number of AI provider failures by provider",
		},
		[]string{"provider"},
	)

	// ── Plugin metrics ──
	m.PluginExecutionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "renos_plugin_executions_total",
			Help: "Total number of plugin executions by plugin",
		},
		[]string{"plugin_id", "action"},
	)
	m.PluginExecutionDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "renos_plugin_execution_duration_ms",
			Help:    "Plugin execution duration in milliseconds",
			Buckets: []float64{10, 50, 100, 500, 1000, 5000, 10000, 30000},
		},
		[]string{"plugin_id"},
	)
	m.PluginWorkersActive = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "renos_plugin_workers_active",
			Help: "Number of active plugin workers",
		},
	)

	// ── Workflow metrics ──
	m.WorkflowExecutionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "renos_workflow_executions_total",
			Help: "Total number of workflow executions by status",
		},
		[]string{"status"},
	)
	m.WorkflowDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "renos_workflow_duration_ms",
			Help:    "Workflow execution duration in milliseconds",
			Buckets: []float64{100, 500, 1000, 5000, 10000, 30000, 60000, 300000},
		},
		[]string{"workflow_type"},
	)
	m.WorkflowStepsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "renos_workflow_steps_total",
			Help: "Total number of workflow steps processed",
		},
		[]string{"step_type", "status"},
	)

	// ── Circuit breaker metrics ──
	m.CircuitBreakerTrips = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "renos_circuit_breaker_trips_total",
			Help: "Total number of circuit breaker trips by device",
		},
		[]string{"device_id"},
	)

	// ── SLO metrics ──
	m.SLOAvailability = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "renos_slo_availability",
			Help: "SLO availability ratio (0.0-1.0) by SLO name",
		},
		[]string{"slo_name", "window"},
	)
	m.SLOErrorBudget = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "renos_slo_error_budget",
			Help: "Remaining SLO error budget (0.0-1.0) by SLO name",
		},
		[]string{"slo_name", "window"},
	)
	m.SLOLatency = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "renos_slo_latency_p99",
			Help: "P99 latency in milliseconds by SLO name",
		},
		[]string{"slo_name"},
	)

	// Register all collectors
	m.registry.MustRegister(
		m.HTTPRequestsTotal,
		m.HTTPRequestDuration,
		m.HTTPRequestsActive,
		m.HTTPResponseSize,
		m.HTTPErrorTotal,
		m.DBQueriesTotal,
		m.DBQueryDuration,
		m.DBPoolActive,
		m.DBPoolIdle,
		m.DBPoolWaitCount,
		m.DBPoolWaitDuration,
		m.NATSPublishTotal,
		m.NATSSubscribeTotal,
		m.NATSReconnects,
		m.NATSDisconnects,
		m.RedisCommandDuration,
		m.RedisErrorsTotal,
		m.AgentsOnline,
		m.AgentsTotal,
		m.AgentHeartbeatLatency,
		m.AgentCommandLatency,
		m.TerminalActiveSessions,
		m.TerminalBytesTotal,
		m.AIPromptCount,
		m.AILatency,
		m.AITokensTotal,
		m.AIProviderFailures,
		m.PluginExecutionsTotal,
		m.PluginExecutionDuration,
		m.PluginWorkersActive,
		m.WorkflowExecutionsTotal,
		m.WorkflowDuration,
		m.WorkflowStepsTotal,
		m.CircuitBreakerTrips,
		m.SLOAvailability,
		m.SLOErrorBudget,
		m.SLOLatency,
	)

	metrics = m

	slog.Info("prometheus metrics initialized", "collectors", 35)

	return m
}

// MetricsHandler returns the Prometheus HTTP handler.
func MetricsHandler() http.Handler {
	if metrics == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("metrics not initialized"))
		})
	}
	return promhttp.HandlerFor(metrics.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// GetMetrics returns the global Metrics instance.
func GetMetrics() *Metrics {
	return metrics
}

// MetricsMiddleware records HTTP request metrics.
func MetricsMiddleware(next http.Handler) http.Handler {
	if metrics == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metrics.HTTPRequestsActive.Inc()
		defer metrics.HTTPRequestsActive.Dec()

		start := time.Now()
		ww := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(ww, r)

		duration := time.Since(start).Milliseconds()
		status := strconv.Itoa(ww.statusCode)
		method := r.Method
		path := r.URL.Path

		metrics.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(method, path).Observe(float64(duration))
		metrics.HTTPResponseSize.WithLabelValues(method, path).Observe(float64(ww.size))

		if ww.statusCode >= 400 {
			metrics.HTTPErrorTotal.WithLabelValues(status, path).Inc()
		}
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.NewResponseController(rw.ResponseWriter).Hijack()
}
