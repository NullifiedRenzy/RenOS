// Package telemetry implements an opt-in, anonymous usage telemetry sender.
//
// Privacy policy:
//   - Telemetry is DISABLED by default and only activates when the
//     RENOS_TELEMETRY_ENABLED environment variable is set to "true".
//   - Only anonymous, aggregate data is collected: application version,
//     OS / architecture, a random per-install ID, process uptime and
//     counts of devices, workflows and plugins. No personal data, no
//     hostnames, no IP addresses and no secrets are ever transmitted.
//   - Failures are silently ignored and the sender never blocks shutdown.
package telemetry

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"time"
)

// DefaultEndpoint is the telemetry ingestion endpoint.
const DefaultEndpoint = "https://telemetry.renos.io/v1/ping"

// DefaultInterval is how often a ping is sent.
const DefaultInterval = 24 * time.Hour

// DefaultInstanceIDFile is where the stable random instance ID is persisted.
const DefaultInstanceIDFile = "/var/lib/renos/telemetry-instance-id"

// Config controls the telemetry sender.
type Config struct {
	// Enabled opts the sender in. Telemetry is never enabled by default.
	Enabled bool
	// Endpoint is the HTTP(S) URL to POST pings to.
	Endpoint string
	// Interval is the period between pings. Zero falls back to DefaultInterval.
	Interval time.Duration
	// Version is the application version reported in the payload.
	Version string
	// Hostname is retained for future use but is deliberately NOT included
	// in the anonymous payload.
	Hostname string
	// InstanceID is a pre-supplied stable instance ID. When empty a random
	// one is generated and persisted at InstanceIDFile so it stays stable
	// across restarts.
	InstanceID string
	// InstanceIDFile is where the generated instance ID is persisted.
	InstanceIDFile string
}

// Counts holds the anonymous usage counters attached to each ping.
type Counts struct {
	Devices   int `json:"devices"`
	Workflows int `json:"workflows"`
	Plugins   int `json:"plugins"`
}

// payload is the exact JSON document sent to the endpoint. It is the
// whitelist of every field telemetry is allowed to transmit.
type payload struct {
	Version       string `json:"version"`
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	InstanceID    string `json:"instance_id"`
	UptimeSeconds int64  `json:"uptime_seconds"`
	Devices       int    `json:"devices"`
	Workflows     int    `json:"workflows"`
	Plugins       int    `json:"plugins"`
	Timestamp     string `json:"timestamp"`
}

// Telemetry is the sender. It is safe for concurrent use.
type Telemetry struct {
	cfg       Config
	client    *http.Client
	instance  string
	startTime time.Time
	sendCount atomic.Int64

	cancel  context.CancelFunc
	done    chan struct{}
	started atomic.Bool
}

// New builds a Telemetry sender from cfg. Defaults are applied for the
// endpoint, interval and instance ID file when the caller left them empty.
func New(cfg Config) *Telemetry {
	if cfg.Endpoint == "" {
		cfg.Endpoint = DefaultEndpoint
	}
	if cfg.Interval <= 0 {
		cfg.Interval = DefaultInterval
	}
	if cfg.InstanceIDFile == "" {
		cfg.InstanceIDFile = DefaultInstanceIDFile
	}
	t := &Telemetry{
		cfg:       cfg,
		client:    &http.Client{Timeout: 10 * time.Second},
		startTime: time.Now(),
	}
	t.instance = t.resolveInstanceID(cfg.InstanceID)
	return t
}

// resolveInstanceID returns the caller-supplied ID or a stable random ID
// that is loaded from / persisted to InstanceIDFile.
func (t *Telemetry) resolveInstanceID(provided string) string {
	if provided != "" {
		return provided
	}
	if b, err := os.ReadFile(t.cfg.InstanceIDFile); err == nil {
		id := string(bytes.TrimSpace(b))
		if id != "" {
			return id
		}
	}
	id, err := newInstanceID()
	if err != nil {
		return ""
	}
	if t.cfg.InstanceIDFile != "" {
		if dir := filepath.Dir(t.cfg.InstanceIDFile); dir != "" {
			if err := os.MkdirAll(dir, 0o755); err == nil {
				_ = os.WriteFile(t.cfg.InstanceIDFile, []byte(id+"\n"), 0o644)
			}
		}
	}
	return id
}

// SendCount reports how many pings have been attempted. Useful in tests.
func (t *Telemetry) SendCount() int64 {
	return t.sendCount.Load()
}

// Start begins the background sender loop. It returns immediately when the
// sender is disabled, so a disabled config does absolutely nothing. The
// goroutine exits when ctx is cancelled and never blocks shutdown.
// CountsProvider is called right before each ping.
func (t *Telemetry) Start(ctx context.Context, counts CountsProvider) {
	if !t.cfg.Enabled {
		slog.Info("telemetry disabled")
		return
	}
	if counts == nil {
		counts = func() Counts { return Counts{} }
	}

	t.startTime = time.Now()
	t.done = make(chan struct{})
	cancelCtx, cancel := context.WithCancel(ctx)
	t.cancel = cancel
	t.started.Store(true)

	slog.Info("telemetry enabled", "endpoint", t.cfg.Endpoint, "interval", t.cfg.Interval, "instance_id", t.instance)

	go func() {
		defer close(t.done)
		ticker := time.NewTicker(t.cfg.Interval)
		defer ticker.Stop()

		t.sendOnce(counts())

		for {
			select {
			case <-cancelCtx.Done():
				slog.Info("telemetry stopped")
				return
			case <-ticker.C:
				t.sendOnce(counts())
			}
		}
	}()
}

// Stop cancels the sender context and waits for the background goroutine to
// exit so telemetry never blocks or leaks during shutdown. It is a no-op
// when Start was never called or the sender is disabled.
func (t *Telemetry) Stop() {
	if !t.started.Load() {
		return
	}
	if t.cancel != nil {
		t.cancel()
	}
	<-t.done
	t.started.Store(false)
}

// CountsProvider supplies the current anonymous counters.
type CountsProvider func() Counts

// sendOnce builds the payload and POSTs it. All failures are silently
// ignored; telemetry must never interfere with the core service.
func (t *Telemetry) sendOnce(counts Counts) {
	t.sendCount.Add(1)
	body, err := t.buildPayload(counts)
	if err != nil {
		slog.Debug("telemetry: failed to build payload", "error", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), t.client.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.cfg.Endpoint, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "renos-telemetry/"+t.cfg.Version)

	resp, err := t.client.Do(req)
	if err != nil {
		slog.Debug("telemetry: ping failed", "error", err)
		return
	}
	defer resp.Body.Close()
	slog.Debug("telemetry: ping sent", "status", resp.StatusCode)
}

// buildPayload serializes the anonymous payload. It only ever includes the
// whitelisted fields from the payload struct, so secrets and personal data
// cannot leak into the document.
func (t *Telemetry) buildPayload(counts Counts) ([]byte, error) {
	p := payload{
		Version:       t.cfg.Version,
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		InstanceID:    t.instance,
		UptimeSeconds: int64(time.Since(t.startTime).Seconds()),
		Devices:       counts.Devices,
		Workflows:     counts.Workflows,
		Plugins:       counts.Plugins,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
	}
	return json.Marshal(p)
}

// newInstanceID generates a random hex identifier.
func newInstanceID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate instance id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
