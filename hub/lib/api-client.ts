import type {
  LoginResponse, ListDevicesResponse, DeviceDetailResponse,
  CommandResponse, ListCommandsResponse,
  ListContainersResponse, ContainerActionResponse,
  ListDockerImagesResponse, ListDockerVolumesResponse, ListDockerNetworksResponse,
  ListSystemdServicesResponse, ServiceActionResponse,
  FileActionResponse, ListNetworkInterfacesResponse,
  ListStorageDevicesResponse, ListVirtualMachinesResponse,
  ListUpdatesResponse, ListPluginsResponse,
  ChatResponse, ListConversationsResponse, ConversationDetail,
  DeviceGroup, GroupDetailResponse, GroupCommandResponse, GroupAction,
  Job, Schedule, CreateSchedulePayload, ScheduleRun, UserListItem,
  Policy, CreatePolicyPayload,
  Metrics, JobStats, SystemLog, SystemLogListResponse,
  ApiErrorBody,
  ListExecutionsResponse, ExecutionResponse, ExecutionOutputResponse,
  CreateExecutionPayload,
  FileTransfer, ListFileTransfersResponse, FileTransferResponse,
  FilePermission, FilePermissionResponse, ListFilePermissionsResponse,
  ListProcessesResponse, ProcessHistoryResponse, ProcessAggregatesResponse,
  ProcessActionResponse, ListProcessActionsResponse,
  ListComposeStacksResponse, ListContainerMetricsResponse,
  ListProxmoxConnectionsResponse, ListProxmoxNodesResponse, ListProxmoxVMsResponse,
  ListProxmoxLXCsResponse, ListProxmoxStorageResponse, ListProxmoxNetworksResponse,
  ListProxmoxSnapshotsResponse, ListProxmoxTasksResponse,
  ListInstalledPluginsResponse, InstalledPluginResponse,
  PluginRoutesResponse, PluginPagesResponse, PluginSidebarResponse,
  PluginWorkersResponse, PluginSchedulesResponse,
  AISettingsResponse,
  FleetDashboardResponse, DeviceTagsResponse, TagKeysResponse, DeviceTagSearchResponse,
  DeviceInventoryResponse, DeviceInventory, DeviceInventoryListResponse,
  ComplianceChecksResponse, ComplianceCheck, ComplianceResultsResponse, ComplianceResult,
  ComplianceSummaryResponse, BulkOperation, BulkOperationsResponse,
  Workflow, WorkflowStep, WorkflowRun, WorkflowStepRun, WorkflowTemplate,
  WorkflowsResponse, WorkflowStepsResponse, WorkflowRunsResponse, WorkflowRunDetailResponse, WorkflowTemplatesResponse,
  MonitoringMetric, AlertRule, AlertInstance, MonitoringDashboard, MonitoringWidget, MonitoringLog,
  MonitoringMetricsResponse, AlertRulesResponse, AlertInstancesResponse, AlertSummaryResponse,
  MonitoringDashboardsResponse, MonitoringLogsResponse, MonitoringLogStatsResponse, MonitoringDashboardDetail,
  VaultSecret, VaultSecretVersion, VaultKey, VaultPolicy, VaultAuditEntry,
  VaultSecretsResponse, VaultSecretVersionsResponse, VaultKeysResponse, VaultPoliciesResponse, VaultAuditLogResponse,
  NotificationProvider, NotificationTemplate, NotificationRule, NotificationHistoryEntry,
  NotificationProvidersResponse, NotificationTemplatesResponse, NotificationRulesResponse, NotificationHistoryResponse, NotificationStatsResponse,
  MarketplacePlugin, MarketplaceInstallation, MarketplacePluginsResponse, MarketplaceInstallationsResponse, MarketplaceInstallHistoryResponse,
  IdentityProvider, OAuth2App, APIToken, IAMRole, IAMRoleAssignment, ApprovalWorkflow, ApprovalRequest,
  IdentityProvidersResponse, OAuth2AppsResponse, APITokensResponse, IAMRolesResponse, IAMRoleAssignmentsResponse,
  ApprovalWorkflowsResponse, ApprovalRequestsResponse, IAMAuditLogResponse,
  PreflightResponse, SystemInfoResponse, InstallationStatusResponse,
  VersionInfoResponse, UpdateCheckResponse, UpdatePlanResponse, UpdateHistoryResponse, UpdateBackupsResponse,
  ClusterDashboardResponse, ClusterNodesResponse, ClusterTasksResponse, ClusterStateResponse, ClusterElectionsResponse,
  ClusterNode, ClusterTask, ClusterState, ClusterMetric,
  PushRegistrationsResponse, MobileSessionsResponse, OfflineSyncResponse, DeviceApprovalsResponse,
  MobileDashboard, OfflineSyncItem, DeviceApproval,
  WebhookSubscriptionsResponse, WebhookDeliveriesResponse, WebhookEventsResponse, DeveloperAPIKeysResponse,
  WebhookSubscription, WebhookDelivery, DeveloperAPIKey,
  OnboardingStatus, DemoStatus, DemoSeedResult, HealthDashboard,
  SearchResponse,
  InAppNotification, UnreadCountResponse,
} from "@/lib/types";

const API_BASE = "/api/v1";

// Default page size requested for unbounded list endpoints. The server clamps
// to [1, 1000]; requesting the max preserves the previous "return everything"
// behaviour up to the cap.
export const LIST_LIMIT = 1000;

export class ApiRequestError extends Error {
  constructor(
    message: string,
    public code: string,
    public status: number,
  ) {
    super(message);
    this.name = "ApiRequestError";
  }
}

function isApiErrorBody(obj: unknown): obj is ApiErrorBody {
  return typeof obj === "object" && obj !== null && "error" in obj;
}

class ApiClient {
  private accessToken: string | null = null;

  setAccessToken(token: string | null) {
    this.accessToken = token;
  }

  getAccessToken(): string | null {
    return this.accessToken;
  }

  private async request<T>(
    method: string,
    path: string,
    body?: unknown,
    options?: { signal?: AbortSignal },
  ): Promise<T> {
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
    };

    if (this.accessToken) {
      headers["Authorization"] = `Bearer ${this.accessToken}`;
    }

    let res: Response;
    try {
      res = await fetch(`${API_BASE}${path}`, {
        method,
        headers,
        body: body ? JSON.stringify(body) : undefined,
        signal: options?.signal,
      });
    } catch {
      throw new ApiRequestError("Connection error. Is the server running?", "NETWORK_ERROR", 0);
    }

    if (!res.ok) {
      if (res.status === 401) {
        this.accessToken = null;
        try { localStorage.removeItem("renos_auth"); } catch { }
        if (typeof window !== "undefined" && !window.location.pathname.startsWith("/login")) {
          window.location.href = "/login";
        }
      }
      let errorMessage = `Request failed: ${res.status}`;
      let errorCode = "UNKNOWN_ERROR";
      try {
        const errorBody: unknown = await res.json();
        if (isApiErrorBody(errorBody)) {
          errorMessage = errorBody.error.message;
          errorCode = errorBody.error.code;
        }
      } catch { }
      throw new ApiRequestError(errorMessage, errorCode, res.status);
    }

