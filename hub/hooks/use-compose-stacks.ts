"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";

export function useComposeStacks(deviceId: string) {
  return useQuery({
    queryKey: ["compose-stacks", deviceId],
    queryFn: () => api.listComposeStacks(deviceId),
    enabled: !!deviceId,
    refetchInterval: 60_000,
    staleTime: 30_000,
  });
}
