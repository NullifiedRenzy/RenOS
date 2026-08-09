#!/usr/bin/env bash
# lib.sh — shared library for the RenOS end-to-end QA suite.
# Sourced by run-e2e.sh and all phase scripts.
#
# Policies:
#   * Every phase records exactly one step outcome: PASS / FAIL / SKIP.
#   * A phase may SKIP when its service is unavailable; it never fakes PASS.
#   * All endpoint paths used here are real routes in core/internal/api/router.go.
#   * API base URL is $RENOS_URL (default http://localhost:8443).
#   * Auth: user-facing requests carry `Authorization: Bearer $AUTH_TOKEN` once
#     login succeeds; agent-facing requests (device_id/secret bodies) work via
#     the CSRF exempt paths defined by isDeviceAuthRequest.

set -uo pipefail

E2E_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$E2E_ROOT/../../.." && pwd)"

# ---------------------------------------------------------------------------
# Output helpers
# ---------------------------------------------------------------------------
GREEN=$'\033[0;32m'; RED=$'\033[0;31m'; YELLOW=$'\033[0;33m'; CYAN=$'\033[0;36m'; BOLD=$'\033[1m'; NC=$'\033[0m'

_info()    { printf '%b[..]%b %s\n' "$CYAN" "$NC" "$*"; }
_note()    { printf '%b[--]%b %s\n' "$YELLOW" "$NC" "$*"; }
_ok()      { printf '%b[OK]%b %s\n' "$GREEN" "$NC" "$*"; }
_err()     { printf '%b[ERR]%b %s\n' "$RED" "$NC" "$*" >&2; }
_banner()  { printf '%b%s%b\n' "$BOLD" "$*" "$NC"; }

# ---------------------------------------------------------------------------
# Global state
# ---------------------------------------------------------------------------
CURRENT_STEP=""
CORE_UP=false
AUTH_TOKEN=""
API_STATUS=""
API_BODY=""
OUT=""
RC=0
ASSERT_PASS=0
ASSERT_FAIL=0
PHASE_BASE_FAIL=0
RESULTS_PHASE=()
RESULTS_STATUS=()
RESULTS_MSG=()
RESULTS_FILE=""
E2E_TMP=""

E2E_DEVICE_ID=""
E2E_DEVICE_SECRET=""
E2E_COMMAND_ID=""
E2E_WORKFLOW_ID=""
E2E_BACKUP_DIR=""

# ---------------------------------------------------------------------------
# Environment / defaults
# ---------------------------------------------------------------------------
RENOS_URL="${RENOS_URL:-http://localhost:8443}"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@renos.local}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin}"
HEALTH_TIMEOUT="${HEALTH_TIMEOUT:-90}"
E2E_COMMAND="${E2E_COMMAND:-apt update}"
E2E_RESULTS_DIR="${E2E_RESULTS_DIR:-$E2E_ROOT/results}"
SKIP_INSTALL="${SKIP_INSTALL:-false}"
QUICK="${QUICK:-false}"
SETUP_ONLY="${SETUP_ONLY:-false}"
CLEANUP="${CLEANUP:-false}"
REPORT_ARG=""

load_env() {
  if [[ -f "$REPO_ROOT/.env" ]]; then
    set -a
    # shellcheck disable=SC1091
    source "$REPO_ROOT/.env" 2>/dev/null || true
    set +a
  fi
}

# ---------------------------------------------------------------------------
# Utils
# ---------------------------------------------------------------------------
random_name() {
  local prefix="${1:-renos-e2e}"
  printf '%s-%s-%s-%s' "$prefix" "$(date +%s)" "$$" "$RANDOM"
}

# run_cmd <cmd...> — run a command, capturing combined output in $OUT and the
# exit status in $RC (global). Never aborts on failure.
run_cmd() {
  OUT="$( "$@" 2>&1 )"
  RC=$?
}

