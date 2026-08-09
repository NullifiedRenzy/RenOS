# Docker Host

Container health monitoring, compose stack deployment, and image cleanup for a
single Docker host.

## What's included

| File | Type | Description |
|---|---|---|
| `container-health-monitor.json` | workflow | Restarts unhealthy containers and detects restart loops (every 5 min) |
| `compose-stack-deploy.json` | workflow | Webhook-triggered compose deploy with health check |
| `image-cleanup.json` | workflow | Nightly prune of images, build cache, and volumes |
| `schedule-image-cleanup.json` | schedule | Cron command that runs `docker system prune -af --volumes` |
| `alert-container-restart.json` | alert rule | Warning when a container restart count > 3 in 5 minutes |

## What to change before importing

- The Docker host device is `docker-01`; rename it in `config.device` and in
  schedule `target_ids`.
- `compose-stack-deploy.json` points at `/srv/apps/app/docker-compose.yml` and
  checks `http://localhost/healthz` — update the compose path and health URL to
  match your app.
- `container-health-monitor.json` relies on containers declaring a Docker
  `healthcheck`. Without one, no container shows `unhealthy`.
- `alert-container-restart.json` expects the `container.restarts` metric from
  the container-metrics sync on the agent.

## How to import

```bash
./examples/library/import.sh docker-host https://renos.local admin@renos.local
```

Re-running is safe: already-imported items (matched by name) are skipped.
