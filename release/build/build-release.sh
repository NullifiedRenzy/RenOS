#!/usr/bin/env bash
set -euo pipefail

VERSION="1.0.0"
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH:-$(date -u +%s)}"
export SOURCE_DATE_EPOCH

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
OUTPUT_DIR="${REPO_ROOT}/release/artifacts"
BUILD_DIR="${OUTPUT_DIR}/_build"
DRY_RUN=false
GPG_KEY_ID="${GPG_KEY_ID:-}"
PROJECT="renos"

# target  ->  Go os/arch
declare -A TARGETS=(
  ["linux-amd64"]="linux amd64"
  ["linux-arm64"]="linux arm64"
  ["darwin-amd64"]="darwin amd64"
  ["darwin-arm64"]="darwin arm64"
  ["windows-amd64"]="windows amd64"
)

# target  ->  Rust triple
declare -A RUST_TARGETS=(
  ["linux-amd64"]="x86_64-unknown-linux-gnu"
  ["linux-arm64"]="aarch64-unknown-linux-gnu"
  ["darwin-amd64"]="x86_64-apple-darwin"
  ["darwin-arm64"]="aarch64-apple-darwin"
  ["windows-amd64"]="x86_64-pc-windows-gnu"
)

# component -> source directory (relative to REPO_ROOT)
declare -A COMPONENT_SRC=(
  ["core"]="core"
  ["renctl"]="renctl"
  ["renos-plugin"]="core"
  ["agent"]="agent"
)

# component -> main package path (relative to component source dir)
declare -A COMPONENT_MAIN=(
  ["core"]="./cmd/renos-core"
  ["renctl"]="."
  ["renos-plugin"]="./cmd/renos-plugin"
  ["agent"]=""
)

# component -> binary name
declare -A COMPONENT_BIN=(
  ["core"]="renos-core"
  ["renctl"]="renctl"
  ["renos-plugin"]="renos-plugin"
  ["agent"]="renos-agent"
)

# component -> language
declare -A COMPONENT_LANG=(
  ["core"]="go"
  ["renctl"]="go"
  ["renos-plugin"]="go"
  ["agent"]="rust"
)

# which platforms to build each component for
# format: space-separated target keys
declare -A COMPONENT_PLATFORMS=(
  ["core"]="linux-amd64 linux-arm64"
  ["renctl"]="linux-amd64 linux-arm64"
  ["renos-plugin"]="linux-amd64 linux-arm64"
  ["agent"]="linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64"
)

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run) DRY_RUN=true; shift ;;
    --gpg-key=*) GPG_KEY_ID="${1#*=}"; shift ;;
    --gpg-key) GPG_KEY_ID="$2"; shift 2 ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

cleanup() { rm -rf "$BUILD_DIR"; }
trap cleanup EXIT

info()  { echo "[INFO]  $*"; }
warn()  { echo "[WARN]  $*" >&2; }
fatal() { echo "[FATAL] $*" >&2; exit 1; }

command_exists() { command -v "$1" &>/dev/null; }

check_prerequisites() {
  command_exists go         || fatal "go is required"
  command_exists cargo      || fatal "cargo is required"
  command_exists gpg        || fatal "gpg is required"
  command_exists sha256sum  || fatal "sha256sum is required"
  command_exists tar        || fatal "tar is required"

  local go_version
  go_version=$(go version | grep -oP 'go(\d+\.\d+)' | head -1 | tr -d 'go' || true)
  if [[ -n "$go_version" ]]; then
    IFS='.' read -r go_major go_minor <<< "$go_version"
    if [[ "$go_major" -lt 1 ]] || { [[ "$go_major" -eq 1 ]] && [[ "$go_minor" -lt 21 ]]; }; then
      fatal "Go 1.21+ required (found $go_major.$go_minor)"
    fi
  fi
}

