package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/renos-project/renos/core/internal/db"
)

// --- Metrics ---

func recordMetricHandler(metrics *db.MonitoringMetricRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var m db.MonitoringMetric
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		if m.MetricName == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "metric_name is required")
			return
		}
		if m.RecordedAt.IsZero() {
			m.RecordedAt = time.Now()
		}
		if m.Labels == nil {
			m.Labels = json.RawMessage(`{}`)
		}
		if err := metrics.Record(r.Context(), &m); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to record metric: "+err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
	}
}

func queryMetricsHandler(metrics *db.MonitoringMetricRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metricName := r.URL.Query().Get("name")
		deviceID := r.URL.Query().Get("device_id")
		sinceStr := r.URL.Query().Get("since")
		if metricName == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "name parameter is required")
			return
		}
		since := time.Now().Add(-1 * time.Hour)
		if sinceStr != "" {
			if t, err := time.Parse(time.RFC3339, sinceStr); err == nil {
				since = t
			}
		}
		var list []*db.MonitoringMetric
		var err error
		if deviceID != "" {
			list, err = metrics.QueryByDevice(r.Context(), deviceID, since, 500)
		} else {
			list, err = metrics.Query(r.Context(), deviceID, metricName, since, 500)
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to query metrics: "+err.Error())
			return
		}
		if list == nil {
			list = []*db.MonitoringMetric{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}

func latestMetricsByDeviceHandler(metrics *db.MonitoringMetricRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID := chi.URLParam(r, "deviceId")
		list, err := metrics.LatestByDevice(r.Context(), deviceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to get latest metrics: "+err.Error())
			return
		}
		if list == nil {
			list = []*db.MonitoringMetric{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}

// --- Alert Rules ---

func listAlertRulesHandler(alertRules *db.AlertRuleRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := alertRules.ListAll(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list rules: "+err.Error())
			return
		}
		if list == nil {
			list = []*db.AlertRule{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}

func createAlertRuleHandler(alertRules *db.AlertRuleRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var rule db.AlertRule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		if rule.Name == "" || rule.MetricName == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "name and metric_name are required")
			return
		}
		if rule.Condition == "" {
			rule.Condition = "above"
		}
		if rule.Severity == "" {
			rule.Severity = "warning"
		}
		rule.Enabled = true
		if rule.NotificationChannels == nil {
			rule.NotificationChannels = json.RawMessage(`[]`)
		}
		if err := alertRules.Create(r.Context(), &rule); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to create rule: "+err.Error())
			return
		}
		audits.Create(r.Context(), "user", getUserID(r), "alert_rule.create", "rule", rule.ID)
		writeJSON(w, http.StatusCreated, rule)
	}
}

func updateAlertRuleHandler(alertRules *db.AlertRuleRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		existing, err := alertRules.FindByID(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", "rule not found")
			return
		}
		var update db.AlertRule
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		if update.Name != "" {
			existing.Name = update.Name
		}
		if update.Description != "" {
			existing.Description = update.Description
		}
		if update.MetricName != "" {
			existing.MetricName = update.MetricName
		}
		if update.Condition != "" {
			existing.Condition = update.Condition
		}
		if update.Threshold != 0 {
			existing.Threshold = update.Threshold
		}
		if update.Severity != "" {
			existing.Severity = update.Severity
		}
		if update.NotificationChannels != nil {
			existing.NotificationChannels = update.NotificationChannels
		}
		existing.Enabled = update.Enabled
		if err := alertRules.Update(r.Context(), existing); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to update rule: "+err.Error())
			return
		}
		audits.Create(r.Context(), "user", getUserID(r), "alert_rule.update", "rule", id)
		writeJSON(w, http.StatusOK, existing)
	}
}

func deleteAlertRuleHandler(alertRules *db.AlertRuleRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := alertRules.Delete(r.Context(), id); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete rule: "+err.Error())
			return
		}
		audits.Create(r.Context(), "user", getUserID(r), "alert_rule.delete", "rule", id)
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

// --- Alert Instances ---

func listActiveAlertsHandler(alerts *db.AlertInstanceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := alerts.ListActive(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list alerts: "+err.Error())
			return
		}
		if list == nil {
			list = []*db.AlertInstance{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}

func listRecentAlertsHandler(alerts *db.AlertInstanceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := alerts.ListRecent(r.Context(), 100)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list alerts: "+err.Error())
			return
		}
		if list == nil {
			list = []*db.AlertInstance{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}

func acknowledgeAlertHandler(alerts *db.AlertInstanceRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		uid := getUserID(r)
		if err := alerts.Acknowledge(r.Context(), id, uid); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to acknowledge alert: "+err.Error())
			return
		}
		audits.Create(r.Context(), "user", uid, "alert.acknowledge", "alert", id)
		writeJSON(w, http.StatusOK, map[string]string{"status": "acknowledged"})
	}
}

func resolveAlertHandler(alerts *db.AlertInstanceRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := alerts.Resolve(r.Context(), id); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to resolve alert: "+err.Error())
			return
		}
		audits.Create(r.Context(), "user", getUserID(r), "alert.resolve", "alert", id)
		writeJSON(w, http.StatusOK, map[string]string{"status": "resolved"})
	}
}

func alertSummaryHandler(alerts *db.AlertInstanceRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		summary, err := alerts.GetSummary(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to get summary: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, summary)
	}
}

// --- Dashboards ---

