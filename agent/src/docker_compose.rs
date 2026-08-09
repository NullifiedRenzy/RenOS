use std::process::Command;

use serde::Serialize;

#[derive(Debug, Serialize, Clone)]
pub struct ComposeStack {
    pub name: String,
    pub status: String,
    pub config_files: String,
    pub services: String,
    pub working_dir: String,
}

pub fn is_compose_available() -> bool {
    Command::new("docker")
        .args(["compose", "version"])
        .output()
        .map(|o| o.status.success())
        .unwrap_or(false)
}

pub fn list_compose_stacks() -> Result<Vec<ComposeStack>, String> {
    let output = Command::new("docker")
        .args(["compose", "ls", "--format", "json"])
        .output()
        .map_err(|e| format!("failed to run docker compose ls: {e}"))?;

    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        return Err(format!("docker compose ls failed: {}", stderr.trim()));
    }

    let stdout = String::from_utf8_lossy(&output.stdout);
    if stdout.trim().is_empty() {
        return Ok(Vec::new());
    }

    let raw: Vec<serde_json::Value> = serde_json::from_str(stdout.trim())
        .map_err(|e| format!("failed to parse docker compose ls output: {e}"))?;

    let mut stacks = Vec::new();
    for entry in raw {
        let name = entry.get("Name")
            .and_then(|v| v.as_str())
            .unwrap_or("")
            .to_string();
        let status = entry.get("Status")
            .and_then(|v| v.as_str())
            .unwrap_or("")
            .to_string();
        let config_files = entry.get("ConfigFiles")
            .and_then(|v| v.as_str())
            .unwrap_or("")
            .to_string();
        let working_dir = entry.get("WorkingDir")
            .and_then(|v| v.as_str())
            .unwrap_or("")
            .to_string();

        // Get services for this compose project
        let services = get_compose_services(&working_dir);

        stacks.push(ComposeStack {
            name,
            status,
            config_files,
            services,
            working_dir,
        });
    }

    Ok(stacks)
}

fn get_compose_services(working_dir: &str) -> String {
    if working_dir.is_empty() {
        return "[]".to_string();
    }

    let output = Command::new("docker")
        .args(["compose", "ps", "--format", "json"])
        .current_dir(working_dir)
        .output();

    match output {
        Ok(o) if o.status.success() => {
            let stdout = String::from_utf8_lossy(&o.stdout);
            if stdout.trim().is_empty() {
                return "[]".to_string();
            }

            let raw: Vec<serde_json::Value> = match serde_json::from_str(stdout.trim()) {
                Ok(v) => v,
                Err(_) => return "[]".to_string(),
            };

            let services: Vec<serde_json::Value> = raw.iter().filter_map(|entry| {
                let name = entry.get("Name")?.as_str()?.to_string();
                let state = entry.get("State")?.as_str()?.to_string();
                let health = entry.get("Health")?.as_str().unwrap_or("").to_string();
                Some(serde_json::json!({
                    "name": name,
                    "state": state,
                    "health": health,
                }))
            }).collect();

            serde_json::to_string(&services).unwrap_or_else(|_| "[]".to_string())
        }
        _ => "[]".to_string(),
    }
}

pub async fn submit_compose_stacks(
    api_url: &str,
    device_id: &str,
    device_secret: &str,
    stacks: &[ComposeStack],
) -> Result<(), String> {
    let client = reqwest::Client::new();
    let url = format!("{}/api/v1/devices/{}/compose-stacks", api_url, device_id);

    #[derive(serde::Serialize)]
    struct SyncRequest<'a> {
        device_id: &'a str,
        device_secret: &'a str,
        stacks: &'a [ComposeStack],
    }

    let resp = client
        .post(&url)
        .json(&SyncRequest {
            device_id,
            device_secret,
            stacks,
        })
        .timeout(std::time::Duration::from_secs(10))
        .send()
        .await
        .map_err(|e| format!("sync compose stacks request failed: {e}"))?;

    let status = resp.status();
    if !status.is_success() {
        let body = resp.text().await.unwrap_or_default();
        return Err(format!("sync compose stacks rejected ({}): {}", status, body));
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_is_compose_available() {
        // Just ensure it doesn't panic
        let _ = is_compose_available();
    }
}
