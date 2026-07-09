#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

echo "==> Stopping Agent..."
pkill -f "renos-agent" 2>/dev/null || true

echo "==> Stopping Hub..."
pkill -f "next dev" 2>/dev/null || true

echo "==> Stopping Core..."
pkill -f "renos-core" 2>/dev/null || true

echo "==> Stopping infrastructure..."
docker compose -f "$ROOT_DIR/docker-compose.yml" down

echo "✓ All services stopped."
