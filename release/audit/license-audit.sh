#!/usr/bin/env bash
# ============================================================================
# RenOS License Audit
# ============================================================================
# Scans all Go modules, npm packages, and Rust crates for license information.
# Classifies each dependency's license as compatible, conditional, incompatible,
# or unknown relative to this project's MIT license.
#
# Exit codes:
#   0 - All licenses compatible
#   1 - Warnings (conditional or unknown licenses)
#   2 - Incompatible licenses found
# ============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
REPORT_DIR="${PROJECT_DIR}/release/reports"
REPORT_FILE="${REPORT_DIR}/license-audit.md"

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

PROJECT_LICENSE="MIT"

mkdir -p "$REPORT_DIR"

# --------------------------------------------------------------------------
# Helpers
# --------------------------------------------------------------------------

log()  { printf '[LICENSE AUDIT] %s\n' "$*"; }
warn() { printf '[LICENSE AUDIT] WARNING: %s\n' "$*" >&2; }
err()  { printf '[LICENSE AUDIT] ERROR: %s\n' "$*" >&2; }

EXIT_CODE=0

detect_tool() {
  command -v "$1" >/dev/null 2>&1
}

write_section() {
  local heading="$1"
  local content="$2"
  printf '\n## %s\n\n%s\n' "$heading" "$content" >> "$REPORT_FILE"
}

# --------------------------------------------------------------------------
# License classification
# --------------------------------------------------------------------------
# Compatible: MIT, Apache-2.0, BSD-2/3, ISC, Zlib, Unlicense, CC0-1.0
# Conditional: MPL-2.0, LGPL-2.1/3.0, BSL-1.0
# Incompatible: GPL-2.0/3.0, AGPL-1.0/3.0, SSPL, BUSL-1.1, proprietary

classify_license() {
  local lic="$1"
  lic="$(echo "$lic" | tr '[:upper:]' '[:lower:]' | sed 's/^"//;s/"$//;s/^ *//;s/ *$//')"

  case "$lic" in
    mit*|apache*|apache-2.0*|bsd*|bsd-2*|bsd-3*|isc*|zlib*|unlicense*|cc0*|cc0-1.0*)
      echo "compatible"
      ;;
    mpl*|mpl-2.0*|lgpl*|lgpl-2.1*|lgpl-3.0*|bsl-1.0*)
      echo "conditional"
      ;;
    gpl*|gpl-2.0*|gpl-3.0*|agpl*|agpl-1.0*|agpl-3.0*|sspl*|busl*|busl-1.1*|proprietary*)
      echo "incompatible"
      ;;
    "")
      echo "unknown"
      ;;
    *)
      echo "unknown"
      ;;
  esac
}

# Extract the last segment after @ in npm package spec
# Handles scoped packages: @scope/name@version → version
# Handles regular packages: name@version → version
npm_pkg_version() {
  local spec="$1"
  echo "${spec##*@}"
}

npm_pkg_name() {
  local spec="$1"
  # For scoped packages, the name is everything up to the last @
  # @scope/name@version → @scope/name
  # lodash@version → lodash
  local name="${spec%@*}"
  if [ -z "$name" ]; then
    # Edge case: starts with @ but no version (shouldn't happen)
    echo "$spec"
  else
    echo "$name"
  fi
}

