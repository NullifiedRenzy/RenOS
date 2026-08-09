#!/usr/bin/env bash
# RenOS API curl examples
# Usage: source curl.sh  (exports TOKEN)
set -euo pipefail

BASE="${RENOS_URL:-http://localhost:8443}"
EMAIL="${RENOS_EMAIL:-admin@renos.local}"
PASS="${RENOS_PASSWORD:-admin}"

echo "=== RenOS API Examples ==="
echo "Server: $BASE"
echo ""

# ── Authentication ──────────────────────────────────────────────────────

login() {
  echo "── Login ──"
  TOKEN=$(curl -s -X POST "$BASE/api/v1/auth/login" \
    -H 'Content-Type: application/json' \
    -d "{\"email\":\"$EMAIL\",\"password\":\"$PASS\"}" | jq -r '.token')
  echo "Token: ${TOKEN:0:20}..."
  export TOKEN
}

auth_hdr() { echo "-H 'Authorization: Bearer $TOKEN'"; }

# ── Devices ─────────────────────────────────────────────────────────────

list_devices() {
  echo "── List Devices ──"
  curl -s -H "Authorization: Bearer $TOKEN" "$BASE/api/v1/devices?page=1&per_page=10" | jq .
}

get_device() {
  echo "── Get Device ──"
  curl -s -H "Authorization: Bearer $TOKEN" "$BASE/api/v1/devices/${1}" | jq .
}

run_command() {
  echo "── Run Command ──"
  curl -s -X POST -H "Authorization: Bearer $TOKEN" \
    -H 'Content-Type: application/json' \
    -d "{\"command\":\"$2\",\"timeout\":30}" \
    "$BASE/api/v1/devices/${1}/commands" | jq .
}

list_commands() {
  echo "── List Command History ──"
  curl -s -H "Authorization: Bearer $TOKEN" \
    "$BASE/api/v1/devices/${1}/commands?page=1&per_page=5" | jq .
}

# ── Workflows ───────────────────────────────────────────────────────────

list_workflows() {
  echo "── List Workflows ──"
  curl -s -H "Authorization: Bearer $TOKEN" "$BASE/api/v1/workflows?page=1&per_page=20" | jq .
}

get_workflow() {
  echo "── Get Workflow ──"
  curl -s -H "Authorization: Bearer $TOKEN" "$BASE/api/v1/workflows/${1}" | jq .
}

execute_workflow() {
  echo "── Execute Workflow ──"
  curl -s -X POST -H "Authorization: Bearer $TOKEN" \
    "$BASE/api/v1/workflows/${1}/execute" | jq .
}

# ── Jobs ────────────────────────────────────────────────────────────────

list_jobs() {
  echo "── List Jobs ──"
  curl -s -H "Authorization: Bearer $TOKEN" "$BASE/api/v1/jobs?page=1&per_page=10" | jq .
}

get_job() {
  echo "── Get Job ──"
  curl -s -H "Authorization: Bearer $TOKEN" "$BASE/api/v1/jobs/${1}" | jq .
}

cancel_job() {
  echo "── Cancel Job ──"
  curl -s -X POST -H "Authorization: Bearer $TOKEN" \
    "$BASE/api/v1/jobs/${1}/cancel" | jq .
}

# ── Groups ──────────────────────────────────────────────────────────────

list_groups() {
  echo "── List Groups ──"
  curl -s -H "Authorization: Bearer $TOKEN" "$BASE/api/v1/groups" | jq .
}

create_group() {
  echo "── Create Group ──"
  curl -s -X POST -H "Authorization: Bearer $TOKEN" \
    -H 'Content-Type: application/json' \
    -d "{\"name\":\"$1\",\"description\":\"$2\"}" \
    "$BASE/api/v1/groups" | jq .
}

delete_group() {
  echo "── Delete Group ──"
  curl -s -X DELETE -H "Authorization: Bearer $TOKEN" \
    "$BASE/api/v1/groups/${1}" | jq .
}

# ── Users ───────────────────────────────────────────────────────────────

list_users() {
  echo "── List Users ──"
  curl -s -H "Authorization: Bearer $TOKEN" "$BASE/api/v1/users?page=1&per_page=20" | jq .
}

create_user() {
  echo "── Create User ──"
  curl -s -X POST -H "Authorization: Bearer $TOKEN" \
    -H 'Content-Type: application/json' \
    -d "{\"email\":\"$1\",\"password\":\"$2\",\"role\":\"operator\"}" \
    "$BASE/api/v1/users" | jq .
}

delete_user() {
  echo "── Delete User ──"
  curl -s -X DELETE -H "Authorization: Bearer $TOKEN" \
    "$BASE/api/v1/users/${1}" | jq .
}

# ── Schedules ───────────────────────────────────────────────────────────

list_schedules() {
  echo "── List Schedules ──"
  curl -s -H "Authorization: Bearer $TOKEN" "$BASE/api/v1/schedules" | jq .
}

create_schedule() {
  echo "── Create Schedule ──"
  curl -s -X POST -H "Authorization: Bearer $TOKEN" \
    -H 'Content-Type: application/json' \
    -d "{\"name\":\"$1\",\"type\":\"cron\",\"cron_expression\":\"$2\",\"actions\":[{\"type\":\"workflow\",\"workflow_id\":\"$3\"}]}" \
    "$BASE/api/v1/schedules" | jq .
}

# ── Alerts ──────────────────────────────────────────────────────────────

