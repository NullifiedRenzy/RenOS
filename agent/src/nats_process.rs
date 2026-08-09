use futures::StreamExt;
use serde::{Deserialize, Serialize};

use crate::processes;

#[derive(Debug, Deserialize)]
struct ProcessListRequest {
    device_id: String,
}

#[derive(Debug, Deserialize)]
struct ProcessActionRequest {
    action_id: String,
    device_id: String,
    pid: u32,
    action: String,
}

#[derive(Debug, Serialize)]
struct ProcessSnapshotMessage {
    device_id: String,
    processes: Vec<processes::ProcessInfo>,
}

#[derive(Debug, Serialize)]
struct ProcessActionResultMessage {
    action_id: String,
    exit_code: i32,
    output: String,
    error: String,
}

pub async fn start_nats_process_handler(
    nc: async_nats::Client,
    device_id: String,
) -> tokio::task::JoinHandle<()> {
    tokio::spawn(async move {
        // Subscribe to process list requests
        let list_subject = format!("process.list.{}", device_id);
        let mut list_sub = match nc.subscribe(list_subject.clone()).await {
            Ok(sub) => sub,
            Err(e) => {
                tracing::error!("failed to subscribe to {}: {}", list_subject, e);
                return;
            }
        };

        // Subscribe to process actions
        let action_subject = format!("process.action.{}", device_id);
        let mut action_sub = match nc.subscribe(action_subject.clone()).await {
            Ok(sub) => sub,
            Err(e) => {
                tracing::error!("failed to subscribe to {}: {}", action_subject, e);
                return;
            }
        };

        tracing::info!("NATS process handler subscribed to {} and {}", list_subject, action_subject);

        // Handle list requests and action requests concurrently
        loop {
            tokio::select! {
                Some(msg) = list_sub.next() => {
                    let _req: ProcessListRequest = match serde_json::from_slice(&msg.payload) {
                        Ok(r) => r,
                        Err(e) => {
                            tracing::warn!("failed to deserialize process list request: {}", e);
                            continue;
                        }
                    };

                    let processes_list = processes::collect_processes();
                    tracing::info!("collected {} processes for snapshot", processes_list.len());

                    let snapshot = ProcessSnapshotMessage {
                        device_id: device_id.clone(),
                        processes: processes_list,
                    };

                    let snapshot_subject = format!("process.snapshot.{}", device_id);
                    if let Err(e) = nc.publish(
                        snapshot_subject,
                        serde_json::to_vec(&snapshot).unwrap().into(),
                    ).await {
                        tracing::warn!("failed to publish process snapshot: {}", e);
                    }
                }
                Some(msg) = action_sub.next() => {
                    let req: ProcessActionRequest = match serde_json::from_slice(&msg.payload) {
                        Ok(r) => r,
                        Err(e) => {
                            tracing::warn!("failed to deserialize process action request: {}", e);
                            continue;
                        }
                    };

                    tracing::info!("executing process {} on PID {}", req.action, req.pid);

                    let (exit_code, output, error) = processes::execute_process_action(&req.action, req.pid);

                    let result = ProcessActionResultMessage {
                        action_id: req.action_id,
                        exit_code,
                        output,
                        error,
                    };

                    let result_subject = format!("process.result.{}", device_id);
                    if let Err(e) = nc.publish(
                        result_subject,
                        serde_json::to_vec(&result).unwrap().into(),
                    ).await {
                        tracing::warn!("failed to publish process action result: {}", e);
                    }
                }
                else => {
                    tracing::warn!("NATS process subscriptions ended");
                    break;
                }
            }
        }
    })
}

#[cfg(test)]
mod tests {
    use super::*;
}
