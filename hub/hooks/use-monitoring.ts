"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import type { MonitoringWidget } from "@/lib/types";

export function useMonitoringMetrics(name: string, deviceId?: string) {
  return useQuery({
    queryKey: ["monitoring-metrics", name, deviceId],
    queryFn: async () => {
      const since = new Date(Date.now() - 3600000).toISOString();
      const res = await api.queryMetrics(name, deviceId, since);
      return res.data;
    },
    refetchInterval: 30000,
  });
}

export function useLatestDeviceMetrics(deviceId: string) {
  return useQuery({
    queryKey: ["device-metrics", deviceId],
    queryFn: async () => {
      const res = await api.getLatestDeviceMetrics(deviceId);
      return res.data;
    },
    enabled: !!deviceId,
    refetchInterval: 30000,
  });
}

export function useAlertRules() {
  return useQuery({
    queryKey: ["alert-rules"],
    queryFn: async () => {
      const res = await api.getAlertRules();
      return res.data;
    },
  });
}

export function useCreateAlertRule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (rule: Record<string, unknown>) => api.createAlertRule(rule as any),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["alert-rules"] }),
  });
}

export function useDeleteAlertRule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteAlertRule(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["alert-rules"] }),
  });
}

export function useActiveAlerts() {
  return useQuery({
    queryKey: ["active-alerts"],
    queryFn: async () => {
      const res = await api.getActiveAlerts();
      return res.data;
    },
    refetchInterval: 10000,
  });
}

export function useRecentAlerts() {
  return useQuery({
    queryKey: ["recent-alerts"],
    queryFn: async () => {
      const res = await api.getRecentAlerts();
      return res.data;
    },
    refetchInterval: 30000,
  });
}

export function useAlertSummary() {
  return useQuery({
    queryKey: ["alert-summary"],
    queryFn: async () => {
      const res = await api.getAlertSummary();
      return res.data;
    },
  });
}

export function useAcknowledgeAlert() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.acknowledgeAlert(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["active-alerts"] });
      qc.invalidateQueries({ queryKey: ["recent-alerts"] });
    },
  });
}

export function useResolveAlert() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.resolveAlert(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["active-alerts"] });
      qc.invalidateQueries({ queryKey: ["recent-alerts"] });
      qc.invalidateQueries({ queryKey: ["alert-summary"] });
    },
  });
}

export function useMonitoringDashboards() {
  return useQuery({
    queryKey: ["monitoring-dashboards"],
    queryFn: async () => {
      const res = await api.getMonitoringDashboards();
      return res.data;
    },
  });
}

export function useMonitoringDashboard(id: string) {
  return useQuery({
    queryKey: ["monitoring-dashboard", id],
    queryFn: async () => {
      const res = await api.getMonitoringDashboard(id);
      return res.data;
    },
    enabled: !!id,
    refetchInterval: 15000,
  });
}

export function useUpdateMonitoringDashboard(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (update: { name?: string; description?: string; is_default?: boolean; layout?: unknown; widgets?: Partial<MonitoringWidget>[] }) =>
      api.updateMonitoringDashboard(id, update),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["monitoring-dashboards"] });
      qc.invalidateQueries({ queryKey: ["monitoring-dashboard", id] });
    },
  });
}

export function useDeleteMonitoringWidget() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteMonitoringWidget(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["monitoring-dashboards"] });
      qc.invalidateQueries({ queryKey: ["monitoring-dashboard"] });
    },
  });
}

export function useCreateMonitoringDashboard() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (dash: Record<string, unknown>) => api.createMonitoringDashboard(dash as any),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["monitoring-dashboards"] }),
  });
}

export function useDeleteMonitoringDashboard() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteMonitoringDashboard(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["monitoring-dashboards"] }),
  });
}

export function useMonitoringLogs(level?: string, query?: string) {
  return useQuery({
    queryKey: ["monitoring-logs", level, query],
    queryFn: async () => {
      const res = await api.getMonitoringLogs(level, query);
      return res.data;
    },
    refetchInterval: 15000,
  });
}

export function useMonitoringLogStats() {
  return useQuery({
    queryKey: ["monitoring-log-stats"],
    queryFn: async () => {
      const res = await api.getMonitoringLogStats();
      return res.data;
    },
  });
}