list_active_alerts() {
  echo "── List Active Alerts ──"
  curl -s -H "Authorization: Bearer $TOKEN" "$BASE/api/v1/alerts/active" | jq .
}

acknowledge_alert() {
  echo "── Acknowledge Alert ──"
  curl -s -X POST -H "Authorization: Bearer $TOKEN" \
    "$BASE/api/v1/alerts/${1}/acknowledge" | jq .
}

# ── Vault ───────────────────────────────────────────────────────────────

list_vault_secrets() {
  echo "── List Vault Secrets ──"
  curl -s -H "Authorization: Bearer $TOKEN" "$BASE/api/v1/vault/secrets" | jq .
}

create_vault_secret() {
  echo "── Create Vault Secret ──"
  curl -s -X POST -H "Authorization: Bearer $TOKEN" \
    -H 'Content-Type: application/json' \
    -d "{\"name\":\"$1\",\"value\":\"$2\",\"path\":\"/$1\"}" \
    "$BASE/api/v1/vault/secrets" | jq .
}

# ── Terminal ────────────────────────────────────────────────────────────

list_terminal_sessions() {
  echo "── List Terminal Sessions ──"
  curl -s -H "Authorization: Bearer $TOKEN" \
    "$BASE/api/v1/devices/${1}/terminal/sessions" | jq .
}

# ── Executions (Process Management) ─────────────────────────────────────

create_execution() {
  echo "── Create Execution ──"
  curl -s -X POST -H "Authorization: Bearer $TOKEN" \
    -H 'Content-Type: application/json' \
    -d "{\"command\":\"$2\",\"args\":$3,\"workdir\":\"$4\"}" \
    "$BASE/api/v1/devices/${1}/executions" | jq .
}

get_execution() {
  echo "── Get Execution ──"
  curl -s -H "Authorization: Bearer $TOKEN" \
    "$BASE/api/v1/devices/${1}/executions/${2}" | jq .
}

cancel_execution() {
  echo "── Cancel Execution ──"
  curl -s -X POST -H "Authorization: Bearer $TOKEN" \
    "$BASE/api/v1/devices/${1}/executions/${2}/cancel" | jq .
}

# ── Policies ────────────────────────────────────────────────────────────

list_policies() {
  echo "── List Policies ──"
  curl -s -H "Authorization: Bearer $TOKEN" "$BASE/api/v1/policies" | jq .
}

# ── Cluster ─────────────────────────────────────────────────────────────

list_cluster_nodes() {
  echo "── List Cluster Nodes ──"
  curl -s -H "Authorization: Bearer $TOKEN" "$BASE/api/v1/cluster/nodes" | jq .
}

# ── Webhooks ────────────────────────────────────────────────────────────

list_webhooks() {
  echo "── List Webhooks ──"
  curl -s -H "Authorization: Bearer $TOKEN" "$BASE/api/v1/webhooks" | jq .
}

create_webhook() {
  echo "── Create Webhook ──"
  curl -s -X POST -H "Authorization: Bearer $TOKEN" \
    -H 'Content-Type: application/json' \
    -d "{\"name\":\"$1\",\"url\":\"$2\",\"events\":[\"device.online\",\"device.offline\"]}" \
    "$BASE/api/v1/webhooks" | jq .
}

# ── Plugins ─────────────────────────────────────────────────────────────

list_plugins() {
  echo "── List Plugins ──"
  curl -s -H "Authorization: Bearer $TOKEN" "$BASE/api/v1/plugins" | jq .
}

install_plugin() {
  echo "── Install Plugin ──"
  curl -s -X POST -H "Authorization: Bearer $TOKEN" \
    -H 'Content-Type: application/json' \
    -d "{\"source\":\"$1\"}" \
    "$BASE/api/v1/plugins/install" | jq .
}

# ── Health ──────────────────────────────────────────────────────────────

health_check() {
  echo "── Health Check ──"
  curl -s "$BASE/api/v1/healthz" | jq .
}

readiness_check() {
  echo "── Readiness Check ──"
  curl -s "$BASE/api/v1/readyz" | jq .
}

version_info() {
  echo "── Version ──"
  curl -s "$BASE/api/v1/version" | jq .
}

# ── WebSocket (streaming) ───────────────────────────────────────────────

ws_example() {
  echo "── WebSocket (requires websocat/wscat) ──"
  echo "wscat -c \"$BASE/api/v1/ws\" -H \"Authorization: Bearer \$TOKEN\""
}

# ── SSE Stream ──────────────────────────────────────────────────────────

sse_example() {
  echo "── SSE Stream ──"
  echo "curl -s -N -H \"Authorization: Bearer \$TOKEN\" -H \"Accept: text/event-stream\" \"$BASE/api/v1/deploy/stream\""
}

# ── Main ────────────────────────────────────────────────────────────────

main() {
  login

  echo ""
  echo "=== Available functions ==="
  echo "Run any function directly:"
  echo "  list_devices"
  echo "  get_device <id>"
  echo "  run_command <device_id> '<command>'"
  echo "  list_commands <device_id>"
  echo "  list_workflows"
  echo "  execute_workflow <workflow_id>"
  echo "  list_jobs"
  echo "  create_group <name> <description>"
  echo "  list_users"
  echo "  create_user <email> <password>"
  echo "  list_schedules"
  echo "  list_active_alerts"
  echo "  health_check"
  echo "  version_info"
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  main
fi
