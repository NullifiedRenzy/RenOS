package observability

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
)

// Options holds all dependencies for initializing observability subsystems.
type Options struct {
	DB      *pgxpool.Pool
	NC      *nats.Conn
	RDB     *redis.Client
	Version string
}

// InitAll initializes all observability subsystems (tracing, metrics, SLO, health)
// and returns the HTTP mux with all observability endpoints registered,
// plus a shutdown function.
func InitAll(cfg *ObservabilityConfig, opts *Options) (http.Handler, func()) {
	var cleanupFuncs []func()

	// 1. Tracing
	tracingCleanup := InitTracing(cfg)
	cleanupFuncs = append(cleanupFuncs, tracingCleanup)

	// 2. Metrics
	InitMetrics()

	// 3. SLO
	GetSLO()
	StartSLOAggregator(1 * time.Minute)

	// 4. Version
	SetVersion(opts.Version)

	// 5. Observability HTTP mux
	mux := http.NewServeMux()

	// ── Metrics endpoint ──
	if cfg.Metrics.Enabled {
		mux.Handle(cfg.Metrics.Path, MetricsMiddleware(MetricsHandler()))
		slog.Info("metrics endpoint registered", "path", cfg.Metrics.Path)
	}

	// ── Health endpoints ──
	mux.HandleFunc("/livez", LivezHandler())
	mux.HandleFunc("/startupz", StartupzHandler(opts.DB, opts.RDB))
	if cfg.Health.DeepEnabled {
		mux.Handle(cfg.Health.Path, MetricsMiddleware(http.HandlerFunc(DeepHealthHandler(opts.DB, opts.NC, opts.RDB))))
		slog.Info("deep health endpoint registered", "path", cfg.Health.Path)
	}

	// ── SLO endpoint ──
	if cfg.SLO.Enabled {
		mux.Handle(cfg.SLO.Path, MetricsMiddleware(SLOHandler()))
		slog.Info("slo endpoint registered", "path", cfg.SLO.Path)
	}

	return mux, func() {
		for _, f := range cleanupFuncs {
			f()
		}
	}
}

// StartSLOAggregator periodically evaluates and exports SLO metrics.
func StartSLOAggregator(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			s := GetSLO()
			statuses := s.Evaluate()
			slog.Debug("slo aggregation", "count", len(statuses))
		}
	}()
}
