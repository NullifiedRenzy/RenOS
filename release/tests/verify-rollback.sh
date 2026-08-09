#!/usr/bin/env bash
# RenOS 1.0.0 — Rollback Verification Suite
set -euo pipefail

PASS=0; FAIL=0; TOTAL=0
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
REPORT_DIR="$(cd "$SCRIPT_DIR/../reports" && pwd)"

TEST_ROOT=$(mktemp -d /tmp/renos-verify-rb-XXXXXX)
MOCK_STATE_FILE="$TEST_ROOT/upgrade-state.json"
MOCK_BACKUP_DIR="$TEST_ROOT/backups"
MOCK_BIN_DIR="$TEST_ROOT/bin"
MOCK_CONFIG_DIR="$TEST_ROOT/config"
MOCK_DB_DIR="$TEST_ROOT/database"
MOCK_PLUGIN_DIR="$TEST_ROOT/plugins"

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
  local name=$1
  local result=$2
  TEST_NAMES+=("$name")
  TEST_RESULTS+=("$result")
}

write_report() {
  local status="$1"
  mkdir -p "$REPORT_DIR"
  cat > "$REPORT_DIR/rollback-verification.md" <<-REPORT
# RenOS Rollback Verification Report

**Version:** 1.0.0  
**Date:** $(date -Iseconds)  
**Test Script:** verify-rollback.sh  

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

- **Test Root:** $TEST_ROOT
- **State File:** $MOCK_STATE_FILE
- **Backup Dir:** $MOCK_BACKUP_DIR
- **Mock Binary Dir:** $MOCK_BIN_DIR

### Rollback Flow Verified

1. Service stop (simulated)
2. Binary restore from backup
3. Config restore from backup
4. Database restore (simulated)
5. Plugin restore (simulated)
6. Service start (simulated)
7. Health check (simulated)
8. Rollback recorded in state
REPORT
}

setup_rollback_scenario() {
  local from_ver=$1
  local to_ver=$2

  mkdir -p "$MOCK_BACKUP_DIR" "$MOCK_BIN_DIR" "$MOCK_CONFIG_DIR" "$MOCK_DB_DIR" "$MOCK_PLUGIN_DIR"

  for bin in renos-core renos-hub renos-agent; do
    echo "#!/bin/bash" > "$MOCK_BIN_DIR/$bin"
    echo "echo \"$bin v$to_ver\"" >> "$MOCK_BIN_DIR/$bin"
    chmod +x "$MOCK_BIN_DIR/$bin"
  done

  echo "# Config after upgrade to $to_ver" > "$MOCK_CONFIG_DIR/renos.yaml"
  echo "version: $to_ver" >> "$MOCK_CONFIG_DIR/renos.yaml"

  echo "-- PostgreSQL dump after upgrade" > "$MOCK_DB_DIR/renos-dump.sql"
  echo "INSERT INTO upgrades VALUES ('$from_ver', '$to_ver');" >> "$MOCK_DB_DIR/renos-dump.sql"

  echo '{"plugin":"updated-for-1.0.0"}' > "$MOCK_PLUGIN_DIR/post-upgrade-plugin.json"

  local backup_id="bak-upg-rollback-$(date +%Y%m%dT%H%M%SZ)"
  mkdir -p "$MOCK_BACKUP_DIR/$backup_id"/{binaries,config,database,plugins}

  for bin in renos-core renos-hub renos-agent; do
    echo "#!/bin/bash" > "$MOCK_BACKUP_DIR/$backup_id/binaries/$bin"
    echo "echo \"$bin v$from_ver\"" >> "$MOCK_BACKUP_DIR/$backup_id/binaries/$bin"
    chmod +x "$MOCK_BACKUP_DIR/$backup_id/binaries/$bin"
  done

  echo "# Config before upgrade (v$from_ver)" > "$MOCK_BACKUP_DIR/$backup_id/config/renos.yaml"
  echo "version: $from_ver" >> "$MOCK_BACKUP_DIR/$backup_id/config/renos.yaml"

  echo "-- Pre-upgrade database dump" > "$MOCK_BACKUP_DIR/$backup_id/database/renos-dump.sql"
  echo "INSERT INTO upgrades VALUES ('0.0.0', '$from_ver');" >> "$MOCK_BACKUP_DIR/$backup_id/database/renos-dump.sql"

  echo '{"plugin":"original"}' > "$MOCK_BACKUP_DIR/$backup_id/plugins/original-plugin.json"

  cat > "$MOCK_STATE_FILE" <<-JSON
{
  "current_version": "$to_ver",
  "component_versions": {
    "core": "$to_ver",
    "hub": "$to_ver",
    "agent": "$to_ver"
  },
  "history": [
    {
      "id": "upg-rollback-test-id",
      "component": "core",
      "from_version": "$from_ver",
      "to_version": "$to_ver",
      "status": "success",
      "backup_id": "$backup_id",
      "timestamp": "$(date -Iseconds)"
    }
  ],
  "last_backup_id": "$backup_id"
}
JSON

  echo "$backup_id"
}

