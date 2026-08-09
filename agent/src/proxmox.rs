use serde::{Deserialize, Serialize};
use std::collections::HashMap;

// --- Auth ---

#[derive(Debug, Clone)]
pub enum AuthMethod {
    Token { token_id: String, token_secret: String },
    Password { username: String, password: String },
}

#[derive(Debug, Clone)]
pub struct ProxmoxConnection {
    pub host: String,
    pub port: u16,
    pub auth: AuthMethod,
    pub verify_ssl: bool,
}

// --- API Response Wrappers ---

#[derive(Debug, Deserialize)]
struct ApiResponse<T> {
    data: T,
}

#[derive(Debug, Deserialize)]
struct NodeInfo {
    node: String,
    status: String,
    cpu: Option<f64>,
    maxcpu: Option<u64>,
    mem: Option<u64>,
    maxmem: Option<u64>,
    disk: Option<u64>,
    maxdisk: Option<u64>,
    uptime: Option<u64>,
    ssl_fingerprint: Option<String>,
    version: Option<String>,
}

#[derive(Debug, Deserialize)]
struct QemuStatus {
    vmid: String,
    name: Option<String>,
    status: Option<String>,
    cpu: Option<f64>,
    cpus: Option<u64>,
    mem: Option<u64>,
    maxmem: Option<u64>,
    disk: Option<u64>,
    maxdisk: Option<u64>,
    netin: Option<u64>,
    netout: Option<u64>,
    ipaddress: Option<String>,
    ostype: Option<String>,
    template: Option<u64>,
    ha: Option<HaState>,
}

#[derive(Debug, Deserialize)]
struct HaState {
    #[serde(rename = "group")]
    _group: Option<String>,
    status: Option<String>,
}

#[derive(Debug, Deserialize)]
struct LxcStatus {
    vmid: String,
    name: Option<String>,
    status: Option<String>,
    cpu: Option<f64>,
    cpus: Option<u64>,
    mem: Option<u64>,
    maxmem: Option<u64>,
    disk: Option<u64>,
    maxdisk: Option<u64>,
    ipaddress: Option<String>,
    ostype: Option<String>,
    template: Option<u64>,
}

#[derive(Debug, Deserialize)]
struct StorageInfo {
    storage: String,
    #[serde(rename = "type")]
    storage_type: Option<String>,
    status: Option<String>,
    total: Option<u64>,
    used: Option<u64>,
    content: Option<String>,
}

#[derive(Debug, Deserialize)]
struct NetworkInfo {
    iface: String,
    #[serde(rename = "type")]
    iface_type: Option<String>,
    address: Option<String>,
    netmask: Option<String>,
    gateway: Option<String>,
    bridge: Option<String>,
    vlan: Option<u64>,
    active: Option<u64>,
}

#[derive(Debug, Deserialize)]
struct SnapshotInfo {
    #[serde(rename = "snap")]
    snap_id: String,
    name: Option<String>,
    description: Option<String>,
    parent: Option<String>,
}

#[derive(Debug, Deserialize)]
struct TaskInfo {
    upid: String,
    #[serde(rename = "type")]
    task_type: Option<String>,
    status: Option<String>,
    user_name: Option<String>,
    starttime: Option<u64>,
    endtime: Option<u64>,
    exitstatus: Option<String>,
}

// --- Sync Payloads (matching Core types) ---

#[derive(Debug, Serialize, Clone)]
pub struct SyncNode {
    pub node_name: String,
    pub status: String,
    pub cpu: f64,
    pub max_cpu: i64,
    pub memory: i64,
    pub max_memory: i64,
    pub disk: i64,
    pub max_disk: i64,
    pub uptime: i64,
    pub ssl_fingerprint: String,
    pub version: String,
}

#[derive(Debug, Serialize, Clone)]
pub struct SyncVM {
    pub node_name: String,
    pub vm_id: i64,
    pub name: String,
    pub status: String,
    pub cpu: i64,
    pub cores: i64,
    pub memory: i64,
    pub disk: i64,
    pub ip_address: String,
    pub os_type: String,
    pub template: bool,
    pub ha_state: String,
}

