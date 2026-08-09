# Changelog

## v1.0.0-production (2026-08-03)

### Production Readiness Milestone

#### One-command Install
- `installer/linux/one-command-install.sh` — zero-knowledge install for
  Ubuntu 24.04/22.04, Debian 13, Fedora 42 (`curl -sSL https://renos.local/install.sh | sudo bash`)
- `installer/linux/install.sh` — full systemd-based production install with
  rollback, `--dry-run`, `--uninstall`, `NONINTERACTIVE` mode
- `installer/docker/run.sh` — Docker deployment launcher (`--dev`, `--build`,
  `--pull`, `--update`, `--uninstall`)

#### Fresh-Install Validation
- `installer/linux/fresh-install-test.sh` — end-to-end validation on a blank
  VM (install + verify services, API, auth, DB/Redis/NATS, WebSocket, TLS),
  PASS/FAIL/WARN report to `/tmp/renos-install-results.txt`
- `installer/validate/renos-validate.sh` — pre/post install checks with
  `--json`/`--report` output and exit-code based gating

#### First-Run Wizard
- Hub onboarding flow backed by `GET /api/v1/onboarding/status` and
  `POST /api/v1/onboarding/complete`
- `installer/wizard/wizard` — interactive configuration generator
  (`--config`, `--output`, `--dry-run`)

#### Demo Mode
- `POST /api/v1/demo/seed` (admin) — seeds a realistic demo dataset
  (`core/internal/api/demo.go`)

#### Example Library
- `examples/library/` with 8 scenario packs (docker-host, enterprise,
  gaming-server, home-lab, kubernetes-cluster, proxmox-cluster,
  raspberry-pi-fleet, small-business) and `import.sh` one-click importer
  (`--all`, `--dry-run`, env-driven auth)

#### UI Polish, Accessibility & Error UX
- Dashboard polish and accessibility pass
- Improved error surfaces across the hub and API error responses

#### Opt-in Telemetry
- Anonymous usage telemetry, off by default
  (`RENOS_TELEMETRY_ENABLED`, `RENOS_TELEMETRY_ENDPOINT`,
  `RENOS_TELEMETRY_INSTANCE_ID`) — version, OS/arch, random instance id,
  resource counts; no names, IPs, or secrets
- `core/internal/telemetry/`

#### Crash Reporter
- Captures panics with sanitized environment (secrets redacted) to
  `RENOS_CRASH_DIR` (`core/internal/crashreport/`)

#### Health Dashboard
- `GET /api/v1/health/dashboard` (viewer+) — weighted 0-100 health score
  over DB, Redis, NATS, certificates, backups, CPU, memory, and disk
  (`core/internal/api/health_dashboard.go`)

#### Production Checklist
- `PRODUCTION_CHECKLIST.md` — operator checklist: install/validation,
  accounts, secrets, TLS, firewall, backups, DR, monitoring, alerting,
  logging, rate limiting, resource limits, AI, updates, with a
  "Score your readiness" rubric and verified commands

## v1.0.0-rc2 (2026-07-29)

### M56: Reliability & Chaos Engineering

#### Chaos Testing
- Service kill/recovery: PostgreSQL, Redis, NATS, Core, Hub
- Network partition simulation via iptables
- Automated recovery verification via `/readyz` endpoint

#### Recovery Validation
- NATS reconnect handlers added (`DisconnectErrHandler`, `ReconnectHandler`, `ClosedHandler`, `ErrorHandler`)
- Background health monitor goroutine (`StartHealthMonitor`) for connection flapping detection
- Exponential backoff verified (30s base, 10m max, 2^retry factor)
- Retryable vs non-retryable error classification (8 retryable, 6 non-retryable)

#### Stress & Load Testing
- `scripts/stress_test.sh` — API burst (500 req), device registration (50), workflows (20), terminals (5)
- `scripts/chaos_test.sh` — Chaos engineering suite with pass/fail tracking
- Go benchmarks for all hot paths (writeJSON, writeError, sanitize, rate limiter)

#### Concurrency Audit
- Zero data races after fixes (3 races found and eliminated)
- `Hub.ClientCount()` added — channel-based safe client count query
- Rate limiter test avoids cleanup goroutine race

#### Memory Safety
- All unbounded maps bounded: rate limiter (10k), enrollment tokens (1k), circuit breaker, device rate limiter
- Periodic cleanup goroutines for circuit breaker and device rate limiter (5-min intervals)

