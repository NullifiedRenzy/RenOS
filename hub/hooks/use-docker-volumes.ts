"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";

export function useDockerVolumes(deviceId: string) {
  return useQuery({
    queryKey: ["docker-volumes", deviceId],
    queryFn: () => api.listDockerVolumes(deviceId),
    enabled: !!deviceId,
    refetchInterval: 120_000,
    staleTime: 60_000,
  });
}
