use sysinfo::{Disks, System};

pub struct HealthData {
    pub hostname: String,
    pub uptime_seconds: u64,
    pub cpu_percent: f64,
    pub memory_total_gb: f64,
    pub memory_used_gb: f64,
    pub disk_total_gb: f64,
    pub disk_used_gb: f64,
    pub local_ip: String,
}

pub fn collect() -> HealthData {
    let mut system = System::new_all();
    system.refresh_all();

    let cpu_percent = system.global_cpu_usage() as f64;

    let memory_total_gb = system.total_memory() as f64 / (1024.0 * 1024.0 * 1024.0);
    let memory_used_gb = system.used_memory() as f64 / (1024.0 * 1024.0 * 1024.0);

    let disks = Disks::new_with_refreshed_list();
    let mut disk_total_gb = 0.0;
    let mut disk_used_gb = 0.0;
    for disk in &disks {
        let total = disk.total_space() as f64 / (1024.0 * 1024.0 * 1024.0);
        let available = disk.available_space() as f64 / (1024.0 * 1024.0 * 1024.0);
        disk_total_gb += total;
        disk_used_gb += total - available;
    }

    let hostname = std::env::var("HOSTNAME")
        .or_else(|_| std::env::var("COMPUTERNAME"))
        .unwrap_or_else(|_| "unknown".to_string());

    let uptime_seconds = System::uptime();

    let local_ip = detect_local_ip();

    HealthData {
        hostname,
        uptime_seconds,
        cpu_percent,
        memory_total_gb,
        memory_used_gb,
        disk_total_gb,
        disk_used_gb,
        local_ip,
    }
}

fn detect_local_ip() -> String {
    use sysinfo::Networks;
    let networks = Networks::new_with_refreshed_list();
    for (_name, data) in &networks {
        for ip in data.ip_networks() {
            if !ip.addr.is_loopback() && ip.addr.is_ipv4() {
                return ip.to_string();
            }
        }
    }
    String::new()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_collect_returns_data() {
        let data = collect();
        assert!(!data.hostname.is_empty());
        assert!(data.memory_total_gb > 0.0);
        assert!(data.disk_total_gb > 0.0);
    }
}
