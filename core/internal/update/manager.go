package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/renos-project/renos/core/internal/db"
)

const CurrentVersion = "0.1.0"

type Channel string

const (
	ChannelStable Channel = "stable"
	ChannelBeta   Channel = "beta"
	ChannelNightly Channel = "nightly"
)

type ReleaseInfo struct {
	Version     string   `json:"version"`
	Channel     Channel  `json:"channel"`
	ReleaseDate string   `json:"release_date"`
	Components  []string `json:"components"`
	Changelog   string   `json:"changelog"`
	DownloadURL string   `json:"download_url"`
	Checksum    string   `json:"checksum"`
	MinVersion  string   `json:"min_version"`
}

type UpdatePlanner struct {
	currentVersion string
	channel        Channel
	releases       []ReleaseInfo
	history        *db.UpdateHistoryRepository
	backups        *db.UpdateBackupRepository
}

func NewUpdatePlanner(history *db.UpdateHistoryRepository, backups *db.UpdateBackupRepository) *UpdatePlanner {
	return &UpdatePlanner{
		currentVersion: CurrentVersion,
		channel:        ChannelStable,
		history:        history,
		backups:        backups,
	}
}

func (u *UpdatePlanner) CurrentVersion() string { return u.currentVersion }
func (u *UpdatePlanner) Channel() Channel { return u.channel }
func (u *UpdatePlanner) SetChannel(ch Channel) { u.channel = ch }

type UpdatePlan struct {
	CurrentVersion string      `json:"current_version"`
	TargetVersion  string      `json:"target_version"`
	Channel        Channel     `json:"channel"`
	Components     []string    `json:"components"`
	Steps          []PlanStep  `json:"steps"`
	RequiresBackup bool        `json:"requires_backup"`
	EstimatedTime  string      `json:"estimated_time"`
}

type PlanStep struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Component   string `json:"component"`
	Action      string `json:"action"`
	Order       int    `json:"order"`
}

func (u *UpdatePlanner) CheckForUpdates(ctx context.Context) (*ReleaseInfo, error) {
	releases, err := u.fetchReleases(ctx)
	if err != nil { return nil, err }
	u.releases = releases

	for _, r := range releases {
		if r.Channel == u.channel && isNewer(u.currentVersion, r.Version) {
			return &r, nil
		}
	}
	return nil, nil
}

func (u *UpdatePlanner) PlanUpdate(target *ReleaseInfo) *UpdatePlan {
	plan := &UpdatePlan{
		CurrentVersion: u.currentVersion,
		TargetVersion:  target.Version,
		Channel:        Channel(target.Channel),
		Components:     target.Components,
		RequiresBackup: true,
		EstimatedTime:  "5-10 minutes",
		Steps: []PlanStep{
			{Name: "backup", Description: "Create pre-update backup", Component: "system", Action: "backup", Order: 1},
			{Name: "download", Description: "Download update packages", Component: "system", Action: "download", Order: 2},
			{Name: "stop_hub", Description: "Stop Hub service", Component: "hub", Action: "stop", Order: 3},
			{Name: "stop_core", Description: "Stop Core service", Component: "core", Action: "stop", Order: 4},
			{Name: "migrate_db", Description: "Run database migrations", Component: "core", Action: "migrate", Order: 5},
			{Name: "update_core", Description: "Update Core binary", Component: "core", Action: "update", Order: 6},
			{Name: "update_hub", Description: "Update Hub assets", Component: "hub", Action: "update", Order: 7},
			{Name: "update_agent", Description: "Update Agent binary", Component: "agent", Action: "update", Order: 8},
			{Name: "start_core", Description: "Start Core service", Component: "core", Action: "start", Order: 9},
			{Name: "start_hub", Description: "Start Hub service", Component: "hub", Action: "start", Order: 10},
			{Name: "verify", Description: "Verify health and connectivity", Component: "system", Action: "verify", Order: 11},
		},
	}
	return plan
}

func (u *UpdatePlanner) ApplyUpdate(ctx context.Context, plan *UpdatePlan, historyEntry *db.UpdateHistoryEntry) error {
	for _, step := range plan.Steps {
		if err := ctx.Err(); err != nil { return err }
		var err error
		switch step.Action {
		case "backup":
			err = u.createBackup(ctx, plan.TargetVersion)
		case "download":
			err = u.downloadUpdate(ctx, plan.TargetVersion)
		case "stop":
			err = u.stopComponent(step.Component)
		case "migrate":
			err = u.runMigrations(ctx)
		case "update":
			err = u.updateComponent(ctx, step.Component, plan.TargetVersion)
		case "start":
			err = u.startComponent(step.Component)
		case "verify":
			err = u.verifyHealth(ctx)
		}
		if err != nil {
			u.history.Fail(ctx, historyEntry.ID, fmt.Sprintf("step %s failed: %v", step.Name, err))
			return err
		}
	}
	u.history.Complete(ctx, historyEntry.ID)
	return nil
}

