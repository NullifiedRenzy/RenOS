"use client";

import { useCommands } from "@/hooks/use-commands";
import { formatTime } from "@/lib/format";
import type { Command } from "@/lib/types";

const STATUS_CONFIG: Record<string, { label: string; className: string }> = {
  pending: { label: "Pending", className: "badge-info" },
  running: { label: "Running", className: "badge-warning" },
  completed: { label: "Completed", className: "badge-success" },
  failed: { label: "Failed", className: "badge-error" },
  rejected: { label: "Rejected", className: "badge-error" },
};

function statusBadge(status: string) {
  const cfg = STATUS_CONFIG[status] ?? { label: status, className: "badge" };
  return <span className={cfg.className}>{cfg.label}</span>;
}

function formatOutputLine(line: string, i: number) {
  return (
    <span key={i} className="block">
      {line || "\u00A0"}
    </span>
  );
}

interface CommandRowProps {
  cmd: Command;
}

function CommandRow({ cmd }: CommandRowProps) {
  const hasOutput = cmd.output || cmd.error;
  const isFinished = cmd.status === "completed" || cmd.status === "failed" || cmd.status === "rejected";

  return (
    <div className="py-3 first:pt-0 last:pb-0 border-b border-[var(--color-border)] last:border-b-0">
      <div className="flex items-center justify-between mb-1">
        <div className="flex items-center gap-2">
          <code className="text-body-small font-mono text-[var(--color-text-primary)]">{cmd.command}</code>
          {statusBadge(cmd.status)}
        </div>
        <span className="text-caption text-[var(--color-text-tertiary)]">{formatTime(cmd.requested_at)}</span>
      </div>

      {isFinished && cmd.exit_code !== null && (
        <p className="text-caption text-[var(--color-text-tertiary)] mb-1">
          Exit code: {cmd.exit_code}
        </p>
      )}

      {hasOutput && (
        <div className="mt-1.5 rounded-sm bg-[var(--color-surface)] p-2 font-mono text-caption max-h-32 overflow-y-auto">
          {cmd.output && cmd.output.split("\n").map((line, i) => formatOutputLine(line, i))}
          {cmd.error && (
            <span className="text-[var(--color-error)]">
              {cmd.error.split("\n").map((line, i) => formatOutputLine(line, i))}
            </span>
          )}
        </div>
      )}
    </div>
  );
}

interface CommandHistoryProps {
  deviceId: string;
}

export function CommandHistory({ deviceId }: CommandHistoryProps) {
  const { data, isLoading, error } = useCommands(deviceId);
  const commands = data?.commands ?? [];

  return (
    <div className="card">
      <h2 className="text-headline-4 text-[var(--color-text-primary)] mb-3">Command History</h2>

      {isLoading && (
        <div className="space-y-3">
          <div className="skeleton h-12 w-full" />
          <div className="skeleton h-12 w-full" />
          <div className="skeleton h-12 w-full" />
        </div>
      )}

      {error && (
        <p className="text-body-small text-[var(--color-error)]">
          Failed to load command history. {(error as Error).message}
        </p>
      )}

      {!isLoading && !error && commands.length === 0 && (
        <p className="text-body text-[var(--color-text-tertiary)] py-4 text-center">
          No commands have been run on this device yet.
        </p>
      )}

      {!isLoading && !error && commands.length > 0 && (
        <div>
          {commands.map((cmd) => (
            <CommandRow key={cmd.id} cmd={cmd} />
          ))}
        </div>
      )}
    </div>
  );
}
