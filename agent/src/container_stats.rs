use std::process::Command;

use serde::{Deserialize, Serialize};

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct ContainerMetric {
    pub container_id: String,
    pub container_name: String,
    pub cpu_percent: f64,
    pub memory_usage_bytes: i64,
    pub memory_limit_bytes: i64,
    pub memory_percent: f64,
    pub network_rx_bytes: i64,
    pub network_tx_bytes: i64,
    pub block_read_bytes: i64,
    pub block_write_bytes: i64,
    pub pids: i32,
}

pub fn collect_container_stats() -> Result<Vec<ContainerMetric>, String> {
    let output = Command::new("docker")
        .args([
            "stats", "--no-stream", "--no-trunc",
            "--format", "{{.ID}}\t{{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}\t{{.NetIO}}\t{{.BlockIO}}\t{{.PIDs}}",
        ])
        .output()
        .map_err(|e| format!("failed to run docker stats: {e}"))?;

    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        return Err(format!("docker stats failed: {}", stderr.trim()));
    }

    let stdout = String::from_utf8_lossy(&output.stdout);
    let mut metrics = Vec::new();

    for line in stdout.lines() {
        let line = line.trim();
        if line.is_empty() {
            continue;
        }

        let parts: Vec<&str> = line.splitn(8, '\t').collect();
        if parts.len() < 8 {
            continue;
        }

        let cpu_percent = parse_percent(parts[2]);
        let (mem_usage, mem_limit) = parse_mem_usage(parts[3]);
        let mem_percent = parse_percent(parts[4]);
        let (net_rx, net_tx) = parse_net_io(parts[5]);
        let (block_r, block_w) = parse_block_io(parts[6]);
        let pids = parts[7].trim().parse::<i32>().unwrap_or(0);

        metrics.push(ContainerMetric {
            container_id: parts[0].to_string(),
            container_name: parts[1].to_string(),
            cpu_percent,
            memory_usage_bytes: mem_usage,
            memory_limit_bytes: mem_limit,
            memory_percent: mem_percent,
            network_rx_bytes: net_rx,
            network_tx_bytes: net_tx,
            block_read_bytes: block_r,
            block_write_bytes: block_w,
            pids,
        });
    }

    Ok(metrics)
}

fn parse_percent(s: &str) -> f64 {
    s.trim().trim_end_matches('%').parse::<f64>().unwrap_or(0.0)
}

fn parse_mem_usage(s: &str) -> (i64, i64) {
    // Format: "123.4MiB / 1.5GiB" or "0B / 0B"
    let parts: Vec<&str> = s.splitn(2, '/').collect();
    if parts.len() == 2 {
        (parse_bytes(parts[0]), parse_bytes(parts[1]))
    } else {
        (0, 0)
    }
}

fn parse_net_io(s: &str) -> (i64, i64) {
    // Format: "1.5MB / 2.3MB" or "0B / 0B"
    let parts: Vec<&str> = s.splitn(2, '/').collect();
    if parts.len() == 2 {
        (parse_bytes(parts[0]), parse_bytes(parts[1]))
    } else {
        (0, 0)
    }
}

fn parse_block_io(s: &str) -> (i64, i64) {
    // Format: "12.3MB / 5.6MB" or "0B / 0B"
    let parts: Vec<&str> = s.splitn(2, '/').collect();
    if parts.len() == 2 {
        (parse_bytes(parts[0]), parse_bytes(parts[1]))
    } else {
        (0, 0)
    }
}

fn parse_bytes(s: &str) -> i64 {
    let s = s.trim();
    if s.is_empty() || s == "0B" {
        return 0;
    }

    let s = s.trim_end_matches("B");
    let (num_str, multiplier) = if let Some(pos) = s.find(|c: char| c.is_alphabetic()) {
        (&s[..pos], &s[pos..])
    } else {
        (s, "")
    };

    let num: f64 = num_str.trim().parse().unwrap_or(0.0);
    let bytes = match multiplier.trim() {
        "K" | "Ki" => num * 1024.0,
        "M" | "Mi" => num * 1024.0 * 1024.0,
        "G" | "Gi" => num * 1024.0 * 1024.0 * 1024.0,
        "T" | "Ti" => num * 1024.0 * 1024.0 * 1024.0 * 1024.0,
        _ => num,
    };

    bytes as i64
}

pub async fn submit_container_metrics(
    api_url: &str,
    device_id: &str,
    device_secret: &str,
    metrics: &[ContainerMetric],
) -> Result<(), String> {
    let client = reqwest::Client::new();
    let url = format!("{}/api/v1/devices/{}/container-metrics", api_url, device_id);

    #[derive(serde::Serialize)]
    struct SyncRequest<'a> {
        device_id: &'a str,
        device_secret: &'a str,
        metrics: &'a [ContainerMetric],
    }

    let resp = client
        .post(&url)
        .json(&SyncRequest {
            device_id,
            device_secret,
            metrics,
        })
        .timeout(std::time::Duration::from_secs(10))
        .send()
        .await
        .map_err(|e| format!("sync container metrics request failed: {e}"))?;

    let status = resp.status();
    if !status.is_success() {
        let body = resp.text().await.unwrap_or_default();
        return Err(format!("sync container metrics rejected ({}): {}", status, body));
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_percent() {
        assert!((parse_percent("12.34%") - 12.34).abs() < 0.01);
        assert!((parse_percent("0.00%") - 0.0).abs() < 0.01);
    }

    #[test]
    fn test_parse_bytes() {
        assert_eq!(parse_bytes("0B"), 0);
        assert_eq!(parse_bytes("1024B"), 1024);
        assert_eq!(parse_bytes("1KiB"), 1024);
        assert_eq!(parse_bytes("1MiB"), 1024 * 1024);
        assert_eq!(parse_bytes("1GiB"), 1024 * 1024 * 1024);
    }

    #[test]
    fn test_parse_mem_usage() {
        let (usage, limit) = parse_mem_usage("123.4MiB / 1.5GiB");
        assert!(usage > 0);
        assert!(limit > 0);
        assert!(limit > usage);
    }
}
