"use client";

import { useDockerImages } from "@/hooks/use-docker-images";

interface DockerImagesProps {
  deviceId: string;
}

export function DockerImages({ deviceId }: DockerImagesProps) {
  const { data, isLoading, error } = useDockerImages(deviceId);
  const images = data?.images ?? [];

  return (
    <div className="card">
      <h2 className="text-headline-4 text-[var(--color-text-primary)] mb-3">Docker Images</h2>

      {isLoading && (
        <div className="space-y-2">
          <div className="skeleton h-8 w-full" />
          <div className="skeleton h-8 w-full" />
        </div>
      )}

      {error && (
        <p className="text-body text-[var(--color-text-tertiary)] py-3 text-center">
          Unable to load Docker images.
        </p>
      )}

      {!isLoading && !error && images.length === 0 && (
        <p className="text-body text-[var(--color-text-tertiary)] py-3 text-center">
          No Docker images found.
        </p>
      )}

      {!isLoading && !error && images.length > 0 && (
        <div className="overflow-x-auto">
          <table className="w-full text-left">
            <thead>
              <tr className="text-caption text-[var(--color-text-tertiary)] border-b border-[var(--color-border)]">
                <th className="pb-2 pr-3 font-medium">Repository</th>
                <th className="pb-2 pr-3 font-medium">Tag</th>
                <th className="pb-2 pr-3 font-medium">Size</th>
                <th className="pb-2 font-medium hidden md:table-cell">Created</th>
              </tr>
            </thead>
            <tbody>
              {images.map(img => (
                <tr key={img.id} className="border-b border-[var(--color-border)] last:border-b-0">
                  <td className="py-2 pr-3 text-body-small font-mono text-[var(--color-text-primary)]">
                    {img.repository || "(unnamed)"}
                  </td>
                  <td className="py-2 pr-3 text-body-small text-[var(--color-text-secondary)]">
                    {img.tag}
                  </td>
                  <td className="py-2 pr-3 text-body-small text-[var(--color-text-secondary)]">
                    {img.size}
                  </td>
                  <td className="py-2 text-body-small text-[var(--color-text-secondary)] hidden md:table-cell">
                    {img.created}
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
