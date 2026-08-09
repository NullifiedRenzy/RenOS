"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api-client";

export function useVaultSecrets(query?: string) {
  return useQuery({
    queryKey: ["vault-secrets", query],
    queryFn: async () => {
      const res = await api.getVaultSecrets(query);
      return res.data;
    },
  });
}

export function useVaultSecret(id: string) {
  return useQuery({
    queryKey: ["vault-secret", id],
    queryFn: async () => {
      const res = await api.getVaultSecret(id);
      return res.data;
    },
    enabled: !!id,
  });
}

export function useVaultSecretVersions(id: string) {
  return useQuery({
    queryKey: ["vault-secret-versions", id],
    queryFn: async () => {
      const res = await api.getVaultSecretVersions(id);
      return res.data;
    },
    enabled: !!id,
  });
}

export function useCreateVaultSecret() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (secret: { name: string; path: string; value: string; description?: string; secret_type?: string }) =>
      api.createVaultSecret(secret),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["vault-secrets"] }),
  });
}

export function useUpdateVaultSecret() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Record<string, unknown> }) =>
      api.updateVaultSecret(id, data as any),
    onSuccess: (_d, vars) => {
      qc.invalidateQueries({ queryKey: ["vault-secrets"] });
      qc.invalidateQueries({ queryKey: ["vault-secret", vars.id] });
    },
  });
}

export function useDeleteVaultSecret() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteVaultSecret(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["vault-secrets"] }),
  });
}

export function useVaultKeys() {
  return useQuery({
    queryKey: ["vault-keys"],
    queryFn: async () => {
      const res = await api.getVaultKeys();
      return res.data;
    },
  });
}

export function useVaultPolicies() {
  return useQuery({
    queryKey: ["vault-policies"],
    queryFn: async () => {
      const res = await api.getVaultPolicies();
      return res.data;
    },
  });
}

export function useVaultAuditLog(secretId?: string) {
  return useQuery({
    queryKey: ["vault-audit", secretId],
    queryFn: async () => {
      const res = await api.getVaultAuditLog(secretId);
      return res.data;
    },
  });
}
