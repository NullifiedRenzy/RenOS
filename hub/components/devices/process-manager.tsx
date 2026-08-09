"use client";

import { useCallback, useMemo, useRef, useState, memo } from "react";
import { useVirtualizer } from "@tanstack/react-virtual";
import {
  useProcesses,
  useSearchProcesses,
  useProcessHistory,
  useProcessAggregates,
  useCreateProcessAction,
  useProcessActions,
} from "@/hooks/use-processes";
import { useDialogEscape } from "@/hooks/use-dialog";
import { formatBytes } from "@/lib/format";
import type { ProcessInfo, ProcessHistoryEntry } from "@/lib/types";

const ACTION_CONFIG: Record<string, { label: string; className: string }> = {
  kill: { label: "Kill", className: "badge-error" },
  terminate: { label: "Terminate", className: "badge-warning" },
  suspend: { label: "Suspend", className: "badge-info" },
  resume: { label: "Resume", className: "badge-success" },
  restart: { label: "Restart", className: "badge-info" },
};

function formatUptime(seconds: number): string {
  if (seconds <= 0) return "\u2014";
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}

function MiniGraph({ data, color }: { data: number[]; color: string }) {
  if (data.length < 2) return <div className="h-8" />;
  const max = Math.max(...data, 1);
  const points = data.map((v, i) => {
    const x = (i / (data.length - 1)) * 100;
    const y = 100 - (v / max) * 100;
    return `${x},${y}`;
  }).join(" ");
  return (
    <svg viewBox="0 0 100 100" preserveAspectRatio="none" className="h-8 w-full">
      <polyline
        fill="none"
        stroke={color}
        strokeWidth="2"
        vectorEffect="non-scaling-stroke"
        points={points}
      />
    </svg>
  );
}

interface ProcessRowProps {
  proc: ProcessInfo;
  selected: boolean;
  onSelect: (pid: number) => void;
  onAction: (pid: number, name: string, action: string) => void;
}

const ProcessRow = memo(function ProcessRow({ proc, selected, onSelect, onAction }: ProcessRowProps) {
  return (
    <tr
      className={`h-9 border-b border-[var(--color-border)] cursor-pointer transition-colors ${
        selected ? "bg-[var(--color-accent-bg)]" : "hover:bg-[var(--color-surface-hover)]"
      }`}
      onClick={() => onSelect(proc.pid)}
    >
      <td className="px-3 py-2 text-body-small font-mono text-[var(--color-text-primary)]">{proc.pid}</td>
      <td className="px-3 py-2 text-body-small text-[var(--color-text-primary)] max-w-[200px] truncate">{proc.name}</td>
      <td className="px-3 py-2 text-body-small text-[var(--color-text-primary)]">{proc.cpu_usage.toFixed(1)}%</td>
      <td className="px-3 py-2 text-body-small text-[var(--color-text-primary)]">{proc.memory_usage.toFixed(1)}%</td>
      <td className="px-3 py-2 text-body-small text-[var(--color-text-primary)]">{proc.threads}</td>
      <td className="px-3 py-2 text-body-small text-[var(--color-text-secondary)]">{proc.owner}</td>
      <td className="px-3 py-2 text-body-small text-[var(--color-text-secondary)]">{proc.status}</td>
      <td className="px-3 py-2">
        <div className="flex gap-1" onClick={(e) => e.stopPropagation()}>
          <button
            className="btn-ghost text-caption px-2 py-1"
            onClick={() => onAction(proc.pid, proc.name, "kill")}
          >
            Kill
          </button>
          <button
            className="btn-ghost text-caption px-2 py-1"
            onClick={() => onAction(proc.pid, proc.name, "terminate")}
          >
            Stop
          </button>
        </div>
      </td>
    </tr>
  );
});

interface ProcessManagerProps {
  deviceId: string;
}

