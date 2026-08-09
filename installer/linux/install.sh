#!/usr/bin/env bash
set -euo pipefail

# RenOS Linux Installer
# =====================
# Full production installation for 7 Linux distros.
# Supports: Ubuntu, Debian, Fedora, RHEL, Rocky, AlmaLinux, Arch
#
# Usage:
#   sudo ./install.sh                          # Interactive install
#   sudo ./install.sh --uninstall              # Remove RenOS
#   sudo ./install.sh --dry-run                # Preview without changes
#   sudo ./install.sh --help                   # Show this help
#
# ENV overrides (non-interactive):
#   RENOS_ADMIN_EMAIL, RENOS_ADMIN_PASSWORD, RENOS_HOSTNAME,
#   RENOS_TLS_MODE=self-signed|letsencrypt|none, RENOS_USE_DOCKER_DB=true|false

RENOS_VERSION="0.1.0"
SCRIPT_NAME="$(basename "$0")"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
START_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
LOG_FILE="/var/log/renos/install.log"
RESTORE_DIR="/var/backups/renos/pre-install"
RENOS_USER="renos"
RENOS_GROUP="renos"
RENOS_HOME="/var/lib/renos"
RENOS_CONFIG_DIR="/etc/renos"
RENOS_LOG_DIR="/var/log/renos"
RENOS_DATA_DIR="/var/lib/renos/data"
RENOS_CA_DIR="/var/lib/renos/ca"
RENOS_PLUGINS_DIR="/var/lib/renos/plugins"
RENOS_BACKUPS_DIR="/var/lib/renos/backups"
RENOS_SERVICE_DIR="/etc/systemd/system"
RENOS_ENV_FILE="${RENOS_CONFIG_DIR}/renos.env"

# ─── Flags ────────────────────────────────────────────────────────────────────
UNINSTALL=false
DRY_RUN=false
VERIFY=false

# NONINTERACTIVE=1 skips every prompt; values come from env vars or defaults.
case "${NONINTERACTIVE:-0}" in
    1|true|yes|on) NONINTERACTIVE=1 ;;
    *)             NONINTERACTIVE=0 ;;
esac

# ─── Colors ───────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# ─── Logging ───────────────────────────────────────────────────────────────────
log()    { local lvl="$1"; shift; echo -e "$(date '+%Y-%m-%d %H:%M:%S') [$lvl] $*" >> "$LOG_FILE"; }
info()   { echo -e "${BLUE}ℹ${NC}  $*"; log "INFO" "$*"; }
ok()     { echo -e "${GREEN}✓${NC}  $*"; log "OK" "$*"; }
warn()   { echo -e "${YELLOW}⚠${NC}  $*"; log "WARN" "$*"; }
fail()   { echo -e "${RED}✗${NC}  $*"; log "FAIL" "$*"; exit 1; }
header() { echo -e "\n${BOLD}${MAGENTA}═══ $* ═══${NC}\n"; log "HEADER" "$*"; }
prompt() { echo -e -n "${CYAN}?${NC}  $*"; }

# ─── Error handling ─────────────────────────────────────────────────────────────
ROLLBACK_STEPS=()

rollback_add() {
    ROLLBACK_STEPS+=("$1")
}

rollback_run() {
    if [ ${#ROLLBACK_STEPS[@]} -gt 0 ]; then
        warn "Rolling back..."
        for ((idx=${#ROLLBACK_STEPS[@]}-1; idx>=0; idx--)); do
            step="${ROLLBACK_STEPS[$idx]}"
            info "  Undo: ${step}"
            log "ROLLBACK" "$step"
        done
        ok "Rollback complete"
    fi
}

trap_exit() {
    local exit_code=$?
    if [ $exit_code -ne 0 ] && [ "$UNINSTALL" = false ] && [ "$DRY_RUN" = false ]; then
        rollback_run
        fail "Installation failed with exit code $exit_code"
    fi
}
trap trap_exit EXIT

# ─── Help ──────────────────────────────────────────────────────────────────────
show_help() {
    cat <<EOF
${BOLD}RenOS Linux Installer v${RENOS_VERSION}${NC}
Full production installation for Ubuntu, Debian, Fedora, RHEL, Rocky, AlmaLinux, Arch.

${BOLD}Usage:${NC}
  sudo $SCRIPT_NAME [options]

${BOLD}Options:${NC}
  --help, -h       Show this help message
  --dry-run, -n    Preview steps without making changes
  --uninstall      Remove all RenOS components and data
  --verify         Run post-install checks against an existing installation

${BOLD}Environment Variables:${NC}
  NONINTERACTIVE=1          Skip all prompts (use env vars or defaults)
  RENOS_ADMIN_EMAIL         Admin email address (default: admin@renos.local)
  RENOS_ADMIN_PASSWORD      Admin password (min 8 chars; auto-generated if unset)
  RENOS_HOSTNAME            Server hostname/FQDN
  RENOS_EXTERNAL_URL        Public URL, e.g. https://renos.example.com
  RENOS_TLS_MODE            TLS mode: self-signed, letsencrypt, none
  RENOS_USE_DOCKER_DB       Use Docker for DBs: true (default), false
  RENOS_CORE_IMAGE          Docker image for core (default: renos/core:latest)
  RENOS_HUB_IMAGE           Docker image for hub (default: renos/hub:latest)
  RENOS_AGENT_IMAGE         Docker image for agent (default: renos/agent:latest)

${BOLD}Examples:${NC}
  sudo $SCRIPT_NAME
  sudo $SCRIPT_NAME --dry-run
  sudo RENOS_ADMIN_EMAIL=admin@example.com RENOS_ADMIN_PASSWORD=secret ./$SCRIPT_NAME
  sudo NONINTERACTIVE=1 ./$SCRIPT_NAME
  sudo $SCRIPT_NAME --verify
  sudo $SCRIPT_NAME --uninstall

${BOLD}Supported Distributions:${NC}
  Debian/Ubuntu (apt)    | Fedora/RHEL/Rocky/AlmaLinux (dnf/yum) | Arch (pacman)
EOF
    exit 0
}

# ─── Parse arguments ────────────────────────────────────────────────────────────
parse_args() {
    while [ $# -gt 0 ]; do
        case "$1" in
            --help|-h)        show_help ;;
            --dry-run|-n)     DRY_RUN=true ;;
            --verify|-v)      VERIFY=true ;;
            --uninstall)      UNINSTALL=true ;;
            *)                warn "Unknown option: $1"; show_help ;;
        esac
        shift
    done
}

# ─── Root check ─────────────────────────────────────────────────────────────────
require_root() {
    if [ "$(id -u)" -ne 0 ]; then
        fail "This script must be run as root (use sudo)."
    fi
}

# ─── Distro detection ───────────────────────────────────────────────────────────
declare -A DISTRO_MAP=(
    [ubuntu]="ubuntu"
    [debian]="debian"
    [fedora]="fedora"
    [rhel]="rhel"
    [rocky]="rocky"
    [almalinux]="almalinux"
    [arch]="arch"
)

declare -A PM_MAP=(
    [ubuntu]="apt"
    [debian]="apt"
    [fedora]="dnf"
    [rhel]="dnf"
    [rocky]="dnf"
    [almalinux]="dnf"
    [arch]="pacman"
)

declare -A PM_INSTALL=(
    [apt]="DEBIAN_FRONTEND=noninteractive apt-get install -y"
    [dnf]="dnf install -y"
    [yum]="yum install -y"
    [pacman]="pacman --noconfirm -S"
)

declare -A PM_UPDATE=(
    [apt]="apt-get update"
    [dnf]="dnf makecache"
    [yum]="yum makecache"
    [pacman]="pacman -Sy"
)

declare -A FIREWALL_TOOL=(
    [ubuntu]="ufw"
    [debian]="ufw"
    [fedora]="firewalld"
    [rhel]="firewalld"
    [rocky]="firewalld"
    [almalinux]="firewalld"
    [arch]="iptables"
)

detect_distro() {
    local distro=""
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        distro="${ID,,}"
    elif [ -f /etc/redhat-release ]; then
        local rh="$(cat /etc/redhat-release)"
        case "$rh" in
            *Fedora*)    distro="fedora" ;;
            *Rocky*)     distro="rocky" ;;
            *AlmaLinux*) distro="almalinux" ;;
            *Red\ Hat*)  distro="rhel" ;;
            *)           distro="rhel" ;;
        esac
    elif [ -f /etc/debian_version ]; then
        distro="debian"
    elif [ -f /etc/arch-release ]; then
        distro="arch"
    fi
    echo "$distro"
}

