"use client";

import { useState } from "react";
import { useAuth } from "@/hooks/use-auth";
import {
  useAlertRules,
  useActiveAlerts,
  useRecentAlerts,
  useAlertSummary,
  useMonitoringDashboards,
  useMonitoringLogs,
  useMonitoringLogStats,
  useResolveAlert,
  useAcknowledgeAlert,
  useCreateAlertRule,
  useDeleteAlertRule,
} from "@/hooks/use-monitoring";
import type { AlertRule, AlertInstance, MonitoringLog } from "@/lib/types";
import { DashboardBuilder } from "@/components/monitoring/dashboard-builder";

type MonitoringTab = "overview" | "alerts" | "logs" | "dashboards";

const severityColors: Record<string, string> = {
  info: "var(--color-info)",
  warning: "var(--color-warning)",
  critical: "var(--color-error)",
};

const logLevelColors: Record<string, string> = {
  debug: "var(--color-text-tertiary)",
  info: "var(--color-info)",
  warn: "var(--color-warning)",
  error: "var(--color-error)",
  fatal: "var(--color-error)",
};

const alertStatusColors: Record<string, string> = {
  firing: "var(--color-error)",
  resolved: "var(--color-success)",
  silenced: "var(--color-text-tertiary)",
};

export default function MonitoringPage() {
  const { can } = useAuth();
  const [activeTab, setActiveTab] = useState<MonitoringTab>("overview");

  return (
    <div className="space-y-6">
      <h1 className="text-headline-1 text-[var(--color-text-primary)]">Monitoring</h1>

      <div className="flex gap-2 border-b border-[var(--color-border)] pb-0">
        {(["overview", "alerts", "logs", "dashboards"] as MonitoringTab[]).map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 text-body-small rounded-t-lg border border-b-0 transition-colors capitalize ${
              activeTab === tab
                ? "bg-[var(--color-bg-primary)] text-[var(--color-text-primary)] border-[var(--color-border)] font-medium"
                : "bg-transparent text-[var(--color-text-tertiary)] border-transparent hover:text-[var(--color-text-secondary)]"
            }`}
          >
            {tab}
          </button>
        ))}
      </div>

      <div className="mt-6">
        {activeTab === "overview" && <OverviewTab />}
        {activeTab === "alerts" && <AlertsTab />}
        {activeTab === "logs" && <LogsTab />}
        {activeTab === "dashboards" && <DashboardsTab />}
      </div>
    </div>
  );
}

function OverviewTab() {
  const { data: alertSummary } = useAlertSummary();
  const { data: activeAlerts } = useActiveAlerts();
  const { data: logStats } = useMonitoringLogStats();

  const firing = alertSummary?.firing ?? 0;
  const resolved = alertSummary?.resolved ?? 0;

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg p-4">
          <div className="text-caption text-[var(--color-text-tertiary)]">Firing Alerts</div>
          <div className="text-headline-2 mt-1" style={{ color: firing > 0 ? "var(--color-error)" : "var(--color-success)" }}>{firing}</div>
        </div>
        <div className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg p-4">
          <div className="text-caption text-[var(--color-text-tertiary)]">Resolved</div>
          <div className="text-headline-2 mt-1 text-[var(--color-success)]">{resolved}</div>
        </div>
        {Object.entries(logStats || {}).map(([level, count]) => (
          <div key={level} className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg p-4">
            <div className="text-caption text-[var(--color-text-tertiary)] capitalize">{level} Logs</div>
            <div className="text-headline-2 mt-1" style={{ color: logLevelColors[level] || "var(--color-text-primary)" }}>{count}</div>
          </div>
        ))}
      </div>

      {activeAlerts && activeAlerts.length > 0 && (
        <div className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg p-6">
          <h3 className="text-headline-3 text-[var(--color-text-primary)] mb-4">Active Alerts</h3>
          <div className="space-y-2">
            {activeAlerts.map((alert: AlertInstance) => (
              <div key={alert.id} className="flex items-center justify-between p-3 bg-[var(--color-bg-tertiary)] rounded-lg">
                <div className="flex items-center gap-3">
                  <span className="w-3 h-3 rounded-full" style={{ backgroundColor: alertStatusColors[alert.status] || "var(--color-text-tertiary)" }} />
                  <span className="text-body-small text-[var(--color-text-primary)]">{alert.message || `Alert on ${alert.rule_id.slice(0, 8)}`}</span>
                </div>
                <span className="text-caption" style={{ color: alertStatusColors[alert.status] }}>{alert.status}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function AlertsTab() {
  const { data: rules } = useAlertRules();
  const { data: recentAlerts } = useRecentAlerts();
  const [showCreate, setShowCreate] = useState(false);
  const deleteRule = useDeleteAlertRule();
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-headline-2 text-[var(--color-text-primary)]">Alert Rules</h2>
        <button onClick={() => setShowCreate(!showCreate)} className="px-3 py-1.5 bg-[var(--color-accent)] text-[var(--color-text-on-accent)] rounded-lg text-caption hover:opacity-90">
          {showCreate ? "Cancel" : "+ New Rule"}
        </button>
      </div>

      {showCreate && <CreateAlertRuleForm onClose={() => setShowCreate(false)} />}

      {rules && rules.length > 0 && (
        <div className="space-y-2">
          {rules.map((rule: AlertRule) => (
            <div key={rule.id} className="flex items-center justify-between p-4 bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg">
              <div>
                <div className="flex items-center gap-2">
                  <span className="w-2 h-2 rounded-full" style={{ backgroundColor: severityColors[rule.severity] || "var(--color-text-tertiary)" }} />
                  <span className="text-body-small text-[var(--color-text-primary)] font-medium">{rule.name}</span>
                  <span className="text-caption text-[var(--color-text-tertiary)]">{rule.metric_name} {rule.condition} {rule.threshold}</span>
                  <span className="text-caption px-2 py-0.5 rounded capitalize" style={{ backgroundColor: severityColors[rule.severity] + "20", color: severityColors[rule.severity] }}>
                    {rule.severity}
                  </span>
                </div>
                {rule.description && <p className="text-caption text-[var(--color-text-tertiary)] mt-1 ml-4">{rule.description}</p>}
              </div>
              <div className="flex items-center gap-2">
                <span className="text-caption" style={{ color: rule.enabled ? "var(--color-success)" : "var(--color-text-tertiary)" }}>
                  {rule.enabled ? "Enabled" : "Disabled"}
                </span>
                {confirmDelete === rule.id ? (
                  <div className="flex gap-2">
                    <button onClick={() => deleteRule.mutate(rule.id, { onSuccess: () => setConfirmDelete(null) })} className="text-caption text-[var(--color-error)] hover:underline">Confirm</button>
                    <button onClick={() => setConfirmDelete(null)} className="text-caption text-[var(--color-text-tertiary)] hover:underline">Cancel</button>
                  </div>
                ) : (
                  <button onClick={() => setConfirmDelete(rule.id)} className="text-caption text-[var(--color-text-tertiary)] hover:text-[var(--color-error)]">Delete</button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {recentAlerts && recentAlerts.length > 0 && (
        <div className="mt-6">
          <h2 className="text-headline-2 text-[var(--color-text-primary)] mb-4">Recent Alerts</h2>
          <div className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg overflow-hidden">
            <table className="w-full">
              <thead>
                <tr className="border-b border-[var(--color-border)]">
                  <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Status</th>
                  <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Message</th>
                  <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Value</th>
                  <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Started</th>
                  <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Actions</th>
                </tr>
              </thead>
              <tbody>
                {recentAlerts.map((alert: AlertInstance) => (
                  <tr key={alert.id} className="border-b border-[var(--color-border)] last:border-0 hover:bg-[var(--color-bg-tertiary)]">
                    <td className="px-4 py-3">
                      <span className="text-caption px-2 py-1 rounded capitalize" style={{ backgroundColor: (alertStatusColors[alert.status] || "var(--color-text-tertiary)") + "20", color: alertStatusColors[alert.status] || "var(--color-text-tertiary)" }}>
                        {alert.status}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-body-small text-[var(--color-text-primary)]">{alert.message || "—"}</td>
                    <td className="px-4 py-3 text-body-small text-[var(--color-text-secondary)]">{alert.value}</td>
                    <td className="px-4 py-3 text-caption text-[var(--color-text-tertiary)]">{new Date(alert.started_at).toLocaleString()}</td>
                    <td className="px-4 py-3">
                      {alert.status === "firing" && (
                        <AlertActions alertId={alert.id} />
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}

function AlertActions({ alertId }: { alertId: string }) {
  const ackAlert = useAcknowledgeAlert();
  const resolveAlert = useResolveAlert();

  return (
    <div className="flex gap-2">
      <button onClick={() => ackAlert.mutate(alertId)} className="text-caption text-[var(--color-accent)] hover:underline">Ack</button>
      <button onClick={() => resolveAlert.mutate(alertId)} className="text-caption text-[var(--color-success)] hover:underline">Resolve</button>
    </div>
  );
}

function CreateAlertRuleForm({ onClose }: { onClose: () => void }) {
  const createRule = useCreateAlertRule();
  const [name, setName] = useState("");
  const [metricName, setMetricName] = useState("");
  const [condition, setCondition] = useState("above");
  const [threshold, setThreshold] = useState("0");
  const [severity, setSeverity] = useState("warning");

  const handleSubmit = () => {
    if (!name || !metricName) return;
    createRule.mutate(
      { name, metric_name: metricName, condition, threshold: parseFloat(threshold), severity, enabled: true, notification_channels: [] },
      { onSuccess: onClose }
    );
  };

  return (
    <div className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg p-6">
      <h3 className="text-headline-3 text-[var(--color-text-primary)] mb-4">New Alert Rule</h3>
      <div className="grid grid-cols-2 gap-4">
        <input type="text" placeholder="Rule name" value={name} onChange={(e) => setName(e.target.value)} className="px-3 py-2 bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-lg text-body-small text-[var(--color-text-primary)]" />
        <input type="text" placeholder="Metric name" value={metricName} onChange={(e) => setMetricName(e.target.value)} className="px-3 py-2 bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-lg text-body-small text-[var(--color-text-primary)]" />
        <select value={condition} onChange={(e) => setCondition(e.target.value)} className="px-3 py-2 bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-lg text-body-small text-[var(--color-text-primary)]">
          <option value="above">Above</option>
          <option value="below">Below</option>
          <option value="equals">Equals</option>
          <option value="not_equals">Not Equals</option>
        </select>
        <input type="number" placeholder="Threshold" value={threshold} onChange={(e) => setThreshold(e.target.value)} className="px-3 py-2 bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-lg text-body-small text-[var(--color-text-primary)]" />
        <select value={severity} onChange={(e) => setSeverity(e.target.value)} className="px-3 py-2 bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-lg text-body-small text-[var(--color-text-primary)]">
          <option value="info">Info</option>
          <option value="warning">Warning</option>
          <option value="critical">Critical</option>
        </select>
      </div>
      <div className="flex gap-2 justify-end mt-4">
        <button onClick={onClose} className="px-4 py-2 text-body-small text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]">Cancel</button>
        <button onClick={handleSubmit} disabled={!name || !metricName || createRule.isPending} className="px-4 py-2 bg-[var(--color-accent)] text-[var(--color-text-on-accent)] rounded-lg text-body-small hover:opacity-90 disabled:opacity-50">Create</button>
      </div>
    </div>
  );
}

function LogsTab() {
  const [filter, setFilter] = useState("");
  const [level, setLevel] = useState("");
  const { data: logs, isLoading } = useMonitoringLogs(level || undefined, filter || undefined);

  return (
    <div className="space-y-4">
      <div className="flex gap-3">
        <input type="text" placeholder="Search logs..." value={filter} onChange={(e) => setFilter(e.target.value)} className="flex-1 px-3 py-2 bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg text-body-small text-[var(--color-text-primary)]" />
        <select value={level} onChange={(e) => setLevel(e.target.value)} className="px-3 py-2 bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg text-body-small text-[var(--color-text-primary)]">
          <option value="">All Levels</option>
          <option value="debug">Debug</option>
          <option value="info">Info</option>
          <option value="warn">Warn</option>
          <option value="error">Error</option>
          <option value="fatal">Fatal</option>
        </select>
      </div>

      {isLoading ? (
        <Loading message="Loading logs..." />
      ) : !logs || logs.length === 0 ? (
        <EmptyState message="No logs found" />
      ) : (
        <div className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg overflow-hidden font-mono text-caption">
          <div className="max-h-[600px] overflow-y-auto">
            {logs.map((log: MonitoringLog) => (
              <div key={log.id} className="flex items-start gap-3 px-4 py-2 border-b border-[var(--color-border)] last:border-0 hover:bg-[var(--color-bg-tertiary)]">
                <span className="text-[var(--color-text-tertiary)] shrink-0">{new Date(log.recorded_at).toLocaleTimeString()}</span>
                <span className="w-12 text-center shrink-0 uppercase" style={{ color: logLevelColors[log.level] || "var(--color-text-tertiary)" }}>{log.level}</span>
                <span className="text-[var(--color-text-tertiary)] shrink-0">{log.source}</span>
                <span className="text-[var(--color-text-primary)]">{log.message}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function DashboardsTab() {
  const { data: dashboards, isLoading } = useMonitoringDashboards();
  const [selected, setSelected] = useState<string | null>(null);

  if (selected) {
    return <DashboardBuilder dashboardId={selected} onBack={() => setSelected(null)} />;
  }

  if (isLoading) return <Loading message="Loading dashboards..." />;

  const list = dashboards ?? [];

  return (
    <div className="space-y-4">
      {list.length === 0 ? (
        <EmptyState message="No monitoring dashboards yet" />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {list.map((d) => (
            <button
              key={d.id}
              type="button"
              onClick={() => setSelected(d.id)}
              className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg p-5 hover:border-[var(--color-accent)] transition-colors cursor-pointer text-left"
            >
              <div className="text-body text-[var(--color-text-primary)] font-medium">{d.name}</div>
              {d.description && <p className="text-body-small text-[var(--color-text-tertiary)] mt-1">{d.description}</p>}
              <div className="text-caption text-[var(--color-text-tertiary)] mt-3">{d.widgets?.length ?? 0} widgets</div>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

function Loading({ message }: { message: string }) {
  return <div className="flex items-center justify-center py-12"><div className="text-body-small text-[var(--color-text-tertiary)]">{message}</div></div>;
}

function EmptyState({ message }: { message: string }) {
  return <div className="flex items-center justify-center py-12"><div className="text-body-small text-[var(--color-text-tertiary)]">{message}</div></div>;
}
