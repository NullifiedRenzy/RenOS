import Link from "next/link";
import { memo } from "react";
import type { Device } from "@/lib/types";
import { formatDate, getMetricPercent } from "@/lib/utils";

interface DeviceCardProps {
  device: Device;
}

export const DeviceCard = memo(function DeviceCard({ device }: DeviceCardProps) {
  const statusColor: Record<string, string> = {
    online: "var(--color-success)",
    offline: "var(--color-text-tertiary)",
    degraded: "var(--color-warning)",
    decommissioned: "var(--color-text-tertiary)",
  };

  const statusLabel = device.status.charAt(0).toUpperCase() + device.status.slice(1);

  const cpu = getMetricPercent(device, "cpu");
  const mem = getMetricPercent(device, "memory");

  return (
    <Link href={`/devices/${device.id}`} className="card-interactive block">
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-3">
          <span className="inline-block w-2 h-2 rounded-full" aria-hidden="true" style={{ backgroundColor: statusColor[device.status] ?? "var(--color-text-tertiary)" }} />
          <div>
            <h3 className="text-body font-medium text-[var(--color-text-primary)]">{device.hostname}</h3>
            <p className="text-caption text-[var(--color-text-tertiary)]">{device.os} &middot; {device.arch}</p>
          </div>
        </div>
        <span className="badge-info">{statusLabel}</span>
      </div>
      {(cpu > 0 || mem > 0) && (
        <div className="flex gap-4 mt-3">
          <div className="flex-1">
            <div className="flex items-center justify-between mb-1">
              <span className="text-caption text-[var(--color-text-tertiary)]">CPU</span>
              <span className="text-caption text-[var(--color-text-primary)]">{cpu}%</span>
            </div>
            <div className="h-1 rounded-full bg-[var(--color-surface)] overflow-hidden">
              <div className="h-full rounded-full bg-[var(--color-accent)]" style={{ width: `${Math.min(cpu, 100)}%` }} />
            </div>
          </div>
          <div className="flex-1">
            <div className="flex items-center justify-between mb-1">
              <span className="text-caption text-[var(--color-text-tertiary)]">RAM</span>
              <span className="text-caption text-[var(--color-text-primary)]">{mem}%</span>
            </div>
            <div className="h-1 rounded-full bg-[var(--color-surface)] overflow-hidden">
              <div className="h-full rounded-full bg-[var(--color-info)]" style={{ width: `${Math.min(mem, 100)}%` }} />
            </div>
          </div>
        </div>
      )}
      <p className="text-caption text-[var(--color-text-tertiary)] mt-3">Last seen: {formatDate(device.last_seen)}</p>
    </Link>
  );
});
