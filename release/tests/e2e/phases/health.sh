#!/usr/bin/env bash
# phases/health.sh — verify liveness/readiness endpoints are wired and responsive.

phase_health() {
  step "health"
  require_api || return 0

  local code body
  code=$(http_code "$RENOS_URL/api/v1/healthz")
  assert_eq "GET /api/v1/healthz -> 200" "$code" 200
  if [[ "$code" == "200" ]]; then
    body=$(curl -s -k -m 10 "$RENOS_URL/api/v1/healthz")
    assert_contains "healthz reports ok" "$body" ok
    CORE_UP=true
  else
    CORE_UP=false
  fi

  if [[ "$CORE_UP" == "true" ]]; then
    code=$(http_code "$RENOS_URL/api/v1/readyz")
    case "$code" in
      200) assert_ok "GET /api/v1/readyz -> 200 (deps healthy)" 0 ;;
      503) assert_ok "GET /api/v1/readyz -> 503 (degraded but responding)" 0 ;;
      *)   assert_eq "GET /api/v1/readyz responds" "$code" 200 ;;
    esac

    code=$(http_code "$RENOS_URL/livez")
    if [[ "$code" != "000" ]]; then
      assert_ok "GET /livez responded (HTTP $code)" 0
    else
      _note "GET /livez unreachable (older build) — not an assertion"
    fi
    code=$(http_code "$RENOS_URL/startupz")
    if [[ "$code" != "000" ]]; then
      assert_ok "GET /startupz responded (HTTP $code)" 0
    else
      _note "GET /startupz unreachable (older build) — not an assertion"
    fi

    if [[ -n "$AUTH_TOKEN" ]]; then
      api_req GET /fleet/dashboard
      assert_eq "GET /api/v1/fleet/dashboard -> 200" "$API_STATUS" 200
    else
      _note "no auth token — skipping fleet dashboard check"
    fi
    step_summary "core healthy" "core health assertions failed"
  else
    step_fail "core not healthy — healthz returned HTTP $code"
  fi
  return 0
}
