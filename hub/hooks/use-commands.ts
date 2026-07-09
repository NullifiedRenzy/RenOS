"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api-client";

export function useCommands(deviceId: string) {
  return useQuery({
    queryKey: ["commands", deviceId],
    queryFn: () => api.listCommands(deviceId),
    enabled: !!deviceId,
    refetchInterval: 5_000,
    staleTime: 2_000,
  });
}

export function useRequestCommand(deviceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (command: string) => api.requestCommand(deviceId, command),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["commands", deviceId] });
    },
  });
}
