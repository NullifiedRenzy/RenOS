use futures::StreamExt;
use serde::Deserialize;

use crate::proxmox;

#[derive(Debug, Deserialize)]
struct PowerActionRequest {
    action: String,
    #[serde(rename = "type")]
    vm_type: String,
    node_name: String,
    vm_id: i64,
}

pub async fn start_nats_proxmox_handler(
    nc: async_nats::Client,
    device_id: String,
    proxmox_conns: Vec<proxmox::ProxmoxConnection>,
) -> tokio::task::JoinHandle<()> {
    tokio::spawn(async move {
        let subject = format!("proxmox.power.{}", device_id);
        let mut sub = match nc.subscribe(subject.clone()).await {
            Ok(sub) => sub,
            Err(e) => {
                tracing::error!("failed to subscribe to {}: {}", subject, e);
                return;
            }
        };

        tracing::info!("NATS proxmox handler subscribed to {}", subject);

        loop {
            tokio::select! {
                Some(msg) = sub.next() => {
                    let req: PowerActionRequest = match serde_json::from_slice(&msg.payload) {
                        Ok(r) => r,
                        Err(e) => {
                            tracing::warn!("failed to deserialize proxmox power request: {}", e);
                            continue;
                        }
                    };

                    tracing::info!(
                        "executing proxmox power {} on {}/{} ({})",
                        req.action, req.node_name, req.vm_id, req.vm_type
                    );

                    // Try each configured connection until one works
                    let mut success = false;
                    for conn in &proxmox_conns {
                        let mut client = proxmox::ProxmoxClient::new(conn.clone());
                        match client.authenticate().await {
                            Ok(()) => {
                                match client.power_action(&req.node_name, req.vm_id, &req.action, &req.vm_type).await {
                                    Ok(()) => {
                                        tracing::info!(
                                            "proxmox power {} completed on {}/{}",
                                            req.action, req.node_name, req.vm_id
                                        );
                                        success = true;
                                        break;
                                    }
                                    Err(e) => {
                                        tracing::warn!("proxmox power action failed: {}", e);
                                    }
                                }
                            }
                            Err(e) => {
                                tracing::warn!("proxmox auth failed: {}", e);
                            }
                        }
                    }

                    if !success {
                        tracing::error!(
                            "failed to execute proxmox power {} on {}/{} on all connections",
                            req.action, req.node_name, req.vm_id
                        );
                    }
                }
                else => {
                    tracing::warn!("NATS proxmox subscription ended");
                    break;
                }
            }
        }
    })
}
