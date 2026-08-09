#!/usr/bin/env bash
# =============================================================================
# RenOS Docker Installer
# =============================================================================
# One-command launcher for the RenOS Docker-based production deployment.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/renos-project/renos/main/installer/docker/run.sh | bash
#   ./run.sh
#   ./run.sh --dev        # Development mode (self-signed TLS, debug logging)
#   ./run.sh --build      # Force rebuild images
#   ./run.sh --pull       # Pull pre-built images from registry
#   ./run.sh --update     # Pull latest images and recreate changed containers
#   ./run.sh --uninstall  # Stop and remove all containers, volumes, images
#   ./run.sh --help       # Show this help
#
# Flags:
#   --dev         Development mode: self-signed TLS, debug logging, exposed ports
#   --build       Force rebuild Docker images from source
#   --pull        Pull pre-built images (ignored if --build is set)
#   --update      Pull latest images and recreate changed containers
#   --uninstall   Stop and remove all containers, volumes, and images
#   --help        Display this help message and exit
#
# The script is fully idempotent: running it multiple times is safe.
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
COMPOSE_FILE="$SCRIPT_DIR/docker-compose.yml"
ENV_FILE="$SCRIPT_DIR/.env"
COMPOSE_PROJECT="renos"

# ─── Colors ───────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

info()  { echo -e "${BLUE}ℹ${NC}  $*"; }
ok()    { echo -e "${GREEN}✓${NC}  $*"; }
warn()  { echo -e "${YELLOW}⚠${NC}  $*"; }
fail()  { echo -e "${RED}✗${NC}  $*"; exit 1; }
header(){ echo -e "\n${CYAN}══ $* ══${NC}\n"; }

# ─── Help ─────────────────────────────────────────────────────────────────────
show_help() {
    cat <<EOF
RenOS Docker Installer

Usage:  $0 [OPTIONS]

Options:
  --dev         Development mode (self-signed TLS, debug logging, exposed ports)
  --build       Force rebuild Docker images from source
  --pull        Pull pre-built images from container registry
  --update      Pull latest images and recreate changed containers
  --uninstall   Stop and remove all containers, volumes, images, and .env
  --help        Show this message

Without options, the script runs in production mode:
  - Generates .env with secure secrets (if missing)
  - Builds images (or pulls if --pull is given)
  - Starts all services
  - Waits for health checks
  - Creates the admin user on first run
  - Prints the dashboard URL

The script is fully idempotent. Re-run to update configuration or restart.
EOF
    exit 0
}

# ─── Parse Flags ──────────────────────────────────────────────────────────────
DEV_MODE=false
FORCE_BUILD=false
PULL_IMAGES=false
UPDATE_MODE=false
UNINSTALL_MODE=false

for arg in "$@"; do
    case "$arg" in
        --dev)       DEV_MODE=true ;;
        --build)     FORCE_BUILD=true ;;
        --pull)      PULL_IMAGES=true ;;
        --update)    UPDATE_MODE=true ;;
        --uninstall) UNINSTALL_MODE=true ;;
        --help)      show_help ;;
        *)           fail "Unknown option: $arg. Use --help for usage." ;;
    esac
done

# ─── Uninstall Mode ───────────────────────────────────────────────────────────
if [ "$UNINSTALL_MODE" = true ]; then
    header "Uninstalling RenOS"
    warn "This will remove all containers, volumes, and data."
    echo ""
    read -p "  Are you sure? Type 'yes' to confirm: " CONFIRM
    if [ "$CONFIRM" != "yes" ]; then
        echo "  Aborted."
        exit 0
    fi
    echo ""

    info "Stopping services..."
    docker compose -p "$COMPOSE_PROJECT" -f "$COMPOSE_FILE" down --remove-orphans 2>/dev/null || true

    info "Removing volumes..."
    docker compose -p "$COMPOSE_PROJECT" -f "$COMPOSE_FILE" down -v 2>/dev/null || true

    info "Removing RenOS Docker images..."
    docker images --filter "reference=renos-*" -q 2>/dev/null | xargs -r docker rmi -f 2>/dev/null || true

    if [ -f "$ENV_FILE" ]; then
        info "Removing .env file..."
        rm -f "$ENV_FILE"
    fi

    ok "RenOS has been uninstalled."
    exit 0
fi