# Try to read license from a LICENSE file in the Go module cache
detect_go_license_from_cache() {
  local cache_path="$1"
  local lic_type="unknown"
  local lic_url=""

  for lic_file in LICENSE LICENSE.txt LICENSE.md COPYING; do
    if [ -f "$cache_path/$lic_file" ]; then
      lic_url="file://$cache_path/$lic_file"
      # Try common header patterns
      detected="$(head -5 "$cache_path/$lic_file" | grep -oiE '(MIT|Apache|BSD|GPL|LGPL|AGPL|MPL|ISC|Zlib|Unlicense|CC0|BSL|SSPL|BUSL|Boost)[^a-z]*[0-9.]*' | head -1 || true)"
      if [ -n "$detected" ]; then
        lic_type="$detected"
        break
      fi
      # Fallback: check first line keywords
      first_line="$(head -1 "$cache_path/$lic_file" 2>/dev/null)"
      case "$first_line" in
        *"MIT"*)       lic_type="MIT";        break ;;
        *"Apache"*)    lic_type="Apache-2.0"; break ;;
        *"BSD"*)       lic_type="BSD";        break ;;
        *"GNU"*"GPL"*) lic_type="GPL";        break ;;
        *"GPL"*)       lic_type="GPL";        break ;;
        *"LGPL"*)      lic_type="LGPL";       break ;;
        *"MPL"*)       lic_type="MPL-2.0";    break ;;
        *"ISC"*)       lic_type="ISC";        break ;;
        *"Unlicense"*) lic_type="Unlicense";  break ;;
        *"CC0"*)       lic_type="CC0-1.0";    break ;;
        *"Boost"*)     lic_type="BSL-1.0";    break ;;
      esac
    fi
  done

  echo "$lic_type|||$lic_url"
}

# --------------------------------------------------------------------------
# Begin report
# --------------------------------------------------------------------------

TIMESTAMP="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"

cat > "$REPORT_FILE" <<EOF
# RenOS License Audit Report

**Generated:** $TIMESTAMP
**Project License:** $PROJECT_LICENSE

**Classification:**
- **compatible** — Can be used freely (MIT, Apache-2.0, BSD, ISC, etc.)
- **conditional** — May have restrictions (MPL, LGPL, BSL-1.0)
- **incompatible** — Cannot be used (GPL, AGPL, SSPL, proprietary)
- **unknown** — License not detected or unrecognized

EOF

# ============================================================================
# Go Module License Scan
# ============================================================================

printf '\n## Go Module Licenses\n\n' >> "$REPORT_FILE"

GO_COMPATIBLE=0
GO_CONDITIONAL=0
GO_INCOMPATIBLE=0
GO_UNKNOWN=0

# Temp file for collecting Go entries
GO_ENTRIES_FILE="$(mktemp)"
trap 'rm -f "$GO_ENTRIES_FILE"' EXIT

for mod_dir in "${GO_MODULES[@]}"; do
  if [ ! -f "$mod_dir/go.mod" ]; then
    continue
  fi

  module_name="$(head -1 "$mod_dir/go.mod" | awk '{print $2}' 2>/dev/null || echo "unknown")"
  log "Scanning Go module: $module_name"

  # Get all dependencies
  if detect_tool go; then
    deps="$(cd "$mod_dir" && go list -m all 2>/dev/null)" || true
  else
    deps="$(parse_go_mod_deps "$mod_dir/go.mod")"
  fi

  while IFS= read -r dep; do
    [ -z "$dep" ] && continue

    dep_name="$(echo "$dep" | awk '{print $1}')"
    dep_ver="$(echo "$dep" | awk '{print $2}')"

    # Skip the module itself
    [ "$dep_name" = "$module_name" ] && continue
    [ -z "$dep_name" ] && continue

    lic_type="unknown"
    lic_url=""

    # Check Go module cache for LICENSE file
    if detect_tool go; then
      mod_cache="$(go env GOMODCACHE 2>/dev/null || echo "")"
      if [ -n "$mod_cache" ]; then
        cache_path="${mod_cache}/${dep_name}@${dep_ver}"
        if [ -d "$cache_path" ]; then
          result="$(detect_go_license_from_cache "$cache_path")"
          lic_type="${result%%|||*}"
          lic_url="${result##*|||}"
        fi
      fi
    fi

    classification="$(classify_license "$lic_type")"
    printf '%s|%s|%s|%s|%s\n' "$dep_name" "$dep_ver" "$lic_type" "$classification" "$lic_url" >> "$GO_ENTRIES_FILE"

    case "$classification" in
      compatible)   GO_COMPATIBLE=$((GO_COMPATIBLE + 1)) ;;
      conditional)  GO_CONDITIONAL=$((GO_CONDITIONAL + 1)) ;;
      incompatible) GO_INCOMPATIBLE=$((GO_INCOMPATIBLE + 1)); EXIT_CODE=2 ;;
      unknown)      GO_UNKNOWN=$((GO_UNKNOWN + 1)); [ "$EXIT_CODE" -eq 0 ] && EXIT_CODE=1 ;;
    esac
  done <<< "$deps"