setup_signing() {
  if [[ -n "$GPG_KEY_ID" ]]; then
    if ! gpg --list-keys "$GPG_KEY_ID" &>/dev/null; then
      fatal "GPG key $GPG_KEY_ID not found in keyring"
    fi
    info "Using GPG key: $GPG_KEY_ID"
    return
  fi
  if ! gpg --list-keys "RENOS_RELEASE" &>/dev/null; then
    info "No GPG_KEY_ID set and no RENOS_RELEASE key found. Generating ephemeral release key."
    cat > /tmp/gpg_batch.$$ <<-EOF
      %echo Generating RenOS release signing key
      Key-Type: RSA
      Key-Length: 4096
      Subkey-Type: RSA
      Subkey-Length: 4096
      Name-Real: RenOS Release
      Name-Email: release@renos.local
      Expire-Date: 0
      %no-protection
      %commit
      %echo Done
EOF
    gpg --batch --gen-key /tmp/gpg_batch.$$
    rm -f /tmp/gpg_batch.$$
  fi
  GPG_KEY_ID=$(gpg --list-keys --with-colons "RENOS_RELEASE" 2>/dev/null | awk -F: '/^pub:/ {print $5; exit}')
  info "Using auto-generated GPG key: $GPG_KEY_ID"
}

run() {
  if $DRY_RUN; then
    echo "[DRY-RUN]" "$@"
    return 0
  fi
  "$@"
}

build_go_component() {
  local component="$1" os="$2" arch="$3" output_dir="$4"
  local binary="${COMPONENT_BIN[$component]}"
  [[ "$os" == "windows" ]] && binary="${binary}.exe"

  local src_dir="${REPO_ROOT}/${COMPONENT_SRC[$component]}"
  local main_pkg="${COMPONENT_MAIN[$component]}"

  if [[ ! -d "$src_dir" ]]; then
    warn "Source directory not found: $src_dir — skipping $component for $os/$arch"
    return 1
  fi

  local ldflags="-s -w"
  case "$component" in
    core|renos-plugin)
      ldflags+=" -X github.com/renos-project/renos/core/api.Version=${VERSION}"
      ldflags+=" -X github.com/renos-project/renos/core/api.Commit=${COMMIT}"
      ldflags+=" -X github.com/renos-project/renos/core/api.BuildTime=${BUILD_TIME}"
      ;;
    renctl)
      ldflags+=" -X main.version=${VERSION}"
      ldflags+=" -X main.commit=${COMMIT}"
      ldflags+=" -X main.buildTime=${BUILD_TIME}"
      ;;
  esac

  info "Building $component ($binary) for $os/$arch"
  run env \
    GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 \
    go build -trimpath -ldflags "$ldflags" -buildvcs=false \
    -o "${output_dir}/${binary}" "${main_pkg}"
}

build_agent_rust() {
  local os="$1" arch="$2" output_dir="$3"
  local target_key="${os}-${arch}"
  local rust_target="${RUST_TARGETS[$target_key]}"
  local binary="${COMPONENT_BIN[agent]}"
  [[ "$os" == "windows" ]] && binary="${binary}.exe"

  if [[ -z "$rust_target" ]]; then
    fatal "Unsupported Rust target: $os/$arch"
  fi

  local src_dir="${REPO_ROOT}/agent"
  if [[ ! -d "$src_dir" ]]; then
    warn "Agent source directory not found: $src_dir — skipping"
    return 1
  fi

  info "Building agent (Rust) for $os/$arch (target: $rust_target)"

  run cargo build --release --target "$rust_target" \
    --manifest-path "${src_dir}/Cargo.toml"

  local built="${src_dir}/target/${rust_target}/release/${binary}"
  if [[ -f "$built" ]]; then
    cp "$built" "${output_dir}/${binary}"
  else
    fatal "Rust build produced no binary at $built"
  fi
}

build_component() {
  local component="$1" target_key="$2" output_dir="$3"
  IFS=' ' read -r os arch <<< "${TARGETS[$target_key]}"

  if [[ "${COMPONENT_LANG[$component]}" == "rust" ]]; then
    build_agent_rust "$os" "$arch" "$output_dir"
  else
    build_go_component "$component" "$os" "$arch" "$output_dir"
  fi
}

sign_binary() {
  local file="$1"
  if [[ ! -f "$file" ]]; then
    warn "File not found for signing: $file"
    return 1
  fi
  info "Signing $file"
  run gpg --detach-sign --armor --default-key "$GPG_KEY_ID" "$file"
}

