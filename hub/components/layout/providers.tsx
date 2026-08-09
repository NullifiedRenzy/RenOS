"use client";

import { useEffect } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useAuthStore } from "@/stores/auth-store";
import { useUiStore } from "@/stores/ui-store";
import { api } from "@/lib/api-client";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 30_000,
      gcTime: 300_000,
    },
  },
});

function HydrateStores({ children }: { children: React.ReactNode }) {
  const hydrateAuth = useAuthStore((s) => s.hydrate);
  const initTheme = useUiStore((s) => s.initTheme);

  useEffect(() => {
    hydrateAuth();
    initTheme();
  }, [hydrateAuth, initTheme]);

  useEffect(() => {
    const unsub = useAuthStore.subscribe((state, prev) => {
      if (state.token !== prev.token) {
        api.setAccessToken(state.token);
      }
    });
    return unsub;
  }, []);

  useEffect(() => {
    const stored = useAuthStore.getState();
    if (stored.token) {
      api.setAccessToken(stored.token);
    }
  }, []);

  return <>{children}</>;
}

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <QueryClientProvider client={queryClient}>
      <HydrateStores>{children}</HydrateStores>
    </QueryClientProvider>
  );
}
