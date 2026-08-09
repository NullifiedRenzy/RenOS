"use client";

import { useState, useEffect, useCallback } from "react";
import { api } from "@/lib/api-client";
import type { ClusterNode, ClusterTask, ClusterMetric, ClusterDashboard } from "@/lib/types";

const statusColors: Record<string, string> = {
  active: "bg-[var(--color-success)]",
  offline: "bg-[var(--color-text-tertiary)]",
  failed: "bg-[var(--color-error)]",
};

const roleColors: Record<string, string> = {
  leader: "text-[var(--color-warning)] font-bold",
  follower: "text-[var(--color-text-secondary)]",
  candidate: "text-[var(--color-info)]",
  standby: "text-[var(--color-text-tertiary)]",
};

const taskStatusColors: Record<string, string> = {
  pending: "text-[var(--color-text-secondary)]",
  assigned: "text-[var(--color-info)]",
  running: "text-[var(--color-warning)]",
  completed: "text-[var(--color-success)]",
  failed: "text-[var(--color-error)]",
};

export default function ClusterPage() {
  const [dashboard, setDashboard] = useState<ClusterDashboard | null>(null);
  const [tasks, setTasks] = useState<ClusterTask[]>([]);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState<"nodes" | "tasks">("nodes");
  const [creatingTask, setCreatingTask] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [dRes, tRes] = await Promise.all([
        api.getClusterDashboard(),
        api.getClusterTasks(),
      ]);
      setDashboard(dRes.data);
      setTasks(tRes.data ?? []);
    } catch { /* ignore */ } finally { setLoading(false); }
  }, []);

  useEffect(() => { load(); }, [load]);

  const rebalance = async () => {
    await api.rebalanceClusterTasks();
    await load();
  };

  if (loading) return <div className="p-6 text-center text-[var(--color-text-tertiary)]">Loading cluster info...</div>;

  const leader = dashboard?.nodes.find(n => n.role === "leader");
  const metrics = dashboard?.metrics ?? [];

  return (
    <div className="p-6 max-w-6xl mx-auto">
      <div className="mb-6">
        <h1 className="text-headline-2 font-semibold text-[var(--color-text-primary)]">Cluster</h1>
        <p className="text-sm text-[var(--color-text-secondary)] mt-1">Multi-node coordination, leader election, and distributed tasks</p>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-4 gap-4 mb-6">
        <div className="bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-xl p-4">
          <div className="text-xs text-[var(--color-text-tertiary)] mb-1">Nodes</div>
          <div className="text-2xl font-bold text-[var(--color-text-primary)]">{dashboard?.nodes.length ?? 0}</div>
        </div>
        <div className="bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-xl p-4">
          <div className="text-xs text-[var(--color-text-tertiary)] mb-1">Leader</div>
          <div className="text-lg font-bold text-[var(--color-warning)]">{leader?.hostname ?? "none"}</div>
        </div>
        <div className="bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-xl p-4">
          <div className="text-xs text-[var(--color-text-tertiary)] mb-1">Term</div>
          <div className="text-2xl font-bold text-[var(--color-text-primary)]">{dashboard?.term ?? 0}</div>
        </div>
        <div className="bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-xl p-4">
          <div className="text-xs text-[var(--color-text-tertiary)] mb-1">Pending Tasks</div>
          <div className="text-2xl font-bold text-[var(--color-text-primary)]">{dashboard?.pending_tasks ?? 0}</div>
        </div>
      </div>

      {/* Tab switch */}
      <div className="flex gap-4 mb-4 border-b border-[var(--color-border)] pb-2">
        {(["nodes", "tasks"] as const).map(t => (
          <button key={t} onClick={() => setTab(t)}
            className={`text-sm font-medium pb-1 border-b-2 transition ${tab === t ? "border-[var(--color-text-primary)] text-[var(--color-text-primary)]" : "border-transparent text-[var(--color-text-tertiary)] hover:text-[var(--color-text-secondary)]"}`}>
            {t === "nodes" ? "Nodes" : "Tasks"}
          </button>
        ))}
        <div className="flex-1" />
        <button onClick={rebalance} className="text-xs bg-[var(--color-surface)] hover:bg-[var(--color-surface-hover)] text-[var(--color-text-secondary)] px-3 py-1 rounded-lg transition">
          Rebalance
        </button>
      </div>

      {tab === "nodes" && (
        <div className="bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-xl overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-[var(--color-border)] text-[var(--color-text-tertiary)]">
                <th scope="col" className="text-left p-3">Status</th>
                <th scope="col" className="text-left p-3">Node ID</th>
                <th scope="col" className="text-left p-3">Hostname</th>
                <th scope="col" className="text-left p-3">Address</th>
                <th scope="col" className="text-left p-3">Role</th>
                <th scope="col" className="text-left p-3">Version</th>
                <th scope="col" className="text-left p-3">Last Heartbeat</th>
              </tr>
            </thead>
            <tbody>
              {dashboard?.nodes.map(node => (
                <tr key={node.id} className="border-b border-[var(--color-border)] hover:bg-[var(--color-surface-hover)]">
                  <td className="p-3">
                    <span className={`inline-block w-2 h-2 rounded-full ${statusColors[node.status]}`} aria-hidden="true" />
                    <span className="sr-only">{node.status}</span>
                  </td>
                  <td className="p-3 font-mono text-[var(--color-text-secondary)]">{node.node_id}</td>
                  <td className="p-3 text-[var(--color-text-primary)]">{node.hostname}</td>
                  <td className="p-3 font-mono text-[var(--color-text-secondary)]">{node.address}:{node.port}</td>
                  <td className={`p-3 ${roleColors[node.role]}`}>{node.role}</td>
                  <td className="p-3 font-mono text-[var(--color-text-secondary)]">{node.version}</td>
                  <td className="p-3 text-[var(--color-text-tertiary)]">{node.last_heartbeat_at ? new Date(node.last_heartbeat_at).toLocaleTimeString() : "—"}</td>
                </tr>
              ))}
              {(!dashboard?.nodes || dashboard.nodes.length === 0) && (
                <tr><td colSpan={7} className="p-6 text-center text-[var(--color-text-tertiary)]">No cluster nodes</td></tr>
              )}
            </tbody>
          </table>
        </div>
      )}

      {tab === "tasks" && (
        <div className="bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-xl overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-[var(--color-border)] text-[var(--color-text-tertiary)]">
                <th scope="col" className="text-left p-3">Status</th>
                <th scope="col" className="text-left p-3">Type</th>
                <th scope="col" className="text-left p-3">Assigned To</th>
                <th scope="col" className="text-left p-3">Priority</th>
                <th scope="col" className="text-left p-3">Retries</th>
                <th scope="col" className="text-left p-3">Created</th>
              </tr>
            </thead>
            <tbody>
              {tasks.map(task => (
                <tr key={task.id} className="border-b border-[var(--color-border)] hover:bg-[var(--color-surface-hover)]">
                  <td className={`p-3 ${taskStatusColors[task.status]}`}>{task.status}</td>
                  <td className="p-3 font-mono text-[var(--color-text-secondary)]">{task.task_type}</td>
                  <td className="p-3 text-[var(--color-text-secondary)]">{task.assigned_to ?? "—"}</td>
                  <td className="p-3 text-[var(--color-text-secondary)]">{task.priority}</td>
                  <td className="p-3 text-[var(--color-text-secondary)]">{task.retries}/{task.max_retries}</td>
                  <td className="p-3 text-[var(--color-text-tertiary)]">{new Date(task.created_at).toLocaleString()}</td>
                </tr>
              ))}
              {tasks.length === 0 && (
                <tr><td colSpan={6} className="p-6 text-center text-[var(--color-text-tertiary)]">No tasks</td></tr>
              )}
            </tbody>
          </table>
        </div>
      )}

      {/* Node Metrics */}
      {metrics.length > 0 && (
        <div className="mt-6">
          <h2 className="text-headline-3 text-[var(--color-text-primary)] mb-3">Node Metrics</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {metrics.map(m => (
              <div key={m.id} className="bg-[var(--color-surface-elevated)] border border-[var(--color-border)] rounded-xl p-4">
                <div className="text-sm text-[var(--color-text-tertiary)] mb-2">{m.node_id}</div>
                <div className="space-y-1 text-xs">
                  <div className="flex justify-between">
                    <span className="text-[var(--color-text-secondary)]">CPU</span>
                    <span className="text-[var(--color-text-primary)] font-mono">{m.cpu_usage.toFixed(1)}%</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-[var(--color-text-secondary)]">Memory</span>
                    <span className="text-[var(--color-text-primary)] font-mono">{m.memory_usage.toFixed(1)}%</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-[var(--color-text-secondary)]">Disk</span>
                    <span className="text-[var(--color-text-primary)] font-mono">{m.disk_usage.toFixed(1)}%</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-[var(--color-text-secondary)]">Tasks</span>
                    <span className="text-[var(--color-text-primary)] font-mono">{m.task_count}</span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
