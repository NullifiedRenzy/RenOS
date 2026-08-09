"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";

export function usePolicies() {
  return useQuery({
    queryKey: ["policies"],
    queryFn: () => api.listPolicies(),
    refetchInterval: 30_000,
    staleTime: 10_000,
  });
}

export function usePolicy(id: string) {
  return useQuery({
    queryKey: ["policy", id],
    queryFn: () => api.getPolicy(id),
    enabled: !!id,
    refetchInterval: 15_000,
    staleTime: 5_000,
  });
}
