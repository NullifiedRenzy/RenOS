"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";

export function useContainerMetrics(deviceId: string) {
  return useQuery({
    queryKey: ["container-metrics", deviceId],
    queryFn: () => api.listContainerMetrics(deviceId),
    enabled: !!deviceId,
    refetchInterval: 15_000,
    staleTime: 5_000,
  });
}

export function useContainerMetricHistory(deviceId: string, containerId: string | null, minutes?: number) {
  return useQuery({
    queryKey: ["container-metric-history", deviceId, containerId, minutes],
    queryFn: () => api.getContainerMetricHistory(deviceId, containerId!, minutes),
    enabled: !!deviceId && !!containerId,
    refetchInterval: 15_000,
    staleTime: 5_000,
  });
}
