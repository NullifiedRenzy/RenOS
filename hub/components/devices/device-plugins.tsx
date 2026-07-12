"use client";

import { usePlugins, usePluginAction, getPluginStatusColor } from "@/hooks/use-plugins";

interface PluginsProps {
  deviceId: string;
}

export function Plugins({ deviceId }: PluginsProps) {
  const { data, isLoading, error } = usePlugins(deviceId);
  const pluginAction = usePluginAction(deviceId);
  const plugins = data?.plugins ?? [];

  const handleAction = (name: string, action: string) => {
    pluginAction.mutate({ name, action });
  };

  return (
    <div className="card">
      <h2 className="text-headline-4 text-[var(--color-text-primary)] mb-3">Plugins</h2>

      {isLoading && (
        <div className="space-y-2">
          <div className="skeleton h-8 w-full" />
          <div className="skeleton h-8 w-full" />
        </div>
      )}

      {error && (
        <p className="text-body text-[var(--color-text-tertiary)] py-3 text-center">
          Unable to load plugins.
        </p>
      )}

      {!isLoading && !error && plugins.length === 0 && (
        <p className="text-body text-[var(--color-text-tertiary)] py-3 text-center">
          No plugins installed. Drop executable scripts into /etc/renos/plugins/ on the device.
        </p>
      )}

      {!isLoading && !error && plugins.length > 0 && (
        <div className="overflow-x-auto">
          <table className="w-full text-left">
            <thead>
              <tr className="text-caption text-[var(--color-text-tertiary)] border-b border-[var(--color-border)]">
                <th className="pb-2 pr-3 font-medium">Name</th>
                <th className="pb-2 pr-3 font-medium hidden md:table-cell">Description</th>
                <th className="pb-2 pr-3 font-medium">Status</th>
                <th className="pb-2 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              {plugins.map(p => (
                <tr key={p.id} className="border-b border-[var(--color-border)] last:border-b-0">
                  <td className="py-2 pr-3">
                    <span className="text-body-small font-mono text-[var(--color-text-primary)]">{p.name}</span>
                  </td>
                  <td className="py-2 pr-3 hidden md:table-cell">
                    <span className="text-body-small text-[var(--color-text-secondary)]">
                      {p.description || "—"}
                    </span>
                  </td>
                  <td className="py-2 pr-3">
                    <div className="flex items-center gap-1.5">
                      <span
                        className="inline-block w-2 h-2 rounded-full"
                        style={{ backgroundColor: getPluginStatusColor(p.status) }}
                      />
                      <span className="text-body-small text-[var(--color-text-secondary)]">
                        {p.status}
                      </span>
                    </div>
                  </td>
                  <td className="py-2">
                    <button
                      onClick={() => handleAction(p.name, "run")}
                      disabled={pluginAction.isPending}
                      className="btn-ghost text-body-small px-2 py-0.5"
                    >
                      Run
                    </button>
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