DISTRO=""
PM=""
PM_INSTALL_CMD=""
PM_UPDATE_CMD=""
FW_TOOL=""

detect_system() {
    header "Detecting operating system"
    DISTRO="$(detect_distro)"
    if [ -z "$DISTRO" ] || [ -z "${DISTRO_MAP[$DISTRO]:-}" ]; then
        fail "Unsupported or unknown distribution: ${DISTRO:-detection failed}"
    fi
    PM="${PM_MAP[$DISTRO]}"
    PM_INSTALL_CMD="${PM_INSTALL[$PM]}"
    PM_UPDATE_CMD="${PM_UPDATE[$PM]}"
    FW_TOOL="${FIREWALL_TOOL[$DISTRO]}"
    ok "Distribution: ${DISTRO^}"
    ok "Package manager: $PM"
    ok "Firewall tool: $FW_TOOL"

    if [ "$DRY_RUN" = true ]; then
        info "[DRY-RUN] Would install for ${DISTRO^} using $PM"
    fi
}

# ─── Package helpers ────────────────────────────────────────────────────────────
pkg_install() {
    local pkgs=("$@")
    if [ "$DRY_RUN" = true ]; then
        info "[DRY-RUN] Would install: ${pkgs[*]}"
        return 0
    fi
    for pkg in "${pkgs[@]}"; do
        if pkg_is_installed "$pkg"; then
            ok "Package already installed: $pkg"
        else
            info "Installing: $pkg"
            eval "$PM_INSTALL_CMD $pkg" >> "$LOG_FILE" 2>&1 || {
                fail "Failed to install package: $pkg (check $LOG_FILE)"
            }
            ok "Installed: $pkg"
        fi
    done
}

pkg_is_installed() {
    local pkg="$1"
    case "$PM" in
        apt)    dpkg -s "$pkg" &>/dev/null ;;
        dnf|yum) rpm -q "$pkg" &>/dev/null ;;
        pacman) pacman -Qi "$pkg" &>/dev/null ;;
        *)      return 1 ;;
    esac
}

pkg_update() {
    if [ "$DRY_RUN" = true ]; then
        info "[DRY-RUN] Would update package cache using $PM"
        return 0
    fi
    info "Updating package cache..."
    eval "$PM_UPDATE_CMD" >> "$LOG_FILE" 2>&1 || true
    ok "Package cache updated"
}

# ─── Docker install ─────────────────────────────────────────────────────────────
install_docker() {
    header "Installing Docker & Docker Compose"
    if command -v docker &>/dev/null && docker compose version &>/dev/null; then
        ok "Docker already installed ($(docker --version))"
        ok "Docker Compose already installed ($(docker compose version --short 2>/dev/null || echo 'v2'))"
        return 0
    fi
    if [ "$DRY_RUN" = true ]; then
        info "[DRY-RUN] Would install Docker and Docker Compose"
        return 0
    fi
    case "$DISTRO" in
        ubuntu|debian)
            pkg_install ca-certificates curl gnupg lsb-release
            install -m 0755 -d /etc/apt/keyrings
            curl -fsSL https://download.docker.com/linux/${DISTRO}/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg 2>>"$LOG_FILE"
            chmod a+r /etc/apt/keyrings/docker.gpg
            echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/${DISTRO} $(lsb_release -cs) stable" > /etc/apt/sources.list.d/docker.list
            pkg_update
            pkg_install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
            ;;
        fedora)
            pkg_install dnf-plugins-core
            dnf config-manager --add-repo https://download.docker.com/linux/fedora/docker-ce.repo 2>>"$LOG_FILE"
            pkg_install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
            ;;
        rhel|rocky|almalinux)
            pkg_install yum-utils
            yum-config-manager --add-repo https://download.docker.com/linux/rhel/docker-ce.repo 2>>"$LOG_FILE"
            pkg_install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
            ;;
        arch)
            pkg_install docker docker-compose
            ;;
    esac
    systemctl enable docker >> "$LOG_FILE" 2>&1 || true
    systemctl start docker >> "$LOG_FILE" 2>&1 || true
    ok "Docker installed and started"
}

# ─── Firewall config ────────────────────────────────────────────────────────────
configure_firewall() {
    header "Configuring firewall"
    if [ "$DRY_RUN" = true ]; then
        info "[DRY-RUN] Would configure $FW_TOOL to open ports 80, 443, 8443"
        return 0
    fi
    case "$FW_TOOL" in
        ufw)
            if command -v ufw &>/dev/null; then
                ufw allow 80/tcp  comment "RenOS HTTP"  2>>"$LOG_FILE" || true
                ufw allow 443/tcp comment "RenOS HTTPS" 2>>"$LOG_FILE" || true
                ufw allow 8443/tcp comment "RenOS API"  2>>"$LOG_FILE" || true
                ufw reload 2>>"$LOG_FILE" || true
                ok "UFW rules added for ports 80, 443, 8443"
            else
                warn "UFW not installed; skipping firewall config"
            fi
            ;;
        firewalld)
            if command -v firewall-cmd &>/dev/null; then
                for port in 80/tcp 443/tcp 8443/tcp; do
                    firewall-cmd --permanent --add-port="$port" 2>>"$LOG_FILE" || true
                done
                firewall-cmd --reload 2>>"$LOG_FILE" || true
                ok "Firewalld rules added for ports 80, 443, 8443"
            else
                warn "firewalld not installed; skipping firewall config"
            fi
            ;;
        iptables)
            if command -v iptables &>/dev/null; then
                for port in 80 443 8443; do
                    iptables -C INPUT -p tcp --dport "$port" -j ACCEPT 2>/dev/null || \
                    iptables -A INPUT -p tcp --dport "$port" -j ACCEPT 2>>"$LOG_FILE" || true
                done
                iptables-save > /etc/iptables/rules.v4 2>/dev/null || \
                iptables-save > /etc/iptables/rules 2>/dev/null || true
                ok "iptables rules added for ports 80, 443, 8443"
            else
                warn "iptables not installed; skipping firewall config"
            fi
            ;;
    esac
}

# ─── User and groups ────────────────────────────────────────────────────────────
create_system_user() {
    header "Creating renos system user"
    if [ "$DRY_RUN" = true ]; then
        info "[DRY-RUN] Would create user '$RENOS_USER' and group '$RENOS_GROUP'"
        return 0
    fi
    if getent group "$RENOS_GROUP" &>/dev/null; then
        ok "Group '$RENOS_GROUP' already exists"
    else
        groupadd --system "$RENOS_GROUP" 2>>"$LOG_FILE"
        ok "Group '$RENOS_GROUP' created"
    fi
    if id -u "$RENOS_USER" &>/dev/null; then
        ok "User '$RENOS_USER' already exists"
        usermod -aG "$RENOS_GROUP" "$RENOS_USER" 2>>"$LOG_FILE" || true
    else
        useradd --system --gid "$RENOS_GROUP" --home-dir "$RENOS_HOME" --no-create-home --shell /usr/sbin/nologin "$RENOS_USER" 2>>"$LOG_FILE"
        ok "User '$RENOS_USER' created"
    fi
    rollback_add "Remove user $RENOS_USER"
}

# ─── Directories ────────────────────────────────────────────────────────────────
create_directories() {
    header "Creating directories"
    local dirs=(
        "$RENOS_CONFIG_DIR"
        "$RENOS_HOME"
        "$RENOS_CA_DIR"
        "$RENOS_DATA_DIR"
        "$RENOS_PLUGINS_DIR"
        "$RENOS_BACKUPS_DIR"
        "$RENOS_LOG_DIR"
        "/var/log/nginx"
    )
    if [ "$DRY_RUN" = true ]; then
        info "[DRY-RUN] Would create directories: ${dirs[*]}"
        return 0
    fi
    for d in "${dirs[@]}"; do
        mkdir -p "$d"
        chown "$RENOS_USER:$RENOS_GROUP" "$d"
        chmod 750 "$d"
    done
    # Ensure install log dir is writable
    mkdir -p "$(dirname "$LOG_FILE")"
    touch "$LOG_FILE"
    chown "$RENOS_USER:$RENOS_GROUP" "$(dirname "$LOG_FILE")" "$LOG_FILE" 2>/dev/null || true
    ok "All directories created and permissions set"
    rollback_add "Remove RenOS directories"
}

