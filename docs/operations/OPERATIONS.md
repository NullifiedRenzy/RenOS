# Operations Guide

## Backup & Restore

### Creating Backups

```bash
# Manual backup
./scripts/backup.sh

# Backup to specific directory
./scripts/backup.sh /mnt/backups

# Automated daily backup (add to crontab)
0 2 * * * /opt/renos/scripts/backup.sh /mnt/renos-backups
```

Backups are stored as compressed SQL dumps in `./backups/`. The last 30 backups are kept.

### Restoring Backups

```bash
# Interactive restore (prompts for confirmation)
./scripts/restore.sh backups/renos-backup-20260723-120000.sql.gz

# List available backups
ls -lh backups/
```

**Warning:** Restore drops and recreates the database. All current data will be lost.

### What's Backed Up

- All database tables (users, devices, commands, schedules, policies, etc.)
- Audit logs
- System logs
- AI conversations

### What's NOT Backed Up

- CA certificates (`data/ca/`)
- Docker volumes (Redis, NATS data)
- Agent identities on managed devices

## Monitoring

### Health Checks

```bash
# Quick health check
curl -sf http://localhost/api/v1/healthz

# Detailed readiness check (includes DB, NATS, Redis)
curl -s http://localhost/api/v1/readyz | jq
```

### Docker Health Status

```bash
docker compose -f docker-compose.prod.yml ps
```

All services include health checks that report their status automatically.

### Logs

```bash
# All services
docker compose -f docker-compose.prod.yml logs -f

# Specific service
docker compose -f docker-compose.prod.yml logs -f core

# Last 100 lines
docker compose -f docker-compose.prod.yml logs --tail=100 core
```

### System Logs (via API)

```bash
# List recent logs
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost/api/v1/logs?limit=50

# Filter by level
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost/api/v1/logs?level=error&limit=20"
```

## Maintenance

### Database Migrations

```bash
# Run pending migrations
docker compose -f docker-compose.prod.yml exec core renos-core migrate

# Rollback last migration
docker compose -f docker-compose.prod.yml exec core renos-core migrate down 1

# Rollback 5 migrations
docker compose -f docker-compose.prod.yml exec core renos-core migrate down 5
```

### Updating RenOS

```bash
# Pull latest images
docker compose -f docker-compose.prod.yml pull

# Restart with new images
docker compose -f docker-compose.prod.yml up -d

# Verify health
docker compose -f docker-compose.prod.yml ps
curl http://localhost/api/v1/healthz
```

### Resetting Everything

**WARNING: This destroys all data.**

```bash
docker compose -f docker-compose.prod.yml down -v
./install.sh
```

## Scaling

### Current Architecture

RenOS is designed for single-server deployments. The Core service handles:
- REST API (stateless, horizontally scalable with shared DB)
- WebSocket connections (in-memory hub, single instance)
- Job processing (in-memory worker pool, single instance)

### Database Connection Pool

Configure via `RENOS_POOL_SIZE` (default: 5). Increase for higher concurrency.

### Worker Pool

Configure via:
- `RENOS_POOL_SIZE` — Worker goroutines
- `RENOS_MAX_CONCURRENT_JOBS` — Parallel jobs
- `RENOS_MAX_CONCURRENT_DEVICE_ACTIONS` — Parallel device actions
