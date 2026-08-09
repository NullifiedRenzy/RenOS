"use client";

import { useState, useEffect, useCallback } from "react";
import { api } from "@/lib/api-client";
import type { PushRegistration, MobileSession, DeviceApproval } from "@/lib/types";

const statusColors: Record<string, string> = {
  pending: "text-[var(--color-warning)]",
  approved: "text-[var(--color-success)]",
  rejected: "text-[var(--color-error)]",
};

export default function MobilePage() {
  const [pushRegistrations, setPushRegistrations] = useState<PushRegistration[]>([]);
  const [sessions, setSessions] = useState<MobileSession[]>([]);
  const [approvals, setApprovals] = useState<DeviceApproval[]>([]);
  const [dashboard, setDashboard] = useState<{ total_devices: number; online_devices: number; offline_devices: number } | null>(null);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState<"overview" | "push" | "sessions" | "approvals">("overview");

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [pRes, sRes, aRes, dRes] = await Promise.all([
        api.listPushRegistrations(),
        api.listMobileSessions(),
        api.listDeviceApprovals(),
        api.getMobileDashboard(),
      ]);
      setPushRegistrations(pRes.data ?? []);
      setSessions(sRes.data ?? []);
      setApprovals(aRes.data ?? []);
      setDashboard(dRes.data);
    } catch { /* ignore */ } finally { setLoading(false); }
  }, []);

  useEffect(() => { load(); }, [load]);

  const approveDevice = async (id: string) => {
    await api.approveDevice(id);
    await load();
  };

  const rejectDevice = async (id: string) => {
    await api.rejectDevice(id, "Rejected by admin");
    await load();
  };

  const revokeAll = async () => {
    if (!confirm("Revoke all mobile sessions?")) return;
    await api.listMobileSessions();
    await load();
  };

  if (loading) return <div className="p-6 text-center text-[var(--color-text-tertiary)]">Loading mobile platform...</div>;

  return (
    <div className="p-6 max-w-6xl mx-auto">
      <div className="mb-6">
        <h1 className="text-headline-2 font-semibold text-[var(--color-text-primary)]">Mobile Platform</h1>
        <p className="text-sm text-[var(--color-text-secondary)] mt-1">Push notifications, device approvals, offline sync, and mobile sessions</p>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-4 gap-4 mb-6">
        <div className="bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-xl p-4">
          <div className="text-xs text-[var(--color-text-tertiary)] mb-1">Total Devices</div>
          <div className="text-2xl font-bold text-[var(--color-text-primary)]">{dashboard?.total_devices ?? 0}</div>
        </div>
        <div className="bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-xl p-4">
          <div className="text-xs text-[var(--color-text-tertiary)] mb-1">Online</div>
          <div className="text-2xl font-bold text-[var(--color-success)]">{dashboard?.online_devices ?? 0}</div>
        </div>
        <div className="bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-xl p-4">
          <div className="text-xs text-[var(--color-text-tertiary)] mb-1">Push Tokens</div>
          <div className="text-2xl font-bold text-[var(--color-text-primary)]">{pushRegistrations.length}</div>
        </div>
        <div className="bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-xl p-4">
          <div className="text-xs text-[var(--color-text-tertiary)] mb-1">Pending Approvals</div>
          <div className="text-2xl font-bold text-[var(--color-warning)]">{approvals.filter(a => a.status === "pending").length}</div>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-4 mb-4 border-b border-[var(--color-border)] pb-2">
        {(["overview", "push", "sessions", "approvals"] as const).map(t => (
          <button key={t} onClick={() => setTab(t)}
            className={`text-sm font-medium pb-1 border-b-2 transition ${tab === t ? "border-[var(--color-text-primary)] text-[var(--color-text-primary)]" : "border-transparent text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]"}`}>
            {t === "overview" ? "Overview" : t === "push" ? "Push Tokens" : t === "sessions" ? "Sessions" : "Approvals"}
          </button>
        ))}
      </div>

      {tab === "overview" && (
        <div className="grid grid-cols-2 gap-4">
          <div className="bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-xl p-5">
            <h3 className="text-sm font-semibold text-[var(--color-text-primary)] mb-3">Platform Status</h3>
            <div className="space-y-2 text-sm">
              <div className="flex justify-between"><span className="text-[var(--color-text-secondary)]">Registered Push Tokens</span><span className="text-[var(--color-text-primary)]">{pushRegistrations.length}</span></div>
              <div className="flex justify-between"><span className="text-[var(--color-text-secondary)]">Active Sessions</span><span className="text-[var(--color-text-primary)]">{sessions.length}</span></div>
              <div className="flex justify-between"><span className="text-[var(--color-text-secondary)]">Pending Approvals</span><span className="text-[var(--color-warning)]">{approvals.filter(a => a.status === "pending").length}</span></div>
            </div>
          </div>
          <div className="bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-xl p-5">
            <h3 className="text-sm font-semibold text-[var(--color-text-primary)] mb-3">Recent Approvals</h3>
            <div className="space-y-2">
              {approvals.slice(0, 5).map(a => (
                <div key={a.id} className="flex items-center justify-between text-sm">
                  <div>
                    <span className="text-[var(--color-text-secondary)]">{a.device_name || a.device_id}</span>
                    <span className={`ml-2 ${statusColors[a.status]}`}>{a.status}</span>
                  </div>
                </div>
              ))}
              {approvals.length === 0 && <p className="text-[var(--color-text-tertiary)] text-sm">No approvals yet</p>}
            </div>
          </div>
        </div>
      )}

      {tab === "push" && (
        <div className="bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-xl overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-[var(--color-border)] text-[var(--color-text-tertiary)]">
                <th scope="col" className="text-left p-3">Platform</th>
                <th scope="col" className="text-left p-3">Device</th>
                <th scope="col" className="text-left p-3">Model</th>
                <th scope="col" className="text-left p-3">OS</th>
                <th scope="col" className="text-left p-3">App Version</th>
                <th scope="col" className="text-left p-3">Enabled</th>
                <th scope="col" className="text-left p-3">Actions</th>
              </tr>
            </thead>
            <tbody>
              {pushRegistrations.map(p => (
                <tr key={p.id} className="border-b border-[var(--color-border)] hover:bg-[var(--color-surface)]">
                  <td className="p-3 font-mono text-[var(--color-text-secondary)]">{p.platform}</td>
                  <td className="p-3 text-[var(--color-text-primary)]">{p.device_name}</td>
                  <td className="p-3 text-[var(--color-text-secondary)]">{p.device_model}</td>
                  <td className="p-3 text-[var(--color-text-secondary)]">{p.os_version}</td>
                  <td className="p-3 text-[var(--color-text-secondary)]">{p.app_version}</td>
                  <td className="p-3"><span className={`inline-block w-2 h-2 rounded-full ${p.enabled ? "bg-[var(--color-success)]" : "bg-[var(--color-text-tertiary)]"}`} /></td>
                  <td className="p-3">
                    <button onClick={async () => { await api.deletePushRegistration(p.id); await load(); }}
                      className="text-xs text-[var(--color-error)] hover:text-[var(--color-error)]">Remove</button>
                  </td>
                </tr>
              ))}
              {pushRegistrations.length === 0 && (
                <tr><td colSpan={7} className="p-6 text-center text-[var(--color-text-tertiary)]">No push tokens registered</td></tr>
              )}
            </tbody>
          </table>
        </div>
      )}

      {tab === "sessions" && (
        <div className="bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-xl overflow-hidden">
          <div className="p-3 border-b border-[var(--color-border)] flex justify-end">
            <button onClick={revokeAll} className="text-xs bg-[var(--color-error-tint)] hover:bg-[var(--color-error-tint)] text-[var(--color-error)] px-3 py-1 rounded-lg transition">
              Revoke All
            </button>
          </div>
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-[var(--color-border)] text-[var(--color-text-tertiary)]">
                <th scope="col" className="text-left p-3">Device</th>
                <th scope="col" className="text-left p-3">Model</th>
                <th scope="col" className="text-left p-3">IP</th>
                <th scope="col" className="text-left p-3">Biometric</th>
                <th scope="col" className="text-left p-3">Push</th>
                <th scope="col" className="text-left p-3">Last Active</th>
                <th scope="col" className="text-left p-3">Actions</th>
              </tr>
            </thead>
            <tbody>
              {sessions.map(s => (
                <tr key={s.id} className="border-b border-[var(--color-border)] hover:bg-[var(--color-surface)]">
                  <td className="p-3 text-[var(--color-text-primary)]">{s.device_name}</td>
                  <td className="p-3 text-[var(--color-text-secondary)]">{s.device_model}</td>
                  <td className="p-3 font-mono text-[var(--color-text-secondary)]">{s.ip_address}</td>
                  <td className="p-3"><span className={`inline-block w-2 h-2 rounded-full ${s.biometric_enabled ? "bg-[var(--color-success)]" : "bg-[var(--color-text-tertiary)]"}`} /></td>
                  <td className="p-3"><span className={`inline-block w-2 h-2 rounded-full ${s.push_enabled ? "bg-[var(--color-success)]" : "bg-[var(--color-text-tertiary)]"}`} /></td>
                  <td className="p-3 text-[var(--color-text-tertiary)]">{new Date(s.last_active_at).toLocaleString()}</td>
                  <td className="p-3">
                    <button onClick={async () => { await api.deleteMobileSession(s.id); await load(); }}
                      className="text-xs text-[var(--color-error)] hover:text-[var(--color-error)]">Revoke</button>
                  </td>
                </tr>
              ))}
              {sessions.length === 0 && (
                <tr><td colSpan={7} className="p-6 text-center text-[var(--color-text-tertiary)]">No mobile sessions</td></tr>
              )}
            </tbody>
          </table>
        </div>
      )}

      {tab === "approvals" && (
        <div className="bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-xl overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-[var(--color-border)] text-[var(--color-text-tertiary)]">
                <th scope="col" className="text-left p-3">Status</th>
                <th scope="col" className="text-left p-3">Device</th>
                <th scope="col" className="text-left p-3">Model</th>
                <th scope="col" className="text-left p-3">IP</th>
                <th scope="col" className="text-left p-3">Fingerprint</th>
                <th scope="col" className="text-left p-3">Created</th>
                <th scope="col" className="text-left p-3">Actions</th>
              </tr>
            </thead>
            <tbody>
              {approvals.map(a => (
                <tr key={a.id} className="border-b border-[var(--color-border)] hover:bg-[var(--color-surface)]">
                  <td className={`p-3 ${statusColors[a.status]}`}>{a.status}</td>
                  <td className="p-3 text-[var(--color-text-primary)]">{a.device_name || a.device_id}</td>
                  <td className="p-3 text-[var(--color-text-secondary)]">{a.device_model}</td>
                  <td className="p-3 font-mono text-[var(--color-text-secondary)]">{a.ip_address}</td>
                  <td className="p-3 font-mono text-[var(--color-text-tertiary)] text-xs">{a.fingerprint?.slice(0, 16)}...</td>
                  <td className="p-3 text-[var(--color-text-tertiary)]">{new Date(a.created_at).toLocaleString()}</td>
                  <td className="p-3 space-x-2">
                    {a.status === "pending" && (
                      <>
                        <button onClick={() => approveDevice(a.id)} className="text-xs text-[var(--color-success)] hover:text-[var(--color-success)]">Approve</button>
                        <button onClick={() => rejectDevice(a.id)} className="text-xs text-[var(--color-error)] hover:text-[var(--color-error)]">Reject</button>
                      </>
                    )}
                  </td>
                </tr>
              ))}
              {approvals.length === 0 && (
                <tr><td colSpan={7} className="p-6 text-center text-[var(--color-text-tertiary)]">No device approval requests</td></tr>
              )}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