#[derive(Debug, Serialize, Clone)]
pub struct SyncLXC {
    pub node_name: String,
    pub vm_id: i64,
    pub name: String,
    pub status: String,
    pub cpu: i64,
    pub cores: i64,
    pub memory: i64,
    pub disk: i64,
    pub ip_address: String,
    pub os_type: String,
    pub template: bool,
}

#[derive(Debug, Serialize, Clone)]
pub struct SyncStorage {
    pub node_name: String,
    pub storage_id: String,
    pub storage_type: String,
    pub status: String,
    pub total: i64,
    pub used: i64,
    pub content: String,
}

#[derive(Debug, Serialize, Clone)]
pub struct SyncNetwork {
    pub node_name: String,
    pub interface: String,
    pub iface_type: String,
    pub address: String,
    pub netmask: String,
    pub gateway: String,
    pub bridge: String,
    pub vlan: Option<i64>,
    pub active: bool,
}

#[derive(Debug, Serialize, Clone)]
pub struct SyncSnapshot {
    pub node_name: String,
    pub vm_id: i64,
    pub snapshot_id: String,
    pub name: String,
    pub description: String,
    pub parent: String,
}

#[derive(Debug, Serialize, Clone)]
pub struct SyncTask {
    pub node_name: String,
    pub upid: String,
    pub task_type: String,
    pub status: String,
    pub user_name: String,
    pub start_time: i64,
    pub end_time: i64,
    pub exit_status: String,
}

#[derive(Debug, Serialize)]
pub struct SyncPayload {
    pub device_id: String,
    pub device_secret: String,
    pub nodes: Vec<SyncNode>,
    pub vms: Vec<SyncVM>,
    pub lxcs: Vec<SyncLXC>,
    pub storage: Vec<SyncStorage>,
    pub networks: Vec<SyncNetwork>,
    pub snapshots: Vec<SyncSnapshot>,
    pub tasks: Vec<SyncTask>,
}

// --- API Client ---

pub struct ProxmoxClient {
    conn: ProxmoxConnection,
    client: reqwest::Client,
    ticket: Option<String>,
}

impl ProxmoxClient {
    pub fn new(conn: ProxmoxConnection) -> Self {
        let client_builder = if conn.verify_ssl {
            reqwest::Client::builder()
        } else {
            reqwest::Client::builder()
                .danger_accept_invalid_certs(true)
        };

        Self {
            conn,
            client: client_builder.build().unwrap_or_default(),
            ticket: None,
        }
    }

    pub async fn authenticate(&mut self) -> Result<(), String> {
        match &self.conn.auth {
            AuthMethod::Token { .. } => Ok(()),
            AuthMethod::Password { username, password } => {
                let url = format!("https://{}:{}/api2/json/access/ticket", self.conn.host, self.conn.port);
                let params = HashMap::from([
                    ("username", username.as_str()),
                    ("password", password.as_str()),
                ]);

                let resp: ApiResponse<serde_json::Value> = self.client.post(&url)
                    .form(&params)
                    .send()
                    .await
                    .map_err(|e| format!("auth request failed: {e}"))?
                    .json()
                    .await
                    .map_err(|e| format!("auth response parse failed: {e}"))?;

                if let Some(ticket) = resp.data.get("ticket").and_then(|v| v.as_str()) {
                    self.ticket = Some(ticket.to_string());
                    Ok(())
                } else {
                    Err("no ticket in auth response".to_string())
                }
            }
        }
    }

    async fn get<T: serde::de::DeserializeOwned>(&mut self, path: &str) -> Result<T, String> {
        let url = format!("https://{}:{}{}", self.conn.host, self.conn.port, path);

        let mut req = self.client.get(&url);

        match &self.conn.auth {
            AuthMethod::Token { token_id, token_secret } => {
                req = req.header("PVEAPIToken", format!("{}={}", token_id, token_secret));
            }
            AuthMethod::Password { .. } => {
                if let Some(ref ticket) = self.ticket {
                    req = req.header("Cookie", format!("PVEAuthCookie={}", ticket));
                } else {
                    return Err("not authenticated".to_string());
                }
            }
        }

        req.send()
            .await
            .map_err(|e| format!("GET {} failed: {e}", path))?
            .json()
            .await
            .map_err(|e| format!("parse GET {} failed: {e}", path))
    }

