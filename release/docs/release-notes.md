# RenOS 1.0.0 Release Notes

**Version:** 1.0.0  
**Release Date:** 2026-07-30  
**Download:** https://github.com/renos/renos/releases/tag/v1.0.0  
**Documentation:** https://docs.renos.io/v1.0.0

---

## What's New in RenOS 1.0.0

### Core Platform
- Distributed workflow engine with DAG-based execution and retry policies
- Event-driven architecture with NATS-backed message bus
- Modular pipeline framework supporting custom middleware chains
- Resource-aware scheduler with priority queues and backpressure
- State machine SDK for long-running workflow orchestration
- Pluggable storage backends (PostgreSQL, Redis, NATS JetStream)

### Hub UI
- Redesigned dashboard with real-time pipeline monitoring
- Workflow graph visualizer with live node status
- Plugin marketplace browser with one-click install
- Team management and RBAC configuration UI
- Audit log viewer with search and export

### Agent
- Cross-platform agent runtime (Linux, Windows, macOS)
- Auto-update mechanism with rollback support
- Pluggable capability system (executor, watcher, forwarder)
- mTLS-based control plane communication
- Offline operation mode with local queue buffering

### Plugins & Marketplace
- Plugin SDK v1.0.0 with stable API surface
- Marketplace with 40+ community plugins
- Plugin sandboxing with resource limits
- Versioned plugin manifests with dependency resolution

### CLI (`renctl`)
- Subcommands: `run`, `workflow`, `agent`, `plugin`, `config`, `namespace`, `logs`
- JSON and YAML output formatting with `-o` flag
- Tab-completion for bash, zsh, fish

### SDKs
- **Go SDK v1.0.0** — idiomatic Go client with generics support
- **TypeScript SDK v1.0.0** — full type definitions, ESM + CJS
- **Python SDK v1.0.0** — async/await native, type-annotated
- **C# SDK v1.0.0** — .NET 8.0+, nullable enabled
- **Rust SDK v1.0.0** — async traits, zero-cost abstractions

### Developer Platform
- OpenAPI 3.1 specification with 100+ endpoints
- Swagger UI bundled with control-plane deployment
- Webhook system with retry and signature verification
- API key management with scoped permissions

### Installer Platform
- **Linux:** `.deb`, `.rpm`, `.apk`, static binary tarballs
- **Windows:** MSI installer, portable ZIP
- **macOS:** Homebrew formula, `.pkg` installer
- **Docker:** Multi-arch images (linux/amd64, linux/arm64)
- **Kubernetes:** Helm chart with production defaults

### Observability
- OpenTelemetry tracing (W3C TraceContext, Jaeger, Zipkin exporters)
- Prometheus metrics with pre-built Grafana dashboards
- Structured JSON logging with configurable levels
- Health check endpoints (`/healthz`, `/readyz`)

---

## Installation

### Quick Start (Docker)
```bash
docker run -d --name renos-control \
  -p 8080:8080 -p 4222:4222 \
  renos/control-plane:1.0.0
docker run -d --name renos-agent \
  --link renos-control \
  renos/agent:1.0.0
```

### Kubernetes (Helm)
```bash
helm repo add renos https://charts.renos.io
helm install renos renos/renos \
  --namespace renos --create-namespace \
  --version 1.0.0
```

### Native Installation
```bash
# Linux (deb)
sudo apt-get install ./renos_1.0.0_amd64.deb

# macOS
brew install renos/tap/renos

# Windows
msiexec /i renos-1.0.0-x64.msi
```

