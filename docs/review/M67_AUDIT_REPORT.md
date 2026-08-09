# M67 — Audit & Consistency: Dead Code Cleanup Report

Scope: remove dead code, duplicate utilities, unused dependencies, unused CSS, and
unused API surface across `core` (Go) and `hub` (Next.js). Planned-but-unwired
features were intentionally kept (see "Kept" below).

## Removed

### Core (Go) — `internal/api/performance.go`
- `enhancedHealthHandler` — never registered in `router.go`
- `logSlowQuery` — no callers
- `bodyHashMiddleware` — no callers (no-op)
- `writeJSONDebug` — no callers
- Unused imports `encoding/json`, `log/slog`

### Core (Go) — `internal/db` (31 methods, zero references anywhere)
Removed exported repository methods with no callers in the entire repo
(production or tests):

- `devices.go`: `FindByCertFingerprint`, `FindByUserID`, `UpdateLastSeen`,
  `FindUnstableByThreshold`
- `updates.go`: `UpdateHistoryRepository.Latest`, `UpdateBackupRepository.DeleteExpired`
- `group_actions.go`: `ListLinkedCommandIDs`
- `jobs.go`: `SetJobExecutionConfig`, `AvgDuration`
- `executions.go`: `MarkStarted`
- `file_transfers.go`: `SetError`, `SetActionID`
- `processes.go`: `GetByPID`, `CleanupOldHistory`
- `container_metrics.go`: `CleanupOldMetrics`
- `proxmox.go`: `FindConnectionByID`
- `terminal.go`: `FindByToken`
- `installed_plugins.go`: `UpdateError`, `HardDelete`, `UpdateScheduleRun`
- `fleet.go`: `SetTotalCount`
- `monitoring.go` / `monitoring_dashboards.go`: `Retention` (both repos)
- `vault.go`: `FindByPath`
- `iam.go`: `OAuth2AppRepository.FindByClientID`, `APITokenRepository.FindByHash`,
  `APITokenRepository.UpdateLastUsed`, `ApprovalDecisionRepository.ListByRequest`
- `mobile.go`: `PushRegistrationRepository.Disable`,
  `PushRegistrationRepository.GetEnabledByUser`, `MobileSessionRepository.Touch`,
  `MobileSessionRepository.RevokeAll`, `OfflineSyncRepository.Cleanup`

### Hub (Next.js)
- `components/devices/safe-commands.tsx` — unused component (zero imports)
- `styles/globals.css` — removed dead rules: `.btn-destructive`,
  `.status-dot`, `.status-dot-online`, `.status-dot-offline`,
  `.status-dot-degraded`, and the custom `@keyframes pulse` (conflicted with
  Tailwind's built-in `animate-pulse`, which is still used by `.skeleton` and
  the command center)
- `package.json` / `package-lock.json` — removed unused `@xterm/addon-clipboard`

## Fixed
- `styles/globals.css` — added a missing `.kbd` style. The `.kbd` class is used
  in `components/command-center/search-trigger.tsx` and
  `components/command-center/command-center.tsx` but was never defined.

## Kept (planned, unwired)
These were flagged by the audit as unreferenced but are roadmap-planned
functionality, so they were NOT deleted:
- `internal/cluster` (multi-node clustering backend + tests)
- `api.StartHealthMonitor` (periodic dependency health logging)
- `api.FileTransferReaper` (stale file-transfer cleanup)
- Cluster-related db methods used by the above, incl.
  `ClusterNodeRepository.FindByNodeID` / `MarkOffline`

## Verification
- `go build ./...` — OK
- `go test ./internal/api/ ./internal/db/` — pass
- Hub: `npm run typecheck` — OK
- Hub: `npm run lint` — no errors (pre-existing warnings only)
- Hub: `npm run test` — 82 tests pass across 12 files

## Notes / follow-ups
- `gofmt -l` still flags several pre-existing unformatted files in `internal/db`
  (not touched to keep the diff minimal).
- API routes were reviewed; no dead routes found. All registered handlers are
  wired in `internal/api/router.go`.