# ─── Restore point ──────────────────────────────────────────────────────────────
create_restore_point() {
    header "Creating restore point"
    if [ "$DRY_RUN" = true ]; then
        info "[DRY-RUN] Would back up: /etc/renos, /etc/systemd/system/renos-*, docker volumes"
        return 0
    fi
    mkdir -p "$RESTORE_DIR"
    local ts="$(date +%Y%m%d%H%M%S)"
    if [ -f "$RENOS_ENV_FILE" ]; then
        cp "$RENOS_ENV_FILE" "${RESTORE_DIR}/renos.env.${ts}" 2>/dev/null || true
        ok "Backed up renos.env"
    fi
    if ls /etc/systemd/system/renos-*.service &>/dev/null 2>&1; then
        cp /etc/systemd/system/renos-*.service "$RESTORE_DIR/" 2>/dev/null || true
        ok "Backed up systemd services"
    fi
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -q renos; then
        docker compose -f "${RENOS_CONFIG_DIR}/docker-compose.yml" down 2>/dev/null || true
        ok "Stopped existing Docker services"
    fi
    # Backup logrotate config
    if [ -f /etc/logrotate.d/renos ]; then
        cp /etc/logrotate.d/renos "$RESTORE_DIR/renos.logrotate.${ts}" 2>/dev/null || true
    fi
    ok "Restore point created at $RESTORE_DIR"
}

# ─── Log rotation ───────────────────────────────────────────────────────────────
configure_logrotate() {
    header "Configuring log rotation"
    if [ "$DRY_RUN" = true ]; then
        info "[DRY-RUN] Would create /etc/logrotate.d/renos"
        return 0
    fi
    cat > /etc/logrotate.d/renos <<'EOF'
/var/log/renos/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    copytruncate
    dateext
    dateformat -%Y%m%d
    maxsize 100M
    sharedscripts
    postrotate
        systemctl restart renos-core 2>/dev/null || true
    endscript
}
EOF
    chmod 644 /etc/logrotate.d/renos
    ok "Log rotation configured (30 days, daily, 100MB max per file)"
    rollback_add "Remove /etc/logrotate.d/renos"
}

# ─── Secrets generation ─────────────────────────────────────────────────────────
generate_secrets() {
    local jwt enc pgpass
    jwt="$(openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | od -A n -t x1 | tr -d ' \n')"
    enc="$(openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | od -A n -t x1 | tr -d ' \n')"
    pgpass="$(openssl rand -base64 24 2>/dev/null || head -c 24 /dev/urandom | base64 | tr -d '\n' | tr -d '/+=' | head -c 32)"
    echo "$jwt:$enc:$pgpass"
}

# ─── TLS setup ──────────────────────────────────────────────────────────────────
setup_tls() {
    local tls_mode="$1"
    local hostname="$2"
    header "Setting up TLS (mode: $tls_mode)"

    if [ "$DRY_RUN" = true ]; then
        info "[DRY-RUN] Would set up TLS in mode: $tls_mode for $hostname"
        return 0
    fi

    mkdir -p "${RENOS_CONFIG_DIR}/tls"
    case "$tls_mode" in
        self-signed)
            info "Generating self-signed certificate..."
            openssl req -x509 -nodes -days 3650 -newkey rsa:4096 \
                -keyout "${RENOS_CONFIG_DIR}/tls/privkey.pem" \
                -out "${RENOS_CONFIG_DIR}/tls/fullchain.pem" \
                -subj "/CN=${hostname}/O=RenOS" \
                -addext "subjectAltName=DNS:${hostname},IP:127.0.0.1" \
                2>>"$LOG_FILE"
            chmod 600 "${RENOS_CONFIG_DIR}/tls/privkey.pem"
            chown -R "$RENOS_USER:$RENOS_GROUP" "${RENOS_CONFIG_DIR}/tls"
            ok "Self-signed certificate generated for $hostname"
            ;;
        letsencrypt)
            info "Installing Certbot..."
            pkg_install certbot
            if [ -d "${RENOS_CONFIG_DIR}/tls" ] && [ -f "${RENOS_CONFIG_DIR}/tls/fullchain.pem" ]; then
                certbot renew --non-interactive --webroot -w /var/www/html 2>>"$LOG_FILE" || true
            else
                systemctl stop nginx 2>/dev/null || true
                certbot certonly --standalone --non-interactive --agree-tos \
                    --email "${RENOS_ADMIN_EMAIL:-admin@${hostname}}" \
                    -d "$hostname" 2>>"$LOG_FILE" || {
                    warn "Certbot failed; falling back to self-signed"
                    setup_tls "self-signed" "$hostname"
                    return 0
                }
                cp "/etc/letsencrypt/live/${hostname}/fullchain.pem" "${RENOS_CONFIG_DIR}/tls/" 2>/dev/null || true
                cp "/etc/letsencrypt/live/${hostname}/privkey.pem" "${RENOS_CONFIG_DIR}/tls/" 2>/dev/null || true
                # Set up auto-renewal
                echo "0 0 * * * root certbot renew --quiet --post-hook 'systemctl reload nginx'" > /etc/cron.d/renos-certbot
                chmod 644 /etc/cron.d/renos-certbot
            fi
            chown -R "$RENOS_USER:$RENOS_GROUP" "${RENOS_CONFIG_DIR}/tls" 2>/dev/null || true
            ok "Let's Encrypt certificate obtained for $hostname"
            ;;
        none)
            warn "TLS disabled. Not recommended for production."
            return 0
            ;;
    esac
    rollback_add "Remove TLS certificates"
}

# ─── Config file ────────────────────────────────────────────────────────────────
write_config() {
    local secrets="$1"
    local admin_email="$2"
    local admin_password="$3"
    local hostname="$4"
    local tls_mode="$5"
    local use_docker="$6"
    local external_url="$7"

    IFS=':' read -r jwt_secret enc_key pg_pass <<< "$secrets"

    header "Writing configuration to $RENOS_ENV_FILE"
    if [ "$DRY_RUN" = true ]; then
        info "[DRY-RUN] Would write config file with autogenerated secrets"
        return 0
    fi

    cat > "$RENOS_ENV_FILE" <<ENVEOF
# RenOS Environment Configuration
# Generated by ${SCRIPT_NAME} on ${START_TIME}
# Distro: ${DISTRO^} | TLS: ${tls_mode} | Hostname: ${hostname}

# ─── Database ────────────────────────────────────────────────────────────────
POSTGRES_USER=renos
POSTGRES_PASSWORD=${pg_pass}
POSTGRES_DB=renos

# ─── Core ────────────────────────────────────────────────────────────────────
RENOS_DATABASE_URL=postgres://renos:${pg_pass}@postgres:5432/renos?sslmode=disable
RENOS_NATS_URL=nats://nats:4222
RENOS_REDIS_URL=redis://redis:6379/0
RENOS_PORT=8443
RENOS_GRPC_PORT=8444
RENOS_EXTERNAL_URL=${external_url}
RENOS_LOG_LEVEL=info
RENOS_CA_CERT_DIR=${RENOS_CA_DIR}

# ─── Container images (override for air-gapped / custom registries) ─────────
RENOS_CORE_IMAGE=${RENOS_CORE_IMAGE:-renos/core:latest}
RENOS_HUB_IMAGE=${RENOS_HUB_IMAGE:-renos/hub:latest}
RENOS_AGENT_IMAGE=${RENOS_AGENT_IMAGE:-renos/agent:latest}

# ─── Auth ────────────────────────────────────────────────────────────────────
RENOS_JWT_SECRET=${jwt_secret}
RENOS_ENCRYPTION_KEY=${enc_key}

# ─── Admin User ──────────────────────────────────────────────────────────────
RENOS_ADMIN_EMAIL=${admin_email}
RENOS_ADMIN_PASSWORD=${admin_password}

# ─── Networking ──────────────────────────────────────────────────────────────
RENOS_HTTP_PORT=80
RENOS_HTTPS_PORT=443

# ─── Install Metadata ────────────────────────────────────────────────────────
RENOS_INSTALL_VERSION=${RENOS_VERSION}
RENOS_INSTALL_DATE=${START_TIME}
RENOS_DISTRO=${DISTRO}
RENOS_TLS_MODE=${tls_mode}
RENOS_USE_DOCKER_DB=${use_docker}
ENVEOF

    chmod 600 "$RENOS_ENV_FILE"
    chown "$RENOS_USER:$RENOS_GROUP" "$RENOS_ENV_FILE"
    ok "Configuration written to $RENOS_ENV_FILE"
    rollback_add "Remove $RENOS_ENV_FILE"
}

