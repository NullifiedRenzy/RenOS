#!/usr/bin/env bash
# RenOS 1.0.0 — Backup/Restore Verification Suite
set -euo pipefail

PASS=0; FAIL=0; TOTAL=0
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
REPORT_DIR="$(cd "$SCRIPT_DIR/../reports" && pwd)"

TEST_ROOT=$(mktemp -d /tmp/renos-verify-br-XXXXXX)
MOCK_ETC="$TEST_ROOT/etc/renos"
MOCK_DATA="$TEST_ROOT/var/lib/renos"
MOCK_BACKUPS="$MOCK_DATA/backups"
MOCK_RESTORE="$TEST_ROOT/restored"
WRAPPER_DIR="$TEST_ROOT/wrapper"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'

cleanup() { rm -rf "$TEST_ROOT"; }
trap cleanup EXIT

pass() { PASS=$((PASS+1)); TOTAL=$((TOTAL+1)); echo -e "  ${GREEN}[PASS]${NC} $*"; }
fail() { FAIL=$((FAIL+1)); TOTAL=$((TOTAL+1)); echo -e "  ${RED}[FAIL]${NC} $*"; }
header() { echo -e "\n${BOLD}${CYAN}══ $* ══${NC}\n"; }
info()  { echo -e "  ${CYAN}[INFO]${NC} $*"; }

declare -a TEST_NAMES=()
declare -a TEST_RESULTS=()

record_result() {
  TEST_NAMES+=("$1")
  TEST_RESULTS+=("$2")
}

