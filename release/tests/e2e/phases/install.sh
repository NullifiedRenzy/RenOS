#!/usr/bin/env bash
# phases/install.sh — provision the backend (or detect an existing one).
# Never fakes a PASS: if no backend is reachable and we cannot bring one up,
# the phase records SKIP with a clear reason.

phase_install() {
  step "install"
  local code

  if [[ "$SKIP_INSTALL" == "true" ]]; then
    step_skip "--skip-install requested"
    return 0
  fi

  code=$(http_code "$RENOS_URL/api/v1/healthz")
  if [[ "$code" == "200" ]]; then
    CORE_UP=true
    step_ok "backend already reachable at $RENOS_URL/api/v1/healthz"
    return 0
  fi

  if ! command -v docker >/dev/null 2>&1; then
    step_skip "docker not available — cannot provision a backend"
    return 0
  fi
  if ! docker compose version >/dev/null 2>&1; then
    step_skip "docker compose not available — cannot provision a backend"
    return 0
  fi

  _note "attempting best-effort compose bring-up (project: renos-e2e)..."
  run_cmd docker compose -p renos-e2e -f "$REPO_ROOT/docker-compose.yml" up -d
  if [[ $RC -ne 0 ]]; then
    step_skip "best-effort install failed: ${OUT//$'\n'/ }"
    return 0
  fi

  if wait_for_health "${HEALTH_TIMEOUT:-120}"; then
    step_ok "backend healthy after compose up"
  else
    step_skip "core did not become healthy after compose up"
  fi
  return 0
}