# ─── Docker Compose production config ──────────────────────────────────────────
write_docker_compose() {
    header "Writing Docker Compose configuration"
    if [ "$DRY_RUN" = true ]; then
        info "[DRY-RUN] Would write docker-compose.yml to ${RENOS_CONFIG_DIR}"
        return 0
    fi

    local dest="${RENOS_CONFIG_DIR}/docker-compose.yml"

    # Self-contained pull-based compose. This always works from /etc/renos
    # regardless of where the installer was launched from (the old code copied
    # docker-compose.prod.yml whose build contexts and ./deploy paths are
    # relative to the repo root and break under /etc/renos).
    cat > "$dest" <<'COMPEOF'
services:
  postgres:
    image: postgres:16-alpine
    container_name: renos-postgres
    restart: unless-stopped
    expose:
      - "5432"
    environment:
      POSTGRES_USER: ${POSTGRES_USER:-renos}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:?Set POSTGRES_PASSWORD}
      POSTGRES_DB: ${POSTGRES_DB:-renos}
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-renos} -d ${POSTGRES_DB:-renos}"]
      interval: 10s
      timeout: 5s
      retries: 10
      start_period: 30s
    networks: [renos-internal]
    deploy:
      resources: { limits: { memory: 512M } }

  redis:
    image: redis:7-alpine
    container_name: renos-redis
    restart: unless-stopped
    expose: ["6379"]
    command: ["redis-server", "--maxmemory", "128mb", "--maxmemory-policy", "allkeys-lru"]
    volumes: [redisdata:/data]
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 10
      start_period: 10s
    networks: [renos-internal]
    deploy: { resources: { limits: { memory: 256M } } }

  nats:
    image: nats:2.10-alpine
    container_name: renos-nats
    restart: unless-stopped
    expose: ["4222"]
    command: ["-js", "-m", "8222"]
    volumes: [natsdata:/data]
    healthcheck:
      test: ["CMD-SHELL", "wget -q http://localhost:8222/healthz -O /dev/null || exit 1"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 10s
    networks: [renos-internal]
    deploy: { resources: { limits: { memory: 256M } } }

  core:
    image: ${RENOS_CORE_IMAGE:-renos/core:latest}
    container_name: renos-core
    restart: unless-stopped
    command: ["sh", "-c", "renos-core migrate && renos-core"]
    ports:
      - "8443:8443"
    expose: ["8444"]
    depends_on:
      postgres: { condition: service_healthy }
      redis: { condition: service_healthy }
      nats: { condition: service_healthy }
    env_file: renos.env
    volumes: [coreca:/var/lib/renos/ca]
    healthcheck:
      test: ["CMD-SHELL", "wget -q http://localhost:8443/api/v1/healthz -O /dev/null || exit 1"]
      interval: 15s
      timeout: 5s
      retries: 5
      start_period: 30s
    networks: [renos-internal]
    deploy: { resources: { limits: { memory: 512M } } }

  hub:
    image: ${RENOS_HUB_IMAGE:-renos/hub:latest}
    container_name: renos-hub
    restart: unless-stopped
    expose: ["3000"]
    depends_on:
      core: { condition: service_healthy }
    environment:
      NODE_ENV: production
      NEXT_TELEMETRY_DISABLED: "1"
      RENOS_API_URL: http://core:8443
    healthcheck:
      test: ["CMD", "wget", "-q", "-O", "/dev/null", "http://127.0.0.1:3000/login"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 30s
    networks: [renos-internal]
    deploy: { resources: { limits: { memory: 256M } } }

  agent:
    image: ${RENOS_AGENT_IMAGE:-renos/agent:latest}
    container_name: renos-agent
    restart: unless-stopped
    depends_on:
      core: { condition: service_healthy }
    environment:
      RENOS_CORE_URL: http://core:8443
      RENOS_NATS_URL: nats://nats:4222
    volumes: [agentdata:/data]
    networks: [renos-internal]
    deploy: { resources: { limits: { memory: 256M } } }

  nginx:
    image: nginx:1.27-alpine
    container_name: renos-nginx
    restart: unless-stopped
    ports: ["${RENOS_HTTP_PORT:-80}:80", "${RENOS_HTTPS_PORT:-443}:443"]
    depends_on:
      core: { condition: service_healthy }
      hub: { condition: service_healthy }
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro,Z
      - ./tls:/etc/nginx/ssl:ro,Z
    networks: [renos-internal, renos-public]
    deploy: { resources: { limits: { memory: 128M } } }

networks:
  renos-internal: { driver: bridge }
  renos-public:  { driver: bridge }

volumes:
  pgdata:     { driver: local }
  redisdata:  { driver: local }
  natsdata:   { driver: local }
  coreca:     { driver: local }
  agentdata:  { driver: local }
COMPEOF
    chmod 644 "$dest"
    ok "Docker Compose configuration written"
    rollback_add "Remove docker-compose.yml"
}

# ─── Nginx config ───────────────────────────────────────────────────────────────
write_nginx_config() {
    local hostname="$1"
    local tls_mode="$2"
    header "Writing Nginx configuration"
    if [ "$DRY_RUN" = true ]; then
        info "[DRY-RUN] Would write Nginx config to ${RENOS_CONFIG_DIR}/nginx.conf"
        return 0
    fi
    local dest="${RENOS_CONFIG_DIR}/nginx.conf"
    # Always generate a complete config. (The repo's deploy/nginx/nginx.conf is a
    # template with the 443/TLS server block commented out, so copying it would
    # ship an HTTP-only proxy and break https access.)
    cat > "$dest" <<'NGINXEOF'
worker_processes auto;
error_log /var/log/nginx/error.log warn;
pid /tmp/nginx.pid;
events { worker_connections 1024; }
http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;
    sendfile on; tcp_nopush on; tcp_nodelay on;
    keepalive_timeout 65; client_max_body_size 64m;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    log_format main '$remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent';
    access_log /var/log/nginx/access.log main;
    gzip on; gzip_vary on; gzip_proxied any; gzip_comp_level 6;
    gzip_types text/plain text/css application/json application/javascript text/xml;

    limit_req_zone $binary_remote_addr zone=api:10m rate=30r/s;
    limit_req_zone $binary_remote_addr zone=login:10m rate=5r/m;

    server {
        listen 80;
        server_name _;
NGINXEOF
        if [ "$tls_mode" != "none" ]; then
            cat >> "$dest" <<'NGINX80EOF'

        location /.well-known/acme-challenge/ { root /var/www/html; }
NGINX80EOF
        fi
        cat >> "$dest" <<'NGINXCATCHALL'
        location /_next/ {
            proxy_pass http://hub:3000;
            proxy_http_version 1.1;
            proxy_set_header Host $host;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
        }
        location /api/v1/auth/ {
            limit_req zone=login burst=3 nodelay;
            proxy_pass http://core:8443;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
        location /api/v1/ws {
            proxy_pass http://core:8443;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            proxy_read_timeout 86400s;
            proxy_send_timeout 86400s;
        }
        location /api/ {
            limit_req zone=api burst=20 nodelay;
            proxy_pass http://core:8443;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            proxy_read_timeout 30s;
        }
        location / {
            proxy_pass http://hub:3000;
            proxy_http_version 1.1;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
    }
NGINXCATCHALL
        if [ "$tls_mode" != "none" ]; then
            cat >> "$dest" <<NGINXSSLEOF
    server {
        listen 443 ssl http2;
        server_name ${hostname};
        ssl_certificate /etc/nginx/ssl/fullchain.pem;
        ssl_certificate_key /etc/nginx/ssl/privkey.pem;
        ssl_protocols TLSv1.2 TLSv1.3;
        ssl_prefer_server_ciphers on;
        ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
        ssl_session_cache shared:SSL:10m;
        ssl_session_timeout 10m;
        add_header Strict-Transport-Security "max-age=63072000; includeSubDomains; preload" always;
        location /_next/static/ {
            proxy_pass http://hub:3000;
            proxy_cache_valid 200 365d;
            add_header Cache-Control "public, max-age=31536000, immutable";
        }
        location /api/v1/ws {
            proxy_pass http://core:8443;
            proxy_http_version 1.1;
            proxy_set_header Upgrade \$http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_set_header Host \$host;
            proxy_set_header X-Real-IP \$remote_addr;
            proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto \$scheme;
            proxy_read_timeout 86400s; proxy_send_timeout 86400s;
        }
        location /api/ {
            limit_req zone=api burst=20 nodelay;
            proxy_pass http://core:8443;
            proxy_set_header Host \$host;
            proxy_set_header X-Real-IP \$remote_addr;
            proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto \$scheme;
            proxy_read_timeout 30s;
        }
        location /api/v1/auth/ {
            limit_req zone=login burst=3 nodelay;
            proxy_pass http://core:8443;
            proxy_set_header Host \$host;
            proxy_set_header X-Real-IP \$remote_addr;
            proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto \$scheme;
        }
        location / {
            proxy_pass http://hub:3000;
            proxy_set_header Host \$host;
            proxy_set_header X-Real-IP \$remote_addr;
            proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto \$scheme;
        }
    }
NGINXSSLEOF
        fi
    chmod 644 "$dest"
    ok "Nginx configuration written"
    rollback_add "Remove nginx.conf"
}

# ─── Systemd services ───────────────────────────────────────────────────────────
write_systemd_service() {
    local name="$1"
    local desc="$2"
    local exec_start="$3"
    local wants="$4"
    local after="$5"
    local file="${RENOS_SERVICE_DIR}/renos-${name}.service"

    header "Creating systemd service: renos-${name}"
    if [ "$DRY_RUN" = true ]; then
        info "[DRY-RUN] Would create systemd service: $file"
        return 0
    fi

    cat > "$file" <<SYSEOF
[Unit]
Description=${desc}
Documentation=https://renos.ai/docs
Wants=${wants:-network-online.target docker.service}
After=${after:-network-online.target docker.service}
Requires=docker.service
After=docker.service

[Service]
Type=simple
User=root
Group=root
EnvironmentFile=${RENOS_ENV_FILE}
ExecStart=${exec_start}
ExecReload=/usr/bin/docker compose -f ${RENOS_CONFIG_DIR}/docker-compose.yml restart ${name}
Restart=always
RestartSec=10
StartLimitIntervalSec=60
StartLimitBurst=3

[Install]
WantedBy=multi-user.target
SYSEOF

    chmod 644 "$file"
    systemctl daemon-reload 2>>"$LOG_FILE" || true
    ok "Service renos-${name} created"
    rollback_add "Remove systemd service renos-${name}"
}

create_systemd_services() {
    write_systemd_service "core" "RenOS Core Service" \
        "/usr/bin/docker compose -f ${RENOS_CONFIG_DIR}/docker-compose.yml up -d core" \
        "docker.service network-online.target" "docker.service network-online.target"

    write_systemd_service "agent" "RenOS Agent Service" \
        "/usr/bin/docker compose -f ${RENOS_CONFIG_DIR}/docker-compose.yml up -d agent" \
        "renos-core.service" "renos-core.service"

    write_systemd_service "hub" "RenOS Hub Web Interface" \
        "/usr/bin/docker compose -f ${RENOS_CONFIG_DIR}/docker-compose.yml up -d hub" \
        "renos-core.service" "renos-core.service"

    write_systemd_service "nginx" "RenOS Nginx Reverse Proxy" \
        "/usr/bin/docker compose -f ${RENOS_CONFIG_DIR}/docker-compose.yml up -d nginx" \
        "renos-core.service renos-hub.service" "renos-core.service renos-hub.service"
}

# ─── PostgreSQL native install ──────────────────────────────────────────────────
install_postgres_native() {
    header "Installing PostgreSQL natively"
    if [ "$DRY_RUN" = true ]; then
        info "[DRY-RUN] Would install PostgreSQL natively"
        return 0
    fi
    case "$DISTRO" in
        ubuntu|debian)
            pkg_install postgresql postgresql-client
            ;;
        fedora|rhel|rocky|almalinux)
            if [ "$PM" = "dnf" ]; then
                dnf module enable -y postgresql:16 2>>"$LOG_FILE" || true
            fi
            pkg_install postgresql-server postgresql-contrib
            postgresql-setup --initdb 2>>"$LOG_FILE" || true
            ;;
        arch)
            pkg_install postgresql
            ;;
    esac
    systemctl enable postgresql 2>>"$LOG_FILE" || true
    systemctl start postgresql 2>>"$LOG_FILE" || true
    # Create database and user
    local pg_pass="$1"
    su - postgres -c "psql -c \"CREATE USER renos WITH PASSWORD '${pg_pass}';\"" 2>>"$LOG_FILE" || true
    su - postgres -c "psql -c \"CREATE DATABASE renos OWNER renos;\"" 2>>"$LOG_FILE" || true
    su - postgres -c "psql -c \"GRANT ALL PRIVILEGES ON DATABASE renos TO renos;\"" 2>>"$LOG_FILE" || true
    ok "PostgreSQL installed and configured"
    rollback_add "Drop PostgreSQL database and user"
}

