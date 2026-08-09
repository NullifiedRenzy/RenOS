#!/usr/bin/env bash
# =============================================================================
# RenOS CI Prep Helper
# =============================================================================
# Prepares a CI environment for fresh-install-test.sh / one-command-install.sh:
#   1. (--install-docker) Installs Docker and starts dockerd in systemd-less
#      containers (fedora/debian CI containers have no systemd).
#   2. Builds renos/core:latest, renos/hub:latest, renos/agent:latest from this
#      repo so the installer's compose stack has local images to run.
#
# Usage:
#   bash ci-prep.sh                      # build images (docker already present)
#   bash ci-prep.sh --install-docker     # install+start docker, then build
#   bash ci-prep.sh --help
#
# Exit codes:
#   0   Success
#   1   Failure
# =============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

INSTALL_DOCKER=false

for arg in "$@"; do
    case "$arg" in
        --install-docker) INSTALL_DOCKER=true ;;
        --help|-h)
            echo "Usage: ci-prep.sh [--install-docker]"; exit 0 ;;
        *) echo "Unknown option: $arg" >&2; exit 2 ;;
    esac
done

echo "ℹ  Repo root: $REPO_ROOT"

detect_distro() {
    if [ -f /etc/os-release ]; then
        # shellcheck disable=SC1091
        . /etc/os-release
        echo "${ID:-unknown} ${VERSION_ID:-unknown}"
    fi
}

install_docker() {
    echo "ℹ  Installing Docker (container mode)"
    local distro
    distro="$(detect_distro)"
    case "$distro" in
        ubuntu*|debian*)
            apt-get update -qq
            DEBIAN_FRONTEND=noninteractive apt-get install -y -qq \
                ca-certificates curl gnupg lsb-release >/dev/null
            install -m 0755 -d /etc/apt/keyrings
            curl -fsSL "https://download.docker.com/linux/${distro%% *}/gpg" | \
                gpg --dearmor -o /etc/apt/keyrings/docker.gpg
            echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/${distro%% *} $(lsb_release -cs) stable" \
                > /etc/apt/sources.list.d/docker.list
            apt-get update -qq
            DEBIAN_FRONTEND=noninteractive apt-get install -y -qq \
                docker-ce docker-ce-cli containerd.io docker-buildx-plugin \
                docker-compose-plugin >/dev/null
            ;;
        fedora*)
            dnf install -y -q dnf-plugins-core
            dnf config-manager --add-repo \
                https://download.docker.com/linux/fedora/docker-ce.repo
            dnf install -y -q docker-ce docker-ce-cli containerd.io \
                docker-buildx-plugin docker-compose-plugin
            ;;
        *)
            echo "✗  Unsupported distro for --install-docker: $distro" >&2
            exit 1
            ;;
    esac
}

start_dockerd() {
    if docker info &>/dev/null 2>&1; then
        echo "✓  Docker daemon already running"
        return 0
    fi
    echo "ℹ  Starting dockerd"
    mkdir -p /var/log/renos
    nohup dockerd > /var/log/renos/dockerd.log 2>&1 &
    for _ in $(seq 1 60); do
        docker info &>/dev/null 2>&1 && { echo "✓  Docker daemon started"; return 0; }
        sleep 1
    done
    echo "✗  dockerd failed to start (see /var/log/renos/dockerd.log)" >&2
    return 1
}

build_images() {
    local -a images=(
        "core $REPO_ROOT/core renos/core:latest"
        "hub $REPO_ROOT/hub renos/hub:latest"
        "agent $REPO_ROOT . renos/agent:latest"
    )
    local -a built=()
    local name ctx tag
    for entry in "${images[@]}"; do
        read -r name ctx tag <<< "$entry"
        echo "▶  Building $tag (context: $ctx)"
        if [ -f "$ctx/Dockerfile" ]; then
            docker build --cache-from=type=gha --cache-to=type=gha,mode=max \
                -t "$tag" "$ctx"
        else
            docker build --cache-from=type=gha --cache-to=type=gha,mode=max \
                -f "$ctx/Dockerfile" -t "$tag" "$REPO_ROOT"
        fi
        built+=("$tag")
    done
    echo "✓  Images ready: ${built[*]}"
}

if [ "$INSTALL_DOCKER" = true ]; then
    install_docker
    start_dockerd
else
    docker info &>/dev/null 2>&1 || {
        echo "✗  Docker is not running. Run 'ci-prep.sh --install-docker' in containers." >&2
        exit 1
    }
fi

build_images
