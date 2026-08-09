import {
  Activity, Bell, Bot, Boxes, ClipboardList, Cloud, Code2, Cpu,
  FileClock, Gauge, KeySquare, LayoutDashboard, LifeBuoy,
  ListChecks, Logs, type LucideIcon, MonitorCog, Moon, Package,
  PanelLeft, Play, Rocket, Settings, ShieldCheck, Sun, Terminal,
  Users, Vault, Workflow,
} from "lucide-react";

export type CommandPick = "devices" | "workflows";

export interface CommandRunContext {
  navigate: (href: string) => void;
  execute: (id: string, arg?: { id: string; title: string }) => void;
}

export interface Command {
  id: string;
  title: string;
  keywords: string;
  icon: LucideIcon;
  group: string;
  shortcut?: string;
  pick?: CommandPick;
  run: (ctx: CommandRunContext, arg?: { id: string; title: string }) => void;
}

export function getCommands(): Command[] {
  return [
    { id: "open-dashboard", title: "Open Dashboard", keywords: "home overview start", icon: LayoutDashboard, group: "Navigate", shortcut: "G D", run: ({ navigate }) => navigate("/dashboard") },
    { id: "open-devices", title: "Open Devices", keywords: "hosts inventory machines fleet", icon: MonitorCog, group: "Navigate", shortcut: "G V", run: ({ navigate }) => navigate("/devices") },
    { id: "open-jobs", title: "Open Jobs", keywords: "runs executions tasks", icon: ClipboardList, group: "Navigate", shortcut: "G J", run: ({ navigate }) => navigate("/jobs") },
    { id: "open-workflows", title: "Open Workflows", keywords: "automations pipelines", icon: Workflow, group: "Navigate", run: ({ navigate }) => navigate("/workflows") },
    { id: "open-schedules", title: "Open Schedules", keywords: "cron timers recurring", icon: FileClock, group: "Navigate", run: ({ navigate }) => navigate("/schedules") },
    { id: "open-plugins", title: "Open Plugins", keywords: "extensions addons modules", icon: Package, group: "Navigate", run: ({ navigate }) => navigate("/plugins") },
    { id: "open-policies", title: "Open Policies", keywords: "rules security allow deny", icon: ShieldCheck, group: "Navigate", run: ({ navigate }) => navigate("/policies") },
    { id: "open-vault", title: "Open Vault", keywords: "secrets keys credentials", icon: Vault, group: "Navigate", shortcut: "G S", run: ({ navigate }) => navigate("/vault") },
    { id: "open-users", title: "Open Users", keywords: "accounts people members", icon: Users, group: "Navigate", run: ({ navigate }) => navigate("/users") },
    { id: "open-ai", title: "Open AI Assistant", keywords: "chat llm assistant", icon: Bot, group: "Navigate", run: ({ navigate }) => navigate("/ai") },
    { id: "open-health", title: "Open Health", keywords: "status uptime checks", icon: Activity, group: "Navigate", run: ({ navigate }) => navigate("/health") },
    { id: "open-logs", title: "View Logs", keywords: "log entries audit syslog", icon: Logs, group: "Navigate", run: ({ navigate }) => navigate("/logs") },
    { id: "open-notifications", title: "Open Notifications", keywords: "alerts bell mentions", icon: Bell, group: "Navigate", run: ({ navigate }) => navigate("/notifications") },
    { id: "open-cluster", title: "Open Cluster", keywords: "nodes kubernetes swarm", icon: Boxes, group: "Navigate", run: ({ navigate }) => navigate("/cluster") },
    { id: "open-containers", title: "Open Containers", keywords: "docker pods images", icon: Boxes, group: "Navigate", run: ({ navigate }) => navigate("/devices") },
    { id: "open-monitoring", title: "Open Monitoring", keywords: "metrics dashboards graphs", icon: Gauge, group: "Navigate", run: ({ navigate }) => navigate("/monitoring") },
    { id: "open-marketplace", title: "Open Marketplace", keywords: "install discover browse", icon: Rocket, group: "Navigate", run: ({ navigate }) => navigate("/marketplace") },
    { id: "open-updates", title: "Open Updates", keywords: "upgrade version patch", icon: Cloud, group: "Navigate", run: ({ navigate }) => navigate("/updates") },
    { id: "open-settings", title: "Open Settings", keywords: "preferences configuration options", icon: Settings, group: "Navigate", shortcut: "G ,", run: ({ navigate }) => navigate("/settings") },
    { id: "open-developer", title: "Open Developer Console", keywords: "api keys webhooks tokens", icon: Code2, group: "Navigate", run: ({ navigate }) => navigate("/developer") },

    { id: "restart-device", title: "Restart Device", keywords: "reboot power cycle restart machine", icon: Cpu, group: "Actions", pick: "devices", run: ({ execute }, arg) => execute("restart-device", arg) },
    { id: "run-workflow", title: "Run Workflow", keywords: "execute trigger pipeline automation", icon: Play, group: "Actions", pick: "workflows", run: ({ execute }, arg) => execute("run-workflow", arg) },
    { id: "create-secret", title: "Create Secret", keywords: "new vault secret key credential", icon: KeySquare, group: "Actions", run: ({ navigate }) => navigate("/vault?new=1") },
    { id: "open-terminal", title: "Open Terminal", keywords: "shell ssh console", icon: Terminal, group: "Actions", pick: "devices", run: ({ execute }, arg) => execute("open-terminal", arg) },

    { id: "toggle-theme", title: "Toggle Dark Mode", keywords: "theme dark light appearance", icon: Moon, group: "Preferences", shortcut: "G T", run: ({ execute }) => execute("toggle-theme") },
    { id: "toggle-sidebar", title: "Toggle Sidebar", keywords: "collapse panel navigation", icon: PanelLeft, group: "Preferences", run: ({ execute }) => execute("toggle-sidebar") },
    { id: "help-shortcuts", title: "Show Keyboard Shortcuts", keywords: "help keys shortcuts kbd", icon: LifeBuoy, group: "Preferences", shortcut: "?", run: ({ execute }) => execute("help-shortcuts") },
    { id: "list-checks", title: "Production Checklist", keywords: "readiness checklist production", icon: ListChecks, group: "Help", run: ({ navigate }) => navigate("/health") },
    { id: "light-mode", title: "Switch to Light Mode", keywords: "theme appearance light", icon: Sun, group: "Preferences", run: ({ execute }) => execute("light-mode") },
  ];
}

export const COMMAND_GROUPS = ["Navigate", "Actions", "Preferences", "Help"] as const;
