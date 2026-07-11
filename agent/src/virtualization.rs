use std::process::Command;

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct VmInfo {
    pub domain_name: String,
    pub uuid: String,
    pub state: String,
    pub vcpu_max: i32,
    pub vcpu_current: i32,
    pub memory_max_bytes: i64,
    pub memory_current_bytes: i64,
    pub os_type: String,
    pub autostart: bool,
    pub persistent: bool,
}

#[derive(Debug)]
pub enum VirtError {
    NotAvailable,
    CommandFailed(String),
}

impl std::fmt::Display for VirtError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            VirtError::NotAvailable => write!(f, "virsh not available"),
            VirtError::CommandFailed(msg) => write!(f, "command failed: {msg}"),
        }
    }
}

pub fn is_available() -> bool {
    Command::new("virsh")
        .arg("--version")
        .output()
        .map(|o| o.status.success())
        .unwrap_or(false)
}

fn run_virsh(args: &[&str]) -> Result<String, VirtError> {
    let output = Command::new("virsh")
        .args(args)
        .output()
        .map_err(|e| VirtError::CommandFailed(e.to_string()))?;

    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        return Err(VirtError::CommandFailed(stderr.trim().to_string()));
    }

    Ok(String::from_utf8_lossy(&output.stdout).to_string())
}

fn parse_virsh_list(output: &str) -> Vec<(String, String)> {
    let mut domains = Vec::new();
    for line in output.lines() {
        if line.trim().is_empty() || line.contains("----") || line.starts_with(" Id ") {
            continue;
        }
        // Format: " 1     domain_name   running" or " -     domain_name   shut off"
        let trimmed = line.trim();
        let parts: Vec<&str> = trimmed.split_whitespace().collect();
        if parts.len() >= 3 {
            let (name, state) = if trimmed.ends_with("shut off") {
                (parts[1..parts.len() - 2].join(" "), "shut off".to_string())
            } else {
                (
                    parts[1..parts.len() - 1].join(" "),
                    parts[parts.len() - 1].to_string(),
                )
            };
            domains.push((name, state));
        }
    }
    domains
}

pub fn list_vms() -> Result<Vec<VmInfo>, VirtError> {
    if !is_available() {
        return Err(VirtError::NotAvailable);
    }

    let list_output = run_virsh(&["list", "--all"])?;
    let domains = parse_virsh_list(&list_output);

    let mut vms = Vec::new();
    for (name, state) in domains {
        // Get domain info from dominfo
        let info_output = run_virsh(&["dominfo", &name]).ok();
        let dom_info = parse_dominfo(info_output.as_deref().unwrap_or(""));

        // Get UUID
        let uuid = run_virsh(&["domuuid", &name])
            .ok()
            .map(|u| u.trim().to_string())
            .unwrap_or_default();

        vms.push(VmInfo {
            domain_name: name,
            uuid,
            state,
            vcpu_max: dom_info.0,
            vcpu_current: dom_info.1,
            memory_max_bytes: dom_info.2,
            memory_current_bytes: dom_info.3,
            os_type: dom_info.4,
            autostart: dom_info.5,
            persistent: dom_info.6,
        });
    }

    Ok(vms)
}

fn parse_dominfo(output: &str) -> (i32, i32, i64, i64, String, bool, bool) {
    let mut vcpu_max = 0i32;
    let mut vcpu_current = 0i32;
    let mut max_mem = 0i64;
    let mut used_mem = 0i64;
    let mut os_type = String::new();
    let mut autostart = false;
    let mut persistent = false;

    for line in output.lines() {
        let line = line.trim();
        if let Some(val) = line.strip_prefix("CPU(s):") {
            vcpu_current = val.trim().parse().unwrap_or(0);
        } else if let Some(val) = line.strip_prefix("Max CPU(s):") {
            vcpu_max = val.trim().parse().unwrap_or(vcpu_current);
        } else if let Some(val) = line.strip_prefix("Max memory:") {
            max_mem = parse_kibibytes(val.trim());
        } else if let Some(val) = line.strip_prefix("Used memory:") {
            used_mem = parse_kibibytes(val.trim());
        } else if let Some(val) = line.strip_prefix("OS Type:") {
            os_type = val.trim().to_string();
        } else if line.contains("Autostart:") {
            autostart = line.contains("enable");
        } else if line.contains("Persistent:") {
            persistent = line.contains("yes");
        }
    }

    (
        vcpu_max,
        vcpu_current,
        max_mem,
        used_mem,
        os_type,
        autostart,
        persistent,
    )
}

