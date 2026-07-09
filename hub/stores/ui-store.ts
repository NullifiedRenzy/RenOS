import { create } from "zustand";

type Theme = "light" | "dark" | "system";

interface UiState {
  sidebarOpen: boolean;
  theme: Theme;
  resolvedTheme: "light" | "dark";
  toggleSidebar: () => void;
  setSidebarOpen: (open: boolean) => void;
  setTheme: (theme: Theme) => void;
  initTheme: () => void;
}

function getSystemTheme(): "light" | "dark" {
  if (typeof window === "undefined") return "light";
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

function resolveTheme(theme: Theme): "light" | "dark" {
  if (theme === "system") return getSystemTheme();
  return theme;
}

function applyTheme(resolved: "light" | "dark") {
  if (typeof document !== "undefined") {
    document.documentElement.classList.toggle("dark", resolved === "dark");
  }
}

export const useUiStore = create<UiState>((set, get) => ({
  sidebarOpen: true,
  theme: "system",
  resolvedTheme: "light",

  toggleSidebar: () => set((s) => ({ sidebarOpen: !s.sidebarOpen })),

  setSidebarOpen: (open) => set({ sidebarOpen: open }),

  setTheme: (theme) => {
    const resolved = resolveTheme(theme);
    applyTheme(resolved);
    try { localStorage.setItem("renos_theme", theme); } catch { }
    set({ theme, resolvedTheme: resolved });
  },

  initTheme: () => {
    try {
      const stored = localStorage.getItem("renos_theme") as Theme | null;
      const theme = stored ?? "system";
      const resolved = resolveTheme(theme);
      applyTheme(resolved);
      set({ theme, resolvedTheme: resolved });
    } catch {
      const resolved = resolveTheme("system");
      applyTheme(resolved);
      set({ theme: "system", resolvedTheme: resolved });
    }
  },
}));
