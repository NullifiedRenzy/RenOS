"use client";

import { useEffect } from "react";
import dynamic from "next/dynamic";
import { useRouter } from "next/navigation";
import { Sidebar } from "@/components/layout/sidebar";
import { ThemeToggle } from "@/components/layout/theme-toggle";
import { LocaleToggle } from "@/components/layout/locale-toggle";
import { SkipLink } from "@/components/ui/skip-link";
import { OnboardingWizard } from "@/components/onboarding/onboarding-wizard";
import { SearchTrigger } from "@/components/command-center/search-trigger";
import { NotificationBell } from "@/components/notifications/notification-bell";
import { useAuthStore } from "@/stores/auth-store";
import { useUiStore } from "@/stores/ui-store";
import { useWebSocket } from "@/hooks/use-websocket";
import { useOnboarding } from "@/hooks/use-onboarding";
import { useLocale } from "@/lib/i18n/use-locale";
import { cn } from "@/lib/utils";

const AiSidebar = dynamic(() => import("@/components/ai/ai-sidebar").then((m) => m.AiSidebar), {
  ssr: false,
});
const CommandCenter = dynamic(
  () => import("@/components/command-center/command-center").then((m) => m.CommandCenter),
  { ssr: false },
);

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { t } = useLocale();
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const sidebarOpen = useUiStore((s) => s.sidebarOpen);
  const toggleSidebar = useUiStore((s) => s.toggleSidebar);
  const { status, loading: onboardingLoading, error: onboardingError } = useOnboarding();

  useWebSocket();

  useEffect(() => {
    if (!isAuthenticated) {
      router.push("/login");
    }
  }, [isAuthenticated, router]);

  if (!isAuthenticated) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-[var(--color-bg)]">
        <div className="skeleton w-8 h-8 rounded-full" />
      </div>
    );
  }

  // While the onboarding status is being fetched, show a skeleton so the
  // wizard doesn't flash in after the dashboard has already painted.
  if (onboardingLoading) {
    return (
      <div className="min-h-screen bg-[var(--color-bg)] lg:flex">
        <aside className="hidden lg:flex lg:w-72 border-r border-[var(--color-border)] flex-shrink-0 flex-col gap-4 p-8">
          <div className="skeleton h-4 w-24" />
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="skeleton h-9 w-full" />
          ))}
        </aside>
        <main className="flex-1 p-6 space-y-6">
          <div className="skeleton h-8 w-72" />
          <div className="skeleton h-28 w-full max-w-xl" />
          <div className="skeleton h-10 w-40" />
        </main>
      </div>
    );
  }

  // On first login, show the onboarding wizard instead of the app until it is
  // marked complete. If the status check fails, fall through so access is not
  // blocked by the onboarding check.
  if (!onboardingError && status && !status.onboarding_complete) {
    return <OnboardingWizard />;
  }

  return (
    <div className="flex h-screen overflow-hidden bg-[var(--color-bg)]">
      <SkipLink />
      <Sidebar />
      <div className="flex-1 flex flex-col overflow-hidden">
        <header className="flex items-center justify-between h-12 px-4 border-b border-[var(--color-border)] bg-[var(--color-bg)] flex-shrink-0">
          <div className="flex items-center gap-2">
            <button onClick={toggleSidebar} className="btn-ghost p-1 text-lg" aria-label={sidebarOpen ? t("nav.collapseSidebar") : t("nav.expandSidebar")} aria-expanded={sidebarOpen}>
              {"\u2630"}
            </button>
            <SearchTrigger />
          </div>
          <div className="flex items-center gap-2">
            <NotificationBell />
            <LocaleToggle />
            <ThemeToggle />
          </div>
        </header>
        <main id="main-content" tabIndex={-1} className={cn("flex-1 overflow-y-auto p-6")}>
          {children}
        </main>
      </div>
      <AiSidebar />
      <CommandCenter />
    </div>
  );
}
