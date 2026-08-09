#!/usr/bin/env bash
# =============================================================================
# RenOS One-Command Installer
# =============================================================================
# Zero-knowledge install for RenOS on:
#   Ubuntu 24.04, Ubuntu 22.04, Debian 13, Fedora 42
#
# This wrapper is self-contained: it can fetch the RenOS installer from a
# release URL (RENOS_INSTALLER_URL) or use installer/linux/install.sh when it
# sits alongside it in a checkout.
#
# Usage:
#   curl -sSL https://renos.local/install.sh | sudo bash     # (when distributed)
#   sudo ./one-command-install.sh                            # from a checkout
#   sudo NONINTERACTIVE=1 ./one-command-install.sh           # skip all prompts
#   sudo ./one-command-install.sh --dry-run                  # print the plan
#   sudo ./one-command-install.sh --uninstall                # remove RenOS
#   sudo ./one-command-install.sh --update                   # re-run install
#   sudo ./one-command-install.sh --yes                      # accept prompts
#
# Environment:
#   NONINTERACTIVE          Skip prompts (1|true|yes|on)
#   RENOS_ADMIN_EMAIL       Admin email (default: admin@renos.local)
#   RENOS_ADMIN_PASSWORD    Admin password (auto-generated if unset)
#   RENOS_EXTERNAL_URL      Public URL (default: https://<hostname>)
#   RENOS_TLS_MODE          self-signed | letsencrypt | none
#   RENOS_INSTALLER_URL     URL to download installer/install.sh from
#
# Exit codes:
#   0   Success
#   1   Failure
#   2   Usage error
# =============================================================================

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOCAL_INSTALLER="${SCRIPT_DIR}/install.sh"

INSTALLER=""
STATE_DIR="/var/lib/renos/.install-state"
RESULTS_FILE="/tmp/renos-install-results.txt"
INSTALL_LOG="/tmp/renos-one-command-install.log"
CREDS_FILE="/tmp/renos-admin-credentials.txt"

RENOS_VERSION="0.1.0"

DRY_RUN=false
UNINSTALL=false
UPDATE=false
YES_MODE=false

PASS_COUNT=0
FAIL_COUNT=0
WARN_COUNT=0

# ─── Help ──────────────────────────────────────────────────────────────────────
show_help() {
    cat <<EOF
RenOS One-Command Installer v${RENOS_VERSION}

Usage: $(basename "$0") [OPTIONS]

Options:
  --dry-run        Print the 14-step plan and exit without changes
  --uninstall      Remove RenOS, its data, and state markers
  --update         Re-run the installer over an existing install (keeps secrets)
  --yes            Accept all prompts (equivalent to NONINTERACTIVE=1)
  --help, -h       Show this help message

Environment:
  NONINTERACTIVE        Skip all prompts (1|true|yes|on)
  RENOS_ADMIN_EMAIL     Admin email (default: admin@renos.local)
  RENOS_ADMIN_PASSWORD  Admin password (auto-generated if unset)
  RENOS_EXTERNAL_URL    Public URL (default: https://<hostname>)
  RENOS_TLS_MODE        self-signed | letsencrypt | none
  RENOS_INSTALLER_URL   Where to fetch install.sh from (when not in a checkout)

Exit codes:
  0   Success
  1   Failure
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

info()    { echo -e "${BLUE}ℹ${NC}  $*"; }
step_ok() { echo -e "${GREEN}✓${NC}  $*"; }
step_skip(){ echo -e "${YELLOW}−${NC}  $*"; }
step_fail(){ echo -e "${RED}✗${NC}  $*"; }
warn()    { echo -e "${YELLOW}⚠${NC}  $*"; }
fail()    { echo -e "${RED}✗${NC}  $*"; }
header()  { echo -e "\n${BOLD}══ $* ══${NC}\n"; }

record() {
    local status="$1"; shift
    local check="$1"; shift
    echo "${status} | ${check} | $*" >> "$RESULTS_FILE"
    case "$status" in
        PASS) ((PASS_COUNT++)) || true ;;
        FAIL) ((FAIL_COUNT++)) || true ;;
        WARN) ((WARN_COUNT++)) || true ;;
    esac
}

# ─── Helpers ───────────────────────────────────────────────────────────────────
is_root() { [ "$(id -u)" -eq 0 ]; }

mark() { mkdir -p "$STATE_DIR"; printf '%s' "${2:-$1}" > "$STATE_DIR/$1"; }

get_mark() { cat "$STATE_DIR/$1" 2>/dev/null || echo ""; }

have_install_marker() { [ -f "$STATE_DIR/installed" ]; }

