use std::net::UdpSocket;
use std::process::Command;

#[derive(Debug, Clone, serde::Serialize, serde::Deserialize)]
pub struct InterfaceInfo {
    pub name: String,
    pub mac_address: String,
    pub ipv4: Vec<String>,
    pub ipv6: Vec<String>,
    pub status: String,
    pub speed: i64,
    pub interface_type: String,
    pub is_default_gateway: bool,
    pub dns_servers: Vec<String>,
    pub mtu: i32,
}

#[derive(serde::Deserialize)]
struct IpAddrEntry {
    #[serde(default)]
    ifname: String,
    #[serde(default)]
    address: String,
    #[serde(default)]
    flags: Vec<String>,
    #[serde(default)]
    mtu: i32,
    #[serde(default)]
    link_type: String,
    #[allow(dead_code)]
    #[serde(default)]
    operstate: String,
    #[serde(default)]
    addr_info: Vec<AddrInfo>,
}

#[derive(serde::Deserialize)]
struct AddrInfo {
    #[serde(default)]
    family: String,
    #[serde(default)]
    local: String,
    #[serde(default)]
    prefixlen: u8,
    #[serde(default)]
    scope: String,
}

#[derive(serde::Deserialize)]
struct IpRouteEntry {
    #[serde(default)]
    dst: String,
    #[serde(default)]
    dev: String,
}

#[derive(Debug)]
pub enum NetworkError {
    CommandFailed(String),
    Parse(String),
    NoOutput,
}

impl std::fmt::Display for NetworkError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            NetworkError::CommandFailed(msg) => write!(f, "command failed: {msg}"),
            NetworkError::Parse(msg) => write!(f, "parse error: {msg}"),
            NetworkError::NoOutput => write!(f, "no output from command"),
        }
    }
}

fn run_ip(args: &[&str]) -> Result<String, NetworkError> {
    let output = Command::new("ip")
        .args(args)
        .output()
        .map_err(|e| NetworkError::CommandFailed(e.to_string()))?;

    if !output.status.success() {
        let stderr = String::from_utf8_lossy(&output.stderr);
        return Err(NetworkError::CommandFailed(stderr.trim().to_string()));
    }

    let stdout = String::from_utf8_lossy(&output.stdout).to_string();
    if stdout.trim().is_empty() {
        return Err(NetworkError::NoOutput);
    }
    Ok(stdout)
}

fn read_dns_servers() -> Vec<String> {
    let mut servers = Vec::new();

    // Try resolvectl first (systemd-resolved)
    if let Ok(output) = Command::new("resolvectl").arg("status").output() {
        if output.status.success() {
            let stdout = String::from_utf8_lossy(&output.stdout);
            for line in stdout.lines() {
                let line = line.trim();
                if let Some(val) = line.strip_prefix("DNS Servers: ") {
                    for s in val.split_whitespace() {
                        let s = s.trim_matches(|c| c == ' ' || c == ',');
                        if !s.is_empty() {
                            servers.push(s.to_string());
                        }
                    }
                }
            }
            if !servers.is_empty() {
                return servers;
            }
        }
    }

    // Fallback: parse /etc/resolv.conf
    if let Ok(content) = std::fs::read_to_string("/etc/resolv.conf") {
        for line in content.lines() {
            let line = line.trim();
            if let Some(val) = line.strip_prefix("nameserver ") {
                let val = val.trim();
                if !val.is_empty() {
                    servers.push(val.to_string());
                }
            }
        }
    }

    servers
}

fn read_interface_speed(name: &str) -> i64 {
    let path = format!("/sys/class/net/{name}/speed");
    if let Ok(content) = std::fs::read_to_string(&path) {
        if let Ok(speed) = content.trim().parse::<i64>() {
            if speed > 0 {
                return speed;
            }
        }
    }
    0
}

pub fn list_interfaces() -> Result<Vec<InterfaceInfo>, NetworkError> {
    let json_output = run_ip(&["-j", "addr"])?;
    let entries: Vec<IpAddrEntry> =
        serde_json::from_str(&json_output).map_err(|e| NetworkError::Parse(e.to_string()))?;

    // Determine default gateway interface
    let mut default_gateways: Vec<String> = Vec::new();
    if let Ok(route_output) = run_ip(&["-j", "route", "show", "default"]) {
        if let Ok(routes) = serde_json::from_str::<Vec<IpRouteEntry>>(&route_output) {
            for r in &routes {
                if r.dst == "default" {
                    default_gateways.push(r.dev.clone());
                }
            }
        }
    }

    let dns_servers = read_dns_servers();

    let mut interfaces: Vec<InterfaceInfo> = entries
        .iter()
        .map(|entry| {
            let is_up = entry.flags.iter().any(|f| f == "UP");
            let is_loopback = entry.link_type == "loopback" || entry.ifname == "lo";

            let mut ipv4: Vec<String> = Vec::new();
            let mut ipv6: Vec<String> = Vec::new();

            for addr in &entry.addr_info {
                match addr.family.as_str() {
                    "inet" => {
                        let cidr = format!("{}/{}", addr.local, addr.prefixlen);
                        if !ipv4.contains(&cidr) {
                            ipv4.push(cidr);
                        }
                    }
                    "inet6" if addr.scope != "link" => {
                        let cidr = format!("{}/{}", addr.local, addr.prefixlen);
                        if !ipv6.contains(&cidr) {
                            ipv6.push(cidr);
                        }
                    }
                    _ => {}
                }
            }

            InterfaceInfo {
                name: entry.ifname.clone(),
                mac_address: entry.address.clone(),
                ipv4,
                ipv6,
                status: if is_up {
                    "up".to_string()
                } else {
                    "down".to_string()
                },
                speed: if !is_loopback {
                    read_interface_speed(&entry.ifname)
                } else {
                    0
                },
                interface_type: if is_loopback {
                    "loopback".to_string()
                } else {
                    entry.link_type.clone()
                },
                is_default_gateway: default_gateways.contains(&entry.ifname),
                dns_servers: dns_servers.clone(),
                mtu: entry.mtu,
            }
        })
        .collect();

    interfaces.sort_by(|a, b| a.name.cmp(&b.name));
    Ok(interfaces)
}

