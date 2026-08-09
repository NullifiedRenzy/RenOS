"use client";

import { useState } from "react";
import { useAuth } from "@/hooks/use-auth";
import {
  useVaultSecrets,
  useVaultSecretVersions,
  useCreateVaultSecret,
  useDeleteVaultSecret,
  useVaultKeys,
  useVaultAuditLog,
} from "@/hooks/use-vault";
import type { VaultSecret, VaultSecretVersion, VaultKey, VaultAuditEntry } from "@/lib/types";

type VaultTab = "secrets" | "keys" | "audit";

const typeIcons: Record<string, string> = {
  generic: "\u{1F512}",
  password: "\u{1F4DD}",
  api_key: "\u{1F511}",
  certificate: "\u{1F4DC}",
  ssh_key: "\u{1F6E1}",
  token: "\u{1F3B2}",
};

export default function VaultPage() {
  const { can } = useAuth();
  const [activeTab, setActiveTab] = useState<VaultTab>("secrets");

  return (
    <div className="space-y-6">
      <h1 className="text-headline-1 text-[var(--color-text-primary)]">RenVault</h1>

      <div className="flex gap-2 border-b border-[var(--color-border)] pb-0">
        {(["secrets", "keys", "audit"] as VaultTab[]).map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-4 py-2 text-body-small rounded-t-lg border border-b-0 transition-colors capitalize ${
              activeTab === tab
                ? "bg-[var(--color-bg-primary)] text-[var(--color-text-primary)] border-[var(--color-border)] font-medium"
                : "bg-transparent text-[var(--color-text-tertiary)] border-transparent hover:text-[var(--color-text-secondary)]"
            }`}
          >
            {tab}
          </button>
        ))}
      </div>

      <div className="mt-6">
        {activeTab === "secrets" && <SecretsTab />}
        {activeTab === "keys" && <KeysTab />}
        {activeTab === "audit" && <AuditTab />}
      </div>
    </div>
  );
}

function SecretsTab() {
  const { can } = useAuth();
  const { data: secrets, isLoading } = useVaultSecrets();
  const deleteSecret = useDeleteVaultSecret();
  const [search, setSearch] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const [selectedSecret, setSelectedSecret] = useState<string | null>(null);
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null);

  const filtered = search
    ? (secrets ?? []).filter((s: VaultSecret) => s.name.toLowerCase().includes(search.toLowerCase()) || s.path.toLowerCase().includes(search.toLowerCase()))
    : secrets ?? [];

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3">
        <input
          type="text"
          placeholder="Search secrets..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="flex-1 px-3 py-2 bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg text-body-small text-[var(--color-text-primary)]"
        />
        {can("operator") && (
          <button onClick={() => setShowCreate(!showCreate)} className="px-4 py-2 bg-[var(--color-accent)] text-white rounded-lg text-body-small hover:opacity-90">
            {showCreate ? "Cancel" : "+ New Secret"}
          </button>
        )}
      </div>

      {showCreate && <CreateSecretForm onClose={() => setShowCreate(false)} />}

      {isLoading ? (
        <Loading message="Loading secrets..." />
      ) : filtered.length === 0 ? (
        <EmptyState message="No secrets found" />
      ) : (
        <div className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg overflow-hidden">
          <table className="w-full">
            <thead>
              <tr className="border-b border-[var(--color-border)]">
                <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Name</th>
                <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Path</th>
                <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Type</th>
                <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Version</th>
                <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Updated</th>
                <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Actions</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((s: VaultSecret) => (
                <tr key={s.id} className="border-b border-[var(--color-border)] last:border-0 hover:bg-[var(--color-bg-tertiary)]">
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <span>{typeIcons[s.secret_type] || "\u{1F512}"}</span>
                      <span className="text-body-small text-[var(--color-text-primary)] font-medium">{s.name}</span>
                    </div>
                  </td>
                  <td className="px-4 py-3 text-body-small text-[var(--color-text-secondary)] font-mono">{s.path}</td>
                  <td className="px-4 py-3 text-caption text-[var(--color-text-tertiary)]">{s.secret_type}</td>
                  <td className="px-4 py-3 text-caption text-[var(--color-text-tertiary)]">v{s.version}</td>
                  <td className="px-4 py-3 text-caption text-[var(--color-text-tertiary)]">{new Date(s.updated_at).toLocaleString()}</td>
                  <td className="px-4 py-3">
                    <div className="flex gap-2">
                      <button onClick={() => setSelectedSecret(selectedSecret === s.id ? null : s.id)} className="text-caption text-[var(--color-accent)] hover:underline">
                        {selectedSecret === s.id ? "Hide" : "Versions"}
                      </button>
                      {confirmDelete === s.id ? (
                        <div className="flex gap-2">
                          <button onClick={() => deleteSecret.mutate(s.id, { onSuccess: () => { setConfirmDelete(null); setSelectedSecret(null); } })} className="text-caption text-[var(--color-error)] hover:underline">Confirm</button>
                          <button onClick={() => setConfirmDelete(null)} className="text-caption text-[var(--color-text-tertiary)] hover:underline">Cancel</button>
                        </div>
                      ) : (
                        <button onClick={() => setConfirmDelete(s.id)} className="text-caption text-[var(--color-text-tertiary)] hover:text-[var(--color-error)]">Delete</button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {selectedSecret && <VersionHistory secretId={selectedSecret} />}
    </div>
  );
}

function VersionHistory({ secretId }: { secretId: string }) {
  const { data: versions } = useVaultSecretVersions(secretId);

  return (
    <div className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg p-6">
      <h3 className="text-headline-3 text-[var(--color-text-primary)] mb-4">Version History</h3>
      {!versions || versions.length === 0 ? (
        <p className="text-body-small text-[var(--color-text-tertiary)]">No version history</p>
      ) : (
        <div className="space-y-2">
          {versions.map((v: VaultSecretVersion) => (
            <div key={v.id} className="flex items-center justify-between p-3 bg-[var(--color-bg-tertiary)] rounded-lg">
              <div className="flex items-center gap-3">
                <span className="text-body-small text-[var(--color-text-primary)] font-medium">v{v.version}</span>
                <span className="text-caption text-[var(--color-text-tertiary)]">{new Date(v.created_at).toLocaleString()}</span>
                {v.changed_by && <span className="text-caption text-[var(--color-text-tertiary)]">by {v.changed_by.slice(0, 8)}</span>}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function CreateSecretForm({ onClose }: { onClose: () => void }) {
  const createSecret = useCreateVaultSecret();
  const [name, setName] = useState("");
  const [path, setPath] = useState("");
  const [value, setValue] = useState("");
  const [description, setDescription] = useState("");
  const [secretType, setSecretType] = useState("generic");

  const handleSubmit = () => {
    if (!name || !path || !value) return;
    createSecret.mutate(
      { name, path, value, description, secret_type: secretType },
      { onSuccess: onClose }
    );
  };

  return (
    <div className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg p-6">
      <h3 className="text-headline-3 text-[var(--color-text-primary)] mb-4">New Secret</h3>
      <div className="grid grid-cols-2 gap-4">
        <input type="text" placeholder="Name" value={name} onChange={(e) => setName(e.target.value)} className="px-3 py-2 bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-lg text-body-small text-[var(--color-text-primary)]" />
        <input type="text" placeholder="Path (e.g. /prod/db/password)" value={path} onChange={(e) => setPath(e.target.value)} className="px-3 py-2 bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-lg text-body-small text-[var(--color-text-primary)] font-mono" />
        <input type="password" placeholder="Secret value" value={value} onChange={(e) => setValue(e.target.value)} className="col-span-2 px-3 py-2 bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-lg text-body-small text-[var(--color-text-primary)]" />
        <input type="text" placeholder="Description (optional)" value={description} onChange={(e) => setDescription(e.target.value)} className="px-3 py-2 bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-lg text-body-small text-[var(--color-text-primary)]" />
        <select value={secretType} onChange={(e) => setSecretType(e.target.value)} className="px-3 py-2 bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-lg text-body-small text-[var(--color-text-primary)]">
          <option value="generic">Generic</option>
          <option value="password">Password</option>
          <option value="api_key">API Key</option>
          <option value="certificate">Certificate</option>
          <option value="ssh_key">SSH Key</option>
          <option value="token">Token</option>
        </select>
      </div>
      <div className="flex gap-2 justify-end mt-4">
        <button onClick={onClose} className="px-4 py-2 text-body-small text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]">Cancel</button>
        <button onClick={handleSubmit} disabled={!name || !path || !value || createSecret.isPending} className="px-4 py-2 bg-[var(--color-accent)] text-white rounded-lg text-body-small hover:opacity-90 disabled:opacity-50">Create</button>
      </div>
    </div>
  );
}

function KeysTab() {
  const { data: keys, isLoading } = useVaultKeys();

  if (isLoading) return <Loading message="Loading keys..." />;
  const list = keys ?? [];

  return (
    <div className="space-y-4">
      {list.length === 0 ? (
        <EmptyState message="No encryption keys" />
      ) : (
        <div className="space-y-2">
          {list.map((k: VaultKey) => (
            <div key={k.id} className="flex items-center justify-between p-4 bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg">
              <div>
                <div className="flex items-center gap-2">
                  <span className="w-2 h-2 rounded-full" style={{ backgroundColor: k.enabled ? "var(--color-success)" : "var(--color-text-tertiary)" }} />
                  <span className="text-body-small text-[var(--color-text-primary)] font-medium">{k.name}</span>
                  <span className="text-caption text-[var(--color-text-tertiary)]">{k.algorithm}</span>
                </div>
                <div className="text-caption text-[var(--color-text-tertiary)] mt-1">
                  Rotated: {new Date(k.rotated_at).toLocaleString()}
                  {k.expires_at && ` | Expires: ${new Date(k.expires_at).toLocaleString()}`}
                </div>
              </div>
              <span className="text-caption" style={{ color: k.enabled ? "var(--color-success)" : "var(--color-text-tertiary)" }}>
                {k.enabled ? "Active" : "Disabled"}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function AuditTab() {
  const { data: logs, isLoading } = useVaultAuditLog();

  if (isLoading) return <Loading message="Loading audit log..." />;

  const list = logs ?? [];
  if (list.length === 0) return <EmptyState message="No audit entries" />;

  return (
    <div className="bg-[var(--color-bg-secondary)] border border-[var(--color-border)] rounded-lg overflow-hidden">
      <table className="w-full">
        <thead>
          <tr className="border-b border-[var(--color-border)]">
            <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Action</th>
            <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Secret</th>
            <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">User</th>
            <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">IP</th>
            <th scope="col" className="px-4 py-3 text-left text-caption text-[var(--color-text-tertiary)]">Time</th>
          </tr>
        </thead>
        <tbody>
          {list.map((e: VaultAuditEntry) => (
            <tr key={e.id} className="border-b border-[var(--color-border)] last:border-0 hover:bg-[var(--color-bg-tertiary)]">
              <td className="px-4 py-3 text-body-small text-[var(--color-text-primary)]">{e.action}</td>
              <td className="px-4 py-3 text-body-small text-[var(--color-text-secondary)]">{e.secret_id?.slice(0, 8) ?? "—"}</td>
              <td className="px-4 py-3 text-body-small text-[var(--color-text-secondary)]">{e.user_id?.slice(0, 8) ?? "—"}</td>
              <td className="px-4 py-3 text-caption text-[var(--color-text-tertiary)]">{e.ip_address}</td>
              <td className="px-4 py-3 text-caption text-[var(--color-text-tertiary)]">{new Date(e.created_at).toLocaleString()}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function Loading({ message }: { message: string }) {
  return <div className="flex items-center justify-center py-12"><div className="text-body-small text-[var(--color-text-tertiary)]">{message}</div></div>;
}

function EmptyState({ message }: { message: string }) {
  return <div className="flex items-center justify-center py-12"><div className="text-body-small text-[var(--color-text-tertiary)]">{message}</div></div>;
}
