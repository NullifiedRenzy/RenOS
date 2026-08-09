#!/usr/bin/env bash
#===============================================================================
# RenOS Validation System
#   Pre-installation and post-installation validation checks.
#   Usage: renos-validate.sh [--pre|--post|--json|--report FILE|--verbose|--quick]
#===============================================================================
set -euo pipefail

# ── Colors ────────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'
info()  { echo -e "${BLUE}[INFO]${NC}  $*"; }
ok()    { echo -e "${GREEN}[PASS]${NC} $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC} $*" >&2; }
err()   { echo -e "${RED}[FAIL]${NC} $*" >&2; }
die()   { err "$@"; exit 2; }
header(){ echo -e "\n${BOLD}${CYAN}══ $* ══${NC}\n"; }
sub()   { echo -e "\n${CYAN}── $* ──${NC}"; }

# ── Mode flags ────────────────────────────────────────────────────────────────
MODE=""               # pre | post | both (default both)
JSON_OUTPUT=false
REPORT_FILE=""
VERBOSE=false
QUICK=false
EXIT_CODE=0
GLOBAL_WARNINGS=0
GLOBAL_FAILURES=0
RESULTS_JSON='[]'

# ── Utility: check command existence ─────────────────────────────────────────
__has()  { command -v "$1" &>/dev/null; }
__ver_ge() {
  local actual=$1; local required=$2
  printf '%s\n' "$required" "$actual" | sort -V -C
}

# ── Result accumulator ───────────────────────────────────────────────────────
__record() {
  local status=$1; local check=$2; local message=$3
  case "$status" in
    pass) ((GLOBAL_WARNINGS)) || true; message="${message:-Check passed}" ;;
    warn) ((GLOBAL_WARNINGS++)) || true ;;
    fail) ((GLOBAL_FAILURES++)) || true ;;
  esac
  if $JSON_OUTPUT || [[ -n "$REPORT_FILE" ]]; then
    RESULTS_JSON=$(python3 -c "
import json
r = json.loads('$RESULTS_JSON')
r.append({'status': '$status', 'check': '$check', 'message': '$message'})
print(json.dumps(r))
" 2>/dev/null || echo "$RESULTS_JSON")
  fi
}

