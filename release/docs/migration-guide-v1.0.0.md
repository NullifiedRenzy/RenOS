# Migration Guide: RenOS v0.3.0-rc1 → v1.0.0

This guide covers migrating from RenOS v0.3.0-rc1 to v1.0.0. Read all sections before starting.

---

## Table of Contents

1. [Database Migration Steps](#1-database-migration-steps)
2. [Configuration Changes](#2-configuration-changes)
3. [API Changes](#3-api-changes)
4. [Plugin API Changes](#4-plugin-api-changes)
5. [Agent Protocol Changes](#5-agent-protocol-changes)
6. [CLI Command Changes](#6-cli-command-changes)
7. [SDK Breaking Changes](#7-sdk-breaking-changes)
8. [Step-by-Step Migration](#8-step-by-step-migration)
9. [Rollback Procedure](#9-rollback-procedure)
10. [Verification Checklist](#10-verification-checklist)

---

## 1. Database Migration Steps

### Prerequisites
- PostgreSQL 15+ (v0.3.0-rc1 supports PG 13+, but v1.0.0 requires PG 15 minimum)
- Backup of the existing database

### Run Migrations

The v1.0.0 control-plane runs schema migrations automatically on startup. However, manual steps are required for certain changes:

```sql
-- 1. Backfill created_at from created (if column renamed was deferred)
ALTER TABLE workflows ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ;
UPDATE workflows SET created_at = created WHERE created_at IS NULL;
-- Column 'created' is dropped by the migration automatically

-- 2. New indexes for performance
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflows_status_created
  ON workflows (status, created_at DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_workflow_runs_workflow_id
  ON workflow_runs (workflow_id, started_at DESC);

-- 3. Verify no orphaned plugin instances
SELECT COUNT(*) FROM plugin_instances pi
  LEFT JOIN plugins p ON pi.plugin_id = p.id
  WHERE p.id IS NULL;
```

### Schema Changes Summary

| Table | Change |
|-------|--------|
| `workflows` | `created` → `created_at` (TIMESTAMPTZ); added `version`, `draft_of` columns |
| `workflow_runs` | Added `trigger_type`, `trigger_event` columns |
| `agents` | Added `capabilities` (JSONB), `last_seen` columns |
| `plugin_instances` | Added `config_hash`, `status` columns |
| `audit_log` | New table for audit trail |
| `api_keys` | Added `scopes` column; `key_hash` now uses bcrypt |

---

## 2. Configuration Changes

### Environment Variable Remappings

| v0.3.0-rc1 (old) | v1.0.0 (new) | Notes |
|-------------------|---------------|-------|
| `RENOS_CONTROL_API_ADDR` | `RENOS_API_BIND_ADDRESS` | Format `:8080` or `0.0.0.0:8080` |
| `RENOS_STORE_TYPE` | `RENOS_STORAGE_BACKEND` | Values: `postgres`, `redis`, `nats-jetstream` |
| `RENOS_NATS_URL` | `RENOS_NATS_ENDPOINT` | Comma-separated for cluster |
| `RENOS_REDIS_ADDR` | `RENOS_REDIS_ENDPOINT` | |
| `RENOS_REDIS_PASSWORD` | `RENOS_REDIS_PASSWORD` | Unchanged (kept for clarity) |
| `RENOS_LOG_LEVEL` | `RENOS_LOG_LEVEL` | Unchanged |
| `RENOS_JWT_SECRET` | `RENOS_AUTH_JWT_SECRET` | |

### New Required Variables

These variables must be set for v1.0.0:

```bash
RENOS_API_BIND_ADDRESS=":8080"
RENOS_STORAGE_BACKEND="postgres"
RENOS_NATS_ENDPOINT="nats://localhost:4222"
RENOS_AUTH_JWT_SECRET="<generate-secure-secret>"
```

### Removed Variables

- `RENOS_USE_LEGACY_API` (no longer supported)
- `RENOS_PLUGIN_ALLOW_UNSIGNED` (replaced by `RENOS_PLUGIN_POLICY_FILE`)
- `RENOS_AGENT_GRPC_PORT` (agents now communicate via NATS only)

---

## 3. API Changes

### Deprecated Endpoints (Removed in v1.0.0)

| v0.3.0-rc1 Endpoint | v1.0.0 Replacement |
|---------------------|---------------------|
| `POST /api/v1alpha1/workflows/submit` | `POST /api/v1/workflows` |
| `GET /api/v1alpha1/workflows/{id}` | `GET /api/v1/workflows/{id}` |
| `POST /api/v1alpha1/agents/register` | `POST /api/v1/agents` |
| `GET /api/v1/health` | `GET /healthz`, `GET /readyz` |
| `PUT /api/v1/workflows/{id}` | `PATCH /api/v1/workflows/{id}` |

### Response Format Changes

| Field (old) | Field (new) | Notes |
|-------------|-------------|-------|
| `workflow.metadata.created` | `workflow.metadata.created_at` | ISO 8601 with timezone |
| `workflow.metadata.updated` | `workflow.metadata.updated_at` | ISO 8601 with timezone |
| `status.message` | `status.reason` | Machine-readable enum |
| `pagination.next_cursor` | `pagination.cursor` | Opaque string format |

### New Headers

- `X-Request-Id` — returned on all responses
- `X-RateLimit-Left` — replaces `X-RateLimit-Remaining`
- `X-RateLimit-Reset` — epoch timestamp

---

## 4. Plugin API Changes

### Required Code Changes

```go
// v0.3.0-rc1
func (p *MyPlugin) Init(ctx context.Context) error

// v1.0.0
func (p *MyPlugin) Init(ctx context.Context, cfg PluginConfig) error
```

```go
// v0.3.0-rc1
func (p *MyPlugin) Execute(input map[string]any) (map[string]any, error)

// v1.0.0
func (p *MyPlugin) Execute(ctx context.Context, input map[string]any) (map[string]any, error)
```

### Manifest Changes

```yaml
# v0.3.0-rc1
api_version: "v1alpha1"
name: my-plugin

# v1.0.0
api_version: "v1.0.0"
name: my-plugin
version: "1.0.0"
capabilities:
  - executor
```

### Plugin SDK

All plugins must be rebuilt against the Plugin API v1.0.0 SDK. The v0.3.0 plugin runtime will reject plugins with an older `api_version`.

---

## 5. Agent Protocol Changes

### Protocol Version

- Old: `AgentProtocol/0.1`
- New: `AgentProtocol/1.0`

### Wire Format

Heartbeat messages now include agent capabilities:

```json
{
  "type": "heartbeat",
  "protocol": "AgentProtocol/1.0",
  "agent_id": "...",
  "capabilities": ["executor", "watcher"],
  "load": { "cpu": 0.45, "mem": 0.30 }
}
```

### Authentication

- v0.3.0-rc1: pre-shared token in header
- v1.0.0: mTLS with client certificate; token-based auth deprecated

Agents must be provisioned with a client certificate signed by the control-plane CA.

---

## 6. CLI Command Changes

| v0.3.0-rc1 | v1.0.0 | Notes |
|-------------|--------|-------|
| `renctl submit` | `renctl workflow create` | |
| `renctl list` | `renctl workflow list` | |
| `renctl describe <id>` | `renctl workflow get <id>` | |
| `renctl delete <id>` | `renctl workflow delete <id>` | |
| `renctl logs <id>` | `renctl logs <id>` | Now requires `--follow` to tail |
| `renctl config set` | `renctl config set` | Unchanged |
| `renctl plugin install` | `renctl plugin install` | Unchanged |
| — | `renctl namespace` | New command |
| — | `renctl agent list` | New command |

### Output Format

Default output is now a table format. Use `-o json` or `-o yaml` for structured output.

---

## 7. SDK Breaking Changes

### Go SDK

```go
// v0.3.0-rc1
client := renos.NewClient("http://localhost:8080")

// v1.0.0
client, err := renos.NewClient(
    renos.WithEndpoint("http://localhost:8080"),
    renos.WithAPIKey("sk-..."),
)
```

- `SubmitWorkflow()` → `CreateWorkflow()`
- Response types now use `*renos.Workflow` instead of `*renos.WorkflowSubmission`
- Error types wrap `renos.APIError` with structured details

### TypeScript SDK

```typescript
// v0.3.0-rc1
const client = new RenosClient("http://localhost:8080");

// v1.0.0
const client = new RenosClient({ endpoint: "http://localhost:8080", apiKey: "sk-..." });
```

- All methods return Promises (removed callback overloads)
- `client.submitWorkflow()` → `client.createWorkflow()`
- Full ESM support; CJS available via `renos` (commonjs) entrypoint

### Python SDK

```python
# v0.3.0-rc1
client = RenosClient("http://localhost:8080")

# v1.0.0
async with RenosClient(endpoint="http://localhost:8080", api_key="sk-...") as client:
    ...
```

- Async-only: all I/O methods are coroutines
- `client.submit_workflow()` → `client.create_workflow()`
- Type-annotated throughout; mypy strict-compatible

### C# SDK

```csharp
// v0.3.0-rc1
var client = new RenosClient("http://localhost:8080");

// v1.0.0
var client = new RenosClient(new RenosClientOptions {
    Endpoint = "http://localhost:8080",
    ApiKey = "sk-..."
});
```

- All async methods return `Task<T>` consistently (no more `void` returns)
- `SubmitWorkflowAsync()` → `CreateWorkflowAsync()`
- Nullable reference types enabled

### Rust SDK

```rust
// v0.3.0-rc1
let client = RenosClient::new("http://localhost:8080");

// v1.0.0
let client = RenosClient::builder()
    .endpoint("http://localhost:8080")
    .api_key("sk-...")
    .build()?;
```

- Builder pattern replaces direct construction
- `client.submit_workflow()` → `client.create_workflow()`
- All methods are async and return `Result<T, RenosError>`
- `RenosError` is an opaque error type (use `Display` for messages)

---

## 8. Step-by-Step Migration

### Phase 1: Preparation

1. **Backup everything**

   ```bash
   # Database
   pg_dump -Fc -f renos-v0.3.0-rc1-backup.dump renos

   # Configuration
   cp -r /etc/renos /etc/renos.backup

   # Plugin binaries
   cp -r /var/lib/renos/plugins /var/lib/renos/plugins.backup

   # Agent certificates
   cp -r /etc/renos/agents /etc/renos/agents.backup
   ```

2. **Review release notes and breaking changes** (this document)

3. **Update environment variables** per Section 2

### Phase 2: Database Migration

4. **Upgrade PostgreSQL to 15+** (if needed)

5. **Run the control-plane binary** in migration-only mode:

   ```bash
   renos-control migrate --config /etc/renos/renos.yaml
   ```

   Or let the control-plane run migrations on startup (default).

### Phase 3: Core Upgrade

6. **Stop v0.3.0-rc1 control-plane**

   ```bash
   systemctl stop renos-control
   ```

7. **Deploy v1.0.0 control-plane** with updated config

   ```bash
   # Native
   apt-get install ./renos-control_1.0.0_amd64.deb
   systemctl start renos-control

   # Kubernetes
   helm upgrade renos renos/renos --namespace renos --version 1.0.0 -f values-v1.yaml
   ```

8. **Verify control-plane health**

   ```bash
   curl -s http://localhost:8080/healthz
   # Expected: {"status": "ok"}
   ```

### Phase 4: Hub Upgrade

9. **Upgrade Hub UI** (deployed with control-plane in Helm; standalone upgrade if separate):

   ```bash
   # If using Helm, the UI is upgraded with the control-plane
   # If standalone:
   apt-get install ./renos-hub_1.0.0_amd64.deb
   ```

10. **Verify Hub UI** loads at `https://<host>/` and all workflows are visible.

### Phase 5: Agent Upgrade

11. **Upgrade each agent** (one at a time for stateful pipelines):

    ```bash
    # Linux
    apt-get install ./renos-agent_1.0.0_amd64.deb

    # macOS
    brew upgrade renos/tap/renos-agent

    # Windows
    msiexec /i renos-agent-1.0.0-x64.msi
    ```

12. **Verify agent registration**

    ```bash
    renctl agent list
    # Expected: all agents show status "connected" with protocol "1.0"
    ```

### Phase 6: Plugin Migration

13. **Rebuild all plugins** against Plugin API v1.0.0

14. **Update plugin manifests** to `api_version: "v1.0.0"`

15. **Reinstall plugins**

    ```bash
    renctl plugin install ./my-plugin-v1.0.0.zip
    ```

### Phase 7: Verification

16. **Run the verification checklist** (Section 10)

---

## 9. Rollback Procedure

### Prerequisites

- Backup file: `renos-v0.3.0-rc1-backup.dump`
- Backup config: `/etc/renos.backup`
- v0.3.0-rc1 binaries available

### Steps

1. **Stop v1.0.0 services**

   ```bash
   systemctl stop renos-control renos-hub renos-agent
   ```

2. **Restore database**

   ```bash
   dropdb renos
   createdb renos
   pg_restore -d renos renos-v0.3.0-rc1-backup.dump
   ```

3. **Restore configuration**

   ```bash
   rm -rf /etc/renos
   cp -r /etc/renos.backup /etc/renos
   ```

4. **Install v0.3.0-rc1 binaries**

   ```bash
   dpkg -i ./renos-control-v0.3.0-rc1.deb
   dpkg -i ./renos-agent-v0.3.0-rc1.deb
   ```

5. **Restore plugins**

   ```bash
   rm -rf /var/lib/renos/plugins
   cp -r /var/lib/renos/plugins.backup /var/lib/renos/plugins
   ```

6. **Start services**

   ```bash
   systemctl start renos-control renos-agent
   ```

7. **Verify rollback** using the verification checklist against v0.3.0-rc1 behavior.

**Note:** The Hub UI cannot be rolled back independently — if upgraded, redeploy v0.3.0-rc1 Hub from backup.

---

## 10. Verification Checklist

### Core Functionality

- [ ] Control-plane starts without errors (`journalctl -u renos-control`)
- [ ] All API endpoints respond (run `renctl workflow list`)
- [ ] Workflow submission succeeds (`renctl workflow create -f test-workflow.yaml`)
- [ ] Workflow execution completes with expected output
- [ ] Workflow history is intact (all runs visible)

### Agents

- [ ] All agents show `connected` status with protocol 1.0
- [ ] Agent heartbeat data visible in UI
- [ ] New workflows are dispatched to agents
- [ ] Agent auto-update functions (test on one non-production agent)

### Plugins

- [ ] All marketplace plugins install successfully
- [ ] Custom plugins rebuilt and installed
- [ ] Plugin execution produces correct results
- [ ] Plugin sandboxing active (test with resource-intensive plugin)

### Hub UI

- [ ] Dashboard loads with correct workflow counts
- [ ] Workflow graph renders with live node status
- [ ] Plugin marketplace browsable
- [ ] Audit log entries visible
- [ ] RBAC roles applied correctly

### CLI

- [ ] `renctl workflow list` shows all workflows
- [ ] `renctl workflow get <id>` shows details
- [ ] `renctl agent list` shows agent status
- [ ] `renctl logs <id> --follow` streams logs
- [ ] `renctl namespace` commands work
- [ ] Tab completion loads for current shell

### Observability

- [ ] Prometheus metrics endpoint responds (`/metrics`)
- [ ] Grafana dashboard shows correct data
- [ ] Traces visible in Jaeger/Zipkin
- [ ] Logs written in JSON format

### Security

- [ ] mTLS connections between control-plane and agents
- [ ] API key authentication works
- [ ] Non-root container user (`renos`)
- [ ] No secrets in logs

### Data Integrity

- [ ] All historical workflows accessible
- [ ] Pagination works for large result sets
- [ ] Export/import workflow round-trip succeeds
- [ ] Database indexes in place (`\di` in psql)
