"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";

export function useStorageDevices(deviceId: string) {
  return useQuery({
    queryKey: ["storage-devices", deviceId],
    queryFn: () => api.listStorageDevices(deviceId),
    refetchInterval: 180_000,
    staleTime: 120_000,
  });
}

export function getDeviceTypeIcon(type: string): string {
  const icons: Record<string, string> = {
    disk: "\uD83D\uDCBF",
    part: "\uD83D\uDCC1",
    rom: "\uD83D\uDCC0",
    loop: "\uD83D\uDD04",
    crypt: "\uD83D\uDD12",
    lvm: "\uD83D\uDCE6",
  };
  return icons[type] ?? "\uD83D\uDCBE";
}
