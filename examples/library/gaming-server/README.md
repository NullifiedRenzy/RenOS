# Gaming Server

Game server status checks, crash auto-restart, and performance metrics for a
dedicated gaming box.

## What's included

| File | Type | Description |
|---|---|---|
| `game-server-status.json` | workflow | Verifies the process and status endpoint every 5 minutes |
| `auto-restart-on-crash.json` | workflow | Detects a crashed process, restarts it, verifies recovery |
| `performance-metrics.json` | workflow | Captures CPU, memory, latency, and tick rate every 10 minutes |
| `alert-high-latency.json` | alert rule | Warning when latency exceeds 100 ms for 5 minutes |

## What to change before importing

- The gaming device is `game-01`; rename it in each workflow's `config.device`.
- The examples assume a `srcds_linux`-style process (Source Dedicated Server)
  with a status endpoint on port 8080 and a restart script at
  `/opt/games/restart.sh`. Adjust the process name, status URL, and restart
  command to match your game and tooling.
- `alert-high-latency.json` expects the `game.latency_ms` metric; publish it
  from the `performance-metrics` workflow or your exporter.

## How to import

```bash
./examples/library/import.sh gaming-server https://renos.local admin@renos.local
```

Re-running is safe: already-imported items (matched by name) are skipped.
