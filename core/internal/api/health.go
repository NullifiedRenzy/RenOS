package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
)

var Version = "dev"
var Commit = "unknown"
var BuildTime = "unknown"

func healthHandler(pool *pgxpool.Pool, nc *nats.Conn, rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"version": Version,
		})
	}
}

func versionHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"version":    Version,
			"commit":     Commit,
			"build_time": BuildTime,
		})
	}
}

func readyHandler(pool *pgxpool.Pool, nc *nats.Conn, rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		checks := readyChecks(pool, nc, rdb)

		allOK := true
		if checks.Database.Status != "ok" {
			allOK = false
		}
		if checks.NATS.Status != "ok" {
			allOK = false
		}
		if rdb != nil && checks.Redis.Status != "ok" {
			allOK = false
		}

		resp := map[string]interface{}{
			"version": Version,
			"checks":  checks,
		}

		status := http.StatusOK
		if !allOK {
			resp["status"] = "degraded"
			status = http.StatusServiceUnavailable
		} else {
			resp["status"] = "ok"
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(resp)
	}
}

type dependencyCheck struct {
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
	LatencyMs int64  `json:"latency_ms"`
}

type readyCheckSet struct {
	Database dependencyCheck `json:"database"`
	NATS     dependencyCheck `json:"nats"`
	Redis    dependencyCheck `json:"redis,omitempty"`
}

func readyChecks(pool *pgxpool.Pool, nc *nats.Conn, rdb *redis.Client) readyCheckSet {
	checks := readyCheckSet{
		Database: checkDatabase(pool),
		NATS:     checkNATS(nc),
	}

	if rdb != nil {
		checks.Redis = checkRedis(rdb)
	}

	return checks
}

func checkDatabase(pool *pgxpool.Pool) dependencyCheck {
	if pool == nil {
		return dependencyCheck{
			Status: "error",
			Error:  "not connected",
		}
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := pool.Ping(ctx)
	duration := time.Since(start).Milliseconds()

	if err != nil {
		return dependencyCheck{
			Status:    "error",
			Error:     err.Error(),
			LatencyMs: duration,
		}
	}

	return dependencyCheck{
		Status:    "ok",
		LatencyMs: duration,
	}
}

func checkNATS(nc *nats.Conn) dependencyCheck {
	if nc == nil {
		return dependencyCheck{
			Status: "error",
			Error:  "not connected",
		}
	}

	start := time.Now()
	connStatus := nc.Status()
	duration := time.Since(start).Milliseconds()

	if connStatus != nats.CONNECTED {
		return dependencyCheck{
			Status: "error",
			Error:  connStatus.String(),
		}
	}

	return dependencyCheck{
		Status:    "ok",
		LatencyMs: duration,
	}
}

func checkRedis(rdb *redis.Client) dependencyCheck {
	if rdb == nil {
		return dependencyCheck{
			Status: "error",
			Error:  "not connected",
		}
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := rdb.Ping(ctx).Err()
	duration := time.Since(start).Milliseconds()

	if err != nil {
		return dependencyCheck{
			Status:    "error",
			Error:     err.Error(),
			LatencyMs: duration,
		}
	}

	return dependencyCheck{
		Status:    "ok",
		LatencyMs: duration,
	}
}

// StartHealthMonitor periodically checks all dependencies and logs their status.
// This provides connection flapping detection and background health visibility.
func StartHealthMonitor(ctx context.Context, pool *pgxpool.Pool, nc *nats.Conn, rdb *redis.Client, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Run immediately on start
		checks := readyChecks(pool, nc, rdb)
		logHealthStatus(checks)

		for {
			select {
			case <-ctx.Done():
				slog.Info("health monitor stopped")
				return
			case <-ticker.C:
				checks := readyChecks(pool, nc, rdb)
				logHealthStatus(checks)
			}
		}
	}()
}

func logHealthStatus(checks readyCheckSet) {
	if checks.Database.Status != "ok" {
		slog.Error("health: database unhealthy", "error", checks.Database.Error, "latency_ms", checks.Database.LatencyMs)
	}
	if checks.NATS.Status != "ok" {
		slog.Error("health: nats unhealthy", "error", checks.NATS.Error)
	}
	if checks.Redis.Status != "ok" {
		slog.Error("health: redis unhealthy", "error", checks.Redis.Error)
	}
}
