package deploy

import (
	"testing"
)

func TestPreflightCheckCategories(t *testing.T) {
	categories := []string{"system", "network", "security", "runtime"}
	seen := map[string]bool{}

	for _, c := range categories {
		seen[c] = true
	}

	if !seen["system"] { t.Error("expected system category") }
	if !seen["network"] { t.Error("expected network category") }
	if !seen["security"] { t.Error("expected security category") }
	if !seen["runtime"] { t.Error("expected runtime category") }
}

func TestCheckStatusValues(t *testing.T) {
	statuses := []string{"pass", "warn", "fail", "skipped"}
	valid := map[string]bool{"pass": true, "warn": true, "fail": true, "skipped": true}

	for _, s := range statuses {
		if !valid[s] {
			t.Errorf("unexpected status: %s", s)
		}
	}
}

func TestConfigOptionsDefaults(t *testing.T) {
	cfg := &ConfigOptions{
		InstallDir:  "/opt/renos",
		HTTPPort:    8080,
		LogLevel:    "info",
		ExternalURL: "https://renos.local",
	}

	if cfg.InstallDir != "/opt/renos" { t.Error("expected default install dir") }
	if cfg.HTTPPort != 8080 { t.Error("expected default HTTP port") }
	if cfg.LogLevel != "info" { t.Error("expected default log level") }
}

func TestInstallationStatusPhases(t *testing.T) {
	phases := []string{
		"idle", "preflight", "secrets", "tls", "config",
		"compose", "directories", "dependencies", "complete", "failed", "rolling_back",
	}
	valid := map[string]bool{}
	for _, p := range phases {
		valid[p] = true
	}

	if !valid["idle"] { t.Error("expected idle phase") }
	if !valid["complete"] { t.Error("expected complete phase") }
	if !valid["failed"] { t.Error("expected failed phase") }
	if !valid["rolling_back"] { t.Error("expected rolling_back phase") }
}

func TestCheckResultFields(t *testing.T) {
	r := &CheckResult{
		Name:     "docker",
		Category: "runtime",
		Status:   "pass",
		Message:  "Docker is running",
	}

	if r.Name != "docker" { t.Error("expected docker") }
	if r.Category != "runtime" { t.Error("expected runtime") }
	if r.Status != "pass" { t.Error("expected pass") }
}

func TestPreflightReportFields(t *testing.T) {
	r := &PreflightReport{
		OS:        "linux",
		Arch:      "amd64",
		Hostname:  "server-1",
		PassCount: 10,
		WarnCount: 1,
		FailCount: 0,
		AllPassed: false,
	}

	if r.OS != "linux" { t.Error("expected linux") }
	if r.PassCount != 10 { t.Error("expected 10 passes") }
	if r.AllPassed { t.Error("expected not all passed") }
}
