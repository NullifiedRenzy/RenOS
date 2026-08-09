package api

import (
	"net/http"
	"runtime"
	"time"
)

func pprofHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message": "pprof endpoints available at /debug/pprof/",
			"note":    "Enable pprof by setting RENOS_PPROF=true",
		})
	}
}

func memoryStatsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats := getMemoryStats()
		writeJSON(w, http.StatusOK, stats)
	}
}

func serverStatsHandler() http.HandlerFunc {
	type ServerStats struct {
		Memory       MemoryStats `json:"memory"`
		Uptime       string      `json:"uptime"`
		UptimeSeconds float64    `json:"uptime_seconds"`
		Version      string      `json:"version"`
		GoVersion    string      `json:"go_version"`
		GoOS         string      `json:"go_os"`
		GoArch       string      `json:"go_arch"`
		NumCPU       int         `json:"num_cpu"`
	}
	startTime := time.Now()
	return func(w http.ResponseWriter, r *http.Request) {
		elapsed := time.Since(startTime)
		writeJSON(w, http.StatusOK, ServerStats{
			Memory:        getMemoryStats(),
			Uptime:        elapsed.String(),
			UptimeSeconds: elapsed.Seconds(),
			Version:       Version,
			GoVersion:     runtime.Version(),
			GoOS:          runtime.GOOS,
			GoArch:        runtime.GOARCH,
			NumCPU:        runtime.NumCPU(),
		})
	}
}

func connectionStatsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, peak, total := connTracker.stats()
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"current_connections": current,
			"peak_connections":    peak,
			"total_connections":   total,
		})
	}
}

func slowRequestStatsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"slow_request_count": getSlowRequestCount(),
			"threshold_ms":       slowLogger.threshold.Milliseconds(),
		})
	}
}

func gcStatsHandler() http.HandlerFunc {
	type GCStats struct {
		NumGC        uint32  `json:"num_gc"`
		PauseTotalNs uint64  `json:"pause_total_ns"`
		LastGC       string  `json:"last_gc"`
		NextGC       uint64  `json:"next_gc_bytes"`
		HeapAlloc    uint64  `json:"heap_alloc_bytes"`
		HeapInuse    uint64  `json:"heap_inuse_bytes"`
		HeapIdle     uint64  `json:"heap_idle_bytes"`
		HeapReleased uint64  `json:"heap_released_bytes"`
		HeapObjects  uint64  `json:"heap_objects"`
		StackInuse   uint64  `json:"stack_inuse_bytes"`
		MSpanInuse   uint64  `json:"mspan_inuse_bytes"`
		MCacheInuse  uint64  `json:"mcache_inuse_bytes"`
		TotalAlloc   uint64  `json:"total_alloc_bytes"`
		Sys          uint64  `json:"sys_bytes"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		writeJSON(w, http.StatusOK, GCStats{
			NumGC:        m.NumGC,
			PauseTotalNs: m.PauseTotalNs,
			LastGC:       time.Unix(0, int64(m.LastGC)).UTC().Format(time.RFC3339),
			NextGC:       m.NextGC,
			HeapAlloc:    m.HeapAlloc,
			HeapInuse:    m.HeapInuse,
			HeapIdle:     m.HeapIdle,
			HeapReleased: m.HeapReleased,
			HeapObjects:  m.HeapObjects,
			StackInuse:   m.StackInuse,
			MSpanInuse:   m.MSpanInuse,
			MCacheInuse:  m.MCacheInuse,
			TotalAlloc:   m.TotalAlloc,
			Sys:          m.Sys,
		})
	}
}

func configSummaryHandler(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"log_level":     deps.Cfg.LogLevel.String(),
			"external_url":  deps.Cfg.ExternalURL,
			"rate_limit": map[string]interface{}{
				"login":      deps.Cfg.RateLimitLogin,
				"api":        deps.Cfg.RateLimitAPI,
				"ws_connect": deps.Cfg.RateLimitWSConnect,
			},
			"max_body_size":      deps.Cfg.MaxBodySize,
			"max_ws_conns":       deps.Cfg.MaxWSConnsPerUser,
		})
	}
}

func envInfoHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"go_version":    runtime.Version(),
			"go_os":         runtime.GOOS,
			"go_arch":       runtime.GOARCH,
			"num_cpu":       runtime.NumCPU(),
			"num_goroutine": runtime.NumGoroutine(),
		})
	}
}

// --- Response compression hint ---

func compressionHintMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Vary", "Accept-Encoding")
		next.ServeHTTP(w, r)
	})
}

// --- Debug endpoint for request inspection ---

func debugRequestHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		info := map[string]interface{}{
			"method":      r.Method,
			"url":         r.URL.String(),
			"proto":       r.Proto,
			"remote_addr": r.RemoteAddr,
			"host":        r.Host,
			"content_type": r.Header.Get("Content-Type"),
			"user_agent":  r.Header.Get("User-Agent"),
			"referer":     r.Header.Get("Referer"),
			"accept":      r.Header.Get("Accept"),
			"auth":        r.Header.Get("Authorization") != "",
			"content_length": r.ContentLength,
		}
		writeJSON(w, http.StatusOK, info)
	}
}
