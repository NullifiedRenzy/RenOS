# RenOS Documentation Review

**Date:** 2026-08-03
**Scope:** `docs/`, `installer/docs/`, `release/docs/`, and root docs
(`README.md`, `RELEASE.md`, `CHANGELOG.md`)
**Method:** Manual inventory + command verification against the actual
repository (Makefile targets, `core/cmd/renos-core`, `renctl/`,
`agent/`, `scripts/`, `installer/backup/`, `installer/validate/`,
`installer/upgrade/`, `installer/wizard/`, `deploy/`).
**Constraint:** Platform/docs only — no code was modified.

---

## Summary

The documentation is broad but drifted from the code in several areas. The
top issues:

1. **Two backup systems are documented interchangeably.**
   `scripts/backup.sh` (legacy, `./backups/`, keeps 30) vs
   `installer/backup/renos-backup.sh` (`/var/lib/renos/backups`, retention 7,
   encryption/remote/verify). Docs disagree about which is canonical and
   about backup location.
2. **The `renos-control` binary does not exist.** Three release docs reference
   it; the real binary is `renos-core`.
3. **`make migrate` and the Makefile are out of sync.** The target runs
   `go run ./cmd/renos-core --migrate-only`, but the binary only accepts
   `migrate` (it would start the server instead of migrating).
4. **Non-existent env vars are documented.** `RENOS_AGENT_HEARTBEAT_INTERVAL`
   and `RENOS_AGENT_OFFLINE_TIMEOUT` are not configurable (hardcoded in Core;
   the agent uses `RENOS_HEARTBEAT_INTERVAL`).
5. **`deploy/nginx/ssl/` does not exist.** The Nginx config uses
   `/etc/nginx/ssl/`.
6. **Several "quick start" docs contradict each other**, and the two supposed
   master indexes (`docs/README.md`, `docs/INDEX.md`) are near-empty stubs.
7. **Features that don't exist are not flagged.** MFA/2FA and SMTP/email
   notifications are absent; docs (and the new production checklist) should
   say so explicitly.
8. **Missing coverage of newly shipped features:** one-command install,
   fresh-install validation, first-run wizard/onboarding, demo mode,
   example library, opt-in telemetry, crash reporter, health dashboard.

---

## 1. Inventory & status

Legend: **OK** = verified accurate · **FIX** = inaccurate, fix required ·
**STUB** = near-empty · **STALE** = out of date · **NR** = not reviewed in
this pass (large/design docs).

### Root

| File | Status | Notes |
|---|---|---|
| `README.md` | FIX | Wrong binary path (`./renos-core`), wrong env vars, legacy backup only. Fixed 2026-08-03. |
| `CHANGELOG.md` | FIX | No section for the production-readiness milestone. New top section added 2026-08-03. |
| `RELEASE.md` | OK | Process doc; no command drift found. |
| `PRODUCTION_CHECKLIST.md` | OK | Created 2026-08-03 (Task 1 deliverable). |

### docs/ — top level

| File | Status | Notes |
|---|---|---|
| `README.md` | STUB | 3 lines; should be the docs index. |
| `INDEX.md` | STUB | 2 lines; duplicates the README stub. |
| `index.html` | STALE | 93 KB generated page; references `./renos-core --config config.yaml` and `deploy/nginx/ssl/`; no `--config` flag exists. |
| `ENGINEERING_SPEC.md` | NR | Design spec. |
| `PROJECT_VISION.md` | NR | Vision doc. |
| `ROADMAP.md` | NR | Roadmap. |
| `reliability-report.md` | NR | Contains the only "demo" mention found in docs. |
| `observability.md` | FIX | `./renos-core` invocation paths; otherwise strong (endpoints/tracing verified accurate). |
| `sre-guide.md` | OK | Endpoints verified; runbooks valid. |

### docs/development/

