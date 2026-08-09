# Proxmox Cluster

VM snapshots, cluster resource monitoring, and VM lifecycle automation for a
Proxmox VE cluster.

## What's included

| File | Type | Description |
|---|---|---|
| `vm-snapshot.json` | workflow | Nightly snapshots of VMs 100/101/102 with verify + prune |
| `cluster-resource-monitor.json` | workflow | `pvecm`/`ha-manager` status and per-node resource checks |
| `vm-lifecycle.json` | workflow | Manual run: graceful stop → snapshot → start → boot verify |
| `schedule-vm-snapshot.json` | schedule | Cron command that snapshots the production VMs |
| `alert-cluster-cpu.json` | alert rule | Warning when node CPU exceeds 90% |

## What to change before importing

- Nodes are `pve-01` and `pve-02`; VMs are `100`, `101`, `102`. Rename them to
  your node/VM IDs in `config.device`/`config.command` and schedule
  `target_ids`.
- Worker nodes are matched via the `proxmox-nodes` device group — create it or
  use individual device names.
- `vm-lifecycle.json` uses a `{{vmid}}` placeholder; fill it in before running.
- Snapshot pruning uses the helper `renos-proxmox prune-snapshots`; swap it for
  your own retention script if you don't use that helper.
- `alert-cluster-cpu.json` expects the `cluster.cpu_percent` metric from your
  node exporter.

## How to import

```bash
./examples/library/import.sh proxmox-cluster https://renos.local admin@renos.local
```

Re-running is safe: already-imported items (matched by name) are skipped.