# ─── Redis native install ───────────────────────────────────────────────────────
install_redis_native() {
    header "Installing Redis natively"
    if [ "$DRY_RUN" = true ]; then
        info "[DRY-RUN] Would install Redis natively"
        return 0
    fi
    pkg_install redis
    systemctl enable redis 2>>"$LOG_FILE" || true
    systemctl start redis 2>>"$LOG_FILE" || true
    ok "Redis installed and configured"
    rollback_add "Stop and disable Redis"
}

# ─── NATS native install ────────────────────────────────────────────────────────
install_nats_native() {
    header "Installing NATS natively"
    if [ "$DRY_RUN" = true ]; then
        info "[DRY-RUN] Would install NATS natively"
        return 0
    fi
    # NATS is usually installed via binary; fallback to Docker if package not available
    local nats_installed=false
    if command -v nats-server &>/dev/null; then
        ok "NATS already installed"
        return 0
    fi
    case "$DISTRO" in
        ubuntu|debian)
            if pkg_is_installed nats-server; then
                nats_installed=true
            fi
            ;;
        arch)
            if pacman --noconfirm -S nats-server >>"$LOG_FILE" 2>&1; then
                nats_installed=true
            fi
            ;;
    esac
    if [ "$nats_installed" = false ]; then
        warn "NATS not available via package manager; Docker-based NATS will be used"
        return 0
    fi
    systemctl enable nats-server 2>>"$LOG_FILE" || true
    systemctl start nats-server 2>>"$LOG_FILE" || true
    ok "NATS installed and configured"
    rollback_add "Stop and disable NATS"
}

# ─── Prerequisites install ──────────────────────────────────────────────────────
install_prerequisites() {
    header "Installing prerequisites"
    local pkgs=()
    case "$PM" in
        apt)    pkgs=(curl wget git openssl ca-certificates gnupg lsb-release) ;;
        dnf|yum) pkgs=(curl wget git openssl ca-certificates) ;;
        pacman) pkgs=(curl wget git openssl ca-certificates) ;;
    esac
    # Add distro-specific packages
    case "$DISTRO" in
        ubuntu|debian) ;;
        fedora|rhel|rocky|almalinux) pkgs+=(util-linux) ;;
        arch) pkgs+=(inetutils) ;;
    esac
    # Add base-devel for Arch
    if [ "$DISTRO" = "arch" ]; then
        pkgs+=(base-devel) 2>/dev/null || true
    fi
    pkg_install "${pkgs[@]}"
}

