#!/usr/bin/env bash
set -euo pipefail

VERSION="1.0.0"
REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
ARTIFACTS_DIR="${REPO_ROOT}/release/artifacts"
RELEASE_NOTES="${REPO_ROOT}/release/docs/release-notes.md"
DRY_RUN=false
GPG_SIGN=false
GPG_KEY_ID="${GPG_KEY_ID:-}"

usage() {
  cat <<-EOF
Usage: $0 <version-tag> [options]

Draft a GitHub release for RenOS.

Arguments:
  version-tag    Git tag for the release (e.g. v1.0.0)

Options:
  --artifacts-dir=path   Path to artifacts directory (default: release/artifacts)
  --notes=path           Path to release notes file (default: release/docs/release-notes.md)
  --gpg-key=id           GPG key to sign the tag (default: \$GPG_KEY_ID)
  --gpg-sign             Sign the git tag with GPG
  --dry-run              Print actions without executing
  -h, --help             Show this help message
EOF
  exit 0
}

info()  { echo "[INFO]  $*"; }
warn()  { echo "[WARN]  $*" >&2; }
fatal() { echo "[FATAL] $*" >&2; exit 1; }

command_exists() { command -v "$1" &>/dev/null; }

check_prerequisites() {
  command_exists gh    || fatal "gh CLI is required (https://cli.github.com)"
  command_exists git   || fatal "git is required"
  command_exists gpg   || fatal "gpg is required"

  if ! gh auth status 2>&1 | grep -q "Logged in"; then
    fatal "Not logged into GitHub CLI. Run 'gh auth login' first."
  fi
}

validate_tag() {
  local tag="$1"
  if ! echo "$tag" | grep -qP '^v\d+\.\d+\.\d+(-rc\d+)?$'; then
    fatal "Invalid tag format: $tag (expected vX.Y.Z or vX.Y.Z-rcN)"
  fi
}

extract_release_notes() {
  local tag="$1"
  local version="${tag#v}"

  if [[ ! -f "$RELEASE_NOTES" ]]; then
    warn "Release notes file not found: $RELEASE_NOTES"
    echo "## RenOS ${tag}" > /tmp/renos-release-notes.md
    echo "" >> /tmp/renos-release-notes.md
    echo "Release artifacts for RenOS ${version}." >> /tmp/renos-release-notes.md
    echo "/tmp/renos-release-notes.md"
    return
  fi

  local tmpfile
  tmpfile=$(mktemp)
  awk "/^## \\[${version}\\]/,/^## /" "$RELEASE_NOTES" | head -n -1 > "$tmpfile"

  if [[ ! -s "$tmpfile" ]]; then
    warn "No release notes found for version ${version} in ${RELEASE_NOTES}"
    echo "## RenOS ${tag}" > "$tmpfile"
    echo "" >> "$tmpfile"
  fi

  echo "$tmpfile"
}

gather_artifacts() {
  if [[ ! -d "$ARTIFACTS_DIR" ]]; then
    fatal "Artifacts directory not found: $ARTIFACTS_DIR"
  fi

  local artifacts=()
  while IFS= read -r -d '' f; do
    artifacts+=("$f")
  done < <(find "$ARTIFACTS_DIR" -maxdepth 1 -type f \( -name "*.tar.gz" -o -name "*-SHA256SUMS*" -o -name "build-info.json" \) -print0)

  if [[ ${#artifacts[@]} -eq 0 ]]; then
    fatal "No release artifacts found in ${ARTIFACTS_DIR}"
  fi

  printf '%s\0' "${artifacts[@]}"
}

run() {
  if $DRY_RUN; then
    echo "[DRY-RUN]" "$@"
    return 0
  fi
  "$@"
}

main() {
  local tag=""

  while [[ $# -gt 0 ]]; do
    case "$1" in
      -h|--help) usage ;;
      --dry-run) DRY_RUN=true; shift ;;
      --gpg-sign) GPG_SIGN=true; shift ;;
      --gpg-key=*) GPG_KEY_ID="${1#*=}"; shift ;;
      --gpg-key) GPG_KEY_ID="$2"; shift 2 ;;
      --artifacts-dir=*) ARTIFACTS_DIR="${1#*=}"; shift ;;
      --artifacts-dir) ARTIFACTS_DIR="$2"; shift 2 ;;
      --notes=*) RELEASE_NOTES="${1#*=}"; shift ;;
      --notes) RELEASE_NOTES="$2"; shift 2 ;;
      -*)
        fatal "Unknown option: $1"
        ;;
      *)
        if [[ -z "$tag" ]]; then
          tag="$1"
          shift
        else
          fatal "Unexpected argument: $1"
        fi
        ;;
    esac
  done

  if [[ -z "$tag" ]]; then
    fatal "Usage: $0 <version-tag>"
  fi

  check_prerequisites
  validate_tag "$tag"
  ARTIFACTS_DIR="$(cd "$ARTIFACTS_DIR" 2>/dev/null && pwd)" || fatal "Artifacts dir not found: $ARTIFACTS_DIR"
  RELEASE_NOTES="$(cd "$(dirname "$RELEASE_NOTES")" 2>/dev/null && pwd)/$(basename "$RELEASE_NOTES")" || true

  info "RenOS Release Drafter"
  info "Tag:          ${tag}"
  info "Artifacts:    ${ARTIFACTS_DIR}"
  info "Notes:        ${RELEASE_NOTES}"
  info "GPG sign:     ${GPG_SIGN}"
  info "GPG key:      ${GPG_KEY_ID:-default}"
  $DRY_RUN && info "*** DRY RUN — no changes will be made ***"
  info ""

  # --- Gather release artifacts ---
  info "Gathering artifacts..."
  local artifacts_map
  artifacts_map=$(gather_artifacts)

  # --- Create release notes ---
  info "Preparing release notes..."
  local notes_file
  notes_file=$(extract_release_notes "$tag")

  # --- Create and push git tag ---
  if git rev-parse "$tag" &>/dev/null; then
    info "Tag $tag already exists locally"
  else
    info "Creating git tag $tag..."
    if $GPG_SIGN; then
      local gpg_flag=""
      [[ -n "$GPG_KEY_ID" ]] && gpg_flag="-u $GPG_KEY_ID"
      run git tag $gpg_flag -s "$tag" -m "RenOS release $tag"
    else
      run git tag "$tag" -m "RenOS release $tag"
    fi
    run git push origin "$tag"
  fi

  # --- Build artifact list for gh release ---
  local artifact_args=()
  if [[ ${#artifacts_map[@]} -gt 0 ]]; then
    while IFS= read -r -d '' artifact; do
      artifact_args+=("$artifact")
    done < <(printf "%s" "$artifacts_map")
  fi

  # --- Create GitHub Release ---
  info "Creating GitHub release $tag..."
  if [[ ${#artifact_args[@]} -gt 0 ]]; then
    run gh release create "$tag" \
      --title "RenOS $tag" \
      --notes-file "$notes_file" \
      "${artifact_args[@]}"
  else
    run gh release create "$tag" \
      --title "RenOS $tag" \
      --notes-file "$notes_file"
  fi

  info ""
  info "=== Release $tag drafted ==="

  run rm -f "$notes_file"
}

main "$@"