    pub async fn collect_all(&mut self) -> Result<CollectedData, String> {
        self.authenticate().await?;

        let nodes = self.get_nodes().await?;
        let mut all_vms = Vec::new();
        let mut all_lxcs = Vec::new();
        let mut all_storage = Vec::new();
        let mut all_networks = Vec::new();
        let mut all_snapshots = Vec::new();
        let mut all_tasks = Vec::new();

        for node in &nodes {
            match self.get_vms(&node.node_name).await {
                Ok(vms) => all_vms.extend(vms),
                Err(e) => tracing::warn!("failed to get VMs on {}: {}", node.node_name, e),
            }
            match self.get_lxcs(&node.node_name).await {
                Ok(lxcs) => all_lxcs.extend(lxcs),
                Err(e) => tracing::warn!("failed to get LXCs on {}: {}", node.node_name, e),
            }
            match self.get_storage(&node.node_name).await {
                Ok(s) => all_storage.extend(s),
                Err(e) => tracing::warn!("failed to get storage on {}: {}", node.node_name, e),
            }
            match self.get_networks(&node.node_name).await {
                Ok(n) => all_networks.extend(n),
                Err(e) => tracing::warn!("failed to get networks on {}: {}", node.node_name, e),
            }
            match self.get_snapshots(&node.node_name).await {
                Ok(s) => all_snapshots.extend(s),
                Err(e) => tracing::warn!("failed to get snapshots on {}: {}", node.node_name, e),
            }
            match self.get_tasks(&node.node_name).await {
                Ok(t) => all_tasks.extend(t),
                Err(e) => tracing::warn!("failed to get tasks on {}: {}", node.node_name, e),
            }
        }

        Ok(CollectedData {
            nodes,
            vms: all_vms,
            lxcs: all_lxcs,
            storage: all_storage,
            networks: all_networks,
            snapshots: all_snapshots,
            tasks: all_tasks,
        })
    }

    async fn get_nodes(&mut self) -> Result<Vec<SyncNode>, String> {
        let raw: ApiResponse<Vec<NodeInfo>> = self.get("/nodes").await?;

        Ok(raw.data.into_iter().map(|n| SyncNode {
            node_name: n.node.clone(),
            status: if n.status == "online" { "online".to_string() } else { "offline".to_string() },
            cpu: n.cpu.unwrap_or(0.0),
            max_cpu: n.maxcpu.unwrap_or(0) as i64,
            memory: n.mem.unwrap_or(0) as i64,
            max_memory: n.maxmem.unwrap_or(0) as i64,
            disk: n.disk.unwrap_or(0) as i64,
            max_disk: n.maxdisk.unwrap_or(0) as i64,
            uptime: n.uptime.unwrap_or(0) as i64,
            ssl_fingerprint: n.ssl_fingerprint.unwrap_or_default(),
            version: n.version.unwrap_or_default(),
        }).collect())
    }

    async fn get_vms(&mut self, node: &str) -> Result<Vec<SyncVM>, String> {
        let path = format!("/nodes/{}/qemu", node);
        let raw: ApiResponse<Vec<QemuStatus>> = self.get(&path).await?;

        Ok(raw.data.into_iter().map(|v| SyncVM {
            node_name: node.to_string(),
            vm_id: v.vmid.parse().unwrap_or(0),
            name: v.name.unwrap_or_default(),
            status: v.status.unwrap_or_default(),
            cpu: v.cpu.unwrap_or(0.0) as i64,
            cores: v.cpus.unwrap_or(0) as i64,
            memory: v.mem.unwrap_or(0) as i64,
            disk: v.disk.unwrap_or(0) as i64,
            ip_address: v.ipaddress.unwrap_or_default(),
            os_type: v.ostype.unwrap_or_default(),
            template: v.template.unwrap_or(0) == 1,
            ha_state: v.ha.map(|h| h.status.unwrap_or_default()).unwrap_or_default(),
        }).collect())
    }

