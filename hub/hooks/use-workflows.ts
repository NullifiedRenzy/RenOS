"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api-client";

export function useWorkflows() {
  return useQuery({
    queryKey: ["workflows"],
    queryFn: async () => {
      const res = await api.getWorkflows();
      return res.data;
    },
  });
}

export function useWorkflow(id: string) {
  return useQuery({
    queryKey: ["workflow", id],
    queryFn: async () => {
      const res = await api.getWorkflow(id);
      return res.data;
    },
    enabled: !!id,
  });
}

export function useWorkflowSteps(id: string) {
  return useQuery({
    queryKey: ["workflow-steps", id],
    queryFn: async () => {
      const res = await api.getWorkflowSteps(id);
      return res.data;
    },
    enabled: !!id,
  });
}

export function useWorkflowRuns(id: string) {
  return useQuery({
    queryKey: ["workflow-runs", id],
    queryFn: async () => {
      const res = await api.getWorkflowRuns(id);
      return res.data;
    },
    enabled: !!id,
  });
}

export function useWorkflowRun(workflowId: string, runId: string) {
  return useQuery({
    queryKey: ["workflow-run", workflowId, runId],
    queryFn: async () => {
      const res = await api.getWorkflowRun(workflowId, runId);
      return res.data;
    },
    enabled: !!workflowId && !!runId,
  });
}

export function useAllWorkflowRuns() {
  return useQuery({
    queryKey: ["workflow-runs-all"],
    queryFn: async () => {
      const res = await api.getAllWorkflowRuns();
      return res.data;
    },
    refetchInterval: 5000,
  });
}

export function useWorkflowTemplates() {
  return useQuery({
    queryKey: ["workflow-templates"],
    queryFn: async () => {
      const res = await api.getWorkflowTemplates();
      return res.data;
    },
  });
}

export function useCreateWorkflow() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (wf: Record<string, unknown>) => api.createWorkflow(wf as any),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["workflows"] }),
  });
}

export function useUpdateWorkflow() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Record<string, unknown> }) => api.updateWorkflow(id, data as any),
    onSuccess: (_d, vars) => {
      qc.invalidateQueries({ queryKey: ["workflows"] });
      qc.invalidateQueries({ queryKey: ["workflow", vars.id] });
    },
  });
}

export function useDeleteWorkflow() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteWorkflow(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["workflows"] }),
  });
}

export function useToggleWorkflow() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, enabled }: { id: string; enabled: boolean }) => api.toggleWorkflow(id, enabled),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["workflows"] }),
  });
}

export function useExecuteWorkflow() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, triggerData }: { id: string; triggerData?: Record<string, unknown> }) =>
      api.executeWorkflow(id, triggerData),
    onSuccess: (_d, vars) => {
      qc.invalidateQueries({ queryKey: ["workflow-runs", vars.id] });
      qc.invalidateQueries({ queryKey: ["workflow-runs-all"] });
    },
  });
}

export function useCreateWorkflowStep() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ workflowId, step }: { workflowId: string; step: Record<string, unknown> }) =>
      api.createWorkflowStep(workflowId, step as any),
    onSuccess: (_d, vars) => {
      qc.invalidateQueries({ queryKey: ["workflow-steps", vars.workflowId] });
    },
  });
}

export function useCreateWorkflowFromTemplate() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (templateId: string) => api.createWorkflowFromTemplate(templateId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["workflows"] }),
  });
}