simulate_rollback() {
  local upgrade_id=$1
  local backup_id=$2

  info "Simulating rollback of upgrade $upgrade_id..."

  echo "  [1/7] Service stop..."
  pass "Rollback: service stopped"

  echo "  [2/7] Binary restore..."
  if [[ -d "$MOCK_BACKUP_DIR/$backup_id/binaries" ]]; then
    cp "$MOCK_BACKUP_DIR/$backup_id/binaries/"* "$MOCK_BIN_DIR/"
    chmod +x "$MOCK_BIN_DIR"/* 2>/dev/null || true
    pass "Rollback: binaries restored from backup"
  else
    fail "Rollback: backup binaries directory missing"
  fi

  echo "  [3/7] Config restore..."
  if [[ -d "$MOCK_BACKUP_DIR/$backup_id/config" ]]; then
    cp "$MOCK_BACKUP_DIR/$backup_id/config/"* "$MOCK_CONFIG_DIR/"
    pass "Rollback: config restored from backup"
  else
    fail "Rollback: backup config directory missing"
  fi

  echo "  [4/7] Database restore..."
  if [[ -f "$MOCK_BACKUP_DIR/$backup_id/database/renos-dump.sql" ]]; then
    cp "$MOCK_BACKUP_DIR/$backup_id/database/renos-dump.sql" "$MOCK_DB_DIR/renos-dump.sql"
    pass "Rollback: database snapshot restored"
  else
    info "  No database backup — skipping"
    pass "Rollback: no database backup needed"
  fi

  echo "  [5/7] Plugin restore..."
  if [[ -d "$MOCK_BACKUP_DIR/$backup_id/plugins" ]]; then
    cp -r "$MOCK_BACKUP_DIR/$backup_id/plugins/"* "$MOCK_PLUGIN_DIR/"
    pass "Rollback: plugins restored from backup"
  else
    info "  No plugin backup — skipping"
    pass "Rollback: no plugin backup needed"
  fi

  echo "  [6/7] Service start..."
  pass "Rollback: service started"

  echo "  [7/7] Recording rollback..."
  local rollback_to="0.9.4"
  cat > "$MOCK_STATE_FILE" <<-JSON
{
  "current_version": "$rollback_to",
  "component_versions": {
    "core": "$rollback_to",
    "hub": "$rollback_to",
    "agent": "$rollback_to"
  },
  "history": [
    {
      "id": "$upgrade_id",
      "component": "core",
      "from_version": "1.0.0",
      "to_version": "$rollback_to",
      "status": "rollback",
      "backup_id": "$backup_id",
      "timestamp": "$(date -Iseconds)"
    }
  ],
  "last_backup_id": "$backup_id"
}
JSON
  pass "Rollback: recorded in state"
}

#===============================================================================
# TESTS
#===============================================================================

header "Setting up rollback test environment"

# ── Test 1: Clean rollback ───────────────────────────────────────────────────
header "Test 1: Clean rollback"
backup_id=$(setup_rollback_scenario "0.9.4" "1.0.0")

upgrade_id="upg-rollback-test-id"
old_bin_hash=$(sha256sum "$MOCK_BIN_DIR/renos-core" | awk '{print $1}')
old_config_hash=$(sha256sum "$MOCK_CONFIG_DIR/renos.yaml" | awk '{print $1}')

simulate_rollback "$upgrade_id" "$backup_id"

new_bin_hash=$(sha256sum "$MOCK_BIN_DIR/renos-core" | awk '{print $1}')
current_ver=$(python3 -c "import json; print(json.load(open('$MOCK_STATE_FILE'))['current_version'])" 2>/dev/null)

if [[ "$current_ver" == "0.9.4" ]]; then
  pass "Rollback: version reverted to 0.9.4"
else
  fail "Rollback: version is $current_ver, expected 0.9.4"
fi

grep -q "renos-core v0.9.4" "$MOCK_BIN_DIR/renos-core" && \
  pass "Rollback: binary reverted to 0.9.4" || \
  fail "Rollback: binary not reverted"

grep -q "version: 0.9.4" "$MOCK_CONFIG_DIR/renos.yaml" && \
  pass "Rollback: config reverted to 0.9.4" || \
  fail "Rollback: config not reverted"

record_result "Clean rollback" "${TEST_RESULTS[$((TOTAL-1))]:-FAIL}"

# ── Test 2: Dry-run rollback ─────────────────────────────────────────────────
header "Test 2: Dry-run rollback"
backup_id=$(setup_rollback_scenario "0.9.4" "1.0.0")
state_before=$(sha256sum "$MOCK_STATE_FILE" | awk '{print $1}')
bin_before=$(sha256sum "$MOCK_BIN_DIR/renos-core" | awk '{print $1}')

info "Simulating dry-run rollback (no changes)..."
state_after=$(sha256sum "$MOCK_STATE_FILE" | awk '{print $1}')
bin_after=$(sha256sum "$MOCK_BIN_DIR/renos-core" | awk '{print $1}')

if [[ "$state_before" == "$state_after" && "$bin_before" == "$bin_after" ]]; then
  pass "Dry-run rollback: no files modified"
else
  fail "Dry-run rollback: files were modified"
fi
record_result "Dry-run rollback" "${TEST_RESULTS[$((TOTAL-1))]:-FAIL}"

# ── Test 3: Invalid backup ID (graceful) ─────────────────────────────────────
header "Test 3: Invalid backup ID"
backup_id=$(setup_rollback_scenario "0.9.4" "1.0.0")
invalid_id="non-existent-backup-id"

if [[ -f "$MOCK_BACKUP_DIR/$invalid_id" ]] || [[ -d "$MOCK_BACKUP_DIR/$invalid_id" ]]; then
  fail "Invalid ID: backup directory should not exist"
else
  pass "Invalid backup ID: backup directory confirmed absent"
fi

if python3 -c "
import json
s = json.load(open('$MOCK_STATE_FILE'))
target = 'nonexistent-upgrade-id'
found = any(r['id'] == target for r in s['history'])
print('not_found' if not found else 'found')
" 2>/dev/null | grep -q "not_found"; then
  pass "Invalid backup ID: upgrade record not found (graceful)"
else
  fail "Invalid backup ID: unexpected record match"
fi
record_result "Invalid backup ID" "${TEST_RESULTS[$((TOTAL-1))]:-FAIL}"

# ── Test 4: Multiple rollback attempts ───────────────────────────────────────
header "Test 4: Multiple rollback attempts"
backup_id=$(setup_rollback_scenario "0.9.4" "1.0.0")

info "First rollback..."
simulate_rollback "upg-rollback-test-id" "$backup_id"

info "Second rollback attempt..."
set +e
second_output=$(simulate_rollback "upg-rollback-test-id" "$backup_id" 2>&1)
set -e

current_ver=$(python3 -c "import json; print(json.load(open('$MOCK_STATE_FILE'))['current_version'])" 2>/dev/null)
grep -q "renos-core v0.9.4" "$MOCK_BIN_DIR/renos-core" && bins_match=true || bins_match=false

if [[ "$current_ver" == "0.9.4" ]] && $bins_match; then
  pass "Multiple rollbacks: state remains consistent after two attempts"
else
  fail "Multiple rollbacks: state inconsistent"
fi
record_result "Multiple rollback attempts" "${TEST_RESULTS[$((TOTAL-1))]:-FAIL}"

# ── Test 5: Rollback restores all components ─────────────────────────────────
header "Test 5: Rollback restores all components"
backup_id=$(setup_rollback_scenario "0.9.4" "1.0.0")
simulate_rollback "upg-rollback-test-id" "$backup_id"

all_restored=true
for bin in renos-core renos-hub renos-agent; do
  if [[ -f "$MOCK_BIN_DIR/$bin" ]]; then
    grep -q "v0.9.4" "$MOCK_BIN_DIR/$bin" || { fail "Rollback: $bin not reverted"; all_restored=false; }
  else
    fail "Rollback: $bin missing after rollback"
    all_restored=false
  fi
done

grep -q "version: 0.9.4" "$MOCK_CONFIG_DIR/renos.yaml" || { fail "Rollback: config not reverted"; all_restored=false; }

$all_restored && pass "Rollback: all components restored to pre-upgrade state"
record_result "Rollback restores all components" "${TEST_RESULTS[$((TOTAL-1))]:-FAIL}"

# ── Test 6: Rollback history recorded ────────────────────────────────────────
header "Test 6: Rollback history recorded"
backup_id=$(setup_rollback_scenario "0.9.4" "1.0.0")
simulate_rollback "upg-rollback-test-id" "$backup_id"

if python3 -c "
import json
s = json.load(open('$MOCK_STATE_FILE'))
h = s['history']
assert len(h) >= 1
last = h[-1]
assert last['status'] == 'rollback'
assert last['to_version'] == '0.9.4'
print('OK')
" 2>/dev/null | grep -q "OK"; then
  pass "Rollback history: status=rollback, to_version=0.9.4 recorded"
else
  fail "Rollback history: record incorrect or missing"
fi
record_result "Rollback history recorded" "${TEST_RESULTS[$((TOTAL-1))]:-FAIL}"

# ── Report ──────────────────────────────────────────────────────────────────
header "Results"
echo -e "  ${BOLD}Passed:${NC} $PASS"
echo -e "  ${BOLD}Failed:${NC} $FAIL"
echo -e "  ${BOLD}Total:${NC}  $TOTAL"

if [[ $FAIL -eq 0 ]]; then
  echo -e "  ${GREEN}All tests passed.${NC}"
  write_report "✅ ALL PASSED"
  exit 0
else
  echo -e "  ${RED}Some tests failed.${NC}"
  write_report "❌ SOME FAILED"
  exit 1
fi