| File | Status | Notes |
|---|---|---|
| `QUICK_START.md` | FIX | Made canonical; fixed agent binary path, enroll-token TTL claim, dev/prod mix. |
| `getting-started.md` | FIX | Wrong Make targets (`make up/down/ps/reset/status/dev`), wrong engineering-spec path. Now points to QUICK_START. |
| `dev-scripts.md` | FIX | `renos-agent enroll "$TOKEN"` — the CLI requires `--token`. |
| `PHASES.md` | NR | Dev phases. |

### docs/operations/

| File | Status | Notes |
|---|---|---|
| `OPERATIONS.md` | FIX | Legacy backup only; CA path `data/ca/` wrong (`RENOS_CA_CERT_DIR` = `/var/lib/renos/ca`); no pointer to `renos-backup.sh`. |
| `DEPLOYMENT.md` | FIX | `deploy/nginx/ssl/` wrong; legacy cron backup only. |

### docs/api/, docs/standards/, docs/upgrade/, docs/architecture/, docs/capabilities/, docs/opencode/

| File | Status | Notes |
|---|---|---|
| `api/USAGE.md` | OK | Endpoints verified against `router.go`; `ttl_minutes` body is ignored (TTL fixed at 15 min) — minor. |
| `standards/SECURITY.md` | OK | Vault/AES-256-GCM, CSP etc. verified. No MFA claim. |
| `upgrade/v0.1.0-to-v0.2.0.md`, `v0.2.0-to-v0.3.0.md`, `v0.3.0-to-v1.0.0.md`, `index.md` | FIX | `RENOS_AGENT_HEARTBEAT_INTERVAL` / `RENOS_AGENT_OFFLINE_TIMEOUT` listed as core env vars — not configurable. |
| `architecture/ADR-001-…`, `SYSTEM_ARCHITECTURE.md` | NR | Design docs. |
| `capabilities/AI.md`, `DEVICE_MANAGEMENT.md`, `NETWORKING.md`, `VIRTUALIZATION.md` | NR | Design docs. |
| `standards/CODING_STANDARDS.md` | NR | Standards doc. |
| `opencode/CONTINUE_PROMPT.md`, `START_HERE.md` | NR | Agent-internal docs. |

### installer/docs/

| File | Status | Notes |
|---|---|---|
| `README.md` | FIX | Has its own quick start + "wizard" mention; overlaps QUICK_START. |
| `installation-guide.md` | FIX | Long guide; verify backup path claims; reference `renos-backup.sh`. |
| `disaster-recovery.md` | FIX | `renos-backup.sh --config` — flag does not exist (valid: `--full --db-only --files-only --list --verify --prune --encrypt --remote`). |
| `recovery-guide.md` | FIX | Overlaps disaster-recovery.md. |
| `upgrade-guide.md` | NR | Overlaps `docs/upgrade/*`. |
| `air-gapped-deployment.md` | NR | |
| `cluster-deployment.md` | FIX | Uses `renos-cluster-backup.sh --config` — verify against the shipped script. |
| `kubernetes-deployment.md` | NR | |

### release/docs/

| File | Status | Notes |
|---|---|---|
| `release-notes.md` | FIX | `docker run --name renos-control` → `renos-core`. |
| `RELEASE_READY.md` | FIX | Build commands reference `cmd/renos-control`, `cmd/renos-agent`, `cmd/renctl` — none exist at those paths. |
| `known-issues.md` | FIX | `renos-control` → `renos-core`. |
| `migration-guide-v1.0.0.md` | FIX | Multiple `renos-control` references + `renos-control_1.0.0_amd64.deb` package name. |
| `compatibility-matrix.md`, `support-matrix.md` | NR | |

---

## 2. Command verification

Every "documented command" below was checked against the repository.

