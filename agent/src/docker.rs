use std::process::Command;

#[derive(Debug, Clone, serde::Serialize)]
pub struct ContainerInfo {
    pub container_id: String,
    pub name: String,
    pub image: String,
    pub status: String,
    pub state: String,
    pub ports: String,
}

#[derive(Debug)]
pub enum DockerError {
    NotInstalled,
    Execution(String),
}

impl std::fmt::Display for DockerError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            DockerError::NotInstalled => write!(f, "docker not installed"),
            DockerError::Execution(msg) => write!(f, "execution error: {msg}"),
        }
    }
}

pub fn is_installed() -> bool {
    Command::new("docker")
        .arg("version")
        .output()
        .map(|o| o.status.success())
        .unwrap_or(false)
}

pub fn list_containers() -> Result<Vec<ContainerInfo>, DockerError> {
    let output = Command::new("docker")
        .args([
            "ps", "-a",
            "--no-trunc",
            "--format",
            "{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}\t{{.State}}\t{{.Ports}}",
        ])
        .output()
        .map_err(|e| {
            if e.kind() == std::io::ErrorKind::NotFound {
                DockerError::NotInstalled
            } else {
                DockerError::Execution(e.to_string())
            }
        })?;

    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        return Err(DockerError::Execution(stderr.trim().to_string()));
    }

    let stdout = String::from_utf8_lossy(&output.stdout);
    let mut containers = Vec::new();

    for line in stdout.lines() {
        let line = line.trim();
        if line.is_empty() {
            continue;
        }

        let parts: Vec<&str> = line.splitn(6, '\t').collect();
        if parts.len() < 6 {
            continue;
        }

        containers.push(ContainerInfo {
            container_id: parts[0].to_string(),
            name: parts[1].to_string(),
            image: parts[2].to_string(),
            status: parts[3].to_string(),
            state: parts[4].to_string(),
            ports: parts[5].to_string(),
        });
    }

    Ok(containers)
}

#[derive(serde::Serialize)]
struct SyncRequest<'a> {
    device_id: &'a str,
    device_secret: &'a str,
    containers: &'a [ContainerInfo],
}

pub async fn submit_containers(
    api_url: &str,
    device_id: &str,
    device_secret: &str,
    containers: &[ContainerInfo],
) -> Result<(), String> {
    let client = reqwest::Client::new();
    let url = format!("{}/api/v1/devices/{}/containers", api_url, device_id);

    let resp = client
        .post(&url)
        .json(&SyncRequest { device_id, device_secret, containers })
        .timeout(std::time::Duration::from_secs(10))
        .send()
        .await
        .map_err(|e| format!("sync request failed: {e}"))?;

    let status = resp.status();
    if !status.is_success() {
        let body = resp.text().await.unwrap_or_default();
        return Err(format!("sync rejected ({}): {}", status, body));
    }

    Ok(())
}
