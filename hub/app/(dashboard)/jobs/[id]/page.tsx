"use client";

import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import { useJob } from "@/hooks/use-jobs";
import { useAuth } from "@/hooks/use-auth";
import { EmptyState } from "@/components/ui/empty-state";

const statusColors: Record<string, string> = {
  pending: "var(--color-text-tertiary)",
  running: "var(--color-accent)",
  completed: "var(--color-success)",
  failed: "var(--color-error)",
  cancelled: "var(--color-warning)",
  stopped: "var(--color-warning)",
};

export default function JobDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const queryClient = useQueryClient();

  const { data, isLoading, error } = useJob(id);
  const job = data?.job ?? null;
  const { can } = useAuth();

  const cancelMutation = useMutation({
    mutationFn: () => api.cancelJob(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["job", id] }),
  });

  const retryMutation = useMutation({
    mutationFn: () => api.retryJob(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["job", id] }),
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
          <p className="text-body text-[var(--color-error)]">Failed to load job</p>
        </div>
      </div>
    );
  }

  if (!job) {
    return (
      <EmptyState
        title="Job not found"
        description="The job you are looking for does not exist."
        action={{ label: "Back to Jobs", onClick: () => router.push("/jobs") }}
      />
    );
  }

  const total = job.total_devices;
  const done = job.completed_devices + job.failed_devices;
  const percent = total > 0 ? Math.round((done / total) * 100) : 0;

  const canCancel = job.status === "pending" || job.status === "running";
  const canRetry = job.status === "failed";

  return (
    <div className="space-y-6 max-w-3xl">
      <div>
        <Link href="/jobs" className="btn-ghost text-body-small">&larr; All Jobs</Link>
      </div>

      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-headline-1 text-[var(--color-text-primary)]">
            {job.type.replace(/_/g, " ")}
          </h1>
          <p className="text-body text-[var(--color-text-secondary)] mt-1">
            Created {new Date(job.created_at).toLocaleString()}
          </p>
        </div>
        <div className="flex gap-2">
          {can("write") && canCancel && (
            <button
              onClick={() => cancelMutation.mutate()}
              disabled={cancelMutation.isPending}
              className="btn-ghost text-body-small px-3 py-1.5 text-[var(--color-error)] border border-[var(--color-error)]"
            >
              {cancelMutation.isPending ? "Cancelling..." : "Cancel"}
            </button>
          )}
          {can("write") && canRetry && (
            <button
              onClick={() => retryMutation.mutate()}
              disabled={retryMutation.isPending}
              className="btn-primary text-body-small px-3 py-1.5"
            >
              {retryMutation.isPending ? "Retrying..." : "Retry"}
            </button>
          )}
        </div>
      </div>

      <div className="card p-4 space-y-4">
        <h2 className="text-headline-4 text-[var(--color-text-primary)]">Progress</h2>

        <div className="w-full h-3 bg-[var(--color-border)] rounded-full overflow-hidden">
          <div
            className="h-full rounded-full transition-all duration-500"
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

        <div className="grid grid-cols-3 gap-4 text-center">
          <div className="bg-[var(--color-surface)] rounded-sm p-3">
            <p className="text-headline-2 text-[var(--color-text-primary)]">{job.total_devices}</p>
            <p className="text-caption text-[var(--color-text-tertiary)]">Total devices</p>
          </div>
          <div className="bg-[var(--color-surface)] rounded-sm p-3">
            <p className="text-headline-2 text-[var(--color-success)]">{job.completed_devices}</p>
            <p className="text-caption text-[var(--color-text-tertiary)]">Succeeded</p>
          </div>
          <div className="bg-[var(--color-surface)] rounded-sm p-3">
            <p className="text-headline-2 text-[var(--color-error)]">{job.failed_devices}</p>
            <p className="text-caption text-[var(--color-text-tertiary)]">Failed</p>
          </div>
        </div>
      </div>

      <div className="card p-4 space-y-3">
        <h2 className="text-headline-4 text-[var(--color-text-primary)]">Details</h2>
        <dl className="grid grid-cols-2 gap-y-3 gap-x-4">
          {[
            ["Job ID", job.id],
            ["Type", job.type.replace(/_/g, " ")],
            ["Status",
              <span key="status" style={{ color: statusColors[job.status] ?? "var(--color-text-tertiary)" }}>
                {job.status.charAt(0).toUpperCase() + job.status.slice(1)}
              </span>,
            ],
            ["Retries", `${job.retries} / ${job.max_retries}`],
            ["Group Action ID", job.group_action_id ?? "N/A"],
            ["Max Job Duration", job.max_job_duration_ms > 0 ? `${(job.max_job_duration_ms / 1000).toFixed(0)}s` : "Default"],
            ["Max Device Exec", job.max_device_execution_ms > 0 ? `${(job.max_device_execution_ms / 1000).toFixed(0)}s` : "Default"],
            ["Stop on Failure", job.stop_on_failure ? "Yes" : "No"],
            ["Batch Size", job.batch_size > 0 ? String(job.batch_size) : "Default"],
            ["Batch Delay", job.batch_delay_ms > 0 ? `${job.batch_delay_ms}ms` : "Default"],
            ...(job.stopped_reason ? [["Stopped Reason", job.stopped_reason] as const] : []),
            ["Created", new Date(job.created_at).toLocaleString()],
            ["Started", job.started_at ? new Date(job.started_at).toLocaleString() : "Not started"],
            ["Completed", job.completed_at ? new Date(job.completed_at).toLocaleString() : "In progress"],
            ["Cancelled", job.cancelled_at ? new Date(job.cancelled_at).toLocaleString() : "N/A"],
          ].map(([label, value]) => (
            <div key={label as string}>
              <dt className="text-caption text-[var(--color-text-tertiary)]">{label as string}</dt>
              <dd className="text-body-small text-[var(--color-text-primary)] font-medium break-all">{value}</dd>
            </div>
          ))}
        </dl>
      </div>

      <div className="card p-4 space-y-3">
        <h2 className="text-headline-4 text-[var(--color-text-primary)]">Payload</h2>
        <pre className="text-body-small font-mono text-[var(--color-text-secondary)] bg-[var(--color-surface)] rounded-sm p-3 overflow-x-auto whitespace-pre-wrap">
          {JSON.stringify(job.payload, null, 2)}
        </pre>
      </div>
    </div>
  );
}