#### Tests Added
- `internal/bus/nats_test.go` — NATS reconnect, queue subscribe, slow consumer (4 integration tests)
- `internal/api/reliability_test.go` — Concurrent safety, context cancellation, backoff (10 unit tests)
- `internal/api/benchmark_test.go` — writeJSON: 4,710 ns/op, sanitize: 127 ns/op (0 allocs), rate limiter: 441 ns/op (7 benchmarks)

### Security Hardening (S1-S24)

#### Critical (S1-S10)
- **S1:** `.gitignore` hardened, `.env.example` secrets cleared, `install.sh` password validation, production secret validation in `validate.go`, pre-commit hook for secret scanning
- **S2:** Google API key moved from URL query param to `x-goog-api-key` header (Chat, StreamChat, ListModels)
- **S3:** Production validation blocks empty/short JWT secret, encryption key, and default admin password
- **S4:** Deploy endpoints (`/status`, `/stream`, `/start`, `/rollback`) now require admin JWT auth
- **S5:** SQL injection fixed in webhooks — uses `json.Marshal` instead of string concatenation
- **S6:** Context bugs fixed — `workflows.go` and `deploy.go` use `context.Background()`
- **S7:** Vault encryption — new `enc` package (AES-256-GCM with SHA-256 key derivation), vault handlers encrypt before storage
- **S8:** `/metrics` and `/ai/settings` endpoints now require auth + role
- **S9:** CSP hardened — removed `unsafe-eval` and `unsafe-inline` from `script-src`
- **S10:** Token generation fixed — `terminal.go` and `iam.go` now use `crypto/rand`

#### High (S11-S24)
- **S11:** SQL parameter mismatch fixed in terminal sessions query
- **S12:** Path traversal blocked in agent `file_transfers.rs` via `validate_remote_path()`
- **S13:** Terminal session token hidden from JSON serialization (`json:"-"`)
- **S14:** Rate limiter map bounded (max 10,000) with LRU eviction of oldest entry
- **S15:** Method override restricted to safe methods only (GET, HEAD, OPTIONS)
- **S16:** Enrollment token map bounded (max 1,000) with expired token cleanup on generation
- **S17:** File transfer `remote_path` validated — blocks `..` and null bytes
- **S18:** WebSocket origin tightened — requires `localhost:PORT` format, wildcard removed
- **S19:** CSP `script-src` hardened — removed `'unsafe-inline'`, now only `'self'`
- **S20:** Error messages sanitized — `sanitizeErrorMessage()` strips internal paths, DB errors, connection info
- **S21:** CSRF protection — `csrfProtectionMW` requires Origin, X-Requested-With, or Authorization on state-changing requests; exempts agent endpoints
- **S22:** Command policy validation — `isBlockedCommand()` blocks destructive commands (`rm -rf /`, `mkfs`, fork bombs)
- **S23:** Pagination limits enforced — `limit` capped at 200, `offset` capped at 10,000
- **S24:** Stale file transfer cleanup — `FileTransferReaper` goroutine runs every 5 minutes

### Performance (P1-P8)

- **P1:** N+1 query eliminated — `isCancelled()` now called once per batch instead of per device
- **P2:** Fleet dashboard rewritten — replaced 5 unbounded `FindAll()` queries with efficient COUNT/aggregate queries; eliminated O(N*M) correlated subquery
- **P3:** `hasRole()` map allocation fixed — moved package-level `var` to avoid per-call allocation in hot WS broadcast path
- **P4:** Sequential COUNT queries combined — `metricsHandler` and `jobStatsHandler` use single `CountByStatuses()` with `GROUP BY` instead of 5-7 individual `SELECT COUNT(*)` calls
- **P5:** Unbounded maps eviction — `CircuitBreaker` and `DeviceRateLimiter` now have periodic cleanup goroutines (5-min intervals)
- **P6:** `ContainerRepository.ReplaceForDevice` — uses `pgx.Batch` for INSERTs instead of individual round-trips
- **P7:** Scheduler goroutine safety — `ListEnabledDue` paginated with `LIMIT 50`, executor goroutines capped with semaphore (max 20)
- **P8:** Fleet dashboard now returns `int64` for all numeric fields — consistent type safety

