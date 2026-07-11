"use client";

import { useMutation, useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import type { FileEntry } from "@/lib/types";

interface RequestFileActionParams {
  deviceId: string;
  actionType: "list_directory" | "read_file" | "search_files";
  path: string;
  query?: string;
}

export function useRequestFileAction(deviceId: string) {
  return useMutation({
    mutationFn: async ({ actionType, path, query }: Omit<RequestFileActionParams, "deviceId">) => {
      return api.requestFileAction(deviceId, actionType, path, query);
    },
  });
}

export function useFileActionResult(actionId: string | null, enabled: boolean) {
  return useQuery({
    queryKey: ["file-action", actionId],
    queryFn: async () => {
      if (!actionId) throw new Error("no action id");
      return api.getFileAction(actionId);
    },
    enabled: !!actionId && enabled,
    refetchInterval: (query) => {
      const data = query.state.data;
      if (!data || data.file_action.status === "completed" || data.file_action.status === "failed") {
        return false;
      }
      return 3000;
    },
  });
}

export function parseFileEntries(resultJson: unknown): FileEntry[] {
  if (!resultJson) return [];
  if (Array.isArray(resultJson)) return resultJson as FileEntry[];
  return [];
}

export function formatFileSize(bytes: number): string {
  if (bytes === 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

export function formatTimestamp(timestamp: string): string {
  const d = new Date(parseFloat(timestamp) * 1000);
  return d.toLocaleString();
}
