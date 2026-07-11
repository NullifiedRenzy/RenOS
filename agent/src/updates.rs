use std::process::Command;

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct PackageUpdate {
    pub package_name: String,
    pub current_version: String,
    pub available_version: String,
    pub severity: String,
    pub status: String,
}

#[derive(Debug)]
pub enum UpdateError {
    NotAvailable,
    CommandFailed(String),
}

impl std::fmt::Display for UpdateError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            UpdateError::NotAvailable => write!(f, "apt not available"),
            UpdateError::CommandFailed(msg) => write!(f, "command failed: {msg}"),
        }
    }
}

pub fn is_available() -> bool {
    Command::new("apt")
        .arg("--version")
        .output()
        .map(|o| o.status.success())
        .unwrap_or(false)
}

fn run_apt(args: &[&str]) -> Result<String, UpdateError> {
    let output = Command::new("apt")
        .args(args)
        .output()
        .map_err(|e| UpdateError::CommandFailed(e.to_string()))?;

    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        return Err(UpdateError::CommandFailed(stderr.trim().to_string()));
    }

    Ok(String::from_utf8_lossy(&output.stdout).to_string())
}

pub fn list_updates() -> Result<Vec<PackageUpdate>, UpdateError> {
    if !is_available() {
        return Err(UpdateError::NotAvailable);
    }

    // First update the package list
    let _ = run_apt(&["update", "-qq"]);

    // List upgradable packages
    let output = run_apt(&["list", "--upgradable"])?;

    let mut updates = Vec::new();
    for line in output.lines() {
        let line = line.trim();
        if line.is_empty() || line.starts_with("Listing") || !line.contains('/') {
            continue;
        }
        // Format: "package-name/stable 1.2.3 amd64 [upgradable from: 1.2.2]"
        if let Some(pkg_part) = line.split_whitespace().next() {
            let parts: Vec<&str> = pkg_part.split('/').collect();
            let package_name = parts[0].to_string();

            let mut current = String::new();
            let mut available = String::new();

            // Parse version info from the line
            let tokens: Vec<&str> = line.split_whitespace().collect();
            if tokens.len() >= 3 {
                available = tokens[1].to_string();
            }

            // Extract "upgradable from: X.Y.Z" from brackets
            if let Some(bracket_start) = line.find('[') {
                if let Some(bracket_end) = line.find(']') {
                    let inside = &line[bracket_start + 1..bracket_end];
                    if let Some(from) = inside.strip_prefix("upgradable from: ") {
                        current = from.trim().to_string();
                    }
                }
            }

            let severity = if line.contains("security") || line.contains("Security") {
                "security".to_string()
            } else {
                "standard".to_string()
            };

            updates.push(PackageUpdate {
                package_name,
                current_version: current,
                available_version: available,
                severity,
                status: "pending".to_string(),
            });
        }
    }

    Ok(updates)
}

#[derive(serde::Serialize)]
struct SyncRequest<'a> {
    device_id: &'a str,
    device_secret: &'a str,
    updates: &'a [PackageUpdate],
}

pub async fn submit_updates(
    api_url: &str,
    device_id: &str,
    device_secret: &str,
    updates: &[PackageUpdate],
) -> Result<(), String> {
    let client = reqwest::Client::new();
    let url = format!("{}/api/v1/devices/{}/updates", api_url, device_id);

    let resp = client
        .post(&url)
        .json(&SyncRequest {
            device_id,
            device_secret,
            updates,
        })
        .timeout(std::time::Duration::from_secs(10))
        .send()
        .await
        .map_err(|e| format!("sync updates request failed: {e}"))?;

    let status = resp.status();
    if !status.is_success() {
        let body = resp.text().await.unwrap_or_default();
        return Err(format!("sync updates rejected ({}): {}", status, body));
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_is_available() {
        let _ = is_available();
    }

    #[test]
    fn test_parse_apt_list() {
        let sample = "Listing... Done
vim/bullseye-security 8.2.1234-1 amd64 [upgradable from: 8.2.1000-1]
curl/bullseye 7.74.0-1.3+deb11u7 amd64 [upgradable from: 7.74.0-1.3+deb11u1]
";
        let result = parse_apt_list_output(sample);
        assert_eq!(result.len(), 2);
        assert_eq!(result[0].package_name, "vim");
        assert_eq!(result[0].available_version, "8.2.1234-1");
        assert_eq!(result[0].current_version, "8.2.1000-1");
        assert_eq!(result[0].severity, "security");
        assert_eq!(result[1].package_name, "curl");
        assert_eq!(result[1].severity, "standard");
    }

    #[test]
    fn test_parse_apt_list_empty() {
        let sample = "Listing... Done
";
        let result = parse_apt_list_output(sample);
        assert!(result.is_empty());
    }

    fn parse_apt_list_output(output: &str) -> Vec<PackageUpdate> {
        let mut updates = Vec::new();
        for line in output.lines() {
            let line = line.trim();
            if line.is_empty() || line.starts_with("Listing") || !line.contains('/') {
                continue;
            }
            if let Some(pkg_part) = line.split_whitespace().next() {
                let parts: Vec<&str> = pkg_part.split('/').collect();
                let package_name = parts[0].to_string();

                let mut current = String::new();
                let mut available = String::new();

                let tokens: Vec<&str> = line.split_whitespace().collect();
                if tokens.len() >= 3 {
                    available = tokens[1].to_string();
                }

                if let Some(bracket_start) = line.find('[') {
                    if let Some(bracket_end) = line.find(']') {
                        let inside = &line[bracket_start + 1..bracket_end];
                        if let Some(from) = inside.strip_prefix("upgradable from: ") {
                            current = from.trim().to_string();
                        }
                    }
                }

                let severity = if line.contains("security") || line.contains("Security") {
                    "security".to_string()
                } else {
                    "standard".to_string()
                };

                updates.push(PackageUpdate {
                    package_name,
                    current_version: current,
                    available_version: available,
                    severity,
                    status: "pending".to_string(),
                });
            }
        }
        updates
    }
}
