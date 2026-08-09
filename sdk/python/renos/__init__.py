"""
RenOS Python SDK

Usage:
    from renos import RenOSClient

    # Login
    client = RenOSClient.login("https://renos.example.com", "user@example.com", "password")

    # Or with existing token
    client = RenOSClient(base_url="https://renos.example.com", token="jwt...")

    # List devices
    devices = client.list_devices(page=1, per_page=50)
    for device in devices.data:
        print(device.name, device.status)

    # Stream events
    for event in client.stream_events("/api/v1/deploy/stream"):
        print(event)

    # WebSocket
    ws = client.websocket("/api/v1/ws")
    for message in ws:
        print(message)
"""

from __future__ import annotations

import json
import time
from dataclasses import dataclass, field
from typing import Any, Callable, Generic, Iterator, Optional, TypeVar

import requests
from pydantic import BaseModel

T = TypeVar("T")

DEFAULT_BASE_URL = "http://localhost:8443"


class RenOSException(Exception):
    def __init__(self, code: str, message: str, details: Any = None):
        self.code = code
        self.details = details
        super().__init__(f"[{code}] {message}")


# --- API Models ---


class Device(BaseModel):
    id: str
    name: str
    hostname: str
    os: str
    arch: str
    status: str
    ip: str
    agent_version: str
    tags: dict[str, str] = {}
    enrolled_at: Optional[str] = None
    last_seen_at: Optional[str] = None
    created_at: Optional[str] = None
    updated_at: Optional[str] = None


class DeviceDetail(Device):
    cpu_usage: float = 0.0
    memory_usage: float = 0.0
    disk_usage: float = 0.0
    load_average: float = 0.0
    uptime: int = 0


class Command(BaseModel):
    id: str
    device_id: str
    command: str
    status: str
    exit_code: Optional[int] = None
    output: str = ""
    error: str = ""
    timeout: int = 0
    created_at: Optional[str] = None
    completed_at: Optional[str] = None


class Job(BaseModel):
    id: str
    type: str
    status: str
    progress: float = 0.0
    total: int = 0
    completed: int = 0
    failed: int = 0
    created_at: Optional[str] = None
    completed_at: Optional[str] = None
    error: str = ""


class Workflow(BaseModel):
    id: str
    name: str
    description: str = ""
    steps: list[WorkflowStep] = []
    enabled: bool = True
    created_at: Optional[str] = None
    updated_at: Optional[str] = None


class WorkflowStep(BaseModel):
    id: str
    workflow_id: str
    name: str
    type: str
    config: Any = None
    order: int = 0
    created_at: Optional[str] = None


class WorkflowRun(BaseModel):
    id: str
    workflow_id: str
    status: str
    started_at: Optional[str] = None
    completed_at: Optional[str] = None
    error: str = ""
    steps: list[WorkflowStep] = []


class User(BaseModel):
    id: str
    email: str
    display_name: str = ""
    role: str = "viewer"
    avatar_url: str = ""
    created_at: Optional[str] = None
    updated_at: Optional[str] = None


class Group(BaseModel):
    id: str
    name: str
    description: str = ""
    device_count: int = 0
    created_at: Optional[str] = None
    updated_at: Optional[str] = None


class Policy(BaseModel):
    id: str
    name: str
    description: str = ""
    type: str = ""
    conditions: Any = None
    action: str = ""
    priority: int = 0
    enabled: bool = True
    created_at: Optional[str] = None
    updated_at: Optional[str] = None


class Schedule(BaseModel):
    id: str
    name: str
    type: str = ""
    cron_expression: str = ""
    interval_seconds: int = 0
    actions: list[Any] = []
    enabled: bool = True
    last_run: Optional[str] = None
    next_run: Optional[str] = None
    created_at: Optional[str] = None
    updated_at: Optional[str] = None


class VaultSecret(BaseModel):
    id: str
    name: str
    path: str = ""
    secret_type: str = ""
    version: int = 1
    metadata: Any = None
    created_at: Optional[str] = None
    updated_at: Optional[str] = None


