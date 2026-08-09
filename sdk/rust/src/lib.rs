//! RenOS Rust SDK
//!
//! # Usage
//!
//! ```rust,no_run
//! use renos_sdk::Client;
//!
//! #[tokio::main]
//! async fn main() -> Result<(), Box<dyn std::error::Error>> {
//!     let client = Client::login("https://renos.example.com", "admin@example.com", "password").await?;
//!
//!     let devices = client.list_devices(1, 50).await?;
//!     for device in &devices.data {
//!         println!("{}: {}", device.name, device.status);
//!     }
//!
//!     let cmd = client.execute_command("device-uuid", "uptime").await?;
//!     println!("Command {}: {}", cmd.id, cmd.status);
//!
//!     Ok(())
//! }
//! ```

use bytes::Bytes;
use futures_util::StreamExt;
use http::HeaderValue;
use pin_project::pin_project;
use reqwest::{
    header::{AUTHORIZATION, CONTENT_TYPE},
    Client as HttpClient, Response,
};
use serde::{de::DeserializeOwned, Deserialize, Serialize};
use std::{
    pin::Pin,
    sync::Arc,
    task::{Context, Poll},
};
use tokio::sync::RwLock;
use url::Url;

const DEFAULT_BASE_URL: &str = "http://localhost:8443";

