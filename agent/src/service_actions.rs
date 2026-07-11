use std::process::Command;

use crate::systemd;

#[derive(Debug, serde::Deserialize)]
struct PendingActionsResponse {
    service_actions: Vec<PendingAction>,
}

#[derive(Debug, serde::Deserialize)]
pub struct PendingAction {
    pub id: String,
    pub unit_name: String,
    pub action: String,
}

#[derive(Debug, serde::Serialize)]
struct PendingRequestBody<'a> {
    device_id: &'a str,
    device_secret: &'a str,
}

#[derive(Debug, serde::Serialize)]
struct SubmitBody<'a> {
    device_id: &'a str,
    device_secret: &'a str,
    exit_code: i32,
    output: &'a str,
    error: &'a str,
}

#[derive(Debug)]
pub enum Error {
    Http(String),
    Rejected(String),
}

impl std::fmt::Display for Error {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Error::Http(msg) => write!(f, "http error: {msg}"),
            Error::Rejected(msg) => write!(f, "rejected by core: {msg}"),
        }
    }
}

pub async fn poll(
    api_url: &str,
    device_id: &str,
    device_secret: &str,
) -> Result<Vec<PendingAction>, Error> {
    let client = reqwest::Client::new();
    let url = format!("{}/api/v1/service-actions/pending", api_url);

    let resp = client
        .post(&url)
        .json(&PendingRequestBody {
            device_id,
            device_secret,
        })
        .timeout(std::time::Duration::from_secs(10))
        .send()
        .await
        .map_err(|e| Error::Http(format!("poll request failed: {e}")))?;

    let status = resp.status();
    if !status.is_success() {
        let body = resp.text().await.unwrap_or_default();
        return Err(Error::Rejected(format!(
            "poll rejected ({}): {}",
            status, body
        )));
    }

    let data: PendingActionsResponse = resp
        .json()
        .await
        .map_err(|e| Error::Http(format!("poll parse failed: {e}")))?;

    Ok(data.service_actions)
}

pub async fn submit_result(
    api_url: &str,
    device_id: &str,
    device_secret: &str,
    action_id: &str,
    exit_code: i32,
    output: &str,
    error: &str,
) -> Result<(), Error> {
    let client = reqwest::Client::new();
    let url = format!("{}/api/v1/service-actions/{}/result", api_url, action_id);

    let resp = client
        .post(&url)
        .json(&SubmitBody {
            device_id,
            device_secret,
            exit_code,
            output,
            error,
        })
        .timeout(std::time::Duration::from_secs(10))
        .send()
        .await
        .map_err(|e| Error::Http(format!("submit failed: {e}")))?;

    let status = resp.status();
    if !status.is_success() {
        let body = resp.text().await.unwrap_or_default();
        return Err(Error::Rejected(format!(
            "submit rejected ({}): {}",
            status, body
        )));
    }

    Ok(())
}

pub fn validate_unit_name(unit_name: &str) -> Result<(), String> {
    if !unit_name.ends_with(".service") {
        return Err(format!("unit name must end with .service: {unit_name}"));
    }
    if unit_name.contains(' ') {
        return Err(format!("unit name must not contain spaces: {unit_name}"));
    }
    if unit_name.contains('/') {
        return Err(format!("unit name must not contain slashes: {unit_name}"));
    }
    for ch in unit_name.chars() {
        match ch {
            'a'..='z' | 'A'..='Z' | '0'..='9' | '.' | '-' | '_' | '@' | ':' | '\\' => {}
            _ => {
                return Err(format!(
                    "unit name contains invalid character {ch:?}: {unit_name}"
                ));
            }
        }
    }
    Ok(())
}

pub fn is_protected(unit_name: &str, protected_list: &[String]) -> bool {
    protected_list.iter().any(|p| p == unit_name)
}

#[allow(dead_code)]
pub fn is_stop_restart(action: &str) -> bool {
    action == "stop" || action == "restart"
}