detect_os() {
    local distro="unknown" version="unknown"
    if [ -f /etc/os-release ]; then
        # shellcheck disable=SC1091
        . /etc/os-release
        distro="${ID:-unknown}"
        version="${VERSION_ID:-unknown}"
    fi
    echo "$distro $version"
}

check_disk() {
    local free_mb
    free_mb="$(df -Pm /var 2>/dev/null | awk 'NR==2{print $4}')"
    if [ -z "$free_mb" ] || [ "${free_mb//[^0-9]/}" -lt 5120 ] 2>/dev/null; then
        return 1
    fi
    return 0
}

install_pkg() {
    if command -v apt-get >/dev/null 2>&1; then
        DEBIAN_FRONTEND=noninteractive apt-get install -y -qq "$@" >/dev/null 2>&1
    elif command -v dnf >/dev/null 2>&1; then
        dnf install -y -q "$@" >/dev/null 2>&1
    elif command -v yum >/dev/null 2>&1; then
        yum install -y -q "$@" >/dev/null 2>&1
    else
        return 1
    fi
}

ensure_prereqs() {
    local missing=""
    command -v curl   >/dev/null 2>&1 || missing="$missing curl"
    command -v openssl >/dev/null 2>&1 || missing="$missing openssl"
    command -v tar    >/dev/null 2>&1 || missing="$missing tar"
    if [ -n "$missing" ]; then
        info "Installing missing prerequisites:$missing"
        install_pkg $missing || return 1
    fi
    return 0
}

resolve_installer() {
    if [ -f "$LOCAL_INSTALLER" ]; then
        INSTALLER="$LOCAL_INSTALLER"
        return 0
    fi
    local url="${RENOS_INSTALLER_URL:-}"
    if [ -z "$url" ]; then
        # No release artifacts yet; require a checkout alongside this script.
        return 1
    fi
    local tmp
    tmp="$(mktemp /tmp/renos-install-XXXXXX.sh)"
    info "Downloading installer from $url"
    if curl -fsSL --connect-timeout 15 --max-time 120 -o "$tmp" "$url"; then
        INSTALLER="$tmp"
        return 0
    fi
    return 1
}

admin_credentials() {
    local email="${RENOS_ADMIN_EMAIL:-}"
    local password="${RENOS_ADMIN_PASSWORD:-}"

    if [ -f "$CREDS_FILE" ]; then
        email="${email:-$(grep '^email=' "$CREDS_FILE" | cut -d= -f2-)}"
        password="${password:-$(grep '^password=' "$CREDS_FILE" | cut -d= -f2-)}"
    fi

    if $UPDATE && [ -z "$password" ] && have_install_marker; then
        password="$(get_mark admin-password)"
    fi

    email="${email:-admin@renos.local}"
    if [ -z "$password" ]; then
        password="$(openssl rand -base64 24 2>/dev/null | tr -dc 'A-Za-z0-9' | head -c 16)"
    fi

    printf 'email=%s\npassword=%s\n' "$email" "$password" > "$CREDS_FILE"
    chmod 600 "$CREDS_FILE"
    echo "$email" "$password"
}

http_code() {
    curl -sk --connect-timeout 5 --max-time 15 -o /dev/null -w '%{http_code}' "$@" 2>/dev/null || echo "000"
}

check_health() {
    local code
    for ep in "https://localhost:8443/api/v1/healthz" \
              "http://localhost:8443/api/v1/healthz" \
              "https://localhost:443/api/v1/healthz"; do
        code="$(http_code "$ep")"
        [ "$code" = "200" ] && { echo "$ep"; return 0; }
    done
    echo "$ep"
    return 1
}

# ─── Steps ─────────────────────────────────────────────────────────────────────
step_detect_os() {
    echo -e "${BOLD}  1) Detect OS & version${NC}"
    local distro
    distro="$(detect_os)"
    local id="${distro%% *}" ver="${distro#* }"
    case "$id $ver" in
        "ubuntu 24.04"|"ubuntu 22.04"|"debian 13"|"fedora 42")
            step_ok "Supported: $id $ver"
            record PASS "os_detect" "Supported OS: $id $ver"
            return 0
            ;;
        *)
            if $YES_MODE; then
                step_ok "Continuing on unlisted distro: $id $ver (forced by --yes)"
                record WARN "os_detect" "Unlisted distro: $id $ver"
                return 0
            fi
            step_fail "Unsupported distro: $id $ver (expected Ubuntu 24.04/22.04, Debian 13, Fedora 42)"
            record FAIL "os_detect" "Unsupported distro: $id $ver"
            return 1
            ;;
    esac
}