| # | Location | Documented command | Correct command / action | Status |
|---|---|---|---|---|
| 1 | `README.md:302` | `./renos-core version` | `cd core && go run ./cmd/renos-core/ version` or `./bin/renos-core version` (after `make build-core`) | FIXED |
| 2 | `README.md:147-148` | `RENOS_AGENT_HEARTBEAT_INTERVAL` / `RENOS_AGENT_OFFLINE_TIMEOUT` | Not env-configurable in Core (hardcoded 30s/90s). Agent uses `RENOS_HEARTBEAT_INTERVAL`. | FIXED (rows removed) |
| 3 | `docs/development/QUICK_START.md:35,38` | `./renos-agent enroll --token …`, `./renos-agent run` | Binary lives at `agent/target/release/renos-agent`. `run` is the default subcommand. | FIXED |
| 4 | `docs/development/QUICK_START.md:43-46` + `api/USAGE.md:84` | enroll-token body `{"ttl_minutes": 15}` | Body is ignored; TTL is fixed at 15 min (not configurable). | FIXED (QUICK_START) |
| 5 | `docs/development/dev-scripts.md:48` | `renos-agent enroll "$ENROLL_TOKEN"` | CLI requires a flag: `renos-agent enroll --token "$ENROLL_TOKEN"` | FIXED |
| 6 | `docs/development/getting-started.md:124-198` | `make up` `make down` `make ps` `make reset` `make status` `make dev` | Only `docker-up`/`docker-down`/`docker-logs` exist; use `docker compose …` | FIXED |
| 7 | `docs/development/getting-started.md:156` | `../engineering-specification.txt` | File is `../ENGINEERING_SPEC.md` | FIXED |
| 8 | `docs/observability.md:159,164` | `RENOS_… ./renos-core` | `cd core && go run ./cmd/renos-core/` (no root binary) | DEFERRED |
| 9 | `docs/index.html:301,1169` | `./renos-core --config config.yaml` | No `--config` flag exists (env-only configuration) | DEFERRED |
| 10 | `docs/operations/DEPLOYMENT.md:27-33` | `deploy/nginx/ssl/fullchain.pem` etc. | Directory does not exist; `deploy/nginx/nginx.conf` references `/etc/nginx/ssl/` | DEFERRED |
| 11 | `docs/operations/OPERATIONS.md:41` | CA at `data/ca/` | `RENOS_CA_CERT_DIR`, default `/var/lib/renos/ca` | DEFERRED |
| 12 | `docs/operations/OPERATIONS.md:3-30`, `DEPLOYMENT.md:42-46` | `./scripts/backup.sh` as the backup story | Supported production tool is `installer/backup/renos-backup.sh`; `scripts/backup.sh` is legacy | DEFERRED |
| 13 | `docs/upgrade/*.md` | `RENOS_AGENT_HEARTBEAT_INTERVAL` / `RENOS_AGENT_OFFLINE_TIMEOUT` | Same as #2 | DEFERRED |
| 14 | `release/docs/release-notes.md:77` | `docker run --name renos-control` | Container/image is `renos-core` | DEFERRED |
| 15 | `release/docs/known-issues.md:60` | `renos-control` | `renos-core` | DEFERRED |
| 16 | `release/docs/migration-guide-v1.0.0.md` | `renos-control migrate`, `renos-control_1.0.0_amd64.deb` | `renos-core` | DEFERRED |
| 17 | `release/docs/RELEASE_READY.md` | `go build ./cmd/renos-control`, `./cmd/renos-agent`, `./cmd/renctl` | Core: `cd core && go build ./cmd/renos-core`; Agent: `cd agent && cargo build --release`; CLI: `cd renctl && go build` | DEFERRED |
| 18 | `Makefile:102` | `make migrate` → `go run ./cmd/renos-core --migrate-only` | Binary accepts `migrate` only; `--migrate-only` is ignored (server would start). Use `cd core && go run ./cmd/renos-core/ migrate` | FLAGGED (Makefile not modified) |
| 19 | `installer/docs/disaster-recovery.md:140` | `renos-backup.sh --config` | Flag does not exist (`--full --db-only --files-only --list --verify --prune --encrypt --remote`) | DEFERRED |
| 20 | `PRODUCTION_CHECKLIST.md` (this pass) | `./renctl …` | The prebuilt binary is `renctl/renctl`; `./renctl` is a directory | FIXED |

