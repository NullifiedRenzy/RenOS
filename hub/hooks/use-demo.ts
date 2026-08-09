"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api-client";

export function useDemoStatus() {
  return useQuery({
    queryKey: ["demo-status"],
    queryFn: () => api.getDemoStatus(),
    staleTime: 60_000,
    retry: 1,
  });
}

export function useSeedDemo() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => api.seedDemo(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["demo-status"] });
      queryClient.invalidateQueries({ queryKey: ["devices"] });
      queryClient.invalidateQueries({ queryKey: ["onboarding-status"] });
    },
  });
}

export function useDemo() {
  const demoStatusQuery = useDemoStatus();
  const seedMutation = useSeedDemo();

  return {
    demoStatus: demoStatusQuery,
    seedDemo: seedMutation,
  };
}
