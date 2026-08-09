"use client";

import { useState } from "react";
import Link from "next/link";
import { useSchedules } from "@/hooks/use-schedules";
import { useAuth } from "@/hooks/use-auth";
import { EmptyState } from "@/components/ui/empty-state";

const statusColors: Record<string, string> = {
  true: "var(--color-success)",
  false: "var(--color-text-tertiary)",
};

export default function SchedulesPage() {
  const { data, isLoading, error } = useSchedules();
  const { can } = useAuth();
  const [showEnabled, setShowEnabled] = useState<string>("all");

  const schedules = data?.schedules ?? [];
  const filtered = showEnabled === "all"
    ? schedules
    : schedules.filter((s) => String(s.enabled) === showEnabled);

  const counts = {
    all: schedules.length,
    enabled: schedules.filter((s) => s.enabled).length,
    disabled: schedules.filter((s) => !s.enabled).length,
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-headline-1 text-[var(--color-text-primary)]">Schedules</h1>
          <p className="text-body text-[var(--color-text-secondary)] mt-1">
            {isLoading ? "Loading..." : `${schedules.length} schedule${schedules.length !== 1 ? "s" : ""}`}
          </p>
        </div>
        {can("write") && (
          <Link href="/schedules/new" className="btn-primary text-body-small px-3 py-1.5">
            + New Schedule
          </Link>
        )}
      </div>

      <div className="flex gap-2 flex-wrap">
        {(["all", "enabled", "disabled"] as const).map((s) => (
          <button
            key={s}
            onClick={() => setShowEnabled(s)}
            className={`text-body-small px-2.5 py-1 rounded-sm transition-colors ${
              showEnabled === s
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
          <p className="text-body text-[var(--color-error)]">Failed to load schedules</p>
        </div>
      )}

      {!isLoading && !error && filtered.length === 0 && (
        <EmptyState
          title="No schedules found"
          description="Create a schedule to automate recurring actions."
          action={{ label: "Create Schedule", onClick: () => window.location.href = "/schedules/new" }}
        />
      )}

      {filtered.length > 0 && (
        <div className="space-y-2">
          {filtered.map((schedule) => (
            <Link
              key={schedule.id}
              href={`/schedules/${schedule.id}`}
              className="card p-4 block hover:bg-[var(--color-surface)] transition-colors"
            >
              <div className="flex items-center gap-3">
                <span className={`w-2 h-2 rounded-full flex-shrink-0`}
                  style={{ backgroundColor: schedule.enabled ? "var(--color-success)" : "var(--color-text-tertiary)" }}
                />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center justify-between">
                    <p className="text-body font-medium text-[var(--color-text-primary)] truncate">
                      {schedule.name}
                    </p>
                    <span className="text-caption font-medium text-[var(--color-text-tertiary)]">
                      {schedule.type.replace(/_/g, " ")}
                    </span>
                  </div>
                  <p className="text-body-small text-[var(--color-text-tertiary)] mt-0.5">
                    {schedule.cron_expression} &middot; {schedule.target_type} &middot; {schedule.target_ids?.length ?? 0} target{schedule.target_ids?.length !== 1 ? "s" : ""}
                  </p>
                  <p className="text-caption text-[var(--color-text-tertiary)] mt-0.5">
                    Next: {new Date(schedule.next_run_at).toLocaleString()}
                    {schedule.last_run_at ? ` \u00B7 Last: ${new Date(schedule.last_run_at).toLocaleString()}` : ""}
                  </p>
                </div>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