#[derive(Debug, thiserror::Error)]
pub enum RenosError {
    #[error("API error: [{code}] {message}")]
    ApiError {
        code: String,
        message: String,
        details: Option<serde_json::Value>,
    },
    #[error("HTTP error: {0}")]
    Http(#[from] reqwest::Error),
    #[error("WebSocket error: {0}")]
    Ws(String),
    #[error("Stream error: {0}")]
    Stream(String),
    #[error("Serialization error: {0}")]
    Serde(#[from] serde_json::Error),
    #[error("Invalid URL: {0}")]
    InvalidUrl(String),
}

impl RenosError {
    pub fn api_error(code: &str, message: &str, details: Option<serde_json::Value>) -> Self {
        Self::ApiError { code: code.to_string(), message: message.to_string(), details }
    }
}

// --- Client ---

pub struct Client {
    http: HttpClient,
    base_url: String,
    token: Arc<RwLock<Option<String>>>,
    api_key: Arc<RwLock<Option<String>>>,
}

impl Client {
    pub fn new(base_url: &str, token: Option<&str>) -> Self {
        let base = if base_url.is_empty() { DEFAULT_BASE_URL.to_string() } else { base_url.trim_end_matches('/').to_string() };

        let http = HttpClient::builder()
            .default_headers({
                let mut h = reqwest::header::HeaderMap::new();
                h.insert(CONTENT_TYPE, HeaderValue::from_static("application/json"));
                h
            })
            .build()
            .expect("Failed to build HTTP client");

        Self { http, base_url: base, token: Arc::new(RwLock::new(token.map(String::from))), api_key: Arc::new(RwLock::new(None)) }
    }

    pub fn with_api_key(base_url: &str, api_key: &str) -> Self {
        let client = Self::new(base_url, None);
        *client.api_key.blocking_write() = Some(api_key.to_string());
        client
    }

    pub async fn login(server_url: &str, email: &str, password: &str) -> Result<Self, RenosError> {
        let base_url = if server_url.is_empty() { DEFAULT_BASE_URL.to_string() } else { server_url.trim_end_matches('/').to_string() };
        let client = HttpClient::new();
        let body = serde_json::json!({"email": email, "password": password});
        let resp = client.post(format!("{}/api/v1/auth/login", base_url)).json(&body).send().await?;
        if !resp.status().is_success() {
            let text = resp.text().await.unwrap_or_default();
            return Err(RenosError::api_error("LOGIN_FAILED", &text, None));
        }
        #[derive(Deserialize)]
        struct LoginResp { token: String }
        let data: LoginResp = resp.json().await?;
        Ok(Self::new(&base_url, Some(&data.token)))
    }

    pub async fn set_token(&self, token: &str) {
        *self.token.write().await = Some(token.to_string());
    }

    pub async fn set_api_key(&self, key: &str) {
        *self.api_key.write().await = Some(key.to_string());
    }

    fn add_auth(&self, builder: reqwest::RequestBuilder) -> reqwest::RequestBuilder {
        let token = self.token.blocking_read().clone();
        let api_key = self.api_key.blocking_read().clone();
        let b = if let Some(t) = token { builder.header(AUTHORIZATION, format!("Bearer {}", t)) } else { builder };
        if let Some(k) = api_key { b.header("X-API-Key", k) } else { b }
    }

    async fn request<T: DeserializeOwned>(&self, method: &str, path: &str) -> Result<T, RenosError> {
        let url = format!("{}{}", self.base_url, path);
        let m = reqwest::Method::from_bytes(method.as_bytes()).unwrap();
        let builder = self.add_auth(self.http.request(m, &url));
        Self::handle_response(builder.send().await?).await
    }

    async fn request_with_body<T: DeserializeOwned, B: Serialize>(&self, method: &str, path: &str, body: &B) -> Result<T, RenosError> {
        let url = format!("{}{}", self.base_url, path);
        let m = reqwest::Method::from_bytes(method.as_bytes()).unwrap();
        let builder = self.add_auth(self.http.request(m, &url).json(body));
        Self::handle_response(builder.send().await?).await
    }

    async fn request_with_query<T: DeserializeOwned>(&self, method: &str, path: &str, params: &[(&str, &str)]) -> Result<T, RenosError> {
        let url = format!("{}{}", self.base_url, path);
        let m = reqwest::Method::from_bytes(method.as_bytes()).unwrap();
        let builder = self.add_auth(self.http.request(m, &url).query(params));
        Self::handle_response(builder.send().await?).await
    }

    async fn handle_response<T: DeserializeOwned>(resp: Response) -> Result<T, RenosError> {
        if !resp.status().is_success() {
            let status = resp.status();
            let text = resp.text().await.unwrap_or_default();
            if let Ok(err) = serde_json::from_str::<ApiErrorBody>(&text) {
                return Err(RenosError::ApiError { code: err.code, message: err.message, details: err.details });
            }
            return Err(RenosError::api_error("HTTP_ERROR", &text, None));
        }
        Ok(resp.json().await?)
    }

    async fn paginated<T: DeserializeOwned>(&self, path: &str, page: i32, per_page: i32) -> Result<PaginatedResponse<T>, RenosError> {
        self.request_with_query("GET", path, &[("page", &page.to_string()), ("per_page", &per_page.to_string())]).await
    }

    // --- API methods ---

    pub async fn list_devices(&self, page: i32, per_page: i32) -> Result<PaginatedResponse<Device>, RenosError> {
        self.paginated("/api/v1/devices", page, per_page).await
    }
    pub async fn get_device(&self, id: &str) -> Result<Device, RenosError> {
        self.request("GET", &format!("/api/v1/devices/{}", id)).await
    }
    pub async fn execute_command(&self, device_id: &str, command: &str) -> Result<Command, RenosError> {
        #[derive(Serialize)]
        struct CmdBody<'a> { command: &'a str }
        self.request_with_body("POST", &format!("/api/v1/devices/{}/commands", device_id), &CmdBody { command }).await
    }
    pub async fn get_command(&self, device_id: &str, cmd_id: &str) -> Result<Command, RenosError> {
        self.request("GET", &format!("/api/v1/devices/{}/commands/{}", device_id, cmd_id)).await
    }
    pub async fn list_commands(&self, device_id: &str, page: i32, per_page: i32) -> Result<PaginatedResponse<Command>, RenosError> {
        self.paginated(&format!("/api/v1/devices/{}/commands", device_id), page, per_page).await
    }
    pub async fn list_jobs(&self, page: i32, per_page: i32) -> Result<PaginatedResponse<Job>, RenosError> {
        self.paginated("/api/v1/jobs", page, per_page).await
    }
    pub async fn get_job(&self, id: &str) -> Result<Job, RenosError> {
        self.request("GET", &format!("/api/v1/jobs/{}", id)).await
    }
    pub async fn cancel_job(&self, id: &str) -> Result<(), RenosError> {
        self.request::<serde_json::Value>("POST", &format!("/api/v1/jobs/{}/cancel", id)).await?;
        Ok(())
    }
    pub async fn retry_job(&self, id: &str) -> Result<(), RenosError> {
        self.request::<serde_json::Value>("POST", &format!("/api/v1/jobs/{}/retry", id)).await?;
        Ok(())
    }
    pub async fn list_workflows(&self, page: i32, per_page: i32) -> Result<PaginatedResponse<Workflow>, RenosError> {
        self.paginated("/api/v1/workflows", page, per_page).await
    }
    pub async fn get_workflow(&self, id: &str) -> Result<Workflow, RenosError> {
        self.request("GET", &format!("/api/v1/workflows/{}", id)).await
    }
    pub async fn execute_workflow(&self, id: &str) -> Result<WorkflowRun, RenosError> {
        self.request("POST", &format!("/api/v1/workflows/{}/execute", id)).await
    }
    pub async fn list_users(&self, page: i32, per_page: i32) -> Result<PaginatedResponse<User>, RenosError> {
        self.paginated("/api/v1/users", page, per_page).await
    }
    pub async fn list_alerts(&self, page: i32, per_page: i32) -> Result<PaginatedResponse<AlertInstance>, RenosError> {
        self.paginated("/api/v1/alerts/active", page, per_page).await
    }
    pub async fn list_groups(&self, page: i32, per_page: i32) -> Result<PaginatedResponse<Group>, RenosError> {
        self.paginated("/api/v1/groups", page, per_page).await
    }
    pub async fn get_group(&self, id: &str) -> Result<Group, RenosError> {
        self.request("GET", &format!("/api/v1/groups/{}", id)).await
    }
    pub async fn list_schedules(&self, page: i32, per_page: i32) -> Result<PaginatedResponse<Schedule>, RenosError> {
        self.paginated("/api/v1/schedules", page, per_page).await
    }
    pub async fn list_policies(&self, page: i32, per_page: i32) -> Result<PaginatedResponse<Policy>, RenosError> {
        self.paginated("/api/v1/policies", page, per_page).await
    }
    pub async fn list_vault_secrets(&self, page: i32, per_page: i32) -> Result<PaginatedResponse<VaultSecret>, RenosError> {
        self.paginated("/api/v1/vault/secrets", page, per_page).await
    }
    pub async fn list_cluster_nodes(&self, page: i32, per_page: i32) -> Result<PaginatedResponse<ClusterNode>, RenosError> {
        self.paginated("/api/v1/cluster/nodes", page, per_page).await
    }
    pub async fn list_terminal_sessions(&self, device_id: &str, page: i32, per_page: i32) -> Result<PaginatedResponse<TerminalSession>, RenosError> {
        self.paginated(&format!("/api/v1/devices/{}/terminal/sessions", device_id), page, per_page).await
    }
    pub async fn list_executions(&self, device_id: &str, page: i32, per_page: i32) -> Result<PaginatedResponse<Execution>, RenosError> {
        self.paginated(&format!("/api/v1/devices/{}/executions", device_id), page, per_page).await
    }
    pub async fn create_execution(&self, device_id: &str, command: &str, args: &[String]) -> Result<Execution, RenosError> {
        #[derive(Serialize)]
        struct ExecBody<'a> { command: &'a str, args: &'a [String] }
        self.request_with_body("POST", &format!("/api/v1/devices/{}/executions", device_id), &ExecBody { command, args }).await
    }
    pub async fn get_execution(&self, device_id: &str, exec_id: &str) -> Result<Execution, RenosError> {
        self.request("GET", &format!("/api/v1/devices/{}/executions/{}", device_id, exec_id)).await
    }
    pub async fn cancel_execution(&self, device_id: &str, exec_id: &str) -> Result<(), RenosError> {
        self.request::<serde_json::Value>("POST", &format!("/api/v1/devices/{}/executions/{}/cancel", device_id, exec_id)).await?;
        Ok(())
    }

    /// Create an SSE stream for the given path
    pub async fn stream_events(&self, path: &str) -> Result<SseStream, RenosError> {
        let url = format!("{}{}", self.base_url, path);
        let builder = self.add_auth(self.http.get(&url).header(reqwest::header::ACCEPT, "text/event-stream"));
        let resp = builder.send().await?;
        if !resp.status().is_success() {
            return Err(RenosError::Stream(format!("Stream returned {}", resp.status())));
        }
        Ok(SseStream { inner: resp.bytes_stream(), buf: String::new() })
    }

    /// Connect to a WebSocket endpoint
    pub async fn websocket(&self, path: &str) -> Result<WebSocket, RenosError> {
        let http_url = format!("{}{}", self.base_url, path);
        let ws_url = if http_url.starts_with("https") { http_url.replacen("https", "wss", 1) } else { http_url.replacen("http", "ws", 1) };
        let parsed = Url::parse(&ws_url).map_err(|e| RenosError::InvalidUrl(format!("Invalid WebSocket URL: {}", e)))?;
        let token = self.token.read().await.clone();
        let api_key = self.api_key.read().await.clone();

        let mut request = http::Request::builder().uri(parsed.as_str()).method("GET");
        if let Some(t) = &token { request = request.header("Authorization", format!("Bearer {}", t)); }
        if let Some(k) = &api_key { request = request.header("X-API-Key", k); }

        let (ws, _) = tokio_tungstenite::connect_async(request).await.map_err(|e| RenosError::Ws(format!("WebSocket connect error: {}", e)))?;
        Ok(WebSocket { inner: ws, token, api_key, url: parsed, reconnect_enabled: false })
    }
}

// --- Paginated Response ---

#[derive(Debug, Deserialize)]
pub struct PaginatedResponse<T> {
    pub total: i32,
    pub page: i32,
    pub per_page: i32,
    #[serde(default)]
    pub data: Vec<T>,
}

#[derive(Debug, Deserialize)]
struct ApiErrorBody {
    code: String,
    message: String,
    details: Option<serde_json::Value>,
}

// --- SSE Stream ---

#[pin_project]
pub struct SseStream {
    #[pin]
    inner: reqwest::header::IntoIter,
    buf: String,
}

impl futures_util::Stream for SseStream {
    type Item = Result<String, RenosError>;

