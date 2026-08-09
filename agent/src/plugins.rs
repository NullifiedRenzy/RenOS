#[cfg(unix)]
use std::os::unix::fs::PermissionsExt;
use std::path::Path;
use std::process::Command;
use std::sync::mpsc;
use std::thread;
use std::time::Duration;

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct PluginInfo {
    pub name: String,
    pub description: String,
    pub plugin_type: String,
    pub status: String,
    pub last_output: String,
    pub last_exit_code: i32,
}

#[derive(Debug)]
pub enum PluginError {
    DirMissing,
    #[allow(dead_code)]
    CommandFailed(String),
}

impl std::fmt::Display for PluginError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            PluginError::DirMissing => write!(f, "plugins directory not found"),
            PluginError::CommandFailed(msg) => write!(f, "command failed: {msg}"),
        }
    }
}

const PLUGIN_DIRS: &[&str] = &["/etc/renos/plugins", "/usr/local/lib/renos/plugins"];

fn get_plugin_dir() -> Option<String> {
    for dir in PLUGIN_DIRS {
        if Path::new(dir).is_dir() {
            return Some(dir.to_string());
        }
    }
    None
}

fn read_description(path: &Path) -> String {
    if let Ok(content) = std::fs::read_to_string(path) {
        for line in content.lines() {
            let line = line.trim();
            let desc = line
                .strip_prefix("# Description:")
                .or_else(|| line.strip_prefix("# description:"))
                .map(|s| s.trim().to_string())
                .unwrap_or_default();
            if !desc.is_empty() {
                return desc;
            }
        }
    }
    String::new()
}

pub fn list_plugins() -> Result<Vec<PluginInfo>, PluginError> {
    let dir_path = get_plugin_dir().ok_or(PluginError::DirMissing)?;
    let dir = Path::new(&dir_path);

    let mut plugins = Vec::new();

    let entries = match std::fs::read_dir(dir) {
        Ok(e) => e,
        Err(_) => return Err(PluginError::DirMissing),
    };

    for entry in entries.flatten() {
        let path = entry.path();
        if !path.is_file() {
            continue;
        }

        // Check if executable
        let is_exec = std::fs::metadata(&path)
            .map(|m| {
                #[cfg(unix)]
                { m.permissions().mode() & 0o111 != 0 }
                #[cfg(not(unix))]
                { m.len() > 0 }
            })
            .unwrap_or(false);

        if !is_exec {
            continue;
        }

        let name = path
            .file_name()
            .and_then(|n| n.to_str())
            .unwrap_or("")
            .to_string();

        if name.is_empty() {
            continue;
        }

        let description = read_description(&path);

        plugins.push(PluginInfo {
            name,
            description,
            plugin_type: "script".to_string(),
            status: "idle".to_string(),
            last_output: String::new(),
            last_exit_code: 0,
        });
    }

    Ok(plugins)
}

#[allow(dead_code)]
pub fn run_plugin(name: &str) -> Result<(String, i32), PluginError> {
    let dir_path = get_plugin_dir().ok_or(PluginError::DirMissing)?;

    // Prevent path traversal — reject names with path separators
    if name.contains('/') || name.contains('\\') || name.contains("..") {
        return Err(PluginError::CommandFailed(format!("invalid plugin name: {name}")));
    }

    let path = Path::new(&dir_path).join(name);

    // Verify the resolved path is still within the plugin directory
    let canonical = std::fs::canonicalize(&path).map_err(|_| {
        PluginError::CommandFailed(format!("plugin not found: {name}"))
    })?;
    let canonical_dir = std::fs::canonicalize(&dir_path).map_err(|_| {
        PluginError::DirMissing
    })?;
    if !canonical.starts_with(&canonical_dir) {
        return Err(PluginError::CommandFailed(format!("plugin path escape denied: {name}")));
    }

    if !canonical.is_file() {
        return Err(PluginError::CommandFailed(format!("plugin not found: {name}")));
    }

    let (tx, rx) = mpsc::channel();
    let child_path = canonical.clone();
    thread::spawn(move || {
        let output = Command::new(&child_path)
            .env_clear()
            .env("PATH", "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin")
            .output();
        let _ = tx.send(output);
    });

    let timeout = Duration::from_secs(30);
    match rx.recv_timeout(timeout) {
        Ok(Ok(output)) => {
            let stdout = String::from_utf8_lossy(&output.stdout).to_string();
            let exit_code = output.status.code().unwrap_or(-1);
            Ok((stdout, exit_code))
        }
        Ok(Err(e)) => Err(PluginError::CommandFailed(e.to_string())),
        Err(_) => Err(PluginError::CommandFailed(format!(
            "plugin {name} timed out after {}s",
            timeout.as_secs()
        ))),
    }
}

#[derive(serde::Serialize)]
struct SyncRequest<'a> {
    device_id: &'a str,
    device_secret: &'a str,
    plugins: &'a [PluginInfo],
}

