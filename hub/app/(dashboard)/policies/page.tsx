"use client";

import { useState } from "react";
import Link from "next/link";
import { usePolicies } from "@/hooks/use-policies";
import { useAuth } from "@/hooks/use-auth";
import { EmptyState } from "@/components/ui/empty-state";

const actionColors: Record<string, string> = {
  deny: "var(--color-error)",
  warn: "var(--color-warning)",
  allow: "var(--color-success)",
};

export default function PoliciesPage() {
  const { data, isLoading, error } = usePolicies();
  const { can } = useAuth();
  const [showEnabled, setShowEnabled] = useState<string>("all");

  const policies = data?.policies ?? [];
  const filtered = showEnabled === "all"
    ? policies
    : policies.filter((p) => String(p.enabled) === showEnabled);

  const counts = {
    all: policies.length,
    enabled: policies.filter((p) => p.enabled).length,
    disabled: policies.filter((p) => !p.enabled).length,
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-headline-1 text-[var(--color-text-primary)]">Policies</h1>
          <p className="text-body text-[var(--color-text-secondary)] mt-1">
            {isLoading ? "Loading..." : `${policies.length} polic${policies.length !== 1 ? "ies" : "y"}`}
          </p>
        </div>
        {can("manage_policies") && (
          <Link href="/policies/new" className="btn-primary text-body-small px-3 py-1.5">
            + New Policy
          </Link>
        )}
      </div>

      <div className="flex gap-2 flex-wrap">
        {(["all", "enabled", "disabled"] as const).map((s) => (
          <button
            key={s}
            onClick={() => setShowEnabled(s)}
            className={`text-body-small px-2.5 py-1 rounded-sm transition-colors ${
              showEnabled === s
                ? "bg-[var(--color-accent-tint)] text-[var(--color-accent)] font-medium"
                : "text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)]"
            }`}
          >
            {s.charAt(0).toUpperCase() + s.slice(1)} ({counts[s]})
          </button>
        ))}
      </div>

      {error && (
        <div className="card text-center py-8">
          <p className="text-body text-[var(--color-error)]">Failed to load policies</p>
        </div>
      )}

      {!isLoading && !error && filtered.length === 0 && (
        <EmptyState
          title="No policies found"
          description="Create a policy to enforce execution guardrails."
          action={{ label: "Create Policy", onClick: () => window.location.href = "/policies/new" }}
        />
      )}

      {filtered.length > 0 && (
        <div className="space-y-2">
          {filtered.map((policy) => (
            <Link
              key={policy.id}
              href={`/policies/${policy.id}`}
              className="card p-4 block hover:bg-[var(--color-surface)] transition-colors"
            >
              <div className="flex items-center gap-3">
                <span className={`w-2 h-2 rounded-full flex-shrink-0`}
                  style={{ backgroundColor: policy.enabled ? "var(--color-success)" : "var(--color-text-tertiary)" }}
                />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center justify-between">
                    <p className="text-body font-medium text-[var(--color-text-primary)] truncate">
                      {policy.name}
                    </p>
                    <span
                      className="text-caption font-medium px-1.5 py-0.5 rounded-sm"
                      style={{
                        backgroundColor: actionColors[policy.action] + "20",
                        color: actionColors[policy.action],
                      }}
                    >
                      {policy.action.toUpperCase()}
                    </span>
                  </div>
                  <p className="text-body-small text-[var(--color-text-tertiary)] mt-0.5">
                    {policy.type === "*" ? "All types" : policy.type.replace(/_/g, " ")}
                    {" \u00B7 "}Priority {policy.priority}
                    {policy.description ? ` \u00B7 ${policy.description}` : ""}
                  </p>
                </div>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
