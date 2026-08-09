"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api-client";

export function useProcesses(deviceId: string) {
  return useQuery({
    queryKey: ["processes", deviceId],
    queryFn: () => api.listProcesses(deviceId),
    enabled: !!deviceId,
    refetchInterval: 5_000,
    staleTime: 2_000,
  });
}

export function useSearchProcesses(deviceId: string, query: string) {
  return useQuery({
    queryKey: ["processes", deviceId, "search", query],
    queryFn: () => api.searchProcesses(deviceId, query),
    enabled: !!deviceId && query.length > 0,
    refetchInterval: 5_000,
    staleTime: 2_000,
  });
}

export function useProcessHistory(deviceId: string, minutes?: number) {
  return useQuery({
    queryKey: ["process-history", deviceId, minutes],
    queryFn: () => api.getProcessHistory(deviceId, minutes),
    enabled: !!deviceId,
    refetchInterval: 10_000,
    staleTime: 5_000,
  });
}

export function useProcessAggregates(deviceId: string) {
  return useQuery({
    queryKey: ["process-aggregates", deviceId],
    queryFn: () => api.getProcessAggregates(deviceId),
    enabled: !!deviceId,
    refetchInterval: 5_000,
    staleTime: 2_000,
  });
}

export function useCreateProcessAction(deviceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: { pid: number; action_type: string }) =>
      api.createProcessAction(deviceId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["process-actions", deviceId] });
      queryClient.invalidateQueries({ queryKey: ["processes", deviceId] });
    },
  });
}

export function useProcessActions(deviceId: string) {
  return useQuery({
    queryKey: ["process-actions", deviceId],
    queryFn: () => api.listProcessActions(deviceId),
    enabled: !!deviceId,
    refetchInterval: 3_000,
    staleTime: 1_000,
  });
}
