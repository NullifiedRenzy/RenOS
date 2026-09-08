"use client";

import { useEffect, useState } from "react";
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
  const [ready, setReady] = useState(false);
  const hydrateAuth = useAuthStore((s) => s.hydrate);
  const initTheme = useUiStore((s) => s.initTheme);

  useEffect(() => {
    hydrateAuth();
    initTheme();

    const stored = useAuthStore.getState();
    api.setAccessToken(stored.token);
    setReady(true);
  }, [hydrateAuth, initTheme]);

  useEffect(() => {
    const unsub = useAuthStore.subscribe((state, prev) => {
      if (state.token !== prev.token) {
        api.setAccessToken(state.token);
      }
    });
    return unsub;
  }, []);

  // Child effects may request protected data as soon as they mount.
  if (!ready) {
    return (
      <div role="status" aria-label="Loading" className="min-h-screen flex items-center justify-center bg-[var(--color-bg)]">
        <div className="skeleton w-8 h-8 rounded-full" />
      </div>
    );
  }

  return <>{children}</>;
}

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <QueryClientProvider client={queryClient}>
      <HydrateStores>{children}</HydrateStores>
    </QueryClientProvider>
  );
}
