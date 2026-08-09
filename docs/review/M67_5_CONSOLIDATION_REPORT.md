# M67.5 — Internal Platform Consolidation & Technical Debt Elimination Report

Scope: reduce internal duplication and eliminate provably-unused code across
`core` (Go) and `hub` (Next.js) with **zero behavior change** — no new
user-facing features, no API route / DB schema / CLI / SDK / plugin / test
changes. Every deletion was verified by grepping the full repo for references
(production and tests) before removal, then by a full rebuild + test pass.

## Summary

| Metric | Count |
|---|---|
| New shared modules | 3 (`lib/format.ts`, `components/ui/action-output-modal.tsx`, `components/ui/state-dot.tsx`) |
| Files deleted | 2 (`components/devices/empty-state.tsx`, `core/internal/agent/` empty dir) |
| Local helpers removed | ~18 (duplicated formatters, status/state helpers, status components) |
| Components/hooks consolidated | EmptyState (2 → 1), ActionOutputModal (2 → 1), StateDot (2 → 1), Loading/ErrorState/EmptyState in fleet page (3 local → shared) |
| Core Go dead functions/types removed | 9 (`HasRole`, `ResetRateLimiters`, `validateStringLen`, `validateEmail`, `requireRole`, `requestSizeTracker` + 3 methods, `connectRequest`) |
| Inline auth-logic copies deduplicated | 2 (`connectHandler`, `heartbeatHandler` → `verifyDeviceSecret`) |
| Lines removed (net) | ~250 (hub ~120, core ~130) |

---

## Hub — consolidation

### 1. Shared formatters — new `hub/lib/format.ts`
Created a single, dependency-free formatting module and removed all per-file
redefinitions. Behavior preserved exactly at every call site (identical
rounding, units, and edge cases):

| Function | Former duplicate locations |
|---|---|
| `formatBytes` | `process-manager.tsx`, `devices/[id]/docker/page.tsx`, `devices/[id]/proxmox/page.tsx` |
| `formatFileSize` | `hooks/use-file-actions.ts` (was also `formatStorageSize` in `hooks/use-storage.ts`) |
| `formatUptime` | `devices/[id]/proxmox/page.tsx`, `fleet/page.tsx` (fleet's always-hours form preserved via `{ alwaysHours: true }`; proxmox's `"down"` label via `{ empty: "down" }`) |
| `formatTime` | `command-history.tsx`, `terminal-execution.tsx` (identical em-dash + `toLocaleTimeString`) |
| `formatTimestamp` | `hooks/use-file-actions.ts` |
| `usagePercent` | `hooks/use-storage.ts` |

Consumers updated to import from `@/lib/format`:
`device-storage.tsx`, `device-file-browser.tsx`, `process-manager.tsx`,
`devices/[id]/docker/page.tsx`, `devices/[id]/proxmox/page.tsx`,
`fleet/page.tsx`, `command-history.tsx`, `terminal-execution.tsx`.
The `formatStorageSize` name was dropped in favor of `formatFileSize`
(identical implementation).

### 2. EmptyState unification — 2 components → 1
`components/ui/empty-state.tsx` is now the single EmptyState. It accepts the
superset API of both prior variants:
- `icon?: string` (emoji, legacy `devices/empty-state` style) or the default
  SVG badge (legacy `ui/empty-state` style)
- `action?: { label, onClick }` (renders `btn-primary`) or `React.ReactNode`

Deleted `components/devices/empty-state.tsx` and re-pointed all 11 importing
pages (`devices`, `devices/[id]`, `groups`, `groups/[id]`, `jobs`, `jobs/[id]`,
`schedules`, `schedules/[id]`, `users`, `policies`, `policies/[id]`). The
dashboard's `devices-empty-state.tsx` continues to use the SVG/ReactNode
variant unchanged.

### 3. ActionOutputModal — 2 byte-for-byte copies → 1
`components/ui/action-output-modal.tsx` replaces the duplicated modals in
`device-containers.tsx` and `devices/[id]/docker/page.tsx`. The only original
difference (max width) is now a `size?: "md" | "lg"` prop — the docker page
passes `size="lg"`, containers uses the default. Error/exit-code rendering was
identical in both copies and preserved.

### 4. Container status dot — 2 copies → 1
`components/ui/state-dot.tsx` exports `StateDot` (and `stateDotColor`) using
the merged `STATE_COLORS` map (superset incl. `removing`). Replaces local
`STATE_COLORS` + `stateDot` in `device-containers.tsx` and
`devices/[id]/docker/page.tsx`. (proxmox's `statusDot` also renders a status
label, so it was left in place — semantically different.)

### 5. fleet page — 3 local status components → shared
Replaced the page-local `Loading`, `ErrorState`, `EmptyState` (plain
text-only) with the shared `LoadingState` / `ErrorState` / `EmptyState`
primitives. Error display now surfaces the actual error object rather than a
static string (strict improvement, same trigger conditions).

