"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api-client";

export function useNotificationProviders() {
  return useQuery({
    queryKey: ["notif-providers"],
    queryFn: () => api.getNotificationProviders(),
  });
}

export function useCreateNotificationProvider() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (p: Record<string, unknown>) =>
      api.createNotificationProvider(p as any),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["notif-providers"] }),
  });
}

export function useDeleteNotificationProvider() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteNotificationProvider(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["notif-providers"] }),
  });
}

export function useNotificationTemplates() {
  return useQuery({
    queryKey: ["notif-templates"],
    queryFn: () => api.getNotificationTemplates(),
  });
}

export function useCreateNotificationTemplate() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (t: Record<string, unknown>) =>
      api.createNotificationTemplate(t as any),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["notif-templates"] }),
  });
}

export function useDeleteNotificationTemplate() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteNotificationTemplate(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["notif-templates"] }),
  });
}

export function useNotificationRules() {
  return useQuery({
    queryKey: ["notif-rules"],
    queryFn: () => api.getNotificationRules(),
  });
}

export function useCreateNotificationRule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (r: Record<string, unknown>) =>
      api.createNotificationRule(r as any),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["notif-rules"] }),
  });
}

export function useDeleteNotificationRule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteNotificationRule(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["notif-rules"] }),
  });
}

export function useNotificationHistory() {
  return useQuery({
    queryKey: ["notif-history"],
    queryFn: () => api.getNotificationHistory(),
    refetchInterval: 30000,
  });
}

export function useNotificationStats() {
  return useQuery({
    queryKey: ["notif-stats"],
    queryFn: () => api.getNotificationStats(),
  });
}
