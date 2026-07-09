# Changelog

## v0.1.0-nova (2026-07-08)

### Milestone M01 — Repository Setup
- Monorepo structure with `core/`, `agent/`, `hub/`, `proto/`, `docs/`
- CI/CD with GitHub Actions (lint, test, build, release)
- Proto definitions for agent v1 gRPC API
- Makefile with unified build/test/lint targets

### Milestone M02 — Docker Compose
- PostgreSQL 16, Redis 7, NATS 2.10 dev containers
- Healthchecks on all services, persistent volumes
- `renos-dev` bridge network, `.env.example` config

### Milestone M03 — Go RenCore Skeleton
- HTTP server with chi router, structured logging (slog)
- Configuration from env vars with sensible defaults
- Health (`/healthz`) and readiness (`/readyz`) endpoints
- pgx connection pool, NATS client, optional Redis
- Graceful shutdown with SIGINT/SIGTERM
- 11 unit tests covering health/ready/DB/NATS checks

### Milestone M04 — PostgreSQL Schema + Migrations
- 10 migration files (5 up + 5 down) for users, devices, sessions, audits, settings
- golang-migrate with `iofs` + `pgx5` driver
- CLI `migrate` / `migrate down` commands in main.go

### Milestone M05 — Rust RenAgent Skeleton
- CLI with `run`, `enroll`, `version` subcommands
- Config from env + file, platform data dirs (Linux/macOS/Windows)
- Ed25519 keypair generation and storage
- Tonic gRPC client skeleton with graceful shutdown
- 9 test cases across config, identity, platform modules

### Milestone M06 — Device Enrollment
- Ed25519 CA generation/storage, X.509 device cert signing
- Enrollment token API with configurable TTL
- HTTP enrollment flow with bcrypt-secured device secret
- Agent keypair generation (`ed25519-dalek`), identity/cert storage
- Admin user seeding, JWT authentication
- Audit logging for enroll/connect events

### Milestone M07 — Heartbeat Reporting
- `POST /api/v1/devices/heartbeat` endpoint with device_secret auth
- Health data collection (CPU, RAM, disk, uptime, hostname, IP)
- `sysinfo`-based Rust health collector
- Offline checker goroutine marks devices offline after timeout
- Status transition audit entries (offline→online)

### Milestone M08 — RenHub Dashboard
- Next.js 15 web UI with TypeScript, Tailwind CSS, Zustand
- Login page with JWT auth flow and localStorage persistence
- Dashboard shell with collapsible sidebar and top bar
- Live device list with inline CPU/RAM progress bars
- Device detail page with health score ring, metrics, network info
- Dark mode toggle (light/dark/system), persisted preference
- React Query for data fetching with loading/error/empty states
- Graceful API error handling with 401 auto-redirect

### Milestone M09 — WebSocket Live Updates
- `GET /api/v1/ws` WebSocket endpoint with JWT query-param auth
- Event hub/broadcaster in Go using `gorilla/websocket`
- Real-time `device.health`, `device.online`, `device.offline` broadcasts
- React hook with exponential backoff reconnection (1s–30s)
- React Query cache invalidation on WebSocket events

### Infrastructure
- Dockerfiles for Core (multi-stage Go), Agent (multi-stage Rust), Hub (Next.js standalone)
- GitHub Actions CI: lint → test → build for all components
- GitHub Actions Release: build & push Docker images to GHCR

---

## Upcoming

- **M11**: Buffer / bugfix / polish
- **v0.2**: Command execution, file browser, automations
- **v0.3**: AI chat, plugins, advanced dashboards
