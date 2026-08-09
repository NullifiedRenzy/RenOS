"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useAuth } from "@/hooks/use-auth";
import { api, ApiRequestError } from "@/lib/api-client";

const policyTypes = [
  { value: "*", label: "All Types" },
  { value: "command", label: "Command" },
  { value: "container_action", label: "Container Action" },
  { value: "service_action", label: "Service Action" },
  { value: "update", label: "Update" },
  { value: "schedule", label: "Schedule" },
  { value: "job", label: "Job" },
];

const policyActions = [
  { value: "deny", label: "Deny" },
  { value: "warn", label: "Warn" },
  { value: "allow", label: "Allow" },
];

const conditionOptions = [
  { key: "max_devices", label: "Max Devices", type: "number" as const, placeholder: "e.g. 20" },
  { key: "requires_confirmation", label: "Requires Confirmation", type: "checkbox" as const },
  { key: "restrict_cron_frequency", label: "Min Cron Interval (min)", type: "number" as const, placeholder: "e.g. 5" },
  { key: "allowed_commands", label: "Allowed Commands (comma-separated)", type: "text" as const, placeholder: "e.g. apt update, systemctl status" },
  { key: "restricted_targets", label: "Restricted Targets (comma-separated)", type: "text" as const, placeholder: "e.g. prod-db-01" },
];

export default function NewPolicyPage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { can } = useAuth();

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [type, setType] = useState("*");
  const [action, setAction] = useState("deny");
  const [priority, setPriority] = useState(0);
  const [conditions, setConditions] = useState<Record<string, unknown>>({});
  const [error, setError] = useState("");

  const createMutation = useMutation({
    mutationFn: (data: Parameters<typeof api.createPolicy>[0]) => api.createPolicy(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["policies"] });
      router.push("/policies");
    },
    onError: (err: Error) => {
      setError(err instanceof ApiRequestError ? err.message : "Failed to create policy");
    },
  });

  const handleConditionChange = (key: string, value: unknown) => {
    setConditions((prev) => ({ ...prev, [key]: value }));
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    if (!name.trim()) { setError("Name is required"); return; }

    const cleaned: Record<string, unknown> = {};
    for (const [k, v] of Object.entries(conditions)) {
      if (v === "" || v === undefined || v === null) continue;
      if (typeof v === "number" && v === 0) continue;
      if (typeof v === "string" && v.trim() === "") continue;
      if (k === "allowed_commands" || k === "restricted_targets") {
        if (typeof v === "string" && v.trim()) {
          cleaned[k] = v.split(",").map((s: string) => s.trim()).filter(Boolean);
        }
      } else {
        cleaned[k] = v;
      }
    }

    createMutation.mutate({
      name: name.trim(),
      description: description.trim(),
      type,
      conditions: cleaned,
      action,
      priority,
    });
  };

  if (!can("manage_policies")) {
    return (
      <div className="space-y-6 max-w-2xl">
        <div>
          <Link href="/policies" className="btn-ghost text-body-small">&larr; All Policies</Link>
        </div>
        <div className="card text-center py-8">
          <p className="text-body text-[var(--color-error)]">Insufficient permissions. Admin role required to create policies.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6 max-w-2xl">
      <div>
        <Link href="/policies" className="btn-ghost text-body-small">&larr; All Policies</Link>
      </div>

      <div>
        <h1 className="text-headline-1 text-[var(--color-text-primary)]">New Policy</h1>
        <p className="text-body text-[var(--color-text-secondary)] mt-1">
          Define a system-level execution guardrail
        </p>
      </div>

      <form onSubmit={handleSubmit} className="card p-6 space-y-5">
        {error && (
          <div className="text-body-small text-[var(--color-error)] bg-[var(--color-error)]/10 p-3 rounded-sm">
            {error}
          </div>
        )}

        <div className="space-y-1">
          <label className="text-body-small font-medium text-[var(--color-text-primary)]">Name</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="input w-full"
            placeholder="e.g. Max devices per action"
            maxLength={255}
          />
        </div>

        <div className="space-y-1">
          <label className="text-body-small font-medium text-[var(--color-text-primary)]">Description</label>
          <input
            type="text"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className="input w-full"
            placeholder="e.g. Limit any action to at most 20 devices"
            maxLength={500}
          />
        </div>

        <div className="grid grid-cols-3 gap-4">
          <div className="space-y-1">
            <label className="text-body-small font-medium text-[var(--color-text-primary)]">Type</label>
            <select
              value={type}
              onChange={(e) => setType(e.target.value)}
              className="input w-full"
            >
              {policyTypes.map((t) => (
                <option key={t.value} value={t.value}>{t.label}</option>
              ))}
            </select>
          </div>

          <div className="space-y-1">
            <label className="text-body-small font-medium text-[var(--color-text-primary)]">Action</label>
            <select
              value={action}
              onChange={(e) => setAction(e.target.value)}
              className="input w-full"
            >
              {policyActions.map((a) => (
                <option key={a.value} value={a.value}>{a.label}</option>
              ))}
            </select>
          </div>

          <div className="space-y-1">
            <label className="text-body-small font-medium text-[var(--color-text-primary)]">Priority</label>
            <input
              type="number"
              value={priority}
              onChange={(e) => setPriority(parseInt(e.target.value) || 0)}
              className="input w-full"
              placeholder="100"
            />
          </div>
        </div>

        <div className="space-y-3">
          <h3 className="text-headline-4 text-[var(--color-text-primary)]">Conditions</h3>
          <p className="text-caption text-[var(--color-text-tertiary)]">
            Configure conditions that determine when this policy applies.
          </p>
          {conditionOptions.map((opt) => (
            <div key={opt.key} className="space-y-1">
              <label className="text-body-small font-medium text-[var(--color-text-primary)]">
                {opt.label}
              </label>
              {opt.type === "checkbox" ? (
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={!!conditions[opt.key]}
                    onChange={(e) => handleConditionChange(opt.key, e.target.checked)}
                    className="w-4 h-4"
                  />
                  <span className="text-body-small text-[var(--color-text-secondary)]">
                    Requires user confirmation before executing
                  </span>
                </label>
              ) : (
                <input
                  type={opt.type}
                  value={(conditions[opt.key] as string) ?? ""}
                  onChange={(e) => {
                    const val = opt.type === "number" ? (parseInt(e.target.value) || 0) : e.target.value;
                    handleConditionChange(opt.key, val);
                  }}
                  className="input w-full font-mono"
                  placeholder={opt.placeholder}
                />
              )}
            </div>
          ))}
        </div>

        <div className="flex gap-3 pt-2">
          <button
            type="submit"
            disabled={createMutation.isPending}
            className="btn-primary text-body-small px-4 py-2"
          >
            {createMutation.isPending ? "Creating..." : "Create Policy"}
          </button>
          <Link href="/policies" className="btn-ghost text-body-small px-4 py-2">
            Cancel
          </Link>
        </div>
      </form>
    </div>
  );
}
