"use client";

import { useState } from "react";
import { useIAM } from "@/hooks/use-iam";
import { useDialogEscape } from "@/hooks/use-dialog";

type Tab = "providers" | "oauth2" | "tokens" | "roles" | "approvals" | "audit";

function Modal({ open, onClose, title, children }: { open: boolean; onClose: () => void; title: string; children: React.ReactNode }) {
  const dialogRef = useDialogEscape(onClose, open);
  if (!open) return null;
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60" onClick={onClose}>
      <div ref={dialogRef} role="dialog" aria-modal="true" aria-label={title} tabIndex={-1}
        className="bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-xl p-6 w-full max-w-lg max-h-[80vh] overflow-y-auto outline-none" onClick={(e) => e.stopPropagation()}>
        <div className="flex justify-between items-center mb-4">
          <h3 className="text-headline-3 text-[var(--color-text-primary)]">{title}</h3>
          <button onClick={onClose} className="text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)] text-lg">✕</button>
        </div>
        {children}
      </div>
    </div>
  );
}

function Input({ label, value, onChange, placeholder, type = "text" }: { label: string; value: string; onChange: (v: string) => void; placeholder?: string; type?: string }) {
  return (
    <div className="mb-3">
      <label className="block text-xs text-[var(--color-text-secondary)] mb-1">{label}</label>
      <input type={type} value={value} onChange={(e) => onChange(e.target.value)} placeholder={placeholder}
        className="w-full bg-[var(--color-surface)] border border-[var(--color-border-strong)] rounded-lg px-3 py-2 text-sm text-[var(--color-text-primary)] placeholder:text-[var(--color-text-tertiary)] focus:outline-none focus:border-[var(--color-accent)]" />
    </div>
  );
}

