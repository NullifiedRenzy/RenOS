#!/usr/bin/env bash
# ──────────────────────────────────────────────────────────────────────────────
# RenOS 1.0.0 — Comprehensive Test Runner
#
# Runs all test suites and generates a consolidated test report.
# Usage: ./run-all-tests.sh [--quick] [--report-only]
#   --quick        Skip fuzz tests, only run unit/integration/lint
#   --report-only  Generate report from existing test artifacts, skip tests
# ──────────────────────────────────────────────────────────────────────────────
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
RELEASE_DIR="${ROOT_DIR}"
CORE_DIR="${ROOT_DIR}/core"
HUB_DIR="${ROOT_DIR}/hub"
AGENT_DIR="${ROOT_DIR}/agent"
REPORT_DIR="${ROOT_DIR}/reports"
TIMESTAMP="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

QUICK=false
REPORT_ONLY=false
for arg in "$@"; do
  case "$arg" in
    --quick) QUICK=true ;;
    --report-only) REPORT_ONLY=true ;;
  esac
done

mkdir -p "${REPORT_DIR}"

# ── Helpers ──────────────────────────────────────────────────────────────────

PASS=0
FAIL=0
SKIP=0
RESULTS=()

pass()  { PASS=$((PASS + 1)); RESULTS+=("✅ PASS  | $1"); echo "  ✅ $1"; }
fail()  { FAIL=$((FAIL + 1)); RESULTS+=("❌ FAIL  | $1"); echo "  ❌ $1"; }
skip()  { SKIP=$((SKIP + 1)); RESULTS+=("⏭️ SKIP  | $1"); echo "  ⏭️ $1"; }
header(){ echo ""; echo "── $1 ─────────────────────────────"; }

cleanup() {
  if [ "${GOCOVERDIR:-}" != "" ] && [ -d "$GOCOVERDIR" ]; then
    rm -rf "$GOCOVERDIR"
  fi
}
trap cleanup EXIT

# ── Go Unit Tests (with race detector) ───────────────────────────────────────

run_go_tests() {
  header "Go Unit Tests (core)"

  if [ ! -d "${CORE_DIR}" ]; then
    fail "Go unit tests — core directory not found"
    return
  fi

  cd "${CORE_DIR}"

  GOCOVERDIR=$(mktemp -d)
  export GOCOVERDIR

  set +e
  go test \
    -count=1 \
    -race \
    -cover \
    -covermode=atomic \
    -timeout=300s \
    -v \
    ./... 2>&1 | tee "${REPORT_DIR}/go-test-output.log"
  local exit_code=${PIPESTATUS[0]}
  set -e

  if [ ${exit_code} -eq 0 ]; then
    pass "Go unit tests (all passed)"
  else
    fail "Go unit tests (exit code ${exit_code})"
  fi

  cd "${RELEASE_DIR}"
}

# ── Fuzz Tests (brief run for CI) ───────────────────────────────────────────

run_fuzz_tests() {
  header "Fuzz Tests"

  if [ ! -d "${RELEASE_DIR}/tests/fuzz" ]; then
    skip "Fuzz tests — directory not found"
    return
  fi

  if [ "${QUICK}" = true ]; then
    skip "Fuzz tests (--quick mode)"
    return
  fi

  FUZZ_DIR="${RELEASE_DIR}/tests/fuzz"
  local duration="15s"

  set +e
  if [ -f "${FUZZ_DIR}/run-fuzz.sh" ]; then
    bash "${FUZZ_DIR}/run-fuzz.sh" "${duration}" 2>&1 | tee "${REPORT_DIR}/fuzz-output.log"
    local exit_code=${PIPESTATUS[0]}
  else
    cd "${CORE_DIR}"
    go test -fuzz="^Fuzz" -fuzztime="${duration}" -count=1 "${FUZZ_DIR}" 2>&1 | tee "${REPORT_DIR}/fuzz-output.log"
    local exit_code=${PIPESTATUS[0]}
    cd "${RELEASE_DIR}"
  fi
  set -e

  if [ ${exit_code} -eq 0 ]; then
    pass "Fuzz tests (all passed)"
  else
    fail "Fuzz tests (exit code ${exit_code})"
  fi
}

# ── Hub (npm) Tests ──────────────────────────────────────────────────────────

run_hub_tests() {
  header "Hub Tests (npm)"

  if [ ! -d "${HUB_DIR}" ]; then
    skip "Hub tests — directory not found"
    return
  fi

  if [ ! -f "${HUB_DIR}/package.json" ]; then
    skip "Hub tests — no package.json"
    return
  fi

  cd "${HUB_DIR}"

  set +e
  npm test 2>&1 | tee "${REPORT_DIR}/hub-test-output.log"
  local exit_code=${PIPESTATUS[0]}
  set -e

  if [ ${exit_code} -eq 0 ]; then
    pass "npm tests (all passed)"
  else
    fail "npm tests (exit code ${exit_code})"
  fi

  cd "${RELEASE_DIR}"
}