step_root() {
    echo -e "${BOLD}  2) Verify root access${NC}"
    if is_root; then
        step_ok "Running as root"
        record PASS "root" "Running as root"
        return 0
    fi
    info "Elevating via sudo"
    if command -v sudo >/dev/null 2>&1 && sudo -n true 2>/dev/null; then
        exec sudo -E "$0" "$@"
    fi
    step_fail "Need root privileges (re-run with sudo)"
    record FAIL "root" "Not root and cannot sudo"
    return 1
}

step_existing() {
    echo -e "${BOLD}  3) Detect existing install${NC}"
    if have_install_marker; then
        local ver inst_at
        ver="$(get_mark version)"
        inst_at="$(get_mark installed-at)"
        step_ok "Existing RenOS install found (version ${ver:-?}, installed ${inst_at:-?})"
        if $UPDATE; then
            step_ok "Update mode: will re-run installer, preserving secrets"
        elif ! $YES_MODE; then
            info "Re-run is safe (idempotent) — secrets are preserved automatically."
        fi
        record PASS "existing" "Found existing install v${ver:-?}"
        return 0
    fi
    step_ok "No existing install detected (fresh install)"
    record PASS "existing" "Fresh install"
    return 0
}

step_prereqs() {
    echo -e "${BOLD}  4) Install prerequisites (curl, openssl, tar)${NC}"
    if ensure_prereqs; then
        step_ok "Prerequisites available"
        record PASS "prereqs" "curl/openssl/tar present"
        return 0
    fi
    step_fail "Failed to install prerequisites"
    record FAIL "prereqs" "Could not install curl/openssl/tar"
    return 1
}

step_dns() {
    echo -e "${BOLD}  5) Resolve hostname & external URL${NC}"
    local hostname
    hostname="$(hostname -f 2>/dev/null || echo 'localhost')"
    local ext="${RENOS_EXTERNAL_URL:-}"
    ext="${ext:-https://${hostname}}"
    if [[ "$ext" =~ ^https?:// ]]; then
        step_ok "External URL: $ext"
        record PASS "dns" "External URL: $ext"
        return 0
    fi
    step_fail "Invalid RENOS_EXTERNAL_URL: $ext"
    record FAIL "dns" "Invalid external URL: $ext"
    return 1
}

step_disk() {
    echo -e "${BOLD}  6) Check available disk space${NC}"
    if check_disk; then
        local free_mb
        free_mb="$(df -Pm /var 2>/dev/null | awk 'NR==2{print $4}')"
        step_ok "Sufficient disk space ($((free_mb / 1024)) GB free on /var)"
        record PASS "disk" "Disk space OK"
        return 0
    fi
    step_fail "Less than 5 GB free on /var"
    record FAIL "disk" "Insufficient disk space"
    return 1
}

step_dirs() {
    echo -e "${BOLD}  7) Prepare /etc/renos directories${NC}"
    mkdir -p /etc/renos /etc/renos/tls /etc/renos/env /etc/renos/compose /etc/renos/nginx \
        /var/lib/renos /var/log/renos /opt/renos
    if [ -d /etc/renos ]; then
        step_ok "Directories ready under /etc/renos"
        record PASS "dirs" "/etc/renos prepared"
        return 0
    fi
    step_fail "Could not create /etc/renos"
    record FAIL "dirs" "Cannot create /etc/renos"
    return 1
}

step_credentials() {
    echo -e "${BOLD}  8) Configure admin credentials${NC}"
    local email password
    read -r email password < <(admin_credentials)
    local marker_email
    marker_email="$(get_mark admin-email)"
    if [ -n "$marker_email" ] && [ "$marker_email" != "$email" ]; then
        warn "Admin email changed from $marker_email to $email"
    fi
    step_ok "Admin email: $email"
    info "Password stored in $CREDS_FILE (mode 600)"
    record PASS "credentials" "Admin configured: $email"
    return 0
}

step_installer() {
    echo -e "${BOLD}  9) Resolve RenOS installer${NC}"
    if resolve_installer; then
        step_ok "Using installer: $INSTALLER"
        record PASS "installer" "Installer: $INSTALLER"
        return 0
    fi
    step_fail "No installer found. Place this script next to installer/linux/install.sh or set RENOS_INSTALLER_URL."
    record FAIL "installer" "No installer available"
    return 1
}

step_tls() {
    echo -e "${BOLD} 10) Configure TLS${NC}"
    local tls_mode="${RENOS_TLS_MODE:-self-signed}"
    case "$tls_mode" in
        self-signed)  step_ok "TLS: self-signed (auto-generated)" ;;
        letsencrypt)  step_ok "TLS: Let's Encrypt (requires public DNS + port 80)" ;;
        none)         warn "TLS: none — traffic will be plain HTTP" ;;
        *)            step_fail "Unknown RENOS_TLS_MODE: $tls_mode"; record FAIL "tls" "Bad TLS mode"; return 1 ;;
    esac
    record PASS "tls" "TLS mode: $tls_mode"
    return 0
}

