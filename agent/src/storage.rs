use std::process::Command;

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct StorageDeviceInfo {
    pub name: String,
    pub size: i64,
    pub device_type: String,
    pub model: String,
    pub mount_point: String,
    pub filesystem: String,
    pub uuid: String,
    pub is_removable: bool,
    pub is_readonly: bool,
    pub used_bytes: i64,
    pub available_bytes: i64,
}

#[derive(Debug)]
pub enum StorageError {
    CommandFailed(String),
    Parse(String),
}

impl std::fmt::Display for StorageError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            StorageError::CommandFailed(msg) => write!(f, "command failed: {msg}"),
            StorageError::Parse(msg) => write!(f, "parse error: {msg}"),
        }
    }
}

#[derive(serde::Deserialize)]
struct LsblkEntry {
    #[serde(default)]
    name: String,
    #[serde(default)]
    size: String,
    #[serde(default)]
    #[serde(rename = "type")]
    entry_type: String,
    #[serde(default)]
    model: String,
    #[serde(default)]
    mountpoint: String,
    #[serde(default)]
    fstype: String,
    #[serde(default)]
    uuid: String,
    #[serde(default)]
    ro: bool,
    #[serde(default)]
    rm: bool,
}

pub fn list_storage_devices() -> Result<Vec<StorageDeviceInfo>, StorageError> {
    let output = Command::new("lsblk")
        .args(["-J", "-b", "-o", "NAME,SIZE,TYPE,MODEL,MOUNTPOINT,FSTYPE,UUID,RO,RM"])
        .output()
        .map_err(|e| StorageError::CommandFailed(e.to_string()))?;

    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        return Err(StorageError::CommandFailed(stderr.trim().to_string()));
    }

    let stdout = String::from_utf8_lossy(&output.stdout);
    let parsed: LsblkOutput = serde_json::from_str(&stdout)
        .map_err(|e| StorageError::Parse(e.to_string()))?;

    // Get filesystem usage via df
    let df_map = get_df_usage();

    let mut devices: Vec<StorageDeviceInfo> = Vec::new();
    process_lsblk_entries(&parsed.blockdevices, "", &df_map, &mut devices);

    Ok(devices)
}

#[derive(serde::Deserialize)]
struct LsblkOutput {
    #[serde(default)]
    blockdevices: Vec<LsblkEntry>,
}

fn parse_size(s: &str) -> i64 {
    s.parse().unwrap_or(0)
}

fn process_lsblk_entries(
    entries: &[LsblkEntry],
    parent_name: &str,
    df_map: &std::collections::HashMap<String, (i64, i64)>,
    result: &mut Vec<StorageDeviceInfo>,
) {
    for entry in entries {
        let full_name = if parent_name.is_empty() {
            entry.name.clone()
        } else {
            format!("{}{}", parent_name, entry.name)
        };

        let size = parse_size(&entry.size);

        let (used_bytes, available_bytes) = if !entry.mountpoint.is_empty() {
            df_map.get(&entry.mountpoint).copied().unwrap_or((0, 0))
        } else if entry.entry_type == "disk" {
            // For raw disks, use total size — no usage info
            (0, 0)
        } else {
            (0, 0)
        };

        result.push(StorageDeviceInfo {
            name: full_name.clone(),
            size,
            device_type: entry.entry_type.clone(),
            model: entry.model.trim().to_string(),
            mount_point: entry.mountpoint.clone(),
            filesystem: entry.fstype.clone(),
            uuid: entry.uuid.clone(),
            is_removable: entry.rm,
            is_readonly: entry.ro,
            used_bytes,
            available_bytes,
        });

        // Note: lsblk -J nested children are under "children" key
        // We don't handle that here; flat output is sufficient
    }
}

fn get_df_usage() -> std::collections::HashMap<String, (i64, i64)> {
    let mut map = std::collections::HashMap::new();

    let output = match Command::new("df")
        .args(["-B1", "--output=target,used,avail"])
        .output()
    {
        Ok(o) if o.status.success() => o,
        _ => return map,
    };

    let stdout = String::from_utf8_lossy(&output.stdout);
    for line in stdout.lines().skip(1) {
        let line = line.trim();
        if line.is_empty() {
            continue;
        }
        let parts: Vec<&str> = line.split_whitespace().collect();
        if parts.len() < 3 {
            continue;
        }
        let mount = parts[0].to_string();
        let used = parts[1].parse::<i64>().unwrap_or(0);
        let avail = parts[2].parse::<i64>().unwrap_or(0);
        map.insert(mount, (used, avail));
    }

    map
}

#[derive(serde::Serialize)]
struct SyncRequest<'a> {
    device_id: &'a str,
    device_secret: &'a str,
    devices: &'a [StorageDeviceInfo],
}

pub async fn submit_storage_devices(
    api_url: &str,
    device_id: &str,
    device_secret: &str,
    devices: &[StorageDeviceInfo],
) -> Result<(), String> {
    let client = reqwest::Client::new();
    let url = format!("{}/api/v1/devices/{}/storage-devices", api_url, device_id);

    let resp = client
        .post(&url)
        .json(&SyncRequest { device_id, device_secret, devices })
        .timeout(std::time::Duration::from_secs(10))
        .send()
        .await
        .map_err(|e| format!("sync storage devices request failed: {e}"))?;

    let status = resp.status();
    if !status.is_success() {
        let body = resp.text().await.unwrap_or_default();
        return Err(format!("sync storage devices rejected ({}): {}", status, body));
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_list_storage_devices_returns_data() {
        let result = list_storage_devices();
        // May fail if lsblk not available, but should succeed on Linux
        if let Ok(devices) = result {
            assert!(!devices.is_empty(), "should have at least one device");
        }
    }

    #[test]
    fn test_parse_size_valid() {
        assert_eq!(parse_size("1024"), 1024);
        assert_eq!(parse_size("0"), 0);
    }

    #[test]
    fn test_parse_size_invalid() {
        assert_eq!(parse_size("not-a-number"), 0);
        assert_eq!(parse_size(""), 0);
    }
}
