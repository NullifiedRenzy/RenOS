export interface Device {
  id: string;
  user_id: string;
  hostname: string;
  os: string;
  arch: string;
  agent_version: string;
  cert_fingerprint: string;
  enrolled_at: string | null;
  last_seen: string | null;
  status: "online" | "offline" | "degraded" | "decommissioned";
  labels: string;
  capabilities: string;
  metadata: string;
  created_at: string;
  updated_at: string;
}

export interface LastHealth {
  hostname: string;
  uptime_seconds: number;
  cpu_percent: number;
  memory_total_gb: number;
  memory_used_gb: number;
  disk_total_gb: number;
  disk_used_gb: number;
  local_ip: string;
  reported_at: string;
}

export interface User {
  id: string;
  email: string;
  display_name: string;
  role: string;
}

export interface LoginResponse {
  token: string;
  user: User;
}

export interface ListDevicesResponse {
  devices: Device[];
}

export interface DeviceDetailResponse {
  device: Device;
}

export interface ApiErrorBody {
  error: {
    code: string;
    message: string;
  };
}
