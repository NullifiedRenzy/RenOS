"use client";

import { useState, useMemo } from "react";
import { useSystemdServices } from "@/hooks/use-systemd-services";
import { useServiceAction } from "@/hooks/use-service-actions";
import type { SystemdService } from "@/lib/types";

interface SystemdServicesProps {
  deviceId: string;
}

const STATE_COLORS: Record<string, string> = {
  active: "var(--color-success)",
  inactive: "var(--color-text-tertiary)",
  failed: "var(--color-error)",
  activating: "var(--color-warning)",
  deactivating: "var(--color-warning)",
};

const ENABLED_LABELS: Record<string, string> = {
  enabled: "enabled",
  disabled: "disabled",
  static: "static",
  indirect: "indirect",
  generated: "generated",
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

export function SystemdServices({ deviceId }: SystemdServicesProps) {
  const { data, isLoading, error, refetch, isRefetching } = useSystemdServices(deviceId);
  const action = useServiceAction(deviceId);
  const services = useMemo(() => data?.services ?? [], [data?.services]);

  const [actionResult, setActionResult] = useState<{ title: string; output: string; error: string; exitCode: number | null } | null>(null);
  const [search, setSearch] = useState("");
  const [confirmTarget, setConfirmTarget] = useState<SystemdService | null>(null);
  const [confirmAction, setConfirmAction] = useState<string>("");

  const isActionPending = action.isPending;

  const canStart = (state: string) => ["inactive", "failed"].includes(state.toLowerCase());
  const canStop = (state: string) => ["active"].includes(state.toLowerCase());
  const canRestart = (state: string) => ["active", "inactive", "failed"].includes(state.toLowerCase());

  const filtered = useMemo(() => {
    if (!search.trim()) return services;
    const q = search.toLowerCase();
    return services.filter(
      s => s.unit_name.toLowerCase().includes(q) || s.description.toLowerCase().includes(q),
    );
  }, [services, search]);

  const handleAction = (unitName: string, actionType: string) => {
    setActionResult(null);
    setConfirmTarget(null);
    setConfirmAction("");
    action.mutate(
      { unitName, action: actionType },
      {
        onSuccess: (result) => {
          const act = result.service_action;
          setActionResult({
            title: `${actionType.charAt(0).toUpperCase() + actionType.slice(1)} — ${unitName}`,
            output: act.output ?? "",
            error: act.error ?? "",
            exitCode: act.exit_code,
          });
        },
      },
    );
  };

  const requestConfirm = (svc: SystemdService, actionType: string) => {
    setConfirmTarget(svc);
    setConfirmAction(actionType);
  };

  const cancelConfirm = () => {
    setConfirmTarget(null);
    setConfirmAction("");
  };

  return (
    <>
      <div className="card">
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-3">
            <h2 className="text-headline-4 text-[var(--color-text-primary)]">Systemd Services</h2>
            <input
              type="text"
              placeholder="Filter services\u2026"
              value={search}
              onChange={e => setSearch(e.target.value)}
              className="px-3 py-1.5 text-body-small rounded-sm border border-[var(--color-border)] bg-[var(--color-bg-primary)] text-[var(--color-text-primary)] placeholder:text-[var(--color-text-tertiary)] outline-none focus:border-[var(--color-accent)] w-56"
            />
          </div>
          <div className="flex items-center gap-2">
            {data?.services && (
              <span className="text-caption text-[var(--color-text-tertiary)]">
                {filtered.length}/{services.length} service{services.length !== 1 ? "s" : ""}
              </span>
            )}
            <button
              onClick={() => refetch()}
              disabled={isRefetching}
              className="btn-ghost text-body-small"
            >
              {isRefetching ? "Refreshing..." : "Refresh"}
            </button>
          </div>
        </div>

        {isLoading && (
          <div className="space-y-2">
            <div className="skeleton h-8 w-full" />
            <div className="skeleton h-8 w-full" />
            <div className="skeleton h-8 w-full" />
          </div>
        )}

        {error && (
          <p className="text-body text-[var(--color-text-tertiary)] py-3 text-center">
            Unable to load systemd services.
          </p>
        )}

        {!isLoading && !error && services.length === 0 && (
          <p className="text-body text-[var(--color-text-tertiary)] py-3 text-center">
            No systemd services found.
          </p>
        )}

        {!isLoading && !error && services.length > 0 && filtered.length === 0 && (
          <p className="text-body text-[var(--color-text-tertiary)] py-3 text-center">
            No services match &ldquo;{search}&rdquo;.
          </p>
        )}

        {!isLoading && !error && filtered.length > 0 && (
          <div className="overflow-x-auto">
            <table className="w-full text-left">
              <thead>
                <tr className="text-caption text-[var(--color-text-tertiary)] border-b border-[var(--color-border)]">
                  <th scope="col" className="pb-2 pr-3 font-medium">Service</th>
                  <th scope="col" className="pb-2 pr-3 font-medium">Status</th>
                  <th scope="col" className="pb-2 pr-3 font-medium hidden md:table-cell">Enabled</th>
                  <th scope="col" className="pb-2 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map(s => (
                  <tr key={s.id} className="border-b border-[var(--color-border)] last:border-b-0">
                    <td className="py-2 pr-3">
                      <div className="flex items-center gap-2">
                        {stateDot(s.active_state)}
                        <div>
                          <p className="text-body-small font-mono text-[var(--color-text-primary)]">{s.unit_name}</p>
                          {s.description && (
                            <p className="text-caption text-[var(--color-text-tertiary)]">{s.description}</p>
                          )}
                        </div>
                      </div>
                    </td>
                    <td className="py-2 pr-3">
                      <span className="text-body-small text-[var(--color-text-secondary)]">
                        {s.active_state}{s.sub_state && s.sub_state !== s.active_state ? ` (${s.sub_state})` : ""}
                      </span>
                    </td>
                    <td className="py-2 pr-3 hidden md:table-cell">
                      <span className="text-body-small text-[var(--color-text-secondary)]">
                        {s.enabled ? (ENABLED_LABELS[s.enabled] ?? s.enabled) : "\u2014"}
                      </span>
                    </td>
                    <td className="py-2">
                      <div className="flex items-center gap-1">
                        <button
                          onClick={() => handleAction(s.unit_name, "start")}
                          disabled={isActionPending || !canStart(s.active_state)}
                          className="btn-ghost text-body-small px-2 py-0.5"
                          title={canStart(s.active_state) ? "Start" : `Cannot start (${s.active_state})`}
                        >
                          Start
                        </button>
                        <button
                          onClick={() => requestConfirm(s, "stop")}
                          disabled={isActionPending || !canStop(s.active_state)}
                          className="btn-ghost text-body-small px-2 py-0.5"
                          title={canStop(s.active_state) ? "Stop" : `Cannot stop (${s.active_state})`}
                        >
                          Stop
                        </button>
                        <button
                          onClick={() => requestConfirm(s, "restart")}
                          disabled={isActionPending || !canRestart(s.active_state)}
                          className="btn-ghost text-body-small px-2 py-0.5"
                          title={canRestart(s.active_state) ? "Restart" : `Cannot restart (${s.active_state})`}
                        >
                          Restart
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {confirmTarget && (
        <div className="card border border-[var(--color-warning)]">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-headline-4 text-[var(--color-text-primary)]">
              {confirmAction === "stop" ? "Stop" : "Restart"} &mdash; {confirmTarget.unit_name}
            </h2>
          </div>
          <p className="text-body text-[var(--color-text-secondary)] mb-4">
            Are you sure you want to {confirmAction} <strong>{confirmTarget.unit_name}</strong>?
            {confirmTarget.description && (
              <> ({confirmTarget.description})</>
            )}
          </p>
          <div className="flex items-center gap-2">
            <button
              onClick={() => handleAction(confirmTarget.unit_name, confirmAction)}
              className="btn-primary text-body-small px-4 py-1.5"
            >
              Confirm {confirmAction}
            </button>
            <button
              onClick={cancelConfirm}
              className="btn-ghost text-body-small px-4 py-1.5"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {actionResult && (
        <div className="card">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-headline-4 text-[var(--color-text-primary)]">{actionResult.title}</h2>
            <button
              onClick={() => setActionResult(null)}
              className="btn-ghost text-body-small"
            >
              Close
            </button>
          </div>
          {actionResult.exitCode !== null && (
            <p className="text-body-small text-[var(--color-text-secondary)] mb-2">
              Exit code: {actionResult.exitCode}
            </p>
          )}
          {actionResult.output && (
            <pre className="bg-[var(--color-bg-secondary)] rounded p-3 overflow-x-auto text-body-small font-mono text-[var(--color-text-primary)] whitespace-pre-wrap">
              {actionResult.output}
            </pre>
          )}
          {actionResult.error && (
            <pre className="bg-[var(--color-bg-secondary)] rounded p-3 overflow-x-auto text-body-small font-mono text-[var(--color-error)] whitespace-pre-wrap mt-2">
              {actionResult.error}
            </pre>
          )}
          {!actionResult.output && !actionResult.error && (
            <p className="text-body text-[var(--color-text-tertiary)] py-2 text-center">
              Action submitted. Result will appear once the agent processes it.
            </p>
          )}
        </div>
      )}
    </>
  );
}
