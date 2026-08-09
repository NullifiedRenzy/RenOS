#!/usr/bin/env bash
# phases/enroll.sh — full agent enrollment: enroll token, RSA keypair,
# enroll, connect, and confirm the device appears in the device list.

phase_enroll() {
  step "enroll"
  require_api || return 0
  require_auth || return 0

  local code body host pubder etoken
  E2E_DEVICE_ID=""
  E2E_DEVICE_SECRET=""

  api_req POST /devices/enroll-token
  assert_eq "POST /api/v1/devices/enroll-token -> 200" "$API_STATUS" 200
  etoken=$(json_get "$API_BODY" token)
  assert_ok "enroll token generated" $([[ -n "$etoken" ]] && echo 0 || echo 1)
  [[ -n "$etoken" ]] || { step_fail "no enroll token returned"; return 0; }

  openssl genpkey -algorithm RSA -out "$E2E_TMP/device-key.pem" -pkeyopt rsa_keygen_bits:2048 >/dev/null 2>&1
  pubder=$(openssl rsa -in "$E2E_TMP/device-key.pem" -pubout -outform DER 2>/dev/null | base64 -w0)
  assert_ok "device public key (PKIX/DER) generated" $([[ -n "$pubder" ]] && echo 0 || echo 1)
  [[ -n "$pubder" ]] || { step_fail "failed to generate device keypair"; return 0; }

  host=$(random_name "e2e-node")
  body=$(printf '{"token":%s,"public_key_base64":%s,"hostname":%s,"os":"linux","arch":"amd64","agent_version":"0.1.0-e2e"}' \
    "$(json_str "$etoken")" "$(json_str "$pubder")" "$(json_str "$host")")
  api_req POST /devices/enroll "$body"
  assert_eq "POST /api/v1/devices/enroll -> 201" "$API_STATUS" 201

  E2E_DEVICE_ID=$(json_get "$API_BODY" device_id)
  E2E_DEVICE_SECRET=$(json_get "$API_BODY" device_secret)
  assert_ok "device_id returned" $([[ -n "$E2E_DEVICE_ID" ]] && echo 0 || echo 1)
  assert_ok "device_secret returned" $([[ -n "$E2E_DEVICE_SECRET" ]] && echo 0 || echo 1)

  if [[ -n "$E2E_DEVICE_ID" && -n "$E2E_DEVICE_SECRET" ]]; then
    body=$(printf '{"device_id":%s,"device_secret":%s}' "$(json_str "$E2E_DEVICE_ID")" "$(json_str "$E2E_DEVICE_SECRET")")
    api_req POST /devices/connect "$body"
    assert_eq "POST /api/v1/devices/connect -> 200" "$API_STATUS" 200
  fi

  api_req GET /devices
  assert_eq "GET /api/v1/devices -> 200" "$API_STATUS" 200
  if [[ -n "$E2E_DEVICE_ID" ]]; then
    assert_contains "device listed in /api/v1/devices" "$API_BODY" "$E2E_DEVICE_ID"
  fi

  step_summary "device enrolled, connected, and listed ($E2E_DEVICE_ID)" "enrollment flow assertions failed"
  return 0
}
