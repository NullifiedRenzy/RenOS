# Home Lab

Monitoring, nightly backups, and weekly cleanup for a small home lab with a
NAS, media servers, and Docker hosts.

## What's included

| File | Type | Description |
|---|---|---|
| `media-server-uptime.json` | workflow | Restarts Plex/Jellyfin if they stop responding (every 5 min) |
| `nightly-backup.json` | workflow | restic backup + verify + prune of media and configs (02:00 UTC) |
| `disk-cleanup.json` | workflow | Docker prune, journal vacuum, old download cleanup (weekly) |
| `schedule-nightly-backup.json` | schedule | Alternative single-command nightly backup via a cron schedule |
| `alert-disk-space.json` | alert rule | Critical alert when disk usage exceeds 90% |

## What to change before importing

- **Device names** (`media-01`, `media-02`, `nas-01`) appear in workflow step
  `config.device` and in schedule `target_ids`. Rename them to the actual device
  names or IDs in your lab.
- The backup commands assume `restic` is installed on `nas-01` with a password
  file at `/etc/restic/password`. Adjust paths and flags as needed.
- `alert-disk-space.json` evaluates the `disk.usage_percent` metric. Make sure
  your agents report it (default RenOS monitoring collects host metrics).
- Cron expressions are in UTC; change `trigger_config.timezone` if desired.

## How to import

One-click import (from the repo root):

```bash
./examples/library/import.sh home-lab https://renos.local admin@renos.local
```

Or import each file through the UI/API:

```bash
curl -X POST https://renos.local/api/v1/workflows -H 'Authorization: Bearer $TOKEN' \
  -d @media-server-uptime.json
```

The importer is idempotent: re-running it skips anything that already exists
(matched by name).
