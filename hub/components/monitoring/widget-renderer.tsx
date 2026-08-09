"use client";

import { Activity, Boxes, Brain, Cpu, Gauge, MemoryStick, Server, ServerCrash, Layers, Terminal } from "lucide-react";
import type {
  MonitoringWidget,
  WidgetData,
  DevicesOverviewData,
  DeviceStatusData,
  MetricData,
  AlertsWidgetData,
  LogEntryData,
  WorkflowsWidgetData,
  AIUsageData,
  ContainersWidgetData,
  ProxmoxWidgetData,
} from "@/lib/types";
import { cn } from "@/lib/utils";

const logLevelColors: Record<string, string> = {
  debug: "var(--color-text-tertiary)",
  info: "var(--color-info)",
  warn: "var(--color-warning)",
  error: "var(--color-error)",
  fatal: "var(--color-error)",
};

const statusColors: Record<string, string> = {
  online: "var(--color-success)",
  offline: "var(--color-text-tertiary)",
  error: "var(--color-error)",
};

export const WIDGET_PALETTE: { type: string; label: string; description: string; defaultW: number; defaultH: number }[] = [
  { type: "devices-overview", label: "Devices Overview", description: "Device counts by status and OS.", defaultW: 4, defaultH: 3 },
  { type: "device-status", label: "Device Status", description: "Per-device online/offline state.", defaultW: 6, defaultH: 4 },
  { type: "cpu", label: "CPU Usage", description: "Recent CPU usage samples.", defaultW: 6, defaultH: 3 },
  { type: "memory", label: "Memory Usage", description: "Recent memory usage samples.", defaultW: 6, defaultH: 3 },
  { type: "alerts", label: "Active Alerts", description: "Firing alert count and messages.", defaultW: 4, defaultH: 3 },
  { type: "logs", label: "Recent Logs", description: "Latest monitoring log entries.", defaultW: 6, defaultH: 4 },
  { type: "workflows", label: "Workflows", description: "Workflow totals and recent runs.", defaultW: 4, defaultH: 3 },
  { type: "ai-usage", label: "AI Usage", description: "Conversation totals.", defaultW: 3, defaultH: 2 },
  { type: "containers", label: "Containers", description: "Container counts by state.", defaultW: 3, defaultH: 2 },
  { type: "proxmox", label: "Proxmox", description: "Cluster resources summary.", defaultW: 4, defaultH: 3 },
];

const PALETTE_ICONS: Record<string, typeof Server> = {
  "devices-overview": Server,
  "device-status": ServerCrash,
  cpu: Cpu,
  memory: MemoryStick,
  alerts: Gauge,
  logs: Terminal,
  workflows: Activity,
  "ai-usage": Brain,
  containers: Boxes,
  proxmox: Layers,
};

function Stat({ label, value, tone }: { label: string; value: number | string; tone?: string }) {
  return (
    <div>
      <div className="text-caption text-[var(--color-text-tertiary)]">{label}</div>
      <div className="text-headline-2 mt-0.5" style={{ color: tone ?? "var(--color-text-primary)" }}>
        {value}
      </div>
    </div>
  );
}

function DevicesOverviewCard({ data }: { data: DevicesOverviewData }) {
  return (
    <div className="grid grid-cols-3 gap-3">
      <Stat label="Total" value={data.total} />
      <Stat label="Online" value={data.online} tone="var(--color-success)" />
      <Stat label="Offline" value={data.offline} tone="var(--color-text-tertiary)" />
    </div>
  );
}

function DeviceStatusCard({ data }: { data: DeviceStatusData[] }) {
  return (
    <ul className="space-y-1.5">
      {data.map((d, i) => (
        <li key={`${d.hostname}-${i}`} className="flex items-center gap-2 text-caption">
          <span
            className="w-2 h-2 rounded-full shrink-0"
            style={{ backgroundColor: statusColors[d.status] ?? "var(--color-text-tertiary)" }}
          />
          <span className="text-[var(--color-text-primary)] truncate">{d.hostname}</span>
          <span className="ml-auto text-[var(--color-text-tertiary)] capitalize shrink-0">{d.status}</span>
        </li>
      ))}
    </ul>
  );
}

