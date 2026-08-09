#!/usr/bin/env bash
# phases/rollback.sh — run the standalone rollback verifier and, when core is
# reachable, exercise rollback-related API endpoints safely (never performs an
# actual rollback: the rollback POST is issued with an empty backup_id, which
# the backend rejects with 400 — proving the route is wired).

phase_rollback() {
  step "rollback"
  local suite="$REPO_ROOT/release/tests/verify-rollback.sh"

  if [[ -x "$suite" ]]; then
    if [[ "$QUICK" == "true" ]]; then
      _note "(--quick: skipping verify-rollback.sh)"
    else
      run_cmd bash "$suite"
      assert_ok "release/tests/verify-rollback.sh exited 0" $RC
      if [[ $RC -ne 0 ]]; then
        _note "verify-rollback.sh failed — see release/reports/rollback.md"
      fi
    fi
  else
    _note "verify-rollback.sh not present — relying on rollback API checks"
  fi

  if [[ "$CORE_UP" == "true" ]] && [[ -n "$AUTH_TOKEN" ]]; then
    api_req GET /updates/backups
    assert_eq "GET /api/v1/updates/backups -> 200" "$API_STATUS" 200
    api_req POST /updates/rollback '{"backup_id":""}'
    assert_eq "POST /api/v1/updates/rollback (empty backup_id) -> 400" "$API_STATUS" 400
  else
    _note "core/auth unavailable — skipping rollback API checks"
  fi

  step_summary "rollback verifier + rollback API checks passed" "rollback verifier or rollback API checks failed"
  return 0
}
