import WebSocket from 'ws';
import fetch, { RequestInit, Response } from 'node-fetch';

const DEFAULT_BASE_URL = 'http://localhost:8443';

export interface RenOSConfig {
  baseURL?: string;
  token?: string;
  apiKey?: string;
}

export class RenOSError extends Error {
  constructor(
    public code: string,
    message: string,
    public details?: unknown,
  ) {
    super(message);
    this.name = 'RenOSError';
  }
}

export interface LoginResponse {
  token: string;
  user_id: string;
  expires_at: string;
}

export interface PaginatedResponse<T> {
  total: number;
  page: number;
  per_page: number;
  data: T[];
}

export interface Device {
  id: string;
  name: string;
  hostname: string;
  os: string;
  arch: string;
  status: string;
  ip: string;
  agent_version: string;
  tags: Record<string, string>;
  enrolled_at: string;
  last_seen_at: string;
  created_at: string;
  updated_at: string;
}

export interface DeviceDetail extends Device {
  cpu_usage: number;
  memory_usage: number;
  disk_usage: number;
  load_average: number;
  uptime: number;
}

export interface Command {
  id: string;
  device_id: string;
  command: string;
  status: string;
  exit_code: number;
  output: string;
  error: string;
  timeout: number;
  created_at: string;
  completed_at: string;
}

export interface Job {
  id: string;
  type: string;
  status: string;
  progress: number;
  total: number;
  completed: number;
  failed: number;
  created_at: string;
  completed_at: string;
  error: string;
}

export interface Workflow {
  id: string;
  name: string;
  description: string;
  steps: WorkflowStep[];
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface WorkflowStep {
  id: string;
  workflow_id: string;
  name: string;
  type: string;
  config: unknown;
  order: number;
  created_at: string;
}

export interface WorkflowRun {
  id: string;
  workflow_id: string;
  status: string;
  started_at: string;
  completed_at: string;
  error: string;
  steps: WorkflowStep[];
}

export interface User {
  id: string;
  email: string;
  display_name: string;
  role: string;
  avatar_url: string;
  created_at: string;
  updated_at: string;
}

export interface Group {
  id: string;
  name: string;
  description: string;
  device_count: number;
  created_at: string;
  updated_at: string;
}

export interface Policy {
  id: string;
  name: string;
  description: string;
  type: string;
  conditions: unknown;
  action: string;
  priority: number;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface Schedule {
  id: string;
  name: string;
  type: string;
  cron_expression: string;
  interval_seconds: number;
  actions: unknown[];
  enabled: boolean;
  last_run: string;
  next_run: string;
  created_at: string;
  updated_at: string;
}

export interface VaultSecret {
  id: string;
  name: string;
  path: string;
  secret_type: string;
  version: number;
  metadata: unknown;
  created_at: string;
  updated_at: string;
}

export interface AlertInstance {
  id: string;
  rule_id: string;
  name: string;
  severity: string;
  status: string;
  metric_value: number;
  starts_at: string;
  acknowledged_at: string;
  resolved_at: string;
}

export interface AlertRule {
  id: string;
  name: string;
  metric: string;
  condition: string;
  threshold: number;
  severity: string;
  enabled: boolean;
  created_at: string;
}

export interface ClusterNode {
  id: string;
  name: string;
  address: string;
  role: string;
  status: string;
  joined_at: string;
  last_heartbeat: string;
}

export interface TerminalSession {
  id: string;
  device_id: string;
  user_id: string;
  status: string;
  created_at: string;
  closed_at: string;
}

export interface Execution {
  id: string;
  device_id: string;
  command: string;
  args: string[];
  status: string;
  exit_code: number;
  output: string;
  error: string;
  workdir: string;
  pid: number;
  started_at: string;
  completed_at: string;
  created_at: string;
}

export interface FileTransfer {
  id: string;
  device_id: string;
  filename: string;
  path: string;
  direction: string;
  size: number;
  transferred: number;
  status: string;
  progress: number;
  error: string;
  created_at: string;
  completed_at: string;
}

export interface StreamEvent {
  type: string;
  data: unknown;
}

export class RenOSClient {
  private baseURL: string;
  private token?: string;
  private apiKey?: string;

  constructor(config?: RenOSConfig) {
    this.baseURL = config?.baseURL ?? DEFAULT_BASE_URL;
    this.token = config?.token;
    this.apiKey = config?.apiKey;
  }

