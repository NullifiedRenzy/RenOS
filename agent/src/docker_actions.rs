use std::process::Command;

use crate::docker;
use crate::retry;

#[derive(Debug, serde::Deserialize)]
struct PendingActionsResponse {
    container_actions: Vec<PendingAction>,
}

#[derive(Debug, serde::Deserialize)]
pub struct PendingAction {
    pub id: String,
    pub container_id: String,
    pub action: String,
}

#[derive(Debug, serde::Serialize)]
struct PendingRequestBody<'a> {
    device_id: &'a str,
    device_secret: &'a str,
}

#[derive(Debug, serde::Serialize)]
struct SubmitBody<'a> {
    device_id: &'a str,
    device_secret: &'a str,
    exit_code: i32,
    output: &'a str,
    error: &'a str,
}

#[derive(Debug)]
pub enum Error {
    Http(String),
    Rejected(String),
}

impl std::fmt::Display for Error {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Error::Http(msg) => write!(f, "http error: {msg}"),
            Error::Rejected(msg) => write!(f, "rejected by core: {msg}"),
        }
    }
}

pub async fn poll(
    api_url: &str,
    device_id: &str,
    device_secret: &str,
) -> Result<Vec<PendingAction>, Error> {
    let client = reqwest::Client::new();
    let url = format!("{}/api/v1/container-actions/pending", api_url);

    let resp = client
        .post(&url)
        .json(&PendingRequestBody {
            device_id,
            device_secret,
        })
        .timeout(std::time::Duration::from_secs(10))
        .send()
        .await
        .map_err(|e| Error::Http(format!("poll request failed: {e}")))?;

    let status = resp.status();
    if !status.is_success() {
        let body = resp.text().await.unwrap_or_default();
        return Err(Error::Rejected(format!(
            "poll rejected ({}): {}",
            status, body
        )));
    }

    let data: PendingActionsResponse = resp
        .json()
        .await
        .map_err(|e| Error::Http(format!("poll parse failed: {e}")))?;

    Ok(data.container_actions)
}

pub async fn submit_result(
    api_url: &str,
    device_id: &str,
    device_secret: &str,
    action_id: &str,
    exit_code: i32,
    output: &str,
    error: &str,
) -> Result<(), Error> {
    let url = format!("{}/api/v1/container-actions/{}/result", api_url, action_id);
    let mut backoff = retry::Backoff::new(
        std::time::Duration::from_millis(500),
        std::time::Duration::from_secs(30),
    );
    let mut last_err = None;

    for attempt in 1..=3 {
        let client = reqwest::Client::new();
        let resp = client
            .post(&url)
            .json(&SubmitBody {
                device_id,
                device_secret,
                exit_code,
                output,
                error,
            })
            .timeout(std::time::Duration::from_secs(10))
            .send()
            .await;

        match resp {
            Ok(r) if r.status().is_success() => return Ok(()),
            Ok(r) => {
                let status = r.status();
                let body = r.text().await.unwrap_or_default();
                last_err = Some(Error::Rejected(format!("submit rejected ({status}): {body}")));
            }
            Err(e) => {
                last_err = Some(Error::Http(format!("submit failed: {e}")));
            }
        }

        tracing::warn!("attempt {}/3 failed", attempt);
        if attempt < 3 {
            let delay = backoff.next_with_jitter();
            tracing::info!("retrying in {}ms", delay.as_millis());
            tokio::time::sleep(delay).await;
        }
    }

    Err(last_err.unwrap_or_else(|| Error::Http("submit failed".to_string())))
}

fn run_docker_command(args: &[&str]) -> (i32, String, String) {
    match Command::new("docker").args(args).output() {
        Ok(output) => {
            let exit_code = output.status.code().unwrap_or(1);
            let stdout = String::from_utf8_lossy(&output.stdout).trim().to_string();
            let stderr = String::from_utf8_lossy(&output.stderr).trim().to_string();
            (exit_code, stdout, stderr)
        }
        Err(e) => (1, String::new(), format!("failed to execute docker: {e}")),
    }
}

pub fn execute_action(action: &str, container_id: &str) -> (i32, String, String) {
    if !docker::is_installed() {
        return (
            1,
            String::new(),
            "docker not installed on this device".to_string(),
        );
    }

    match action {
        "start" => run_docker_command(&["start", container_id]),
        "stop" => run_docker_command(&["stop", container_id]),
        "restart" => run_docker_command(&["restart", container_id]),
        "kill" => run_docker_command(&["kill", container_id]),
        "remove" => run_docker_command(&["rm", "-f", container_id]),
        "pause" => run_docker_command(&["pause", container_id]),
        "unpause" => run_docker_command(&["unpause", container_id]),
        "inspect" => run_docker_command(&["inspect", container_id]),
        "logs" => run_docker_command(&["logs", "--tail", "200", container_id]),
        "stats" => run_docker_command(&["stats", "--no-stream", "--no-trunc", container_id]),
        "image_pull" => run_docker_command(&["pull", container_id]),
        "image_remove" => run_docker_command(&["rmi", "-f", container_id]),
        "image_prune" => run_docker_command(&["image", "prune", "-a", "-f"]),
        "volume_remove" => run_docker_command(&["volume", "rm", container_id]),
        "volume_prune" => run_docker_command(&["volume", "prune", "-f"]),
        "network_disconnect" => {
            let parts: Vec<&str> = container_id.splitn(2, ':').collect();
            if parts.len() == 2 {
                run_docker_command(&["network", "disconnect", "-f", parts[0], parts[1]])
            } else {
                (1, String::new(), "network_disconnect requires format 'network_id:container_id'".to_string())
            }
        }
        "network_connect" => {
            let parts: Vec<&str> = container_id.splitn(2, ':').collect();
            if parts.len() == 2 {
                run_docker_command(&["network", "connect", parts[0], parts[1]])
            } else {
                (1, String::new(), "network_connect requires format 'network_id:container_id'".to_string())
            }
        }
        "system_prune" => run_docker_command(&["system", "prune", "-a", "-f"]),
        _ => (1, String::new(), format!("unknown action: {action}")),
    }
}