fn run_systemctl(args: &[&str]) -> (i32, String, String) {
    match Command::new("systemctl").args(args).output() {
        Ok(output) => {
            let exit_code = output.status.code().unwrap_or(1);
            let stdout = String::from_utf8_lossy(&output.stdout).trim().to_string();
            let stderr = String::from_utf8_lossy(&output.stderr).trim().to_string();
            (exit_code, stdout, stderr)
        }
        Err(e) => (
            1,
            String::new(),
            format!("failed to execute systemctl: {e}"),
        ),
    }
}

pub fn execute_action(
    action: &str,
    unit_name: &str,
    protected_list: &[String],
) -> (i32, String, String) {
    if !systemd::is_available() {
        return (
            1,
            String::new(),
            "systemd not available on this device".to_string(),
        );
    }

    if let Err(e) = validate_unit_name(unit_name) {
        return (1, String::new(), e);
    }

    if is_protected(unit_name, protected_list) {
        return (
            1,
            String::new(),
            format!("action rejected: {unit_name} is protected"),
        );
    }

    match action {
        "start" => run_systemctl(&["start", unit_name]),
        "stop" => run_systemctl(&["stop", unit_name]),
        "restart" => run_systemctl(&["restart", unit_name]),
        _ => (1, String::new(), format!("unknown action: {action}")),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_validate_unit_name_ok() {
        assert!(validate_unit_name("cups.service").is_ok());
        assert!(validate_unit_name("systemd-journald.service").is_ok());
        assert!(validate_unit_name("sshd-keygen@ecdsa.service").is_ok());
        assert!(
            validate_unit_name("systemd-fsck@dev-disk-by\\x2duuid-E1E4\\x2d8C1A.service").is_ok()
        );
    }

    #[test]
    fn test_validate_unit_name_rejects_bad_names() {
        assert!(validate_unit_name("cups").is_err(), "must end in .service");
        assert!(
            validate_unit_name(" cups.service").is_err(),
            "no leading spaces"
        );
        assert!(validate_unit_name("cups .service").is_err(), "no spaces");
        assert!(validate_unit_name("cups/.service").is_err(), "no slashes");
        assert!(
            validate_unit_name("cups;rm -rf /.service").is_err(),
            "no shell chars"
        );
        assert!(validate_unit_name("cups|service").is_err(), "no pipe");
        assert!(validate_unit_name("").is_err(), "empty");
    }

    #[test]
    fn test_is_protected_blocks_critical_services() {
        let list = vec![
            "sshd.service".to_string(),
            "docker.service".to_string(),
            "dbus.service".to_string(),
        ];
        assert!(is_protected("sshd.service", &list));
        assert!(is_protected("docker.service", &list));
        assert!(is_protected("dbus.service", &list));
        assert!(!is_protected("cups.service", &list));
        assert!(!is_protected("systemd-journald.service", &list));
    }

    #[test]
    fn test_is_stop_restart_identifies_destructive_actions() {
        assert!(is_stop_restart("stop"));
        assert!(is_stop_restart("restart"));
        assert!(!is_stop_restart("start"));
    }

    #[test]
    fn test_execute_action_rejects_invalid_name() {
        let (code, _stdout, stderr) = execute_action("start", "not a valid name", &[]);
        assert_ne!(code, 0);
        assert!(stderr.contains("must end with .service"));
    }

    #[test]
    fn test_execute_action_rejects_protected_service() {
        let (code, _stdout, stderr) =
            execute_action("start", "sshd.service", &["sshd.service".to_string()]);
        assert_ne!(code, 0);
        assert!(stderr.contains("protected"));
    }

    #[test]
    fn test_execute_action_unknown_action() {
        let (code, _stdout, stderr) = execute_action("reload", "cups.service", &[]);
        assert_ne!(code, 0);
        assert!(stderr.contains("unknown action"));
    }
}
