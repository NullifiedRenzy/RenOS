#!/usr/bin/env bash
# =============================================================================
# RenOS Fresh Install Validation Script
# =============================================================================
# Validates a RenOS installation on a blank VM for:
#   Ubuntu 24.04, Ubuntu 22.04, Debian 13, Fedora 42
#
# It runs installer/linux/install.sh in NONINTERACTIVE mode and then verifies
# every component end-to-end (services, API, auth, DB/Redis/NATS, WebSocket,
# dashboard, TLS), recording PASS/FAIL/WARN into /tmp/renos-install-results.txt.
#
# Usage:
#   sudo ./fresh-install-test.sh                # full test (install + verify)
#   sudo ./fresh-install-test.sh --ci           # deterministic, CI-friendly
#   sudo ./fresh-install-test.sh --no-install   # verify an existing install
#   sudo ./fresh-install-test.sh --help
#
# Environment:
#   RENOS_ADMIN_EMAIL     admin email (default: admin@renos.local)
#   RENOS_ADMIN_PASSWORD  admin password (auto-generated if unset)
#   RENOS_EXTERNAL_URL    external URL (default: https://<hostname>)
#
# Exit codes:
#   0  all checks passed (warnings allowed)
#   1  one or more checks failed
#   2  usage error
# =============================================================================

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALLER="${SCRIPT_DIR}/install.sh"

RESULTS_FILE="${RENOS_RESULTS_FILE:-/tmp/renos-install-results.txt}"
INSTALL_LOG="${RENOS_INSTALL_LOG:-/tmp/renos-fresh-install.log}"
CREDS_FILE="${RENOS_CREDS_FILE:-/tmp/renos-admin-credentials.txt}"

CI_MODE=false
SKIP_INSTALL=false
CONTAINER_MODE=false

PASS_COUNT=0
FAIL_COUNT=0
WARN_COUNT=0

# ─── Help ──────────────────────────────────────────────────────────────────────
show_help() {
    cat <<EOF
RenOS Fresh Install Validation Script

Usage: $(basename "$0") [OPTIONS]

Options:
  --ci              Deterministic output (no ANSI colors), exit 1 on any failure
  --no-install      Skip the installer; only verify an existing installation
  --container       Systemd-less container (CI): start Docker + stack manually
  --installer PATH  Path to installer/linux/install.sh (default: alongside this script)
  --help, -h        Show this help message

Environment:
  RENOS_ADMIN_EMAIL     Admin email (default: admin@renos.local)
  RENOS_ADMIN_PASSWORD  Admin password (auto-generated if unset)
  RENOS_EXTERNAL_URL    Public URL (default: https://<hostname>)

Exit codes:
  0   All checks passed
  1   One or more checks failed
  2   Usage error
EOF
    exit 0
}

# ─── Output helpers ────────────────────────────────────────────────────────────
if [ -t 1 ] && [ "${CI:-0}" != "1" ]; then
    RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
    BLUE='\033[0;34m'; BOLD='\033[1m'; NC='\033[0m'
else
    RED=''; GREEN=''; YELLOW=''; BLUE=''; BOLD=''; NC=''
fi

info()  { echo -e "${BLUE}ℹ${NC}  $*"; }
ok()    { echo -e "${GREEN}✓${NC}  $*"; }
warn()  { echo -e "${YELLOW}⚠${NC}  $*"; }
fail()  { echo -e "${RED}✗${NC}  $*"; }
header(){ echo -e "\n${BOLD}══ $* ══${NC}\n"; }

# ─── Result recording ──────────────────────────────────────────────────────────
record() {
    local status="$1"; shift
    local check="$1"; shift
    local message="$*"
    echo "${status} | ${check} | ${message}" >> "$RESULTS_FILE"
    case "$status" in
        PASS) ((PASS_COUNT++)) || true ;;
        FAIL) ((FAIL_COUNT++)) || true ;;
        WARN) ((WARN_COUNT++)) || true ;;
    esac
}

# ─── Root / sudo handling ──────────────────────────────────────────────────────
need_root() {
    if [ "$(id -u)" -ne 0 ]; then
        exec sudo -E "$0" "$@"
    fi
}

# ─── Parse arguments ───────────────────────────────────────────────────────────
for arg in "$@"; do
    case "$arg" in
        --ci)        CI_MODE=true ;;
        --no-install) SKIP_INSTALL=true ;;
        --container) CONTAINER_MODE=true ;;
        --installer) : ;;
        --installer=*) INSTALLER="${arg#*=}" ;;
        --help|-h)   show_help ;;
        *)           echo "Unknown option: $arg"; show_help ;;
    esac
done