See [Installation Guide](https://docs.renos.io/v1.0.0/install) for full details.

---

## Upgrade Notes from v0.3.0-rc1

1. **Back up your database and configuration before upgrading.**
2. The minimum PostgreSQL version is now 15; upgrade PostgreSQL first if needed.
3. Environment variable `RENOS_CONTROL_API_ADDR` is replaced by `RENOS_API_BIND_ADDRESS`.
4. The v0.3.0 plugin API is **not** compatible — all plugins must be rebuilt against Plugin API v1.0.0.
5. Agent protocol v0.1 is no longer supported — agents must update to v1.0.0.
6. The `renctl submit` command is renamed to `renctl workflow create`.
7. Several deprecated API endpoints have been removed (see Breaking Changes below).

See the [Migration Guide](./migration-guide-v1.0.0.md) for step-by-step instructions.

---

## Breaking Changes

### API
- Endpoint `POST /api/v1alpha1/workflows/submit` removed — use `POST /api/v1/workflows`
- Response field `workflow.metadata.created` replaced by `workflow.metadata.created_at`
- Pagination cursor format changed from base64 to opaque string
- Rate-limit headers renamed: `X-RateLimit-Remaining` → `X-RateLimit-Left`

### Configuration
- `RENOS_CONTROL_API_ADDR` → `RENOS_API_BIND_ADDRESS`
- `RENOS_STORE_TYPE` → `RENOS_STORAGE_BACKEND`
- `RENOS_NATS_URL` → `RENOS_NATS_ENDPOINT`
- Redis config keys unified under `RENOS_REDIS_*` prefix

### Plugin API
- `Plugin.Init(ctx)` signature changed to `Plugin.Init(ctx, config)`
- `Plugin.Execute` now receives context with deadline
- Plugin manifest `api_version` must be `v1.0.0`

### CLI
- `renctl submit` → `renctl workflow create`
- `renctl list` → `renctl workflow list`
- `renctl describe` → `renctl workflow get`
- `renctl logs` now requires `--follow` for tailing (was default)

### SDKs
- All SDKs: `Client.SubmitWorkflow()` → `Client.CreateWorkflow()`
- Go SDK: `NewClient()` now requires `opts ...ClientOption`
- TS SDK: client constructor uses named params object
- Python SDK: `async with Client(...)` context manager required
- C# SDK: methods return `Task<T>` consistently
- Rust SDK: `Client::builder()` pattern replaces `Client::new()`

---

## New Features by Component

| Component | Feature |
|-----------|---------|
| Core | DAG-based workflow engine with conditional branching |
| Core | Event-driven triggers (cron, webhook, schedule) |
| Core | Workflow versioning and draft/publish lifecycle |
| Hub | Real-time pipeline status via WebSocket |
| Hub | Plugin marketplace with search and categories |
| Hub | Team management with RBAC |
| Agent | Auto-update with staged rollouts |
| Agent | Offline queue with replay on reconnect |
| Agent | Custom capability plugins |
| CLI | JSON/YAML output formatting |
| CLI | Tab completion for all shells |
| SDKs | Full generated clients for 5 languages |
| SDKs | Retry and circuit-breaker built-in |
| Installer | Multi-arch Docker images |
| Installer | Production-grade Helm chart |
| Observability | OpenTelemetry distributed tracing |
| Observability | 20+ Prometheus metrics |
| Observability | Grafana dashboard pack |

---

## Bug Fixes Since RC1

- Fixed WebSocket connection dropping under high throughput (#2841)
- Resolved race condition in workflow state transition (#2912)
- Fixed agent registration timeout when NATS is under load (#2945)
- Corrected pagination cursor decoding for large datasets (#2978)
- Fixed plugin hot-reload crash on manifest parse error (#3001)
- Resolved memory leak in long-running workflow watcher (#3015)
- Fixed CLI `workflow get` output truncation (#3033)
- Corrected Homebrew formula dependency resolution (#3050)
- Fixed Windows agent service shutdown hang (#3062)
- Resolved Docker Compose health check race (#3078)
- Fixed Helm chart ingress TLS secret mount path (#3090)
- Corrected SDK timestamp serialization across timezones (#3105)

---

## Performance Improvements

| Benchmark | v0.3.0-rc1 | v1.0.0 | Improvement |
|-----------|-----------|--------|-------------|
| Workflow execution (1K nodes, noop) | 4.2s | 1.8s | 57% faster |
| Throughput (concurrent workflows) | 250/s | 840/s | 3.36x |
| Agent registration (100 agents) | 12.3s | 2.1s | 5.86x |
| API latency p99 (list workflows) | 340ms | 45ms | 87% lower |
| Plugin execution overhead | 15ms | 2ms | 87% lower |
| Memory — idle control-plane | 128MB | 72MB | 44% reduction |
| Memory — per 100 workflows | 64MB | 28MB | 56% reduction |
| Database — workflow insert (100/s) | 850ms p99 | 120ms p99 | 86% lower |

---

## Security Improvements

- All artifacts signed with Sigstore cosign
- SBOM (SPDX 2.3) generated for every release
- Container images now run as non-root by default
- mTLS enforced between control-plane and agents
- API keys hashed with bcrypt at rest
- Dependency audit: all CVEs with severity >= Medium remediated
- Plugin sandboxing with seccomp and AppArmor profiles
- Rate limiting on authentication endpoints
- Secrets never logged in plaintext

---

## Known Issues

See [Known Issues](./known-issues.md) for the full list of KI-001 through KI-008.

---

## Contributors

We thank the following contributors for their work on this release:

Aditya Sharma, Aisha Patel, Carlos Mendez, Chen Wei, Dmitry Volkov, Elena Petrova, Emily Zhang, Hiroshi Tanaka, James Wilson, Jonas Lindberg, Juan Carlos Reyes, Kehinde Ogunleye, Lucie Bernard, Maria Santos, Mateusz Kowalski, Nikhil Rao, Omar Hassan, Priya Singh, Rajesh Kumar, Sarah Mitchell, Thomas Mueller, Yuki Tanaka, Zara Ahmed

---

## Statistics

| Metric | Value |
|--------|-------|
| Go source lines | ~200,000 |
| TypeScript source lines | ~50,000 |
| Rust source lines | ~15,000 |
| API endpoints | 100+ |
| Unit tests | 200+ |
| Integration tests | 150+ |
| E2E tests | 75+ |
| Plugins (marketplace) | 40+ |
| Docker image layers | 2 (distroless) |
| Docker image size | ~45MB (control-plane) |

---

*For questions or support, visit [https://community.renos.io](https://community.renos.io) or open an issue at [https://github.com/renos/renos/issues](https://github.com/renos/renos/issues).*