create_tarball() {
  local component="$1" target_key="$2" dir="$3"
  IFS=' ' read -r os arch <<< "${TARGETS[$target_key]}"
  local tarball_name="${PROJECT}-${component}-${VERSION}-${os}-${arch}.tar.gz"
  local tarball_path="${OUTPUT_DIR}/${tarball_name}"

  if [[ ! -d "$dir" ]] || [[ -z "$(ls -A "$dir" 2>/dev/null)" ]]; then
    warn "No artifacts in $dir — skipping tarball"
    return 1
  fi

  info "Creating tarball: $tarball_name"
  run tar -czf "$tarball_path" -C "$(dirname "$dir")" "$(basename "$dir")"
  echo "$tarball_path"
}

generate_checksums() {
  local checksums_file="${OUTPUT_DIR}/${PROJECT}-${VERSION}-SHA256SUMS"
  info "Generating checksums: $checksums_file"

  {
    echo "# RenOS ${VERSION} - SHA256 Checksums"
    echo "# Generated: ${BUILD_TIME}"
    echo "# Signed with: GPG key ${GPG_KEY_ID}"
    echo ""
  } > "$checksums_file"

  cd "$OUTPUT_DIR"
  for f in *.tar.gz *.asc; do
    [[ -f "$f" ]] || continue
    sha256sum "$f" >> "$checksums_file"
  done
  cd "$REPO_ROOT"

  info "Signing checksums file"
  run gpg --detach-sign --armor --default-key "$GPG_KEY_ID" "$checksums_file"
  echo "$checksums_file"
}

generate_build_info() {
  local info_file="${OUTPUT_DIR}/build-info.json"
  info "Generating build metadata: $info_file"

  cat > "$info_file" <<-EOF
{
  "project": "${PROJECT}",
  "version": "${VERSION}",
  "commit": "${COMMIT}",
  "buildTime": "${BUILD_TIME}",
  "sourceDateEpoch": ${SOURCE_DATE_EPOCH},
  "gpgKeyId": "${GPG_KEY_ID}",
  "artifacts": [
EOF

  local first=true
  cd "$OUTPUT_DIR"
  for f in *.tar.gz; do
    [[ -f "$f" ]] || continue
    $first || echo "," >> "$info_file"
    first=false
    local size
    size=$(stat -c%s "$f" 2>/dev/null || stat -f%z "$f" 2>/dev/null)
    local sha256
    sha256=$(sha256sum "$f" | cut -d' ' -f1)
    cat >> "$info_file" <<-PART
    {
      "file": "${f}",
      "size": ${size},
      "sha256": "${sha256}"
    }
PART
  done
  cd "$REPO_ROOT"

  cat >> "$info_file" <<-EOF
  ]
}
EOF
  echo "$info_file"
}

main() {
  mkdir -p "$OUTPUT_DIR" "$BUILD_DIR"

  check_prerequisites
  setup_signing

  info "RenOS Release Builder v${VERSION}"
  info "Commit:       ${COMMIT}"
  info "Build time:   ${BUILD_TIME}"
  info "Source Epoch: ${SOURCE_DATE_EPOCH}"
  info "Output:       ${OUTPUT_DIR}"
  $DRY_RUN && info "*** DRY RUN — no files will be written ***"

  for component in core renctl renos-plugin agent; do
    IFS=' ' read -r -a platforms <<< "${COMPONENT_PLATFORMS[$component]}"
    for target_key in "${platforms[@]}"; do
      IFS=' ' read -r os arch <<< "${TARGETS[$target_key]}"
      local outdir="${BUILD_DIR}/${target_key}/${component}"
      mkdir -p "$outdir"
      if build_component "$component" "$target_key" "$outdir"; then
        for f in "$outdir"/*; do
          [[ -f "$f" ]] && sign_binary "$f"
        done
        create_tarball "$component" "$target_key" "$outdir" || true
      fi
    done
  done

  generate_checksums
  generate_build_info

  info ""
  info "=== Build complete ==="
  info "Artifacts: ${OUTPUT_DIR}"
  ls -lh "$OUTPUT_DIR"/*.tar.gz "$OUTPUT_DIR"/*-SHA256SUMS* "$OUTPUT_DIR"/build-info.json 2>/dev/null || true
}

main "$@"