# Support "--installer PATH" (consumes next arg)
ARGS=("$@")
for ((i=0; i<${#ARGS[@]}; i++)); do
    if [ "${ARGS[$i]}" = "--installer" ]; then
        INSTALLER="${ARGS[$((i+1))]:-}"
    fi
done

if [ ! -f "$INSTALLER" ]; then
    echo "Installer not found at: $INSTALLER" >&2
    exit 2
fi
INSTALLER="$(cd "$(dirname "$INSTALLER")" && pwd)/$(basename "$INSTALLER")"

# Everything after this point needs root.
need_root "$@"

# ─── OS detection ──────────────────────────────────────────────────────────────
detect_os() {
    local distro="unknown" version="unknown"
    if [ -f /etc/os-release ]; then
        # shellcheck disable=SC1091
        . /etc/os-release
        distro="${ID:-unknown}"
        version="${VERSION_ID:-unknown}"
    elif [ -f /etc/redhat-release ]; then
        distro="rhel"
        version="unknown"
    fi
    echo "$distro $version"
}

# ─── Install phase ─────────────────────────────────────────────────────────────
run_installer() {
    local email="${RENOS_ADMIN_EMAIL:-admin@renos.local}"
    local password="${RENOS_ADMIN_PASSWORD:-}"
    local external_url="${RENOS_EXTERNAL_URL:-}"

    if [ -z "$password" ]; then
        password="$(openssl rand -base64 24 2>/dev/null | tr -dc 'A-Za-z0-9' | head -c 16)"
    fi
    printf 'email=%s\npassword=%s\n' "$email" "$password" > "$CREDS_FILE"
    chmod 600 "$CREDS_FILE"

    header "Running RenOS installer (non-interactive)"
    info "Installer: $INSTALLER"
    info "Admin email: $email"

    local rc=0
    if [ -n "$external_url" ]; then
        NONINTERACTIVE=1 \
        RENOS_ADMIN_EMAIL="$email" \
        RENOS_ADMIN_PASSWORD="$password" \
        RENOS_EXTERNAL_URL="$external_url" \
        RENOS_TLS_MODE="${RENOS_TLS_MODE:-self-signed}" \
        RENOS_USE_DOCKER_DB=true \
        bash "$INSTALLER" 2>&1 | tee -a "$INSTALL_LOG"
    else
        NONINTERACTIVE=1 \
        RENOS_ADMIN_EMAIL="$email" \
        RENOS_ADMIN_PASSWORD="$password" \
        RENOS_TLS_MODE="${RENOS_TLS_MODE:-self-signed}" \
        RENOS_USE_DOCKER_DB=true \
        bash "$INSTALLER" 2>&1 | tee -a "$INSTALL_LOG"
    fi
    rc="${PIPESTATUS[0]}"

    if [ "$rc" -eq 0 ]; then
        ok "Installer completed successfully (exit 0)"
        record PASS "install" "Installer completed successfully"
    else
        fail "Installer exited with code $rc (see $INSTALL_LOG)"
        record FAIL "install" "Installer exited with code $rc"
    fi
}

# ─── Verification helpers ──────────────────────────────────────────────────────
http_code() { # url [extra curl args...]
    local url="$1"; shift
    curl -sk --connect-timeout 5 --max-time 15 -o /dev/null -w '%{http_code}' "$@" "$url" 2>/dev/null || echo "000"
}

# ─── Verification checks ───────────────────────────────────────────────────────
check_docker() {
    if docker info &>/dev/null 2>&1; then
        ok "Docker daemon is running"
        record PASS "docker" "Docker daemon running"
    else
        fail "Docker daemon is NOT running"
        record FAIL "docker" "Docker daemon not running"
    fi
}

check_containers() {
    local all_ok=true
    for c in renos-core renos-postgres renos-redis renos-nats renos-hub renos-nginx; do
        if docker inspect --format='{{.State.Status}}' "$c" 2>/dev/null | grep -q running; then
            ok "Container running: $c"
        else
            local status
            status="$(docker inspect --format='{{.State.Status}}' "$c" 2>/dev/null || echo "not-found")"
            warn "Container not running: $c ($status)"
            all_ok=false
        fi
    done
    if $all_ok; then
        record PASS "containers" "All RenOS containers running"
    else
        record FAIL "containers" "One or more RenOS containers not running"
    fi
}

check_systemd() {
    local all_ok=true
    for svc in renos-core renos-agent renos-hub renos-nginx; do
        if systemctl is-active --quiet "$svc" 2>/dev/null; then
            ok "Systemd service active: $svc"
        else
            warn "Systemd service not active: $svc"
            all_ok=false
        fi
    done
    if $all_ok; then
        record PASS "systemd" "All RenOS systemd services active"
    else
        record WARN "systemd" "Some systemd services inactive (may use non-systemd init)"
    fi
}

check_api() {
    local endpoints=(
        "https://localhost:8443/api/v1/healthz"
        "http://localhost:8443/api/v1/healthz"
        "https://localhost:443/api/v1/healthz"
        "http://localhost:80/api/v1/healthz"
    )
    local found=""
    local code
    for ep in "${endpoints[@]}"; do
        code="$(http_code "$ep")"
        if [ "$code" = "200" ]; then
            found="$ep"
            break
        fi
    done
    if [ -n "$found" ]; then
        local body
        body="$(curl -sk --max-time 15 "$found" 2>/dev/null)"
        if echo "$body" | grep -q '"ok"\|"status":"ok"'; then
            ok "API healthz responds 200 at $found"
            record PASS "api_healthz" "API healthy at $found"
        else
            warn "API healthz returned 200 but unexpected body at $found"
            record WARN "api_healthz" "Unexpected body: $body"
        fi
    else
        fail "API healthz not reachable on any endpoint (last code: $code)"
        record FAIL "api_healthz" "Not reachable on 8443/443/80"
    fi
}

check_readyz() {
    local readyz
    readyz="$(curl -sk --max-time 15 https://localhost:8443/api/v1/readyz 2>/dev/null || \
              curl -sk --max-time 15 http://localhost:8443/api/v1/readyz 2>/dev/null || echo "")"
    if [ -z "$readyz" ]; then
        readyz="$(curl -sk --max-time 15 https://localhost:443/api/v1/readyz 2>/dev/null || echo "")"
    fi

    check_dep() {
        local name="$1" label="$2"
        if echo "$readyz" | grep -q "\"$name\"[^}]*\"ok\""; then
            ok "$label reachable (via /readyz)"
            record PASS "$name" "$label reachable"
        else
            fail "$label NOT reachable (via /readyz)"
            record FAIL "$name" "$label unreachable"
        fi
    }
    check_dep "database" "PostgreSQL"
    check_dep "nats"     "NATS"
    check_dep "redis"    "Redis"
}

check_login() {
    local email password
    email="$(grep '^email=' "$CREDS_FILE" 2>/dev/null | cut -d= -f2-)"
    password="$(grep '^password=' "$CREDS_FILE" 2>/dev/null | cut -d= -f2-)"
    if [ -z "$email" ] || [ -z "$password" ]; then
        warn "Admin credentials file missing; skipping login check"
        record WARN "login" "No credentials available"
        return
    fi

    local login
    login="$(curl -sk --max-time 15 -X POST "https://localhost:8443/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"${email}\",\"password\":\"${password}\"}" 2>/dev/null || \
        curl -sk --max-time 15 -X POST "http://localhost:8443/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"${email}\",\"password\":\"${password}\"}" 2>/dev/null || echo "")"

    if echo "$login" | grep -q '"token"'; then
        ok "Admin login succeeded (JWT token issued) for $email"
        record PASS "login" "Admin login works"
    else
        fail "Admin login FAILED for $email (response: ${login:0:120})"
        record FAIL "login" "Admin login failed"
    fi
}

check_websocket() {
    local code
    code="$(curl -sk --max-time 15 -o /dev/null -w '%{http_code}' \
        -H "Connection: Upgrade" -H "Upgrade: websocket" \
        -H "Sec-WebSocket-Version: 13" -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" \
        "https://localhost:8443/api/v1/ws" 2>/dev/null || echo "000")"
    if [ "$code" = "000" ]; then
        code="$(curl -sk --max-time 15 -o /dev/null -w '%{http_code}' \
            -H "Connection: Upgrade" -H "Upgrade: websocket" \
            -H "Sec-WebSocket-Version: 13" -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" \
            "http://localhost:8443/api/v1/ws" 2>/dev/null || echo "000")"
    fi
    if [ "$code" != "000" ] && [ "$code" != "404" ]; then
        ok "WebSocket endpoint /api/v1/ws present (HTTP $code; 401=expected without token)"
        record PASS "websocket" "WS endpoint exists (HTTP $code)"
    else
        fail "WebSocket endpoint /api/v1/ws not reachable (HTTP ${code:-none})"
        record FAIL "websocket" "WS endpoint missing"
    fi
}

