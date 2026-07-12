"use client";

import { useQuery, useMutation } from "@tanstack/react-query";
import { api } from "@/lib/api-client";

export function usePlugins(deviceId: string) {
  return useQuery({
    queryKey: ["plugins", deviceId],
    queryFn: () => api.listPlugins(deviceId),
    refetchInterval: 300_000,
    staleTime: 240_000,
  });
}

export function usePluginAction(deviceId: string) {
  return useMutation({
    mutationFn: async ({ name, action }: { name: string; action: string }) => {
      return api.requestPluginAction(deviceId, name, action);
    },
  });
}

const STATUS_COLORS: Record<string, string> = {
  idle: "var(--color-text-tertiary)",
  running: "var(--color-primary)",
  success: "var(--color-success)",
  failed: "var(--color-error)",
};

export function getPluginStatusColor(status: string): string {
  return STATUS_COLORS[status.toLowerCase()] ?? "var(--color-text-tertiary)";
}
