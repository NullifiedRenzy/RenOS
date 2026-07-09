import type {
  LoginResponse, ListDevicesResponse, DeviceDetailResponse,
  CommandResponse, ListCommandsResponse,
  ListContainersResponse, ContainerActionResponse,
  ListDockerImagesResponse, ListDockerVolumesResponse, ListDockerNetworksResponse,
  ApiErrorBody,
} from "@/lib/types";

const API_BASE = "/api/v1";

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

    return res.json();
  }

  async login(email: string, password: string): Promise<LoginResponse> {
    const data = await this.request<LoginResponse>("POST", "/auth/login", { email, password });
    this.setAccessToken(data.token);
    return data;
  }

  async listDevices(): Promise<ListDevicesResponse> {
    return this.request<ListDevicesResponse>("GET", "/devices");
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
}

export const api = new ApiClient();
