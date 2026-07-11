"use client";

import { useState, useCallback } from "react";
import { useRequestFileAction, useFileActionResult, parseFileEntries, formatFileSize, formatTimestamp } from "@/hooks/use-file-actions";
import type { FileEntry } from "@/lib/types";

interface FileBrowserProps {
  deviceId: string;
}

const FILE_TYPE_ICONS: Record<string, string> = {
  dir: "\uD83D\uDCC1",
  file: "\uD83D\uDCC4",
  symlink: "\uD83D\uDD17",
};

function entryIcon(entry: FileEntry): string {
  return FILE_TYPE_ICONS[entry.file_type] ?? "\u2753";
}

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
  };
  return icons[ext] ?? (name.startsWith(".") ? "\uD83D\uDD12" : "");
}

function fileIcon(entry: FileEntry): string {
  if (entry.file_type === "dir") return FILE_TYPE_ICONS.dir;
  const ext = entry.name.split(".").pop()?.toLowerCase() ?? "";
  return entry.file_type === "symlink"
    ? FILE_TYPE_ICONS.symlink
    : ext === entry.name
      ? "\uD83D\uDCC4"
      : (extensionIcon(entry.name) || "\uD83D\uDCC4");
}

export function FileBrowser({ deviceId }: FileBrowserProps) {
  const requestAction = useRequestFileAction(deviceId);
  const [currentPath, setCurrentPath] = useState("/");
  const [history, setHistory] = useState<string[]>([]);
  const [searchQuery, setSearchQuery] = useState("");
  const [actionId, setActionId] = useState<string | null>(null);
  const [browseError, setBrowseError] = useState<string | null>(null);

  const { data: actionResult, isFetching: isPolling } = useFileActionResult(actionId, true);

  const entries = parseFileEntries(actionResult?.file_action?.result_json);
  const isLoading = requestAction.isPending || (!!actionId && (!actionResult || actionResult.file_action.status === "pending" || actionResult.file_action.status === "running"));

  const navigateTo = useCallback((path: string) => {
    setHistory(prev => [...prev, currentPath]);
    setCurrentPath(path);
    setActionId(null);
    setBrowseError(null);
    requestAction.mutate(
      { actionType: "list_directory", path },
      {
        onSuccess: (result) => {
          setActionId(result.file_action.id);
        },
        onError: (err) => {
          setBrowseError(err instanceof Error ? err.message : "Request failed");
        },
      },
    );
  }, [currentPath, requestAction]);

  const goBack = useCallback(() => {
    if (history.length === 0) return;
    const prev = history[history.length - 1];
    setHistory(prev => prev.slice(0, -1));
    setCurrentPath(prev);
    setActionId(null);
    setBrowseError(null);
    requestAction.mutate(
      { actionType: "list_directory", path: prev },
      {
        onSuccess: (result) => {
          setActionId(result.file_action.id);
        },
        onError: (err) => {
          setBrowseError(err instanceof Error ? err.message : "Request failed");
        },
      },
    );
  }, [history, requestAction]);

  const handleSearch = useCallback(() => {
    if (!searchQuery.trim()) return;
    setActionId(null);
    setBrowseError(null);
    requestAction.mutate(
      { actionType: "search_files", path: currentPath, query: searchQuery },
      {
        onSuccess: (result) => {
          setActionId(result.file_action.id);
        },
        onError: (err) => {
          setBrowseError(err instanceof Error ? err.message : "Request failed");
        },
      },
    );
  }, [currentPath, searchQuery, requestAction]);

  const refresh = useCallback(() => {
    setActionId(null);
    setBrowseError(null);
    requestAction.mutate(
      { actionType: "list_directory", path: currentPath },
      {
        onSuccess: (result) => {
          setActionId(result.file_action.id);
        },
        onError: (err) => {
          setBrowseError(err instanceof Error ? err.message : "Request failed");
        },
      },
    );
  }, [currentPath, requestAction]);

  const initialBrowse = useCallback(() => {
    setActionId(null);
    setBrowseError(null);
    setCurrentPath("/");
    setHistory([]);
    setSearchQuery("");
    requestAction.mutate(
      { actionType: "list_directory", path: "/" },
      {
        onSuccess: (result) => {
          setActionId(result.file_action.id);
        },
        onError: (err) => {
          setBrowseError(err instanceof Error ? err.message : "Request failed");
        },
      },
    );
  }, [requestAction]);

  return (
    <div className="card">
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
          {history.length > 0 && (
            <button
              onClick={goBack}
              disabled={isLoading}
              className="btn-ghost text-body-small"
            >
              &larr; Back
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
          <button
            onClick={handleSearch}
            disabled={isLoading || !searchQuery.trim()}
            className="btn-ghost text-body-small"
          >
            Search
          </button>
          <button
            onClick={refresh}
            disabled={isLoading}
            className="btn-ghost text-body-small"
          >
            {isLoading ? "Loading..." : "Refresh"}
          </button>
        </div>
      </div>

      <div className="flex items-center gap-1 mb-3 text-caption text-[var(--color-text-tertiary)] font-mono">
        <span className="text-[var(--color-text-secondary)]">Path:</span>
        <span>{currentPath}</span>
        {actionResult?.file_action?.output && actionResult.file_action.status === "completed" && !isPolling && (
          <span className="ml-2 text-[var(--color-text-tertiary)]">
            ({actionResult.file_action.output})
          </span>
        )}
      </div>

      {browseError && (
        <p className="text-body text-[var(--color-error)] py-3 text-center">
          {browseError}
        </p>
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
        <p className="text-body text-[var(--color-text-tertiary)] py-3 text-center">
          No files found.
        </p>
      )}

      {!browseError && !isLoading && entries.length > 0 && (
        <div className="overflow-x-auto">
          <table className="w-full text-left">
            <thead>
              <tr className="text-caption text-[var(--color-text-tertiary)] border-b border-[var(--color-border)]">
                <th className="pb-2 pr-2 font-medium w-6" />
                <th className="pb-2 pr-3 font-medium">Name</th>
                <th className="pb-2 pr-3 font-medium text-right w-20">Size</th>
                <th className="pb-2 pr-3 font-medium w-24 hidden sm:table-cell">Permissions</th>
                <th className="pb-2 font-medium w-36 hidden md:table-cell">Modified</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((entry, i) => (
                <tr
                  key={`${entry.path}-${i}`}
                  className="border-b border-[var(--color-border)] last:border-b-0 hover:bg-[var(--color-bg-secondary)] cursor-pointer"
                  onClick={() => {
                    if (entry.file_type === "dir") {
                      navigateTo(entry.path);
                    }
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
    </div>
  );
}
