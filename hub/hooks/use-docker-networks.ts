"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";

export function useDockerNetworks(deviceId: string) {
  return useQuery({
    queryKey: ["docker-networks", deviceId],
    queryFn: () => api.listDockerNetworks(deviceId),
    enabled: !!deviceId,
    refetchInterval: 120_000,
    staleTime: 60_000,
  });
}
