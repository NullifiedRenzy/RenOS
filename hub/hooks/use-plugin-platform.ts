"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import type {
  InstalledPlugin,
  PluginRoute,
  PluginPage,
  PluginSidebarEntry,
  PluginWorker,
  PluginSchedule,
} from "@/lib/types";

export function useInstalledPlugins() {
  return useQuery({
    queryKey: ["plugin-platform"],
    queryFn: () => api.listInstalledPlugins(),
    refetchInterval: 30_000,
  });
}

export function useInstalledPlugin(name: string) {
  return useQuery({
    queryKey: ["plugin-platform", name],
    queryFn: () => api.getInstalledPlugin(name),
    enabled: !!name,
  });
}

export function useEnablePlugin() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (name: string) => api.enablePlugin(name),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["plugin-platform"] });
    },
  });
}

export function useDisablePlugin() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (name: string) => api.disablePlugin(name),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["plugin-platform"] });
    },
  });
}

export function useRemovePlugin() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (name: string) => api.removePlugin(name),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["plugin-platform"] });
    },
  });
}

export function useUpdatePluginConfig() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ name, config }: { name: string; config: Record<string, unknown> }) =>
      api.updatePluginConfig(name, config),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: ["plugin-platform", variables.name] });
    },
  });
}

export function usePluginRoutes(name: string) {
  return useQuery({
    queryKey: ["plugin-platform", name, "routes"],
    queryFn: () => api.getPluginRoutes(name),
    enabled: !!name,
  });
}

export function usePluginPages(name: string) {
  return useQuery({
    queryKey: ["plugin-platform", name, "pages"],
    queryFn: () => api.getPluginPages(name),
    enabled: !!name,
  });
}

export function usePluginWorkers(name: string) {
  return useQuery({
    queryKey: ["plugin-platform", name, "workers"],
    queryFn: () => api.getPluginWorkers(name),
    enabled: !!name,
    refetchInterval: 10_000,
  });
}

export function usePluginSchedules(name: string) {
  return useQuery({
    queryKey: ["plugin-platform", name, "schedules"],
    queryFn: () => api.getPluginSchedules(name),
    enabled: !!name,
    refetchInterval: 15_000,
  });
}

export function useAllPluginSidebar() {
  return useQuery({
    queryKey: ["plugin-platform", "sidebar"],
    queryFn: () => api.getAllPluginSidebar(),
    refetchInterval: 60_000,
  });
}

export function useAllPluginPages() {
  return useQuery({
    queryKey: ["plugin-platform", "pages"],
    queryFn: () => api.getAllPluginPages(),
    refetchInterval: 60_000,
  });
}

export function useAllPluginRoutes() {
  return useQuery({
    queryKey: ["plugin-platform", "routes"],
    queryFn: () => api.getAllPluginRoutes(),
    refetchInterval: 60_000,
  });
}
