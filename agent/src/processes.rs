use serde::Serialize;
use sysinfo::{ProcessStatus, System, Users};

#[derive(Debug, Serialize, Clone)]
pub struct ProcessInfo {
    pub pid: u32,
    pub name: String,
    pub executable: String,
    pub command_line: String,
    pub cpu_usage: f64,
    pub memory_bytes: u64,
    pub memory_usage: f64,
    pub threads: u32,
    pub open_handles: u32,
    pub parent_pid: u32,
    pub owner: String,
    pub status: String,
    pub start_time: u64,
}

pub fn collect_processes() -> Vec<ProcessInfo> {
    let mut system = System::new_all();
    system.refresh_all();

    let users = Users::new_with_refreshed_list();
    let total_memory = system.total_memory();

    let mut processes = Vec::new();

    for (pid, process) in system.processes() {
        let name = process.name().to_string_lossy().to_string();
        let exe = process
            .exe()
            .map(|p| p.to_string_lossy().to_string())
            .unwrap_or_default();
        let cmd: String = process
            .cmd()
            .iter()
            .map(|s| s.to_string_lossy().to_string())
            .collect::<Vec<_>>()
            .join(" ");

        let owner_name = process
            .user_id()
            .and_then(|uid| users.get_user_by_id(uid))
            .map(|u| u.name().to_string())
            .unwrap_or_default();

        let status = format_process_status(process.status());

        let threads = process.tasks().map(|t| t.len() as u32).unwrap_or(0);
        let start_time = process.start_time();

        processes.push(ProcessInfo {
            pid: pid.as_u32(),
            name,
            executable: exe,
            command_line: cmd,
            cpu_usage: process.cpu_usage() as f64,
            memory_bytes: process.memory(),
            memory_usage: if total_memory > 0 {
                process.memory() as f64 / total_memory as f64 * 100.0
            } else {
                0.0
            },
            threads,
            open_handles: 0,
            parent_pid: process.parent().map(|p| p.as_u32()).unwrap_or(0),
            owner: owner_name,
            status,
            start_time,
        });
    }

    processes.sort_by(|a, b| {
        b.cpu_usage
            .partial_cmp(&a.cpu_usage)
            .unwrap_or(std::cmp::Ordering::Equal)
    });
    processes
}

fn format_process_status(status: ProcessStatus) -> String {
    match status {
        ProcessStatus::Run => "running".to_string(),
        ProcessStatus::Sleep => "sleeping".to_string(),
        ProcessStatus::Stop => "stopped".to_string(),
        ProcessStatus::Zombie => "zombie".to_string(),
        ProcessStatus::Idle => "idle".to_string(),
        ProcessStatus::UninterruptibleDiskSleep => "disk_sleep".to_string(),
        ProcessStatus::Tracing => "tracing".to_string(),
        ProcessStatus::Dead => "dead".to_string(),
        ProcessStatus::Wakekill => "wakekill".to_string(),
        ProcessStatus::Waking => "waking".to_string(),
        ProcessStatus::Parked => "parked".to_string(),
        ProcessStatus::LockBlocked => "lock_blocked".to_string(),
        ProcessStatus::Unknown(s) => format!("unknown({})", s),
    }
}

pub fn execute_process_action(action: &str, pid: u32) -> (i32, String, String) {
    match action {
        "kill" => {
            #[cfg(unix)]
            {
                unsafe {
                    if libc::kill(pid as i32, libc::SIGKILL) == 0 {
                        (0, format!("Sent SIGKILL to PID {}", pid), String::new())
                    } else {
                        (
                            1,
                            String::new(),
                            format!(
                                "Failed to kill PID {}: {}",
                                pid,
                                std::io::Error::last_os_error()
                            ),
                        )
                    }
                }
            }
            #[cfg(not(unix))]
            {
                match std::process::Command::new("taskkill")
                    .args(&["/F", "/PID", &pid.to_string()])
                    .output()
                {
                    Ok(output) => {
                        let stdout = String::from_utf8_lossy(&output.stdout).to_string();
                        let stderr = String::from_utf8_lossy(&output.stderr).to_string();
                        (output.status.code().unwrap_or(1), stdout, stderr)
                    }
                    Err(e) => (
                        1,
                        String::new(),
                        format!("Failed to kill PID {}: {}", pid, e),
                    ),
                }
            }
        }
        "terminate" => {
            #[cfg(unix)]
            {
                unsafe {
                    if libc::kill(pid as i32, libc::SIGTERM) == 0 {
                        (0, format!("Sent SIGTERM to PID {}", pid), String::new())
                    } else {
                        (
                            1,
                            String::new(),
                            format!(
                                "Failed to terminate PID {}: {}",
                                pid,
                                std::io::Error::last_os_error()
                            ),
                        )
                    }
                }
            }
            #[cfg(not(unix))]
            {
                match std::process::Command::new("taskkill")
                    .args(&["/PID", &pid.to_string()])
                    .output()
                {
                    Ok(output) => {
                        let stdout = String::from_utf8_lossy(&output.stdout).to_string();
                        let stderr = String::from_utf8_lossy(&output.stderr).to_string();
                        (output.status.code().unwrap_or(1), stdout, stderr)
                    }
                    Err(e) => (
                        1,
                        String::new(),
                        format!("Failed to terminate PID {}: {}", pid, e),
                    ),
                }
            }
        }
        "suspend" => {
            #[cfg(unix)]
            {
                unsafe {
                    if libc::kill(pid as i32, libc::SIGSTOP) == 0 {
                        (0, format!("Sent SIGSTOP to PID {}", pid), String::new())
                    } else {
                        (
                            1,
                            String::new(),
                            format!(
                                "Failed to suspend PID {}: {}",
                                pid,
                                std::io::Error::last_os_error()
                            ),
                        )
                    }
                }
            }
            #[cfg(not(unix))]
            {
                (
                    1,
                    String::new(),
                    "Suspend not supported on Windows".to_string(),
                )
            }
        }
        "resume" => {
            #[cfg(unix)]
            {
                unsafe {
                    if libc::kill(pid as i32, libc::SIGCONT) == 0 {
                        (0, format!("Sent SIGCONT to PID {}", pid), String::new())
                    } else {
                        (
                            1,
                            String::new(),
                            format!(
                                "Failed to resume PID {}: {}",
                                pid,
                                std::io::Error::last_os_error()
                            ),
                        )
                    }
                }
            }
            #[cfg(not(unix))]
            {
                (
                    1,
                    String::new(),
                    "Resume not supported on Windows".to_string(),
                )
            }
        }
        "restart" => {
            #[cfg(unix)]
            {
                unsafe {
                    if libc::kill(pid as i32, libc::SIGHUP) == 0 {
                        (0, format!("Sent SIGHUP to PID {}", pid), String::new())
                    } else {
                        (
                            1,
                            String::new(),
                            format!(
                                "Failed to restart PID {}: {}",
                                pid,
                                std::io::Error::last_os_error()
                            ),
                        )
                    }
                }
            }
            #[cfg(not(unix))]
            {
                (
                    1,
                    String::new(),
                    "Restart not supported on Windows".to_string(),
                )
            }
        }
        _ => (1, String::new(), format!("Unknown action: {}", action)),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_collect_processes() {
        let processes = collect_processes();
        assert!(!processes.is_empty());
    }

    #[test]
    fn test_format_process_status() {
        assert_eq!(format_process_status(ProcessStatus::Run), "running");
        assert_eq!(format_process_status(ProcessStatus::Sleep), "sleeping");
    }
}