func listMonitoringDashboardsHandler(dashboards *db.MonitoringDashboardRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := dashboards.ListAll(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list dashboards: "+err.Error())
			return
		}
		if list == nil {
			list = []*db.MonitoringDashboard{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}

func getMonitoringDashboardHandler(dashboards *db.MonitoringDashboardRepository, widgets *db.MonitoringWidgetRepository, provider widgetDataProvider) http.HandlerFunc {
	type dashDetail struct {
		*db.MonitoringDashboard
		Widgets []*db.MonitoringWidget `json:"widgets"`
		Data    map[string]json.RawMessage `json:"widget_data"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		d, err := dashboards.FindByID(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", "dashboard not found")
			return
		}
		ws, _ := widgets.ListByDashboard(r.Context(), id)
		if ws == nil {
			ws = []*db.MonitoringWidget{}
		}
		data := hydrateDashboardData(r.Context(), provider, ws)
		writeJSON(w, http.StatusOK, dashDetail{MonitoringDashboard: d, Widgets: ws, Data: data})
	}
}

// hydrateDashboardData resolves each widget to display data keyed by widget ID.
// Widgets whose provider fails are omitted so a single bad widget never breaks
// the whole dashboard.
func hydrateDashboardData(ctx context.Context, provider widgetDataProvider, ws []*db.MonitoringWidget) map[string]json.RawMessage {
	if provider == nil {
		return map[string]json.RawMessage{}
	}
	data := make(map[string]json.RawMessage, len(ws))
	for _, widget := range ws {
		hydrated, err := provider.Hydrate(ctx, widget)
		if err != nil {
			continue
		}
		data[widget.ID] = hydrated
	}
	return data
}

func createMonitoringDashboardHandler(dashboards *db.MonitoringDashboardRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var d db.MonitoringDashboard
		if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		if d.Name == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "name is required")
			return
		}
		if d.Layout == nil {
			d.Layout = json.RawMessage(`[]`)
		}
		uid := getUserID(r)
		d.CreatedBy = &uid
		if err := dashboards.Create(r.Context(), &d); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to create dashboard: "+err.Error())
			return
		}
		audits.Create(r.Context(), "user", uid, "dashboard.create", "dashboard", d.ID)
		writeJSON(w, http.StatusCreated, d)
	}
}

func createMonitoringWidgetHandler(widgets *db.MonitoringWidgetRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var w2 db.MonitoringWidget
		if err := json.NewDecoder(r.Body).Decode(&w2); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		if w2.Title == "" {
			writeError(w, http.StatusBadRequest, "invalid_request", "title is required")
			return
		}
		if w2.WidgetType == "" {
			w2.WidgetType = "chart"
		}
		w2.DashboardID = id
		if w2.Config == nil {
			w2.Config = json.RawMessage(`{}`)
		}
		if w2.Width == 0 {
			w2.Width = 6
		}
		if w2.Height == 0 {
			w2.Height = 4
		}
		if err := widgets.Create(r.Context(), &w2); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to create widget: "+err.Error())
			return
		}
		audits.Create(r.Context(), "user", getUserID(r), "dashboard.widget.create", "dashboard", id)
		writeJSON(w, http.StatusCreated, w2)
	}
}

// updateMonitoringDashboardHandler saves a dashboard's metadata and replaces its
// full widget set atomically. This is the persistence target for the dashboard
// builder (drag/resize/add/remove in one request).
func updateMonitoringDashboardHandler(dashboards *db.MonitoringDashboardRepository, widgets *db.MonitoringWidgetRepository, audits *db.AuditRepository) http.HandlerFunc {
	type updateRequest struct {
		Name        string             `json:"name"`
		Description string             `json:"description"`
		IsDefault   bool               `json:"is_default"`
		Layout      json.RawMessage    `json:"layout"`
		Widgets     []*db.MonitoringWidget `json:"widgets"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		d, err := dashboards.FindByID(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", "dashboard not found")
			return
		}
		var req updateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		if req.Name != "" {
			d.Name = req.Name
		}
		d.Description = req.Description
		d.IsDefault = req.IsDefault
		if req.Layout != nil {
			d.Layout = req.Layout
		}
		if err := dashboards.Update(r.Context(), d); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to update dashboard: "+err.Error())
			return
		}
		if req.Widgets != nil {
			if err := widgets.ReplaceForDashboard(r.Context(), id, req.Widgets); err != nil {
				writeError(w, http.StatusInternalServerError, "internal_error", "failed to update widgets: "+err.Error())
				return
			}
		}
		audits.Create(r.Context(), "user", getUserID(r), "dashboard.update", "dashboard", id)
		writeJSON(w, http.StatusOK, d)
	}
}

func deleteMonitoringWidgetHandler(widgets *db.MonitoringWidgetRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := widgets.DeleteByID(r.Context(), id); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete widget: "+err.Error())
			return
		}
		audits.Create(r.Context(), "user", getUserID(r), "dashboard.widget.delete", "dashboard", id)
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

func deleteMonitoringDashboardHandler(dashboards *db.MonitoringDashboardRepository, widgets *db.MonitoringWidgetRepository, audits *db.AuditRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		_ = widgets.DeleteByDashboard(r.Context(), id)
		if err := dashboards.Delete(r.Context(), id); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete dashboard: "+err.Error())
			return
		}
		audits.Create(r.Context(), "user", getUserID(r), "dashboard.delete", "dashboard", id)
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

// --- Logs ---

func listMonitoringLogsHandler(logs *db.MonitoringLogRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		level := r.URL.Query().Get("level")
		query := r.URL.Query().Get("q")
		var list []*db.MonitoringLog
		var err error
		if query != "" || level != "" {
			list, err = logs.Search(r.Context(), query, level, 100)
		} else {
			list, err = logs.ListRecent(r.Context(), 100)
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to list logs: "+err.Error())
			return
		}
		if list == nil {
			list = []*db.MonitoringLog{}
		}
		writeJSON(w, http.StatusOK, list)
	}
}

func monitoringLogStatsHandler(logs *db.MonitoringLogRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := logs.GetStats(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to get stats: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, stats)
	}
}
