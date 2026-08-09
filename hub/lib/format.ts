const FIXED_BYTE_UNITS = ["B", "KB", "MB", "GB", "TB"];
const FILE_SIZE_UNITS = ["B", "KB", "MB", "GB", "TB"];

export function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${FIXED_BYTE_UNITS[i]}`;
}

export function formatFileSize(bytes: number): string {
  if (bytes === 0) return "0 B";
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(i === 0 ? 0 : 1)} ${FILE_SIZE_UNITS[i]}`;
}

export interface FormatUptimeOptions {
  empty?: string;
  alwaysHours?: boolean;
}

export function formatUptime(seconds: number, options: FormatUptimeOptions = {}): string {
  const { empty = "0d 0h", alwaysHours = false } = options;
  if (seconds <= 0) return empty;
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  if (d > 0) return `${d}d ${h}h`;
  if (alwaysHours || h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}

export function formatTime(iso: string | null): string {
  if (!iso) return "\u2014";
  return new Date(iso).toLocaleTimeString();
}

export function formatTimestamp(timestamp: string): string {
  const d = new Date(parseFloat(timestamp) * 1000);
  return d.toLocaleString();
}

export function usagePercent(used: number, total: number): number {
  if (total <= 0) return 0;
  return Math.round((used / total) * 100);
}
