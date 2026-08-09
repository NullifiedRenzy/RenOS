"use client";

import { useQuery } from "@tanstack/react-query";
import { api, LIST_LIMIT } from "@/lib/api-client";

export function useGroups() {
  return useQuery({
    queryKey: ["groups"],
    queryFn: () => api.listGroups(LIST_LIMIT),
    refetchInterval: 30_000,
    staleTime: 10_000,
  });
}

export function useGroup(id: string) {
  return useQuery({
    queryKey: ["group", id],
    queryFn: () => api.getGroup(id),
    enabled: !!id,
    refetchInterval: 15_000,
    staleTime: 5_000,
  });
}

export function useGroupAction(actionId: string) {
  return useQuery({
    queryKey: ["group-action", actionId],
    queryFn: () => api.getGroupAction(actionId),
    enabled: !!actionId,
    refetchInterval: 3_000,
  });
}