fn parse_kibibytes(s: &str) -> i64 {
    let s = s.trim();
    if let Some(val) = s.strip_suffix(" KiB") {
        val.trim().parse::<i64>().unwrap_or(0)
    } else if let Some(val) = s.strip_suffix(" MiB") {
        val.trim().parse::<i64>().unwrap_or(0) * 1024
    } else if let Some(val) = s.strip_suffix(" GiB") {
        val.trim().parse::<i64>().unwrap_or(0) * 1024 * 1024
    } else {
        s.parse().unwrap_or(0)
    }
}

#[derive(serde::Serialize)]
struct SyncRequest<'a> {
    device_id: &'a str,
    device_secret: &'a str,
    vms: &'a [VmInfo],
}

pub async fn submit_vms(
    api_url: &str,
    device_id: &str,
    device_secret: &str,
    vms: &[VmInfo],
) -> Result<(), String> {
    let client = reqwest::Client::new();
    let url = format!("{}/api/v1/devices/{}/virtual-machines", api_url, device_id);

    let resp = client
        .post(&url)
        .json(&SyncRequest {
            device_id,
            device_secret,
            vms,
        })
        .timeout(std::time::Duration::from_secs(10))
        .send()
        .await
        .map_err(|e| format!("sync vms request failed: {e}"))?;

    let status = resp.status();
    if !status.is_success() {
        let body = resp.text().await.unwrap_or_default();
        return Err(format!("sync vms rejected ({}): {}", status, body));
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_virsh_list() {
        let sample = " Id   Name   State
---------------------------
 1    ubuntu-vm   running
 -    debian-vm   shut off
 2    fedora-vm   running
";
        let domains = parse_virsh_list(sample);
        assert_eq!(domains.len(), 3);
        assert_eq!(domains[0].0, "ubuntu-vm");
        assert_eq!(domains[0].1, "running");
        assert_eq!(domains[1].0, "debian-vm");
        assert_eq!(domains[1].1, "shut off");
    }

    #[test]
    fn test_parse_virsh_list_empty() {
        let sample = " Id   Name   State
---------------------------
";
        let domains = parse_virsh_list(sample);
        assert!(domains.is_empty());
    }

    #[test]
    fn test_parse_kibibytes() {
        assert_eq!(parse_kibibytes("1048576 KiB"), 1048576);
        assert_eq!(parse_kibibytes("1024 MiB"), 1048576);
        assert_eq!(parse_kibibytes("1 GiB"), 1048576);
        assert_eq!(parse_kibibytes("2097152"), 2097152);
    }

    #[test]
    fn test_parse_dominfo() {
        let sample = "Id:             1
Name:           test-vm
UUID:           abc-123
OS Type:        hvm
State:          running
CPU(s):         2
Max CPU(s):     4
Max memory:     2097152 KiB
Used memory:    1048576 KiB
Persistent:     yes
Autostart:      disable
";
        let (vcpu_max, vcpu_current, max_mem, used_mem, os_type, autostart, persistent) =
            parse_dominfo(sample);
        assert_eq!(vcpu_max, 4);
        assert_eq!(vcpu_current, 2);
        assert_eq!(max_mem, 2097152);
        assert_eq!(used_mem, 1048576);
        assert_eq!(os_type, "hvm");
        assert!(!autostart);
        assert!(persistent);
    }

    #[test]
    fn test_is_available() {
        // Just check it doesn't panic
        let _ = is_available();
    }
}
