"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useAuth } from "@/hooks/use-auth";
import { useUiStore } from "@/stores/ui-store";
import { cn } from "@/lib/utils";

const navItems = [
  {
    section: "Main",
    items: [
      { label: "Dashboard", href: "/dashboard", icon: "\u{1F4CA}" },
    ],
  },
  {
    section: "Devices",
    items: [
      { label: "All Devices", href: "/devices", icon: "\u{1F4BB}" },
    ],
  },
];

export function Sidebar() {
  const pathname = usePathname();
  const { sidebarOpen } = useUiStore();
  const { user, logout } = useAuth();

  const initial = user?.display_name?.charAt(0)?.toUpperCase() ?? "?";

  return (
    <aside
      className={cn(
        "flex flex-col border-r border-[var(--color-border)] bg-[var(--color-bg)] transition-all duration-200 ease-in-out overflow-hidden flex-shrink-0",
        sidebarOpen ? "w-60" : "w-0 md:w-16",
      )}
    >
      <div className="flex items-center h-12 px-4 border-b border-[var(--color-border)]">
        <span className="text-headline-4 text-[var(--color-text-primary)]">RenOS</span>
      </div>

      <nav className="flex-1 overflow-y-auto py-4 px-3 space-y-6">
        {navItems.map((section) => (
          <div key={section.section}>
            {sidebarOpen && (
              <p className="text-caption text-[var(--color-text-tertiary)] px-2 pb-2 uppercase tracking-wider">
                {section.section}
              </p>
            )}
            <ul className="space-y-1">
              {section.items.map((item) => {
                const isActive = pathname === item.href || pathname.startsWith(item.href + "/");
                return (
                  <li key={item.href}>
                    <Link
                      href={item.href}
                      className={cn(
                        "flex items-center gap-3 px-2 h-9 rounded-sm text-body transition-colors duration-150",
                        isActive
                          ? "bg-[var(--color-accent-tint)] text-[var(--color-accent)] font-medium"
                          : "text-[var(--color-text-secondary)] hover:bg-[var(--color-surface)] hover:text-[var(--color-text-primary)]",
                      )}
                    >
                      <span className="text-lg w-5 text-center flex-shrink-0">{item.icon}</span>
                      {sidebarOpen && <span>{item.label}</span>}
                    </Link>
                  </li>
                );
              })}
            </ul>
          </div>
        ))}
      </nav>

      <div className="border-t border-[var(--color-border)] p-3 space-y-2">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 rounded-full bg-[var(--color-surface)] flex items-center justify-center text-body-small font-medium text-[var(--color-text-primary)]">
            {initial}
          </div>
          {sidebarOpen && (
            <div className="flex-1 min-w-0">
              <p className="text-body-small text-[var(--color-text-primary)] truncate">
                {user?.display_name ?? "User"}
              </p>
              <p className="text-caption text-[var(--color-text-tertiary)]">{user?.role ?? ""}</p>
            </div>
          )}
        </div>
        {sidebarOpen && (
          <button onClick={logout} className="btn-ghost w-full justify-start text-body-small text-[var(--color-text-tertiary)]">
            Sign out
          </button>
        )}
      </div>
    </aside>
  );
}
