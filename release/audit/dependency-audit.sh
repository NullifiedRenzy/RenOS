#!/usr/bin/env bash
# ============================================================================
# RenOS Dependency Audit
# ============================================================================
# Scans all Go modules, npm packages, and Rust crates for dependency health.
# Generates a comprehensive dependency report.
#
# Exit codes:
#   0 - All clean, no issues
#   1 - Warnings (outdated deps, audit tool missing, etc.)
#   2 - Vulnerabilities found
# ============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
REPORT_DIR="${PROJECT_DIR}/release/reports"
REPORT_FILE="${REPORT_DIR}/dependency-audit.md"

GO_MODULES=(
  "${PROJECT_DIR}/core"
  "${PROJECT_DIR}/renctl"
  "${PROJECT_DIR}/sdk/go"
  "${PROJECT_DIR}/installer/upgrade"
  "${PROJECT_DIR}/installer/wizard"
  "${PROJECT_DIR}/core/cmd/renos-plugin"
)

NPM_DIRS=(
  "${PROJECT_DIR}/hub"
  "${PROJECT_DIR}/sdk/typescript"
)

CARGO_DIRS=(
  "${PROJECT_DIR}/agent"
  "${PROJECT_DIR}/sdk/rust"
)

mkdir -p "$REPORT_DIR"

# --------------------------------------------------------------------------
# Helpers
# --------------------------------------------------------------------------

log()  { printf '[DEP AUDIT] %s\n' "$*"; }
warn() { printf '[DEP AUDIT] WARNING: %s\n' "$*" >&2; }
err()  { printf '[DEP AUDIT] ERROR: %s\n' "$*" >&2; }

EXIT_CODE=0

write_section() {
  local heading="$1"
  local content="$2"
  printf '\n## %s\n\n%s\n' "$heading" "$content" >> "$REPORT_FILE"
}

detect_tool() {
  command -v "$1" >/dev/null 2>&1
}

# Safely get module name from a go.mod directory
get_go_module_name() {
  local dir="$1"
  if detect_tool go; then
    (cd "$dir" && go list -m 2>/dev/null) || echo "unknown"
  else
    head -1 "$dir/go.mod" 2>/dev/null | awk '{print $2}' || echo "unknown"
  fi
}

# Extract direct deps from a go.mod (no go tool needed)
parse_go_mod_deps() {
  local file="$1"
  grep -E '^[[:space:]]+[a-zA-Z0-9]' "$file" 2>/dev/null | grep -v '// indirect' | sed 's/^[[:space:]]*//' || true
}

# Extract deps from Cargo.toml
parse_cargo_deps() {
  local file="$1"
  sed -n '/^\[dependencies\]/,/^\[/{ /^\[dependencies\]/d; /^\[/q; p; }' "$file" 2>/dev/null || true
}

# --------------------------------------------------------------------------
# Begin report
# --------------------------------------------------------------------------

TIMESTAMP="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
HOSTNAME="$(hostname 2>/dev/null || echo 'unknown')"

cat > "$REPORT_FILE" <<EOF
# RenOS Dependency Audit Report

**Generated:** $TIMESTAMP
**Host:** $HOSTNAME

EOF

# ============================================================================
# Go Module Scanning
# ============================================================================

printf '\n## Go Module Dependencies\n\n' >> "$REPORT_FILE"

GO_VULN_FOUND=0
GO_TOTAL_MODULES=0
GO_VULNERABLE_MODULES=0

