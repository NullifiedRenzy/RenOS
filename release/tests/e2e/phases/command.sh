#!/usr/bin/env bash
# phases/command.sh — submit a command to the enrolled device, have the
# (simulated) agent report a result, and confirm the command is listed.

phase_command() {
  step "command"
  require_api || return 0
  require_auth || return 0
  if [[ -z "${E2E_DEVICE_ID:-}" ]]; then
    step_skip "no enrolled device (enroll phase skipped/failed)"
    return 0
  fi

  local code body
  E2E_COMMAND_ID=""

  body='{"command":"echo e2e-denied"}'
  api_req POST "/devices/$E2E_DEVICE_ID/commands" "$body"
  assert_eq "non-allowlisted command blocked by policy -> 403" "$API_STATUS" 403

  body="{\"command\":$(json_str "$E2E_COMMAND")}"
  api_req POST "/devices/$E2E_DEVICE_ID/commands" "$body"
  assert_eq "POST /api/v1/devices/{id}/commands -> 201" "$API_STATUS" 201
  E2E_COMMAND_ID=$(json_get "$API_BODY" command.id)
  assert_ok "command id returned" $([[ -n "$E2E_COMMAND_ID" ]] && echo 0 || echo 1)

  if [[ -n "$E2E_COMMAND_ID" ]]; then
    body=$(printf '{"device_id":%s,"device_secret":%s,"exit_code":0,"output":%s,"error":""}' \
      "$(json_str "$E2E_DEVICE_ID")" "$(json_str "$E2E_DEVICE_SECRET")" "$(json_str "${E2E_COMMAND} output")")
    api_req POST "/commands/$E2E_COMMAND_ID/result" "$body"
    assert_eq "POST /api/v1/commands/{id}/result -> 200" "$API_STATUS" 200
    assert_contains "command marked completed" "$API_BODY" completed

    api_req GET "/devices/$E2E_DEVICE_ID/commands"
    assert_eq "GET /api/v1/devices/{id}/commands -> 200" "$API_STATUS" 200
    assert_contains "command listed for device" "$API_BODY" "$E2E_COMMAND_ID"
  fi

  api_req GET /jobs
  assert_eq "GET /api/v1/jobs -> 200" "$API_STATUS" 200
  api_req GET /jobs/stats
  assert_eq "GET /api/v1/jobs/stats -> 200" "$API_STATUS" 200

  step_summary "command submitted, executed (simulated), and listed ($E2E_COMMAND_ID)" "command flow assertions failed"
  return 0
}
