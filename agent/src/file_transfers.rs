use std::path::{Path, PathBuf};

const DEFAULT_CHUNK_SIZE: usize = 1_048_576; // 1MB

/// Validate that a remote path has no traversal components.
/// Returns the canonicalized path if valid, or an error if path is unsafe.
fn validate_remote_path(path: &str) -> Result<PathBuf, String> {
    // Reject paths with .. components
    if path.contains("..") {
        return Err(format!("path contains traversal component: {}", path));
    }
    // Reject absolute paths that escape the base directory
    let p = Path::new(path);
    if p.is_absolute() {
        // Allow paths under /tmp or /var/tmp as a safety net
        let allowed_prefixes = ["/tmp/", "/var/tmp/"];
        let path_str = path.to_string();
        if !allowed_prefixes.iter().any(|prefix| path_str.starts_with(prefix)) {
            return Err(format!("absolute path not in allowed directory: {}", path));
        }
    }
    // Check for null bytes
    if path.contains('\0') {
        return Err("path contains null byte".to_string());
    }
    Ok(p.to_path_buf())
}

#[derive(Debug, serde::Deserialize)]
struct PendingTransfersResponse {
    file_transfers: Vec<PendingTransfer>,
}

#[derive(Debug, serde::Deserialize)]
pub struct PendingTransfer {
    pub id: String,
    pub transfer_type: String,
    pub filename: String,
    pub remote_path: String,
    pub total_size: i64,
    pub chunk_size: i32,
    pub total_chunks: i32,
    pub uploaded_chunks: i32,
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
}

pub async fn poll_pending(
    api_url: &str,
    device_id: &str,
    device_secret: &str,
) -> Result<Vec<PendingTransfer>, String> {
    let client = reqwest::Client::new();
    let url = format!("{}/api/v1/file-transfers/pending", api_url);

    let resp = client
        .post(&url)
        .json(&PendingRequestBody {
            device_id,
            device_secret,
        })
        .timeout(std::time::Duration::from_secs(10))
        .send()
        .await
        .map_err(|e| format!("file transfers poll failed: {e}"))?;

    let status = resp.status();
    if !status.is_success() {
        let body = resp.text().await.unwrap_or_default();
        return Err(format!("poll rejected ({}): {}", status, body));
    }

    let data: PendingTransfersResponse = resp
        .json()
        .await
        .map_err(|e| format!("poll parse failed: {e}"))?;

    Ok(data.file_transfers)
}

pub async fn download_chunk(
    api_url: &str,
    transfer_id: &str,
    chunk_number: i32,
) -> Result<Vec<u8>, String> {
    let client = reqwest::Client::new();
    let url = format!(
        "{}/api/v1/file-transfers/{}/chunks/{}",
        api_url, transfer_id, chunk_number
    );

    let resp = client
        .get(&url)
        .timeout(std::time::Duration::from_secs(60))
        .send()
        .await
        .map_err(|e| format!("download chunk failed: {e}"))?;

    let status = resp.status();
    if !status.is_success() {
        let body = resp.text().await.unwrap_or_default();
        return Err(format!("download chunk rejected ({}): {}", status, body));
    }

    resp.bytes()
        .await
        .map(|b| b.to_vec())
        .map_err(|e| format!("download chunk read failed: {e}"))
}

pub async fn upload_chunk(
    api_url: &str,
    transfer_id: &str,
    chunk_number: i32,
    data: &[u8],
) -> Result<(), String> {
    let client = reqwest::Client::new();
    let url = format!(
        "{}/api/v1/file-transfers/{}/chunks/{}",
        api_url, transfer_id, chunk_number
    );

    let resp = client
        .post(&url)
        .body(data.to_vec())
        .header("Content-Type", "application/octet-stream")
        .timeout(std::time::Duration::from_secs(60))
        .send()
        .await
        .map_err(|e| format!("upload chunk failed: {e}"))?;

    let status = resp.status();
    if !status.is_success() {
        let body = resp.text().await.unwrap_or_default();
        return Err(format!("upload chunk rejected ({}): {}", status, body));
    }

    Ok(())
}

pub async fn assemble_transfer(
    api_url: &str,
    device_id: &str,
    device_secret: &str,
    transfer_id: &str,
) -> Result<(), String> {
    let client = reqwest::Client::new();
    let url = format!(
        "{}/api/v1/file-transfers/{}/assemble",
        api_url, transfer_id
    );

    let resp = client
        .post(&url)
        .json(&SubmitBody {
            device_id,
            device_secret,
        })
        .timeout(std::time::Duration::from_secs(30))
        .send()
        .await
        .map_err(|e| format!("assemble transfer failed: {e}"))?;

    let status = resp.status();
    if !status.is_success() {
        let body = resp.text().await.unwrap_or_default();
        return Err(format!("assemble rejected ({}): {}", status, body));
    }

    Ok(())
}

pub async fn process_upload(
    api_url: &str,
    device_id: &str,
    device_secret: &str,
    transfer: &PendingTransfer,
) -> Result<(), String> {
    let chunk_size = if transfer.chunk_size > 0 {
        transfer.chunk_size as usize
    } else {
        DEFAULT_CHUNK_SIZE
    };

    // Validate remote path to prevent reading sensitive files
    validate_remote_path(&transfer.remote_path)?;

    // Read file from disk and upload chunks
    let data = std::fs::read(&transfer.remote_path)
        .map_err(|e| format!("failed to read file {}: {e}", transfer.remote_path))?;

    for chunk_num in 0..transfer.total_chunks {
        let start = (chunk_num as usize) * chunk_size;
        let end = std::cmp::min(start + chunk_size, data.len());
        if start >= data.len() {
            break;
        }

        upload_chunk(api_url, &transfer.id, chunk_num, &data[start..end]).await?;
        tracing::info!(
            "uploaded chunk {}/{} for transfer {}",
            chunk_num + 1,
            transfer.total_chunks,
            transfer.id
        );
    }

    assemble_transfer(api_url, device_id, device_secret, &transfer.id).await?;
    Ok(())
}

pub async fn process_download(
    api_url: &str,
    device_id: &str,
    device_secret: &str,
    transfer: &PendingTransfer,
) -> Result<(), String> {
    let chunk_size = if transfer.chunk_size > 0 {
        transfer.chunk_size as usize
    } else {
        DEFAULT_CHUNK_SIZE
    };

    // Validate remote path to prevent path traversal attacks
    validate_remote_path(&transfer.remote_path)?;

    // Ensure parent directory exists
    if let Some(parent) = Path::new(&transfer.remote_path).parent() {
        std::fs::create_dir_all(parent)
            .map_err(|e| format!("failed to create parent directory: {e}"))?;
    }

    let total_size = transfer.total_size as usize;
    let mut file_data = vec![0u8; total_size];
    let mut offset = 0;

    for chunk_num in 0..transfer.total_chunks {
        let chunk = download_chunk(api_url, &transfer.id, chunk_num).await?;
        let copy_len = std::cmp::min(chunk.len(), total_size - offset);
        file_data[offset..offset + copy_len].copy_from_slice(&chunk[..copy_len]);
        offset += copy_len;
        tracing::info!(
            "downloaded chunk {}/{} for transfer {} ({} bytes)",
            chunk_num + 1,
            transfer.total_chunks,
            transfer.id,
            copy_len
        );
    }

    std::fs::write(&transfer.remote_path, &file_data[..offset])
        .map_err(|e| format!("failed to write file {}: {e}", transfer.remote_path))?;

    assemble_transfer(api_url, device_id, device_secret, &transfer.id).await?;
    Ok(())
}
