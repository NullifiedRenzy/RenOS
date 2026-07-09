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
  setAuth: (user: User, token: string) => void;
  logout: () => void;
  hydrate: () => void;
}

function persist(user: User, token: string) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ user, token }));
  } catch { }
}

function clear() {
  try {
    localStorage.removeItem(STORAGE_KEY);
  } catch { }
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  token: null,
  isAuthenticated: false,

  setAuth: (user, token) => {
    persist(user, token);
    set({ user, token, isAuthenticated: true });
  },

  logout: () => {
    clear();
    set({ user: null, token: null, isAuthenticated: false });
  },

  hydrate: () => {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (!raw) return;
      const stored: StoredAuth = JSON.parse(raw);
      if (stored.user && stored.token) {
        set({ user: stored.user, token: stored.token, isAuthenticated: true });
      }
    } catch { }
  },
}));