    async fn get_lxcs(&mut self, node: &str) -> Result<Vec<SyncLXC>, String> {
        let path = format!("/nodes/{}/lxc", node);
        let raw: ApiResponse<Vec<LxcStatus>> = self.get(&path).await?;

        Ok(raw.data.into_iter().map(|l| SyncLXC {
            node_name: node.to_string(),
            vm_id: l.vmid.parse().unwrap_or(0),
            name: l.name.unwrap_or_default(),
            status: l.status.unwrap_or_default(),
            cpu: l.cpu.unwrap_or(0.0) as i64,
            cores: l.cpus.unwrap_or(0) as i64,
            memory: l.mem.unwrap_or(0) as i64,
            disk: l.disk.unwrap_or(0) as i64,
            ip_address: l.ipaddress.unwrap_or_default(),
            os_type: l.ostype.unwrap_or_default(),
            template: l.template.unwrap_or(0) == 1,
        }).collect())
    }

    async fn get_storage(&mut self, node: &str) -> Result<Vec<SyncStorage>, String> {
        let path = format!("/nodes/{}/storage", node);
        let raw: ApiResponse<Vec<StorageInfo>> = self.get(&path).await?;

        Ok(raw.data.into_iter().map(|s| SyncStorage {
            node_name: node.to_string(),
            storage_id: s.storage,
            storage_type: s.storage_type.unwrap_or_default(),
            status: s.status.unwrap_or_default(),
            total: s.total.unwrap_or(0) as i64,
            used: s.used.unwrap_or(0) as i64,
            content: s.content.unwrap_or_default(),
        }).collect())
    }

    async fn get_networks(&mut self, node: &str) -> Result<Vec<SyncNetwork>, String> {
        let path = format!("/nodes/{}/network", node);
        let raw: ApiResponse<Vec<NetworkInfo>> = self.get(&path).await?;

        Ok(raw.data.into_iter().map(|n| SyncNetwork {
            node_name: node.to_string(),
            interface: n.iface,
            iface_type: n.iface_type.unwrap_or_default(),
            address: n.address.unwrap_or_default(),
            netmask: n.netmask.unwrap_or_default(),
            gateway: n.gateway.unwrap_or_default(),
            bridge: n.bridge.unwrap_or_default(),
            vlan: n.vlan.map(|v| v as i64),
            active: n.active.unwrap_or(0) == 1,
        }).collect())
    }

    async fn get_snapshots(&mut self, node: &str) -> Result<Vec<SyncSnapshot>, String> {
        // Snapshots are per-VM, we need to list VMs first then query snapshots
        let path = format!("/nodes/{}/qemu", node);
        let vms: ApiResponse<Vec<QemuStatus>> = self.get(&path).await?;

        let mut snapshots = Vec::new();
        for vm in vms.data {
            let snap_path = format!("/nodes/{}/qemu/{}/snapshot", node, vm.vmid);
            match self.get::<ApiResponse<Vec<SnapshotInfo>>>(&snap_path).await {
                Ok(snap_resp) => {
                    for s in snap_resp.data {
                        snapshots.push(SyncSnapshot {
                            node_name: node.to_string(),
                            vm_id: vm.vmid.parse().unwrap_or(0),
                            snapshot_id: s.snap_id,
                            name: s.name.unwrap_or_default(),
                            description: s.description.unwrap_or_default(),
                            parent: s.parent.unwrap_or_default(),
                        });
                    }
                }
                Err(e) => tracing::warn!("failed to get snapshots for vm {} on {}: {}", vm.vmid, node, e),
            }
        }

        // Also check LXC snapshots
        let lxc_path = format!("/nodes/{}/lxc", node);
        let lxcs: ApiResponse<Vec<LxcStatus>> = self.get(&lxc_path).await?;

        for lxc in lxcs.data {
            let snap_path = format!("/nodes/{}/lxc/{}/snapshot", node, lxc.vmid);
            match self.get::<ApiResponse<Vec<SnapshotInfo>>>(&snap_path).await {
                Ok(snap_resp) => {
                    for s in snap_resp.data {
                        snapshots.push(SyncSnapshot {
                            node_name: node.to_string(),
                            vm_id: lxc.vmid.parse().unwrap_or(0),
                            snapshot_id: s.snap_id,
                            name: s.name.unwrap_or_default(),
                            description: s.description.unwrap_or_default(),
                            parent: s.parent.unwrap_or_default(),
                        });
                    }
                }
                Err(e) => tracing::warn!("failed to get snapshots for lxc {} on {}: {}", lxc.vmid, node, e),
            }
        }

        Ok(snapshots)
    }

