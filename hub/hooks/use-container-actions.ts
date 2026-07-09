"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api-client";

export function useContainerAction(deviceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ containerId, action }: { containerId: string; action: string }) =>
      api.requestContainerAction(deviceId, containerId, action),
    onSuccess: () => {
      queryClient.refetchQueries({ queryKey: ["containers", deviceId] });
    },
  });
}