# ── JSON report output ───────────────────────────────────────────────────────
__write_report() {
  local final_status="pass"
  if [[ $GLOBAL_FAILURES -gt 0 ]]; then final_status="fail"
  elif [[ $GLOBAL_WARNINGS -gt 0 ]]; then final_status="warn"
  fi
  local summary
  summary=$(python3 -c "
import json
s = {
  'status': '$final_status',
  'timestamp': '$(date -Iseconds)',
  'hostname': '$(hostname)',
  'mode': '${MODE:-all}',
  'failures': $GLOBAL_FAILURES,
  'warnings': $GLOBAL_WARNINGS,
  'results': json.loads('$RESULTS_JSON')
}
print(json.dumps(s, indent=2))
" 2>/dev/null || echo '{"error":"report generation failed"}')

  if [[ -n "$REPORT_FILE" ]]; then
    echo "$summary" > "$REPORT_FILE"
    ok "Report written to $REPORT_FILE"
  fi

  if $JSON_OUTPUT; then
    echo "$summary"
  fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# PRE-INSTALLATION CHECKS
# ═══════════════════════════════════════════════════════════════════════════════

# ── CPU Checks ────────────────────────────────────────────────────────────────
check_cpu() {
  sub "CPU"
  local cores; cores=$(nproc 2>/dev/null || getconf _NPROCESSORS_ONLN || echo 0)
  if [[ "$cores" -ge 2 ]]; then
    ok "CPU cores: $cores (minimum 2)"
    __record pass "cpu_cores" "$cores cores available"
  else
    err "CPU cores: $cores (minimum 2 required)"
    __record fail "cpu_cores" "Insufficient CPU cores: $cores (need 2)"
  fi

  local sse42=false; local aesni=false
  if [[ -f /proc/cpuinfo ]]; then
    if grep -q 'sse4_2' /proc/cpuinfo 2>/dev/null; then sse42=true; fi
    if grep -q 'aes' /proc/cpuinfo 2>/dev/null; then aesni=true; fi
  fi
  if $QUICK; then
    $sse42 && ok "SSE4.2: supported" || warn "SSE4.2: not detected (may affect performance)"
    $aesni && ok "AES-NI: supported" || warn "AES-NI: not detected (may affect encryption perf)"
    __record warn "cpu_flags" "Quick mode — CPU flags not comprehensively checked"
  else
    if $sse42; then ok "SSE4.2: supported"; __record pass "cpu_sse42" "supported"
    else err "SSE4.2: NOT supported (required)"; __record fail "cpu_sse42" "SSE4.2 required but not found"; fi
    if $aesni; then ok "AES-NI: supported"; __record pass "cpu_aesni" "supported"
    else err "AES-NI: NOT supported (required)"; __record fail "cpu_aesni" "AES-NI required but not found"; fi
  fi
}

# ── RAM ───────────────────────────────────────────────────────────────────────
check_ram() {
  sub "Memory"
  local total_kb; total_kb=$(awk '/MemTotal/ {print $2}' /proc/meminfo 2>/dev/null || echo 0)
  local total_gb=$(( total_kb / 1024 / 1024 ))
  if [[ "$total_gb" -ge 4 ]]; then
    ok "RAM: ${total_gb}GB (minimum 4GB)"
    __record pass "ram" "${total_gb}GB available"
  else
    err "RAM: ${total_gb}GB (minimum 4GB required)"
    __record fail "ram" "Insufficient RAM: ${total_gb}GB (need 4GB)"
  fi

  local avail_kb; avail_kb=$(awk '/MemAvailable/ {print $2}' /proc/meminfo 2>/dev/null || echo 0)
  local avail_gb=$(( avail_kb / 1024 / 1024 ))
  info "Available: ${avail_gb}GB"
}

# ── Disk ──────────────────────────────────────────────────────────────────────
check_disk() {
  sub "Disk Space"
  local min_kb=$(( 20 * 1024 * 1024 ))  # 20GB in KB
  local paths=("/var/lib" "/etc" "/tmp")
  local all_ok=true

  for p in "${paths[@]}"; do
    if [[ ! -d "$p" ]]; then warn "Path $p does not exist — skipping"; continue; fi
    local avail_kb; avail_kb=$(df -k "$p" 2>/dev/null | awk 'NR==2 {print $4}')
    local avail_gb=$(( avail_kb / 1024 / 1024 ))
    if [[ "$avail_kb" -ge "$min_kb" ]]; then
      ok "${p}: ${avail_gb}GB free (minimum 20GB)"
      __record pass "disk_${p//\//_}" "${avail_gb}GB free on $p"
    else
      err "${p}: ${avail_gb}GB free (minimum 20GB required)"
      __record fail "disk_${p//\//_}" "Insufficient space on $p: ${avail_gb}GB (need 20GB)"
      all_ok=false
    fi
  done

  $all_ok || ((EXIT_CODE)) || true
}

# ── Ports ─────────────────────────────────────────────────────────────────────
check_ports() {
  sub "Port Availability"
  local ports=(80 443 8443 5432 6379 4222)
  local conflicts=()
  local tool=""

  if __has ss; then tool="ss"
  elif __has netstat; then tool="netstat"
  else warn "Neither ss nor netstat found — cannot check ports"; return; fi

  for port in "${ports[@]}"; do
    local in_use=false
    if [[ "$tool" == "ss" ]]; then
      ss -tlnp "sport = :$port" 2>/dev/null | grep -q LISTEN && in_use=true
    else
      netstat -tlnp 2>/dev/null | grep -q ":$port " && in_use=true
    fi
    if $in_use; then
      local owner=""
      if [[ "$tool" == "ss" ]]; then
        owner=$(ss -tlnp "sport = :$port" 2>/dev/null | awk -F'"' '/users:/{print $2}')
      else
        owner=$(netstat -tlnp 2>/dev/null | grep ":$port " | awk '{print $7}')
      fi
      warn "Port $port in use by ${owner:-unknown process}"
      conflicts+=("$port")
    else
      ok "Port $port available"
    fi
  done

  if [[ ${#conflicts[@]} -eq 0 ]]; then
    __record pass "ports" "All required ports available"
  else
    __record fail "ports" "Port conflicts: ${conflicts[*]}"
    ((EXIT_CODE)) || true
  fi
}

# ── Kernel ────────────────────────────────────────────────────────────────────
check_kernel() {
  sub "Kernel"
  local kernel_ver; kernel_ver=$(uname -r | cut -d- -f1)
  if __ver_ge "$kernel_ver" "5.0"; then
    ok "Kernel version: $kernel_ver (minimum 5.x)"
    __record pass "kernel_version" "Kernel $kernel_ver"
  else
    err "Kernel version: $kernel_ver (minimum 5.x required)"
    __record fail "kernel_version" "Kernel $kernel_ver too old (need 5.x+)"
  fi

  local required_modules=(overlay br_netfilter)
  for mod in "${required_modules[@]}"; do
    if lsmod 2>/dev/null | grep -q "^$mod"; then
      ok "Kernel module: $mod loaded"
      __record pass "kernel_mod_$mod" "Loaded"
    else
      warn "Kernel module: $mod not loaded (may be needed for containers)"
      __record warn "kernel_mod_$mod" "Not loaded"
    fi
  done
}

# ── Docker ────────────────────────────────────────────────────────────────────
check_docker() {
  sub "Docker"

  if __has docker; then
    ok "Docker CLI found"
    __record pass "docker_cli" "Installed"

    local docker_ver
    docker_ver=$(docker version --format '{{.Server.Version}}' 2>/dev/null || docker --version 2>/dev/null | awk '{print $3}' | sed 's/,//')
    if [[ -n "$docker_ver" ]]; then
      if __has sort && printf '%s\n' "24" "$docker_ver" | sort -V -C; then
        ok "Docker version: $docker_ver (>= 24)"
        __record pass "docker_version" "$docker_ver"
      else
        warn "Docker version: $docker_ver (24+ recommended)"
        __record warn "docker_version" "Version $docker_ver (24+ recommended)"
      fi
    fi

    if docker info &>/dev/null; then
      ok "Docker daemon is running"
      __record pass "docker_running" "Daemon running"
    else
      err "Docker daemon is NOT running"
      __record fail "docker_running" "Docker daemon not running"
    fi
  else
    err "Docker CLI not found"
    __record fail "docker_cli" "Docker not installed"
  fi

  if __has docker && docker compose version &>/dev/null; then
    ok "Docker Compose v2 found"
    __record pass "docker_compose" "v2 found"
  elif docker-compose --version &>/dev/null; then
    warn "Docker Compose v1 found (v2 recommended)"
    __record warn "docker_compose" "v1 found, v2 recommended"
  else
    warn "Docker Compose not found"
    __record warn "docker_compose" "Not found"
  fi
}

# ── Filesystem ────────────────────────────────────────────────────────────────
check_filesystem() {
  sub "Filesystem"
  local paths=("$RENOS_DATA" "/var/lib/docker")
  for p in "${paths[@]}"; do
    if [[ ! -d "$p" ]]; then
      p=$(dirname "$p")
    fi
    if [[ -d "$p" ]]; then
      local fstype; fstype=$(df -T "$p" 2>/dev/null | awk 'NR==2 {print $2}')
      local mount_opts; mount_opts=$(awk -v p="$p" '$2 == p {print $4}' /proc/mounts 2>/dev/null | head -1)
      case "$fstype" in
        ext4|xfs) ok "${p}: ${fstype} (recommended)" ; __record pass "fs_${p//\//_}" "${fstype}" ;;
        *) warn "${p}: ${fstype} (ext4 or xfs recommended)" ; __record warn "fs_${p//\//_}" "${fstype}" ;;
      esac
      info "  Mount options: ${mount_opts:-unknown}"
    fi
  done
}

# ── Certificates ──────────────────────────────────────────────────────────────
check_certs() {
  sub "TLS Certificates"
  local cert_dirs=("${RENOS_ETC}/certs" "/etc/ssl/certs")
  local found=false

  for dir in "${cert_dirs[@]}"; do
    if [[ -d "$dir" ]]; then
      while IFS= read -r -d '' cert; do
        found=true
        if __has openssl; then
          local expiry
          expiry=$(openssl x509 -enddate -noout -in "$cert" 2>/dev/null | cut -d= -f2-)
          local subject; subject=$(openssl x509 -subject -noout -in "$cert" 2>/dev/null | sed 's/subject= //')
          local expiry_epoch
          expiry_epoch=$(date -d "$expiry" +%s 2>/dev/null || echo 0)
          local now; now=$(date +%s)
          if [[ "$expiry_epoch" -gt "$now" ]]; then
            local days_left=$(( (expiry_epoch - now) / 86400 ))
            if [[ "$days_left" -lt 30 ]]; then
              warn "Certificate expires soon: $(basename "$cert") — ${days_left} days"
              __record warn "cert_$(basename "$cert")" "Expires in ${days_left} days"
            else
              ok "$(basename "$cert") — valid, ${days_left} days remaining"
              __record pass "cert_$(basename "$cert")" "Valid, ${days_left}d left"
            fi
          else
            err "$(basename "$cert") — EXPIRED on $expiry"
            __record fail "cert_$(basename "$cert")" "Expired $expiry"
          fi
          # Validate chain
          if openssl verify -CAfile "${dir}/ca.crt" "$cert" &>/dev/null; then
            ok "  Chain: valid"
          else
            local ca_file="${dir}/ca.crt"
            if [[ -f "$ca_file" ]]; then
              if openssl verify -CAfile "$ca_file" "$cert" &>/dev/null; then
                ok "  Chain: valid (with CA)"
              else
                warn "  Chain: could not verify"
                __record warn "cert_chain_$(basename "$cert")" "Chain verification issue"
              fi
            fi
          fi
        else
          info "$(basename "$cert") — openssl not available to check"
        fi
      done < <(find "$dir" -maxdepth 1 -type f \( -name '*.crt' -o -name '*.pem' \) -print0 2>/dev/null)
    fi
  done

  $found || warn "No TLS certificates found for inspection"
}

# ── Firewall ──────────────────────────────────────────────────────────────────
check_firewall() {
  sub "Firewall"

  if __has ufw; then
    local ufw_status; ufw_status=$(ufw status 2>/dev/null)
    if echo "$ufw_status" | grep -qi "active"; then
      ok "UFW is active"
      __record pass "firewall" "UFW active"
      for port in 80 443 8443; do
        if echo "$ufw_status" | grep -q "$port"; then
          ok "  Port $port allowed"
        else
          warn "  Port $port not explicitly allowed"
        fi
      done
    else
      warn "UFW is inactive"
      __record warn "firewall" "UFW inactive"
    fi
  elif __has firewall-cmd; then
    if firewall-cmd --state &>/dev/null; then
      ok "firewalld is active"
      __record pass "firewall" "firewalld active"
      local zone; zone=$(firewall-cmd --get-default-zone 2>/dev/null)
      for port in 80/tcp 443/tcp 8443/tcp; do
        if firewall-cmd --zone="$zone" --query-port="$port" &>/dev/null; then
          ok "  Port $port allowed"
        else
          warn "  Port $port not allowed in $zone zone"
        fi
      done
    else
      warn "firewalld is inactive"
      __record warn "firewall" "firewalld inactive"
    fi
  elif __has iptables; then
    if iptables -L -n --line-numbers 2>/dev/null | head -5 | grep -q "Chain"; then
      ok "iptables rules present"
      __record pass "firewall" "iptables rules present"
    else
      warn "No iptables rules detected"
      __record warn "firewall" "No iptables rules"
    fi
  else
    warn "No firewall tool detected (ufw/firewall-cmd/iptables)"
    __record warn "firewall" "No firewall tool detected"
  fi
}

# ── DNS ───────────────────────────────────────────────────────────────────────
check_dns() {
  sub "DNS"

  local hostname; hostname=$(hostname 2>/dev/null || echo "localhost")
  local fqdn; fqdn=$(hostname -f 2>/dev/null || echo "$hostname")

  if __has host || __has dig || __has nslookup; then
    if __has host; then
      local resolve; resolve=$(host "$fqdn" 2>/dev/null) && {
        ok "Forward DNS: $fqdn resolves"
        __record pass "dns_forward" "$fqdn resolves"
        local ip; ip=$(echo "$resolve" | awk '/has address/ {print $4}')
        if [[ -n "$ip" ]]; then
          local reverse; reverse=$(host "$ip" 2>/dev/null) && {
            ok "Reverse DNS: $ip resolves to $(echo "$reverse" | awk '/domain name pointer/ {print $5}')"
            __record pass "dns_reverse" "$ip resolves"
          } || warn "Reverse DNS: no PTR record for $ip"
        fi
      } || {
        warn "Forward DNS: $fqdn does not resolve"
        __record warn "dns_forward" "$fqdn does not resolve"
      }
    else
      warn "DNS resolution check requires 'host' command"
    fi
  else
    warn "DNS lookup tools not available"
    __record warn "dns_tools" "No DNS tools (host/dig/nslookup)"
  fi
}

# ── Network ───────────────────────────────────────────────────────────────────
check_network() {
  sub "Network"

  if __has curl; then
    if curl -s --connect-timeout 5 -o /dev/null -w '%{http_code}' https://google.com &>/dev/null; then
      ok "Internet connectivity: yes"
      __record pass "network_internet" "Internet reachable"
    else
      warn "Internet connectivity: limited or none"
      __record warn "network_internet" "Internet not reachable"
    fi

    local latency
    latency=$(curl -s --connect-timeout 5 -o /dev/null -w '%{time_total}' https://google.com 2>/dev/null || echo "0")
    if [[ -n "$latency" && "$latency" != "0" ]]; then
      local latency_ms; latency_ms=$(echo "$latency * 1000" | bc 2>/dev/null | cut -d. -f1)
      if [[ -n "$latency_ms" && "$latency_ms" -lt 200 ]]; then
        ok "Latency: ${latency_ms}ms to google.com"
      else
        warn "Latency: ${latency_ms:-unknown}ms to google.com"
      fi
    fi
  else
    warn "curl not available — skipping network checks"
    __record warn "network_tools" "curl not available"
  fi
}

# ── SELinux / AppArmor ───────────────────────────────────────────────────────
check_mandatory_access() {
  sub "Mandatory Access Control"

  if __has getenforce; then
    local mode; mode=$(getenforce 2>/dev/null)
    case "$mode" in
      Enforcing)
        ok "SELinux: Enforcing"
        __record pass "selinux" "Enforcing"
        # Check if any renos-specific policies exist
        if semodule -l 2>/dev/null | grep -q renos; then
          ok "  RenOS SELinux policy loaded"
        else
          warn "  No RenOS-specific SELinux policy found — may need to create one"
        fi
        ;;
      Permissive)
        warn "SELinux: Permissive (Enforcing recommended)"
        __record warn "selinux" "Permissive mode"
        ;;
      Disabled)
        warn "SELinux: Disabled (Enforcing recommended)"
        __record warn "selinux" "Disabled"
        ;;
    esac
  elif __has aa-status; then
    local aa_status; aa_status=$(aa-status 2>/dev/null)
    if echo "$aa_status" | grep -qi "enforcing"; then
      ok "AppArmor: Enforcing"
      __record pass "apparmor" "Enforcing"
    elif echo "$aa_status" | grep -qi "complain"; then
      warn "AppArmor: Complain mode"
      __record warn "apparmor" "Complain mode"
    else
      if aa-status --enabled &>/dev/null; then
        ok "AppArmor: enabled"
        __record pass "apparmor" "Enabled"
      else
        warn "AppArmor: not enabled"
        __record warn "apparmor" "Not enabled"
      fi
    fi
  else
    warn "Neither SELinux nor AppArmor detected"
    __record warn "mac" "No MAC system detected"
  fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# POST-INSTALLATION CHECKS
# ═══════════════════════════════════════════════════════════════════════════════

# ── Services (systemd) ───────────────────────────────────────────────────────
check_services() {
  sub "Services (systemd)"
  local services=("renos-api" "renos-worker" "renos-scheduler" "renos-nats" "postgresql" "redis")
  if __has docker; then
    local docker_containers
    docker_containers=$(docker ps --format '{{.Names}}' 2>/dev/null | grep renos || true)
    if [[ -n "$docker_containers" ]]; then
      services=()
      while IFS= read -r c; do services+=("docker:$c"); done <<< "$docker_containers"
    fi
  fi

  for svc in "${services[@]}"; do
    if [[ "$svc" == docker:* ]]; then
      local name="${svc#docker:}"
      if docker inspect --format='{{.State.Status}}' "$name" 2>/dev/null | grep -q "running"; then
        ok "Docker container: $name is running"
        __record pass "service_$name" "Running"
      else
        err "Docker container: $name is NOT running"
        __record fail "service_$name" "Not running"
      fi
    else
      if systemctl is-active --quiet "$svc" 2>/dev/null; then
        ok "$svc is running"
        __record pass "service_$svc" "Active"
      else
        local enabled; enabled=$(systemctl is-enabled "$svc" 2>/dev/null || echo "unknown")
        if [[ "$enabled" == "enabled" ]]; then
          err "$svc is NOT running (but enabled)"
          __record fail "service_$svc" "Not running but enabled"
        else
          warn "$svc is not present or disabled"
          __record warn "service_$svc" "Not present/disabled"
        fi
      fi
    fi
  done
}

# ── API healthz ──────────────────────────────────────────────────────────────
check_api() {
  sub "API Health"
  if ! __has curl; then warn "curl not available — skipping API checks"; return; fi

  local endpoints=("https://localhost:8443/healthz" "http://localhost:80/healthz" "https://localhost:443/healthz")
  local found=false
  for ep in "${endpoints[@]}"; do
    local http_code
    http_code=$(curl -sk --connect-timeout 5 -o /dev/null -w '%{http_code}' "$ep" 2>/dev/null || echo "000")
    if [[ "$http_code" == "200" ]]; then
      ok "API responds at $ep (HTTP 200)"
      __record pass "api_healthz" "Responds at $ep"
      found=true
      break
    elif [[ "$http_code" != "000" ]]; then
      warn "API at $ep responded HTTP $http_code (expected 200)"
      __record warn "api_healthz_${ep//\//_}" "HTTP $http_code"
      found=true
    fi
  done
  $found || { err "API health endpoint not reachable on any port"; __record fail "api_healthz" "Not reachable"; }
}

# ── Database ─────────────────────────────────────────────────────────────────
check_database() {
  sub "Database"
  if __has psql; then
    if PGPASSWORD="${PGPASSWORD:-}" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT 1;" &>/dev/null; then
      ok "PostgreSQL reachable at ${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}"
      __record pass "postgresql" "Reachable"
    else
      err "PostgreSQL NOT reachable at ${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}"
      __record fail "postgresql" "Unreachable"
    fi
  else
    warn "psql not available — skipping database check"
    __record warn "postgresql" "psql not available"
  fi
}

# ── Redis ────────────────────────────────────────────────────────────────────
check_redis() {
  sub "Redis"
  if __has redis-cli; then
    if redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" PING 2>/dev/null | grep -q "PONG"; then
      ok "Redis reachable at ${REDIS_HOST}:${REDIS_PORT}"
      __record pass "redis" "Reachable"
    else
      err "Redis NOT reachable at ${REDIS_HOST}:${REDIS_PORT}"
      __record fail "redis" "Unreachable"
    fi
  else
    warn "redis-cli not available — skipping Redis check"
    __record warn "redis" "redis-cli not available"
  fi
}

# ── NATS ─────────────────────────────────────────────────────────────────────
check_nats() {
  sub "NATS"
  if __has nats; then
    if nats server check --server nats://localhost:4222 &>/dev/null; then
      ok "NATS reachable at localhost:4222"
      __record pass "nats" "Reachable"
    else
      err "NATS NOT reachable at localhost:4222"
      __record fail "nats" "Unreachable"
    fi
  elif __has curl; then
    # Try NATS HTTP monitoring
    if curl -s --connect-timeout 3 http://localhost:8222/healthz &>/dev/null; then
      ok "NATS monitoring endpoint reachable"
      __record pass "nats" "Monitoring reachable"
    else
      warn "NATS monitoring not available on :8222 — may use different port"
      __record warn "nats" "Monitoring not found"
    fi
  else
    warn "Neither nats CLI nor curl available — skipping NATS check"
    __record warn "nats" "Tools not available"
  fi
}

# ── WebSocket ─────────────────────────────────────────────────────────────────
check_websocket() {
  sub "WebSocket"
  if __has python3; then
    if python3 -c "import websocket" 2>/dev/null; then
      local ws_hosts=("wss://localhost:8443/ws" "ws://localhost:80/ws")
      local found=false
      for ws in "${ws_hosts[@]}"; do
        if python3 -c "
import websocket, sys
try:
    ws = websocket.create_connection('$ws', timeout=5)
    ws.close()
    sys.exit(0)
except:
    sys.exit(1)
" &>/dev/null; then
          ok "WebSocket connection to $ws works"
          __record pass "websocket" "Connected to $ws"
          found=true
          break
        fi
      done
      $found || { warn "WebSocket connections failed on all endpoints"; __record warn "websocket" "No connection"; }
    else
      warn "Python websocket-client not installed — skipping WebSocket check"
      __record warn "websocket" "websocket-client not installed"
    fi
  else
    warn "python3 not available — skipping WebSocket check"
    __record warn "websocket" "python3 not available"
  fi
}

# ── TLS ──────────────────────────────────────────────────────────────────────
check_tls_post() {
  sub "TLS Certificate (Live)"
  if ! __has curl && ! __has openssl; then
    warn "curl/openssl not available — skipping live TLS check"
    return
  fi

  if __has curl; then
    local cert_info
    cert_info=$(curl -skv https://localhost:8443/healthz 2>&1 | grep -i "server certificate" || true)
    if [[ -n "$cert_info" ]]; then
      ok "TLS handshake successful on :8443"
      __record pass "tls_live" "Handshake OK"
    else
      local http_code
      http_code=$(curl -sk -o /dev/null -w '%{http_code}' --connect-timeout 5 https://localhost:8443/ 2>/dev/null || echo "000")
      if [[ "$http_code" != "000" ]]; then
        ok "TLS endpoint responds (HTTP $http_code)"
        __record pass "tls_live" "Responds HTTP $http_code"
      else
        warn "TLS endpoint not responding — may not be running"
        __record warn "tls_live" "Not responding"
      fi
    fi
  fi
}

# ── Auth ──────────────────────────────────────────────────────────────────────
check_auth() {
  sub "Authentication"
  if ! __has curl; then warn "curl not available — skipping auth check"; return; fi

  local api_url="https://localhost:8443"
  local test_user="${RENOS_ADMIN_USER:-admin}"
  local test_pass="${RENOS_ADMIN_PASS:-admin}"

  if [[ -f "${RENOS_ETC}/credentials" ]]; then
    test_user=$(grep -o 'username=[^ ]*' "${RENOS_ETC}/credentials" 2>/dev/null | cut -d= -f2 || echo "$test_user")
    test_pass=$(grep -o 'password=[^ ]*' "${RENOS_ETC}/credentials" 2>/dev/null | cut -d= -f2 || echo "$test_pass")
  fi

  local login_result
  login_result=$(curl -sk --connect-timeout 5 -X POST \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"$test_user\",\"password\":\"$test_pass\"}" \
    "${api_url}/api/login" 2>/dev/null || echo "")

  if echo "$login_result" | grep -qi "token\|success\|200"; then
    ok "Admin user '$test_user' can authenticate"
    __record pass "auth" "Admin authentication works"
  else
    warn "Admin user '$test_user' could not authenticate at ${api_url}/api/login"
    __record warn "auth" "Login failed"
  fi
}

# ── Dashboard ────────────────────────────────────────────────────────────────
check_dashboard() {
  sub "Dashboard"
  if ! __has curl; then warn "curl not available — skipping dashboard check"; return; fi

  local urls=("https://localhost:8443" "http://localhost:80" "https://localhost:443")
  local found=false
  for url in "${urls[@]}"; do
    local http_code
    http_code=$(curl -sk --connect-timeout 5 -o /dev/null -w '%{http_code}' "$url" 2>/dev/null || echo "000")
    if [[ "$http_code" == "200" ]]; then
      ok "Dashboard loads at $url (HTTP 200)"
      __record pass "dashboard" "Loads at $url"
      found=true
      break
    elif [[ "$http_code" == "302" || "$http_code" == "301" ]]; then
      ok "Dashboard at $url redirects (HTTP $http_code)"
      found=true
    fi
  done
  $found || { warn "Dashboard not reachable on any URL"; __record warn "dashboard" "Not reachable"; }
}

# ── Agent connection ─────────────────────────────────────────────────────────
check_agent() {
  sub "Agent Connection"
  if __has curl; then
    local api_url="https://localhost:8443"
    local agents
    agents=$(curl -sk --connect-timeout 5 "${api_url}/api/agents" 2>/dev/null || echo "")
    if echo "$agents" | grep -qi "connected\|online\|\"agents\""; then
      ok "Agent endpoint responds with data"
      __record pass "agent" "Connected"
    else
      warn "No agents connected or agent API not reachable"
      __record warn "agent" "No connected agents"
    fi
  fi
}

# ── Plugin system ────────────────────────────────────────────────────────────
check_plugins() {
  sub "Plugin System"
  if [[ -d "$PLUGIN_PATH" ]]; then
    local plugin_count
    plugin_count=$(find "$PLUGIN_PATH" -maxdepth 1 -type d 2>/dev/null | wc -l)
    if [[ "$plugin_count" -gt 1 ]]; then
      ok "$((plugin_count - 1)) plugin(s) installed"
      __record pass "plugins" "$((plugin_count - 1)) installed"
    else
      warn "No plugins installed in $PLUGIN_PATH"
      __record warn "plugins" "No plugins installed"
    fi
  else
    warn "Plugin directory $PLUGIN_PATH does not exist"
    __record warn "plugins" "Directory not found"
  fi

  if [[ -f "${RENOS_ETC}/plugins.json" ]]; then
    info "Plugin config found: ${RENOS_ETC}/plugins.json"
  fi
}

# ═══════════════════════════════════════════════════════════════════════════════
# MAIN
# ═══════════════════════════════════════════════════════════════════════════════

usage() {
  cat <<-EOF
${BOLD}RenOS Validation System${NC}
Usage: $(basename "$0") [OPTIONS]

${BOLD}Options:${NC}
  --pre            Run pre-installation checks only
  --post           Run post-installation checks only
  --json           Output results as JSON
  --report FILE    Write report to FILE
  --verbose        Show detailed output for each check
  --quick          Fast mode (skip expensive checks)
  --help           Show this help message

${BOLD}Exit Codes:${NC}
  0   All checks passed
  1   Warnings (non-critical issues found)
  2   Failures (critical issues found)

${BOLD}Examples:${NC}
  $(basename "$0") --pre
  $(basename "$0") --post --json
  $(basename "$0") --pre --report /tmp/renos-validation.json
  $(basename "$0") --post --verbose
  $(basename "$0") --quick
EOF
  exit 0
}

main() {
  [[ $# -eq 0 ]] && MODE="all"

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --pre) MODE="pre"; shift ;;
      --post) MODE="post"; shift ;;
      --json) JSON_OUTPUT=true; shift ;;
      --report) shift; REPORT_FILE="$1"; shift ;;
      --verbose) VERBOSE=true; shift ;;
      --quick) QUICK=true; shift ;;
      --help|-h) usage ;;
      *) die "Unknown option: $1" ;;
    esac
  done

  # Root check — only warn for pre, require for post
  if [[ "$MODE" == "post" ]] && [[ $EUID -ne 0 ]]; then
    warn "Some post-installation checks require root access"
  fi

  # Source config for paths
  SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  CONFIG_PATH="${SCRIPT_DIR}/../backup/renos-backup-config.sh"
  [[ -f "$CONFIG_PATH" ]] && source "$CONFIG_PATH" 2>/dev/null

  header "RenOS Validation — $(date)"
  info "Hostname: $(hostname)"
  info "Mode: ${MODE:-all}"
  $QUICK && info "Quick mode: enabled"

  if [[ "$MODE" == "pre" || "$MODE" == "all" ]]; then
    header "PRE-INSTALLATION CHECKS"
    check_cpu
    check_ram
    check_disk
    check_ports
    check_kernel
    check_docker
    check_filesystem
    check_certs
    check_firewall
    check_dns
    check_network
    check_mandatory_access
  fi

  if [[ "$MODE" == "post" || "$MODE" == "all" ]]; then
    header "POST-INSTALLATION CHECKS"
    check_services
    check_api
    check_database
    check_redis
    check_nats
    check_websocket
    check_tls_post
    check_auth
    check_dashboard
    check_agent
    check_plugins
  fi

  # Summary
  header "Validation Summary"
  if [[ $GLOBAL_FAILURES -gt 0 ]]; then
    echo -e "  ${RED}${GLOBAL_FAILURES} failure(s)${NC}"
  fi
  if [[ $GLOBAL_WARNINGS -gt 0 ]]; then
    echo -e "  ${YELLOW}${GLOBAL_WARNINGS} warning(s)${NC}"
  fi
  if [[ $GLOBAL_FAILURES -eq 0 && $GLOBAL_WARNINGS -eq 0 ]]; then
    echo -e "  ${GREEN}All checks passed${NC}"
  fi

  # Write report
  if $JSON_OUTPUT || [[ -n "$REPORT_FILE" ]]; then
    __write_report
  fi

  # Exit code
  if [[ $GLOBAL_FAILURES -gt 0 ]]; then
    exit 2
  elif [[ $GLOBAL_WARNINGS -gt 0 ]]; then
    exit 1
  fi
  exit 0
}

main "$@"
