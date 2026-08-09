#!/usr/bin/env bash
# RenOS 1.0.0 — Master Verification Runner
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPORT_DIR="$(cd "$SCRIPT_DIR/../reports" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'

header() { echo -e "\n${BOLD}${CYAN}══ $* ══${NC}\n"; }
info()  { echo -e "  ${CYAN}[INFO]${NC} $*"; }

SCRIPTS=(
  "verify-backup-restore.sh:Backup/Restore Verification"
  "verify-upgrade.sh:Upgrade Verification"
  "verify-rollback.sh:Rollback Verification"
)

RESULTS_DIR=$(mktemp -d /tmp/renos-verify-all-XXXXXX)
OVERALL_PASS=0
OVERALL_FAIL=0
declare -a SCRIPT_RESULTS=()
declare -a SCRIPT_NAMES=()
declare -a SCRIPT_STATUSES=()

run_test_script() {
  local script=$1
  local label=$2
  local script_path="$SCRIPT_DIR/$script"
  local result_file="$RESULTS_DIR/$script.exit"
  local output_file="$RESULTS_DIR/$script.log"

  info "Running: $label ($script)..."
  set +e
  bash "$script_path" > "$output_file" 2>&1
  local rc=$?
  set -e

  echo "$rc" > "$result_file"
  SCRIPT_NAMES+=("$label")
  SCRIPT_RESULTS+=("$script")

  if [[ $rc -eq 0 ]]; then
    echo -e "  ${GREEN}[PASS]${NC} $label"
    SCRIPT_STATUSES+=("PASS")
    OVERALL_PASS=$((OVERALL_PASS + 1))
  else
    echo -e "  ${RED}[FAIL]${NC} $label (exit $rc)"
    SCRIPT_STATUSES+=("FAIL")
    OVERALL_FAIL=$((OVERALL_FAIL + 1))
  fi
}

generate_combined_report() {
  local status="$1"
  local report_file="$REPORT_DIR/verification-report.md"

  mkdir -p "$REPORT_DIR"
  cat > "$report_file" <<REPORT
# RenOS 1.0.0 — Combined Verification Report

**Date:** $(date -Iseconds)  
**Runner:** verify-all.sh  

## Overall Summary

- **Total Suites:** $((OVERALL_PASS + OVERALL_FAIL))
- **Passed:** $OVERALL_PASS
- **Failed:** $OVERALL_FAIL
- **Status:** $status

## Suite Results

| # | Suite | Script | Result |
|---|-------|--------|--------|
$(for i in "${!SCRIPT_NAMES[@]}"; do printf "| %d | %s | %s | %s |\n" $((i+1)) "${SCRIPT_NAMES[$i]}" "${SCRIPT_RESULTS[$i]}" "${SCRIPT_STATUSES[$i]}"; done)

## Individual Reports

$(for i in "${!SCRIPT_NAMES[@]}"; do
  script="${SCRIPT_RESULTS[$i]}"
  log_file="$RESULTS_DIR/$script.log"
  pass_count=$(grep -c '\[PASS\]' "$log_file" 2>/dev/null || echo 0)
  fail_count=$(grep -c '\[FAIL\]' "$log_file" 2>/dev/null || echo 0)
  echo ""
  echo "### ${SCRIPT_NAMES[$i]}"
  echo ""
  echo "- **Script:** ${SCRIPT_RESULTS[$i]}"
  echo "- **Status:** ${SCRIPT_STATUSES[$i]}"
  echo "- **Passed:** ${pass_count}"
  echo "- **Failed:** ${fail_count}"
  echo "- **Report:** \`${REPORT_DIR}/${script%.sh}-verification.md\`"
done)

## Execution Environment

- **Repository Root:** $REPO_ROOT
- **Reports Directory:** $REPORT_DIR
- **Tests Directory:** $SCRIPT_DIR
- **Test Runner:** $(hostname)
- **OS:** $(uname -a)

REPORT
  echo -e "\n  ${CYAN}[INFO]${NC} Combined report: $report_file"
}

#===============================================================================
# MAIN
#===============================================================================

header "RenOS 1.0.0 — Master Verification Suite"

if [[ $EUID -ne 0 ]]; then
  echo -e "  ${YELLOW}[WARN]${NC} Not running as root. Some tests may fail."
  echo -e "  ${YELLOW}[WARN]${NC} Backup/restore tests require root."
fi

for entry in "${SCRIPTS[@]}"; do
  script="${entry%%:*}"
  label="${entry#*:}"
  run_test_script "$script" "$label"
done

# Summary
header "Overall Results"
echo -e "  ${GREEN}Suites passed:${NC} $OVERALL_PASS"
echo -e "  ${RED}Suites failed:${NC} $OVERALL_FAIL"

if [[ $OVERALL_FAIL -eq 0 ]]; then
  echo -e "  ${GREEN}All verification suites passed.${NC}"
  generate_combined_report "✅ ALL PASSED"
  rm -rf "$RESULTS_DIR"
  exit 0
else
  echo -e "  ${RED}Some verification suites failed.${NC}"
  echo -e "  ${YELLOW}Check individual reports for details.${NC}"
  generate_combined_report "❌ SOME FAILED"
  rm -rf "$RESULTS_DIR"
  exit 1
fi
