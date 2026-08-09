# Raspberry Pi Fleet

Fleet health reporting, SD card wear monitoring, and auto-restart for a
cluster of Raspberry Pis.

## What's included

| File | Type | Description |
|---|---|---|
| `fleet-health-dashboard.json` | workflow | Collects temp/load/uptime from every Pi every 5 minutes |
| `sd-card-wear-monitor.json` | workflow | Daily SD wear-level check from the eMMC/SD controller |
| `auto-restart-on-hang.json` | workflow | Reboots a Pi whose heartbeat file is stale (every 2 min) |
| `schedule-fleet-health.json` | schedule | Cron command that runs a fleet status script |
| `alert-sd-card-wear.json` | alert rule | Warning when SD wear exceeds 80% |

## What to change before importing

- Device names are `pi-01`…`pi-04`. Rename them or add/remove steps to match
  your actual fleet (copy a step block and change `device`).
- For `auto-restart-on-hang.json`, each Pi must write a heartbeat file (for
  example via a systemd timer touching `/tmp/renos-heartbeat`). The 300-second
  window is `5 * 60`; tune it to your heartbeat interval.
- `schedule-fleet-health.json` targets the `pis` group; create that group or
  replace `target_ids` with your group/device IDs.
- `alert-sd-card-wear.json` expects the `sd.wear_level` metric from the
  `sd-card-wear-monitor` workflow.

## How to import

```bash
./examples/library/import.sh raspberry-pi-fleet https://renos.local admin@renos.local
```

Re-running is safe: already-imported items (matched by name) are skipped.
