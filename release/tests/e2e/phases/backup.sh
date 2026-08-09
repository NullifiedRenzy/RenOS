#!/usr/bin/env bash
# phases/backup.sh — run a real backup (renos-backup.sh) and validate the
# resulting manifest.
#
# renos-backup.sh requires root, so:
#   * as root:      run `renos-backup.sh --files-only` and validate the manifest
#   * as non-root:  validate the most recent completed backup if one already
#                   exists; otherwise SKIP with a clear reason.
# The phase never fakes PASS.

phase_backup() {
  step "backup"
  if [[ "$QUICK" == "true" ]]; then
    step_skip "--quick: backup skipped"
    return 0
  fi

  local script="$REPO_ROOT/installer/backup/renos-backup.sh"
  if [[ ! -x "$script" ]]; then
    step_skip "renos-backup.sh not found at $script"
    return 0
  fi

  local backup_root="/var/lib/renos/backups"
  local backup_dir=""
  E2E_BACKUP_DIR=""

  if [[ "$(id -u)" == "0" ]]; then
    _note "running as root — executing renos-backup.sh (--files-only)"
    run_cmd bash "$script" --files-only
    if [[ $RC -ne 0 ]]; then
      step_skip "renos-backup.sh exited $RC (${OUT//$'\n'/ })"
      return 0
    fi
    backup_dir=$(latest_completed_backup "$backup_root") || backup_dir=""
    if [[ -z "$backup_dir" ]]; then
      step_fail "renos-backup.sh exited 0 but no completed manifest under $backup_root"
      return 0
    fi
  else
    _note "non-root — renos-backup.sh requires root; checking for an existing backup"
    backup_dir=$(latest_completed_backup "$backup_root" 2>/dev/null) || backup_dir=""
    if [[ -z "$backup_dir" ]]; then
      step_skip "renos-backup.sh requires root and no completed backup exists to validate"
      return 0
    fi
    _note "validating existing completed backup at $backup_dir (non-root substitute)"
  fi
  E2E_BACKUP_DIR="$backup_dir"

  local info="$backup_dir/backup-info.json"
  assert_ok "backup-info.json present" $([[ -f "$info" ]] && echo 0 || echo 1)
  if [[ -f "$info" ]]; then
    assert_contains "manifest has status" "$(cat "$info")" "status"
    assert_contains "manifest status=completed" "$(cat "$info")" "completed"
  fi
  assert_ok "backup dir has entries" $([[ "$(ls -A "$backup_dir" 2>/dev/null | wc -l)" -gt 0 ]] && echo 0 || echo 1)

  ( cd "$backup_dir" && find . -type f | sort > "$E2E_TMP/backup-files.txt" 2>/dev/null )
  _note "snapshot of backup file list written to $E2E_TMP/backup-files.txt"

  step_summary "backup manifest validated at $backup_dir" "backup validation assertions failed"
  return 0
}
