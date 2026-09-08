import { create } from "zustand";
import type { User } from "@/lib/types";

const STORAGE_KEY = "renos_auth";

interface StoredAuth {
  user: User;
  token: string;
}

interface AuthState {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  hasHydrated: boolean;
  setAuth: (user: User, token: string, remember: boolean) => void;
  logout: () => void;
  hydrate: () => void;
  hasRole: (role: string) => boolean;
  can: (permission: string) => boolean;
}

function persist(user: User, token: string, remember: boolean) {
  try {
    const value = JSON.stringify({ user, token });
    if (remember) {
      localStorage.setItem(STORAGE_KEY, value);
      sessionStorage.removeItem(STORAGE_KEY);
    } else {
      sessionStorage.setItem(STORAGE_KEY, value);
      localStorage.removeItem(STORAGE_KEY);
    }
  } catch { }
}

function clear() {
  try {
    localStorage.removeItem(STORAGE_KEY);
    sessionStorage.removeItem(STORAGE_KEY);
  } catch { }
}

function computePermissions(role: string | undefined): Set<string> {
  switch (role) {
    case "admin":
      return new Set(["read", "write", "admin", "manage_roles", "manage_enroll", "manage_policies"]);
    case "operator":
      return new Set(["read", "write"]);
    case "viewer":
      return new Set(["read"]);
    default:
      return new Set(["read"]);
  }
}

export const useAuthStore = create<AuthState>((set, get) => ({
  user: null,
  token: null,
  isAuthenticated: false,
  hasHydrated: false,

  setAuth: (user, token, remember) => {
    persist(user, token, remember);
    set({ user, token, isAuthenticated: true, hasHydrated: true });
  },

  logout: () => {
    clear();
    set({ user: null, token: null, isAuthenticated: false });
  },

  hydrate: () => {
    try {
      const raw = localStorage.getItem(STORAGE_KEY) ?? sessionStorage.getItem(STORAGE_KEY);
      if (!raw) {
        set({ hasHydrated: true });
        return;
      }
      const stored: StoredAuth = JSON.parse(raw);
      if (stored.user && stored.token) {
        set({ user: stored.user, token: stored.token, isAuthenticated: true, hasHydrated: true });
        return;
      }
    } catch {
      clear();
    }
    set({ hasHydrated: true });
  },

  hasRole: (role: string) => {
    const user = get().user;
    return user?.role === role;
  },

  can: (permission: string) => {
    const user = get().user;
    const perms = computePermissions(user?.role);
    return perms.has(permission);
  },
}));
