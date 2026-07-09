"use client";

import { useState, useEffect } from "react";
import { useContainers } from "@/hooks/use-containers";
import { useContainerAction } from "@/hooks/use-container-actions";
import { formatDate } from "@/lib/utils";
import type { Container } from "@/lib/types";

const STATE_COLORS: Record<string, string> = {
  running: "var(--color-success)",
  exited: "var(--color-text-tertiary)",
  paused: "var(--color-warning)",
  restarting: "var(--color-info)",
  removing: "var(--color-warning)",
  dead: "var(--color-error)",
  created: "var(--color-info)",
};

function stateDot(state: string) {
  const color = STATE_COLORS[state.toLowerCase()] ?? "var(--color-text-tertiary)";
  return (
    <span
      className="inline-block w-2 h-2 rounded-full flex-shrink-0"
      style={{ backgroundColor: color }}
    />
  );
}

function RefreshIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
      <polyline points="23 4 23 10 17 10" />
      <polyline points="1 20 1 14 7 14" />
      <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15" />
    </svg>
  );
}

function formatPorts(ports: string): string {
  if (!ports || ports === "—") return "—";
  return ports.split(", ").join(", ");
}

interface PortsPopupProps {
  ports: string;
}

function PortsDisplay({ ports }: PortsPopupProps) {
  if (!ports || ports === "—") return <span className="text-body-small text-[var(--color-text-tertiary)]">—</span>;

  const entries = ports.split(", ").filter(Boolean);
  if (entries.length === 0) return <span className="text-body-small text-[var(--color-text-tertiary)]">—</span>;

  return (
    <div className="text-body-small text-[var(--color-text-secondary)]">
      {entries.length <= 2 ? (
        formatPorts(ports)
      ) : (
        <span title={entries.join(", ")}>
          {entries[0]}, {entries[1]}, +{entries.length - 2} more
        </span>
      )}
    </div>
  );
}

interface ActionOutputModalProps {
  title: string;
  output: string;
  error: string;
  exitCode: number | null;
  onClose: () => void;
}

function ActionOutputModal({ title, output, error, exitCode, onClose }: ActionOutputModalProps) {
  const isError = exitCode !== 0;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40" onClick={onClose}>
      <div className="bg-[var(--color-surface)] rounded-md shadow-lg max-w-2xl w-full mx-4 max-h-[80vh] flex flex-col" onClick={e => e.stopPropagation()}>
        <div className="flex items-center justify-between p-4 border-b border-[var(--color-border)]">
          <h3 className="text-body font-medium text-[var(--color-text-primary)]">{title}</h3>
          <button className="btn-ghost p-1 text-body-small" onClick={onClose}>&#10005;</button>
        </div>
        <pre className="p-4 overflow-auto text-body-small font-mono text-[var(--color-text-primary)] whitespace-pre-wrap flex-1">
          {isError && error ? error : output || "(empty output)"}
        </pre>
        {exitCode !== null && (
          <div className="px-4 pb-3 text-caption text-[var(--color-text-tertiary)]">
            Exit code: {exitCode}
          </div>
        )}
      </div>
    </div>
  );
}

interface DeviceContainersProps {
  deviceId: string;
}

