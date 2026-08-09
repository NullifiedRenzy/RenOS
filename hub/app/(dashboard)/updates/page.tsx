"use client";

import { useState, useEffect, useCallback } from "react";
import { api } from "@/lib/api-client";
import type { VersionInfo, UpdateHistoryEntry, UpdateBackup } from "@/lib/types";

export default function UpdatesPage() {
  const [versionInfo, setVersionInfo] = useState<VersionInfo | null>(null);
  const [history, setHistory] = useState<UpdateHistoryEntry[]>([]);
  const [backups, setBackups] = useState<UpdateBackup[]>([]);
  const [loading, setLoading] = useState(true);
  const [channel, setChannel] = useState("stable");
  const [checking, setChecking] = useState(false);
  const [updating, setUpdating] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [vRes, hRes, bRes] = await Promise.all([
        api.getVersionInfo(),
        api.getUpdateHistory(),
        api.getUpdateBackups(),
      ]);
      setVersionInfo(vRes.data);
      setHistory(hRes.data ?? []);
      setBackups(bRes.data ?? []);
      if (vRes.data.channel) setChannel(vRes.data.channel);
    } catch { /* ignore */ } finally { setLoading(false); }
  }, []);

  useEffect(() => { load(); }, [load]);

  const checkUpdates = async () => {
    setChecking(true);
    try {
      await api.checkForUpdates();
      await load();
    } catch { /* ignore */ } finally { setChecking(false); }
  };

  const startUpdate = async () => {
    setUpdating(true);
    try {
      await api.startUpdate(undefined, channel);
      await load();
    } catch { /* ignore */ } finally { setUpdating(false); }
  };

  const changeChannel = async (ch: string) => {
    await api.setUpdateChannel(ch);
    setChannel(ch);
  };

  const rollback = async (backupId: string) => {
    await api.rollbackUpdate(backupId);
    await load();
  };

  if (loading) return <div className="p-6 text-center text-[var(--color-text-tertiary)]">Loading update info...</div>;

  return (
    <div className="p-6 max-w-5xl mx-auto">
      <div className="mb-6">
        <h1 className="text-headline-2 font-semibold text-[var(--color-text-primary)]">Updates</h1>
        <p className="text-sm text-[var(--color-text-secondary)] mt-1">Manage RenOS versions and updates</p>
      </div>

      {/* Current Version */}
      <div className="bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-xl p-5 mb-6">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-headline-3 text-[var(--color-text-primary)]">Version</h2>
            <p className="text-sm text-[var(--color-text-secondary)] mt-1">
              Current: <span className="text-[var(--color-text-primary)] font-mono">{versionInfo?.current_version}</span>
              {versionInfo?.update_available && (
                <span className="ml-2">
                  → <span className="text-[var(--color-success)] font-mono">{versionInfo?.latest_version}</span>
                  <span className="text-xs text-[var(--color-success)] ml-1">(available)</span>
                </span>
              )}
            </p>
          </div>
          <div className="flex gap-2">
            <button onClick={checkUpdates} disabled={checking}
              className="bg-[var(--color-surface)] hover:bg-[var(--color-surface-hover)] text-[var(--color-text-primary)] px-4 py-2 rounded-lg text-sm transition-colors disabled:opacity-50">
              {checking ? "Checking..." : "Check for Updates"}
            </button>
            {versionInfo?.update_available && (
              <button onClick={startUpdate} disabled={updating}
                className="bg-[var(--color-accent)] hover:bg-[var(--color-accent-hover)] text-[var(--color-text-on-accent)] px-4 py-2 rounded-lg text-sm transition-colors disabled:opacity-50">
                {updating ? "Updating..." : "Update Now"}
              </button>
            )}
          </div>
        </div>
      </div>

      {/* Release Channel */}
      <div className="bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-xl p-5 mb-6">
        <h2 className="text-sm font-semibold text-[var(--color-text-secondary)] mb-3">Release Channel</h2>
        <div className="flex gap-2">
          {["stable", "beta", "nightly"].map((ch) => (
            <button key={ch} onClick={() => changeChannel(ch)}
              className={`px-4 py-2 rounded-lg text-sm transition-colors ${channel === ch ? "bg-[var(--color-surface-hover)] text-[var(--color-text-primary)]" : "bg-[var(--color-surface)] text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"}`}>
              {ch.charAt(0).toUpperCase() + ch.slice(1)}
            </button>
          ))}
        </div>
        <p className="text-xs text-[var(--color-text-tertiary)] mt-2">
          {channel === "stable" && "Production-ready releases, thoroughly tested"}
          {channel === "beta" && "Pre-release versions for testing"}
          {channel === "nightly" && "Daily builds, may be unstable"}
        </p>
      </div>

      {/* Update History */}
      <div className="bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-xl p-5 mb-6">
        <h2 className="text-sm font-semibold text-[var(--color-text-secondary)] mb-3">Update History</h2>
        {history.length === 0 ? (
          <p className="text-[var(--color-text-tertiary)] text-sm">No updates performed yet</p>
        ) : (
          <div className="space-y-2">
            {history.map((h) => (
              <div key={h.id} className="flex items-center justify-between px-4 py-3 bg-[var(--color-surface)] rounded-lg">
                <div>
                  <span className="text-sm text-[var(--color-text-primary)] font-mono">{h.version_from}</span>
                  <span className="text-xs text-[var(--color-text-tertiary)] mx-2">→</span>
                  <span className="text-sm text-[var(--color-text-primary)] font-mono">{h.version_to}</span>
                  <span className="text-xs text-[var(--color-text-tertiary)] ml-2">({h.channel})</span>
                </div>
                <div className="flex items-center gap-3">
                  <span className={`text-[10px] px-2 py-0.5 rounded ${
                    h.status === "completed" ? "bg-[var(--color-success-tint)] text-[var(--color-success)]" :
                    h.status === "failed" ? "bg-[var(--color-error-tint)] text-[var(--color-error)]" :
                    h.status === "rolled_back" ? "bg-[var(--color-warning-tint)] text-[var(--color-warning)]" :
                    "bg-[var(--color-surface)] text-[var(--color-text-secondary)]"
                  }`}>{h.status}</span>
                  <span className="text-xs text-[var(--color-text-tertiary)]">{new Date(h.started_at).toLocaleDateString()}</span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Backups */}
      <div className="bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-xl p-5">
        <h2 className="text-sm font-semibold text-[var(--color-text-secondary)] mb-3">Backups</h2>
        {backups.length === 0 ? (
          <p className="text-[var(--color-text-tertiary)] text-sm">No backups available</p>
        ) : (
          <div className="space-y-2">
            {backups.map((b) => (
              <div key={b.id} className="flex items-center justify-between px-4 py-3 bg-[var(--color-surface)] rounded-lg">
                <div>
                  <span className="text-sm text-[var(--color-text-primary)] font-mono">v{b.version}</span>
                  <span className="text-xs text-[var(--color-text-tertiary)] ml-2">{b.backup_type}</span>
                  <span className="text-xs text-[var(--color-text-tertiary)] ml-2">{(b.size_bytes / 1024 / 1024).toFixed(1)} MB</span>
                </div>
                <div className="flex items-center gap-3">
                  <span className={`text-[10px] px-2 py-0.5 rounded ${
                    b.status === "ready" ? "bg-[var(--color-success-tint)] text-[var(--color-success)]" : "bg-[var(--color-surface)] text-[var(--color-text-tertiary)]"
                  }`}>{b.status}</span>
                  {b.status === "ready" && (
                    <button onClick={() => rollback(b.id)} className="text-xs text-[var(--color-warning)] hover:text-[var(--color-warning)]">Rollback to this</button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
