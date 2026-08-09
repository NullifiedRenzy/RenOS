"use client";

import { useMetrics, useJobStats } from "@/hooks/use-observability";

const levelColors: Record<string, string> = {
  total_jobs: "var(--color-accent)",
  running_jobs: "var(--color-accent)",
  failed_jobs: "var(--color-error)",
  completed_jobs: "var(--color-success)",
  stopped_jobs: "var(--color-warning)",
};

const statCards = [
  { key: "total_jobs", label: "Total Jobs", fmt: (v: number) => v.toLocaleString() },
  { key: "running_jobs", label: "Running", fmt: (v: number) => v.toLocaleString() },
  { key: "failed_jobs", label: "Failed", fmt: (v: number) => v.toLocaleString() },
  { key: "completed_jobs", label: "Completed", fmt: (v: number) => v.toLocaleString() },
  { key: "stopped_jobs", label: "Stopped", fmt: (v: number) => v.toLocaleString() },
  { key: "policy_blocks", label: "Policy Blocks", fmt: (v: number) => v.toLocaleString() },
  { key: "schedule_executions", label: "Schedule Execs", fmt: (v: number) => v.toLocaleString() },
];

export default function ObservabilityPage() {
  const { data: metrics, isLoading: mLoading, error: mError } = useMetrics();
  const { data: jobStats, isLoading: jsLoading, error: jsError } = useJobStats();

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-headline-1 text-[var(--color-text-primary)]">Observability</h1>
        <p className="text-body text-[var(--color-text-secondary)] mt-1">
          System metrics, job insights, and performance overview
        </p>
      </div>

      {/* Metrics cards */}
      <div>
        <h2 className="text-headline-4 text-[var(--color-text-primary)] mb-3">Metrics</h2>
        {mLoading && <div className="skeleton h-24 w-full" />}
        {mError && (
          <div className="card text-center py-4">
            <p className="text-body text-[var(--color-error)]">Failed to load metrics</p>
          </div>
        )}
        {metrics && (
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3">
            {statCards.map((card) => {
              const value = (metrics as unknown as Record<string, number>)[card.key] ?? 0;
              return (
                <div key={card.key} className="card p-4 text-center">
                  <p className="text-headline-2 font-bold" style={{ color: levelColors[card.key] ?? "var(--color-text-primary)" }}>
                    {card.fmt(value)}
                  </p>
                  <p className="text-caption text-[var(--color-text-tertiary)] mt-1">{card.label}</p>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Job stats */}
      <div>
        <h2 className="text-headline-4 text-[var(--color-text-primary)] mb-3">Job Insights</h2>
        {jsLoading && <div className="skeleton h-24 w-full" />}
        {jsError && (
          <div className="card text-center py-4">
            <p className="text-body text-[var(--color-error)]">Failed to load job stats</p>
          </div>
        )}
        {jobStats && (
          <div className="card p-4 space-y-3">
            <dl className="grid grid-cols-2 md:grid-cols-3 gap-y-3 gap-x-4">
              {[
                ["Total Jobs", jobStats.total_jobs],
                ["Running", jobStats.running_jobs],
                ["Failed", jobStats.failed_jobs],
                ["Completed", jobStats.completed_jobs],
                ["Stopped", jobStats.stopped_jobs],
                ["Cancelled", jobStats.cancelled_jobs],
                ["Avg Duration", jobStats.avg_job_duration_ms > 0 ? `${(jobStats.avg_job_duration_ms / 1000).toFixed(1)}s` : "N/A"],
              ].map(([label, value]) => (
                <div key={label as string}>
                  <dt className="text-caption text-[var(--color-text-tertiary)]">{label as string}</dt>
                  <dd className="text-body-small text-[var(--color-text-primary)] font-medium">{String(value)}</dd>
                </div>
              ))}
            </dl>
          </div>
        )}
      </div>
    </div>
  );
}
