# RenOS Disaster Recovery Guide

## Table of Contents

- [Backup Overview](#backup-overview)
- [Creating Backups](#creating-backups)
- [Restoring from Backup](#restoring-from-backup)
- [Disaster Scenarios and Recovery Steps](#disaster-scenarios-and-recovery-steps)
- [Rebuilding a Node from Scratch](#rebuilding-a-node-from-scratch)
- [Cluster Recovery](#cluster-recovery)
- [Restoring TLS Certificates](#restoring-tls-certificates)
- [Verification After Recovery](#verification-after-recovery)

---

## Backup Overview

The RenOS backup system provides comprehensive protection for all critical data components. Backups are created using the `renos-backup.sh` script located in the `installer/backup/` directory.

### Backup Components

| Component | Description | Backup Type | Criticality |
|-----------|-------------|-------------|-------------|
| **Database** | PostgreSQL database (devices, users, workflows, configurations) | Full dump (pg_dump) | Critical |
| **Redis** | Cache data, session state, rate limiting | RDB snapshot | High |
| **Vault** | Encrypted secrets, API keys, tokens | File archive | Critical |
| **Plugins** | Installed plugin configurations | File archive | Medium |
| **Marketplace** | Downloaded marketplace items | File archive | Low |
| **User Data** | Device configs, workflow definitions, schedules | File archive | High |
| **History** | Workflow execution history, audit logs | File archive | Medium |
| **CA Certificates** | Internal certificate authority and TLS certs | File archive | Critical |
| **Config Files** | RenOS configuration, environment file | File archive | Critical |

### Backup Architecture

```
                         ┌─────────────────────┐
                         │   renos-backup.sh    │
                         │  (backup orchestrator)│
                         └──────────┬──────────┘
                                    │
            ┌───────────────────────┼───────────────────────┐
            │                       │                       │
            ▼                       ▼                       ▼
   ┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐
   │   PostgreSQL     │   │     Redis       │   │  File System    │
   │   pg_dumpall     │   │   SAVE command  │   │   tar + gzip    │
   └────────┬────────┘   └────────┬────────┘   └────────┬────────┘
            │                     │                      │
            └─────────────────────┼──────────────────────┘
                                  │
                                  ▼
                     ┌─────────────────────┐
                     │   Backup Directory   │
                     │ /var/lib/renos/      │
                     │   backups/           │
                     └──────────┬──────────┘
                                │
                    ┌───────────┴───────────┐
                    │                       │
                    ▼                       ▼
          ┌─────────────────┐   ┌─────────────────┐
          │   Encryption    │   │  Remote Upload  │
          │   age / gpg     │   │ S3 / SCP / Local│
          └─────────────────┘   └─────────────────┘
```

---

## Creating Backups

### Manual Backup

```bash
# Full backup (all components)
sudo renos-backup.sh --full

# Database-only backup
sudo renos-backup.sh --db-only

# Files-only backup (skip database and Redis)
sudo renos-backup.sh --files-only

# Full backup with encryption
sudo renos-backup.sh --full --encrypt

# Full backup with remote upload
sudo renos-backup.sh --full --remote

# Full backup with encryption AND remote upload
sudo renos-backup.sh --full --encrypt --remote
```

### Automated Backup (Cron)

The backup system installs a cron job by default that runs daily at 2:00 AM:

```bash
# Check if cron job is installed
crontab -l | grep renos-backup

# Expected output:
# 0 2 * * * root /usr/local/bin/renos-backup.sh --full --encrypt --remote
```

To customize the cron schedule:

```bash
# Edit the cron configuration
sudo crontab -e

# Example schedules:
# Every 6 hours
0 */6 * * * /usr/local/bin/renos-backup.sh --full --encrypt --remote

# Twice daily
0 2,14 * * * /usr/local/bin/renos-backup.sh --full --encrypt --remote

# Weekly full backup, daily incremental
0 2 * * 0 /usr/local/bin/renos-backup.sh --full --encrypt --remote
0 6 * * 1-6 /usr/local/bin/renos-backup.sh --db-only --encrypt --remote
```

### Backup Before Upgrades

Always create a full backup before performing an upgrade:

```bash
# Create a pre-upgrade backup
sudo renos-backup.sh --full --encrypt --remote

# Verify the backup was created
sudo renos-backup.sh --list

# Check backup integrity
sudo renos-backup.sh --verify renos-backup-20250101-020000
```

### Backup Configuration

The backup system is configured in `/etc/renos/backup-config.sh` (or `installer/backup/renos-backup-config.sh`):

```bash
# ── General Settings ──────────────────────────────────
BACKUP_ROOT="/var/lib/renos/backups"
BACKUP_PREFIX="renos-backup"
BACKUP_DIR_FORMAT="%Y%m%d-%H%M%S"
RETENTION_COUNT=7
COMPRESSION_LEVEL=6

# ── Component Toggles ─────────────────────────────────
BACKUP_DATABASE=true
BACKUP_REDIS=true
BACKUP_VAULT=true
BACKUP_PLUGINS=true
BACKUP_MARKETPLACE=true
BACKUP_USER_DATA=true
BACKUP_HISTORY=true
BACKUP_CA_CERTS=true
BACKUP_CONFIG=true

# ── Remote Storage ────────────────────────────────────
REMOTE_ENABLED=true
REMOTE_TYPE="s3"  # s3, scp, local
S3_BUCKET="renos-backups"
S3_REGION="us-east-1"
SCP_HOST="backup-server.example.com"
SCP_USER="root"
SCP_PATH="/var/backups/renos"

# ── Encryption ────────────────────────────────────────
ENCRYPT_ENABLED=true
ENCRYPT_TO="age1..."
```

### Listing Available Backups

```bash
# List all backups
sudo renos-backup.sh --list

# Output:
# ══ Available Backups in /var/lib/renos/backups ══
#   renos-backup-20250101-020000  (2.3G)  2025-01-01 02:00:01  [completed]
#   renos-backup-20241231-020000  (2.1G)  2024-12-31 02:00:02  [completed]
#   renos-backup-20241230-020000  (2.0G)  2024-12-30 02:00:00  [completed]
```

### Verifying Backups

```bash
# Verify a specific backup by ID
sudo renos-backup.sh --verify renos-backup-20250101-020000

# Output:
# ══ Verifying Backup: renos-backup-20250101-020000 ══
#   ✓ database: /var/lib/renos/backups/renos-backup-20250101-020000/renos-db.dump (512M)
#   ✓ redis: /var/lib/renos/backups/renos-backup-20250101-020000/redis-dump.rdb (128M)
#   ✓ vault: /var/lib/renos/backups/renos-backup-20250101-020000/renos-vault.tar.gz (1.2M)
#   ✓ plugins: /var/lib/renos/backups/renos-backup-20250101-020000/renos-plugins.tar.gz (4.5M)
#   ✓ marketplace: .../renos-marketplace.tar.gz (256M)
#   ✓ userdata: .../renos-userdata.tar.gz (890M)
#   ✓ history: .../renos-history.tar.gz (1.5G)
#   ✓ certs: .../renos-certs.tar.gz (2.1M)
#   ✓ config: .../renos-config.tar.gz (45K)
# ✓ Backup integrity verified successfully
```

### Backup Storage Strategies

| Strategy | Retention | Storage | Recovery Time | Cost |
|----------|-----------|---------|---------------|------|
| Local only | 7 days | Disk | Minutes | Low |
| Local + S3 | 30 days local, 90 days S3 | Disk + Cloud | Minutes (local) / Hours (S3) | Medium |
| Local + SCP | 7 days local, 365 days remote | Disk + NAS | Minutes (local) / Hours (remote) | Medium |
| S3 only | 90 days + Glacier archival | Cloud | Hours | High |
| Tape | 365 days + offsite | Tape library | Hours | High |

---

## Restoring from Backup

### Full Restore

Restore all components from a backup:

```bash
# Restore from backup ID
sudo renos-restore.sh --restore renos-backup-20250101-020000

# Restore from latest backup
sudo renos-restore.sh --restore latest
```

### Manual Full Restore Process

**Step 1: Stop all RenOS services**

```bash
systemctl stop renos-core renos-hub renos-agent renos-nginx
docker compose -f /etc/renos/docker-compose.yml down
```

**Step 2: Locate the backup**

```bash
ls -la /var/lib/renos/backups/renos-backup-20250101-*/
```

**Step 3: Restore the database**

```bash
# For custom format dumps (compressed):
pg_restore --no-owner --clean --if-exists \
  -d renos /path/to/backup/renos-db.dump

# For plain SQL dumps:
psql -U renos -d renos < /path/to/backup/renos-db.sql
```

**Step 4: Restore Redis**

```bash
# Stop Redis
systemctl stop redis  # or: docker compose stop redis

# Copy the RDB file
cp /path/to/backup/redis-dump.rdb /var/lib/redis/dump.rdb

# Start Redis
systemctl start redis  # or: docker compose start redis
```

**Step 5: Restore file archives**

```bash
# Restore all file-based backups
for archive in /path/to/backup/renos-*.tar.gz; do
  component=$(basename "$archive" .tar.gz | sed 's/renos-//')
  echo "Restoring $component..."
  tar -xzf "$archive" -C /
done
```

**Step 6: Restore TLS certificates**

```bash
# Restore CA and TLS certificates
tar -xzf /path/to/backup/renos-certs.tar.gz -C /
chmod 600 /etc/renos/tls/privkey.pem 2>/dev/null || true
```

**Step 7: Restart services**

```bash
systemctl start renos-core renos-hub renos-agent renos-nginx
```

### Partial Restore

Restore specific components:

```bash
# Database only restore
sudo renos-restore.sh --restore renos-backup-20250101-020000 --component database

# Configuration only restore
sudo renos-restore.sh --restore renos-backup-20250101-020000 --component config

# User data only restore
sudo renos-restore.sh --restore renos-backup-20250101-020000 --component userdata
```

### Restoring Encrypted Backups

If encryption (age) was used:

```bash
# Decrypt with age private key
age --decrypt -i ~/.config/renos/backup-key.txt \
  -o renos-db.dump \
  renos-db.dump.age
```

If encryption (gpg) was used:

```bash
# Decrypt with GPG
gpg --decrypt --output renos-db.dump renos-db.dump.gpg
```

### Restoring from Remote Storage

If backups were uploaded to S3:

```bash
# Download from S3 using rclone
rclone copy renos-s3:renos-backups/renos-backup-20250101-020000/ /tmp/restore/

# Download from S3 using aws CLI
aws s3 cp --recursive s3://renos-backups/renos-backup-20250101-020000/ /tmp/restore/

# Download via SCP
rsync -avz root@backup-server:/var/backups/renos/renos-backup-20250101-020000/ /tmp/restore/
```

---

## Disaster Scenarios and Recovery Steps

### Scenario 1: Database Corruption

**Symptoms:**
- API returns 500 errors for database queries
- Dashboard shows no devices or empty data
- Error logs contain: `pq: relation "..." does not exist` or `FATAL: database is corrupted`
- `docker compose logs postgres` shows checksum or page verification errors

**Recovery Steps:**

**Step 1: Stop RenOS services**

```bash
systemctl stop renos-core renos-hub renos-agent
```

**Step 2: Attempt PostgreSQL repair**

```bash
# Enter the PostgreSQL container
docker compose exec postgres bash

# Attempt to recover the database
pg_ctl -D /var/lib/postgresql/data -l /tmp/pg_recovery.log start

# If that fails, attempt emergency repair
pg_resetwal -f /var/lib/postgresql/data

# Check if PostgreSQL starts
pg_ctl -D /var/lib/postgresql/data start
```

**Step 3: If repair fails, restore from backup**

```bash
# Stop and remove corrupted data
docker compose down
docker volume rm renos_pgdata

# Restart with clean database
docker compose up -d postgres

# Wait for PostgreSQL to initialize
sleep 30

# Restore from latest backup
docker compose exec -T postgres pg_restore -U renos -d renos -c \
  < /var/lib/renos/backups/renos-backup-20250101-020000/renos-db.dump
```

**Step 4: Verify and restart**

```bash
# Check database integrity
docker compose exec postgres pg_isready -U renos

# Verify data
docker compose exec postgres psql -U renos -c "SELECT count(*) FROM devices;"

# Start remaining services
systemctl start renos-core renos-hub renos-agent
```

### Scenario 2: Failed Upgrade

**Symptoms:**
- Core service fails to start after upgrade
- Database migration errors in logs
- Incompatible configuration detected
- API returns version mismatch errors

**Recovery Steps:**

**Step 1: Stop all services**

```bash
systemctl stop renos-core renos-hub renos-agent renos-nginx
```

**Step 2: Check upgrade state**

```bash
renos-upgrade status
renos-upgrade history
```

**Step 3: Rollback via upgrade manager**

```bash
# Find the upgrade record
renos-upgrade history

# Rollback
renos-upgrade rollback upg-1700000000123456789

# If upgrade manager fails, manual rollback:
```

**Step 4: Manual rollback**

```bash
# Restore previous Docker images
docker compose pull
# OR pin to specific versions in docker-compose.yml

# Restore configuration
cp /etc/renos/renos.env.backup /etc/renos/renos.env

# Restore database
docker compose down
docker volume rm renos_pgdata
docker compose up -d postgres
sleep 30
docker compose exec -T postgres pg_restore -U renos -d renos -c \
  < pre-upgrade-backup.sql

# Restart services
systemctl start renos-core renos-hub renos-agent renos-nginx
```

**Step 5: Document the failure**

```bash
# Save logs for support
journalctl -u renos-core --since "1 hour ago" > /tmp/failed-upgrade.log
renos-upgrade history > /tmp/upgrade-history.log
```

### Scenario 3: Hardware Failure

**Symptoms:**
- Server is unresponsive (power, network, disk failure)
- Disk I/O errors in system logs
- RAID controller alerts
- Filesystem corruption detected

**Recovery Steps:**

**Step 1: Provision new hardware**

```bash
# Order replacement hardware with same or better specifications:
# - Same CPU architecture (x86_64 / arm64)
# - Equal or greater RAM
# - Equal or greater disk capacity (SSD/NVMe)
# - Same network configuration
```

**Step 2: Install RenOS on new hardware**

```bash
# Follow the Installation Guide for your platform
sudo bash install.sh --dry-run  # Preview
sudo bash install.sh             # Install
```

**Step 3: Stop new RenOS services**

```bash
systemctl stop renos-core renos-hub renos-agent renos-nginx
docker compose -f /etc/renos/docker-compose.yml down
```

**Step 4: Restore from backup**

```bash
# If backups are on the failed server:
# Remount the old disk if accessible
mount /dev/sdb1 /mnt/old-backups
cp -r /mnt/old-backups/renos-backup-* /var/lib/renos/backups/

# If backups are remote (S3):
aws s3 cp --recursive s3://renos-backups/ /var/lib/renos/backups/

# Restore all components
sudo renos-restore.sh --restore latest
```

**Step 5: Verify and resume**

```bash
# Start services
systemctl start renos-core renos-hub renos-agent renos-nginx

# Verify restoration
renos-health-check.sh
```

**Step 6: Re-enroll devices (if needed)**

If device state cannot be fully restored, devices will reconnect automatically when they detect the server is back online. For devices that require re-enrollment:

```bash
# Generate new enrollment tokens
renos-cli tokens create --role device
```

### Scenario 4: Accidental Deletion

**Common accidental deletions:**
- User/device deleted from dashboard
- Workflow or schedule removed
- Configuration file deleted or corrupted
- Database table dropped
- Docker volume removed

**Recovery Steps:**

**For configuration file deletion:**

```bash
# Restore from backup
sudo renos-restore.sh --restore renos-backup-20250101-020000 --component config

# Or manually extract
tar -xzf /var/lib/renos/backups/renos-backup-*/renos-config.tar.gz -C /

# Restart affected services
systemctl restart renos-core
```

**For user/device deletion:**

```bash
# Restore database from backup
sudo renos-restore.sh --restore renos-backup-20250101-020000 --component database

# Or selectively restore specific data using psql
docker compose exec -T postgres psql -U renos -d renos < recovered-data.sql
```

**For Docker volume deletion (WARNING: data loss):**

```bash
# If the volume was accidentally removed:
docker volume ls | grep renos

# If the volume no longer exists, restore from backup:
docker compose down
docker volume rm renos_pgdata  # Remove corrupted/empty volume
docker compose up -d postgres
sleep 30
docker compose exec -T postgres pg_restore -U renos -d renos -c \
  < /var/lib/renos/backups/renos-backup-*/renos-db.dump
```

### Scenario 5: Security Breach

**Symptoms:**
- Unauthorized access detected in audit logs
- Admin account compromised
- API keys or secrets leaked
- Suspicious device enrollments
- Data exfiltration detected

**Immediate Response:**

**Step 1: Isolate the system**

```bash
# Block external network access (temporarily)
iptables -A INPUT -p tcp --dport 443 -j DROP
iptables -A INPUT -p tcp --dport 80 -j DROP

# Or disconnect network interface
ifconfig eth0 down
```

**Step 2: Rotate all secrets**

```bash
# Generate new secrets
NEW_JWT_SECRET=$(openssl rand -hex 32)
NEW_ENCRYPTION_KEY=$(openssl rand -hex 32)
NEW_PGPASSWORD=$(openssl rand -base64 24)
NEW_REDIS_PASSWORD=$(openssl rand -base64 24)

# Update environment file
sed -i "s/RENOS_JWT_SECRET=.*/RENOS_JWT_SECRET=$NEW_JWT_SECRET/" /etc/renos/renos.env
sed -i "s/RENOS_ENCRYPTION_KEY=.*/RENOS_ENCRYPTION_KEY=$NEW_ENCRYPTION_KEY/" /etc/renos/renos.env

# Update database password
docker compose exec postgres psql -U renos -c \
  "ALTER USER renos WITH PASSWORD '$NEW_PGPASSWORD';"

# Update Redis password
docker compose exec redis redis-cli \
  CONFIG SET requirepass "$NEW_REDIS_PASSWORD"
```

**Step 3: Revoke all tokens and sessions**

```bash
# Invalidate all JWT tokens (requires core API)
curl -X POST -k https://localhost/api/v1/auth/revoke-all \
  -H "Authorization: Bearer $ADMIN_TOKEN"

# Force all users to re-authenticate
# Clear session store
docker compose exec redis redis-cli FLUSHALL
```

**Step 4: Restore from clean backup**

```bash
# Identify the last known-good backup (before breach)
sudo renos-backup.sh --list

# Full restore
sudo renos-restore.sh --restore renos-backup-20241230-020000
```

**Step 5: Apply security patches**

```bash
# Update to latest version
renos-upgrade run latest

# Run security audit
renos-security-audit.sh
```

**Step 6: Re-enable network**

```bash
# Restore network access after securing the system
iptables -D INPUT -p tcp --dport 443 -j DROP
ifconfig eth0 up
```

**Post-Incident:**
1. Rotate all user passwords (force password reset)
2. Re-issue all enrollment tokens
3. Review audit logs for unauthorized activity
4. Update firewall rules and network policies
5. Conduct security review and apply recommendations
6. File incident report as per organizational policy

---

## Rebuilding a Node from Scratch

### Prerequisites

- Same OS version and architecture as the original node
- Access to the latest backup
- Same hostname or DNS configuration
- Network connectivity restored

### Full Node Rebuild Process

**Step 1: Provision the operating system**

```bash
# Install the same OS version as the original node
# Configure hostname, network, and DNS
hostnamectl set-hostname renos-node1.example.com
```

**Step 2: Install RenOS**

```bash
# Download and run the installer
curl -fsSL https://install.renos.ai -o install.sh
sudo bash install.sh
```

**Step 3: Stop newly installed services**

```bash
systemctl stop renos-core renos-hub renos-agent renos-nginx
docker compose -f /etc/renos/docker-compose.yml down
```

**Step 4: Restore configuration**

```bash
# Restore environment file
tar -xzf /var/lib/renos/backups/renos-backup-*/renos-config.tar.gz -C /
```

**Step 5: Restore TLS certificates**

```bash
tar -xzf /var/lib/renos/backups/renos-backup-*/renos-certs.tar.gz -C /
chmod 600 /etc/renos/tls/privkey.pem
```

**Step 6: Restore database**

```bash
# Start PostgreSQL
docker compose up -d postgres
sleep 30

# Drop default database and restore
docker compose exec postgres psql -U renos -c "DROP DATABASE IF EXISTS renos;"
docker compose exec -T postgres pg_restore -U renos -d renos \
  /var/lib/renos/backups/renos-backup-*/renos-db.dump
```

**Step 7: Restore file data**

```bash
# Restore user data, vault, plugins, etc.
for archive in /var/lib/renos/backups/renos-backup-*/renos-*.tar.gz; do
  component=$(basename "$archive" .tar.gz | sed 's/renos-//')
  # Skip config as already restored, skip certs as already restored
  case "$component" in
    config|certs) continue ;;
    *) tar -xzf "$archive" -C / ;;
  esac
done
```

**Step 8: Start services**

```bash
systemctl start renos-core renos-hub renos-agent renos-nginx
```

**Step 9: Verify**

```bash
renos-health-check.sh
renos-backup.sh --list
curl -sk https://localhost/api/v1/healthz
```

---

## Cluster Recovery

### Single Node Failure in Cluster

If one node fails in a multi-node cluster:

**Step 1: Verify cluster status**

```bash
# On any surviving node
docker compose exec nats nats server check cluster
docker compose exec postgres psql -U renos -c "SELECT * FROM pg_stat_replication;"
```

**Step 2: Remove failed node from cluster**

```bash
# NATS cluster auto-detects failure; verify routes
docker compose exec nats nats server report connections

# For database replication, promote a replica if needed
docker compose exec postgres psql -U renos -c \
  "SELECT pg_promote();"
```

**Step 3: Rebuild the failed node**

```bash
# On the failed node, reinstall and join the cluster
# See: Rebuilding a Node from Scratch

# After restoration, the node re-joins the NATS cluster automatically
```

### Full Cluster Recovery

If all cluster nodes fail:

**Step 1: Identify the node with the latest data**

```bash
# Check backup timestamps on each surviving node
ls -lt /var/lib/renos/backups/ | head -5
```

**Step 2: Rebuild primary node**

Follow [Rebuilding a Node from Scratch](#rebuilding-a-node-from-scratch) on the node with the latest backup.

**Step 3: Restore cluster configuration**

```bash
# Update NATS cluster configuration
cat >> /etc/renos/renos.env << 'EOF'
RENOS_NATS_CLUSTER_ENABLED=true
RENOS_NATS_CLUSTER_NODES=nats://node1:6222,nats://node2:6222,nats://node3:6222
EOF
```

**Step 4: Add secondary nodes**

```bash
# On each secondary node:
# 1. Install RenOS
# 2. Join to primary node
# 3. Configure replication
```

### Database Cluster Recovery

For PostgreSQL streaming replication:

```bash
# On the primary, check replication status
docker compose exec postgres psql -U renos -c "SELECT * FROM pg_stat_replication;"

# Rebuild a standby from backup
docker compose exec postgres psql -U renos -c \
  "SELECT pg_backup_start('standby rebuild');"
# Copy data to standby
# On standby:
docker compose exec postgres psql -U renos -c \
  "SELECT pg_backup_stop();"
```

---

## Restoring TLS Certificates

### Certificate Types

| Certificate | Location | Purpose | Expiry |
|-------------|----------|---------|--------|
| **Server TLS** | `/etc/renos/tls/fullchain.pem` | HTTPS termination | 90 days (Let's Encrypt) / 10 years (self-signed) |
| **Server TLS Key** | `/etc/renos/tls/privkey.pem` | Private key for TLS | Matches cert |
| **CA Certificate** | `/var/lib/renos/ca/ca.pem` | Internal CA for device auth | 10 years |
| **CA Key** | `/var/lib/renos/ca/ca-key.pem` | CA private key | Matches CA cert |

### Restore from Backup

```bash
# Extract certificates from backup
tar -xzf /var/lib/renos/backups/renos-backup-*/renos-certs.tar.gz -C /

# Set correct permissions
chmod 600 /etc/renos/tls/privkey.pem
chmod 644 /etc/renos/tls/fullchain.pem
chown -R renos:renos /etc/renos/tls

# Restart services
systemctl restart renos-nginx renos-core
```

### Regenerate Self-Signed Certificates

If certificates are not available in backup:

```bash
# Regenerate self-signed certificate
openssl req -x509 -nodes -days 3650 -newkey rsa:4096 \
  -keyout /etc/renos/tls/privkey.pem \
  -out /etc/renos/tls/fullchain.pem \
  -subj "/CN=$(hostname -f)" \
  -addext "subjectAltName=DNS:$(hostname -f),IP:127.0.0.1"

chmod 600 /etc/renos/tls/privkey.pem
chown -R renos:renos /etc/renos/tls

# Restart nginx
systemctl restart renos-nginx
```

### Re-obtain Let's Encrypt Certificates

```bash
# Stop nginx temporarily
systemctl stop renos-nginx

# Obtain new certificate
certbot certonly --standalone --non-interactive --agree-tos \
  --email admin@example.com -d renos.example.com

# Copy to RenOS directory
cp /etc/letsencrypt/live/renos.example.com/fullchain.pem /etc/renos/tls/
cp /etc/letsencrypt/live/renos.example.com/privkey.pem /etc/renos/tls/
chmod 600 /etc/renos/tls/privkey.pem

# Start nginx
systemctl start renos-nginx
```

### Restore CA Certificates

The CA certificates are critical for device authentication:

```bash
# Restore from backup
tar -xzf /var/lib/renos/backups/renos-backup-*/renos-certs.tar.gz -C /
chmod 600 /var/lib/renos/ca/ca-key.pem
chown -R renos:renos /var/lib/renos/ca

# If CA is lost, devices will need re-enrollment with new certificates
```

---

## Verification After Recovery

### Health Verification Checklist

After any recovery operation, verify the following:

- [ ] All services are running (core, hub, agent, nginx)
- [ ] Database is accessible and data is intact
- [ ] Redis is responding and cache is populated
- [ ] NATS cluster (if applicable) is healthy
- [ ] API health endpoint returns 200
- [ ] Dashboard loads correctly
- [ ] TLS certificates are valid and not expired
- [ ] Devices show as connected
- [ ] Backups are running on schedule
- [ ] No errors in application logs
- [ ] All integrations and plugins are functional

### Verification Commands

```bash
# System health
renos-health-check.sh

# API health
curl -sk https://localhost/api/v1/healthz | jq .

# Service status
systemctl status renos-core renos-hub renos-agent renos-nginx

# Container status
docker compose ps

# Database check
docker compose exec postgres psql -U renos -c "\dt"

# Redis check
docker compose exec redis redis-cli ping

# NATS check
docker compose exec nats nats server check connection

# TLS verification
openssl s_client -connect localhost:443 -servername localhost < /dev/null 2>/dev/null | openssl x509 -text -noout | grep -A2 Validity

# Data integrity check
docker compose exec postgres psql -U renos -c "SELECT count(*) FROM devices;"
docker compose exec postgres psql -U renos -c "SELECT count(*) FROM users;"
```

### Data Integrity Verification

```bash
# Check for database corruption
docker compose exec postgres psql -U renos -d renos -c "VACUUM VERBOSE ANALYZE;"

# Check for missing indexes
docker compose exec postgres psql -U renos -d renos -c "SELECT * FROM pg_stat_user_tables WHERE seq_scan > 0 AND idx_scan = 0;"

# Verify foreign key consistency
docker compose exec postgres psql -U renos -d renos -c "SELECT 'Data integrity check passed' AS status;"
```

### Recovery Validation Report

Generate a recovery validation report:

```bash
#!/bin/bash
# recovery-validation-report.sh

echo "=== RenOS Recovery Validation Report ==="
echo "Date: $(date)"
echo ""

echo "--- Service Status ---"
systemctl is-active renos-core renos-hub renos-agent renos-nginx

echo ""
echo "--- API Health ---"
curl -sk -o /dev/null -w "%{http_code}" https://localhost/api/v1/healthz
echo ""

echo ""
echo "--- Database ---"
docker compose exec postgres pg_isready -U renos &>/dev/null && echo "OK" || echo "FAILED"

echo ""
echo "--- Redis ---"
docker compose exec redis redis-cli ping &>/dev/null && echo "OK" || echo "FAILED"

echo ""
echo "--- Data Counts ---"
docker compose exec postgres psql -U renos -t -c "SELECT 'devices: ' || count(*) FROM devices;" 2>/dev/null
docker compose exec postgres psql -U renos -t -c "SELECT 'users: ' || count(*) FROM users;" 2>/dev/null
docker compose exec postgres psql -U renos -t -c "SELECT 'workflows: ' || count(*) FROM workflows;" 2>/dev/null

echo ""
echo "=== Report Complete ==="
```

### Post-Recovery Tasks

1. **Verify backup schedule** is operational
2. **Test a new backup** to confirm backup system works
3. **Update documentation** with recovery timeline and lessons learned
4. **Notify stakeholders** that services are restored
5. **Schedule post-incident review** within 48 hours
6. **Review monitoring alerts** to ensure they would have caught the incident earlier

### Recovery Time Objectives

| Scenario | Target RTO | Target RPO |
|----------|-----------|------------|
| Database corruption | 30 minutes | 1 hour |
| Failed upgrade | 15 minutes | 5 minutes (pre-upgrade backup) |
| Hardware failure | 4 hours | 24 hours |
| Accidental deletion | 1 hour | 1 hour |
| Security breach | 2 hours | 24 hours (pre-compromise backup) |
| Single node failure (cluster) | 5 minutes | 0 (replication) |
| Full cluster failure | 4 hours | 1 hour |
