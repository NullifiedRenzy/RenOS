use std::path::PathBuf;

use crate::config::Config;
use crate::identity::{DeviceIdentity, KeyMaterial};

pub struct EnrollmentResult {
    pub device_id: String,
    #[allow(dead_code)]
    pub device_cert_pem: String,
    #[allow(dead_code)]
    pub ca_cert_pem: String,
    pub cert_expiry_unix: i64,
    #[allow(dead_code)]
    pub device_secret: String,
}

// M06 uses HTTP enrollment as a temporary v0.1 bridge.
// Real gRPC streaming + mTLS comes in M07.
// The device_secret is a long-term secret used by /connect until mTLS replaces it.
pub async fn enroll(
    cfg: &Config,
    _identity: &DeviceIdentity,
    key: &KeyMaterial,
    token: &str,
) -> Result<EnrollmentResult, String> {
    let url = format!("{}/api/v1/devices/enroll", cfg.api_url);
    let client = reqwest::Client::new();

    let pub_key_b64 = base64_encode(&key.public_key_der);

    let body = serde_json::json!({
        "token": token,
        "public_key_base64": pub_key_b64,
        "hostname": hostname(),
        "os": os_string(),
        "arch": arch_string(),
        "agent_version": env!("CARGO_PKG_VERSION"),
    });

    tracing::info!("sending enrollment request to {}", url);

    let resp = client
        .post(&url)
        .json(&body)
        .timeout(std::time::Duration::from_secs(30))
        .send()
        .await
        .map_err(|e| format!("enrollment request failed: {e}"))?;

    let status = resp.status();
    if !status.is_success() {
        let error_body = resp.text().await.unwrap_or_default();
        return Err(format!("enrollment rejected ({}): {}", status, error_body));
    }

    let result: serde_json::Value = resp
        .json()
        .await
        .map_err(|e| format!("failed to parse enrollment response: {e}"))?;

    let device_id = result["device_id"]
        .as_str()
        .ok_or("missing device_id in response")?
        .to_string();
    let device_cert_pem = result["cert_pem"]
        .as_str()
        .ok_or("missing cert_pem in response")?
        .to_string();
    let ca_cert_pem = result["ca_cert_pem"]
        .as_str()
        .ok_or("missing ca_cert_pem in response")?
        .to_string();
    let cert_expiry_unix = result["cert_expiry_unix"]
        .as_i64()
        .ok_or("missing cert_expiry_unix in response")?;
    let device_secret = result["device_secret"]
        .as_str()
        .ok_or("missing device_secret in response")?
        .to_string();

    let data_dir = crate::platform::resolve_data_dir();
    let cert_path = PathBuf::from(&data_dir).join("device-cert.pem");
    let ca_path = PathBuf::from(&data_dir).join("ca-cert.pem");
    let secret_path = PathBuf::from(&data_dir).join("device-secret.txt");

    if let Some(parent) = cert_path.parent() {
        let _ = std::fs::create_dir_all(parent);
    }
    let _ = std::fs::write(&cert_path, &device_cert_pem);
    let _ = std::fs::write(&ca_path, &ca_cert_pem);
    let _ = std::fs::write(&secret_path, &device_secret);

    let identity_path = PathBuf::from(&data_dir).join("identity.json");
    let enrolled_identity = crate::identity::DeviceIdentity {
        device_id: device_id.clone(),
        is_enrolled: true,
    };
    if let Ok(json) = serde_json::to_string_pretty(&enrolled_identity) {
        let _ = std::fs::write(&identity_path, &json);
    }

    Ok(EnrollmentResult {
        device_id,
        device_cert_pem,
        ca_cert_pem,
        cert_expiry_unix,
        device_secret,
    })
}

fn base64_encode(bytes: &[u8]) -> String {
    const CHARS: &[u8] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    let mut result = String::new();
    for chunk in bytes.chunks(3) {
        let b0 = chunk[0] as u32;
        let b1 = chunk.get(1).copied().unwrap_or(0) as u32;
        let b2 = chunk.get(2).copied().unwrap_or(0) as u32;
        let triplet = (b0 << 16) | (b1 << 8) | b2;
        result.push(CHARS[((triplet >> 18) & 0x3F) as usize] as char);
        result.push(CHARS[((triplet >> 12) & 0x3F) as usize] as char);
        if chunk.len() > 1 {
            result.push(CHARS[((triplet >> 6) & 0x3F) as usize] as char);
        } else {
            result.push('=');
        }
        if chunk.len() > 2 {
            result.push(CHARS[(triplet & 0x3F) as usize] as char);
        } else {
            result.push('=');
        }
    }
    result
}

fn hostname() -> String {
    std::env::var("HOSTNAME")
        .or_else(|_| std::env::var("COMPUTERNAME"))
        .unwrap_or_else(|_| "unknown".to_string())
}

fn os_string() -> String {
    std::env::consts::OS.to_string()
}

fn arch_string() -> String {
    std::env::consts::ARCH.to_string()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_hostname_returns_string() {
        let h = hostname();
        assert!(!h.is_empty());
    }

    #[test]
    fn test_os_arch_not_empty() {
        assert!(!os_string().is_empty());
        assert!(!arch_string().is_empty());
    }
}
