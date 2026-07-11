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

export interface Command {
  id: string;
  device_id: string;
  user_id: string;
  command: string;
  status: "pending" | "running" | "completed" | "failed" | "rejected";
  exit_code: number | null;
  output: string | null;
  error: string | null;
  requested_at: string;
  started_at: string | null;
  completed_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface CommandResponse {
  command: Command;
}

export interface ListCommandsResponse {
  commands: Command[];
}

export interface Container {
  id: string;
  device_id: string;
  container_id: string;
  name: string;
  image: string;
  status: string;
  state: string;
  ports: string;
  created_at: string;
  updated_at: string;
}

export interface ListContainersResponse {
  containers: Container[];
}

export interface ContainerAction {
  id: string;
  device_id: string;
  user_id: string;
  container_id: string;
  action: "start" | "stop" | "restart" | "logs" | "stats";
  status: "pending" | "running" | "completed" | "failed";
  exit_code: number | null;
  output: string | null;
  error: string | null;
  requested_at: string;
  started_at: string | null;
  completed_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface ContainerActionResponse {
  container_action: ContainerAction;
}

export interface DockerImage {
  id: string;
  device_id: string;
  image_id: string;
  repository: string;
  tag: string;
  size: string;
  created: string;
  created_at: string;
  updated_at: string;
}

export interface ListDockerImagesResponse {
  images: DockerImage[];
}

export interface DockerVolume {
  id: string;
  device_id: string;
  name: string;
  driver: string;
  mountpoint: string;
  created_at: string;
  updated_at: string;
}

export interface ListDockerVolumesResponse {
  volumes: DockerVolume[];
}

export interface DockerNetwork {
  id: string;
  device_id: string;
  network_id: string;
  name: string;
  driver: string;
  scope: string;
  created_at: string;
  updated_at: string;
}

export interface ListDockerNetworksResponse {
  networks: DockerNetwork[];
}

export interface SystemdService {
  id: string;
  device_id: string;
  unit_name: string;
  load_state: string;
  active_state: string;
  sub_state: string;
  description: string;
  enabled: string;
  created_at: string;
  updated_at: string;
}

export interface ListSystemdServicesResponse {
  services: SystemdService[];
}

export interface ServiceAction {
  id: string;
  device_id: string;
  user_id: string;
  unit_name: string;
  action: "start" | "stop" | "restart";
  status: "pending" | "running" | "completed" | "failed";
  exit_code: number | null;
  output: string | null;
  error: string | null;
  requested_at: string;
  created_at: string;
  updated_at: string;
}

export interface ServiceActionResponse {
  service_action: ServiceAction;
}

export interface FileEntry {
  name: string;
  path: string;
  size: number;
  file_type: "file" | "dir" | "symlink";
  permissions: string;
  modified: string;
}

export interface FileAction {
  id: string;
  device_id: string;
  user_id: string;
  action_type: "list_directory" | "read_file" | "search_files";
  path: string;
  query: string;
  status: "pending" | "running" | "completed" | "failed";
  exit_code: number | null;
  output: string | null;
  error: string | null;
  result_json: FileEntry[] | { content: string; size: number; path: string } | null;
  requested_at: string;
  started_at: string | null;
  completed_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface FileActionResponse {
  file_action: FileAction;
}

export interface ApiErrorBody {
  error: {
    code: string;
    message: string;
  };
}
