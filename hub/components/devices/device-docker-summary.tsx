"use client";

interface DockerSummaryProps {
  containerCount: number;
  runningCount: number;
  imageCount: number;
  volumeCount: number;
  networkCount: number;
}

export function DockerSummaryCard({ containerCount, runningCount, imageCount, volumeCount, networkCount }: DockerSummaryProps) {
  const items = [
    { label: "Containers", value: containerCount, sub: `${runningCount} running` },
    { label: "Images", value: imageCount },
    { label: "Volumes", value: volumeCount },
    { label: "Networks", value: networkCount },
  ];

  return (
    <div className="card">
      <h2 className="text-headline-4 text-[var(--color-text-primary)] mb-3">Docker Summary</h2>
      <div className="grid grid-cols-2 gap-3">
        {items.map(item => (
          <div key={item.label} className="p-3 rounded-sm bg-[var(--color-surface)]">
            <p className="text-caption text-[var(--color-text-tertiary)]">{item.label}</p>
            <p className="text-headline-3 text-[var(--color-text-primary)]">{item.value}</p>
            {item.sub && (
              <p className="text-caption text-[var(--color-success)]">{item.sub}</p>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
