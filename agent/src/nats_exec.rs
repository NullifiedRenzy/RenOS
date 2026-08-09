use serde::{Deserialize, Serialize};
use futures::StreamExt;
use tokio::io::AsyncReadExt;

#[derive(Debug, Deserialize)]
pub struct ExecutionJob {
    pub execution_id: String,
    pub command: String,
    pub shell: String,
    pub timeout_ms: u64,
    pub max_output_bytes: i64,
}

#[derive(Debug, Serialize)]
struct OutputMessage {
    execution_id: String,
    stream: String,
    content: String,
    seq: i32,
}

#[derive(Debug, Serialize)]
struct CompleteMessage {
    execution_id: String,
    exit_code: i32,
}

pub fn resolve_shell(shell: &str) -> Vec<String> {
    match shell {
        "powershell" => {
            vec![
                "powershell.exe".to_string(),
                "-NoProfile".to_string(),
                "-NonInteractive".to_string(),
                "-Command".to_string(),
            ]
        }
        "cmd" => {
            vec!["cmd.exe".to_string(), "/C".to_string()]
        }
        "auto" | "bash" => {
            if cfg!(target_os = "windows") {
                vec![
                    "powershell.exe".to_string(),
                    "-NoProfile".to_string(),
                    "-NonInteractive".to_string(),
                    "-Command".to_string(),
                ]
            } else {
                vec![
                    "/bin/bash".to_string(),
                    "-lc".to_string(),
                ]
            }
        }
        _ => {
            if cfg!(target_os = "windows") {
                vec![
                    "powershell.exe".to_string(),
                    "-NoProfile".to_string(),
                    "-NonInteractive".to_string(),
                    "-Command".to_string(),
                ]
            } else {
                vec![
                    "/bin/bash".to_string(),
                    "-lc".to_string(),
                ]
            }
        }
    }
}

pub async fn run_execution(
    nc: &async_nats::Client,
    _device_id: &str,
    job: ExecutionJob,
) {
    tracing::info!(
        "executing job {} (shell={}): {}",
        job.execution_id,
        job.shell,
        job.command
    );

    // Mark as running (not strictly needed, but signals intent)
    let shell_prefix = resolve_shell(&job.shell);

    let mut cmd_parts = shell_prefix;
    cmd_parts.push(job.command.clone());

    let program = cmd_parts[0].clone();
    let args = cmd_parts[1..].to_vec();

    let timeout = std::time::Duration::from_millis(job.timeout_ms);
    let max_output = job.max_output_bytes.max(0) as usize;

    let result = tokio::time::timeout(timeout, async {
        let mut child = tokio::process::Command::new(&program)
            .args(&args)
            .stdout(std::process::Stdio::piped())
            .stderr(std::process::Stdio::piped())
            .kill_on_drop(true)
            .spawn()
            .map_err(|e| format!("spawn failed: {e}"))?;

        let mut stdout = child.stdout.take().unwrap();
        let mut stderr = child.stderr.take().unwrap();

        let mut stdout_buf = Vec::new();
        let mut stderr_buf = Vec::new();

        // Read stdout and stderr concurrently
        let mut seq = 0;
        let mut total_bytes = 0usize;

        loop {
            let mut stdout_buf_read = [0u8; 8192];
            let mut stderr_buf_read = [0u8; 8192];
            tokio::select! {
                n = stdout.read(&mut stdout_buf_read) => {
                    match n {
                        Ok(0) => break,
                        Ok(n) => {
                            let chunk = String::from_utf8_lossy(&stdout_buf_read[..n]).to_string();
                            total_bytes += chunk.len();
                            if max_output > 0 && total_bytes > max_output {
                                let msg = OutputMessage {
                                    execution_id: job.execution_id.clone(),
                                    stream: "stdout".to_string(),
                                    content: "\n[output truncated - max size exceeded]\n".to_string(),
                                    seq,
                                };
                                let _ = nc.publish(
                                    format!("exec.output.{}", job.execution_id),
                                    serde_json::to_vec(&msg).unwrap().into(),
                                ).await;
                                break;
                            }
                            stdout_buf.extend_from_slice(&stdout_buf_read[..n]);
                            seq += 1;
                            let msg = OutputMessage {
                                execution_id: job.execution_id.clone(),
                                stream: "stdout".to_string(),
                                content: chunk,
                                seq,
                            };
                            let _ = nc.publish(
                                format!("exec.output.{}", job.execution_id),
                                serde_json::to_vec(&msg).unwrap().into(),
                            ).await;
                        }
                        Err(_) => break,
                    }
                }
                n = stderr.read(&mut stderr_buf_read) => {
                    match n {
                        Ok(0) => {},
                        Ok(n) => {
                            let chunk = String::from_utf8_lossy(&stderr_buf_read[..n]).to_string();
                            total_bytes += chunk.len();
                            if max_output > 0 && total_bytes > max_output {
                                break;
                            }
                            stderr_buf.extend_from_slice(&stderr_buf_read[..n]);
                            seq += 1;
                            let msg = OutputMessage {
                                execution_id: job.execution_id.clone(),
                                stream: "stderr".to_string(),
                                content: chunk,
                                seq,
                            };
                            let _ = nc.publish(
                                format!("exec.output.{}", job.execution_id),
                                serde_json::to_vec(&msg).unwrap().into(),
                            ).await;
                        }
                        Err(_) => {},
                    }
                }
            }
        }

        match child.wait().await {
            Ok(status) => {
                let exit_code = status.code().unwrap_or(1);
                Ok::<i32, String>(exit_code)
            }
            Err(e) => Err(format!("wait failed: {e}")),
        }
    })
    .await;

    let exit_code = match result {
        Ok(Ok(code)) => code,
        Ok(Err(e)) => {
            tracing::error!("execution {} failed: {}", job.execution_id, e);
            1
        }
        Err(_) => {
            tracing::warn!("execution {} timed out after {}ms", job.execution_id, job.timeout_ms);
            124 // timeout exit code
        }
    };

    let complete_msg = CompleteMessage {
        execution_id: job.execution_id.clone(),
        exit_code,
    };

    let _ = nc.publish(
        format!("exec.complete.{}", job.execution_id),
        serde_json::to_vec(&complete_msg).unwrap().into(),
    ).await;

    tracing::info!(
        "execution {} completed (exit={})",
        job.execution_id,
        exit_code
    );
}

