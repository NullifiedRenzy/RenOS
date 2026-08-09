"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { api } from "@/lib/api-client";
import type { PreflightCheck, PreflightReport, InstallationStatus, SystemInfo } from "@/lib/types";

function StatusIcon({ status }: { status: string }) {
  if (status === "pass") return <span className="text-[var(--color-success)]">✓</span>;
  if (status === "warn") return <span className="text-[var(--color-warning)]">⚠</span>;
  if (status === "fail") return <span className="text-[var(--color-error)]">✗</span>;
  return <span className="text-[var(--color-text-tertiary)]">—</span>;
}

function CheckCard({ check }: { check: PreflightCheck }) {
  return (
    <div className={`flex items-start gap-3 px-4 py-3 rounded-lg border ${
      check.status === "fail" ? "border-[var(--color-error)] bg-[var(--color-error-tint)]" :
      check.status === "warn" ? "border-[var(--color-warning)] bg-[var(--color-warning-tint)]" :
      "border-[var(--color-border)] bg-[var(--color-surface-elevated)]"
    }`}>
      <StatusIcon status={check.status} />
      <div className="flex-1 min-w-0">
        <div className="text-sm text-[var(--color-text-primary)] font-medium">{check.name}</div>
        <div className="text-xs text-[var(--color-text-secondary)] mt-0.5">{check.message}</div>
        {check.details && <div className="text-xs text-[var(--color-text-tertiary)] mt-0.5">{check.details}</div>}
      </div>
      <span className="text-[10px] text-[var(--color-text-tertiary)] uppercase shrink-0">{check.category}</span>
    </div>
  );
}

function ProgressBar({ progress }: { progress: number }) {
  return (
    <div className="w-full bg-[var(--color-surface)] rounded-full h-2">
      <div className="bg-[var(--color-success)] h-2 rounded-full transition-all duration-500" style={{ width: `${progress}%` }} />
    </div>
  );
}

