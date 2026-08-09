"use client";

import { useParams } from "next/navigation";
import Link from "next/link";
import { useDevice } from "@/hooks/use-devices";
import {
  useProxmoxConnections,
  useProxmoxNodes,
  useProxmoxVMs,
  useProxmoxLXCs,
  useProxmoxStorage,
  useProxmoxSnapshots,
  useProxmoxTasks,
  useProxmoxPowerAction,
} from "@/hooks/use-proxmox";
import { formatDate } from "@/lib/utils";
import { formatBytes, formatUptime } from "@/lib/format";
import { useState } from "react";
import { useDialogEscape } from "@/hooks/use-dialog";

const STATUS_COLORS: Record<string, string> = {
  running: "var(--color-success)",
  online: "var(--color-success)",
  connected: "var(--color-success)",
  active: "var(--color-success)",
  stopped: "var(--color-text-tertiary)",
  offline: "var(--color-error)",
  disconnected: "var(--color-text-tertiary)",
  inactive: "var(--color-text-tertiary)",
  paused: "var(--color-warning)",
  suspended: "var(--color-warning)",
  error: "var(--color-error)",
  failed: "var(--color-error)",
  completed: "var(--color-success)",
};

function statusDot(status: string) {
  const color = STATUS_COLORS[status.toLowerCase()] ?? "var(--color-text-tertiary)";
  return (
    <span className="inline-flex items-center gap-1.5">
      <span className="inline-block w-2 h-2 rounded-full flex-shrink-0" style={{ backgroundColor: color }} />
      <span style={{ color }}>{status}</span>
    </span>
  );
}

function ResourceBar({ used, total, label }: { used: number; total: number; label: string }) {
  const pct = total > 0 ? (used / total) * 100 : 0;
  const color = pct > 90 ? "var(--color-error)" : pct > 70 ? "var(--color-warning)" : "var(--color-info)";
  return (
    <div className="flex flex-col gap-0.5">
      <div className="flex justify-between text-[11px]" style={{ color: "var(--color-text-secondary)" }}>
        <span>{label}</span>
        <span>{formatBytes(used)} / {formatBytes(total)}</span>
      </div>
      <div className="w-full h-1.5 rounded-full" style={{ backgroundColor: "var(--color-border)" }}>
        <div className="h-full rounded-full transition-all" style={{ width: `${Math.min(pct, 100)}%`, backgroundColor: color }} />
      </div>
    </div>
  );
}