pub fn start_nats_executor(
    nc: async_nats::Client,
    device_id: String,
) -> tokio::task::JoinHandle<()> {
    tokio::spawn(async move {
        let subject = format!("exec.run.{}", device_id);
        tracing::info!("subscribing to NATS subject: {}", subject);

        let mut subscriber = match nc.subscribe(subject.clone()).await {
            Ok(sub) => sub,
            Err(e) => {
                tracing::error!("failed to subscribe to {}: {}", subject, e);
                return;
            }
        };

        tracing::info!("NATS executor subscribed to {}", subject);

        while let Some(msg) = subscriber.next().await {
            let job: ExecutionJob = match serde_json::from_slice(&msg.payload) {
                Ok(j) => j,
                Err(e) => {
                    tracing::warn!("failed to deserialize execution job: {}", e);
                    continue;
                }
            };

            let nc_clone = nc.clone();
            let device_id_clone = device_id.clone();
            tokio::spawn(async move {
                run_execution(&nc_clone, &device_id_clone, job).await;
            });
        }

        tracing::warn!("NATS executor subscription ended for {}", subject);
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_resolve_shell_auto_unix() {
        if cfg!(not(target_os = "windows")) {
            let parts = resolve_shell("auto");
            assert_eq!(parts[0], "/bin/bash");
            assert_eq!(parts[1], "-lc");
        }
    }

    #[test]
    fn test_resolve_shell_powershell() {
        let parts = resolve_shell("powershell");
        assert_eq!(parts[0], "powershell.exe");
        assert!(parts.contains(&"-NoProfile".to_string()));
    }

    #[test]
    fn test_resolve_shell_bash_explicit() {
        if cfg!(not(target_os = "windows")) {
            let parts = resolve_shell("bash");
            assert_eq!(parts[0], "/bin/bash");
        }
    }

    #[test]
    fn test_resolve_shell_cmd() {
        let parts = resolve_shell("cmd");
        assert_eq!(parts[0], "cmd.exe");
        assert!(parts.contains(&"/C".to_string()));
    }
}
