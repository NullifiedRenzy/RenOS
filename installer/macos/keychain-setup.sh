#!/bin/bash
#
# RenOS Agent — Keychain Integration
# ====================================
# Stores and retrieves enrollment credentials using the macOS Keychain.
# Uses the security(1) command-line tool.
#
# Usage:
#   source ./keychain-setup.sh
#   store_credential "com.renos.agent" "<token>" "<server-url>" "<device-id>"
#   get_credential "com.renos.agent"
#   delete_credential "com.renos.agent"
#
set -euo pipefail

RENOS_KEYCHAIN="${RENOS_KEYCHAIN:-/Library/Keychains/System.keychain}"
RENOS_KEYCHAIN_SERVICE="${RENOS_KEYCHAIN_SERVICE:-com.renos.agent}"
RENOS_ACCESS_GROUP="${RENOS_ACCESS_GROUP:-com.renos.agent}"

log()   { printf "[keychain] %s\n" "$*"; }
warn()  { printf "[keychain] WARNING: %s\n" "$*" >&2; }
error() { printf "[keychain] ERROR: %s\n" "$*" >&2; }

# ---------------------------------------------------------------
# Store credential: upserts a generic password item in the keychain.
# Arguments:
#   $1 — service name (default: com.renos.agent)
#   $2 — enrollment token (the password)
#   $3 — server URL (stored in account attribute)
#   $4 — device ID (stored in label / comment)
# Returns:
#   0 on success, non-zero on failure.
# ---------------------------------------------------------------
store_credential() {
    local service="${1:-${RENOS_KEYCHAIN_SERVICE}}"
    local token="${2:-}"
    local server_url="${3:-}"
    local device_id="${4:-}"

    if [[ -z "${token}" ]]; then
        error "token is required"
        return 1
    fi

    # Remove existing item first to avoid duplicates
    delete_credential "${service}" 2>/dev/null || true

    # Store generic password item
    security add-generic-password \
        -a "${server_url}" \
        -s "${service}" \
        -w "${token}" \
        -l "RenOS Agent: ${device_id}" \
        -D "RenOS Enrollment Token" \
        -j "${device_id}" \
        -A \
        -T "" \
        -U \
        "${RENOS_KEYCHAIN}" 2>/dev/null || {

        # Fallback: try without specifying keychain
        security add-generic-password \
            -a "${server_url}" \
            -s "${service}" \
            -w "${token}" \
            -l "RenOS Agent: ${device_id}" \
            -D "RenOS Enrollment Token" \
            -j "${device_id}" \
            -A \
            -T "" \
            -U 2>/dev/null || {
            error "Failed to store credential in keychain"
            return 1
        }
    }

    log "Credential stored for service '${service}' (device: ${device_id})"

    # Verify
    local verify_token
    verify_token=$(get_credential "${service}" 2>/dev/null || true)
    if [[ -n "${verify_token}" ]]; then
        echo "${device_id}"
        return 0
    else
        warn "Credential stored but verification read returned empty"
        return 1
    fi
}

# ---------------------------------------------------------------
# Get credential: retrieves the enrollment token from keychain.
# Arguments:
#   $1 — service name (default: com.renos.agent)
# Returns:
#   The token (password) on stdout, or non-zero on failure.
# ---------------------------------------------------------------
get_credential() {
    local service="${1:-${RENOS_KEYCHAIN_SERVICE}}"

    local token
    token=$(security find-generic-password \
        -s "${service}" \
        -w \
        "${RENOS_KEYCHAIN}" 2>/dev/null) || {

        # Fallback: try without specifying keychain
        token=$(security find-generic-password \
            -s "${service}" \
            -w 2>/dev/null) || {
            error "Credential not found for service '${service}'"
            return 1
        }
    }

    if [[ -n "${token}" ]]; then
        echo "${token}"
        return 0
    else
        error "Credential for service '${service}' is empty"
        return 1
    fi
}

