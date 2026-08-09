"use client";

import { useState } from "react";
import { useLogs } from "@/hooks/use-observability";

const levelOrder = ["error", "warn", "info", "debug"];

export default function LogsPage() {
  const [level, setLevel] = useState<string>("");
  const [source, setSource] = useState<string>("");
  const [offset, setOffset] = useState(0);
  const limit = 100;

  const { data, isLoading, error } = useLogs({ level: level || undefined, source: source || undefined, limit, offset });

  const logs = data?.logs ?? [];
  const total = data?.total ?? 0;

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-headline-1 text-[var(--color-text-primary)]">System Logs</h1>
        <p className="text-body text-[var(--color-text-secondary)] mt-1">
          Centralized log viewer for jobs, policies, and system events
        </p>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-3 items-end">
        <div className="flex flex-col gap-1">
          <label className="text-caption text-[var(--color-text-tertiary)]">Level</label>
          <select
            className="input"
            value={level}
            onChange={(e) => { setLevel(e.target.value); setOffset(0); }}
          >
            <option value="">All</option>
            {levelOrder.map((l) => (
              <option key={l} value={l}>{l.toUpperCase()}</option>
            ))}
          </select>
        </div>
        <div className="flex flex-col gap-1">
          <label className="text-caption text-[var(--color-text-tertiary)]">Source</label>
          <input
            className="input"
            placeholder="e.g. scheduler"
            value={source}
            onChange={(e) => { setSource(e.target.value); setOffset(0); }}
          />
        </div>
      </div>

      {/* Logs table */}
      {isLoading && <div className="skeleton h-48 w-full" />}
      {error && (
        <div className="card text-center py-4">
          <p className="text-body text-[var(--color-error)]">Failed to load logs</p>
        </div>
      )}
      {logs.length === 0 && !isLoading && (
        <div className="card text-center py-6">
          <p className="text-body text-[var(--color-text-tertiary)]">No logs found</p>
        </div>
      )}
      {logs.length > 0 && (
        <div className="card overflow-x-auto">
          <table className="w-full text-body-small">
            <thead>
              <tr className="border-b border-[var(--color-border)] text-caption text-[var(--color-text-tertiary)]">
                <th scope="col" className="text-left p-2">Time</th>
                <th scope="col" className="text-left p-2">Level</th>
                <th scope="col" className="text-left p-2">Source</th>
                <th scope="col" className="text-left p-2">Message</th>
              </tr>
            </thead>
            <tbody>
              {logs.map((log) => (
                <tr key={log.id} className="border-b border-[var(--color-border)] hover:bg-[var(--color-surface)]">
                  <td className="p-2 whitespace-nowrap text-[var(--color-text-secondary)]">
                    {new Date(log.created_at).toLocaleString()}
                  </td>
                  <td className="p-2">
                    <span className={levelColorClass(log.level)}>{log.level.toUpperCase()}</span>
                  </td>
                  <td className="p-2 text-[var(--color-text-secondary)]">{log.source}</td>
                  <td className="p-2 text-[var(--color-text-primary)]">{log.message}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Pagination */}
      {total > limit && (
        <div className="flex items-center justify-between text-body-small text-[var(--color-text-secondary)]">
          <span>Showing {offset + 1}–{Math.min(offset + limit, total)} of {total}</span>
          <div className="flex gap-2">
            <button
              className="btn-ghost px-3 py-1"
              disabled={offset === 0}
              onClick={() => setOffset(Math.max(0, offset - limit))}
            >
              Previous
            </button>
            <button
              className="btn-ghost px-3 py-1"
              disabled={offset + limit >= total}
              onClick={() => setOffset(offset + limit)}
            >
              Next
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

function levelColorClass(level: string) {
  switch (level) {
    case "error": return "text-[var(--color-error)] font-semibold";
    case "warn":  return "text-[var(--color-warning)] font-semibold";
    case "info":  return "text-[var(--color-accent)]";
    default:      return "text-[var(--color-text-tertiary)]";
  }
}