class AlertInstance(BaseModel):
    id: str
    rule_id: str
    name: str
    severity: str
    status: str
    metric_value: float = 0.0
    starts_at: Optional[str] = None
    acknowledged_at: Optional[str] = None
    resolved_at: Optional[str] = None


class AlertRule(BaseModel):
    id: str
    name: str
    metric: str
    condition: str
    threshold: float = 0.0
    severity: str
    enabled: bool = True
    created_at: Optional[str] = None


class ClusterNode(BaseModel):
    id: str
    name: str
    address: str = ""
    role: str = ""
    status: str
    joined_at: Optional[str] = None
    last_heartbeat: Optional[str] = None


class TerminalSession(BaseModel):
    id: str
    device_id: str
    user_id: str
    status: str
    created_at: Optional[str] = None
    closed_at: Optional[str] = None


class Execution(BaseModel):
    id: str
    device_id: str
    command: str
    args: list[str] = []
    status: str
    exit_code: Optional[int] = None
    output: str = ""
    error: str = ""
    workdir: str = ""
    pid: int = 0
    started_at: Optional[str] = None
    completed_at: Optional[str] = None
    created_at: Optional[str] = None


class FileTransfer(BaseModel):
    id: str
    device_id: str
    filename: str
    path: str = ""
    direction: str
    size: int = 0
    transferred: int = 0
    status: str
    progress: float = 0.0
    error: str = ""
    created_at: Optional[str] = None
    completed_at: Optional[str] = None


# --- Paginated Response ---


@dataclass
class PaginatedResponse(Generic[T]):
    total: int
    page: int
    per_page: int
    data: list[T]


# --- Client ---


