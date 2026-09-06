"use client";

import { useCallback } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/auth-store";
import { api } from "@/lib/api-client";

export function useAuth() {
  const router = useRouter();
  const {
    setAuth,
    logout: clearAuth,
    isAuthenticated,
    user,
    token,
    hasRole,
    can,
  } = useAuthStore();

  const login = useCallback(
    async (email: string, password: string, totpCode?: string) => {
      const result = await api.login(email, password, totpCode);
      if (result.user && result.token) {
        setAuth(result.user, result.token);
      }
      return result;
    },
    [setAuth],
  );

  const logout = useCallback(() => {
    api.setAccessToken(null);
    clearAuth();
    router.push("/login");
  }, [clearAuth, router]);

  return { login, logout, isAuthenticated, user, token, hasRole, can };
}
