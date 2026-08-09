"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import { usePolicy } from "@/hooks/use-policies";
import { useAuth } from "@/hooks/use-auth";
import { EmptyState } from "@/components/ui/empty-state";

const actionColors: Record<string, string> = {
  deny: "var(--color-error)",
  warn: "var(--color-warning)",
  allow: "var(--color-success)",
};

export default function PolicyDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const queryClient = useQueryClient();

  const { data, isLoading, error } = usePolicy(id);
  const { can } = useAuth();

  const [showDisableConfirm, setShowDisableConfirm] = useState(false);
  const [showEnableConfirm, setShowEnableConfirm] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);

  const policy = data?.policy ?? null;

  const enableMutation = useMutation({
    mutationFn: () => api.enablePolicy(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["policy", id] });
      setShowEnableConfirm(false);
    },
  });

  const disableMutation = useMutation({
    mutationFn: () => api.disablePolicy(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["policy", id] });
      setShowDisableConfirm(false);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: () => api.deletePolicy(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["policies"] });
      router.push("/policies");
    },
  });

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="skeleton h-8 w-48" />
        <div className="skeleton h-48" />
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-6">
        <button onClick={() => router.back()} className="btn-ghost text-body-small">&larr; Back</button>
        <div className="card text-center py-8">
          <p className="text-body text-[var(--color-error)]">Failed to load policy</p>
        </div>
      </div>
    );
  }

  if (!policy) {
    return (
      <EmptyState
        title="Policy not found"
        description="The policy you are looking for does not exist."
        action={{ label: "Back to Policies", onClick: () => router.push("/policies") }}
      />
    );
  }

  return (
    <div className="space-y-6 max-w-3xl">
      <div>
        <Link href="/policies" className="btn-ghost text-body-small">&larr; All Policies</Link>
      </div>

      <div className="flex items-center justify-between">
        <div>
          <div className="flex items-center gap-3">
            <span
              className={`w-3 h-3 rounded-full`}
              style={{ backgroundColor: policy.enabled ? "var(--color-success)" : "var(--color-text-tertiary)" }}
            />
            <h1 className="text-headline-1 text-[var(--color-text-primary)]">
              {policy.name}
            </h1>
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
          <p className="text-body text-[var(--color-text-secondary)] mt-1">
            {policy.type === "*" ? "All types" : policy.type.replace(/_/g, " ")}
            {" \u00B7 "}Priority {policy.priority}
            {" \u00B7 "}Created {new Date(policy.created_at).toLocaleString()}
          </p>
        </div>
        <div className="flex gap-2">
          {can("manage_policies") && (
            <>
              {policy.enabled ? (
                <>
                  {!showDisableConfirm ? (
                    <button
                      onClick={() => setShowDisableConfirm(true)}
                      className="btn-ghost text-body-small px-3 py-1.5 text-[var(--color-warning)] border border-[var(--color-warning)]"
                    >
                      Disable
                    </button>
                  ) : (
                    <div className="flex items-center gap-2">
                      <span className="text-caption text-[var(--color-warning)]">Disable?</span>
                      <button
                        onClick={() => disableMutation.mutate()}
                        disabled={disableMutation.isPending}
                        className="btn-ghost text-body-small px-2 py-1 text-[var(--color-warning)] border border-[var(--color-warning)]"
                      >
                        {disableMutation.isPending ? "..." : "Confirm"}
                      </button>
                      <button
                        onClick={() => setShowDisableConfirm(false)}
                        className="btn-ghost text-body-small px-2 py-1"
                      >
                        Cancel
                      </button>
                    </div>
                  )}
                </>
              ) : (
                <>
                  {!showEnableConfirm ? (
                    <button
                      onClick={() => setShowEnableConfirm(true)}
                      className="btn-primary text-body-small px-3 py-1.5"
                    >
                      Enable
                    </button>
                  ) : (
                    <div className="flex items-center gap-2">
                      <span className="text-caption text-[var(--color-success)]">Enable?</span>
                      <button
                        onClick={() => enableMutation.mutate()}
                        disabled={enableMutation.isPending}
                        className="btn-ghost text-body-small px-2 py-1 text-[var(--color-success)] border border-[var(--color-success)]"
                      >
                        {enableMutation.isPending ? "..." : "Confirm"}
                      </button>
                      <button
                        onClick={() => setShowEnableConfirm(false)}
                        className="btn-ghost text-body-small px-2 py-1"
                      >
                        Cancel
                      </button>
                    </div>
                  )}
                </>
              )}

              {!showDeleteConfirm ? (
                <button
                  onClick={() => setShowDeleteConfirm(true)}
                  className="btn-ghost text-body-small px-3 py-1.5 text-[var(--color-error)] border border-[var(--color-error)]"
                >
                  Delete
                </button>
              ) : (
                <div className="flex items-center gap-2">
                  <span className="text-caption text-[var(--color-error)]">Delete permanently?</span>
                  <button
                    onClick={() => deleteMutation.mutate()}
                    disabled={deleteMutation.isPending}
                    className="btn-ghost text-body-small px-2 py-1 text-[var(--color-error)] border border-[var(--color-error)]"
                  >
                    {deleteMutation.isPending ? "..." : "Confirm"}
                  </button>
                  <button
                    onClick={() => setShowDeleteConfirm(false)}
                    className="btn-ghost text-body-small px-2 py-1"
                  >
                    Cancel
                  </button>
                </div>
              )}
            </>
          )}
        </div>
      </div>

      <div className="card p-4 space-y-3">
        <h2 className="text-headline-4 text-[var(--color-text-primary)]">Details</h2>
        <dl className="grid grid-cols-2 gap-y-3 gap-x-4">
          {[
            ["Policy ID", policy.id],
            ["Name", policy.name],
            ["Description", policy.description || "N/A"],
            ["Type", policy.type === "*" ? "All types" : policy.type.replace(/_/g, " ")],
            ["Action", policy.action.toUpperCase()],
            ["Priority", String(policy.priority)],
            ["Status", policy.enabled ? "Enabled" : "Disabled"],
            ["Created", new Date(policy.created_at).toLocaleString()],
            ["Updated", new Date(policy.updated_at).toLocaleString()],
          ].map(([label, value]) => (
            <div key={label as string}>
              <dt className="text-caption text-[var(--color-text-tertiary)]">{label as string}</dt>
              <dd className="text-body-small text-[var(--color-text-primary)] font-medium break-all">{value as string}</dd>
            </div>
          ))}
        </dl>
      </div>

      <div className="card p-4 space-y-3">
        <h2 className="text-headline-4 text-[var(--color-text-primary)]">Conditions</h2>
        <pre className="text-body-small font-mono text-[var(--color-text-secondary)] bg-[var(--color-surface)] rounded-sm p-3 overflow-x-auto whitespace-pre-wrap">
          {JSON.stringify(policy.conditions, null, 2)}
        </pre>
      </div>
    </div>
  );
}
