"use client";

import { useQuery } from "@tanstack/react-query";
import { api, LIST_LIMIT } from "@/lib/api-client";

export function useDevices() {
  return useQuery({
    queryKey: ["devices"],
    queryFn: () => api.listDevices(LIST_LIMIT),
    refetchInterval: 30_000,
    staleTime: 10_000,
  });
}

export function useDevice(id: string) {
  return useQuery({
    queryKey: ["device", id],
    queryFn: () => api.getDevice(id),
    enabled: !!id,
    refetchInterval: 15_000,
    staleTime: 5_000,
  });
}