class RenOSClient:
    def __init__(
        self,
        base_url: str = DEFAULT_BASE_URL,
        token: Optional[str] = None,
        api_key: Optional[str] = None,
    ):
        self.base_url = base_url.rstrip("/")
        self.token = token
        self.api_key = api_key
        self._session = requests.Session()
        self._session.headers.update({"Content-Type": "application/json"})

    @staticmethod
    def login(
        server_url: str, email: str, password: str
    ) -> "RenOSClient":
        base_url = (server_url or DEFAULT_BASE_URL).rstrip("/")
        resp = requests.post(
            f"{base_url}/api/v1/auth/login",
            json={"email": email, "password": password},
        )
        if resp.status_code != 200:
            raise RenOSException(
                "LOGIN_FAILED",
                f"Login failed ({resp.status_code}): {resp.text}",
            )
        data = resp.json()
        return RenOSClient(base_url=base_url, token=data["token"])

    def set_token(self, token: str) -> None:
        self.token = token

    def set_api_key(self, key: str) -> None:
        self.api_key = key

    def _request(
        self, method: str, path: str, **kwargs: Any
    ) -> Any:
        url = f"{self.base_url}{path}"
        headers = {}
        if self.token:
            headers["Authorization"] = f"Bearer {self.token}"
        if self.api_key:
            headers["X-API-Key"] = self.api_key

        if "json" in kwargs:
            kwargs.setdefault("headers", {}).update(headers)
        else:
            kwargs.setdefault("headers", headers)

        resp = self._session.request(method, url, **kwargs)
        if not resp.ok:
            try:
                err = resp.json()
                raise RenOSException(
                    err.get("code", "API_ERROR"),
                    err.get("message", resp.text),
                    err.get("details"),
                )
            except ValueError:
                raise RenOSException("API_ERROR", resp.text)
        if resp.status_code == 204:
            return None
        return resp.json()

    def _paginated(
        self,
        path: str,
        model_cls: type,
        page: int = 1,
        per_page: int = 50,
    ) -> PaginatedResponse:
        data = self._request(
            "GET", path, params={"page": page, "per_page": per_page}
        )
        items = [model_cls(**item) for item in data.get("data", [])]
        return PaginatedResponse(
            total=data.get("total", 0),
            page=data.get("page", 1),
            per_page=data.get("per_page", 50),
            data=items,
        )

    def list_devices(self, page: int = 1, per_page: int = 50) -> PaginatedResponse[Device]:
        return self._paginated("/api/v1/devices", Device, page, per_page)

    def get_device(self, device_id: str) -> Device:
        return Device(**self._request("GET", f"/api/v1/devices/{device_id}"))

    def execute_command(self, device_id: str, command: str) -> Command:
        return Command(**self._request("POST", f"/api/v1/devices/{device_id}/commands", json={"command": command}))

    def get_command(self, device_id: str, cmd_id: str) -> Command:
        return Command(**self._request("GET", f"/api/v1/devices/{device_id}/commands/{cmd_id}"))

    def list_commands(self, device_id: str, page: int = 1, per_page: int = 50) -> PaginatedResponse[Command]:
        return self._paginated(f"/api/v1/devices/{device_id}/commands", Command, page, per_page)

    def list_jobs(self, page: int = 1, per_page: int = 50) -> PaginatedResponse[Job]:
        return self._paginated("/api/v1/jobs", Job, page, per_page)

    def get_job(self, job_id: str) -> Job:
        return Job(**self._request("GET", f"/api/v1/jobs/{job_id}"))

    def cancel_job(self, job_id: str) -> None:
        self._request("POST", f"/api/v1/jobs/{job_id}/cancel")

    def retry_job(self, job_id: str) -> None:
        self._request("POST", f"/api/v1/jobs/{job_id}/retry")

    def list_workflows(self, page: int = 1, per_page: int = 50) -> PaginatedResponse[Workflow]:
        return self._paginated("/api/v1/workflows", Workflow, page, per_page)

    def get_workflow(self, workflow_id: str) -> Workflow:
        return Workflow(**self._request("GET", f"/api/v1/workflows/{workflow_id}"))

    def execute_workflow(self, workflow_id: str) -> WorkflowRun:
        return WorkflowRun(**self._request("POST", f"/api/v1/workflows/{workflow_id}/execute"))

    def list_users(self, page: int = 1, per_page: int = 50) -> PaginatedResponse[User]:
        return self._paginated("/api/v1/users", User, page, per_page)

    def list_alerts(self, page: int = 1, per_page: int = 50) -> PaginatedResponse[AlertInstance]:
        return self._paginated("/api/v1/alerts/active", AlertInstance, page, per_page)

    def list_groups(self, page: int = 1, per_page: int = 50) -> PaginatedResponse[Group]:
        return self._paginated("/api/v1/groups", Group, page, per_page)

    def get_group(self, group_id: str) -> Group:
        return Group(**self._request("GET", f"/api/v1/groups/{group_id}"))

    def list_schedules(self, page: int = 1, per_page: int = 50) -> PaginatedResponse[Schedule]:
        return self._paginated("/api/v1/schedules", Schedule, page, per_page)

    def list_policies(self, page: int = 1, per_page: int = 50) -> PaginatedResponse[Policy]:
        return self._paginated("/api/v1/policies", Policy, page, per_page)

    def list_vault_secrets(self, page: int = 1, per_page: int = 50) -> PaginatedResponse[VaultSecret]:
        return self._paginated("/api/v1/vault/secrets", VaultSecret, page, per_page)

    def list_cluster_nodes(self, page: int = 1, per_page: int = 50) -> PaginatedResponse[ClusterNode]:
        return self._paginated("/api/v1/cluster/nodes", ClusterNode, page, per_page)

    def list_terminal_sessions(
        self, device_id: str, page: int = 1, per_page: int = 50
    ) -> PaginatedResponse[TerminalSession]:
        return self._paginated(
            f"/api/v1/devices/{device_id}/terminal/sessions",
            TerminalSession, page, per_page
        )

    def list_executions(
        self, device_id: str, page: int = 1, per_page: int = 50
    ) -> PaginatedResponse[Execution]:
        return self._paginated(
            f"/api/v1/devices/{device_id}/executions", Execution, page, per_page
        )

    def create_execution(
        self, device_id: str, command: str, args: Optional[list[str]] = None
    ) -> Execution:
        return Execution(
            **self._request(
                "POST",
                f"/api/v1/devices/{device_id}/executions",
                json={"command": command, "args": args or []},
            )
        )

    def get_execution(self, device_id: str, exec_id: str) -> Execution:
        return Execution(
            **self._request(
                "GET", f"/api/v1/devices/{device_id}/executions/{exec_id}"
            )
        )

    def cancel_execution(self, device_id: str, exec_id: str) -> None:
        self._request(
            "POST", f"/api/v1/devices/{device_id}/executions/{exec_id}/cancel"
        )

    def stream_events(self, path: str) -> "StreamIterator":
        url = f"{self.base_url}{path}"
        headers = {"Accept": "text/event-stream"}
        if self.token:
            headers["Authorization"] = f"Bearer {self.token}"
        if self.api_key:
            headers["X-API-Key"] = self.api_key

        resp = self._session.get(url, headers=headers, stream=True)
        if not resp.ok:
            raise RenOSException("STREAM_ERROR", f"Stream returned {resp.status_code}")
        return StreamIterator(resp)

    def websocket(self, path: str) -> "WebSocketClient":
        ws_url = self.base_url.replace("http://", "ws://").replace("https://", "wss://")
        return WebSocketClient(
            f"{ws_url}{path}", token=self.token, api_key=self.api_key
        )


