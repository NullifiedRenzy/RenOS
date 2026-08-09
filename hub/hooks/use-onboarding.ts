"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api-client";

export function useOnboarding() {
  const queryClient = useQueryClient();

  const statusQuery = useQuery({
    queryKey: ["onboarding-status"],
    queryFn: () => api.getOnboardingStatus(),
    staleTime: 60_000,
    retry: 1,
  });

  const completeMutation = useMutation({
    mutationFn: () => api.completeOnboarding(),
    onSuccess: (status) => {
      queryClient.setQueryData(["onboarding-status"], status);
    },
  });

  const refresh = () => queryClient.invalidateQueries({ queryKey: ["onboarding-status"] });

  return {
    status: statusQuery.data,
    loading: statusQuery.isLoading,
    error: statusQuery.error,
    markComplete: completeMutation.mutateAsync,
    completing: completeMutation.isPending,
    refresh,
  };
}