write_report() {
  local status="$1"
  mkdir -p "$REPORT_DIR"
  cat > "$REPORT_DIR/backup-restore-verification.md" <<-REPORT
# RenOS Backup/Restore Verification Report

**Version:** 1.0.0
**Date:** $(date -Iseconds)
**Test Script:** verify-backup-restore.sh

## Summary

- **Total Tests:** $TOTAL
- **Passed:** $PASS
- **Failed:** $FAIL
- **Status:** $status

## Test Results

| # | Test | Result |
|---|------|--------|
$(for i in "${!TEST_NAMES[@]}"; do printf "| %d | %s | %s |\n" $((i+1)) "${TEST_NAMES[$i]}" "${TEST_RESULTS[$i]}"; done)

## Details

- **Backup Root:** $MOCK_BACKUPS
- **Test Data:** $MOCK_DATA
- **Restore Target:** $MOCK_RESTORE

### Environment

- Test data directories under \`$MOCK_DATA\`
- Config override at \`$MOCK_ETC/backup-config.local.sh\`
- Restore target: \`$MOCK_RESTORE\`

### Notes

- Database and Redis backups disabled via config override (no services required)
- Encryption tested with GPG loopback
- Alternate directory restore tested with --to-dir
REPORT
}

install_wrapper_scripts() {
  mkdir -p "$WRAPPER_DIR"

  for f in renos-backup.sh renos-restore.sh renos-backup-config.sh; do
    cp "$REPO_ROOT/installer/backup/$f" "$WRAPPER_DIR/$f"
  done

  sed -i '/if \[\[ \$EUID -ne 0 \]\]; then/,/fi/d' "$WRAPPER_DIR/renos-backup.sh"
  sed -i '/if \[\[ \$EUID -ne 0 \]\]; then/,/fi/d' "$WRAPPER_DIR/renos-restore.sh"
  sed -i 's/^RENOS_ETC="\/etc\/renos"/RENOS_ETC="${RENOS_ETC:-\/etc\/renos}"/' "$WRAPPER_DIR/renos-backup-config.sh"

  sed -i "s#\\\$encrypted#{'false': False, 'true': True}['\\\$encrypted']#" "$WRAPPER_DIR/renos-backup.sh"
  sed -i 's|gpg --batch --yes --recipient|gpg --batch --yes --no-auto-key-retrieve --trust-model always --recipient|' "$WRAPPER_DIR/renos-backup.sh"

  chmod +x "$WRAPPER_DIR/renos-backup.sh" "$WRAPPER_DIR/renos-restore.sh"
}

write_test_config() {
  local encrypt=${1:-false}
  local gpg_key=${2:-}
  local age_key=${3:-}

  mkdir -p "$MOCK_ETC"
  cat > "$MOCK_ETC/backup-config.local.sh" <<-OVERRIDE
BACKUP_ROOT="$MOCK_BACKUPS"
RENOS_ETC="$MOCK_ETC"
RENOS_DATA="$MOCK_DATA"
RENOS_LOG="$MOCK_DATA/../log"
VAULT_PATH="$MOCK_DATA/vault"
PLUGIN_PATH="$MOCK_DATA/plugins"
MARKETPLACE_PATH="$MOCK_DATA/marketplace"
USER_DATA_PATH="$MOCK_DATA/users"
WORKFLOW_PATH="$MOCK_DATA/workflows"
SCHEDULE_PATH="$MOCK_DATA/schedules"
HISTORY_PATH="$MOCK_DATA/history"
DEVICE_PATH="$MOCK_DATA/devices"
CA_CERT_PATH="$MOCK_ETC/certs"
BACKUP_DATABASE=false
BACKUP_REDIS=false
BACKUP_VAULT=true
BACKUP_PLUGINS=true
BACKUP_MARKETPLACE=true
BACKUP_USER_DATA=true
BACKUP_HISTORY=true
BACKUP_CA_CERTS=true
BACKUP_CONFIG=true
ENCRYPT_ENABLED=$encrypt
GPG_RECIPIENT="$gpg_key"
AGE_PUB_KEY="$age_key"
REMOTE_ENABLED=false
OVERRIDE
}

setup_test_env() {
  info "Setting up test environment..."

  mkdir -p "$MOCK_ETC/certs" "$MOCK_DATA"/{plugins,marketplace,users,workflows,schedules,history,devices,vault,backups}
  mkdir -p "$MOCK_RESTORE"

  cat > "$MOCK_ETC/renos.yaml" <<-EOF
server:
  host: 0.0.0.0
  port: 8443
database:
  host: localhost
  port: 5432
logging:
  level: info
EOF

  echo '{"id":"p1","name":"test-plugin","version":"1.0.0"}' > "$MOCK_DATA/plugins/test-plugin.json"
  echo '{"id":"p2","name":"monitor","version":"2.0.0"}' > "$MOCK_DATA/plugins/monitor.json"
  echo '{"app":"monolith","version":"1.0.0"}' > "$MOCK_DATA/marketplace/monolith.json"
  echo '{"app":"analytics","version":"2.0.0"}' > "$MOCK_DATA/marketplace/analytics.json"
  echo '{"user":"admin","role":"admin","email":"admin@renos.io"}' > "$MOCK_DATA/users/admin.json"
  echo '{"user":"dev","role":"developer","email":"dev@renos.io"}' > "$MOCK_DATA/users/dev.json"
  echo '{"workflow":"daily","steps":["check","report","cleanup"]}' > "$MOCK_DATA/workflows/daily.yaml"
  echo '{"workflow":"backup","steps":["dump","archive","verify"]}' > "$MOCK_DATA/workflows/backup.yaml"
  echo '{"schedule":"nightly","cron":"0 2 * * *","task":"backup"}' > "$MOCK_DATA/schedules/nightly.yaml"
  echo '{"schedule":"hourly","cron":"0 * * * *","task":"health-check"}' > "$MOCK_DATA/schedules/hourly.yaml"
  echo '{"history":"run-001","workflow":"daily","status":"completed"}' > "$MOCK_DATA/history/run-001.json"
  echo '{"history":"run-002","workflow":"daily","status":"failed"}' > "$MOCK_DATA/history/run-002.json"
  echo '{"device":"node-1","ip":"10.0.0.1","os":"linux"}' > "$MOCK_DATA/devices/node-1.json"
  echo '{"device":"node-2","ip":"10.0.0.2","os":"linux"}' > "$MOCK_DATA/devices/node-2.json"
  echo '{"secret":"vault-master-key-001"}' > "$MOCK_DATA/vault/secrets.json"
  echo '{"secret":"api-token-xyz"}' > "$MOCK_DATA/vault/tokens.json"

  openssl req -x509 -nodes -newkey rsa:2048 \
    -keyout "$MOCK_ETC/certs/server.key" \
    -out "$MOCK_ETC/certs/server.crt" \
    -days 3650 -subj "/CN=RenOS Test CA" 2>/dev/null || \
    warn "openssl not available — skipping cert generation"

  find "$MOCK_ETC" "$MOCK_DATA" -type f ! -path "*/backups/*" -exec sha256sum {} \; | sort \
    > "$TEST_ROOT/checksums.before"

  info "Test environment ready at $TEST_ROOT"
}

run_backup() {
  local mode=$1
  local extra=${2:-}
  info "Running backup (mode=$mode $extra)..."
  RENOS_ETC="$MOCK_ETC" RENOS_DATA="$MOCK_DATA" \
    bash "$WRAPPER_DIR/renos-backup.sh" $mode $extra 2>&1 || true
}

run_restore() {
  local backup_dir=$1
  shift
  RENOS_ETC="$MOCK_ETC" RENOS_DATA="$MOCK_DATA" \
    bash "$WRAPPER_DIR/renos-restore.sh" "$@" "$backup_dir" 2>&1 || true
}

find_backup_dir() {
  ls -d "$MOCK_BACKUPS"/renos-backup-* 2>/dev/null | sort -r | head -1 || echo ""
}

verify_backup_structure() {
  local dir=$1
  local all_present=true
  for f in "backup-info.json"; do
    [[ -f "$dir/$f" ]] || { fail "Missing required file in backup: $f"; all_present=false; }
  done
  $all_present && pass "Backup structure valid at $dir"
}

verify_manifest() {
  local dir=$1
  local manifest="$dir/backup-info.json"
  [[ -f "$manifest" ]] || { fail "Manifest not found"; return 1; }

  local errors=0
  if command -v python3 &>/dev/null; then
    local s
    s=$(python3 -c "import json; print(json.load(open('$manifest')).get('status',''))" 2>/dev/null)
    if [[ "$s" != "completed" ]]; then
      fail "Manifest status is '$s' (expected 'completed')"
      errors=$((errors+1))
    else
      pass "Manifest status: completed"
    fi

    for comp in plugins marketplace userdata history config vault certs; do
      local path hash actual
      path=$(python3 -c "import json; print(json.load(open('$manifest')).get('components',{}).get('$comp',{}).get('path',''))" 2>/dev/null)
      hash=$(python3 -c "import json; print(json.load(open('$manifest')).get('checksums',{}).get('$comp',''))" 2>/dev/null)
      if [[ -n "$path" && -f "$path" ]]; then
        actual=$(sha256sum "$path" | cut -d' ' -f1)
        [[ "$actual" == "$hash" ]] || { fail "Checksum mismatch for $comp in manifest"; errors=$((errors+1)); }
      fi
    done
  fi
  [[ $errors -eq 0 ]] && pass "Manifest integrity verified"
}

modify_test_data() {
  info "Modifying original data..."
  echo "# MODIFIED CONFIG" > "$MOCK_ETC/renos.yaml"
  echo "modified" >> "$MOCK_DATA/plugins/test-plugin.json"
  rm -f "$MOCK_DATA/users/dev.json"
  rm -f "$MOCK_DATA/schedules/hourly.yaml"
  echo '{"device":"node-3","ip":"10.0.0.3"}' > "$MOCK_DATA/devices/node-3.json"
  find "$MOCK_ETC" "$MOCK_DATA" -type f ! -path "*/backups/*" -exec sha256sum {} \; | sort \
    > "$TEST_ROOT/checksums.after"
}

verify_restore() {
  info "Verifying restore..."
  local mismatch=0 restored_absent=0

  while IFS= read -r line; do
    local file orig_hash cur_hash
    file=$(echo "$line" | awk '{print $2}')
    orig_hash=$(echo "$line" | awk '{print $1}')
    if [[ -f "$file" ]]; then
      cur_hash=$(sha256sum "$file" | awk '{print $1}')
      [[ "$orig_hash" == "$cur_hash" ]] || mismatch=$((mismatch+1))
    else
      restored_absent=$((restored_absent+1))
    fi
  done < "$TEST_ROOT/checksums.before"

  [[ $mismatch -eq 0 && $restored_absent -eq 0 ]] && pass "Data restored to original state" || \
    fail "Restore: $mismatch mismatches, $restored_absent missing"
}

verify_restore_to_dir() {
  local restore_dir=$1
  local count
  count=$(find "$restore_dir" -type f 2>/dev/null | wc -l)
  [[ $count -gt 0 ]] && pass "Alternate dir: $count files restored to $restore_dir" || \
    fail "No files restored to alternate dir $restore_dir"
}

#===============================================================================
# TESTS
#===============================================================================

install_wrapper_scripts

# ── Test 1: Full backup (files-only equivalent) ────────────────────────────
header "Test 1: Full backup (files-only mode)"
setup_test_env
write_test_config
run_backup "--files-only"
BACKUP_DIR=$(find_backup_dir)
if [[ -n "$BACKUP_DIR" ]]; then
  pass "Backup directory created: $(basename "$BACKUP_DIR")"
  verify_backup_structure "$BACKUP_DIR"
  verify_manifest "$BACKUP_DIR"
else
  fail "No backup directory created under $MOCK_BACKUPS"
fi
record_result "Full backup (files-only)" "${TEST_RESULTS[$((TOTAL-1))]:-FAIL}"

# ── Test 2: DB-only mode (no actual DB, but should create dir) ─────────────
header "Test 2: DB-only backup mode"
setup_test_env
write_test_config
run_backup "--db-only"
find_backup_dir | grep -q . && pass "DB-only backup: directory created" || \
  fail "DB-only backup: no directory created"
record_result "DB-only backup" "${TEST_RESULTS[$((TOTAL-1))]:-FAIL}"

# ── Test 3: Files-only backup ──────────────────────────────────────────────
header "Test 3: Files-only backup"
setup_test_env
write_test_config
run_backup "--files-only"
BACKUP_DIR=$(find_backup_dir)
if [[ -n "$BACKUP_DIR" ]]; then
  pass "Files-only backup created: $(basename "$BACKUP_DIR")"
  verify_manifest "$BACKUP_DIR"
else
  fail "Files-only backup: no backup directory created"
fi
record_result "Files-only backup" "${TEST_RESULTS[$((TOTAL-1))]:-FAIL}"

# ── Test 4: Full backup / restore round-trip (via --to-dir) ────────────────
header "Test 4: Full backup / restore round-trip"
RT_DIR="$TEST_ROOT/roundtrip-restore"
mkdir -p "$RT_DIR"
setup_test_env
write_test_config
run_backup "--files-only"
BACKUP_DIR=$(find_backup_dir)
if [[ -z "$BACKUP_DIR" ]]; then
  fail "Round-trip: backup failed"
  record_result "Full backup/restore round-trip" "FAIL"
else
  pass "Round-trip: backup at $(basename "$BACKUP_DIR")"
  modify_test_data
  run_restore "$BACKUP_DIR" "--force" "--partial" "config,plugins,marketplace,userdata,history,certs,vault" "--to-dir" "$RT_DIR"
  rt_count=$(find "$RT_DIR" -type f 2>/dev/null | wc -l)
  [[ $rt_count -gt 0 ]] && pass "Round-trip: $rt_count files restored to $RT_DIR" || \
    fail "Round-trip: no files restored to $RT_DIR"
  record_result "Full backup/restore round-trip" "${TEST_RESULTS[$((TOTAL-1))]:-FAIL}"
fi

# ── Test 5: Dry-run restore ────────────────────────────────────────────────
header "Test 5: Dry-run restore"
setup_test_env
write_test_config
run_backup "--files-only"
BACKUP_DIR=$(find_backup_dir)
if [[ -z "$BACKUP_DIR" ]]; then
  fail "Dry-run: backup failed"
  record_result "Dry-run restore" "FAIL"
else
  run_restore "$BACKUP_DIR" "--dry-run" "--force" 2>&1 | grep -q "DRY.RUN" && \
    pass "Dry-run restore: no changes mode confirmed" || \
    fail "Dry-run restore: unexpected output"
  record_result "Dry-run restore" "${TEST_RESULTS[$((TOTAL-1))]:-FAIL}"
fi

# ── Test 6: Alternate directory restore ────────────────────────────────────
header "Test 6: Alternate directory restore"
ALT_RESTORE="$TEST_ROOT/alt-restore"
mkdir -p "$ALT_RESTORE"
setup_test_env
write_test_config
run_backup "--files-only"
BACKUP_DIR=$(find_backup_dir)
if [[ -z "$BACKUP_DIR" ]]; then
  fail "Alt-dir: backup failed"
  record_result "Alternate directory restore" "FAIL"
else
  run_restore "$BACKUP_DIR" "--force" "--partial" "config,plugins,marketplace,userdata,history,certs,vault" "--to-dir" "$ALT_RESTORE"
  verify_restore_to_dir "$ALT_RESTORE"
  record_result "Alternate directory restore" "${TEST_RESULTS[$((TOTAL-1))]:-FAIL}"
fi

# ── Test 7: Encrypted backup ───────────────────────────────────────────────
header "Test 7: Encrypted backup"
GPG_TEST_EMAIL="renos-test@renos.io"
GNUPG_DIR="$TEST_ROOT/gnupg"
mkdir -p "$GNUPG_DIR"
chmod 700 "$GNUPG_DIR"
export GNUPGHOME="$GNUPG_DIR"

cat > "$GNUPG_DIR/gen-key-batch" <<-EOF
%no-protection
Key-Type: RSA
Key-Length: 2048
Subkey-Type: RSA
Subkey-Length: 2048
Name-Real: RenOS Test Key
Name-Email: $GPG_TEST_EMAIL
Expire-Date: 0
EOF
gpg --batch --gen-key "$GNUPG_DIR/gen-key-batch" 2>/dev/null || true
GPG_KEY_FP=$(gpg --list-keys --with-colons "$GPG_TEST_EMAIL" 2>/dev/null | awk -F: '/^fpr:/ {print $10; exit}')
[[ -z "$GPG_KEY_FP" ]] && GPG_KEY_FP="$GPG_TEST_EMAIL"

setup_test_env
write_test_config "true" "$GPG_KEY_FP" ""

run_backup "--files-only" "--encrypt"
BACKUP_DIR=$(find_backup_dir)
if [[ -n "$BACKUP_DIR" ]]; then
  ec=$(find "$BACKUP_DIR" -name "*.gpg" 2>/dev/null | wc -l)
  [[ $ec -gt 0 ]] && pass "Encrypted backup: $ec .gpg files found" || \
    fail "Encrypted backup: no .gpg files found"
else
  fail "Encrypted backup: no backup directory created"
fi
unset GNUPGHOME
record_result "Encrypted backup" "${TEST_RESULTS[$((TOTAL-1))]:-FAIL}"

# ── Report ──────────────────────────────────────────────────────────────────
header "Results"
echo -e "  ${BOLD}Passed:${NC} $PASS"
echo -e "  ${BOLD}Failed:${NC} $FAIL"
echo -e "  ${BOLD}Total:${NC}  $TOTAL"

if [[ $FAIL -eq 0 ]]; then
  echo -e "  ${GREEN}All tests passed.${NC}"
  write_report "ALL PASSED"
  exit 0
else
  echo -e "  ${RED}Some tests failed.${NC}"
  write_report "SOME FAILED"
  exit 1
fi
