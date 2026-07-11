use serde::{Deserialize, Serialize};
use std::path::PathBuf;

use crate::platform;

#[derive(Debug, Serialize, Deserialize)]
pub struct DeviceIdentity {
    pub device_id: String,
    pub is_enrolled: bool,
}

pub struct KeyMaterial {
    #[allow(dead_code)]
    pub private_key_seed: [u8; 32],
    pub public_key_der: Vec<u8>,
}

fn ed25519_pkix_public_key(public_key: &[u8; 32]) -> Vec<u8> {
    let mut der = Vec::with_capacity(44);
    der.extend_from_slice(&[0x30, 0x2a]);
    der.extend_from_slice(&[0x30, 0x05]);
    der.extend_from_slice(&[0x06, 0x03, 0x2b, 0x65, 0x70]);
    der.push(0x03);
    der.push(0x21);
    der.push(0x00);
    der.extend_from_slice(public_key);
    der
}

fn generate_keypair(private_key_path: &PathBuf) -> KeyMaterial {
    use rand::RngCore;
    let mut seed = [0u8; 32];
    rand::rngs::OsRng.fill_bytes(&mut seed);
    let signing_key = ed25519_dalek::SigningKey::from_bytes(&seed);
    let verifying_key = signing_key.verifying_key();
    let pkix_der = ed25519_pkix_public_key(&verifying_key.to_bytes());

    if let Some(parent) = private_key_path.parent() {
        let _ = std::fs::create_dir_all(parent);
    }

    let b64 = base64_encode(&seed);
    let pem = format!(
        "-----BEGIN PRIVATE KEY-----\n{}\n-----END PRIVATE KEY-----",
        b64
    );
    let _ = std::fs::write(private_key_path, &pem);

    KeyMaterial {
        private_key_seed: seed,
        public_key_der: pkix_der,
    }
}

fn load_key(private_key_path: &PathBuf) -> Option<KeyMaterial> {
    let content = std::fs::read_to_string(private_key_path).ok()?;
    let mut b64 = String::new();
    for line in content.lines() {
        if line.starts_with("-----") {
            continue;
        }
        b64.push_str(line.trim());
    }
    let der = base64_decode(&b64);
    if der.len() != 32 {
        return None;
    }

    let mut seed = [0u8; 32];
    seed.copy_from_slice(&der);
    let signing_key = ed25519_dalek::SigningKey::from_bytes(&seed);
    let verifying_key = signing_key.verifying_key();
    let pkix_der = ed25519_pkix_public_key(&verifying_key.to_bytes());

    Some(KeyMaterial {
        private_key_seed: seed,
        public_key_der: pkix_der,
    })
}

pub fn load_or_generate() -> (DeviceIdentity, KeyMaterial) {
    let data_dir = platform::resolve_data_dir();
    let identity_path = PathBuf::from(&data_dir).join("identity.json");
    let key_path = PathBuf::from(&data_dir).join("device-key.pem");
    let cert_path = PathBuf::from(&data_dir).join("device-cert.pem");

    let identity = if identity_path.exists() {
        match std::fs::read_to_string(&identity_path).and_then(|s| {
            serde_json::from_str::<DeviceIdentity>(&s)
                .map_err(|e| std::io::Error::new(std::io::ErrorKind::InvalidData, e))
        }) {
            Ok(id) => id,
            Err(e) => {
                tracing::warn!("failed to parse identity, generating new: {e}");
                new_identity(&identity_path)
            }
        }
    } else {
        new_identity(&identity_path)
    };

    let key = match load_key(&key_path) {
        Some(k) => k,
        None => {
            tracing::info!("generating new Ed25519 keypair");
            generate_keypair(&key_path)
        }
    };

    if cert_path.exists() && !identity.is_enrolled {
        let mut id = identity;
        id.is_enrolled = true;
        save_identity(&identity_path, &id);
        tracing::info!("device is enrolled: {}", id.device_id);
        (id, key)
    } else {
        tracing::info!(
            "device identity: {} (enrolled: {})",
            identity.device_id,
            identity.is_enrolled
        );
        (identity, key)
    }
}

