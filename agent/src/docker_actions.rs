use std::process::Command;

use crate::docker;

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
        .json(&PendingRequestBody { device_id, device_secret })
        .timeout(std::time::Duration::from_secs(10))
        .send()
        .await
        .map_err(|e| Error::Http(format!("poll request failed: {e}")))?;

    let status = resp.status();
    if !status.is_success() {
        let body = resp.text().await.unwrap_or_default();
        return Err(Error::Rejected(format!("poll rejected ({}): {}", status, body)));
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
    let client = reqwest::Client::new();
    let url = format!("{}/api/v1/container-actions/{}/result", api_url, action_id);

    let resp = client
        .post(&url)
        .json(&SubmitBody { device_id, device_secret, exit_code, output, error })
        .timeout(std::time::Duration::from_secs(10))
        .send()
        .await
        .map_err(|e| Error::Http(format!("submit failed: {e}")))?;

    let status = resp.status();
    if !status.is_success() {
        let body = resp.text().await.unwrap_or_default();
        return Err(Error::Rejected(format!("submit rejected ({}): {}", status, body)));
    }

    Ok(())
}

pub fn execute_action(action: &str, container_id: &str) -> (i32, String, String) {
    if !docker::is_installed() {
        return (1, String::new(), "docker not installed on this device".to_string());
    }

    let result = match action {
        "start" => Command::new("docker")
            .args(["start", container_id])
            .output(),
        "stop" => Command::new("docker")
            .args(["stop", container_id])
            .output(),
        "restart" => Command::new("docker")
            .args(["restart", container_id])
            .output(),
        _ => return (1, String::new(), format!("unknown action: {action}")),
    };

    match result {
        Ok(output) => {
            let exit_code = output.status.code().unwrap_or(1);
            let stdout = String::from_utf8_lossy(&output.stdout).trim().to_string();
            let stderr = String::from_utf8_lossy(&output.stderr).trim().to_string();
            (exit_code, stdout, stderr)
        }
        Err(e) => (1, String::new(), format!("failed to execute docker {action}: {e}")),
    }
}