### Verified correct (spot checks)

- `./install.sh`, `installer/docker/run.sh`, `installer/linux/install.sh`
  (`--uninstall`, `--dry-run`, `NONINTERACTIVE`), `one-command-install.sh`
- `installer/validate/renos-validate.sh --pre|--post|--json|--report|--quick|--verbose`
- `installer/backup/renos-backup.sh` flags and `renos-restore.sh
  --dry-run|--partial database,config|--to-dir`
- `cd core && go run ./cmd/renos-core/ migrate` and `migrate down [N]`
- `renctl/renctl` (prebuilt CLI binary — note: the binary is `renctl/renctl`,
  not `./renctl`, which is a directory): `login --server|--email|--password`,
  `devices`, `jobs`, `workflows`, `plugins`, `cluster`, `vault`, `users`,
  `roles`, `alerts`, `metrics`, `terminal`, `docker`, `proxmox`
- `docker compose exec core renos-core migrate` (in-container binary is
  `renos-core`, per `docker-compose.prod.yml`)
- `/api/v1/healthz`, `/api/v1/readyz`, `/api/v1/version`,
  `/api/v1/health/dashboard`, `/api/v1/metrics` (auth) and `/metrics`
  (Prometheus scrape)
- `make build`, `make test`, `make lint`, `make check`, `make help`,
  `make docker-up`, `make docker-down`, `make docker-logs`

---

## 3. Duplication map

| Topic | Files | Recommendation |
|---|---|---|
| Quick start / getting started | `README.md`, `docs/development/QUICK_START.md`, `docs/development/getting-started.md`, `installer/docs/README.md`, `installer/docs/installation-guide.md` | `QUICK_START.md` is canonical; README links to it; `getting-started.md` now points to it; installer docs link out. |
| Backups | `scripts/backup.sh`/`restore.sh` (legacy), `installer/backup/renos-backup.sh`/`renos-restore.sh`/`renos-backup-config.sh`; documented in `README.md`, `OPERATIONS.md`, `DEPLOYMENT.md`, `installation-guide.md`, `disaster-recovery.md`, `recovery-guide.md` | Make `renos-backup.sh` canonical; mark `scripts/` legacy; single backup section in README. |
| Disaster recovery | `installer/docs/disaster-recovery.md`, `installer/docs/recovery-guide.md` | Merge into one; keep `recovery-guide.md` as a quick reference. |
| Monitoring / ops | `docs/operations/OPERATIONS.md`, `docs/operations/DEPLOYMENT.md`, `docs/sre-guide.md`, `docs/observability.md` | OPERATIONS = runbook, DEPLOYMENT = install, observability = internals, sre-guide = SLO/runbooks. Add cross-links. |
| Docs index | `docs/README.md` (3 lines), `docs/INDEX.md` (2 lines), `docs/index.html` (93 KB stale HTML) | Make `docs/README.md` the single index; delete or regenerate `index.html`. |
| Upgrades | `docs/upgrade/*.md`, `installer/docs/upgrade-guide.md`, `release/docs/migration-guide-v1.0.0.md` | One "upgrade" narrative + one migration guide; the rest link to it. |
| Release history | `CHANGELOG.md`, `release/docs/release-notes.md`, `RELEASE.md` | Keep CHANGELOG as source of truth; release-notes summarize it; RELEASE.md is the process. |

---

## 4. Accuracy & security observations (verified)

- **MFA / 2FA:** not implemented anywhere (Core or hub). No TOTP/OTP code.
  Documentation should state "not yet available" and recommend mitigations
  (strong passwords, RBAC, rate limiting, network restriction).
- **Email / SMTP:** no SMTP configuration exists. Alert delivery is
  webhook-only. Docs should not imply email alerts.
