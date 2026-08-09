"use client";

import type { SuggestedAction } from "@/lib/types";

const riskColors: Record<string, string> = {
  low: "var(--color-success)",
  medium: "var(--color-warning)",
  high: "var(--color-error)",
};

export function SuggestedActions({ actions }: { actions: SuggestedAction[] }) {
  if (actions.length === 0) return null;

  return (
    <div className="space-y-2 mt-3">
      <p className="text-[11px] font-medium text-[var(--color-text-tertiary)] uppercase tracking-wide">
        Suggested Actions
      </p>
      {actions.map((action) => (
        <SuggestedActionCard key={action.id} action={action} />
      ))}
    </div>
  );
}

function SuggestedActionCard({ action }: { action: SuggestedAction }) {
  const riskColor = riskColors[action.risk_level?.toLowerCase()] || "var(--color-text-tertiary)";

  return (
    <div className="border border-[var(--color-border)] rounded-md p-3 bg-[var(--color-surface)]">
      <div className="flex items-center justify-between mb-1">
        <span className="text-body-small font-medium text-[var(--color-text-primary)]">
          {action.action}
        </span>
        <span
          className="text-[10px] px-1.5 py-0.5 rounded-sm font-medium"
          style={{ backgroundColor: riskColor + "20", color: riskColor }}
        >
          {action.risk_level}
        </span>
      </div>
      {action.target && (
        <p className="text-[12px] text-[var(--color-text-secondary)] mb-1">
          Target: <span className="font-mono">{action.target}</span>
        </p>
      )}
      <p className="text-[12px] text-[var(--color-text-tertiary)] mb-2">
        {action.description}
      </p>
      {action.command && (
        <div className="bg-[var(--color-bg)] border border-[var(--color-border)] rounded p-2 mb-2">
          <code className="text-[12px] text-[var(--color-accent)] break-all">{action.command}</code>
        </div>
      )}
      <div className="flex gap-2">
        <button className="text-[12px] px-3 py-1 bg-[var(--color-success)] text-white rounded-sm hover:opacity-90 transition-opacity">
          Approve
        </button>
        <button className="text-[12px] px-3 py-1 bg-[var(--color-surface-elevated)] text-[var(--color-text-secondary)] border border-[var(--color-border)] rounded-sm hover:bg-[var(--color-surface-hover)] transition-colors">
          Dismiss
        </button>
      </div>
    </div>
  );
}