# ---------------------------------------------------------------
# Get credential attributes (JSON)
# Arguments:
#   $1 — service name (default: com.renos.agent)
# Returns:
#   JSON with server_url and device_id, or non-zero on failure.
# ---------------------------------------------------------------
get_credential_attrs() {
    local service="${1:-${RENOS_KEYCHAIN_SERVICE}}"

    local attrs
    attrs=$(security find-generic-password \
        -s "${service}" \
        -g \
        "${RENOS_KEYCHAIN}" 2>&1) || {

        attrs=$(security find-generic-password \
            -s "${service}" \
            -g 2>&1) || {
            error "Credential attributes not found for service '${service}'"
            return 1
        }
    }

    local server_url device_id
    server_url=$(echo "${attrs}" | grep -o '"acct"<blob>="[^"]*"' | sed 's/"acct"<blob>="//;s/"$//' || echo "")
    device_id=$(echo "${attrs}" | grep -o '"icmt"<blob>="[^"]*"' | sed 's/"icmt"<blob>="//;s/"$//' || echo "")

    printf '{"server_url":"%s","device_id":"%s"}\n' "${server_url}" "${device_id}"
}

# ---------------------------------------------------------------
# Delete credential: removes the keychain item.
# Arguments:
#   $1 — service name (default: com.renos.agent)
# Returns:
#   0 on success, non-zero on failure or not found.
# ---------------------------------------------------------------
delete_credential() {
    local service="${1:-${RENOS_KEYCHAIN_SERVICE}}"

    security delete-generic-password \
        -s "${service}" \
        "${RENOS_KEYCHAIN}" 2>/dev/null || {

        # Fallback: try without specifying keychain
        security delete-generic-password \
            -s "${service}" \
            2>/dev/null || {
            log "No existing credential to delete for service '${service}'"
            return 0
        }
    }

    log "Credential deleted for service '${service}'"
    return 0
}

# ---------------------------------------------------------------
# Check if credential exists
# Arguments:
#   $1 — service name (default: com.renos.agent)
# Returns:
#   0 if exists, 1 if not.
# ---------------------------------------------------------------
has_credential() {
    local service="${1:-${RENOS_KEYCHAIN_SERVICE}}"

    if security find-generic-password \
        -s "${service}" \
        -w \
        "${RENOS_KEYCHAIN}" &>/dev/null 2>&1; then
        return 0
    fi

    if security find-generic-password \
        -s "${service}" \
        -w &>/dev/null 2>&1; then
        return 0
    fi

    return 1
}

# ---------------------------------------------------------------
# Rotate token: updates the password on an existing item.
# Arguments:
#   $1 — service name (default: com.renos.agent)
#   $2 — new token
# Returns:
#   0 on success.
# ---------------------------------------------------------------
rotate_credential() {
    local service="${1:-${RENOS_KEYCHAIN_SERVICE}}"
    local new_token="${2:-}"

    if [[ -z "${new_token}" ]]; then
        error "new token is required for rotation"
        return 1
    fi

    if ! has_credential "${service}"; then
        error "Cannot rotate: no existing credential for service '${service}'"
        return 1
    fi

    # Retrieve current attributes
    local server_url device_id
    server_url=$(get_credential_attrs "${service}" 2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin).get('server_url',''))" 2>/dev/null || echo "")
    device_id=$(get_credential_attrs "${service}" 2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin).get('device_id',''))" 2>/dev/null || echo "")

    store_credential "${service}" "${new_token}" "${server_url}" "${device_id}"
}

# ---------------------------------------------------------------
# List all RenOS credentials
# ---------------------------------------------------------------
list_credentials() {
    log "RenOS Keychain credentials:"

    security dump-keychain \
        "${RENOS_KEYCHAIN}" 2>/dev/null \
        | grep -A 2 "RenOS Agent" || {
        log "No RenOS credentials found."
        return 0
    }
}

# ---------------------------------------------------------------
# Main (when executed directly)
# ---------------------------------------------------------------
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-help}" in
        store)
            shift
            store_credential "$@"
            ;;
        get)
            shift
            get_credential "$@"
            ;;
        delete)
            shift
            delete_credential "$@"
            ;;
        attrs)
            shift
            get_credential_attrs "$@"
            ;;
        has)
            shift
            has_credential "$@" && echo "present" || echo "absent"
            ;;
        rotate)
            shift
            rotate_credential "$@"
            ;;
        list)
            list_credentials
            ;;
        help|--help|-h)
            cat <<-HELP
Usage: ${0} <command> [args]

Commands:
  store [service] <token> <server_url> <device_id>   Store credential
  get    [service]                                     Retrieve token
  delete [service]                                     Delete credential
  attrs  [service]                                     Get credential attributes (JSON)
  has    [service]                                     Check if credential exists
  rotate [service] <new_token>                         Rotate token
  list                                                 List all RenOS credentials

Default service: com.renos.agent
HELP
            ;;
        *)
            error "Unknown command: ${1}"
            exit 1
            ;;
    esac
fi
