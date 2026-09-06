"use client";

import { useMemo, useState } from "react";
import { Bell, CheckCheck, Search } from "lucide-react";
import {
  useInAppNotifications,
  useInAppUnreadCount,
  useMarkInAppRead,
  useMarkAllInAppRead,
} from "@/hooks/use-inapp-notifications";
import type { InAppNotification } from "@/lib/types";
import { cn, formatDate } from "@/lib/utils";
import { LoadingState } from "@/components/ui/loading-state";
import { ErrorState } from "@/components/ui/error-state";

type InboxFilter =
  "all" | "unread" | "alert" | "workflow" | "job" | "ai" | "system";

const FILTERS: { id: InboxFilter; label: string }[] = [
  { id: "all", label: "All" },
  { id: "unread", label: "Unread" },
  { id: "alert", label: "Alerts" },
  { id: "workflow", label: "Workflow" },
  { id: "job", label: "Jobs" },
  { id: "ai", label: "AI" },
  { id: "system", label: "System" },
];

const TYPE_STYLES: Record<string, string> = {
  alert: "text-[var(--color-error)] bg-[var(--color-error)]/10",
  workflow: "text-[var(--color-accent)] bg-[var(--color-accent)]/10",
  job: "text-[var(--color-info)] bg-[var(--color-info)]/10",
  ai: "text-[var(--color-warning)] bg-[var(--color-warning)]/10",
  system: "text-[var(--color-text-secondary)] bg-[var(--color-surface-hover)]",
  mention: "text-[var(--color-accent)] bg-[var(--color-accent)]/10",
};

const TYPE_LABELS: Record<string, string> = {
  alert: "Alert",
  workflow: "Workflow",
  job: "Job",
  ai: "AI",
  system: "System",
  mention: "Mention",
};

export function InboxTab() {
  const [filter, setFilter] = useState<InboxFilter>("all");
  const [search, setSearch] = useState("");

  const {
    data: notifications,
    isLoading,
    error,
  } = useInAppNotifications({
    unread: filter === "unread" || undefined,
    type: ["alert", "workflow", "job", "ai", "system"].includes(filter)
      ? filter
      : undefined,
    search: search.trim() || undefined,
    limit: 100,
  });
  const { data: unread } = useInAppUnreadCount();
  const markRead = useMarkInAppRead();
  const markAll = useMarkAllInAppRead();

  const unreadCount = unread?.count ?? 0;
  const list = useMemo(() => notifications ?? [], [notifications]);

  const summary = useMemo(() => {
    const byType = new Map<string, number>();
    for (const n of list) byType.set(n.type, (byType.get(n.type) ?? 0) + 1);
    return Object.fromEntries(byType);
  }, [list]);

  return (
    <div className="space-y-4">
      <div className="flex flex-col sm:flex-row gap-3 sm:items-center sm:justify-between">
        <div className="flex flex-wrap gap-1.5">
          {FILTERS.map((f) => (
            <button
              key={f.id}
              onClick={() => setFilter(f.id)}
              aria-pressed={filter === f.id}
              className={cn(
                "px-3 py-1.5 rounded-md text-caption border transition-colors",
                filter === f.id
                  ? "bg-[var(--color-accent)]/10 text-[var(--color-accent)] border-[var(--color-accent)]/30 font-medium"
                  : "text-[var(--color-text-secondary)] border-[var(--color-border)] hover:text-[var(--color-text-primary)]",
              )}
            >
              {f.label}
              {f.id === "unread" && unreadCount > 0 && (
                <span className="ml-1.5 px-1.5 py-0.5 rounded-full bg-[var(--color-error)] text-white text-[10px] font-semibold">
                  {unreadCount}
                </span>
              )}
            </button>
          ))}
        </div>

        <div className="flex items-center gap-2">
          <label className="relative flex-1 sm:flex-none">
            <span className="sr-only">Search notifications</span>
            <Search
              size={14}
              className="absolute left-2.5 top-1/2 -translate-y-1/2 text-[var(--color-text-tertiary)]"
            />
            <input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search notifications…"
              className="w-full sm:w-56 h-9 pl-8 pr-3 rounded-md border border-[var(--color-border)] bg-[var(--color-surface)] text-caption text-[var(--color-text-primary)] outline-none focus:border-[var(--color-accent)]"
            />
          </label>
          {unreadCount > 0 && (
            <button
              onClick={() => markAll.mutate()}
              className="btn-ghost text-caption flex items-center gap-1.5"
            >
              <CheckCheck size={14} aria-hidden="true" />
              Mark all read
            </button>
          )}
        </div>
      </div>

      {Object.keys(summary).length > 0 && (
        <p className="text-caption text-[var(--color-text-tertiary)]">
          {list.length} shown ·{" "}
          {Object.entries(summary)
            .map(([t, c]) => `${TYPE_LABELS[t] ?? t}: ${c}`)
            .join(" · ")}
        </p>
      )}

      {isLoading ? (
        <LoadingState message="Loading notifications…" />
      ) : error ? (
        <ErrorState error={error as Error} />
      ) : list.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <Bell size={28} className="mb-3 text-[var(--color-text-tertiary)]" />
          <p className="text-body text-[var(--color-text-primary)]">
            No notifications
          </p>
          <p className="text-caption text-[var(--color-text-tertiary)] mt-1">
            {filter === "all" && !search
              ? "You'll see alerts, workflow and job updates here."
              : "Try a different filter or search."}
          </p>
        </div>
      ) : (
        <ul className="space-y-2">
          {list.map((n: InAppNotification) => (
            <li
              key={n.id}
              className={cn(
                "flex items-start gap-3 p-3.5 rounded-lg border border-[var(--color-border)]",
                !n.is_read &&
                  "border-[var(--color-accent)]/30 bg-[var(--color-accent)]/5",
              )}
            >
              <span
                className={cn(
                  "mt-0.5 shrink-0 px-2 py-0.5 rounded text-[10px] font-semibold uppercase tracking-wide",
                  TYPE_STYLES[n.type] ?? TYPE_STYLES.system,
                )}
              >
                {TYPE_LABELS[n.type] ?? n.type}
              </span>
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <h3 className="text-body font-medium text-[var(--color-text-primary)]">
                    {n.title}
                  </h3>
                  {!n.is_read && (
                    <span
                      className="w-2 h-2 rounded-full bg-[var(--color-accent)] shrink-0"
                      aria-label="Unread"
                    />
                  )}
                </div>
                {n.body && (
                  <p className="text-caption text-[var(--color-text-secondary)] mt-0.5">
                    {n.body}
                  </p>
                )}
                <p className="text-caption text-[var(--color-text-tertiary)] mt-1">
                  {formatDate(n.created_at)}
                </p>
              </div>
              {!n.is_read && (
                <button
                  onClick={() => markRead.mutate(n.id)}
                  aria-label={`Mark "${n.title}" as read`}
                  className="shrink-0 btn-ghost p-1.5 rounded-md text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)]"
                >
                  <CheckCheck size={15} />
                </button>
              )}
            </li>
          ))}
        </ul>
      )}

      <span className="sr-only" aria-live="polite">
        {unreadCount} unread notifications
      </span>
    </div>
  );
}
