"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";

export function useMetrics() {
  return useQuery({
    queryKey: ["metrics"],
    queryFn: () => api.getMetrics(),
    refetchInterval: 10_000,
    staleTime: 5_000,
  });
}

export function useJobStats() {
  return useQuery({
    queryKey: ["job-stats"],
    queryFn: () => api.getJobStats(),
    refetchInterval: 15_000,
    staleTime: 10_000,
  });
}

export function useLogs(params?: { level?: string; source?: string; limit?: number; offset?: number }) {
  return useQuery({
    queryKey: ["logs", params],
    queryFn: () => api.listLogs(params),
    refetchInterval: 10_000,
    staleTime: 5_000,
  });
}
