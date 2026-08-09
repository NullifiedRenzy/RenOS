# RenOS 1.0.0 Release Readiness Checklist

This checklist tracks all verification steps required before the v1.0.0 release is considered ready for shipment.

---

## Code Quality

### Build
- [ ] **Control-plane builds** — `go build ./cmd/renos-control` succeeds
- [ ] **Agent builds** — `go build ./cmd/renos-agent` succeeds
- [ ] **CLI builds** — `go build ./cmd/renctl` succeeds
- [ ] **Hub UI builds** — `npm run build` produces optimized output
- [ ] **All SDKs build**
  - [ ] Go SDK: `go build ./...`
  - [ ] TypeScript SDK: `tsc --noEmit`
  - [ ] Python SDK: `poetry build`
  - [ ] C# SDK: `dotnet build`
  - [ ] Rust SDK: `cargo build`
- [ ] **Docker images build** for all platforms (linux/amd64, linux/arm64)
- [ ] **Helm chart lint** — `helm lint charts/renos`
- [ ] **Cross-compilation** verified for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64

### Tests
- [ ] **Unit tests pass** — `go test ./...` (200+ tests)
- [ ] **Integration tests pass** — `make test-integration` (150+ tests)
- [ ] **Fuzz tests** — 10 min each on critical parsers (YAML, JSON, plugin manifest)
- [ ] **E2E tests pass** — `make test-e2e` (75+ tests)
- [ ] **Hub UI tests** — `npm test` (Jest + React Testing Library)
- [ ] **Hub UI E2E** — `npx cypress run` (critical user journeys)

### Lint & Static Analysis
- [ ] **Go vet** — `go vet ./...` passes with zero warnings
- [ ] **Staticcheck** — `staticcheck ./...` passes
- [ ] **golangci-lint** — all linters pass
- [ ] **ESLint** — `npm run lint` passes for Hub UI
- [ ] **Prettier** — `npm run format:check` passes
- [ ] **Python SDK** — `mypy --strict src/` passes
- [ ] **C# SDK** — `dotnet format --verify-no-changes`
- [ ] **Rust SDK** — `cargo clippy -- -D warnings`

### Race Detection
- [ ] **Race detector** — `go test -race ./...` passes (no races)

---

## Security

### Dependency Audits
- [ ] **Go dependencies** — `go list -m all | nancy sleuth` — no CVEs ≥ Medium
- [ ] **NPM dependencies** — `npm audit` — no critical/high vulnerabilities
- [ ] **Python dependencies** — `safety check` — no vulnerabilities
- [ ] **NuGet dependencies** — `dotnet list package --vulnerable` — clean
- [ ] **Rust dependencies** — `cargo audit` — no advisories

### License Compliance
- [ ] **License audit completed** — all dependencies use compatible licenses
- [ ] **NOTICE file** generated with all required attributions

### SBOM
- [ ] **SPDX 2.3 SBOM** generated for all artifacts
- [ ] **CycloneDX SBOM** available for container images

### Artifact Signing
- [ ] **Container images** signed with cosign
- [ ] **Binary artifacts** signed with GPG
- [ ] **Helm chart** signed with GPG
- [ ] **Checksums** (SHA256) published for all download artifacts

### Security Scanning
- [ ] **Container image scan** — Trivy scan reports 0 critical/high CVEs
- [ ] **SAST scan** — Semgrep/CodeQL scan complete, findings reviewed
- [ ] **DAST scan** — API endpoint fuzzing completed, no high-severity findings
- [ ] **Secret scanning** — GitLeaks scan of entire repo — zero secrets detected

### Penetration Testing
- [ ] **External pentest** completed by third-party — all findings remediated
- [ ] **Internal pentest** completed — no high-severity remaining

---

## Testing

### Functional Testing
- [ ] **All API endpoints** tested (100+ endpoints)
- [ ] **Workflow CRUD** — create, read, update, delete, list, export, import
- [ ] **All trigger types** — manual, cron, webhook, event
- [ ] **Plugin lifecycle** — install, configure, enable, disable, uninstall
- [ ] **Agent lifecycle** — register, heartbeat, execute, disconnect, reconnect
- [ ] **Authentication flows** — API key, mTLS, JWT
- [ ] **RBAC** — all roles and permissions verified

### Cross-Platform Testing
- [ ] **Linux (amd64)** — full test suite
- [ ] **Linux (arm64)** — full test suite
- [ ] **macOS (amd64)** — agent + CLI tests
- [ ] **macOS (arm64)** — agent + CLI tests
- [ ] **Windows (amd64)** — agent + CLI + MSI installer tests
- [ ] **Docker (linux/amd64)** — full integration test
- [ ] **Docker (linux/arm64)** — integration test

### Browser Compatibility
- [ ] **Chrome** (latest) — Hub UI full test
- [ ] **Firefox** (latest) — Hub UI full test
- [ ] **Safari** (latest) — Hub UI full test
- [ ] **Edge** (latest) — Hub UI smoke test

### Accessibility
- [ ] **WCAG 2.1 AA** compliance checked (Wave tool)
- [ ] **Keyboard navigation** verified for all Hub UI flows
- [ ] **Screen reader** compatibility verified (NVDA, VoiceOver)

---

## Infrastructure

### CI Pipeline
- [ ] **CI pipeline** green on all branches for the last 7 days
- [ ] **All CI stages** pass: lint → build → unit → integration → e2e → publish
- [ ] **Flaky tests** identified and quarantined (≤ 5 allowed)
- [ ] **CI pipeline** completes under 30 minutes