    async fn get_tasks(&mut self, node: &str) -> Result<Vec<SyncTask>, String> {
        let path = format!("/nodes/{}/tasks", node);
        let raw: ApiResponse<Vec<TaskInfo>> = self.get(&path).await?;

        Ok(raw.data.into_iter().map(|t| SyncTask {
            node_name: node.to_string(),
            upid: t.upid,
            task_type: t.task_type.unwrap_or_default(),
            status: t.status.unwrap_or_default(),
            user_name: t.user_name.unwrap_or_default(),
            start_time: t.starttime.unwrap_or(0) as i64,
            end_time: t.endtime.unwrap_or(0) as i64,
            exit_status: t.exitstatus.unwrap_or_default(),
        }).collect())
    }

    pub async fn power_action(&mut self, node: &str, vmid: i64, action: &str, vm_type: &str) -> Result<(), String> {
        let endpoint = match vm_type {
            "qemu" => format!("/nodes/{}/qemu/{}/status/{}", node, vmid, action),
            "lxc" => format!("/nodes/{}/lxc/{}/status/{}", node, vmid, action),
            _ => return Err(format!("unknown vm type: {}", vm_type)),
        };

        let url = format!("https://{}:{}{}", self.conn.host, self.conn.port, endpoint);

        let mut req = self.client.post(&url);
        match &self.conn.auth {
            AuthMethod::Token { token_id, token_secret } => {
                req = req.header("PVEAPIToken", format!("{}={}", token_id, token_secret));
            }
            AuthMethod::Password { .. } => {
                if let Some(ref ticket) = self.ticket {
                    req = req.header("Cookie", format!("PVEAuthCookie={}", ticket));
                } else {
                    return Err("not authenticated".to_string());
                }
            }
        }

        req.send()
            .await
            .map_err(|e| format!("power action request failed: {e}"))?;

        Ok(())
    }
}

pub struct CollectedData {
    pub nodes: Vec<SyncNode>,
    pub vms: Vec<SyncVM>,
    pub lxcs: Vec<SyncLXC>,
    pub storage: Vec<SyncStorage>,
    pub networks: Vec<SyncNetwork>,
    pub snapshots: Vec<SyncSnapshot>,
    pub tasks: Vec<SyncTask>,
}

// --- Proxmox detection ---

pub fn is_proxmox() -> bool {
    std::process::Command::new("pveversion")
        .output()
        .map(|o| o.status.success())
        .unwrap_or(false)
}

// --- Submit to Core ---

pub async fn submit_proxmox_data(
    api_url: &str,
    device_id: &str,
    device_secret: &str,
    data: &CollectedData,
) -> Result<(), String> {
    let client = reqwest::Client::new();
    let url = format!("{}/api/v1/devices/{}/proxmox", api_url, device_id);

    let payload = SyncPayload {
        device_id: device_id.to_string(),
        device_secret: device_secret.to_string(),
        nodes: data.nodes.clone(),
        vms: data.vms.clone(),
        lxcs: data.lxcs.clone(),
        storage: data.storage.clone(),
        networks: data.networks.clone(),
        snapshots: data.snapshots.clone(),
        tasks: data.tasks.clone(),
    };

    let resp = client
        .post(&url)
        .json(&payload)
        .timeout(std::time::Duration::from_secs(30))
        .send()
        .await
        .map_err(|e| format!("proxmox sync request failed: {e}"))?;

    let status = resp.status();
    if !status.is_success() {
        let body = resp.text().await.unwrap_or_default();
        return Err(format!("proxmox sync rejected ({}): {}", status, body));
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_is_proxmox() {
        // Just ensure it doesn't panic
        let _ = is_proxmox();
    }
}
