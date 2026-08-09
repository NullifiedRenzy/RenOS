#!/usr/bin/env bash
# phases/upgrade.sh — run the standalone upgrade verifier and, when core is
# reachable, exercise the update-API endpoints (read-only / safe mutations).

phase_upgrade() {
  step "upgrade"
  local suite="$REPO_ROOT/release/tests/verify-upgrade.sh"

  if [[ -x "$suite" ]]; then
    if [[ "$QUICK" == "true" ]]; then
      _note "(--quick: skipping verify-upgrade.sh)"
    else
      run_cmd bash "$suite"
      assert_ok "release/tests/verify-upgrade.sh exited 0" $RC
      if [[ $RC -ne 0 ]]; then
        _note "verify-upgrade.sh failed — see release/reports/upgrade.md"
      fi
    fi
  else
    _note "verify-upgrade.sh not present — relying on update API checks"
  fi

  if [[ "$CORE_UP" == "true" ]] && [[ -n "$AUTH_TOKEN" ]]; then
    api_req GET /updates
    assert_eq "GET /api/v1/updates -> 200" "$API_STATUS" 200
    api_req GET /updates/history
    assert_eq "GET /api/v1/updates/history -> 200" "$API_STATUS" 200
    api_req PUT /updates/channel '{"channel":"stable"}'
    assert_eq "PUT /api/v1/updates/channel -> 200" "$API_STATUS" 200
  else
    _note "core/auth unavailable — skipping update API checks"
  fi

  step_summary "upgrade verifier + update API checks passed" "upgrade verifier or update API checks failed"
  return 0
}
