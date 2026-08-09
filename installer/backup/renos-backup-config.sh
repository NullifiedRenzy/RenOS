#!/usr/bin/env bash
# RenOS Backup Configuration
# Source this file in renos-backup.sh and renos-restore.sh
# Path: /home/ren/Projects/RenOS/installer/backup/renos-backup-config.sh

set -o pipefail

# ──────────────────────────────────────────────
# General Settings
# ──────────────────────────────────────────────
BACKUP_ROOT="/var/lib/renos/backups"
BACKUP_PREFIX="renos-backup"
BACKUP_DIR_FORMAT="%Y%m%d-%H%M%S"
RETENTION_COUNT=7
COMPRESSION_LEVEL=6
COMPRESSION_CMD="gzip"
DECOMPRESSION_CMD="gunzip"
TAR_EXT="tar.gz"

# ──────────────────────────────────────────────
# Components to include (set to "true" or "false")
# ──────────────────────────────────────────────
BACKUP_DATABASE=true
BACKUP_REDIS=true
BACKUP_VAULT=true
BACKUP_PLUGINS=true
BACKUP_MARKETPLACE=true
BACKUP_USER_DATA=true
BACKUP_HISTORY=true
BACKUP_CA_CERTS=true
BACKUP_CONFIG=true

# ──────────────────────────────────────────────
# Paths
# ──────────────────────────────────────────────
RENOS_ETC="/etc/renos"
RENOS_DATA="/var/lib/renos"
RENOS_LOG="/var/log/renos"
POSTGRES_DB="renos"
POSTGRES_USER="renos"
POSTGRES_HOST="localhost"
POSTGRES_PORT=5432
REDIS_HOST="localhost"
REDIS_PORT=6379
REDIS_DB=0
VAULT_PATH="${RENOS_DATA}/vault"
PLUGIN_PATH="${RENOS_DATA}/plugins"
MARKETPLACE_PATH="${RENOS_DATA}/marketplace"
USER_DATA_PATH="${RENOS_DATA}/users"
WORKFLOW_PATH="${RENOS_DATA}/workflows"
SCHEDULE_PATH="${RENOS_DATA}/schedules"
HISTORY_PATH="${RENOS_DATA}/history"
DEVICE_PATH="${RENOS_DATA}/devices"
CA_CERT_PATH="/etc/renos/certs"

# ──────────────────────────────────────────────
# Remote / Encryption
# ──────────────────────────────────────────────
REMOTE_ENABLED=false
REMOTE_TYPE="s3"  # s3, scp, local
REMOTE_URL=""
REMOTE_PATH="backups/renos"
RCLONE_REMOTE="renos-s3"
S3_BUCKET="renos-backups"
S3_REGION="us-east-1"
SCP_USER="root"
SCP_HOST=""
SCP_PATH="/var/backups/renos"
ENCRYPT_ENABLED=false
ENCRYPT_TO=""
AGE_PUB_KEY=""
GPG_RECIPIENT=""

# ──────────────────────────────────────────────
# Cron schedule (example: daily at 2am)
# ──────────────────────────────────────────────
CRON_SCHEDULE="0 2 * * *"
CRON_USER="root"

# ──────────────────────────────────────────────
# pg_dump / pg_restore options
# ──────────────────────────────────────────────
PGDUMP_OPTS="--format=c --compress=9 --no-owner --no-acl"
PGRESTORE_OPTS="--no-owner --no-acl --clean --if-exists"

# ──────────────────────────────────────────────
# Validate config
# ──────────────────────────────────────────────
__renos_validate_config() {
  local errors=0

  if [[ ! -d "$BACKUP_ROOT" ]] && [[ "${BASH_SOURCE[1]}" != *"renos-restore.sh" ]]; then
    mkdir -p "$BACKUP_ROOT" 2>/dev/null || {
      echo "WARNING: Cannot create backup root $BACKUP_ROOT"
    }
  fi

  if [[ "$COMPRESSION_LEVEL" -lt 1 || "$COMPRESSION_LEVEL" -gt 9 ]]; then
    echo "ERROR: COMPRESSION_LEVEL must be 1-9, got $COMPRESSION_LEVEL"
    ((errors++))
  fi
  if [[ "$RETENTION_COUNT" -lt 1 ]]; then
    echo "ERROR: RETENTION_COUNT must be >= 1"
    ((errors++))
  fi

  return "$errors"
}

# Load optional local overrides
__renos_config_override="${RENOS_ETC}/backup-config.local.sh"
if [[ -f "$__renos_config_override" ]]; then
  source "$__renos_config_override"
fi

__renos_validate_config
