"use client";

import { useState } from "react";
import { useMarketplace } from "@/hooks/use-marketplace";
import type { MarketplacePlugin } from "@/lib/types";

const CATEGORIES = ["all", "infrastructure", "monitoring", "security", "automation", "devtools", "general"];

function PluginCard({ plugin, installed, onInstall, onUpgrade, onUninstall }: {
  plugin: MarketplacePlugin;
  installed: boolean;
  onInstall: () => Promise<void>;
  onUpgrade: () => Promise<void>;
  onUninstall: () => Promise<void>;
}) {
  const [actionLoading, setActionLoading] = useState(false);

  const handleAction = async (action: () => Promise<void>) => {
    setActionLoading(true);
    try { await action(); } finally { setActionLoading(false); }
  };

  return (
    <div className="border border-[var(--color-border)] rounded-lg p-5 hover:border-[var(--color-border-strong)] transition-colors">
      <div className="flex items-start justify-between mb-3">
        <div>
          <div className="flex items-center gap-2">
            <h3 className="font-semibold text-[var(--color-text-primary)]">{plugin.display_name || plugin.name}</h3>
            {plugin.verified && (
              <span className="text-[10px] bg-[var(--color-success-tint)] text-[var(--color-success)] px-1.5 py-0.5 rounded font-medium">VERIFIED</span>
            )}
          </div>
          <p className="text-xs text-[var(--color-text-tertiary)] mt-0.5">by {plugin.author} · v{plugin.version}</p>
        </div>
        <span className="text-[10px] bg-[var(--color-surface)] text-[var(--color-text-secondary)] px-2 py-1 rounded capitalize">{plugin.category}</span>
      </div>
      <p className="text-sm text-[var(--color-text-secondary)] mb-3 line-clamp-2">{plugin.description}</p>
      <div className="flex items-center justify-between">
        <div className="flex gap-1">
          {plugin.tags?.slice(0, 3).map((tag: string) => (
            <span key={tag} className="text-[10px] bg-[var(--color-surface)] text-[var(--color-text-tertiary)] px-1.5 py-0.5 rounded">{tag}</span>
          ))}
        </div>
        <div className="flex items-center gap-3 text-xs text-[var(--color-text-tertiary)]">
          <span>{plugin.downloads.toLocaleString()} downloads</span>
          <span>{plugin.rating > 0 ? `${plugin.rating.toFixed(1)}★` : ""}</span>
        </div>
      </div>
      <div className="mt-4 flex gap-2">
        {installed ? (
          <>
            <button onClick={() => handleAction(onUpgrade)} disabled={actionLoading}
              className="flex-1 text-xs bg-[var(--color-surface)] hover:bg-[var(--color-surface-hover)] text-[var(--color-text-primary)] py-2 rounded transition-colors disabled:opacity-50">
              {actionLoading ? "Working..." : "Upgrade"}
            </button>
            <button onClick={() => handleAction(onUninstall)} disabled={actionLoading}
              className="text-xs bg-[var(--color-surface-elevated)] hover:bg-[var(--color-error-tint)] text-[var(--color-error)] px-3 py-2 rounded transition-colors disabled:opacity-50">
              Remove
            </button>
          </>
        ) : (
          <button onClick={() => handleAction(onInstall)} disabled={actionLoading}
            className="flex-1 text-xs bg-[var(--color-accent)] hover:bg-[var(--color-accent-hover)] text-[var(--color-text-on-accent)] py-2 rounded transition-colors disabled:opacity-50">
            {actionLoading ? "Installing..." : "Install"}
          </button>
        )}
      </div>
    </div>
  );
}

export default function MarketplacePage() {
  const { plugins, installations, loading, error, search, install, upgrade, uninstall } = useMarketplace();
  const [query, setQuery] = useState("");
  const [category, setCategory] = useState("all");

  const filtered = plugins.filter((p) => {
    if (category !== "all" && p.category !== category) return false;
    return true;
  });

  const installedNames = new Set(installations.map((i) => i.plugin_id));

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    search(query);
  };

  return (
    <div className="p-6 max-w-7xl mx-auto">
      <div className="mb-6">
        <h1 className="text-headline-2 font-semibold text-[var(--color-text-primary)]">Marketplace</h1>
        <p className="text-sm text-[var(--color-text-secondary)] mt-1">Discover and install plugins for RenOS</p>
      </div>

      {error && (
        <div className="bg-[var(--color-error-tint)] border border-[var(--color-error)] rounded-lg p-3 mb-4 text-sm text-[var(--color-error)]">{error}</div>
      )}

      <div className="flex flex-col sm:flex-row gap-3 mb-6">
        <form onSubmit={handleSearch} className="flex-1 flex gap-2">
          <input value={query} onChange={(e) => setQuery(e.target.value)}
            placeholder="Search plugins..." className="flex-1 bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-lg px-4 py-2 text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-tertiary)] focus:outline-none focus:border-[var(--color-border-strong)]" />
          <button type="submit" className="bg-[var(--color-surface)] hover:bg-[var(--color-surface-hover)] text-[var(--color-text-primary)] px-4 py-2 rounded-lg text-sm transition-colors">Search</button>
        </form>
        <div className="flex gap-1 overflow-x-auto">
          {CATEGORIES.map((cat) => (
            <button key={cat} onClick={() => setCategory(cat)}
              className={`px-3 py-1.5 rounded-lg text-xs whitespace-nowrap transition-colors ${category === cat ? "bg-[var(--color-surface-hover)] text-[var(--color-text-primary)]" : "bg-[var(--color-surface-elevated)] text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"}`}>
              {cat.charAt(0).toUpperCase() + cat.slice(1)}
            </button>
          ))}
        </div>
      </div>

      {loading ? (
        <div className="text-center py-12 text-[var(--color-text-tertiary)] text-sm">Loading marketplace...</div>
      ) : filtered.length === 0 ? (
        <div className="text-center py-12">
          <p className="text-[var(--color-text-tertiary)] text-sm">No plugins found</p>
          <p className="text-[var(--color-text-tertiary)] text-xs mt-1">Try a different search or category</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {filtered.map((plugin) => (
            <PluginCard
              key={plugin.name}
              plugin={plugin}
              installed={installedNames.has(plugin.name)}
              onInstall={() => install(plugin.name)}
              onUpgrade={() => upgrade(plugin.name)}
              onUninstall={() => uninstall(plugin.name)}
            />
          ))}
        </div>
      )}

      {installations.length > 0 && (
        <div className="mt-8">
          <h2 className="text-headline-3 text-[var(--color-text-primary)] mb-3">Installed ({installations.length})</h2>
          <div className="space-y-2">
            {installations.map((inst) => {
              const plugin = plugins.find((p) => p.id === inst.plugin_id);
              return (
                <div key={inst.id} className="flex items-center justify-between bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-lg px-4 py-3">
                  <div>
                    <span className="text-sm text-[var(--color-text-primary)] font-medium">{plugin?.display_name || inst.plugin_id}</span>
                    <span className="text-xs text-[var(--color-text-tertiary)] ml-2">v{inst.installed_version}</span>
                  </div>
                  <span className={`text-[10px] px-2 py-0.5 rounded font-medium ${
                    inst.status === "installed" ? "bg-[var(--color-success-tint)] text-[var(--color-success)]" :
                    inst.status === "upgrading" ? "bg-[var(--color-info-tint)] text-[var(--color-info)]" :
                    inst.status === "rolling_back" ? "bg-[var(--color-warning-tint)] text-[var(--color-warning)]" :
                    "bg-[var(--color-error-tint)] text-[var(--color-error)]"
                  }`}>{inst.status}</span>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
