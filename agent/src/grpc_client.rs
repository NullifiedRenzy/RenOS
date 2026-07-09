use crate::config::Config;

// M06: temporary HTTP-based connect until gRPC streaming + mTLS arrives in M07.
// Authentication uses the device_secret obtained during enrollment.
pub struct Client {
    api_url: String,
    #[allow(dead_code)]
    core_url: String,
    device_id: Option<String>,
    device_secret: Option<String>,
}

impl Client {
    pub fn new(cfg: &Config) -> Self {
        Self {
            api_url: cfg.api_url.clone(),
            core_url: cfg.core_url.clone(),
            device_id: None,
            device_secret: None,
        }
    }

    pub fn set_device_id(&mut self, id: String) {
        self.device_id = Some(id);
    }

    pub fn set_device_secret(&mut self, secret: String) {
        self.device_secret = Some(secret);
    }

    pub async fn connect_to_core(&self) -> Result<(), String> {
        let device_id = self.device_id.as_ref().ok_or("not enrolled")?;
        let device_secret = self.device_secret.as_ref().ok_or("missing device secret")?;

        let url = format!("{}/api/v1/devices/connect", self.api_url);
        let client = reqwest::Client::new();

        let resp = client
            .post(&url)
            .json(&serde_json::json!({
                "device_id": device_id,
                "device_secret": device_secret,
            }))
            .timeout(std::time::Duration::from_secs(10))
            .send()
            .await
            .map_err(|e| format!("connect request failed: {e}"))?;

        let status = resp.status();
        if !status.is_success() {
            let error_body = resp.text().await.unwrap_or_default();
            return Err(format!("connect rejected ({}): {}", status, error_body));
        }

        tracing::info!("device marked online with Core");
        Ok(())
    }

    pub async fn send_heartbeat(&self, health: &crate::health::HealthData) -> Result<(), String> {
        let device_id = self.device_id.as_ref().ok_or("not enrolled")?;
        let device_secret = self.device_secret.as_ref().ok_or("missing device secret")?;

        let url = format!("{}/api/v1/devices/heartbeat", self.api_url);
        let client = reqwest::Client::new();

        let resp = client
            .post(&url)
            .json(&serde_json::json!({
                "device_id": device_id,
                "device_secret": device_secret,
                "hostname": health.hostname,
                "uptime_seconds": health.uptime_seconds,
                "cpu_percent": health.cpu_percent,
                "memory_total_gb": health.memory_total_gb,
                "memory_used_gb": health.memory_used_gb,
                "disk_total_gb": health.disk_total_gb,
                "disk_used_gb": health.disk_used_gb,
                "local_ip": health.local_ip,
            }))
            .timeout(std::time::Duration::from_secs(10))
            .send()
            .await
            .map_err(|e| format!("heartbeat request failed: {e}"))?;

        let status = resp.status();
        if !status.is_success() {
            let error_body = resp.text().await.unwrap_or_default();
            return Err(format!("heartbeat rejected ({}): {}", status, error_body));
        }

        tracing::info!("heartbeat sent to Core");
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::config;

    #[test]
    fn test_client_new() {
        let cfg = config::Config::default();
        let client = Client::new(&cfg);
        assert!(client.device_id.is_none());
        assert!(client.device_secret.is_none());
    }

    #[tokio::test]
    async fn test_connect_without_enroll_fails() {
        let cfg = config::Config::default();
        let client = Client::new(&cfg);
        let result = client.connect_to_core().await;
        assert!(result.is_err());
    }
}
