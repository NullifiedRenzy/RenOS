#!/usr/bin/env bash
set -euo pipefail

# RenOS Stress Test (M56)
# Simulates high load conditions by creating many concurrent connections, 
# workflows, and operations against a running Core instance.

BASE="${BASE_URL:-http://localhost:8080}"
AUTH_HEADER="Authorization: Bearer ${ADMIN_TOKEN:?Set ADMIN_TOKEN=...}"

log() { echo "[$(date +%H:%M:%S)] $*"; }

# ──────────────────────────────────────────
# 1. Sequential API stress: burst requests
# ──────────────────────────────────────────
stress_api() {
    local total="${1:-1000}"
    local concurrency="${2:-20}"
    local endpoint="${3:-/api/v1/healthz}"
    local successes=0
    local failures=0

    log "Stress API: $total requests, $concurrency concurrent, $endpoint"

    for ((i=0; i<total; i+=concurrency)); do
        local batch=$((concurrency))
        if ((i + batch > total)); then batch=$((total - i)); fi

        local pids=""
        for ((j=0; j<batch; j++)); do
            (
                if curl -sf "${BASE}${endpoint}" >/dev/null 2>&1; then
                    exit 0
                else
                    exit 1
                fi
            ) &
            pids+=" $!"
        done

        for pid in $pids; do
            wait "$pid" && ((successes++)) || ((failures++))
        done
    done

    log "API stress: $successes succeeded, $failures failed"
    [[ $failures -eq 0 ]] || log "WARNING: API stress had $failures failures"
}

# ──────────────────────────────────────────
# 2. Device simulation: register many devices
# ──────────────────────────────────────────
stress_devices() {
    local count="${1:-100}"

    if ! curl -sf "${BASE}/api/v1/healthz" >/dev/null 2>&1; then
        log "Core not reachable, skipping device stress"
        return
    fi

    log "Simulating $count device registrations..."
    for ((i=0; i<count; i++)); do
        local dev_id="stress-device-${i}-$(date +%s)"
        curl -sf -X POST "${BASE}/api/v1/devices" \
            -H "Content-Type: application/json" \
            -H "$AUTH_HEADER" \
            -d "{\"id\":\"${dev_id}\",\"hostname\":\"stress-${i}\",\"os\":\"Linux\"}" \
            >/dev/null 2>&1 || true

        if ((i % 10 == 0)); then
            log "  ... $i devices created"
        fi
    done
    log "Device stress complete: $count devices"
}

# ──────────────────────────────────────────
# 3. Workflow stress: create many workflows
# ──────────────────────────────────────────
stress_workflows() {
    local count="${1:-50}"

    if ! curl -sf "${BASE}/api/v1/healthz" >/dev/null 2>&1; then
        log "Core not reachable, skipping workflow stress"
        return
    fi

    log "Creating $count workflows..."
    for ((i=0; i<count; i++)); do
        curl -sf -X POST "${BASE}/api/v1/workflows" \
            -H "Content-Type: application/json" \
            -H "$AUTH_HEADER" \
            -d "{\"name\":\"stress-workflow-${i}\",\"steps\":[{\"type\":\"delay\",\"params\":{\"duration_ms\":100}}]}" \
            >/dev/null 2>&1 || true

        if ((i % 10 == 0)); then
            log "  ... $i workflows created"
        fi
    done
    log "Workflow stress complete: $count workflows"
}

# ──────────────────────────────────────────
# 4. Terminal stress: concurrent sessions
# ──────────────────────────────────────────
stress_terminals() {
    local count="${1:-10}"

    if ! command -v websocat &>/dev/null; then
        log "websocat not found, skipping terminal stress. Install with: cargo install websocat"
        return
    fi

    log "Opening $count concurrent terminal sessions (simulated)..."
    for ((i=0; i<count; i++)); do
        websocat -E "wss://localhost:8080/api/v1/ws?token=dummy" \
            >/dev/null 2>&1 &
    done
    log "Terminal stress: $count sessions opened"
}

# ──────────────────────────────────────────
# Main
# ──────────────────────────────────────────
log "=== RenOS Stress Test ==="
log "Target: $BASE"
log

stress_api 500 20 "/api/v1/healthz"

log
stress_devices 50

log
stress_workflows 20

log
stress_terminals 5

log
log "=== Stress Test Complete ==="
log "Check Core logs for any errors: docker logs renos-core"