# ── Hub Lint & Typecheck ─────────────────────────────────────────────────────

run_hub_lint() {
  header "Hub Lint & Typecheck"

  cd "${HUB_DIR}"

  set +e
  npm run lint 2>&1 | tee "${REPORT_DIR}/hub-lint-output.log"
  local lint_exit=${PIPESTATUS[0]}

  npm run typecheck 2>&1 | tee "${REPORT_DIR}/hub-typecheck-output.log"
  local tc_exit=${PIPESTATUS[0]}
  set -e

  if [ ${lint_exit} -eq 0 ] && [ ${tc_exit} -eq 0 ]; then
    pass "Hub lint & typecheck"
  else
    local msg=""
    [ ${lint_exit} -ne 0 ] && msg+="lint(${lint_exit}) "
    [ ${tc_exit} -ne 0 ] && msg+="typecheck(${tc_exit})"
    fail "Hub lint & typecheck — ${msg}"
  fi

  cd "${RELEASE_DIR}"
}

# ── Cargo Tests (agent) ──────────────────────────────────────────────────────

run_cargo_tests() {
  header "Agent Tests (cargo)"

  if [ ! -d "${AGENT_DIR}" ]; then
    skip "Agent tests — directory not found"
    return
  fi

  if [ ! -f "${AGENT_DIR}/Cargo.toml" ]; then
    skip "Agent tests — no Cargo.toml"
    return
  fi

  cd "${AGENT_DIR}"

  set +e
  cargo test 2>&1 | tee "${REPORT_DIR}/agent-test-output.log"
  local exit_code=${PIPESTATUS[0]}
  set -e

  if [ ${exit_code} -eq 0 ]; then
    pass "Cargo tests (all passed)"
  else
    fail "Cargo tests (exit code ${exit_code})"
  fi

  cd "${RELEASE_DIR}"
}

# ── ShellCheck ───────────────────────────────────────────────────────────────

run_shellcheck() {
  header "ShellCheck"

  if ! command -v shellcheck &>/dev/null; then
    skip "ShellCheck — not installed"
    return
  fi

  local scripts
  scripts=$(find "${RELEASE_DIR}" -name '*.sh' -type f 2>/dev/null || true)
  scripts="$scripts $(find "${ROOT_DIR}/scripts" -name '*.sh' -type f 2>/dev/null || true)"
  scripts="$scripts $(find "${ROOT_DIR}/deploy" -name '*.sh' -type f 2>/dev/null || true)"
  scripts="$scripts $(find "${ROOT_DIR}/install.sh" -type f 2>/dev/null || true)"

  if [ -z "${scripts}" ]; then
    skip "ShellCheck — no shell scripts found"
    return
  fi

  set +e
  echo "${scripts}" | xargs shellcheck -x -S style 2>&1 | tee "${REPORT_DIR}/shellcheck-output.log"
  local exit_code=${PIPESTATUS[0]}
  set -e

  if [ ${exit_code} -eq 0 ]; then
    pass "ShellCheck (all scripts clean)"
  else
    fail "ShellCheck (exit code ${exit_code})"
  fi
}

# ── YAML Validation ──────────────────────────────────────────────────────────

run_yaml_validation() {
  header "YAML Validation"

  if ! command -v yamllint &>/dev/null; then
    skip "YAML validation — yamllint not installed"
    return
  fi

  local yaml_files
  yaml_files=$(find "${ROOT_DIR}" -name '*.yml' -o -name '*.yaml' | grep -v node_modules | grep -v '.git/' | grep -v target/ | sort || true)

  if [ -z "${yaml_files}" ]; then
    skip "YAML validation — no YAML files found"
    return
  fi

  set +e
  echo "${yaml_files}" | xargs yamllint --strict 2>&1 | tee "${REPORT_DIR}/yamllint-output.log"
  local exit_code=${PIPESTATUS[0]}
  set -e

  if [ ${exit_code} -eq 0 ]; then
    pass "YAML validation (all files valid)"
  else
    fail "YAML validation (exit code ${exit_code})"
  fi
}

# ── OpenAPI Spec Validation ──────────────────────────────────────────────────

