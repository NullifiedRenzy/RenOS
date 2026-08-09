#!/bin/bash
#
# RenOS Agent — Post-Installation Script
# Runs as root after package installation.
#
set -euo pipefail

RENOS_SUPPORT="/Library/Application Support/RenOS"
RENOS_LOG="/var/log/renos"
LAUNCH_AGENT_DIR="/Library/LaunchAgents"
LAUNCH_AGENT_PLIST="${LAUNCH_AGENT_DIR}/com.renos.agent.plist"
RENOS_BIN="${RENOS_SUPPORT}/renos-agent"
KEYCHAIN_SCRIPT="${RENOS_SUPPORT}/keychain-setup.sh"
CONFIG_FILE="${RENOS_SUPPORT}/config.json"
ENROLLMENT_STAMP="${RENOS_SUPPORT}/.enrolled"

log()   { printf "[postinstall] %s\n" "$*"; }
warn()  { printf "[postinstall] WARNING: %s\n" "$*" >&2; }
error() { printf "[postinstall] ERROR: %s\n" "$*" >&2; }

# ---------------------------------------------------------------
# 1. Create directory structure
# ---------------------------------------------------------------
setup_directories() {
    log "Creating directory structure..."

    mkdir -p "${RENOS_SUPPORT}"
    mkdir -p "${RENOS_SUPPORT}/certs"
    mkdir -p "${RENOS_SUPPORT}/config"
    mkdir -p "${RENOS_SUPPORT}/state"
    mkdir -p "${RENOS_LOG}"

    # Restrict permissions
    chown root:wheel "${RENOS_SUPPORT}"
    chmod 755 "${RENOS_SUPPORT}"

    chown root:wheel "${RENOS_LOG}"
    chmod 755 "${RENOS_LOG}"
}

# ---------------------------------------------------------------
# 2. Install LaunchAgent
# ---------------------------------------------------------------
install_launchagent() {
    log "Installing LaunchAgent plist..."

    if [[ -f "${LAUNCH_AGENT_PLIST}" ]]; then
        # Unload existing if running
        launchctl bootout system "${LAUNCH_AGENT_PLIST}" 2>/dev/null || true
    fi

    chown root:wheel "${LAUNCH_AGENT_PLIST}"
    chmod 644 "${LAUNCH_AGENT_PLIST}"
}

# ---------------------------------------------------------------
# 3. Configure firewall rules
# ---------------------------------------------------------------
setup_firewall() {
    log "Configuring firewall rules..."

    local SOCKET_FW="/usr/libexec/ApplicationFirewall/socketfilterfw"

    if [[ ! -x "${SOCKET_FW}" ]]; then
        warn "socketfilterfw not found; skipping firewall configuration."
        return
    fi

    # Add renos-agent to firewall whitelist
    if [[ -f "${RENOS_BIN}" ]]; then
        "${SOCKET_FW}" --add "${RENOS_BIN}" 2>/dev/null || true

        # Allow incoming connections for management
        "${SOCKET_FW}" --unblockapp "${RENOS_BIN}" 2>/dev/null || true

        log "Firewall rule added for ${RENOS_BIN}"
    fi

    # Ensure firewall is enabled
    local fw_state
    fw_state=$("${SOCKET_FW}" --getglobalstate 2>/dev/null | grep -o "ENABLED" || echo "DISABLED")
    if [[ "${fw_state}" == "DISABLED" ]]; then
        warn "macOS Application Firewall is disabled. Enable it for security:"
        warn "  ${SOCKET_FW} --setglobalstate on"
    fi
}

# ---------------------------------------------------------------
# 4. Register with Keychain
# ---------------------------------------------------------------
setup_keychain() {
    log "Setting up Keychain integration..."

    if [[ -f "${KEYCHAIN_SCRIPT}" ]]; then
        chmod 755 "${KEYCHAIN_SCRIPT}"

        # Source the script to make its functions available
        # shellcheck source=/dev/null
        source "${KEYCHAIN_SCRIPT}"

        # Perform initial keychain setup
        local enrolled=""
        if command -v store_credential &>/dev/null; then
            enrolled=$(store_credential \
                "com.renos.agent" \
                "enrollment-pending" \
                "$(hostname -f 2>/dev/null || echo "unknown")" \
                "device-$(uuidgen 2>/dev/null || echo "$(date +%s)")" 2>/dev/null || true)

            if [[ -n "${enrolled}" ]]; then
                log "Initial keychain credential stored: ${enrolled}"
            fi
        fi
    else
        warn "keychain-setup.sh not found at expected path: ${KEYCHAIN_SCRIPT}"
    fi
}