/// Send a Wake-on-LAN magic packet to the given MAC address.
/// Broadcasts on UDP port 9 to 255.255.255.255.
#[allow(dead_code)]
pub fn send_wol(mac: &str) -> Result<String, NetworkError> {
    let mac_clean: String = mac.chars().filter(|c| c.is_ascii_hexdigit()).collect();
    if mac_clean.len() != 12 {
        return Err(NetworkError::CommandFailed(format!(
            "invalid MAC address: {mac}"
        )));
    }

    let mac_bytes: Vec<u8> = (0..6)
        .map(|i| u8::from_str_radix(&mac_clean[i * 2..i * 2 + 2], 16).unwrap())
        .collect();

    // Magic packet: 6 bytes of 0xFF + 16 repetitions of the MAC
    let mut packet = Vec::with_capacity(102);
    packet.extend_from_slice(&[0xFF; 6]);
    for _ in 0..16 {
        packet.extend_from_slice(&mac_bytes);
    }

    let socket = UdpSocket::bind("0.0.0.0:0")
        .map_err(|e| NetworkError::CommandFailed(format!("bind socket: {e}")))?;

    socket
        .set_broadcast(true)
        .map_err(|e| NetworkError::CommandFailed(format!("set broadcast: {e}")))?;

    socket
        .send_to(&packet, "255.255.255.255:9")
        .map_err(|e| NetworkError::CommandFailed(format!("send WOL packet: {e}")))?;

    Ok(format!("WOL packet sent to {mac}"))
}

#[derive(serde::Serialize)]
struct SyncRequest<'a> {
    device_id: &'a str,
    device_secret: &'a str,
    interfaces: &'a [InterfaceInfo],
}

pub async fn submit_interfaces(
    api_url: &str,
    device_id: &str,
    device_secret: &str,
    interfaces: &[InterfaceInfo],
) -> Result<(), String> {
    let client = reqwest::Client::new();
    let url = format!(
        "{}/api/v1/devices/{}/network-interfaces",
        api_url, device_id
    );

    let resp = client
        .post(&url)
        .json(&SyncRequest {
            device_id,
            device_secret,
            interfaces,
        })
        .timeout(std::time::Duration::from_secs(10))
        .send()
        .await
        .map_err(|e| format!("sync network interfaces request failed: {e}"))?;

    let status = resp.status();
    if !status.is_success() {
        let body = resp.text().await.unwrap_or_default();
        return Err(format!("sync interfaces rejected ({}): {}", status, body));
    }

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_list_interfaces_returns_data() {
        let result = list_interfaces();
        // May fail if `ip` command not available, but should succeed on Linux
        if let Ok(interfaces) = result {
            assert!(!interfaces.is_empty(), "should have at least lo");
            let lo = interfaces.iter().find(|i| i.name == "lo");
            assert!(lo.is_some(), "loopback should exist");
            if let Some(lo) = lo {
                assert_eq!(lo.interface_type, "loopback");
                assert!(lo.mac_address.is_empty() || lo.mac_address == "00:00:00:00:00:00");
            }
        }
    }

    #[test]
    fn test_list_interfaces_mac_address_format() {
        if let Ok(interfaces) = list_interfaces() {
            for iface in &interfaces {
                if !iface.mac_address.is_empty() && iface.interface_type != "loopback" {
                    let clean: String = iface
                        .mac_address
                        .chars()
                        .filter(|c| c.is_ascii_hexdigit())
                        .collect();
                    assert_eq!(
                        clean.len(),
                        12,
                        "MAC should be 12 hex chars: {}",
                        iface.mac_address
                    );
                }
            }
        }
    }

    #[test]
    fn test_send_wol_invalid_mac() {
        let result = send_wol("not-a-mac");
        assert!(result.is_err());
    }

    #[test]
    fn test_send_wol_valid_mac_format_accepted() {
        // Just tests that the function accepts the format
        // (won't actually succeed sending a WOL packet in CI, but the validation should pass)
        let result = send_wol("aa:bb:cc:dd:ee:ff");
        // May fail on send (no permission), but should not fail on MAC validation
        assert!(!result.is_err() || result.as_ref().err().unwrap().to_string().contains("send"));
    }
}
