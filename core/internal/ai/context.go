package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/renos-project/renos/core/internal/db"
)

type ContextBuilder struct {
	devices          *db.DeviceRepository
	commands         *db.CommandRepository
	containers       *db.ContainerRepository
	composeStacks    *db.ComposeStackRepository
	processes        *db.ProcessRepository
	jobs             *db.JobRepository
	schedules        *db.ScheduleRepository
	policies         *db.PolicyRepository
	installedPlugins *db.InstalledPluginRepository
	systemLogs       *db.SystemLogRepository
	containerMetrics *db.ContainerMetricsRepository
	proxmox          *db.ProxmoxRepository
}

type AIContext struct {
	Devices         []DeviceSummary      `json:"devices"`
	RecentJobs      []JobSummary         `json:"recent_jobs"`
	Containers      []ContainerSummary   `json:"containers"`
	ComposeStacks   []ComposeSummary     `json:"compose_stacks"`
	Policies        []PolicySummary      `json:"policies"`
	Plugins         []PluginSummary      `json:"plugins"`
	Alerts          []Alert              `json:"alerts"`
	Metrics         *MetricsSummary      `json:"metrics,omitempty"`
	ProxmoxNodes    int                  `json:"proxmox_nodes"`
	OnlineDevices   int                  `json:"online_devices"`
	OfflineDevices  int                  `json:"offline_devices"`
	TotalDevices    int                  `json:"total_devices"`
}

type DeviceSummary struct {
	Name      string `json:"name"`
	OS        string `json:"os"`
	Status    string `json:"status"`
	IPAddress string `json:"ip_address"`
}

type JobSummary struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	DeviceID string `json:"device_id"`
}

type ContainerSummary struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Device string `json:"device"`
}

type ComposeSummary struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Device string `json:"device"`
}

type PolicySummary struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type PluginSummary struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Status  string `json:"status"`
}

type Alert struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Source   string `json:"source"`
}

type MetricsSummary struct {
	AvgCPU     float64 `json:"avg_cpu"`
	AvgMemory  float64 `json:"avg_memory"`
	AvgDisk    float64 `json:"avg_disk"`
}

func NewContextBuilder(
	devices *db.DeviceRepository,
	commands *db.CommandRepository,
	containers *db.ContainerRepository,
	composeStacks *db.ComposeStackRepository,
	processes *db.ProcessRepository,
	jobs *db.JobRepository,
	schedules *db.ScheduleRepository,
	policies *db.PolicyRepository,
	installedPlugins *db.InstalledPluginRepository,
	systemLogs *db.SystemLogRepository,
	containerMetrics *db.ContainerMetricsRepository,
	proxmox *db.ProxmoxRepository,
) *ContextBuilder {
	return &ContextBuilder{
		devices:          devices,
		commands:         commands,
		containers:       containers,
		composeStacks:    composeStacks,
		processes:        processes,
		jobs:             jobs,
		schedules:        schedules,
		policies:         policies,
		installedPlugins: installedPlugins,
		systemLogs:       systemLogs,
		containerMetrics: containerMetrics,
		proxmox:          proxmox,
	}
}

func (cb *ContextBuilder) Build(ctx context.Context) (*AIContext, error) {
	actx := &AIContext{}

	devices, err := cb.devices.FindAll(ctx)
	if err == nil {
		for _, d := range devices {
			status := d.Status
			actx.Devices = append(actx.Devices, DeviceSummary{
				Name:      d.Hostname,
				OS:        d.OS,
				Status:    status,
				IPAddress: "",
			})
			if status == "online" {
				actx.OnlineDevices++
			} else {
				actx.OfflineDevices++
			}
			actx.TotalDevices++
		}
	}

	if jobsRepo := cb.jobs; jobsRepo != nil {
		recent, err := jobsRepo.FindAll(ctx, 10, 0)
		if err == nil {
			for i, j := range recent {
				if i >= 10 {
					break
				}
				actx.RecentJobs = append(actx.RecentJobs, JobSummary{
					ID:     j.ID,
					Name:   j.Type,
					Status: j.Status,
				})
			}
		}
	}

	if policiesRepo := cb.policies; policiesRepo != nil {
		policyList, err := policiesRepo.FindAll(ctx)
		if err == nil {
			for _, p := range policyList {
				actx.Policies = append(actx.Policies, PolicySummary{
					Name:    p.Name,
					Enabled: p.Enabled,
				})
			}
		}
	}

	if pluginRepo := cb.installedPlugins; pluginRepo != nil {
		pluginList, err := pluginRepo.ListAll(ctx)
		if err == nil {
			for _, p := range pluginList {
				actx.Plugins = append(actx.Plugins, PluginSummary{
					Name:    p.Name,
					Version: p.Version,
					Status:  p.Status,
				})
			}
		}
	}

	actx.Alerts = cb.detectAlerts(ctx, devices, actx)

	return actx, nil
}

func (cb *ContextBuilder) detectAlerts(ctx context.Context, devices []*db.Device, actx *AIContext) []Alert {
	var alerts []Alert
	for _, d := range devices {
		if d.Status != "online" {
			alerts = append(alerts, Alert{
				Severity: "warning",
				Message:  fmt.Sprintf("Device %s is offline", d.Hostname),
				Source:   "device",
			})
		}
	}
	if actx.OfflineDevices > 0 && actx.TotalDevices > 0 {
		pct := float64(actx.OfflineDevices) / float64(actx.TotalDevices) * 100
		if pct > 50 {
			alerts = append(alerts, Alert{
				Severity: "critical",
				Message:  fmt.Sprintf("%.0f%% of devices are offline", pct),
				Source:   "fleet",
			})
		}
	}
	return alerts
}

func (actx *AIContext) ToPromptString() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Fleet: %d devices (%d online, %d offline)\n", actx.TotalDevices, actx.OnlineDevices, actx.OfflineDevices))
	if len(actx.Devices) > 0 {
		sb.WriteString("\nDevices:\n")
		for _, d := range actx.Devices {
			sb.WriteString(fmt.Sprintf("  - %s [%s] OS=%s IP=%s\n", d.Name, d.Status, d.OS, d.IPAddress))
		}
	}
	if len(actx.RecentJobs) > 0 {
		sb.WriteString("\nRecent Jobs:\n")
		for _, j := range actx.RecentJobs {
			sb.WriteString(fmt.Sprintf("  - %s [%s] device=%s\n", j.Name, j.Status, j.DeviceID))
		}
	}
	if len(actx.Policies) > 0 {
		sb.WriteString("\nPolicies:\n")
		for _, p := range actx.Policies {
			status := "disabled"
			if p.Enabled {
				status = "enabled"
			}
			sb.WriteString(fmt.Sprintf("  - %s [%s]\n", p.Name, status))
		}
	}
	if len(actx.Plugins) > 0 {
		sb.WriteString("\nPlugins:\n")
		for _, p := range actx.Plugins {
			sb.WriteString(fmt.Sprintf("  - %s v%s [%s]\n", p.Name, p.Version, p.Status))
		}
	}
	if len(actx.Alerts) > 0 {
		sb.WriteString("\nActive Alerts:\n")
		for _, a := range actx.Alerts {
			sb.WriteString(fmt.Sprintf("  - [%s] %s (%s)\n", a.Severity, a.Message, a.Source))
		}
	}
	return sb.String()
}
