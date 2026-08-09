"use client";

import { useState, useEffect, useCallback } from "react";
import { api } from "@/lib/api-client";
import type { MarketplacePlugin, MarketplaceInstallation } from "@/lib/types";

export function useMarketplace() {
  const [plugins, setPlugins] = useState<MarketplacePlugin[]>([]);
  const [installations, setInstallations] = useState<MarketplaceInstallation[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [pluginsRes, instRes] = await Promise.all([
        api.getMarketplacePlugins(),
        api.getMarketplaceInstallations(),
      ]);
      setPlugins(pluginsRes.data ?? []);
      setInstallations(instRes.data ?? []);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load marketplace");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const search = useCallback(async (q: string) => {
    setLoading(true);
    try {
      const res = await api.getMarketplacePlugins(q);
      setPlugins(res.data ?? []);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, []);

  const install = useCallback(async (name: string) => {
    await api.installMarketplacePlugin(name);
    await load();
  }, [load]);

  const upgrade = useCallback(async (name: string) => {
    await api.upgradeMarketplacePlugin(name);
    await load();
  }, [load]);

  const uninstall = useCallback(async (name: string) => {
    await api.uninstallMarketplacePlugin(name);
    await load();
  }, [load]);

  const rollback = useCallback(async (name: string, version: string) => {
    await api.rollbackMarketplacePlugin(name, version);
    await load();
  }, [load]);

  const isInstalled = useCallback((pluginName: string) => {
    return installations.some((i) => i.plugin_id === pluginName || i.installed_version);
  }, [installations]);

  return { plugins, installations, loading, error, load, search, install, upgrade, uninstall, rollback, isInstalled };
}