step_run_installer() {
    echo -e "${BOLD} 11) Run RenOS installer (non-interactive)${NC}"
    local email password
    read -r email password < <(admin_credentials)
    local tls_mode="${RENOS_TLS_MODE:-self-signed}"
    local rc=0
    NONINTERACTIVE=1 \
    RENOS_ADMIN_EMAIL="$email" \
    RENOS_ADMIN_PASSWORD="$password" \
    RENOS_EXTERNAL_URL="${RENOS_EXTERNAL_URL:-}" \
    RENOS_TLS_MODE="$tls_mode" \
    RENOS_USE_DOCKER_DB=true \
    bash "$INSTALLER" 2>&1 | tee -a "$INSTALL_LOG"
    rc="${PIPESTATUS[0]}"
    if [ "$rc" -eq 0 ]; then
        step_ok "Installer completed (exit 0)"
        record PASS "installer_run" "Installer exit 0"
        return 0
    fi
    step_fail "Installer exited $rc (see $INSTALL_LOG)"
    record FAIL "installer_run" "Installer exit $rc"
    return 1
}

step_health() {
    echo -e "${BOLD} 12) Verify services (API health)${NC}"
    local ep
    if ep="$(check_health)"; then
        step_ok "API healthy at $ep"
        record PASS "health" "API healthy at $ep"
        return 0
    fi
    step_fail "API not healthy (last try: $ep)"
    record FAIL "health" "API unhealthy"
    return 1
}

step_validate() {
    echo -e "${BOLD} 13) Run full validation${NC}"
    local validator="${SCRIPT_DIR}/fresh-install-test.sh"
    if [ -f "$validator" ]; then
        RENOS_RESULTS_FILE="/tmp/renos-validation-results.txt" \
        RENOS_INSTALL_LOG="/tmp/renos-validation.log" \
        bash "$validator" --no-install 2>&1 | tee -a "$INSTALL_LOG"
        local rc="${PIPESTATUS[0]}"
        if [ "$rc" -eq 0 ]; then
            step_ok "Validation passed"
            record PASS "validate" "fresh-install-test passed"
            return 0
        fi
        step_fail "Validation failed (exit $rc)"
        record FAIL "validate" "fresh-install-test exit $rc"
        return 1
    fi
    info "fresh-install-test.sh not available; running quick check instead"
    if check_health >/dev/null 2>&1; then
        step_ok "Quick validation passed"
        record PASS "validate" "Quick health check passed"
        return 0
    fi
    step_fail "Quick validation failed"
    record FAIL "validate" "Quick health check failed"
    return 1
}

step_summary() {
    echo -e "${BOLD} 14) Summary${NC}"
    local email
    email="$(grep '^email=' "$CREDS_FILE" 2>/dev/null | cut -d= -f2- || echo 'admin@renos.local')"
    local distro
    distro="$(detect_os)"
    mark installed 1
    mark version "$RENOS_VERSION"
    mark admin-email "$email"
    mark installed-at "$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
    step_ok "RenOS is ready."
    info "Dashboard : ${RENOS_EXTERNAL_URL:-https://$(hostname -f 2>/dev/null || echo localhost)}"
    info "Admin     : $email (password in $CREDS_FILE)"
    info "Distro    : $distro"
    record PASS "summary" "RenOS is ready."
    return 0
}

# ─── Uninstall ─────────────────────────────────────────────────────────────────
do_uninstall() {
    header "RenOS Uninstall"
    local rc=0
    if [ -n "$INSTALLER" ] && [ -f "$INSTALLER" ]; then
        info "Running installer uninstall mode"
        NONINTERACTIVE=1 RENOS_UNINSTALL_CONFIRM=yes bash "$INSTALLER" --uninstall 2>&1 | tee -a "$INSTALL_LOG"
        rc="${PIPESTATUS[0]}"
    else
        if command -v docker >/dev/null 2>&1 && docker ps -q 2>/dev/null | grep -q .; then
            info "Stopping RenOS containers"
            docker rm -f renos-core renos-hub renos-agent renos-nginx renos-postgres renos-redis renos-nats 2>/dev/null
        fi
        systemctl stop renos-core renos-agent renos-hub renos-nginx 2>/dev/null
    fi
    rm -rf "$STATE_DIR"
    rm -f "$CREDS_FILE"
    if [ "$rc" -eq 0 ]; then
        step_ok "RenOS removed"
        record PASS "uninstall" "Removed"
        exit 0
    fi
    step_fail "Uninstall encountered issues (see $INSTALL_LOG)"
    record FAIL "uninstall" "Exit $rc"
    exit 1
}

