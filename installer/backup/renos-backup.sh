#!/usr/bin/env bash
#===============================================================================
# RenOS Backup System
#   Backs up databases, Redis, vault, configs, user data, and more.
#   Usage: renos-backup.sh [--full|--db-only|--files-only|--list|--restore ID|--prune|--encrypt|--remote|--help]
#===============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/renos-backup-config.sh"

# ── Colors & helpers ──────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'
info()  { echo -e "${BLUE}[INFO]${NC}  $*"; }
ok()    { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*" >&2; }
err()   { echo -e "${RED}[ERROR]${NC} $*" >&2; }
die()   { err "$@"; exit 1; }
header(){ echo -e "\n${BOLD}${CYAN}══ $* ══${NC}\n"; }
spinner(){ local pid=$1; local msg=$2; local chars='|/-\'; local i=0; while kill -0 "$pid" 2>/dev/null; do i=$(( (i+1) %4 )); printf "\r${CYAN}[%c]${NC} %s" "${chars:$i:1}" "$msg"; sleep 0.15; done; printf "\r${GREEN}[✓]${NC} %s\n" "$msg"; }

# ── Timestamps & paths ────────────────────────────────────────────────────────
TIMESTAMP=$(date +"$BACKUP_DIR_FORMAT")
BACKUP_DIR="${BACKUP_ROOT}/${BACKUP_PREFIX}-${TIMESTAMP}"
MANIFEST="${BACKUP_DIR}/backup-info.json"
DB_DUMP="${BACKUP_DIR}/renos-db.dump"
REDIS_DUMP="${BACKUP_DIR}/redis-dump.rdb"
FILES_ARCHIVE="${BACKUP_DIR}/renos-files.tar.gz"
ENCRYPTED_SUFFIX=".age"

# ── Sanity checks ─────────────────────────────────────────────────────────────
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

# ── Manifest helpers ──────────────────────────────────────────────────────────
__manifest_init() {
  mkdir -p "$BACKUP_DIR"
  cat > "$MANIFEST" <<-EOF
{
  "backup_id": "${TIMESTAMP}",
  "timestamp": "$(date -Iseconds)",
  "hostname": "$(hostname)",
  "renos_version": "${RENOS_VERSION:-unknown}",
  "components": {},
  "checksums": {},
  "size_bytes": 0,
  "encrypted": false,
  "status": "incomplete"
}
EOF
}

__manifest_add() {
  local component=$1; local file=$2; local label=$3
  local size=0; local sha256=""
  if [[ -f "$file" ]]; then
    size=$(stat -c%s "$file" 2>/dev/null || stat -f%z "$file" 2>/dev/null || echo 0)
    sha256=$(sha256sum "$file" | cut -d' ' -f1)
  fi
  local tmp; tmp=$(mktemp)
  python3 -c "
import json, sys
with open('$MANIFEST') as f: d = json.load(f)
d['components']['$component'] = {'file': '$label', 'path': '$file'}
d['checksums']['$component'] = '$sha256'
d['size_bytes'] += $size
with open('$tmp', 'w') as f: json.dump(d, f, indent=2)
" 2>/dev/null || \
  warn "Failed to update manifest for $component (python3 not available)"
  [[ -f "$tmp" ]] && mv "$tmp" "$MANIFEST"
}

__manifest_finalize() {
  local encrypted="${1:-false}"
  local tmp; tmp=$(mktemp)
  python3 -c "
import json, os
with open('$MANIFEST') as f: d = json.load(f)
d['status'] = 'completed'
d['encrypted'] = $encrypted
d['size_bytes'] = sum(os.path.getsize(p) for c,data in d['components'].items() if os.path.exists(data['path']) for p in [data['path']]) if d['components'] else os.path.getsize('$BACKUP_DIR')
with open('$tmp', 'w') as f: json.dump(d, f, indent=2)
" 2>/dev/null
  [[ -f "$tmp" ]] && mv "$tmp" "$MANIFEST"
}

# ── Backup functions ──────────────────────────────────────────────────────────

backup_database() {
  $BACKUP_DATABASE || return 0
  header "Database (PostgreSQL)"
  __require_cmds pg_dump
  info "Dumping database ${POSTGRES_DB}..."
  PGPASSWORD="${PGPASSWORD:-}" pg_dump $PGDUMP_OPTS \
    -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" \
    -d "$POSTGRES_DB" -f "$DB_DUMP" 2>&1 | tee -a "${BACKUP_DIR}/pg_dump.log"
  if [[ ${PIPESTATUS[0]} -eq 0 ]]; then
    __manifest_add "database" "$DB_DUMP" "renos-db.dump"
    ok "Database dumped: $(numfmt --to=iec-i "$(stat -c%s "$DB_DUMP" 2>/dev/null || echo 0)" 2>/dev/null || stat -c%s "$DB_DUMP")"
  else
    err "Database dump failed — check ${BACKUP_DIR}/pg_dump.log"
    return 1
  fi
}

backup_redis() {
  $BACKUP_REDIS || return 0
  header "Redis"
  __require_cmds redis-cli
  local redis_dump_path=""
  info "Triggering Redis SAVE..."

  redis_dump_path=$(redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" CONFIG GET dir 2>/dev/null | tail -1 || echo "/var/lib/redis")
  redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" SAVE 2>&1 | tee -a "${BACKUP_DIR}/redis_save.log"
  if [[ ${PIPESTATUS[0]} -eq 0 ]]; then
    local rdb_file="${redis_dump_path}/dump.rdb"
    if [[ -f "$rdb_file" ]]; then
      cp "$rdb_file" "$REDIS_DUMP"
      __manifest_add "redis" "$REDIS_DUMP" "redis-dump.rdb"
      ok "Redis data saved: $(numfmt --to=iec-i "$(stat -c%s "$REDIS_DUMP" 2>/dev/null || echo 0)" 2>/dev/null || stat -c%s "$REDIS_DUMP")"
    else
      err "Redis dump.rdb not found at $rdb_file"
      return 1
    fi
  else
    err "Redis SAVE command failed"
    return 1
  fi
}

backup_vault() {
  $BACKUP_VAULT || return 0
  header "Vault Secrets"
  if [[ -d "$VAULT_PATH" ]]; then
    tar -czf "${BACKUP_DIR}/renos-vault.tar.gz" -C "$(dirname "$VAULT_PATH")" "$(basename "$VAULT_PATH")" 2>/dev/null
    __manifest_add "vault" "${BACKUP_DIR}/renos-vault.tar.gz" "renos-vault.tar.gz"
    ok "Vault secrets backed up"
  else
    warn "Vault path $VAULT_PATH does not exist — skipping"
  fi
}

backup_plugins() {
  $BACKUP_PLUGINS || return 0
  header "Plugin Configs"
  if [[ -d "$PLUGIN_PATH" ]]; then
    tar -czf "${BACKUP_DIR}/renos-plugins.tar.gz" -C "$(dirname "$PLUGIN_PATH")" "$(basename "$PLUGIN_PATH")" 2>/dev/null
    __manifest_add "plugins" "${BACKUP_DIR}/renos-plugins.tar.gz" "renos-plugins.tar.gz"
    ok "Plugin configs backed up"
  else
    warn "Plugin path $PLUGIN_PATH does not exist — skipping"
  fi
}

backup_marketplace() {
  $BACKUP_MARKETPLACE || return 0
  header "Marketplace Data"
  if [[ -d "$MARKETPLACE_PATH" ]]; then
    tar -czf "${BACKUP_DIR}/renos-marketplace.tar.gz" -C "$(dirname "$MARKETPLACE_PATH")" "$(basename "$MARKETPLACE_PATH")" 2>/dev/null
    __manifest_add "marketplace" "${BACKUP_DIR}/renos-marketplace.tar.gz" "renos-marketplace.tar.gz"
    ok "Marketplace data backed up"
  else
    warn "Marketplace path $MARKETPLACE_PATH does not exist — skipping"
  fi
}

backup_user_data() {
  $BACKUP_USER_DATA || return 0
  header "User Data (devices, workflows, schedules)"
  local has_data=false
  local tar_list=()

  for p in "$DEVICE_PATH" "$WORKFLOW_PATH" "$SCHEDULE_PATH" "$USER_DATA_PATH"; do
    [[ -d "$p" ]] && tar_list+=("$p") && has_data=true
  done

  if $has_data; then
    tar -czf "${BACKUP_DIR}/renos-userdata.tar.gz" \
      "${tar_list[@]}" 2>/dev/null
    __manifest_add "userdata" "${BACKUP_DIR}/renos-userdata.tar.gz" "renos-userdata.tar.gz"
    ok "User data backed up"
  else
    warn "No user data directories found — skipping"
  fi
}

backup_history() {
  $BACKUP_HISTORY || return 0
  header "Workflow History"
  if [[ -d "$HISTORY_PATH" ]]; then
    tar -czf "${BACKUP_DIR}/renos-history.tar.gz" -C "$(dirname "$HISTORY_PATH")" "$(basename "$HISTORY_PATH")" 2>/dev/null
    __manifest_add "history" "${BACKUP_DIR}/renos-history.tar.gz" "renos-history.tar.gz"
    ok "Workflow history backed up"
  else
    warn "History path $HISTORY_PATH does not exist — skipping"
  fi
}

backup_ca_certs() {
  $BACKUP_CA_CERTS || return 0
  header "CA Certificates"
  local cert_files=()
  if [[ -d "$CA_CERT_PATH" ]]; then
    while IFS= read -r -d '' f; do cert_files+=("$f"); done < <(find "$CA_CERT_PATH" -type f \( -name '*.crt' -o -name '*.pem' -o -name '*.cert' \) -print0 2>/dev/null)
  fi

  local ca_system="/etc/ssl/certs"
  if [[ -d "$ca_system" ]]; then
    while IFS= read -r -d '' f; do
      if [[ "$f" == *renos* || "$f" == *RenOS* ]]; then
        cert_files+=("$f")
      fi
    done < <(find "$ca_system" -type f \( -name '*.crt' -o -name '*.pem' \) -print0 2>/dev/null)
  fi

  if [[ ${#cert_files[@]} -gt 0 ]]; then
    tar -czf "${BACKUP_DIR}/renos-certs.tar.gz" \
      "${cert_files[@]}" 2>/dev/null
    __manifest_add "certs" "${BACKUP_DIR}/renos-certs.tar.gz" "renos-certs.tar.gz"
    ok "CA certificates backed up (${#cert_files[@]} files)"
  else
    warn "No RenOS CA certificates found — skipping"
  fi
}

backup_config() {
  $BACKUP_CONFIG || return 0
  header "Config Files"
  if [[ -d "$RENOS_ETC" ]]; then
    tar -czf "${BACKUP_DIR}/renos-config.tar.gz" \
      --exclude='*.local.sh' \
      -C "$(dirname "$RENOS_ETC")" "$(basename "$RENOS_ETC")" 2>/dev/null
    __manifest_add "config" "${BACKUP_DIR}/renos-config.tar.gz" "renos-config.tar.gz"
    ok "Config files backed up from $RENOS_ETC"
  else
    warn "Config path $RENOS_ETC does not exist — skipping"
  fi
}

# ── Compression of individual archives ─────────────────────────────────────────
# Files are already tar.gz; this step re-packs loose files if any.

# ── Encryption ─────────────────────────────────────────────────────────────────
encrypt_backup() {
  $ENCRYPT_ENABLED || return 0
  header "Encrypting Backup"

  if command -v age &>/dev/null && [[ -n "$AGE_PUB_KEY" ]]; then
    local enc_files=()
    while IFS= read -r -d '' f; do
      case "$f" in
        *.age|*backup-info.json) continue ;;
        *) enc_files+=("$f") ;;
      esac
    done < <(find "$BACKUP_DIR" -type f -print0)

    for f in "${enc_files[@]}"; do
      info "Encrypting: $(basename "$f")"
      age -r "$AGE_PUB_KEY" -o "${f}${ENCRYPTED_SUFFIX}" "$f" && rm -f "$f"
    done
    __manifest_finalize true
    ok "Backup encrypted with age"

  elif command -v gpg &>/dev/null && [[ -n "$GPG_RECIPIENT" ]]; then
    local enc_files=()
    while IFS= read -r -d '' f; do
      case "$f" in
        *.gpg|*backup-info.json) continue ;;
        *) enc_files+=("$f") ;;
      esac
    done < <(find "$BACKUP_DIR" -type f -print0)

    for f in "${enc_files[@]}"; do
      info "Encrypting: $(basename "$f")"
      gpg --batch --yes --recipient "$GPG_RECIPIENT" --encrypt "$f" && rm -f "$f"
    done
    __manifest_finalize true
    ok "Backup encrypted with gpg"

  else
    warn "Encryption enabled but no key/recipient configured — skipping"
    __manifest_finalize false
  fi
}

# ── Remote upload ──────────────────────────────────────────────────────────────
upload_remote() {
  $REMOTE_ENABLED || return 0
  header "Remote Upload"

  case "$REMOTE_TYPE" in
    s3)
      if command -v rclone &>/dev/null; then
        info "Uploading via rclone to ${RCLONE_REMOTE}:${REMOTE_PATH}/..."
        rclone copy "$BACKUP_DIR" "${RCLONE_REMOTE}:${REMOTE_PATH}/$(basename "$BACKUP_DIR")" \
          --progress --checksum 2>&1 | tee -a "${BACKUP_DIR}/rclone.log"
        if [[ ${PIPESTATUS[0]} -eq 0 ]]; then
          ok "Uploaded to S3 via rclone"
        else
          err "rclone upload failed — check ${BACKUP_DIR}/rclone.log"
        fi
      elif command -v aws &>/dev/null; then
        info "Uploading via aws CLI to s3://${S3_BUCKET}/${REMOTE_PATH}/..."
        aws s3 cp "$BACKUP_DIR" "s3://${S3_BUCKET}/${REMOTE_PATH}/$(basename "$BACKUP_DIR")" \
          --recursive --region "$S3_REGION" 2>&1 | tee -a "${BACKUP_DIR}/aws_s3.log"
        if [[ ${PIPESTATUS[0]} -eq 0 ]]; then
          ok "Uploaded to S3 via aws CLI"
        else
          err "aws s3 upload failed — check ${BACKUP_DIR}/aws_s3.log"
        fi
      else
        err "Neither rclone nor aws CLI found — cannot upload to S3"
      fi
      ;;
    scp)
      if [[ -n "$SCP_HOST" ]]; then
        info "Uploading via SCP to ${SCP_USER}@${SCP_HOST}:${SCP_PATH}/..."
        rsync -avz --progress "$BACKUP_DIR/" "${SCP_USER}@${SCP_HOST}:${SCP_PATH}/$(basename "$BACKUP_DIR")/" \
          2>&1 | tee -a "${BACKUP_DIR}/scp.log"
        if [[ ${PIPESTATUS[0]} -eq 0 ]]; then
          ok "Uploaded via SCP/rsync"
        else
          err "SCP upload failed — check ${BACKUP_DIR}/scp.log"
        fi
      else
        err "SCP host not configured"
      fi
      ;;
    local)
      if [[ -n "$REMOTE_PATH" ]]; then
        mkdir -p "$REMOTE_PATH"
        cp -r "$BACKUP_DIR" "$REMOTE_PATH/"
        ok "Copied to local path: $REMOTE_PATH"
      fi
      ;;
    *)
      warn "Unknown remote type: $REMOTE_TYPE"
      ;;
  esac
}

