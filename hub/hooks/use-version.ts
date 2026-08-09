"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";

interface VersionInfo {
  version: string;
  commit: string;
  build_time: string;
}

export function useVersion() {
  return useQuery<VersionInfo>({
    queryKey: ["version"],
    queryFn: () => api.getVersion(),
    staleTime: Infinity,
    retry: false,
  });
}
