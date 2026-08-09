# Enterprise

Compliance audits, employee onboarding, and patch management for a larger
fleet with stricter controls.

## What's included

| File | Type | Description |
|---|---|---|
| `compliance-audit.json` | workflow | Weekly inventory, password/patch/media-policy audit |
| `employee-onboarding.json` | workflow | Manual run: create account, add groups, provision device, notify HR |
| `patch-management.json` | workflow | Weekly staged patch cycle with reboot and verification |
| `schedule-patch-window.json` | schedule | Cron job that triggers `install` updates on the servers group |
| `alert-compliance-drift.json` | alert rule | Critical alert when compliance score < 95 |

## What to change before importing

- **Placeholders**: `employee-onboarding.json` uses `{{first_name}}`,
  `{{last_name}}`, `{{email}}`, and `{{groups}}` placeholders — fill them in when
  you run the workflow.
- Device group names (`servers`, `provisioning`, `dc-01`) must match your
  environment, or replace them with real device names/IDs.
- `schedule-patch-window.json` uses `target_type: "group"` with the `servers`
  group ID; update `target_ids` to the real group ID.
- `alert-compliance-drift.json` expects the `compliance.score` metric. Point
  `metric_name` at the metric your compliance job publishes.
- Audit commands assume the `renos-agent` binary supports the `inventory`,
  `audit`, and `check-updates` subcommands; adjust to your actual tooling.

## How to import

```bash
./examples/library/import.sh enterprise https://renos.local admin@renos.local
```

Re-running is safe: already-imported items (matched by name) are skipped.
