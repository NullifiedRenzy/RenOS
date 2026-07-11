use std::collections::HashMap;
use std::process::Command;

use crate::systemd;

#[derive(Debug, Clone, serde::Serialize)]
pub struct ServiceInfo {
    pub unit_name: String,
    pub load_state: String,
    pub active_state: String,
    pub sub_state: String,
    pub description: String,
    pub enabled: String,
}

#[derive(serde::Deserialize)]
struct SystemdUnitJson {
    #[serde(default)]
    unit: String,
    #[serde(default)]
    load: String,
    #[serde(default)]
    active: String,
    #[serde(default)]
    sub: String,
    #[serde(default)]
    description: String,
}

#[derive(serde::Deserialize)]
struct UnitFileJson {
    #[serde(default)]
    unit_file: String,
    #[serde(default)]
    state: String,
}

pub fn list_services() -> Result<Vec<ServiceInfo>, systemd::SystemdError> {
    let units_json = run_systemctl_json(&[
        "list-units",
        "--type=service",
        "--all",
        "--no-legend",
        "--no-pager",
        "--output=json",
    ])?;

    let units: Vec<SystemdUnitJson> = serde_json::from_str(&units_json)
        .unwrap_or_default();

    let unit_files_json = run_systemctl_json(&[
        "list-unit-files",
        "--type=service",
        "--no-legend",
        "--no-pager",
        "--output=json",
    ])?;

    let unit_files: Vec<UnitFileJson> = serde_json::from_str(&unit_files_json)
        .unwrap_or_default();

    let enabled_map: HashMap<String, String> = unit_files
        .into_iter()
        .map(|f| (f.unit_file.clone(), f.state))
        .collect();

    let mut services = Vec::new();
    for u in units {
        let enabled = enabled_map
            .get(&u.unit)
            .cloned()
            .unwrap_or_default();

        services.push(ServiceInfo {
            unit_name: u.unit,
            load_state: u.load,
            active_state: u.active,
            sub_state: u.sub,
            description: u.description,
            enabled,
        });
    }

    Ok(services)
}

fn run_systemctl_json(args: &[&str]) -> Result<String, systemd::SystemdError> {
    let output = Command::new("systemctl")
        .args(args)
        .output()
        .map_err(|e| {
            if e.kind() == std::io::ErrorKind::NotFound {
                systemd::SystemdError::NotInstalled
            } else {
                systemd::SystemdError::Execution(e.to_string())
            }
        })?;

    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        return Err(systemd::SystemdError::Execution(stderr.trim().to_string()));
    }

    Ok(String::from_utf8_lossy(&output.stdout).to_string())
}

#[derive(serde::Serialize)]
struct SyncRequest<'a> {
    device_id: &'a str,
    device_secret: &'a str,
    services: &'a [ServiceInfo],
}

pub async fn submit_services(
    api_url: &str,
    device_id: &str,
    device_secret: &str,
    services: &[ServiceInfo],
) -> Result<(), String> {
    let client = reqwest::Client::new();
    let url = format!("{}/api/v1/devices/{}/systemd-services", api_url, device_id);

    let resp = client
        .post(&url)
        .json(&SyncRequest { device_id, device_secret, services })
        .timeout(std::time::Duration::from_secs(10))
        .send()
        .await
        .map_err(|e| format!("sync systemd services request failed: {e}"))?;

    let status = resp.status();
    if !status.is_success() {
        let body = resp.text().await.unwrap_or_default();
        return Err(format!("sync systemd services rejected ({}): {}", status, body));
    }

    Ok(())
}
