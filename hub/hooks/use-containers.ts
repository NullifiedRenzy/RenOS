"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";

export function useContainers(deviceId: string) {
  return useQuery({
    queryKey: ["containers", deviceId],
    queryFn: () => api.listContainers(deviceId),
    enabled: !!deviceId,
    refetchInterval: 30_000,
    staleTime: 10_000,
  });
}
