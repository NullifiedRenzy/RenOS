"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";

export function useNetworkInterfaces(deviceId: string) {
  return useQuery({
    queryKey: ["network-interfaces", deviceId],
    queryFn: () => api.listNetworkInterfaces(deviceId),
    refetchInterval: 120_000,
    staleTime: 60_000,
  });
}

export function getInterfaceIcon(type: string): string {
  const icons: Record<string, string> = {
    ether: "\uD83D\uDD0C",
    loopback: "\uD83D\uDD04",
    wireless: "\uD83D\uDCF6",
    bridge: "\uD83D\uDD17",
    bond: "\uD83D\uDCAA",
    vlan: "\uD83C\uDF10",
    tun: "\uD83D\uDD12",
    tap: "\uD83D\uDD12",
    veth: "\uD83D\uDD17",
  };
  return icons[type] ?? "\uD83D\uDDC4\uFE0F";
}

export function formatSpeed(speed: number): string {
  if (speed <= 0) return "\u2014";
  if (speed >= 1000) return `${(speed / 1000).toFixed(1)} Gbps`;
  return `${speed} Mbps`;
}
