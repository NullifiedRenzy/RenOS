"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import type { CreateExecutionPayload } from "@/lib/types";

export function useExecutions(deviceId: string, limit?: number) {
  return useQuery({
    queryKey: ["executions", deviceId, limit],
    queryFn: () => api.listExecutions(deviceId, limit),
    enabled: !!deviceId,
    refetchInterval: 3_000,
    staleTime: 1_000,
  });
}

export function useExecutionOutput(deviceId: string, execId: string | null) {
  return useQuery({
    queryKey: ["execution-output", deviceId, execId],
    queryFn: () => api.getExecutionOutput(deviceId, execId!),
    enabled: !!deviceId && !!execId,
    refetchInterval: 1_000,
    staleTime: 500,
  });
}

export function useCreateExecution(deviceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateExecutionPayload) =>
      api.createExecution(deviceId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["executions", deviceId] });
    },
  });
}

export function useCancelExecution(deviceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (execId: string) =>
      api.cancelExecution(deviceId, execId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["executions", deviceId] });
    },
  });
}
