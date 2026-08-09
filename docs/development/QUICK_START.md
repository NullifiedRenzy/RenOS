# Quick Start Guide

**This is the canonical quick start.** `getting-started.md` covers the
step-by-step development workflow; `dev-scripts.md` covers the bootstrap
scripts.

This guide walks you through deploying RenOS from zero to a working system.

## Option A: Production (Docker)

### 1. Clone and install

```bash
git clone https://github.com/renos-project/renos.git
cd renos
chmod +x install.sh
./install.sh
```

The installer will:
- Check that Docker is installed and running
- Generate secure JWT and encryption keys
- Prompt for admin credentials and external URL
- Create `.env` with all configuration
- Pull Docker images (or build locally)
- Start all services (PostgreSQL, Redis, NATS, Core, Hub, Nginx)
- Wait for health checks to pass

> Linux systemd install: `sudo ./installer/linux/install.sh`.
> One-command install: `sudo ./installer/linux/one-command-install.sh`.
> After install, run `sudo ./installer/validate/renos-validate.sh --post` and
> work through [PRODUCTION_CHECKLIST.md](../../PRODUCTION_CHECKLIST.md).

### 2. Access the dashboard

Open `http://localhost` in your browser and log in with the credentials you set during installation.

### 3. Install an agent

On each device you want to manage, run the agent binary and enroll it with an
enrollment token (the prebuilt binary lives at `agent/target/release/renos-agent`):

```bash
# Download the agent binary (from releases or build locally)
./agent/target/release/renos-agent enroll --token <ENROLLMENT_TOKEN>
./agent/target/release/renos-agent run
```

`run` is the default subcommand, so `./agent/target/release/renos-agent` is
equivalent after enrollment.

To get an enrollment token (admin only; token TTL is fixed at 15 minutes):
```bash
curl -s -X POST http://localhost/api/v1/devices/enroll-token \
  -H "Authorization: Bearer <ADMIN_TOKEN>"
```

## Option B: Development

### 1. Clone and setup

```bash
git clone https://github.com/renos-project/renos.git
cd renos
cp .env.example .env
```

### 2. Start infrastructure

```bash
docker compose up -d
```

This starts PostgreSQL, Redis, and NATS.

### 3. Run database migrations

```bash
cd core && go run ./cmd/renos-core/ migrate
```

### 4. Start Core

```bash
cd core && go run ./cmd/renos-core/
# API available at http://localhost:8443
```

### 5. Start Hub

```bash
cd hub && npm install && npm run dev
# UI available at http://localhost:3000
```

### 6. Login

Open `http://localhost:3000` and log in:
- **Email:** admin@renos.local
- **Password:** admin

Change these credentials immediately in a real environment.

## Useful Commands

```bash
# Check service health
curl http://localhost:8443/api/v1/healthz

# List devices
curl -H "Authorization: Bearer $TOKEN" http://localhost:8443/api/v1/devices

# View logs
docker compose -f docker-compose.prod.yml logs -f core

# Production backup (see PRODUCTION_CHECKLIST.md for the full set)
sudo ./installer/backup/renos-backup.sh --full

# Stop everything
docker compose -f docker-compose.prod.yml down
```

## Next Steps

- [Production Checklist](../../PRODUCTION_CHECKLIST.md) — operator checklist
- [Development Workflow](./getting-started.md) — step-by-step dev setup
- [Dev Scripts](./dev-scripts.md) — `dev-up.sh` / `dev-down.sh`
- [Configuration Reference](../../.env.example) — All environment variables
- [Architecture Overview](../architecture/SYSTEM_ARCHITECTURE.md) — How it all fits together
- [API Usage](../api/USAGE.md) — Full API reference
- [Security](../standards/SECURITY.md) — Security best practices
