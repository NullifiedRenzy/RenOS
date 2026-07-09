use std::process::Command;

use crate::docker;

#[derive(Debug, Clone, serde::Serialize)]
pub struct NetworkInfo {
    pub network_id: String,
    pub name: String,
    pub driver: String,
    pub scope: String,
}

pub fn list_networks() -> Result<Vec<NetworkInfo>, docker::DockerError> {
    let output = Command::new("docker")
        .args([
            "network",
            "ls",
            "--format",
            "{{.ID}}\t{{.Name}}\t{{.Driver}}\t{{.Scope}}",
        ])
        .output()
        .map_err(|e| {
            if e.kind() == std::io::ErrorKind::NotFound {
                docker::DockerError::NotInstalled
            } else {
                docker::DockerError::Execution(e.to_string())
            }
        })?;

    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        return Err(docker::DockerError::Execution(stderr.trim().to_string()));
    }

    let stdout = String::from_utf8_lossy(&output.stdout);
    let mut networks = Vec::new();

    for line in stdout.lines() {
        let line = line.trim();
        if line.is_empty() {
            continue;
        }

        let parts: Vec<&str> = line.splitn(4, '\t').collect();
        if parts.len() < 4 {
            continue;
        }

        networks.push(NetworkInfo {
            network_id: parts[0].to_string(),
            name: parts[1].to_string(),
            driver: parts[2].to_string(),
            scope: parts[3].to_string(),
        });
    }

    Ok(networks)
}

#[derive(serde::Serialize)]
struct SyncRequest<'a> {
    device_id: &'a str,
    device_secret: &'a str,
    networks: &'a [NetworkInfo],
}

pub async fn submit_networks(
    api_url: &str,
    device_id: &str,
    device_secret: &str,
    networks: &[NetworkInfo],
) -> Result<(), String> {
    let client = reqwest::Client::new();
    let url = format!("{}/api/v1/devices/{}/docker-networks", api_url, device_id);

    let resp = client
        .post(&url)
        .json(&SyncRequest { device_id, device_secret, networks })
        .timeout(std::time::Duration::from_secs(10))
        .send()
        .await
        .map_err(|e| format!("sync networks request failed: {e}"))?;

    let status = resp.status();
    if !status.is_success() {
        let body = resp.text().await.unwrap_or_default();
        return Err(format!("sync networks rejected ({}): {}", status, body));
    }

    Ok(())
}
