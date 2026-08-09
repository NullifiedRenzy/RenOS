#!/usr/bin/env bash
set -euo pipefail

# RenOS Chaos Engineering Test Suite (M56)
# Kills core services and validates recovery via /readyz endpoint.

BASE_URL="${BASE_URL:-http://localhost:8080}"
READYZ="${BASE_URL}/api/v1/readyz"
HEALTHZ="${BASE_URL}/api/v1/healthz"
TIMEOUT_SEC="${TIMEOUT_SEC:-60}"
PASS=0
FAIL=0

log()   { echo "[$(date +%H:%M:%S)] $*"; }
pass()  { log "PASS: $*"; ((PASS++)); }
fail()  { log "FAIL: $*"; ((FAIL++)); }

# Wait until /readyz returns 200
wait_ready() {
    local deadline=$((SECONDS + TIMEOUT_SEC))
    while [[ $SECONDS -lt $deadline ]]; do
        if curl -sf "$READYZ" >/dev/null 2>&1; then
            return 0
        fi
        sleep 1
    done
    return 1
}

# Assert that /readyz eventually returns 200
assert_recovers() {
    local label="$1"
    if wait_ready; then
        pass "$label recovered within ${TIMEOUT_SEC}s"
    else
        fail "$label did NOT recover within ${TIMEOUT_SEC}s"
    fi
}

# Assert that a service kill command eventually causes readyz to return 503
assert_impact() {
    local label="$1"
    local deadline=$((SECONDS + 30))
    while [[ $SECONDS -lt $deadline ]]; do
        if ! curl -sf "$READYZ" >/dev/null 2>&1; then
            pass "$label caused readyz failure (expected)"
            return 0
        fi
        sleep 1
    done
    fail "$label did NOT cause readyz failure (unexpected resilience?)"
}

kill_pid_on_port() {
    local port="$1"
    local pid
    pid=$(lsof -t -i ":$port" 2>/dev/null || true)
    if [[ -n "$pid" ]]; then
        kill -9 "$pid" 2>/dev/null || true
        log "killed PID $pid on port $port"
    fi
}

# --- Prerequisites ---
log "=== RenOS Chaos Engineering Suite ==="
log "Target: $BASE_URL"
log "Readyz: $READYZ"
log

if ! curl -sf "$HEALTHZ" >/dev/null 2>&1; then
    fail "Core is not running at $BASE_URL"
    log "Start Core first with: docker compose up -d"
    exit 1
fi
pass "Core is reachable"

# ────────────────────────────────────────────
# 1. Kill PostgreSQL
# ────────────────────────────────────────────
log; log "=== 1. PostgreSQL Failure ==="
kill_pid_on_port 5432
assert_impact "PostgreSQL kill"
assert_recovers "PostgreSQL"

# ────────────────────────────────────────────
# 2. Kill Redis
# ────────────────────────────────────────────
log; log "=== 2. Redis Failure ==="
kill_pid_on_port 6379
assert_impact "Redis kill"
assert_recovers "Redis"

# ────────────────────────────────────────────
# 3. Kill NATS
# ────────────────────────────────────────────
log; log "=== 3. NATS Failure ==="
kill_pid_on_port 4222
assert_impact "NATS kill"
assert_recovers "NATS"

# ────────────────────────────────────────────
# 4. Kill Core process (SIGKILL, expect Docker restart)
# ────────────────────────────────────────────
log; log "=== 4. Core Process Failure ==="
local core_pid
core_pid=$(lsof -t -i ":8080" 2>/dev/null || true)
if [[ -n "$core_pid" ]]; then
    kill -9 "$core_pid"
    log "killed Core PID $core_pid"
    assert_impact "Core kill"
    assert_recovers "Core restart"
else
    fail "Core PID not found on port 8080"
fi

# ────────────────────────────────────────────
# 5. Kill Hub (Next.js) — if running
# ────────────────────────────────────────────
log; log "=== 5. Hub Failure ==="
local hub_pid
hub_pid=$(lsof -t -i ":3000" 2>/dev/null || true)
if [[ -n "$hub_pid" ]]; then
    kill -9 "$hub_pid"
    log "killed Hub PID $hub_pid"
    # Hub is separate; just verify Core still healthy
    assert_recovers "Hub kill (Core unaffected)"
else
    log "Hub not running on 3000, skipping"
fi

# ────────────────────────────────────────────
# 6. Network simulation
# ────────────────────────────────────────────
log; log "=== 6. Network Partition Simulation ==="

# Block NATS port to simulate partition
if command -v iptables &>/dev/null; then
    iptables -A INPUT -p tcp --dport 4222 -j DROP 2>/dev/null || true
    iptables -A OUTPUT -p tcp --dport 4222 -j DROP 2>/dev/null || true
    log "iptables: DROP on port 4222 (NATS partition)"
    assert_impact "NATS network partition"
    iptables -D INPUT -p tcp --dport 4222 -j DROP 2>/dev/null || true
    iptables -D OUTPUT -p tcp --dport 4222 -j DROP 2>/dev/null || true
    log "iptables: restored NATS traffic"
    assert_recovers "NATS partition recovery"
else
    log "iptables not available, skipping network partition test"
fi

# ────────────────────────────────────────────
# Summary
# ────────────────────────────────────────────
log; log "=== RESULTS ==="
log "Passed: $PASS"
log "Failed: $FAIL"

if [[ $FAIL -gt 0 ]]; then
    log "SOME CHAOS TESTS FAILED"
    exit 1
else
    log "ALL CHAOS TESTS PASSED"
    exit 0
fi
