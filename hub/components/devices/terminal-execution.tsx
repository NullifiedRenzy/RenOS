"use client";

import { useState, useRef, useEffect } from "react";
import { useCreateExecution, useExecutions, useExecutionOutput, useCancelExecution } from "@/hooks/use-executions";
import { formatTime } from "@/lib/format";
import type { Execution, ExecutionOutput } from "@/lib/types";

const STATUS_CONFIG: Record<string, { label: string; className: string }> = {
  pending: { label: "Pending", className: "badge-info" },
  running: { label: "Running", className: "badge-warning" },
  completed: { label: "Completed", className: "badge-success" },
  failed: { label: "Failed", className: "badge-error" },
  cancelled: { label: "Cancelled", className: "badge-error" },
};

function statusBadge(status: string) {
  const cfg = STATUS_CONFIG[status] ?? { label: status, className: "badge" };
  return <span className={cfg.className}>{cfg.label}</span>;
}

interface TerminalOutputProps {
  deviceId: string;
  executionId: string;
}

function TerminalOutput({ deviceId, executionId }: TerminalOutputProps) {
  const { data } = useExecutionOutput(deviceId, executionId);
  const output = data?.output ?? [];
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (containerRef.current) {
      containerRef.current.scrollTop = containerRef.current.scrollHeight;
    }
  }, [output.length]);

  if (output.length === 0) {
    return (
      <div className="font-mono text-caption text-[var(--color-text-tertiary)] py-2">
        Waiting for output...
      </div>
    );
  }

  return (
    <div
      ref={containerRef}
      className="bg-[#0d1117] text-[#c9d1d9] font-mono text-caption p-3 rounded-sm max-h-64 overflow-y-auto"
    >
      {output.map((chunk: ExecutionOutput) => (
        <span
          key={chunk.id}
          className={chunk.stream === "stderr" ? "text-[#f85149]" : ""}
        >
          {chunk.content}
        </span>
      ))}
    </div>
  );
}

interface ExecutionRowProps {
  exec: Execution;
  deviceId: string;
  isSelected: boolean;
  onSelect: () => void;
  onCancel: () => void;
}

function ExecutionRow({ exec, deviceId, isSelected, onSelect, onCancel }: ExecutionRowProps) {
  const isRunning = exec.status === "pending" || exec.status === "running";

  return (
    <div
      className={`py-3 first:pt-0 last:pb-0 border-b border-[var(--color-border)] last:border-b-0 cursor-pointer hover:bg-[var(--color-surface)] transition-colors ${isSelected ? "bg-[var(--color-surface)]" : ""}`}
      onClick={onSelect}
    >
      <div className="flex items-center justify-between mb-1">
        <div className="flex items-center gap-2">
          <code className="text-body-small font-mono text-[var(--color-text-primary)]">{exec.command}</code>
          {statusBadge(exec.status)}
        </div>
        <div className="flex items-center gap-2">
          {isRunning && (
            <button
              className="btn-ghost text-caption text-[var(--color-error)]"
              onClick={(e) => {
                e.stopPropagation();
                onCancel();
              }}
            >
              Cancel
            </button>
          )}
          <span className="text-caption text-[var(--color-text-tertiary)]">{formatTime(exec.requested_at)}</span>
        </div>
      </div>

      {exec.exit_code !== null && (exec.status === "completed" || exec.status === "failed") && (
        <p className="text-caption text-[var(--color-text-tertiary)]">
          Exit code: {exec.exit_code}
        </p>
      )}

      {isSelected && (
        <div className="mt-2">
          <TerminalOutput deviceId={deviceId} executionId={exec.id} />
        </div>
      )}
    </div>
  );
}

interface TerminalExecutionProps {
  deviceId: string;
}

export function TerminalExecution({ deviceId }: TerminalExecutionProps) {
  const [command, setCommand] = useState("");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const createExec = useCreateExecution(deviceId);
  const cancelExec = useCancelExecution(deviceId);
  const { data: execsData } = useExecutions(deviceId, 20);
  const executions = execsData?.executions ?? [];

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!command.trim()) return;
    createExec.mutate(
      { command: command.trim() },
      {
        onSuccess: (data) => {
          setSelectedId(data.execution.id);
          setCommand("");
        },
      }
    );
  };

  return (
    <div className="card">
      <h2 className="text-headline-4 text-[var(--color-text-primary)] mb-3">Remote Execution</h2>
      <p className="text-body-small text-[var(--color-text-tertiary)] mb-4">
        Execute commands in real-time via NATS. Output streams live.
      </p>

      <form onSubmit={handleSubmit} className="flex gap-2 mb-4">
        <input
          type="text"
          value={command}
          onChange={(e) => setCommand(e.target.value)}
          placeholder="Enter command..."
          className="flex-1 font-mono text-body-small bg-[var(--color-surface)] border border-[var(--color-border)] rounded-sm px-3 py-2 text-[var(--color-text-primary)] placeholder:text-[var(--color-text-tertiary)] focus:outline-none focus:border-[var(--color-accent)]"
          disabled={createExec.isPending}
        />
        <button
          type="submit"
          className="btn-primary text-body-small"
          disabled={!command.trim() || createExec.isPending}
        >
          {createExec.isPending ? "Running..." : "Execute"}
        </button>
      </form>

      {createExec.isError && (
        <p className="text-body-small text-[var(--color-error)] mb-3">
          {(createExec.error as Error).message}
        </p>
      )}

      {executions.length === 0 ? (
        <p className="text-body text-[var(--color-text-tertiary)] py-4 text-center">
          No executions yet. Type a command above to get started.
        </p>
      ) : (
        <div>
          {executions.map((exec) => (
            <ExecutionRow
              key={exec.id}
              exec={exec}
              deviceId={deviceId}
              isSelected={selectedId === exec.id}
              onSelect={() => setSelectedId(selectedId === exec.id ? null : exec.id)}
              onCancel={() => cancelExec.mutate(exec.id)}
            />
          ))}
        </div>
      )}
    </div>
  );
}
