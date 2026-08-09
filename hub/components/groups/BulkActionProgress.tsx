"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import Link from "next/link";

interface BulkActionProgressProps {
  jobId: string;
}

const statusColors: Record<string, string> = {
  pending: "var(--color-text-tertiary)",
  running: "var(--color-accent)",
  completed: "var(--color-success)",
  failed: "var(--color-error)",
  cancelled: "var(--color-warning)",
};

export function BulkActionProgress({ jobId }: BulkActionProgressProps) {
  const { data, isLoading } = useQuery({
    queryKey: ["job", jobId],
    queryFn: () => api.getJob(jobId),
    refetchInterval: 2_000,
  });

  const job = data?.job;

  if (isLoading || !job) {
    return (
      <div className="bg-[var(--color-surface)] rounded-sm p-3">
        <div className="skeleton h-4 w-32 mb-2" />
        <div className="skeleton h-2 w-full" />
      </div>
    );
  }

  const total = job.total_devices;
  const done = job.completed_devices + job.failed_devices;
  const succeeded = job.completed_devices;
  const failed = job.failed_devices;
  const percent = total > 0 ? Math.round((done / total) * 100) : 0;
  const isDone = job.status === "completed" || job.status === "failed" || job.status === "cancelled";

  return (
    <Link href={`/jobs/${jobId}`} className="bg-[var(--color-surface)] rounded-sm p-3 space-y-2 block hover:opacity-80 transition-opacity">
      <div className="flex items-center justify-between">
        <p className="text-body-small text-[var(--color-text-primary)] font-medium">
          {job.type.replace(/_/g, " ")}
        </p>
        <span
          className="text-caption font-medium"
          style={{ color: statusColors[job.status] ?? "var(--color-text-tertiary)" }}
        >
          {job.status.charAt(0).toUpperCase() + job.status.slice(1)}
        </span>
      </div>

      <div className="w-full h-2 bg-[var(--color-border)] rounded-full overflow-hidden">
        <div
          className="h-full rounded-full transition-all duration-500"
          style={{
            width: `${percent}%`,
            backgroundColor: isDone
              ? failed > 0
                ? "var(--color-warning)"
                : "var(--color-success)"
              : "var(--color-accent)",
          }}
        />
      </div>

      <div className="flex gap-3 text-caption text-[var(--color-text-tertiary)]">
        <span>{succeeded} succeeded</span>
        {failed > 0 && <span className="text-[var(--color-error)]">{failed} failed</span>}
        {!isDone && <span>{total - done} remaining</span>}
        {isDone && <span className="text-[var(--color-success)]">Done</span>}
      </div>
    </Link>
  );
}
