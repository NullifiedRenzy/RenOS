"use client";

import { useMutation, useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import type { FileEntry, FileTransfer } from "@/lib/types";

interface RequestFileActionParams {
  deviceId: string;
  actionType: "list_directory" | "read_file" | "search_files" | "create_directory" | "delete_path" | "rename_path" | "move_path" | "copy_path" | "set_permissions" | "upload_file" | "download_file" | "preview_file";
  path: string;
  query?: string;
  targetPath?: string;
  metadata?: Record<string, unknown>;
}

export function useRequestFileAction(deviceId: string) {
  return useMutation({
    mutationFn: async ({ actionType, path, query, targetPath, metadata }: Omit<RequestFileActionParams, "deviceId">) => {
      return api.requestFileAction(deviceId, actionType, path, query, targetPath, metadata);
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

export function useListFileActions(deviceId: string) {
  return useQuery({
    queryKey: ["file-actions", deviceId],
    queryFn: () => api.listFileActions(deviceId),
    refetchInterval: 30000,
  });
}

export function parseFileEntries(resultJson: unknown): FileEntry[] {
  if (!resultJson) return [];
  if (Array.isArray(resultJson)) return resultJson as FileEntry[];
  return [];
}

// File Transfer hooks
export function useCreateFileTransfer(deviceId: string) {
  return useMutation({
    mutationFn: (data: {
      transfer_type: "upload" | "download";
      filename: string;
      remote_path: string;
      total_size: number;
      chunk_size?: number;
    }) => api.createFileTransfer(deviceId, data),
  });
}

export function useFileTransfer(transferId: string | null, enabled: boolean) {
  return useQuery({
    queryKey: ["file-transfer", transferId],
    queryFn: async () => {
      if (!transferId) throw new Error("no transfer id");
      return api.getFileTransfer(transferId);
    },
    enabled: !!transferId && enabled,
    refetchInterval: (query) => {
      const data = query.state.data;
      if (!data || data.file_transfer.status === "completed" || data.file_transfer.status === "failed" || data.file_transfer.status === "cancelled") {
        return false;
      }
      return 2000;
    },
  });
}

export function useListFileTransfers(deviceId: string) {
  return useQuery({
    queryKey: ["file-transfers", deviceId],
    queryFn: () => api.listFileTransfers(deviceId),
    refetchInterval: 10000,
  });
}

// Chunked upload helper
export async function uploadFileInChunks(
  deviceId: string,
  file: File,
  remotePath: string,
  chunkSize: number = 1024 * 1024,
  onProgress?: (uploaded: number, total: number) => void,
): Promise<FileTransfer> {
  const totalChunks = Math.ceil(file.size / chunkSize);

  const transfer = await api.createFileTransfer(deviceId, {
    transfer_type: "upload",
    filename: file.name,
    remote_path: remotePath,
    total_size: file.size,
    chunk_size: chunkSize,
  });

  const transferId = transfer.file_transfer.id;

  for (let i = 0; i < totalChunks; i++) {
    const start = i * chunkSize;
    const end = Math.min(start + chunkSize, file.size);
    const chunk = file.slice(start, end);
    const arrayBuffer = await chunk.arrayBuffer();
    await api.uploadChunk(transferId, i, arrayBuffer);
    onProgress?.(i + 1, totalChunks);
  }

  await api.assembleTransfer(transferId);
  return (await api.getFileTransfer(transferId)).file_transfer;
}
