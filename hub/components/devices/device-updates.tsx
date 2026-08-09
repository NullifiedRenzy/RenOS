"use client";

import { useUpdates, useUpdateAction, getSeverityColor, UPDATE_ACTIONS } from "@/hooks/use-updates";

interface UpdatesProps {
  deviceId: string;
}

export function Updates({ deviceId }: UpdatesProps) {
  const { data, isLoading, error } = useUpdates(deviceId);
  const updateAction = useUpdateAction(deviceId);
  const updates = data?.updates ?? [];

  const hasSecurity = updates.some(u => u.severity === "security");

  return (
    <div className="card">
      <div className="flex items-center justify-between mb-3">
        <h2 className="text-headline-4 text-[var(--color-text-primary)]">Updates</h2>
        <div className="flex items-center gap-1">
          {UPDATE_ACTIONS.map(action => (
            <button
              key={action.value}
              onClick={() => updateAction.mutate(action.value)}
              disabled={updateAction.isPending}
              className="btn-ghost text-body-small px-2 py-0.5"
            >
              {action.label}
            </button>
          ))}
        </div>
      </div>

      {hasSecurity && (
        <div className="bg-[var(--color-error)]/10 border border-[var(--color-error)]/30 rounded-md px-3 py-2 mb-3">
          <p className="text-body-small text-[var(--color-error)]">
            Security updates available
          </p>
        </div>
      )}

      {isLoading && (
        <div className="space-y-2">
          <div className="skeleton h-8 w-full" />
          <div className="skeleton h-8 w-full" />
        </div>
      )}

      {error && (
        <p className="text-body text-[var(--color-text-tertiary)] py-3 text-center">
          Unable to load updates.
        </p>
      )}

      {!isLoading && !error && updates.length === 0 && (
        <p className="text-body text-[var(--color-text-tertiary)] py-3 text-center">
          System is up to date.
        </p>
      )}

      {!isLoading && !error && updates.length > 0 && (
        <div className="overflow-x-auto">
          <table className="w-full text-left">
            <thead>
              <tr className="text-caption text-[var(--color-text-tertiary)] border-b border-[var(--color-border)]">
                <th scope="col" className="pb-2 pr-3 font-medium">Package</th>
                <th scope="col" className="pb-2 pr-3 font-medium">Current</th>
                <th scope="col" className="pb-2 pr-3 font-medium">Available</th>
                <th scope="col" className="pb-2 font-medium">Severity</th>
              </tr>
            </thead>
            <tbody>
              {updates.map(u => (
                <tr key={u.id} className="border-b border-[var(--color-border)] last:border-b-0">
                  <td className="py-2 pr-3">
                    <span className="text-body-small font-mono text-[var(--color-text-primary)]">{u.package_name}</span>
                  </td>
                  <td className="py-2 pr-3">
                    <span className="text-body-small text-[var(--color-text-secondary)]">{u.current_version}</span>
                  </td>
                  <td className="py-2 pr-3">
                    <span className="text-body-small text-[var(--color-text-secondary)]">{u.available_version}</span>
                  </td>
                  <td className="py-2">
                    <div className="flex items-center gap-1.5">
                      <span
                        className="inline-block w-2 h-2 rounded-full"
                        style={{ backgroundColor: getSeverityColor(u.severity) }}
                      />
                      <span className="text-body-small text-[var(--color-text-secondary)] capitalize">
                        {u.severity}
                      </span>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
