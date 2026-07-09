"use client";

import { useDevices } from "@/hooks/use-devices";
import { DeviceCard } from "@/components/devices/device-card";
import { EmptyState } from "@/components/devices/empty-state";

export default function DevicesPage() {
  const { data, isLoading, error } = useDevices();
  const devices = data?.devices ?? [];

  const onlineCount = devices.filter((d) => d.status === "online").length;
  const offlineCount = devices.filter((d) => d.status === "offline").length;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-headline-1 text-[var(--color-text-primary)]">Devices</h1>
          <p className="text-body text-[var(--color-text-secondary)] mt-1">
            {isLoading ? "Loading..." : `${devices.length} device${devices.length !== 1 ? "s" : ""} · ${onlineCount} online · ${offlineCount} offline`}
          </p>
        </div>
      </div>

      {isLoading && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="skeleton h-28" />
          ))}
        </div>
      )}

      {error && !isLoading && (
        <div className="card text-center py-8">
          <p className="text-body text-[var(--color-error)]">Failed to load devices</p>
          <p className="text-body-small text-[var(--color-text-tertiary)] mt-1">
            {(error as Error).message}
          </p>
        </div>
      )}

      {!isLoading && !error && devices.length === 0 && (
        <EmptyState
          title="No devices enrolled"
          description="Enroll your first device using the RenOS Agent CLI to see it here."
        />
      )}

      {!isLoading && !error && devices.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {devices.map((device) => (
            <DeviceCard key={device.id} device={device} />
          ))}
        </div>
      )}
    </div>
  );
}