### Changed
- **CSRF:** All POST/PUT/PATCH/DELETE requests now require `Origin`, `X-Requested-With`, or `Authorization` header (agent endpoints exempt)
- **CSP:** `script-src` no longer allows `'unsafe-inline'` — inline scripts blocked
- **Method Override:** `X-HTTP-Method-Override` only allows GET/HEAD/OPTIONS (was unrestricted)
- **WebSocket:** Origin must be `localhost:PORT` — `.renos.local` wildcard removed

## v1.0.0-rc1 (2026-07-28)

### Release Candidate 1

This is the first release candidate for RenOS v1.0, bringing together all major
subsystems into a cohesive platform for device management, orchestration, and
operations.

### Added

#### Core Platform (M34-M38)
- Process management with live monitoring and control
- Docker container management (list, start, stop, restart, logs, stats)
- Proxmox VE integration (nodes, VMs, LXCs, storage, networks, snapshots)
- Terminal/PTY sessions with recording and playback
- Plugin platform with routes, pages, sidebar, workers, schedules

#### AI & Intelligence (M40)
- RenAI provider interface (Ollama, OpenAI, Anthropic, Google)
- Context engine for device-aware conversations
- Prompt framework with templates
- Conversation engine with streaming support
- Hub AI sidebar and settings page

#### Fleet Management (M41)
- Device tags with search and filtering
- Hardware/software inventory collection
- Compliance checks and results tracking
- Bulk operations (command, update, reboot)
- Fleet dashboard with OS distribution analytics

#### Automation (M42)
- Workflow engine with steps, runs, and templates
- Trigger types: manual, schedule, event, webhook
- Step types: command, http, delay, condition, device_action
- Workflow templates for common operations

#### Monitoring (M43)
- Metrics collection and querying
- Alert rules with conditions and thresholds
- Alert instances with acknowledgment and resolution
- Monitoring dashboards with configurable widgets
- Log aggregation with level filtering and search

#### Security (M44-M47)
- RenVault: secrets management with versioning, keys, policies, grants
- Notification system with providers (email, Slack, webhook, PagerDuty)
- Marketplace for plugin discovery and management
- Identity & Access Management:
  - OIDC/OAuth2/LDAP identity providers
  - OAuth2 application registration
  - API token management
  - RBAC roles and assignments
  - Approval workflows with multi-step approvals
  - IAM audit log

#### Deployment (M48)
- Preflight checks (Docker, CPU, RAM, disk, kernel, firewall, networking)
- Automated installer with config generation
- TLS certificate generation
- Docker Compose template generation
- Installation rollback support

#### Updates (M49)
- Version detection and update planning
- Release channels: stable, beta, nightly
- Backup before update with rollback
- Update history tracking

#### Clustering (M50)
- Leader election with Raft-like protocol
- Heartbeat monitoring
- Distributed task assignment
- Cluster state management (key-value with TTL)
- Node metrics collection

#### Mobile Platform (M51)
- Push notification registration (FCM, APNs, WebPush)
- Mobile session management with biometric support
- Offline sync queue for offline-first operations
- Device approval workflow
- Mobile-optimized dashboard summaries

#### Developer Platform (M52)
- Webhook subscriptions with event filtering
- Webhook delivery audit log
- Developer API keys with scoped permissions
- API documentation page

#### Quality (M53)
- Go test suite (34+ tests across 5 packages)
- GitHub Actions CI with security scanning and coverage
- golangci-lint configuration with 50+ linters
- Makefile for common development tasks

#### Security & Performance (M54)
- Security headers (CSP, HSTS, X-Frame-Options, etc.)
- Enhanced CORS with configurable origins
- Slow request logging with configurable threshold
- Connection tracking (current, peak, total)
- Request timing headers
- Method override support (X-HTTP-Method-Override)
- Server stats endpoints (/stats, /stats/memory, /stats/gc)
- Debug endpoints (/debug/config, /debug/request)

### Architecture
- **Backend:** Go 1.25, PostgreSQL 16, Redis 7, NATS 2.10
- **Agent:** Rust 1.88 with mTLS communication
- **Hub:** Next.js 15 with Tailwind CSS
- **Protocol:** gRPC with Protocol Buffers

### Database
- 48 migrations covering all subsystems
- Foreign key constraints with ON DELETE CASCADE
- Proper indexing for query performance
- JSONB for flexible metadata storage

### API
- RESTful API with consistent JSON envelope (`{"data": ...}`)
- JWT authentication with role-based access control
- Rate limiting per endpoint category
- Request/response logging with request IDs
- WebSocket support for real-time updates
