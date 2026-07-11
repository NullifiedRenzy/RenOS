use std::path::Path;
use std::time::UNIX_EPOCH;

const MAX_READ_SIZE: u64 = 1_048_576; // 1MB max file read

#[derive(Debug, serde::Serialize)]
struct FileEntry {
    name: String,
    path: String,
    size: u64,
    file_type: String,
    permissions: String,
    modified: String,
}

#[derive(Debug, serde::Deserialize)]
struct PendingActionsResponse {
    file_actions: Vec<PendingAction>,
}

#[derive(Debug, serde::Deserialize)]
pub struct PendingAction {
    pub id: String,
    pub action_type: String,
    pub path: String,
    pub query: String,
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
    result_json: &'a str,
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
    let url = format!("{}/api/v1/file-actions/pending", api_url);

    let resp = client
        .post(&url)
        .json(&PendingRequestBody {
            device_id,
            device_secret,
        })
        .timeout(std::time::Duration::from_secs(10))
        .send()
        .await
        .map_err(|e| Error::Http(format!("file actions poll failed: {e}")))?;

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

    Ok(data.file_actions)
}

#[allow(clippy::too_many_arguments)]
pub async fn submit_result(
    api_url: &str,
    device_id: &str,
    device_secret: &str,
    action_id: &str,
    exit_code: i32,
    output: &str,
    error: &str,
    result_json: &str,
) -> Result<(), Error> {
    let client = reqwest::Client::new();
    let url = format!("{}/api/v1/file-actions/{}/result", api_url, action_id);

    let resp = client
        .post(&url)
        .json(&SubmitBody {
            device_id,
            device_secret,
            exit_code,
            output,
            error,
            result_json,
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

pub fn validate_path(path: &str) -> Result<(), String> {
    if path.is_empty() {
        return Err("path must not be empty".to_string());
    }
    if path.contains('\0') {
        return Err("path must not contain null bytes".to_string());
    }
    let p = Path::new(path);
    for c in p.components() {
        use std::path::Component;
        if matches!(c, Component::ParentDir) {
            return Err("path must not contain '..'".to_string());
        }
    }
    Ok(())
}

fn format_permissions(mode: std::fs::Permissions) -> String {
    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        let mode = mode.mode();
        let mut s = String::with_capacity(9);
        s.push(if mode & 0o400 != 0 { 'r' } else { '-' });
        s.push(if mode & 0o200 != 0 { 'w' } else { '-' });
        s.push(if mode & 0o100 != 0 { 'x' } else { '-' });
        s.push(if mode & 0o040 != 0 { 'r' } else { '-' });
        s.push(if mode & 0o020 != 0 { 'w' } else { '-' });
        s.push(if mode & 0o010 != 0 { 'x' } else { '-' });
        s.push(if mode & 0o004 != 0 { 'r' } else { '-' });
        s.push(if mode & 0o002 != 0 { 'w' } else { '-' });
        s.push(if mode & 0o001 != 0 { 'x' } else { '-' });
        s
    }
    #[cfg(not(unix))]
    {
        let _ = mode;
        "---------".to_string()
    }
}

fn format_time(modified: std::time::SystemTime) -> String {
    match modified.duration_since(UNIX_EPOCH) {
        Ok(d) => {
            let secs = d.as_secs();
            let nanos = d.subsec_nanos();
            let millis = nanos / 1_000_000;
            format!("{}", secs as f64 + millis as f64 / 1000.0)
        }
        Err(_) => String::new(),
    }
}

fn list_directory(path: &str) -> (i32, String, String, String) {
    let mut entries: Vec<FileEntry> = Vec::new();

    let dir = match std::fs::read_dir(path) {
        Ok(d) => d,
        Err(e) => {
            return (
                1,
                String::new(),
                format!("failed to read directory: {e}"),
                "[]".to_string(),
            );
        }
    };

    for entry in dir {
        let entry = match entry {
            Ok(e) => e,
            Err(_) => continue,
        };

        let name = entry.file_name().to_string_lossy().to_string();
        let entry_path = entry.path().to_string_lossy().to_string();
        let metadata = match entry.metadata() {
            Ok(m) => m,
            Err(_) => continue,
        };

        let file_type = if metadata.is_dir() {
            "dir".to_string()
        } else if metadata.is_symlink() {
            "symlink".to_string()
        } else {
            "file".to_string()
        };

        entries.push(FileEntry {
            name,
            path: entry_path,
            size: metadata.len(),
            file_type,
            permissions: format_permissions(metadata.permissions()),
            modified: format_time(metadata.modified().unwrap_or(std::time::SystemTime::now())),
        });
    }

    entries.sort_by(|a, b| {
        if a.file_type != b.file_type {
            if a.file_type == "dir" {
                std::cmp::Ordering::Less
            } else if b.file_type == "dir" {
                std::cmp::Ordering::Greater
            } else {
                a.name.cmp(&b.name)
            }
        } else {
            a.name.cmp(&b.name)
        }
    });

    let json = serde_json::to_string(&entries).unwrap_or_else(|_| "[]".to_string());
    let count = entries.len();
    (0, format!("listed {count} entries"), String::new(), json)
}

fn read_file(path: &str) -> (i32, String, String, String) {
    let metadata = match std::fs::metadata(path) {
        Ok(m) => m,
        Err(e) => {
            return (
                1,
                String::new(),
                format!("failed to access file: {e}"),
                "null".to_string(),
            )
        }
    };

    if metadata.is_dir() {
        return (
            1,
            String::new(),
            "path is a directory, not a file".to_string(),
            "null".to_string(),
        );
    }

    if metadata.len() > MAX_READ_SIZE {
        return (
            1,
            String::new(),
            format!(
                "file too large ({} bytes, max {MAX_READ_SIZE})",
                metadata.len()
            ),
            "null".to_string(),
        );
    }

    let content = match std::fs::read_to_string(path) {
        Ok(c) => c,
        Err(e) => {
            return (
                1,
                String::new(),
                format!("failed to read file: {e}"),
                "null".to_string(),
            )
        }
    };

    let json = format!(
        r#"{{"content": {}, "size": {}, "path": {}}}"#,
        serde_json::Value::String(content.clone()),
        metadata.len(),
        serde_json::Value::String(path.to_string()),
    );

    let preview = if content.len() > 200 {
        format!("{}... ({} chars total)", &content[..200], content.len())
    } else {
        content.clone()
    };

    (0, preview, String::new(), json)
}

fn search_files(path: &str, query: &str) -> (i32, String, String, String) {
    let mut results: Vec<FileEntry> = Vec::new();
    let query_lower = query.to_lowercase();

    let mut dirs = vec![path.to_string()];
    let mut depth = 0;

    while !dirs.is_empty() && depth < 5 {
        let current = dirs.remove(0);
        let dir = match std::fs::read_dir(&current) {
            Ok(d) => d,
            Err(_) => continue,
        };

        for entry in dir {
            if results.len() >= 200 {
                break;
            }
            let entry = match entry {
                Ok(e) => e,
                Err(_) => continue,
            };

            let name = entry.file_name().to_string_lossy().to_string();
            let entry_path = entry.path();

            if entry_path.is_dir() {
                dirs.push(entry_path.to_string_lossy().to_string());
            }

            if !name.to_lowercase().contains(&query_lower) {
                continue;
            }

            let metadata = match entry.metadata() {
                Ok(m) => m,
                Err(_) => continue,
            };

            let file_type = if metadata.is_dir() {
                "dir".to_string()
            } else if metadata.is_symlink() {
                "symlink".to_string()
            } else {
                "file".to_string()
            };

            results.push(FileEntry {
                name,
                path: entry_path.to_string_lossy().to_string(),
                size: metadata.len(),
                file_type,
                permissions: format_permissions(metadata.permissions()),
                modified: format_time(metadata.modified().unwrap_or(std::time::SystemTime::now())),
            });
        }
        depth += 1;
    }

    results.sort_by(|a, b| a.path.cmp(&b.path));

    let json = serde_json::to_string(&results).unwrap_or_else(|_| "[]".to_string());
    let count = results.len();
    (0, format!("found {count} matches"), String::new(), json)
}

pub fn execute_action(action_type: &str, path: &str, query: &str) -> (i32, String, String, String) {
    if let Err(e) = validate_path(path) {
        return (1, String::new(), e, "null".to_string());
    }

    match action_type {
        "list_directory" => list_directory(path),
        "read_file" => read_file(path),
        "search_files" => search_files(path, query),
        _ => (
            1,
            String::new(),
            format!("unknown action type: {action_type}"),
            "null".to_string(),
        ),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_validate_path_ok() {
        assert!(validate_path("/").is_ok());
        assert!(validate_path("/home/user").is_ok());
        assert!(validate_path("/var/log/syslog").is_ok());
        assert!(validate_path("/tmp/test_dir").is_ok());
    }

    #[test]
    fn test_validate_path_rejects_bad_paths() {
        assert!(validate_path("").is_err(), "empty");
        assert!(validate_path("/../etc").is_err(), "parent dir");
        assert!(validate_path("../etc/passwd").is_err(), "relative parent");
        assert!(validate_path("foo/../../bar").is_err(), "nested parent");
        assert!(validate_path("/etc/passwd\0").is_err(), "null byte");
    }

    #[test]
    fn test_validate_path_allows_hidden_files() {
        assert!(validate_path("/home/user/.ssh").is_ok());
        assert!(validate_path("/home/user/.config").is_ok());
    }

    #[test]
    fn test_list_directory_tmp() {
        let tmp = std::env::temp_dir().to_string_lossy().to_string();
        let (code, stdout, stderr, _result_json) = list_directory(&tmp);
        assert_eq!(code, 0, "listing tmp should succeed: {stderr}");
        assert!(!stdout.is_empty(), "should have output");
    }

    #[test]
    fn test_list_directory_bad_path() {
        let (code, _stdout, stderr, _result_json) = list_directory("/nonexistent/path");
        assert_ne!(code, 0);
        assert!(stderr.contains("failed to read directory"));
    }

    #[test]
    fn test_execute_action_rejects_bad_path() {
        let (code, _stdout, stderr, _result_json) = execute_action("list_directory", "../etc", "");
        assert_ne!(code, 0);
        assert!(stderr.contains("must not contain"));
    }

    #[test]
    fn test_execute_action_unknown() {
        let (code, _stdout, stderr, _result_json) = execute_action("unknown_action", "/tmp", "");
        assert_ne!(code, 0);
        assert!(stderr.contains("unknown action type"));
    }
}