export default function ProxmoxPage() {
  const params = useParams();
  const deviceId = params.id as string;
  const { data: device } = useDevice(deviceId);
  const { data: connsData } = useProxmoxConnections(deviceId);
  const { data: nodesData } = useProxmoxNodes(deviceId);
  const { data: vmsData } = useProxmoxVMs(deviceId);
  const { data: lxcsData } = useProxmoxLXCs(deviceId);
  const { data: storageData } = useProxmoxStorage(deviceId);
  const { data: snapshotsData } = useProxmoxSnapshots(deviceId);
  const { data: tasksData } = useProxmoxTasks(deviceId);
  const powerAction = useProxmoxPowerAction(deviceId);

  const [selectedNode, setSelectedNode] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<"tree" | "tasks">("tree");
  const [confirmAction, setConfirmAction] = useState<{ node: string; vmId: number; action: string; type: string; name: string } | null>(null);
  const dialogRef = useDialogEscape(() => setConfirmAction(null), !!confirmAction);

  const connections = connsData?.connections ?? [];
  const nodes = nodesData?.nodes ?? [];
  const vms = vmsData?.vms ?? [];
  const lxcs = lxcsData?.lxcs ?? [];
  const storage = storageData?.storage ?? [];
  const snapshots = snapshotsData?.snapshots ?? [];
  const tasks = tasksData?.tasks ?? [];

  const filteredVms = selectedNode ? vms.filter(v => v.node_name === selectedNode) : vms;
  const filteredLxcs = selectedNode ? lxcs.filter(l => l.node_name === selectedNode) : lxcs;
  const filteredStorage = selectedNode ? storage.filter(s => s.node_name === selectedNode) : storage;
  const filteredSnapshots = selectedNode ? snapshots.filter(s => s.node_name === selectedNode) : snapshots;

  const doPowerAction = () => {
    if (!confirmAction) return;
    powerAction.mutate({
      node_name: confirmAction.node,
      vm_id: confirmAction.vmId,
      action: confirmAction.action,
      type: confirmAction.type,
    });
    setConfirmAction(null);
  };

  if (!device) {
    return (
      <div className="p-6" style={{ color: "var(--color-text-secondary)" }}>
        Loading device...
      </div>
    );
  }

  const deviceInfo = device.device;

  return (
    <div className="flex flex-col gap-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Link
            href={`/devices/${deviceId}`}
            className="text-sm px-2 py-1 rounded"
            style={{ color: "var(--color-text-secondary)" }}
          >
            ← Back
          </Link>
          <h1 className="text-lg font-semibold" style={{ color: "var(--color-text-primary)" }}>
            Proxmox VE — {deviceInfo.hostname}
          </h1>
        </div>
        <div className="flex gap-1">
          {(["tree", "tasks"] as const).map(tab => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className="px-3 py-1.5 text-xs rounded font-medium capitalize"
              style={{
                backgroundColor: activeTab === tab ? "var(--color-accent)" : "transparent",
                color: activeTab === tab ? "white" : "var(--color-text-secondary)",
              }}
            >
              {tab === "tree" ? "Cluster" : "Tasks"}
            </button>
          ))}
        </div>
      </div>

      {/* Connection info */}
      {connections.length === 0 && (
        <div className="p-4 rounded-lg text-sm" style={{ backgroundColor: "var(--color-surface)", color: "var(--color-text-secondary)" }}>
          No Proxmox connections configured. Add a connection in the device settings.
        </div>
      )}

      {activeTab === "tree" && (
        <div className="grid gap-4" style={{ gridTemplateColumns: "280px 1fr" }}>
          {/* Tree sidebar */}
          <div className="flex flex-col gap-2 p-3 rounded-lg" style={{ backgroundColor: "var(--color-surface)", border: "1px solid var(--color-border)" }}>
            <div className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--color-text-tertiary)" }}>
              Cluster
            </div>

            {connections.map(conn => (
              <div key={conn.id} className="flex items-center gap-2 px-2 py-1 rounded text-sm" style={{ color: "var(--color-text-primary)" }}>
                {statusDot(conn.status)}
                <span className="font-medium truncate">{conn.name || conn.host}</span>
              </div>
            ))}

            {nodes.length > 0 && (
              <div className="mt-2 border-t pt-2" style={{ borderColor: "var(--color-border)" }}>
                <div className="text-xs font-medium uppercase tracking-wider mb-1" style={{ color: "var(--color-text-tertiary)" }}>
                  Nodes
                </div>
                {nodes.map(node => (
                  <button
                    key={node.id}
                    onClick={() => setSelectedNode(selectedNode === node.node_name ? null : node.node_name)}
                    className="w-full flex items-center gap-2 px-2 py-1.5 rounded text-sm text-left transition-colors"
                    style={{
                      backgroundColor: selectedNode === node.node_name ? "var(--color-accent-subtle)" : "transparent",
                      color: "var(--color-text-primary)",
                    }}
                  >
                    {statusDot(node.status)}
                    <span className="truncate">{node.node_name}</span>
                  </button>
                ))}
              </div>
            )}

            {selectedNode && (
              <button
                onClick={() => setSelectedNode(null)}
                className="mt-1 text-xs px-2 py-1 rounded"
                style={{ color: "var(--color-accent)" }}
              >
                Show all nodes
              </button>
            )}
          </div>

          {/* Main content */}
          <div className="flex flex-col gap-4">
            {/* Node cards */}
            {(selectedNode ? nodes.filter(n => n.node_name === selectedNode) : nodes).map(node => (
              <div key={node.id} className="p-4 rounded-lg" style={{ backgroundColor: "var(--color-surface)", border: "1px solid var(--color-border)" }}>
                <div className="flex items-center justify-between mb-3">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-semibold" style={{ color: "var(--color-text-primary)" }}>{node.node_name}</span>
                    {statusDot(node.status)}
                  </div>
                  <span className="text-xs" style={{ color: "var(--color-text-tertiary)" }}>
                    {node.version} · Uptime {formatUptime(node.uptime, { empty: "down" })}
                  </span>
                </div>
                <div className="grid gap-3" style={{ gridTemplateColumns: "1fr 1fr 1fr" }}>
                  <ResourceBar used={Math.round(node.cpu * node.max_cpu)} total={node.max_cpu} label="CPU" />
                  <ResourceBar used={node.memory} total={node.max_memory} label="Memory" />
                  <ResourceBar used={node.disk} total={node.max_disk} label="Disk" />
                </div>
              </div>
            ))}

            {/* VMs */}
            {filteredVms.length > 0 && (
              <div className="rounded-lg overflow-hidden" style={{ backgroundColor: "var(--color-surface)", border: "1px solid var(--color-border)" }}>
                <div className="px-4 py-2 text-xs font-medium uppercase tracking-wider" style={{ backgroundColor: "var(--color-surface-elevated)", color: "var(--color-text-tertiary)" }}>
                  Virtual Machines ({filteredVms.length})
                </div>
                <table className="w-full text-sm">
                  <thead>
                    <tr style={{ borderBottom: "1px solid var(--color-border)" }}>
                      <th scope="col" className="text-left px-4 py-2 text-xs font-medium" style={{ color: "var(--color-text-tertiary)" }}>VMID</th>
                      <th scope="col" className="text-left px-4 py-2 text-xs font-medium" style={{ color: "var(--color-text-tertiary)" }}>Name</th>
                      <th scope="col" className="text-left px-4 py-2 text-xs font-medium" style={{ color: "var(--color-text-tertiary)" }}>Node</th>
                      <th scope="col" className="text-left px-4 py-2 text-xs font-medium" style={{ color: "var(--color-text-tertiary)" }}>Status</th>
                      <th scope="col" className="text-right px-4 py-2 text-xs font-medium" style={{ color: "var(--color-text-tertiary)" }}>CPU</th>
                      <th scope="col" className="text-right px-4 py-2 text-xs font-medium" style={{ color: "var(--color-text-tertiary)" }}>Memory</th>
                      <th scope="col" className="text-right px-4 py-2 text-xs font-medium" style={{ color: "var(--color-text-tertiary)" }}>Disk</th>
                      <th scope="col" className="text-right px-4 py-2 text-xs font-medium" style={{ color: "var(--color-text-tertiary)" }}>Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredVms.map(vm => (
                      <tr key={vm.id} style={{ borderBottom: "1px solid var(--color-border)" }}>
                        <td className="px-4 py-2 font-mono text-xs" style={{ color: "var(--color-text-secondary)" }}>{vm.vm_id}</td>
                        <td className="px-4 py-2 font-medium" style={{ color: "var(--color-text-primary)" }}>
                          {vm.name || `VM ${vm.vm_id}`}
                          {vm.template && <span className="ml-1 text-xs" style={{ color: "var(--color-text-tertiary)" }}>(template)</span>}
                        </td>
                        <td className="px-4 py-2 text-xs" style={{ color: "var(--color-text-secondary)" }}>{vm.node_name}</td>
                        <td className="px-4 py-2">{statusDot(vm.status)}</td>
                        <td className="px-4 py-2 text-right text-xs" style={{ color: "var(--color-text-secondary)" }}>{vm.cores} cores</td>
                        <td className="px-4 py-2 text-right text-xs" style={{ color: "var(--color-text-secondary)" }}>{formatBytes(vm.memory)}</td>
                        <td className="px-4 py-2 text-right text-xs" style={{ color: "var(--color-text-secondary)" }}>{formatBytes(vm.disk)}</td>
                        <td className="px-4 py-2 text-right">
                          {!vm.template && vm.status === "stopped" && (
                            <button onClick={() => setConfirmAction({ node: vm.node_name, vmId: vm.vm_id, action: "start", type: "qemu", name: vm.name || String(vm.vm_id) })}
                              className="text-xs px-2 py-0.5 rounded" style={{ color: "var(--color-success)" }}>Start</button>
                          )}
                          {!vm.template && vm.status === "running" && (
                            <>
                              <button onClick={() => setConfirmAction({ node: vm.node_name, vmId: vm.vm_id, action: "shutdown", type: "qemu", name: vm.name || String(vm.vm_id) })}
                                className="text-xs px-2 py-0.5 rounded" style={{ color: "var(--color-warning)" }}>Shutdown</button>
                              <button onClick={() => setConfirmAction({ node: vm.node_name, vmId: vm.vm_id, action: "stop", type: "qemu", name: vm.name || String(vm.vm_id) })}
                                className="text-xs px-2 py-0.5 rounded ml-1" style={{ color: "var(--color-error)" }}>Stop</button>
                            </>
                          )}
                          {!vm.template && (vm.status === "running" || vm.status === "stopped") && (
                            <button onClick={() => setConfirmAction({ node: vm.node_name, vmId: vm.vm_id, action: "reboot", type: "qemu", name: vm.name || String(vm.vm_id) })}
                              className="text-xs px-2 py-0.5 rounded ml-1" style={{ color: "var(--color-info)" }}>Reboot</button>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            {/* LXCs */}
            {filteredLxcs.length > 0 && (
              <div className="rounded-lg overflow-hidden" style={{ backgroundColor: "var(--color-surface)", border: "1px solid var(--color-border)" }}>
                <div className="px-4 py-2 text-xs font-medium uppercase tracking-wider" style={{ backgroundColor: "var(--color-surface-elevated)", color: "var(--color-text-tertiary)" }}>
                  LXC Containers ({filteredLxcs.length})
                </div>
                <table className="w-full text-sm">
                  <thead>
                    <tr style={{ borderBottom: "1px solid var(--color-border)" }}>
                      <th scope="col" className="text-left px-4 py-2 text-xs font-medium" style={{ color: "var(--color-text-tertiary)" }}>VMID</th>
                      <th scope="col" className="text-left px-4 py-2 text-xs font-medium" style={{ color: "var(--color-text-tertiary)" }}>Name</th>
                      <th scope="col" className="text-left px-4 py-2 text-xs font-medium" style={{ color: "var(--color-text-tertiary)" }}>Node</th>
                      <th scope="col" className="text-left px-4 py-2 text-xs font-medium" style={{ color: "var(--color-text-tertiary)" }}>Status</th>
                      <th scope="col" className="text-right px-4 py-2 text-xs font-medium" style={{ color: "var(--color-text-tertiary)" }}>CPU</th>
                      <th scope="col" className="text-right px-4 py-2 text-xs font-medium" style={{ color: "var(--color-text-tertiary)" }}>Memory</th>
                      <th scope="col" className="text-right px-4 py-2 text-xs font-medium" style={{ color: "var(--color-text-tertiary)" }}>Disk</th>
                      <th scope="col" className="text-right px-4 py-2 text-xs font-medium" style={{ color: "var(--color-text-tertiary)" }}>Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredLxcs.map(lxc => (
                      <tr key={lxc.id} style={{ borderBottom: "1px solid var(--color-border)" }}>
                        <td className="px-4 py-2 font-mono text-xs" style={{ color: "var(--color-text-secondary)" }}>{lxc.vm_id}</td>
                        <td className="px-4 py-2 font-medium" style={{ color: "var(--color-text-primary)" }}>
                          {lxc.name || `CT ${lxc.vm_id}`}
                          {lxc.template && <span className="ml-1 text-xs" style={{ color: "var(--color-text-tertiary)" }}>(template)</span>}
                        </td>
                        <td className="px-4 py-2 text-xs" style={{ color: "var(--color-text-secondary)" }}>{lxc.node_name}</td>
                        <td className="px-4 py-2">{statusDot(lxc.status)}</td>
                        <td className="px-4 py-2 text-right text-xs" style={{ color: "var(--color-text-secondary)" }}>{lxc.cores} cores</td>
                        <td className="px-4 py-2 text-right text-xs" style={{ color: "var(--color-text-secondary)" }}>{formatBytes(lxc.memory)}</td>
                        <td className="px-4 py-2 text-right text-xs" style={{ color: "var(--color-text-secondary)" }}>{formatBytes(lxc.disk)}</td>
                        <td className="px-4 py-2 text-right">
                          {!lxc.template && lxc.status === "stopped" && (
                            <button onClick={() => setConfirmAction({ node: lxc.node_name, vmId: lxc.vm_id, action: "start", type: "lxc", name: lxc.name || String(lxc.vm_id) })}
                              className="text-xs px-2 py-0.5 rounded" style={{ color: "var(--color-success)" }}>Start</button>
                          )}
                          {!lxc.template && lxc.status === "running" && (
                            <>
                              <button onClick={() => setConfirmAction({ node: lxc.node_name, vmId: lxc.vm_id, action: "shutdown", type: "lxc", name: lxc.name || String(lxc.vm_id) })}
                                className="text-xs px-2 py-0.5 rounded" style={{ color: "var(--color-warning)" }}>Shutdown</button>
                              <button onClick={() => setConfirmAction({ node: lxc.node_name, vmId: lxc.vm_id, action: "stop", type: "lxc", name: lxc.name || String(lxc.vm_id) })}
                                className="text-xs px-2 py-0.5 rounded ml-1" style={{ color: "var(--color-error)" }}>Stop</button>
                            </>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            {/* Storage */}
            {filteredStorage.length > 0 && (
              <div className="rounded-lg overflow-hidden" style={{ backgroundColor: "var(--color-surface)", border: "1px solid var(--color-border)" }}>
                <div className="px-4 py-2 text-xs font-medium uppercase tracking-wider" style={{ backgroundColor: "var(--color-surface-elevated)", color: "var(--color-text-tertiary)" }}>
                  Storage ({filteredStorage.length})
                </div>
                <table className="w-full text-sm">
                  <thead>
                    <tr style={{ borderBottom: "1px solid var(--color-border)" }}>
                      <th scope="col" className="text-left px-4 py-2 text-xs font-medium" style={{ color: "var(--color-text-tertiary)" }}>ID</th>
                      <th scope="col" className="text-left px-4 py-2 text-xs font-medium" style={{ color: "var(--color-text-tertiary)" }}>Type</th>
                      <th scope="col" className="text-left px-4 py-2 text-xs font-medium" style={{ color: "var(--color-text-tertiary)" }}>Node</th>
                      <th scope="col" className="text-left px-4 py-2 text-xs font-medium" style={{ color: "var(--color-text-tertiary)" }}>Status</th>
                      <th scope="col" className="text-left px-4 py-2 text-xs font-medium" style={{ color: "var(--color-text-tertiary)" }}>Content</th>
                      <th scope="col" className="px-4 py-2 text-xs font-medium" style={{ color: "var(--color-text-tertiary)" }}>Usage</th>
                    </tr>
                  </thead>
                  <tbody>
                    {filteredStorage.map(s => (
                      <tr key={s.id} style={{ borderBottom: "1px solid var(--color-border)" }}>
                        <td className="px-4 py-2 font-mono text-xs" style={{ color: "var(--color-text-secondary)" }}>{s.storage_id}</td>
                        <td className="px-4 py-2 text-xs" style={{ color: "var(--color-text-secondary)" }}>{s.storage_type}</td>
                        <td className="px-4 py-2 text-xs" style={{ color: "var(--color-text-secondary)" }}>{s.node_name}</td>
                        <td className="px-4 py-2">{statusDot(s.status)}</td>
                        <td className="px-4 py-2 text-xs" style={{ color: "var(--color-text-secondary)" }}>{s.content}</td>
                        <td className="px-4 py-2 w-48">
                          <ResourceBar used={s.used} total={s.total} label="" />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            {nodes.length === 0 && connections.length > 0 && (
              <div className="p-4 rounded-lg text-sm" style={{ backgroundColor: "var(--color-surface)", color: "var(--color-text-secondary)" }}>
                Connected but no data received yet. The agent will sync shortly.
              </div>
            )}
          </div>
        </div>
      )}

      {activeTab === "tasks" && (
        <div className="rounded-lg overflow-hidden" style={{ backgroundColor: "var(--color-surface)", border: "1px solid var(--color-border)" }}>
          <table className="w-full text-sm">
            <thead>
              <tr style={{ borderBottom: "1px solid var(--color-border)" }}>
                <th scope="col" className="text-left px-4 py-2 text-xs font-medium" style={{ color: "var(--color-text-tertiary)" }}>Type</th>
                <th scope="col" className="text-left px-4 py-2 text-xs font-medium" style={{ color: "var(--color-text-tertiary)" }}>Node</th>
                <th scope="col" className="text-left px-4 py-2 text-xs font-medium" style={{ color: "var(--color-text-tertiary)" }}>Status</th>
                <th scope="col" className="text-left px-4 py-2 text-xs font-medium" style={{ color: "var(--color-text-tertiary)" }}>User</th>
                <th scope="col" className="text-left px-4 py-2 text-xs font-medium" style={{ color: "var(--color-text-tertiary)" }}>Started</th>
                <th scope="col" className="text-left px-4 py-2 text-xs font-medium" style={{ color: "var(--color-text-tertiary)" }}>Ended</th>
                <th scope="col" className="text-left px-4 py-2 text-xs font-medium" style={{ color: "var(--color-text-tertiary)" }}>Exit</th>
              </tr>
            </thead>
            <tbody>
              {tasks.map(task => (
                <tr key={task.id} style={{ borderBottom: "1px solid var(--color-border)" }}>
                  <td className="px-4 py-2 font-mono text-xs" style={{ color: "var(--color-text-secondary)" }}>{task.task_type}</td>
                  <td className="px-4 py-2 text-xs" style={{ color: "var(--color-text-secondary)" }}>{task.node_name}</td>
                  <td className="px-4 py-2">{statusDot(task.status)}</td>
                  <td className="px-4 py-2 text-xs" style={{ color: "var(--color-text-secondary)" }}>{task.user_name}</td>
                  <td className="px-4 py-2 text-xs" style={{ color: "var(--color-text-secondary)" }}>
                    {task.start_time > 0 ? new Date(task.start_time * 1000).toLocaleString() : "—"}
                  </td>
                  <td className="px-4 py-2 text-xs" style={{ color: "var(--color-text-secondary)" }}>
                    {task.end_time > 0 ? new Date(task.end_time * 1000).toLocaleString() : "—"}
                  </td>
                  <td className="px-4 py-2 text-xs" style={{ color: "var(--color-text-secondary)" }}>{task.exit_status || "—"}</td>
                </tr>
              ))}
              {tasks.length === 0 && (
                <tr>
                  <td colSpan={7} className="px-4 py-6 text-center text-sm" style={{ color: "var(--color-text-tertiary)" }}>
                    No tasks recorded yet
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      )}

      {/* Confirm modal */}
      {confirmAction && (
        <div className="fixed inset-0 z-50 flex items-center justify-center" style={{ backgroundColor: "rgba(0,0,0,0.5)" }}>
          <div ref={dialogRef} role="dialog" aria-modal="true" aria-label={`Confirm ${confirmAction.action}`} tabIndex={-1}
            className="p-5 rounded-lg w-96 outline-none" style={{ backgroundColor: "var(--color-surface)", border: "1px solid var(--color-border)" }}>
            <h3 className="text-sm font-semibold mb-3" style={{ color: "var(--color-text-primary)" }}>
              Confirm {confirmAction.action}
            </h3>
            <p className="text-sm mb-4" style={{ color: "var(--color-text-secondary)" }}>
              {confirmAction.action.charAt(0).toUpperCase() + confirmAction.action.slice(1)} {confirmAction.type.toUpperCase()} <strong>{confirmAction.name}</strong> (VMID {confirmAction.vmId}) on node <strong>{confirmAction.node}</strong>?
            </p>
            <div className="flex justify-end gap-2">
              <button onClick={() => setConfirmAction(null)} className="px-3 py-1.5 text-xs rounded" style={{ color: "var(--color-text-secondary)" }}>
                Cancel
              </button>
              <button
                onClick={doPowerAction}
                disabled={powerAction.isPending}
                className="px-3 py-1.5 text-xs rounded font-medium text-white"
                style={{ backgroundColor: confirmAction.action === "stop" || confirmAction.action === "shutdown" ? "var(--color-error)" : "var(--color-success)" }}
              >
                {powerAction.isPending ? "..." : confirmAction.action}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