# ── Retention / Pruning ───────────────────────────────────────────────────────
prune_backups() {
  header "Pruning Old Backups"

  local all_backups
  mapfile -t all_backups < <(find "$BACKUP_ROOT" -maxdepth 1 -type d -name "${BACKUP_PREFIX}-*" -printf '%T@ %p\n' 2>/dev/null | sort -rn | awk '{print $2}')
  local count="${#all_backups[@]}"

  info "Found $count backups, retention limit: $RETENTION_COUNT"

  if [[ "$count" -le "$RETENTION_COUNT" ]]; then
    ok "No backups to prune (${count} <= ${RETENTION_COUNT})"
    return 0
  fi

  local to_delete=$(( count - RETENTION_COUNT ))
  info "Removing $to_delete old backup(s)..."

  for (( i = count - 1; i >= RETENTION_COUNT; i-- )); do
    local dir="${all_backups[$i]}"
    if [[ -d "$dir" ]]; then
      info "Removing: $dir"
      rm -rf "$dir"
      ok "Pruned: $(basename "$dir")"
    fi
  done
}

# ── List backups ───────────────────────────────────────────────────────────────
list_backups() {
  header "Available Backups in $BACKUP_ROOT"
  local found=false
  while IFS= read -r -d '' dir; do
    found=true
    local name; name=$(basename "$dir")
    local manifest_file="${dir}/backup-info.json"
    local size; size=$(du -sh "$dir" 2>/dev/null | cut -f1)
    local created; created=$(stat -c '%y' "$dir" 2>/dev/null | cut -d. -f1)
    local status="unknown"
    if [[ -f "$manifest_file" ]]; then
      status=$(python3 -c "import json; print(json.load(open('$manifest_file')).get('status','unknown'))" 2>/dev/null || grep -o '"status":"[^"]*"' "$manifest_file" | cut -d'"' -f4)
    fi
    echo -e "  ${BOLD}$name${NC}  (${size})  ${created}  [${status}]"
  done < <(find "$BACKUP_ROOT" -maxdepth 1 -type d -name "${BACKUP_PREFIX}-*" -print0 | sort -z)

  $found || echo "  No backups found."
}

