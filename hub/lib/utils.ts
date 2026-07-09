import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import type { Device, LastHealth } from "@/lib/types";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  return `${hours}h ${minutes}m`;
}

export function formatDate(iso: string | null): string {
  if (!iso) return "Never";
  const date = new Date(iso);
  const now = new Date();
  const diff = now.getTime() - date.getTime();

  if (diff < 60_000) return "Just now";
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`;
  return date.toLocaleDateString();
}

export function healthScoreColor(score: number): string {
  if (score >= 90) return "var(--color-success)";
  if (score >= 70) return "var(--color-warning)";
  return "var(--color-error)";
}

export function computeHealthScore(device: Device): number {
  let health: LastHealth | null = null;
  try {
    const meta = JSON.parse(device.metadata);
    health = meta.last_health ?? null;
  } catch {
    return 0;
  }
  if (!health) return 0;

  const cpu = 100 - health.cpu_percent;
  const mem = 100 - (health.memory_used_gb / health.memory_total_gb) * 100;
  const disk = 100 - (health.disk_used_gb / health.disk_total_gb) * 100;

  return Math.round((cpu * 0.4 + mem * 0.35 + disk * 0.25));
}

export function getLastHealth(device: Device): LastHealth | null {
  try {
    const meta = JSON.parse(device.metadata);
    return meta.last_health ?? null;
  } catch {
    return null;
  }
}

export function getMetricPercent(device: Device, metric: "cpu" | "memory" | "disk"): number {
  const health = getLastHealth(device);
  if (!health) return 0;
  switch (metric) {
    case "cpu": return Math.round(health.cpu_percent);
    case "memory": return Math.round((health.memory_used_gb / health.memory_total_gb) * 100);
    case "disk": return Math.round((health.disk_used_gb / health.disk_total_gb) * 100);
  }
}
