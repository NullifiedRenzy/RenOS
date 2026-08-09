#!/usr/bin/env bash
set -euo pipefail

# RenOS Database Backup Script
# ============================
# Creates a timestamped pg_dump of the RenOS database.
#
# Usage:
#   ./scripts/backup.sh                     # Backup to ./backups/
#   ./scripts/backup.sh /path/to/dir        # Backup to custom directory
#
# Environment:
#   RENOS_DATABASE_URL  - Database connection string (uses docker compose service name by default)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BACKUP_DIR="${1:-$ROOT_DIR/backups}"
DB_URL="${RENOS_DATABASE_URL:-postgres://renos:renos@localhost:5432/renos?sslmode=disable}"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
BACKUP_FILE="$BACKUP_DIR/renos-backup-$TIMESTAMP.sql.gz"

mkdir -p "$BACKUP_DIR"

echo "==> Backing up RenOS database..."
echo "    Database: $(echo "$DB_URL" | sed 's/:[^@]*@/:***@/')"
echo "    Output:   $BACKUP_FILE"

# Check if docker compose postgres is available
if docker compose -f "$ROOT_DIR/docker-compose.yml" ps postgres 2>/dev/null | grep -q "Up"; then
    docker compose -f "$ROOT_DIR/docker-compose.yml" exec -T postgres \
        pg_dump -U renos renos | gzip > "$BACKUP_FILE"
else
    # Try direct pg_dump
    if command -v pg_dump &>/dev/null; then
        PGPASSWORD="$(echo "$DB_URL" | sed -n 's|.*:\([^@]*\)@.*|\1|p')" \
            pg_dump "$DB_URL" | gzip > "$BACKUP_FILE"
    else
        echo "ERROR: Neither docker compose postgres nor pg_dump is available."
        echo "Install postgresql-client or ensure docker compose services are running."
        exit 1
    fi
fi

FILESIZE=$(du -h "$BACKUP_FILE" | cut -f1)
echo ""
echo "Backup complete: $BACKUP_FILE ($FILESIZE)"
echo ""

# Cleanup old backups (keep last 30)
BACKUP_COUNT=$(ls -1 "$BACKUP_DIR"/renos-backup-*.sql.gz 2>/dev/null | wc -l)
if [ "$BACKUP_COUNT" -gt 30 ]; then
    REMOVE_COUNT=$((BACKUP_COUNT - 30))
    echo "Cleaning up old backups (keeping last 30, removing $REMOVE_COUNT)..."
    ls -1t "$BACKUP_DIR"/renos-backup-*.sql.gz | tail -n "$REMOVE_COUNT" | xargs rm -f
    echo "Cleanup complete."
fi