# ── Verify backup ──────────────────────────────────────────────────────────────
verify_backup() {
  local backup_dir=$1
  header "Verifying Backup: $(basename "$backup_dir")"

  if [[ ! -d "$backup_dir" ]]; then
    die "Backup directory does not exist: $backup_dir"
  fi

  local manifest_file="${backup_dir}/backup-info.json"
  if [[ ! -f "$manifest_file" ]]; then
    err "No backup-info.json found — invalid backup"
    return 2
  fi

  local all_good=true
  local component
  for component in database redis vault plugins marketplace userdata history certs config; do
    local file_path=""
    local expected_hash=""
    if command -v python3 &>/dev/null; then
      file_path=$(python3 -c "
import json
with open('$manifest_file') as f: d = json.load(f)
c = d.get('components', {}).get('$component', {})
print(c.get('path', ''))
" 2>/dev/null)
      expected_hash=$(python3 -c "
import json
with open('$manifest_file') as f: d = json.load(f)
print(d.get('checksums', {}).get('$component', ''))
" 2>/dev/null)
    else
      file_path="${backup_dir}/renos-${component}.tar.gz"
      [[ "$component" == "database" ]] && file_path="${backup_dir}/renos-db.dump"
      [[ "$component" == "redis" ]] && file_path="${backup_dir}/redis-dump.rdb"
      expected_hash=$(grep -o "\"$component\":\"[^\"]*\"" "$manifest_file" | cut -d'"' -f4)
    fi

    if [[ -z "$file_path" || ! -f "$file_path" ]]; then
      if [[ "$component" == "plugins" ]] || [[ "$component" == "marketplace" ]] || [[ "$component" == "history" ]] || [[ "$component" == "vault" ]] || [[ "$component" == "certs" ]]; then
        continue  # optional components
      fi
      warn "Missing: $component (expected: $file_path)"
      all_good=false
      continue
    fi

    if [[ -n "$expected_hash" ]]; then
      local actual_hash
      actual_hash=$(sha256sum "$file_path" | cut -d' ' -f1)
      if [[ "$actual_hash" != "$expected_hash" ]]; then
        err "Checksum mismatch for $component: expected $expected_hash, got $actual_hash"
        all_good=false
        continue
      fi
    fi

    local fsize
    fsize=$(numfmt --to=iec-i "$(stat -c%s "$file_path" 2>/dev/null || echo 0)" 2>/dev/null || stat -c%s "$file_path" 2>/dev/null)
    ok "$component: $file_path ($fsize)"
  done

  if $all_good; then
    ok "Backup integrity verified successfully"
    return 0
  else
    err "Backup integrity check FAILED"
    return 2
  fi
}

# ── Main ───────────────────────────────────────────────────────────────────────

usage() {
  cat <<-EOF
${BOLD}RenOS Backup System${NC}
Usage: $(basename "$0") [OPTION]

${BOLD}Options:${NC}
  --full          Full backup (all components)
  --db-only       Database-only backup
  --files-only    Files-only backup (skip DB and Redis)
  --list          List available backups
  --verify ID     Verify backup integrity
  --prune         Prune old backups per retention policy
  --encrypt       Enable encryption (overrides config)
  --remote        Upload to remote storage
  --help          Show this help message

${BOLD}Examples:${NC}
  $(basename "$0") --full
  $(basename "$0") --db-only --encrypt --remote
  $(basename "$0") --list
  $(basename "$0") --verify renos-backup-20250101-120000
  $(basename "$0") --prune
EOF
  exit 0
}

main() {
  local mode="full"
  local verify_id=""
  local do_encrypt=$ENCRYPT_ENABLED
  local do_remote=$REMOTE_ENABLED

  [[ $# -eq 0 ]] && usage

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --full) mode="full"; shift ;;
      --db-only) mode="db-only"; shift ;;
      --files-only) mode="files-only"; shift ;;
      --list) list_backups; exit 0 ;;
      --verify) mode="verify"; verify_id="$2"; shift 2 ;;
      --prune) prune_backups; exit $? ;;
      --encrypt) do_encrypt=true; shift ;;
      --remote) do_remote=true; shift ;;
      --help|-h) usage ;;
      *) die "Unknown option: $1. Use --help for usage." ;;
    esac
  done

  case "$mode" in
    verify)
      local target="${BACKUP_ROOT}/${verify_id}"
      if [[ ! -d "$target" ]]; then
        target="$verify_id"
      fi
      verify_backup "$target"
      exit $?
      ;;
  esac

  # Root check
  if [[ $EUID -ne 0 ]]; then
    die "This script must be run as root (sudo)."
  fi

  header "RenOS Backup — $(date)"
  info "Mode: $mode"
  info "Target: $BACKUP_DIR"

  __manifest_init

  local errors=0

  case "$mode" in
    full)
      backup_database || ((errors++))
      backup_redis || ((errors++))
      backup_vault
      backup_plugins
      backup_marketplace
      backup_user_data
      backup_history
      backup_ca_certs
      backup_config || ((errors++))
      ;;
    db-only)
      BACKUP_PLUGINS=false; BACKUP_MARKETPLACE=false; BACKUP_USER_DATA=false
      BACKUP_HISTORY=false; BACKUP_CA_CERTS=false; BACKUP_CONFIG=false
      BACKUP_VAULT=false; BACKUP_REDIS=false
      backup_database || ((errors++))
      ;;
    files-only)
      BACKUP_DATABASE=false; BACKUP_REDIS=false
      backup_vault; backup_plugins; backup_marketplace
      backup_user_data; backup_history; backup_ca_certs
      backup_config || ((errors++))
      ;;
  esac

  __manifest_finalize false

  # Encrypt if requested
  if $do_encrypt; then
    ENCRYPT_ENABLED=true
    encrypt_backup
  fi

  # Remote upload
  if $do_remote; then
    upload_remote
  fi

  # Prune
  prune_backups

  header "Backup Summary"
  local total_size
  total_size=$(du -sh "$BACKUP_DIR" 2>/dev/null | cut -f1)
  if [[ $errors -eq 0 ]]; then
    ok "Backup completed successfully: $BACKUP_DIR (${total_size})"
  else
    warn "Backup completed with $errors error(s): $BACKUP_DIR (${total_size})"
  fi

  exit "$errors"
}

main "$@"