done

# Write Go table
GO_TOTAL=$((GO_COMPATIBLE + GO_CONDITIONAL + GO_INCOMPATIBLE + GO_UNKNOWN))
{
  printf '| Dependency | Version | License | License URL | Classification |\n'
  printf '|---|---|---|---|---|\n'
  sort "$GO_ENTRIES_FILE" | while IFS='|' read -r name ver lic class url; do
    printf '| %s | %s | %s | %s | %s |\n' "$name" "$ver" "$lic" "$url" "$class"
  done
} >> "$REPORT_FILE"

# ============================================================================
# npm License Scan
# ============================================================================

printf '\n## npm Package Licenses\n\n' >> "$REPORT_FILE"

NPM_COMPATIBLE=0
NPM_CONDITIONAL=0
NPM_INCOMPATIBLE=0
NPM_UNKNOWN=0

NPM_ENTRIES_FILE="$(mktemp)"
trap 'rm -f "$NPM_ENTRIES_FILE"' EXIT

for npm_dir in "${NPM_DIRS[@]}"; do
  if [ ! -f "$npm_dir/package.json" ]; then
    continue
  fi

  pkg_name="$(node -e "console.log(require('${npm_dir}/package.json').name || 'unknown')" 2>/dev/null || echo "unknown")"
  log "Scanning npm package: $pkg_name"

  if detect_tool node; then
    deps_json="$(node -e "
      const p = require('${npm_dir}/package.json');
      const deps = {...(p.dependencies||{}), ...(p.devDependencies||{})};
      for (const [k,v] of Object.entries(deps)) console.log(k + '@' + v);
    " 2>/dev/null)" || true

    while IFS= read -r dep_line; do
      [ -z "$dep_line" ] && continue

      name="$(npm_pkg_name "$dep_line")"
      ver="$(npm_pkg_version "$dep_line")"

      lic_type="unknown"
      lic_url=""

      # Try to find installed package license from node_modules
      pkg_json="${npm_dir}/node_modules/${name}/package.json"
      if [ -f "$pkg_json" ]; then
        lic_info="$(node -e "
          try {
            const p = require('${pkg_json}');
            const lic = p.license || (p.licenses && p.licenses[0] && p.licenses[0].type) || 'unknown';
            const url = p.license && p.license.url ? p.license.url : '';
            console.log(lic + '|||' + url);
          } catch(e) { console.log('unknown|||'); }
        " 2>/dev/null || echo "unknown|||")"
        lic_type="${lic_info%%|||*}"
        lic_url="${lic_info##*|||}"
      fi

      classification="$(classify_license "$lic_type")"
      printf '%s|%s|%s|%s|%s\n' "$name" "$ver" "$lic_type" "$classification" "$lic_url" >> "$NPM_ENTRIES_FILE"

      case "$classification" in
        compatible)   NPM_COMPATIBLE=$((NPM_COMPATIBLE + 1)) ;;
        conditional)  NPM_CONDITIONAL=$((NPM_CONDITIONAL + 1)) ;;
        incompatible) NPM_INCOMPATIBLE=$((NPM_INCOMPATIBLE + 1)); EXIT_CODE=2 ;;
        unknown)      NPM_UNKNOWN=$((NPM_UNKNOWN + 1)); [ "$EXIT_CODE" -eq 0 ] && EXIT_CODE=1 ;;
      esac
    done <<< "$deps_json"
  fi
