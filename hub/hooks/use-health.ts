"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";

export function useHealthDashboard() {
  return useQuery({
    queryKey: ["health-dashboard"],
    queryFn: () => api.getHealthDashboard(),
    refetchInterval: 30_000,
    staleTime: 10_000,
  });
}
