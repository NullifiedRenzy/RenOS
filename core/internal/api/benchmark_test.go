package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Benchmark writeJSON encoding path
func BenchmarkWriteJSON(b *testing.B) {
	data := struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Status  string `json:"status"`
		Devices int    `json:"devices"`
	}{
		ID:      "dev-001",
		Name:    "test-device",
		Status:  "online",
		Devices: 1000,
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		writeJSON(rec, http.StatusOK, data)
	}
}

// Benchmark writeJSON with large payload
func BenchmarkWriteJSONLarge(b *testing.B) {
	devices := make([]map[string]interface{}, 100)
	for i := range devices {
		devices[i] = map[string]interface{}{
			"id":     "dev-" + string(rune('A'+i%26)) + string(rune('0'+i/26)),
			"name":   "device-" + string(rune('A'+i%26)),
			"status": "online",
			"os":     "Linux 6.8",
			"tags":   []string{"production", "web", "us-east"},
		}
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		writeJSON(rec, http.StatusOK, map[string]interface{}{
			"devices": devices,
			"total":   len(devices),
		})
	}
}

// Benchmark writeError path
func BenchmarkWriteError(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		writeError(rec, http.StatusNotFound, "NOT_FOUND", "device not found")
	}
}

// Benchmark sanitizeErrorMessage
func BenchmarkSanitizeErrorMessage(b *testing.B) {
	messages := []string{
		"connection refused on 127.0.0.1:5432",
		"pq: relation does not exist",
		"no such host: db.internal",
		"command is required",
		"device not found",
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sanitizeErrorMessage(messages[i%len(messages)])
	}
}

// Benchmark rate limiter Allow
func BenchmarkRateLimiterAllow(b *testing.B) {
	rl := &rateLimiter{
		clients:  make(map[string]*clientLimiter),
		rate:     1000,
		burst:    2000,
		cleanupT: time.Hour,
		maxSize:  10000,
	}

	keys := make([]string, 100)
	for i := range keys {
		keys[i] = "client-" + string(rune('A'+i%26)) + string(rune('0'+i/26))
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rl.Allow(keys[i%len(keys)])
	}
}

// Benchmark JSON encoding (simulates hot path)
func BenchmarkJSONMarshalResponse(b *testing.B) {
	resp := map[string]interface{}{
		"data": map[string]interface{}{
			"id":     "dev-001",
			"status": "online",
			"config": map[string]interface{}{
				"interval_ms": 5000,
				"timeout_s":   30,
				"retries":     3,
			},
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		json.Marshal(resp)
	}
}

// Benchmark concurrent writeJSON (simulates API server load)
func BenchmarkConcurrentWriteJSON(b *testing.B) {
	data := map[string]string{"status": "ok", "version": "1.0.0"}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rec := httptest.NewRecorder()
			writeJSON(rec, http.StatusOK, data)
		}
	})
}

// Benchmark writeJSON of a realistic 500-device list (GET /devices default cap).
func BenchmarkWriteJSONDeviceList(b *testing.B) {
	devices := make([]map[string]interface{}, 500)
	for i := range devices {
		devices[i] = map[string]interface{}{
			"id":         "01234567-89ab-cdef-0123-456789abcdef",
			"user_id":    "aaaaaaaa-bbbb-cccc-dddd-eeeeffff0000",
			"hostname":   "node-" + string(rune('A'+i%26)),
			"os":         "Linux 6.8.0",
			"arch":       "x86_64",
			"agent_version": "1.0.0",
			"enrolled_at": "2026-08-06T12:00:00Z",
			"last_seen":   "2026-08-06T12:00:00Z",
			"status":      "online",
			"labels":      map[string]string{"env": "prod", "rack": "a1"},
			"capabilities": []string{"exec", "containers", "metrics"},
			"created_at":  "2026-08-06T12:00:00Z",
			"updated_at":  "2026-08-06T12:00:00Z",
		}
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		writeJSON(rec, http.StatusOK, map[string]interface{}{"devices": devices})
		b.SetBytes(int64(rec.Body.Len()))
	}
}

// Benchmark writeJSON of a realistic 200-job list (GET /jobs default cap).
func BenchmarkWriteJSONJobList(b *testing.B) {
	jobs := make([]map[string]interface{}, 200)
	for i := range jobs {
		jobs[i] = map[string]interface{}{
			"id":                 "01234567-89ab-cdef-0123-456789abcdef",
			"type":               "run_command",
			"status":             "completed",
			"total_devices":      50,
			"completed_devices":  50,
			"failed_devices":     0,
			"retries":            0,
			"max_retries":        3,
			"created_by":         "aaaaaaaa-bbbb-cccc-dddd-eeeeffff0000",
			"created_at":         "2026-08-06T12:00:00Z",
			"started_at":         "2026-08-06T12:00:01Z",
			"completed_at":       "2026-08-06T12:00:42Z",
		}
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		writeJSON(rec, http.StatusOK, map[string]interface{}{"jobs": jobs})
		b.SetBytes(int64(rec.Body.Len()))
	}
}

// Benchmark writeJSON of a bounded 200-notification inbox list.
func BenchmarkWriteJSONNotificationList(b *testing.B) {
	notes := make([]map[string]interface{}, 200)
	for i := range notes {
		notes[i] = map[string]interface{}{
			"id":         "01234567-89ab-cdef-0123-456789abcdef",
			"user_id":    "aaaaaaaa-bbbb-cccc-dddd-eeeeffff0000",
			"type":       "workflow",
			"title":      "Workflow completed",
			"body":       "The 'provision' workflow finished on 12 devices.",
			"is_read":    i%2 == 0,
			"created_at": "2026-08-06T12:00:00Z",
		}
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		writeJSON(rec, http.StatusOK, notes)
		b.SetBytes(int64(rec.Body.Len()))
	}
}
