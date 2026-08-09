#!/usr/bin/env bash
#
# RenOS Example Library — one-click pack importer
#
# Imports the JSON files of a scenario pack from examples/library/<pack>
# into a running RenOS instance using the public v1 REST API.
#
# Usage:
#   ./import.sh <pack> [base-url] [email] [password]
#
# Examples:
#   ./import.sh home-lab https://renos.local admin@renos.local
#   ./import.sh docker-host http://localhost:8443 admin@renos.local
#   ./import.sh --all https://renos.local admin@renos.local
#
# Options:
#   --all        import every pack in the library
#   --dry-run    validate the files and print what would be created (no login)
#   -h, --help   show this help
#
# Environment:
#   RENOS_URL       base URL (default http://localhost:8443)
#   RENOS_EMAIL     login email (default admin@renos.local)
#   RENOS_PASSWORD  login password (prompted if not passed as an argument)
#   RENOS_INSECURE  set to 1 to use curl -k for self-signed TLS
#   RENOS_TIMEOUT   curl timeout in seconds (default 30)
#
# The script is idempotent: resources that already exist (matched by name)
# are skipped. It uses jq when available, python3 otherwise, and grep as a
# last-resort fallback.
#
# API endpoints used:
#   POST /api/v1/auth/login               → { "token": "..." }
#   GET  /api/v1/workflows                → [ workflow, ... ]           (idempotency)
#   POST /api/v1/workflows                → { workflow }                (create)
#   POST /api/v1/workflows/{id}/steps     → { step }                    (create step)
#   GET  /api/v1/schedules                → { "schedules": [...] }      (idempotency)
#   POST /api/v1/schedules                → { "schedule": {...} }       (create)
#   GET  /api/v1/alerts/rules             → [ rule, ... ]               (idempotency)
#   POST /api/v1/alerts/rules             → { rule }                    (create)
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB_DIR="$SCRIPT_DIR"

# ── Configuration (env overrides) ────────────────────────────────────────────
DEFAULT_URL="${RENOS_URL:-http://localhost:8443}"
DEFAULT_EMAIL="${RENOS_EMAIL:-admin@renos.local}"
TIMEOUT="${RENOS_TIMEOUT:-30}"
INSECURE=""
if [[ "${RENOS_INSECURE:-0}" == "1" ]]; then INSECURE="-k"; fi

ALL=0
DRY_RUN=0

CREATED=0
SKIPPED=0
FAILED=0