### 6. Dependency audit
Checked every runtime dependency against real imports across
`lib/`, `components/`, `app/`, `hooks/`, `stores/`:
`@tanstack/react-query`, `@tanstack/react-virtual`, all `@xterm/*` addons,
`clsx`, `tailwind-merge` (both via `lib/utils.ts`), `lucide-react`, `next`,
`react`, `react-dom`, and `zustand` (used by `stores/*`). **All are used** —
no unused runtime dependency found; nothing to remove.

### 7. CSS dead-selector pass
Removed the never-referenced `.card-interactive` rule (+ hover) from
`styles/globals.css`. Verified `confetti-fall` (used by onboarding-wizard),
`animate-fade-in-up`, `.kbd`, `.input`, all `.btn-*`/`.badge-*` rules, and
`.skeleton` are still referenced.

---

## Core (Go) — dead code removal

Verified zero references anywhere (production + tests) before each deletion.

| Symbol | Location | Notes |
|---|---|---|
| `HasRole` | `internal/db/policies.go` | no callers; dropped now-unused `auth` import |
| `ResetRateLimiters` | `internal/api/middleware.go` | "useful for tests" — no test actually calls it |
| `validateStringLen` | `internal/api/middleware.go` | no callers; also removed the now-unused `max*Len` const block and `fmt` import |
| `validateEmail` | `internal/api/middleware.go` | no callers (email validation now lives in the auth flow) |
| `requireRole` | `internal/api/middleware.go` | convenience wrapper with zero callers; `requireOneOfRoles` + `roleRequired` are heavily wired and retained |
| `requestSizeTracker` (+ `newRequestSizeTracker`, `track`, `getTotal`) | `internal/api/security.go` | instantiated into package var `sizeTracker` but never read/called anywhere |
| `requestSizeBuckets` | `internal/api/security.go` | unused var |
| `connectRequest` DTO | `internal/api/devices.go` | folded into the existing `deviceAuthRequest` |

### Device-auth deduplication
`connectHandler` and `heartbeatHandler` each carried a ~27-line inline copy of
the device-secret verification (missing-field check → `FindByID` → metadata
parse → bcrypt `VerifyPassword`) with identical error strings. Both now call
the shared `verifyDeviceSecret` helper used by the other ~24 device endpoints.
Identical status codes and messages; `device` variable still drives
online/offline transitions. Dropped the now-unused `auth` import in
`heartbeat.go`.

### Removed empty directory
`core/internal/agent/` was empty — removed.

---

## Verification (all green)

- `core`: `go build ./...` — OK; `go vet ./...` — OK; `go test ./...` — pass
  (all packages)
- `renctl`: `go build ./...` — OK
- `sdk/go`: `go build ./...` — OK
- `hub`: `npm run typecheck` — OK
- `hub`: `npm run lint` — no errors (pre-existing warnings only, none in
  touched files)
- `hub`: `npm run test` — 82 tests pass across 12 files
- `hub`: `npm run build` — OK (exit 0, no errors)

---

## Assessed but intentionally deferred (remaining debt)

- **Confirm dialogs** — hand-rolled per page and visually heterogeneous
  (device-containers, device-systemd-services, process-manager, docker,
  proxmox, groups, file-browser, …). Converting them to a shared
  `ConfirmDialog` built on `components/ui/dialog.tsx` would change visuals and
  focus/portal behavior; deferred to a dedicated pass.
- **Generic hooks (`useSelection` / `useSort` / `usePagination`)** — no proven
  duplication: selection exists in only 3 files, sorting in 1
  (`process-manager.tsx`), pagination in 0. A generic layer would be speculative.
- **Middleware overlap** — chi `RequestID`, `requestIDMW`, and
  `observability.LogCorrelationMiddleware` all touch `X-Request-ID`; the two
  rate limiters and the triplicated device-auth flows were consolidated where
  provably redundant, but the middleware stack itself is behavior-sensitive and
  was left wired as-is.
- **Rust SDK (`sdk/rust`)** — pre-existing compile errors from crate API drift
  (`http::header::IntoIter` generics, `Builder: IntoClientRequest`, missing
  `send` on `WebSocketStream`). Untouched by this milestone; requires a
  dependency-version alignment pass.
- **`gofmt -l`** — pre-existing unformatted files in `internal/db` (from M67).

## Recommendations
1. Migrate hand-rolled confirm dialogs onto `ui/dialog.tsx` behind a single
   `ConfirmDialog` component in a dedicated UI milestone.
2. Align the Rust SDK's `tokio-tungstenite` / `http` crate versions with its
   API usage.
3. Consider consolidating the three `X-Request-ID` producers into one.
