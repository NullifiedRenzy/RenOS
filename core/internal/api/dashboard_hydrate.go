package api

import (
	"context"
	"encoding/json"
	"time"

	"github.com/renos-project/renos/core/internal/db"
)

// widgetDataProvider hydrates config-driven monitoring widgets. Implementations
// may be backed by the database (DefaultWidgetDataProvider) or by a fake in tests.
type widgetDataProvider interface {
	Hydrate(ctx context.Context, w *db.MonitoringWidget) (json.RawMessage, error)
}

// DefaultWidgetDataProvider produces display data for the config-driven widget
// types introduced by M65. Unsupported types resolve to an empty JSON object so
// the frontend can always read a value from the data map.
type DefaultWidgetDataProvider struct {
	Devices   *db.DeviceRepository
	Alerts    *db.AlertInstanceRepository
	Logs      *db.MonitoringLogRepository
	Workflows *db.WorkflowRepository
	Runs      *db.WorkflowRunRepository
	AI        *db.AIConversationRepository
	Containers *db.ContainerRepository
	Proxmox   *db.ProxmoxRepository
	Metrics   *db.MonitoringMetricRepository
}

func (p *DefaultWidgetDataProvider) Hydrate(ctx context.Context, w *db.MonitoringWidget) (json.RawMessage, error) {
	switch w.WidgetType {
	case "devices-overview":
		return p.hydrateDevicesOverview(ctx)
	case "device-status":
		return p.hydrateDeviceStatus(ctx)
	case "cpu":
		return p.hydrateMetric(ctx, "cpu_usage")
	case "memory":
		return p.hydrateMetric(ctx, "memory_usage")
	case "alerts":
		return p.hydrateAlerts(ctx)
	case "logs":
		return p.hydrateLogs(ctx)
	case "workflows":
		return p.hydrateWorkflows(ctx)
	case "ai-usage":
		return p.hydrateAIUsage(ctx)
	case "containers":
		return p.hydrateContainers(ctx)
	case "proxmox":
		return p.hydrateProxmox(ctx)
	default:
		return json.RawMessage(`{}`), nil
	}
}

type deviceOverviewData struct {
	Total      int64           `json:"total"`
	Online     int64           `json:"online"`
	Offline    int64           `json:"offline"`
	Error      int64           `json:"error"`
	ByStatus   map[string]int64 `json:"by_status"`
	OSDist     map[string]int64 `json:"os_dist"`
	Unreachable int64          `json:"unreachable"`
}

func (p *DefaultWidgetDataProvider) hydrateDevicesOverview(ctx context.Context) (json.RawMessage, error) {
	counts, err := p.Devices.CountByStatus(ctx)
	if err != nil {
		return nil, err
	}
	if counts.OSDist == nil {
		counts.OSDist = map[string]int64{}
	}
	// The overview is computed from status distribution; CountByStatus only
	// returns total/online, so derive the rest from the full device list.
	byStatus := map[string]int64{}
	offline := int64(0)
	errorN := int64(0)
	devices, err := p.Devices.FindAll(ctx)
	if err == nil {
		for _, d := range devices {
			byStatus[d.Status]++
			switch d.Status {
			case "offline":
				offline++
			case "error":
				errorN++
			}
		}
	}
	if byStatus == nil {
		byStatus = map[string]int64{}
	}
	return json.Marshal(deviceOverviewData{
		Total:    counts.Total,
		Online:   counts.Online,
		Offline:  offline,
		Error:    errorN,
		ByStatus: byStatus,
		OSDist:   counts.OSDist,
	})
}

type deviceStatusData struct {
	Hostname string `json:"hostname"`
	Status   string `json:"status"`
	OS       string `json:"os"`
	LastSeen string `json:"last_seen"`
}

func (p *DefaultWidgetDataProvider) hydrateDeviceStatus(ctx context.Context) (json.RawMessage, error) {
	devices, err := p.Devices.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]deviceStatusData, 0, len(devices))
	for _, d := range devices {
		lastSeen := ""
		if d.LastSeen != nil {
			lastSeen = d.LastSeen.Format(time.RFC3339)
		}
		out = append(out, deviceStatusData{
			Hostname: d.Hostname,
			Status:   d.Status,
			OS:       d.OS,
			LastSeen: lastSeen,
		})
	}
	return json.Marshal(out)
}

type metricSample struct {
	Value     float64 `json:"value"`
	DeviceID  *string `json:"device_id,omitempty"`
	RecordedAt string `json:"recorded_at"`
}

func (p *DefaultWidgetDataProvider) hydrateMetric(ctx context.Context, metricName string) (json.RawMessage, error) {
	since := time.Now().Add(-1 * time.Hour)
	points, err := p.Metrics.Query(ctx, "", metricName, since, 100)
	if err != nil {
		return nil, err
	}
	out := make([]metricSample, 0, len(points))
	for _, m := range points {
		out = append(out, metricSample{
			Value:      m.MetricValue,
			DeviceID:   m.DeviceID,
			RecordedAt: m.RecordedAt.Format(time.RFC3339),
		})
	}
	return json.Marshal(map[string]any{"metric": metricName, "samples": out})
}

