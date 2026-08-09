#!/usr/bin/env bash
set -euo pipefail

VERSION="1.0.0"
BUILD_SCRIPT="$(cd "$(dirname "$0")" && pwd)/build-release.sh"
ARTIFACTS_DIR="$(cd "$(dirname "$0")/../artifacts" && pwd)"
WORK_DIR=$(mktemp -d)
FIX_MODE=false

BUILD1_DIR="${WORK_DIR}/build1"
BUILD2_DIR="${WORK_DIR}/build2"

cleanup() { rm -rf "$WORK_DIR"; }
trap cleanup EXIT

info()    { echo "[INFO]  $*"; }
pass()    { echo "[PASS]  $*"; }
fail()    { echo "[FAIL]  $*" >&2; }
fatal()   { echo "[FATAL] $*" >&2; exit 1; }

command_exists() { command -v "$1" &>/dev/null; }

check_prerequisites() {
  command_exists go         || fatal "go is required"
  command_exists cargo      || fatal "cargo is required"
  command_exists sha256sum  || fatal "sha256sum is required"
  command_exists diff       || fatal "diff is required"
}

apply_fixes() {
  if ! $FIX_MODE; then
    return
  fi

  info "Applying reproducibility fixes"

  # Fix 1: Fixed Go flags — strip build IDs
  GOFLAGS="${GOFLAGS:--trimpath} -ldflags=-buildid="
  export GOFLAGS

  # Fix 2: Fixed timestamps for Git embeds
  export GIT_COMMITTER_DATE="Thu, 1 Jan 1970 00:00:00 +0000"
  export GIT_AUTHOR_DATE="${GIT_AUTHOR_DATE:-$GIT_COMMITTER_DATE}"

  # Fix 3: Fixed source epoch
  export SOURCE_DATE_EPOCH=0

  # Fix 4: Disable CGO
  export CGO_ENABLED=0

  # Fix 5: Deterministic temp / cache dirs
  export TMPDIR="${WORK_DIR}/tmp"
  mkdir -p "$TMPDIR"
  export GOCACHE="${WORK_DIR}/gocache"
  export GOPATH="${WORK_DIR}/gopath"
  export GOMODCACHE="${WORK_DIR}/gopath/pkg/mod"

  # Fix 6: Rust reproducibility
  export CARGO_TARGET_DIR="${WORK_DIR}/cargo-target"
  export CARGO_INCREMENTAL=0
  RUSTFLAGS="${RUSTFLAGS:-} -C link-args=-Wl,--build-id=none"
  export RUSTFLAGS

  # Fix 7: Locale / timezone / umask
  export LANG=C.UTF-8
  export LC_ALL=C.UTF-8
  export TZ=UTC
  umask 0022

  # Fix 8: Ensure Rust targets are installed
  if command -v rustup &>/dev/null; then
    for t in \
      x86_64-unknown-linux-gnu \
      aarch64-unknown-linux-gnu \
      x86_64-apple-darwin \
      aarch64-apple-darwin \
      x86_64-pc-windows-gnu; do
      rustup target add "$t" 2>/dev/null || true
    done
  fi

  info "All reproducibility fixes applied"
}

build_and_checksum() {
  local build_dir="$1"
  local label="$2"

  mkdir -p "$build_dir"
  local artifacts_dir="${build_dir}/artifacts"
  mkdir -p "$artifacts_dir"

  local src_link="${build_dir}/src"
  local repo_root
  repo_root="$(cd "$(dirname "$BUILD_SCRIPT")/../.." && pwd)"
  ln -sfn "$repo_root" "$src_link"

  info "Running build $label..."
  (
    export SOURCE_DATE_EPOCH=0
    export GOPATH="${build_dir}/gopath"
    export GOCACHE="${build_dir}/gocache"
    export GOMODCACHE="${build_dir}/gopath/pkg/mod"
    export CARGO_TARGET_DIR="${build_dir}/cargo-target"
    export LANG=C.UTF-8
    export LC_ALL=C.UTF-8
    export TZ=UTC
    umask 0022

    cd "$src_link"
    bash "$BUILD_SCRIPT" 2>&1 | while IFS= read -r line; do
      echo "[${label}] $line"
    done

    local generated_artifacts="${repo_root}/release/artifacts"
    if [[ -d "$generated_artifacts" ]]; then
      cp -r "$generated_artifacts"/* "$artifacts_dir"/
    fi
  )

  local checksums
  checksums=$(find "$artifacts_dir" -name "*SHA256SUMS" -type f | head -1)
  if [[ -z "$checksums" ]]; then
    # fallback: build may have left artifacts in the repo artifacts dir
    local fallback="${repo_root}/release/artifacts"
    checksums=$(find "$fallback" -name "*SHA256SUMS" -type f | head -1 2>/dev/null || true)
    if [[ -z "$checksums" ]]; then
      fatal "No SHA256SUMS file found in build $label output"
    fi
    cp "$checksums" "$artifacts_dir"/
  fi

  local out="${build_dir}/checksums.sorted"
  grep -v '^#' "$checksums" | sort > "$out"
  echo "$out"
}

compare_builds() {
  local cs1="$1" cs2="$2"

  if [[ ! -f "$cs1" ]]; then
    fail "First build checksums not found: $cs1"
    return 1
  fi
  if [[ ! -f "$cs2" ]]; then
    fail "Second build checksums not found: $cs2"
    return 1
  fi

  local diff_output
  diff_output=$(diff --unified=0 "$cs1" "$cs2" || true)

  if [[ -z "$diff_output" ]]; then
    info ""
    info "=============================================="
    info "  ALL ARTIFACTS ARE REPRODUCIBLE"
    info "=============================================="
    return 0
  else
    info ""
    info "=============================================="
    info "  REPRODUCIBILITY CHECK FAILED"
    info "=============================================="
    echo "$diff_output"
    return 1
  fi
}

main() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --fix) FIX_MODE=true; shift ;;
      -h|--help)
        echo "Usage: $0 [--fix]"
        echo ""
        echo "Verifies that RenOS release builds are reproducible by building"
        echo "the same version twice and comparing checksums."
        echo ""
        echo "Options:"
        echo "  --fix    Apply known workarounds for common"
        echo "           non-reproducibility issues"
        echo "  -h, --help  Show this help message"
        exit 0
        ;;
      *) fatal "Unknown option: $1";;
    esac
  done

  check_prerequisites
  $FIX_MODE && apply_fixes

  info "RenOS Reproducible Build Verifier v${VERSION}"
  info "Build script:  ${BUILD_SCRIPT}"
  info "Work dir:      ${WORK_DIR}"
  info "Fix mode:      ${FIX_MODE}"
  info ""

  info "=== Build 1/2 ==="
  local cs1
  cs1=$(build_and_checksum "$BUILD1_DIR" "build1")
  info "Checksums: $cs1"

  info ""
  info "=== Build 2/2 ==="
  local cs2
  cs2=$(build_and_checksum "$BUILD2_DIR" "build2")
  info "Checksums: $cs2"

  info ""
  info "=== Comparing checksums ==="
  compare_builds "$cs1" "$cs2"
}

main "$@"
