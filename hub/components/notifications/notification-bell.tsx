"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { Bell, CheckCheck, ExternalLink } from "lucide-react";
import { useInAppUnreadCount, useInAppNotifications, useMarkAllInAppRead } from "@/hooks/use-inapp-notifications";
import type { InAppNotification } from "@/lib/types";
import { useLocale } from "@/lib/i18n/use-locale";
import { cn, formatDate } from "@/lib/utils";

const TYPE_STYLES: Record<string, string> = {
  alert: "text-[var(--color-error)] bg-[var(--color-error)]/10",
  workflow: "text-[var(--color-accent)] bg-[var(--color-accent)]/10",
  job: "text-[var(--color-info)] bg-[var(--color-info)]/10",
  ai: "text-[var(--color-warning)] bg-[var(--color-warning)]/10",
  system: "text-[var(--color-text-secondary)] bg-[var(--color-surface-hover)]",
  mention: "text-[var(--color-accent)] bg-[var(--color-accent)]/10",
};

export function NotificationBell() {
  const router = useRouter();
  const { t } = useLocale();
  const [open, setOpen] = useState(false);
  const [selected, setSelected] = useState(0);
  const panelRef = useRef<HTMLDivElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);

  const { data: unread } = useInAppUnreadCount();
  const { data: notifications = [] } = useInAppNotifications({ limit: 20 });
  const markAll = useMarkAllInAppRead();

  const unreadCount = unread?.count ?? 0;

  useEffect(() => {
    if (!open) return;
    setSelected(0);
    const onClick = (e: MouseEvent) => {
      if (
        panelRef.current &&
        !panelRef.current.contains(e.target as Node) &&
        buttonRef.current &&
        !buttonRef.current.contains(e.target as Node)
      ) {
        setOpen(false);
      }
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    document.addEventListener("mousedown", onClick);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onClick);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  const onKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setSelected((s) => Math.min(s + 1, notifications.length - 1));
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        setSelected((s) => Math.max(s - 1, 0));
      } else if (e.key === "Enter") {
        e.preventDefault();
        const n = notifications[selected];
        if (n) {
          setOpen(false);
          router.push("/notifications");
        }
      }
    },
    [notifications, selected, router],
  );

  return (
    <div className="relative">
      <button
        ref={buttonRef}
        onClick={() => setOpen((o) => !o)}
        aria-label={unreadCount > 0 ? t("notifications.ariaOpen", { count: unreadCount }) : t("notifications.title")}
        aria-expanded={open}
        aria-haspopup="true"
        className="relative btn-ghost p-1.5 rounded-md text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]"
      >
        <Bell size={18} />
        {unreadCount > 0 && (
          <span className="absolute -top-0.5 -right-0.5 min-w-[16px] h-4 px-1 rounded-full bg-[var(--color-error)] text-[10px] font-semibold text-white flex items-center justify-center">
            {unreadCount > 99 ? "99+" : unreadCount}
          </span>
        )}
      </button>

      {open && (
        <div
          ref={panelRef}
          role="dialog"
          aria-label={t("notifications.title")}
          onKeyDown={onKeyDown}
          className="absolute right-0 top-9 w-80 max-w-[calc(100vw-2rem)] rounded-lg border border-[var(--color-border)] bg-[var(--color-surface)] shadow-xl overflow-hidden"
          style={{ animation: "fade-in-up 120ms ease-out" }}
        >
          <div className="flex items-center justify-between px-3 py-2 border-b border-[var(--color-border)]">
            <h2 className="text-body font-semibold text-[var(--color-text-primary)]">{t("notifications.title")}</h2>
            <div className="flex items-center gap-1">
              {unreadCount > 0 && (
                <button
                  onClick={() => markAll.mutate()}
                  className="flex items-center gap-1 px-2 py-1 rounded-md text-caption text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-hover)] hover:text-[var(--color-text-primary)]"
                >
                  <CheckCheck size={13} aria-hidden="true" />
                  {t("notifications.markAllRead")}
                </button>
              )}
              <button
                onClick={() => { setOpen(false); router.push("/notifications"); }}
                aria-label={t("notifications.viewAll")}
                className="p-1.5 rounded-md text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-hover)] hover:text-[var(--color-text-primary)]"
              >
                <ExternalLink size={14} />
              </button>
            </div>
          </div>

          <div className="max-h-96 overflow-y-auto">
            {notifications.length === 0 ? (
              <div className="px-4 py-10 text-center">
                <Bell size={22} className="mx-auto mb-2 text-[var(--color-text-tertiary)]" />
                <p className="text-caption text-[var(--color-text-tertiary)]">{t("notifications.empty")}</p>
              </div>
            ) : (
              notifications.map((n: InAppNotification, i: number) => (
                <button
                  key={n.id}
                  onMouseEnter={() => setSelected(i)}
                  onClick={() => { setOpen(false); router.push("/notifications"); }}
                  className={cn(
                    "w-full text-left px-3 py-2.5 flex items-start gap-2.5 border-b border-[var(--color-border)] last:border-0",
                    i === selected ? "bg-[var(--color-surface-hover)]" : "hover:bg-[var(--color-surface-hover)]",
                    !n.is_read && "bg-[var(--color-accent)]/5",
                  )}
                >
                  <span
                    className={cn(
                      "mt-0.5 shrink-0 px-1.5 py-0.5 rounded text-[10px] font-semibold uppercase tracking-wide",
                      TYPE_STYLES[n.type] ?? TYPE_STYLES.system,
                    )}
                  >
                    {t(`notifications.types.${n.type}`)}
                  </span>
                  <span className="min-w-0 flex-1">
                    <span className="flex items-center gap-1.5">
                      <span className="text-body font-medium text-[var(--color-text-primary)] truncate">{n.title}</span>
                      {!n.is_read && (
                        <span className="w-2 h-2 rounded-full bg-[var(--color-accent)] shrink-0" aria-label={t("notifications.unread")} />
                      )}
                    </span>
                    <span className="block text-caption text-[var(--color-text-tertiary)] line-clamp-2">{n.body}</span>
                    <span className="block mt-0.5 text-caption text-[var(--color-text-tertiary)] opacity-70">
                      {formatDate(n.created_at)}
                    </span>
                  </span>
                </button>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  );
}
