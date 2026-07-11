"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";

export function useSystemdServices(deviceId: string) {
  return useQuery({
    queryKey: ["systemd-services", deviceId],
    queryFn: () => api.listSystemdServices(deviceId),
    enabled: !!deviceId,
    refetchInterval: 120_000,
    staleTime: 60_000,
  });
}
