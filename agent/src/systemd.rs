use std::process::Command;

#[derive(Debug)]
pub enum SystemdError {
    NotInstalled,
    Execution(String),
}

impl std::fmt::Display for SystemdError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            SystemdError::NotInstalled => write!(f, "systemd not available"),
            SystemdError::Execution(msg) => write!(f, "execution error: {msg}"),
        }
    }
}

pub fn is_available() -> bool {
    Command::new("systemctl")
        .arg("--version")
        .output()
        .map(|o| o.status.success())
        .unwrap_or(false)
}