    fn poll_next(self: Pin<&mut Self>, _cx: &mut Context<'_>) -> Poll<Option<Self::Item>> {
        Poll::Ready(None)
    }
}

// --- WebSocket ---

pub struct WebSocket {
    inner: tokio_tungstenite::WebSocketStream<tokio_tungstenite::MaybeTlsStream<tokio::net::TcpStream>>,
    token: Option<String>,
    api_key: Option<String>,
    url: Url,
    reconnect_enabled: bool,
}

impl WebSocket {
    pub async fn recv(&mut self) -> Option<Result<String, RenosError>> {
        use tokio_tungstenite::tungstenite::Message;
        loop {
            match self.inner.next().await {
                Some(Ok(Message::Text(text))) => return Some(Ok(text)),
                Some(Ok(Message::Close(_))) => {
                    if self.reconnect_enabled {
                        if let Err(e) = self.reconnect().await {
                            return Some(Err(RenosError::Ws(format!("Reconnect failed: {}", e))));
                        }
                        continue;
                    }
                    return None;
                }
                Some(Err(e)) => {
                    if self.reconnect_enabled {
                        if let Err(re) = self.reconnect().await {
                            return Some(Err(RenosError::Ws(format!("Reconnect failed: {}", re))));
                        }
                        continue;
                    }
                    return Some(Err(RenosError::Ws(e.to_string())));
                }
                None => return None,
                _ => continue,
            }
        }
    }

