#!/usr/bin/env bash
set -euo pipefail

# ── RenOS 1.0.0 — Fuzz Test Runner ───────────────────────────────────────────
# Runs all fuzz tests for a minimum duration per target.
# Usage: ./run-fuzz.sh [duration] [parallel]
#   duration   minimum time per fuzz target (default: 30s)
#   parallel   number of concurrent fuzz processes (default: 1)

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
CORE_DIR="${ROOT_DIR}/core"
RELEASE_DIR="${ROOT_DIR}/release"
FUZZ_DIR="${RELEASE_DIR}/tests/fuzz"

DURATION="${1:-30s}"
PARALLEL="${2:-1}"

export GOFLAGS="${GOFLAGS:--count=1}"
export GORACE="${GORACE:-halt_on_error=1}"

echo "============================================"
echo " RenOS 1.0.0 — Fuzz Test Suite"
echo "============================================"
echo " Go version : $(go version 2>/dev/null | awk '{print $3}')"
echo " Duration   : ${DURATION} per target"
echo " Parallel   : ${PARALLEL}"
echo " Core dir   : ${CORE_DIR}"
echo " Fuzz dir   : ${FUZZ_DIR}"
echo "============================================"
echo ""

# Targets mapped to their function names
declare -A TARGETS
TARGETS=(
  ["FuzzAPIRequest"]="FuzzAPIRequest"
  ["FuzzJSONParse"]="FuzzJSONParse"
  ["FuzzWebSocket"]="FuzzWebSocket"
  ["FuzzDeviceID"]="FuzzDeviceID"
  ["FuzzWorkflowName"]="FuzzWorkflowName"
  ["FuzzJWTToken"]="FuzzJWTToken"
)

RESULTS_DIR="${FUZZ_DIR}/results"
mkdir -p "${RESULTS_DIR}"

FAILED=0
PASSED=0

run_target() {
  local target=$1
  local func=$2
  local corpus="${FUZZ_DIR}/testdata/fuzz/${func}"
  local logfile="${RESULTS_DIR}/${func}.log"
  local cache="${RESULTS_DIR}/${func}.cache"

  echo "  🔎 Running ${func} for ${DURATION}..."

  if cd "${CORE_DIR}" 2>/dev/null; then
    :
  else
    echo "  ❌ ${func}: core directory not found at ${CORE_DIR}"
    return 1
  fi

  set +e
  go test -fuzz="^${func}$" \
    -fuzztime="${DURATION}" \
    -parallel="${PARALLEL}" \
    -cpu="$(nproc 2>/dev/null || echo 4)" \
    -count=1 \
    -v \
    "${FUZZ_DIR}" \
    2>&1 | tee "${logfile}"
  local exit_code=${PIPESTATUS[0]}
  set -e

  cd "${RELEASE_DIR}"

  if [ ${exit_code} -eq 0 ]; then
    echo "  ✅ ${func} — PASSED"
    return 0
  else
    echo "  ❌ ${func} — FAILED (see ${logfile})"
    return 1
  fi
}

for target in "${!TARGETS[@]}"; do
  func="${TARGETS[$target]}"
  echo ""
  if run_target "${target}" "${func}"; then
    PASSED=$((PASSED + 1))
  else
    FAILED=$((FAILED + 1))
  fi
done

echo ""
echo "============================================"
echo " Fuzz Test Summary"
echo "============================================"
echo " Passed: ${PASSED}"
echo " Failed: ${FAILED}"
echo " Total:  $((PASSED + FAILED))"
echo "============================================"

if [ "${FAILED}" -gt 0 ]; then
  echo ""
  echo "Failed targets:"
  for target in "${!TARGETS[@]}"; do
    func="${TARGETS[$target]}"
    logfile="${RESULTS_DIR}/${func}.log"
    if [ -f "${logfile}" ] && grep -q "^--- FAIL" "${logfile}" 2>/dev/null; then
      echo "  - ${func} (${logfile})"
    fi
  done
  exit 1
fi
