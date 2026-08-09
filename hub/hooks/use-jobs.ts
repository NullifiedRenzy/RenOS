"use client";

import { useQuery } from "@tanstack/react-query";
import { api, LIST_LIMIT } from "@/lib/api-client";

export function useJobs() {
  return useQuery({
    queryKey: ["jobs"],
    queryFn: () => api.listJobs(LIST_LIMIT),
    refetchInterval: 10_000,
    staleTime: 5_000,
  });
}

export function useJob(id: string) {
  return useQuery({
    queryKey: ["job", id],
    queryFn: () => api.getJob(id),
    enabled: !!id,
    refetchInterval: 3_000,
    staleTime: 1_000,
  });
}
