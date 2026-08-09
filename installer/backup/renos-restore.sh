#!/usr/bin/env bash
#===============================================================================
# RenOS Restore System
#   Restores a RenOS backup from a given backup directory.
#   Usage: renos-restore.sh [OPTIONS] BACKUP_DIR
#===============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/renos-backup-config.sh"

# ── Colors ────────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'
info()  { echo -e "${BLUE}[INFO]${NC}  $*"; }
ok()    { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*" >&2; }
err()   { echo -e "${RED}[ERROR]${NC} $*" >&2; }
die()   { err "$@"; exit 1; }
header(){ echo -e "\n${BOLD}${CYAN}══ $* ══${NC}\n"; }
step()  { echo -e "\n${BOLD}${CYAN}── $* ──${NC}"; }

# ── State ─────────────────────────────────────────────────────────────────────
DRY_RUN=false
PARTIAL=()
FORCE=false
TO_DIR=""
BACKUP_DIR=""
declare -a COMPONENT_ORDER=(database redis vault plugins marketplace userdata history certs config)

# ── Requirement checks ────────────────────────────────────────────────────────
__require_cmds() {
  local missing=false
  for cmd in "$@"; do
    if ! command -v "$cmd" &>/dev/null; then
      err "Required command not found: $cmd"
      missing=true
    fi
  done
  $missing && die "Install missing dependencies and retry."
}

# ── Decrypt file if encrypted ─────────────────────────────────────────────────
__maybe_decrypt() {
  local src=$1
  local dst=$2

  if [[ -f "${src}.age" ]]; then
    if command -v age &>/dev/null; then
      info "Decrypting with age: $(basename "${src}.age")"
      $DRY_RUN || age --decrypt -o "$dst" "${src}.age" || return 1
      return 0
    else
      err "age not found — cannot decrypt ${src}.age"
      return 1
    fi
  elif [[ -f "${src}.gpg" ]]; then
    if command -v gpg &>/dev/null; then
      info "Decrypting with gpg: $(basename "${src}.gpg")"
      $DRY_RUN || gpg --batch --yes --decrypt -o "$dst" "${src}.gpg" || return 1
      return 0
    else
      err "gpg not found — cannot decrypt ${src}.gpg"
      return 1
    fi
  elif [[ -f "$src" ]]; then
    $DRY_RUN || cp "$src" "$dst"
    return 0
  fi

  return 1
}

# ── Service management ────────────────────────────────────────────────────────
__services() {
  local action=$1  # stop | start | status
  local svcs=("renos-api" "renos-worker" "renos-scheduler" "renos-nats" "postgresql" "redis")
  local success=true

  case "$action" in
    stop)
      step "Stopping RenOS services..."
      for svc in "${svcs[@]}"; do
        if systemctl is-active --quiet "$svc" 2>/dev/null; then
          info "Stopping $svc..."
          $DRY_RUN || systemctl stop "$svc" || { warn "Failed to stop $svc"; success=false; }
        fi
      done
      ;;
    start)
      step "Starting RenOS services..."
      for svc in "${svcs[@]}"; do
        info "Starting $svc..."
        $DRY_RUN || systemctl start "$svc" || { warn "Failed to start $svc"; success=false; }
      done
      ;;
    status)
      local all_ok=true
      for svc in "${svcs[@]}"; do
        if systemctl is-active --quiet "$svc" 2>/dev/null; then
          ok "$svc is running"
        else
          warn "$svc is not running"
          all_ok=false
        fi
      done
      $all_ok || return 1
      ;;
  esac

  $success
}

