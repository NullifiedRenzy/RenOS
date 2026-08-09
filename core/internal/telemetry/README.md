# Telemetry

RenOS Core ships with an **opt-in, anonymous** usage telemetry sender. It helps
the project understand which versions and platform combinations are in use so
reliability work can be prioritized.

## Privacy policy

- **Anonymous only.** The payload contains exactly the following fields:
  application version, OS and architecture, a random per-install identifier,
  process uptime, timestamp, and counts of devices, workflows and plugins.
- **No personal data.** No hostnames, no usernames, no email addresses, no IP
  addresses and no device details are collected or transmitted.
- **No secrets.** Configuration values, environment variables, tokens, API keys
  and passwords are never sent.
- **Never enabled by default.** Telemetry only activates when the operator
  explicitly sets `RENOS_TELEMETRY_ENABLED=true`. The default is `false`.
- **Failures are silent.** If the endpoint is unreachable the sender logs at
  debug level and moves on. Telemetry never blocks or delays shutdown.

## Configuration

| Environment variable            | Default                               | Description                          |
|---------------------------------|---------------------------------------|--------------------------------------|
| `RENOS_TELEMETRY_ENABLED`       | `false`                               | Opts in to anonymous telemetry.      |
| `RENOS_TELEMETRY_ENDPOINT`      | `https://telemetry.renos.io/v1/ping`  | Ingestion endpoint.                  |
| `RENOS_TELEMETRY_INSTANCE_ID`   | *(random, persisted)*                 | Stable random instance identifier.   |

The instance identifier is generated once and persisted to
`/var/lib/renos/telemetry-instance-id` so it stays stable across restarts.

## How it works

A background goroutine sends a small JSON `POST` to the endpoint on startup and
then once every 24 hours. The HTTP client has a 10 second timeout. Stopping the
service cancels the context and waits for the goroutine to exit cleanly.

## Example payload

```json
{
  "version": "0.1.0",
  "os": "linux",
  "arch": "amd64",
  "instance_id": "1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d",
  "uptime_seconds": 86400,
  "devices": 4,
  "workflows": 2,
  "plugins": 1,
  "timestamp": "2026-08-03T12:00:00Z"
}
```

If you have questions about what is collected, open an issue in the RenOS
repository.