    pub async fn send(&mut self, message: &str) -> Result<(), RenosError> {
        use tokio_tungstenite::tungstenite::Message;
        self.inner.send(Message::Text(message.to_string())).await.map_err(|e| RenosError::Ws(e.to_string()))
    }

    pub fn enable_reconnect(&mut self) { self.reconnect_enabled = true; }
    pub fn disable_reconnect(&mut self) { self.reconnect_enabled = false; }

    async fn reconnect(&mut self) -> Result<(), RenosError> {
        let mut request = http::Request::builder().uri(self.url.as_str()).method("GET");
        if let Some(t) = &self.token { request = request.header("Authorization", format!("Bearer {}", t)); }
        if let Some(k) = &self.api_key { request = request.header("X-API-Key", k); }
        let (ws, _) = tokio_tungstenite::connect_async(request).await.map_err(|e| RenosError::Ws(format!("Reconnect failed: {}", e)))?;
        self.inner = ws;
        Ok(())
    }

    pub async fn close(self) -> Result<(), RenosError> {
        use tokio_tungstenite::tungstenite::Message;
        self.inner.send(Message::Close(None)).await.map_err(|e| RenosError::Ws(e.to_string()))
    }
}

// --- Model types ---

#[derive(Debug, Deserialize, Serialize)]
pub struct Device {
    pub id: String, pub name: String, pub hostname: String, pub os: String, pub arch: String,
    pub status: String, pub ip: String, pub agent_version: String,
    #[serde(default)] pub tags: std::collections::HashMap<String, String>,
    pub enrolled_at: Option<String>, pub last_seen_at: Option<String>,
    pub created_at: Option<String>, pub updated_at: Option<String>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct DeviceDetail {
    pub id: String, pub name: String, pub hostname: String, pub os: String, pub arch: String,
    pub status: String, pub ip: String, pub agent_version: String,
    #[serde(default)] pub tags: std::collections::HashMap<String, String>,
    pub enrolled_at: Option<String>, pub last_seen_at: Option<String>,
    pub created_at: Option<String>, pub updated_at: Option<String>,
    pub cpu_usage: f64, pub memory_usage: f64, pub disk_usage: f64, pub load_average: f64, pub uptime: i32,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct Command {
    pub id: String, pub device_id: String, pub command: String, pub status: String,
    pub exit_code: Option<i32>, #[serde(default)] pub output: String, #[serde(default)] pub error: String,
    pub timeout: i32, pub created_at: Option<String>, pub completed_at: Option<String>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct Job {
    pub id: String, #[serde(rename = "type")] pub job_type: String, pub status: String,
    pub progress: f64, pub total: i32, pub completed: i32, pub failed: i32,
    pub created_at: Option<String>, pub completed_at: Option<String>, #[serde(default)] pub error: String,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct Workflow {
    pub id: String, pub name: String, #[serde(default)] pub description: String,
    #[serde(default)] pub steps: Vec<WorkflowStep>, pub enabled: bool,
    pub created_at: Option<String>, pub updated_at: Option<String>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct WorkflowStep {
    pub id: String, pub workflow_id: String, pub name: String,
    #[serde(rename = "type")] pub step_type: String, pub config: Option<serde_json::Value>,
    pub order: i32, pub created_at: Option<String>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct WorkflowRun {
    pub id: String, pub workflow_id: String, pub status: String,
    pub started_at: Option<String>, pub completed_at: Option<String>,
    #[serde(default)] pub error: String, #[serde(default)] pub steps: Vec<WorkflowStep>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct User {
    pub id: String, pub email: String, #[serde(default)] pub display_name: String,
    #[serde(default)] pub role: String, #[serde(default)] pub avatar_url: String,
    pub created_at: Option<String>, pub updated_at: Option<String>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct Group {
    pub id: String, pub name: String, #[serde(default)] pub description: String,
    pub device_count: i32, pub created_at: Option<String>, pub updated_at: Option<String>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct Policy {
    pub id: String, pub name: String, #[serde(default)] pub description: String,
    #[serde(rename = "type")] pub policy_type: String, pub conditions: Option<serde_json::Value>,
    pub action: String, pub priority: i32, pub enabled: bool,
    pub created_at: Option<String>, pub updated_at: Option<String>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct Schedule {
    pub id: String, pub name: String, #[serde(rename = "type")] pub schedule_type: String,
    pub cron_expression: String, pub interval_seconds: i32, #[serde(default)] pub actions: Vec<serde_json::Value>,
    pub enabled: bool, pub last_run: Option<String>, pub next_run: Option<String>,
    pub created_at: Option<String>, pub updated_at: Option<String>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct VaultSecret {
    pub id: String, pub name: String, #[serde(default)] pub path: String,
    pub secret_type: String, pub version: i32, pub metadata: Option<serde_json::Value>,
    pub created_at: Option<String>, pub updated_at: Option<String>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct AlertInstance {
    pub id: String, pub rule_id: String, pub name: String, pub severity: String, pub status: String,
    pub metric_value: f64, pub starts_at: Option<String>, pub acknowledged_at: Option<String>,
    pub resolved_at: Option<String>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct AlertRule {
    pub id: String, pub name: String, pub metric: String, pub condition: String,
    pub threshold: f64, pub severity: String, pub enabled: bool, pub created_at: Option<String>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct ClusterNode {
    pub id: String, pub name: String, pub address: String, pub role: String, pub status: String,
    pub joined_at: Option<String>, pub last_heartbeat: Option<String>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct TerminalSession {
    pub id: String, pub device_id: String, pub user_id: String, pub status: String,
    pub created_at: Option<String>, pub closed_at: Option<String>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct Execution {
    pub id: String, pub device_id: String, pub command: String, #[serde(default)] pub args: Vec<String>,
    pub status: String, pub exit_code: Option<i32>, #[serde(default)] pub output: String,
    #[serde(default)] pub error: String, #[serde(default)] pub workdir: String, pub pid: i32,
    pub started_at: Option<String>, pub completed_at: Option<String>, pub created_at: Option<String>,
}

#[derive(Debug, Deserialize, Serialize)]
pub struct FileTransfer {
    pub id: String, pub device_id: String, pub filename: String, #[serde(default)] pub path: String,
    pub direction: String, pub size: i32, pub transferred: i32, pub status: String,
    pub progress: f64, #[serde(default)] pub error: String,
    pub created_at: Option<String>, pub completed_at: Option<String>,
}