# ─── Banner ───────────────────────────────────────────────────────────────────
echo ""
echo "╔══════════════════════════════════════════╗"
echo "║        RenOS Docker Installer            ║"
echo "║    Your devices. One intelligent system. ║"
echo "╚══════════════════════════════════════════╝"
echo ""

if [ "$DEV_MODE" = true ]; then
    warn "Development mode enabled"
fi

# ─── 1. Check Prerequisites ───────────────────────────────────────────────────
header "Checking Prerequisites"

if ! command -v docker &>/dev/null; then
    fail "Docker is not installed. See https://docs.docker.com/get-docker/"
fi
ok "Docker found: $(docker --version)"

if ! docker compose version &>/dev/null; then
    fail "Docker Compose V2 is not installed. Update Docker."
fi
ok "Compose found: $(docker compose version)"

if ! docker info &>/dev/null 2>&1; then
    fail "Docker daemon is not running."
fi
ok "Docker daemon running"

if [ "$DEV_MODE" = false ]; then
    if [ ! -f /etc/nginx/ssl/fullchain.pem ] && [ ! -f "$SCRIPT_DIR/nginx.conf" ]; then
        warn "No TLS certificates found. Run.sh will generate self-signed certs."
    fi
fi

# ─── 2. Generate / Validate .env ──────────────────────────────────────────────
header "Configuration"

GENERATED_ENV=false

generate_secret() {
    openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | xxd -p | tr -d '\n'
}

generate_password() {
    openssl rand -base64 24 2>/dev/null | tr -d '/+=' | head -c 32 || \
    head -c 24 /dev/urandom | base64 | tr -d '/+=' | head -c 32
}

