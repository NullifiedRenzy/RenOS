import type { Device } from "@/lib/types";
import { computeHealthScore } from "@/lib/utils";

interface StatCardsProps {
  devices: Device[];
}

export function StatCards({ devices }: StatCardsProps) {
  const onlineCount = devices.filter((d) => d.status === "online").length;
  const offlineCount = devices.filter((d) => d.status === "offline").length;
  const avgScore = devices.length > 0
    ? Math.round(devices.reduce((sum, d) => sum + computeHealthScore(d), 0) / devices.length)
    : 0;
  const alertCount = devices.filter((d) => computeHealthScore(d) < 70).length;

  const stats = [
    { label: "Health Score", value: avgScore > 0 ? `${avgScore}/100` : "--", trend: null, color: avgScore >= 90 ? "var(--color-success)" : avgScore >= 70 ? "var(--color-warning)" : "var(--color-error)" },
    { label: "Devices", value: `${onlineCount} online`, sub: `${offlineCount} offline`, color: "var(--color-accent)" },
    { label: "Alerts", value: `${alertCount}`, sub: alertCount === 1 ? "Needs attention" : alertCount > 1 ? "Need attention" : "All clear", color: alertCount > 0 ? "var(--color-warning)" : "var(--color-success)" },
  ];

  return (
    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
      {stats.map((stat) => (
        <div key={stat.label} className="card">
          <p className="text-caption text-[var(--color-text-tertiary)]">{stat.label}</p>
          <p className="text-headline-3 mt-1" style={{ color: stat.color }}>{stat.value}</p>
          {"sub" in stat && stat.sub && (
            <p className="text-body-small text-[var(--color-text-tertiary)] mt-1">{stat.sub}</p>
          )}
        </div>
      ))}
    </div>
  );
}
