"use client";

import { useState } from "react";
import { useAuth } from "@/hooks/use-auth";
import {
  useNotificationProviders,
  useNotificationTemplates,
  useNotificationRules,
  useNotificationHistory,
  useNotificationStats,
  useCreateNotificationProvider,
  useDeleteNotificationProvider,
  useCreateNotificationTemplate,
  useDeleteNotificationTemplate,
  useCreateNotificationRule,
  useDeleteNotificationRule,
} from "@/hooks/use-notifications";
import type { NotificationProvider, NotificationTemplate, NotificationRule, NotificationHistoryEntry } from "@/lib/types";
import { ErrorState } from "@/components/ui/error-state";
import { ScreenReaderOnly } from "@/components/ui/screen-reader-only";
import { InboxTab } from "@/components/notifications/inbox-tab";

type NotifTab = "inbox" | "providers" | "templates" | "rules" | "history";

const statusColors: Record<string, string> = {
  sent: "var(--color-success)",
  pending: "var(--color-warning)",
  failed: "var(--color-error)",
  retrying: "var(--color-info)",
};

const providerTypeIcons: Record<string, string> = {
  email: "\u{2709}",
  slack: "\u{1F4AC}",
  webhook: "\u{1F310}",
  discord: "\u{1F5EC}",
  telegram: "\u{2708}",
  sms: "\u{1F4F1}",
  push: "\u{1F514}",
};

const tabLabels: Record<NotifTab, string> = {
  inbox: "Inbox",
  providers: "Providers",
  templates: "Templates",
  rules: "Rules",
  history: "History",
};

export default function NotificationsPage() {
  const { can } = useAuth();
  const [activeTab, setActiveTab] = useState<NotifTab>("inbox");

  return (
    <div className="space-y-6">
      <h1 className="text-headline-1 text-[var(--color-text-primary)]">Notifications</h1>

      <div className="flex gap-2 border-b border-[var(--color-border)] pb-0 overflow-x-auto">
        {(["inbox", "providers", "templates", "rules", "history"] as NotifTab[]).map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            aria-current={activeTab === tab ? "page" : undefined}
            className={`px-4 py-2 text-body-small rounded-t-lg border border-b-0 transition-colors capitalize ${
              activeTab === tab
                ? "bg-[var(--color-bg-primary)] text-[var(--color-text-primary)] border-[var(--color-border)] font-medium"
                : "bg-transparent text-[var(--color-text-tertiary)] border-transparent hover:text-[var(--color-text-secondary)]"
            }`}
          >
            {tabLabels[tab]}
          </button>
        ))}
      </div>

      <div className="mt-6">
        {activeTab === "inbox" && <InboxTab />}
        {activeTab === "providers" && <ProvidersTab />}
        {activeTab === "templates" && <TemplatesTab />}
        {activeTab === "rules" && <RulesTab />}
        {activeTab === "history" && <HistoryTab />}
      </div>
    </div>
  );
}

