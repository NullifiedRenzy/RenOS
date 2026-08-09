# RenOS Backup & Restore System

Production-grade backup/restore tooling for RenOS installations.

## Files

| File | Purpose |
|---|---|
| `renos-backup.sh` | Main backup orchestration script |
| `renos-restore.sh` | Disaster recovery restore script |
| `renos-backup-config.sh` | Shared configuration (sourced by both) |

## Quick Start

```bash
# Full backup with encryption + remote upload
sudo ./renos-backup.sh --full --encrypt --remote

# List available backups
sudo ./renos-backup.sh --list

# Verify a backup
sudo ./renos-backup.sh --verify renos-backup-20250101-120000

# Prune old backups
sudo ./renos-backup.sh --prune

# Restore from a backup
sudo ./renos-restore.sh /var/lib/renos/backups/renos-backup-20250101-120000

# Dry-run restore (validate only)
sudo ./renos-restore.sh --dry-run /var/lib/renos/backups/renos-backup-20250101-120000

# Partial restore (database + config only)
sudo ./renos-restore.sh --partial database,config /var/lib/renos/backups/renos-backup-20250101-120000

# Restore to alternate location
sudo ./renos-restore.sh --to-dir /tmp/restore-test /var/lib/renos/backups/renos-backup-20250101-120000
```

## Configuration

Edit `renos-backup-config.sh` or create `/etc/renos/backup-config.local.sh` to override:

```bash
# Retention
RETENTION_COUNT=14

# Encryption (age preferred)
ENCRYPT_ENABLED=true
AGE_PUB_KEY="age1..."

# Remote storage (S3 via rclone)
REMOTE_ENABLED=true
REMOTE_TYPE=s3
RCLONE_REMOTE="renos-s3"
```

## Backup Storage

Backups are stored in `/var/lib/renos/backups/renos-backup-YYYYMMDD-HHMMSS/`:

```
renos-backup-20250101-120000/
├── backup-info.json       # Manifest with checksums
├── renos-db.dump          # PostgreSQL custom-format dump
├── redis-dump.rdb         # Redis RDB snapshot
├── renos-vault.tar.gz     # Vault secrets
├── renos-plugins.tar.gz   # Plugin configurations
├── renos-marketplace.tar.gz
├── renos-userdata.tar.gz  # Devices, workflows, schedules
├── renos-history.tar.gz   # Workflow execution history
├── renos-certs.tar.gz     # CA certificates
├── renos-config.tar.gz    # /etc/renos/ configuration
└── *.age                  # Encrypted variants (if --encrypt)
```

## Encryption

Primarily supports **age** (`age -r PUBKEY`). Falls back to GPG. Encrypt each
component file individually. The manifest (`backup-info.json`) is **not**
encrypted so restore tools can discover components.

## Retention

Keeps the last `RETENTION_COUNT` backups (default: 7). Older backups are
pruned automatically after each successful backup. Run `--prune` standalone
to perform cleanup at any time.

## Safety

- All operations are idempotent — running backups concurrently is safe.
- Restore creates a full backup directory; uses checksums for integrity.
- `--dry-run` on restore validates the backup without touching the system.
- Service stop/start is handled automatically during restore.
- Components missing from a backup are skipped gracefully.

## Schedule (cron)

```
# Daily backup at 2am
0 2 * * * /opt/renos/installer/backup/renos-backup.sh --full --encrypt --remote
```
