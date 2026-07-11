"use client";

import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { useDevice } from "@/hooks/use-devices";
import { useContainers } from "@/hooks/use-containers";
import { useSystemdServices } from "@/hooks/use-systemd-services";
import { HealthScore } from "@/components/devices/health-score";
import { DeviceMetrics } from "@/components/devices/device-metrics";
import { SafeCommands } from "@/components/devices/safe-commands";
import { CommandHistory } from "@/components/devices/command-history";
import { DeviceContainers } from "@/components/devices/device-containers";
import { DockerSummaryCard } from "@/components/devices/device-docker-summary";
import { DockerImages } from "@/components/devices/device-docker-images";
import { DockerVolumes } from "@/components/devices/device-docker-volumes";
import { DockerNetworks } from "@/components/devices/device-docker-networks";
import { SystemdServices } from "@/components/devices/device-systemd-services";
import { FileBrowser } from "@/components/devices/device-file-browser";
import { NetworkInfo } from "@/components/devices/device-network";
import { StorageExplorer } from "@/components/devices/device-storage";
import { EmptyState } from "@/components/devices/empty-state";
import { computeHealthScore, getMetricPercent, formatDate, formatDuration, getLastHealth } from "@/lib/utils";

export default function DeviceDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const { data, isLoading, error } = useDevice(id);
  const device = data?.device ?? null;
  const { data: containersData } = useContainers(id);
  const containers = containersData?.containers ?? [];
  const containerCount = containers.length;
  const runningCount = containers.filter(c => c.state === "running").length;
  const { data: servicesData } = useSystemdServices(id);
  const services = servicesData?.services ?? [];

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="skeleton h-8 w-48" />
        <div className="grid grid-cols-3 gap-4">
          <div className="skeleton h-32 col-span-2" />
          <div className="skeleton h-32" />
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-6">
        <button onClick={() => router.back()} className="btn-ghost text-body-small">&larr; Back</button>
        <div className="card text-center py-8">
          <p className="text-body text-[var(--color-error)]">Failed to load device</p>
          <p className="text-body-small text-[var(--color-text-tertiary)] mt-1">{(error as Error).message}</p>
        </div>
      </div>
    );
  }

  if (!device) {
    return (
      <EmptyState
        title="Device not found"
        description="The device you are looking for does not exist or has been removed."
        action={{ label: "Back to Devices", onClick: () => router.push("/devices") }}
      />
    );
  }

  const score = computeHealthScore(device);
  const health = getLastHealth(device);
  const statusColor: Record<string, string> = {
    online: "var(--color-success)",
    offline: "var(--color-text-tertiary)",
    degraded: "var(--color-warning)",
    decommissioned: "var(--color-text-tertiary)",
  };

  return (
    <div className="space-y-6 max-w-4xl">
      <div>
        <Link href="/devices" className="btn-ghost text-body-small">&larr; All Devices</Link>
      </div>

      <div className="flex items-start justify-between">
        <div className="flex items-center gap-4">
          <span
            className="inline-block w-3 h-3 rounded-full flex-shrink-0"
            style={{ backgroundColor: statusColor[device.status] ?? "var(--color-text-tertiary)" }}
          />
          <div>
            <h1 className="text-headline-1 text-[var(--color-text-primary)]">{device.hostname}</h1>
            <p className="text-body text-[var(--color-text-secondary)] mt-1">
              {device.os} &middot; {device.arch} &middot; v{device.agent_version}
            </p>
          </div>
        </div>
        <HealthScore score={score} size="lg" />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <div className="lg:col-span-2 space-y-4">
          <div className="card">
            <h2 className="text-headline-4 text-[var(--color-text-primary)] mb-4">Resource Usage</h2>
            {health ? (
              <DeviceMetrics
                cpu={getMetricPercent(device, "cpu")}
                memory={getMetricPercent(device, "memory")}
                disk={getMetricPercent(device, "disk")}
              />
            ) : (
              <p className="text-body text-[var(--color-text-tertiary)] py-4 text-center">
                No health data reported yet
              </p>
            )}
          </div>

          <DeviceContainers deviceId={device.id} />

          <DockerImages deviceId={device.id} />
          <DockerVolumes deviceId={device.id} />
          <DockerNetworks deviceId={device.id} />

          <SystemdServices deviceId={device.id} />

          <FileBrowser deviceId={device.id} />

          <NetworkInfo deviceId={device.id} />

          <StorageExplorer deviceId={device.id} />

          {device.status === "online" && (
            <SafeCommands deviceId={device.id} />
          )}

          <div className="card">
            <h2 className="text-headline-4 text-[var(--color-text-primary)] mb-4">Details</h2>
            <dl className="grid grid-cols-2 gap-y-3 gap-x-4">
              {[
                ["Device ID", device.id],
                ["Status", device.status.charAt(0).toUpperCase() + device.status.slice(1)],
                ["Hostname", device.hostname],
                ["OS", device.os],
                ["Architecture", device.arch],
                ["Agent Version", device.agent_version],
                ["Enrolled", formatDate(device.enrolled_at)],
                ["Last Seen", formatDate(device.last_seen)],
              ].map(([label, value]) => (
                <div key={label}>
                  <dt className="text-caption text-[var(--color-text-tertiary)]">{label}</dt>
                  <dd className="text-body-small text-[var(--color-text-primary)] font-medium">{value}</dd>
                </div>
              ))}
            </dl>
          </div>

          <CommandHistory deviceId={device.id} />
        </div>

        <div className="space-y-4">
          <DockerSummaryCard
            containerCount={containerCount}
            runningCount={runningCount}
            imageCount={0}
            volumeCount={0}
            networkCount={0}
            serviceCount={services.length}
          />

          <div className="card">
            <h2 className="text-headline-4 text-[var(--color-text-primary)] mb-3">Health Score</h2>
            <div className="flex items-center justify-center py-4">
              <HealthScore score={score} size="lg" />
            </div>
            <p className="text-center text-body-small text-[var(--color-text-secondary)]">
              {score >= 90 ? "Excellent" : score >= 70 ? "Good" : score >= 50 ? "Fair" : "Poor"}
            </p>
          </div>

          {health && (
            <div className="card">
              <h2 className="text-headline-4 text-[var(--color-text-primary)] mb-3">Uptime</h2>
              <p className="text-headline-2 text-[var(--color-accent)]">{formatDuration(health.uptime_seconds)}</p>
              <p className="text-caption text-[var(--color-text-tertiary)] mt-1">
                Last reported: {formatDate(health.reported_at)}
              </p>
            </div>
          )}

          {health?.local_ip && (
            <div className="card">
              <h2 className="text-headline-4 text-[var(--color-text-primary)] mb-3">Network</h2>
              <p className="text-body font-mono text-[var(--color-text-primary)]">{health.local_ip}</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
