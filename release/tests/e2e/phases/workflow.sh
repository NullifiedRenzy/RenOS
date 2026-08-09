#!/usr/bin/env bash
# phases/workflow.sh — create a workflow, add a step, execute it, and confirm
# the workflow shows up in the list.

phase_workflow() {
  step "workflow"
  require_api || return 0
  require_auth || return 0

  local name body code
  E2E_WORKFLOW_ID=""
  name=$(random_name "wf")

  body=$(printf '{"name":%s,"description":"e2e workflow","trigger_type":"manual"}' "$(json_str "$name")")
  api_req POST /workflows "$body"
  assert_eq "POST /api/v1/workflows -> 201" "$API_STATUS" 201
  E2E_WORKFLOW_ID=$(json_get "$API_BODY" data.id)
  assert_ok "workflow id returned" $([[ -n "$E2E_WORKFLOW_ID" ]] && echo 0 || echo 1)

  if [[ -n "$E2E_WORKFLOW_ID" ]]; then
    body='{"name":"step-1","step_type":"action","config":{}}'
    api_req POST "/workflows/$E2E_WORKFLOW_ID/steps" "$body"
    assert_eq "POST /api/v1/workflows/{id}/steps -> 201" "$API_STATUS" 201

    api_req POST "/workflows/$E2E_WORKFLOW_ID/execute" '{"trigger_data":{}}'
    assert_eq "POST /api/v1/workflows/{id}/execute -> 201" "$API_STATUS" 201

    api_req GET "/workflows/$E2E_WORKFLOW_ID"
    assert_eq "GET /api/v1/workflows/{id} -> 200" "$API_STATUS" 200
  fi

  api_req GET /workflows
  assert_eq "GET /api/v1/workflows -> 200" "$API_STATUS" 200
  if [[ -n "$E2E_WORKFLOW_ID" ]]; then
    assert_contains "workflow listed" "$API_BODY" "$E2E_WORKFLOW_ID"
  fi

  step_summary "workflow created, executed, and listed ($E2E_WORKFLOW_ID)" "workflow flow assertions failed"
  return 0
}
