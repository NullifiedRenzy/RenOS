package observability

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type SLOType string

const (
	SLODefault      SLOType = "default"
	SLOAvailability SLOType = "availability"
	SLOLatency      SLOType = "latency"
	SLOErrorBudget  SLOType = "error_budget"
)

type SLOTarget struct {
	Name         string
	Type         SLOType
	Target       float64
	Window       time.Duration
	Availability float64
	ErrorBudget  float64
	P99Latency   float64
}

type SLO struct {
	mu      sync.RWMutex
	targets []SLOTarget
	events  []SLOEvent
}

type SLOEvent struct {
	Name      string    `json:"name"`
	Success   bool      `json:"success"`
	Duration  float64   `json:"duration_ms"`
	Timestamp time.Time `json:"timestamp"`
}

type SLOStatus struct {
	Name         string  `json:"name"`
	Target       float64 `json:"target"`
	Window       string  `json:"window"`
	Availability float64 `json:"availability"`
	ErrorBudget  float64 `json:"error_budget"`
	P99Latency   float64 `json:"p99_latency_ms"`
	Passing      bool    `json:"passing"`
}

var sloInstance *SLO
var sloOnce sync.Once

func GetSLO() *SLO {
	sloOnce.Do(func() {
		sloInstance = &SLO{
			targets: []SLOTarget{
				{Name: "api_availability", Type: SLOAvailability, Target: 0.995, Window: 30 * 24 * time.Hour},
				{Name: "api_latency_p99", Type: SLOLatency, Target: 500, Window: 30 * 24 * time.Hour, P99Latency: 500},
				{Name: "db_query_latency_p99", Type: SLOLatency, Target: 100, Window: 30 * 24 * time.Hour, P99Latency: 100},
				{Name: "agent_heartbeat_reliability", Type: SLOAvailability, Target: 0.99, Window: 24 * time.Hour},
				{Name: "nats_delivery_reliability", Type: SLOAvailability, Target: 0.999, Window: 30 * 24 * time.Hour},
				{Name: "ai_response_latency_p99", Type: SLOLatency, Target: 30000, Window: 24 * time.Hour, P99Latency: 30000},
			},
		}
	})
	return sloInstance
}

func (s *SLO) RecordEvent(name string, success bool, durationMs float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events = append(s.events, SLOEvent{
		Name:      name,
		Success:   success,
		Duration:  durationMs,
		Timestamp: time.Now(),
	})

	// Trim events older than 30 days
	cutoff := time.Now().Add(-30 * 24 * time.Hour)
	for len(s.events) > 0 && s.events[0].Timestamp.Before(cutoff) {
		s.events = append([]SLOEvent{}, s.events[1:]...)
	}
}

func (s *SLO) Evaluate() []SLOStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	var statuses []SLOStatus

	for _, target := range s.targets {
		windowStart := now.Add(-target.Window)
		var successCount, totalCount int
		var latencies []float64

		for _, e := range s.events {
			if e.Name != target.Name || e.Timestamp.Before(windowStart) {
				continue
			}
			totalCount++
			if e.Success {
				successCount++
			}
			latencies = append(latencies, e.Duration)
		}

		availability := 1.0
		if totalCount > 0 {
			availability = float64(successCount) / float64(totalCount)
		}

		p99 := 0.0
		if len(latencies) > 0 {
			p99 = percentile(latencies, 99)
		}

		sloTarget := target.Target
		errorBudget := 1.0 - (1.0-availability)/(1.0-sloTarget)
		if target.Type == SLOLatency {
			sloTarget = target.P99Latency
			errorBudget = 1.0 - (p99 / sloTarget)
			if p99 < sloTarget {
				errorBudget = 1.0
			}
		}
		if errorBudget < 0 {
			errorBudget = 0
		}
		if errorBudget > 1 {
			errorBudget = 1
		}

		passing := errorBudget > 0

		statuses = append(statuses, SLOStatus{
			Name:         target.Name,
			Target:       sloTarget,
			Window:       target.Window.String(),
			Availability: availability,
			ErrorBudget:  errorBudget,
			P99Latency:   p99,
			Passing:      passing,
		})

		// Update Prometheus metrics
		if m := GetMetrics(); m != nil {
			m.SLOAvailability.WithLabelValues(target.Name, target.Window.String()).Set(availability)
			m.SLOErrorBudget.WithLabelValues(target.Name, target.Window.String()).Set(errorBudget)
			m.SLOLatency.WithLabelValues(target.Name).Set(p99)
		}
	}

	return statuses
}

func percentile(values []float64, p int) float64 {
	if len(values) == 0 {
		return 0
	}

	// Simple percentile: sort and pick index
	// For production, use a proper sorting algorithm
	n := len(values)
	// Use a simple selection approach for p99
	idx := (p * n) / 100
	if idx >= n {
		idx = n - 1
	}

	// Quickselect-ish: just find the value at idx
	return quickSelect(values, idx)
}

func quickSelect(arr []float64, k int) float64 {
	if len(arr) == 0 {
		return 0
	}
	if len(arr) == 1 {
		return arr[0]
	}

	// Simple approach: copy and sort only up to k+1
	// For small slices this is fine
	sorted := make([]float64, len(arr))
	copy(sorted, arr)

	// Bubble up the k-th largest using partial sort
	for i := 0; i <= k && i < len(sorted); i++ {
		maxIdx := i
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j] > sorted[maxIdx] {
				maxIdx = j
			}
		}
		sorted[i], sorted[maxIdx] = sorted[maxIdx], sorted[i]
	}

	if k >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	return sorted[k]
}

// SLOHandler returns the SLO status endpoint handler.
func SLOHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := GetSLO()
		statuses := s.Evaluate()

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(statuses); err != nil {
			slog.Error("failed to encode SLO status", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
