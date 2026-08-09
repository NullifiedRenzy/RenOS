"use client";

import { useQuery } from "@tanstack/react-query";
import { api, LIST_LIMIT } from "@/lib/api-client";

export function useSchedules() {
  return useQuery({
    queryKey: ["schedules"],
    queryFn: () => api.listSchedules(LIST_LIMIT),
    refetchInterval: 30_000,
    staleTime: 10_000,
  });
}

export function useSchedule(id: string) {
  return useQuery({
    queryKey: ["schedule", id],
    queryFn: () => api.getSchedule(id),
    enabled: !!id,
    refetchInterval: 15_000,
    staleTime: 5_000,
  });
}

export function useScheduleRuns(id: string) {
  return useQuery({
    queryKey: ["schedule-runs", id],
    queryFn: () => api.listScheduleRuns(id),
    enabled: !!id,
    refetchInterval: 10_000,
    staleTime: 5_000,
  });
}
