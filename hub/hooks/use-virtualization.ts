"use client";

import { useQuery, useMutation } from "@tanstack/react-query";
import { api } from "@/lib/api-client";

export function useVirtualMachines(deviceId: string) {
  return useQuery({
    queryKey: ["virtual-machines", deviceId],
    queryFn: () => api.listVirtualMachines(deviceId),
    refetchInterval: 180_000,
    staleTime: 120_000,
  });
}

export function useVMAction(deviceId: string) {
  return useMutation({
    mutationFn: async ({ domainName, action }: { domainName: string; action: string }) => {
      return api.requestVMAction(deviceId, domainName, action);
    },
  });
}

export function formatVmMemory(bytes: number): string {
  if (bytes <= 0) return "0";
  const gib = bytes / (1024 * 1024 * 1024);
  if (gib >= 1) return `${gib.toFixed(1)} GiB`;
  const mib = bytes / (1024 * 1024);
  return `${mib.toFixed(0)} MiB`;
}

const STATE_COLORS: Record<string, string> = {
  running: "var(--color-success)",
  "shut off": "var(--color-text-tertiary)",
  paused: "var(--color-warning)",
  crashed: "var(--color-error)",
  blocked: "var(--color-warning)",
};

export function getVmStateColor(state: string): string {
  return STATE_COLORS[state.toLowerCase()] ?? "var(--color-text-tertiary)";
}
