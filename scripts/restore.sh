#!/usr/bin/env bash
set -euo pipefail

# RenOS Database Restore Script
# ==============================
# Restores a RenOS database from a backup created by backup.sh.
#
# Usage:
#   ./scripts/restore.sh /path/to/backup.sql.gz
#
# WARNING: This will DROP all existing data and replace it with the backup.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
DB_URL="${RENOS_DATABASE_URL:-postgres://renos:renos@localhost:5432/renos?sslmode=disable}"

if [ $# -lt 1 ]; then
    echo "Usage: $0 <backup-file.sql.gz>"
    echo ""
    echo "Available backups:"
    ls -lh "$ROOT_DIR/backups"/renos-backup-*.sql.gz 2>/dev/null || echo "  (none found in $ROOT_DIR/backups/)"
    exit 1
fi

BACKUP_FILE="$1"

if [ ! -f "$BACKUP_FILE" ]; then
    echo "ERROR: Backup file not found: $BACKUP_FILE"
    exit 1
fi

echo "WARNING: This will DROP and REPLACE all data in the RenOS database."
echo "  Backup file: $BACKUP_FILE"
echo "  Database:    $(echo "$DB_URL" | sed 's/:[^@]*@/:***@/')"
echo ""
read -p "Are you sure? (yes/no): " CONFIRM
if [ "$CONFIRM" != "yes" ]; then
    echo "Aborted."
    exit 0
fi

echo ""
echo "==> Stopping Core (if running)..."
docker compose -f "$ROOT_DIR/docker-compose.yml" stop core 2>/dev/null || true

echo "==> Dropping and recreating database..."
if docker compose -f "$ROOT_DIR/docker-compose.yml" ps postgres 2>/dev/null | grep -q "Up"; then
    docker compose -f "$ROOT_DIR/docker-compose.yml" exec -T postgres \
        psql -U renos -d postgres -c "DROP DATABASE IF EXISTS renos;" -c "CREATE DATABASE renos OWNER renos;"

    echo "==> Restoring from backup..."
    gunzip -c "$BACKUP_FILE" | docker compose -f "$ROOT_DIR/docker-compose.yml" exec -T postgres \
        psql -U renos -d renos --single-transaction
else
    if command -v psql &>/dev/null; then
        PGPASSWORD="$(echo "$DB_URL" | sed -n 's|.*:\([^@]*\)@.*|\1|p')" \
            psql "$DB_URL" -d postgres -c "DROP DATABASE IF EXISTS renos;" -c "CREATE DATABASE renos OWNER renos;"

        echo "==> Restoring from backup..."
        gunzip -c "$BACKUP_FILE" | PGPASSWORD="$(echo "$DB_URL" | sed -n 's|.*:\([^@]*\)@.*|\1|p')" \
            psql "$DB_URL" -d renos --single-transaction
    else
        echo "ERROR: Neither docker compose postgres nor psql is available."
        exit 1
    fi
fi

echo "==> Running migrations to ensure schema is up to date..."
cd "$ROOT_DIR/core" && go run ./cmd/renos-core/ migrate 2>/dev/null || echo "  (migrations skipped - run manually if needed)"

echo "==> Restarting Core..."
docker compose -f "$ROOT_DIR/docker-compose.yml" start core 2>/dev/null || true

echo ""
echo "Restore complete!"