# ── Health checks ─────────────────────────────────────────────────────────────
__health_checks() {
  step "Running health checks..."

  local errors=0

  # Systemd services
  info "Checking systemd services..."
  __services status || ((errors++))

  # API health endpoint
  if command -v curl &>/dev/null; then
    info "Checking API health endpoint..."
    if $DRY_RUN; then
      ok "[DRY-RUN] Would check https://localhost:8443/healthz"
    else
      local http_code
      http_code=$(curl -sk -o /dev/null -w '%{http_code}' --connect-timeout 5 \
        https://localhost:8443/healthz 2>/dev/null || echo "000")
      if [[ "$http_code" == "200" ]]; then
        ok "API health endpoint responded with 200"
      else
        err "API health endpoint returned $http_code (expected 200)"
        ((errors++))
      fi
    fi
  fi

  # Database
  info "Checking database connectivity..."
  if $DRY_RUN; then
    ok "[DRY-RUN] Would check PostgreSQL at ${POSTGRES_HOST}:${POSTGRES_PORT}"
  else
    if command -v psql &>/dev/null; then
      PGPASSWORD="${PGPASSWORD:-}" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT 1;" &>/dev/null && \
        ok "Database reachable" || { err "Database unreachable"; ((errors++)); }
    fi
  fi

  # Redis
  info "Checking Redis connectivity..."
  if $DRY_RUN; then
    ok "[DRY-RUN] Would check Redis at ${REDIS_HOST}:${REDIS_PORT}"
  else
    if command -v redis-cli &>/dev/null; then
      redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" PING &>/dev/null && \
        ok "Redis reachable" || { err "Redis unreachable"; ((errors++)); }
    fi
  fi

  # NATS
  if command -v nats &>/dev/null; then
    info "Checking NATS connectivity..."
    if $DRY_RUN; then
      ok "[DRY-RUN] Would check NATS at localhost:4222"
    else
      nats server check --server nats://localhost:4222 2>/dev/null && \
        ok "NATS reachable" || warn "NATS not reachable (may need configuration)"
    fi
  fi

  # TLS
  if command -v openssl &>/dev/null; then
    info "Checking TLS certificate validity..."
    if $DRY_RUN; then
      ok "[DRY-RUN] Would check TLS cert expiry"
    else
      local cert_file="${RENOS_ETC}/certs/server.crt"
      if [[ -f "$cert_file" ]]; then
        local expiry
        expiry=$(openssl x509 -enddate -noout -in "$cert_file" 2>/dev/null | cut -d= -f2)
        local expiry_epoch
        expiry_epoch=$(date -d "$expiry" +%s 2>/dev/null || echo 0)
        local now; now=$(date +%s)
        if [[ "$expiry_epoch" -gt "$now" ]]; then
          local days_left=$(( (expiry_epoch - now) / 86400 ))
          ok "TLS certificate valid (${days_left} days until expiry)"
        else
          err "TLS certificate EXPIRED on $expiry"
          ((errors++))
        fi
      else
        warn "No TLS certificate found at $cert_file"
      fi
    fi
  fi

  return "$errors"
}

# ── Restore database ──────────────────────────────────────────────────────────
restore_database() {
  step "Restoring PostgreSQL database"
  __require_cmds pg_restore

  local src="${BACKUP_DIR}/renos-db.dump"
  local decrypted="${BACKUP_DIR}/.restore-db.dump"

  __maybe_decrypt "$src" "$decrypted" || {
    warn "Database backup not found — skipping"
    return 0
  }

  if $DRY_RUN; then
    ok "[DRY-RUN] Would run: pg_restore $PGRESTORE_OPTS -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB $decrypted"
    rm -f "$decrypted"
    return 0
  fi

  info "Dropping and recreating database ${POSTGRES_DB}..."
  PGPASSWORD="${PGPASSWORD:-}" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d postgres \
    -c "DROP DATABASE IF EXISTS ${POSTGRES_DB};" 2>/dev/null || warn "Could not drop database"
  PGPASSWORD="${PGPASSWORD:-}" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d postgres \
    -c "CREATE DATABASE ${POSTGRES_DB};" 2>/dev/null || die "Could not create database"

  info "Restoring from dump..."
  PGPASSWORD="${PGPASSWORD:-}" pg_restore $PGRESTORE_OPTS \
    -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" \
    -d "$POSTGRES_DB" "$decrypted" 2>&1 | tee -a "${BACKUP_DIR}/restore-db.log"
  local rc=${PIPESTATUS[0]}

  rm -f "$decrypted"

  if [[ $rc -eq 0 ]]; then
    ok "Database restored successfully"
  else
    err "Database restore failed (exit code $rc) — check ${BACKUP_DIR}/restore-db.log"
  fi
  return "$rc"
}

# ── Restore Redis ─────────────────────────────────────────────────────────────
restore_redis() {
  step "Restoring Redis"
  __require_cmds redis-cli

  local src="${BACKUP_DIR}/redis-dump.rdb"
  local decrypted="${BACKUP_DIR}/.restore-redis.rdb"

  __maybe_decrypt "$src" "$decrypted" || {
    warn "Redis backup not found — skipping"
    return 0
  }

  if $DRY_RUN; then
    ok "[DRY-RUN] Would copy $decrypted to Redis dump dir and restart Redis"
    rm -f "$decrypted"
    return 0
  fi

  local redis_dump_dir
  redis_dump_dir=$(redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" CONFIG GET dir 2>/dev/null | tail -1 || echo "/var/lib/redis")

  info "Stopping Redis..."
  systemctl stop redis 2>/dev/null || systemctl stop redis-server 2>/dev/null || warn "Could not stop Redis"

  info "Copying dump.rdb to ${redis_dump_dir}/..."
  cp "$decrypted" "${redis_dump_dir}/dump.rdb"
  chown redis:redis "${redis_dump_dir}/dump.rdb" 2>/dev/null || true

  info "Starting Redis..."
  systemctl start redis 2>/dev/null || systemctl start redis-server 2>/dev/null || {
    err "Could not start Redis after restore"
    rm -f "$decrypted"
    return 1
  }

  rm -f "$decrypted"

  if redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" PING &>/dev/null; then
    ok "Redis restored and running"
  else
    err "Redis is not responding after restore"
    return 1
  fi
}

# ── Restore files from tar.gz ─────────────────────────────────────────────────
__restore_tar() {
  local component=$1
  local archive_name=$2
  local target_dir=$3
  local use_base=$4  # true = extract with basename, false = strip components

  local src="${BACKUP_DIR}/${archive_name}"
  local decrypted="${BACKUP_DIR}/.restore-${archive_name}"

  __maybe_decrypt "$src" "$decrypted" || {
    warn "${component} backup not found — skipping"
    return 0
  }

  local effective_target="$target_dir"
  if [[ -n "$TO_DIR" ]]; then
    effective_target="$TO_DIR"
  fi

  if $DRY_RUN; then
    ok "[DRY-RUN] Would extract $archive_name to $effective_target"
    rm -f "$decrypted"
    return 0
  fi

  mkdir -p "$effective_target"

  info "Restoring ${component} to ${effective_target}..."
  tar -xzf "$decrypted" -C "$effective_target" 2>&1 | tee -a "${BACKUP_DIR}/restore-${component}.log"
  local rc=${PIPESTATUS[0]}

  rm -f "$decrypted"

  if [[ $rc -eq 0 ]]; then
    ok "${component} restored to ${effective_target}"
  else
    err "${component} restore failed — check ${BACKUP_DIR}/restore-${component}.log"
  fi
  return "$rc"
}

restore_vault()     { __restore_tar "vault"     "renos-vault.tar.gz"     "$(dirname "$VAULT_PATH")"     true; }
restore_plugins()   { __restore_tar "plugins"   "renos-plugins.tar.gz"    "$(dirname "$PLUGIN_PATH")"    true; }
restore_marketplace(){ __restore_tar "marketplace" "renos-marketplace.tar.gz" "$(dirname "$MARKETPLACE_PATH")" true; }
restore_user_data() { __restore_tar "userdata"  "renos-userdata.tar.gz"   "$(dirname "$USER_DATA_PATH")" true; }
restore_history()   { __restore_tar "history"   "renos-history.tar.gz"    "$(dirname "$HISTORY_PATH")"   true; }
restore_certs()     { __restore_tar "certs"     "renos-certs.tar.gz"      "/"                           false; }
restore_config()    {
  if [[ -n "$TO_DIR" ]]; then
    __restore_tar "config" "renos-config.tar.gz" "$TO_DIR" true
  else
    __restore_tar "config" "renos-config.tar.gz" "$(dirname "$RENOS_ETC")" true
  fi
}

# ── Verify backup integrity ───────────────────────────────────────────────────
verify_integrity() {
  step "Verifying Backup Integrity"

  local manifest_file="${BACKUP_DIR}/backup-info.json"
  local errors=0

  if [[ ! -f "$manifest_file" ]]; then
    err "Manifest file (backup-info.json) not found — cannot verify integrity"
    return 2
  fi

  if command -v python3 &>/dev/null; then
    local status
    status=$(python3 -c "import json; print(json.load(open('$manifest_file')).get('status','unknown'))" 2>/dev/null)
    if [[ "$status" != "completed" ]]; then
      err "Backup status is '$status', expected 'completed'"
      ((errors++))
    fi
  fi

  for component in "${COMPONENT_ORDER[@]}"; do
    local fname=""
    case "$component" in
      database) fname="renos-db.dump" ;;
      redis)    fname="redis-dump.rdb" ;;
      vault)    fname="renos-vault.tar.gz" ;;
      plugins)  fname="renos-plugins.tar.gz" ;;
      marketplace) fname="renos-marketplace.tar.gz" ;;
      userdata) fname="renos-userdata.tar.gz" ;;
      history)  fname="renos-history.tar.gz" ;;
      certs)    fname="renos-certs.tar.gz" ;;
      config)   fname="renos-config.tar.gz" ;;
    esac

    local found=false
    [[ -f "${BACKUP_DIR}/${fname}" ]] && found=true
    [[ -f "${BACKUP_DIR}/${fname}.age" ]] && found=true
    [[ -f "${BACKUP_DIR}/${fname}.gpg" ]] && found=true

    # Check manifest for expected hash
    local expected_hash=""
    if command -v python3 &>/dev/null; then
      expected_hash=$(python3 -c "
import json
with open('$manifest_file') as f: d = json.load(f)
print(d.get('checksums', {}).get('$component', ''))
" 2>/dev/null)
    fi

    if [[ -n "$expected_hash" ]]; then
      local actual_file=""
      [[ -f "${BACKUP_DIR}/${fname}" ]] && actual_file="${BACKUP_DIR}/${fname}"
      if [[ -z "$actual_file" ]]; then
        # encrypted — note it
        ok "[ENCRYPTED] ${fname} — skipping hash verification"
        continue
      fi
      local actual_hash
      actual_hash=$(sha256sum "$actual_file" | cut -d' ' -f1)
      if [[ "$actual_hash" != "$expected_hash" ]]; then
        err "Checksum mismatch for ${component}: expected ${expected_hash}, got ${actual_hash}"
        ((errors++))
      else
        ok "Checksum OK: ${fname}"
      fi
    else
      if $found; then
        ok "Present: ${fname}"
      else
        # Optional components may be absent
        case "$component" in
          vault|plugins|marketplace|history|certs)
            # optional — skip
            ;;
          *)
            warn "Missing component: ${fname}"
            ((errors++))
            ;;
        esac
      fi
    fi
  done

  if [[ $errors -eq 0 ]]; then
    ok "Backup integrity check passed"
  else
    err "${errors} integrity error(s) found"
  fi
  return "$errors"
}

