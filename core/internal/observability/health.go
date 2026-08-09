package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
)

type DependencyStatus string

const (
	StatusHealthy   DependencyStatus = "healthy"
	StatusDegraded  DependencyStatus = "degraded"
	StatusUnhealthy DependencyStatus = "unhealthy"
)

type DeepHealthResponse struct {
	Status      DependencyStatus    `json:"status"`
	Version     string              `json:"version"`
	Uptime      string              `json:"uptime"`
	Timestamp   time.Time           `json:"timestamp"`
	Dependencies []DependencyHealth `json:"dependencies"`
	GoStats     GoStats             `json:"go_stats"`
}

type DependencyHealth struct {
	Name    string            `json:"name"`
	Status  DependencyStatus  `json:"status"`
	Latency string            `json:"latency"`
	Error   string            `json:"error,omitempty"`
}

type GoStats struct {
	Goroutines  int     `json:"goroutines"`
	GCPercent   int     `json:"gc_percent"`
	MemoryMB    float64 `json:"memory_mb"`
	HeapObjects uint64  `json:"heap_objects"`
	NextGC      string  `json:"next_gc"`
}

var (
	startTime     = time.Now()
	serverVersion string
)

func SetVersion(v string) {
	serverVersion = v
}

func GetUptime() string {
	return time.Since(startTime).Round(time.Second).String()
}

// DeepHealthHandler performs deep health checks on all dependencies.
func DeepHealthHandler(db *pgxpool.Pool, nc *nats.Conn, rdb *redis.Client) http.HandlerFunc {
	var mu sync.Mutex
	return func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		var deps []DependencyHealth
		overall := StatusHealthy

		// PostgreSQL check
		dbDep := checkPostgres(db)
		deps = append(deps, dbDep)
		if dbDep.Status != StatusHealthy && overall != StatusUnhealthy {
			overall = dbDep.Status
		}

		// NATS check
		natsDep := checkNATS(nc)
		deps = append(deps, natsDep)
		if natsDep.Status != StatusHealthy && overall != StatusUnhealthy {
			overall = natsDep.Status
		}

		// Redis check
		redisDep := checkRedis(rdb)
		deps = append(deps, redisDep)
		if redisDep.Status != StatusHealthy && overall != StatusUnhealthy {
			overall = redisDep.Status
		}

		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		resp := DeepHealthResponse{
			Status:       overall,
			Version:      serverVersion,
			Uptime:       GetUptime(),
			Timestamp:    time.Now(),
			Dependencies: deps,
			GoStats: GoStats{
				Goroutines:  runtime.NumGoroutine(),
				GCPercent:   int(m.GCCPUFraction * 100),
				MemoryMB:    float64(m.Alloc) / 1024 / 1024,
				HeapObjects: m.HeapObjects,
				NextGC:      fmt.Sprintf("%.1f MB", float64(m.NextGC)/1024/1024),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		statusCode := http.StatusOK
		if overall == StatusUnhealthy {
			statusCode = http.StatusServiceUnavailable
		} else if overall == StatusDegraded {
			statusCode = http.StatusOK
		}
		w.WriteHeader(statusCode)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			slog.Error("failed to encode deep health response", "error", err)
		}
	}
}

// LivezHandler returns a simple liveness probe (always OK if server is running).
func LivezHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"status":"ok"}`)
	}
}

// StartupzHandler returns readiness of dependencies for startup probe.
func StartupzHandler(db *pgxpool.Pool, rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var allReady bool
		status := "ok"
		errors := []string{}

		if db != nil {
			if err := db.Ping(r.Context()); err != nil {
				allReady = false
				errors = append(errors, fmt.Sprintf("postgres: %v", err))
			}
		}
		if rdb != nil {
			if err := rdb.Ping(r.Context()).Err(); err != nil {
				allReady = false
				errors = append(errors, fmt.Sprintf("redis: %v", err))
			}
		}

		if !allReady {
			status = "not_ready"
		}

		w.Header().Set("Content-Type", "application/json")
		if !allReady {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": status,
			"errors": errors,
		})
	}
}

func checkPostgres(db *pgxpool.Pool) DependencyHealth {
	if db == nil {
		return DependencyHealth{Name: "postgres", Status: StatusUnhealthy, Error: "not initialized"}
	}
	start := time.Now()
	if err := db.Ping(context.Background()); err != nil {
		return DependencyHealth{
			Name: "postgres", Status: StatusUnhealthy, Error: err.Error(),
			Latency: time.Since(start).String(),
		}
	}
	return DependencyHealth{
		Name: "postgres", Status: StatusHealthy,
		Latency: time.Since(start).String(),
	}
}

func checkNATS(nc *nats.Conn) DependencyHealth {
	if nc == nil {
		return DependencyHealth{Name: "nats", Status: StatusUnhealthy, Error: "not initialized"}
	}
	start := time.Now()
	status := nc.Status()
	switch status {
	case nats.CONNECTED:
		return DependencyHealth{
			Name: "nats", Status: StatusHealthy,
			Latency: time.Since(start).String(),
		}
	case nats.RECONNECTING:
		return DependencyHealth{
			Name: "nats", Status: StatusDegraded, Error: "reconnecting",
			Latency: time.Since(start).String(),
		}
	default:
		return DependencyHealth{
			Name: "nats", Status: StatusUnhealthy,
			Error: fmt.Sprintf("connection state: %s", status),
			Latency: time.Since(start).String(),
		}
	}
}

func checkRedis(rdb *redis.Client) DependencyHealth {
	if rdb == nil {
		return DependencyHealth{Name: "redis", Status: StatusUnhealthy, Error: "not initialized"}
	}
	start := time.Now()
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return DependencyHealth{
			Name: "redis", Status: StatusUnhealthy, Error: err.Error(),
			Latency: time.Since(start).String(),
		}
	}
	return DependencyHealth{
		Name: "redis", Status: StatusHealthy,
		Latency: time.Since(start).String(),
	}
}
