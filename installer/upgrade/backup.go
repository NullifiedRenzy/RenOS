package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	backupDir     = "/var/lib/renos/backups"
	dbDumpCommand = "pg_dump"
	dbName        = "renos"
)

type BackupResult struct {
	BackupDir string `json:"backup_dir"`
	BackupID  string `json:"backup_id"`
	Size      int64  `json:"size"`
	Timestamp string `json:"timestamp"`
	Component string `json:"component"`
}

func PreUpgradeBackup(ctx context.Context, component, currentVersion string) (*BackupResult, error) {
	timestamp := time.Now().UTC().Format("20060102T150405Z")
	backupID := fmt.Sprintf("bak-%s-%s-%s", component, currentVersion, timestamp)
	target := filepath.Join(backupDir, backupID)

	if err := os.MkdirAll(target, 0755); err != nil {
		return nil, fmt.Errorf("create backup directory %s: %w", target, err)
	}

	dirs := []string{
		filepath.Join(target, "binaries"),
		filepath.Join(target, "config"),
		filepath.Join(target, "database"),
		filepath.Join(target, "plugins"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return nil, fmt.Errorf("create backup subdirectory %s: %w", d, err)
		}
	}

	if err := backupBinaries(component, filepath.Join(target, "binaries")); err != nil {
		return nil, fmt.Errorf("backup binaries: %w", err)
	}
	if err := backupConfig(component, filepath.Join(target, "config")); err != nil {
		return nil, fmt.Errorf("backup config: %w", err)
	}
	if component == "database" || component == "all" {
		if err := backupDatabase(filepath.Join(target, "database")); err != nil {
			return nil, fmt.Errorf("backup database: %w", err)
		}
	}
	if component == "plugins" || component == "all" {
		if err := backupPlugins(filepath.Join(target, "plugins")); err != nil {
			return nil, fmt.Errorf("backup plugins: %w", err)
		}
	}

	var totalSize int64
	filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	return &BackupResult{
		BackupDir: target,
		BackupID:  backupID,
		Size:      totalSize,
		Timestamp: timestamp,
		Component: component,
	}, nil
}

func backupBinaries(component, dest string) error {
	srcMap := map[string]string{
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
		if _, err := os.Stat(src); err == nil {
			data, err := os.ReadFile(src)
			if err != nil {
				return fmt.Errorf("read binary %s: %w", src, err)
			}
			destFile := filepath.Join(dest, filepath.Base(src))
			if err := os.WriteFile(destFile, data, 0755); err != nil {
				return fmt.Errorf("write backup binary %s: %w", destFile, err)
			}
		}
	}
	return nil
}

func backupConfig(component, dest string) error {
	configDirs := []string{
		"/etc/renos",
	}
	if _, err := os.Stat(configDirs[0]); err == nil {
		return copyDir(configDirs[0], dest)
	}
	return nil
}

func backupDatabase(dest string) error {
	if _, err := exec.LookPath(dbDumpCommand); err != nil {
		return nil
	}
	dumpFile := filepath.Join(dest, "renos-dump.sql")
	cmd := exec.Command(dbDumpCommand, "-d", dbName, "-f", dumpFile)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("database dump failed: %s: %w", string(output), err)
	}
	return nil
}

func backupPlugins(dest string) error {
	pluginDir := "/var/lib/renos/plugins"
	if _, err := os.Stat(pluginDir); err == nil {
		return copyDir(pluginDir, dest)
	}
	return nil
}

func copyDir(src, dest string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}

func ListBackups() ([]string, error) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read backup directory: %w", err)
	}
	var backups []string
	for _, e := range entries {
		if e.IsDir() {
			backups = append(backups, e.Name())
		}
	}
	return backups, nil
}

func PruneOldBackups(retention int) error {
	backups, err := ListBackups()
	if err != nil {
		return err
	}
	if len(backups) <= retention {
		return nil
	}

	for i := 0; i < len(backups)-retention; i++ {
		path := filepath.Join(backupDir, backups[i])
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("remove old backup %s: %w", path, err)
		}
	}
	return nil
}