### Reproducible Builds
- [ ] **Go builds** are reproducible (same source → same binary)
- [ ] **Docker images** are reproducible (same source → same digest)
- [ ] **Build timestamps** stripped from binaries

### Release Automation
- [ ] **Release workflow** tested in dry-run mode
- [ ] **Changelog** generated from commit history
- [ ] **Git tag** created (`v1.0.0`)
- [ ] **GitHub Release** draft ready with all artifacts
- [ ] **Homebrew formula** updated
- [ ] **Docker Hub** images pushed and scanned
- [ ] **Helm chart** published to chart repository

### Artifacts Checklist
- [ ] `renos-control_1.0.0_linux_amd64.deb`
- [ ] `renos-control_1.0.0_linux_amd64.rpm`
- [ ] `renos-control_1.0.0_linux_arm64.deb`
- [ ] `renos-control_1.0.0_linux_arm64.rpm`
- [ ] `renos-agent_1.0.0_linux_amd64.deb`
- [ ] `renos-agent_1.0.0_linux_amd64.rpm`
- [ ] `renos-agent_1.0.0_linux_arm64.deb`
- [ ] `renos-agent_1.0.0_linux_arm64.rpm`
- [ ] `renos-agent_1.0.0_darwin_amd64.pkg`
- [ ] `renos-agent_1.0.0_darwin_arm64.pkg`
- [ ] `renos-agent_1.0.0_windows_amd64.msi`
- [ ] `renctl_1.0.0_linux_amd64.tar.gz`
- [ ] `renctl_1.0.0_linux_arm64.tar.gz`
- [ ] `renctl_1.0.0_darwin_amd64.tar.gz`
- [ ] `renctl_1.0.0_darwin_arm64.tar.gz`
- [ ] `renctl_1.0.0_windows_amd64.zip`
- [ ] `renos-hub_1.0.0.tar.gz` (static assets)
- [ ] `renos_1.0.0_helm.tar.gz` (Helm chart)
- [ ] `renos_1.0.0_sbom.spdx.json` (SBOM)
- [ ] `renos_1.0.0_checksums.txt` (SHA256)
- [ ] `renos_1.0.0_signed.asc` (GPG signature)
- [ ] Docker image: `renos/control-plane:1.0.0`
- [ ] Docker image: `renos/agent:1.0.0`
- [ ] Docker image: `renos/hub:1.0.0`

---

## Documentation

### Release Documentation
- [ ] Release notes written and reviewed (`release-notes.md`)
- [ ] Known issues documented (`known-issues.md`)
- [ ] Migration guide complete (`migration-guide-v1.0.0.md`)
- [ ] Support matrix updated (`support-matrix.md`)
- [ ] Compatibility matrix updated (`compatibility-matrix.md`)
- [ ] API reference updated (OpenAPI spec)
- [ ] Breaking changes clearly called out
- [ ] Upgrade guide tested by following it step-by-step

### User Documentation
- [ ] **Getting Started** guide updated
- [ ] **Installation** guide covers all deployment methods
- [ ] **Configuration** reference complete
- [ ] **CLI reference** documents all commands and flags
- [ ] **SDK docs** generated for all 5 languages
- [ ] **Plugin development** guide updated
- [ ] **API documentation** (Swagger UI) deployed
- [ ] **FAQ** updated with common issues

### Operational Documentation
- [ ] **Production deployment** guide complete
- [ ] **Monitoring & alerting** guide complete
- [ ] **Backup & restore** procedures documented
- [ ] **Troubleshooting** guide updated

---

## Verification

### Backup/Restore
- [ ] **Database backup** (`pg_dump`) works and produces valid dump
- [ ] **Database restore** (`pg_restore`) succeeds and data is consistent
- [ ] **Config backup** procedure documented and tested
- [ ] **Full restore test** — backup, wipe, restore, verify all workflows intact

### Upgrade & Rollback
- [ ] **Upgrade from v0.3.0-rc1** tested on a production-like dataset
- [ ] **Upgrade from v0.3.0-rc1** with 100+ workflows completes in < 5 min
- [ ] **Rollback from v1.0.0 to v0.3.0-rc1** tested and documented
- [ ] **Rollback verification** — all workflows accessible after rollback

### Benchmarks
- [ ] **Workflow execution benchmark** meets performance goals
- [ ] **Throughput benchmark** (workflows/second) meets targets
- [ ] **Latency benchmark** (p50/p95/p99) meets SLOs
- [ ] **Memory benchmark** — no regression from v0.3.0-rc1
- [ ] **Benchmark results** recorded in release notes

### Smoke Tests
- [ ] **Fresh install** from scratch (all deployment methods)
- [ ] **Workflow creation** via API, CLI, and UI
- [ ] **Workflow execution** with real plugin (OpenAI, at minimum)
- [ ] **Agent registration** and task dispatch
- [ ] **Plugin marketplace** browsing and installation
- [ ] **User authentication** and RBAC enforcement
- [ ] **Audit log** entries generated correctly
- [ ] **Metrics endpoint** returns valid Prometheus data
- [ ] **Health checks** return expected status

---

## Sign-Off

| Role | Name | Date | Signature |
|------|------|------|-----------|
| Engineering Lead | | | |
| QA Lead | | | |
| Security Lead | | | |
| Product Manager | | | |
| Docs Lead | | | |
| Release Manager | | | |

---

## Final Go/No-Go

- All checkboxes above are either checked ✅ or explicitly waived with reason.
- No open blocker bugs (P0, P1) remain.
- Known issues are documented and acceptable per product management.
- Release manager confirms go decision.

**Go / No-Go Decision:** _________________ **Date:** _________________
