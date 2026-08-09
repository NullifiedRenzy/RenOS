"use client";

import { useState, useEffect, useCallback } from "react";
import { api } from "@/lib/api-client";
import type { WebhookSubscription, WebhookDelivery, DeveloperAPIKey } from "@/lib/types";

const eventCategories: Record<string, string[]> = {
  "Device": ["device.online", "device.offline", "device.registered"],
  "Alerts": ["alert.fired", "alert.resolved", "alert.acknowledged"],
  "Deploy": ["deploy.started", "deploy.completed", "deploy.failed"],
  "Update": ["update.available", "update.started", "update.completed", "update.failed"],
  "Cluster": ["cluster.leader_elected", "cluster.node_offline"],
  "Workflow": ["workflow.started", "workflow.completed", "workflow.failed"],
  "Plugin": ["plugin.installed", "plugin.removed"],
  "Approval": ["approval.requested", "approval.approved", "approval.rejected"],
  "Mobile": ["mobile.device_approved", "mobile.device_rejected"],
};

export default function DeveloperPage() {
  const [webhooks, setWebhooks] = useState<WebhookSubscription[]>([]);
  const [apiKeys, setApiKeys] = useState<DeveloperAPIKey[]>([]);
  const [events, setEvents] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState<"webhooks" | "api-keys" | "docs">("webhooks");
  const [showNewWebhook, setShowNewWebhook] = useState(false);
  const [showNewKey, setShowNewKey] = useState(false);
  const [newWebhookUrl, setNewWebhookUrl] = useState("");
  const [newWebhookEvents, setNewWebhookEvents] = useState<string[]>([]);
  const [newKeyName, setNewKeyName] = useState("");
  const [newKeyScopes, setNewKeyScopes] = useState<string[]>([]);
  const [createdSecret, setCreatedSecret] = useState("");
  const [createdRawKey, setCreatedRawKey] = useState("");
  const [deliveries, setDeliveries] = useState<WebhookDelivery[]>([]);
  const [selectedWebhook, setSelectedWebhook] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [wRes, kRes, eRes] = await Promise.all([
        api.getWebhookSubscriptions(),
        api.getDeveloperAPIKeys(),
        api.getWebhookEvents(),
      ]);
      setWebhooks(wRes.data ?? []);
      setApiKeys(kRes.data ?? []);
      setEvents(eRes.data ?? []);
    } catch { /* ignore */ } finally { setLoading(false); }
  }, []);

  useEffect(() => { load(); }, [load]);

  const createWebhook = async () => {
    if (!newWebhookUrl || newWebhookEvents.length === 0) return;
    const res = await api.createWebhookSubscription({ url: newWebhookUrl, events: newWebhookEvents });
    setCreatedSecret(res.secret);
    setShowNewWebhook(false);
    setNewWebhookUrl("");
    setNewWebhookEvents([]);
    await load();
  };

  const createAPIKey = async () => {
    if (!newKeyName) return;
    const res = await api.createDeveloperAPIKey({ name: newKeyName, scopes: newKeyScopes });
    setCreatedRawKey(res.raw_key);
    setShowNewKey(false);
    setNewKeyName("");
    setNewKeyScopes([]);
    await load();
  };

  const viewDeliveries = async (webhookId: string) => {
    setSelectedWebhook(webhookId);
    const res = await api.getWebhookDeliveries(webhookId);
    setDeliveries(res.data ?? []);
    setTab("webhooks");
  };

  if (loading) return <div className="p-6 text-center text-[var(--color-text-tertiary)]">Loading developer platform...</div>;

  return (
    <div className="p-6 max-w-6xl mx-auto">
      <div className="mb-6">
        <h1 className="text-headline-2 font-semibold text-[var(--color-text-primary)]">Developer Platform</h1>
        <p className="text-sm text-[var(--color-text-secondary)] mt-1">Webhooks, API keys, and API documentation</p>
      </div>

      {/* Tabs */}
      <div className="flex gap-4 mb-4 border-b border-[var(--color-border)] pb-2">
        {(["webhooks", "api-keys", "docs"] as const).map(t => (
          <button key={t} onClick={() => { setTab(t); setSelectedWebhook(null); }}
            className={`text-sm font-medium pb-1 border-b-2 transition ${tab === t ? "border-white text-[var(--color-text-primary)]" : "border-transparent text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]"}`}>
            {t === "webhooks" ? "Webhooks" : t === "api-keys" ? "API Keys" : "API Docs"}
          </button>
        ))}
      </div>

      {tab === "webhooks" && (
        <div>
          <div className="flex justify-between items-center mb-4">
            <h2 className="text-headline-3 text-[var(--color-text-primary)]">Webhook Subscriptions</h2>
            <button onClick={() => setShowNewWebhook(true)} className="bg-[var(--color-accent)] hover:bg-[var(--color-accent-hover)] text-[var(--color-text-on-accent)] text-sm px-4 py-2 rounded-lg transition">
              New Webhook
            </button>
          </div>

          {createdSecret && (
            <div className="bg-[var(--color-success-tint)] border border-emerald-700 rounded-xl p-4 mb-4">
              <p className="text-sm text-[var(--color-success)] font-medium">Webhook created! Secret (store securely):</p>
              <code className="text-xs text-[var(--color-success)] font-mono break-all">{createdSecret}</code>
              <button onClick={() => setCreatedSecret("")} className="ml-2 text-xs text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]">Dismiss</button>
            </div>
          )}

          {showNewWebhook && (
            <div className="bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-xl p-5 mb-4">
              <h3 className="text-sm font-semibold text-[var(--color-text-primary)] mb-3">Create Webhook</h3>
              <input value={newWebhookUrl} onChange={e => setNewWebhookUrl(e.target.value)}
                placeholder="https://your-server.com/webhook" className="w-full bg-[var(--color-surface)] border border-[var(--color-border-strong)] rounded-lg px-3 py-2 text-sm text-[var(--color-text-primary)] mb-3" />
              <div className="mb-3">
                <p className="text-xs text-[var(--color-text-tertiary)] mb-2">Events:</p>
                <div className="space-y-1 max-h-40 overflow-y-auto">
                  {events.map(ev => (
                    <label key={ev} className="flex items-center gap-2 text-sm text-[var(--color-text-secondary)]">
                      <input type="checkbox" checked={newWebhookEvents.includes(ev)}
                        onChange={e => setNewWebhookEvents(prev => e.target.checked ? [...prev, ev] : prev.filter(x => x !== ev))}
                        className="rounded" />
                      {ev}
                    </label>
                  ))}
                </div>
              </div>
              <div className="flex gap-2">
                <button onClick={createWebhook} className="bg-[var(--color-accent)] hover:bg-[var(--color-accent-hover)] text-[var(--color-text-on-accent)] text-sm px-4 py-2 rounded-lg transition">Create</button>
                <button onClick={() => setShowNewWebhook(false)} className="bg-[var(--color-surface)] hover:bg-[var(--color-surface-hover)] text-[var(--color-text-secondary)] text-sm px-4 py-2 rounded-lg transition">Cancel</button>
              </div>
            </div>
          )}

          <div className="space-y-3">
            {webhooks.map(wh => (
              <div key={wh.id} className="bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-xl p-4">
                <div className="flex items-center justify-between">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className={`inline-block w-2 h-2 rounded-full ${wh.active ? "bg-[var(--color-success)]" : "bg-[var(--color-text-tertiary)]"}`} />
                      <span className="text-sm font-medium text-[var(--color-text-primary)] truncate">{wh.url}</span>
                    </div>
                    <div className="flex gap-1 mt-2 flex-wrap">
                      {wh.events.map(ev => (
                        <span key={ev} className="text-xs bg-[var(--color-surface)] text-[var(--color-text-secondary)] px-2 py-0.5 rounded-full">{ev}</span>
                      ))}
                    </div>
                  </div>
                  <div className="flex gap-2 ml-4">
                    <button onClick={() => viewDeliveries(wh.id)} className="text-xs text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]">Deliveries</button>
                    <button onClick={async () => { await api.toggleWebhookSubscription(wh.id, !wh.active); await load(); }}
                      className={`text-xs ${wh.active ? "text-[var(--color-warning)] hover:text-[var(--color-warning)]" : "text-[var(--color-success)] hover:text-[var(--color-success)]"}`}>
                      {wh.active ? "Disable" : "Enable"}
                    </button>
                    <button onClick={async () => { if (confirm("Delete webhook?")) { await api.deleteWebhookSubscription(wh.id); await load(); } }}
                      className="text-xs text-[var(--color-error)] hover:text-[var(--color-error)]">Delete</button>
                  </div>
                </div>
              </div>
            ))}
            {webhooks.length === 0 && <p className="text-[var(--color-text-tertiary)] text-sm text-center py-8">No webhooks configured</p>}
          </div>

          {selectedWebhook && deliveries.length > 0 && (
            <div className="mt-6">
              <h3 className="text-sm font-semibold text-[var(--color-text-primary)] mb-3">Recent Deliveries</h3>
              <div className="bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-xl overflow-hidden">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-[var(--color-border)] text-[var(--color-text-tertiary)]">
                      <th scope="col" className="text-left p-3">Status</th>
                      <th scope="col" className="text-left p-3">Event</th>
                      <th scope="col" className="text-left p-3">HTTP Code</th>
                      <th scope="col" className="text-left p-3">Retries</th>
                      <th scope="col" className="text-left p-3">Delivered At</th>
                    </tr>
                  </thead>
                  <tbody>
                    {deliveries.map(d => (
                      <tr key={d.id} className="border-b border-[var(--color-border)]">
                        <td className="p-3"><span className={`inline-block w-2 h-2 rounded-full ${d.success ? "bg-[var(--color-success)]" : "bg-[var(--color-error)]"}`} /></td>
                        <td className="p-3 font-mono text-[var(--color-text-secondary)]">{d.event}</td>
                        <td className="p-3 text-[var(--color-text-secondary)]">{d.status_code}</td>
                        <td className="p-3 text-[var(--color-text-secondary)]">{d.retry_count}</td>
                        <td className="p-3 text-[var(--color-text-tertiary)]">{new Date(d.delivered_at).toLocaleString()}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </div>
      )}

      {tab === "api-keys" && (
        <div>
          <div className="flex justify-between items-center mb-4">
            <h2 className="text-headline-3 text-[var(--color-text-primary)]">API Keys</h2>
            <button onClick={() => setShowNewKey(true)} className="bg-[var(--color-accent)] hover:bg-[var(--color-accent-hover)] text-[var(--color-text-on-accent)] text-sm px-4 py-2 rounded-lg transition">
              New API Key
            </button>
          </div>

          {createdRawKey && (
            <div className="bg-[var(--color-success-tint)] border border-emerald-700 rounded-xl p-4 mb-4">
              <p className="text-sm text-[var(--color-success)] font-medium">API key created! Copy it now (it won&apos;t be shown again):</p>
              <code className="text-xs text-[var(--color-success)] font-mono break-all">{createdRawKey}</code>
              <button onClick={() => setCreatedRawKey("")} className="ml-2 text-xs text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]">Dismiss</button>
            </div>
          )}

          {showNewKey && (
            <div className="bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-xl p-5 mb-4">
              <h3 className="text-sm font-semibold text-[var(--color-text-primary)] mb-3">Create API Key</h3>
              <input value={newKeyName} onChange={e => setNewKeyName(e.target.value)}
                placeholder="Key name (e.g. CI/CD)" className="w-full bg-[var(--color-surface)] border border-[var(--color-border-strong)] rounded-lg px-3 py-2 text-sm text-[var(--color-text-primary)] mb-3" />
              <div className="mb-3">
                <p className="text-xs text-[var(--color-text-tertiary)] mb-2">Scopes:</p>
                <div className="flex flex-wrap gap-2">
                  {["read", "write", "admin", "deploy", "devices", "alerts"].map(scope => (
                    <label key={scope} className="flex items-center gap-1 text-sm text-[var(--color-text-secondary)]">
                      <input type="checkbox" checked={newKeyScopes.includes(scope)}
                        onChange={e => setNewKeyScopes(prev => e.target.checked ? [...prev, scope] : prev.filter(x => x !== scope))}
                        className="rounded" />
                      {scope}
                    </label>
                  ))}
                </div>
              </div>
              <div className="flex gap-2">
                <button onClick={createAPIKey} className="bg-[var(--color-accent)] hover:bg-[var(--color-accent-hover)] text-[var(--color-text-on-accent)] text-sm px-4 py-2 rounded-lg transition">Create</button>
                <button onClick={() => setShowNewKey(false)} className="bg-[var(--color-surface)] hover:bg-[var(--color-surface-hover)] text-[var(--color-text-secondary)] text-sm px-4 py-2 rounded-lg transition">Cancel</button>
              </div>
            </div>
          )}

          <div className="bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-xl overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-[var(--color-border)] text-[var(--color-text-tertiary)]">
                  <th scope="col" className="text-left p-3">Name</th>
                  <th scope="col" className="text-left p-3">Prefix</th>
                  <th scope="col" className="text-left p-3">Scopes</th>
                  <th scope="col" className="text-left p-3">Active</th>
                  <th scope="col" className="text-left p-3">Last Used</th>
                  <th scope="col" className="text-left p-3">Actions</th>
                </tr>
              </thead>
              <tbody>
                {apiKeys.map(k => (
                  <tr key={k.id} className="border-b border-[var(--color-border)] hover:bg-[var(--color-surface-hover)]">
                    <td className="p-3 text-[var(--color-text-primary)]">{k.name}</td>
                    <td className="p-3 font-mono text-[var(--color-text-secondary)]">{k.key_prefix}...</td>
                    <td className="p-3">
                      <div className="flex gap-1 flex-wrap">
                        {k.scopes.map(s => <span key={s} className="text-xs bg-[var(--color-surface)] text-[var(--color-text-secondary)] px-2 py-0.5 rounded-full">{s}</span>)}
                      </div>
                    </td>
                    <td className="p-3"><span className={`inline-block w-2 h-2 rounded-full ${k.active ? "bg-[var(--color-success)]" : "bg-[var(--color-text-tertiary)]"}`} /></td>
                    <td className="p-3 text-[var(--color-text-tertiary)] text-xs">{k.last_used_at ? new Date(k.last_used_at).toLocaleString() : "Never"}</td>
                    <td className="p-3 space-x-2">
                      <button onClick={async () => { await api.toggleDeveloperAPIKey(k.id, !k.active); await load(); }}
                        className={`text-xs ${k.active ? "text-[var(--color-warning)] hover:text-[var(--color-warning)]" : "text-[var(--color-success)] hover:text-[var(--color-success)]"}`}>
                        {k.active ? "Revoke" : "Enable"}
                      </button>
                      <button onClick={async () => { if (confirm("Delete API key?")) { await api.deleteDeveloperAPIKey(k.id); await load(); } }}
                        className="text-xs text-[var(--color-error)] hover:text-[var(--color-error)]">Delete</button>
                    </td>
                  </tr>
                ))}
                {apiKeys.length === 0 && (
                  <tr><td colSpan={6} className="p-6 text-center text-[var(--color-text-tertiary)]">No API keys</td></tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {tab === "docs" && (
        <div className="bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-xl p-6">
          <h2 className="text-headline-3 text-[var(--color-text-primary)] mb-4">API Documentation</h2>
          <div className="space-y-4 text-sm text-[var(--color-text-secondary)]">
            <div>
              <h3 className="text-[var(--color-text-primary)] font-medium mb-1">Base URL</h3>
              <code className="bg-[var(--color-surface)] px-3 py-1 rounded text-[var(--color-text-secondary)]">/api/v1</code>
            </div>
            <div>
              <h3 className="text-[var(--color-text-primary)] font-medium mb-1">Authentication</h3>
              <p className="text-[var(--color-text-secondary)]">All endpoints require a Bearer token: <code className="bg-[var(--color-surface)] px-2 py-0.5 rounded text-[var(--color-text-secondary)]">Authorization: Bearer {"<token>"}</code></p>
            </div>
            <div>
              <h3 className="text-[var(--color-text-primary)] font-medium mb-2">Available Endpoints</h3>
              <div className="space-y-1 max-h-96 overflow-y-auto">
                {[
                  "GET  /devices              - List all devices",
                  "GET  /devices/{id}          - Get device details",
                  "POST /commands              - Execute command on device",
                  "GET  /cluster/dashboard     - Cluster overview",
                  "GET  /cluster/nodes         - List cluster nodes",
                  "GET  /updates               - Version info",
                  "POST /updates/check         - Check for updates",
                  "GET  /deploy/preflight      - Deployment preflight checks",
                  "POST /deploy/start          - Start installation",
                  "GET  /mobile/dashboard      - Mobile-optimized dashboard",
                  "POST /mobile/push/register  - Register push token",
                  "GET  /developer/webhooks    - List webhooks",
                  "GET  /developer/api-keys    - List API keys",
                  "GET  /monitoring/logs       - Query logs",
                  "GET  /alerts/active         - Active alerts",
                  "GET  /vault/secrets         - List secrets",
                  "GET  /marketplace           - Browse marketplace",
                ].map(ep => (
                  <div key={ep} className="font-mono text-xs text-[var(--color-text-secondary)] bg-[var(--color-surface)] px-3 py-1 rounded">{ep}</div>
                ))}
              </div>
            </div>
            <div>
              <h3 className="text-[var(--color-text-primary)] font-medium mb-1">Webhook Events</h3>
              <div className="grid grid-cols-2 gap-2">
                {Object.entries(eventCategories).map(([cat, evts]) => (
                  <div key={cat}>
                    <p className="text-[var(--color-text-tertiary)] text-xs mb-1">{cat}</p>
                    {evts.map(ev => <div key={ev} className="text-xs text-[var(--color-text-secondary)] font-mono">{ev}</div>)}
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