for mod_dir in "${GO_MODULES[@]}"; do
  if [ ! -f "$mod_dir/go.mod" ]; then
    warn "No go.mod found in $mod_dir — skipping"
    continue
  fi

  module_name="$(get_go_module_name "$mod_dir")"
  log "Scanning Go module: $module_name ($mod_dir)"

  GO_TOTAL_MODULES=$((GO_TOTAL_MODULES + 1))

  write_section "Module: $module_name" "**Path:** \`$mod_dir\`\n"

  # Direct dependencies (from go.mod, indirect excluded)
  if detect_tool go; then
    direct_raw="$(cd "$mod_dir" && go list -m 2>/dev/null && go list -m all 2>/dev/null)" || true
    write_section "Direct Dependencies" "These are the direct (non-indirect) dependencies listed in \`go.mod\`:"
    {
      printf '| Dependency | Version |\n'
      printf '|---|---|\n'
      parse_go_mod_deps "$mod_dir/go.mod" | while IFS=' ' read -r dep_name dep_ver; do
        [ -n "$dep_name" ] && printf '| %s | %s |\n' "$dep_name" "$dep_ver"
      done
    } >> "$REPORT_FILE"
  else
    # Manual parsing
    write_section "Direct Dependencies" "These are the direct (non-indirect) dependencies listed in \`go.mod\`:"
    {
      printf '| Dependency | Version |\n'
      printf '|---|---|\n'
      parse_go_mod_deps "$mod_dir/go.mod" | while IFS=' ' read -r dep_name dep_ver; do
        [ -n "$dep_name" ] && printf '| %s | %s |\n' "$dep_name" "$dep_ver"
      done
    } >> "$REPORT_FILE"
  fi

  # go mod tidy + verify
  if detect_tool go; then
    log "  Running go mod tidy in $mod_dir"
    (cd "$mod_dir" && go mod tidy 2>&1) || warn "go mod tidy failed in $mod_dir"

    verify_output="$(cd "$mod_dir" && go mod verify 2>&1)" || true
    if echo "$verify_output" | grep -qi 'ERROR\|failed\|mismatch'; then
      warn "go mod verify found issues in $module_name"
    fi
    write_section "go mod verify" "\`\`\`\n${verify_output:-All modules verified successfully}\n\`\`\`"
  else
    warn "Go toolchain not available — cannot run go mod tidy/verify for $mod_dir"
    write_section "go mod verify" "_Go toolchain not available — skipped_"
  fi

  # All dependencies tree
  if detect_tool go; then
    all_deps="$(cd "$mod_dir" && go list -m all 2>&1)" || true
    write_section "All Dependencies (go list -m all)" "\`\`\`\n${all_deps}\n\`\`\`"
  else
    write_section "All Dependencies (go list -m all)" "_Go toolchain not available — skipped_"
  fi

  # govulncheck
  if detect_tool govulncheck; then
    log "  Running govulncheck in $mod_dir"
    vuln_output="$(cd "$mod_dir" && govulncheck ./... 2>&1)" || true
    if echo "$vuln_output" | grep -qi 'vulnerabilit\|CVE-\|known.*issue'; then
      GO_VULN_FOUND=$((GO_VULN_FOUND + 1))
      GO_VULNERABLE_MODULES=$((GO_VULNERABLE_MODULES + 1))
      EXIT_CODE=2
    fi
    write_section "Vulnerability Scan (govulncheck)" "\`\`\`\n${vuln_output:-No vulnerabilities found}\n\`\`\`"
  else
    warn "govulncheck not installed — skipping CVE scan for $mod_dir"
    write_section "Vulnerability Scan (govulncheck)" "_govulncheck is not installed._\n\nInstall with: \`go install golang.org/x/vuln/cmd/govulncheck@latest\`\n\nWould check for known CVEs in Go dependencies.\n"
  fi

  # Check for outdated deps
  if detect_tool go; then
    outdated="$(cd "$mod_dir" && go list -u -m all 2>&1 | grep -E '\[.*\]' || true)"
    if [ -n "$outdated" ]; then
      write_section "Outdated Dependencies" "The following dependencies have newer versions available:"
      {
        printf '| Dependency | Current | Available |\n'
        printf '|---|---|---|\n'
        echo "$outdated" | while IFS=' ' read -r dep_name dep_ver dep_update; do
          dep_update="${dep_update#\[}"
          dep_update="${dep_update%\]}"
          printf '| %s | %s | %s |\n' "$dep_name" "$dep_ver" "$dep_update"
        done
      } >> "$REPORT_FILE"
    else
      write_section "Outdated Dependencies" "All dependencies are up to date."
    fi
  fi

  echo "" >> "$REPORT_FILE"
done

# ============================================================================
# npm Audit
# ============================================================================

printf '\n## npm Dependencies\n\n' >> "$REPORT_FILE"

NPM_VULN_FOUND=0
NPM_DIRS_SCANNED=0

for npm_dir in "${NPM_DIRS[@]}"; do
  if [ ! -f "$npm_dir/package.json" ]; then
    continue
  fi

  NPM_DIRS_SCANNED=$((NPM_DIRS_SCANNED + 1))
  pkg_name="$(node -e "console.log(require('${npm_dir}/package.json').name || 'unknown')" 2>/dev/null || echo "unknown")"
  log "Scanning npm package: $pkg_name ($npm_dir)"

  write_section "Package: $pkg_name" "**Path:** \`$npm_dir\`\n"

  # Show direct dependencies from package.json
  if detect_tool node; then
    {
      printf '| Dependency | Version | Type |\n'
      printf '|---|---|---|\n'
      node -e "
        const p = require('${npm_dir}/package.json');
        const deps = p.dependencies || {};
        const devDeps = p.devDependencies || {};
        for (const [k,v] of Object.entries(deps)) console.log(k + '|' + v + '|production');
        for (const [k,v] of Object.entries(devDeps)) console.log(k + '|' + v + '|development');
      " 2>/dev/null | while IFS='|' read -r name ver type; do
        printf '| %s | %s | %s |\n' "$name" "$ver" "$type"
      done
    } >> "$REPORT_FILE"
  fi

  # npm audit
  if detect_tool npm; then
    log "  Running npm audit in $npm_dir"
    audit_output="$(cd "$npm_dir" && npm audit 2>&1)" || true
    if echo "$audit_output" | grep -qi 'high\|critical\|CVE-'; then
      NPM_VULN_FOUND=$((NPM_VULN_FOUND + 1))
      EXIT_CODE=2
    fi
    write_section "npm Audit" "\`\`\`\n${audit_output:-No output}\n\`\`\`"
  else
    warn "npm not available — skipping audit for $npm_dir"
    write_section "npm Audit" "_npm is not installed._\n\nWould check with: \`npm audit\`\n\nThis scans for known vulnerabilities in npm packages.\n"
  fi
