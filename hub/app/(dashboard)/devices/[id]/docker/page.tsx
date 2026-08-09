"use client";

import { useParams } from "next/navigation";
import Link from "next/link";
import { useDevice } from "@/hooks/use-devices";
import { useContainers } from "@/hooks/use-containers";
import { useDockerImages } from "@/hooks/use-docker-images";
import { useDockerVolumes } from "@/hooks/use-docker-volumes";
import { useDockerNetworks } from "@/hooks/use-docker-networks";
import { useComposeStacks } from "@/hooks/use-compose-stacks";
import { useContainerMetrics } from "@/hooks/use-container-metrics";
import { useContainerAction } from "@/hooks/use-container-actions";
import { formatBytes } from "@/lib/format";
import { formatDate } from "@/lib/utils";
import { ActionOutputModal } from "@/components/ui/action-output-modal";
import { StateDot } from "@/components/ui/state-dot";
import type { Container, DockerImage, ComposeStack, ContainerMetric } from "@/lib/types";
import { useState } from "react";

function MetricBar({ label, value, max, color }: { label: string; value: number; max: number; color: string }) {
  const pct = max > 0 ? Math.min((value / max) * 100, 100) : 0;
  return (
    <div className="flex items-center gap-2">
      <span className="text-caption text-[var(--color-text-tertiary)] w-16">{label}</span>
      <div className="flex-1 h-2 bg-[var(--color-border)] rounded-full overflow-hidden">
        <div className="h-full rounded-full transition-all" style={{ width: `${pct}%`, backgroundColor: color }} />
      </div>
      <span className="text-caption text-[var(--color-text-secondary)] w-20 text-right">{pct.toFixed(1)}%</span>
    </div>
  );
}

