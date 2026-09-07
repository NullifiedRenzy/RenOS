"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Activity, type LucideIcon } from "lucide-react";
import { useAuth } from "@/hooks/use-auth";
import { useVersion } from "@/hooks/use-version";
import { useUiStore } from "@/stores/ui-store";
import { useLocale } from "@/lib/i18n/use-locale";
import { cn } from "@/lib/utils";
import type { TranslationKey } from "@/lib/i18n/translations";

interface NavItem {
  labelKey: TranslationKey;
  href: string;
  icon: string | LucideIcon;
}

const navItems: { sectionKey: TranslationKey; items: NavItem[] }[] = [
  {
    sectionKey: "nav.main",
    items: [{ labelKey: "nav.dashboard", href: "/dashboard", icon: "\u{1F4CA}" }],
  },
  {
    sectionKey: "nav.devices",
    items: [{ labelKey: "nav.allDevices", href: "/devices", icon: "\u{1F4BB}" }],
  },
  {
    sectionKey: "nav.orchestration",
    items: [
      { labelKey: "nav.groups", href: "/groups", icon: "\u{1F465}" },
      { labelKey: "nav.jobs", href: "/jobs", icon: "\u2699\uFE0F" },
      { labelKey: "nav.schedules", href: "/schedules", icon: "\u23F0" },
    ],
  },
  {
    sectionKey: "nav.ai",
    items: [{ labelKey: "nav.aiAssistant", href: "/ai", icon: "\u{1F916}" }],
  },
  {
    sectionKey: "nav.monitoring",
    items: [
      { labelKey: "nav.observability", href: "/observability", icon: "\u{1F4C8}" },
      { labelKey: "nav.health", href: "/health", icon: Activity },
      { labelKey: "nav.logs", href: "/logs", icon: "\u{1F4DD}" },
    ],
  },
  {
    sectionKey: "nav.platform",
    items: [
      { labelKey: "nav.cluster", href: "/cluster", icon: "\u{1F310}" },
      { labelKey: "nav.deploy", href: "/deploy", icon: "\u{1F680}" },
      { labelKey: "nav.updates", href: "/updates", icon: "\u2B06\uFE0F" },
      { labelKey: "nav.vault", href: "/vault", icon: "\u{1F512}" },
      { labelKey: "nav.notifications", href: "/notifications", icon: "\u{1F514}" },
      { labelKey: "nav.mobile", href: "/mobile", icon: "\u{1F4F1}" },
    ],
  },
  {
    sectionKey: "nav.fleet",
    items: [
      { labelKey: "nav.tagsAndGroups", href: "/fleet", icon: "\u{1F465}" },
      { labelKey: "nav.marketplace", href: "/marketplace", icon: "\u{1F6D2}" },
    ],
  },
  {
    sectionKey: "nav.security",
    items: [
      { labelKey: "nav.iam", href: "/iam", icon: "\u{1F510}" },
      { labelKey: "nav.developer", href: "/developer", icon: "\u{1F4BB}" },
    ],
  },
  {
    sectionKey: "nav.administration",
    items: [
      { labelKey: "nav.plugins", href: "/plugins", icon: "\u{1F9E9}" },
      { labelKey: "nav.policies", href: "/policies", icon: "\u{1F6E1}" },
      { labelKey: "nav.users", href: "/users", icon: "\u{1F465}" },
      { labelKey: "nav.settings", href: "/settings", icon: "\u2699\uFE0F" },
      { labelKey: "nav.designSystem", href: "/design-system", icon: "\u{1F3A8}" },
    ],
  },
];

export function Sidebar() {
  const pathname = usePathname();
  const { t } = useLocale();
  const { sidebarOpen } = useUiStore();
  const setSidebarOpen = useUiStore((s) => s.setSidebarOpen);
  const { user, logout, can } = useAuth();
  const { data: version } = useVersion();

  const initial = user?.display_name?.charAt(0)?.toUpperCase() ?? "?";
  const closeMobileSidebar = () => {
    if (typeof window !== "undefined" && window.matchMedia("(max-width: 767px)").matches) {
      setSidebarOpen(false);
    }
  };

  return (
    <aside
      className={cn(
        "fixed inset-y-0 left-0 z-40 flex flex-col border-r border-[var(--color-border)] bg-[var(--color-bg)] transition-all duration-200 ease-in-out overflow-hidden flex-shrink-0 shadow-xl md:relative md:inset-auto md:z-auto md:shadow-none",
        sidebarOpen
          ? "w-64 max-w-[85vw] translate-x-0 md:w-60 md:max-w-none"
          : "w-64 max-w-[85vw] -translate-x-full md:w-16 md:max-w-none md:translate-x-0",
      )}
    >
      <div className="flex items-center h-12 px-4 border-b border-[var(--color-border)]">
        {sidebarOpen ? (
          <span className="text-headline-4 text-[var(--color-text-primary)]">RenOS</span>
        ) : (
          <span className="sr-only">RenOS</span>
        )}
      </div>

      <nav aria-label={t("nav.mainNavigation")} className="flex-1 overflow-y-auto py-4 px-3 space-y-6">
        {navItems.map((section) => {
          if (section.sectionKey === "nav.administration" && !can("manage_roles")) return null;
          return (
            <div key={section.sectionKey}>
              {sidebarOpen && (
                <p className="text-caption text-[var(--color-text-tertiary)] px-2 pb-2 uppercase tracking-wider">
                  {t(section.sectionKey)}
                </p>
              )}
              <ul className="space-y-1">
                {section.items.map((item) => {
                  const isActive = pathname === item.href || pathname.startsWith(item.href + "/");
                  return (
                    <li key={item.href}>
                      <Link
                        href={item.href}
                        aria-current={isActive ? "page" : undefined}
                        onClick={closeMobileSidebar}
                        className={cn(
                          "flex items-center gap-3 px-2 h-9 rounded-sm text-body transition-colors duration-150",
                          isActive
                            ? "bg-[var(--color-accent-tint)] text-[var(--color-accent)] font-medium"
                            : "text-[var(--color-text-secondary)] hover:bg-[var(--color-surface)] hover:text-[var(--color-text-primary)]",
                        )}
                      >
                        <span className="text-lg w-5 text-center flex-shrink-0">
                          {typeof item.icon === "string" ? (
                            item.icon
                          ) : (
                            <item.icon className="h-[18px] w-[18px] inline-block" />
                          )}
                        </span>
                        {sidebarOpen && <span>{t(item.labelKey)}</span>}
                      </Link>
                    </li>
                  );
                })}
              </ul>
            </div>
          );
        })}
      </nav>

      <div className="border-t border-[var(--color-border)] p-3 space-y-2">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 rounded-full bg-[var(--color-surface)] flex items-center justify-center text-body-small font-medium text-[var(--color-text-primary)]">
            {initial}
          </div>
          {sidebarOpen && (
            <div className="flex-1 min-w-0">
              <p className="text-body-small text-[var(--color-text-primary)] truncate">
                {user?.display_name ?? t("common.user")}
              </p>
              <p className="text-caption text-[var(--color-text-tertiary)]">{user?.role ?? ""}</p>
            </div>
          )}
        </div>
        {sidebarOpen && (
          <>
            <button onClick={logout} className="btn-ghost w-full justify-start text-body-small text-[var(--color-text-tertiary)]">
              {t("common.signOut")}
            </button>
            {version && (
              <p className="text-caption text-[var(--color-text-tertiary)] px-2">
                v{version.version}
              </p>
            )}
          </>
        )}
      </div>
    </aside>
  );
}
