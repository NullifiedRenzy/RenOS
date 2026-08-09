use serde::Deserialize;
use std::path::PathBuf;

use crate::platform;

#[derive(Debug, Clone, Deserialize)]
struct ConfigFile {
    core_url: Option<String>,
    api_url: Option<String>,
    nats_url: Option<String>,
    heartbeat_interval_secs: Option<u64>,
    log_level: Option<String>,
    protected_services: Option<Vec<String>>,
    proxmox_connections: Option<Vec<ProxmoxConfig>>,
}

#[derive(Debug, Clone, Deserialize)]
pub struct ProxmoxConfig {
    pub host: String,
    pub port: Option<u16>,
    pub auth_type: Option<String>,
    pub token_id: Option<String>,
    pub token_secret: Option<String>,
    pub username: Option<String>,
    pub password: Option<String>,
    pub verify_ssl: Option<bool>,
}

#[derive(Debug, Clone)]
pub struct Config {
    pub core_url: String,
    pub api_url: String,
    pub nats_url: String,
    pub identity_path: PathBuf,
    #[allow(dead_code)]
    pub key_path: PathBuf,
    pub cert_path: PathBuf,
    pub ca_cert_path: PathBuf,
    pub plugin_dir: PathBuf,
    pub cache_dir: PathBuf,
    pub heartbeat_interval_secs: u64,
    pub log_level: String,
    pub protected_services: Vec<String>,
    pub proxmox_connections: Vec<ProxmoxConfig>,
}

impl Default for Config {
    fn default() -> Self {
        let data = platform::resolve_data_dir();
        Self {
            core_url: "http://localhost:8444".to_string(),
            api_url: "http://localhost:8443".to_string(),
            nats_url: "nats://localhost:4222".to_string(),
            identity_path: PathBuf::from(&data).join("identity.json"),
            key_path: PathBuf::from(&data).join("device-key.pem"),
            cert_path: PathBuf::from(&data).join("device-cert.pem"),
            ca_cert_path: PathBuf::from(&data).join("ca-cert.pem"),
            plugin_dir: PathBuf::from(&data).join("plugins"),
            cache_dir: PathBuf::from(&data).join("cache"),
            heartbeat_interval_secs: 30,
            log_level: "info".to_string(),
            protected_services: vec![
                "sshd.service".to_string(),
                "ssh.service".to_string(),
                "NetworkManager.service".to_string(),
                "systemd-logind.service".to_string(),
                "dbus.service".to_string(),
                "firewalld.service".to_string(),
                "docker.service".to_string(),
                "containerd.service".to_string(),
                "renos-agent.service".to_string(),
                "renos-core.service".to_string(),
                "renos-hub.service".to_string(),
            ],
            proxmox_connections: Vec::new(),
        }
    }
}

fn load_from_file() -> Option<ConfigFile> {
    let path = PathBuf::from(platform::resolve_data_dir()).join("config.toml");
    if !path.exists() {
        return None;
    }
    let content = std::fs::read_to_string(path).ok()?;
    toml::from_str(&content).ok()
}

fn apply_env_overrides(cfg: &mut Config) {
    if let Ok(v) = std::env::var("RENOS_CORE_URL") {
        cfg.core_url = v;
    }
    if let Ok(v) = std::env::var("RENOS_API_URL") {
        cfg.api_url = v;
    }
    if let Ok(v) = std::env::var("RENOS_NATS_URL") {
        cfg.nats_url = v;
    }
    if let Ok(v) = std::env::var("RENOS_HEARTBEAT_INTERVAL") {
        if let Ok(n) = v.parse() {
            cfg.heartbeat_interval_secs = n;
        }
    }
    if let Ok(v) = std::env::var("RENOS_LOG_LEVEL") {
        cfg.log_level = v;
    }
    if let Ok(v) = std::env::var("RENOS_IDENTITY_PATH") {
        cfg.identity_path = PathBuf::from(v);
    }
    if let Ok(v) = std::env::var("RENOS_CERT_PATH") {
        cfg.cert_path = PathBuf::from(v);
    }
    if let Ok(v) = std::env::var("RENOS_CA_CERT_PATH") {
        cfg.ca_cert_path = PathBuf::from(v);
    }
    if let Ok(v) = std::env::var("RENOS_PLUGIN_DIR") {
        cfg.plugin_dir = PathBuf::from(v);
    }
    if let Ok(v) = std::env::var("RENOS_CACHE_DIR") {
        cfg.cache_dir = PathBuf::from(v);
    }
}

pub fn load() -> Config {
    let mut cfg = Config::default();

    if let Some(file_cfg) = load_from_file() {
        if let Some(v) = file_cfg.core_url {
            cfg.core_url = v;
        }
        if let Some(v) = file_cfg.api_url {
            cfg.api_url = v;
        }
        if let Some(v) = file_cfg.nats_url {
            cfg.nats_url = v;
        }
        if let Some(v) = file_cfg.heartbeat_interval_secs {
            cfg.heartbeat_interval_secs = v;
        }
        if let Some(v) = file_cfg.log_level {
            cfg.log_level = v;
        }
    }

    apply_env_overrides(&mut cfg);

    if let Some(file_cfg) = load_from_file() {
        if let Some(v) = file_cfg.protected_services {
            if !v.is_empty() {
                cfg.protected_services = v;
            }
        }
        if let Some(v) = file_cfg.proxmox_connections {
            cfg.proxmox_connections = v;
        }
    }

    if let Ok(v) = std::env::var("RENOS_PROTECTED_SERVICES") {
        let list: Vec<String> = v
            .split(',')
            .map(|s| s.trim().to_string())
            .filter(|s| !s.is_empty())
            .collect();
        if !list.is_empty() {
            cfg.protected_services = list;
        }
    }

    cfg
}

#[cfg(test)]
mod tests {
    use super::*;

    use std::sync::Mutex;

    /// Serialises tests that mutate process-wide env vars so they don't race.
    static ENV_MUTEX: Mutex<()> = Mutex::new(());

    #[test]
    fn test_default_config() {
        let cfg = Config::default();
        assert_eq!(cfg.core_url, "http://localhost:8444");
        assert_eq!(cfg.api_url, "http://localhost:8443");
        assert_eq!(cfg.nats_url, "nats://localhost:4222");
        assert_eq!(cfg.heartbeat_interval_secs, 30);
        assert_eq!(cfg.log_level, "info");
    }

    #[test]
    fn test_load_returns_defaults_without_env_or_file() {
        let _guard = ENV_MUTEX.lock().unwrap();
        std::env::remove_var("RENOS_CORE_URL");
        std::env::remove_var("RENOS_NATS_URL");
        let cfg = load();
        assert_eq!(cfg.core_url, "http://localhost:8444");
        assert_eq!(cfg.nats_url, "nats://localhost:4222");
    }

    #[test]
    fn test_env_override() {
        let _guard = ENV_MUTEX.lock().unwrap();
        let key = "RENOS_CORE_URL";
        let test_url = "http://core.test:8444";
        std::env::set_var(key, test_url);
        let cfg = load();
        assert_eq!(cfg.core_url, test_url);
        std::env::remove_var(key);
    }
}
