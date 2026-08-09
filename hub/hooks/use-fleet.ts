"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api-client";

export function useFleetDashboard() {
  return useQuery({
    queryKey: ["fleet-dashboard"],
    queryFn: async () => {
      const res = await api.getFleetDashboard();
      return res.data;
    },
  });
}

export function useDeviceTags(deviceId: string) {
  return useQuery({
    queryKey: ["device-tags", deviceId],
    queryFn: async () => {
      const res = await api.getDeviceTags(deviceId);
      return res.data;
    },
    enabled: !!deviceId,
  });
}

export function useSetDeviceTag() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ deviceId, key, value }: { deviceId: string; key: string; value: string }) =>
      api.setDeviceTag(deviceId, key, value),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: ["device-tags", vars.deviceId] });
    },
  });
}

export function useRemoveDeviceTag() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ deviceId, tagKey }: { deviceId: string; tagKey: string }) =>
      api.removeDeviceTag(deviceId, tagKey),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: ["device-tags", vars.deviceId] });
    },
  });
}

export function useTagKeys() {
  return useQuery({
    queryKey: ["tag-keys"],
    queryFn: async () => {
      const res = await api.getTagKeys();
      return res.data;
    },
  });
}

export function useDeviceInventoryList() {
  return useQuery({
    queryKey: ["device-inventory"],
    queryFn: async () => {
      const res = await api.listDeviceInventory();
      return res.data;
    },
  });
}

export function useDeviceInventory(deviceId: string) {
  return useQuery({
    queryKey: ["device-inventory", deviceId],
    queryFn: async () => {
      const res = await api.getDeviceInventory(deviceId);
      return res.data;
    },
    enabled: !!deviceId,
  });
}

export function useComplianceChecks() {
  return useQuery({
    queryKey: ["compliance-checks"],
    queryFn: async () => {
      const res = await api.getComplianceChecks();
      return res.data;
    },
  });
}

export function useCreateComplianceCheck() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (check: Record<string, unknown>) => api.createComplianceCheck(check as any),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["compliance-checks"] }),
  });
}

export function useComplianceResults(deviceId?: string) {
  return useQuery({
    queryKey: ["compliance-results", deviceId],
    queryFn: async () => {
      const res = await api.getComplianceResults(deviceId);
      return res.data;
    },
  });
}

export function useComplianceSummary() {
  return useQuery({
    queryKey: ["compliance-summary"],
    queryFn: async () => {
      const res = await api.getComplianceSummary();
      return res.data;
    },
  });
}

export function useBulkOperations() {
  return useQuery({
    queryKey: ["bulk-operations"],
    queryFn: async () => {
      const res = await api.listBulkOperations();
      return res.data;
    },
    refetchInterval: 5000,
  });
}

export function useCreateBulkOperation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (op: Record<string, unknown>) => api.createBulkOperation(op as any),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["bulk-operations"] }),
  });
}

export function useCancelBulkOperation() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.cancelBulkOperation(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["bulk-operations"] }),
  });
}