export default function IAMPage() {
  const iam = useIAM();
  const [tab, setTab] = useState<Tab>("providers");
  const [showModal, setShowModal] = useState(false);

  const tabs: { key: Tab; label: string; count?: number }[] = [
    { key: "providers", label: "Identity Providers", count: iam.providers.length },
    { key: "oauth2", label: "OAuth2 Apps", count: iam.oauth2Apps.length },
    { key: "tokens", label: "API Tokens", count: iam.tokens.length },
    { key: "roles", label: "RBAC Roles", count: iam.roles.length },
    { key: "approvals", label: "Approvals", count: iam.requests.filter((r) => r.status === "pending").length },
    { key: "audit", label: "Audit Log", count: iam.auditLog.length },
  ];

  return (
    <div className="p-6 max-w-7xl mx-auto">
      <div className="mb-6">
        <h1 className="text-headline-2 font-semibold text-[var(--color-text-primary)]">Identity & Access</h1>
        <p className="text-sm text-[var(--color-text-secondary)] mt-1">Manage providers, tokens, roles, and approval workflows</p>
      </div>

      {iam.error && <div className="bg-[var(--color-error-tint)] border border-[var(--color-error)] rounded-lg p-3 mb-4 text-sm text-[var(--color-error)]">{iam.error}</div>}

      <div className="flex gap-1 mb-6 overflow-x-auto border-b border-[var(--color-border)] pb-2">
        {tabs.map((t) => (
          <button key={t.key} onClick={() => setTab(t.key)}
            className={`px-4 py-2 rounded-lg text-sm whitespace-nowrap transition-colors ${tab === t.key ? "bg-[var(--color-surface)] text-[var(--color-text-primary)]" : "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"}`}>
            {t.label}
            {t.count !== undefined && t.count > 0 && (
              <span className="ml-2 text-[10px] bg-[var(--color-surface-hover)] text-[var(--color-text-secondary)] px-1.5 py-0.5 rounded-full">{t.count}</span>
            )}
          </button>
        ))}
      </div>

      {iam.loading ? (
        <div className="text-center py-12 text-[var(--color-text-tertiary)] text-sm">Loading...</div>
      ) : (
        <>
          {/* Identity Providers */}
          {tab === "providers" && (
            <div>
              <div className="flex justify-between items-center mb-4">
                <h2 className="text-headline-3 text-[var(--color-text-primary)]">Identity Providers</h2>
                <button onClick={() => setShowModal(true)} className="bg-[var(--color-surface)] hover:bg-[var(--color-surface-hover)] text-[var(--color-text-primary)] px-3 py-1.5 rounded-lg text-xs transition-colors">Add Provider</button>
              </div>
              {iam.providers.length === 0 ? (
                <p className="text-[var(--color-text-tertiary)] text-sm">No identity providers configured</p>
              ) : (
                <div className="space-y-2">
                  {iam.providers.map((p) => (
                    <div key={p.id} className="flex items-center justify-between bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-lg px-4 py-3">
                      <div>
                        <span className="text-sm text-[var(--color-text-primary)] font-medium">{p.name}</span>
                        <span className="text-xs text-[var(--color-text-tertiary)] ml-2 uppercase">{p.provider_type}</span>
                      </div>
                      <div className="flex items-center gap-3">
                        <span className={`text-[10px] px-2 py-0.5 rounded ${p.enabled ? "bg-[var(--color-success-tint)] text-[var(--color-success)]" : "bg-[var(--color-surface)] text-[var(--color-text-tertiary)]"}`}>
                          {p.enabled ? "enabled" : "disabled"}
                        </span>
                        <button onClick={() => iam.deleteProvider(p.id)} className="text-xs text-[var(--color-error)] hover:text-[var(--color-error)]">Delete</button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* OAuth2 Apps */}
          {tab === "oauth2" && (
            <div>
              <div className="flex justify-between items-center mb-4">
                <h2 className="text-headline-3 text-[var(--color-text-primary)]">OAuth2 Applications</h2>
                <button onClick={() => setShowModal(true)} className="bg-[var(--color-surface)] hover:bg-[var(--color-surface-hover)] text-[var(--color-text-primary)] px-3 py-1.5 rounded-lg text-xs transition-colors">Register App</button>
              </div>
              {iam.oauth2Apps.length === 0 ? (
                <p className="text-[var(--color-text-tertiary)] text-sm">No OAuth2 apps registered</p>
              ) : (
                <div className="space-y-2">
                  {iam.oauth2Apps.map((a) => (
                    <div key={a.id} className="flex items-center justify-between bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-lg px-4 py-3">
                      <div>
                        <span className="text-sm text-[var(--color-text-primary)] font-medium">{a.name}</span>
                        <span className="text-xs text-[var(--color-text-tertiary)] ml-2 font-mono">{a.client_id.slice(0, 12)}...</span>
                      </div>
                      <div className="flex items-center gap-3">
                        <span className={`text-[10px] px-2 py-0.5 rounded ${a.enabled ? "bg-[var(--color-success-tint)] text-[var(--color-success)]" : "bg-[var(--color-surface)] text-[var(--color-text-tertiary)]"}`}>
                          {a.enabled ? "enabled" : "disabled"}
                        </span>
                        <button onClick={() => iam.deleteOAuth2App(a.id)} className="text-xs text-[var(--color-error)] hover:text-[var(--color-error)]">Delete</button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* API Tokens */}
          {tab === "tokens" && (
            <div>
              <div className="flex justify-between items-center mb-4">
                <h2 className="text-headline-3 text-[var(--color-text-primary)]">API Tokens</h2>
                <button onClick={() => setShowModal(true)} className="bg-[var(--color-surface)] hover:bg-[var(--color-surface-hover)] text-[var(--color-text-primary)] px-3 py-1.5 rounded-lg text-xs transition-colors">Create Token</button>
              </div>
              {iam.tokens.length === 0 ? (
                <p className="text-[var(--color-text-tertiary)] text-sm">No API tokens</p>
              ) : (
                <div className="space-y-2">
                  {iam.tokens.map((t) => (
                    <div key={t.id} className="flex items-center justify-between bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-lg px-4 py-3">
                      <div>
                        <span className="text-sm text-[var(--color-text-primary)] font-medium">{t.name}</span>
                        <span className="text-xs text-[var(--color-text-tertiary)] ml-2 font-mono">{t.token_prefix}...</span>
                        {t.last_used_at && <span className="text-xs text-[var(--color-text-tertiary)] ml-2">last used {new Date(t.last_used_at).toLocaleDateString()}</span>}
                      </div>
                      <button onClick={() => iam.deleteToken(t.id)} className="text-xs text-[var(--color-error)] hover:text-[var(--color-error)]">Revoke</button>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* RBAC Roles */}
          {tab === "roles" && (
            <div>
              <div className="flex justify-between items-center mb-4">
                <h2 className="text-headline-3 text-[var(--color-text-primary)]">RBAC Roles & Assignments</h2>
                <button onClick={() => setShowModal(true)} className="bg-[var(--color-surface)] hover:bg-[var(--color-surface-hover)] text-[var(--color-text-primary)] px-3 py-1.5 rounded-lg text-xs transition-colors">Create Role</button>
              </div>
              <div className="space-y-2 mb-6">
                {iam.roles.map((r) => (
                  <div key={r.id} className="flex items-center justify-between bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-lg px-4 py-3">
                    <div>
                      <span className="text-sm text-[var(--color-text-primary)] font-medium">{r.display_name || r.name}</span>
                      {r.is_system && <span className="text-[10px] bg-[var(--color-info-tint)] text-[var(--color-info)] ml-2 px-1.5 py-0.5 rounded">system</span>}
                      <p className="text-xs text-[var(--color-text-tertiary)] mt-0.5">{r.description}</p>
                    </div>
                    {!r.is_system && <button onClick={() => iam.deleteRole(r.id)} className="text-xs text-[var(--color-error)] hover:text-[var(--color-error)]">Delete</button>}
                  </div>
                ))}
              </div>
              <h3 className="text-sm font-semibold text-[var(--color-text-secondary)] mb-2">Assignments</h3>
              <div className="space-y-2">
                {iam.assignments.map((a) => {
                  const role = iam.roles.find((r) => r.id === a.role_id);
                  return (
                    <div key={a.id} className="flex items-center justify-between bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-lg px-4 py-3">
                      <div>
                        <span className="text-xs text-[var(--color-text-secondary)]">User</span>
                        <span className="text-sm text-[var(--color-text-primary)] ml-2 font-mono">{a.user_id.slice(0, 8)}...</span>
                        <span className="text-xs text-[var(--color-text-tertiary)] mx-2">→</span>
                        <span className="text-sm text-[var(--color-text-primary)]">{role?.name || a.role_id}</span>
                        {a.resource_type !== "*" && <span className="text-xs text-[var(--color-text-tertiary)] ml-2">on {a.resource_type}/{a.resource_id}</span>}
                      </div>
                      <button onClick={() => iam.deleteAssignment(a.id)} className="text-xs text-[var(--color-error)] hover:text-[var(--color-error)]">Remove</button>
                    </div>
                  );
                })}
              </div>
            </div>
          )}

          {/* Approvals */}
          {tab === "approvals" && (
            <div>
              <div className="flex justify-between items-center mb-4">
                <h2 className="text-headline-3 text-[var(--color-text-primary)]">Approval Workflows & Requests</h2>
                <button onClick={() => setShowModal(true)} className="bg-[var(--color-surface)] hover:bg-[var(--color-surface-hover)] text-[var(--color-text-primary)] px-3 py-1.5 rounded-lg text-xs transition-colors">Create Workflow</button>
              </div>
              <div className="space-y-2 mb-6">
                {iam.workflows.map((w) => (
                  <div key={w.id} className="flex items-center justify-between bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-lg px-4 py-3">
                    <div>
                      <span className="text-sm text-[var(--color-text-primary)] font-medium">{w.name}</span>
                      <span className="text-xs text-[var(--color-text-tertiary)] ml-2">{w.required_approvals} approval(s) · {w.action_pattern}</span>
                    </div>
                    <div className="flex items-center gap-3">
                      <span className={`text-[10px] px-2 py-0.5 rounded ${w.enabled ? "bg-[var(--color-success-tint)] text-[var(--color-success)]" : "bg-[var(--color-surface)] text-[var(--color-text-tertiary)]"}`}>
                        {w.enabled ? "enabled" : "disabled"}
                      </span>
                      <button onClick={() => iam.deleteWorkflow(w.id)} className="text-xs text-[var(--color-error)] hover:text-[var(--color-error)]">Delete</button>
                    </div>
                  </div>
                ))}
              </div>
              <h3 className="text-sm font-semibold text-[var(--color-text-secondary)] mb-2">Pending Requests</h3>
              <div className="space-y-2">
                {iam.requests.filter((r) => r.status === "pending").length === 0 ? (
                  <p className="text-[var(--color-text-tertiary)] text-sm">No pending requests</p>
                ) : (
                  iam.requests.filter((r) => r.status === "pending").map((req) => (
                    <div key={req.id} className="flex items-center justify-between bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-lg px-4 py-3">
                      <div>
                        <span className="text-sm text-[var(--color-text-primary)]">{req.action_type}</span>
                        <span className="text-xs text-[var(--color-text-tertiary)] ml-2">by {req.requested_by.slice(0, 8)}...</span>
                        <span className="text-xs text-[var(--color-text-tertiary)] ml-2">{req.approval_count}/{req.required_count} approvals</span>
                      </div>
                      <div className="flex gap-2">
                        <button onClick={() => iam.approveRequest(req.id)} className="text-xs bg-[var(--color-success-tint)] hover:bg-[var(--color-success-tint)] text-[var(--color-success)] px-3 py-1 rounded transition-colors">Approve</button>
                        <button onClick={() => iam.rejectRequest(req.id)} className="text-xs bg-[var(--color-error-tint)] hover:bg-[var(--color-error-tint)] text-[var(--color-error)] px-3 py-1 rounded transition-colors">Reject</button>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>
          )}

          {/* Audit Log */}
          {tab === "audit" && (
            <div>
              <h2 className="text-headline-3 text-[var(--color-text-primary)] mb-4">IAM Audit Log</h2>
              {iam.auditLog.length === 0 ? (
                <p className="text-[var(--color-text-tertiary)] text-sm">No audit entries</p>
              ) : (
                <div className="space-y-1">
                  {iam.auditLog.map((e) => (
                    <div key={e.id} className="flex items-center gap-3 bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded px-3 py-2 text-xs">
                      <span className="text-[var(--color-text-tertiary)] w-36 shrink-0">{new Date(e.created_at).toLocaleString()}</span>
                      <span className="text-[var(--color-text-primary)] font-medium w-48 shrink-0">{e.action}</span>
                      <span className="text-[var(--color-text-secondary)] font-mono">{e.resource_type}/{e.resource_id}</span>
                      <span className="text-[var(--color-text-tertiary)] ml-auto">actor: {e.actor_id.slice(0, 8)}...</span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}
        </>
      )}
    </div>
  );
}
