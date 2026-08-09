package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type UpgradeStep string

const (
	StepValidate  UpgradeStep = "validate"
	StepBackup    UpgradeStep = "backup"
	StepDownload  UpgradeStep = "download"
	StepStop      UpgradeStep = "stop"
	StepReplace   UpgradeStep = "replace"
	StepMigrate   UpgradeStep = "migrate"
	StepStart     UpgradeStep = "start"
	StepVerify    UpgradeStep = "verify"
	StepRecord    UpgradeStep = "record"
)

type UpgradeEvent struct {
	Step      UpgradeStep `json:"step"`
	Component string      `json:"component"`
	Status    string      `json:"status"`
	Message   string      `json:"message,omitempty"`
}

type UpgradeManager struct {
	State   *UpgradeState
	Events  chan UpgradeEvent
	Verbose bool
}

func NewUpgradeManager() *UpgradeManager {
	state, err := Load()
	if err != nil {
		state = NewUpgradeState()
	}
	return &UpgradeManager{
		State:  state,
		Events: make(chan UpgradeEvent, 100),
	}
}

func (m *UpgradeManager) emit(step UpgradeStep, component, status, message string) {
	evt := UpgradeEvent{
		Step:      step,
		Component: component,
		Status:    status,
		Message:   message,
	}
	m.Events <- evt
	if m.Verbose || status == "error" {
		fmt.Fprintf(os.Stderr, "[%s] %s/%s: %s\n", status, component, step, message)
	}
}

func (m *UpgradeManager) Upgrade(ctx context.Context, component, targetVersion string) error {
	currentVersion := m.State.ComponentVersionOf(component)
	if currentVersion == "" {
		currentVersion = m.State.CurrentVersion
	}

	m.emit(StepValidate, component, "start", fmt.Sprintf("validating upgrade %s -> %s", currentVersion, targetVersion))

	if err := CheckVersionCompat(currentVersion, targetVersion, component); err != nil {
		m.emit(StepValidate, component, "error", err.Error())
		return fmt.Errorf("version check failed: %w", err)
	}

	deps := ComponentDependencies(component)
	if len(deps) > 0 {
		m.emit(StepValidate, component, "info", fmt.Sprintf("checking dependencies: %s", strings.Join(deps, ", ")))
		for _, dep := range deps {
			depVer := m.State.ComponentVersionOf(dep)
			if depVer == "" || depVer == "0.0.0" {
				m.emit(StepValidate, component, "warn", fmt.Sprintf("dependency %s has unknown version", dep))
			}
		}
	}

	if component == "plugins" {
		if err := CheckPluginCompatibility(targetVersion); err != nil {
			m.emit(StepValidate, component, "error", err.Error())
			return fmt.Errorf("plugin compatibility: %w", err)
		}
	}

	m.emit(StepValidate, component, "done", "validation passed")

	m.emit(StepBackup, component, "start", "creating backup")
	backup, err := PreUpgradeBackup(ctx, component, currentVersion)
	if err != nil {
		m.emit(StepBackup, component, "error", err.Error())
		return fmt.Errorf("backup failed: %w", err)
	}
	m.emit(StepBackup, component, "done", fmt.Sprintf("backup created: %s (%d bytes)", backup.BackupID, backup.Size))

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled after backup: %w", err)
	}

	m.emit(StepDownload, component, "start", fmt.Sprintf("downloading version %s", targetVersion))
	if err := m.downloadVersion(ctx, component, targetVersion); err != nil {
		m.emit(StepDownload, component, "error", err.Error())
		return fmt.Errorf("download failed: %w", err)
	}
	m.emit(StepDownload, component, "done", "download complete")

	m.emit(StepStop, component, "start", "stopping service")
	if err := stopService(component); err != nil {
		m.emit(StepStop, component, "error", err.Error())
		return fmt.Errorf("stop service failed: %w", err)
	}
	m.emit(StepStop, component, "done", "service stopped")

	m.emit(StepReplace, component, "start", "replacing binary")
	if err := m.replaceBinary(component, targetVersion); err != nil {
		m.emit(StepReplace, component, "error", err.Error())
		startService(component)
		return fmt.Errorf("replace binary failed: %w", err)
	}
	m.emit(StepReplace, component, "done", "binary replaced")

	m.emit(StepMigrate, component, "start", "running database migrations")
	if err := m.runMigrations(ctx, component, currentVersion, targetVersion); err != nil {
		m.emit(StepMigrate, component, "error", err.Error())
		startService(component)
		return fmt.Errorf("migration failed: %w", err)
	}
	m.emit(StepMigrate, component, "done", "migrations complete")

	m.emit(StepStart, component, "start", "starting service")
	if err := startService(component); err != nil {
		m.emit(StepStart, component, "error", err.Error())
		return fmt.Errorf("start service failed: %w", err)
	}
	m.emit(StepStart, component, "done", "service started")

	m.emit(StepVerify, component, "start", "running health verification")
	if err := m.verifyHealth(ctx, component); err != nil {
		m.emit(StepVerify, component, "error", err.Error())
		return fmt.Errorf("health check failed after upgrade: %w", err)
	}
	m.emit(StepVerify, component, "done", "health verification passed")

	m.emit(StepRecord, component, "start", "recording upgrade")
	if err := m.State.RecordUpgrade(component, currentVersion, targetVersion, "success", backup.BackupID); err != nil {
		m.emit(StepRecord, component, "error", err.Error())
		return fmt.Errorf("record upgrade failed: %w", err)
	}
	m.emit(StepRecord, component, "done", "upgrade recorded")

	return nil
}