# ── Main restore flow ─────────────────────────────────────────────────────────

usage() {
  cat <<-EOF
${BOLD}RenOS Restore System${NC}
Usage: $(basename "$0") [OPTIONS] BACKUP_DIR

${BOLD}Options:${NC}
  --dry-run       Validate the restore process without making changes
  --partial LIST  Restore only specific components (comma-separated)
                  Components: database,redis,vault,plugins,marketplace,userdata,history,certs,config
  --force         Skip all confirmation prompts
  --to-dir PATH   Restore files to a different directory
  --help          Show this help message

${BOLD}Examples:${NC}
  $(basename "$0") /var/lib/renos/backups/renos-backup-20250101-120000
  $(basename "$0") --dry-run /var/lib/renos/backups/renos-backup-20250101-120000
  $(basename "$0") --partial database,config /var/lib/renos/backups/renos-backup-20250101-120000
  $(basename "$0") --to-dir /tmp/restore-test /var/lib/renos/backups/renos-backup-20250101-120000
  $(basename "$0") --force /var/lib/renos/backups/renos-backup-20250101-120000
EOF
  exit 0
}

main() {
  # Root check
  if [[ $EUID -ne 0 ]]; then
    die "This script must be run as root (sudo)."
  fi

  [[ $# -eq 0 ]] && usage

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --dry-run) DRY_RUN=true; shift ;;
      --partial) shift; IFS=',' read -ra PARTIAL <<< "$1"; shift ;;
      --force) FORCE=true; shift ;;
      --to-dir) shift; TO_DIR="$1"; shift ;;
      --help|-h) usage ;;
      -*)
        # Check if it's a backup dir starting with -
        if [[ -d "$1" ]]; then
          BACKUP_DIR="$1"; shift
        else
          die "Unknown option: $1"
        fi
        ;;
      *)
        if [[ -z "$BACKUP_DIR" ]]; then
          BACKUP_DIR="$1"; shift
        else
          die "Unexpected argument: $1"
        fi
        ;;
    esac
  done

  if [[ -z "$BACKUP_DIR" ]]; then
    die "Backup directory is required. See --help."
  fi

  # Resolve backup dir (relative to BACKUP_ROOT or absolute)
  if [[ ! -d "$BACKUP_DIR" ]]; then
    local resolved="${BACKUP_ROOT}/${BACKUP_DIR}"
    if [[ -d "$resolved" ]]; then
      BACKUP_DIR="$resolved"
    else
      die "Backup directory not found: $BACKUP_DIR (tried: $BACKUP_DIR and $resolved)"
    fi
  fi

  # Make absolute
  BACKUP_DIR="$(cd "$BACKUP_DIR" && pwd)"

  header "RenOS Restore — $(date)"
  info "Backup: $BACKUP_DIR"
  $DRY_RUN && info "DRY RUN MODE — no changes will be made"
  [[ -n "$TO_DIR" ]] && info "Restore target: $TO_DIR"

  # Confirmation
  if ! $FORCE && ! $DRY_RUN; then
    echo -e "\n${YELLOW}WARNING: This will OVERWRITE existing data.${NC}"
    read -r -p "Are you sure you want to proceed? [y/N] " response
    case "$response" in
      [yY][eE][sS]|[yY]) ;;
      *) die "Restore cancelled by user." ;;
    esac
  fi

  # 1. Verify integrity
  verify_integrity || {
    if $FORCE; then
      warn "Integrity check failed but --force is set — continuing"
    else
      die "Integrity check failed. Use --force to restore anyway."
    fi
  }

  # 2. Stop services
  if $DRY_RUN; then
    info "[DRY-RUN] Would stop services"
  else
    __services stop
  fi

  # 3. Restore components
  local errors=0
  local do_all=false

  if [[ ${#PARTIAL[@]} -eq 0 ]]; then
    do_all=true
  fi

  for component in "${COMPONENT_ORDER[@]}"; do
    if $do_all || [[ " ${PARTIAL[*]} " == *" $component "* ]]; then
      case "$component" in
        database)    restore_database || ((errors++)) ;;
        redis)       restore_redis || ((errors++)) ;;
        vault)       restore_vault || ((errors++)) ;;
        plugins)     restore_plugins || ((errors++)) ;;
        marketplace) restore_marketplace || ((errors++)) ;;
        userdata)    restore_user_data || ((errors++)) ;;
        history)     restore_history || ((errors++)) ;;
        certs)       restore_certs || ((errors++)) ;;
        config)      restore_config || ((errors++)) ;;
      esac
    fi
  done

  # 4. Start services
  if $DRY_RUN; then
    info "[DRY-RUN] Would start services"
  elif [[ $errors -gt 0 ]]; then
    warn "Some components had errors — inspect logs before starting services"
    if ! $FORCE; then
      read -r -p "Start services anyway? [y/N] " response
      [[ "$response" =~ ^[yY] ]] && __services start || warn "Services not started"
    else
      __services start
    fi
  else
    __services start
  fi

  # 5. Health checks
  local health_errors=0
  if $DRY_RUN; then
    info "[DRY-RUN] Would run health checks"
  else
    __health_checks || health_errors=$?
  fi

  # Summary
  header "Restore Summary"
  if $DRY_RUN; then
    ok "Dry-run completed — no changes made"
  elif [[ $errors -eq 0 && $health_errors -eq 0 ]]; then
    ok "Restore completed successfully"
  else
    warn "Restore completed with ${errors} restore error(s) and ${health_errors} health check issue(s)"
  fi

  exit $(( errors + health_errors > 0 ? 1 : 0 ))
}

main "$@"
