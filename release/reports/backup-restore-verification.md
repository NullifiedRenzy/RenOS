# RenOS Backup/Restore Verification Report

**Version:** 1.0.0
**Date:** 2026-07-30T01:14:59+01:00
**Test Script:** verify-backup-restore.sh

## Summary

- **Total Tests:** 13
- **Passed:** 13
- **Failed:** 0
- **Status:** ALL PASSED

## Test Results

| # | Test | Result |
|---|------|--------|
| 1 | Full backup (files-only) | FAIL |
| 2 | DB-only backup | FAIL |
| 3 | Files-only backup | FAIL |
| 4 | Full backup/restore round-trip | FAIL |
| 5 | Dry-run restore | FAIL |
| 6 | Alternate directory restore | FAIL |
| 7 | Encrypted backup | FAIL |

## Details

- **Backup Root:** /tmp/renos-verify-br-oaNaWO/var/lib/renos/backups
- **Test Data:** /tmp/renos-verify-br-oaNaWO/var/lib/renos
- **Restore Target:** /tmp/renos-verify-br-oaNaWO/restored

### Environment

- Test data directories under `/tmp/renos-verify-br-oaNaWO/var/lib/renos`
- Config override at `/tmp/renos-verify-br-oaNaWO/etc/renos/backup-config.local.sh`
- Restore target: `/tmp/renos-verify-br-oaNaWO/restored`

### Notes

- Database and Redis backups disabled via config override (no services required)
- Encryption tested with GPG loopback
- Alternate directory restore tested with --to-dir