check_dashboard() {
    local found=""
    local code
    for ep in "https://localhost:443/" "http://localhost:80/" "https://localhost:8443/"; do
        code="$(http_code "$ep")"
        if [ "$code" = "200" ] || [ "$code" = "301" ] || [ "$code" = "302" ]; then
            found="$ep (HTTP $code)"
            break
        fi
    done
    if [ -n "$found" ]; then
        ok "Dashboard responds at $found"
        record PASS "dashboard" "Dashboard reachable at $found"
    else
        fail "Dashboard not reachable (last code: $code)"
        record FAIL "dashboard" "Dashboard unreachable"
    fi
}

check_tls() {
    local cert="/etc/renos/tls/fullchain.pem"
    local key="/etc/renos/tls/privkey.pem"
    if [ -f "$cert" ] && [ -f "$key" ]; then
        if openssl x509 -checkend 86400 -noout -in "$cert" 2>/dev/null; then
            local subj days
            subj="$(openssl x509 -subject -noout -in "$cert" 2>/dev/null | sed 's/^subject=//')"
            days="$(openssl x509 -enddate -noout -in "$cert" 2>/dev/null | cut -d= -f2-)"
            ok "TLS certificate present and valid: $subj (expires $days)"
            record PASS "tls" "Certificate valid"
        else
            fail "TLS certificate present but expired or invalid"
            record FAIL "tls" "Certificate expired/invalid"
        fi
    else
        fail "TLS certificates missing ($cert, $key)"
        record FAIL "tls" "Certificates missing"
    fi
}