  static async login(
    serverURL: string,
    email: string,
    password: string,
  ): Promise<RenOSClient> {
    const baseURL = serverURL || DEFAULT_BASE_URL;
    const resp = await fetch(`${baseURL}/api/v1/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    });
    if (!resp.ok) {
      const body = await resp.text();
      throw new RenOSError('LOGIN_FAILED', `Login failed (${resp.status}): ${body}`);
    }
    const data = (await resp.json()) as LoginResponse;
    return new RenOSClient({ baseURL, token: data.token });
  }

  setToken(token: string): void {
    this.token = token;
  }

  setAPIKey(key: string): void {
    this.apiKey = key;
  }

  private async request<T>(
    method: string,
    path: string,
    options?: { body?: unknown; params?: Record<string, string | number> },
  ): Promise<T> {
    let url = `${this.baseURL}${path}`;
    if (options?.params) {
      const qs = new URLSearchParams();
      for (const [k, v] of Object.entries(options.params)) {
        qs.set(k, String(v));
      }
      url += `?${qs.toString()}`;
    }

    const init: RequestInit = {
      method,
      headers: {
        'Content-Type': 'application/json',
      },
    };

    if (this.token) {
      (init.headers as Record<string, string>)['Authorization'] = `Bearer ${this.token}`;
    }
    if (this.apiKey) {
      (init.headers as Record<string, string>)['X-API-Key'] = this.apiKey;
    }
    if (options?.body) {
      init.body = JSON.stringify(options.body);
    }

    const resp = await fetch(url, init);
    if (!resp.ok) {
      let code = 'API_ERROR';
      let message = `HTTP ${resp.status}`;
      let details: unknown;
      try {
        const errBody = await resp.json() as { code?: string; message?: string; details?: unknown };
        code = errBody.code ?? code;
        message = errBody.message ?? message;
        details = errBody.details;
      } catch {
        message = await resp.text().catch(() => message);
      }
      throw new RenOSError(code, message, details);
    }
    return resp.json() as Promise<T>;
  }

  listDevices(page = 1, perPage = 50): Promise<PaginatedResponse<Device>> {
    return this.request('GET', '/api/v1/devices', { params: { page, per_page: perPage } });
  }

  getDevice(id: string): Promise<Device> {
    return this.request('GET', `/api/v1/devices/${id}`);
  }

  executeCommand(deviceID: string, command: string): Promise<Command> {
    return this.request('POST', `/api/v1/devices/${deviceID}/commands`, {
      body: { command },
    });
  }

  getCommand(deviceID: string, cmdID: string): Promise<Command> {
    return this.request('GET', `/api/v1/devices/${deviceID}/commands/${cmdID}`);
  }

  listCommands(deviceID: string, page = 1, perPage = 50): Promise<PaginatedResponse<Command>> {
    return this.request('GET', `/api/v1/devices/${deviceID}/commands`, {
      params: { page, per_page: perPage },
    });
  }

  listJobs(page = 1, perPage = 50): Promise<PaginatedResponse<Job>> {
    return this.request('GET', '/api/v1/jobs', { params: { page, per_page: perPage } });
  }

  getJob(id: string): Promise<Job> {
    return this.request('GET', `/api/v1/jobs/${id}`);
  }

  cancelJob(id: string): Promise<void> {
    return this.request('POST', `/api/v1/jobs/${id}/cancel`);
  }

  retryJob(id: string): Promise<void> {
    return this.request('POST', `/api/v1/jobs/${id}/retry`);
  }

  listWorkflows(page = 1, perPage = 50): Promise<PaginatedResponse<Workflow>> {
    return this.request('GET', '/api/v1/workflows', { params: { page, per_page: perPage } });
  }

  getWorkflow(id: string): Promise<Workflow> {
    return this.request('GET', `/api/v1/workflows/${id}`);
  }

  executeWorkflow(id: string): Promise<WorkflowRun> {
    return this.request('POST', `/api/v1/workflows/${id}/execute`);
  }

  listUsers(page = 1, perPage = 50): Promise<PaginatedResponse<User>> {
    return this.request('GET', '/api/v1/users', { params: { page, per_page: perPage } });
  }

  listAlerts(page = 1, perPage = 50): Promise<PaginatedResponse<AlertInstance>> {
    return this.request('GET', '/api/v1/alerts/active', { params: { page, per_page: perPage } });
  }

  listGroups(page = 1, perPage = 50): Promise<PaginatedResponse<Group>> {
    return this.request('GET', '/api/v1/groups', { params: { page, per_page: perPage } });
  }

  getGroup(id: string): Promise<Group> {
    return this.request('GET', `/api/v1/groups/${id}`);
  }

  listSchedules(page = 1, perPage = 50): Promise<PaginatedResponse<Schedule>> {
    return this.request('GET', '/api/v1/schedules', { params: { page, per_page: perPage } });
  }

  listPolicies(page = 1, perPage = 50): Promise<PaginatedResponse<Policy>> {
    return this.request('GET', '/api/v1/policies', { params: { page, per_page: perPage } });
  }

  listVaultSecrets(page = 1, perPage = 50): Promise<PaginatedResponse<VaultSecret>> {
    return this.request('GET', '/api/v1/vault/secrets', { params: { page, per_page: perPage } });
  }

  listClusterNodes(page = 1, perPage = 50): Promise<PaginatedResponse<ClusterNode>> {
    return this.request('GET', '/api/v1/cluster/nodes', { params: { page, per_page: perPage } });
  }

  listTerminalSessions(deviceID: string, page = 1, perPage = 50): Promise<PaginatedResponse<TerminalSession>> {
    return this.request('GET', `/api/v1/devices/${deviceID}/terminal/sessions`, {
      params: { page, per_page: perPage },
    });
  }

  listExecutions(deviceID: string, page = 1, perPage = 50): Promise<PaginatedResponse<Execution>> {
    return this.request('GET', `/api/v1/devices/${deviceID}/executions`, {
      params: { page, per_page: perPage },
    });
  }

  createExecution(deviceID: string, command: string, args?: string[]): Promise<Execution> {
    return this.request('POST', `/api/v1/devices/${deviceID}/executions`, {
      body: { command, args },
    });
  }

  getExecution(deviceID: string, execID: string): Promise<Execution> {
    return this.request('GET', `/api/v1/devices/${deviceID}/executions/${execID}`);
  }

  cancelExecution(deviceID: string, execID: string): Promise<void> {
    return this.request('POST', `/api/v1/devices/${deviceID}/executions/${execID}/cancel`);
  }

  async streamEvents(path: string): Promise<Stream> {
    const url = `${this.baseURL}${path}`;
    const headers: Record<string, string> = {
      Accept: 'text/event-stream',
    };
    if (this.token) headers['Authorization'] = `Bearer ${this.token}`;
    if (this.apiKey) headers['X-API-Key'] = this.apiKey;

    const resp = await fetch(url, { headers });
    if (!resp.ok) {
      throw new RenOSError('STREAM_ERROR', `Stream returned ${resp.status}`);
    }

    const stream = new Stream();
    const reader = resp.body!.getReader();
    const decoder = new TextDecoder();

    const pump = async () => {
      while (true) {
        const { done, value } = await reader.read();
        if (done) {
          stream.close();
          return;
        }
        const text = decoder.decode(value, { stream: true });
        for (const line of text.split('\n')) {
          if (line.startsWith('data: ')) {
            try {
              const data = JSON.parse(line.slice(6));
              stream.emit({ type: 'message', data });
            } catch {
              stream.emit({ type: 'raw', data: line.slice(6) });
            }
          }
        }
      }
    };
    pump().catch((err) => stream.emit({ type: 'error', data: err }));
    return stream;
  }

  connectWebSocket(path: string): RenOSWebSocket {
    let wsURL = `${this.baseURL}${path}`.replace(/^http/, 'ws');
    const ws = new RenOSWebSocket(wsURL, this.token, this.apiKey);
    return ws;
  }
}

export class Stream {
  private listeners: Array<(event: StreamEvent) => void> = [];
  private _closed = false;

  onEvent(cb: (event: StreamEvent) => void): () => void {
    this.listeners.push(cb);
    return () => {
      this.listeners = this.listeners.filter((l) => l !== cb);
    };
  }

  emit(event: StreamEvent): void {
    if (this._closed) return;
    for (const listener of this.listeners) {
      listener(event);
    }
  }

  get closed(): boolean {
    return this._closed;
  }

  close(): void {
    this._closed = true;
    this.listeners = [];
  }
}

export class RenOSWebSocket {
  private ws: WebSocket | null = null;
  private url: string;
  private token?: string;
  private apiKey?: string;
  private listeners: Array<(data: unknown) => void> = [];
  private reconnectEnabled = false;
  private reconnectDelay = 1000;
  private maxReconnectDelay = 30000;
  private shouldClose = false;

  constructor(url: string, token?: string, apiKey?: string) {
    this.url = url;
    this.token = token;
    this.apiKey = apiKey;
    this.connect();
  }

  private connect(): void {
    if (this.shouldClose) return;
    const headers: Record<string, string> = {};
    if (this.token) headers['Authorization'] = `Bearer ${this.token}`;
    if (this.apiKey) headers['X-API-Key'] = this.apiKey;

    this.ws = new WebSocket(this.url, { headers });

    this.ws.on('open', () => {
      this.reconnectDelay = 1000;
    });

    this.ws.on('message', (data: WebSocket.Data) => {
      try {
        const parsed = JSON.parse(data.toString());
        for (const listener of this.listeners) {
          listener(parsed);
        }
      } catch {
        for (const listener of this.listeners) {
          listener(data.toString());
        }
      }
    });

    this.ws.on('close', () => {
      this.ws = null;
      if (this.reconnectEnabled && !this.shouldClose) {
        setTimeout(() => {
          this.reconnectDelay = Math.min(this.reconnectDelay * 2, this.maxReconnectDelay);
          this.connect();
        }, this.reconnectDelay);
      }
    });

    this.ws.on('error', () => {
      this.ws?.close();
    });
  }

  onMessage(cb: (data: unknown) => void): () => void {
    this.listeners.push(cb);
    return () => {
      this.listeners = this.listeners.filter((l) => l !== cb);
    };
  }

  send(data: unknown): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(typeof data === 'string' ? data : JSON.stringify(data));
    }
  }

  enableReconnect(): void {
    this.reconnectEnabled = true;
  }

  disableReconnect(): void {
    this.reconnectEnabled = false;
  }

  close(): void {
    this.shouldClose = true;
    this.ws?.close();
    this.ws = null;
    this.listeners = [];
  }
}