export default function DeployPage() {
  const [preflight, setPreflight] = useState<PreflightReport | null>(null);
  const [status, setStatus] = useState<InstallationStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [installing, setInstalling] = useState(false);
  const [systemInfo, setSystemInfo] = useState<SystemInfo | null>(null);
  const eventSourceRef = useRef<EventSource | null>(null);

  // Form
  const [adminEmail, setAdminEmail] = useState("admin@renos.local");
  const [adminPassword, setAdminPassword] = useState("");
  const [externalUrl, setExternalUrl] = useState("https://localhost:8443");
  const [hostname, setHostname] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const pfRes = await api.getPreflightChecks();
      const siRes = await api.getSystemInfo();
      const stRes = await api.getInstallationStatus();
      setPreflight(pfRes.data);
      setSystemInfo(siRes.data);
      setStatus(stRes.data);
      if (siRes.data.hostname) {
        setHostname(String(siRes.data.hostname));
      }
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const startInstall = async () => {
    setInstalling(true);
    try {
      await api.startInstallation({
        admin_email: adminEmail,
        admin_password: adminPassword || undefined,
        external_url: externalUrl,
        hostname,
        non_interactive: true,
      });

      // Connect to SSE stream
      const es = new EventSource("/api/v1/deploy/stream");
      eventSourceRef.current = es;
      es.onmessage = (e) => {
        try {
          const data = JSON.parse(e.data);
          if (!data.keepalive) {
            setStatus(data);
            if (data.phase === "complete" || data.phase === "failed") {
              es.close();
              setInstalling(false);
            }
          }
        } catch { /* ignore */ }
      };
      es.onerror = () => { es.close(); setInstalling(false); };
    } catch {
      setInstalling(false);
    }
  };

  const rollback = async () => {
    await api.rollbackInstallation();
    await load();
  };

  if (loading) return <div className="p-6 text-center text-[var(--color-text-tertiary)]">Loading deployment info...</div>;

  const passCount = preflight?.pass_count ?? 0;
  const warnCount = preflight?.warn_count ?? 0;
  const failCount = preflight?.fail_count ?? 0;

  return (
    <div className="p-6 max-w-4xl mx-auto">
      <div className="mb-6">
        <h1 className="text-headline-2 font-semibold text-[var(--color-text-primary)]">RenDeploy</h1>
        <p className="text-sm text-[var(--color-text-secondary)] mt-1">Install and manage RenOS on this system</p>
      </div>

      {/* System Info */}
      {systemInfo && (
        <div className="bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-xl p-4 mb-6">
          <h2 className="text-sm font-semibold text-[var(--color-text-secondary)] mb-3">System Information</h2>
          <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
            <div>
              <span className="text-[10px] text-[var(--color-text-tertiary)] uppercase">OS</span>
              <div className="text-sm text-[var(--color-text-primary)]">{systemInfo.os}</div>
            </div>
            <div>
              <span className="text-[10px] text-[var(--color-text-tertiary)] uppercase">Arch</span>
              <div className="text-sm text-[var(--color-text-primary)]">{systemInfo.arch}</div>
            </div>
            <div>
              <span className="text-[10px] text-[var(--color-text-tertiary)] uppercase">Hostname</span>
              <div className="text-sm text-[var(--color-text-primary)]">{systemInfo.hostname}</div>
            </div>
            <div>
              <span className="text-[10px] text-[var(--color-text-tertiary)] uppercase">CPU Cores</span>
              <div className="text-sm text-[var(--color-text-primary)]">{systemInfo.cpu_cores}</div>
            </div>
            <div>
              <span className="text-[10px] text-[var(--color-text-tertiary)] uppercase">Docker</span>
              <div className="text-sm text-[var(--color-text-primary)]">{systemInfo.docker}</div>
            </div>
            <div>
              <span className="text-[10px] text-[var(--color-text-tertiary)] uppercase">Docker Compose</span>
              <div className="text-sm text-[var(--color-text-primary)]">{systemInfo.docker_compose}</div>
            </div>
          </div>
        </div>
      )}

      {/* Preflight Checks */}
      {preflight && (
        <div className="mb-6">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-headline-3 text-[var(--color-text-primary)]">Pre-flight Checks</h2>
            <div className="flex gap-3 text-xs">
              <span className="text-[var(--color-success)]">{passCount} passed</span>
              <span className="text-[var(--color-warning)]">{warnCount} warnings</span>
              <span className="text-[var(--color-error)]">{failCount} failed</span>
            </div>
          </div>
          <div className="space-y-2">
            {preflight.results.map((check, i) => <CheckCard key={i} check={check} />)}
          </div>
        </div>
      )}

      {/* Installation Progress */}
      {status && status.phase !== "idle" && (
        <div className="bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-xl p-4 mb-6">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-sm font-semibold text-[var(--color-text-secondary)]">Installation Progress</h2>
            <span className={`text-xs px-2 py-0.5 rounded ${
              status.phase === "complete" ? "bg-[var(--color-success-tint)] text-[var(--color-success)]" :
              status.phase === "failed" ? "bg-[var(--color-error-tint)] text-[var(--color-error)]" :
              "bg-[var(--color-info-tint)] text-[var(--color-info)]"
            }`}>{status.phase}</span>
          </div>
          <ProgressBar progress={status.progress} />
          <p className="text-xs text-[var(--color-text-secondary)] mt-2">{status.message}</p>
          {status.error && <p className="text-xs text-[var(--color-error)] mt-1">{status.error}</p>}
        </div>
      )}

      {/* Configuration Form */}
      <div className="bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-xl p-5 mb-6">
        <h2 className="text-sm font-semibold text-[var(--color-text-secondary)] mb-4">Configuration</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label className="block text-xs text-[var(--color-text-secondary)] mb-1">Admin Email</label>
            <input value={adminEmail} onChange={(e) => setAdminEmail(e.target.value)}
              className="w-full bg-[var(--color-surface)] border border-[var(--color-border-strong)] rounded-lg px-3 py-2 text-sm text-[var(--color-text-primary)] focus:outline-none focus:border-[var(--color-accent)]" />
          </div>
          <div>
            <label className="block text-xs text-[var(--color-text-secondary)] mb-1">Admin Password</label>
            <input type="password" value={adminPassword} onChange={(e) => setAdminPassword(e.target.value)} placeholder="Leave empty for default"
              className="w-full bg-[var(--color-surface)] border border-[var(--color-border-strong)] rounded-lg px-3 py-2 text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-tertiary)] focus:outline-none focus:border-[var(--color-accent)]" />
          </div>
          <div>
            <label className="block text-xs text-[var(--color-text-secondary)] mb-1">External URL</label>
            <input value={externalUrl} onChange={(e) => setExternalUrl(e.target.value)}
              className="w-full bg-[var(--color-surface)] border border-[var(--color-border-strong)] rounded-lg px-3 py-2 text-sm text-[var(--color-text-primary)] focus:outline-none focus:border-[var(--color-accent)]" />
          </div>
          <div>
            <label className="block text-xs text-[var(--color-text-secondary)] mb-1">Hostname</label>
            <input value={hostname} onChange={(e) => setHostname(e.target.value)} placeholder="Auto-detect"
              className="w-full bg-[var(--color-surface)] border border-[var(--color-border-strong)] rounded-lg px-3 py-2 text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-tertiary)] focus:outline-none focus:border-[var(--color-accent)]" />
          </div>
        </div>
      </div>

      {/* Actions */}
      <div className="flex gap-3">
        <button onClick={startInstall} disabled={installing || (preflight?.fail_count ?? 0) > 0}
          className="flex-1 bg-[var(--color-accent)] hover:bg-[var(--color-accent-hover)] text-[var(--color-text-on-accent)] py-3 rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
          {installing ? "Installing..." : "Install RenOS"}
        </button>
        <button onClick={rollback} disabled={installing}
          className="bg-[var(--color-surface)] hover:bg-[var(--color-surface-hover)] text-[var(--color-text-secondary)] px-6 py-3 rounded-lg text-sm transition-colors disabled:opacity-50">
          Rollback
        </button>
        <button onClick={load} disabled={installing}
          className="bg-[var(--color-surface)] hover:bg-[var(--color-surface-hover)] text-[var(--color-text-secondary)] px-6 py-3 rounded-lg text-sm transition-colors disabled:opacity-50">
          Refresh
        </button>
      </div>
    </div>
  );
}
