"use client";

import { useStorageDevices, getDeviceTypeIcon } from "@/hooks/use-storage";
import { formatFileSize, usagePercent } from "@/lib/format";

interface StorageExplorerProps {
  deviceId: string;
}

export function StorageExplorer({ deviceId }: StorageExplorerProps) {
  const { data, isLoading, error } = useStorageDevices(deviceId);
  const devices = data?.devices ?? [];

  const disks = devices.filter(d => d.device_type === "disk");
  const partitions = devices.filter(d => d.device_type !== "disk");

  return (
    <div className="card">
      <h2 className="text-headline-4 text-[var(--color-text-primary)] mb-3">Storage</h2>

      {isLoading && (
        <div className="space-y-2">
          <div className="skeleton h-8 w-full" />
          <div className="skeleton h-8 w-full" />
        </div>
      )}

      {error && (
        <p className="text-body text-[var(--color-text-tertiary)] py-3 text-center">
          Unable to load storage devices.
        </p>
      )}

      {!isLoading && !error && devices.length === 0 && (
        <p className="text-body text-[var(--color-text-tertiary)] py-3 text-center">
          No storage devices found.
        </p>
      )}

      {!isLoading && !error && devices.length > 0 && (
        <div className="space-y-3">
          {disks.length > 0 && (
            <div>
              <h3 className="text-body-small text-[var(--color-text-secondary)] mb-2 font-medium">
                Physical Disks ({disks.length})
              </h3>
              {disks.map(d => (
                <div
                  key={d.id}
                  className="border border-[var(--color-border)] rounded-sm p-3 mb-2"
                >
                  <div className="flex items-center justify-between mb-1">
                    <div className="flex items-center gap-2">
                      <span className="text-body">{getDeviceTypeIcon(d.device_type)}</span>
                      <span className="text-body-small font-mono font-medium text-[var(--color-text-primary)]">
                        {d.name}
                      </span>
                      {d.model && (
                        <span className="text-caption text-[var(--color-text-tertiary)]">
                          {d.model}
                        </span>
                      )}
                    </div>
                    <span className="text-body-small text-[var(--color-text-secondary)]">
                      {formatFileSize(d.size)}
                    </span>
                  </div>
                  {d.is_removable && (
                    <span className="text-caption text-[var(--color-warning)]">Removable</span>
                  )}
                  {d.is_readonly && (
                    <span className="text-caption text-[var(--color-warning)] ml-2">Read-only</span>
                  )}
                </div>
              ))}
            </div>
          )}

          {partitions.filter(p => p.mount_point).length > 0 && (
            <div>
              <h3 className="text-body-small text-[var(--color-text-secondary)] mb-2 font-medium">
                Mounted Filesystems
              </h3>
              {partitions.filter(p => p.mount_point).map(p => {
                const pct = usagePercent(p.used_bytes, p.used_bytes + p.available_bytes);
                const barColor = pct >= 90
                  ? "var(--color-error)"
                  : pct >= 75
                    ? "var(--color-warning)"
                    : "var(--color-success)";
                return (
                  <div
                    key={p.id}
                    className="border border-[var(--color-border)] rounded-sm p-3 mb-2"
                  >
                    <div className="flex items-center justify-between mb-1">
                      <div className="flex items-center gap-2">
                        <span className="text-body">{getDeviceTypeIcon(p.device_type)}</span>
                        <span className="text-body-small font-mono font-medium text-[var(--color-text-primary)]">
                          {p.mount_point}
                        </span>
                        <span className="text-caption text-[var(--color-text-tertiary)]">
                          {p.filesystem}
                        </span>
                      </div>
                      <span className="text-body-small text-[var(--color-text-secondary)]">
                        {formatFileSize(p.used_bytes)} / {formatFileSize(p.used_bytes + p.available_bytes)}
                      </span>
                    </div>
                    <div className="mt-2">
                      <div className="w-full h-2 bg-[var(--color-bg-secondary)] rounded-full overflow-hidden">
                        <div
                          className="h-full rounded-full transition-all"
                          style={{ width: `${Math.min(pct, 100)}%`, backgroundColor: barColor }}
                        />
                      </div>
                      <div className="flex justify-between mt-1">
                        <span className="text-caption text-[var(--color-text-tertiary)]">
                          {p.name}
                        </span>
                        <span className="text-caption text-[var(--color-text-tertiary)]">
                          {pct}% used
                        </span>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          )}

          {partitions.filter(p => !p.mount_point).length > 0 && (
            <div>
              <h3 className="text-body-small text-[var(--color-text-secondary)] mb-2 font-medium">
                Unmounted Partitions
              </h3>
              {partitions.filter(p => !p.mount_point).map(p => (
                <div
                  key={p.id}
                  className="border border-[var(--color-border)] rounded-sm p-3 mb-2"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <span className="text-body">{getDeviceTypeIcon(p.device_type)}</span>
                      <span className="text-body-small font-mono text-[var(--color-text-primary)]">
                        {p.name}
                      </span>
                      {p.filesystem && (
                        <span className="text-caption text-[var(--color-text-tertiary)]">
                          {p.filesystem}
                        </span>
                      )}
                    </div>
                    <span className="text-body-small text-[var(--color-text-secondary)]">
                      {formatFileSize(p.size)}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