class StreamIterator:
    def __init__(self, response: requests.Response):
        self._response = response
        self._iterator = response.iter_lines()

    def __iter__(self) -> Iterator[Any]:
        return self

    def __next__(self) -> Any:
        for line in self._iterator:
            if not line:
                continue
            decoded = line.decode("utf-8")
            if decoded.startswith("data: "):
                try:
                    return json.loads(decoded[6:])
                except json.JSONDecodeError:
                    return decoded[6:]
        raise StopIteration

    def close(self) -> None:
        self._response.close()


class WebSocketClient:
    def __init__(
        self,
        url: str,
        token: Optional[str] = None,
        api_key: Optional[str] = None,
    ):
        self.url = url
        self.token = token
        self.api_key = api_key
        self._ws: Any = None
        self._reconnect = False
        self._reconnect_delay = 1.0
        self._max_reconnect_delay = 30.0
        self._closed = False
        self._connect()

    def _connect(self) -> None:
        if self._closed:
            return
        import websockets.sync.client as ws_client

        extra_headers = {}
        if self.token:
            extra_headers["Authorization"] = f"Bearer {self.token}"
        if self.api_key:
            extra_headers["X-API-Key"] = self.api_key

        try:
            self._ws = ws_client.connect(self.url, additional_headers=extra_headers)
            self._reconnect_delay = 1.0
        except Exception:
            if self._reconnect:
                time.sleep(self._reconnect_delay)
                self._reconnect_delay = min(
                    self._reconnect_delay * 2, self._max_reconnect_delay
                )
                self._connect()

    def recv(self) -> str:
        if self._ws is None:
            raise ConnectionError("WebSocket not connected")
        try:
            return self._ws.recv()
        except Exception:
            if self._reconnect:
                self._connect()
                return self.recv()
            raise

    def send(self, data: Any) -> None:
        if self._ws is None:
            raise ConnectionError("WebSocket not connected")
        payload = json.dumps(data) if not isinstance(data, str) else data
        self._ws.send(payload)

    def __iter__(self) -> Iterator[str]:
        return self

    def __next__(self) -> str:
        return self.recv()

    def enable_reconnect(self) -> None:
        self._reconnect = True

    def disable_reconnect(self) -> None:
        self._reconnect = False

    def close(self) -> None:
        self._closed = True
        if self._ws:
            self._ws.close()
