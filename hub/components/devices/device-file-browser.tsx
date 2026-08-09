"use client";

import { useState, useCallback, useRef, useEffect } from "react";
import {
  useRequestFileAction,
  useFileActionResult,
  parseFileEntries,
  uploadFileInChunks,
} from "@/hooks/use-file-actions";
import { formatFileSize, formatTimestamp } from "@/lib/format";
import { api } from "@/lib/api-client";
import { useDialogEscape } from "@/hooks/use-dialog";
import type { FileEntry } from "@/lib/types";

interface FileBrowserProps {
  deviceId: string;
}

const FILE_TYPE_ICONS: Record<string, string> = {
  dir: "\uD83D\uDCC1",
  file: "\uD83D\uDCC4",
  symlink: "\uD83D\uDD17",
};

function extensionIcon(name: string): string {
  const ext = name.split(".").pop()?.toLowerCase() ?? "";
  const icons: Record<string, string> = {
    ts: "\uD83D\uDEE0\uFE0F",
    tsx: "\uD83D\uDEE0\uFE0F",
    js: "\uD83D\uDCBB",
    jsx: "\uD83D\uDCBB",
    json: "\uD83D\uDCCB",
    md: "\uD83D\uDCDD",
    txt: "\uD83D\uDCDD",
    yml: "\u2699\uFE0F",
    yaml: "\u2699\uFE0F",
    toml: "\u2699\uFE0F",
    rs: "\u26A1",
    go: "\uD83D\uDC0C",
    sql: "\uD83D\uDDC3\uFE0F",
    css: "\uD83C\uDFA8",
    html: "\uD83C\uDF10",
    sh: "\uD83D\uDCBB",
    py: "\uD83D\uDC0D",
    log: "\uD83D\uDCDD",
    conf: "\u2699\uFE0F",
    cfg: "\u2699\uFE0F",
    lock: "\uD83D\uDD12",
    gitignore: "\uD83D\uDD12",
    dockerignore: "\uD83D\uDD12",
    png: "\uD83D\uDDBC\uFE0F",
    jpg: "\uD83D\uDDBC\uFE0F",
    jpeg: "\uD83D\uDDBC\uFE0F",
    gif: "\uD83D\uDDBC\uFE0F",
    webp: "\uD83D\uDDBC\uFE0F",
    pdf: "\uD83D\uDCC4",
    zip: "\uD83D\uDCE6",
    tar: "\uD83D\uDCE6",
    gz: "\uD83D\uDCE6",
  };
  return icons[ext] ?? (name.startsWith(".") ? "\uD83D\uDD12" : "");
}

function fileIcon(entry: FileEntry): string {
  if (entry.file_type === "dir") return FILE_TYPE_ICONS.dir;
  if (entry.file_type === "symlink") return FILE_TYPE_ICONS.symlink;
  return extensionIcon(entry.name) || "\uD83D\uDCC4";
}

type ContextMenuAction = "open" | "preview" | "rename" | "delete" | "copy" | "move" | "download" | "permissions" | "new_folder";

interface ContextMenuState {
  visible: boolean;
  x: number;
  y: number;
  entry: FileEntry | null;
}

interface UploadProgress {
  filename: string;
  uploaded: number;
  total: number;
  status: "uploading" | "done" | "error";
  error?: string;
}

interface RenameDialog {
  visible: boolean;
  entry: FileEntry | null;
  newName: string;
}

interface NewFolderDialog {
  visible: boolean;
  name: string;
}

interface PreviewDialog {
  visible: boolean;
  entry: FileEntry | null;
  content: string | null;
  loading: boolean;
}