func (u *UpdatePlanner) Rollback(ctx context.Context, backupID string) error {
	backup, err := u.backups.FindByID(ctx, backupID)
	if err != nil { return fmt.Errorf("backup not found: %w", err) }
	if backup.Status != "ready" { return fmt.Errorf("backup is not ready") }

	u.stopComponent("hub")
	u.stopComponent("core")

	cmd := exec.CommandContext(ctx, "cp", "-r", backup.BackupPath+"/*", "/opt/renos/")
	cmd.Run()

	if backup.DatabaseDumpPath != "" {
		cmd = exec.CommandContext(ctx, "pg_restore", "-d", "renos", backup.DatabaseDumpPath)
		cmd.Run()
	}

	u.startComponent("core")
	u.startComponent("hub")
	return nil
}

func (u *UpdatePlanner) fetchReleases(ctx context.Context) ([]ReleaseInfo, error) {
	// In production, this fetches from releases.renos.dev
	// For now, return current version as the only release
	return []ReleaseInfo{
		{Version: CurrentVersion, Channel: ChannelStable, Components: []string{"core", "hub", "agent"}},
	}, nil
}

func (u *UpdatePlanner) createBackup(ctx context.Context, version string) error {
	backupDir := filepath.Join("/opt/renos/backups", version+"-"+time.Now().Format("20060102150405"))
	if err := os.MkdirAll(backupDir, 0755); err != nil { return err }

	// Backup config files
	cmd := exec.CommandContext(ctx, "cp", "-r", "/opt/renos/.env", backupDir+"/")
	cmd.Run()
	cmd = exec.CommandContext(ctx, "cp", "-r", "/opt/renos/deploy", backupDir+"/")
	cmd.Run()

	// Database dump
	dumpPath := filepath.Join(backupDir, "database.sql")
	cmd = exec.CommandContext(ctx, "pg_dump", "-U", "renos", "renos")
	dumpFile, _ := os.Create(dumpPath)
	if dumpFile != nil {
		cmd.Stdout = dumpFile
		cmd.Run()
		dumpFile.Close()
	}

	checksum := u.computeChecksum(backupDir)

	b := &db.UpdateBackup{
		Version:          version,
		BackupType:       "pre_update",
		BackupPath:       backupDir,
		DatabaseDumpPath: dumpPath,
		Checksum:         checksum,
		Status:           "ready",
	}
	u.backups.Create(ctx, b)
	return nil
}

func (u *UpdatePlanner) downloadUpdate(ctx context.Context, version string) error {
	// Placeholder for actual download logic
	return nil
}

func (u *UpdatePlanner) stopComponent(name string) error {
	services := map[string]string{"core": "core", "hub": "hub", "agent": "agent"}
	if svc, ok := services[name]; ok {
		cmd := exec.Command("docker", "compose", "-f", "/opt/renos/deploy/docker-compose.prod.yml", "stop", svc)
		cmd.Run()
	}
	return nil
}

func (u *UpdatePlanner) startComponent(name string) error {
	services := map[string]string{"core": "core", "hub": "hub", "agent": "agent"}
	if svc, ok := services[name]; ok {
		cmd := exec.Command("docker", "compose", "-f", "/opt/renos/deploy/docker-compose.prod.yml", "start", svc)
		cmd.Run()
	}
	return nil
}

func (u *UpdatePlanner) runMigrations(ctx context.Context) error {
	// Migrations are handled by the core service on startup
	return nil
}

func (u *UpdatePlanner) updateComponent(ctx context.Context, name, version string) error {
	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", "/opt/renos/deploy/docker-compose.prod.yml", "pull", name)
	return cmd.Run()
}

func (u *UpdatePlanner) verifyHealth(ctx context.Context) error {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("http://localhost:8443/api/v1/healthz")
	if err != nil { return fmt.Errorf("health check failed: %w", err) }
	defer resp.Body.Close()
	if resp.StatusCode != 200 { return fmt.Errorf("health check returned %d", resp.StatusCode) }
	return nil
}

func (u *UpdatePlanner) computeChecksum(dir string) string {
	h := sha256.New()
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() { return nil }
		f, err := os.Open(path)
		if err != nil { return nil }
		defer f.Close()
		io.Copy(h, f)
		return nil
	})
	return hex.EncodeToString(h.Sum(nil))
}

func isNewer(current, target string) bool {
	// Simple semver comparison
	return target > current
}

type VersionInfo struct {
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	Channel        string `json:"channel"`
	UpdateAvailable bool  `json:"update_available"`
}

func (u *UpdatePlanner) GetVersionInfo() *VersionInfo {
	latest := CurrentVersion
	if len(u.releases) > 0 {
		for _, r := range u.releases {
			if r.Channel == u.channel && isNewer(latest, r.Version) {
				latest = r.Version
			}
		}
	}
	return &VersionInfo{
		CurrentVersion: CurrentVersion,
		LatestVersion:  latest,
		Channel:        string(u.channel),
		UpdateAvailable: isNewer(CurrentVersion, latest),
	}
}