    const json = await res.json();
    if (json && typeof json === "object" && "data" in json) {
      return json.data as T;
    }
    return json as T;
  }

  async login(email: string, password: string): Promise<LoginResponse> {
    const data = await this.request<LoginResponse>("POST", "/auth/login", { email, password });
    this.setAccessToken(data.token);
    return data;
  }

  async listDevices(limit?: number, offset?: number): Promise<ListDevicesResponse> {
    const params = new URLSearchParams();
    if (limit !== undefined) params.set("limit", String(limit));
    if (offset !== undefined) params.set("offset", String(offset));
    return this.request<ListDevicesResponse>("GET", `/devices?${params}`);
  }

  async getDevice(id: string): Promise<DeviceDetailResponse> {
    return this.request<DeviceDetailResponse>("GET", `/devices/${id}`);
  }

  async requestCommand(deviceId: string, command: string): Promise<CommandResponse> {
    return this.request<CommandResponse>("POST", `/devices/${deviceId}/commands`, { command });
  }

  async listCommands(deviceId: string): Promise<ListCommandsResponse> {
    return this.request<ListCommandsResponse>("GET", `/devices/${deviceId}/commands`);
  }

  // --- Executions (NATS-based real-time) ---

  async createExecution(deviceId: string, data: CreateExecutionPayload): Promise<ExecutionResponse> {
    return this.request<ExecutionResponse>("POST", `/devices/${deviceId}/executions`, data);
  }

  async listExecutions(deviceId: string, limit?: number, offset?: number): Promise<ListExecutionsResponse> {
    const params = new URLSearchParams();
    if (limit) params.set("limit", String(limit));
    if (offset) params.set("offset", String(offset));
    const qs = params.toString();
    return this.request<ListExecutionsResponse>("GET", `/devices/${deviceId}/executions${qs ? `?${qs}` : ""}`);
  }

  async getExecution(deviceId: string, execId: string): Promise<ExecutionResponse> {
    return this.request<ExecutionResponse>("GET", `/devices/${deviceId}/executions/${execId}`);
  }

  async getExecutionOutput(deviceId: string, execId: string): Promise<ExecutionOutputResponse> {
    return this.request<ExecutionOutputResponse>("GET", `/devices/${deviceId}/executions/${execId}/output`);
  }

  async cancelExecution(deviceId: string, execId: string): Promise<void> {
    return this.request("POST", `/devices/${deviceId}/executions/${execId}/cancel`);
  }

  async listContainers(deviceId: string): Promise<ListContainersResponse> {
    return this.request<ListContainersResponse>("GET", `/devices/${deviceId}/containers`);
  }

  async requestContainerAction(deviceId: string, containerId: string, action: string): Promise<ContainerActionResponse> {
    return this.request<ContainerActionResponse>("POST", `/devices/${deviceId}/containers/${containerId}/${action}`);
  }

  async listDockerImages(deviceId: string): Promise<ListDockerImagesResponse> {
    return this.request<ListDockerImagesResponse>("GET", `/devices/${deviceId}/docker-images`);
  }

  async listDockerVolumes(deviceId: string): Promise<ListDockerVolumesResponse> {
    return this.request<ListDockerVolumesResponse>("GET", `/devices/${deviceId}/docker-volumes`);
  }

  async listDockerNetworks(deviceId: string): Promise<ListDockerNetworksResponse> {
    return this.request<ListDockerNetworksResponse>("GET", `/devices/${deviceId}/docker-networks`);
  }

  // --- Compose Stacks ---

  async listComposeStacks(deviceId: string): Promise<ListComposeStacksResponse> {
    return this.request<ListComposeStacksResponse>("GET", `/devices/${deviceId}/compose-stacks`);
  }

  // --- Container Metrics (live) ---

  async listContainerMetrics(deviceId: string): Promise<ListContainerMetricsResponse> {
    return this.request<ListContainerMetricsResponse>("GET", `/devices/${deviceId}/container-metrics`);
  }

  async getContainerMetricHistory(deviceId: string, containerId: string, minutes?: number): Promise<ListContainerMetricsResponse> {
    const params = new URLSearchParams();
    if (minutes) params.set("minutes", String(minutes));
    const qs = params.toString();
    return this.request<ListContainerMetricsResponse>("GET", `/devices/${deviceId}/container-metrics/${encodeURIComponent(containerId)}${qs ? `?${qs}` : ""}`);
  }

  async listSystemdServices(deviceId: string): Promise<ListSystemdServicesResponse> {
    return this.request<ListSystemdServicesResponse>("GET", `/devices/${deviceId}/systemd-services`);
  }

  async requestServiceAction(deviceId: string, unitName: string, action: string): Promise<ServiceActionResponse> {
    return this.request<ServiceActionResponse>("POST", `/devices/${deviceId}/services/${encodeURIComponent(unitName)}/${action}`);
  }

  async listNetworkInterfaces(deviceId: string): Promise<ListNetworkInterfacesResponse> {
    return this.request<ListNetworkInterfacesResponse>("GET", `/devices/${deviceId}/network-interfaces`);
  }

  async listStorageDevices(deviceId: string): Promise<ListStorageDevicesResponse> {
    return this.request<ListStorageDevicesResponse>("GET", `/devices/${deviceId}/storage-devices`);
  }

  async listVirtualMachines(deviceId: string): Promise<ListVirtualMachinesResponse> {
    return this.request<ListVirtualMachinesResponse>("GET", `/devices/${deviceId}/virtual-machines`);
  }

  async requestVMAction(deviceId: string, domainName: string, action: string): Promise<unknown> {
    return this.request("POST", `/devices/${deviceId}/vms/${encodeURIComponent(domainName)}/${action}`);
  }

  async listUpdates(deviceId: string): Promise<ListUpdatesResponse> {
    return this.request<ListUpdatesResponse>("GET", `/devices/${deviceId}/updates`);
  }

  async requestUpdateAction(deviceId: string, action: string): Promise<unknown> {
    return this.request("POST", `/devices/${deviceId}/updates/${action}`);
  }

  async listPlugins(deviceId: string): Promise<ListPluginsResponse> {
    return this.request<ListPluginsResponse>("GET", `/devices/${deviceId}/plugins`);
  }

  async requestPluginAction(deviceId: string, name: string, action: string): Promise<unknown> {
    return this.request("POST", `/devices/${deviceId}/plugins/${encodeURIComponent(name)}/${action}`);
  }

  async chat(message: string, conversationId?: string): Promise<ChatResponse> {
    return this.request<ChatResponse>("POST", "/ai/chat", {
      conversation_id: conversationId ?? "",
      message,
    });
  }

  async streamChat(message: string, conversationId?: string): Promise<ReadableStream<Uint8Array>> {
    const response = await fetch(`${API_BASE}/ai/stream-chat`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        ...(this.accessToken ? { Authorization: `Bearer ${this.accessToken}` } : {}),
      },
      body: JSON.stringify({
        conversation_id: conversationId ?? "",
        message,
      }),
    });
    if (!response.ok) {
      throw new Error(`Stream request failed: ${response.status}`);
    }
    return response.body!;
  }

  async listConversations(): Promise<ListConversationsResponse> {
    return this.request<ListConversationsResponse>("GET", "/ai/conversations");
  }

  async getConversation(id: string): Promise<ConversationDetail> {
    return this.request<ConversationDetail>("GET", `/ai/conversations/${id}`);
  }

  async deleteConversation(id: string): Promise<{ message: string }> {
    return this.request<{ message: string }>("DELETE", `/ai/conversations/${id}`);
  }

  async searchConversations(query: string): Promise<ListConversationsResponse> {
    return this.request<ListConversationsResponse>("GET", `/ai/conversations/search?q=${encodeURIComponent(query)}`);
  }

  async getAISettings(): Promise<AISettingsResponse> {
    return this.request<AISettingsResponse>("GET", "/ai/settings");
  }

  async requestFileAction(deviceId: string, actionType: string, path: string, query?: string, targetPath?: string, metadata?: Record<string, unknown>): Promise<FileActionResponse> {
    return this.request<FileActionResponse>("POST", `/devices/${deviceId}/files/${actionType}`, { path, query, target_path: targetPath, metadata });
  }

  async getFileAction(actionId: string): Promise<FileActionResponse> {
    return this.request<FileActionResponse>("GET", `/devices/file-actions/${actionId}`);
  }

  async listFileActions(deviceId: string): Promise<{ file_actions: import("@/lib/types").FileAction[] }> {
    return this.request("GET", `/devices/${deviceId}/files`);
  }

  // --- File Transfers (chunked upload/download) ---

  async createFileTransfer(deviceId: string, data: {
    transfer_type: "upload" | "download";
    filename: string;
    remote_path: string;
    total_size: number;
    chunk_size?: number;
  }): Promise<FileTransferResponse> {
    return this.request<FileTransferResponse>("POST", `/devices/${deviceId}/file-transfers`, data);
  }

  async listFileTransfers(deviceId: string, limit?: number, offset?: number): Promise<ListFileTransfersResponse> {
    const params = new URLSearchParams();
    if (limit) params.set("limit", String(limit));
    if (offset) params.set("offset", String(offset));
    const qs = params.toString();
    return this.request<ListFileTransfersResponse>("GET", `/devices/${deviceId}/file-transfers${qs ? `?${qs}` : ""}`);
  }

  async getFileTransfer(transferId: string): Promise<FileTransferResponse> {
    return this.request<FileTransferResponse>("GET", `/devices/file-transfers/${transferId}`);
  }

  async uploadChunk(transferId: string, chunkNumber: number, data: ArrayBuffer): Promise<{ chunk_number: number; checksum: string; uploaded_chunks: number; total_chunks: number }> {
    const headers: Record<string, string> = {};
    if (this.accessToken) {
      headers["Authorization"] = `Bearer ${this.accessToken}`;
    }
    const res = await fetch(`${API_BASE}/devices/file-transfers/${transferId}/chunks/${chunkNumber}`, {
      method: "POST",
      headers: {
        ...headers,
        "Content-Type": "application/octet-stream",
      },
      body: data,
    });
    if (!res.ok) {
      throw new ApiRequestError(`Chunk upload failed: ${res.status}`, "CHUNK_UPLOAD_FAILED", res.status);
    }
    return res.json();
  }

  async downloadChunk(transferId: string, chunkNumber: number): Promise<ArrayBuffer> {
    const headers: Record<string, string> = {};
    if (this.accessToken) {
      headers["Authorization"] = `Bearer ${this.accessToken}`;
    }
    const res = await fetch(`${API_BASE}/devices/file-transfers/${transferId}/chunks/${chunkNumber}`, {
      method: "GET",
      headers,
    });
    if (!res.ok) {
      throw new ApiRequestError(`Chunk download failed: ${res.status}`, "CHUNK_DOWNLOAD_FAILED", res.status);
    }
    return res.arrayBuffer();
  }

  async assembleTransfer(transferId: string): Promise<FileTransferResponse> {
    return this.request<FileTransferResponse>("POST", `/devices/file-transfers/${transferId}/assemble`);
  }

  async cancelTransfer(transferId: string): Promise<FileTransferResponse> {
    return this.request<FileTransferResponse>("POST", `/devices/file-transfers/${transferId}/cancel`);
  }

  // --- File Permissions (RBAC) ---

  async upsertFilePermission(deviceId: string, data: {
    user_id: string;
    can_browse?: boolean;
    can_modify?: boolean;
    can_upload?: boolean;
    can_download?: boolean;
    can_delete?: boolean;
    allowed_roots?: string[];
  }): Promise<FilePermissionResponse> {
    return this.request<FilePermissionResponse>("POST", `/devices/${deviceId}/file-permissions`, data);
  }

  async listFilePermissions(deviceId: string): Promise<ListFilePermissionsResponse> {
    return this.request<ListFilePermissionsResponse>("GET", `/devices/${deviceId}/file-permissions`);
  }

  async getFilePermissionForUser(deviceId: string, userId: string): Promise<FilePermissionResponse> {
    return this.request<FilePermissionResponse>("GET", `/devices/${deviceId}/file-permissions/${userId}`);
  }

  async deleteFilePermission(deviceId: string, userId: string): Promise<void> {
    return this.request("DELETE", `/devices/${deviceId}/file-permissions/${userId}`);
  }

  // --- Process Management ---

  async listProcesses(deviceId: string): Promise<ListProcessesResponse> {
    return this.request<ListProcessesResponse>("GET", `/devices/${deviceId}/processes`);
  }

  async searchProcesses(deviceId: string, query: string): Promise<ListProcessesResponse> {
    return this.request<ListProcessesResponse>("GET", `/devices/${deviceId}/processes/search?q=${encodeURIComponent(query)}`);
  }

  async getProcessHistory(deviceId: string, minutes?: number): Promise<ProcessHistoryResponse> {
    const params = new URLSearchParams();
    if (minutes) params.set("minutes", String(minutes));
    const qs = params.toString();
    return this.request<ProcessHistoryResponse>("GET", `/devices/${deviceId}/processes/history${qs ? `?${qs}` : ""}`);
  }

  async getProcessAggregates(deviceId: string): Promise<ProcessAggregatesResponse> {
    return this.request<ProcessAggregatesResponse>("GET", `/devices/${deviceId}/processes/aggregates`);
  }

  async createProcessAction(deviceId: string, data: { pid: number; action_type: string }): Promise<ProcessActionResponse> {
    return this.request<ProcessActionResponse>("POST", `/devices/${deviceId}/processes/actions`, data);
  }

  async listProcessActions(deviceId: string, limit?: number, offset?: number): Promise<ListProcessActionsResponse> {
    const params = new URLSearchParams();
    if (limit) params.set("limit", String(limit));
    if (offset) params.set("offset", String(offset));
    const qs = params.toString();
    return this.request<ListProcessActionsResponse>("GET", `/devices/${deviceId}/processes/actions${qs ? `?${qs}` : ""}`);
  }

  // --- Groups & Multi-Device ---

  async listGroups(limit?: number, offset?: number): Promise<{ groups: DeviceGroup[] }> {
    const params = new URLSearchParams();
    if (limit !== undefined) params.set("limit", String(limit));
    if (offset !== undefined) params.set("offset", String(offset));
    return this.request("GET", `/groups?${params}`);
  }

  async createGroup(data: { name: string; description: string }): Promise<{ group: DeviceGroup }> {
    return this.request("POST", "/groups", data);
  }

  async getGroup(id: string): Promise<GroupDetailResponse> {
    return this.request<GroupDetailResponse>("GET", `/groups/${id}`);
  }

  async updateGroup(id: string, data: { name: string; description: string }): Promise<{ group: DeviceGroup }> {
    return this.request("PUT", `/groups/${id}`, data);
  }

  async deleteGroup(id: string): Promise<void> {
    return this.request("DELETE", `/groups/${id}`);
  }

  async addDeviceToGroup(groupId: string, deviceId: string): Promise<void> {
    return this.request("POST", `/groups/${groupId}/devices`, { device_id: deviceId });
  }

  async removeDeviceFromGroup(groupId: string, deviceId: string): Promise<void> {
    return this.request("DELETE", `/groups/${groupId}/devices/${deviceId}`);
  }

  async runGroupCommand(groupId: string, command: string): Promise<GroupCommandResponse> {
    return this.request<GroupCommandResponse>("POST", `/groups/${groupId}/commands`, { command });
  }

  async runMultiDeviceCommand(deviceIds: string[], command: string): Promise<GroupCommandResponse> {
    return this.request<GroupCommandResponse>("POST", "/multi-device/commands", { device_ids: deviceIds, command });
  }

  async getGroupAction(actionId: string): Promise<{ group_action: GroupAction }> {
    return this.request("GET", `/group-actions/${actionId}`);
  }

  // --- Jobs ---

  async listJobs(limit?: number, offset?: number): Promise<{ jobs: Job[] }> {
    const params = new URLSearchParams();
    if (limit !== undefined) params.set("limit", String(limit));
    if (offset !== undefined) params.set("offset", String(offset));
    return this.request("GET", `/jobs?${params}`);
  }

  async getJob(id: string): Promise<{ job: Job }> {
    return this.request("GET", `/jobs/${id}`);
  }

  async cancelJob(id: string): Promise<void> {
    return this.request("POST", `/jobs/${id}/cancel`);
  }

  async retryJob(id: string): Promise<void> {
    return this.request("POST", `/jobs/${id}/retry`);
  }

  // --- Schedules ---

  async listSchedules(limit?: number, offset?: number): Promise<{ schedules: Schedule[] }> {
    const params = new URLSearchParams();
    if (limit !== undefined) params.set("limit", String(limit));
    if (offset !== undefined) params.set("offset", String(offset));
    return this.request("GET", `/schedules?${params}`);
  }

  async getSchedule(id: string): Promise<{ schedule: Schedule }> {
    return this.request("GET", `/schedules/${id}`);
  }

  async createSchedule(data: CreateSchedulePayload): Promise<{ schedule: Schedule }> {
    return this.request("POST", "/schedules", data);
  }

  async updateSchedule(id: string, data: CreateSchedulePayload): Promise<{ schedule: Schedule }> {
    return this.request("PUT", `/schedules/${id}`, data);
  }

  async deleteSchedule(id: string): Promise<void> {
    return this.request("DELETE", `/schedules/${id}`);
  }

  async enableSchedule(id: string): Promise<void> {
    return this.request("POST", `/schedules/${id}/enable`);
  }

  async disableSchedule(id: string): Promise<void> {
    return this.request("POST", `/schedules/${id}/disable`);
  }

  async listScheduleRuns(id: string): Promise<{ runs: ScheduleRun[] }> {
    return this.request("GET", `/schedules/${id}/runs`);
  }

  // --- Policies ---

  async listPolicies(): Promise<{ policies: Policy[] }> {
    return this.request("GET", "/policies");
  }

  async getPolicy(id: string): Promise<{ policy: Policy }> {
    return this.request("GET", `/policies/${id}`);
  }

  async createPolicy(data: CreatePolicyPayload): Promise<{ policy: Policy }> {
    return this.request("POST", "/policies", data);
  }

  async updatePolicy(id: string, data: CreatePolicyPayload): Promise<{ policy: Policy }> {
    return this.request("PUT", `/policies/${id}`, data);
  }

  async deletePolicy(id: string): Promise<void> {
    return this.request("DELETE", `/policies/${id}`);
  }

  async enablePolicy(id: string): Promise<void> {
    return this.request("POST", `/policies/${id}/enable`);
  }

  async disablePolicy(id: string): Promise<void> {
    return this.request("POST", `/policies/${id}/disable`);
  }

  // --- Users / Roles ---

  async listUsers(limit?: number, offset?: number): Promise<{ users: UserListItem[] }> {
    const params = new URLSearchParams();
    if (limit !== undefined) params.set("limit", String(limit));
    if (offset !== undefined) params.set("offset", String(offset));
    return this.request("GET", `/users?${params}`);
  }

  async updateUserRole(id: string, role: string): Promise<void> {
    return this.request("PUT", `/users/${id}/role`, { role });
  }

  // --- Observability ---

  async getMetrics(): Promise<Metrics> {
    return this.request("GET", "/metrics");
  }

  async getJobStats(): Promise<JobStats> {
    return this.request("GET", "/jobs/stats");
  }

  async listLogs(params?: { level?: string; source?: string; since?: string; until?: string; offset?: number; limit?: number }): Promise<SystemLogListResponse> {
    const search = new URLSearchParams();
    if (params?.level) search.set("level", params.level);
    if (params?.source) search.set("source", params.source);
    if (params?.since) search.set("since", params.since);
    if (params?.until) search.set("until", params.until);
    if (params?.offset !== undefined) search.set("offset", String(params.offset));
    if (params?.limit) search.set("limit", String(params.limit));
    const qs = search.toString();
    return this.request("GET", `/logs${qs ? `?${qs}` : ""}`);
  }

  async getVersion(): Promise<{ version: string; commit: string; build_time: string }> {
    return this.request("GET", "/version");
  }

  // --- Proxmox ---

  async listProxmoxConnections(deviceId: string): Promise<ListProxmoxConnectionsResponse> {
    return this.request("GET", `/devices/${deviceId}/proxmox/connections`);
  }

  async createProxmoxConnection(deviceId: string, conn: {
    name: string; host: string; port?: number; auth_type: string;
    token_id?: string; token_secret?: string; username?: string; password?: string;
    verify_ssl?: boolean;
  }): Promise<{ connection: import("@/lib/types").ProxmoxConnection }> {
    return this.request("POST", `/devices/${deviceId}/proxmox/connections`, conn);
  }

  async deleteProxmoxConnection(deviceId: string, connectionId: string): Promise<void> {
    return this.request("DELETE", `/devices/${deviceId}/proxmox/connections/${connectionId}`);
  }

  async listProxmoxNodes(deviceId: string, connectionId?: string): Promise<ListProxmoxNodesResponse> {
    const qs = connectionId ? `?connection_id=${connectionId}` : "";
    return this.request("GET", `/devices/${deviceId}/proxmox/nodes${qs}`);
  }

  async listProxmoxVMs(deviceId: string, connectionId?: string): Promise<ListProxmoxVMsResponse> {
    const qs = connectionId ? `?connection_id=${connectionId}` : "";
    return this.request("GET", `/devices/${deviceId}/proxmox/vms${qs}`);
  }

  async listProxmoxLXCs(deviceId: string, connectionId?: string): Promise<ListProxmoxLXCsResponse> {
    const qs = connectionId ? `?connection_id=${connectionId}` : "";
    return this.request("GET", `/devices/${deviceId}/proxmox/lxcs${qs}`);
  }

  async listProxmoxStorage(deviceId: string, connectionId?: string): Promise<ListProxmoxStorageResponse> {
    const qs = connectionId ? `?connection_id=${connectionId}` : "";
    return this.request("GET", `/devices/${deviceId}/proxmox/storage${qs}`);
  }

  async listProxmoxNetworks(deviceId: string, connectionId?: string): Promise<ListProxmoxNetworksResponse> {
    const qs = connectionId ? `?connection_id=${connectionId}` : "";
    return this.request("GET", `/devices/${deviceId}/proxmox/networks${qs}`);
  }

  async listProxmoxSnapshots(deviceId: string, connectionId?: string): Promise<ListProxmoxSnapshotsResponse> {
    const qs = connectionId ? `?connection_id=${connectionId}` : "";
    return this.request("GET", `/devices/${deviceId}/proxmox/snapshots${qs}`);
  }

  async listProxmoxTasks(deviceId: string, connectionId?: string, limit?: number): Promise<ListProxmoxTasksResponse> {
    const params = new URLSearchParams();
    if (connectionId) params.set("connection_id", connectionId);
    if (limit) params.set("limit", String(limit));
    const qs = params.toString();
    return this.request("GET", `/devices/${deviceId}/proxmox/tasks${qs ? `?${qs}` : ""}`);
  }

  async proxmoxPowerAction(deviceId: string, payload: {
    node_name: string; vm_id: number; action: string; type: string;
  }): Promise<{ status: string }> {
    return this.request("POST", `/devices/${deviceId}/proxmox/power`, payload);
  }

  // --- Terminal ---

  async listTerminalSessions(deviceId: string, limit?: number): Promise<import("@/lib/types").ListTerminalSessionsResponse> {
    const qs = limit ? `?limit=${limit}` : "";
    return this.request("GET", `/devices/${deviceId}/terminal/sessions${qs}`);
  }

  async listMyTerminalSessions(limit?: number): Promise<import("@/lib/types").ListTerminalSessionsResponse> {
    const qs = limit ? `?limit=${limit}` : "";
    return this.request("GET", `/terminal/sessions${qs}`);
  }

  async getTerminalSession(deviceId: string, sessionId: string): Promise<import("@/lib/types").TerminalSessionDetailResponse> {
    return this.request("GET", `/devices/${deviceId}/terminal/sessions/${sessionId}`);
  }

  async getTerminalRecording(deviceId: string, sessionId: string): Promise<import("@/lib/types").TerminalRecordingResponse> {
    return this.request("GET", `/devices/${deviceId}/terminal/sessions/${sessionId}/recording`);
  }

  // --- Plugin Platform (Admin) ---

  async listInstalledPlugins(): Promise<ListInstalledPluginsResponse> {
    return this.request("GET", `/plugin-platform/`);
  }

  async getInstalledPlugin(name: string): Promise<InstalledPluginResponse> {
    return this.request("GET", `/plugin-platform/${encodeURIComponent(name)}`);
  }

  async enablePlugin(name: string): Promise<{ message: string; name: string }> {
    return this.request("POST", `/plugin-platform/${encodeURIComponent(name)}/enable`);
  }

  async disablePlugin(name: string): Promise<{ message: string; name: string }> {
    return this.request("POST", `/plugin-platform/${encodeURIComponent(name)}/disable`);
  }

  async removePlugin(name: string): Promise<{ message: string; name: string }> {
    return this.request("DELETE", `/plugin-platform/${encodeURIComponent(name)}`);
  }

  async updatePluginConfig(name: string, config: Record<string, unknown>): Promise<{ message: string; name: string }> {
    return this.request("PUT", `/plugin-platform/${encodeURIComponent(name)}/config`, { config });
  }

  async getPluginRoutes(name: string): Promise<PluginRoutesResponse> {
    return this.request("GET", `/plugin-platform/${encodeURIComponent(name)}/routes`);
  }

  async getPluginPages(name: string): Promise<PluginPagesResponse> {
    return this.request("GET", `/plugin-platform/${encodeURIComponent(name)}/pages`);
  }

  async getPluginWorkers(name: string): Promise<PluginWorkersResponse> {
    return this.request("GET", `/plugin-platform/${encodeURIComponent(name)}/workers`);
  }

  async getPluginSchedules(name: string): Promise<PluginSchedulesResponse> {
    return this.request("GET", `/plugin-platform/${encodeURIComponent(name)}/schedules`);
  }

  async getAllPluginSidebar(): Promise<PluginSidebarResponse> {
    return this.request("GET", `/plugin-platform/sidebar/all`);
  }

  async getAllPluginRoutes(): Promise<PluginRoutesResponse> {
    return this.request("GET", `/plugin-platform/routes/all`);
  }

  async getAllPluginPages(): Promise<PluginPagesResponse> {
    return this.request("GET", `/plugin-platform/pages/all`);
  }

  // --- Fleet Management ---

  async getFleetDashboard(): Promise<FleetDashboardResponse> {
    return this.request("GET", `/fleet/dashboard`);
  }

  async getDeviceTags(deviceId: string): Promise<DeviceTagsResponse> {
    return this.request("GET", `/tags/devices/${encodeURIComponent(deviceId)}`);
  }

  async setDeviceTag(deviceId: string, key: string, value: string): Promise<{ status: string }> {
    return this.request("PUT", `/tags/devices/${encodeURIComponent(deviceId)}`, { key, value });
  }

  async removeDeviceTag(deviceId: string, tagKey: string): Promise<{ status: string }> {
    return this.request("DELETE", `/tags/devices/${encodeURIComponent(deviceId)}/${encodeURIComponent(tagKey)}`);
  }

  async getTagKeys(): Promise<TagKeysResponse> {
    return this.request("GET", `/tags/keys`);
  }

  async findDevicesByTag(key: string, value?: string): Promise<DeviceTagSearchResponse> {
    const params = new URLSearchParams({ key });
    if (value) params.set("value", value);
    return this.request("GET", `/tags/search?${params.toString()}`);
  }

  async getDeviceInventory(deviceId: string): Promise<DeviceInventoryResponse> {
    return this.request("GET", `/inventory/devices/${encodeURIComponent(deviceId)}`);
  }

  async upsertDeviceInventory(deviceId: string, inventory: Partial<DeviceInventory>): Promise<{ status: string }> {
    return this.request("PUT", `/inventory/devices/${encodeURIComponent(deviceId)}`, inventory);
  }

  async listDeviceInventory(): Promise<DeviceInventoryListResponse> {
    return this.request("GET", `/inventory`);
  }

  async getComplianceChecks(): Promise<ComplianceChecksResponse> {
    return this.request("GET", `/compliance/checks`);
  }

  async createComplianceCheck(check: Partial<ComplianceCheck>): Promise<ComplianceCheck> {
    return this.request("POST", `/compliance/checks`, check);
  }

  async getComplianceResults(deviceId?: string): Promise<ComplianceResultsResponse> {
    const params = deviceId ? `?device_id=${encodeURIComponent(deviceId)}` : "";
    return this.request("GET", `/compliance/results${params}`);
  }

  async upsertComplianceResult(result: Partial<ComplianceResult>): Promise<{ status: string }> {
    return this.request("POST", `/compliance/results`, result);
  }

  async getComplianceSummary(): Promise<ComplianceSummaryResponse> {
    return this.request("GET", `/compliance/summary`);
  }

  async createBulkOperation(op: Partial<BulkOperation>): Promise<BulkOperation> {
    return this.request("POST", `/bulk`, op);
  }

  async listBulkOperations(): Promise<BulkOperationsResponse> {
    return this.request("GET", `/bulk`);
  }

  async getBulkOperation(id: string): Promise<BulkOperation> {
    return this.request("GET", `/bulk/${encodeURIComponent(id)}`);
  }

  async cancelBulkOperation(id: string): Promise<{ status: string }> {
    return this.request("POST", `/bulk/${encodeURIComponent(id)}/cancel`);
  }

  // --- Automation Engine ---

  async getWorkflows(): Promise<WorkflowsResponse> {
    return this.request("GET", `/workflows`);
  }

  async getWorkflow(id: string): Promise<WorkflowsResponse> {
    return this.request("GET", `/workflows/${encodeURIComponent(id)}`);
  }

  async createWorkflow(wf: Partial<Workflow>): Promise<Workflow> {
    return this.request("POST", `/workflows`, wf);
  }

  async updateWorkflow(id: string, wf: Partial<Workflow>): Promise<Workflow> {
    return this.request("PUT", `/workflows/${encodeURIComponent(id)}`, wf);
  }

  async deleteWorkflow(id: string): Promise<{ status: string }> {
    return this.request("DELETE", `/workflows/${encodeURIComponent(id)}`);
  }

  async toggleWorkflow(id: string, enabled: boolean): Promise<{ enabled: boolean }> {
    return this.request("POST", `/workflows/${encodeURIComponent(id)}/toggle`, { enabled });
  }

  async executeWorkflow(id: string, triggerData?: Record<string, unknown>): Promise<WorkflowRun> {
    return this.request("POST", `/workflows/${encodeURIComponent(id)}/execute`, { trigger_data: triggerData ?? {} });
  }

  async getWorkflowSteps(id: string): Promise<WorkflowStepsResponse> {
    return this.request("GET", `/workflows/${encodeURIComponent(id)}/steps`);
  }

  async createWorkflowStep(id: string, step: Partial<WorkflowStep>): Promise<WorkflowStep> {
    return this.request("POST", `/workflows/${encodeURIComponent(id)}/steps`, step);
  }

  async getWorkflowRuns(id: string): Promise<WorkflowRunsResponse> {
    return this.request("GET", `/workflows/${encodeURIComponent(id)}/runs`);
  }

  async getWorkflowRun(workflowId: string, runId: string): Promise<WorkflowRunDetailResponse> {
    return this.request("GET", `/workflows/${encodeURIComponent(workflowId)}/runs/${encodeURIComponent(runId)}`);
  }

  async getAllWorkflowRuns(): Promise<WorkflowRunsResponse> {
    return this.request("GET", `/workflows/runs`);
  }

  async getWorkflowTemplates(): Promise<WorkflowTemplatesResponse> {
    return this.request("GET", `/workflows/templates`);
  }

  async getWorkflowTemplate(id: string): Promise<WorkflowTemplatesResponse> {
    return this.request("GET", `/workflows/templates/${encodeURIComponent(id)}`);
  }

  async createWorkflowFromTemplate(templateId: string): Promise<Workflow> {
    return this.request("POST", `/workflows/templates/${encodeURIComponent(templateId)}/create`);
  }

  // --- Monitoring Platform ---

  async recordMetric(metric: Partial<MonitoringMetric>): Promise<{ status: string }> {
    return this.request("POST", `/monitoring/metrics`, metric);
  }

  async queryMetrics(name: string, deviceId?: string, since?: string): Promise<MonitoringMetricsResponse> {
    const params = new URLSearchParams({ name });
    if (deviceId) params.set("device_id", deviceId);
    if (since) params.set("since", since);
    return this.request("GET", `/monitoring/metrics?${params.toString()}`);
  }

  async getLatestDeviceMetrics(deviceId: string): Promise<MonitoringMetricsResponse> {
    return this.request("GET", `/monitoring/metrics/device/${encodeURIComponent(deviceId)}`);
  }

  async getAlertRules(): Promise<AlertRulesResponse> {
    return this.request("GET", `/alerts/rules`);
  }

  async createAlertRule(rule: Partial<AlertRule>): Promise<AlertRule> {
    return this.request("POST", `/alerts/rules`, rule);
  }

  async updateAlertRule(id: string, rule: Partial<AlertRule>): Promise<AlertRule> {
    return this.request("PUT", `/alerts/rules/${encodeURIComponent(id)}`, rule);
  }

  async deleteAlertRule(id: string): Promise<{ status: string }> {
    return this.request("DELETE", `/alerts/rules/${encodeURIComponent(id)}`);
  }

  async getActiveAlerts(): Promise<AlertInstancesResponse> {
    return this.request("GET", `/alerts/active`);
  }

  async getRecentAlerts(): Promise<AlertInstancesResponse> {
    return this.request("GET", `/alerts/recent`);
  }

  async getAlertSummary(): Promise<AlertSummaryResponse> {
    return this.request("GET", `/alerts/summary`);
  }

  async acknowledgeAlert(id: string): Promise<{ status: string }> {
    return this.request("POST", `/alerts/${encodeURIComponent(id)}/acknowledge`);
  }

  async resolveAlert(id: string): Promise<{ status: string }> {
    return this.request("POST", `/alerts/${encodeURIComponent(id)}/resolve`);
  }

  async getMonitoringDashboards(): Promise<MonitoringDashboardsResponse> {
    return this.request("GET", `/dashboards`);
  }

  async getMonitoringDashboard(id: string): Promise<{ data: MonitoringDashboardDetail }> {
    return this.request("GET", `/dashboards/${encodeURIComponent(id)}`);
  }

  async createMonitoringDashboard(dash: Partial<MonitoringDashboard>): Promise<MonitoringDashboard> {
    return this.request("POST", `/dashboards`, dash);
  }

  async createMonitoringWidget(dashId: string, widget: Partial<MonitoringWidget>): Promise<MonitoringWidget> {
    return this.request("POST", `/dashboards/${encodeURIComponent(dashId)}/widgets`, widget);
  }

  async updateMonitoringDashboard(id: string, update: { name?: string; description?: string; is_default?: boolean; layout?: unknown; widgets?: Partial<MonitoringWidget>[] }): Promise<MonitoringDashboard> {
    return this.request("PUT", `/dashboards/${encodeURIComponent(id)}`, update);
  }

  async deleteMonitoringWidget(id: string): Promise<{ status: string }> {
    return this.request("DELETE", `/dashboards/widgets/${encodeURIComponent(id)}`);
  }

  async deleteMonitoringDashboard(id: string): Promise<{ status: string }> {
    return this.request("DELETE", `/dashboards/${encodeURIComponent(id)}`);
  }

  async getMonitoringLogs(level?: string, query?: string): Promise<MonitoringLogsResponse> {
    const params = new URLSearchParams();
    if (level) params.set("level", level);
    if (query) params.set("q", query);
    const qs = params.toString();
    return this.request("GET", `/monitoring/logs${qs ? "?" + qs : ""}`);
  }

  async getMonitoringLogStats(): Promise<MonitoringLogStatsResponse> {
    return this.request("GET", `/monitoring/logs/stats`);
  }

  // --- RenVault ---

  async getVaultSecrets(query?: string): Promise<VaultSecretsResponse> {
    const params = query ? `?q=${encodeURIComponent(query)}` : "";
    return this.request("GET", `/vault/secrets${params}`);
  }

  async getVaultSecret(id: string): Promise<{ data: VaultSecret }> {
    return this.request("GET", `/vault/secrets/${encodeURIComponent(id)}`);
  }

  async createVaultSecret(secret: { name: string; path: string; value: string; description?: string; secret_type?: string; metadata?: Record<string, unknown> }): Promise<VaultSecret> {
    return this.request("POST", `/vault/secrets`, secret);
  }

  async updateVaultSecret(id: string, data: Partial<VaultSecret & { value: string }>): Promise<VaultSecret> {
    return this.request("PUT", `/vault/secrets/${encodeURIComponent(id)}`, data);
  }

  async deleteVaultSecret(id: string): Promise<{ status: string }> {
    return this.request("DELETE", `/vault/secrets/${encodeURIComponent(id)}`);
  }

  async getVaultSecretVersions(id: string): Promise<VaultSecretVersionsResponse> {
    return this.request("GET", `/vault/secrets/${encodeURIComponent(id)}/versions`);
  }

  async getVaultKeys(): Promise<VaultKeysResponse> {
    return this.request("GET", `/vault/keys`);
  }

  async toggleVaultKey(id: string, enabled: boolean): Promise<{ enabled: boolean }> {
    return this.request("POST", `/vault/keys/${encodeURIComponent(id)}/toggle`, { enabled });
  }

  async getVaultPolicies(): Promise<VaultPoliciesResponse> {
    return this.request("GET", `/vault/policies`);
  }

  async getVaultAuditLog(secretId?: string): Promise<VaultAuditLogResponse> {
    const params = secretId ? `?secret_id=${encodeURIComponent(secretId)}` : "";
    return this.request("GET", `/vault/audit${params}`);
  }

  // --- Notifications ---

  async getNotificationProviders(): Promise<NotificationProvidersResponse> {
    return this.request("GET", `/notifications/providers`);
  }

  async createNotificationProvider(provider: Partial<NotificationProvider>): Promise<NotificationProvider> {
    return this.request("POST", `/notifications/providers`, provider);
  }

  async toggleNotificationProvider(id: string, enabled: boolean): Promise<{ enabled: boolean }> {
    return this.request("POST", `/notifications/providers/${encodeURIComponent(id)}/toggle`, { enabled });
  }

  async deleteNotificationProvider(id: string): Promise<{ status: string }> {
    return this.request("DELETE", `/notifications/providers/${encodeURIComponent(id)}`);
  }

  async getNotificationTemplates(): Promise<NotificationTemplatesResponse> {
    return this.request("GET", `/notifications/templates`);
  }

  async createNotificationTemplate(template: Partial<NotificationTemplate>): Promise<NotificationTemplate> {
    return this.request("POST", `/notifications/templates`, template);
  }

  async deleteNotificationTemplate(id: string): Promise<{ status: string }> {
    return this.request("DELETE", `/notifications/templates/${encodeURIComponent(id)}`);
  }

  async getNotificationRules(): Promise<NotificationRulesResponse> {
    return this.request("GET", `/notifications/rules`);
  }

  async createNotificationRule(rule: Partial<NotificationRule>): Promise<NotificationRule> {
    return this.request("POST", `/notifications/rules`, rule);
  }

  async deleteNotificationRule(id: string): Promise<{ status: string }> {
    return this.request("DELETE", `/notifications/rules/${encodeURIComponent(id)}`);
  }

  async getNotificationHistory(): Promise<NotificationHistoryResponse> {
    return this.request("GET", `/notifications/history`);
  }

  async getNotificationStats(): Promise<NotificationStatsResponse> {
    return this.request("GET", `/notifications/stats`);
  }

  // --- Marketplace ---

  async getMarketplacePlugins(query?: string): Promise<MarketplacePluginsResponse> {
    const params = query ? `?q=${encodeURIComponent(query)}` : "";
    return this.request("GET", `/marketplace${params}`);
  }

  async getMarketplacePlugin(name: string): Promise<{ data: MarketplacePlugin }> {
    return this.request("GET", `/marketplace/${encodeURIComponent(name)}`);
  }

  async getMarketplaceInstallations(): Promise<MarketplaceInstallationsResponse> {
    return this.request("GET", `/marketplace/installed`);
  }

  async installMarketplacePlugin(name: string): Promise<MarketplaceInstallation> {
    return this.request("POST", `/marketplace/${encodeURIComponent(name)}/install`);
  }

  async upgradeMarketplacePlugin(name: string): Promise<MarketplaceInstallation> {
    return this.request("POST", `/marketplace/${encodeURIComponent(name)}/upgrade`);
  }

  async rollbackMarketplacePlugin(name: string, version: string): Promise<MarketplaceInstallation> {
    return this.request("POST", `/marketplace/${encodeURIComponent(name)}/rollback`, { version });
  }

  async uninstallMarketplacePlugin(name: string): Promise<{ status: string }> {
    return this.request("DELETE", `/marketplace/${encodeURIComponent(name)}`);
  }

  async getMarketplaceInstallHistory(installationId: string): Promise<MarketplaceInstallHistoryResponse> {
    return this.request("GET", `/marketplace/installations/${encodeURIComponent(installationId)}/history`);
  }

  async syncMarketplace(plugins: MarketplacePlugin[]): Promise<{ synced: number }> {
    return this.request("POST", `/marketplace/sync`, { plugins });
  }

  // --- Identity & Access Management ---

  async getIdentityProviders(): Promise<IdentityProvidersResponse> {
    return this.request("GET", `/iam/providers`);
  }

  async createIdentityProvider(provider: Partial<IdentityProvider>): Promise<IdentityProvider> {
    return this.request("POST", `/iam/providers`, provider);
  }

  async deleteIdentityProvider(id: string): Promise<{ status: string }> {
    return this.request("DELETE", `/iam/providers/${encodeURIComponent(id)}`);
  }

  async getOAuth2Apps(): Promise<OAuth2AppsResponse> {
    return this.request("GET", `/iam/oauth2/apps`);
  }

  async createOAuth2App(app: { name: string; redirect_uris?: string[]; scopes?: string[]; description?: string }): Promise<{ app: OAuth2App; client_secret: string }> {
    return this.request("POST", `/iam/oauth2/apps`, app);
  }

  async deleteOAuth2App(id: string): Promise<{ status: string }> {
    return this.request("DELETE", `/iam/oauth2/apps/${encodeURIComponent(id)}`);
  }

  async getAPITokens(): Promise<APITokensResponse> {
    return this.request("GET", `/iam/tokens`);
  }

  async createAPIToken(token: { name: string; scopes?: string[]; expires_at?: string }): Promise<{ token: APIToken; raw_token: string }> {
    return this.request("POST", `/iam/tokens`, token);
  }

  async deleteAPIToken(id: string): Promise<{ status: string }> {
    return this.request("DELETE", `/iam/tokens/${encodeURIComponent(id)}`);
  }

  async getIAMRoles(): Promise<IAMRolesResponse> {
    return this.request("GET", `/iam/roles`);
  }

  async createIAMRole(role: { name: string; display_name?: string; description?: string; permissions?: string[] }): Promise<IAMRole> {
    return this.request("POST", `/iam/roles`, role);
  }

  async deleteIAMRole(id: string): Promise<{ status: string }> {
    return this.request("DELETE", `/iam/roles/${encodeURIComponent(id)}`);
  }

  async getIAMRoleAssignments(): Promise<IAMRoleAssignmentsResponse> {
    return this.request("GET", `/iam/roles/assignments`);
  }

  async createIAMRoleAssignment(assignment: { user_id: string; role_id: string; resource_type?: string; resource_id?: string }): Promise<IAMRoleAssignment> {
    return this.request("POST", `/iam/roles/assignments`, assignment);
  }

  async deleteIAMRoleAssignment(id: string): Promise<{ status: string }> {
    return this.request("DELETE", `/iam/roles/assignments/${encodeURIComponent(id)}`);
  }

  async getApprovalWorkflows(): Promise<ApprovalWorkflowsResponse> {
    return this.request("GET", `/iam/approvals/workflows`);
  }

  async createApprovalWorkflow(wf: Partial<ApprovalWorkflow>): Promise<ApprovalWorkflow> {
    return this.request("POST", `/iam/approvals/workflows`, wf);
  }

  async deleteApprovalWorkflow(id: string): Promise<{ status: string }> {
    return this.request("DELETE", `/iam/approvals/workflows/${encodeURIComponent(id)}`);
  }

  async getApprovalRequests(status?: string): Promise<ApprovalRequestsResponse> {
    const params = status ? `?status=${encodeURIComponent(status)}` : "";
    return this.request("GET", `/iam/approvals/requests${params}`);
  }

  async approveRequest(id: string, comment?: string): Promise<{ status: string }> {
    return this.request("POST", `/iam/approvals/requests/${encodeURIComponent(id)}/approve`, { comment: comment ?? "" });
  }

  async rejectRequest(id: string, comment?: string): Promise<{ status: string }> {
    return this.request("POST", `/iam/approvals/requests/${encodeURIComponent(id)}/reject`, { comment: comment ?? "" });
  }

  async getIAMAuditLog(): Promise<IAMAuditLogResponse> {
    return this.request("GET", `/iam/audit`);
  }

  // --- Deployment Platform ---

  async getPreflightChecks(): Promise<PreflightResponse> {
    return this.request("GET", "/deploy/preflight");
  }

  async getSystemInfo(): Promise<SystemInfoResponse> {
    return this.request("GET", "/deploy/info");
  }

  async startInstallation(opts: {
    install_dir?: string;
    admin_email?: string;
    admin_password?: string;
    external_url?: string;
    http_port?: number;
    log_level?: string;
    hostname?: string;
    non_interactive?: boolean;
  }): Promise<{ status: string }> {
    return this.request("POST", "/deploy/start", opts);
  }

  async getInstallationStatus(): Promise<InstallationStatusResponse> {
    return this.request("GET", "/deploy/status");
  }

  async rollbackInstallation(install_dir?: string): Promise<{ status: string }> {
    return this.request("POST", "/deploy/rollback", { install_dir: install_dir ?? "/opt/renos" });
  }

  // --- Updates ---

  async getVersionInfo(): Promise<VersionInfoResponse> {
    return this.request("GET", `/updates`);
  }

  async checkForUpdates(): Promise<UpdateCheckResponse> {
    return this.request("GET", `/updates/check`);
  }

  async planUpdate(target_version?: string, channel?: string): Promise<UpdatePlanResponse> {
    return this.request("POST", `/updates/plan`, { target_version, channel });
  }

  async startUpdate(target_version?: string, channel?: string): Promise<{ status: string; update_id: string }> {
    return this.request("POST", `/updates/start`, { target_version, channel });
  }

  async rollbackUpdate(backup_id: string): Promise<{ status: string }> {
    return this.request("POST", `/updates/rollback`, { backup_id });
  }

  async getUpdateHistory(): Promise<UpdateHistoryResponse> {
    return this.request("GET", `/updates/history`);
  }

  async getUpdateBackups(): Promise<UpdateBackupsResponse> {
    return this.request("GET", `/updates/backups`);
  }

  async setUpdateChannel(channel: string): Promise<{ channel: string }> {
    return this.request("PUT", `/updates/channel`, { channel });
  }

  // --- Cluster ---

  async getClusterDashboard(): Promise<ClusterDashboardResponse> {
    return this.request("GET", "/cluster/dashboard");
  }

  async listClusterNodes(): Promise<ClusterNodesResponse> {
    return this.request("GET", "/cluster/nodes");
  }

  async registerClusterNode(data: { node_id?: string; hostname: string; address: string; port?: number }): Promise<ClusterNode> {
    return this.request("POST", "/cluster/nodes", data);
  }

  async updateClusterNodeStatus(nodeId: string, status: string): Promise<{ status: string }> {
    return this.request("PUT", `/cluster/nodes/status`, { node_id: nodeId, status });
  }

  async getClusterTasks(status?: string): Promise<ClusterTasksResponse> {
    const params = status ? `?status=${encodeURIComponent(status)}` : "";
    return this.request("GET", `/cluster/tasks${params}`);
  }

  async createClusterTask(data: { task_type: string; task_data?: Record<string, unknown>; priority?: number; max_retries?: number }): Promise<ClusterTask> {
    return this.request("POST", "/cluster/tasks", data);
  }

  async rebalanceClusterTasks(): Promise<{ rebalanced: number }> {
    return this.request("POST", "/cluster/tasks/rebalance");
  }

  async getClusterState(): Promise<ClusterStateResponse> {
    return this.request("GET", "/cluster/state");
  }

  async setClusterState(data: { key: string; value: Record<string, unknown>; owner?: string; ttl_seconds?: number }): Promise<ClusterState> {
    return this.request("POST", "/cluster/state", data);
  }

  async getClusterElections(): Promise<ClusterElectionsResponse> {
    return this.request("GET", "/cluster/elections");
  }

  async recordClusterMetrics(data: { node_id: string; cpu_usage?: number; memory_usage?: number; disk_usage?: number }): Promise<ClusterMetric> {
    return this.request("POST", "/cluster/metrics", data);
  }

  // --- Mobile Platform ---

  async registerPushToken(data: { platform: string; token: string; device_name?: string; device_model?: string; os_version?: string; app_version?: string }): Promise<{ status: string }> {
    return this.request("POST", "/mobile/push/register", data);
  }

  async listPushRegistrations(): Promise<PushRegistrationsResponse> {
    return this.request("GET", "/mobile/push");
  }

  async deletePushRegistration(id: string): Promise<{ status: string }> {
    return this.request("DELETE", `/mobile/push?id=${encodeURIComponent(id)}`);
  }

  async getMobileDashboard(): Promise<{ data: MobileDashboard }> {
    return this.request("GET", "/mobile/dashboard");
  }

  async getMobileDevices(): Promise<ListDevicesResponse> {
    return this.request("GET", "/mobile/devices");
  }

  async getMobileAlertsSummary(): Promise<{ data: { active_alerts: unknown[]; count: number } }> {
    return this.request("GET", "/mobile/alerts/summary");
  }

  async listOfflineSyncItems(deviceId: string): Promise<OfflineSyncResponse> {
    return this.request("GET", `/mobile/sync?device_id=${encodeURIComponent(deviceId)}`);
  }

  async enqueueOfflineSyncItem(data: { device_id: string; action: string; payload?: Record<string, unknown>; priority?: number }): Promise<OfflineSyncItem> {
    return this.request("POST", "/mobile/sync", data);
  }

  async completeOfflineSyncItem(itemId: string, error?: string): Promise<{ status: string }> {
    return this.request("PUT", "/mobile/sync", { item_id: itemId, error: error ?? "" });
  }

  async listDeviceApprovals(status?: string): Promise<DeviceApprovalsResponse> {
    const params = status ? `?status=${encodeURIComponent(status)}` : "";
    return this.request("GET", `/mobile/approvals${params}`);
  }

  async requestDeviceApproval(data: { device_id: string; device_name?: string; device_model?: string; ip_address?: string; fingerprint?: string }): Promise<DeviceApproval> {
    return this.request("POST", "/mobile/approvals", data);
  }

  async approveDevice(id: string): Promise<{ status: string }> {
    return this.request("POST", `/mobile/approvals/approve?id=${encodeURIComponent(id)}`);
  }

  async rejectDevice(id: string, reason?: string): Promise<{ status: string }> {
    return this.request("POST", `/mobile/approvals/reject?id=${encodeURIComponent(id)}`, { reason: reason ?? "" });
  }

  async listMobileSessions(): Promise<MobileSessionsResponse> {
    return this.request("GET", "/mobile/sessions");
  }

  async deleteMobileSession(id: string): Promise<{ status: string }> {
    return this.request("DELETE", `/mobile/sessions?id=${encodeURIComponent(id)}`);
  }

  async getMobileSummary(type?: string): Promise<{ data: Record<string, unknown> }> {
    const params = type ? `?type=${encodeURIComponent(type)}` : "";
    return this.request("GET", `/mobile/summary${params}`);
  }

  // --- Developer Platform ---

  async getWebhookSubscriptions(): Promise<WebhookSubscriptionsResponse> {
    return this.request("GET", "/developer/webhooks");
  }

  async createWebhookSubscription(data: { url: string; events: string[] }): Promise<{ subscription: WebhookSubscription; secret: string }> {
    return this.request("POST", "/developer/webhooks", data);
  }

  async deleteWebhookSubscription(id: string): Promise<{ status: string }> {
    return this.request("DELETE", `/developer/webhooks?id=${encodeURIComponent(id)}`);
  }

  async toggleWebhookSubscription(id: string, active: boolean): Promise<{ active: boolean }> {
    return this.request("POST", `/developer/webhooks/toggle?id=${encodeURIComponent(id)}`, { active });
  }

  async getWebhookEvents(): Promise<WebhookEventsResponse> {
    return this.request("GET", "/developer/webhooks/events");
  }

  async getWebhookDeliveries(subscriptionId: string): Promise<WebhookDeliveriesResponse> {
    return this.request("GET", `/developer/webhooks/deliveries?subscription_id=${encodeURIComponent(subscriptionId)}`);
  }

  async getDeveloperAPIKeys(): Promise<DeveloperAPIKeysResponse> {
    return this.request("GET", "/developer/api-keys");
  }

  async createDeveloperAPIKey(data: { name: string; scopes?: string[] }): Promise<{ api_key: DeveloperAPIKey; raw_key: string }> {
    return this.request("POST", "/developer/api-keys", data);
  }

  async deleteDeveloperAPIKey(id: string): Promise<{ status: string }> {
    return this.request("DELETE", `/developer/api-keys?id=${encodeURIComponent(id)}`);
  }

  async toggleDeveloperAPIKey(id: string, active: boolean): Promise<{ active: boolean }> {
    return this.request("POST", `/developer/api-keys/toggle?id=${encodeURIComponent(id)}`, { active });
  }

  // --- Onboarding ---

  async getOnboardingStatus(): Promise<OnboardingStatus> {
    return this.request<OnboardingStatus>("GET", "/onboarding/status");
  }

  async completeOnboarding(): Promise<OnboardingStatus> {
    return this.request<OnboardingStatus>("POST", "/onboarding/complete");
  }

  // --- Demo mode ---

  async getDemoStatus(): Promise<DemoStatus> {
    return this.request<DemoStatus>("GET", "/demo/status");
  }

  async seedDemo(): Promise<DemoSeedResult> {
    return this.request<DemoSeedResult>("POST", "/demo/seed");
  }

  // --- Health dashboard ---

  async getHealthDashboard(): Promise<HealthDashboard> {
    return this.request<HealthDashboard>("GET", "/health/dashboard");
  }

  // --- Global search ---

  async search(query: string, limit = 10, signal?: AbortSignal): Promise<SearchResponse> {
    return this.request<SearchResponse>(
      "GET",
      `/search?q=${encodeURIComponent(query)}&limit=${limit}`,
      undefined,
      { signal },
    );
  }

  // --- In-app notification center ---

  async getInAppNotifications(params?: {
    unread?: boolean;
    type?: string;
    search?: string;
    limit?: number;
    offset?: number;
  }): Promise<InAppNotification[]> {
    const q = new URLSearchParams();
    if (params?.unread) q.set("unread", "true");
    if (params?.type) q.set("type", params.type);
    if (params?.search) q.set("search", params.search);
    if (params?.limit) q.set("limit", String(params.limit));
    if (params?.offset) q.set("offset", String(params.offset));
    const query = q.toString();
    return this.request<InAppNotification[]>("GET", `/notifications/inapp${query ? `?${query}` : ""}`);
  }

  async getInAppUnreadCount(): Promise<UnreadCountResponse> {
    return this.request<UnreadCountResponse>("GET", "/notifications/inapp/unread_count");
  }

  async markInAppRead(id: string): Promise<{ read: boolean }> {
    return this.request<{ read: boolean }>("POST", `/notifications/inapp/${encodeURIComponent(id)}/read`);
  }

  async markAllInAppRead(): Promise<{ read: boolean }> {
    return this.request<{ read: boolean }>("POST", "/notifications/inapp/read-all");
  }
}

export const api = new ApiClient();