# latest_completed_backup [ROOT] — print the newest backup dir under ROOT whose
# backup-info.json reports status=completed. Prints nothing and returns 1 if
# no completed backup is found.
latest_completed_backup() {
  local root="${1:-/var/lib/renos/backups}" d latest=""
  [[ -d "$root" ]] || return 1
  for d in "$root"/*/; do
    [[ -d "$d" ]] || continue
    [[ -f "$d/backup-info.json" ]] || continue
    if [[ "$(json_get "$(cat "$d/backup-info.json")" status)" == "completed" ]]; then
      if [[ -z "$latest" || "$d" > "$latest" ]]; then
        latest="$d"
      fi
    fi
  done
  [[ -n "$latest" ]] || return 1
  printf '%s' "${latest%/}"
}

json_str() {
  python3 -c 'import json,sys; print(json.dumps(sys.argv[1]))' "$1" 2>/dev/null || printf '"%s"' "$1"
}

json_get() {
  python3 -c '
import sys, json
def walk(d, path):
    cur = d
    for key in path.split("."):
        if not isinstance(cur, dict):
            return ""
        cur = cur.get(key)
        if cur is None:
            return ""
    if isinstance(cur, (dict, list)):
        return json.dumps(cur)
    return str(cur)
try:
    print(walk(json.loads(sys.argv[1]), sys.argv[2]))
except Exception:
    print("")
' "$1" "$2" 2>/dev/null
}

# ---------------------------------------------------------------------------
# HTTP helpers
# ---------------------------------------------------------------------------
http_code() {
  curl -s -k -m 10 -o /dev/null -w '%{http_code}' "$1" 2>/dev/null
}

# api_req METHOD PATH [BODY]
#   * Assembles $RENOS_URL/api/v1$PATH (PATH must start with /).
#   * Sets globals: API_STATUS (http code, 000 on transport error), API_BODY.
#   * Adds Authorization: Bearer when AUTH_TOKEN is set.
#   * All requests use a JSON body; CSRF middleware only rejects form-encoded
#     requests without a token, so JSON requests pass through.
api_req() {
  local method=$1 path=$2 body=${3:-}
  local url="$RENOS_URL/api/v1$path"
  local args=(-s -k -m 30 -X "$method" -H 'Content-Type: application/json' -H 'X-Requested-With: renos-e2e' -w '\n%{http_code}')
  if [[ -n "$AUTH_TOKEN" ]]; then
    args+=(-H "Authorization: Bearer $AUTH_TOKEN")
  fi
  if [[ -n "$body" ]]; then
    args+=(-d "$body")
  fi
  local tmpout tmpcode
  tmpout=$(mktemp)
  # -o FILE sends the body to FILE; the -w write-out goes to stdout.
  tmpcode=$(curl "${args[@]}" "$url" -o "$tmpout" 2>/dev/null)
  if [[ -n "$tmpcode" ]]; then
    tmpcode=$(printf '%s' "$tmpcode" | tr -d '\r\n')
    API_BODY=$(cat "$tmpout")
  else
    tmpcode="000"
    API_BODY=""
  fi
  rm -f "$tmpout"
  API_STATUS="$tmpcode"
  printf '%s' "$API_STATUS"
}

wait_for_health() {
  local timeout="${1:-$HEALTH_TIMEOUT}" waited=0 code
  while [[ $waited -lt $timeout ]]; do
    code=$(http_code "$RENOS_URL/api/v1/healthz")
    if [[ "$code" == "200" ]]; then
      CORE_UP=true
      _ok "core healthy at $RENOS_URL/api/v1/healthz"
      return 0
    fi
    sleep 2
    waited=$((waited + 2))
  done
  CORE_UP=false
  _note "core not healthy after ${timeout}s (healthz -> $code)"
  return 1
}

# ---------------------------------------------------------------------------
# Preconditions
# ---------------------------------------------------------------------------
require_api() {
  if [[ "$CORE_UP" != "true" ]]; then
    step_skip "core not reachable ($RENOS_URL/api/v1/healthz)"
    return 1
  fi
  return 0
}

require_auth() {
  if [[ -z "$AUTH_TOKEN" ]]; then
    step_skip "no auth token (login phase failed/skipped)"
    return 1
  fi
  return 0
}

# ---------------------------------------------------------------------------
# Assertions
# ---------------------------------------------------------------------------
assert_ok() {
  local desc=$1 cond=${2:-1}
  if [[ "$cond" == "0" ]]; then
    ASSERT_PASS=$((ASSERT_PASS + 1))
    _ok "  assert: $desc"
  else
    ASSERT_FAIL=$((ASSERT_FAIL + 1))
    _err "  assert: $desc"
  fi
}

assert_eq() {
  local desc=$1 got=$2 want=$3
  if [[ "$got" == "$want" ]]; then
    assert_ok "$desc (got: $got)" 0
  else
    assert_ok "$desc (got: '$got', want: '$want')" 1
  fi
}

assert_contains() {
  local desc=$1 haystack=$2 needle=$3
  if [[ "$haystack" == *"$needle"* ]]; then
    assert_ok "$desc (contains: $needle)" 0
  else
    assert_ok "$desc (missing: '$needle')" 1
  fi
}

# ---------------------------------------------------------------------------
# Step bookkeeping
# ---------------------------------------------------------------------------
step() {
  CURRENT_STEP="$1"
  PHASE_BASE_FAIL="$ASSERT_FAIL"
  _info "== $1 =="
}

_record() {
  RESULTS_PHASE+=("$CURRENT_STEP")
  RESULTS_STATUS+=("$1")
  RESULTS_MSG+=("$2")
  if [[ -n "$RESULTS_FILE" ]]; then
    printf '%s\t%s\t%s\n' "$1" "$CURRENT_STEP" "$2" >>"$RESULTS_FILE"
  fi
}

step_ok() {
  _record PASS "$*"
  _ok "  [PASS] $CURRENT_STEP — $*"
}

step_fail() {
  _record FAIL "$*"
  _err "  [FAIL] $CURRENT_STEP — $*"
}

step_skip() {
  _record SKIP "$*"
  _note "  [SKIP] $CURRENT_STEP — $*"
}

# step_summary <pass_msg> <fail_msg> — finish a phase: PASS only when no
# assertion failed during this phase, otherwise FAIL. Keeps the step outcome
# honest relative to the assertions that ran.
step_summary() {
  local base="${PHASE_BASE_FAIL:-0}"
  if [[ "$ASSERT_FAIL" -gt "$base" ]]; then
    step_fail "$2"
  else
    step_ok "$1"
  fi
}

# ---------------------------------------------------------------------------
# Report
# ---------------------------------------------------------------------------
init_results_file() {
  mkdir -p "$E2E_RESULTS_DIR"
  RESULTS_FILE="$E2E_RESULTS_DIR/e2e-results-$(date +%Y%m%d-%H%M%S).txt"
  printf '# RenOS e2e run %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" >"$RESULTS_FILE"
  printf 'url=%s\n' "$RENOS_URL" >>"$RESULTS_FILE"
  _note "results file: $RESULTS_FILE"
}

write_report() {
  local passed=0 failed=0 skipped=0
  for s in "${RESULTS_STATUS[@]}"; do
    case "$s" in
      PASS) passed=$((passed + 1)) ;;
      FAIL) failed=$((failed + 1)) ;;
      SKIP) skipped=$((skipped + 1)) ;;
    esac
  done
  local i rowfmt='  | %-27s | %-6s | %-14s |'
  printf '\n'
  _banner "E2E Report — $RENOS_URL (env: $ADMIN_EMAIL)"
  printf '%s\n' '  +-----------------------------+--------+----------------+'
  printf "$rowfmt\n" 'Step' 'Status' 'Detail'
  printf '%s\n' '  +-----------------------------+--------+----------------+'
  for i in "${!RESULTS_PHASE[@]}"; do
    printf "$rowfmt\n" "${RESULTS_PHASE[$i]}" "${RESULTS_STATUS[$i]}" "${RESULTS_MSG[$i]:0:14}"
  done
  printf '%s\n' '  +-----------------------------+--------+----------------+'
  printf '  %-27s | %-6s | %d pass, %d fail, %d skip\n' 'TOTAL' '' "$passed" "$failed" "$skipped"
  printf '\n'
  if [[ -n "$REPORT_ARG" ]]; then
    {
      printf '# RenOS e2e report — %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
      printf 'url=%s\n' "$RENOS_URL"
      printf 'result: %s pass, %s fail, %s skip\n' "$passed" "$failed" "$skipped"
      for i in "${!RESULTS_PHASE[@]}"; do
        printf '%s\t%s\t%s\n' "${RESULTS_PHASE[$i]}" "${RESULTS_STATUS[$i]}" "${RESULTS_MSG[$i]}"
      done
    } >"$REPORT_ARG"
    _note "report written to $REPORT_ARG"
  fi
  if [[ "$failed" -gt 0 ]]; then
    _err "E2E FAILED: $passed pass, $failed fail, $skipped skip"
    return 1
  fi
  _ok "E2E PASSED: $passed pass, $skipped skip"
  return 0
}

# ---------------------------------------------------------------------------
# Flags
# ---------------------------------------------------------------------------
has_flag() {
  local needle=$1
  for a in "${ARGS[@]:-}"; do
    [[ "$a" == "$needle" ]] && return 0
  done
  return 1
}