done

# Write npm table
NPM_TOTAL=$((NPM_COMPATIBLE + NPM_CONDITIONAL + NPM_INCOMPATIBLE + NPM_UNKNOWN))
{
  printf '| Dependency | Version | License | License URL | Classification |\n'
  printf '|---|---|---|---|---|\n'
  sort "$NPM_ENTRIES_FILE" | while IFS='|' read -r name ver lic class url; do
    printf '| %s | %s | %s | %s | %s |\n' "$name" "$ver" "$lic" "$url" "$class"
  done
} >> "$REPORT_FILE"

# ============================================================================
# Rust Crate License Scan
# ============================================================================

printf '\n## Rust Crate Licenses\n\n' >> "$REPORT_FILE"

CARGO_COMPATIBLE=0
CARGO_CONDITIONAL=0
CARGO_INCOMPATIBLE=0
CARGO_UNKNOWN=0

CARGO_ENTRIES_FILE="$(mktemp)"
trap 'rm -f "$CARGO_ENTRIES_FILE"' EXIT

for cargo_dir in "${CARGO_DIRS[@]}"; do
  if [ ! -f "$cargo_dir/Cargo.toml" ]; then
    continue
  fi

  crate_name="$(grep -m1 '^name' "$cargo_dir/Cargo.toml" 2>/dev/null | sed 's/.*"\(.*\)"/\1/')"
  log "Scanning Rust crate: $crate_name"

  # Parse dependencies from Cargo.toml
  in_deps=0
  while IFS= read -r line; do
    if [[ "$line" =~ ^\[dependencies\] ]]; then
      in_deps=1
      continue
    fi
    if [[ "$line" =~ ^\[ ]]; then
      in_deps=0
      continue
    fi
    if [ "$in_deps" -eq 1 ] && [[ "$line" =~ ^([a-zA-Z0-9_-]+) ]]; then
      dep_name="${BASH_REMATCH[1]}"
      dep_ver="$(echo "$line" | sed -n 's/.*version *= *"\([^"]*\)".*/\1/p' || echo "?")"

      lic_type="unknown"
      lic_url=""

      # Try to get license from cargo metadata
      if detect_tool cargo; then
        metadata="$(cd "$cargo_dir" && cargo metadata --format-version=1 2>/dev/null || true)"
        if [ -n "$metadata" ]; then
          lic_info="$(echo "$metadata" | python3 -c "
import sys, json
data = json.load(sys.stdin)
for pkg in data.get('packages', []):
    if pkg['name'] == '$dep_name':
        lic = pkg.get('license', '')
        if not lic:
            lic = 'unknown'
        print(lic)
        break
" 2>/dev/null || echo "unknown")"
          lic_type="$lic_info"
        fi
      fi

      classification="$(classify_license "$lic_type")"
      printf '%s|%s|%s|%s|%s\n' "$dep_name" "$dep_ver" "$lic_type" "$classification" "$lic_url" >> "$CARGO_ENTRIES_FILE"

      case "$classification" in
        compatible)   CARGO_COMPATIBLE=$((CARGO_COMPATIBLE + 1)) ;;
        conditional)  CARGO_CONDITIONAL=$((CARGO_CONDITIONAL + 1)) ;;
        incompatible) CARGO_INCOMPATIBLE=$((CARGO_INCOMPATIBLE + 1)); EXIT_CODE=2 ;;
        unknown)      CARGO_UNKNOWN=$((CARGO_UNKNOWN + 1)); [ "$EXIT_CODE" -eq 0 ] && EXIT_CODE=1 ;;
      esac
    fi
  done < "$cargo_dir/Cargo.toml"
done

