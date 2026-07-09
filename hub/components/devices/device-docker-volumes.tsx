"use client";

import { useDockerVolumes } from "@/hooks/use-docker-volumes";

interface DockerVolumesProps {
  deviceId: string;
}

export function DockerVolumes({ deviceId }: DockerVolumesProps) {
  const { data, isLoading, error } = useDockerVolumes(deviceId);
  const volumes = data?.volumes ?? [];

  return (
    <div className="card">
      <h2 className="text-headline-4 text-[var(--color-text-primary)] mb-3">Docker Volumes</h2>

      {isLoading && (
        <div className="space-y-2">
          <div className="skeleton h-8 w-full" />
          <div className="skeleton h-8 w-full" />
        </div>
      )}

      {error && (
        <p className="text-body text-[var(--color-text-tertiary)] py-3 text-center">
          Unable to load Docker volumes.
        </p>
      )}

      {!isLoading && !error && volumes.length === 0 && (
        <p className="text-body text-[var(--color-text-tertiary)] py-3 text-center">
          No Docker volumes found.
        </p>
      )}

      {!isLoading && !error && volumes.length > 0 && (
        <div className="overflow-x-auto">
          <table className="w-full text-left">
            <thead>
              <tr className="text-caption text-[var(--color-text-tertiary)] border-b border-[var(--color-border)]">
                <th className="pb-2 pr-3 font-medium">Name</th>
                <th className="pb-2 pr-3 font-medium">Driver</th>
                <th className="pb-2 font-medium hidden md:table-cell">Mountpoint</th>
              </tr>
            </thead>
            <tbody>
              {volumes.map(v => (
                <tr key={v.id} className="border-b border-[var(--color-border)] last:border-b-0">
                  <td className="py-2 pr-3 text-body-small font-mono text-[var(--color-text-primary)]">
                    {v.name}
                  </td>
                  <td className="py-2 pr-3 text-body-small text-[var(--color-text-secondary)]">
                    {v.driver}
                  </td>
                  <td className="py-2 text-body-small font-mono text-[var(--color-text-secondary)] hidden md:table-cell">
                    {v.mountpoint}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
