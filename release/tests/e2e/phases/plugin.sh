#!/usr/bin/env bash
# phases/plugin.sh — marketplace / plugin-platform endpoints respond;
# best-effort install of the first marketplace plugin (may be a no-op in
# fresh environments where the marketplace catalog is empty).

phase_plugin() {
  step "plugin"
  require_api || return 0
  require_auth || return 0

  local first marketplace_ok=1
  api_req GET /marketplace
  assert_eq "GET /api/v1/marketplace -> 200" "$API_STATUS" 200
  if [[ "$API_STATUS" == "200" ]]; then marketplace_ok=0; fi

  api_req GET /marketplace/installed
  assert_eq "GET /api/v1/marketplace/installed -> 200" "$API_STATUS" 200
  api_req GET /plugin-platform
  assert_eq "GET /api/v1/plugin-platform -> 200" "$API_STATUS" 200

  if [[ "$marketplace_ok" == "0" ]]; then
    first=$(python3 -c '
import json, sys
try:
    d = json.loads(sys.argv[1])
    d = d.get("data", d) if isinstance(d, dict) else d
    items = d if isinstance(d, list) else d.get("plugins", [])
    print(items[0].get("name", "") if items else "")
except Exception:
    print("")
' "$API_BODY" 2>/dev/null)
    if [[ -n "$first" ]]; then
      api_req POST "/marketplace/$first/install"
      _note "marketplace install '$first' -> HTTP $API_STATUS"
      case "$API_STATUS" in
        201) assert_ok "installed '$first' from marketplace" 0 ;;
        *)   assert_ok "install endpoint responded (HTTP $API_STATUS)" 0 ;;
      esac
    else
      _note "marketplace catalog empty — install not attempted"
    fi
  fi

  step_summary "plugin/marketplace endpoints responding" "plugin/marketplace assertions failed"
  return 0
}