# Write Cargo table
CARGO_TOTAL=$((CARGO_COMPATIBLE + CARGO_CONDITIONAL + CARGO_INCOMPATIBLE + CARGO_UNKNOWN))
{
  printf '| Dependency | Version | License | License URL | Classification |\n'
  printf '|---|---|---|---|---|\n'
  sort "$CARGO_ENTRIES_FILE" | while IFS='|' read -r name ver lic class url; do
    printf '| %s | %s | %s | %s | %s |\n' "$name" "$ver" "$lic" "$url" "$class"
  done
} >> "$REPORT_FILE"

# ============================================================================
# Summary Table
# ============================================================================

printf '\n## Summary\n\n' >> "$REPORT_FILE"

TOTAL_COMPATIBLE=$((GO_COMPATIBLE + NPM_COMPATIBLE + CARGO_COMPATIBLE))
TOTAL_CONDITIONAL=$((GO_CONDITIONAL + NPM_CONDITIONAL + CARGO_CONDITIONAL))
TOTAL_INCOMPATIBLE=$((GO_INCOMPATIBLE + NPM_INCOMPATIBLE + CARGO_INCOMPATIBLE))
TOTAL_UNKNOWN=$((GO_UNKNOWN + NPM_UNKNOWN + CARGO_UNKNOWN))
TOTAL_TOTAL=$((TOTAL_COMPATIBLE + TOTAL_CONDITIONAL + TOTAL_INCOMPATIBLE + TOTAL_UNKNOWN))

{
  printf '| Classification | Go | npm | Rust | Total |\n'
  printf '|---|---|---|---|---|\n'
  printf '| Compatible | %d | %d | %d | %d |\n' "$GO_COMPATIBLE" "$NPM_COMPATIBLE" "$CARGO_COMPATIBLE" "$TOTAL_COMPATIBLE"
  printf '| Conditional | %d | %d | %d | %d |\n' "$GO_CONDITIONAL" "$NPM_CONDITIONAL" "$CARGO_CONDITIONAL" "$TOTAL_CONDITIONAL"
  printf '| Incompatible | %d | %d | %d | %d |\n' "$GO_INCOMPATIBLE" "$NPM_INCOMPATIBLE" "$CARGO_INCOMPATIBLE" "$TOTAL_INCOMPATIBLE"
  printf '| Unknown | %d | %d | %d | %d |\n' "$GO_UNKNOWN" "$NPM_UNKNOWN" "$CARGO_UNKNOWN" "$TOTAL_UNKNOWN"
  printf '| **Total** | **%d** | **%d** | **%d** | **%d** |\n' "$((GO_COMPATIBLE + GO_CONDITIONAL + GO_INCOMPATIBLE + GO_UNKNOWN))" "$((NPM_COMPATIBLE + NPM_CONDITIONAL + NPM_INCOMPATIBLE + NPM_UNKNOWN))" "$((CARGO_COMPATIBLE + CARGO_CONDITIONAL + CARGO_INCOMPATIBLE + CARGO_UNKNOWN))" "$TOTAL_TOTAL"
} >> "$REPORT_FILE"

echo "" >> "$REPORT_FILE"

# Final status
if [ "$TOTAL_INCOMPATIBLE" -gt 0 ]; then
  {
    printf '> **❌ Incompatible licenses found.** These dependencies cannot be used with the %s project.\n' "$PROJECT_LICENSE"
    printf '>\n'
    printf '> Exit code: 2\n'
  } >> "$REPORT_FILE"
  EXIT_CODE=2
elif [ "$TOTAL_CONDITIONAL" -gt 0 ] || [ "$TOTAL_UNKNOWN" -gt 0 ]; then
  {
    printf '> **⚠️  Conditional or unknown licenses found.** Review may be required.\n'
    printf '>\n'
    printf '> Exit code: 1\n'
  } >> "$REPORT_FILE"
else
  {
    printf '> **✅ All licenses are compatible.** No issues detected.\n'
    printf '>\n'
    printf '> Exit code: 0\n'
  } >> "$REPORT_FILE"
fi

echo "" >> "$REPORT_FILE"

log "Report generated: $REPORT_FILE"
exit "$EXIT_CODE"
