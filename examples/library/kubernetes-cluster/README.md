# Kubernetes Cluster

Cluster node checks, deployment rollback, and resource-pressure alerts for a
Kubernetes cluster.

## What's included

| File | Type | Description |
|---|---|---|
| `cluster-node-check.json` | workflow | Verifies nodes are Ready and kubelet is running (every 10 min) |
| `deployment-rollback.json` | workflow | Webhook-triggered `kubectl rollout undo` with status verification |
| `schedule-node-check.json` | schedule | Cron command that dumps node + non-running pod status |
| `alert-resource-pressure.json` | alert rule | Warning when node memory pressure exceeds 90% |

## What to change before importing

- The control-plane device is `k8s-master`; rename it in `config.device` and in
  schedule `target_ids`. Worker nodes are matched via the `k8s-nodes` device
  group — create it or use individual device names.
- `deployment-rollback.json` uses `{{deployment}}` and `{{namespace}}`
  placeholders. Fill them in (or pre-fill the config) before triggering the
  webhook.
- `cluster-node-check.json` assumes `kubectl` is configured on `k8s-master`
  (a kubeconfig with cluster-admin or node read access).
- `alert-resource-pressure.json` expects the `node.memory_pressure` metric from
  your monitoring exporter.

## How to import

```bash
./examples/library/import.sh kubernetes-cluster https://renos.local admin@renos.local
```

Re-running is safe: already-imported items (matched by name) are skipped.