# ─── Container-mode helpers ────────────────────────────────────────────────────
# In systemd-less CI containers (e.g. fedora:42) the installer cannot start
# services via systemctl, so we start Docker and the compose stack manually.
ensure_stack_running() {
    if command -v systemctl >/dev/null 2>&1 && systemctl is-system-running &>/dev/null; then
        return 0
    fi
    info "Systemd not running; starting Docker + stack manually (container mode)"

    if command -v dockerd >/dev/null 2>&1 && ! docker info &>/dev/null 2>&1; then
        info "Starting dockerd"
        (nohup dockerd >/var/log/renos/dockerd.log 2>&1 &) || true
        for _ in $(seq 1 30); do
            docker info &>/dev/null 2>&1 && break
            sleep 1
        done
    fi

    if docker info &>/dev/null 2>&1; then
        if [ -f /etc/renos/docker-compose.yml ]; then
            docker compose -f /etc/renos/docker-compose.yml up -d || true
        else
            info "No compose file found; existing containers will be used as-is"
        fi
    else
        warn "Docker daemon unavailable (see /var/log/renos/dockerd.log)"
    fi
}

# ─── Summary ───────────────────────────────────────────────────────────────────
print_summary() {
    echo ""
    echo "──────────────────────────────────────────────────────────"
    echo "  RenOS Fresh Install Test — Results"
    echo "  Distro: $(detect_os)"
    echo "  Time:   $(date -u '+%Y-%m-%dT%H:%M:%SZ')"
    echo "──────────────────────────────────────────────────────────"
    printf "  %-6s %-14s %s\n" "STATUS" "CHECK" "DETAIL"
    printf "  %-6s %-14s %s\n" "──────" "─────────────" "────────────────────────────────────────"
    while IFS='|' read -r status check detail; do
        status="$(echo "$status" | xargs)"
        check="$(echo "$check" | xargs)"
        detail="$(echo "$detail" | xargs)"
        case "$status" in
            PASS) printf "  %s %-14s %s\n" "${GREEN}PASS${NC}" "$check" "$detail" ;;
            FAIL) printf "  %s %-14s %s\n" "${RED}FAIL${NC}" "$check" "$detail" ;;
            WARN) printf "  %s %-14s %s\n" "${YELLOW}WARN${NC}" "$check" "$detail" ;;
        esac
    done < "$RESULTS_FILE"
    echo "──────────────────────────────────────────────────────────"
    echo "  ${GREEN}PASS: $PASS_COUNT${NC}  ${RED}FAIL: $FAIL_COUNT${NC}  ${YELLOW}WARN: $WARN_COUNT${NC}"
    echo ""
    echo "  Full results: $RESULTS_FILE"
    echo "  Install log:  $INSTALL_LOG"
    echo "  Credentials:  $CREDS_FILE"
    echo ""
}

# ─── Main ──────────────────────────────────────────────────────────────────────
header "RenOS Fresh Install Validation"
info "Distro: $(detect_os)"
info "Kernel: $(uname -r)"
info "Date:   $(date -u '+%Y-%m-%dT%H:%M:%SZ')"
echo ""

: > "$RESULTS_FILE"
: > "$INSTALL_LOG"
chmod 600 "$CREDS_FILE" 2>/dev/null || true

if [ "$SKIP_INSTALL" = false ]; then
    run_installer
else
    info "Skipping installer (--no-install); verifying existing installation"
fi

if [ "$CONTAINER_MODE" = true ]; then
    ensure_stack_running
fi

echo ""
header "Verification"

check_docker
check_containers
check_systemd
check_api
check_readyz
check_login
check_websocket
check_dashboard
check_tls

print_summary

if [ "$FAIL_COUNT" -gt 0 ]; then
    echo -e "${RED}RESULT: FAILURE — ${FAIL_COUNT} check(s) failed${NC}" >&2
    exit 1
fi
echo -e "${GREEN}RESULT: SUCCESS — all checks passed (${PASS_COUNT} PASS, ${WARN_COUNT} WARN)${NC}"
exit 0