function ProvidersTab() {
  const { data: providers, isLoading, error } = useNotificationProviders();
  const createProvider = useCreateNotificationProvider();
  const deleteProvider = useDeleteNotificationProvider();
  const [showCreate, setShowCreate] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-headline-2 text-[var(--color-text-primary)]">Providers</h2>
        <button onClick={() => setShowCreate(!showCreate)} className="px-3 py-1.5 bg-[var(--color-accent)] text-white rounded-lg text-caption hover:opacity-90">
          {showCreate ? "Cancel" : "+ New"}
        </button>
      </div>

      {showCreate && <CreateProviderForm onClose={() => setShowCreate(false)} />}

      {error && !isLoading ? (
        <ErrorState error={error} title="Failed to load providers" />
      ) : isLoading ? (
        <Loading message="Loading..." />
      ) : !providers || providers.length === 0 ? (
        <EmptyState message="No notification providers configured" />
      ) : (
        <div className="space-y-2">
          {providers.map((p: NotificationProvider) => (
            <div key={p.id} className="flex items-center justify-between p-4 bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg">
              <div className="flex items-center gap-3">
                <span className="text-xl">{providerTypeIcons[p.provider_type] || "\u{1F514}"}</span>
                <div>
                  <div className="flex items-center gap-2">
                    <span className="text-body-small text-[var(--color-text-primary)] font-medium">{p.name}</span>
                    <span className="text-caption text-[var(--color-text-tertiary)] px-2 py-0.5 rounded bg-[var(--color-bg-tertiary)]">{p.provider_type}</span>
                  </div>
                </div>
              </div>
              <div className="flex items-center gap-3">
                <span className="text-caption" style={{ color: p.enabled ? "var(--color-success)" : "var(--color-text-tertiary)" }}>
                  {p.enabled ? "Active" : "Disabled"}
                </span>
                {confirmDelete === p.id ? (
                  <div className="flex gap-2">
                    <button onClick={() => deleteProvider.mutate(p.id, { onSuccess: () => setConfirmDelete(null) })} className="text-caption text-[var(--color-error)] hover:underline">Confirm</button>
                    <button onClick={() => setConfirmDelete(null)} className="text-caption text-[var(--color-text-tertiary)] hover:underline">Cancel</button>
                  </div>
                ) : (
                  <button onClick={() => setConfirmDelete(p.id)} className="text-caption text-[var(--color-text-tertiary)] hover:text-[var(--color-error)]">Delete</button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function CreateProviderForm({ onClose }: { onClose: () => void }) {
  const createProvider = useCreateNotificationProvider();
  const [name, setName] = useState("");
  const [providerType, setProviderType] = useState("webhook");

  const handleSubmit = () => {
    if (!name) return;
    createProvider.mutate({ name, provider_type: providerType, config: {}, enabled: true }, { onSuccess: onClose });
  };

  return (
    <div className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg p-6">
      <h3 className="text-headline-3 text-[var(--color-text-primary)] mb-4">New Provider</h3>
      <div className="grid grid-cols-2 gap-4">
        <input type="text" placeholder="Name" value={name} onChange={(e) => setName(e.target.value)} className="px-3 py-2 bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-lg text-body-small text-[var(--color-text-primary)]" />
        <select value={providerType} onChange={(e) => setProviderType(e.target.value)} className="px-3 py-2 bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-lg text-body-small text-[var(--color-text-primary)]">
          <option value="email">Email</option>
          <option value="slack">Slack</option>
          <option value="webhook">Webhook</option>
          <option value="discord">Discord</option>
          <option value="telegram">Telegram</option>
          <option value="sms">SMS</option>
          <option value="push">Push</option>
        </select>
      </div>
      <div className="flex gap-2 justify-end mt-4">
        <button onClick={onClose} className="px-4 py-2 text-body-small text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]">Cancel</button>
        <button onClick={handleSubmit} disabled={!name || createProvider.isPending} className="px-4 py-2 bg-[var(--color-accent)] text-white rounded-lg text-body-small hover:opacity-90 disabled:opacity-50">Create</button>
      </div>
    </div>
  );
}

function TemplatesTab() {
  const { data: templates, isLoading, error } = useNotificationTemplates();
  const createTemplate = useCreateNotificationTemplate();
  const deleteTemplate = useDeleteNotificationTemplate();
  const [showCreate, setShowCreate] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-headline-2 text-[var(--color-text-primary)]">Templates</h2>
        <button onClick={() => setShowCreate(!showCreate)} className="px-3 py-1.5 bg-[var(--color-accent)] text-white rounded-lg text-caption hover:opacity-90">
          {showCreate ? "Cancel" : "+ New"}
        </button>
      </div>

      {showCreate && <CreateTemplateForm onClose={() => setShowCreate(false)} />}

      {error && !isLoading ? (
        <ErrorState error={error} title="Failed to load templates" />
      ) : isLoading ? (
        <Loading message="Loading..." />
      ) : !templates || templates.length === 0 ? (
        <EmptyState message="No notification templates" />
      ) : (
        <div className="space-y-2">
          {templates.map((t: NotificationTemplate) => (
            <div key={t.id} className="flex items-center justify-between p-4 bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg">
              <div>
                <div className="text-body-small text-[var(--color-text-primary)] font-medium">{t.name}</div>
                <div className="text-caption text-[var(--color-text-tertiary)] mt-1 line-clamp-1">{t.body}</div>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-caption text-[var(--color-text-tertiary)] px-2 py-0.5 rounded bg-[var(--color-bg-tertiary)]">{t.template_type}</span>
                {confirmDelete === t.id ? (
                  <div className="flex gap-2">
                    <button onClick={() => deleteTemplate.mutate(t.id, { onSuccess: () => setConfirmDelete(null) })} className="text-caption text-[var(--color-error)] hover:underline">Confirm</button>
                    <button onClick={() => setConfirmDelete(null)} className="text-caption text-[var(--color-text-tertiary)] hover:underline">Cancel</button>
                  </div>
                ) : (
                  <button onClick={() => setConfirmDelete(t.id)} className="text-caption text-[var(--color-text-tertiary)] hover:text-[var(--color-error)]">Delete</button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function CreateTemplateForm({ onClose }: { onClose: () => void }) {
  const createTemplate = useCreateNotificationTemplate();
  const [name, setName] = useState("");
  const [subject, setSubject] = useState("");
  const [body, setBody] = useState("");

  const handleSubmit = () => {
    if (!name || !body) return;
    createTemplate.mutate({ name, subject, body, template_type: "message" }, { onSuccess: onClose });
  };

  return (
    <div className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg p-6">
      <h3 className="text-headline-3 text-[var(--color-text-primary)] mb-4">New Template</h3>
      <div className="space-y-4">
        <input type="text" placeholder="Template name" value={name} onChange={(e) => setName(e.target.value)} className="w-full px-3 py-2 bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-lg text-body-small text-[var(--color-text-primary)]" />
        <input type="text" placeholder="Subject" value={subject} onChange={(e) => setSubject(e.target.value)} className="w-full px-3 py-2 bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-lg text-body-small text-[var(--color-text-primary)]" />
        <textarea placeholder="Body (supports {{variables}})" value={body} onChange={(e) => setBody(e.target.value)} rows={4} className="w-full px-3 py-2 bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-lg text-body-small text-[var(--color-text-primary)] font-mono" />
        <div className="flex gap-2 justify-end">
          <button onClick={onClose} className="px-4 py-2 text-body-small text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]">Cancel</button>
          <button onClick={handleSubmit} disabled={!name || !body || createTemplate.isPending} className="px-4 py-2 bg-[var(--color-accent)] text-white rounded-lg text-body-small hover:opacity-90 disabled:opacity-50">Create</button>
        </div>
      </div>
    </div>
  );
}

function RulesTab() {
  const { data: rules, isLoading, error } = useNotificationRules();
  const createRule = useCreateNotificationRule();
  const deleteRule = useDeleteNotificationRule();
  const [showCreate, setShowCreate] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-headline-2 text-[var(--color-text-primary)]">Rules</h2>
        <button onClick={() => setShowCreate(!showCreate)} className="px-3 py-1.5 bg-[var(--color-accent)] text-white rounded-lg text-caption hover:opacity-90">
          {showCreate ? "Cancel" : "+ New"}
        </button>
      </div>

      {showCreate && <CreateRuleForm onClose={() => setShowCreate(false)} />}

      {error && !isLoading ? (
        <ErrorState error={error} title="Failed to load rules" />
      ) : isLoading ? (
        <Loading message="Loading..." />
      ) : !rules || rules.length === 0 ? (
        <EmptyState message="No notification rules" />
      ) : (
        <div className="space-y-2">
          {rules.map((r: NotificationRule) => (
            <div key={r.id} className="flex items-center justify-between p-4 bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg">
              <div>
                <div className="flex items-center gap-2">
                  <span className="w-2 h-2 rounded-full" aria-hidden="true" style={{ backgroundColor: r.enabled ? "var(--color-success)" : "var(--color-text-tertiary)" }} />
                  <ScreenReaderOnly>{r.enabled ? "Enabled" : "Disabled"}</ScreenReaderOnly>
                  <span className="text-body-small text-[var(--color-text-primary)] font-medium">{r.name}</span>
                  <span className="text-caption text-[var(--color-text-tertiary)] px-2 py-0.5 rounded bg-[var(--color-bg-tertiary)]">{r.event_type}</span>
                </div>
                {r.description && <p className="text-caption text-[var(--color-text-tertiary)] mt-1 ml-4">{r.description}</p>}
              </div>
              <div className="flex items-center gap-3">
                <span className="text-caption text-[var(--color-text-tertiary)]">Cooldown: {r.cooldown_seconds}s</span>
                {confirmDelete === r.id ? (
                  <div className="flex gap-2">
                    <button onClick={() => deleteRule.mutate(r.id, { onSuccess: () => setConfirmDelete(null) })} className="text-caption text-[var(--color-error)] hover:underline">Confirm</button>
                    <button onClick={() => setConfirmDelete(null)} className="text-caption text-[var(--color-text-tertiary)] hover:underline">Cancel</button>
                  </div>
                ) : (
                  <button onClick={() => setConfirmDelete(r.id)} className="text-caption text-[var(--color-text-tertiary)] hover:text-[var(--color-error)]">Delete</button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function CreateRuleForm({ onClose }: { onClose: () => void }) {
  const createRule = useCreateNotificationRule();
  const [name, setName] = useState("");
  const [eventType, setEventType] = useState("");
  const [cooldown, setCooldown] = useState("300");

  const handleSubmit = () => {
    if (!name || !eventType) return;
    createRule.mutate(
      { name, event_type: eventType, cooldown_seconds: parseInt(cooldown), conditions: {}, provider_ids: [], enabled: true },
      { onSuccess: onClose }
    );
  };

  return (
    <div className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg p-6">
      <h3 className="text-headline-3 text-[var(--color-text-primary)] mb-4">New Rule</h3>
      <div className="grid grid-cols-3 gap-4">
        <input type="text" placeholder="Rule name" value={name} onChange={(e) => setName(e.target.value)} className="px-3 py-2 bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-lg text-body-small text-[var(--color-text-primary)]" />
        <input type="text" placeholder="Event type (e.g. device.offline)" value={eventType} onChange={(e) => setEventType(e.target.value)} className="px-3 py-2 bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-lg text-body-small text-[var(--color-text-primary)]" />
        <input type="number" placeholder="Cooldown (sec)" value={cooldown} onChange={(e) => setCooldown(e.target.value)} className="px-3 py-2 bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-lg text-body-small text-[var(--color-text-primary)]" />
      </div>
      <div className="flex gap-2 justify-end mt-4">
        <button onClick={onClose} className="px-4 py-2 text-body-small text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]">Cancel</button>
        <button onClick={handleSubmit} disabled={!name || !eventType || createRule.isPending} className="px-4 py-2 bg-[var(--color-accent)] text-white rounded-lg text-body-small hover:opacity-90 disabled:opacity-50">Create</button>
      </div>
    </div>
  );
}

function HistoryTab() {
  const { data: history, isLoading, error } = useNotificationHistory();
  const { data: stats, error: statsError } = useNotificationStats();

  return (
    <div className="space-y-4">
      {statsError && !stats && <ErrorState error={statsError} title="Failed to load stats" />}
      {stats && (
        <div className="grid grid-cols-4 gap-4">
          {Object.entries(stats).map(([status, count]) => (
            <div key={status} className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg p-4">
              <div className="text-caption text-[var(--color-text-tertiary)] capitalize">{status}</div>
              <div className="text-headline-2 mt-1" style={{ color: statusColors[status] || "var(--color-text-primary)" }}>{count}</div>
            </div>
          ))}
        </div>
      )}

      {error && !isLoading ? (
        <ErrorState error={error} title="Failed to load notification history" />
      ) : isLoading ? (
        <Loading message="Loading history..." />
      ) : !history || history.length === 0 ? (
        <EmptyState message="No notification history" />
      ) : (
        <div className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg overflow-hidden">
          <table className="w-full">
            <thead>
              <tr className="border-b border-[var(--color-border)]">
                <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Status</th>
                <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Subject</th>
                <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Event</th>
                <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Sent</th>
              </tr>
            </thead>
            <tbody>
              {history.map((h: NotificationHistoryEntry) => (
                <tr key={h.id} className="border-b border-[var(--color-border)] last:border-0 hover:bg-[var(--color-bg-tertiary)]">
                  <td className="px-4 py-3">
                    <span className="text-caption px-2 py-1 rounded capitalize" style={{ backgroundColor: (statusColors[h.status] || "var(--color-text-tertiary)") + "20", color: statusColors[h.status] || "var(--color-text-tertiary)" }}>
                      {h.status}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-body-small text-[var(--color-text-primary)]">{h.subject || "—"}</td>
                  <td className="px-4 py-3 text-caption text-[var(--color-text-tertiary)]">{h.event_type}</td>
                  <td className="px-4 py-3 text-caption text-[var(--color-text-tertiary)]">{h.sent_at ? new Date(h.sent_at).toLocaleString() : "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function Loading({ message }: { message: string }) {
  return <div className="flex items-center justify-center py-12"><div className="text-body-small text-[var(--color-text-tertiary)]">{message}</div></div>;
}

function EmptyState({ message }: { message: string }) {
  return <div className="flex items-center justify-center py-12"><div className="text-body-small text-[var(--color-text-tertiary)]">{message}</div></div>;
}
