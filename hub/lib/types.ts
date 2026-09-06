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
  device_state: "normal" | "unstable" | "cooldown";
  failure_count: number;
  last_failure_at: string | null;
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
  totp_enabled?: boolean;
}

export interface LoginResponse {
  token?: string;
  user?: User;
  mfa_required?: boolean;
}

export interface MFASetupResponse {
  secret: string;
  otpauth_url: string;
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
  action:
    | "start"
    | "stop"
    | "restart"
    | "logs"
    | "stats"
    | "kill"
    | "remove"
    | "exec"
    | "inspect"
    | "pause"
    | "unpause"
    | "rename"
    | "image_pull"
    | "image_remove"
    | "image_prune"
    | "volume_remove"
    | "volume_prune"
    | "network_disconnect"
    | "network_connect"
    | "system_prune";
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

export interface ComposeStack {
  id: string;
  device_id: string;
  name: string;
  status: string;
  config_files: string;
  services: string;
  working_dir: string;
  created_at: string;
  updated_at: string;
}

export interface ListComposeStacksResponse {
  compose_stacks: ComposeStack[];
}

export interface ContainerMetric {
  id: number;
  device_id: string;
  container_id: string;
  container_name: string;
  cpu_percent: number;
  memory_usage_bytes: number;
  memory_limit_bytes: number;
  memory_percent: number;
  network_rx_bytes: number;
  network_tx_bytes: number;
  block_read_bytes: number;
  block_write_bytes: number;
  pids: number;
  recorded_at: string;
}

export interface ListContainerMetricsResponse {
  metrics: ContainerMetric[];
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
  action_type:
    | "list_directory"
    | "read_file"
    | "search_files"
    | "create_directory"
    | "delete_path"
    | "rename_path"
    | "move_path"
    | "copy_path"
    | "set_permissions"
    | "upload_file"
    | "download_file"
    | "preview_file";
  path: string;
  target_path: string;
  query: string;
  metadata: Record<string, unknown> | null;
  status: "pending" | "running" | "completed" | "failed";
  exit_code: number | null;
  output: string | null;
  error: string | null;
  result_json:
    | FileEntry[]
    | { content: string; size: number; path: string }
    | { created: boolean; path: string }
    | { deleted: boolean; path: string }
    | { renamed: boolean; from: string; to: string }
    | { moved: boolean; from: string; to: string }
    | { copied: boolean; from: string; to: string }
    | { permissions_set: boolean; path: string; mode: string }
    | null;
  requested_at: string;
  started_at: string | null;
  completed_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface FileActionResponse {
  file_action: FileAction;
}

export interface FileTransfer {
  id: string;
  device_id: string;
  user_id: string;
  action_id: string | null;
  transfer_type: "upload" | "download";
  filename: string;
  remote_path: string;
  total_size: number;
  chunk_size: number;
  total_chunks: number;
  uploaded_chunks: number;
  status:
    | "pending"
    | "uploading"
    | "downloading"
    | "assembling"
    | "completed"
    | "failed"
    | "cancelled";
  mime_type: string;
  error: string;
  created_at: string;
  updated_at: string;
  completed_at: string | null;
}

export interface FileTransferResponse {
  file_transfer: FileTransfer;
}

export interface ListFileTransfersResponse {
  file_transfers: FileTransfer[];
  total: number;
  limit: number;
  offset: number;
}

export interface FilePermission {
  id: string;
  device_id: string;
  user_id: string;
  can_browse: boolean;
  can_modify: boolean;
  can_upload: boolean;
  can_download: boolean;
  can_delete: boolean;
  allowed_roots: string[];
  created_at: string;
  updated_at: string;
}

export interface FilePermissionResponse {
  file_permission: FilePermission;
}

export interface ListFilePermissionsResponse {
  file_permissions: FilePermission[];
}

export interface NetworkInterface {
  id: string;
  device_id: string;
  name: string;
  mac_address: string;
  ipv4: string[];
  ipv6: string[];
  status: "up" | "down";
  speed: number;
  interface_type: string;
  is_default_gateway: boolean;
  dns_servers: string[];
  mtu: number;
  created_at: string;
  updated_at: string;
}

export interface ListNetworkInterfacesResponse {
  interfaces: NetworkInterface[];
}

export interface StorageDevice {
  id: string;
  device_id: string;
  name: string;
  size: number;
  device_type: string;
  model: string;
  mount_point: string;
  filesystem: string;
  uuid: string;
  is_removable: boolean;
  is_readonly: boolean;
  used_bytes: number;
  available_bytes: number;
  created_at: string;
  updated_at: string;
}

export interface ListStorageDevicesResponse {
  devices: StorageDevice[];
}

export interface VirtualMachine {
  id: string;
  device_id: string;
  domain_name: string;
  uuid: string;
  state: string;
  vcpu_max: number;
  vcpu_current: number;
  memory_max_bytes: number;
  memory_current_bytes: number;
  os_type: string;
  autostart: boolean;
  persistent: boolean;
  created_at: string;
  updated_at: string;
}

export interface UpdateStatus {
  id: string;
  device_id: string;
  package_name: string;
  current_version: string;
  available_version: string;
  severity: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface ListUpdatesResponse {
  updates: UpdateStatus[];
}

export interface Plugin {
  id: string;
  device_id: string;
  name: string;
  description: string;
  plugin_type: string;
  status: string;
  last_output: string;
  last_exit_code: number;
  last_run_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface ChatMessage {
  role: string;
  content: string;
  tool_calls?: ToolCall[];
  tool_call_id?: string;
  name?: string;
}

export interface ToolCall {
  id: string;
  type: string;
  function: {
    name: string;
    arguments: string;
  };
}

export interface SuggestedAction {
  id: string;
  action: string;
  target: string;
  description: string;
  risk_level: string;
  command: string;
}

export interface Usage {
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
}

export interface ChatResponse {
  conversation_id: string;
  content: string;
  suggested_actions?: SuggestedAction[];
  usage?: Usage;
  model?: string;
}

export interface StreamChatResponse {
  type: string;
  delta?: string;
  conversation_id?: string;
  provider?: string;
  finish_reason?: string;
  usage?: Usage;
}

export interface ConversationSummary {
  id: string;
  title: string;
  message_count: number;
  created_at: string;
  updated_at: string;
}

export interface ConversationDetail {
  id: string;
  title: string;
  messages: ChatMessage[];
}

export interface ListConversationsResponse {
  conversations: ConversationSummary[];
}

export interface AISettingsResponse {
  providers: string[];
}

export interface ListPluginsResponse {
  plugins: Plugin[];
}

export interface ListVirtualMachinesResponse {
  vms: VirtualMachine[];
}

export interface DeviceGroup {
  id: string;
  name: string;
  description: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  device_count: number;
}

export interface CreateGroupPayload {
  name: string;
  description: string;
}

export interface GroupDetailResponse {
  group: DeviceGroup;
  device_ids: string[];
}

export interface GroupAction {
  id: string;
  group_id: string | null;
  action_type: string;
  target_type: string;
  target_value: string;
  status: string;
  requested_by: string;
  total_devices: number;
  completed_devices: number;
  failed_devices: number;
  result_summary: Record<string, unknown>;
  created_at: string;
  completed_at: string | null;
}

export interface RunCommandResult {
  device_id: string;
  command_id: string;
  status: string;
}

export interface GroupCommandResponse {
  group_action: GroupAction;
  job: Job;
}

export interface Job {
  id: string;
  type: string;
  payload: Record<string, unknown>;
  status:
    "pending" | "running" | "completed" | "failed" | "cancelled" | "stopped";
  total_devices: number;
  completed_devices: number;
  failed_devices: number;
  retries: number;
  max_retries: number;
  created_by: string;
  group_action_id: string | null;
  max_job_duration_ms: number;
  max_device_execution_ms: number;
  stop_on_failure: boolean;
  batch_size: number;
  batch_delay_ms: number;
  stopped_reason: string | null;
  created_at: string;
  started_at: string | null;
  completed_at: string | null;
  cancelled_at: string | null;
}

export interface Schedule {
  id: string;
  name: string;
  type: string;
  payload: Record<string, unknown>;
  target_type: string;
  target_ids: string[];
  cron_expression: string;
  enabled: boolean;
  last_run_at: string | null;
  next_run_at: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface ScheduleRun {
  id: string;
  schedule_id: string;
  job_id: string | null;
  status: string;
  started_at: string;
  completed_at: string | null;
}

export interface UserListItem {
  id: string;
  email: string;
  display_name: string;
  role: string;
  totp_enabled: boolean;
  created_at: string;
}

export interface Policy {
  id: string;
  name: string;
  description: string;
  type: string;
  conditions: Record<string, unknown>;
  action: string;
  priority: number;
  enabled: boolean;
  created_by: string | null;
  created_at: string;
  updated_at: string;
}

export interface CreatePolicyPayload {
  name: string;
  description?: string;
  type: string;
  conditions: Record<string, unknown>;
  action: string;
  priority?: number;
  enabled?: boolean;
}

export interface CreateSchedulePayload {
  name: string;
  type: string;
  payload: Record<string, unknown>;
  target_type: string;
  target_ids: string[];
  cron_expression: string;
}

export interface Metrics {
  total_jobs: number;
  running_jobs: number;
  failed_jobs: number;
  completed_jobs: number;
  stopped_jobs: number;
  avg_job_duration_ms: number;
  policy_blocks: number;
  schedule_executions: number;
}

export interface JobStats {
  total_jobs: number;
  running_jobs: number;
  failed_jobs: number;
  completed_jobs: number;
  stopped_jobs: number;
  cancelled_jobs: number;
  avg_job_duration_ms: number;
}

export interface SystemLog {
  id: string;
  level: string;
  source: string;
  message: string;
  metadata: Record<string, unknown>;
  created_at: string;
}

export interface SystemLogListResponse {
  logs: SystemLog[];
  total: number;
  limit: number;
  offset: number;
}

export interface ApiErrorBody {
  error: {
    code: string;
    message: string;
  };
}

export interface Execution {
  id: string;
  device_id: string;
  user_id: string;
  command: string;
  status: "pending" | "running" | "completed" | "failed" | "cancelled";
  exit_code: number | null;
  shell: string;
  timeout_ms: number;
  max_output_bytes: number;
  requested_at: string;
  started_at: string | null;
  completed_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface ExecutionOutput {
  id: number;
  execution_id: string;
  stream: "stdout" | "stderr";
  content: string;
  seq: number;
  created_at: string;
}

export interface ListExecutionsResponse {
  executions: Execution[];
  total: number;
  limit: number;
  offset: number;
}

export interface ExecutionResponse {
  execution: Execution;
}

export interface ExecutionOutputResponse {
  output: ExecutionOutput[];
}

export interface CreateExecutionPayload {
  command: string;
  shell?: string;
  timeout_ms?: number;
  max_output_bytes?: number;
}

// --- Process Management ---

export interface ProcessInfo {
  id: string;
  device_id: string;
  pid: number;
  name: string;
  executable: string;
  command_line: string;
  cpu_usage: number;
  memory_bytes: number;
  memory_usage: number;
  threads: number;
  open_handles: number;
  parent_pid: number;
  owner: string;
  status: string;
  start_time: number;
  collected_at: string;
  created_at: string;
}

export interface ProcessAction {
  id: string;
  device_id: string;
  user_id: string;
  pid: number;
  action_type: "kill" | "terminate" | "suspend" | "resume" | "restart";
  status: "pending" | "running" | "completed" | "failed";
  exit_code: number | null;
  output: string;
  error: string;
  requested_at: string;
  started_at: string | null;
  completed_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface ProcessHistoryEntry {
  id: number;
  device_id: string;
  total_cpu: number;
  total_memory_bytes: number;
  total_memory_usage: number;
  process_count: number;
  recorded_at: string;
}

export interface ListProcessesResponse {
  processes: ProcessInfo[];
  total_cpu: number;
  total_memory_bytes: number;
  total_memory_usage: number;
  process_count: number;
}

export interface ProcessHistoryResponse {
  history: ProcessHistoryEntry[];
}

export interface ProcessAggregatesResponse {
  total_cpu: number;
  total_memory_bytes: number;
  total_memory_usage: number;
  process_count: number;
}

export interface ProcessActionResponse {
  process_action: ProcessAction;
}

export interface ListProcessActionsResponse {
  process_actions: ProcessAction[];
}

// ─── Proxmox ───────────────────────────────────────────

export interface ProxmoxConnection {
  id: string;
  device_id: string;
  name: string;
  host: string;
  port: number;
  auth_type: "token" | "password";
  verify_ssl: boolean;
  status: "connected" | "disconnected" | "error";
  error_message: string;
  last_connected_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface ProxmoxNode {
  id: string;
  connection_id: string;
  node_name: string;
  status: "online" | "offline";
  cpu: number;
  max_cpu: number;
  memory: number;
  max_memory: number;
  disk: number;
  max_disk: number;
  uptime: number;
  ssl_fingerprint: string;
  version: string;
  created_at: string;
  updated_at: string;
}

export interface ProxmoxVM {
  id: string;
  connection_id: string;
  node_name: string;
  vm_id: number;
  name: string;
  status: "running" | "stopped" | "paused" | "suspended";
  cpu: number;
  cores: number;
  memory: number;
  disk: number;
  ip_address: string;
  os_type: string;
  template: boolean;
  ha_state: string;
  created_at: string;
  updated_at: string;
}

export interface ProxmoxLXC {
  id: string;
  connection_id: string;
  node_name: string;
  vm_id: number;
  name: string;
  status: "running" | "stopped";
  cpu: number;
  cores: number;
  memory: number;
  disk: number;
  ip_address: string;
  os_type: string;
  template: boolean;
  created_at: string;
  updated_at: string;
}

export interface ProxmoxStorage {
  id: string;
  connection_id: string;
  node_name: string;
  storage_id: string;
  storage_type: string;
  status: "active" | "inactive";
  total: number;
  used: number;
  content: string;
  created_at: string;
  updated_at: string;
}

export interface ProxmoxNetwork {
  id: string;
  connection_id: string;
  node_name: string;
  interface: string;
  iface_type: string;
  address: string;
  netmask: string;
  gateway: string;
  bridge: string;
  vlan: number | null;
  active: boolean;
  created_at: string;
  updated_at: string;
}

export interface ProxmoxSnapshot {
  id: string;
  connection_id: string;
  node_name: string;
  vm_id: number;
  snapshot_id: string;
  name: string;
  description: string;
  parent: string;
  created_at: string;
  updated_at: string;
}

export interface ProxmoxTask {
  id: string;
  connection_id: string;
  node_name: string;
  upid: string;
  task_type: string;
  status: "running" | "completed" | "failed";
  user_name: string;
  start_time: number;
  end_time: number;
  exit_status: string;
  created_at: string;
  updated_at: string;
}

export interface ListProxmoxConnectionsResponse {
  connections: ProxmoxConnection[];
}

export interface ListProxmoxNodesResponse {
  nodes: ProxmoxNode[];
}

export interface ListProxmoxVMsResponse {
  vms: ProxmoxVM[];
}

export interface ListProxmoxLXCsResponse {
  lxcs: ProxmoxLXC[];
}

export interface ListProxmoxStorageResponse {
  storage: ProxmoxStorage[];
}

export interface ListProxmoxNetworksResponse {
  networks: ProxmoxNetwork[];
}

export interface ListProxmoxSnapshotsResponse {
  snapshots: ProxmoxSnapshot[];
}

export interface ListProxmoxTasksResponse {
  tasks: ProxmoxTask[];
}

// ─── Terminal ───────────────────────────────────────────

export interface TerminalSession {
  id: string;
  device_id: string;
  user_id: string;
  session_token: string;
  status: "active" | "closed" | "timeout" | "error";
  cols: number;
  rows: number;
  shell: string;
  started_at: string;
  last_active_at: string;
  ended_at: string | null;
  close_reason: string;
  ip_address: string;
  user_agent: string;
  created_at: string;
  updated_at: string;
}

export interface TerminalRecording {
  id: string;
  session_id: string;
  direction: "input" | "output";
  seq_num: number;
  recorded_at: string;
}

export interface ListTerminalSessionsResponse {
  sessions: TerminalSession[];
}

export interface TerminalSessionDetailResponse {
  session: TerminalSession;
}

export interface TerminalRecordingResponse {
  recordings: TerminalRecording[];
}

// --- Installed Plugin Platform Types ---

export interface InstalledPlugin {
  id: string;
  name: string;
  display_name: string;
  version: string;
  author: string;
  description: string;
  plugin_type: string;
  status: string;
  error_message: string;
  permissions: string[];
  config: Record<string, unknown>;
  config_schema: Record<string, PluginConfigField>;
  dependencies: PluginDependency[];
  entry_point: string;
  blob_path: string;
  blob_hash: string;
  signature: string;
  enabled_at: string | null;
  disabled_at: string | null;
  last_error_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface PluginConfigField {
  type: string;
  default: unknown;
  description: string;
  required: boolean;
  options?: string[];
}

export interface PluginDependency {
  name: string;
  version: string;
}

export interface PluginRoute {
  id: string;
  plugin_id: string;
  method: string;
  path: string;
  handler_name: string;
  description: string;
  require_auth: boolean;
  min_role: string;
  rate_limit: number;
  created_at: string;
  updated_at: string;
}

export interface PluginPage {
  id: string;
  plugin_id: string;
  page_path: string;
  title: string;
  description: string;
  icon: string;
  component_name: string;
  parent_path: string;
  sort_order: number;
  require_auth: boolean;
  min_role: string;
  created_at: string;
  updated_at: string;
}

export interface PluginSidebarEntry {
  id: string;
  plugin_id: string;
  label: string;
  icon: string;
  path: string;
  parent_label: string;
  sort_order: number;
  min_role: string;
  created_at: string;
  updated_at: string;
}

export interface PluginWorker {
  id: string;
  plugin_id: string;
  worker_name: string;
  description: string;
  handler_name: string;
  concurrency: number;
  status: string;
  last_error: string;
  restart_count: number;
  created_at: string;
  updated_at: string;
}

export interface PluginSchedule {
  id: string;
  plugin_id: string;
  schedule_name: string;
  description: string;
  handler_name: string;
  cron_expr: string;
  interval_seconds: number;
  enabled: boolean;
  last_run_at: string | null;
  next_run_at: string | null;
  last_status: string;
  last_error: string;
  created_at: string;
  updated_at: string;
}

export interface ListInstalledPluginsResponse {
  plugins: InstalledPlugin[];
}

export interface InstalledPluginResponse {
  plugin: InstalledPlugin;
}

export interface PluginRoutesResponse {
  routes: PluginRoute[];
}

export interface PluginPagesResponse {
  pages: PluginPage[];
}

export interface PluginSidebarResponse {
  sidebar: PluginSidebarEntry[];
}

export interface PluginWorkersResponse {
  workers: PluginWorker[];
}

export interface PluginSchedulesResponse {
  schedules: PluginSchedule[];
}

// --- Fleet Management Types ---

export interface DeviceTag {
  id: string;
  device_id: string;
  key: string;
  value: string;
  created_at: string;
  updated_at: string;
}

export interface DeviceTagsResponse {
  data: DeviceTag[];
}

export interface TagKeysResponse {
  data: string[];
}

export interface DeviceTagSearchResponse {
  data: string[];
}

export interface DeviceInventory {
  id: string;
  device_id: string;
  cpu_model: string;
  cpu_cores: number;
  cpu_threads: number;
  cpu_freq_mhz: number;
  gpu_model: string;
  gpu_memory_mb: number;
  ram_total_mb: number;
  ram_used_mb: number;
  storage_total_gb: number;
  storage_used_gb: number;
  storage_type: string;
  network_interfaces: unknown[];
  kernel_version: string;
  os_family: string;
  os_version: string;
  installed_packages: unknown[];
  installed_software: unknown[];
  uptime_seconds: number;
  last_collected_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface DeviceInventoryResponse {
  data: DeviceInventory;
}

export interface DeviceInventoryListResponse {
  data: DeviceInventory[];
}

export interface ComplianceCheck {
  id: string;
  name: string;
  description: string;
  check_type: string;
  severity: string;
  config: Record<string, unknown>;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface ComplianceChecksResponse {
  data: ComplianceCheck[];
}

export interface ComplianceResult {
  id: string;
  check_id: string;
  device_id: string;
  status: string;
  message: string;
  details: Record<string, unknown>;
  last_checked_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface ComplianceResultsResponse {
  data: ComplianceResult[];
}

export interface ComplianceSummaryResponse {
  data: Record<string, number>;
}

export interface BulkOperation {
  id: string;
  name: string;
  action: string;
  target_type: string;
  target_ids: string[];
  status: string;
  total_count: number;
  completed_count: number;
  failed_count: number;
  config: Record<string, unknown>;
  error_message: string;
  created_by: string | null;
  started_at: string | null;
  completed_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface BulkOperationsResponse {
  data: BulkOperation[];
}

export interface FleetDashboardResponse {
  data: {
    total_devices: number;
    online_devices: number;
    offline_devices: number;
    total_groups: number;
    total_tags: number;
    compliance_pass: number;
    compliance_fail: number;
    compliance_pending: number;
    inventory_count: number;
    os_distribution: Record<string, number>;
    tag_distribution: Record<string, number>;
  };
}

// --- Automation Engine Types ---

export interface Workflow {
  id: string;
  name: string;
  description: string;
  enabled: boolean;
  version: number;
  trigger_type: string;
  trigger_config: Record<string, unknown>;
  tags: string[];
  created_by: string | null;
  created_at: string;
  updated_at: string;
}

export interface WorkflowStep {
  id: string;
  workflow_id: string;
  step_order: number;
  name: string;
  step_type: string;
  config: Record<string, unknown>;
  next_step_id: string | null;
  error_handler: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface WorkflowRun {
  id: string;
  workflow_id: string;
  status: string;
  trigger_type: string;
  trigger_data: Record<string, unknown>;
  current_step_id: string | null;
  context: Record<string, unknown>;
  error_message: string;
  started_at: string | null;
  completed_at: string | null;
  created_at: string;
  updated_at: string;
  step_runs?: WorkflowStepRun[];
}

export interface WorkflowStepRun {
  id: string;
  run_id: string;
  step_id: string;
  status: string;
  input: Record<string, unknown>;
  output: Record<string, unknown>;
  error_message: string;
  started_at: string | null;
  completed_at: string | null;
  created_at: string;
}

export interface WorkflowTemplate {
  id: string;
  name: string;
  description: string;
  category: string;
  icon: string;
  trigger_type: string;
  trigger_config: Record<string, unknown>;
  steps: WorkflowStep[];
  tags: string[];
  created_at: string;
  updated_at: string;
}

export interface WorkflowsResponse {
  data: Workflow[];
}

export interface WorkflowStepsResponse {
  data: WorkflowStep[];
}

export interface WorkflowRunsResponse {
  data: WorkflowRun[];
}

export interface WorkflowRunDetailResponse {
  data: WorkflowRun;
}

export interface WorkflowTemplatesResponse {
  data: WorkflowTemplate[];
}

// --- Monitoring Platform Types ---

export interface MonitoringMetric {
  id: string;
  device_id: string | null;
  metric_name: string;
  metric_value: number;
  labels: Record<string, unknown>;
  recorded_at: string;
  created_at: string;
}

export interface AlertRule {
  id: string;
  name: string;
  description: string;
  metric_name: string;
  condition: string;
  threshold: number;
  duration_seconds: number;
  severity: string;
  enabled: boolean;
  notification_channels: unknown[];
  created_at: string;
  updated_at: string;
}

export interface AlertInstance {
  id: string;
  rule_id: string;
  device_id: string | null;
  status: string;
  value: number;
  message: string;
  started_at: string;
  resolved_at: string | null;
  acknowledged_by: string | null;
  acknowledged_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface MonitoringDashboard {
  id: string;
  name: string;
  description: string;
  layout: unknown[];
  is_default: boolean;
  created_by: string | null;
  created_at: string;
  updated_at: string;
  widgets?: MonitoringWidget[];
}

export interface MonitoringWidget {
  id: string;
  dashboard_id: string;
  widget_type: string;
  title: string;
  config: Record<string, unknown>;
  position_x: number;
  position_y: number;
  width: number;
  height: number;
  created_at: string;
}

// M65: hydrated display data for config-driven widget types, keyed by widget ID.
export interface DeviceStatusData {
  hostname: string;
  status: string;
  os: string;
  last_seen?: string;
}

export interface DevicesOverviewData {
  total: number;
  online: number;
  offline: number;
  error: number;
  by_status: Record<string, number>;
  os_dist: Record<string, number>;
}

export interface MetricSampleData {
  value: number;
  device_id?: string;
  recorded_at: string;
}

export interface MetricData {
  metric: string;
  samples: MetricSampleData[];
}

export interface AlertsWidgetData {
  firing: number;
  active: string[];
}

export interface LogEntryData {
  level: string;
  source: string;
  message: string;
  recorded_at: string;
}

export interface WorkflowsWidgetData {
  total: number;
  enabled: number;
  recent_runs: { id: string; status: string }[];
}

export interface AIUsageData {
  conversations: number;
}

export interface ContainersWidgetData {
  total: number;
  running: number;
  stopped: number;
}

export interface ProxmoxWidgetData {
  connections: number;
  nodes: number;
  vms: number;
  lxcs: number;
}

export type WidgetData =
  | DevicesOverviewData
  | DeviceStatusData[]
  | MetricData
  | AlertsWidgetData
  | LogEntryData[]
  | WorkflowsWidgetData
  | AIUsageData
  | ContainersWidgetData
  | ProxmoxWidgetData
  | Record<string, unknown>;

export interface MonitoringDashboardDetail extends MonitoringDashboard {
  widgets: MonitoringWidget[];
  widget_data: Record<string, WidgetData>;
}

export interface MonitoringLog {
  id: string;
  device_id: string | null;
  source: string;
  level: string;
  message: string;
  fields: Record<string, unknown>;
  recorded_at: string;
  created_at: string;
}

export interface MonitoringMetricsResponse {
  data: MonitoringMetric[];
}

export interface AlertRulesResponse {
  data: AlertRule[];
}

export interface AlertInstancesResponse {
  data: AlertInstance[];
}

export interface AlertSummaryResponse {
  data: Record<string, number>;
}

export interface MonitoringDashboardsResponse {
  data: MonitoringDashboard[];
}

export interface MonitoringLogsResponse {
  data: MonitoringLog[];
}

export interface MonitoringLogStatsResponse {
  data: Record<string, number>;
}

// --- RenVault Types ---

export interface VaultSecret {
  id: string;
  name: string;
  description: string;
  path: string;
  secret_type: string;
  version: number;
  metadata: Record<string, unknown>;
  created_by: string | null;
  created_at: string;
  updated_at: string;
}

export interface VaultSecretVersion {
  id: string;
  secret_id: string;
  version: number;
  metadata: Record<string, unknown>;
  changed_by: string | null;
  created_at: string;
}

export interface VaultKey {
  id: string;
  name: string;
  algorithm: string;
  public_key: string;
  rotated_at: string;
  expires_at: string | null;
  enabled: boolean;
  created_at: string;
}

export interface VaultPolicy {
  id: string;
  name: string;
  description: string;
  paths: string[];
  capabilities: string[];
  created_at: string;
  updated_at: string;
}

export interface VaultAuditEntry {
  id: string;
  secret_id: string | null;
  action: string;
  user_id: string | null;
  ip_address: string;
  details: Record<string, unknown>;
  created_at: string;
}

export interface VaultSecretsResponse {
  data: VaultSecret[];
}

export interface VaultSecretVersionsResponse {
  data: VaultSecretVersion[];
}

export interface VaultKeysResponse {
  data: VaultKey[];
}

export interface VaultPoliciesResponse {
  data: VaultPolicy[];
}

export interface VaultAuditLogResponse {
  data: VaultAuditEntry[];
}

// --- Notification Types ---

export interface NotificationProvider {
  id: string;
  name: string;
  provider_type: string;
  config: Record<string, unknown>;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface NotificationTemplate {
  id: string;
  name: string;
  subject: string;
  body: string;
  template_type: string;
  variables: string[];
  created_at: string;
  updated_at: string;
}

export interface NotificationRule {
  id: string;
  name: string;
  description: string;
  event_type: string;
  conditions: Record<string, unknown>;
  provider_ids: string[];
  template_id: string | null;
  cooldown_seconds: number;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface NotificationHistoryEntry {
  id: string;
  rule_id: string | null;
  provider_id: string | null;
  template_id: string | null;
  event_type: string;
  subject: string;
  body: string;
  status: string;
  error_message: string;
  sent_at: string | null;
  created_at: string;
}

export interface NotificationProvidersResponse {
  data: NotificationProvider[];
}

export interface NotificationTemplatesResponse {
  data: NotificationTemplate[];
}

export interface NotificationRulesResponse {
  data: NotificationRule[];
}

export interface NotificationHistoryResponse {
  data: NotificationHistoryEntry[];
}

export interface NotificationStatsResponse {
  data: Record<string, number>;
}

// --- Marketplace Types ---

export interface MarketplacePlugin {
  id: string;
  name: string;
  display_name: string;
  description: string;
  author: string;
  category: string;
  version: string;
  min_renos_version: string;
  download_url: string;
  homepage: string;
  icon: string;
  screenshots: string[];
  tags: string[];
  downloads: number;
  rating: number;
  verified: boolean;
  signature: string;
  checksum: string;
  manifest: Record<string, unknown>;
  last_synced_at?: string;
  created_at: string;
  updated_at: string;
}

export interface MarketplaceInstallation {
  id: string;
  plugin_id: string;
  installed_version: string;
  status: "installed" | "upgrading" | "rolling_back" | "failed";
  config: Record<string, unknown>;
  installed_at: string;
  updated_at: string;
}

export interface MarketplaceInstallHistoryEntry {
  id: string;
  installation_id: string;
  version: string;
  action: "install" | "upgrade" | "rollback";
  checksum: string;
  verified: boolean;
  notes: string;
  created_at: string;
}

export interface MarketplacePluginsResponse {
  data: MarketplacePlugin[];
}

export interface MarketplaceInstallationsResponse {
  data: MarketplaceInstallation[];
}

export interface MarketplaceInstallHistoryResponse {
  data: MarketplaceInstallHistoryEntry[];
}

// --- Identity & Access Management Types ---

export interface IdentityProvider {
  id: string;
  name: string;
  provider_type: "oidc" | "oauth2" | "ldap";
  client_id: string;
  client_secret: string;
  issuer_url: string;
  auth_url: string;
  token_url: string;
  userinfo_url: string;
  jwks_url: string;
  scopes: string[];
  attribute_mapping: Record<string, string>;
  group_mapping: Record<string, string>;
  base_dn: string;
  bind_dn: string;
  bind_password: string;
  user_search_filter: string;
  group_search_filter: string;
  enabled: boolean;
  default_role: string;
  auto_provision: boolean;
  created_at: string;
  updated_at: string;
}

export interface OAuth2App {
  id: string;
  name: string;
  client_id: string;
  client_secret_hash: string;
  redirect_uris: string[];
  grant_types: string[];
  scopes: string[];
  description: string;
  enabled: boolean;
  created_by: string | null;
  created_at: string;
  updated_at: string;
}

export interface APIToken {
  id: string;
  user_id: string;
  name: string;
  token_hash: string;
  token_prefix: string;
  scopes: string[];
  expires_at: string | null;
  last_used_at: string | null;
  enabled: boolean;
  created_at: string;
}

export interface IAMRole {
  id: string;
  name: string;
  display_name: string;
  description: string;
  permissions: string[];
  is_system: boolean;
  created_at: string;
  updated_at: string;
}

export interface IAMRoleAssignment {
  id: string;
  user_id: string;
  role_id: string;
  resource_type: string;
  resource_id: string;
  granted_by: string | null;
  expires_at: string | null;
  created_at: string;
}

export interface ApprovalWorkflow {
  id: string;
  name: string;
  description: string;
  action_pattern: string;
  required_approvals: number;
  approver_roles: string[];
  approver_users: string[];
  timeout_seconds: number;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface ApprovalRequest {
  id: string;
  workflow_id: string;
  action_type: string;
  action_data: Record<string, unknown>;
  requested_by: string;
  status: "pending" | "approved" | "rejected" | "expired" | "cancelled";
  approval_count: number;
  required_count: number;
  expires_at: string;
  decided_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface ApprovalDecision {
  id: string;
  request_id: string;
  approver_id: string;
  decision: "approved" | "rejected";
  comment: string;
  decided_at: string;
}

export interface IAMAuditEntry {
  id: string;
  actor_id: string;
  action: string;
  resource_type: string;
  resource_id: string;
  details: Record<string, unknown>;
  ip_address: string;
  created_at: string;
}

export interface IdentityProvidersResponse {
  data: IdentityProvider[];
}
export interface OAuth2AppsResponse {
  data: OAuth2App[];
}
export interface APITokensResponse {
  data: APIToken[];
}
export interface IAMRolesResponse {
  data: IAMRole[];
}
export interface IAMRoleAssignmentsResponse {
  data: IAMRoleAssignment[];
}
export interface ApprovalWorkflowsResponse {
  data: ApprovalWorkflow[];
}
export interface ApprovalRequestsResponse {
  data: ApprovalRequest[];
}
export interface IAMAuditLogResponse {
  data: IAMAuditEntry[];
}

// --- Deployment Platform Types ---

export interface PreflightCheck {
  name: string;
  category: string;
  status: "pass" | "warn" | "fail" | "skipped";
  message: string;
  details?: string;
}

export interface PreflightReport {
  os: string;
  arch: string;
  hostname: string;
  timestamp: string;
  results: PreflightCheck[];
  pass_count: number;
  warn_count: number;
  fail_count: number;
  all_passed: boolean;
}

export interface InstallationStatus {
  phase: string;
  progress: number;
  message: string;
  error?: string;
  checks?: PreflightCheck[];
  started_at: string;
  completed_at?: string;
}

export interface SystemInfo {
  os: string;
  arch: string;
  hostname: string;
  cpu_cores: number;
  docker: string;
  docker_compose: string;
}

export interface PreflightResponse {
  data: PreflightReport;
}
export interface SystemInfoResponse {
  data: SystemInfo;
}
export interface InstallationStatusResponse {
  data: InstallationStatus;
}

// --- Update Platform Types ---

export interface VersionInfo {
  current_version: string;
  latest_version: string;
  channel: string;
  update_available: boolean;
}

export interface UpdatePlan {
  current_version: string;
  target_version: string;
  channel: string;
  components: string[];
  steps: UpdatePlanStep[];
  requires_backup: boolean;
  estimated_time: string;
}

export interface UpdatePlanStep {
  name: string;
  description: string;
  component: string;
  action: string;
  order: number;
}

export interface UpdateHistoryEntry {
  id: string;
  version_from: string;
  version_to: string;
  channel: string;
  status: string;
  components: string[];
  backup_id?: string;
  error_message: string;
  started_by?: string;
  started_at: string;
  completed_at?: string;
  created_at: string;
}

export interface UpdateBackup {
  id: string;
  version: string;
  backup_type: string;
  backup_path: string;
  database_dump_path: string;
  config_backup_path: string;
  size_bytes: number;
  checksum: string;
  status: string;
  created_at: string;
  expires_at?: string;
}

export interface VersionInfoResponse {
  data: VersionInfo;
}
export interface UpdateCheckResponse {
  data: { update_available: boolean; release?: Record<string, unknown> };
}
export interface UpdatePlanResponse {
  data: UpdatePlan;
}
export interface UpdateHistoryResponse {
  data: UpdateHistoryEntry[];
}
export interface UpdateBackupsResponse {
  data: UpdateBackup[];
}

// --- Cluster Types ---

export interface ClusterNode {
  id: string;
  node_id: string;
  hostname: string;
  address: string;
  port: number;
  role: "leader" | "follower" | "candidate" | "standby";
  status: "active" | "offline" | "failed";
  version: string;
  last_heartbeat_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface ClusterElection {
  id: string;
  term: number;
  candidate_id: string;
  voter_id: string;
  granted: boolean;
  created_at: string;
}

export interface ClusterTask {
  id: string;
  task_type: string;
  task_data: Record<string, unknown>;
  assigned_to: string | null;
  status: "pending" | "assigned" | "running" | "completed" | "failed";
  priority: number;
  retries: number;
  max_retries: number;
  result: Record<string, unknown>;
  error_message: string;
  started_at: string | null;
  completed_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface ClusterState {
  key: string;
  value: Record<string, unknown>;
  owner: string;
  version: number;
  ttl_seconds: number | null;
  expires_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface ClusterMetric {
  id: string;
  node_id: string;
  cpu_usage: number;
  memory_usage: number;
  disk_usage: number;
  active_connections: number;
  task_count: number;
  recorded_at: string;
  created_at: string;
}

export interface ClusterDashboard {
  nodes: ClusterNode[];
  leader_id: string;
  term: number;
  pending_tasks: number;
  metrics: ClusterMetric[];
}

export interface ClusterDashboardResponse {
  data: ClusterDashboard;
}
export interface ClusterNodesResponse {
  data: ClusterNode[];
}
export interface ClusterTasksResponse {
  data: ClusterTask[];
}
export interface ClusterStateResponse {
  data: ClusterState[];
}
export interface ClusterElectionsResponse {
  data: ClusterElection[];
}

// --- Mobile Platform Types ---

export interface PushRegistration {
  id: string;
  user_id: string;
  platform: "fcm" | "apns" | "webpush";
  token: string;
  device_name: string;
  device_model: string;
  os_version: string;
  app_version: string;
  enabled: boolean;
  last_used_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface MobileSession {
  id: string;
  user_id: string;
  device_id: string;
  device_name: string;
  device_model: string;
  os_version: string;
  app_version: string;
  ip_address: string;
  user_agent: string;
  biometric_enabled: boolean;
  push_enabled: boolean;
  last_active_at: string;
  expires_at: string;
  created_at: string;
  updated_at: string;
}

export interface OfflineSyncItem {
  id: string;
  user_id: string;
  device_id: string;
  action: string;
  payload: Record<string, unknown>;
  priority: number;
  status: "pending" | "processing" | "completed" | "failed" | "expired";
  retry_count: number;
  max_retries: number;
  error_message: string;
  created_at: string;
  processed_at: string | null;
}

export interface DeviceApproval {
  id: string;
  user_id: string;
  device_id: string;
  device_name: string;
  device_model: string;
  ip_address: string;
  fingerprint: string;
  status: "pending" | "approved" | "rejected";
  approved_by: string | null;
  approved_at: string | null;
  reject_reason: string;
  expires_at: string;
  created_at: string;
  updated_at: string;
}

export interface MobileDashboard {
  total_devices: number;
  online_devices: number;
  offline_devices: number;
  updated_at: string;
}

export interface PushRegistrationsResponse {
  data: PushRegistration[];
}
export interface MobileSessionsResponse {
  data: MobileSession[];
}
export interface OfflineSyncResponse {
  data: OfflineSyncItem[];
}
export interface DeviceApprovalsResponse {
  data: DeviceApproval[];
}

// --- Developer Platform Types ---

export interface WebhookSubscription {
  id: string;
  user_id: string;
  url: string;
  events: string[];
  secret: string;
  active: boolean;
  headers: Record<string, string>;
  retry_policy: number;
  created_at: string;
  updated_at: string;
}

export interface WebhookDelivery {
  id: string;
  subscription_id: string;
  event: string;
  payload: Record<string, unknown>;
  status_code: number;
  response: string;
  success: boolean;
  retry_count: number;
  delivered_at: string;
}

export interface DeveloperAPIKey {
  id: string;
  user_id: string;
  name: string;
  key_hash: string;
  key_prefix: string;
  scopes: string[];
  expires_at: string | null;
  last_used_at: string | null;
  active: boolean;
  created_at: string;
}

export interface WebhookSubscriptionsResponse {
  data: WebhookSubscription[];
}
export interface WebhookDeliveriesResponse {
  data: WebhookDelivery[];
}
export interface WebhookEventsResponse {
  data: string[];
}
export interface DeveloperAPIKeysResponse {
  data: DeveloperAPIKey[];
}

// --- Onboarding ---

export interface OnboardingStep {
  done: boolean;
  title: string;
  description: string;
}

export interface OnboardingStatus {
  onboarding_complete: boolean;
  admin_user: boolean;
  ai_provider_configured: boolean;
  smtp_configured: boolean;
  notifications_configured: boolean;
  device_enrolled: boolean;
  plugins_installed: number;
  backups_enabled: boolean;
  has_workflows: number;
  steps: Record<string, OnboardingStep>;
}

// --- Demo mode ---

export interface DemoStatus {
  demo_loaded: boolean;
}

export interface DemoSeededEntity {
  id?: string;
  name: string;
}

export interface DemoSeededUser {
  id: string;
  email: string;
  display_name: string;
  role: string;
}

export interface DemoSeedResult {
  seeded: boolean;
  demo_mode: boolean;
  user: DemoSeededUser | null;
  devices: DemoSeededEntity[];
  workflows: DemoSeededEntity[];
  alert_rules: DemoSeededEntity[];
  dashboards: DemoSeededEntity[];
  notifications: DemoSeededEntity[];
  marketplace: DemoSeededEntity[];
  installed_plugins: DemoSeededEntity[];
  jobs: DemoSeededEntity[];
}

// --- Health dashboard ---

export type HealthCheckStatus =
  | "healthy"
  | "degraded"
  | "warning"
  | "not_configured"
  | "disabled"
  | "unknown"
  | "critical"
  | "unhealthy";

export interface HealthCheck {
  status: HealthCheckStatus;
  detail?: string;
  latency_ms?: number;
}

export interface HealthDashboard {
  score: number;
  status: string;
  checks: Record<string, HealthCheck>;
}

// --- Global search ---

export interface SearchResult {
  type: string;
  id: string;
  title: string;
  subtitle: string;
  href: string;
}

export interface SearchResponse {
  query: string;
  total: number;
  results: Record<string, SearchResult[]>;
}

// --- In-app notification center ---

export type InAppNotificationType =
  "alert" | "workflow" | "job" | "ai" | "system" | "mention";

export interface InAppNotification {
  id: string;
  user_id: string;
  title: string;
  body: string;
  type: InAppNotificationType;
  entity_type: string;
  entity_id: string;
  metadata: Record<string, unknown>;
  is_read: boolean;
  read_at: string | null;
  created_at: string;
}

export interface UnreadCountResponse {
  count: number;
}
