"use client";

import {
  Database,
  Server,
  Network,
  Brain,
  Blocks,
  HardDrive,
  Cpu,
  MemoryStick,
  ShieldCheck,
  Archive,
  Activity,
} from "lucide-react";
import { useHealthDashboard } from "@/hooks/use-health";
import { ErrorState } from "@/components/ui/error-state";
import { LoadingCards } from "@/components/ui/loading-state";
import type { HealthCheckStatus } from "@/lib/types";

const checkMeta: Record<string, { label: string; icon: React.ComponentType<{ className?: string }> }> = {
  database: { label: "Database", icon: Database },
  redis: { label: "Redis", icon: Server },
  nats: { label: "NATS", icon: Network },
  ai: { label: "AI", icon: Brain },
  plugins: { label: "Plugins", icon: Blocks },
  storage: { label: "Storage", icon: HardDrive },
  cpu: { label: "CPU", icon: Cpu },
  memory: { label: "Memory", icon: MemoryStick },
  disk: { label: "Disk", icon: HardDrive },
  certificates: { label: "Certificates", icon: ShieldCheck },
  backups: { label: "Backups", icon: Archive },
};

const CHECK_ORDER = [
  "database",
  "redis",
  "nats",
  "ai",
  "plugins",
  "storage",
  "cpu",
  "memory",
  "disk",
  "certificates",
  "backups",
];

const statusConfig: Record<HealthCheckStatus, { label: string; color: string; bg: string }> = {
  healthy: { label: "Healthy", color: "var(--color-success)", bg: "rgba(52, 199, 89, 0.12)" },
  degraded: { label: "Degraded", color: "var(--color-warning)", bg: "rgba(255, 149, 0, 0.12)" },
  warning: { label: "Warning", color: "var(--color-warning)", bg: "rgba(255, 149, 0, 0.12)" },
  critical: { label: "Critical", color: "var(--color-error)", bg: "rgba(255, 59, 48, 0.12)" },
  unhealthy: { label: "Unhealthy", color: "var(--color-error)", bg: "rgba(255, 59, 48, 0.12)" },
  not_configured: { label: "Not configured", color: "var(--color-info)", bg: "rgba(90, 200, 250, 0.12)" },
  disabled: { label: "Disabled", color: "var(--color-text-tertiary)", bg: "rgba(142, 142, 147, 0.12)" },
  unknown: { label: "Unknown", color: "var(--color-text-tertiary)", bg: "rgba(142, 142, 147, 0.12)" },
};

function scoreColor(score: number): string {
  if (score >= 85) return "var(--color-success)";
  if (score >= 60) return "var(--color-warning)";
  return "var(--color-error)";
}

function ScoreRing({ score }: { score: number }) {
  const color = scoreColor(score);
  const r = 52;
  const c = 2 * Math.PI * r;
  const offset = c * (1 - Math.min(100, Math.max(0, score)) / 100);

  return (
    <div className="relative h-40 w-40 flex-shrink-0">
      <svg viewBox="0 0 120 120" className="h-full w-full -rotate-90">
        <circle cx="60" cy="60" r={r} fill="none" stroke="var(--color-surface)" strokeWidth="10" />
        <circle
          cx="60"
          cy="60"
          r={r}
          fill="none"
          stroke={color}
          strokeWidth="10"
          strokeLinecap="round"
          strokeDasharray={c}
          strokeDashoffset={offset}
          className="transition-all duration-700 ease-out"
        />
      </svg>
      <div className="absolute inset-0 flex flex-col items-center justify-center">
        <span className="text-headline-1 font-bold" style={{ color }}>
          {score}
        </span>
        <span className="text-caption text-[var(--color-text-tertiary)]">/ 100</span>
      </div>
    </div>
  );
}

function StatusBadge({ status }: { status: HealthCheckStatus }) {
  const cfg = statusConfig[status] ?? statusConfig.unknown;
  return (
    <span
      className="inline-flex items-center gap-1.5 rounded-sm px-2 py-0.5 text-caption font-medium"
      style={{ backgroundColor: cfg.bg, color: cfg.color }}
    >
      <span className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: cfg.color }} />
      {cfg.label}
    </span>
  );
}

