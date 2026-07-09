interface MetricBarProps {
  label: string;
  percent: number;
  color?: string;
}

function MetricBar({ label, percent, color }: MetricBarProps) {
  return (
    <div>
      <div className="flex items-center justify-between mb-1">
        <span className="text-body-small text-[var(--color-text-secondary)]">
          {label}
        </span>
        <span className="text-body-small text-[var(--color-text-primary)] font-medium">
          {percent}%
        </span>
      </div>
      <div className="h-1.5 rounded-full bg-[var(--color-surface)] overflow-hidden">
        <div
          className="h-full rounded-full transition-all duration-500"
          style={{
            width: `${Math.min(percent, 100)}%`,
            backgroundColor: color || "var(--color-accent)",
          }}
        />
      </div>
    </div>
  );
}

interface DeviceMetricsProps {
  cpu: number;
  memory: number;
  disk: number;
}

export function DeviceMetrics({ cpu, memory, disk }: DeviceMetricsProps) {
  const diskColor = disk > 85 ? "var(--color-error)" : disk > 70 ? "var(--color-warning)" : "var(--color-accent)";

  return (
    <div className="space-y-3">
      <MetricBar label="CPU" percent={cpu} color="var(--color-accent)" />
      <MetricBar label="Memory" percent={memory} color="var(--color-info)" />
      <MetricBar label="Disk" percent={disk} color={diskColor} />
    </div>
  );
}
