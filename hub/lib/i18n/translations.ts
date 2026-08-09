export const locales = ["en"] as const;
export type Locale = (typeof locales)[number];
export const defaultLocale: Locale = "en";
export const localeNames: Record<Locale, string> = { en: "English" };

export const translations = {
  en: {
    common: {
      save: "Save",
      cancel: "Cancel",
      close: "Close",
      delete: "Delete",
      edit: "Edit",
      loading: "Loading…",
      retry: "Retry",
      search: "Search",
      signOut: "Sign out",
      user: "User",
    },
    nav: {
      mainNavigation: "Main navigation",
      collapseSidebar: "Collapse sidebar",
      expandSidebar: "Expand sidebar",
      main: "Main",
      dashboard: "Dashboard",
      devices: "Devices",
      allDevices: "All Devices",
      orchestration: "Orchestration",
      groups: "Groups",
      jobs: "Jobs",
      schedules: "Schedules",
      ai: "AI",
      aiAssistant: "AI Assistant",
      monitoring: "Monitoring",
      observability: "Observability",
      health: "Health",
      logs: "Logs",
      platform: "Platform",
      cluster: "Cluster",
      deploy: "Deploy",
      updates: "Updates",
      vault: "Vault",
      notifications: "Notifications",
      mobile: "Mobile",
      fleet: "Fleet",
      tagsAndGroups: "Tags & Groups",
      marketplace: "Marketplace",
      security: "Security",
      iam: "IAM",
      developer: "Developer",
      administration: "Administration",
      plugins: "Plugins",
      policies: "Policies",
      users: "Users",
      settings: "Settings",
      designSystem: "Design System",
    },
    skip: {
      toMainContent: "Skip to main content",
    },
    theme: {
      light: "Light",
      dark: "Dark",
      system: "System",
      themeLabel: "Theme: {label}",
      switchToDark: "Switch to dark mode",
      switchToLight: "Switch to light mode",
      switchToSystem: "Switch to system theme",
    },
    search: {
      placeholder: "Search…",
      ariaLabel: "Search the platform (Ctrl+K)",
    },
    locale: {
      label: "Language",
    },
    notifications: {
      title: "Notifications",
      markAllRead: "Mark all read",
      viewAll: "View all notifications",
      empty: "You're all caught up.",
      ariaOpen: "Notifications, {count} unread",
      unread: "Unread",
      types: {
        alert: "Alert",
        workflow: "Workflow",
        job: "Job",
        ai: "AI",
        system: "System",
        mention: "Mention",
      },
    },
    time: {
      never: "Never",
      justNow: "Just now",
      minutesAgo: "{count}m ago",
      hoursAgo: "{count}h ago",
    },
  },
} as const;

export type Messages = (typeof translations)["en"];

export type DeepKeys<T> = {
  [K in keyof T & string]: T[K] extends string
    ? K
    : T[K] extends object
      ? `${K}.${DeepKeys<T[K]>}`
      : never;
}[keyof T & string];

export type TranslationKey = DeepKeys<Messages>;