- **Telemetry:** opt-in, anonymous, off by default
  (`RENOS_TELEMETRY_ENABLED=false`, endpoint `RENOS_TELEMETRY_ENDPOINT`).
  Payload excludes names, IPs, secrets.
- **Crash reporter:** `core/internal/crashreport` writes sanitized reports to
  `RENOS_CRASH_DIR` (default `/var/lib/renos/crashes`).
- **Demo mode:** `POST /api/v1/demo/seed` (admin only). **There is no
  `GET /api/v1/demo/status` endpoint** — onboarding status lives at
  `GET /api/v1/onboarding/status`.
- **Health dashboard:** `GET /api/v1/health/dashboard` (auth: viewer+),
  weighted score 0–100 over DB, Redis, NATS, certs, backups, CPU, memory,
  disk.
- **Public (unauthenticated) endpoints to be aware of:** `/api/v1/healthz`,
  `/api/v1/readyz`, `/api/v1/version`, `/api/v1/stats*`, `/api/v1/slos`,
  `/api/v1/deephealth`, `/metrics` (Prometheus), `/livez`, `/startupz`,
  `/api/openapi.*`, `/api/docs`. **Notable:** `/api/v1/stats/env` and
  `/api/v1/debug/config` are also public and summarize env/config — consider
  restricting to admin (flagged for security review, no code changed).
- **Env var drift:** `RENOS_ENROLL_TOKEN_TTL`, `RENOS_CA_CERT_TTL`,
  `RENOS_AGENT_HEARTBEAT_INTERVAL`, `RENOS_AGENT_OFFLINE_TIMEOUT` do **not**
  exist as config; their values are hardcoded (15 min, 365 days, 30 s, 90 s).

---

## 5. FIX LIST

### Done in this pass (2026-08-03)

| # | Item | Location |
|---|---|---|
| F1 | Add production-readiness changelog section (v1.0.0-production) | `CHANGELOG.md` |
| F2 | Make `QUICK_START.md` canonical; fix agent path, enroll-token TTL, dev/prod split | `docs/development/QUICK_START.md` |
| F3 | Point `getting-started.md` to QUICK_START; fix Make targets + engineering-spec link | `docs/development/getting-started.md` |
| F4 | Fix `enroll --token` invocation | `docs/development/dev-scripts.md` |
| F5 | Fix README quick start, `./renos-core` path, env var table, backups | `README.md` |
| F6 | Create production checklist | `PRODUCTION_CHECKLIST.md` |
| F7 | Create this review | `docs/review/DOCUMENTATION_REVIEW.md` |

### Deferred (recommended next)

| # | Item | Priority |
|---|---|---|
| D1 | Replace `renos-control` → `renos-core` in `release/docs/` (4 files) | High |
| D2 | Fix `Makefile` `migrate` target (`--migrate-only` → `migrate`) | High |
| D3 | Fix `deploy/nginx/ssl/` → `/etc/nginx/ssl/` in `DEPLOYMENT.md` + `docs/index.html` | Medium |
| D4 | Replace `./renos-core` / `--config config.yaml` in `observability.md` + `docs/index.html` | Medium |
| D5 | Replace legacy backup docs with `renos-backup.sh` in `OPERATIONS.md`/`DEPLOYMENT.md` | Medium |
| D6 | Remove `RENOS_AGENT_*` env vars from `docs/upgrade/*.md` | Medium |
| D7 | Fix `--config` flag in `installer/docs/disaster-recovery.md`, verify `cluster-deployment.md` | Medium |
| D8 | Consolidate docs index (README.md / INDEX.md / index.html) | Low |
| D9 | Merge disaster-recovery vs recovery-guide | Low |
| D10 | Add MFA/SMTP "not yet available" notes to `standards/SECURITY.md` | Low |
| D11 | Review public `/api/v1/stats/env` and `/api/v1/debug/config` exposure | Security |