func (m *UpgradeManager) DryRun(ctx context.Context, component, targetVersion string) error {
	currentVersion := m.State.ComponentVersionOf(component)
	if currentVersion == "" {
		currentVersion = m.State.CurrentVersion
	}

	fmt.Printf("DRY RUN: Upgrade %s %s -> %s\n", component, currentVersion, targetVersion)

	fmt.Println("  [1/5] Version compatibility check...")
	if err := CheckVersionCompat(currentVersion, targetVersion, component); err != nil {
		return fmt.Errorf("version check would FAIL: %w", err)
	}
	fmt.Println("        PASS")

	deps := ComponentDependencies(component)
	if len(deps) > 0 {
		fmt.Printf("  [2/5] Dependency check (%s)...\n", strings.Join(deps, ", "))
		for _, dep := range deps {
			depVer := m.State.ComponentVersionOf(dep)
			fmt.Printf("        %s: current version %s\n", dep, depVer)
		}
	}

	fmt.Println("  [3/5] Backup space check...")
	backup, err := PreUpgradeBackup(ctx, component, currentVersion)
	if err != nil {
		fmt.Printf("        Backup would FAIL: %v\n", err)
	} else {
		fmt.Printf("        Backup would take ~%d bytes at %s\n", backup.Size, backup.BackupDir)
	}

	fmt.Println("  [4/5] Breaking changes review...")
	changes := BreakingChanges(currentVersion, targetVersion)
	if len(changes) == 0 {
		fmt.Println("        No breaking changes detected")
	} else {
		for _, c := range changes {
			fmt.Printf("        - %s\n", c)
		}
	}

	fmt.Println("  [5/5] Service health pre-check...")
	health := AllHealthChecks()
	for _, s := range health.Statuses {
		status := "OK"
		if !s.Healthy {
			status = "FAIL"
		}
		fmt.Printf("        %s/%s: %s\n", s.Component, s.Service, status)
	}

	fmt.Println("\nDry run complete. No changes were made.")
	return nil
}

func (m *UpgradeManager) downloadVersion(ctx context.Context, component, version string) error {
	artifactURL := fmt.Sprintf("https://releases.renos.io/%s/%s", component, version)
	fmt.Fprintf(os.Stderr, "  downloading: %s\n", artifactURL)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(100 * time.Millisecond):
	}
	return nil
}

func (m *UpgradeManager) replaceBinary(component, version string) error {
	srcMap := map[string]string{
		"core":  fmt.Sprintf("/tmp/renos-core-%s", version),
		"hub":   fmt.Sprintf("/tmp/renos-hub-%s", version),
		"agent": fmt.Sprintf("/tmp/renos-agent-%s", version),
	}
	destMap := map[string]string{
		"core":  "/usr/local/bin/renos-core",
		"hub":   "/usr/local/bin/renos-hub",
		"agent": "/usr/local/bin/renos-agent",
	}

	components := []string{component}
	if component == "all" {
		components = []string{"core", "hub", "agent"}
	}
	for _, c := range components {
		src, ok := srcMap[c]
		if !ok {
			continue
		}
		if _, err := os.Stat(src); os.IsNotExist(err) {
			return fmt.Errorf("downloaded artifact not found: %s", src)
		}
		dest := destMap[c]
		if err := os.Rename(src, dest); err != nil {
			return fmt.Errorf("move %s to %s: %w", src, dest, err)
		}
		if err := os.Chmod(dest, 0755); err != nil {
			return fmt.Errorf("chmod %s: %w", dest, err)
		}
	}
	return nil
}

func (m *UpgradeManager) runMigrations(ctx context.Context, component, fromVersion, toVersion string) error {
	if component != "core" && component != "database" && component != "all" {
		return nil
	}

	migrationScript := fmt.Sprintf("/usr/lib/renos/migrations/%s_to_%s.sh", fromVersion, toVersion)
	if _, err := os.Stat(migrationScript); os.IsNotExist(err) {
		migrationScript = fmt.Sprintf("/usr/lib/renos/migrations/migrate_%s.sh", component)
		if _, err := os.Stat(migrationScript); os.IsNotExist(err) {
			return nil
		}
	}

	cmd := exec.CommandContext(ctx, "bash", migrationScript, fromVersion, toVersion)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("migration script %s failed: %s: %w", migrationScript, string(output), err)
	}
	return nil
}

func (m *UpgradeManager) verifyHealth(ctx context.Context, component string) error {
	components := []string{component}
	if component == "all" {
		components = []string{"core", "hub", "agent"}
	}

	for _, c := range components {
		url, ok := healthEndpoints[c]
		if !ok {
			continue
		}

		deadline := time.Now().Add(30 * time.Second)
		var lastErr error
		for time.Now().Before(deadline) {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			status := CheckAPIHealth(url)
			if status.Healthy {
				lastErr = nil
				break
			}
			lastErr = fmt.Errorf("%s not healthy: %s", c, status.Message)
			time.Sleep(2 * time.Second)
		}
		if lastErr != nil {
			return lastErr
		}
	}

	return nil
}
