"use client";

import { useContainers } from "@/hooks/use-containers";
import { useContainerAction } from "@/hooks/use-container-actions";
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

interface DeviceContainersProps {
  deviceId: string;
}

export function DeviceContainers({ deviceId }: DeviceContainersProps) {
  const { data, isLoading, error } = useContainers(deviceId);
  const action = useContainerAction(deviceId);
  const containers = data?.containers ?? [];

  const canStart = (state: string) => ["exited", "created", "paused"].includes(state.toLowerCase());
  const canStop = (state: string) => ["running"].includes(state.toLowerCase());
  const canRestart = (state: string) => ["running"].includes(state.toLowerCase());

  return (
    <div className="card">
      <h2 className="text-headline-4 text-[var(--color-text-primary)] mb-3">Docker Containers</h2>

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
              {containers.map((c: Container) => (
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
                      {c.status}
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
                          disabled={action.isPending}
                          onClick={() => action.mutate({ containerId: c.container_id, action: "start" })}
                        >
                          {action.isPending && action.variables?.containerId === c.container_id
                            ? "Starting..."
                            : "Start"}
                        </button>
                      )}
                      {canStop(c.state) && (
                        <button
                          className="btn-ghost text-caption px-2 py-0.5 text-[var(--color-error)]"
                          disabled={action.isPending}
                          onClick={() => action.mutate({ containerId: c.container_id, action: "stop" })}
                        >
                          {action.isPending && action.variables?.containerId === c.container_id
                            ? "Stopping..."
                            : "Stop"}
                        </button>
                      )}
                      {canRestart(c.state) && (
                        <button
                          className="btn-ghost text-caption px-2 py-0.5"
                          disabled={action.isPending}
                          onClick={() => action.mutate({ containerId: c.container_id, action: "restart" })}
                        >
                          {action.isPending && action.variables?.containerId === c.container_id
                            ? "Restarting..."
                            : "Restart"}
                        </button>
                      )}
                      {!canStart(c.state) && !canStop(c.state) && !canRestart(c.state) && (
                        <span className="text-caption text-[var(--color-text-tertiary)]">—</span>
                      )}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {action.isError && (
        <p className="text-body-small text-[var(--color-error)] mt-3">
          {(action.error as Error).message}
        </p>
      )}

      {action.isSuccess && (
        <p className="text-body-small text-[var(--color-success)] mt-3">
          Action completed successfully.
        </p>
      )}
    </div>
  );
}