type alertSummaryData struct {
	Firing   int      `json:"firing"`
	Active   []string `json:"active"`
}

func (p *DefaultWidgetDataProvider) hydrateAlerts(ctx context.Context) (json.RawMessage, error) {
	alerts, err := p.Alerts.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	if alerts == nil {
		alerts = []*db.AlertInstance{}
	}
	messages := make([]string, 0, len(alerts))
	for _, a := range alerts {
		msg := a.Message
		if msg == "" {
			msg = "Alert on " + a.RuleID[:min(len(a.RuleID), 8)]
		}
		messages = append(messages, msg)
	}
	return json.Marshal(alertSummaryData{Firing: len(alerts), Active: messages})
}

type logEntryData struct {
	Level     string `json:"level"`
	Source    string `json:"source"`
	Message   string `json:"message"`
	RecordedAt string `json:"recorded_at"`
}

func (p *DefaultWidgetDataProvider) hydrateLogs(ctx context.Context) (json.RawMessage, error) {
	logs, err := p.Logs.ListRecent(ctx, 20)
	if err != nil {
		return nil, err
	}
	if logs == nil {
		logs = []*db.MonitoringLog{}
	}
	out := make([]logEntryData, 0, len(logs))
	for _, l := range logs {
		out = append(out, logEntryData{
			Level:      l.Level,
			Source:     l.Source,
			Message:    l.Message,
			RecordedAt: l.RecordedAt.Format(time.RFC3339),
		})
	}
	return json.Marshal(out)
}

type workflowSummaryData struct {
	Total    int    `json:"total"`
	Enabled  int    `json:"enabled"`
	RecentRuns []runRef `json:"recent_runs"`
}

type runRef struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func (p *DefaultWidgetDataProvider) hydrateWorkflows(ctx context.Context) (json.RawMessage, error) {
	workflows, err := p.Workflows.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	if workflows == nil {
		workflows = []*db.Workflow{}
	}
	enabled := 0
	for _, w := range workflows {
		if w.Enabled {
			enabled++
		}
	}
	runs, err := p.Runs.ListAll(ctx, 10)
	if err != nil {
		runs = nil
	}
	recent := []runRef{}
	if runs != nil {
		for _, r := range runs {
			recent = append(recent, runRef{ID: r.ID, Status: r.Status})
		}
	}
	return json.Marshal(workflowSummaryData{Total: len(workflows), Enabled: enabled, RecentRuns: recent})
}

func (p *DefaultWidgetDataProvider) hydrateAIUsage(ctx context.Context) (json.RawMessage, error) {
	conversations, err := p.AI.Count(ctx)
	if err != nil {
		return nil, err
	}
	type aiUsageData struct {
		Conversations int64 `json:"conversations"`
	}
	return json.Marshal(aiUsageData{Conversations: conversations})
}

type containerSummaryData struct {
	Total   int `json:"total"`
	Running int `json:"running"`
	Stopped int `json:"stopped"`
}

func (p *DefaultWidgetDataProvider) hydrateContainers(ctx context.Context) (json.RawMessage, error) {
	// Containers are stored per device; aggregate counts across all devices.
	devices, err := p.Devices.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	summary := containerSummaryData{}
	for _, d := range devices {
		list, err := p.Containers.ListByDevice(ctx, d.ID)
		if err != nil {
			continue
		}
		for _, c := range list {
			summary.Total++
			if c.State == "running" {
				summary.Running++
			} else {
				summary.Stopped++
			}
		}
	}
	return json.Marshal(summary)
}

type proxmoxSummaryData struct {
	Connections int `json:"connections"`
	Nodes       int `json:"nodes"`
	VMs         int `json:"vms"`
	LXCs        int `json:"lxcs"`
}

func (p *DefaultWidgetDataProvider) hydrateProxmox(ctx context.Context) (json.RawMessage, error) {
	devices, err := p.Devices.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	summary := proxmoxSummaryData{}
	for _, d := range devices {
		conns, err := p.Proxmox.ListConnectionsByDevice(ctx, d.ID)
		if err != nil {
			continue
		}
		for _, c := range conns {
			summary.Connections++
			if nodes, err := p.Proxmox.ListNodes(ctx, c.ID); err == nil {
				summary.Nodes += len(nodes)
			}
			if vms, err := p.Proxmox.ListVMs(ctx, c.ID); err == nil {
				summary.VMs += len(vms)
			}
			if lxcs, err := p.Proxmox.ListLXCs(ctx, c.ID); err == nil {
				summary.LXCs += len(lxcs)
			}
		}
	}
	return json.Marshal(summary)
}
