# Small Business

Office PC health checks, business-hours device policy, and alert escalation
for a small office.

## What's included

| File | Type | Description |
|---|---|---|
| `office-pc-health-check.json` | workflow | Weekday-morning sweep of CPU, memory, disk, and pending updates |
| `business-hours-device-policy.json` | workflow | Verifies PCs online and applies the business-hours policy each weekday |
| `schedule-office-health-check.json` | schedule | Cron command that runs a quick health check on the office group |
| `alert-escalation.json` | alert rule | Critical alert when a device stays offline > 15 minutes |

## What to change before importing

- Create a device group named `office` (Devices → Groups) and add the office
  PCs, or replace `device_group: "office"` with specific device names.
- The schedule uses `target_type: "group"` with `target_ids: ["office"]`. Use
  the actual group ID (or switch to `"device"` with device IDs).
- `alert-escalation.json` assumes the `device.online` metric is reported by
  agents. Adjust `metric_name` to match your fleet metric.
- Cron expressions are in UTC; change `trigger_config.timezone` if your office
  runs in another zone.

## How to import

```bash
./examples/library/import.sh small-business https://renos.local admin@renos.local
```

Re-running is safe: already-imported items (matched by name) are skipped.
