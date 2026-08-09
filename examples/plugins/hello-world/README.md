# Hello World Plugin

A minimal RenOS plugin that demonstrates the core plugin API:
HTTP routes, background workers, and scheduled tasks.

## Manifest

`plugin.json` declares the plugin's identity, permissions, and capabilities.
All plugins must provide a manifest at the root of their directory.

Key fields:
- `type`: `wasm` (WebAssembly) or `container` (Docker sidecar)
- `permissions`: fine-grained capabilities the plugin requests at install
- `routes`: HTTP endpoints registered into the RenOS API gateway
- `workers`: background loops that run at fixed intervals
- `schedules`: cron-triggered tasks

## Entry Point

`main.go` starts an HTTP server on `:8090` and registers three routes:

| Method | Path                                  | Description            |
|--------|---------------------------------------|------------------------|
| GET    | /api/v1/plugins/hello-world/greet     | Returns the greeting   |
| POST   | /api/v1/plugins/hello-world/greet     | Greets a named user    |
| GET    | /api/v1/plugins/hello-world/stats     | Worker run statistics  |

A background goroutine (`workerLoop`) ticks every 60 seconds, simulating
periodic work. The `/stats` endpoint exposes the run count and last tick.

## Installing

```bash
renos plugins:install ./examples/plugins/hello-world/plugin.json
renos plugins:enable hello-world
```

## Testing

```bash
curl http://localhost:8443/api/v1/plugins/hello-world/greet
# {"message":"Hello from RenOS!"}

curl -X POST http://localhost:8443/api/v1/plugins/hello-world/greet \
  -H 'Content-Type: application/json' \
  -d '{"name":"Alice"}'
# {"message":"Hello, Alice! Welcome to RenOS."}

curl http://localhost:8443/api/v1/plugins/hello-world/stats
# {"greet_count":2,"worker_runs":1,"last_worker_run":"...","uptime_seconds":...}
```

## Architecture

```
                   RenOS API Gateway
                         │
          ┌──────────────┼──────────────┐
          │              │              │
     /greet          /stats        workerLoop
          │              │              │
    greetCount    atomic counters    time.Sleep
          │                         60s interval
     response
```

The plugin communicates with RenCore via NATS for lifecycle events and
registers its HTTP routes in the API gateway through the manifest. Each
plugin runs in its own sandbox (Wasm or container) with resource limits
enforced by the Plugin Manager.