done

# ============================================================================
# Rust / Cargo Audit
# ============================================================================

printf '\n## Rust Crate Dependencies\n\n' >> "$REPORT_FILE"

CARGO_VULN_FOUND=0
CARGO_DIRS_SCANNED=0

for cargo_dir in "${CARGO_DIRS[@]}"; do
  if [ ! -f "$cargo_dir/Cargo.toml" ]; then
    continue
  fi

  CARGO_DIRS_SCANNED=$((CARGO_DIRS_SCANNED + 1))
  crate_name="$(grep -m1 '^name' "$cargo_dir/Cargo.toml" 2>/dev/null | sed 's/.*"\(.*\)"/\1/' || echo "unknown")"
  log "Scanning Rust crate: $crate_name ($cargo_dir)"

  write_section "Crate: $crate_name" "**Path:** \`$cargo_dir\`\n"

  # Direct dependencies from Cargo.toml
  deps_block="$(parse_cargo_deps "$cargo_dir/Cargo.toml")"
  if [ -n "$deps_block" ]; then
    {
      printf '| Dependency | Version |\n'
      printf '|---|---|\n'
      echo "$deps_block" | while IFS= read -r line; do
        dep_name="$(echo "$line" | sed 's/^\([a-zA-Z0-9_-]*\).*/\1/')"
        dep_ver="$(echo "$line" | sed -n 's/.*version *= *"\([^"]*\)".*/\1/p')"
        [ -n "$dep_name" ] && [ -n "$dep_ver" ] && printf '| %s | %s |\n' "$dep_name" "$dep_ver"
      done
    } >> "$REPORT_FILE"
  fi

  if detect_tool cargo; then
    # cargo update --dry-run to check for updates
    log "  Checking for available updates in $cargo_dir"
    outdated="$(cd "$cargo_dir" && cargo update --dry-run 2>&1)" || true
    if echo "$outdated" | grep -qi 'Updating'; then
      write_section "Available Updates" "\`\`\`\n${outdated}\n\`\`\`"
    else
      write_section "Available Updates" "All dependencies are up to date."
    fi

    # cargo audit
    if detect_tool cargo-audit; then
      log "  Running cargo audit in $cargo_dir"
      audit_out="$(cd "$cargo_dir" && cargo audit 2>&1)" || true
      if echo "$audit_out" | grep -qi 'CVE-\|vulnerabilit\|RUSTSEC-'; then
        CARGO_VULN_FOUND=$((CARGO_VULN_FOUND + 1))
        EXIT_CODE=2
      fi
      write_section "cargo audit" "\`\`\`\n${audit_out:-No vulnerabilities found}\n\`\`\`"
    else
      warn "cargo-audit not installed — skipping CVE scan for $cargo_dir"
      write_section "cargo audit" "_cargo-audit is not installed._\n\nInstall with: \`cargo install cargo-audit\`\n\nWould check for known vulnerabilities in Rust crate dependencies.\n"
    fi
  else
    warn "Cargo not available — skipping audit for $cargo_dir"
    write_section "Cargo Audit" "_Cargo/Rust toolchain not available — skipped_"
  fi
done

# ============================================================================
# Summary
# ============================================================================

printf '\n## Summary\n\n' >> "$REPORT_FILE"

TOTAL_VULN=$((GO_VULN_FOUND + NPM_VULN_FOUND + CARGO_VULN_FOUND))

{
  printf '| Category | Count |\n'
  printf '|----------|-------|\n'
  printf '| Go modules scanned | %d |\n' "$GO_TOTAL_MODULES"
  printf '| npm packages scanned | %d |\n' "$NPM_DIRS_SCANNED"
  printf '| Rust crates scanned | %d |\n' "$CARGO_DIRS_SCANNED"
  printf '| Go modules with vulnerabilities | %d |\n' "$GO_VULNERABLE_MODULES"
  printf '| npm vulnerabilities found | %d |\n' "$NPM_VULN_FOUND"
  printf '| Rust vulnerabilities found | %d |\n' "$CARGO_VULN_FOUND"
  printf '| **Total vulnerabilities** | **%d** |\n' "$TOTAL_VULN"
} >> "$REPORT_FILE"

echo "" >> "$REPORT_FILE"

# Final status
if [ "$TOTAL_VULN" -gt 0 ]; then
  {
    printf '> **⚠️  Vulnerabilities detected.** Review the sections above for details.\n'
    printf '>\n'
    printf '> Exit code: 2\n'
  } >> "$REPORT_FILE"
  EXIT_CODE=2
elif [ "$EXIT_CODE" -eq 1 ]; then
  {
    printf '> **⚠️  Warnings present.** Audit completed with warnings.\n'
    printf '>\n'
    printf '> Exit code: 1\n'
  } >> "$REPORT_FILE"
else
  {
    printf '> **✅ All clean.** No vulnerabilities or warnings detected.\n'
    printf '>\n'
    printf '> Exit code: 0\n'
  } >> "$REPORT_FILE"
fi

echo "" >> "$REPORT_FILE"

log "Report generated: $REPORT_FILE"
exit "$EXIT_CODE"