pub async fn submit_plugins(
    api_url: &str,
    device_id: &str,
    device_secret: &str,
    plugins: &[PluginInfo],
) -> Result<(), String> {
    let client = reqwest::Client::new();
    let url = format!("{}/api/v1/devices/{}/plugins", api_url, device_id);

    let resp = client
        .post(&url)
        .json(&SyncRequest {
            device_id,
            device_secret,
            plugins,
        })
        .timeout(std::time::Duration::from_secs(10))
        .send()
        .await
        .map_err(|e| format!("sync plugins request failed: {e}"))?;

    let status = resp.status();
    if !status.is_success() {
        let body = resp.text().await.unwrap_or_default();
        return Err(format!("sync plugins rejected ({}): {}", status, body));
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    #[cfg(unix)]
    use std::os::unix::fs::PermissionsExt;
    use std::path::PathBuf;

    fn setup_test_plugin(tmpdir: &PathBuf, name: &str, content: &str, desc: &str) {
        let path = tmpdir.join(name);
        let script = if desc.is_empty() {
            content.to_string()
        } else {
            format!("# Description: {desc}\n{content}")
        };
        std::fs::write(&path, script).unwrap();
        #[cfg(unix)]
        std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o755)).unwrap();
    }

    #[test]
    fn test_list_plugins() {
        let tmpdir = std::env::temp_dir().join("renos-test-plugins");
        let _ = std::fs::remove_dir_all(&tmpdir);
        std::fs::create_dir_all(&tmpdir).unwrap();

        setup_test_plugin(&tmpdir, "disk-check.sh", "#!/bin/sh\necho ok", "Check disk usage");
        setup_test_plugin(&tmpdir, "mem-check.sh", "#!/bin/sh\necho ok", "");

        // Override PLUGIN_DIRS for test
        let result = list_plugins_in_dir(&tmpdir.to_string_lossy());
        assert!(result.is_ok());
        let plugins = result.unwrap();
        assert_eq!(plugins.len(), 2);

        let disk = plugins.iter().find(|p| p.name == "disk-check.sh").unwrap();
        assert_eq!(disk.description, "Check disk usage");
        assert_eq!(disk.plugin_type, "script");
    }

    #[test]
    fn test_run_plugin() {
        let tmpdir = std::env::temp_dir().join("renos-test-plugins-run");
        let _ = std::fs::remove_dir_all(&tmpdir);
        std::fs::create_dir_all(&tmpdir).unwrap();

        let path = tmpdir.join("hello.sh");
        std::fs::write(&path, "#!/bin/sh\necho hello").unwrap();
        #[cfg(unix)]
        std::fs::set_permissions(&path, std::fs::Permissions::from_mode(0o755)).unwrap();

        let result = run_plugin_in_dir(&tmpdir.to_string_lossy(), "hello.sh");
        assert!(result.is_ok());
        let (output, code) = result.unwrap();
        assert_eq!(output.trim(), "hello");
        assert_eq!(code, 0);
    }

    #[test]
    fn test_list_plugins_no_dir() {
        let result = list_plugins_in_dir("/nonexistent/renos/plugins");
        assert!(matches!(result, Err(PluginError::DirMissing)));
    }

    fn list_plugins_in_dir(dirpath: &str) -> Result<Vec<PluginInfo>, PluginError> {
        let dir = Path::new(dirpath);
        if !dir.is_dir() {
            return Err(PluginError::DirMissing);
        }

        let mut plugins = Vec::new();
        for entry in std::fs::read_dir(dir).map_err(|_| PluginError::DirMissing)? {
            let entry = entry.map_err(|_| PluginError::DirMissing)?;
            let path = entry.path();
            if !path.is_file() {
                continue;
            }

            let is_exec = std::fs::metadata(&path)
                .map(|m| {
                    #[cfg(unix)]
                    { m.permissions().mode() & 0o111 != 0 }
                    #[cfg(not(unix))]
                    { m.len() > 0 }
                })
                .unwrap_or(false);

            if !is_exec {
                continue;
            }

            let name = path
                .file_name()
                .and_then(|n| n.to_str())
                .unwrap_or("")
                .to_string();

            if name.is_empty() {
                continue;
            }

            let description = read_description(&path);

            plugins.push(PluginInfo {
                name,
                description,
                plugin_type: "script".to_string(),
                status: "idle".to_string(),
                last_output: String::new(),
                last_exit_code: 0,
            });
        }

        Ok(plugins)
    }

    fn run_plugin_in_dir(dirpath: &str, name: &str) -> Result<(String, i32), PluginError> {
        let path = Path::new(dirpath).join(name);
        if !path.is_file() {
            return Err(PluginError::CommandFailed(format!("plugin not found: {name}")));
        }

        let (tx, rx) = mpsc::channel();
        let child_path = path.clone();
        thread::spawn(move || {
            let output = Command::new(&child_path)
                .env_clear()
                .env("PATH", "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin")
                .output();
            let _ = tx.send(output);
        });

        let timeout = Duration::from_secs(10);
        match rx.recv_timeout(timeout) {
            Ok(Ok(output)) => {
                let stdout = String::from_utf8_lossy(&output.stdout).to_string();
                let exit_code = output.status.code().unwrap_or(-1);
                Ok((stdout, exit_code))
            }
            Ok(Err(e)) => Err(PluginError::CommandFailed(e.to_string())),
            Err(_) => Err(PluginError::CommandFailed(format!(
                "plugin {name} timed out after {}s",
                timeout.as_secs()
            ))),
        }
    }
}
