use crate::retry;
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
    pub target_path: String,
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
    let url = format!("{}/api/v1/file-actions/{}/result", api_url, action_id);
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
                result_json,
            })
            .timeout(std::time::Duration::from_secs(10))
            .send()
            .await;

        match resp {
            Ok(r) if r.status().is_success() => return Ok(()),
            Ok(r) => {
                let status = r.status();
                let body = r.text().await.unwrap_or_default();
                last_err = Some(Error::Rejected(format!(
                    "submit rejected ({status}): {body}"
                )));
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

fn create_directory(path: &str) -> (i32, String, String, String) {
    match std::fs::create_dir_all(path) {
        Ok(()) => (
            0,
            format!("created directory: {path}"),
            String::new(),
            format!(
                r#"{{"created": true, "path": "{}"}}"#,
                serde_json::Value::String(path.to_string())
            ),
        ),
        Err(e) => (
            1,
            String::new(),
            format!("failed to create directory: {e}"),
            "null".to_string(),
        ),
    }
}

fn delete_path(path: &str) -> (i32, String, String, String) {
    let metadata = match std::fs::metadata(path) {
        Ok(m) => m,
        Err(e) => {
            return (
                1,
                String::new(),
                format!("failed to access path: {e}"),
                "null".to_string(),
            );
        }
    };

    let result = if metadata.is_dir() {
        std::fs::remove_dir_all(path)
    } else {
        std::fs::remove_file(path)
    };

    match result {
        Ok(()) => (
            0,
            format!("deleted: {path}"),
            String::new(),
            format!(
                r#"{{"deleted": true, "path": "{}"}}"#,
                serde_json::Value::String(path.to_string())
            ),
        ),
        Err(e) => (
            1,
            String::new(),
            format!("failed to delete: {e}"),
            "null".to_string(),
        ),
    }
}

fn rename_path(path: &str, target_path: &str) -> (i32, String, String, String) {
    match std::fs::rename(path, target_path) {
        Ok(()) => (
            0,
            format!("renamed {path} to {target_path}"),
            String::new(),
            format!(
                r#"{{"renamed": true, "from": "{}", "to": "{}"}}"#,
                serde_json::Value::String(path.to_string()),
                serde_json::Value::String(target_path.to_string()),
            ),
        ),
        Err(e) => (
            1,
            String::new(),
            format!("failed to rename: {e}"),
            "null".to_string(),
        ),
    }
}

fn move_path(path: &str, target_path: &str) -> (i32, String, String, String) {
    match std::fs::rename(path, target_path) {
        Ok(()) => (
            0,
            format!("moved {path} to {target_path}"),
            String::new(),
            format!(
                r#"{{"moved": true, "from": "{}", "to": "{}"}}"#,
                serde_json::Value::String(path.to_string()),
                serde_json::Value::String(target_path.to_string()),
            ),
        ),
        Err(e) => (
            1,
            String::new(),
            format!("failed to move: {e}"),
            "null".to_string(),
        ),
    }
}

fn copy_path(path: &str, target_path: &str) -> (i32, String, String, String) {
    let metadata = match std::fs::metadata(path) {
        Ok(m) => m,
        Err(e) => {
            return (
                1,
                String::new(),
                format!("failed to access source: {e}"),
                "null".to_string(),
            );
        }
    };

    let result = if metadata.is_dir() {
        copy_dir_recursive(path, target_path)
    } else {
        std::fs::copy(path, target_path).map(|_| ())
    };

    match result {
        Ok(()) => (
            0,
            format!("copied {path} to {target_path}"),
            String::new(),
            format!(
                r#"{{"copied": true, "from": "{}", "to": "{}"}}"#,
                serde_json::Value::String(path.to_string()),
                serde_json::Value::String(target_path.to_string()),
            ),
        ),
        Err(e) => (
            1,
            String::new(),
            format!("failed to copy: {e}"),
            "null".to_string(),
        ),
    }
}

#[cfg(unix)]
fn copy_dir_recursive(src: &str, dst: &str) -> Result<(), std::io::Error> {
    std::fs::create_dir_all(dst)?;
    for entry in std::fs::read_dir(src)? {
        let entry = entry?;
        let src_path = entry.path();
        let dst_path = std::path::Path::new(dst).join(entry.file_name());
        if src_path.is_dir() {
            copy_dir_recursive(
                src_path.to_str().unwrap_or(""),
                dst_path.to_str().unwrap_or(""),
            )?;
        } else {
            std::fs::copy(&src_path, &dst_path)?;
        }
    }
    Ok(())
}

#[cfg(not(unix))]
fn copy_dir_recursive(src: &str, dst: &str) -> Result<(), std::io::Error> {
    std::fs::create_dir_all(dst)?;
    for entry in std::fs::read_dir(src)? {
        let entry = entry?;
        let src_path = entry.path();
        let dst_path = std::path::Path::new(dst).join(entry.file_name());
        if src_path.is_dir() {
            copy_dir_recursive(
                src_path.to_str().unwrap_or(""),
                dst_path.to_str().unwrap_or(""),
            )?;
        } else {
            std::fs::copy(&src_path, &dst_path)?;
        }
    }
    Ok(())
}

#[cfg(unix)]
fn set_permissions(path: &str, mode: &str) -> (i32, String, String, String) {
    use std::os::unix::fs::PermissionsExt;

    let octal = match u32::from_str_radix(mode.trim_start_matches("0"), 8) {
        Ok(o) => o,
        Err(_) => {
            return (
                1,
                String::new(),
                format!("invalid permission mode: {mode}"),
                "null".to_string(),
            );
        }
    };

    let perms = std::fs::Permissions::from_mode(octal);
    match std::fs::set_permissions(path, perms) {
        Ok(()) => (
            0,
            format!("set permissions on {path} to {mode}"),
            String::new(),
            format!(
                r#"{{"permissions_set": true, "path": "{}", "mode": "{}"}}"#,
                serde_json::Value::String(path.to_string()),
                serde_json::Value::String(mode.to_string()),
            ),
        ),
        Err(e) => (
            1,
            String::new(),
            format!("failed to set permissions: {e}"),
            "null".to_string(),
        ),
    }
}

#[cfg(not(unix))]
fn set_permissions(_path: &str, _mode: &str) -> (i32, String, String, String) {
    (
        1,
        String::new(),
        "setting permissions is not supported on this platform".to_string(),
        "null".to_string(),
    )
}

fn preview_file(path: &str) -> (i32, String, String, String) {
    let metadata = match std::fs::metadata(path) {
        Ok(m) => m,
        Err(e) => {
            return (
                1,
                String::new(),
                format!("failed to access file: {e}"),
                "null".to_string(),
            );
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

    let size = metadata.len();
    let ext = std::path::Path::new(path)
        .extension()
        .and_then(|e| e.to_str())
        .unwrap_or("")
        .to_lowercase();

    let preview_max: u64 = match ext.as_str() {
        "png" | "jpg" | "jpeg" | "gif" | "webp" | "bmp" | "ico" => 0,
        "pdf" => 0,
        _ => 4096,
    };

    let mut result = serde_json::Map::new();
    result.insert(
        "path".to_string(),
        serde_json::Value::String(path.to_string()),
    );
    result.insert("size".to_string(), serde_json::Value::Number(size.into()));
    result.insert(
        "extension".to_string(),
        serde_json::Value::String(ext.clone()),
    );
    result.insert(
        "permissions".to_string(),
        serde_json::Value::String(format_permissions(metadata.permissions())),
    );
    result.insert(
        "modified".to_string(),
        serde_json::Value::String(format_time(
            metadata.modified().unwrap_or(std::time::SystemTime::now()),
        )),
    );

    if preview_max > 0 && size <= preview_max {
        if let Ok(content) = std::fs::read_to_string(path) {
            result.insert(
                "content".to_string(),
                serde_json::Value::String(content.clone()),
            );
            let preview = if content.len() > 500 {
                format!("{}... ({} chars total)", &content[..500], content.len())
            } else {
                content
            };
            let json = serde_json::to_string(&result).unwrap_or_else(|_| "null".to_string());
            return (0, preview, String::new(), json);
        }
    }

    let preview = match ext.as_str() {
        "png" | "jpg" | "jpeg" | "gif" | "webp" | "bmp" | "ico" => {
            format!("[Image: {ext}, {size} bytes]")
        }
        "pdf" => format!("[PDF document, {size} bytes]"),
        _ if size > preview_max && preview_max > 0 => {
            format!("[Binary or large file, {size} bytes]")
        }
        _ => format!("[File, {size} bytes]"),
    };

    let json = serde_json::to_string(&result).unwrap_or_else(|_| "null".to_string());
    (0, preview, String::new(), json)
}

pub fn execute_action(action_type: &str, path: &str, query: &str) -> (i32, String, String, String) {
    if let Err(e) = validate_path(path) {
        return (1, String::new(), e, "null".to_string());
    }

    match action_type {
        "list_directory" => list_directory(path),
        "read_file" => read_file(path),
        "search_files" => search_files(path, query),
        "create_directory" => create_directory(path),
        "delete_path" => delete_path(path),
        "preview_file" => preview_file(path),
        _ => (
            1,
            String::new(),
            format!("unknown action type: {action_type}"),
            "null".to_string(),
        ),
    }
}

pub fn execute_action_with_target(
    action_type: &str,
    path: &str,
    target_path: &str,
    query: &str,
) -> (i32, String, String, String) {
    if let Err(e) = validate_path(path) {
        return (1, String::new(), e, "null".to_string());
    }
    if !target_path.is_empty() {
        if let Err(e) = validate_path(target_path) {
            return (1, String::new(), e, "null".to_string());
        }
    }

    match action_type {
        "rename_path" => rename_path(path, target_path),
        "move_path" => move_path(path, target_path),
        "copy_path" => copy_path(path, target_path),
        "set_permissions" => set_permissions(path, query),
        _ => execute_action(action_type, path, query),
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
