"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import { useSchedule, useScheduleRuns } from "@/hooks/use-schedules";
import { useAuth } from "@/hooks/use-auth";
import { EmptyState } from "@/components/ui/empty-state";

const jobStatusLink: Record<string, string> = {
  running: "var(--color-accent)",
  completed: "var(--color-success)",
  failed: "var(--color-error)",
  pending: "var(--color-text-tertiary)",
};

export default function ScheduleDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const queryClient = useQueryClient();

  const { data, isLoading, error } = useSchedule(id);
  const { data: runsData } = useScheduleRuns(id);
  const { can } = useAuth();

  const [showDisableConfirm, setShowDisableConfirm] = useState(false);
  const [showEnableConfirm, setShowEnableConfirm] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  const schedule = data?.schedule ?? null;
  const runs = runsData?.runs ?? [];

  const enableMutation = useMutation({
    mutationFn: () => api.enableSchedule(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["schedule", id] });
      setShowEnableConfirm(false);
    },
  });

  const disableMutation = useMutation({
    mutationFn: () => api.disableSchedule(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["schedule", id] });
      setShowDisableConfirm(false);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => api.deleteSchedule(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["schedules"] });
      router.push("/schedules");
    },
  });

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="skeleton h-8 w-48" />
        <div className="skeleton h-48" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-6">
        <button onClick={() => router.back()} className="btn-ghost text-body-small">&larr; Back</button>
        <div className="card text-center py-8">
          <p className="text-body text-[var(--color-error)]">Failed to load schedule</p>
        </div>
      </div>
    );
  }

  if (!schedule) {
    return (
      <EmptyState
        title="Schedule not found"
        description="The schedule you are looking for does not exist."
        action={{ label: "Back to Schedules", onClick: () => router.push("/schedules") }}
      />
    );
  }

  return (
    <div className="space-y-6 max-w-3xl">
      <div>
        <Link href="/schedules" className="btn-ghost text-body-small">&larr; All Schedules</Link>
      </div>

      <div className="flex items-center justify-between">
        <div>
          <div className="flex items-center gap-3">
            <span
              className={`w-3 h-3 rounded-full`}
              style={{ backgroundColor: schedule.enabled ? "var(--color-success)" : "var(--color-text-tertiary)" }}
            />
            <h1 className="text-headline-1 text-[var(--color-text-primary)]">
              {schedule.name}
            </h1>
          </div>
          <p className="text-body text-[var(--color-text-secondary)] mt-1">
            {schedule.type.replace(/_/g, " ")} &middot; Created {new Date(schedule.created_at).toLocaleString()}
          </p>
        </div>
        <div className="flex gap-2">
          {can("write") && (
            <>
              {schedule.enabled ? (
                <>
                  {!showDisableConfirm ? (
                    <button
                      onClick={() => setShowDisableConfirm(true)}
                      className="btn-ghost text-body-small px-3 py-1.5 text-[var(--color-warning)] border border-[var(--color-warning)]"
                    >
                      Disable
                    </button>
                  ) : (
                    <div className="flex items-center gap-2">
                      <span className="text-caption text-[var(--color-warning)]">Disable?</span>
                      <button
                        onClick={() => disableMutation.mutate()}
                        disabled={disableMutation.isPending}
                        className="btn-ghost text-body-small px-2 py-1 text-[var(--color-warning)] border border-[var(--color-warning)]"
                      >
                        {disableMutation.isPending ? "..." : "Confirm"}
                      </button>
                      <button
                        onClick={() => setShowDisableConfirm(false)}
                        className="btn-ghost text-body-small px-2 py-1"
                      >
                        Cancel
                      </button>
                    </div>
                  )}
                </>
              ) : (
                <>
                  {!showEnableConfirm ? (
                    <button
                      onClick={() => setShowEnableConfirm(true)}
                      className="btn-primary text-body-small px-3 py-1.5"
                    >
                      Enable
                    </button>
                  ) : (
                    <div className="flex items-center gap-2">
                      <span className="text-caption text-[var(--color-success)]">Enable?</span>
                      <button
                        onClick={() => enableMutation.mutate()}
                        disabled={enableMutation.isPending}
                        className="btn-ghost text-body-small px-2 py-1 text-[var(--color-success)] border border-[var(--color-success)]"
                      >
                        {enableMutation.isPending ? "..." : "Confirm"}
                      </button>
                      <button
                        onClick={() => setShowEnableConfirm(false)}
                        className="btn-ghost text-body-small px-2 py-1"
                      >
                        Cancel
                      </button>
                    </div>
                  )}
                </>
              )}

              {!showDeleteConfirm ? (
                <button
                  onClick={() => setShowDeleteConfirm(true)}
                  className="btn-ghost text-body-small px-3 py-1.5 text-[var(--color-error)] border border-[var(--color-error)]"
                >
                  Delete
                </button>
              ) : (
                <div className="flex items-center gap-2">
                  <span className="text-caption text-[var(--color-error)]">Delete permanently?</span>
                  <button
                    onClick={() => deleteMutation.mutate()}
                    disabled={deleteMutation.isPending}
                    className="btn-ghost text-body-small px-2 py-1 text-[var(--color-error)] border border-[var(--color-error)]"
                  >
                    {deleteMutation.isPending ? "..." : "Confirm"}
                  </button>
                  <button
                    onClick={() => setShowDeleteConfirm(false)}
                    className="btn-ghost text-body-small px-2 py-1"
                  >
                    Cancel
                  </button>
                </div>
              )}
            </>
          )}
        </div>
      </div>

      <div className="card p-4 space-y-3">
        <h2 className="text-headline-4 text-[var(--color-text-primary)]">Details</h2>
        <dl className="grid grid-cols-2 gap-y-3 gap-x-4">
          {[
            ["Schedule ID", schedule.id],
            ["Name", schedule.name],
            ["Type", schedule.type.replace(/_/g, " ")],
            ["Target Type", schedule.target_type],
            ["Target IDs", schedule.target_ids?.join(", ") ?? "N/A"],
            ["Cron Expression", schedule.cron_expression],
            ["Status", schedule.enabled ? "Enabled" : "Disabled"],
            ["Created", new Date(schedule.created_at).toLocaleString()],
            ["Last Run", schedule.last_run_at ? new Date(schedule.last_run_at).toLocaleString() : "Never"],
            ["Next Run", new Date(schedule.next_run_at).toLocaleString()],
          ].map(([label, value]) => (
            <div key={label as string}>
              <dt className="text-caption text-[var(--color-text-tertiary)]">{label as string}</dt>
              <dd className="text-body-small text-[var(--color-text-primary)] font-medium break-all">{value as string}</dd>
            </div>
          ))}
        </dl>
      </div>

      <div className="card p-4 space-y-3">
        <h2 className="text-headline-4 text-[var(--color-text-primary)]">Payload</h2>
        <pre className="text-body-small font-mono text-[var(--color-text-secondary)] bg-[var(--color-surface)] rounded-sm p-3 overflow-x-auto whitespace-pre-wrap">
          {JSON.stringify(schedule.payload, null, 2)}
        </pre>
      </div>

      <div className="card p-4 space-y-3">
        <h2 className="text-headline-4 text-[var(--color-text-primary)]">Run History</h2>
        {runs.length === 0 ? (
          <p className="text-body-small text-[var(--color-text-tertiary)]">No runs yet.</p>
        ) : (
          <div className="space-y-2">
            {runs.map((run) => (
              <div key={run.id} className="flex items-center gap-3 bg-[var(--color-surface)] rounded-sm p-3">
                <span
                  className="text-lg"
                  style={{ color: jobStatusLink[run.status] ?? "var(--color-text-tertiary)" }}
                >
                  {run.status === "completed" ? "\u2705" : run.status === "failed" ? "\u274C" : "\u23F3"}
                </span>
                <div className="flex-1 min-w-0">
                  <p className="text-body-small text-[var(--color-text-primary)]">
                    {run.status.charAt(0).toUpperCase() + run.status.slice(1)}
                  </p>
                  <p className="text-caption text-[var(--color-text-tertiary)]">
                    {new Date(run.started_at).toLocaleString()}
                    {run.completed_at ? ` \u00B7 ${new Date(run.completed_at).toLocaleString()}` : ""}
                  </p>
                </div>
                {run.job_id && (
                  <Link
                    href={`/jobs/${run.job_id}`}
                    className="btn-ghost text-caption px-2 py-1"
                  >
                    View Job
                  </Link>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