# ---------------------------------------------------------------
# 5. Write default configuration
# ---------------------------------------------------------------
setup_config() {
    log "Creating default configuration..."

    if [[ ! -f "${CONFIG_FILE}" ]]; then
        local MAC_ADDR
        MAC_ADDR=$(ifconfig en0 ether 2>/dev/null | awk '/ether/ {print $2}' || echo "00:00:00:00:00:00")
        local SERIAL
        SERIAL=$(ioreg -c IOPlatformExpertDevice -d 2 | awk -F'"' '/IOPlatformSerialNumber/ {print $4}' || echo "UNKNOWN")
        local UUID
        UUID=$(uuidgen 2>/dev/null || echo "00000000-0000-0000-0000-000000000000")

        cat > "${CONFIG_FILE}" <<-CONFIG
{
    "version": "1.0.0",
    "device_id": "${UUID}",
    "serial_number": "${SERIAL}",
    "mac_address": "${MAC_ADDR}",
    "server_url": "https://api.renos.local",
    "enrolled": false,
    "log_level": "info",
    "log_directory": "${RENOS_LOG}",
    "poll_interval_seconds": 300,
    "tls_cert_path": "${RENOS_SUPPORT}/certs/client.pem",
    "tls_key_path": "${RENOS_SUPPORT}/certs/client-key.pem",
    "keychain_service": "com.renos.agent",
    "auto_update": true,
    "update_url": "https://updates.renos.local/agent"
}
CONFIG
        chown root:wheel "${CONFIG_FILE}"
        chmod 600 "${CONFIG_FILE}"
        log "Default configuration written to ${CONFIG_FILE}"
    else
        log "Configuration already exists at ${CONFIG_FILE}; preserving."
    fi
}

# ---------------------------------------------------------------
# 6. Fix ownership & permissions
# ---------------------------------------------------------------
fix_permissions() {
    log "Fixing permissions..."

    chown -R root:wheel "${RENOS_SUPPORT}"
    chmod 755 "${RENOS_SUPPORT}"
    find "${RENOS_SUPPORT}" -type d -exec chmod 755 {} +
    find "${RENOS_SUPPORT}" -type f -name "*.sh" -exec chmod 755 {} +
    find "${RENOS_SUPPORT}" -type f -name "*.json" -exec chmod 600 {} +
    chmod 644 "${LAUNCH_AGENT_PLIST}"
    chmod 755 "${RENOS_LOG}"
}

# ---------------------------------------------------------------
# 7. Start the LaunchAgent
# ---------------------------------------------------------------
start_agent() {
    log "Starting RenOS Agent LaunchAgent..."

    if [[ -f "${LAUNCH_AGENT_PLIST}" ]]; then
        launchctl bootstrap system "${LAUNCH_AGENT_PLIST}" 2>/dev/null || \
            launchctl load -w "${LAUNCH_AGENT_PLIST}" 2>/dev/null || \
            warn "Could not bootstrap launch agent (may already be loaded)."

        # Verify it's running
        sleep 1
        if launchctl list com.renos.agent &>/dev/null; then
            log "RenOS Agent LaunchAgent started successfully."
        else
            warn "RenOS Agent LaunchAgent did not start. Check logs at ${RENOS_LOG}"
        fi
    else
        warn "LaunchAgent plist not found at ${LAUNCH_AGENT_PLIST}; not starting."
    fi
}

# ---------------------------------------------------------------
# 8. Run device enrollment flow
# ---------------------------------------------------------------
run_enrollment() {
    log "Running device enrollment flow..."

    # Do not re-enroll if already enrolled
    if [[ -f "${ENROLLMENT_STAMP}" ]]; then
        log "Device already enrolled (stamp found at ${ENROLLMENT_STAMP}). Skipping."
        return
    fi

    if [[ -x "${RENOS_BIN}" ]]; then
        # Attempt enrollment; timeout after 30 seconds
        timeout 30 "${RENOS_BIN}" --enroll --config "${CONFIG_FILE}" 2>&1 \
            | while IFS= read -r line; do log "  [renos-agent] ${line}"; done

        local exit_code=${PIPESTATUS[0]}
        if [[ ${exit_code} -eq 0 ]]; then
            date +%s > "${ENROLLMENT_STAMP}"
            chmod 600 "${ENROLLMENT_STAMP}"
            log "Device enrolled successfully."
        elif [[ ${exit_code} -eq 124 ]]; then
            warn "Enrollment timed out after 30 seconds. Manual enrollment required."
        else
            warn "Enrollment exited with code ${exit_code}. Manual enrollment required."
        fi
    else
        warn "renos-agent binary not found at ${RENOS_BIN}; skipping enrollment."
    fi
}

# ---------------------------------------------------------------
# Main
# ---------------------------------------------------------------
main() {
    log "RenOS Agent post-install starting..."

    setup_directories
    install_launchagent
    setup_firewall
    setup_keychain
    setup_config
    fix_permissions
    start_agent
    run_enrollment

    log "RenOS Agent post-install complete."
}

main "$@"