export default function HealthPage() {
  const { data, isLoading, error, refetch } = useHealthDashboard();

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="skeleton h-10 w-40" />
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
          <div className="skeleton h-56 w-full" />
          <div className="skeleton h-56 w-full lg:col-span-2" />
        </div>
        <div className="skeleton h-6 w-32" />
        <LoadingCards count={4} />
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-6">
        <h1 className="text-headline-1 text-[var(--color-text-primary)]">Health</h1>
        <ErrorState
          error={error as Error}
          onRetry={() => refetch()}
          title="Could not load health data"
        />
      </div>
    );
  }

  if (!data) {
    return (
      <div className="space-y-6">
        <h1 className="text-headline-1 text-[var(--color-text-primary)]">Health</h1>
        <ErrorState error={new Error("No health data available")} onRetry={() => refetch()} title="No health data" />
      </div>
    );
  }

  const checks = Object.entries(data.checks);
  const sortedChecks = [...checks].sort(([a], [b]) => {
    const ia = CHECK_ORDER.indexOf(a);
    const ib = CHECK_ORDER.indexOf(b);
    return (ia === -1 ? CHECK_ORDER.length : ia) - (ib === -1 ? CHECK_ORDER.length : ib);
  });

  const healthy = checks.filter(([, c]) => c.status === "healthy").length;
  const warning = checks.filter(([, c]) => c.status === "warning" || c.status === "degraded").length;
  const critical = checks.filter(([, c]) => c.status === "critical" || c.status === "unhealthy").length;
  const notConfigured = checks.filter(
    ([, c]) => c.status === "not_configured" || c.status === "disabled" || c.status === "unknown",
  ).length;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-headline-1 text-[var(--color-text-primary)]">Health</h1>
        <p className="text-body text-[var(--color-text-secondary)] mt-1">
          Live health of every core subsystem powering RenOS
        </p>
      </div>

      {/* Score + summary */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <div className="card flex items-center gap-6">
          <ScoreRing score={data.score} />
          <div>
            <p className="text-body-small font-medium text-[var(--color-text-primary)] mb-2">What the score means</p>
            <ul className="space-y-2">
              <li className="flex items-center gap-2 text-caption text-[var(--color-text-secondary)]">
                <span className="h-2 w-2 rounded-full" style={{ backgroundColor: "var(--color-success)" }} />
                85+ Healthy
              </li>
              <li className="flex items-center gap-2 text-caption text-[var(--color-text-secondary)]">
                <span className="h-2 w-2 rounded-full" style={{ backgroundColor: "var(--color-warning)" }} />
                60–84 Degraded
              </li>
              <li className="flex items-center gap-2 text-caption text-[var(--color-text-secondary)]">
                <span className="h-2 w-2 rounded-full" style={{ backgroundColor: "var(--color-error)" }} />
                Under 60 Critical
              </li>
            </ul>
          </div>
        </div>

        <div className="card lg:col-span-2 flex flex-col justify-center gap-4">
          <div className="flex items-center gap-3">
            <Activity className="h-5 w-5 text-[var(--color-accent)]" />
            <p className="text-body text-[var(--color-text-primary)]">
              Overall status:{" "}
              <span className="font-medium capitalize" style={{ color: scoreColor(data.score) }}>
                {data.status}
              </span>
            </p>
          </div>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            {[
              { label: "Healthy", count: healthy, color: "var(--color-success)" },
              { label: "Warning", count: warning, color: "var(--color-warning)" },
              { label: "Not configured", count: notConfigured, color: "var(--color-info)" },
              { label: "Critical", count: critical, color: "var(--color-error)" },
            ].map((s) => (
              <div key={s.label} className="rounded-sm bg-[var(--color-surface)] border border-[var(--color-border)] p-3">
                <p className="text-headline-3 font-bold" style={{ color: s.color }}>
                  {s.count}
                </p>
                <p className="text-caption text-[var(--color-text-tertiary)] mt-0.5">{s.label}</p>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Check cards */}
      <div>
        <h2 className="text-headline-4 text-[var(--color-text-primary)] mb-3">Checks</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
          {sortedChecks.map(([key, check]) => {
            const meta = checkMeta[key] ?? { label: key, icon: Activity };
            const Icon = meta.icon;
            return (
              <div key={key} className="card p-4">
                <div className="flex items-center justify-between mb-3">
                  <div className="flex items-center gap-3 min-w-0">
                    <div className="h-9 w-9 rounded-md bg-[var(--color-surface)] border border-[var(--color-border)] flex items-center justify-center flex-shrink-0">
                      <Icon className="h-[18px] w-[18px] text-[var(--color-text-secondary)]" />
                    </div>
                    <span className="text-body-small font-medium text-[var(--color-text-primary)] truncate">
                      {meta.label}
                    </span>
                  </div>
                  <StatusBadge status={check.status} />
                </div>
                <p className="text-body-small text-[var(--color-text-secondary)]">
                  {check.detail || "No details available."}
                </p>
                {typeof check.latency_ms === "number" && check.latency_ms > 0 && (
                  <p className="text-caption font-mono text-[var(--color-text-tertiary)] mt-2">
                    {check.latency_ms} ms
                  </p>
                )}
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
