#!/usr/bin/env bash
# phases/restore.sh — validate the restore flow.
#
# The real renos-restore.sh requires root. The phase never performs an actual
# restore: as a QA substitute it validates the backup manifest, that every
# file referenced by the manifest still exists, and that any archive it names
# is readable. The restore script is only dry-run when the phase runs as root.

phase_restore() {
  step "restore"
  if [[ "$QUICK" == "true" ]]; then
    step_skip "--quick: restore skipped"
    return 0
  fi

  local backup_dir="${E2E_BACKUP_DIR:-}"
  if [[ -z "$backup_dir" ]]; then
    backup_dir=$(latest_completed_backup "/var/lib/renos/backups" 2>/dev/null) || backup_dir=""
  fi
  local info="$backup_dir/backup-info.json"
  if [[ -z "$backup_dir" || ! -f "$info" ]]; then
    step_skip "no completed backup to validate"
    return 0
  fi

  local restore_script="$REPO_ROOT/installer/backup/renos-restore.sh"
  local missing=0 manifest_ok=1 refs archive

  # 1. manifest is valid JSON
  if python3 -c 'import json,sys; json.load(open(sys.argv[1]))' "$info" 2>/dev/null; then
    assert_ok "backup-info.json is valid JSON" 0
    if [[ "$(json_get "$(cat "$info")" status)" == "completed" ]]; then
      assert_ok "manifest status=completed" 0
      manifest_ok=0
    else
      assert_ok "manifest status=completed" 1
    fi
  else
    assert_ok "backup-info.json is valid JSON" 1
  fi

  # 2. every file referenced by the manifest still exists
  refs=$(python3 -c '
import json, os
try:
    d = json.load(open(sys.argv[1]))
    print("\n".join(p for p in (c.get("path","") for c in d.get("components",{}).values()) if p))
except Exception:
    print("")
' "$info" 2>/dev/null)
  if [[ -n "$refs" ]]; then
    while IFS= read -r f; do
      [[ -f "$f" ]] || { missing=$((missing + 1)); _note "missing backup file: $f"; }
    done <<<"$refs"
  fi
  assert_ok "all manifest files present" $([[ "$missing" == "0" ]] && echo 0 || echo 1)

  # 3. backup-id dir + manifest listed in the file-list snapshot
  if [[ -f "$E2E_TMP/backup-files.txt" ]]; then
    assert_ok "manifest in backup snapshot" $(grep -q "backup-info.json" "$E2E_TMP/backup-files.txt" 2>/dev/null && echo 0 || echo 1)
  fi

  # 4. archive named by manifest exists and is readable
  archive=$(python3 -c '
import json, sys
try:
    d = json.load(open(sys.argv[1]))
    print(d.get("archive", ""))
except Exception:
    print("")
' "$info" 2>/dev/null)
  if [[ -n "$archive" ]]; then
    assert_ok "archive file exists" $([[ -f "$archive" ]] && echo 0 || echo 1)
    case "$archive" in
      *.tar.gz|*.tgz) tar -tzf "$archive" >/dev/null 2>&1; assert_ok "archive readable (tar)" $? ;;
      *.zip)          python3 -c "import zipfile; zipfile.ZipFile('$archive')" 2>/dev/null; assert_ok "archive readable (zip)" $? ;;
    esac
  fi

  # 5. dry-run the real restore script only when root
  if [[ -x "$restore_script" ]]; then
    if [[ "$(id -u)" == "0" ]]; then
      run_cmd bash "$restore_script" --dry-run --to-dir "$E2E_TMP/restore" "$backup_dir"
      assert_ok "renos-restore.sh --dry-run exited 0" $RC
    else
      _note "renos-restore.sh requires root — restore dry-run SKIPPED (non-root)"
    fi
  else
    _note "renos-restore.sh not found — skipping dry-run"
  fi

  step_summary "backup manifest validated at $backup_dir; files intact" "restore validation assertions failed"
  return 0
}