run_openapi_validation() {
  header "OpenAPI Spec Validation"

  local spec_files=()
  if [ -f "${CORE_DIR}/api/openapi.yaml" ]; then
    spec_files+=("${CORE_DIR}/api/openapi.yaml")
  fi
  if [ -f "${CORE_DIR}/api/openapi.json" ]; then
    spec_files+=("${CORE_DIR}/api/openapi.json")
  fi

  if [ ${#spec_files[@]} -eq 0 ]; then
    skip "OpenAPI validation — no spec files found"
    return
  fi

  if command -v openapi-spec-validator &>/dev/null; then
    local all_valid=true
    for spec in "${spec_files[@]}"; do
      set +e
      openapi-spec-validator "${spec}" 2>&1 | tee -a "${REPORT_DIR}/openapi-validation-output.log"
      local exit_code=${PIPESTATUS[0]}
      set -e
      if [ ${exit_code} -ne 0 ]; then
        all_valid=false
        fail "OpenAPI validation — ${spec}"
      fi
    done
    if [ "${all_valid}" = true ]; then
      pass "OpenAPI validation (all specs valid)"
    fi
  elif command -v npx &>/dev/null && npx --yes @redocly/cli &>/dev/null; then
    local all_valid=true
    for spec in "${spec_files[@]}"; do
      set +e
      npx @redocly/cli lint "${spec}" 2>&1 | tee -a "${REPORT_DIR}/openapi-validation-output.log"
      local exit_code=${PIPESTATUS[0]}
      set -e
      if [ ${exit_code} -ne 0 ]; then
        all_valid=false
        fail "OpenAPI validation — ${spec}"
      fi
    done
    if [ "${all_valid}" = true ]; then
      pass "OpenAPI validation (all specs valid)"
    fi
  else
    skip "OpenAPI validation — no validator tool available"
  fi
}

# ── Report Generation ────────────────────────────────────────────────────────

generate_report() {
  header "Generating Test Report"

  local total=$((PASS + FAIL + SKIP))
  local report="${REPORT_DIR}/test-report.md"

  cat > "${report}" << EOF
# RenOS 1.0.0 — Test Report

| Field | Value |
|---|---|
| **Version** | 1.0.0-RELEASE |
| **Generated** | ${TIMESTAMP} |
| **Mode** | $([ "${QUICK}" = true ] && echo "Quick" || echo "Full") |
| **Trigger** | Manual / CI |

## Summary

| Result | Count |
|---|---|
| ✅ Pass | ${PASS} |
| ❌ Fail | ${FAIL} |
| ⏭️ Skip | ${SKIP} |
| **Total** | **${total}** |

## Results

| Status | Suite |
|---|---|
EOF

  for r in "${RESULTS[@]}"; do
    IFS='|' read -r status suite <<< "${r#*| }"
    echo "${r}" | sed 's/^/| /' >> "${report}"
  done

  cat >> "${report}" << 'EOF'

## Artifacts

| Artifact | Path |
|---|---|
EOF

  for f in "${REPORT_DIR}"/*.log; do
    if [ -f "$f" ]; then
      local name
      name=$(basename "$f")
      echo "| ${name} | \`reports/${name}\` |" >> "${report}"
    fi
  done

  cat >> "${report}" << 'EOF'

## Notes

- Unit tests are run with the Go race detector enabled.
- Fuzz tests run for 15s per target in CI; longer runs are performed in release pipeline.
- Shell scripts are validated with ShellCheck (severity: style).
- YAML files are validated with yamllint in strict mode.
- OpenAPI specs are validated with redocly-cli or openapi-spec-validator.
EOF

  echo "  📄 Report written to ${report}"
}

# ── Main ─────────────────────────────────────────────────────────────────────

echo "══════════════════════════════════════════════════════════════════════════"
echo " RenOS 1.0.0 — Comprehensive Test Runner"
echo " Started: ${TIMESTAMP}"
echo " Mode:    $([ "${QUICK}" = true ] && echo "Quick" || echo "Full")"
echo "══════════════════════════════════════════════════════════════════════════"

if [ "${REPORT_ONLY}" = true ]; then
  echo ""
  echo "Report-only mode: generating report from existing artifacts..."
  generate_report
  echo ""
  echo "══════════════════════════════════════════════════════════════════════════"
  echo " Done."
  echo "══════════════════════════════════════════════════════════════════════════"
  exit 0
fi

run_go_tests
run_fuzz_tests
run_hub_tests
run_hub_lint
run_cargo_tests
run_shellcheck
run_yaml_validation
run_openapi_validation

generate_report

echo ""
echo "══════════════════════════════════════════════════════════════════════════"
echo " Results: ${PASS} passed, ${FAIL} failed, ${SKIP} skipped"
echo " Report:  ${REPORT_DIR}/test-report.md"
echo "══════════════════════════════════════════════════════════════════════════"

if [ "${FAIL}" -gt 0 ]; then
  exit 1
fi
exit 0
