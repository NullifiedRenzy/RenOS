package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/renos-project/renos/core/internal/auth"
	"github.com/renos-project/renos/core/internal/db"
)

// Demo seed identifiers. All demo entities share a distinctive prefix so they
// can be recognized, filtered and (in the future) cleaned up.
const (
	demoEmail        = "demo@renos.local"
	demoPassword     = "demo-demo-123"
	demoDevicePrefix = "demo-"
	demoPluginPrefix = "demo-"
	demoJobPrefix    = "demo-"
	demoVersion      = "1.0.0"
)

// demoSeedResult is the payload returned by a successful demo seed.
type demoSeedResult struct {
	Seeded           bool               `json:"seeded"`
	DemoMode         bool               `json:"demo_mode"`
	User             *demoSeededUser    `json:"user"`
	Devices          []demoSeededEntity `json:"devices"`
	Workflows        []demoSeededEntity `json:"workflows"`
	AlertRules       []demoSeededEntity `json:"alert_rules"`
	Dashboards       []demoSeededEntity `json:"dashboards"`
	Notifications    []demoSeededEntity `json:"notifications"`
	Marketplace      []demoSeededEntity `json:"marketplace"`
	InstalledPlugins []demoSeededEntity `json:"installed_plugins"`
	Jobs             []demoSeededEntity `json:"jobs"`
}

type demoSeededUser struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

type demoSeededEntity struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
}

// demoStatusHandler reports whether the demo dataset has already been seeded,
// so the UI can show a demo toggle without attempting a duplicate seed.
func demoStatusHandler(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSONResponse(w, http.StatusOK, map[string]bool{
			"demo_loaded": alreadySeeded(deps, r),
		})
	}
}

// seedDemoHandler populates the database with a demo dataset: a demo user, a
// small fleet of devices, plugins, workflows, alert rules, a monitoring
// dashboard, a notification provider and a sample job. It returns 409 when
// demo data already exists (idempotency guard).
func seedDemoHandler(deps *Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if alreadySeeded(deps, r) {
			writeError(w, http.StatusConflict, "DEMO_ALREADY_SEEDED", "Demo data already exists")
			return
		}

		result, err := seedDemoData(deps, r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "DEMO_SEED_FAILED", "Failed to seed demo data: "+err.Error())
			return
		}
		writeJSONResponse(w, http.StatusOK, result)
	}
}

// alreadySeeded reports whether the demo dataset already exists, guarding
// against duplicate seeding.
func alreadySeeded(deps *Dependencies, r *http.Request) bool {
	if deps.Users != nil {
		if user, err := deps.Users.FindByEmail(r.Context(), demoEmail); err == nil && user != nil {
			return true
		}
	}
	if deps.Devices != nil {
		if devices, err := deps.Devices.FindAll(r.Context()); err == nil {
			for _, d := range devices {
				if strings.HasPrefix(d.Hostname, demoDevicePrefix) {
					return true
				}
			}
		}
	}
	return false
}

