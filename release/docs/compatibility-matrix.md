# RenOS 1.0.0 Compatibility Matrix

This document details API, protocol, SDK, and artifact compatibility for RenOS 1.0.0.

---

## API Compatibility

### API Versions

| Version | Path Prefix | Status | Stability |
|---------|-------------|--------|-----------|
| v1 | `/api/v1` | Stable | Backward-compatible within v1 |
| v1alpha1 | `/api/v1alpha1` | Experimental | May change without notice |

All stable API endpoints are accessible under `/api/v1`. The `v1alpha1` prefix is deprecated in v1.0.0 and will be removed in v2.0.0.

### Breaking Change Policy (v1 Stable API)

Breaking changes to the stable v1 API will:
1. Be announced at least 2 minor versions in advance
2. Be documented in a dedicated deprecation notice
3. Coexist with the old behavior for at least one minor release cycle

Non-breaking changes include:
- Adding new endpoints
- Adding optional fields to request/response bodies
- Adding new response headers
- Extending enumerations with new values

Breaking changes include:
- Removing or renaming endpoints
- Removing or renaming fields
- Making optional fields required
- Changing response types or status codes
- Changing pagination format

---

## Plugin API Compatibility

| Plugin API Version | RenOS Version | Compatible |
|--------------------|---------------|------------|
| v1alpha1 | 0.3.0-rc1 | ❌ Not compatible |
| v1.0.0 | 1.0.0 | ✅ Full |

### Plugin Manifest Fields

```yaml
# Plugin API v1.0.0 manifest (required fields)
api_version: "v1.0.0"        # Required — must match exactly
name: "my-plugin"             # Required — alphanumeric + hyphens
version: "1.0.0"              # Required — semver
capabilities: ["executor"]    # Required — minimum one
```

### Plugin Runtime Contract

Plugin API v1.0.0 guarantees:
- The `Init(ctx, config)` signature is stable
- The `Execute(ctx, input)` signature is stable
- Context carries a deadline for all executions
- Config is a validated `map[string]any`
- Plugin manifest validation is strict

---

## Agent Protocol Compatibility

| Protocol Version | RenOS Version | Compatible |
|-----------------|---------------|------------|
| AgentProtocol/0.1 | 0.3.0-rc1 | ❌ Not compatible |
| AgentProtocol/1.0 | 1.0.0 | ✅ Full |

### Protocol Changes Summary

| Aspect | AgentProtocol/0.1 | AgentProtocol/1.0 |
|--------|-------------------|-------------------|
| Transport | gRPC (HTTP/2) | NATS (JetStream) |
| Auth | Pre-shared token | mTLS certificate |
| Heartbeat | Fixed interval | Adaptive with load data |
| Capabilities | Not reported | Reported in heartbeat |
| Offline queue | Not supported | Built-in |

### Agent Compatibility Matrix

| Agent Version | Control-Plane v0.3.0-rc1 | Control-Plane v1.0.0 |
|--------------|--------------------------|----------------------|
| v0.3.0-rc1 | ✅ | ❌ |
| v1.0.0 | ❌ | ✅ |

---

## SDK Compatibility

### Language Matrix

| SDK | SDK Version | RenOS v1.0.0 | API v1 | v1alpha1 |
|-----|-------------|---------------|--------|----------|
| Go | v1.0.0 | ✅ | ✅ | ✅ |
| TypeScript | v1.0.0 | ✅ | ✅ | ✅ |
| Python | v1.0.0 | ✅ | ✅ | ✅ |
| C# | v1.0.0 | ✅ | ✅ | ✅ |
| Rust | v1.0.0 | ✅ | ✅ | ✅ |

### SDK Feature Parity

| Feature | Go | TS | Python | C# | Rust |
|---------|----|----|--------|----|------|
| Workflow CRUD | ✅ | ✅ | ✅ | ✅ | ✅ |
| Agent management | ✅ | ✅ | ✅ | ✅ | ✅ |
| Plugin management | ✅ | ✅ | ✅ | ✅ | ✅ |
| Webhook subscription | ✅ | ✅ | ✅ | ✅ | ✅ |
| Audit log query | ✅ | ✅ | ✅ | ✅ | ✅ |
| Retry with backoff | ✅ | ✅ | ✅ | ✅ | ✅ |
| Circuit breaker | ✅ | ✅ | ✅ | ✅ | ✅ |
| mTLS support | ✅ | ✅ | ❌* | ❌* | ✅ |
| Streaming logs | ✅ | ✅ | ✅ | ✅ | ✅ |

*Python and C# SDKs delegate mTLS to the underlying HTTP client configuration.

---

## CLI Compatibility

### CLI Version Matrix

| CLI Version | Control-Plane v0.3.0-rc1 | Control-Plane v1.0.0 |
|-------------|--------------------------|----------------------|
| v0.3.0-rc1 | ✅ | Partial† |
| v1.0.0 | ❌ | ✅ |

