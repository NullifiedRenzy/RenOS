package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type RollbackResult struct {
	UpgradeID string `json:"upgrade_id"`
	Component string `json:"component"`
	Version   string `json:"version"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
}

func ListAvailableRollbacks() []UpgradeRecord {
	state, err := Load()
	if err != nil {
		return nil
	}
	history := state.ListHistory()
	var rollbacks []UpgradeRecord
	for _, r := range history {
		if r.Status == "success" && r.BackupID != "" {
			rollbacks = append(rollbacks, r)
		}
	}
	return rollbacks
}

func Rollback(ctx context.Context, upgradeID string) *RollbackResult {
	result := &RollbackResult{
		UpgradeID: upgradeID,
		Status:    "failed",
	}

	state, err := Load()
	if err != nil {
		result.Message = fmt.Sprintf("load state: %v", err)
		return result
	}

	record := state.GetUpgrade(upgradeID)
	if record == nil {
		result.Message = fmt.Sprintf("upgrade record %s not found", upgradeID)
		return result
	}

	result.UpgradeID = record.ID
	result.Component = record.Component
	result.Version = record.FromVersion

	if record.BackupID == "" {
		result.Message = "no backup available for this upgrade"
		return result
	}

	backupPath := filepath.Join(backupDir, record.BackupID)
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		result.Message = fmt.Sprintf("backup directory %s not found", backupPath)
		return result
	}

	binaries := filepath.Join(backupPath, "binaries")
	config := filepath.Join(backupPath, "config")
	database := filepath.Join(backupPath, "database")
	plugins := filepath.Join(backupPath, "plugins")

	if err := stopService(record.Component); err != nil {
		result.Message = fmt.Sprintf("stop service: %v", err)
		return result
	}

	if err := restoreBinaries(record.Component, binaries); err != nil {
		result.Message = fmt.Sprintf("restore binaries: %v", err)
		startService(record.Component)
		return result
	}

	if err := restoreConfig(config); err != nil {
		result.Message = fmt.Sprintf("restore config: %v", err)
		return result
	}

	if _, err := os.Stat(database); err == nil {
		if err := restoreDatabase(database); err != nil {
			result.Message = fmt.Sprintf("restore database: %v", err)
			return result
		}
	}

	if _, err := os.Stat(plugins); err == nil {
		if err := restorePlugins(plugins); err != nil {
			result.Message = fmt.Sprintf("restore plugins: %v", err)
			return result
		}
	}

	if err := startService(record.Component); err != nil {
		result.Message = fmt.Sprintf("start service after rollback: %v", err)
		return result
	}

	if err := state.RecordUpgrade(record.Component, record.ToVersion, record.FromVersion,
		"rollback", ""); err != nil {
		result.Message = fmt.Sprintf("record rollback: %v", err)
		return result
	}

	result.Status = "success"
	result.Message = fmt.Sprintf("rolled back %s from %s to %s", record.Component, record.ToVersion, record.FromVersion)
	return result
}

func stopService(component string) error {
	components := []string{component}
	if component == "all" {
		components = []string{"core", "hub", "agent"}
	}
	for _, c := range components {
		svc := serviceNames[c]
		if svc == "" {
			continue
		}
		cmd := exec.Command("systemctl", "stop", svc)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("stop %s: %s: %w", svc, strings.TrimSpace(string(output)), err)
		}
	}
	return nil
}

func startService(component string) error {
	components := []string{component}
	if component == "all" {
		components = []string{"core", "hub", "agent"}
	}
	for _, c := range components {
		svc := serviceNames[c]
		if svc == "" {
			continue
		}
		cmd := exec.Command("systemctl", "start", svc)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("start %s: %s: %w", svc, strings.TrimSpace(string(output)), err)
		}
	}
	return nil
}

func restoreBinaries(component, src string) error {
	srcMap := map[string]string{
		"core":  "renos-core",
		"hub":   "renos-hub",
		"agent": "renos-agent",
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
		name, ok := srcMap[c]
		if !ok {
			continue
		}
		srcFile := filepath.Join(src, name)
		if _, err := os.Stat(srcFile); os.IsNotExist(err) {
			continue
		}
		data, err := os.ReadFile(srcFile)
		if err != nil {
			return fmt.Errorf("read backup binary %s: %w", srcFile, err)
		}
		dest := destMap[c]
		if err := os.WriteFile(dest, data, 0755); err != nil {
			return fmt.Errorf("restore binary %s: %w", dest, err)
		}
	}
	return nil
}

func restoreConfig(src string) error {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil
	}
	configDest := "/etc/renos"
	if err := os.MkdirAll(configDest, 0755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	return copyDir(src, configDest)
}

func restoreDatabase(src string) error {
	dumpFile := filepath.Join(src, "renos-dump.sql")
	if _, err := os.Stat(dumpFile); os.IsNotExist(err) {
		return nil
	}
	if _, err := exec.LookPath("psql"); err != nil {
		return fmt.Errorf("psql not available for restore: %w", err)
	}
	cmd := exec.Command("psql", "-d", dbName, "-f", dumpFile)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("database restore failed: %s: %w", string(output), err)
	}
	return nil
}

func restorePlugins(src string) error {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil
	}
	pluginDest := "/var/lib/renos/plugins"
	if err := os.MkdirAll(pluginDest, 0755); err != nil {
		return fmt.Errorf("create plugin directory: %w", err)
	}
	return copyDir(src, pluginDest)
}