export default function DockerPage() {
  const { id } = useParams<{ id: string }>();
  const { data: deviceData } = useDevice(id);
  const device = deviceData?.device;
  const { data: containersData, refetch: refetchContainers } = useContainers(id);
  const { data: imagesData } = useDockerImages(id);
  const { data: volumesData } = useDockerVolumes(id);
  const { data: networksData } = useDockerNetworks(id);
  const { data: stacksData } = useComposeStacks(id);
  const { data: metricsData } = useContainerMetrics(id);
  const action = useContainerAction(id);

  const [tab, setTab] = useState<"containers" | "images" | "volumes" | "networks" | "compose" | "metrics">("containers");
  const [actionResult, setActionResult] = useState<{ title: string; output: string; error: string; exitCode: number | null } | null>(null);

  const containers = containersData?.containers ?? [];
  const images = imagesData?.images ?? [];
  const volumes = volumesData?.volumes ?? [];
  const networks = networksData?.networks ?? [];
  const stacks = stacksData?.compose_stacks ?? [];
  const metrics = metricsData?.metrics ?? [];

  const runningCount = containers.filter(c => c.state === "running").length;
  const stoppedCount = containers.filter(c => c.state === "exited").length;

  const handleAction = (containerId: string, actionType: string, containerName: string) => {
    action.mutate(
      { containerId, action: actionType },
      {
        onSuccess: (result) => {
          if (actionType === "logs" || actionType === "stats" || actionType === "inspect") {
            const act = result.container_action;
            setActionResult({
              title: `${actionType === "logs" ? "Logs" : actionType === "stats" ? "Stats" : "Inspect"} — ${containerName}`,
              output: act.output ?? "",
              error: act.error ?? "",
              exitCode: act.exit_code,
            });
          }
          if (actionType === "stop" || actionType === "start" || actionType === "restart" || actionType === "kill") {
            setTimeout(() => refetchContainers(), 2000);
          }
        },
      },
    );
  };

  if (!device) {
    return (
      <div className="space-y-6">
        <Link href={`/devices/${id}`} className="btn-ghost text-body-small">&larr; Back to Device</Link>
        <div className="card text-center py-8">
          <p className="text-body text-[var(--color-text-tertiary)]">Loading device...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6 max-w-6xl">
      <div>
        <Link href={`/devices/${id}`} className="btn-ghost text-body-small">&larr; {device.hostname}</Link>
      </div>

      <div className="flex items-center justify-between">
        <h1 className="text-headline-1 text-[var(--color-text-primary)]">Docker Management</h1>
        <div className="flex gap-4 text-body-small text-[var(--color-text-secondary)]">
          <span>{runningCount} running</span>
          <span>{stoppedCount} stopped</span>
          <span>{images.length} images</span>
          <span>{volumes.length} volumes</span>
          <span>{stacks.length} compose stacks</span>
        </div>
      </div>

      {/* Tab navigation */}
      <div className="flex gap-1 border-b border-[var(--color-border)]">
        {(["containers", "images", "volumes", "networks", "compose", "metrics"] as const).map((t) => (
          <button
            key={t}
            className={`px-4 py-2 text-body-small font-medium transition-colors ${
              tab === t
                ? "text-[var(--color-accent)] border-b-2 border-[var(--color-accent)]"
                : "text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)]"
            }`}
            onClick={() => setTab(t)}
          >
            {t.charAt(0).toUpperCase() + t.slice(1)}
          </button>
        ))}
      </div>

      {/* Containers Tab */}
      {tab === "containers" && (
        <div className="card">
          <div className="overflow-x-auto">
            <table className="w-full text-left">
              <thead>
                <tr className="text-caption text-[var(--color-text-tertiary)] border-b border-[var(--color-border)]">
                  <th scope="col" className="pb-2 pr-3 font-medium">Name</th>
                  <th scope="col" className="pb-2 pr-3 font-medium">Image</th>
                  <th scope="col" className="pb-2 pr-3 font-medium">Status</th>
                  <th scope="col" className="pb-2 pr-3 font-medium">Ports</th>
                  <th scope="col" className="pb-2 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody>
                {containers.map((c: Container) => {
                  const m = metrics.find((met) => met.container_id === c.container_id);
                  return (
                    <tr key={c.id} className="border-b border-[var(--color-border)] last:border-b-0">
                      <td className="py-2 pr-3 text-body-small font-mono text-[var(--color-text-primary)]">{c.name}</td>
                      <td className="py-2 pr-3 text-body-small text-[var(--color-text-secondary)]">{c.image}</td>
                      <td className="py-2 pr-3 text-body-small">
                        <span className="inline-flex items-center gap-1.5">
                          <StateDot state={c.state} />
                          {c.status}
                        </span>
                        {m && (
                          <div className="mt-1 flex gap-3 text-caption text-[var(--color-text-tertiary)]">
                            <span>CPU {m.cpu_percent.toFixed(1)}%</span>
                            <span>MEM {m.memory_percent.toFixed(1)}%</span>
                          </div>
                        )}
                      </td>
                      <td className="py-2 pr-3 text-body-small text-[var(--color-text-secondary)]">
                        {c.ports || "—"}
                      </td>
                      <td className="py-2 text-body-small">
                        <span className="inline-flex items-center gap-1 flex-wrap">
                          {["running"].includes(c.state) && (
                            <>
                              <button className="btn-ghost text-caption px-2 py-0.5" disabled={action.isPending} onClick={() => handleAction(c.container_id, "stop", c.name)}>Stop</button>
                              <button className="btn-ghost text-caption px-2 py-0.5" disabled={action.isPending} onClick={() => handleAction(c.container_id, "restart", c.name)}>Restart</button>
                              <button className="btn-ghost text-caption px-2 py-0.5 text-[var(--color-error)]" disabled={action.isPending} onClick={() => handleAction(c.container_id, "kill", c.name)}>Kill</button>
                              <button className="btn-ghost text-caption px-2 py-0.5" disabled={action.isPending} onClick={() => handleAction(c.container_id, "pause", c.name)}>Pause</button>
                            </>
                          )}
                          {["paused"].includes(c.state) && (
                            <>
                              <button className="btn-ghost text-caption px-2 py-0.5" disabled={action.isPending} onClick={() => handleAction(c.container_id, "start", c.name)}>Start</button>
                              <button className="btn-ghost text-caption px-2 py-0.5" disabled={action.isPending} onClick={() => handleAction(c.container_id, "unpause", c.name)}>Unpause</button>
                            </>
                          )}
                          {["exited", "created"].includes(c.state) && (
                            <>
                              <button className="btn-ghost text-caption px-2 py-0.5" disabled={action.isPending} onClick={() => handleAction(c.container_id, "start", c.name)}>Start</button>
                              <button className="btn-ghost text-caption px-2 py-0.5 text-[var(--color-error)]" disabled={action.isPending} onClick={() => { if (window.confirm(`Remove "${c.name}"?`)) handleAction(c.container_id, "remove", c.name); }}>Remove</button>
                            </>
                          )}
                          <button className="btn-ghost text-caption px-2 py-0.5" disabled={action.isPending} onClick={() => handleAction(c.container_id, "logs", c.name)}>Logs</button>
                          <button className="btn-ghost text-caption px-2 py-0.5" disabled={action.isPending} onClick={() => handleAction(c.container_id, "inspect", c.name)}>Inspect</button>
                        </span>
                      </td>
                    </tr>
                  );
                })}
                {containers.length === 0 && (
                  <tr><td colSpan={5} className="py-8 text-center text-body text-[var(--color-text-tertiary)]">No containers found</td></tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Images Tab */}
      {tab === "images" && (
        <div className="card">
          <div className="overflow-x-auto">
            <table className="w-full text-left">
              <thead>
                <tr className="text-caption text-[var(--color-text-tertiary)] border-b border-[var(--color-border)]">
                  <th scope="col" className="pb-2 pr-3 font-medium">Repository</th>
                  <th scope="col" className="pb-2 pr-3 font-medium">Tag</th>
                  <th scope="col" className="pb-2 pr-3 font-medium">Image ID</th>
                  <th scope="col" className="pb-2 pr-3 font-medium">Size</th>
                  <th scope="col" className="pb-2 font-medium">Created</th>
                </tr>
              </thead>
              <tbody>
                {images.map((img: DockerImage) => (
                  <tr key={img.id} className="border-b border-[var(--color-border)] last:border-b-0">
                    <td className="py-2 pr-3 text-body-small font-mono text-[var(--color-text-primary)]">{img.repository}</td>
                    <td className="py-2 pr-3 text-body-small text-[var(--color-text-secondary)]">{img.tag}</td>
                    <td className="py-2 pr-3 text-body-small font-mono text-[var(--color-text-secondary)] text-xs">{img.image_id.slice(0, 19)}</td>
                    <td className="py-2 pr-3 text-body-small text-[var(--color-text-secondary)]">{img.size}</td>
                    <td className="py-2 text-body-small text-[var(--color-text-secondary)]">{img.created}</td>
                  </tr>
                ))}
                {images.length === 0 && (
                  <tr><td colSpan={5} className="py-8 text-center text-body text-[var(--color-text-tertiary)]">No images found</td></tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Volumes Tab */}
      {tab === "volumes" && (
        <div className="card">
          <div className="overflow-x-auto">
            <table className="w-full text-left">
              <thead>
                <tr className="text-caption text-[var(--color-text-tertiary)] border-b border-[var(--color-border)]">
                  <th scope="col" className="pb-2 pr-3 font-medium">Name</th>
                  <th scope="col" className="pb-2 pr-3 font-medium">Driver</th>
                  <th scope="col" className="pb-2 font-medium">Mountpoint</th>
                </tr>
              </thead>
              <tbody>
                {volumes.map((vol: any) => (
                  <tr key={vol.id} className="border-b border-[var(--color-border)] last:border-b-0">
                    <td className="py-2 pr-3 text-body-small font-mono text-[var(--color-text-primary)]">{vol.name}</td>
                    <td className="py-2 pr-3 text-body-small text-[var(--color-text-secondary)]">{vol.driver}</td>
                    <td className="py-2 text-body-small text-[var(--color-text-secondary)] font-mono text-xs">{vol.mountpoint}</td>
                  </tr>
                ))}
                {volumes.length === 0 && (
                  <tr><td colSpan={3} className="py-8 text-center text-body text-[var(--color-text-tertiary)]">No volumes found</td></tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Networks Tab */}
      {tab === "networks" && (
        <div className="card">
          <div className="overflow-x-auto">
            <table className="w-full text-left">
              <thead>
                <tr className="text-caption text-[var(--color-text-tertiary)] border-b border-[var(--color-border)]">
                  <th scope="col" className="pb-2 pr-3 font-medium">Name</th>
                  <th scope="col" className="pb-2 pr-3 font-medium">Driver</th>
                  <th scope="col" className="pb-2 pr-3 font-medium">Scope</th>
                  <th scope="col" className="pb-2 font-medium">Network ID</th>
                </tr>
              </thead>
              <tbody>
                {networks.map((net: any) => (
                  <tr key={net.id} className="border-b border-[var(--color-border)] last:border-b-0">
                    <td className="py-2 pr-3 text-body-small font-mono text-[var(--color-text-primary)]">{net.name}</td>
                    <td className="py-2 pr-3 text-body-small text-[var(--color-text-secondary)]">{net.driver}</td>
                    <td className="py-2 pr-3 text-body-small text-[var(--color-text-secondary)]">{net.scope}</td>
                    <td className="py-2 text-body-small font-mono text-[var(--color-text-secondary)] text-xs">{net.network_id.slice(0, 19)}</td>
                  </tr>
                ))}
                {networks.length === 0 && (
                  <tr><td colSpan={4} className="py-8 text-center text-body text-[var(--color-text-tertiary)]">No networks found</td></tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Compose Tab */}
      {tab === "compose" && (
        <div className="card">
          {stacks.length === 0 ? (
            <div className="py-8 text-center text-body text-[var(--color-text-tertiary)]">
              No compose stacks found on this device
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left">
                <thead>
                  <tr className="text-caption text-[var(--color-text-tertiary)] border-b border-[var(--color-border)]">
                    <th scope="col" className="pb-2 pr-3 font-medium">Name</th>
                    <th scope="col" className="pb-2 pr-3 font-medium">Status</th>
                    <th scope="col" className="pb-2 pr-3 font-medium">Services</th>
                    <th scope="col" className="pb-2 font-medium">Config Files</th>
                  </tr>
                </thead>
                <tbody>
                  {stacks.map((s: ComposeStack) => {
                    let svcList: Array<{ name: string; state: string }> = [];
                    try { svcList = JSON.parse(s.services); } catch {}
                    return (
                      <tr key={s.id} className="border-b border-[var(--color-border)] last:border-b-0">
                        <td className="py-2 pr-3 text-body-small font-mono text-[var(--color-text-primary)]">{s.name}</td>
                        <td className="py-2 pr-3 text-body-small text-[var(--color-text-secondary)]">{s.status}</td>
                        <td className="py-2 pr-3 text-body-small text-[var(--color-text-secondary)]">
                          {svcList.length > 0 ? svcList.map((svc) => (
                            <span key={svc.name} className="inline-flex items-center gap-1 mr-2">
                              <StateDot state={svc.state} />
                              {svc.name}
                            </span>
                          )) : "—"}
                        </td>
                        <td className="py-2 text-body-small text-[var(--color-text-secondary)] font-mono text-xs">{s.config_files}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {/* Metrics Tab */}
      {tab === "metrics" && (
        <div className="space-y-4">
          {metrics.length === 0 ? (
            <div className="card py-8 text-center text-body text-[var(--color-text-tertiary)]">
              No container metrics available. Metrics are collected every 15 seconds for running containers.
            </div>
          ) : (
            metrics.map((m: ContainerMetric) => (
              <div key={m.container_id} className="card">
                <div className="flex items-center justify-between mb-3">
                  <h3 className="text-body font-medium text-[var(--color-text-primary)]">{m.container_name}</h3>
                  <span className="text-caption text-[var(--color-text-tertiary)]">
                    {formatDate(m.recorded_at)}
                  </span>
                </div>
                <div className="space-y-2">
                  <MetricBar label="CPU" value={m.cpu_percent} max={100} color="var(--color-accent)" />
                  <MetricBar label="Memory" value={m.memory_percent} max={100} color="var(--color-success)" />
                  <div className="flex gap-6 text-caption text-[var(--color-text-secondary)] mt-2">
                    <span>MEM: {formatBytes(m.memory_usage_bytes)} / {formatBytes(m.memory_limit_bytes)}</span>
                    <span>NET RX: {formatBytes(m.network_rx_bytes)}</span>
                    <span>NET TX: {formatBytes(m.network_tx_bytes)}</span>
                    <span>DISK R: {formatBytes(m.block_read_bytes)}</span>
                    <span>DISK W: {formatBytes(m.block_write_bytes)}</span>
                    <span>PIDs: {m.pids}</span>
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      )}

      {actionResult && (
        <ActionOutputModal
          title={actionResult.title}
          output={actionResult.output}
          error={actionResult.error}
          exitCode={actionResult.exitCode}
          onClose={() => setActionResult(null)}
          size="lg"
        />
      )}
    </div>
  );
}
