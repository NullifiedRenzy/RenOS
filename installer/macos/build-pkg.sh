#!/bin/bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
INSTALLER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DIST_DIR="${INSTALLER_DIR}/dist"
PKG_ROOT="${INSTALLER_DIR}/pkg-root"
VERSION="1.0.0"
IDENTIFIER="com.renos.agent"
PKG_NAME="RenOS-Agent-${VERSION}.pkg"

SIGN_IDENTITY="${PKG_SIGN_IDENTITY:-}"
DEV_TEAM="${PKG_DEV_TEAM:-}"

log() { printf "\033[1;32m[build]\033[0m %s\n" "$*"; }
warn() { printf "\033[1;33m[warning]\033[0m %s\n" "$*"; }
err()  { printf "\033[1;31m[error]\033[0m   %s\n" "$*" >&2; }

cleanup() {
    rm -rf "${PKG_ROOT}" "${INSTALLER_DIR}/component.plist"
}
trap cleanup EXIT

check_dependencies() {
    local missing=0
    for cmd in pkgbuild productbuild codesign; do
        if ! command -v "$cmd" &>/dev/null; then
            err "$cmd not found. Install Xcode Command Line Tools."
            missing=1
        fi
    done
    if [[ "$missing" -ne 0 ]]; then
        exit 1
    fi
}

stage_artifacts() {
    log "Staging built artifacts into pkg-root..."

    mkdir -p "${PKG_ROOT}/Library/Application Support/RenOS"
    mkdir -p "${PKG_ROOT}/Library/LaunchAgents"
    mkdir -p "${PKG_ROOT}/private/var/log/renos"

    if [[ -d "${PROJECT_ROOT}/build" ]]; then
        cp -R "${PROJECT_ROOT}/build/"* "${PKG_ROOT}/Library/Application Support/RenOS/"
    else
        warn "No build/ directory found at ${PROJECT_ROOT}/build."
        warn "Creating placeholder binary for structure validation."
        echo '#!/bin/sh' > "${PKG_ROOT}/Library/Application Support/RenOS/renos-agent"
        echo 'echo "RenOS Agent placeholder"' >> "${PKG_ROOT}/Library/Application Support/RenOS/renos-agent"
        chmod 755 "${PKG_ROOT}/Library/Application Support/RenOS/renos-agent"
    fi

    cp "${INSTALLER_DIR}/com.renos.agent.plist" \
        "${PKG_ROOT}/Library/LaunchAgents/com.renos.agent.plist"

    cp "${INSTALLER_DIR}/keychain-setup.sh" \
        "${PKG_ROOT}/Library/Application Support/RenOS/keychain-setup.sh"
    chmod 755 "${PKG_ROOT}/Library/Application Support/RenOS/keychain-setup.sh"

    chmod 755 "${PKG_ROOT}/Library/Application Support/RenOS/renos-agent"
    chmod 644 "${PKG_ROOT}/Library/LaunchAgents/com.renos.agent.plist"
    chmod 755 "${PKG_ROOT}/private/var/log/renos"
}

sign_artifacts() {
    if [[ -z "$SIGN_IDENTITY" ]]; then
        warn "PKG_SIGN_IDENTITY not set — skipping codesign."
        return
    fi

    log "Signing binaries with identity: ${SIGN_IDENTITY}"

    while IFS= read -r -d '' bin; do
        if file "$bin" | grep -qiE "Mach-O|executable"; then
            codesign -s "$SIGN_IDENTITY" \
                --timestamp \
                --options runtime \
                --prefix "${IDENTIFIER}." \
                -f "$bin" 2>&1 | sed 's/^/  /'
        fi
    done < <(find "${PKG_ROOT}" -type f -perm +111 -print0 2>/dev/null || true)

    codesign -s "$SIGN_IDENTITY" \
        --timestamp \
        --options runtime \
        -f "${PKG_ROOT}/Library/Application Support/RenOS/keychain-setup.sh" 2>&1 | sed 's/^/  /' || true
}

build_component_pkg() {
    log "Building component package..."

    pkgbuild --analyze \
        --root "${PKG_ROOT}" \
        "${INSTALLER_DIR}/component.plist"

    pkgbuild --root "${PKG_ROOT}" \
        --component-plist "${INSTALLER_DIR}/component.plist" \
        --identifier "${IDENTIFIER}.core" \
        --install-location "/" \
        --version "${VERSION}" \
        --scripts "${INSTALLER_DIR}" \
        "${DIST_DIR}/RenOS-Agent-${VERSION}-component.pkg"

    log "Component package created."
}

build_distribution() {
    log "Building distribution package..."

    local sign_flags=()
    if [[ -n "$SIGN_IDENTITY" ]]; then
        sign_flags+=(--sign "$SIGN_IDENTITY")
        if [[ -n "$DEV_TEAM" ]]; then
            sign_flags+=(--keychain "/Library/Keychains/System.keychain")
        fi
    fi

    productbuild \
        --distribution "${INSTALLER_DIR}/Distribution.xml" \
        --package-path "${DIST_DIR}" \
        --resources "${INSTALLER_DIR}/Resources" \
        --identifier "${IDENTIFIER}" \
        --version "${VERSION}" \
        "${sign_flags[@]}" \
        "${DIST_DIR}/${PKG_NAME}"

    log "Distribution package created: ${DIST_DIR}/${PKG_NAME}"
}

create_resources() {
    local res="${INSTALLER_DIR}/Resources"
    mkdir -p "$res"

    for lang_dir in "$res"/*/; do
        local lang
        lang=$(basename "$lang_dir")
        local missing=0
        for f in Welcome.html ReadMe.html License.html Conclusion.html; do
            [[ -f "${lang_dir}/${f}" ]] || { warn "Missing ${lang_dir}/${f}"; missing=1; }
        done
    done

    if ! ls "$res"/*/Welcome.html &>/dev/null 2>&1; then
        warn "No localized resources found. Creating default English resources."
        mkdir -p "${res}/en.lproj"

        cat > "${res}/en.lproj/Welcome.html" <<-WELCOME
<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>Welcome</title></head>
<body style="font-family: -apple-system, sans-serif; font-size: 11pt;">
<h2>Welcome to RenOS Agent</h2>
<p>This installer will guide you through installing RenOS Agent on your Mac.</p>
<p>RenOS Agent enables secure device enrollment, credential management,
and remote management capabilities for your organization.</p>
</body></html>
WELCOME

        cp "${res}/en.lproj/Welcome.html" "${res}/en.lproj/ReadMe.html"
        cp "${res}/en.lproj/Welcome.html" "${res}/en.lproj/License.html"
        cp "${res}/en.lproj/Welcome.html" "${res}/en.lproj/Conclusion.html"
    fi
}

verify_pkg() {
    log "Verifying package..."
    if command -v pkgutil &>/dev/null; then
        pkgutil --check-signature "${DIST_DIR}/${PKG_NAME}" 2>&1 | sed 's/^/  /' || true
    fi
    ls -lh "${DIST_DIR}/${PKG_NAME}"
}

main() {
    log "Building RenOS Agent PKG installer v${VERSION}"

    mkdir -p "$DIST_DIR"
    check_dependencies
    create_resources
    stage_artifacts
    sign_artifacts
    build_component_pkg
    build_distribution
    verify_pkg

    log "Done. PKG available at ${DIST_DIR}/${PKG_NAME}"
}

main "$@"