# ── Help ─────────────────────────────────────────────────────────────────────
usage() {
  sed -n '2,25p' "$0" | sed 's/^# \{0,1\}//'
  echo
  echo "Available packs:"
  local p
  for p in "$LIB_DIR"/*/pack.json; do
    [[ -f "$p" ]] || continue
    printf '  %-20s %s\n' "$(basename "$(dirname "$p")")" "$(name_of "$p")"
  done
}

# ── JSON tool detection ──────────────────────────────────────────────────────
JSON_TOOL=""
if command -v jq >/dev/null 2>&1; then
  JSON_TOOL="jq"
elif command -v python3 >/dev/null 2>&1; then
  JSON_TOOL="py"
elif command -v grep >/dev/null 2>&1; then
  JSON_TOOL="grep"
fi

if [[ -z "$JSON_TOOL" ]]; then
  echo "error: curl plus jq, python3, or grep are required" >&2
  exit 1
fi
if ! command -v curl >/dev/null 2>&1; then
  echo "error: curl is required" >&2
  exit 1
fi

TMP="$(mktemp -d)"
CURL_OUT="$TMP/curl.out"
trap 'rm -rf "$TMP"' EXIT

TOKEN=""
HTTP_STATUS=""

# ── Small JSON helpers (jq / python3 / grep) ────────────────────────────────
name_of() {
  local f="$1"
  case "$JSON_TOOL" in
    jq)   jq -r '.name // empty' "$f" 2>/dev/null || true ;;
    py)   python3 -c 'import sys,json; print(json.load(open(sys.argv[1])).get("name",""))' "$f" 2>/dev/null || true ;;
    grep) grep -oE '"name"[[:space:]]*:[[:space:]]*"[^"]+"' "$f" 2>/dev/null | head -n1 | sed -E 's/^.*:[[:space:]]*"([^"]+)"/\1/' || true ;;
  esac
}

# find_id_by_name <listfile> <name> <container|none>
find_id_by_name() {
  local listfile="$1" name="$2" container="${3:-none}"
  case "$JSON_TOOL" in
    jq)
      if [[ "$container" == "none" ]]; then
        jq -r --arg n "$name" '.[] | select(.name == $n) | .id' "$listfile" 2>/dev/null | head -n1
      else
        jq -r --arg n "$name" --arg c "$container" '.[$c][] | select(.name == $n) | .id' "$listfile" 2>/dev/null | head -n1
      fi
      ;;
    py)
      python3 - "$listfile" "$name" "$container" <<'PY' 2>/dev/null || true
import sys, json
data = json.load(open(sys.argv[1]))
name, container = sys.argv[2], sys.argv[3]
items = data.get(container, []) if container != "none" else (data or [])
for it in items:
    if it.get("name") == name:
        print(it.get("id") or "")
        break
PY
      ;;
    grep)
      local esc
      esc=$(printf '%s' "$name" | sed -e 's/[][().+*?^$\\|{}]/\\&/g')
      if grep -qE "\"name\"[[:space:]]*:[[:space:]]*\"$esc\"" "$listfile" 2>/dev/null; then
        grep -oE '"id"[[:space:]]*:[[:space:]]*"[^"]+"' "$listfile" | head -n1 | sed -E 's/^.*:[[:space:]]*"([^"]+)"/\1/'
      fi
      ;;
  esac
}

extract_id() {
  local f="$1"
  case "$JSON_TOOL" in
    jq)   jq -r '.id // empty' "$f" 2>/dev/null || true ;;
    py)   python3 -c 'import sys,json; print(json.load(open(sys.argv[1])).get("id",""))' "$f" 2>/dev/null || true ;;
    grep) grep -oE '"id"[[:space:]]*:[[:space:]]*"[^"]+"' "$f" | head -n1 | sed -E 's/^.*:[[:space:]]*"([^"]+)"/\1/' || true ;;
  esac
}

count_items() {
  case "$JSON_TOOL" in
    jq)   jq 'length' "$1" 2>/dev/null || echo 0 ;;
    py)   python3 -c 'import sys,json; print(len(json.load(open(sys.argv[1]))))' "$1" 2>/dev/null || echo 0 ;;
    grep) echo 0 ;;
  esac
}

token_from() {
  case "$JSON_TOOL" in
    jq)   printf '%s' "$1" | jq -r '.token // empty' 2>/dev/null || true ;;
    py)   printf '%s' "$1" | python3 -c 'import sys,json; print(json.load(sys.stdin).get("token",""))' 2>/dev/null || true ;;
    grep) printf '%s' "$1" | grep -oE '"token"[[:space:]]*:[[:space:]]*"[^"]+"' | head -n1 | sed -E 's/^.*:[[:space:]]*"([^"]+)"/\1/' || true ;;
  esac
}

validate_json() {
  case "$JSON_TOOL" in
    jq) jq empty "$1" >/dev/null 2>&1 ;;
    py) python3 -c 'import json,sys; json.load(open(sys.argv[1]))' "$1" >/dev/null 2>&1 ;;
    grep) grep -q '"name"' "$1" ;;
  esac
}

# ── Workflow file normalization ──────────────────────────────────────────────
# Library files already use the v1 API schema; these normalizers also accept
# the legacy rich format (triggers[]/nodes[]) from older examples.
make_workflow_body() {  # <infile> <outfile>
  case "$JSON_TOOL" in
    jq)
      jq -c '{name, description: (.description // ""), enabled: true,
              trigger_type: (.trigger_type // .triggers[0].type // "manual"),
              trigger_config: (.trigger_config // .triggers[0].config // {}),
              tags: (.tags // [])}' "$1" > "$2"
      ;;
    py)
      python3 - "$1" > "$2" <<'PY'
import sys, json
d = json.load(open(sys.argv[1]))
if d.get("trigger_type"):
    tt, tc = d["trigger_type"], d.get("trigger_config") or {}
else:
    t0 = (d.get("triggers") or [{}])[0]
    tt, tc = t0.get("type") or "manual", t0.get("config") or {}
out = {"name": d.get("name", ""), "description": d.get("description", ""),
       "enabled": True, "trigger_type": tt, "trigger_config": tc,
       "tags": d.get("tags") or []}
print(json.dumps(out))
PY
      ;;
    grep) cp "$1" "$2" ;;
  esac
}

make_workflow_steps() {  # <infile> <outfile>  -> JSON array of steps
  case "$JSON_TOOL" in
    jq)
      jq -c 'if ((.steps // []) | length) > 0 then .steps
             else ((.nodes // []) | to_entries | map({
               step_order: .key,
               name: .value.name,
               step_type: (.value.type // "action"),
               config: (.value.config + (if .value.retry != null then {retry: .value.retry} else {} end)),
               error_handler: (.value.error_handler // {})
             })) end' "$1" > "$2"
      ;;
    py)
      python3 - "$1" > "$2" <<'PY'
import sys, json
d = json.load(open(sys.argv[1]))
steps = d.get("steps") or []
if not steps:
    steps = []
    for i, n in enumerate(d.get("nodes") or []):
        cfg = dict(n.get("config") or {})
        if n.get("retry") is not None:
            cfg["retry"] = n["retry"]
        steps.append({"step_order": i, "name": n.get("name", ""),
                      "step_type": n.get("type") or "action", "config": cfg,
                      "error_handler": n.get("error_handler") or {}})
print(json.dumps(steps))
PY
      ;;
    grep) echo '[]' > "$2" ;;
  esac
}

extract_step() {  # <arrayfile> <index> <outfile>
  case "$JSON_TOOL" in
    jq) jq -c ".[$2]" "$1" > "$3" ;;
    py) python3 -c 'import sys,json; print(json.dumps(json.load(open(sys.argv[1]))[int(sys.argv[2])]))' "$1" "$2" > "$3" ;;
    grep) : > "$3" ;;
  esac
}

# ── HTTP ─────────────────────────────────────────────────────────────────────
# http_call <method> <url> [bodyfile]  -> prints response body, writes the HTTP
# status code to $HTTP_STATUS_FILE (a file, because command substitution runs
# in a subshell and cannot set the caller's variables). Read it back with
# http_status() after each call.
HTTP_STATUS_FILE="$TMP/http-status"
http_status() { HTTP_STATUS="$(cat "$HTTP_STATUS_FILE" 2>/dev/null || true)"; }

http_call() {
  local method="$1" url="$2" body="${3:-}"
  local args=(-s -o "$CURL_OUT" -w '%{http_code}' $INSECURE --max-time "$TIMEOUT" -X "$method")
  if [[ -n "${TOKEN:-}" ]]; then
    args+=(-H "Authorization: Bearer $TOKEN")
  fi
  args+=(-H 'Content-Type: application/json')
  if [[ -n "$body" ]]; then
    args+=(--data-binary @"$body")
  fi
  local status
  status=$(curl "${args[@]}" "$url" || true)
  printf '%s' "$status" > "$HTTP_STATUS_FILE"
  cat "$CURL_OUT"
}

login() {
  local body="$TMP/login-body"
  printf '{"email": "%s", "password": "%s"}' "$EMAIL" "$PASSWORD" > "$body"
  local resp token
  resp=$(curl -s $INSECURE --max-time "$TIMEOUT" -X POST \
    -H 'Content-Type: application/json' \
    --data-binary @"$body" "$API/auth/login" || true)
  token=$(token_from "$resp")
  if [[ -z "$token" ]]; then
    echo "error: authentication failed (check the base URL and credentials)" >&2
    echo "  response: $(printf '%s' "$resp" | head -c 300)" >&2
    exit 1
  fi
  TOKEN="$token"
  echo "Authenticated as $EMAIL"
}

# ── Importers ────────────────────────────────────────────────────────────────
import_workflow() {
  local file="$1" name
  name=$(name_of "$file")
  if [[ -z "$name" ]]; then
    echo "  [error] workflow file '$file' has no name" >&2
    FAILED=$((FAILED + 1)); return 1
  fi
  if [[ "$DRY_RUN" != "1" ]]; then
    local existing
    existing=$(find_id_by_name "$LIST_WF" "$name" "none" || true)
    if [[ -n "$existing" ]]; then
      echo "  [skip]  workflow '$name' already exists"
      SKIPPED=$((SKIPPED + 1)); return 0
    fi
  else
    echo "  [dry]   would create workflow '$name'"
    CREATED=$((CREATED + 1)); return 0
  fi

  make_workflow_body "$file" "$TMP/wf-body"
  local resp
  resp=$(http_call POST "$API/workflows" "$TMP/wf-body")
  http_status
  echo "$resp" > "$TMP/wf-resp"
  if [[ "$HTTP_STATUS" != "201" ]]; then
    echo "  [error] create workflow '$name' failed (HTTP $HTTP_STATUS): $(printf '%s' "$resp" | head -c 300)" >&2
    FAILED=$((FAILED + 1)); return 1
  fi
  local wfid
  wfid=$(extract_id "$TMP/wf-resp")
  if [[ -z "$wfid" ]]; then
    echo "  [error] create workflow '$name': no id in response" >&2
    FAILED=$((FAILED + 1)); return 1
  fi
  echo "  [ok]    workflow '$name' created ($wfid)"
  CREATED=$((CREATED + 1))

  if [[ "$JSON_TOOL" == "grep" ]]; then
    echo "  [warn]  grep fallback cannot enumerate steps; import steps via the UI" >&2
    return 0
  fi

  make_workflow_steps "$file" "$TMP/steps"
  local total i sname sresp
  total=$(count_items "$TMP/steps")
  for ((i = 0; i < total; i++)); do
    extract_step "$TMP/steps" "$i" "$TMP/step"
    sname=$(name_of "$TMP/step")
    sresp=$(http_call POST "$API/workflows/$wfid/steps" "$TMP/step")
    http_status
    echo "$sresp" > "$TMP/step-resp"
    if [[ "$HTTP_STATUS" != "201" ]]; then
      echo "    [error] step '$sname' failed (HTTP $HTTP_STATUS): $(printf '%s' "$sresp" | head -c 200)" >&2
      FAILED=$((FAILED + 1))
    else
      echo "    [ok]    step '$sname'"
    fi
  done
}

import_schedule() {
  local file="$1" name
  name=$(name_of "$file")
  if [[ -z "$name" ]]; then
    echo "  [error] schedule file '$file' has no name" >&2
    FAILED=$((FAILED + 1)); return 1
  fi
  if [[ "$DRY_RUN" != "1" ]]; then
    local existing
    existing=$(find_id_by_name "$LIST_SCHED" "$name" "schedules" || true)
    if [[ -n "$existing" ]]; then
      echo "  [skip]  schedule '$name' already exists"
      SKIPPED=$((SKIPPED + 1)); return 0
    fi
  else
    echo "  [dry]   would create schedule '$name'"
    CREATED=$((CREATED + 1)); return 0
  fi

  local resp
  resp=$(http_call POST "$API/schedules" "$file")
  http_status
  if [[ "$HTTP_STATUS" != "201" ]]; then
    echo "  [error] create schedule '$name' failed (HTTP $HTTP_STATUS): $(printf '%s' "$resp" | head -c 300)" >&2
    FAILED=$((FAILED + 1)); return 1
  fi
  echo "  [ok]    schedule '$name' created"
  CREATED=$((CREATED + 1))
}

import_alert() {
  local file="$1" name
  name=$(name_of "$file")
  if [[ -z "$name" ]]; then
    echo "  [error] alert file '$file' has no name" >&2
    FAILED=$((FAILED + 1)); return 1
  fi
  if [[ "$DRY_RUN" != "1" ]]; then
    local existing
    existing=$(find_id_by_name "$LIST_ALERT" "$name" "none" || true)
    if [[ -n "$existing" ]]; then
      echo "  [skip]  alert rule '$name' already exists"
      SKIPPED=$((SKIPPED + 1)); return 0
    fi
  else
    echo "  [dry]   would create alert rule '$name'"
    CREATED=$((CREATED + 1)); return 0
  fi

  local resp
  resp=$(http_call POST "$API/alerts/rules" "$file")
  http_status
  if [[ "$HTTP_STATUS" != "201" ]]; then
    echo "  [error] create alert rule '$name' failed (HTTP $HTTP_STATUS): $(printf '%s' "$resp" | head -c 300)" >&2
    FAILED=$((FAILED + 1)); return 1
  fi
  echo "  [ok]    alert rule '$name' created"
  CREATED=$((CREATED + 1))
}

# ── Pack orchestration ───────────────────────────────────────────────────────
run_import() {
  local type="$1" file="$2"
  local fullpath="$PACK_DIR/$file"
  if [[ ! -f "$fullpath" ]]; then
    echo "  [error] item file not found: $file" >&2
    FAILED=$((FAILED + 1)); return 1
  fi
  if ! validate_json "$fullpath"; then
    echo "  [error] invalid JSON in $file" >&2
    FAILED=$((FAILED + 1)); return 1
  fi
  case "$type" in
    workflow)            import_workflow "$fullpath" ;;
    schedule)            import_schedule "$fullpath" ;;
    alert|alert_rule)    import_alert "$fullpath" ;;
    *) echo "  [error] unknown item type '$type' in pack.json" >&2; FAILED=$((FAILED + 1)); return 1 ;;
  esac
}

read_manifest() {
  case "$JSON_TOOL" in
    jq)
      jq -r '.items[]? | [.type, .file] | @tsv' "$1" || true
      ;;
    py)
      python3 -c 'import sys,json
data = json.load(open(sys.argv[1]))
for i in data.get("items", []):
    print(i.get("type") + "\t" + i.get("file"))' "$1" || true
      ;;
    grep)
      grep -oE '\{[[:space:]]*"type"[[:space:]]*:[[:space:]]*"[^"]+",[[:space:]]*"file"[[:space:]]*:[[:space:]]*"[^"]+"[[:space:]]*\}' "$1" 2>/dev/null \
        | sed -E 's/\{[[:space:]]*"type"[[:space:]]*:[[:space:]]*"([^"]+)",[[:space:]]*"file"[[:space:]]*:[[:space:]]*"([^"]+)"[[:space:]]*\}/\1\t\2/' || true
      ;;
  esac
}

import_pack() {
  local dir="$1"
  local name
  name=$(name_of "$dir/pack.json")
  echo ""
  echo "== Pack: ${name:-$(basename "$dir")} ($(basename "$dir")) =="

  local items line type file
  items=$(read_manifest "$dir/pack.json" || true)
  if [[ -z "$items" ]]; then
    echo "  [error] no importable items found in pack.json (check JSON_TOOL: $JSON_TOOL)" >&2
    FAILED=$((FAILED + 1)); return 1
  fi
  while IFS=$'\t' read -r type file; do
    [[ -z "$type" && -z "$file" ]] && continue
    run_import "$type" "$file" || true
  done <<< "$items"
}

summary() {
  echo ""
  echo "=== Summary ==="
  echo "  Created:  $CREATED"
  echo "  Skipped:  $SKIPPED"
  echo "  Failed:   $FAILED"
}

# ── Main ─────────────────────────────────────────────────────────────────────
POSITIONAL=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    --all) ALL=1; shift ;;
    --dry-run) DRY_RUN=1; shift ;;
    -h|--help) usage; exit 0 ;;
    -*) echo "error: unknown option: $1" >&2; usage >&2; exit 1 ;;
    *) POSITIONAL+=("$1"); shift ;;
  esac
done

if [[ "$ALL" == "1" ]]; then
  BASE_URL="${POSITIONAL[0]:-${RENOS_URL:-$DEFAULT_URL}}"
  EMAIL="${POSITIONAL[1]:-${RENOS_EMAIL:-$DEFAULT_EMAIL}}"
  PASSWORD="${POSITIONAL[2]:-${RENOS_PASSWORD:-}}"
  [[ -z "$BASE_URL" || -z "$EMAIL" ]] && { usage >&2; exit 1; }
else
  PACK="${POSITIONAL[0]:-}"
  BASE_URL="${POSITIONAL[1]:-${RENOS_URL:-$DEFAULT_URL}}"
  EMAIL="${POSITIONAL[2]:-${RENOS_EMAIL:-$DEFAULT_EMAIL}}"
  PASSWORD="${POSITIONAL[3]:-${RENOS_PASSWORD:-}}"
  if [[ -z "$PACK" ]]; then
    usage
    exit 0
  fi
  if [[ -z "$EMAIL" ]]; then
    echo "error: email is required" >&2
    exit 1
  fi
fi

BASE_URL="${BASE_URL%/}"
API="$BASE_URL/api/v1"

if [[ "$DRY_RUN" != "1" && -z "$PASSWORD" ]]; then
  if [[ -t 0 ]]; then
    read -r -s -p "Password for $EMAIL: " PASSWORD
    echo ""
  else
    echo "error: no password provided (set RENOS_PASSWORD or pass it as an argument)" >&2
    exit 1
  fi
fi

if [[ "$DRY_RUN" == "1" ]]; then
  echo "DRY RUN — validating packs only, nothing will be imported."
else
  login
  LIST_WF="$TMP/list-workflows"
  LIST_SCHED="$TMP/list-schedules"
  LIST_ALERT="$TMP/list-alerts"
  http_call GET "$API/workflows" > "$LIST_WF" || true
  http_call GET "$API/schedules" > "$LIST_SCHED" || true
  http_call GET "$API/alerts/rules" > "$LIST_ALERT" || true
fi

if [[ "$ALL" == "1" ]]; then
  PACKS=""
  for p in "$LIB_DIR"/*/pack.json; do
    [[ -f "$p" ]] || continue
    PACKS="$PACKS $(basename "$(dirname "$p")")"
  done
  for p in $PACKS; do
    PACK_DIR="$LIB_DIR/$p"
    if [[ ! -f "$PACK_DIR/pack.json" ]]; then
      echo "error: no such pack: $p" >&2
      FAILED=$((FAILED + 1))
      continue
    fi
    import_pack "$PACK_DIR" || true
  done
else
  PACK_DIR="$LIB_DIR/$PACK"
  if [[ ! -f "$PACK_DIR/pack.json" ]]; then
    echo "error: no such pack: '$PACK'" >&2
    echo "  expected $PACK_DIR/pack.json" >&2
    echo "  run ./import.sh --help to list available packs" >&2
    exit 1
  fi
  import_pack "$PACK_DIR" || true
fi

summary
if [[ "$FAILED" -gt 0 ]]; then
  exit 1
fi
exit 0