fn new_identity(path: &PathBuf) -> DeviceIdentity {
    let id = DeviceIdentity {
        device_id: uuid::Uuid::new_v4().to_string(),
        is_enrolled: false,
    };
    save_identity(path, &id);
    id
}

fn save_identity(path: &PathBuf, identity: &DeviceIdentity) {
    if let Some(parent) = path.parent() {
        let _ = std::fs::create_dir_all(parent);
    }
    if let Ok(json) = serde_json::to_string_pretty(identity) {
        let _ = std::fs::write(path, &json);
    }
}

fn base64_encode(bytes: &[u8]) -> String {
    const CHARS: &[u8] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    let mut result = String::new();
    for chunk in bytes.chunks(48) {
        for c in chunk.chunks(3) {
            let b0 = c[0] as u32;
            let b1 = c.get(1).copied().unwrap_or(0) as u32;
            let b2 = c.get(2).copied().unwrap_or(0) as u32;
            let triplet = (b0 << 16) | (b1 << 8) | b2;
            result.push(CHARS[((triplet >> 18) & 0x3F) as usize] as char);
            result.push(CHARS[((triplet >> 12) & 0x3F) as usize] as char);
            if c.len() > 1 {
                result.push(CHARS[((triplet >> 6) & 0x3F) as usize] as char);
            } else {
                result.push('=');
            }
            if c.len() > 2 {
                result.push(CHARS[(triplet & 0x3F) as usize] as char);
            } else {
                result.push('=');
            }
        }
        result.push('\n');
    }
    result.trim_end().to_string()
}

fn base64_decode(input: &str) -> Vec<u8> {
    let input = input.trim();
    let mut result = Vec::with_capacity(input.len() / 4 * 3);
    let chars: Vec<u8> = input.bytes().collect();
    let mut i = 0;
    for &c in &chars {
        if c == b'=' {}
    }
    while i + 4 <= chars.len() {
        let mut vals = [0u32; 4];
        for j in 0..4 {
            vals[j] = match chars[i + j] {
                b'A'..=b'Z' => (chars[i + j] - b'A') as u32,
                b'a'..=b'z' => (chars[i + j] - b'a' + 26) as u32,
                b'0'..=b'9' => (chars[i + j] - b'0' + 52) as u32,
                b'+' => 62,
                b'/' => 63,
                b'=' => 0,
                _ => 0,
            };
        }
        let triplet = (vals[0] << 18) | (vals[1] << 12) | (vals[2] << 6) | vals[3];
        result.push((triplet >> 16) as u8);
        if chars[i + 2] != b'=' {
            result.push((triplet >> 8) as u8);
        }
        if chars[i + 3] != b'=' {
            result.push(triplet as u8);
        }
        i += 4;
    }
    result
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_pkix_public_key_format() {
        let key = [0u8; 32];
        let pkix = ed25519_pkix_public_key(&key);
        assert_eq!(pkix.len(), 44);
        assert_eq!(pkix[0], 0x30);
    }

    #[test]
    fn test_generate_keypair_roundtrip() {
        let tmp = std::env::temp_dir().join(format!("key-{}.pem", uuid::Uuid::new_v4()));
        let km = generate_keypair(&tmp);
        assert_eq!(km.public_key_der.len(), 44);
        assert_eq!(km.private_key_seed.len(), 32);

        let loaded = load_key(&tmp).expect("should load key");
        assert_eq!(loaded.public_key_der, km.public_key_der);
        let _ = std::fs::remove_file(&tmp);
    }

    #[test]
    fn test_device_id_uuid() {
        let id = uuid::Uuid::new_v4().to_string();
        assert!(uuid::Uuid::parse_str(&id).is_ok());
    }

    #[test]
    fn test_base64_roundtrip() {
        let data = b"hello world";
        let encoded = base64_encode(data);
        let decoded = base64_decode(&encoded);
        assert_eq!(decoded, data);
    }
}
