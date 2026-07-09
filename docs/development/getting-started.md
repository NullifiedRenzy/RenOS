# Getting Started with RenOS Development

## Quickest Start

For a one-command bootstrap that starts everything (infrastructure, Core,
Hub, and optionally the Agent), use the dev scripts:

```bash
cp .env.example .env
./scripts/dev-up.sh
```

See [dev-scripts.md](dev-scripts.md) for details.

For a manual step-by-step walkthrough, continue below.

---

## Prerequisites

| Tool | Version | Purpose |
|---|---|---|
| Go | 1.23+ | Core backend |
| Rust | 1.80+ | Agent |
| Node.js | 22+ | Web UI |
| Docker | 24+ | Dependencies |
| Docker Compose | 2.24+ | Service orchestration |

Optional:
- `protoc` — for proto code generation (installed automatically by tonic)
- `golangci-lint` — for Go linting (`make core-lint`)
- `buf` — for proto linting (`make proto-lint`)

## Clone

```bash
git clone https://github.com/renos-project/renos.git
cd renos
```

## Start Dependencies

```bash
docker compose up -d
```

This starts three services on the `renos-dev` network:

| Service | Port | Credentials | Purpose |
|---|---|---|---|
| **PostgreSQL 16** | 5432 | `renos` / `renos` | Primary database |
| **Redis 7** | 6379 | — | Cache, sessions, pub/sub |
| **NATS 2.10** | 4222 | — | Event bus, command queue |

### Verify Everything is Healthy

```bash
# Check all service statuses
docker compose ps

# Expected output:
# NAME             IMAGE                 STATUS          PORTS
# renos-postgres   postgres:16-alpine    Up (healthy)    0.0.0.0:5432->5432/tcp
# renos-redis      redis:7-alpine        Up (healthy)    0.0.0.0:6379->6379/tcp
# renos-nats       nats:2.10-alpine      Up (healthy)    0.0.0.0:4222->4222/tcp

# View live logs from all services
docker compose logs -f

# View logs for a specific service
docker compose logs postgres
docker compose logs nats

# Test PostgreSQL connectivity
psql postgres://renos:renos@localhost:5432/renos -c "SELECT version();"

# Test NATS connectivity
nats pub test "hello" && nats sub --count=1 test
```

## Run Core

```bash
cd core
go run ./cmd/renos-core/
```

Core starts listening on http://localhost:8443.

Verify the health endpoint:
```bash
curl http://localhost:8443/api/v1/healthz
# → {"status":"ok","version":"0.1.0"}

curl http://localhost:8443/api/v1/readyz
# → {"status":"ok","version":"0.1.0","checks":{"database":{...},"nats":{...}}}
```

## Run Hub

```bash
cd hub
npm install
npm run dev
```

Open http://localhost:3000 in your browser.

## Run Agent

```bash
cd agent
cargo run --
```

## Using Make

All common tasks are automated via the Makefile:

```bash
make help           # List all commands

# Docker
make up             # Start dependencies (docker compose up -d)
make down           # Stop dependencies
make logs           # View live logs
make ps             # Show container status
make reset          # STOP AND DELETE ALL DATA (irreversible)
make status         # Show health of all services

# Build and test
make lint           # Run all linters
make test           # Run all tests
make build          # Build all components
make check          # Full pre-commit check

# Development workflow
make dev            # Start dependencies and show next steps
```

## Project Structure

```
renos/
├── core/            # RenCore (Go backend)
├── agent/           # RenAgent (Rust daemon)
├── hub/             # RenHub (Next.js web UI)
├── proto/           # Shared protobuf definitions
├── docs/            # Documentation
├── docker-compose.yml
└── Makefile
```

## Coding Standards

See [Chapter 14 of the Engineering Specification](../engineering-specification.txt)
for full coding standards.

Key points:
- Go: `gofmt` + `golangci-lint`
- Rust: `rustfmt` + `clippy -- -D warnings`
- TypeScript: `prettier` + `eslint`
- Commits: conventional commits (`type(scope): description`)

## Docker Troubleshooting

### Port Conflicts

If port 5432, 6379, or 4222 is already in use:

1. Create a `.env` file from `.env.example`
2. The default ports cannot be overridden via env vars for the Docker
   services themselves. To change them, edit `docker-compose.yml`:
   ```yaml
   ports:
     - "5433:5432"   # Map host 5433 → container 5432
   ```

### Container Won't Start

```bash
# Check logs
docker compose logs postgres

# Common fixes:
# - Port conflict: change host port mapping
# - Volume corruption: make reset (destroys all data)
# - Out of disk: docker system prune
```

### Resetting Everything

```bash
# Stop and remove all data (irreversible)
make reset

# Start fresh
make up
```

### Docker Compose Reference

```bash
docker compose up -d          # Start in background
docker compose down           # Stop but preserve volumes
docker compose down -v        # Stop and delete volumes (all data lost)
docker compose restart        # Restart without data loss
docker compose logs -f        # Follow logs
docker compose ps             # Show container status
docker compose exec postgres psql -U renos -d renos  # Run SQL
```