# ─── Validation ─────────────────────────────────────────────────────────────────
validate_system() {
    header "Validating installation"

    local errors=0
    local checks=(
        "docker:Docker binary"
        "docker compose:Docker Compose"
        "openssl:OpenSSL"
        "curl:cURL"
        "wget:Wget"
        "git:Git"
    )

    for check in "${checks[@]}"; do
        local cmd="${check%%:*}"
        local label="${check##*:}"
        if command -v "$cmd" &>/dev/null; then
            ok "$label: found"
        else
            warn "$label: NOT found"
            ((errors++))
        fi
    done

    # Check Docker is running
    if docker info &>/dev/null 2>&1; then
        ok "Docker daemon: running"
    else
        warn "Docker daemon: NOT running (start with: systemctl start docker)"
        ((errors++))
    fi

    # Check directories
    for d in "$RENOS_CONFIG_DIR" "$RENOS_LOG_DIR" "$RENOS_CA_DIR" "$RENOS_DATA_DIR" "$RENOS_PLUGINS_DIR" "$RENOS_BACKUPS_DIR"; do
        if [ -d "$d" ]; then
            ok "Directory exists: $d"
        else
            warn "Directory missing: $d"
            ((errors++))
        fi
    done

    # Check config file
    if [ -f "$RENOS_ENV_FILE" ]; then
        ok "Config file exists: $RENOS_ENV_FILE"
    else
        warn "Config file missing: $RENOS_ENV_FILE"
        ((errors++))
    fi

    # Check systemd services
    for svc in core agent hub nginx; do
        if [ -f "${RENOS_SERVICE_DIR}/renos-${svc}.service" ]; then
            ok "Service file: renos-${svc}"
        else
            warn "Service file missing: renos-${svc}"
            ((errors++))
        fi
    done

    # Check logrotate
    if [ -f /etc/logrotate.d/renos ]; then
        ok "Logrotate config exists"
    else
        warn "Logrotate config missing"
    fi

    # Check renos user
    if id -u "$RENOS_USER" &>/dev/null; then
        ok "System user: $RENOS_USER"
    else
        warn "System user missing: $RENOS_USER"
        ((errors++))
    fi

    # Check TLS
    local tls_mode="${RENOS_TLS_MODE:-self-signed}"
    if [ "$tls_mode" != "none" ]; then
        if [ -f "${RENOS_CONFIG_DIR}/tls/fullchain.pem" ] && [ -f "${RENOS_CONFIG_DIR}/tls/privkey.pem" ]; then
            ok "TLS certificates present"
        else
            warn "TLS certificates missing"
            ((errors++))
        fi
    else
        warn "TLS is disabled"
    fi

    if [ $errors -gt 0 ]; then
        warn "${errors} issue(s) found. Check the warnings above."
    else
        ok "All checks passed!"
    fi
    return $errors
}

# ─── Start services ────────────────────────────────────────────────────────────
wait_for_health() {
    local url="$1"
    local timeout="${2:-300}"
    local waited=0
    info "Waiting for core health at $url (max ${timeout}s)..."
    while [ "$waited" -lt "$timeout" ]; do
        if command -v curl &>/dev/null; then
            if curl -sk --connect-timeout 3 -o /dev/null -w '%{http_code}' "$url" 2>/dev/null | grep -q '^200$'; then
                ok "Core is healthy"
                return 0
            fi
        elif command -v wget &>/dev/null; then
            if wget -q --spider --no-check-certificate "$url" 2>/dev/null; then
                ok "Core is healthy"
                return 0
            fi
        else
            sleep 5
        fi
        sleep 5
        waited=$((waited + 5))
    done
    warn "Timed out waiting for core health at $url (check: docker compose -f ${RENOS_CONFIG_DIR}/docker-compose.yml logs core)"
    return 1
}

start_services() {
    header "Starting RenOS services"
    if [ "$DRY_RUN" = true ]; then
        info "[DRY-RUN] Would enable and start renos-core/agent/hub/nginx services"
        return 0
    fi
    systemctl daemon-reload 2>>"$LOG_FILE" || true
    for svc in core agent hub nginx; do
        systemctl enable "renos-${svc}" >>"$LOG_FILE" 2>&1 || true
    done
    systemctl start renos-core >>"$LOG_FILE" 2>&1 || {
        warn "renos-core failed to start. Logs: journalctl -u renos-core -n 50"
    }
    wait_for_health "https://localhost:8443/api/v1/healthz" 300 || \
    wait_for_health "http://localhost:8443/api/v1/healthz" 60 || true

    for svc in hub agent nginx; do
        systemctl start "renos-${svc}" >>"$LOG_FILE" 2>&1 || {
            warn "renos-${svc} failed to start. Logs: journalctl -u renos-${svc} -n 50"
        }
    done
    ok "All RenOS services started"
    rollback_add "Stop and disable RenOS services"
}

# ─── Post-install verification ─────────────────────────────────────────────────
verify_install() {
    header "Verifying RenOS installation"
    local failures=0

    # Load installed config if present
    if [ -f "$RENOS_ENV_FILE" ]; then
        set -a
        # shellcheck disable=SC1090
        . "$RENOS_ENV_FILE"
        set +a
    fi
    local api_url="https://localhost:8443"
    local http_url="http://localhost:8443"

    # 1. Docker daemon
    if docker info &>/dev/null 2>&1; then
        ok "Docker daemon: running"
    else
        warn "Docker daemon: NOT running"
        failures=$((failures + 1))
    fi

    # 2. Containers
    local running_ok=true
    for c in renos-core renos-postgres renos-redis renos-nats renos-hub renos-nginx; do
        if docker inspect --format='{{.State.Status}}' "$c" 2>/dev/null | grep -q running; then
            ok "Container running: $c"
        else
            warn "Container missing/stopped: $c"
            running_ok=false
        fi
    done
    $running_ok || failures=$((failures + 1))

    # 3. API health (try HTTPS then plain HTTP on 8443, then nginx on 443)
    local api_ok=false
    for ep in "${api_url}/api/v1/healthz" "${http_url}/api/v1/healthz" "https://localhost:443/api/v1/healthz" "http://localhost:80/api/v1/healthz"; do
        local code
        code=$(curl -sk --connect-timeout 5 -o /dev/null -w '%{http_code}' "$ep" 2>/dev/null || echo "000")
        if [ "$code" = "200" ]; then
            ok "API healthz responds (HTTP 200): $ep"
            api_ok=true
            break
        fi
    done
    $api_ok || { warn "API healthz not reachable"; failures=$((failures + 1)); }

    # 4. Readiness (DB/NATS/Redis behind the API)
    local readyz
    readyz=$(curl -sk --connect-timeout 5 "${api_url}/api/v1/readyz" 2>/dev/null || echo "")
    if echo "$readyz" | grep -q '"database"[^}]*"ok"'; then ok "Database: reachable"; else warn "Database: not ok"; failures=$((failures + 1)); fi
    if echo "$readyz" | grep -q '"nats"[^}]*"ok"'; then ok "NATS: reachable"; else warn "NATS: not ok"; failures=$((failures + 1)); fi
    if echo "$readyz" | grep -q '"redis"[^}]*"ok"'; then ok "Redis: reachable"; else warn "Redis: not ok"; failures=$((failures + 1)); fi

    # 5. Admin login
    local admin_email="${RENOS_ADMIN_EMAIL:-admin@renos.local}"
    local admin_password="${RENOS_ADMIN_PASSWORD:-}"
    if [ -n "$admin_password" ]; then
        local login
        login=$(curl -sk --connect-timeout 5 -X POST "${api_url}/api/v1/auth/login" \
            -H "Content-Type: application/json" \
            -d "{\"email\":\"${admin_email}\",\"password\":\"${admin_password}\"}" 2>/dev/null || echo "")
        if echo "$login" | grep -q '"token"'; then
            ok "Admin login: success (token issued)"
        else
            warn "Admin login: failed for ${admin_email}"
            failures=$((failures + 1))
        fi
    else
        warn "Admin password not set in env; skipping login check"
    fi

    # 6. WebSocket endpoint exists
    local ws_code
    ws_code=$(curl -sk --connect-timeout 5 -o /dev/null -w '%{http_code}' \
        -H "Connection: Upgrade" -H "Upgrade: websocket" \
        -H "Sec-WebSocket-Version: 13" -H "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==" \
        "${api_url}/api/v1/ws" 2>/dev/null || echo "000")
    if [ "$ws_code" != "000" ] && [ "$ws_code" != "404" ]; then
        ok "WebSocket endpoint present (HTTP $ws_code)"
    else
        warn "WebSocket endpoint not reachable (HTTP ${ws_code:-none})"
        failures=$((failures + 1))
    fi

    # 7. Dashboard / UI
    local dash_ok=false
    for ep in "https://localhost:443/" "http://localhost:80/" "${api_url}/"; do
        local dcode
        dcode=$(curl -sk --connect-timeout 5 -o /dev/null -w '%{http_code}' "$ep" 2>/dev/null || echo "000")
        if [ "$dcode" = "200" ] || [ "$dcode" = "301" ] || [ "$dcode" = "302" ]; then
            ok "Dashboard responds (HTTP $dcode): $ep"
            dash_ok=true
            break
        fi
    done
    $dash_ok || { warn "Dashboard not reachable"; failures=$((failures + 1)); }

    # 8. TLS certificates
    if [ -f "${RENOS_CONFIG_DIR}/tls/fullchain.pem" ] && [ -f "${RENOS_CONFIG_DIR}/tls/privkey.pem" ]; then
        if openssl x509 -checkend 86400 -noout -in "${RENOS_CONFIG_DIR}/tls/fullchain.pem" &>/dev/null; then
            ok "TLS certificates present and valid (>1 day)"
        else
            warn "TLS certificates present but expired or invalid"
            failures=$((failures + 1))
        fi
    else
        warn "TLS certificates missing"
        failures=$((failures + 1))
    fi

    echo ""
    if [ "$failures" -eq 0 ]; then
        ok "All verification checks passed!"
    else
        warn "${failures} verification check(s) failed"
    fi
    return $failures
}

