use std::process::Command;

use crate::docker;

#[derive(Debug, Clone, serde::Serialize)]
pub struct ImageInfo {
    pub image_id: String,
    pub repository: String,
    pub tag: String,
    pub size: String,
    pub created: String,
}

pub fn list_images() -> Result<Vec<ImageInfo>, docker::DockerError> {
    let output = Command::new("docker")
        .args([
            "images",
            "--no-trunc",
            "--format",
            "{{.ID}}\t{{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}",
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
    let mut images = Vec::new();

    for line in stdout.lines() {
        let line = line.trim();
        if line.is_empty() {
            continue;
        }

        let parts: Vec<&str> = line.splitn(5, '\t').collect();
        if parts.len() < 5 {
            continue;
        }

        images.push(ImageInfo {
            image_id: parts[0].to_string(),
            repository: parts[1].to_string(),
            tag: parts[2].to_string(),
            size: parts[3].to_string(),
            created: parts[4].to_string(),
        });
    }

    Ok(images)
}

#[derive(serde::Serialize)]
struct SyncRequest<'a> {
    device_id: &'a str,
    device_secret: &'a str,
    images: &'a [ImageInfo],
}

pub async fn submit_images(
    api_url: &str,
    device_id: &str,
    device_secret: &str,
    images: &[ImageInfo],
) -> Result<(), String> {
    let client = reqwest::Client::new();
    let url = format!("{}/api/v1/devices/{}/docker-images", api_url, device_id);

    let resp = client
        .post(&url)
        .json(&SyncRequest {
            device_id,
            device_secret,
            images,
        })
        .timeout(std::time::Duration::from_secs(10))
        .send()
        .await
        .map_err(|e| format!("sync images request failed: {e}"))?;

    let status = resp.status();
    if !status.is_success() {
        let body = resp.text().await.unwrap_or_default();
        return Err(format!("sync images rejected ({}): {}", status, body));
    }

    Ok(())
}
