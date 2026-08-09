package execctrl

import (
	"context"
	"sync"
	"time"

	"github.com/renos-project/renos/core/internal/db"
)

type CircuitBreaker struct {
	mu        sync.Mutex
	failures  map[string]int
	unstable  map[string]time.Time
	threshold int
	resetAfter time.Duration
	deviceRepo *db.DeviceRepository
}

func NewCircuitBreaker(threshold int, resetAfter time.Duration, deviceRepo *db.DeviceRepository) *CircuitBreaker {
	cb := &CircuitBreaker{
		failures:   make(map[string]int),
		unstable:   make(map[string]time.Time),
		threshold:  threshold,
		resetAfter: resetAfter,
		deviceRepo: deviceRepo,
	}
	go cb.cleanup()
	return cb
}

func (cb *CircuitBreaker) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		cb.mu.Lock()
		for k, t := range cb.unstable {
			if time.Since(t) > cb.resetAfter*2 {
				delete(cb.unstable, k)
				delete(cb.failures, k)
			}
		}
		cb.mu.Unlock()
	}
}

func (cb *CircuitBreaker) RecordFailure(ctx context.Context, deviceID string) {
	cb.mu.Lock()
	cb.failures[deviceID]++
	count := cb.failures[deviceID]
	cb.mu.Unlock()

	cb.deviceRepo.IncrementFailure(ctx, deviceID)

	if count >= cb.threshold {
		cb.mu.Lock()
		cb.unstable[deviceID] = time.Now()
		cb.mu.Unlock()
		cb.deviceRepo.SetDeviceState(ctx, deviceID, "unstable")
	}
}

func (cb *CircuitBreaker) RecordSuccess(deviceID string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	prev := cb.failures[deviceID]
	delete(cb.failures, deviceID)
	delete(cb.unstable, deviceID)

	if prev >= cb.threshold {
		cb.deviceRepo.ResetFailureCount(context.Background(), deviceID)
	}
}

func (cb *CircuitBreaker) IsUnstable(deviceID string) bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	t, ok := cb.unstable[deviceID]
	if !ok {
		return false
	}
	if time.Since(t) > cb.resetAfter {
		delete(cb.unstable, deviceID)
		delete(cb.failures, deviceID)
		return false
	}
	return true
}

func (cb *CircuitBreaker) FailureCount(deviceID string) int {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.failures[deviceID]
}
