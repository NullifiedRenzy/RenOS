"use client";

import { useMemo } from "react";
import { useDevices } from "@/hooks/use-devices";
import { DeviceCard } from "@/components/devices/device-card";
import { EmptyState } from "@/components/ui/empty-state";
import { ErrorState } from "@/components/ui/error-state";

export default function DevicesPage() {
  const { data, isLoading, error, refetch } = useDevices();
  const devices = useMemo(() => data?.devices ?? [], [data]);

  const { onlineCount, offlineCount } = useMemo(() => {
    let online = 0;
    let offline = 0;
    for (const d of devices) {
      if (d.status === "online") online += 1;
      else if (d.status === "offline") offline += 1;
    }
    return { onlineCount: online, offlineCount: offline };
  }, [devices]);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-headline-1 text-[var(--color-text-primary)]">Devices</h1>
          <p aria-live="polite" className="text-body text-[var(--color-text-secondary)] mt-1">
            {isLoading ? "Loading..." : `${devices.length} device${devices.length !== 1 ? "s" : ""} · ${onlineCount} online · ${offlineCount} offline`}
          </p>
        </div>
      </div>

      {isLoading && (
        <div aria-hidden="true" className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="skeleton h-28" />
          ))}
        </div>
      )}

      {error && !isLoading && (
        <ErrorState error={error} onRetry={() => refetch()} title="Failed to load devices" />
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
