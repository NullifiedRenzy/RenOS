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

interface DeviceContainersProps {
  deviceId: string;
}

export function DeviceContainers({ deviceId }: DeviceContainersProps) {
  const { data, isLoading, error, dataUpdatedAt, refetch, isRefetching } = useContainers(deviceId);
  const action = useContainerAction(deviceId);
  const containers = data?.containers ?? [];

  const [pendingStopCheck, setPendingStopCheck] = useState<string | null>(null);
  const [restartWarning, setRestartWarning] = useState<string | null>(null);

  const isActionPending = action.isPending;
  const actionTarget = action.variables ?? null;

  const canStart = (state: string) => ["exited", "created", "paused"].includes(state.toLowerCase());
  const canStop = (state: string) => ["running"].includes(state.toLowerCase());
  const canRestart = (state: string) => ["running"].includes(state.toLowerCase());

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

  const handleAction = (containerId: string, actionType: string) => {
    setRestartWarning(null);
    action.mutate(
      { containerId, action: actionType },
      {
        onSuccess: () => {
          if (actionType === "stop") {
            setPendingStopCheck(containerId);
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

      {action.isSuccess && !restartWarning && (
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
                const isLoadingContainer =
                  isActionPending && actionTarget?.containerId === c.container_id;
                const isStopping =
                  isLoadingContainer && actionTarget?.action === "stop";
                const isStarting =
                  isLoadingContainer && actionTarget?.action === "start";
                const isRestarting =
                  isLoadingContainer && actionTarget?.action === "restart";

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
                    <td className="py-2 pr-3 text-body-small text-[var(--color-text-secondary)] hidden md:table-cell">
                      {c.ports || "—"}
                    </td>
                    <td className="py-2 text-body-small">
                      <span className="inline-flex items-center gap-1">
                        {canStart(c.state) && (
                          <button
                            className="btn-ghost text-caption px-2 py-0.5"
                            disabled={isActionPending}
                            onClick={() => handleAction(c.container_id, "start")}
                          >
                            {isStarting ? "Starting…" : "Start"}
                          </button>
                        )}
                        {canStop(c.state) && (
                          <button
                            className="btn-ghost text-caption px-2 py-0.5 text-[var(--color-error)]"
                            disabled={isActionPending}
                            onClick={() => handleAction(c.container_id, "stop")}
                          >
                            {isStopping ? "Stopping…" : "Stop"}
                          </button>
                        )}
                        {canRestart(c.state) && (
                          <button
                            className="btn-ghost text-caption px-2 py-0.5"
                            disabled={isActionPending}
                            onClick={() => handleAction(c.container_id, "restart")}
                          >
                            {isRestarting ? "Restarting…" : "Restart"}
                          </button>
                        )}
                        {!canStart(c.state) && !canStop(c.state) && !canRestart(c.state) && (
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
    </div>
  );
}