# ─── Dry run ───────────────────────────────────────────────────────────────────
do_dry_run() {
    header "RenOS Installer — Dry Run (no changes)"
    echo -e "  Target distro : $(detect_os)"
    echo -e "  Admin email   : ${RENOS_ADMIN_EMAIL:-admin@renos.local}"
    echo -e "  External URL  : ${RENOS_EXTERNAL_URL:-https://$(hostname -f 2>/dev/null || echo localhost)}"
    echo -e "  TLS mode      : ${RENOS_TLS_MODE:-self-signed}"
    echo ""
    echo -e "  14-step plan:"
    for s in \
        "Detect OS & version" \
        "Verify root access" \
        "Detect existing install" \
        "Install prerequisites (curl, openssl, tar)" \
        "Resolve hostname & external URL" \
        "Check available disk space" \
        "Prepare /etc/renos directories" \
        "Configure admin credentials" \
        "Resolve RenOS installer" \
        "Configure TLS" \
        "Run installer (non-interactive)" \
        "Verify services (API health)" \
        "Run full validation" \
        "Summary"; do
        echo -e "  ${BOLD}$s${NC}"
    done
    echo ""
    if [ -f "$LOCAL_INSTALLER" ]; then
        echo -e "  Installer : $LOCAL_INSTALLER (present — will use this)"
    else
        echo -e "  Installer : from RENOS_INSTALLER_URL (none set — will fail without a checkout)"
    fi
    echo ""
    echo -e "  ${YELLOW}Dry run complete. No changes made.${NC}"
    exit 0
}

# ─── Main ──────────────────────────────────────────────────────────────────────
main() {
    local original_args=("$@")

    for arg in "$@"; do
        case "$arg" in
            --dry-run)   DRY_RUN=true ;;
            --uninstall) UNINSTALL=true ;;
            --update)    UPDATE=true ;;
            --yes)       YES_MODE=true ;;
            --help|-h)   show_help ;;
            *)           echo "Unknown option: $arg" >&2; show_help ;;
        esac
    done

    if [ "$DRY_RUN" = true ]; then
        do_dry_run
    fi

    echo -e "${BOLD}${CYAN:-}"
    echo -e "╔══════════════════════════════════════════╗"
    echo -e "║     RenOS One-Command Installer v${RENOS_VERSION}    ║"
    echo -e "╚══════════════════════════════════════════╝"
    echo -e "${NC}"
    echo ""

    : > "$RESULTS_FILE"
    : > "$INSTALL_LOG"

    # Elevate early so the whole flow runs as root (except --help/--dry-run).
    if ! is_root; then
        if command -v sudo >/dev/null 2>&1 && sudo -n true 2>/dev/null; then
            exec sudo -E "$0" "${original_args[@]}"
        fi
        echo "This script must run as root (re-run with sudo)." >&2
        exit 2
    fi

    if [ "$UNINSTALL" = true ]; then
        if [ -f "$LOCAL_INSTALLER" ]; then INSTALLER="$LOCAL_INSTALLER"; fi
        do_uninstall
        exit 0
    fi

    if resolve_installer; then
        :
    else
        fail "No installer available. Place one-command-install.sh next to installer/linux/install.sh or set RENOS_INSTALLER_URL."
        exit 1
    fi

    local all_ok=true
    step_detect_os    || all_ok=false
    step_root         || all_ok=false
    step_existing     || all_ok=false
    step_prereqs      || all_ok=false
    step_dns          || all_ok=false
    step_disk         || all_ok=false
    step_dirs         || all_ok=false
    step_credentials  || all_ok=false
    step_installer    || all_ok=false
    step_tls          || all_ok=false
    step_run_installer|| all_ok=false
    step_health       || all_ok=false
    step_validate     || all_ok=false
    step_summary      || all_ok=false

    echo ""
    echo -e "${GREEN}PASS: $PASS_COUNT${NC}  ${RED}FAIL: $FAIL_COUNT${NC}  ${YELLOW}WARN: $WARN_COUNT${NC}"
    echo -e "Results: $RESULTS_FILE"

    if $all_ok; then
        echo -e "\n${GREEN}RenOS is ready.${NC}"
        exit 0
    fi
    echo -e "\n${RED}One or more steps failed. See $INSTALL_LOG${NC}" >&2
    exit 1
}

main "$@"
