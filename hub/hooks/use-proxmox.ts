"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import type {
  ProxmoxConnection,
  ProxmoxNode,
  ProxmoxVM,
  ProxmoxLXC,
  ProxmoxStorage,
  ProxmoxNetwork,
  ProxmoxSnapshot,
  ProxmoxTask,
} from "@/lib/types";

export function useProxmoxConnections(deviceId: string) {
  return useQuery<{ connections: ProxmoxConnection[] }>({
    queryKey: ["proxmox-connections", deviceId],
    queryFn: () => api.listProxmoxConnections(deviceId),
    refetchInterval: 60_000,
  });
}

export function useProxmoxNodes(deviceId: string) {
  return useQuery<{ nodes: ProxmoxNode[] }>({
    queryKey: ["proxmox-nodes", deviceId],
    queryFn: () => api.listProxmoxNodes(deviceId),
    refetchInterval: 30_000,
  });
}

export function useProxmoxVMs(deviceId: string) {
  return useQuery<{ vms: ProxmoxVM[] }>({
    queryKey: ["proxmox-vms", deviceId],
    queryFn: () => api.listProxmoxVMs(deviceId),
    refetchInterval: 30_000,
  });
}

export function useProxmoxLXCs(deviceId: string) {
  return useQuery<{ lxcs: ProxmoxLXC[] }>({
    queryKey: ["proxmox-lxcs", deviceId],
    queryFn: () => api.listProxmoxLXCs(deviceId),
    refetchInterval: 30_000,
  });
}

export function useProxmoxStorage(deviceId: string) {
  return useQuery<{ storage: ProxmoxStorage[] }>({
    queryKey: ["proxmox-storage", deviceId],
    queryFn: () => api.listProxmoxStorage(deviceId),
    refetchInterval: 60_000,
  });
}

export function useProxmoxNetworks(deviceId: string) {
  return useQuery<{ networks: ProxmoxNetwork[] }>({
    queryKey: ["proxmox-networks", deviceId],
    queryFn: () => api.listProxmoxNetworks(deviceId),
    refetchInterval: 60_000,
  });
}

export function useProxmoxSnapshots(deviceId: string) {
  return useQuery<{ snapshots: ProxmoxSnapshot[] }>({
    queryKey: ["proxmox-snapshots", deviceId],
    queryFn: () => api.listProxmoxSnapshots(deviceId),
    refetchInterval: 60_000,
  });
}

export function useProxmoxTasks(deviceId: string) {
  return useQuery<{ tasks: ProxmoxTask[] }>({
    queryKey: ["proxmox-tasks", deviceId],
    queryFn: () => api.listProxmoxTasks(deviceId, undefined, 50),
    refetchInterval: 15_000,
  });
}

export function useProxmoxPowerAction(deviceId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: { node_name: string; vm_id: number; action: string; type: string }) =>
      api.proxmoxPowerAction(deviceId, payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["proxmox-vms", deviceId] });
      qc.invalidateQueries({ queryKey: ["proxmox-lxcs", deviceId] });
      qc.invalidateQueries({ queryKey: ["proxmox-nodes", deviceId] });
    },
  });
}
