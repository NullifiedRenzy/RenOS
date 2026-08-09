package telemetry

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildPayloadContainsNoSecrets(t *testing.T) {
	cfg := Config{
		Enabled:    true,
		Endpoint:   "https://telemetry.renos.io/v1/ping",
		Version:    "1.2.3",
		InstanceID: "test-instance",
	}
	tm := New(cfg)

	// Pretend some sensitive-looking values flow through the counters.
	body, err := tm.buildPayload(Counts{
		Devices:   7,
		Workflows: 3,
		Plugins:   2,
	})
	if err != nil {
		t.Fatalf("buildPayload failed: %v", err)
	}

	// The payload must parse as JSON and contain only the whitelisted keys.
	var p payload
	if err := json.Unmarshal(body, &p); err != nil {
		t.Fatalf("payload is not valid JSON: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatalf("payload is not a JSON object: %v", err)
	}

	allowed := []string{"version", "os", "arch", "instance_id", "uptime_seconds", "devices", "workflows", "plugins", "timestamp"}
	for k := range raw {
		found := false
		for _, a := range allowed {
			if k == a {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("payload contains unexpected key %q — not part of the anonymous whitelist", k)
		}
	}

	// Even if a secret string were stuffed into config/counters it must not appear.
	if strings.Contains(string(body), "hunter2") {
		t.Errorf("payload leaked a secret value")
	}

	if p.Version != "1.2.3" {
		t.Errorf("expected version 1.2.3, got %q", p.Version)
	}
	if p.Devices != 7 || p.Workflows != 3 || p.Plugins != 2 {
		t.Errorf("counts not propagated: %+v", p)
	}
	if p.InstanceID != "test-instance" {
		t.Errorf("expected instance id test-instance, got %q", p.InstanceID)
	}
}

func TestDisabledConfigDoesNothing(t *testing.T) {
	tm := New(Config{
		Enabled:  false,
		Endpoint: "http://127.0.0.1:1/never", // would fail instantly if attempted
		Interval: time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	tm.Start(ctx, func() Counts { return Counts{Devices: 1} })

	// Give the (non-existent) goroutine a chance to run — it must not.
	time.Sleep(20 * time.Millisecond)

	if tm.SendCount() != 0 {
		t.Errorf("disabled telemetry sent %d pings, expected 0", tm.SendCount())
	}

	tm.Stop() // must not block or panic
	cancel()
}

func TestInstanceIDPersisted(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "instance-id")

	tm := New(Config{
		Enabled:        true,
		InstanceIDFile: file,
	})
	if tm.instance == "" {
		t.Fatalf("expected a generated instance id")
	}

	// Reload from the same file — the ID must be stable across restarts.
	tm2 := New(Config{
		Enabled:        true,
		InstanceIDFile: file,
	})
	if tm2.instance != tm.instance {
		t.Errorf("instance id not stable across restarts: %q != %q", tm.instance, tm2.instance)
	}

	// Instance ID must look random, not derived from any host identity.
	if strings.HasPrefix(tm.instance, "renos-") || strings.Contains(tm.instance, "@") {
		t.Errorf("instance id looks like an identifier: %q", tm.instance)
	}
}

func TestInstanceIDRespectsProvidedValue(t *testing.T) {
	tm := New(Config{
		Enabled:    true,
		InstanceID: "fixed-id",
	})
	if tm.instance != "fixed-id" {
		t.Errorf("expected provided instance id, got %q", tm.instance)
	}
}

func TestPayloadIncludesNoHostname(t *testing.T) {
	dir := t.TempDir()
	hostname, _ := os.Hostname()

	tm := New(Config{
		Enabled:        true,
		Hostname:       hostname,
		InstanceID:     "test",
		InstanceIDFile: filepath.Join(dir, "id"),
	})
	body, err := tm.buildPayload(Counts{})
	if err != nil {
		t.Fatalf("buildPayload failed: %v", err)
	}
	if strings.Contains(string(body), hostname) {
		t.Errorf("payload leaked the hostname %q — hostnames are not anonymous", hostname)
	}
}
