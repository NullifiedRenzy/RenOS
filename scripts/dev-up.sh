#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

# ─── 1. Infrastructure ──────────────────────────────────────────────────────
echo "==> Starting infrastructure (postgres, redis, nats)..."
docker compose -f "$ROOT_DIR/docker-compose.yml" up -d

echo "==> Waiting for postgres to be healthy..."
until docker compose -f "$ROOT_DIR/docker-compose.yml" exec -T postgres pg_isready -U renos >/dev/null 2>&1; do
  sleep 1
done
echo "     postgres is ready."

# ─── 2. Migrations ──────────────────────────────────────────────────────────
echo "==> Running database migrations..."
(cd "$ROOT_DIR/core" && go run ./cmd/renos-core/ migrate)
echo "     migrations complete."

# ─── 3. Core ─────────────────────────────────────────────────────────────────
echo "==> Starting Core..."
RENOS_CA_CERT_DIR=/tmp/renos/ca \
RENOS_DATABASE_URL=postgres://renos:renos@localhost:5432/renos?sslmode=disable \
RENOS_NATS_URL=nats://localhost:4222 \
RENOS_REDIS_URL=redis://localhost:6379/0 \
RENOS_ADMIN_EMAIL=admin@renos.local \
RENOS_ADMIN_PASSWORD=admin \
RENOS_LOG_LEVEL=debug \
setsid go run ./cmd/renos-core/ > /tmp/renos-core.log 2>&1 &
CORE_PID=$!

echo "==> Waiting for Core health..."
until curl -sf http://localhost:8443/api/v1/healthz >/dev/null 2>&1; do
  sleep 1
done
echo "     Core is healthy (PID $CORE_PID)."

# ─── 4. Hub ──────────────────────────────────────────────────────────────────
echo "==> Starting Hub..."
(cd "$ROOT_DIR/hub" && \
  RENOS_API_URL=http://localhost:8443 \
  NEXT_PUBLIC_RENOS_API_URL=http://localhost:8443 \
  setsid npm run dev > /tmp/renos-hub.log 2>&1 &)
echo "     Hub started on http://localhost:3000."

# ─── 5. Agent ────────────────────────────────────────────────────────────────
AGENT_IDENTITY="${RENOS_DATA_DIR:-/tmp/renos/agent}/identity.json"
if [ -f "$AGENT_IDENTITY" ]; then
  echo "==> Starting Agent..."
  RUST_LOG=info \
  RENOS_DATA_DIR=/tmp/renos/agent \
  RENOS_API_URL=http://localhost:8443 \
  setsid "$ROOT_DIR/agent/target/release/renos-agent" run > /tmp/renos-agent.log 2>&1 &
  echo "     Agent started."
else
  echo "==> Skipping Agent — no identity found at $AGENT_IDENTITY"
  echo "     Enroll first with:  RENOS_DATA_DIR=/tmp/renos/agent RENOS_API_URL=http://localhost:8443 \"$ROOT_DIR/agent/target/release/renos-agent\" enroll <TOKEN>"
fi

echo ""
echo "✓ All services are up."
echo "   Core : http://localhost:8443/api/v1/healthz"
echo "   Hub  : http://localhost:3000"
echo "   Agent: tail -f /tmp/renos-agent.log"
echo ""
echo "Use  dev-down.sh  to stop all services."
