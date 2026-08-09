#!/usr/bin/env bash
# run-e2e.sh — end-to-end QA harness for the RenOS user journey.
#
# Usage:
#   ./run-e2e.sh                 full run
#   ./run-e2e.sh --quick         skip install/backup/restore + standalone suites
#   ./run-e2e.sh --setup-only    bring up a backend, verify health, then stop
#   ./run-e2e.sh --skip-install  assume a backend is already running
#   ./run-e2e.sh --cleanup       remove e2e temp artifacts / best-effort compose down
#   ./run-e2e.sh --report FILE   write a machine-readable report to FILE
#
# Environment:
#   RENOS_URL        API base URL      (default http://localhost:8443)
#   ADMIN_EMAIL      login email       (default admin@renos.local)
#   ADMIN_PASSWORD   login password    (default admin)
#   HEALTH_TIMEOUT   health wait secs  (default 90)
#
# Exit code: 0 only if every step PASSed (SKIP is non-fatal).

set -uo pipefail

E2E_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

source "$E2E_ROOT/lib.sh"
for f in "$E2E_ROOT"/phases/*.sh; do
  [[ -f "$f" ]] && source "$f"
done

PHASES=(install login health enroll command workflow terminal plugin ai backup restore upgrade rollback)

usage() {
  cat <<EOF
run-e2e.sh — RenOS end-to-end QA harness

Usage: run-e2e.sh [--quick] [--setup-only] [--skip-install] [--cleanup] [--report FILE] [--help]

Flags:
  --quick          skip install, backup/restore, and standalone verifier suites
  --setup-only     provision backend + verify health, then exit
  --skip-install   assume a backend is already reachable
  --cleanup        remove e2e temp artifacts (best-effort compose down)
  --report FILE    write a machine-readable report to FILE
  --help, -h       show this help

Environment:
  RENOS_URL       API base URL       (default http://localhost:8443)
  ADMIN_EMAIL     login email        (default admin@renos.local)
  ADMIN_PASSWORD  login password     (default admin)
  HEALTH_TIMEOUT  health wait (s)    (default 90)

Exit: 0 only if every step PASSed (SKIP is non-fatal).
EOF
}

banner() {
  _banner "══ RenOS end-to-end QA ══"
  printf '  url:       %s\n' "$RENOS_URL"
  printf '  admin:     %s\n' "$ADMIN_EMAIL"
  printf '  repo:      %s\n' "$REPO_ROOT"
  printf '  mode:      quick=%s setup_only=%s skip_install=%s\n' "$QUICK" "$SETUP_ONLY" "$SKIP_INSTALL"
  printf '  started:   %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}

plan() {
  _banner "Test plan"
  local d
  for p in "${PHASES[@]}"; do
    case "$p" in
      install)   d="provision/detect backend at $RENOS_URL" ;;
      health)    d="liveness/readiness + dashboard endpoints" ;;
      login)     d="POST /api/v1/auth/login -> token" ;;
      enroll)    d="enroll-token + RSA + enroll + connect + list" ;;
      command)   d="device command submit + agent result + jobs" ;;
      workflow)  d="create workflow + step + execute + list" ;;
      terminal)  d="terminal session listing endpoints" ;;
      plugin)    d="marketplace/plugin-platform + best-effort install" ;;
      ai)        d="conversations/settings/chat (graceful if unconfigured)" ;;
      backup)    d="renos-backup.sh -> manifest validation" ;;
      restore)   d="manifest + archive integrity (root-gated dry-run)" ;;
      upgrade)   d="verify-upgrade.sh + update API endpoints" ;;
      rollback)  d="verify-rollback.sh + rollback API endpoints" ;;
      *)         d="" ;;
    esac
    printf '  %-10s %s\n' "$p" "$d"
  done
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --help|-h) MODE="help"; return 0 ;;
      --quick)    QUICK=true; SKIP_INSTALL=true ;;
      --setup-only) SETUP_ONLY=true ;;
      --skip-install) SKIP_INSTALL=true ;;
      --cleanup)  CLEANUP=true ;;
      --report)   shift; REPORT_ARG="$1" ;;
      *)
        _err "unknown flag: $1"
        usage
        exit 2
        ;;
    esac
    shift
  done
}

cleanup_all() {
  _note "cleanup: removing /tmp/renos-e2e-* and best-effort compose down"
  rm -rf /tmp/renos-e2e-*
  docker compose -p renos-e2e down >/dev/null 2>&1 || true
  _ok "cleanup done"
}

cleanup_tmp() {
  [[ -n "$E2E_TMP" && -d "$E2E_TMP" ]] && rm -rf "$E2E_TMP"
}

main() {
  MODE="run"
  parse_args "$@"

  case "$MODE" in
    help) usage; exit 0 ;;
  esac
  if [[ "$CLEANUP" == "true" ]]; then
    cleanup_all
    exit 0
  fi

  load_env
  banner
  plan

  E2E_TMP=$(mktemp -d /tmp/renos-e2e-XXXXXX)
  trap cleanup_tmp EXIT INT TERM
  init_results_file

  _info "waiting for backend health (timeout ${HEALTH_TIMEOUT}s)..."
  wait_for_health "$HEALTH_TIMEOUT"

  if [[ "$SETUP_ONLY" == "true" ]]; then
    phase_install
    phase_health
    write_report
    exit $?
  fi

  for p in "${PHASES[@]}"; do
    "phase_$p"
  done

  write_report
  exit $?
}

main "$@"