if [ ! -f "$ENV_FILE" ]; then
    info ".env not found. Generating with secure defaults..."
    GENERATED_ENV=true

    POSTGRES_PASSWORD=$(generate_password)
    REDIS_PASSWORD=$(generate_password)
    JWT_SECRET=$(generate_secret)
    ENCRYPTION_KEY=$(generate_secret)

    echo ""
    echo -e "${BLUE}Admin User Setup${NC}"
    read -p "  Hostname [renos.local]: " RENOS_HOSTNAME
    RENOS_HOSTNAME="${RENOS_HOSTNAME:-renos.local}"

    read -p "  Admin email [admin@renos.local]: " ADMIN_EMAIL
    ADMIN_EMAIL="${ADMIN_EMAIL:-admin@renos.local}"

    while true; do
        read -s -p "  Admin password: " ADMIN_PASSWORD
        echo ""
        if [ -z "$ADMIN_PASSWORD" ]; then
            echo -e "${RED}  Password cannot be empty.${NC}"
            continue
        fi
        if [ ${#ADMIN_PASSWORD} -lt 8 ]; then
            echo -e "${RED}  Password must be at least 8 characters.${NC}"
            continue
        fi
        shopt -s nocasematch
        if [[ "$ADMIN_PASSWORD" == "admin" || "$ADMIN_PASSWORD" == "password" || "$ADMIN_PASSWORD" == "changeme" || "$ADMIN_PASSWORD" == "12345678" ]]; then
            shopt -u nocasematch
            echo -e "${RED}  Password is too weak. Use a stronger password.${NC}"
            continue
        fi
        shopt -u nocasematch
        break
    done

    echo ""
    echo -e "${BLUE}TLS Configuration${NC}"
    read -p "  TLS mode (selfsigned/letsencrypt) [selfsigned]: " TLS_MODE
    TLS_MODE="${TLS_MODE:-selfsigned}"
    TLS_EMAIL="${ADMIN_EMAIL}"

    RENOS_EXTERNAL_URL="https://${RENOS_HOSTNAME}"

    if [ "$DEV_MODE" = true ]; then
        RENOS_LOG_LEVEL="debug"
        RENOS_EXTERNAL_URL="https://${RENOS_HOSTNAME}"
    else
        RENOS_LOG_LEVEL="info"
    fi

    cat > "$ENV_FILE" <<ENVEOF
# RenOS Docker Environment
# Generated by run.sh on $(date -u +%Y-%m-%dT%H:%M:%SZ)

# Host
RENOS_HOSTNAME=${RENOS_HOSTNAME}
RENOS_HTTP_PORT=80
RENOS_HTTPS_PORT=443

# PostgreSQL
POSTGRES_USER=renos
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
POSTGRES_DB=renos

# Redis
REDIS_PASSWORD=${REDIS_PASSWORD}

# Core
RENOS_EXTERNAL_URL=${RENOS_EXTERNAL_URL}
RENOS_PORT=8443
RENOS_GRPC_PORT=8444
RENOS_LOG_LEVEL=${RENOS_LOG_LEVEL}
RENOS_DATA_DIR=/var/lib/renos/data

# Auth
RENOS_JWT_SECRET=${JWT_SECRET}
RENOS_ENCRYPTION_KEY=${ENCRYPTION_KEY}

# CA
RENOS_CA_CERT_DIR=/var/lib/renos/ca

# Admin
RENOS_ADMIN_EMAIL=${ADMIN_EMAIL}
RENOS_ADMIN_PASSWORD=${ADMIN_PASSWORD}

# Build
RENOS_VERSION=dev
BUILD_COMMIT=unknown
BUILD_TIME=unknown
ENVEOF

    ok "Environment file created at .env"

    # Generate self-signed TLS certificates if needed
    if [ "$TLS_MODE" = "selfsigned" ]; then
        info "Generating self-signed TLS certificate..."
        mkdir -p "$SCRIPT_DIR/ssl"
        openssl req -x509 -nodes -days 365 -newkey rsa:4096 \
            -keyout "$SCRIPT_DIR/ssl/privkey.pem" \
            -out "$SCRIPT_DIR/ssl/fullchain.pem" \
            -subj "/CN=${RENOS_HOSTNAME}" \
            -addext "subjectAltName=DNS:${RENOS_HOSTNAME},DNS:localhost,IP:127.0.0.1" \
            2>/dev/null
        ok "Self-signed certificate generated for ${RENOS_HOSTNAME}"
    fi
else
    ok ".env file exists"

    # Source .env to check required values
    set -a
    source "$ENV_FILE"
    set +a

    MISSING=""
    [ -z "${POSTGRES_PASSWORD:-}" ] && MISSING="$MISSING POSTGRES_PASSWORD"
    [ -z "${REDIS_PASSWORD:-}" ] && MISSING="$MISSING REDIS_PASSWORD"
    [ -z "${RENOS_JWT_SECRET:-}" ] && MISSING="$MISSING RENOS_JWT_SECRET"
    [ -z "${RENOS_ENCRYPTION_KEY:-}" ] && MISSING="$MISSING RENOS_ENCRYPTION_KEY"
    [ -z "${RENOS_EXTERNAL_URL:-}" ] && MISSING="$MISSING RENOS_EXTERNAL_URL"

    if [ -n "$MISSING" ]; then
        warn "Missing required values:$MISSING"
        warn "Regenerate by removing .env: rm .env && ./run.sh"
    fi
fi

# ─── 3. Update Mode ───────────────────────────────────────────────────────────
if [ "$UPDATE_MODE" = true ]; then
    header "Updating RenOS"

    info "Pulling latest images..."
    if [ "$PULL_IMAGES" = true ]; then
        docker compose -p "$COMPOSE_PROJECT" -f "$COMPOSE_FILE" pull
    fi

    info "Rebuilding images..."
    docker compose -p "$COMPOSE_PROJECT" -f "$COMPOSE_FILE" build --pull

    info "Recreating changed containers..."
    docker compose -p "$COMPOSE_PROJECT" -f "$COMPOSE_FILE" up -d --remove-orphans --force-recreate

    ok "Update complete"
fi

# ─── 4. Build / Pull Images ───────────────────────────────────────────────────
header "Preparing Images"

BUILD_ARGS=()

if [ "$FORCE_BUILD" = true ]; then
    info "Building images (--build)..."
    BUILD_ARGS=(--build)
elif [ "$PULL_IMAGES" = true ]; then
    info "Pulling images (--pull)..."
    BUILD_ARGS=(--pull)
fi

if [ "$UPDATE_MODE" = false ]; then
    docker compose -p "$COMPOSE_PROJECT" -f "$COMPOSE_FILE" build ${BUILD_ARGS[@]:-} 2>&1 | tail -5 || true
    ok "Images ready"
fi

# ─── 5. Start Services ────────────────────────────────────────────────────────
header "Starting Services"

docker compose -p "$COMPOSE_PROJECT" -f "$COMPOSE_FILE" up -d --remove-orphans
ok "Services started"

# ─── 6. Wait for Health ───────────────────────────────────────────────────────
header "Waiting for Health Checks"

MAX_WAIT=180
WAITED=0
SERVICES=("postgres" "redis" "nats" "renos-core" "renos-hub")

while [ $WAITED -lt $MAX_WAIT ]; do
    ALL_HEALTHY=true
    for svc in "${SERVICES[@]}"; do
        STATUS=$(docker compose -p "$COMPOSE_PROJECT" -f "$COMPOSE_FILE" ps "$svc" --format "{{.Status}}" 2>/dev/null || echo "")
        if ! echo "$STATUS" | grep -qi "healthy"; then
            ALL_HEALTHY=false
            break
        fi
    done

    if [ "$ALL_HEALTHY" = true ]; then
        break
    fi

    sleep 3
    WAITED=$((WAITED + 3))
    printf "\r  Waiting... (%ds/%ds)" "$WAITED" "$MAX_WAIT"
done
echo ""

if [ $WAITED -ge $MAX_WAIT ]; then
    warn "Some services may still be starting. Check logs:"
    warn "  docker compose -f \"$COMPOSE_FILE\" logs -f"
else
    ok "All services healthy"
fi

# ─── 7. Create Admin User ─────────────────────────────────────────────────────
header "Admin User"

set +e
ADMIN_EMAIL="${RENOS_ADMIN_EMAIL:-admin@renos.local}"
ADMIN_PASSWORD="${RENOS_ADMIN_PASSWORD:-}"

if [ -n "$ADMIN_PASSWORD" ]; then
    # Source .env for all the connection details
    set -a
    source "$ENV_FILE"
    set +a

    CORE_URL="http://localhost:${RENOS_PORT:-8443}"

    # Check if admin already exists
    CREATE_ADMIN=true
    HEALTH_RESP=$(curl -sf "$CORE_URL/api/v1/healthz" 2>/dev/null || echo "")

    if [ -n "$HEALTH_RESP" ]; then
        # Try to login to check if user exists
        LOGIN_RESP=$(curl -sf -X POST "$CORE_URL/api/v1/auth/login" \
            -H "Content-Type: application/json" \
            -d "{\"email\":\"${ADMIN_EMAIL}\",\"password\":\"${ADMIN_PASSWORD}\"}" 2>/dev/null || echo "")

        if echo "$LOGIN_RESP" | grep -q "token"; then
            warn "Admin user already exists — skipping creation"
            CREATE_ADMIN=false
        fi
    fi

    if [ "$CREATE_ADMIN" = true ]; then
        info "Creating admin user..."
        CREATE_RESP=$(curl -sf -X POST "$CORE_URL/api/v1/auth/setup" \
            -H "Content-Type: application/json" \
            -d "{\"email\":\"${ADMIN_EMAIL}\",\"password\":\"${ADMIN_PASSWORD}\"}" 2>/dev/null || echo "")

        if [ -n "$CREATE_RESP" ]; then
            ok "Admin user created: ${ADMIN_EMAIL}"
        else
            warn "Could not create admin user (may already exist or setup not required)"
            info "Login at $CORE_URL with ${ADMIN_EMAIL}"
        fi
    fi
else
    warn "No admin password set. Create an admin manually via the dashboard."
fi
set -e

# ─── 8. Summary ───────────────────────────────────────────────────────────────
header "Installation Complete"

EXTERNAL_URL="${RENOS_EXTERNAL_URL:-https://localhost}"
HTTP_PORT="${RENOS_HTTP_PORT:-80}"
HTTPS_PORT="${RENOS_HTTPS_PORT:-443}"

echo ""
echo "  Dashboard:  ${EXTERNAL_URL}"
echo "  API:        ${EXTERNAL_URL}/api/v1/healthz"
echo "  Admin:      ${ADMIN_EMAIL}"
echo ""
echo "  Services:"
docker compose -p "$COMPOSE_PROJECT" -f "$COMPOSE_FILE" ps --format "    {{.Name}}: {{.Status}}"
echo ""
echo "  Useful commands:"
echo "    $0 --update              Update RenOS"
echo "    docker compose logs -f   View logs"
echo "    docker compose ps        Service status"
echo "    docker compose down      Stop all services"
echo ""
echo "  Next steps:"
echo "    1. Open ${EXTERNAL_URL} and log in"
echo "    2. Change the admin password"
echo "    3. Install RenAgent on your devices"
echo "    4. Configure DNS for ${RENOS_HOSTNAME:-your-hostname}"
echo ""
