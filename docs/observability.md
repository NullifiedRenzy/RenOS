# RenOS Observability Architecture

## Overview

RenOS uses OpenTelemetry for distributed tracing, Prometheus for metrics collection,
structured JSON logging via slog, and liveness/readiness probes for orchestration.

## Architecture Diagram

```
                    ┌──────────────┐
                    │  RenOS Core  │
                    │  (Go 1.25)   │
                    └──────┬───────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
              ▼            ▼            ▼
       ┌──────────┐ ┌──────────┐ ┌──────────┐
       │   OTLP   │ │Prometheus│ │  stdout  │
       │ Collector│ │ /metrics │ │(JSON log)│
       └────┬─────┘ └────┬─────┘ └──────────┘
            │            │
            ▼            ▼
      ┌─────────┐  ┌─────────┐
      │ Jaeger/ │  │Grafana  │
      │ Zipkin  │  │+ Alert- │
      │         │  │ manager │
      └─────────┘  └─────────┘
```

## Components

### 1. Distributed Tracing (OpenTelemetry)

- **Middleware:** `observability.TracingMiddleware` — wraps all HTTP requests
- **Exporter:** OTLP HTTP (default), Zipkin, or stdout
- **Sample Rate:** Configurable via `RENOS_TRACING_SAMPLE_RATE` (default 0.1)
- **Propagation:** W3C TraceContext + Baggage
- **Spans created for:**
  - HTTP requests (method + path)
  - Database queries (`StartDBSpan`)
  - NATS publishes (`StartNATSSpan`)
  - WebSocket messages (`StartWSSpan`)
  - Workflow steps (`StartWorkflowSpan`)

### 2. Metrics (Prometheus)

- **Endpoint:** `/metrics` (scraped by Prometheus)
- **35 metric collectors** organized by subsystem:
  - `renos_http_*` — API request rate, latency, errors, active requests
  - `renos_db_*` — Query rate, latency, connection pool stats
  - `renos_nats_*` — Publish/subscribe rate, reconnects
  - `renos_redis_*` — Command latency, errors
  - `renos_agents_*` — Online count, heartbeat/command latency
  - `renos_terminal_*` — Active sessions, byte transfer
  - `renos_ai_*` — Prompt rate, latency, tokens, provider failures
  - `renos_plugin_*` — Execution rate, duration, workers
  - `renos_workflow_*` — Execution rate, duration, steps
  - `renos_circuit_breaker_*` — Trip count
  - `renos_slo_*` — SLO availability, error budget, latency
- **Go runtime:** `go_goroutines`, `go_memstats_*`, `go_gc_*`
- **Process:** `process_cpu_seconds_total`, `process_resident_memory_bytes`

### 3. Structured Logging (slog)

- **Format:** JSON to stdout
- **Correlation IDs:** `trace_id`, `span_id`, `request_id`, `user_id`, `device_id`
- **SLO events:** Recorded as structured log entries

### 4. Health Endpoints

| Endpoint | Purpose | Dependencies checked |
|----------|---------|---------------------|
| `/livez` | Liveness probe (K8s) | None — always OK |
| `/startupz` | Startup probe (K8s) | PostgreSQL, Redis |
| `/api/v1/healthz` | Application health | None |
| `/api/v1/readyz` | Readiness probe | PostgreSQL, NATS, Redis |
| `/api/v1/deephealth` | Full health report | PostgreSQL, NATS, Redis + Go stats |

### 5. Profiling (pprof)

Enable via `RENOS_PPROF_ENABLED=true` (disabled by default).
Available at `/debug/pprof/` under admin auth.

## Configuration

All observability is configured via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `RENOS_TRACING_ENABLED` | `false` | Enable OpenTelemetry tracing |
| `RENOS_TRACING_EXPORTER` | `otlp` | Trace exporter (otlp, zipkin, stdout) |
| `RENOS_TRACING_ENDPOINT` | `http://localhost:4318` | OTLP HTTP endpoint |
| `RENOS_TRACING_SAMPLE_RATE` | `0.1` | Trace sampling rate (0.0-1.0) |
| `RENOS_METRICS_ENABLED` | `true` | Enable Prometheus metrics |
| `RENOS_METRICS_PATH` | `/metrics` | Metrics HTTP path |
| `RENOS_PPROF_ENABLED` | `false` | Enable pprof endpoints |
| `RENOS_PPROF_PATH` | `/debug/pprof` | pprof HTTP path |
| `RENOS_SLO_ENABLED` | `true` | Enable SLO tracking |
| `RENOS_SLO_PATH` | `/api/v1/slos` | SLO status HTTP path |
| `RENOS_DEEP_HEALTH_ENABLED` | `true` | Enable deep health endpoint |
| `RENOS_DEEP_HEALTH_PATH` | `/api/v1/deephealth` | Deep health HTTP path |
| `RENOS_SERVICE_NAME` | `renos-core` | Service name for traces & metrics |
| `RENOS_ENVIRONMENT` | `production` | Deployment environment |

## Grafana Dashboards

Seven pre-built dashboards are available in `deploy/grafana/dashboards/`:

1. **RenOS — Overview** — HTTP rate, latency, active requests, agents, SLOs
2. **RenOS — Database** — Query rate, latency, connection pool
3. **RenOS — NATS & Messaging** — Publish/subscribe rate, reconnects
4. **RenOS — Agents & Devices** — Online count, heartbeat latency, command latency
5. **RenOS — AI & RenAI** — Prompt rate, latency, tokens, failures
6. **RenOS — Workflows** — Execution rate, duration, steps
7. **RenOS — Plugins** — Execution rate, duration, workers

### Importing Dashboards

```bash
# Via Grafana API
curl -X POST -H "Authorization: Bearer $GRAFANA_TOKEN" \
  -H "Content-Type: application/json" \
  -d @deploy/grafana/dashboards/renos-overview.json \
  https://grafana.example.com/api/dashboards/db
```

## Prometheus Alert Rules

Alert rules are defined in `deploy/prometheus/alerts.yml`. Key alerts:

- **Critical:** RenOSDown, DatabaseDown, NATSDisconnected, AgentsOffline, SLOErrorBudgetExhausted
- **Warning:** HighHTTPErrorRate, HighLatency, DB Latency, Redis Errors, AI Failures

## Usage Examples

### Manual health check
```bash
curl https://renos.example.com/livez
curl https://renos.example.com/startupz
curl https://renos.example.com/api/v1/healthz
curl https://renos.example.com/api/v1/readyz
curl https://renos.example.com/api/v1/deephealth | jq
```

### View SLOs
```bash
curl https://renos.example.com/api/v1/slos | jq
```

### Scrape metrics
```bash
curl https://renos.example.com/metrics | head -50
```

### Enable tracing for debugging
```bash
RENOS_TRACING_ENABLED=true RENOS_TRACING_EXPORTER=stdout ./renos-core
```

### Enable pprof for profiling
```bash
RENOS_PPROF_ENABLED=true ./renos-core
# Then:
go tool pprof http://localhost:8443/debug/pprof/heap
go tool pprof http://localhost:8443/debug/pprof/profile?seconds=30
```

### Custom slog attributes in workflow code
```go
import "github.com/renos-project/renos/core/internal/observability"

ctx := observability.WithWorkflowID(ctx, workflowID)
ctx = observability.WithUserID(ctx, userID)
```
