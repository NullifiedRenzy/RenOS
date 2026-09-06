use base64::engine::general_purpose::STANDARD as BASE64;
use base64::Engine;
use futures::StreamExt;
use serde::{Deserialize, Serialize};

use crate::pty_exec::PtyManager;

// --- NATS message types ---

#[derive(Debug, Deserialize)]
struct TerminalOpenMsg {
    session_id: String,
    cols: u16,
    rows: u16,
    shell: String,
}

#[derive(Debug, Deserialize)]
struct TerminalInputMsg {
    session_id: String,
    data: String,
}

#[derive(Debug, Deserialize)]
struct TerminalResizeMsg {
    session_id: String,
    cols: u16,
    rows: u16,
}

#[derive(Debug, Deserialize)]
struct TerminalCloseMsg {
    session_id: String,
}

#[derive(Debug, Serialize)]
struct TerminalOutputMsg {
    session_id: String,
    data: String,
}

#[derive(Debug, Serialize)]
struct TerminalErrorMsg {
    session_id: String,
    error: String,
}

pub async fn start_nats_terminal_handler(
    nc: async_nats::Client,
    device_id: String,
    pty_manager: PtyManager,
) -> tokio::task::JoinHandle<()> {
    tokio::spawn(async move {
        let open_subject = format!("terminal.open.{}", device_id);
        let mut open_sub = match nc.subscribe(open_subject.clone()).await {
            Ok(sub) => sub,
            Err(e) => {
                tracing::error!("failed to subscribe to {}: {}", open_subject, e);
                return;
            }
        };

        let input_subject = format!("terminal.input.{}", device_id);
        let mut input_sub = match nc.subscribe(input_subject.clone()).await {
            Ok(sub) => sub,
            Err(e) => {
                tracing::error!("failed to subscribe to {}: {}", input_subject, e);
                return;
            }
        };

        let resize_subject = format!("terminal.resize.{}", device_id);
        let mut resize_sub = match nc.subscribe(resize_subject.clone()).await {
            Ok(sub) => sub,
            Err(e) => {
                tracing::error!("failed to subscribe to {}: {}", resize_subject, e);
                return;
            }
        };

        let close_subject = format!("terminal.close.{}", device_id);
        let mut close_sub = match nc.subscribe(close_subject.clone()).await {
            Ok(sub) => sub,
            Err(e) => {
                tracing::error!("failed to subscribe to {}: {}", close_subject, e);
                return;
            }
        };

        tracing::info!(
            "NATS terminal handler subscribed to open={}, input={}, resize={}, close={}",
            open_subject,
            input_subject,
            resize_subject,
            close_subject
        );

        loop {
            tokio::select! {
                Some(msg) = open_sub.next() => {
                    let req: TerminalOpenMsg = match serde_json::from_slice(&msg.payload) {
                        Ok(r) => r,
                        Err(e) => {
                            tracing::warn!("failed to deserialize terminal.open: {}", e);
                            continue;
                        }
                    };

                    tracing::info!(
                        "opening terminal session {} ({}x{}, shell={})",
                        req.session_id, req.cols, req.rows,
                        if req.shell.is_empty() { "auto" } else { &req.shell }
                    );

                    match pty_manager.spawn(req.session_id.clone(), req.cols, req.rows, req.shell).await {
                        Ok(mut rx) => {
                            let session_id = req.session_id.clone();
                            let nc_clone = nc.clone();

                            // Spawn output forwarding task
                            tokio::spawn(async move {
                                while let Some(data) = rx.recv().await {
                                    let encoded = BASE64.encode(&data);
                                    let output = TerminalOutputMsg {
                                        session_id: session_id.clone(),
                                        data: encoded,
                                    };
                                    let subject = format!("terminal.output.{}", session_id);
                                    if let Err(e) = nc_clone.publish(subject, serde_json::to_vec(&output).unwrap().into()).await {
                                        tracing::warn!("failed to publish terminal output: {}", e);
                                        break;
                                    }
                                }
                                tracing::info!("terminal output forwarder ended for session {}", session_id);
                            });

                            tracing::info!("terminal session {} opened successfully", req.session_id);
                        }
                        Err(e) => {
                            tracing::error!("failed to open terminal session {}: {}", req.session_id, e);
                            let err_msg = TerminalErrorMsg {
                                session_id: req.session_id,
                                error: e,
                            };
                            let subject = format!("terminal.error.{}", err_msg.session_id);
                            let _ = nc.publish(subject, serde_json::to_vec(&err_msg).unwrap().into()).await;
                        }
                    }
                }

                Some(msg) = input_sub.next() => {
                    let req: TerminalInputMsg = match serde_json::from_slice(&msg.payload) {
                        Ok(r) => r,
                        Err(e) => {
                            tracing::warn!("failed to deserialize terminal.input: {}", e);
                            continue;
                        }
                    };

                    if let Ok(data) = BASE64.decode(&req.data) {
                        if let Err(e) = pty_manager.write_input(&req.session_id, &data).await {
                            tracing::warn!("failed to write to terminal {}: {}", req.session_id, e);
                        }
                    }
                }

                Some(msg) = resize_sub.next() => {
                    let req: TerminalResizeMsg = match serde_json::from_slice(&msg.payload) {
                        Ok(r) => r,
                        Err(e) => {
                            tracing::warn!("failed to deserialize terminal.resize: {}", e);
                            continue;
                        }
                    };

                    if let Err(e) = pty_manager.resize(&req.session_id, req.cols, req.rows).await {
                        tracing::warn!("failed to resize terminal {}: {}", req.session_id, e);
                    }
                }

                Some(msg) = close_sub.next() => {
                    let req: TerminalCloseMsg = match serde_json::from_slice(&msg.payload) {
                        Ok(r) => r,
                        Err(e) => {
                            tracing::warn!("failed to deserialize terminal.close: {}", e);
                            continue;
                        }
                    };

                    tracing::info!("closing terminal session {}", req.session_id);
                    if let Err(e) = pty_manager.close(&req.session_id).await {
                        tracing::warn!("failed to close terminal {}: {}", req.session_id, e);
                    }
                }

                else => {
                    tracing::warn!("NATS terminal subscriptions ended");
                    break;
                }
            }
        }
    })
}