† v0.3.0-rc1 CLI works with v1.0.0 control-plane for basic operations but fails on renamed commands and displays incorrect field names in output.

### CLI Command Mapping

| v1.0.0 Command | Backward Compatible Alias | Notes |
|----------------|---------------------------|-------|
| `renctl workflow create` | ❌ | Replaces `renctl submit` |
| `renctl workflow list` | ❌ | Replaces `renctl list` |
| `renctl workflow get` | ❌ | Replaces `renctl describe` |
| `renctl workflow delete` | ❌ | Replaces `renctl delete` |
| `renctl agent list` | — | New command |
| `renctl namespace` | — | New command |

---

## File Format Compatibility

### Workflow Definition (YAML)

| Format Version | v0.3.0-rc1 | v1.0.0 |
|---------------|------------|--------|
| v1 (0.3.0-rc1) | ✅ | ✅ (auto-converted on import) |
| v2 (1.0.0) | ❌ | ✅ |

Workflow files created in v0.3.0-rc1 are automatically converted to the v2 format when imported into v1.0.0. The reverse is not supported.

Key format changes:
- `version` field now required in v2
- `on:` trigger syntax changed from list to map
- `steps[].id` now required for all steps
- `steps[].dependencies` replaces `steps[].depends_on`

### Schedule Definition (YAML)

| Format Version | Compatible |
|---------------|------------|
| v0.3.0-rc1 | ✅ Auto-converted |
| v1.0.0 | ✅ Native |

### Plugin Manifest

| Format Version | Compatible |
|---------------|------------|
| v1alpha1 | ❌ |
| v1.0.0 | ✅ |

### Export Format (JSON)

| Format Version | v0.3.0-rc1 | v1.0.0 |
|---------------|------------|--------|
| v0.3.0-rc1 export | ✅ | ✅ (auto-converted) |
| v1.0.0 export | ❌ | ✅ |

---

## Database Schema Migration Compatibility

| From Schema | To Schema | Supported | Notes |
|-------------|-----------|-----------|-------|
| v0.3.0-rc1 | v1.0.0 | ✅ | Automated migration on startup |
| v1.0.0 | v0.3.0-rc1 | ❌ | Rollback requires pg_dump/restore |

### Migration Path

```
v0.3.0-rc1 schema ──automatic──▶ v1.0.0 schema
                                      │
                                      │ (rollback)
                                      ▼
                              Requires pg_dump/restore
                              from backup ──▶ v0.3.0-rc1
```

---

## Helm Chart Compatibility

| Helm Chart Version | Kubernetes Versions | RenOS Version |
|--------------------|---------------------|---------------|
| 1.0.0 | 1.25, 1.26, 1.27, 1.28, 1.29, 1.30 | 1.0.0 |

### Chart Dependencies

| Dependency | Version | Notes |
|------------|---------|-------|
| NATS (nats/nats) | 1.1.x | Included as sub-chart |
| PostgreSQL (bitnami/postgresql) | 15.x | Optional, external DB supported |
| cert-manager | 1.12+ | Required for mTLS |

### Ingress Controller Compatibility

| Ingress Controller | Tested | Notes |
|-------------------|--------|-------|
| ingress-nginx | ✅ | Default in chart values |
| Traefik | ✅ | Community-contributed values |
| HAProxy | ✅ | Community-contributed values |
| AWS ALB Ingress Controller | ✅ | Requires alb-ingress-controller |
| GKE Ingress | ⚠️ | Requires manual annotation adjustments |

---

## Docker Compose Compatibility

| Compose File Version | Compatible |
|---------------------|------------|
| 3.8+ | ✅ |

### Compose Services

| Service | Image | Notes |
|---------|-------|-------|
| control-plane | `renos/control-plane:1.0.0` | Required |
| agent | `renos/agent:1.0.0` | Optional (can run natively) |
| nats | `nats:2.10-alpine` | Required |
| postgres | `postgres:16-alpine` | Optional (external supported) |
| redis | `redis:7-alpine` | Optional |

---

## Deprecation Timeline

| Feature | Deprecated In | Removal Planned |
|---------|--------------|----------------|
| API v1alpha1 | 1.0.0 | 2.0.0 |
| AgentProtocol/0.1 | 1.0.0 | 1.1.0 |
| Token-based agent auth | 1.0.0 | 1.1.0 |
| Plugin API v1alpha1 | 1.0.0 | 1.1.0 |
| `RENOS_CONTROL_API_ADDR` env var | 1.0.0 | 1.2.0 |
| Legacy CLI aliases (`renctl submit`, etc.) | 1.0.0 | 1.2.0 |

---

For compatibility questions not covered here, please open an issue at https://github.com/renos/renos/issues or contact support@renos.io.
