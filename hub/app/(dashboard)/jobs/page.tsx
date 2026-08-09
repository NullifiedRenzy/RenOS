"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { useJobs } from "@/hooks/use-jobs";
import { EmptyState } from "@/components/ui/empty-state";

const statusColors: Record<string, string> = {
  pending: "var(--color-text-tertiary)",
  running: "var(--color-accent)",
  completed: "var(--color-success)",
  failed: "var(--color-error)",
  cancelled: "var(--color-warning)",
};

const statusIcons: Record<string, string> = {
  pending: "\u23F3",
  running: "\u26A1",
  completed: "\u2705",
  failed: "\u274C",
  cancelled: "\uD83D\uDDD1\uFE0F",
};

export default function JobsPage() {
  const { data, isLoading, error } = useJobs();
  const [filter, setFilter] = useState<string>("all");

  const jobs = useMemo(() => data?.jobs ?? [], [data]);
  const filtered = useMemo(
    () => (filter === "all" ? jobs : jobs.filter((j) => j.status === filter)),
    [jobs, filter],
  );
  const counts = useMemo(() => {
    const c: Record<string, number> = { all: jobs.length, pending: 0, running: 0, completed: 0, failed: 0, cancelled: 0 };
    for (const j of jobs) {
      if (c[j.status] !== undefined) c[j.status] += 1;
    }
    return c;
  }, [jobs]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-headline-1 text-[var(--color-text-primary)]">Jobs</h1>
        <p className="text-body text-[var(--color-text-secondary)] mt-1">
          {isLoading ? "Loading..." : `${jobs.length} job${jobs.length !== 1 ? "s" : ""}`}
        </p>
      </div>

      <div className="flex gap-2 flex-wrap">
        {(["all", "pending", "running", "completed", "failed", "cancelled"] as const).map((s) => (
          <button
            key={s}
            onClick={() => setFilter(s)}
            className={`text-body-small px-2.5 py-1 rounded-sm transition-colors ${
              filter === s
                ? "bg-[var(--color-accent-tint)] text-[var(--color-accent)] font-medium"
                : "text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)]"
            }`}
          >
            {s.charAt(0).toUpperCase() + s.slice(1)} ({counts[s]})
          </button>
        ))}
      </div>

      {error && (
        <div className="card text-center py-8">
          <p className="text-body text-[var(--color-error)]">Failed to load jobs</p>
        </div>
      )}

      {!isLoading && !error && filtered.length === 0 && (
        <EmptyState
          title="No jobs found"
          description={filter === "all" ? "Run a group command to create a job." : `No ${filter} jobs.`}
        />
      )}

      {filtered.length > 0 && (
        <div className="space-y-2">
          {filtered.map((job) => {
            const total = job.total_devices;
            const done = job.completed_devices + job.failed_devices;
            const percent = total > 0 ? Math.round((done / total) * 100) : 0;

            return (
              <Link
                key={job.id}
                href={`/jobs/${job.id}`}
                className="card p-4 block hover:bg-[var(--color-surface)] transition-colors"
              >
                <div className="flex items-center gap-3">
                  <span className="text-lg">{statusIcons[job.status] ?? "\u2753"}</span>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center justify-between">
                      <p className="text-body font-medium text-[var(--color-text-primary)] truncate">
                        {job.type.replace(/_/g, " ")}
                      </p>
                      <span
                        className="text-caption font-medium"
                        style={{ color: statusColors[job.status] ?? "var(--color-text-tertiary)" }}
                      >
                        {job.status.charAt(0).toUpperCase() + job.status.slice(1)}
                      </span>
                    </div>
                    <p className="text-body-small text-[var(--color-text-tertiary)] mt-0.5">
                      {done}/{total} devices &middot; {percent}% &middot; {job.retries > 0 ? `${job.retries} retr${job.retries !== 1 ? "ies" : "y"} \u00B7 ` : ""}
                      {new Date(job.created_at).toLocaleString()}
                    </p>
                    {total > 0 && (
                      <div className="w-full h-1.5 bg-[var(--color-border)] rounded-full overflow-hidden mt-1">
                        <div
                          className="h-full rounded-full transition-all duration-300"
                          style={{
                            width: `${percent}%`,
                            backgroundColor:
                              job.status === "completed"
                                ? "var(--color-success)"
                                : job.status === "failed" || job.status === "cancelled"
                                  ? "var(--color-error)"
                                  : "var(--color-accent)",
                          }}
                        />
                      </div>
                    )}
                  </div>
                </div>
              </Link>
            );
          })}
        </div>
      )}
    </div>
  );
}
