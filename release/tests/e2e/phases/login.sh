#!/usr/bin/env bash
# phases/login.sh — authenticate as the admin user.
# Sets global AUTH_TOKEN used by every later phase.

phase_login() {
  step "login"
  require_api || return 0

  local body
  body=$(printf '{"email":%s,"password":%s}' "$(json_str "$ADMIN_EMAIL")" "$(json_str "$ADMIN_PASSWORD")")
  api_req POST /auth/login "$body"

  assert_eq "POST /api/v1/auth/login -> 200" "$API_STATUS" 200
  AUTH_TOKEN=$(json_get "$API_BODY" token)
  assert_ok "auth token extracted" $([[ -n "${AUTH_TOKEN:-}" ]] && echo 0 || echo 1)

  if [[ "$API_STATUS" == "200" && -n "$AUTH_TOKEN" ]]; then
    step_ok "logged in as $ADMIN_EMAIL"
  else
    step_fail "login failed (HTTP $API_STATUS) — check ADMIN_EMAIL/ADMIN_PASSWORD and backend"
  fi
  return 0
}
