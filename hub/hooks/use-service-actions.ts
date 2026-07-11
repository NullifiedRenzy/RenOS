"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api-client";

export function useServiceAction(deviceId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ unitName, action }: { unitName: string; action: string }) =>
      api.requestServiceAction(deviceId, unitName, action),
    onSuccess: () => {
      queryClient.refetchQueries({ queryKey: ["systemd-services", deviceId] });
    },
  });
}