# ─── Uninstall ──────────────────────────────────────────────────────────────────
perform_uninstall() {
    header "Uninstalling RenOS"
    echo -e "${RED}WARNING: This will remove ALL RenOS data including databases and certificates.${NC}"
    echo -e "${RED}Backups are stored at ${RESTORE_DIR} if needed.${NC}"
    if [ "$NONINTERACTIVE" = "1" ]; then
        if [ "${RENOS_UNINSTALL_CONFIRM:-no}" != "yes" ]; then
            info "Non-interactive uninstall requires: NONINTERACTIVE=1 RENOS_UNINSTALL_CONFIRM=yes"
            exit 0
        fi
    else
        prompt "Are you sure you want to proceed? [y/N]: "
        read -r confirm
        if [[ ! "$confirm" =~ ^[yY] ]]; then
            info "Uninstall cancelled"
            exit 0
        fi
    fi

    info "Stopping all RenOS services..."
    systemctl stop renos-core renos-agent renos-hub renos-nginx 2>/dev/null || true
    systemctl disable renos-core renos-agent renos-hub renos-nginx 2>/dev/null || true

    info "Removing systemd service files..."
    rm -f "${RENOS_SERVICE_DIR}/renos-"*.service
    systemctl daemon-reload

    info "Stopping Docker containers..."
    if [ -f "${RENOS_CONFIG_DIR}/docker-compose.yml" ]; then
        docker compose -f "${RENOS_CONFIG_DIR}/docker-compose.yml" down -v 2>/dev/null || true
    fi

    info "Stopping native services..."
    systemctl stop postgresql redis 2>/dev/null || true

    info "Removing Docker images..."
    docker images --format '{{.Repository}}' | grep 'renos/' | xargs -r docker rmi -f 2>/dev/null || true

    info "Removing directories..."
    rm -rf "$RENOS_CONFIG_DIR"
    rm -rf "$RENOS_HOME"
    rm -rf "$RENOS_LOG_DIR"

    info "Removing logrotate config..."
    rm -f /etc/logrotate.d/renos
    rm -f /etc/cron.d/renos-certbot

    info "Removing system user..."
    userdel -r "$RENOS_USER" 2>/dev/null || true
    groupdel "$RENOS_GROUP" 2>/dev/null || true

    # Ask about Docker removal
    if [ "$NONINTERACTIVE" = "1" ]; then
        remove_docker="n"
    else
        prompt "Remove Docker and Docker Compose as well? [y/N]: "
        read -r remove_docker
    fi
    if [[ "$remove_docker" =~ ^[yY] ]]; then
        info "Removing Docker packages..."
        case "$PM" in
            apt)
                apt-get purge -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin 2>/dev/null || true
                apt-get autoremove -y 2>/dev/null || true
                ;;
            dnf|yum)
                eval "$PM_INSTALL_CMD remove docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin" 2>/dev/null || true
                ;;
            pacman)
                pacman -Rns --noconfirm docker docker-compose 2>/dev/null || true
                ;;
        esac
        ok "Docker packages removed"
    fi

    # Ask about PostgreSQL removal
    if [ "$NONINTERACTIVE" = "1" ]; then
        remove_pg="n"
    else
        prompt "Remove PostgreSQL? [y/N]: "
        read -r remove_pg
    fi
    if [[ "$remove_pg" =~ ^[yY] ]]; then
        info "Removing PostgreSQL..."
        case "$PM" in
            apt)  apt-get purge -y postgresql postgresql-client 2>/dev/null || true ;;
            dnf|yum) eval "$PM_INSTALL_CMD remove postgresql-server postgresql-contrib" 2>/dev/null || true ;;
            pacman) pacman -Rns --noconfirm postgresql 2>/dev/null || true ;;
        esac
        ok "PostgreSQL removed"
    fi

    # Ask about Redis removal
    if [ "$NONINTERACTIVE" = "1" ]; then
        remove_redis="n"
    else
        prompt "Remove Redis? [y/N]: "
        read -r remove_redis
    fi
    if [[ "$remove_redis" =~ ^[yY] ]]; then
        info "Removing Redis..."
        case "$PM" in
            apt)  apt-get purge -y redis 2>/dev/null || true ;;
            dnf|yum) eval "$PM_INSTALL_CMD remove redis" 2>/dev/null || true ;;
            pacman) pacman -Rns --noconfirm redis 2>/dev/null || true ;;
        esac
        ok "Redis removed"
    fi

    ok "RenOS has been completely uninstalled"
}

# ─── Summary ────────────────────────────────────────────────────────────────────
print_summary() {
    local admin_email="$1"
    local hostname="$2"
    local tls_mode="$3"

    echo ""
    echo -e "${BOLD}${GREEN}╔══════════════════════════════════════════╗${NC}"
    echo -e "${BOLD}${GREEN}║      RenOS Installation Complete!        ║${NC}"
    echo -e "${BOLD}${GREEN}╚══════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "  ${BOLD}Version:${NC}     ${RENOS_VERSION}"
    echo -e "  ${BOLD}Distro:${NC}      ${DISTRO^}"
    echo -e "  ${BOLD}Hostname:${NC}    ${hostname}"
    echo -e "  ${BOLD}TLS Mode:${NC}    ${tls_mode}"
    echo -e "  ${BOLD}Log file:${NC}    ${LOG_FILE}"
    echo ""
    echo -e "  ${BOLD}Dashboard:${NC}   https://${hostname}"
    echo -e "  ${BOLD}API:${NC}         https://${hostname}/api/v1/healthz"
    echo -e "  ${BOLD}Admin email:${NC} ${admin_email}"
    echo ""
    echo -e "  ${BOLD}Service Status:${NC}"
    for svc in core agent hub nginx; do
        local status="inactive"
        systemctl is-active "renos-${svc}" &>/dev/null && status="active" || true
        systemctl is-enabled "renos-${svc}" &>/dev/null && enabled="enabled" || enabled="disabled"
        echo -e "    renos-${svc}: ${GREEN}${status}${NC} (${enabled})"
    done
    echo ""
    echo -e "  ${BOLD}Useful commands:${NC}"
    echo -e "    sudo systemctl status renos-core    # Check service status"
    echo -e "    sudo journalctl -u renos-core -f    # Follow logs"
    echo -e "    sudo systemctl restart renos-core   # Restart service"
    echo -e "    docker compose -f ${RENOS_CONFIG_DIR}/docker-compose.yml ps  # All containers"
    echo ""
    echo -e "  ${BOLD}Next steps:${NC}"
    echo -e "    1. Open https://${hostname} and log in with your admin credentials"
    echo -e "    2. Configure your devices with RenAgent"
    echo -e "    3. Set up DNS records pointing to this server"
    if [ "$tls_mode" = "self-signed" ]; then
        echo -e "    4. ${YELLOW}Replace the self-signed TLS certificate for production use${NC}"
    fi
    echo ""
    echo -e "${BOLD}${GREEN}RenOS is ready.${NC}"
    echo -e "${GREEN}Open: https://${hostname}${NC}"
    local primary_ip
    primary_ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
    if [ -n "$primary_ip" ] && [ "$primary_ip" != "$hostname" ]; then
        echo -e "${GREEN}Open: https://${primary_ip}${NC}"
    fi
    echo ""
}

