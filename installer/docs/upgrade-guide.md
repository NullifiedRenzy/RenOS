# RenOS Upgrade Guide

## Table of Contents

- [Pre-Upgrade Checklist](#pre-upgrade-checklist)
- [Checking Current Version](#checking-current-version)
- [Using the Upgrade Manager](#using-the-upgrade-manager)
- [Upgrade Paths](#upgrade-paths)
- [Step-by-Step Upgrade for Docker](#step-by-step-upgrade-for-docker)
- [Step-by-Step Upgrade for Kubernetes (Helm)](#step-by-step-upgrade-for-kubernetes-helm)
- [Step-by-Step Upgrade for Native Install](#step-by-step-upgrade-for-native-install)
- [Post-Upgrade Verification](#post-upgrade-verification)
- [Rollback Procedures](#rollback-procedures)
- [Database Migration Notes](#database-migration-notes)
- [Breaking Changes Between Versions](#breaking-changes-between-versions)

---

## Pre-Upgrade Checklist

Before performing any upgrade, complete the following checklist to ensure a safe and successful upgrade:

### Required Actions

- [ ] **Read release notes** for the target version (published on GitHub Releases)
- [ ] **Review breaking changes** section below for version-specific migration steps
- [ ] **Back up all data** — databases, Redis, configuration, and TLS certificates
- [ ] **Verify current version** is on a supported upgrade path
- [ ] **Check disk space** — ensure 2x the current data size is available
- [ ] **Test in staging** environment before production (if possible)
- [ ] **Schedule maintenance window** — some upgrades may cause brief downtime
- [ ] **Notify users** of scheduled maintenance
- [ ] **Verify health** of all components before starting

### Pre-Upgrade Commands

```bash
# 1. Create a full backup
sudo renos-backup.sh --full --encrypt --remote

# 2. Check current version
cat /etc/renos/renos.env | grep RENOS_INSTALL_VERSION

# 3. Check disk space
df -h /var/lib/renos /var/lib/docker

# 4. Verify current health
docker compose -f /etc/renos/docker-compose.yml ps
curl -k https://localhost/api/v1/healthz

# 5. Record current state for rollback
renos-state-snapshot.sh  # Creates a state snapshot at /var/lib/renos/upgrade-state.json
```

### Version Compatibility Check

The upgrade manager includes a built-in version compatibility checker. Before attempting an upgrade:

```bash
# Using the upgrade manager
renos-upgrade check 1.0.0 2.0.0

# Expected output:
# ✓ Version 2.0.0 meets minimum requirements for all components
# ⚠ Warning: Major version change detected
# ⚠ Database migrations required
# ⚠ Configuration file format may have changed
```

---

## Checking Current Version

### Native Installation

```bash
# Method 1: Environment file
grep RENOS_INSTALL_VERSION /etc/renos/renos.env

# Method 2: Upgrade state file
cat /var/lib/renos/upgrade-state.json | grep current_version

# Method 3: API endpoint
curl -sk https://localhost/api/v1/healthz | jq .version
```

### Docker Installation

```bash
# Check image tags
docker images --format "{{.Repository}}:{{.Tag}}" | grep renos

# Check running container versions
docker inspect renos-core --format '{{.Config.Image}}'
docker inspect renos-hub --format '{{.Config.Image}}'

# Via API (if accessible)
curl -sk https://localhost/api/v1/healthz | jq .
```

### Kubernetes Installation

```bash
# Check Helm release version
helm list -n renos

# Check image tags in running pods
kubectl get pods -n renos -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.containers[*].image}{"\n"}{end}'

# Check Helm chart values
helm get values renos -n renos
```

### Upgrade State File

The upgrade state is tracked in `/var/lib/renos/upgrade-state.json`. This JSON file contains:

```json
{
  "current_version": "1.0.0",
  "history": [
    {
      "id": "upg-1700000000123456789",
      "component": "all",
      "from_version": "0.9.0",
      "to_version": "1.0.0",
      "status": "completed",
      "backup_id": "renos-backup-20250101-020000",
      "timestamp": "2025-01-01T02:15:00Z"
    }
  ],
  "last_backup_id": "renos-backup-20250101-020000",
  "component_versions": {
    "core": "1.0.0",
    "hub": "1.0.0",
    "agent": "1.0.0",
    "database": "1.0.0",
    "plugins": "1.0.0"
  }
}
```

---

## Using the Upgrade Manager

The RenOS upgrade manager CLI (`renos-upgrade`) is a Go-based tool that handles version checking, compatibility validation, backup creation, and upgrade execution.

### Installation

The upgrade manager is included with the RenOS installation at `/usr/local/bin/renos-upgrade`. Verify it is installed:

```bash
renos-upgrade --version
```

### Commands

| Command | Description |
|---------|-------------|
| `renos-upgrade check <from> <to>` | Check version compatibility |
| `renos-upgrade plan <from> <to>` | Generate upgrade plan with estimated duration |
| `renos-upgrade run <to>` | Execute upgrade to target version |
| `renos-upgrade status` | Show current upgrade state |
| `renos-upgrade history` | Show upgrade history |
| `renos-upgrade rollback <upgrade-id>` | Rollback a specific upgrade |
| `renos-upgrade list` | List available versions |

### Checking Compatibility

```bash
# Check if upgrade path is valid
renos-upgrade check 1.0.0 2.0.0

# Get detailed compatibility report
renos-upgrade plan 1.0.0 2.0.0
```

### Planning an Upgrade

```bash
# Generate upgrade plan
renos-upgrade plan 1.0.0 2.0.0

# Output:
# ═══ Upgrade Plan: 1.0.0 → 2.0.0 ═══
# 
# Version:  Major upgrade
# 
# Components to upgrade:
#   - core:       1.0.0 → 2.0.0
#   - hub:        1.0.0 → 2.0.0
#   - agent:      1.0.0 → 2.0.0
#   - database:   1.0.0 → 2.0.0  (migration required)
#   - plugins:    1.0.0 → 2.0.0
# 
# Breaking changes:
#   - API v1 endpoints deprecated, use v2
#   - Database schema migration required
#   - New configuration fields mandatory: RENOS_FEATURE_X
# 
# Estimated downtime: 5-10 minutes
# Estimated duration: 15-30 minutes
# 
# Recommended backup before proceeding.
```

### Executing an Upgrade

```bash
# Run upgrade (creates automatic backup before upgrade)
renos-upgrade run 2.0.0

# Run upgrade with pre-existing backup
renos-upgrade run 2.0.0 --backup-id renos-backup-20250101-020000

# Force upgrade (skip compatibility checks)
renos-upgrade run 2.0.0 --force
```

### Checking Upgrade Status

```bash
renos-upgrade status

# Output:
# Current version: 2.0.0
# Last upgrade:    2025-01-01 02:15 UTC (successful)
# Components:      core ✓ hub ✓ agent ✓ database ✓ plugins ✓
```

---

## Upgrade Paths

### Supported Upgrade Paths

| From | To | Direct | Notes |
|------|----|--------|-------|
| 0.9.x | 1.0.0 | Yes | Initial GA upgrade |
| 1.0.x | 1.1.x | Yes | Patch/minor within major |
| 1.0.x | 1.2.x | Yes | Minor only (same major) |
| 1.0.x | 2.0.x | No | Requires 1.2.x first |
| 1.1.x | 1.2.x | Yes | |
| 1.2.x | 2.0.x | Yes | Supported migration path |
| 2.0.x | 2.1.x | Yes | |
| 2.x.x | 3.0.x | No | Requires 2.2.x first |

### Upgrade Path Rules

1. **Patch versions** (x.x.N): Can upgrade directly from any earlier patch
2. **Minor versions** (x.N.x): Can upgrade directly within same major version
3. **Major versions** (N.x.x): Must not skip more than one major version
4. **Downgrades**: Not supported. Rollback via backup restoration only.
5. **LTS versions**: Supported for 2 years after release

### Version Compatibility Matrix

| Component | Minimum Version | Compatible With |
|-----------|----------------|-----------------|
| core | 1.0.0 | hub 1.2.0+, agent 1.1.0+ |
| hub | 1.2.0 | core 1.0.0+ |
| agent | 1.1.0 | core 1.0.0+ |
| plugins | 1.0.0 | core 1.0.0+ |
| database | 0.0.0 | All versions (migrations available) |

### Version Architecture

```
v0.9.x (beta) ──→ v1.0.0 (GA) ──→ v1.1.x ──→ v1.2.x ──→ v2.0.x ──→ ...
                     │                           │
                     └── LTS 1.0.x (2yr support)  └── LTS 1.2.x (2yr support)
```

---

## Step-by-Step Upgrade for Docker

### Upgrading Docker Images

**Step 1: Pull latest images**

```bash
docker compose pull
```

This pulls new versions of all images defined in the docker-compose.yml file.

**Step 2: Back up current state**

```bash
sudo renos-backup.sh --full --encrypt --remote
```

**Step 3: Perform the upgrade**

```bash
# Apply database migrations
docker compose run --rm renos-core renos-core migrate

# Recreate containers with new images
docker compose up -d --force-recreate
```

**Step 4: Verify health**

```bash
# Check container status
docker compose ps

# Check health endpoint
curl -sk https://localhost/api/v1/healthz

# Verify version
docker inspect renos-core --format '{{.Config.Image}}'
```

### Upgrading with docker-compose.yml Changes

If the docker-compose.yml has changed between versions:

```bash
# Step 1: Back up current compose file
cp docker-compose.yml docker-compose.yml.backup

# Step 2: Apply new compose file
# (Replace with new version from repository)

# Step 3: Pull and recreate
docker compose pull
docker compose up -d --force-recreate --remove-orphans

# Step 4: Remove old containers
docker container prune -f
```

### Upgrading Specific Components

```bash
# Upgrade only the core component
docker compose pull renos-core
docker compose up -d --force-recreate renos-core

# Upgrade only the hub component
docker compose pull renos-hub
docker compose up -d --force-recreate renos-hub

# Upgrade the agent
docker compose pull renos-agent
docker compose up -d --force-recreate renos-agent
```

### Monitoring During Upgrade

```bash
# Watch container status during upgrade
watch docker compose ps

# Follow logs for all services
docker compose logs -f

# Follow logs for a specific service
docker compose logs -f renos-core
```

---

## Step-by-Step Upgrade for Kubernetes (Helm)

### Prerequisites

- kubectl configured with cluster access
- Helm 3.8+ installed
- Access to the RenOS Helm repository
- Sufficient cluster resources for rolling update

### Upgrade Steps

**Step 1: Update Helm repository**

```bash
helm repo update renos
```

**Step 2: Review changes**

```bash
# Check what will change
helm diff upgrade renos renos/renos -n renos \
  --values values-production.yaml

# Or without helm-diff plugin:
helm get values renos -n renos > current-values.yaml
# Compare current-values.yaml with your new values file
```

**Step 3: Back up current release**

```bash
# Export current values for rollback
helm get values renos -n renos -o yaml > renos-values-backup.yaml

# Create database backup
kubectl exec -n renos deploy/renos-postgres -- pg_dumpall -U renos | gzip > pre-upgrade-db.sql.gz
```

**Step 4: Perform the Helm upgrade**

```bash
helm upgrade renos renos/renos -n renos \
  --values values-production.yaml \
  --atomic \
  --timeout 15m \
  --cleanup-on-fail
```

The `--atomic` flag ensures automatic rollback if the upgrade fails.

**Step 5: Verify the upgrade**

```bash
# Check release status
helm list -n renos

# Check pods are rolling
kubectl get pods -n renos -w

# Check all pods are ready
kubectl wait --for=condition=Ready pods --all -n renos --timeout=300s

# Verify version
kubectl get pods -n renos -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.containers[*].image}{"\n"}{end}'
```

### Upgrading with Custom Values

If you maintain custom Helm values:

```bash
# Upgrade with values file
helm upgrade renos renos/renos -n renos \
  --values values-production.yaml \
  --set core.replicaCount=3 \
  --set postgres.storage=50Gi \
  --atomic

# Upgrade with --reuse-values (use existing values, only apply chart changes)
helm upgrade renos renos/renos -n renos --reuse-values

# Upgrade with --reset-values (ignore existing, use chart defaults)
helm upgrade renos renos/renos -n renos --reset-values
```

### Upgrading with Helmfile

For organizations managing multiple environments:

```yaml
# helmfile.yaml
repositories:
  - name: renos
    url: https://helm.renos.ai

releases:
  - name: renos
    namespace: renos
    chart: renos/renos
    version: ">=2.0.0"
    values:
      - values-{{ .Environment.Name }}.yaml
    set:
      - name: global.env.RENOS_LOG_LEVEL
        value: info
```

```bash
helmfile -e production apply
```

### Zero-Downtime Upgrade Strategy

For production environments requiring zero downtime:

1. **Ensure replicaCount >= 2** for core and hub
2. **Configure PodDisruptionBudget** (included in chart)
3. **Use rolling update strategy** (default)

```bash
# Configure PDB (already included in chart)
helm upgrade renos renos/renos -n renos \
  --set core.replicaCount=3 \
  --set hub.replicaCount=3 \
  --atomic \
  --timeout 15m
```

The default Kubernetes rolling update strategy ensures:
- At most 25% unavailable during update
- At most 25% surge above desired replicas
- Services remain available throughout the upgrade

---

## Step-by-Step Upgrade for Native Install

### Upgrade via Installer Script

```bash
# Download the latest installer
curl -fsSL https://install.renos.ai -o install-new.sh

# Run upgrade (the installer detects existing installation)
sudo bash install-new.sh
```

The installer script handles upgrades by:
1. Detecting existing RenOS installation
2. Creating a restore point at `/var/backups/renos/pre-install/`
3. Stopping existing services
4. Pulling new Docker images
5. Running database migrations
6. Updating configuration files
7. Starting services with new version

### Manual Upgrade

**Step 1: Stop services**

```bash
systemctl stop renos-core renos-hub renos-agent renos-nginx
```

**Step 2: Back up configuration**

```bash
cp -r /etc/renos /etc/renos.backup.$(date +%Y%m%d)
```

**Step 3: Pull new Docker images**

```bash
docker compose -f /etc/renos/docker-compose.yml pull
```

**Step 4: Run database migrations**

```bash
docker compose -f /etc/renos/docker-compose.yml run --rm renos-core renos-core migrate
```

**Step 5: Start services**

```bash
systemctl start renos-core renos-hub renos-agent renos-nginx
```

**Step 6: Verify**

```bash
systemctl status renos-core renos-hub renos-agent renos-nginx
```

### Upgrading Native PostgreSQL

If using native PostgreSQL (not Docker-based):

```bash
# Stop RenOS services
systemctl stop renos-core renos-hub renos-agent renos-nginx

# Backup database
pg_dumpall -U renos > /tmp/renos-pre-upgrade.sql

# Upgrade PostgreSQL packages
sudo apt-get update && sudo apt-get upgrade postgresql  # Debian/Ubuntu
sudo dnf upgrade postgresql-server  # Fedora/RHEL

# Run pg_upgrade if major version changed
sudo -u postgres pg_upgrade

# Restart PostgreSQL
systemctl restart postgresql

# Start RenOS services
systemctl start renos-core renos-hub renos-agent renos-nginx
```

---

## Post-Upgrade Verification

### Health Check Checklist

After any upgrade, verify the following:

- [ ] All services are running (core, hub, agent, nginx)
- [ ] Database migrations completed successfully
- [ ] API health endpoint returns 200
- [ ] Dashboard loads and is functional
- [ ] Device agents show as connected
- [ ] NATS cluster is healthy
- [ ] Redis is responding
- [ ] TLS certificates are valid
- [ ] Backups are running on schedule
- [ ] Logs show no errors

### Verification Commands

```bash
# System health overview
renos-health-check.sh

# API health
curl -sk https://localhost/api/v1/healthz | jq .

# Database connection
docker compose exec postgres pg_isready -U renos

# Redis connection
docker compose exec redis redis-cli ping

# NATS health
docker compose exec nats nats server check connection

# Container status
docker compose ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}"

# Version verification
grep RENOS_INSTALL_VERSION /etc/renos/renos.env

# Upgrade history
renos-upgrade history
```

### Functional Verification

1. **Log in** to the dashboard at `https://your-server`
2. **Verify user accounts** still work
3. **Check device list** — all previously connected devices should appear
4. **Test device enrollment** by enrolling a new device
5. **Verify workflows** — existing workflows should continue to function
6. **Check notifications** and alerting configuration
7. **Test API calls** from existing integrations

### Smoke Test Suite

```bash
# Run the RenOS smoke tests (if available)
renos-smoke-test.sh

# Or manually test key endpoints
curl -sk https://localhost/api/v1/healthz
curl -sk https://localhost/api/v1/version
curl -sk -o /dev/null -w "%{http_code}" https://localhost/api/v1/devices
curl -sk -o /dev/null -w "%{http_code}" https://localhost/login
```

---

## Rollback Procedures

### Rollback via Upgrade Manager

If the upgrade was performed using the upgrade manager:

```bash
# List recent upgrades
renos-upgrade history

# Rollback to a specific upgrade record
renos-upgrade rollback upg-1700000000123456789

# Force rollback (skip compatibility checks)
renos-upgrade rollback upg-1700000000123456789 --force
```

### Manual Rollback - Docker

**Step 1: Restore database backup**

```bash
# Restore from pre-upgrade backup
gunzip -c pre-upgrade-db.sql.gz | docker compose exec -T postgres psql -U renos

# Or using the backup system
renos-restore.sh --restore renos-backup-20250101-020000
```

**Step 2: Revert Docker images**

```bash
# Tag previous images
docker tag renos/core:2.0.0 renos/core:latest
docker tag renos/hub:2.0.0 renos/hub:latest
docker tag renos/agent:2.0.0 renos/agent:latest
```

Or use specific version tags:

```bash
# Edit docker-compose.yml to pin to previous version
# Change image tags from "latest" to specific version
# e.g., renos/core:1.0.0

# Recreate containers
docker compose up -d --force-recreate
```

**Step 3: Restore configuration**

```bash
# Restore previous config
cp /etc/renos/renos.env.backup /etc/renos/renos.env

# Restart services
systemctl restart renos-core renos-hub renos-agent renos-nginx
```

### Manual Rollback - Kubernetes

```bash
# Method 1: Rollback Helm release
helm rollback renos -n renos

# Rollback to a specific revision
helm rollback renos -n renos <revision-number>

# Check revision history
helm history renos -n renos

# Method 2: Reapply previous values
helm upgrade renos renos/renos -n renos \
  --values renos-values-backup.yaml \
  --atomic \
  --timeout 15m

# Method 3: Restore database
kubectl exec -n renos deploy/renos-postgres -- pg_restore -U renos -d renos -c < backup.sql
```

### Rollback Validation

After rollback, verify:

```bash
# Check all services are running
renos-health-check.sh

# Verify API is responding
curl -sk https://localhost/api/v1/healthz

# Check version has reverted
renos-upgrade status

# Run smoke tests
renos-smoke-test.sh
```

---

## Database Migration Notes

### Migration Strategy

RenOS uses a sequential migration system for database schema changes:

1. **Migrations are idempotent** — running them multiple times is safe
2. **Migrations run on startup** — via the `renos-core migrate` command
3. **Each migration has a unique version** — tracked in a `schema_migrations` table
4. **Downgrade migrations** — available for supported version pairs

### Migration Commands

```bash
# Run all pending migrations
docker compose run --rm renos-core renos-core migrate

# Run migrations to a specific version
docker compose run --rm renos-core renos-core migrate --to 20250101000000

# Check migration status
docker compose run --rm renos-core renos-core migrate --status

# Dry run (show what would be applied)
docker compose run --rm renos-core renos-core migrate --dry-run
```

### Migration Safety

- **Automatic backups**: The upgrade manager creates a database backup before running migrations
- **Transactional migrations**: Each migration runs in a transaction when possible
- **Lock timeouts**: Tables are locked only during schema changes, not during data migrations
- **Rollback scripts**: Each forward migration includes a corresponding rollback

### Data Migration During Upgrades

Major version upgrades may include data migrations:

- **v1.0.0 → v2.0.0**: Device identifier format changed, compatibility layer included
- **v1.2.0 → v2.0.0**: Workflow engine schema updated, runs background migration

Data migrations are designed to:
1. Allow read access to existing data during migration
2. Run in batches to avoid long table locks
3. Include progress logging
4. Pause and resume if interrupted

### Schema Migration Table

```sql
-- The schema_migrations table tracks applied migrations
SELECT * FROM schema_migrations ORDER BY version;

-- Example output:
-- version     | dirty | applied_at
-- ------------+-------+--------------------
-- 2024010100 | f     | 2025-01-01 00:00:00
-- 2024020100 | f     | 2025-02-01 00:00:00
-- 2024030100 | f     | 2025-03-01 00:00:00
```

---

## Breaking Changes Between Versions

### v0.9.x → v1.0.0 (Initial GA)

**Breaking Changes:**

1. **Database schema restructured**
   - Tables renamed from `v0_*` to standard names
   - Automatic migration on upgrade
   - Downgrade script available

2. **API versioning introduced**
   - All endpoints now under `/api/v1/`
   - Old `/api/` endpoints redirect to `/api/v1/`
   - Rate limiting applied to auth endpoints

3. **Configuration file format**
   - New mandatory fields: `RENOS_JWT_SECRET`, `RENOS_ENCRYPTION_KEY`
   - Environment file path changed from `/etc/renos.conf` to `/etc/renos/renos.env`
   - Old format auto-detected and migrated on upgrade

4. **Plugin API changes**
   - Plugin interface version bumped to v2
   - Old v1 plugins continue to work in compatibility mode
   - Plugin manifest now requires `api_version` field

5. **Agent enrollment flow**
   - New enrollment token format (JWT-based)
   - Legacy tokens accepted during grace period
   - Keychain integration for macOS changed

**Automatic Migration:**
- The installer script handles most configuration migrations
- Database migrations run automatically on first startup
- Environment file is re-generated with new format

### v1.0.x → v1.2.x

**Breaking Changes:**

1. **Rate limiting introduced**
   - API rate limits at 30 req/s for general endpoints
   - Auth rate limits at 5 requests per minute
   - Configurable via `RENOS_RATE_LIMIT` environment variable

2. **Log format change**
   - Default format changed to JSON for structured logging
   - Plain text format available via `RENOS_LOG_FORMAT=text`
   - New log fields added for correlation IDs

3. **TLS configuration**
   - Minimum TLS version raised to 1.2
   - TLS 1.0 and 1.1 no longer supported
   - Self-signed certificate generation uses 4096-bit RSA keys

4. **NATS configuration**
   - JetStream enabled by default for message persistence
   - New cluster port (6222) opened for NATS clustering
   - NATS monitoring port (8222) added

### v1.2.x → v2.0.0

**Breaking Changes:**

1. **API v1 endpoints deprecated**
   - All v1 endpoints still functional but emit deprecation warnings
   - New features only available via v2 API
   - v1 API will be removed in v3.0.0

2. **Workflow engine v2**
   - New workflow definition format (YAML-based)
   - Old JSON workflows auto-converted on first load
   - New scheduling capabilities require schema migration

3. **Database migration required**
   - Schema changes for workflow engine
   - Device data restructuring for improved performance
   - Background migration processes in batches

4. **Agent protocol update**
   - New message format for agent-core communication
   - Backward compatible — old agents continue to work
   - New features require agent v2.0.0+

5. **Configuration changes**
   - `RENOS_STORAGE_BACKEND` new field for S3/NFS support
   - `RENOS_MONITORING_ENABLED` replaces individual monitoring flags
   - Old flags continue to work but emit deprecation warnings

### v2.0.x → Future Versions

Breaking changes will be documented in release notes published on the GitHub Releases page. Each major version update will include:

1. Migration guide from previous version
2. List of deprecated features and their replacements
3. Configuration migration tooling
4. Automated rollback scripts
5. Staged migration support for zero-downtime upgrades

---

## Upgrade Frequency Recommendations

| Environment | Update Strategy | Frequency |
|-------------|----------------|-----------|
| Development | Latest | As available |
| Staging | Follow production | 1 week before production |
| Production | N-1 stable releases | Monthly |
| Critical/Security | Immediate | As released |
| Air-gapped | Bundled updates | Quarterly |

## Version Lifecycle

| Phase | Support Level | Duration |
|-------|--------------|----------|
| Active development | Full support | Until next major |
| General availability | Security + bug fixes | 12 months |
| Maintenance | Critical security only | 6 months |
| End of life | No support | N/A |

## Notifications

To receive upgrade notifications:

```bash
# Subscribe to GitHub releases
# Watch the renos/renos repository on GitHub

# Enable in-app notifications
# Dashboard → Settings → Notifications → Product Updates

# Slack integration
# Dashboard → Settings → Integrations → Slack
```