function MetricCard({ data }: { data: MetricData }) {
  const samples = data.samples ?? [];
  const latest = samples.length > 0 ? samples[samples.length - 1].value : null;
  return (
    <div className="space-y-2">
      <div className="flex items-baseline gap-2">
        <span className="text-headline-2 text-[var(--color-text-primary)]">
          {latest === null ? "—" : `${Math.round(latest * 100) / 100}%`}
        </span>
        <span className="text-caption text-[var(--color-text-tertiary)]">{samples.length} samples / 1h</span>
      </div>
      {samples.length > 0 && (
        <div className="flex items-end gap-px h-12">
          {samples.slice(-30).map((s, i) => (
            <div
              key={i}
              className="flex-1 rounded-sm"
              style={{
                height: `${Math.min(100, Math.max(4, s.value))}%`,
                backgroundColor: s.value > 85 ? "var(--color-error)" : "var(--color-accent)",
                opacity: 0.85,
              }}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function AlertsCard({ data }: { data: AlertsWidgetData }) {
  return (
    <div className="space-y-2">
      <Stat label="Firing" value={data.firing} tone={data.firing > 0 ? "var(--color-error)" : "var(--color-success)"} />
      {(data.active ?? []).slice(0, 4).map((msg, i) => (
        <p key={i} className="text-caption text-[var(--color-text-secondary)] truncate">
          {msg}
        </p>
      ))}
    </div>
  );
}

function LogsCard({ data }: { data: LogEntryData[] }) {
  return (
    <ul className="space-y-1 font-mono text-caption">
      {(data ?? []).slice(0, 8).map((l, i) => (
        <li key={i} className="flex items-start gap-2 truncate">
          <span className="text-[var(--color-text-tertiary)] shrink-0">{new Date(l.recorded_at).toLocaleTimeString()}</span>
          <span className="w-10 shrink-0 uppercase" style={{ color: logLevelColors[l.level] ?? "var(--color-text-tertiary)" }}>
            {l.level}
          </span>
          <span className="text-[var(--color-text-primary)] truncate">{l.message}</span>
        </li>
      ))}
    </ul>
  );
}

function WorkflowsCard({ data }: { data: WorkflowsWidgetData }) {
  return (
    <div className="space-y-2">
      <div className="grid grid-cols-2 gap-3">
        <Stat label="Workflows" value={data.total} />
        <Stat label="Enabled" value={data.enabled} tone="var(--color-success)" />
      </div>
      {(data.recent_runs ?? []).slice(0, 3).map((r, i) => (
        <div key={i} className="flex items-center justify-between text-caption">
          <span className="text-[var(--color-text-tertiary)] truncate">{r.id.slice(0, 8)}</span>
          <span className="capitalize" style={{ color: r.status === "completed" ? "var(--color-success)" : r.status === "failed" ? "var(--color-error)" : "var(--color-warning)" }}>
            {r.status}
          </span>
        </div>
      ))}
    </div>
  );
}

function CountCard({ value, label }: { value: number; label: string }) {
  return <Stat label={label} value={value} />;
}

export function WidgetRenderer({ widget, data }: { widget: MonitoringWidget; data?: WidgetData }) {
  switch (widget.widget_type) {
    case "devices-overview":
      return <DevicesOverviewCard data={data as DevicesOverviewData} />;
    case "device-status":
      return <DeviceStatusCard data={(data as DeviceStatusData[]) ?? []} />;
    case "cpu":
    case "memory":
      return <MetricCard data={(data as MetricData) ?? { metric: widget.widget_type, samples: [] }} />;
    case "alerts":
      return <AlertsCard data={(data as AlertsWidgetData) ?? { firing: 0, active: [] }} />;
    case "logs":
      return <LogsCard data={(data as LogEntryData[]) ?? []} />;
    case "workflows":
      return <WorkflowsCard data={(data as WorkflowsWidgetData) ?? { total: 0, enabled: 0, recent_runs: [] }} />;
    case "ai-usage":
      return <CountCard value={(data as AIUsageData)?.conversations ?? 0} label="Conversations" />;
    case "containers":
      return <ContainersCard data={(data as ContainersWidgetData) ?? { total: 0, running: 0, stopped: 0 }} />;
    case "proxmox":
      return <ProxmoxCard data={(data as ProxmoxWidgetData) ?? { connections: 0, nodes: 0, vms: 0, lxcs: 0 }} />;
    default:
      return <p className="text-caption text-[var(--color-text-tertiary)]">Unsupported widget type: {widget.widget_type}</p>;
  }
}

function ContainersCard({ data }: { data: ContainersWidgetData }) {
  return (
    <div className="grid grid-cols-3 gap-3">
      <Stat label="Total" value={data.total} />
      <Stat label="Running" value={data.running} tone="var(--color-success)" />
      <Stat label="Stopped" value={data.stopped} tone="var(--color-text-tertiary)" />
    </div>
  );
}

function ProxmoxCard({ data }: { data: ProxmoxWidgetData }) {
  return (
    <div className="grid grid-cols-2 gap-3">
      <Stat label="Connections" value={data.connections} />
      <Stat label="Nodes" value={data.nodes} />
      <Stat label="VMs" value={data.vms} />
      <Stat label="LXCs" value={data.lxcs} />
    </div>
  );
}

export function PaletteIcon({ type, className }: { type: string; className?: string }) {
  const Icon = PALETTE_ICONS[type] ?? Gauge;
  return <Icon className={cn("w-4 h-4", className)} aria-hidden="true" />;
}
