"use client";

import { useState } from "react";
import { useAuth } from "@/hooks/use-auth";
import {
  useInstalledPlugins,
  useEnablePlugin,
  useDisablePlugin,
  useRemovePlugin,
  usePluginRoutes,
  usePluginPages,
  usePluginWorkers,
  usePluginSchedules,
} from "@/hooks/use-plugin-platform";
import { useDialogEscape } from "@/hooks/use-dialog";
import type { InstalledPlugin } from "@/lib/types";

const statusColors: Record<string, string> = {
  enabled: "var(--color-success)",
  installed: "var(--color-info)",
  disabled: "var(--color-warning)",
  error: "var(--color-error)",
  uninstalled: "var(--color-text-tertiary)",
};

const statusIcons: Record<string, string> = {
  enabled: "\u25CF",
  installed: "\u25CB",
  disabled: "\u25CB",
  error: "\u2716",
  uninstalled: "\u2716",
};

const typeLabels: Record<string, string> = {
  go: "Go Plugin",
  wasm: "WebAssembly",
  container: "Container",
  script: "Script",
};

export default function PluginsPage() {
  const { data, isLoading, error } = useInstalledPlugins();
  const { can } = useAuth();
  const enablePlugin = useEnablePlugin();
  const disablePlugin = useDisablePlugin();
  const removePlugin = useRemovePlugin();
  const [selectedPlugin, setSelectedPlugin] = useState<string | null>(null);
  const [confirmRemove, setConfirmRemove] = useState<string | null>(null);
  const dialogRef = useDialogEscape(() => setConfirmRemove(null), !!confirmRemove);
  const [filterStatus, setFilterStatus] = useState<string>("all");

  const plugins = data?.plugins ?? [];
  const filtered = filterStatus === "all"
    ? plugins
    : plugins.filter((p) => p.status === filterStatus);

  const counts = {
    all: plugins.length,
    enabled: plugins.filter((p) => p.status === "enabled").length,
    installed: plugins.filter((p) => p.status === "installed").length,
    disabled: plugins.filter((p) => p.status === "disabled").length,
    error: plugins.filter((p) => p.status === "error").length,
  };

  const handleEnable = (name: string) => {
    enablePlugin.mutate(name);
  };

  const handleDisable = (name: string) => {
    disablePlugin.mutate(name);
  };

  const handleRemove = (name: string) => {
    removePlugin.mutate(name, {
      onSuccess: () => setConfirmRemove(null),
    });
  };

  if (error) {
    return (
      <div className="space-y-6">
        <h1 className="text-headline-1 text-[var(--color-text-primary)]">Plugins</h1>
        <div className="card-base p-6 text-[var(--color-error)]">
          Failed to load plugins. Please try again.
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-headline-1 text-[var(--color-text-primary)]">Plugins</h1>
          <p className="text-body text-[var(--color-text-secondary)] mt-1">
            {isLoading ? "Loading..." : `${plugins.length} plugin${plugins.length !== 1 ? "s" : ""}`}
          </p>
        </div>
      </div>

      <div className="flex gap-2 flex-wrap">
        {(["all", "enabled", "installed", "disabled", "error"] as const).map((s) => (
          <button
            key={s}
            onClick={() => setFilterStatus(s)}
            className={`text-body-small px-2.5 py-1 rounded-sm transition-colors ${
              filterStatus === s
                ? "bg-[var(--color-accent-tint)] text-[var(--color-accent)] font-medium"
                : "text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)]"
            }`}
          >
            {s.charAt(0).toUpperCase() + s.slice(1)} ({counts[s] ?? 0})
          </button>
        ))}
      </div>

      {filtered.length === 0 ? (
        <div className="card-base p-12 text-center">
          <div className="text-[var(--color-text-tertiary)] text-body">
            {isLoading ? "Loading plugins..." : "No plugins installed"}
          </div>
          <p className="text-body-small text-[var(--color-text-tertiary)] mt-2">
            Add plugins to the plugin directory to get started
          </p>
        </div>
      ) : (
        <div className="space-y-3">
          {filtered.map((plugin) => (
            <PluginCard
              key={plugin.id}
              plugin={plugin}
              isSelected={selectedPlugin === plugin.name}
              onSelect={() => setSelectedPlugin(selectedPlugin === plugin.name ? null : plugin.name)}
              onEnable={() => handleEnable(plugin.name)}
              onDisable={() => handleDisable(plugin.name)}
              onRemove={() => setConfirmRemove(plugin.name)}
              canManage={can("manage_plugins") || can("admin")}
              isEnabling={enablePlugin.isPending}
              isDisabling={disablePlugin.isPending}
            />
          ))}
        </div>
      )}

      {confirmRemove && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div ref={dialogRef} role="dialog" aria-modal="true" aria-label="Remove Plugin" tabIndex={-1} className="card-base p-6 max-w-md w-full mx-4 space-y-4 outline-none">
            <h3 className="text-headline-3 text-[var(--color-text-primary)]">Remove Plugin</h3>
            <p className="text-body text-[var(--color-text-secondary)]">
              Are you sure you want to remove <strong>{confirmRemove}</strong>? This will disable and uninstall the plugin.
            </p>
            <div className="flex gap-3 justify-end">
              <button
                onClick={() => setConfirmRemove(null)}
                className="text-body-small px-3 py-1.5 text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
              >
                Cancel
              </button>
              <button
                onClick={() => handleRemove(confirmRemove)}
                disabled={removePlugin.isPending}
                className="text-body-small px-3 py-1.5 bg-[var(--color-error)] text-white rounded-sm hover:opacity-90 disabled:opacity-50"
              >
                {removePlugin.isPending ? "Removing..." : "Remove"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function PluginCard({
  plugin,
  isSelected,
  onSelect,
  onEnable,
  onDisable,
  onRemove,
  canManage,
  isEnabling,
  isDisabling,
}: {
  plugin: InstalledPlugin;
  isSelected: boolean;
  onSelect: () => void;
  onEnable: () => void;
  onDisable: () => void;
  onRemove: () => void;
  canManage: boolean;
  isEnabling: boolean;
  isDisabling: boolean;
}) {
  return (
    <div className="card-base overflow-hidden">
      <div
        className="p-4 flex items-center gap-4 cursor-pointer hover:bg-[var(--color-surface-hover)] transition-colors"
        onClick={onSelect}
      >
        <div className="w-10 h-10 rounded-md bg-[var(--color-accent-tint)] flex items-center justify-center flex-shrink-0">
          <span className="text-[var(--color-accent)] text-lg font-medium">
            {plugin.display_name?.charAt(0) ?? plugin.name.charAt(0)}
          </span>
        </div>

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-body-medium font-medium text-[var(--color-text-primary)] truncate">
              {plugin.display_name || plugin.name}
            </span>
            <span
              className="text-body-small px-1.5 py-0.5 rounded-sm"
              style={{
                backgroundColor: statusColors[plugin.status] + "20",
                color: statusColors[plugin.status],
              }}
            >
              {statusIcons[plugin.status]} {plugin.status}
            </span>
            <span className="text-body-small text-[var(--color-text-tertiary)]">
              {plugin.version}
            </span>
          </div>
          <p className="text-body-small text-[var(--color-text-tertiary)] truncate mt-0.5">
            {plugin.description || "No description"}
          </p>
        </div>

        <div className="flex items-center gap-2 flex-shrink-0">
          <span className="text-body-small text-[var(--color-text-tertiary)]">
            {typeLabels[plugin.plugin_type] ?? plugin.plugin_type}
          </span>
          {canManage && (
            <div className="flex gap-1">
              {plugin.status !== "enabled" ? (
                <button
                  onClick={(e) => { e.stopPropagation(); onEnable(); }}
                  disabled={isEnabling}
                  className="text-body-small px-2 py-1 bg-[var(--color-success)] text-white rounded-sm hover:opacity-90 disabled:opacity-50"
                >
                  Enable
                </button>
              ) : (
                <button
                  onClick={(e) => { e.stopPropagation(); onDisable(); }}
                  disabled={isDisabling}
                  className="text-body-small px-2 py-1 bg-[var(--color-warning)] text-white rounded-sm hover:opacity-90 disabled:opacity-50"
                >
                  Disable
                </button>
              )}
              <button
                onClick={(e) => { e.stopPropagation(); onRemove(); }}
                className="text-body-small px-2 py-1 bg-[var(--color-error)] text-white rounded-sm hover:opacity-90"
              >
                Remove
              </button>
            </div>
          )}
        </div>
      </div>

      {isSelected && <PluginDetails plugin={plugin} />}
    </div>
  );
}

function PluginDetails({ plugin }: { plugin: InstalledPlugin }) {
  const [activeTab, setActiveTab] = useState<"info" | "routes" | "pages" | "workers" | "schedules">("info");
  const { data: routesData } = usePluginRoutes(plugin.name);
  const { data: pagesData } = usePluginPages(plugin.name);
  const { data: workersData } = usePluginWorkers(plugin.name);
  const { data: schedulesData } = usePluginSchedules(plugin.name);

  const routes = routesData?.routes ?? [];
  const pages = pagesData?.pages ?? [];
  const workers = workersData?.workers ?? [];
  const schedules = schedulesData?.schedules ?? [];

  const tabs = [
    { id: "info" as const, label: "Info" },
    { id: "routes" as const, label: `Routes (${routes.length})` },
    { id: "pages" as const, label: `Pages (${pages.length})` },
    { id: "workers" as const, label: `Workers (${workers.length})` },
    { id: "schedules" as const, label: `Schedules (${schedules.length})` },
  ];

  return (
    <div className="border-t border-[var(--color-border)] bg-[var(--color-surface)]">
      <div className="flex border-b border-[var(--color-border)] px-4">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`text-body-small px-3 py-2 border-b-2 transition-colors ${
              activeTab === tab.id
                ? "border-[var(--color-accent)] text-[var(--color-accent)]"
                : "border-transparent text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)]"
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      <div className="p-4">
        {activeTab === "info" && (
          <div className="space-y-3 text-body-small">
            <div className="grid grid-cols-2 gap-3">
              <div>
                <span className="text-[var(--color-text-tertiary)]">Name: </span>
                <span className="text-[var(--color-text-primary)]">{plugin.name}</span>
              </div>
              <div>
                <span className="text-[var(--color-text-tertiary)]">Author: </span>
                <span className="text-[var(--color-text-primary)]">{plugin.author || "Unknown"}</span>
              </div>
              <div>
                <span className="text-[var(--color-text-tertiary)]">Type: </span>
                <span className="text-[var(--color-text-primary)]">{typeLabels[plugin.plugin_type] ?? plugin.plugin_type}</span>
              </div>
              <div>
                <span className="text-[var(--color-text-tertiary)]">Version: </span>
                <span className="text-[var(--color-text-primary)]">{plugin.version}</span>
              </div>
            </div>
            {plugin.description && (
              <p className="text-[var(--color-text-secondary)]">{plugin.description}</p>
            )}
            {plugin.error_message && (
              <div className="p-2 bg-[var(--color-error)]/10 border border-[var(--color-error)]/20 rounded-sm text-[var(--color-error)]">
                {plugin.error_message}
              </div>
            )}
            {plugin.permissions.length > 0 && (
              <div>
                <span className="text-[var(--color-text-tertiary)]">Permissions: </span>
                <span className="text-[var(--color-text-primary)]">{plugin.permissions.join(", ")}</span>
              </div>
            )}
          </div>
        )}

        {activeTab === "routes" && (
          <RoutesTab routes={routes} />
        )}

        {activeTab === "pages" && (
          <PagesTab pages={pages} />
        )}

        {activeTab === "workers" && (
          <WorkersTab workers={workers} />
        )}

        {activeTab === "schedules" && (
          <SchedulesTab schedules={schedules} />
        )}
      </div>
    </div>
  );
}

function RoutesTab({ routes }: { routes: import("@/lib/types").PluginRoute[] }) {
  if (routes.length === 0) {
    return <p className="text-body-small text-[var(--color-text-tertiary)]">No routes registered</p>;
  }
  return (
    <div className="space-y-2">
      {routes.map((route) => (
        <div key={route.id} className="flex items-center gap-3 text-body-small p-2 bg-[var(--color-surface-secondary)] rounded-sm">
          <span className={`px-1.5 py-0.5 rounded-sm font-mono text-[10px] font-bold ${
            route.method === "GET" ? "bg-[var(--color-success-tint)] text-[var(--color-success)]" :
            route.method === "POST" ? "bg-[var(--color-info-tint)] text-[var(--color-info)]" :
            route.method === "PUT" ? "bg-[var(--color-warning-tint)] text-[var(--color-warning)]" :
            route.method === "DELETE" ? "bg-[var(--color-error-tint)] text-[var(--color-error)]" :
            "bg-[var(--color-text-tertiary)]/20 text-[var(--color-text-tertiary)]"
          }`}>
            {route.method}
          </span>
          <span className="text-[var(--color-text-primary)] font-mono">{route.path}</span>
          {route.description && (
            <span className="text-[var(--color-text-tertiary)] ml-auto">{route.description}</span>
          )}
        </div>
      ))}
    </div>
  );
}

function PagesTab({ pages }: { pages: import("@/lib/types").PluginPage[] }) {
  if (pages.length === 0) {
    return <p className="text-body-small text-[var(--color-text-tertiary)]">No pages registered</p>;
  }
  return (
    <div className="space-y-2">
      {pages.map((page) => (
        <div key={page.id} className="flex items-center gap-3 text-body-small p-2 bg-[var(--color-surface-secondary)] rounded-sm">
          <span className="text-[var(--color-text-primary)]">{page.title || page.page_path}</span>
          <span className="text-[var(--color-text-tertiary)] font-mono">{page.page_path}</span>
          {page.component_name && (
            <span className="text-[var(--color-accent)] ml-auto">{page.component_name}</span>
          )}
        </div>
      ))}
    </div>
  );
}

function WorkersTab({ workers }: { workers: import("@/lib/types").PluginWorker[] }) {
  if (workers.length === 0) {
    return <p className="text-body-small text-[var(--color-text-tertiary)]">No workers registered</p>;
  }
  return (
    <div className="space-y-2">
      {workers.map((worker) => (
        <div key={worker.id} className="flex items-center gap-3 text-body-small p-2 bg-[var(--color-surface-secondary)] rounded-sm">
          <span className={`w-2 h-2 rounded-full ${
            worker.status === "running" ? "bg-[var(--color-success)]" :
            worker.status === "error" ? "bg-[var(--color-error)]" :
            "bg-[var(--color-text-tertiary)]"
          }`} />
          <span className="text-[var(--color-text-primary)]">{worker.worker_name}</span>
          <span className="text-[var(--color-text-tertiary)]">x{worker.concurrency}</span>
          <span className={`text-body-small ml-auto ${
            worker.status === "running" ? "text-[var(--color-success)]" :
            worker.status === "error" ? "text-[var(--color-error)]" :
            "text-[var(--color-text-tertiary)]"
          }`}>
            {worker.status}
          </span>
        </div>
      ))}
    </div>
  );
}

function SchedulesTab({ schedules }: { schedules: import("@/lib/types").PluginSchedule[] }) {
  if (schedules.length === 0) {
    return <p className="text-body-small text-[var(--color-text-tertiary)]">No schedules registered</p>;
  }
  return (
    <div className="space-y-2">
      {schedules.map((schedule) => (
        <div key={schedule.id} className="flex items-center gap-3 text-body-small p-2 bg-[var(--color-surface-secondary)] rounded-sm">
          <span className={`w-2 h-2 rounded-full ${
            schedule.enabled ? "bg-[var(--color-success)]" : "bg-[var(--color-text-tertiary)]"
          }`} />
          <span className="text-[var(--color-text-primary)]">{schedule.schedule_name}</span>
          <span className="text-[var(--color-text-tertiary)] font-mono">
            {schedule.cron_expr || `every ${schedule.interval_seconds}s`}
          </span>
          {schedule.last_status && (
            <span className={`ml-auto ${
              schedule.last_status === "success" ? "text-[var(--color-success)]" : "text-[var(--color-error)]"
            }`}>
              {schedule.last_status}
            </span>
          )}
        </div>
      ))}
    </div>
  );
}
