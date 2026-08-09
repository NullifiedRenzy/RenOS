# Development Scripts

## Overview

The `scripts/` directory provides one-command bootstrap and teardown scripts
for local development. They orchestrate the full stack:

```
docker compose up -d  →  migrate  →  core  →  hub  →  agent
```

## `dev-up.sh` — Start Everything

```bash
./scripts/dev-up.sh
```

This script:

1. **Starts infrastructure** — `docker compose up -d` (PostgreSQL, Redis, NATS)
2. **Waits for PostgreSQL** — polls `pg_isready` until the database accepts
   connections
3. **Runs migrations** — `go run ./cmd/renos-core/ migrate` in `core/`
4. **Starts Core** — launches `renos-core` in the background, waits for
   `/api/v1/healthz` to return 200
5. **Starts Hub** — launches `npm run dev` in `hub/` in the background
6. **Starts Agent** — launches `renos-agent run` if an identity file exists
   at `${RENOS_DATA_DIR:-/tmp/renos/agent}/identity.json`

### Enrollment (first time)

The bootstrap script skips the agent if no identity is present. To enroll:

```bash
# Generate a token (replace TOKEN with the admin JWT from login)
TOKEN=$(curl -s -X POST http://localhost:8443/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@renos.local","password":"admin"}' \
  | python3 -c 'import sys,json; print(json.load(sys.stdin)["token"])')

ENROLL_TOKEN=$(curl -s -X POST http://localhost:8443/api/v1/devices/enroll-token \
  -H "Authorization: Bearer $TOKEN" \
  | python3 -c 'import sys,json; print(json.load(sys.stdin)["token"])')

# Enroll the agent (token is a required --token flag)
RENOS_DATA_DIR=/tmp/renos/agent \
RENOS_API_URL=http://localhost:8443 \
./agent/target/release/renos-agent enroll --token "$ENROLL_TOKEN"

# Rerun dev-up.sh to start the agent
./scripts/dev-up.sh
```

## `dev-down.sh` — Stop Everything

```bash
./scripts/dev-down.sh
```

Stops agent, hub, core, and Docker Compose in that order. Docker volumes
are preserved (data survives restarts).

To destroy all data:

```bash
docker compose down -v
```

## Environment

The scripts source the project root `.env` file. Copy the template first:

```bash
cp .env.example .env
```

Key variables used by the scripts:

| Variable | Dev default | Notes |
|---|---|---|
| `RENOS_DATABASE_URL` | `postgres://renos:renos@localhost:5432/renos?sslmode=disable` | |
| `RENOS_CA_CERT_DIR` | `/tmp/renos/ca` | Avoids `/var/lib/renos` permission issues |
| `RENOS_DATA_DIR` | `/tmp/renos/agent` | Agent identity and secrets |
| `RENOS_API_URL` | `http://localhost:8443` | Core base URL for agent and hub |
| `RENOS_LOG_LEVEL` | `debug` | |

## Troubleshooting

### Port conflicts

```bash
# Check what's using the port
ss -tlnp | grep -E '8443|3000|5432|6379|4222'

# Kill a stale process
pkill -f "renos-core"
```

### Stale agent identity

```bash
# Remove agent identity and re-enroll
rm -rf /tmp/renos/agent
# Then generate a new enrollment token and enroll again
```

### Docker volumes

```bash
# Reset all infrastructure data
docker compose down -v
docker compose up -d
# Then rerun dev-up.sh
```