func seedDemoData(deps *Dependencies, ctx context.Context) (*demoSeedResult, error) {
	createdBy := ""

	result := &demoSeedResult{
		Seeded:   true,
		DemoMode: true,
	}

	// ── Demo user ──
	hash, err := auth.HashPassword(demoPassword)
	if err != nil {
		return nil, err
	}
	demoUser, err := deps.Users.Create(ctx, demoEmail, hash, "Demo User", "admin")
	if err != nil {
		return nil, err
	}
	createdBy = demoUser.ID
	result.User = &demoSeededUser{
		ID:          demoUser.ID,
		Email:       demoUser.Email,
		DisplayName: demoUser.DisplayName,
		Role:        demoUser.Role,
	}

	// ── Demo devices (prefix "demo-") ──
	demoDevices := []struct {
		hostname, os, arch string
	}{
		{"demo-web-01", "ubuntu", "amd64"},
		{"demo-db-01", "ubuntu", "amd64"},
		{"demo-edge-01", "debian", "arm64"},
		{"demo-edge-02", "debian", "arm64"},
		{"demo-worker-01", "alpine", "amd64"},
	}
	for _, dd := range demoDevices {
		device, err := deps.Devices.Create(ctx, createdBy, dd.hostname, dd.os, dd.arch, demoVersion, "demo-"+dd.hostname)
		if err != nil {
			return nil, err
		}
		if deps.Pool != nil {
			_, _ = deps.Pool.Exec(ctx,
				`UPDATE devices SET labels = $2, metadata = $3 WHERE id = $1`,
				device.ID,
				`{"demo": "true", "role": "`+demoRole(dd.hostname)+`"}`,
				`{"demo": "true", "created_by": "demo-seed"}`,
			)
		}
		result.Devices = append(result.Devices, demoSeededEntity{ID: device.ID, Name: device.Hostname})
	}

	// ── Marketplace plugins (prefix "demo-") ──
	marketplace := []*db.MarketplacePlugin{
		{
			Name:            demoPluginPrefix + "nginx-proxy",
			DisplayName:     "Nginx Proxy",
			Description:     "Demo reverse proxy plugin",
			Author:          "RenOS",
			Category:        "networking",
			Version:         demoVersion,
			MinRenOSVersion: "0.1.0",
			Tags:            json.RawMessage(`["demo","proxy"]`),
			Verified:        true,
		},
		{
			Name:            demoPluginPrefix + "postgres-monitor",
			DisplayName:     "Postgres Monitor",
			Description:     "Demo PostgreSQL monitoring plugin",
			Author:          "RenOS",
			Category:        "monitoring",
			Version:         demoVersion,
			MinRenOSVersion: "0.1.0",
			Tags:            json.RawMessage(`["demo","database"]`),
			Verified:        true,
		},
		{
			Name:            demoPluginPrefix + "alertmanager-smtp",
			DisplayName:     "Alertmanager SMTP",
			Description:     "Demo SMTP alerting plugin",
			Author:          "RenOS",
			Category:        "notifications",
			Version:         demoVersion,
			MinRenOSVersion: "0.1.0",
			Tags:            json.RawMessage(`["demo","alerts"]`),
			Verified:        true,
		},
	}
	if deps.MarketplacePlugins != nil {
		for _, p := range marketplace {
			if err := deps.MarketplacePlugins.Upsert(ctx, p); err != nil {
				return nil, err
			}
			result.Marketplace = append(result.Marketplace, demoSeededEntity{ID: p.ID, Name: p.Name})
		}
	}

	// ── Installed plugins ──
	if deps.InstalledPlugins != nil {
		installed := []*db.InstalledPlugin{
			{
				Name:        demoPluginPrefix + "nginx-proxy",
				DisplayName: "Nginx Proxy",
				Version:     demoVersion,
				Author:      "RenOS",
				Description: "Demo reverse proxy plugin",
				PluginType:  "proxy",
				Status:      "enabled",
				Permissions: json.RawMessage(`[]`),
				Config:      json.RawMessage(`{"enabled":true}`),
			},
			{
				Name:        demoPluginPrefix + "alertmanager-smtp",
				DisplayName: "Alertmanager SMTP",
				Version:     demoVersion,
				Author:      "RenOS",
				Description: "Demo SMTP alerting plugin",
				PluginType:  "notifications",
				Status:      "enabled",
				Permissions: json.RawMessage(`[]`),
				Config:      json.RawMessage(`{"enabled":true}`),
			},
		}
		for _, p := range installed {
			if err := deps.InstalledPlugins.Create(ctx, p); err != nil {
				return nil, err
			}
			result.InstalledPlugins = append(result.InstalledPlugins, demoSeededEntity{ID: p.ID, Name: p.Name})
		}
	}

	// ── Demo workflows + steps ──
	if deps.Workflows != nil {
		wf := &db.Workflow{
			Name:          "Server Health Check",
			Description:   "Periodic server health check demo workflow",
			Enabled:       true,
			TriggerType:   "schedule",
			TriggerConfig: json.RawMessage(`{"cron":"*/10 * * * *"}`),
			Tags:          json.RawMessage(`["demo","health"]`),
			CreatedBy:     &createdBy,
		}
		if err := deps.Workflows.Create(ctx, wf); err != nil {
			return nil, err
		}
		result.Workflows = append(result.Workflows, demoSeededEntity{ID: wf.ID, Name: wf.Name})

		if deps.WorkflowSteps != nil {
			steps := []*db.WorkflowStep{
				{
					WorkflowID:   wf.ID,
					StepOrder:    1,
					Name:         "Check CPU load",
					StepType:     "command",
					Config:       json.RawMessage(`{"command":"uptime"}`),
					ErrorHandler: json.RawMessage(`{}`),
				},
				{
					WorkflowID:   wf.ID,
					StepOrder:    2,
					Name:         "Check disk usage",
					StepType:     "command",
					Config:       json.RawMessage(`{"command":"df -h"}`),
					ErrorHandler: json.RawMessage(`{}`),
				},
			}
			for _, s := range steps {
				if err := deps.WorkflowSteps.Create(ctx, s); err != nil {
					return nil, err
				}
			}
		}
	}

	// ── Demo alert rules ──
	if deps.AlertRules != nil {
		alertRules := []*db.AlertRule{
			{
				Name:                 "CPU Usage High",
				Description:          "Demo alert: sustained high CPU usage",
				MetricName:           "cpu.usage",
				Condition:            ">",
				Threshold:            85,
				DurationSeconds:      300,
				Severity:             "critical",
				Enabled:              true,
				NotificationChannels: json.RawMessage(`["smtp"]`),
			},
			{
				Name:                 "Disk Usage High",
				Description:          "Demo alert: disk filling up",
				MetricName:           "disk.usage",
				Condition:            ">",
				Threshold:            90,
				DurationSeconds:      600,
				Severity:             "warning",
				Enabled:              true,
				NotificationChannels: json.RawMessage(`["smtp"]`),
			},
		}
		for _, a := range alertRules {
			if err := deps.AlertRules.Create(ctx, a); err != nil {
				return nil, err
			}
			result.AlertRules = append(result.AlertRules, demoSeededEntity{ID: a.ID, Name: a.Name})
		}
	}

	// ── Demo monitoring dashboard + widgets ──
	if deps.MonitoringDashboards != nil {
		dash := &db.MonitoringDashboard{
			Name:        "Demo Overview",
			Description: "Demo dashboard showing fleet health",
			Layout:      json.RawMessage(`{"type":"grid"}`),
			IsDefault:   true,
			CreatedBy:   &createdBy,
		}
		if err := deps.MonitoringDashboards.Create(ctx, dash); err != nil {
			return nil, err
		}
		result.Dashboards = append(result.Dashboards, demoSeededEntity{ID: dash.ID, Name: dash.Name})

		if deps.MonitoringWidgets != nil {
			widgets := []*db.MonitoringWidget{
				{
					DashboardID: dash.ID,
					WidgetType:  "cpu",
					Title:       "CPU Usage",
					Config:      json.RawMessage(`{"metric":"cpu.usage"}`),
					PositionX:   0, PositionY: 0, Width: 6, Height: 3,
				},
				{
					DashboardID: dash.ID,
					WidgetType:  "disk",
					Title:       "Disk Usage",
					Config:      json.RawMessage(`{"metric":"disk.usage"}`),
					PositionX:   6, PositionY: 0, Width: 6, Height: 3,
				},
				{
					DashboardID: dash.ID,
					WidgetType:  "memory",
					Title:       "Memory Usage",
					Config:      json.RawMessage(`{"metric":"memory.usage"}`),
					PositionX:   0, PositionY: 3, Width: 6, Height: 3,
				},
			}
			for _, wgt := range widgets {
				if err := deps.MonitoringWidgets.Create(ctx, wgt); err != nil {
					return nil, err
				}
			}
		}
	}

	// ── Demo notification provider ──
	if deps.NotifProviders != nil {
		provider := &db.NotificationProvider{
			Name:         "Demo SMTP",
			ProviderType: "smtp",
			Config:       json.RawMessage(`{"host":"localhost","port":25}`),
			Enabled:      true,
		}
		if err := deps.NotifProviders.Create(ctx, provider); err != nil {
			return nil, err
		}
		result.Notifications = append(result.Notifications, demoSeededEntity{ID: provider.ID, Name: provider.Name})
	}

	// ── Demo job ──
	if deps.Jobs != nil {
		job, err := deps.Jobs.Create(ctx,
			demoJobPrefix+"update-check",
			json.RawMessage(`{"action":"check_updates"}`),
			len(result.Devices),
			createdBy,
			nil,
		)
		if err != nil {
			return nil, err
		}
		result.Jobs = append(result.Jobs, demoSeededEntity{ID: job.ID, Name: job.Type})
	}

	return result, nil
}

// demoRole maps a demo device hostname to a coarse role used in device labels.
func demoRole(hostname string) string {
	switch {
	case strings.Contains(hostname, "db"):
		return "database"
	case strings.Contains(hostname, "edge"):
		return "edge"
	case strings.Contains(hostname, "worker"):
		return "worker"
	default:
		return "web"
	}
}
