"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api-client";

export interface InAppFilter {
  unread?: boolean;
  type?: string;
  search?: string;
  limit?: number;
}

export function useInAppNotifications(filter: InAppFilter = {}) {
  return useQuery({
    queryKey: ["inapp-notifications", filter],
    queryFn: () => api.getInAppNotifications(filter),
    refetchInterval: 30_000,
    refetchOnWindowFocus: true,
  });
}

export function useInAppUnreadCount() {
  return useQuery({
    queryKey: ["inapp-unread"],
    queryFn: () => api.getInAppUnreadCount(),
    refetchInterval: 30_000,
    refetchOnWindowFocus: true,
  });
}

export function useMarkInAppRead() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.markInAppRead(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["inapp-notifications"] });
      qc.invalidateQueries({ queryKey: ["inapp-unread"] });
    },
  });
}

export function useMarkAllInAppRead() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => api.markAllInAppRead(),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["inapp-notifications"] });
      qc.invalidateQueries({ queryKey: ["inapp-unread"] });
    },
  });
}
