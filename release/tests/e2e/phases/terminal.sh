#!/usr/bin/env bash
# phases/terminal.sh — terminal session listing endpoints are reachable.

phase_terminal() {
  step "terminal"
  require_api || return 0
  require_auth || return 0

  api_req GET /devices/terminal/sessions
  assert_eq "GET /api/v1/devices/terminal/sessions -> 200" "$API_STATUS" 200

  if [[ -n "${E2E_DEVICE_ID:-}" ]]; then
    api_req GET "/devices/$E2E_DEVICE_ID/terminal/sessions"
    assert_eq "GET /api/v1/devices/{id}/terminal/sessions -> 200" "$API_STATUS" 200
  fi

  step_summary "terminal session endpoints responding" "terminal session endpoints not responding"
  return 0
}