export function DeviceContainers({ deviceId }: DeviceContainersProps) {
  const { data, isLoading, error, dataUpdatedAt, refetch, isRefetching } = useContainers(deviceId);
  const action = useContainerAction(deviceId);
  const containers = data?.containers ?? [];

  const [pendingStopCheck, setPendingStopCheck] = useState<string | null>(null);
  const [restartWarning, setRestartWarning] = useState<string | null>(null);
  const [actionResult, setActionResult] = useState<{ title: string; output: string; error: string; exitCode: number | null } | null>(null);

  const isActionPending = action.isPending;
  const actionTarget = action.variables ?? null;

  const canStart = (state: string) => ["exited", "created", "paused"].includes(state.toLowerCase());
  const canStop = (state: string) => ["running"].includes(state.toLowerCase());
  const canRestart = (state: string) => ["running"].includes(state.toLowerCase());
  const canLogs = (state: string) => ["running", "exited"].includes(state.toLowerCase());
  const canStats = (state: string) => ["running"].includes(state.toLowerCase());

  useEffect(() => {
    if (!pendingStopCheck) return;
    const list = data?.containers;
    if (!list || list.length === 0) return;
    const c = list.find(ct => ct.container_id === pendingStopCheck);
    if (c && c.state === "running") {
      setRestartWarning(`"${c.name}" stopped but appears to have restarted — the container likely has a Docker restart policy configured.`);
    }
    setPendingStopCheck(null);
  }, [data, pendingStopCheck]);

  const handleAction = (containerId: string, actionType: string, containerName: string) => {
    setRestartWarning(null);
    setActionResult(null);
    action.mutate(
      { containerId, action: actionType },
      {
        onSuccess: (result) => {
          if (actionType === "stop") {
            setPendingStopCheck(containerId);
          }
          if (actionType === "logs" || actionType === "stats") {
            const act = result.container_action;
            setActionResult({
              title: `${actionType === "logs" ? "Logs" : "Stats"} — ${containerName}`,
              output: act.output ?? "",
              error: act.error ?? "",
              exitCode: act.exit_code,
            });
          }
        },
      },
    );
  };

  return (
    <div className="card">
      <div className="flex items-center justify-between mb-3">
        <h2 className="text-headline-4 text-[var(--color-text-primary)]">Docker Containers</h2>
        <span className="inline-flex items-center gap-2">
          {dataUpdatedAt > 0 && (
            <span className="text-caption text-[var(--color-text-tertiary)]">
              Updated {formatDate(new Date(dataUpdatedAt).toISOString())}
            </span>
          )}
          <button
            className="btn-ghost p-1.5"
            onClick={() => refetch()}
            disabled={isRefetching}
            title="Refresh containers"
          >
            <span className={isRefetching ? "animate-spin" : ""}>
              <RefreshIcon />
            </span>
          </button>
        </span>
      </div>

      {isLoading && (
        <div className="space-y-2">
          <div className="skeleton h-10 w-full" />
          <div className="skeleton h-10 w-full" />
        </div>
      )}

      {error && (
        <p className="text-body text-[var(--color-text-tertiary)] py-4 text-center">
          Unable to load Docker containers.
        </p>
      )}

      {restartWarning && (
        <p className="text-body-small text-[var(--color-warning)] mb-3 p-2 rounded-sm bg-[var(--color-warning)]/10">
          {restartWarning}
        </p>
      )}

      {action.isError && (
        <p className="text-body-small text-[var(--color-error)] mb-3">
          {(action.error as Error).message}
        </p>
      )}

      {action.isSuccess && !restartWarning && !actionResult && (
        <p className="text-body-small text-[var(--color-success)] mb-3">
          Action completed successfully.
        </p>
      )}

      {!isLoading && !error && containers.length === 0 && (
        <p className="text-body text-[var(--color-text-tertiary)] py-4 text-center">
          No Docker containers found or Docker is not available on this device.
        </p>
      )}

      {!isLoading && !error && containers.length > 0 && (
        <div className="overflow-x-auto">
          <table className="w-full text-left">
            <thead>
              <tr className="text-caption text-[var(--color-text-tertiary)] border-b border-[var(--color-border)]">
                <th className="pb-2 pr-3 font-medium">Name</th>
                <th className="pb-2 pr-3 font-medium">Image</th>
                <th className="pb-2 pr-3 font-medium">Status</th>
                <th className="pb-2 pr-3 font-medium hidden md:table-cell">Ports</th>
                <th className="pb-2 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              {containers.map((c: Container) => {
                const isPerformingAction =
                  isActionPending && actionTarget?.containerId === c.container_id;
                const isStopping =
                  isPerformingAction && actionTarget?.action === "stop";
                const isStarting =
                  isPerformingAction && actionTarget?.action === "start";
                const isRestarting =
                  isPerformingAction && actionTarget?.action === "restart";
                const isFetchingLogs =
                  isPerformingAction && actionTarget?.action === "logs";
                const isFetchingStats =
                  isPerformingAction && actionTarget?.action === "stats";

                return (
                  <tr key={c.id} className="border-b border-[var(--color-border)] last:border-b-0">
                    <td className="py-2 pr-3 text-body-small font-mono text-[var(--color-text-primary)]">
                      {c.name}
                    </td>
                    <td className="py-2 pr-3 text-body-small text-[var(--color-text-secondary)]">
                      {c.image}
                    </td>
                    <td className="py-2 pr-3 text-body-small text-[var(--color-text-primary)]">
                      <span className="inline-flex items-center gap-1.5">
                        {stateDot(c.state)}
                        <span className={isRestarting || isStopping ? "opacity-50" : ""}>
                          {isRestarting ? "Restarting…" : isStopping ? "Stopping…" : c.status}
                        </span>
                      </span>
                    </td>
                    <td className="py-2 pr-3 text-body-small hidden md:table-cell">
                      <PortsDisplay ports={c.ports} />
                    </td>
                    <td className="py-2 text-body-small">
                      <span className="inline-flex items-center gap-1">
                        {canStart(c.state) && (
                          <button
                            className="btn-ghost text-caption px-2 py-0.5"
                            disabled={isActionPending}
                            onClick={() => handleAction(c.container_id, "start", c.name)}
                          >
                            {isStarting ? "Starting…" : "Start"}
                          </button>
                        )}
                        {canStop(c.state) && (
                          <button
                            className="btn-ghost text-caption px-2 py-0.5 text-[var(--color-error)]"
                            disabled={isActionPending}
                            onClick={() => handleAction(c.container_id, "stop", c.name)}
                          >
                            {isStopping ? "Stopping…" : "Stop"}
                          </button>
                        )}
                        {canRestart(c.state) && (
                          <button
                            className="btn-ghost text-caption px-2 py-0.5"
                            disabled={isActionPending}
                            onClick={() => handleAction(c.container_id, "restart", c.name)}
                          >
                            {isRestarting ? "Restarting…" : "Restart"}
                          </button>
                        )}
                        {canLogs(c.state) && (
                          <button
                            className="btn-ghost text-caption px-2 py-0.5"
                            disabled={isActionPending}
                            onClick={() => handleAction(c.container_id, "logs", c.name)}
                          >
                            {isFetchingLogs ? "Fetching…" : "Logs"}
                          </button>
                        )}
                        {canStats(c.state) && (
                          <button
                            className="btn-ghost text-caption px-2 py-0.5"
                            disabled={isActionPending}
                            onClick={() => handleAction(c.container_id, "stats", c.name)}
                          >
                            {isFetchingStats ? "Fetching…" : "Stats"}
                          </button>
                        )}
                        {!canStart(c.state) && !canStop(c.state) && !canRestart(c.state) && !canLogs(c.state) && !canStats(c.state) && (
                          <span className="text-caption text-[var(--color-text-tertiary)]">—</span>
                        )}
                      </span>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {actionResult && (
        <ActionOutputModal
          title={actionResult.title}
          output={actionResult.output}
          error={actionResult.error}
          exitCode={actionResult.exitCode}
          onClose={() => setActionResult(null)}
        />
      )}
    </div>
  );
}
