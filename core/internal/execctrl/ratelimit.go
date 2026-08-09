package execctrl

import (
	"sync"
	"time"
)

type DeviceRateLimiter struct {
	mu       sync.Mutex
	lastTime map[string]time.Time
	cooldown time.Duration
}

func NewDeviceRateLimiter(cooldown time.Duration) *DeviceRateLimiter {
	rl := &DeviceRateLimiter{
		lastTime: make(map[string]time.Time),
		cooldown: cooldown,
	}
	go rl.cleanup()
	return rl
}

func (rl *DeviceRateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-rl.cooldown * 10)
		for k, t := range rl.lastTime {
			if t.Before(cutoff) {
				delete(rl.lastTime, k)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *DeviceRateLimiter) Allow(deviceID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	last, ok := rl.lastTime[deviceID]
	now := time.Now()
	if ok && now.Sub(last) < rl.cooldown {
		return false
	}
	rl.lastTime[deviceID] = now
	return true
}

func (rl *DeviceRateLimiter) Reset(deviceID string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.lastTime, deviceID)
}