export function FileBrowser({ deviceId }: FileBrowserProps) {
  const requestAction = useRequestFileAction(deviceId);
  const [currentPath, setCurrentPath] = useState("/");
  const [history, setHistory] = useState<string[]>([]);
  const [searchQuery, setSearchQuery] = useState("");
  const [actionId, setActionId] = useState<string | null>(null);
  const [browseError, setBrowseError] = useState<string | null>(null);
  const [selectedEntries, setSelectedEntries] = useState<Set<number>>(new Set());
  const [contextMenu, setContextMenu] = useState<ContextMenuState>({ visible: false, x: 0, y: 0, entry: null });
  const [uploads, setUploads] = useState<UploadProgress[]>([]);
  const [renameDialog, setRenameDialog] = useState<RenameDialog>({ visible: false, entry: null, newName: "" });
  const [newFolderDialog, setNewFolderDialog] = useState<NewFolderDialog>({ visible: false, name: "" });
  const [previewDialog, setPreviewDialog] = useState<PreviewDialog>({ visible: false, entry: null, content: null, loading: false });
  const renameDialogRef = useDialogEscape(() => setRenameDialog({ visible: false, entry: null, newName: "" }), renameDialog.visible && !!renameDialog.entry);
  const newFolderDialogRef = useDialogEscape(() => setNewFolderDialog({ visible: false, name: "" }), newFolderDialog.visible);
  const previewDialogRef = useDialogEscape(() => setPreviewDialog({ visible: false, entry: null, content: null, loading: false }), previewDialog.visible);
  const [operationStatus, setOperationStatus] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const dropZoneRef = useRef<HTMLDivElement>(null);

  const { data: actionResult, isFetching: isPolling } = useFileActionResult(actionId, true);
  const entries = parseFileEntries(actionResult?.file_action?.result_json);
  const isLoading = requestAction.isPending || (!!actionId && (!actionResult || actionResult.file_action.status === "pending" || actionResult.file_action.status === "running"));

  // Close context menu on click outside
  useEffect(() => {
    const handler = () => setContextMenu(prev => ({ ...prev, visible: false }));
    if (contextMenu.visible) {
      document.addEventListener("click", handler);
      return () => document.removeEventListener("click", handler);
    }
  }, [contextMenu.visible]);

  const browsePath = useCallback((path: string) => {
    setActionId(null);
    setBrowseError(null);
    setSelectedEntries(new Set());
    requestAction.mutate(
      { actionType: "list_directory", path },
      {
        onSuccess: (result) => setActionId(result.file_action.id),
        onError: (err) => setBrowseError(err instanceof Error ? err.message : "Request failed"),
      },
    );
  }, [requestAction]);

  const navigateTo = useCallback((path: string) => {
    setHistory(prev => [...prev, currentPath]);
    setCurrentPath(path);
    browsePath(path);
  }, [currentPath, browsePath]);

  const goBack = useCallback(() => {
    if (history.length === 0) return;
    const prev = history[history.length - 1];
    setHistory(h => h.slice(0, -1));
    setCurrentPath(prev);
    browsePath(prev);
  }, [history, browsePath]);

  const goUp = useCallback(() => {
    const parts = currentPath.split("/").filter(Boolean);
    if (parts.length <= 1) {
      setCurrentPath("/");
      setHistory([]);
      browsePath("/");
    } else {
      const parent = "/" + parts.slice(0, -1).join("/");
      setCurrentPath(parent);
      setHistory(h => [...h, currentPath]);
      browsePath(parent);
    }
  }, [currentPath, browsePath]);

  const handleSearch = useCallback(() => {
    if (!searchQuery.trim()) return;
    setActionId(null);
    setBrowseError(null);
    requestAction.mutate(
      { actionType: "search_files", path: currentPath, query: searchQuery },
      {
        onSuccess: (result) => setActionId(result.file_action.id),
        onError: (err) => setBrowseError(err instanceof Error ? err.message : "Request failed"),
      },
    );
  }, [currentPath, searchQuery, requestAction]);

  const refresh = useCallback(() => browsePath(currentPath), [currentPath, browsePath]);

  const initialBrowse = useCallback(() => {
    setHistory([]);
    setCurrentPath("/");
    setSearchQuery("");
    setSelectedEntries(new Set());
    browsePath("/");
  }, [browsePath]);

  // Context menu
  const handleContextMenu = useCallback((e: React.MouseEvent, entry: FileEntry | null) => {
    e.preventDefault();
    e.stopPropagation();
    setContextMenu({ visible: true, x: e.clientX, y: e.clientY, entry });
  }, []);

  const executeFileAction = useCallback((actionType: "list_directory" | "read_file" | "search_files" | "create_directory" | "delete_path" | "rename_path" | "move_path" | "copy_path" | "set_permissions" | "upload_file" | "download_file" | "preview_file", path: string, targetPath?: string, query?: string) => {
    setOperationStatus(null);
    requestAction.mutate(
      { actionType, path, targetPath, query },
      {
        onSuccess: (result) => {
          const pollId = result.file_action.id;
          setActionId(pollId);
          const check = setInterval(() => {
            api.getFileAction(pollId).then(r => {
              if (r.file_action.status === "completed") {
                clearInterval(check);
                setOperationStatus(`${actionType} completed`);
                refresh();
              } else if (r.file_action.status === "failed") {
                clearInterval(check);
                setOperationStatus(`Error: ${r.file_action.error || "operation failed"}`);
              }
            });
          }, 2000);
        },
        onError: (err) => setOperationStatus(`Error: ${err instanceof Error ? err.message : "request failed"}`),
      },
    );
  }, [requestAction, refresh]);

  // File operations
  const handleDelete = useCallback((entry: FileEntry) => {
    if (confirm(`Delete ${entry.name}?`)) {
      executeFileAction("delete_path", entry.path);
    }
  }, [executeFileAction]);

  const handleRename = useCallback(() => {
    if (!renameDialog.entry || !renameDialog.newName.trim()) return;
    const parts = renameDialog.entry.path.split("/");
    parts[parts.length - 1] = renameDialog.newName.trim();
    const newPath = parts.join("/");
    executeFileAction("rename_path", renameDialog.entry.path, newPath);
    setRenameDialog({ visible: false, entry: null, newName: "" });
  }, [renameDialog, executeFileAction]);

  const handleNewFolder = useCallback(() => {
    if (!newFolderDialog.name.trim()) return;
    const path = currentPath === "/" ? `/${newFolderDialog.name.trim()}` : `${currentPath}/${newFolderDialog.name.trim()}`;
    executeFileAction("create_directory", path);
    setNewFolderDialog({ visible: false, name: "" });
  }, [currentPath, newFolderDialog, executeFileAction]);

  const handleCopy = useCallback((entry: FileEntry) => {
    const destPath = prompt(`Copy to (current: ${entry.path}):`, entry.path + ".copy");
    if (destPath) executeFileAction("copy_path", entry.path, destPath);
  }, [executeFileAction]);

  const handleMove = useCallback((entry: FileEntry) => {
    const destPath = prompt(`Move to (current: ${entry.path}):`, entry.path);
    if (destPath) executeFileAction("move_path", entry.path, destPath);
  }, [executeFileAction]);

  const handlePreview = useCallback(async (entry: FileEntry) => {
    setPreviewDialog({ visible: true, entry, content: null, loading: true });
    requestAction.mutate(
      { actionType: "preview_file", path: entry.path },
      {
        onSuccess: (result) => {
          const pollId = result.file_action.id;
          const check = setInterval(() => {
            api.getFileAction(pollId).then(r => {
              if (r.file_action.status === "completed") {
                clearInterval(check);
                const resultJson = r.file_action.result_json as { content?: string } | null;
                setPreviewDialog(prev => ({
                  ...prev,
                  content: resultJson?.content || r.file_action.output || "No content",
                  loading: false,
                }));
              } else if (r.file_action.status === "failed") {
                clearInterval(check);
                setPreviewDialog(prev => ({
                  ...prev,
                  content: `Error: ${r.file_action.error}`,
                  loading: false,
                }));
              }
            });
          }, 1000);
        },
        onError: (err) => {
          setPreviewDialog(prev => ({
            ...prev,
            content: `Error: ${err instanceof Error ? err.message : "failed"}`,
            loading: false,
          }));
        },
      },
    );
  }, [requestAction]);

  const handleDownload = useCallback(async (entry: FileEntry) => {
    try {
      setOperationStatus(`Downloading ${entry.name}...`);
      const transfer = await api.createFileTransfer(deviceId, {
        transfer_type: "download",
        filename: entry.name,
        remote_path: entry.path,
        total_size: entry.size,
      });
      const transferId = transfer.file_transfer.id;

      // Agent will pick this up and upload chunks
      // Poll for completion
      const check = setInterval(async () => {
        try {
          const t = await api.getFileTransfer(transferId);
          if (t.file_transfer.status === "completed") {
            clearInterval(check);
            // Download all chunks and reassemble
            const blobParts: ArrayBuffer[] = [];
            for (let i = 0; i < t.file_transfer.total_chunks; i++) {
              const chunk = await api.downloadChunk(transferId, i);
              blobParts.push(chunk);
            }
            const blob = new Blob(blobParts);
            const url = URL.createObjectURL(blob);
            const a = document.createElement("a");
            a.href = url;
            a.download = entry.name;
            a.click();
            URL.revokeObjectURL(url);
            setOperationStatus(`Downloaded ${entry.name}`);
          } else if (t.file_transfer.status === "failed") {
            clearInterval(check);
            setOperationStatus(`Download failed: ${t.file_transfer.error}`);
          }
        } catch {
          // Transfer not ready yet
        }
      }, 2000);
    } catch (err) {
      setOperationStatus(`Download error: ${err instanceof Error ? err.message : "unknown"}`);
    }
  }, [deviceId]);

  // Drag and drop upload
  const handleDrop = useCallback(async (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    const files = Array.from(e.dataTransfer.files);
    if (files.length === 0) return;

    for (const file of files) {
      const remotePath = currentPath === "/" ? `/${file.name}` : `${currentPath}/${file.name}`;
      setUploads(prev => [...prev, { filename: file.name, uploaded: 0, total: 1, status: "uploading" }]);

      try {
        await uploadFileInChunks(
          deviceId,
          file,
          remotePath,
          1024 * 1024,
          (uploaded, total) => {
            setUploads(prev => prev.map(u =>
              u.filename === file.name ? { ...u, uploaded, total } : u
            ));
          },
        );
        setUploads(prev => prev.map(u =>
          u.filename === file.name ? { ...u, status: "done" } : u
        ));
      } catch (err) {
        setUploads(prev => prev.map(u =>
          u.filename === file.name ? { ...u, status: "error", error: err instanceof Error ? err.message : "upload failed" } : u
        ));
      }
    }

    setTimeout(() => {
      setUploads([]);
      refresh();
    }, 3000);
  }, [deviceId, currentPath, refresh]);

  const handleFileInput = useCallback(async (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files || []);
    if (files.length === 0) return;

    for (const file of files) {
      const remotePath = currentPath === "/" ? `/${file.name}` : `${currentPath}/${file.name}`;
      setUploads(prev => [...prev, { filename: file.name, uploaded: 0, total: 1, status: "uploading" }]);

      try {
        await uploadFileInChunks(
          deviceId,
          file,
          remotePath,
          1024 * 1024,
          (uploaded, total) => {
            setUploads(prev => prev.map(u =>
              u.filename === file.name ? { ...u, uploaded, total } : u
            ));
          },
        );
        setUploads(prev => prev.map(u =>
          u.filename === file.name ? { ...u, status: "done" } : u
        ));
      } catch (err) {
        setUploads(prev => prev.map(u =>
          u.filename === file.name ? { ...u, status: "error", error: err instanceof Error ? err.message : "upload failed" } : u
        ));
      }
    }

    setTimeout(() => {
      setUploads([]);
      refresh();
    }, 3000);
  }, [deviceId, currentPath, refresh]);

  // Breadcrumbs
  const pathParts = currentPath.split("/").filter(Boolean);

  return (
    <div className="card">
      {/* Header */}
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <h2 className="text-headline-4 text-[var(--color-text-primary)]">File Browser</h2>
          <button
            onClick={initialBrowse}
            className="btn-ghost text-body-small px-2 py-0.5"
            title="Browse root directory"
          >
            Browse /
          </button>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setNewFolderDialog({ visible: true, name: "" })}
            className="btn-ghost text-body-small"
            title="New folder"
          >
            + Folder
          </button>
          <button
            onClick={() => fileInputRef.current?.click()}
            className="btn-ghost text-body-small"
            title="Upload files"
          >
            Upload
          </button>
          <input
            ref={fileInputRef}
            type="file"
            multiple
            className="hidden"
            onChange={handleFileInput}
          />
          {history.length > 0 && (
            <button onClick={goBack} disabled={isLoading} className="btn-ghost text-body-small">
              &larr; Back
            </button>
          )}
          {pathParts.length > 0 && (
            <button onClick={goUp} disabled={isLoading} className="btn-ghost text-body-small">
              &uarr; Up
            </button>
          )}
          <input
            type="text"
            placeholder="Search files..."
            value={searchQuery}
            onChange={e => setSearchQuery(e.target.value)}
            onKeyDown={e => { if (e.key === "Enter") handleSearch(); }}
            className="px-3 py-1.5 text-body-small rounded-sm border border-[var(--color-border)] bg-[var(--color-bg-primary)] text-[var(--color-text-primary)] placeholder:text-[var(--color-text-tertiary)] outline-none focus:border-[var(--color-accent)] w-44"
          />
          <button onClick={handleSearch} disabled={isLoading || !searchQuery.trim()} className="btn-ghost text-body-small">
            Search
          </button>
          <button onClick={refresh} disabled={isLoading} className="btn-ghost text-body-small">
            {isLoading ? "Loading..." : "Refresh"}
          </button>
        </div>
      </div>

      {/* Breadcrumbs */}
      <div className="flex items-center gap-1 mb-3 text-caption text-[var(--color-text-tertiary)] font-mono flex-wrap">
        <button
          onClick={() => { setCurrentPath("/"); setHistory([]); browsePath("/"); }}
          className="hover:text-[var(--color-accent)] hover:underline"
        >
          /
        </button>
        {pathParts.map((part, i) => {
          const path = "/" + pathParts.slice(0, i + 1).join("/");
          return (
            <span key={i} className="flex items-center gap-1">
              <span className="text-[var(--color-text-tertiary)]">/</span>
              <button
                onClick={() => {
                  setHistory(prev => [...prev, currentPath]);
                  setCurrentPath(path);
                  browsePath(path);
                }}
                className="hover:text-[var(--color-accent)] hover:underline"
              >
                {part}
              </button>
            </span>
          );
        })}
        {actionResult?.file_action?.output && actionResult.file_action.status === "completed" && !isPolling && (
          <span className="ml-2 text-[var(--color-text-tertiary)]">
            ({actionResult.file_action.output})
          </span>
        )}
      </div>

      {/* Upload progress */}
      {uploads.length > 0 && (
        <div className="mb-3 space-y-1">
          {uploads.map((u, i) => (
            <div key={i} className="flex items-center gap-2 text-body-small">
              <span className="text-[var(--color-text-secondary)]">{u.filename}</span>
              {u.status === "uploading" && (
                <div className="flex-1 h-1.5 bg-[var(--color-bg-secondary)] rounded-full overflow-hidden">
                  <div
                    className="h-full bg-[var(--color-accent)] transition-all"
                    style={{ width: `${u.total > 0 ? (u.uploaded / u.total) * 100 : 0}%` }}
                  />
                </div>
              )}
              {u.status === "done" && <span className="text-[var(--color-success)]">Done</span>}
              {u.status === "error" && <span className="text-[var(--color-error)]">{u.error}</span>}
            </div>
          ))}
        </div>
      )}

      {/* Operation status */}
      {operationStatus && (
        <div className="mb-3 text-body-small text-[var(--color-accent)] bg-[var(--color-bg-secondary)] p-2 rounded-sm">
          {operationStatus}
          <button onClick={() => setOperationStatus(null)} className="ml-2 text-[var(--color-text-tertiary)]">&times;</button>
        </div>
      )}

      {browseError && (
        <p className="text-body text-[var(--color-error)] py-3 text-center">{browseError}</p>
      )}

      {!browseError && isLoading && (
        <div className="space-y-1">
          <div className="skeleton h-7 w-full" />
          <div className="skeleton h-7 w-full" />
          <div className="skeleton h-7 w-full" />
          <div className="skeleton h-7 w-3/4" />
        </div>
      )}

      {!browseError && !isLoading && entries.length === 0 && actionResult?.file_action?.status === "completed" && (
        <p className="text-body text-[var(--color-text-tertiary)] py-3 text-center">No files found.</p>
      )}

      {/* Drop zone */}
      {!browseError && !isLoading && entries.length > 0 && (
        <div
          ref={dropZoneRef}
          onDragOver={e => { e.preventDefault(); e.stopPropagation(); }}
          onDrop={handleDrop}
          className="overflow-x-auto"
        >
          <table className="w-full text-left">
            <thead>
              <tr className="text-caption text-[var(--color-text-tertiary)] border-b border-[var(--color-border)]">
                <th scope="col" className="pb-2 pr-2 font-medium w-6" />
                <th scope="col" className="pb-2 pr-3 font-medium">Name</th>
                <th scope="col" className="pb-2 pr-3 font-medium text-right w-20">Size</th>
                <th scope="col" className="pb-2 pr-3 font-medium w-24 hidden sm:table-cell">Permissions</th>
                <th scope="col" className="pb-2 font-medium w-36 hidden md:table-cell">Modified</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((entry, i) => (
                <tr
                  key={`${entry.path}-${i}`}
                  className={`border-b border-[var(--color-border)] last:border-b-0 hover:bg-[var(--color-bg-secondary)] cursor-pointer ${
                    selectedEntries.has(i) ? "bg-[var(--color-accent-light)]" : ""
                  }`}
                  onClick={(e) => {
                    if (e.ctrlKey || e.metaKey) {
                      setSelectedEntries(prev => {
                        const next = new Set(prev);
                        if (next.has(i)) next.delete(i);
                        else next.add(i);
                        return next;
                      });
                    } else {
                      setSelectedEntries(new Set());
                      if (entry.file_type === "dir") navigateTo(entry.path);
                    }
                  }}
                  onContextMenu={(e) => handleContextMenu(e, entry)}
                  onDoubleClick={() => {
                    if (entry.file_type === "dir") navigateTo(entry.path);
                    else handlePreview(entry);
                  }}
                >
                  <td className="py-1.5 pr-2 text-center text-body-small">
                    {fileIcon(entry)}
                  </td>
                  <td className="py-1.5 pr-3">
                    <span className="text-body-small font-mono text-[var(--color-text-primary)]">
                      {entry.name}
                    </span>
                  </td>
                  <td className="py-1.5 pr-3 text-right">
                    <span className="text-body-small text-[var(--color-text-secondary)]">
                      {entry.file_type === "dir" ? "\u2014" : formatFileSize(entry.size)}
                    </span>
                  </td>
                  <td className="py-1.5 pr-3 hidden sm:table-cell">
                    <span className="text-body-small font-mono text-[var(--color-text-secondary)]">
                      {entry.permissions}
                    </span>
                  </td>
                  <td className="py-1.5 hidden md:table-cell">
                    <span className="text-body-small text-[var(--color-text-tertiary)]">
                      {entry.modified ? formatTimestamp(entry.modified) : "\u2014"}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {!browseError && !isLoading && entries.length > 0 && actionResult?.file_action?.action_type === "search_files" && (
        <p className="text-caption text-[var(--color-text-tertiary)] mt-2 text-center">
          Showing search results. Click &ldquo;Refresh&rdquo; to return to directory view.
        </p>
      )}

      {/* Context menu */}
      {contextMenu.visible && (
        <div
          className="fixed z-50 bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-sm shadow-lg py-1 min-w-[160px]"
          style={{ left: contextMenu.x, top: contextMenu.y }}
          onClick={e => e.stopPropagation()}
        >
          {contextMenu.entry && (
            <>
              {contextMenu.entry.file_type === "dir" && (
                <button
                  className="w-full text-left px-3 py-1.5 text-body-small hover:bg-[var(--color-bg-secondary)]"
                  onClick={() => { navigateTo(contextMenu.entry!.path); setContextMenu(prev => ({ ...prev, visible: false })); }}
                >
                  Open
                </button>
              )}
              <button
                className="w-full text-left px-3 py-1.5 text-body-small hover:bg-[var(--color-bg-secondary)]"
                onClick={() => { handlePreview(contextMenu.entry!); setContextMenu(prev => ({ ...prev, visible: false })); }}
              >
                Preview
              </button>
              <button
                className="w-full text-left px-3 py-1.5 text-body-small hover:bg-[var(--color-bg-secondary)]"
                onClick={() => { handleDownload(contextMenu.entry!); setContextMenu(prev => ({ ...prev, visible: false })); }}
              >
                Download
              </button>
              <div className="border-t border-[var(--color-border)] my-1" />
              <button
                className="w-full text-left px-3 py-1.5 text-body-small hover:bg-[var(--color-bg-secondary)]"
                onClick={() => { setRenameDialog({ visible: true, entry: contextMenu.entry, newName: contextMenu.entry!.name }); setContextMenu(prev => ({ ...prev, visible: false })); }}
              >
                Rename
              </button>
              <button
                className="w-full text-left px-3 py-1.5 text-body-small hover:bg-[var(--color-bg-secondary)]"
                onClick={() => { handleCopy(contextMenu.entry!); setContextMenu(prev => ({ ...prev, visible: false })); }}
              >
                Copy
              </button>
              <button
                className="w-full text-left px-3 py-1.5 text-body-small hover:bg-[var(--color-bg-secondary)]"
                onClick={() => { handleMove(contextMenu.entry!); setContextMenu(prev => ({ ...prev, visible: false })); }}
              >
                Move
              </button>
              <div className="border-t border-[var(--color-border)] my-1" />
              <button
                className="w-full text-left px-3 py-1.5 text-body-small text-[var(--color-error)] hover:bg-[var(--color-bg-secondary)]"
                onClick={() => { handleDelete(contextMenu.entry!); setContextMenu(prev => ({ ...prev, visible: false })); }}
              >
                Delete
              </button>
            </>
          )}
          {!contextMenu.entry && (
            <>
              <button
                className="w-full text-left px-3 py-1.5 text-body-small hover:bg-[var(--color-bg-secondary)]"
                onClick={() => { setNewFolderDialog({ visible: true, name: "" }); setContextMenu(prev => ({ ...prev, visible: false })); }}
              >
                New Folder
              </button>
              <button
                className="w-full text-left px-3 py-1.5 text-body-small hover:bg-[var(--color-bg-secondary)]"
                onClick={() => { fileInputRef.current?.click(); setContextMenu(prev => ({ ...prev, visible: false })); }}
              >
                Upload Files
              </button>
              <button
                className="w-full text-left px-3 py-1.5 text-body-small hover:bg-[var(--color-bg-secondary)]"
                onClick={() => { refresh(); setContextMenu(prev => ({ ...prev, visible: false })); }}
              >
                Refresh
              </button>
            </>
          )}
        </div>
      )}

      {/* Rename dialog */}
      {renameDialog.visible && renameDialog.entry && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setRenameDialog({ visible: false, entry: null, newName: "" })}>
          <div ref={renameDialogRef} role="dialog" aria-modal="true" aria-label="Rename" tabIndex={-1} className="bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-sm p-4 w-96 outline-none" onClick={e => e.stopPropagation()}>
            <h3 className="text-headline-4 text-[var(--color-text-primary)] mb-3">Rename</h3>
            <input
              type="text"
              value={renameDialog.newName}
              onChange={e => setRenameDialog(prev => ({ ...prev, newName: e.target.value }))}
              onKeyDown={e => { if (e.key === "Enter") handleRename(); }}
              className="w-full px-3 py-2 text-body-small rounded-sm border border-[var(--color-border)] bg-[var(--color-bg-primary)] text-[var(--color-text-primary)] outline-none focus:border-[var(--color-accent)] mb-3"
              autoFocus
            />
            <div className="flex justify-end gap-2">
              <button onClick={() => setRenameDialog({ visible: false, entry: null, newName: "" })} className="btn-ghost text-body-small">Cancel</button>
              <button onClick={handleRename} className="btn-primary text-body-small">Rename</button>
            </div>
          </div>
        </div>
      )}

      {/* New folder dialog */}
      {newFolderDialog.visible && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setNewFolderDialog({ visible: false, name: "" })}>
          <div ref={newFolderDialogRef} role="dialog" aria-modal="true" aria-label="New Folder" tabIndex={-1} className="bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-sm p-4 w-96 outline-none" onClick={e => e.stopPropagation()}>
            <h3 className="text-headline-4 text-[var(--color-text-primary)] mb-3">New Folder</h3>
            <input
              type="text"
              value={newFolderDialog.name}
              onChange={e => setNewFolderDialog(prev => ({ ...prev, name: e.target.value }))}
              onKeyDown={e => { if (e.key === "Enter") handleNewFolder(); }}
              placeholder="Folder name"
              className="w-full px-3 py-2 text-body-small rounded-sm border border-[var(--color-border)] bg-[var(--color-bg-primary)] text-[var(--color-text-primary)] placeholder:text-[var(--color-text-tertiary)] outline-none focus:border-[var(--color-accent)] mb-3"
              autoFocus
            />
            <div className="flex justify-end gap-2">
              <button onClick={() => setNewFolderDialog({ visible: false, name: "" })} className="btn-ghost text-body-small">Cancel</button>
              <button onClick={handleNewFolder} className="btn-primary text-body-small">Create</button>
            </div>
          </div>
        </div>
      )}

      {/* Preview dialog */}
      {previewDialog.visible && previewDialog.entry && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setPreviewDialog({ visible: false, entry: null, content: null, loading: false })}>
          <div ref={previewDialogRef} role="dialog" aria-modal="true" aria-label={previewDialog.entry.name} tabIndex={-1} className="bg-[var(--color-bg-primary)] border border-[var(--color-border)] rounded-sm p-4 w-[800px] max-h-[80vh] flex flex-col outline-none" onClick={e => e.stopPropagation()}>
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-headline-4 text-[var(--color-text-primary)] truncate">{previewDialog.entry.name}</h3>
              <button onClick={() => setPreviewDialog({ visible: false, entry: null, content: null, loading: false })} className="text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)]">&times;</button>
            </div>
            <div className="flex-1 overflow-auto">
              {previewDialog.loading ? (
                <div className="skeleton h-40 w-full" />
              ) : (
                <pre className="text-body-small font-mono text-[var(--color-text-primary)] whitespace-pre-wrap break-words bg-[var(--color-bg-secondary)] p-3 rounded-sm max-h-[60vh] overflow-auto">
                  {previewDialog.content}
                </pre>
              )}
            </div>
            <div className="flex justify-end gap-2 mt-3">
              <button onClick={() => handleDownload(previewDialog.entry!)} className="btn-ghost text-body-small">Download</button>
              <button onClick={() => setPreviewDialog({ visible: false, entry: null, content: null, loading: false })} className="btn-primary text-body-small">Close</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
