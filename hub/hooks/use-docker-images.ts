"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";

export function useDockerImages(deviceId: string) {
  return useQuery({
    queryKey: ["docker-images", deviceId],
    queryFn: () => api.listDockerImages(deviceId),
    enabled: !!deviceId,
    refetchInterval: 120_000,
    staleTime: 60_000,
  });
}