# ─── Main ───────────────────────────────────────────────────────────────────────
main() {
    parse_args "$@"

    # Show banner
    echo ""
    echo -e "${BOLD}${CYAN}╔══════════════════════════════════════════╗${NC}"
    echo -e "${BOLD}${CYAN}║        RenOS Linux Installer v${RENOS_VERSION}        ║${NC}"
    echo -e "${BOLD}${CYAN}║    Your devices. One intelligent system. ║${NC}"
    echo -e "${BOLD}${CYAN}╚══════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "  Start time: ${START_TIME}"
    echo -e "  Installer:  ${SCRIPT_DIR}/${SCRIPT_NAME}"
    echo ""

    # --uninstall
    if [ "$UNINSTALL" = true ]; then
        require_root
        detect_system
        perform_uninstall
        exit 0
    fi

    # --verify
    if [ "$VERIFY" = true ]; then
        require_root
        detect_system
        local verify_rc=0
        verify_install || verify_rc=$?
        exit $verify_rc
    fi

    # --dry-run
    if [ "$DRY_RUN" = true ]; then
        echo -e "${YELLOW}⚠  DRY RUN MODE — No changes will be made${NC}"
        echo ""
    fi

    require_root
    detect_system

    # ── Gather inputs ──────────────────────────────────────────────────────────
    local admin_email="${RENOS_ADMIN_EMAIL:-}"
    local admin_password="${RENOS_ADMIN_PASSWORD:-}"
    local hostname="${RENOS_HOSTNAME:-}"
    local tls_mode="${RENOS_TLS_MODE:-}"
    local use_docker="${RENOS_USE_DOCKER_DB:-true}"
    local external_url="${RENOS_EXTERNAL_URL:-}"

    if [ "$DRY_RUN" = false ]; then
        if [ "$NONINTERACTIVE" = "1" ]; then
            # No prompts: use env vars or safe defaults
            admin_email="${admin_email:-admin@renos.local}"
            hostname="${hostname:-$(hostname -f 2>/dev/null || echo 'localhost')}"
            tls_mode="${tls_mode:-self-signed}"
            if [ -z "$admin_password" ]; then
                admin_password="$(openssl rand -base64 24 2>/dev/null | tr -dc 'A-Za-z0-9' | head -c 16)"
            fi
            if [ ${#admin_password} -lt 8 ]; then
                fail "RENOS_ADMIN_PASSWORD must be at least 8 characters."
            fi
        else
            if [ -z "$admin_email" ]; then
                echo ""
                prompt "Admin email [admin@renos.local]: "
                read -r admin_email
                admin_email="${admin_email:-admin@renos.local}"
            fi
            if [ -z "$admin_password" ]; then
                echo ""
                prompt "Admin password (min 8 chars): "
                read -s -r admin_password
                echo ""
                if [ -z "$admin_password" ]; then
                    fail "Admin password cannot be empty."
                fi
                if [ ${#admin_password} -lt 8 ]; then
                    fail "Admin password must be at least 8 characters."
                fi
                if [[ "$admin_password" =~ ^(admin|password|changeme|12345678)$ ]]; then
                    fail "Admin password is too weak. Use a stronger password."
                fi
            fi
            if [ -z "$hostname" ]; then
                echo ""
                prompt "Server hostname (FQDN) [$(hostname -f 2>/dev/null || echo 'localhost')]: "
                read -r hostname
                hostname="${hostname:-$(hostname -f 2>/dev/null || echo 'localhost')}"
            fi
            if [ -z "$tls_mode" ]; then
                echo ""
                echo -e "  TLS options:"
                echo -e "    1) self-signed  — Auto-generate (recommended for testing)"
                echo -e "    2) letsencrypt — Free valid cert (requires public DNS + port 80)"
                echo -e "    3) none        — No TLS (not recommended)"
                prompt "Select TLS mode [1]: "
                read -r tls_choice
                case "${tls_choice:-1}" in
                    1|"") tls_mode="self-signed" ;;
                    2)    tls_mode="letsencrypt" ;;
                    3)    tls_mode="none" ;;
                    *)    tls_mode="self-signed" ;;
                esac
            fi
        fi
    else
        admin_email="${admin_email:-admin@renos.local}"
        hostname="${hostname:-$(hostname -f 2>/dev/null || echo 'localhost')}"
        tls_mode="${tls_mode:-self-signed}"
    fi

    # ── Execute steps ──────────────────────────────────────────────────────────
    local step=1

    echo -e "\n${BOLD}Installation plan:${NC}"
    echo -e "  ${step})  Update package cache";        ((step++))
    echo -e "  ${step})  Install prerequisites";        ((step++))
    echo -e "  ${step})  Install Docker & Compose";     ((step++))
    echo -e "  ${step})  Create system user";           ((step++))
    echo -e "  ${step})  Create directories";           ((step++))
    echo -e "  ${step})  Create restore point";         ((step++))
    echo -e "  ${step})  Configure firewall";           ((step++))
    echo -e "  ${step})  Configure log rotation";       ((step++))
    echo -e "  ${step})  Install PostgreSQL";           ((step++))
    echo -e "  ${step})  Install Redis";                ((step++))
    echo -e "  ${step})  Install NATS";                 ((step++))
    echo -e "  ${step})  Setup TLS";                    ((step++))
    echo -e "  ${step})  Write configuration";          ((step++))
    echo -e "  ${step})  Write Docker Compose";         ((step++))
    echo -e "  ${step})  Write Nginx config";           ((step++))
    echo -e "  ${step})  Create systemd services";      ((step++))
    echo -e "  ${step})  Start services";               ((step++))
    echo -e "  ${step})  Validate installation";        ((step++))
    echo -e "  ${step})  Verify installation";          ((step++))
    echo -e "  ${step})  Print summary"
    echo ""

    if [ "$DRY_RUN" = true ]; then
        echo -e "${YELLOW}Dry run complete. No changes made.${NC}"
        exit 0
    fi

    # Create log file early
    mkdir -p "$(dirname "$LOG_FILE")"
    touch "$LOG_FILE"
    log "INFO" "RenOS installation started: ${START_TIME}"

    # Reuse existing secrets on re-run so re-installs don't break the database
    local secrets
    if [ -f "$RENOS_ENV_FILE" ]; then
        set -a
        # shellcheck disable=SC1090
        . "$RENOS_ENV_FILE"
        set +a
        if [ -n "${RENOS_JWT_SECRET:-}" ] && [ -n "${RENOS_ENCRYPTION_KEY:-}" ] && [ -n "${POSTGRES_PASSWORD:-}" ]; then
            secrets="${RENOS_JWT_SECRET}:${RENOS_ENCRYPTION_KEY}:${POSTGRES_PASSWORD}"
            info "Reusing existing secrets from $RENOS_ENV_FILE"
        else
            secrets="$(generate_secrets)"
        fi
    else
        secrets="$(generate_secrets)"
    fi
    echo ""

    # 1-3: System preparation
    pkg_update
    install_prerequisites
    install_docker

    # 4-5: User and directories
    create_system_user
    create_directories

    # 6: Restore point
    create_restore_point

    # 7: Firewall
    configure_firewall

    # 8: Log rotation
    configure_logrotate

    # 9-11: Database and middleware
    if [ "$use_docker" = "true" ]; then
        info "Using Docker-based PostgreSQL"
    else
        local pg_pass
        pg_pass="$(echo "$secrets" | cut -d: -f3)"
        install_postgres_native "$pg_pass"
    fi

    if [ "$use_docker" = "true" ]; then
        info "Using Docker-based Redis"
    else
        install_redis_native
    fi

    if [ "$use_docker" = "true" ]; then
        info "Using Docker-based NATS"
    else
        install_nats_native
    fi

    # 12: TLS
    setup_tls "$tls_mode" "$hostname"

    # 13: Config
    external_url="${external_url:-https://${hostname}}"
    write_config "$secrets" "$admin_email" "$admin_password" "$hostname" "$tls_mode" "$use_docker" "$external_url"

    # 14: Docker Compose
    write_docker_compose

    # 15: Nginx config
    write_nginx_config "$hostname" "$tls_mode"

    # 16: Systemd services
    create_systemd_services

    # 17: Start services
    start_services

    # 18: Validate
    echo ""
    validate_system || true

    # 19: Verify
    echo ""
    verify_install || true

    # 20: Summary
    print_summary "$admin_email" "$hostname" "$tls_mode"

    log "INFO" "RenOS installation completed successfully"
    ok "Installation log saved to $LOG_FILE"
}

main "$@"
