use std::process::Command;

use crate::docker;

#[derive(Debug, Clone, serde::Serialize)]
pub struct VolumeInfo {
    pub name: String,
    pub driver: String,
    pub mountpoint: String,
}

pub fn list_volumes() -> Result<Vec<VolumeInfo>, docker::DockerError> {
    let output = Command::new("docker")
        .args([
            "volume",
            "ls",
            "--format",
            "{{.Name}}\t{{.Driver}}\t{{.Mountpoint}}",
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
    let mut volumes = Vec::new();

    for line in stdout.lines() {
        let line = line.trim();
        if line.is_empty() {
            continue;
        }

        let parts: Vec<&str> = line.splitn(3, '\t').collect();
        if parts.len() < 3 {
            continue;
        }

        volumes.push(VolumeInfo {
            name: parts[0].to_string(),
            driver: parts[1].to_string(),
            mountpoint: parts[2].to_string(),
        });
    }

    Ok(volumes)
}

#[derive(serde::Serialize)]
struct SyncRequest<'a> {
    device_id: &'a str,
    device_secret: &'a str,
    volumes: &'a [VolumeInfo],
}

pub async fn submit_volumes(
    api_url: &str,
    device_id: &str,
    device_secret: &str,
    volumes: &[VolumeInfo],
) -> Result<(), String> {
    let client = reqwest::Client::new();
    let url = format!("{}/api/v1/devices/{}/docker-volumes", api_url, device_id);

    let resp = client
        .post(&url)
        .json(&SyncRequest { device_id, device_secret, volumes })
        .timeout(std::time::Duration::from_secs(10))
        .send()
        .await
        .map_err(|e| format!("sync volumes request failed: {e}"))?;

    let status = resp.status();
    if !status.is_success() {
        let body = resp.text().await.unwrap_or_default();
        return Err(format!("sync volumes rejected ({}): {}", status, body));
    }

    Ok(())
}
