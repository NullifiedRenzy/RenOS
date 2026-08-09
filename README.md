# RenOS

**Your devices. One intelligent system.**

RenOS is an AI-powered operating ecosystem that unifies desktops, servers,
Raspberry Pis, NAS devices, Docker containers, and VMs into one platform.

[![CI](https://github.com/NullifiedRenzy/RenOS/actions/workflows/ci.yml/badge.svg)](https://github.com/NullifiedRenzy/RenOS/actions/workflows/ci.yml)
[![Release](https://github.com/NullifiedRenzy/RenOS/actions/workflows/release.yml/badge.svg)](https://github.com/NullifiedRenzy/RenOS/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

---

## Features

- **Unified Dashboard** — Monitor all devices from a single web UI
- **Real-time Updates** — Live health metrics via WebSocket
- **Device Management** — Docker containers, systemd services, file browsing
- **Multi-device Commands** — Run commands across device groups
- **Scheduling** — Cron-based recurring job execution
- **RBAC** — Role-based access control (admin, operator, viewer)
- **Agent-based** — Lightweight Rust daemon on each managed device
- **PWA Support** — Installable as a Progressive Web App
- **Language SDKs** — Official Go, Python, TypeScript, C#, and Rust clients
- **Cluster & Fleet** — Group devices, run fleet-wide commands and jobs
- **Proxmox & Docker** — VM lifecycle and container/compose management
- **Terminal & Files** — Live sessions, file browsing, and transfers
- **Secrets Vault** — Encrypted secrets storage with policy controls
- **Observability** — Prometheus metrics, Grafana dashboards, alerts
- **Marketplace & Plugins** — Installable plugins with lifecycle management
- **Mobile & Notifications** — Push registrations and in-app notifications
- **renctl CLI** — Manage RenOS from the command line

## Architecture

```
                    ┌──────────────────┐
                    │   RenHub (Web)   │
                    │  (Next.js PWA)   │
                    └────────┬─────────┘
                             │ HTTP / WebSocket
                    ┌────────▼─────────┐
                    │   RenCore (Go)   │
                    │  API + Auth + WS │
                    └─┬────────────────┘
                      │
                      │  HTTP (enroll, heartbeat)
                      ▼
                 ┌─────────┐
                 │RenAgent │
                 │ (Rust)  │
                 └─────────┘
```

| Component | Language | Directory | Description |
|---|---|---|---|
| **RenCore** | Go | `core/` | Central server: REST API, auth, WebSocket, job scheduling |
| **RenAgent** | Rust | `agent/` | Lightweight daemon on each managed device |
| **RenHub** | TypeScript | `hub/` | Web UI (Next.js 15, React 19, Tailwind CSS) |
| **RenCtl** | Go | `renctl/` | Command-line client for the RenOS API |

### Infrastructure

| Service | Purpose |
|---|---|
| PostgreSQL 16 | Primary database |
| Redis 7 | Cache, sessions, rate limiting, pub/sub |
| NATS 2.10 | Internal event bus, command queue (JetStream) |
| Nginx | Reverse proxy (production only) |

---

## Quick Start

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) + Docker Compose V2
- Git

### Production (one command)

```bash
git clone https://github.com/NullifiedRenzy/RenOS.git
cd renos
./install.sh
```

The installer will:
1. Check Docker prerequisites
2. Generate secure credentials
3. Create `.env` with your configuration
4. Pull and start all services
5. Wait for health checks
6. Display login credentials

> Linux (systemd-based) installs and one-command installers:
> `sudo ./installer/linux/one-command-install.sh` or
> `sudo ./installer/linux/install.sh`.

After installing, run the production checklist and validation:
```bash
sudo ./installer/validate/renos-validate.sh --post
# Full operator checklist: see PRODUCTION_CHECKLIST.md
```

### Development (manual)

```bash
# Start infrastructure
docker compose up -d

# Run database migrations
cd core && go run ./cmd/renos-core/ migrate

# Start Core (terminal 1)
cd core && go run ./cmd/renos-core/

# Start Hub (terminal 2)
cd hub && npm install && npm run dev

# Open http://localhost:3000
# Login: admin@renos.local / admin
```

Or use the bootstrap script:
```bash
cp .env.example .env
./scripts/dev-up.sh
```

### Stopping

```bash
# Production
docker compose -f docker-compose.prod.yml down

# Development
./scripts/dev-down.sh
```

---

## Configuration

All configuration is via environment variables. See `.env.example` for the full reference.

### Required (production)

| Variable | Description |
|---|---|
| `RENOS_DATABASE_URL` | PostgreSQL connection string |
| `RENOS_NATS_URL` | NATS server URL |
| `RENOS_JWT_SECRET` | JWT signing key (min 32 chars) |
| `RENOS_ENCRYPTION_KEY` | Secrets encryption key (min 32 chars) |
| `RENOS_EXTERNAL_URL` | Public-facing URL |
| `RENOS_ADMIN_EMAIL` | Initial admin email |
| `RENOS_ADMIN_PASSWORD` | Initial admin password |

### Optional

| Variable | Default | Description |
|---|---|---|
| `RENOS_PORT` | `8443` | HTTP API port |
| `RENOS_REDIS_URL` | `redis://localhost:6379/0` | Redis URL |
| `RENOS_LOG_LEVEL` | `info` | Log level: debug, info, warn, error |
| `RENOS_CA_CERT_DIR` | `/var/lib/renos/ca` | CA certificate directory |
| `RENOS_TELEMETRY_ENABLED` | `false` | Opt-in anonymous telemetry |

> The full list is in `core/internal/config/config.go`. Note: the agent's
> heartbeat interval is configured on the agent via `RENOS_HEARTBEAT_INTERVAL`;
> Core's copy is not env-configurable.

---

## Docker

### Development

```bash
docker compose up -d          # Start infrastructure only
docker compose logs -f        # View logs
docker compose ps             # Check health
docker compose down           # Stop
docker compose down -v        # Stop and delete all data
```

### Production

```bash
docker compose -f docker-compose.prod.yml up -d
```

This runs all services: PostgreSQL, Redis, NATS, Core, Hub, and Nginx.

### Docker Images

Pre-built images are available on GitHub Container Registry:

```bash
docker pull ghcr.io/NullifiedRenzy/renos-core:latest
docker pull ghcr.io/NullifiedRenzy/renos-agent:latest
docker pull ghcr.io/NullifiedRenzy/renos-hub:latest
```

Build locally:
```bash
docker build -t renos-core -f core/Dockerfile core
docker build -t renos-agent -f agent/Dockerfile agent
docker build -t renos-hub -f hub/Dockerfile hub
```

---

## API

The API lives under `/api/v1` on port `8443` (development) or behind the
reverse proxy (production). Health check: `http://localhost:8443/api/v1/healthz`.

### Key Endpoints

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/v1/healthz` | Health check |
| `GET` | `/api/v1/readyz` | Readiness check |
| `GET` | `/api/v1/version` | Version info |
| `POST` | `/api/v1/auth/login` | Authenticate |
| `GET` | `/api/v1/devices` | List devices |
| `POST` | `/api/v1/devices/{id}/commands` | Run command |
| `GET` | `/api/v1/ws` | WebSocket (real-time events) |

### Authentication

```bash
# Login
TOKEN=$(curl -s -X POST http://localhost:8443/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@renos.local","password":"admin"}' | jq -r '.token')

# Use token
curl -H "Authorization: Bearer $TOKEN" http://localhost:8443/api/v1/devices
```

See [docs/api/](docs/api/) for the full API reference.

---

## Development

### Commands

```bash
make help           # Show all available targets
make test           # Run all tests
make lint           # Run all linters
make build          # Build all components
make check          # Full pre-release check (lint + test + build)
```

### Project Structure

```
RenOS/
├── core/           # Go backend (RenCore)
│   ├── cmd/        # Entry point
│   ├── internal/   # Private packages
│   └── migrations/ # Database migrations
├── agent/          # Rust daemon (RenAgent)
│   └── src/        # Source code
├── hub/            # Next.js web UI (RenHub)
│   ├── app/        # Pages and layouts
│   ├── components/ # React components
│   ├── hooks/      # React hooks
│   └── lib/        # Utilities and types
├── renctl/         # Command-line client
├── sdk/            # Language SDKs (Go, Python, TypeScript, C#, Rust)
├── installer/      # Installers (Linux, macOS, Windows, Kubernetes, upgrade)
├── release/        # Release tooling, audit, SBOM, test reports
├── examples/       # Usage examples and workflows
├── proto/          # Protocol Buffer definitions
├── docs/           # Documentation
├── scripts/        # Dev and ops scripts
└── deploy/         # Deployment configs (nginx, prometheus, grafana)
```

### Testing

```bash
# Core (Go)
cd core && go test -v ./...

# Agent (Rust)
cd agent && cargo test

# Hub (Vitest)
cd hub && npm test
```

### Linting

```bash
# Core
cd core && golangci-lint run

# Agent
cd agent && cargo clippy -- -D warnings

# Hub
cd hub && npm run lint && npm run typecheck
```

---

## Backup & Restore

Supported production backups (`installer/backup/`):

```bash
# Full backup
sudo ./installer/backup/renos-backup.sh --full

# List, verify, prune
sudo ./installer/backup/renos-backup.sh --list
sudo ./installer/backup/renos-backup.sh --verify <backup-id>
sudo ./installer/backup/renos-backup.sh --prune

# Restore (dry-run first)
sudo ./installer/backup/renos-restore.sh --dry-run /var/lib/renos/backups/renos-backup-<ID>
```

Legacy dev scripts still exist for quick DB dumps:
```bash
./scripts/backup.sh                    # dump to ./backups/
./scripts/restore.sh backups/renos-backup-20260723-120000.sql.gz
```

---

## Versioning

RenOS follows [Semantic Versioning](https://semver.org/). Check the current version:

```bash
# CLI
cd core && go run ./cmd/renos-core/ version

# API
curl http://localhost:8443/api/v1/version

# UI
Settings page in the dashboard
```

See [CHANGELOG.md](CHANGELOG.md) for the version history.

---

## Deployment

See [RELEASE.md](RELEASE.md) for the release process and [docs/operations/](docs/operations/) for deployment guides.

### Production Topology

```
Browser → Nginx (:80)
            ├── /api/v1/*  → Core (:8443)    [HTTP + WebSocket]
            ├── /_next/*   → Hub (:3000)     [Static assets]
            └── /*         → Hub (:3000)     [Pages]
```

---

## Security

- All secrets are validated on startup
- JWT authentication with configurable expiry
- Rate limiting on login and API endpoints
- Role-based access control (admin, operator, viewer)
- WebSocket origin checks
- Security headers (X-Frame-Options, X-Content-Type-Options, etc.)

See [docs/standards/SECURITY.md](docs/standards/SECURITY.md) for security policies.

---

## License

MIT — see [LICENSE](LICENSE).