export function ProcessManager({ deviceId }: ProcessManagerProps) {
  const [searchQuery, setSearchQuery] = useState("");
  const [sortField, setSortField] = useState<keyof ProcessInfo>("cpu_usage");
  const [sortDir, setSortDir] = useState<"asc" | "desc">("desc");
  const [selectedPid, setSelectedPid] = useState<number | null>(null);
  const [confirmAction, setConfirmAction] = useState<{ pid: number; name: string; action: string } | null>(null);
  const dialogRef = useDialogEscape(() => setConfirmAction(null), !!confirmAction);
  const tableRef = useRef<HTMLDivElement>(null);

  const { data: processData, isLoading } = useProcesses(deviceId);
  const { data: searchData } = useSearchProcesses(deviceId, searchQuery);
  const { data: historyData } = useProcessHistory(deviceId, 30);
  const { data: aggregates } = useProcessAggregates(deviceId);
  const { data: actionsData } = useProcessActions(deviceId);
  const createAction = useCreateProcessAction(deviceId);

  const processes = useMemo(
    () =>
      searchQuery.length > 0 ? (searchData?.processes ?? []) : (processData?.processes ?? []),
    [searchQuery, searchData, processData],
  );
  const history = historyData?.history ?? [];
  const actions = actionsData?.process_actions ?? [];

  const sortedProcesses = useMemo(() => {
    const list = [...processes];
    list.sort((a, b) => {
      const aVal = a[sortField];
      const bVal = b[sortField];
      if (typeof aVal === "number" && typeof bVal === "number") {
        return sortDir === "asc" ? aVal - bVal : bVal - aVal;
      }
      const aStr = String(aVal);
      const bStr = String(bVal);
      return sortDir === "asc" ? aStr.localeCompare(bStr) : bStr.localeCompare(aStr);
    });
    return list;
  }, [processes, sortField, sortDir]);

  const cpuHistory = history.map((h) => h.total_cpu);
  const memHistory = history.map((h) => h.total_memory_usage);

  const virtualizer = useVirtualizer({
    count: sortedProcesses.length,
    getScrollElement: () => tableRef.current,
    estimateSize: () => 37,
    overscan: 15,
  });

  const handleSelect = useCallback((pid: number) => {
    setSelectedPid((cur) => (cur === pid ? null : pid));
  }, []);

  const handleAction = useCallback((pid: number, name: string, action: string) => {
    setConfirmAction({ pid, name, action });
  }, []);

  function handleSort(field: keyof ProcessInfo) {
    if (sortField === field) {
      setSortDir(sortDir === "asc" ? "desc" : "asc");
    } else {
      setSortField(field);
      setSortDir("desc");
    }
  }

  function confirmActionExec() {
    if (!confirmAction) return;
    createAction.mutate({ pid: confirmAction.pid, action_type: confirmAction.action });
    setConfirmAction(null);
  }

  function SortHeader({ field, children }: { field: keyof ProcessInfo; children: React.ReactNode }) {
    return (
      <th
        scope="col"
        aria-sort={sortField === field ? (sortDir === "asc" ? "ascending" : "descending") : "none"}
        className="px-3 py-2 text-left text-caption font-medium text-[var(--color-text-tertiary)] cursor-pointer hover:text-[var(--color-text-primary)] select-none"
        onClick={() => handleSort(field)}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === " ") {
            e.preventDefault();
            handleSort(field);
          }
        }}
        tabIndex={0}
      >
        {children}
        {sortField === field && (
          <span className="ml-1" aria-hidden="true">{sortDir === "asc" ? "\u2191" : "\u2193"}</span>
        )}
      </th>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-headline-4 text-[var(--color-text-primary)]">Processes</h2>
        <span className="text-body-small text-[var(--color-text-tertiary)]">
          {processData?.process_count ?? 0} running
        </span>
      </div>

      <div className="grid grid-cols-3 gap-4">
        <div className="card p-3">
          <div className="text-caption text-[var(--color-text-tertiary)]">CPU</div>
          <div className="text-headline-3 text-[var(--color-text-primary)]">
            {(processData?.total_cpu ?? 0).toFixed(1)}%
          </div>
          <MiniGraph data={cpuHistory} color="var(--color-accent)" />
        </div>
        <div className="card p-3">
          <div className="text-caption text-[var(--color-text-tertiary)]">Memory</div>
          <div className="text-headline-3 text-[var(--color-text-primary)]">
            {(processData?.total_memory_usage ?? 0).toFixed(1)}%
          </div>
          <div className="text-body-small text-[var(--color-text-secondary)]">
            {formatBytes(processData?.total_memory_bytes ?? 0)}
          </div>
          <MiniGraph data={memHistory} color="var(--color-success)" />
        </div>
        <div className="card p-3">
          <div className="text-caption text-[var(--color-text-tertiary)]">Processes</div>
          <div className="text-headline-3 text-[var(--color-text-primary)]">
            {processData?.process_count ?? 0}
          </div>
        </div>
      </div>

      <div className="card">
        <div className="p-3 border-b border-[var(--color-border)]">
          <input
            type="text"
            placeholder="Search processes..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="input w-full"
          />
        </div>
        <div ref={tableRef} className="overflow-auto max-h-[480px]">
          <table className="w-full">
            <thead className="sticky top-0 z-10 bg-[var(--color-surface)]">
              <tr className="border-b border-[var(--color-border)]">
                <SortHeader field="pid">PID</SortHeader>
                <SortHeader field="name">Name</SortHeader>
                <SortHeader field="cpu_usage">CPU</SortHeader>
                <SortHeader field="memory_usage">Memory</SortHeader>
                <SortHeader field="threads">Threads</SortHeader>
                <SortHeader field="owner">Owner</SortHeader>
                <SortHeader field="status">Status</SortHeader>
                <th scope="col" className="px-3 py-2 text-left text-caption font-medium text-[var(--color-text-tertiary)]">Actions</th>
              </tr>
            </thead>
            <tbody>
              {isLoading ? (
                <tr><td colSpan={8} className="p-4 text-center text-body-small text-[var(--color-text-tertiary)]">Loading...</td></tr>
              ) : sortedProcesses.length === 0 ? (
                <tr><td colSpan={8} className="p-4 text-center text-body-small text-[var(--color-text-tertiary)]">No processes found</td></tr>
              ) : (
                (() => {
                  const items = virtualizer.getVirtualItems();
                  return (
                    <>
                      {items.length > 0 && (
                        <tr aria-hidden="true">
                          <td colSpan={8} style={{ height: items[0].start }} />
                        </tr>
                      )}
                      {items.map((virtualRow) => {
                        const proc = sortedProcesses[virtualRow.index];
                        return (
                          <ProcessRow
                            key={proc.pid}
                            proc={proc}
                            selected={selectedPid === proc.pid}
                            onSelect={handleSelect}
                            onAction={handleAction}
                          />
                        );
                      })}
                      {items.length > 0 && (
                        <tr aria-hidden="true">
                          <td colSpan={8} style={{ height: virtualizer.getTotalSize() - items[items.length - 1].end }} />
                        </tr>
                      )}
                    </>
                  );
                })()
              )}
            </tbody>
          </table>
        </div>
      </div>

      {selectedPid !== null && (() => {
        const proc = processes.find((p) => p.pid === selectedPid);
        if (!proc) return null;
        return (
          <div className="card">
            <h3 className="text-headline-4 text-[var(--color-text-primary)] p-3 border-b border-[var(--color-border)]">
              Process Details: {proc.name} (PID {proc.pid})
            </h3>
            <div className="p-3 grid grid-cols-2 gap-3">
              <div>
                <dt className="text-caption text-[var(--color-text-tertiary)]">Executable</dt>
                <dd className="text-body-small text-[var(--color-text-primary)] font-mono break-all">{proc.executable}</dd>
              </div>
              <div>
                <dt className="text-caption text-[var(--color-text-tertiary)]">Command</dt>
                <dd className="text-body-small text-[var(--color-text-primary)] font-mono break-all">{proc.command_line || "\u2014"}</dd>
              </div>
              <div>
                <dt className="text-caption text-[var(--color-text-tertiary)]">Parent PID</dt>
                <dd className="text-body-small text-[var(--color-text-primary)]">{proc.parent_pid}</dd>
              </div>
              <div>
                <dt className="text-caption text-[var(--color-text-tertiary)]">Owner</dt>
                <dd className="text-body-small text-[var(--color-text-primary)]">{proc.owner || "\u2014"}</dd>
              </div>
              <div>
                <dt className="text-caption text-[var(--color-text-tertiary)]">CPU</dt>
                <dd className="text-body-small text-[var(--color-text-primary)]">{proc.cpu_usage.toFixed(1)}%</dd>
              </div>
              <div>
                <dt className="text-caption text-[var(--color-text-tertiary)]">Memory</dt>
                <dd className="text-body-small text-[var(--color-text-primary)]">{formatBytes(proc.memory_bytes)} ({proc.memory_usage.toFixed(1)}%)</dd>
              </div>
              <div>
                <dt className="text-caption text-[var(--color-text-tertiary)]">Threads</dt>
                <dd className="text-body-small text-[var(--color-text-primary)]">{proc.threads}</dd>
              </div>
              <div>
                <dt className="text-caption text-[var(--color-text-tertiary)]">Status</dt>
                <dd className="text-body-small text-[var(--color-text-primary)]">{proc.status}</dd>
              </div>
            </div>
            <div className="p-3 border-t border-[var(--color-border)] flex gap-2">
              {["kill", "terminate", "suspend", "resume", "restart"].map((action) => (
                <button
                  key={action}
                  className={ACTION_CONFIG[action]?.className ?? "badge"}
                  onClick={() => handleAction(proc.pid, proc.name, action)}
                >
                  {ACTION_CONFIG[action]?.label ?? action}
                </button>
              ))}
            </div>
          </div>
        );
      })()}

      {actions.length > 0 && (
        <div className="card">
          <h3 className="text-headline-4 text-[var(--color-text-primary)] p-3 border-b border-[var(--color-border)]">
            Recent Actions
          </h3>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-[var(--color-border)]">
                  <th scope="col" className="px-3 py-2 text-left text-caption font-medium text-[var(--color-text-tertiary)]">Time</th>
                  <th scope="col" className="px-3 py-2 text-left text-caption font-medium text-[var(--color-text-tertiary)]">Action</th>
                  <th scope="col" className="px-3 py-2 text-left text-caption font-medium text-[var(--color-text-tertiary)]">PID</th>
                  <th scope="col" className="px-3 py-2 text-left text-caption font-medium text-[var(--color-text-tertiary)]">Status</th>
                  <th scope="col" className="px-3 py-2 text-left text-caption font-medium text-[var(--color-text-tertiary)]">Output</th>
                </tr>
              </thead>
              <tbody>
                {actions.slice(0, 10).map((action) => (
                  <tr key={action.id} className="border-b border-[var(--color-border)]">
                    <td className="px-3 py-2 text-body-small text-[var(--color-text-secondary)]">
                      {new Date(action.requested_at).toLocaleTimeString()}
                    </td>
                    <td className="px-3 py-2 text-body-small text-[var(--color-text-primary)]">{action.action_type}</td>
                    <td className="px-3 py-2 text-body-small font-mono text-[var(--color-text-primary)]">{action.pid}</td>
                    <td className="px-3 py-2">
                      <span className={action.status === "completed" ? "badge-success" : action.status === "failed" ? "badge-error" : "badge-info"}>
                        {action.status}
                      </span>
                    </td>
                    <td className="px-3 py-2 text-body-small text-[var(--color-text-secondary)] max-w-[200px] truncate">
                      {action.output || action.error || "\u2014"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {confirmAction && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div ref={dialogRef} role="dialog" aria-modal="true" aria-label={`Confirm ${confirmAction.action}`} tabIndex={-1} className="card max-w-sm w-full mx-4 outline-none">
            <h3 className="text-headline-4 text-[var(--color-text-primary)] p-4">
              Confirm {confirmAction.action}
            </h3>
            <p className="px-4 text-body text-[var(--color-text-secondary)]">
              Are you sure you want to {confirmAction.action} <strong>{confirmAction.name}</strong> (PID {confirmAction.pid})?
            </p>
            <div className="flex justify-end gap-2 p-4">
              <button className="btn-ghost" onClick={() => setConfirmAction(null)}>
                Cancel
              </button>
              <button
                className={ACTION_CONFIG[confirmAction.action]?.className ?? "badge-error"}
                onClick={confirmActionExec}
                disabled={createAction.isPending}
              >
                {createAction.isPending ? "Sending..." : confirmAction.action}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
