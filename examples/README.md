# RenOS Examples

Complete, real-world examples showing how to use RenOS features.

## Index

| Directory | Example | Description |
|---|---|---|
| `plugins/hello-world/` | Hello World Plugin | Minimal plugin with HTTP routes, background worker, and schedules |
| `workflows/deploy-flow.json` | Deployment Workflow | Multi-stage canary deploy with health checks and rollback |
| `webhooks/github-deploy.json` | GitHub Auto-Deploy | Trigger deployments from GitHub releases and pushes |
| `api/curl.sh` | API Examples | curl commands for all major RenOS API operations |
| `sdk/go/main.go` | Go SDK | Complete Go client usage with device, workflow, and job operations |
| `sdk/typescript/main.ts` | TypeScript SDK | TypeScript client with WebSocket and SSE streaming |
| `sdk/python/main.py` | Python SDK | Python client with all API operations |
| `automation/backup-workflow.json` | Backup Workflow | Nightly encrypted backup with verification and offsite replication |
| `ai/custom-prompt.json` | AI Prompt Config | Custom system prompt for sysadmin AI assistant with intent patterns |
| `docker/compose-monitor.json` | Docker Compose Monitor | Auto-restart failed services with status tracking and escalation |
| `proxmox/vm-lifecycle.json` | Proxmox VM Lifecycle | Full VM provisioning with approval gates and health validation |
| `terminal/session-audit.json` | Terminal Audit | Security audit workflow with anomaly detection and compliance reporting |

## Example Library

Ready-to-run scenario packs, organized by use case. Each pack is a set of
workflows, schedules, and alert rules in a one-click-importable directory
under `library/<pack>/`.

| Pack | Description |
|---|---|
| `home-lab` | Media server uptime, nightly backups, weekly disk cleanup |
| `small-business` | Office PC health checks and business-hours device policy |
| `enterprise` | Compliance audits, employee onboarding, patch management |
| `raspberry-pi-fleet` | Fleet health, SD-card wear monitoring, auto-restart on hang |
| `docker-host` | Container health, compose deploys, image cleanup |
| `kubernetes-cluster` | Node checks, deployment rollback, resource pressure alerts |
| `gaming-server` | Server status, crash auto-restart, latency alerts |
| `proxmox-cluster` | VM snapshots, cluster resource monitoring, VM lifecycle |

### Importing a pack

```bash
./library/import.sh <pack> [base-url] [email] [password]
```

Examples:

```bash
./library/import.sh home-lab https://renos.local admin@renos.local
./library/import.sh docker-host http://localhost:8443 admin@renos.local
./library/import.sh --all https://renos.local admin@renos.local
./library/import.sh --dry-run home-lab
```

The importer is idempotent (existing resources are skipped by name) and uses
the v1 REST API. See `library/<pack>/README.md` for what each pack installs
and what to customize.

## Usage

Most examples are JSON workflow definitions. Import them via the RenOS UI
or CLI:

```bash
renos workflows:create ./workflows/deploy-flow.json
renos workflows:create ./automation/backup-workflow.json
```

SDK examples require the respective SDK package:

```bash
# Go
cd sdk/go && go run ../../examples/sdk/go/main.go

# TypeScript
npx tsx examples/sdk/typescript/main.ts

# Python
python examples/sdk/python/main.py
```

API examples:

```bash
source examples/api/curl.sh
list_devices
execute_workflow <id>
```

See each example's README or inline comments for details.
