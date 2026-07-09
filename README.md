# RenOS

Your devices. One intelligent system.

RenOS is an AI-powered operating ecosystem that unifies desktops, servers,
Raspberry Pis, NAS devices, Docker containers, and VMs into one platform.

## Architecture

```
                    ┌──────────────────┐
                    │   RenHub (Web)   │
                    │  (Next.js PWA)   │
                    └────────┬─────────┘
                             │ HTTP / WS
                    ┌────────▼─────────┐
                    │   RenCore (Go)   │
                    │  API ─ Auth ─ WS │
                    └─┬────────────────┘
                      │
                      │  HTTP (enroll, heartbeat)
                      ▼
                 ┌─────────┐
                 │RenAgent │
                 │ (Rust)  │
                 └─────────┘
```

## Components

| Component | Language | Directory | Description |
|---|---|---|---|
| **RenCore** | Go | `core/` | Central server: REST API, auth, WebSocket, device management |
| **RenAgent** | Rust | `agent/` | Lightweight daemon on each managed device |
| **RenHub** | TypeScript | `hub/` | Web UI (Next.js 15, Tailwind CSS) |

## Quick Start

### Prerequisites

- Go 1.23+
- Rust 1.80+
- Node.js 22+
- Docker + Docker Compose (for PostgreSQL, Redis, NATS)

### Quickest Start (bootstrap script)

```bash
cp .env.example .env
./scripts/dev-up.sh
```

See `docs/development/dev-scripts.md` for details.

### Manual Start (step by step)

```bash
# 1. Start dependencies (PostgreSQL, Redis, NATS)
docker compose up -d

# 2. Run database migrations
cd core && go run ./cmd/renos-core/ migrate

# 3. Start Core
cd core && go run ./cmd/renos-core/
# → API at http://localhost:8443
# → Health: http://localhost:8443/api/v1/healthz

# 4. Start Hub (in another terminal)
cd hub && npm install && npm run dev
# → UI at http://localhost:3000

# 5. Enroll an Agent
cd agent && cargo run --release -- enroll --token <TOKEN>
cd agent && cargo run --release -- run
# See docs/development/getting-started.md for the full enrollment flow

# 6. Open Dashboard
open http://localhost:3000
# Login: admin@renos.local / admin
```

### Stopping

```bash
./scripts/dev-down.sh     # graceful shutdown
```

## Configuration

Core configuration is via environment variables (see `.env.example` for all options):

| Variable | Default | Description |
|---|---|---|
| `RENOS_PORT` | `8443` | HTTP API port |
| `RENOS_GRPC_PORT` | `8444` | gRPC port (future) |
| `RENOS_DATABASE_URL` | `postgres://renos:renos@localhost:5432/renos?sslmode=disable` | PostgreSQL connection |
| `RENOS_NATS_URL` | `nats://localhost:4222` | NATS connection |
| `RENOS_JWT_SECRET` | `dev-secret-do-not-use-in-production` | JWT signing key |
| `RENOS_ADMIN_EMAIL` | `admin@renos.local` | Initial admin email |
| `RENOS_ADMIN_PASSWORD` | `admin` | Initial admin password |

Hub configuration:

| Variable | Default | Description |
|---|---|---|
| `RENOS_API_URL` | `http://localhost:8443` | Core API base URL (for Next.js rewrite proxy) |

## Development

```bash
# Run all tests
make test

# Run all linters
make lint

# Build all components
make build

# Full pre-release check
make check
```

## WebSocket Notes (Development)

During development, the Hub connects to the Core WebSocket endpoint directly at
`ws://localhost:8443/api/v1/ws?token=JWT` (bypassing the Next.js proxy). For
production, configure a reverse proxy (nginx/Caddy) to serve both Hub and Core
on the same origin. See `RELEASE.md` for details.

## Project Status

**v0.1.0-nova** — Foundation release. Core + single-agent enrollment, heartbeat
reporting, live dashboard with real-time updates.

## License

MIT — see [LICENSE](LICENSE).
