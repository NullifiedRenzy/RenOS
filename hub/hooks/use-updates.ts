"use client";

import { useQuery, useMutation } from "@tanstack/react-query";
import { api } from "@/lib/api-client";

export function useUpdates(deviceId: string) {
  return useQuery({
    queryKey: ["updates", deviceId],
    queryFn: () => api.listUpdates(deviceId),
    refetchInterval: 3_600_000,
    staleTime: 1_800_000,
  });
}

export function useUpdateAction(deviceId: string) {
  return useMutation({
    mutationFn: async (action: string) => {
      return api.requestUpdateAction(deviceId, action);
    },
  });
}

const SEVERITY_COLORS: Record<string, string> = {
  security: "var(--color-error)",
  standard: "var(--color-warning)",
};

export function getSeverityColor(severity: string): string {
  return SEVERITY_COLORS[severity.toLowerCase()] ?? "var(--color-text-tertiary)";
}

export const UPDATE_ACTIONS = [
  { value: "check", label: "Check" },
  { value: "dry-run", label: "Dry Run" },
  { value: "upgrade", label: "Upgrade" },
] as const;
