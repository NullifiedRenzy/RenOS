# RenOS 1.0.0 Known Issues

This document lists known issues in the RenOS 1.0.0 release. Each issue is assigned a tracking ID for reference.

---

## KI-001: WebSocket Reconnection Delay

| Field | Value |
|-------|-------|
| **ID** | KI-001 |
| **Title** | WebSocket reconnection delay after network interruption |
| **Description** | When the control-plane network is interrupted, the Hub UI WebSocket connection takes up to 30 seconds to detect the disconnection and initiate a reconnect. During this window the UI displays stale data without indicating connection loss. |
| **Impact** | Users may make decisions based on stale pipeline state for up to 30 seconds after a network interruption. |
| **Workaround** | Refresh the browser page manually. Set `RENOS_WEBSOCKET_PING_INTERVAL` to a lower value (e.g., `5s`) to reduce detection time. |
| **Affected Versions** | 1.0.0 |
| **Status** | Open — investigation in progress |
| **Category** | Hub |

---

## KI-002: Large Workflow Export Timeout

| Field | Value |
|-------|-------|
| **ID** | KI-002 |
| **Title** | Workflow export fails for very large workflows (>10K nodes) |
| **Description** | Exporting a workflow with more than 10,000 nodes via the Hub UI or API `GET /api/v1/workflows/{id}/export` may result in a gateway timeout (504) after 30 seconds. The export endpoint does not support streaming or pagination. |
| **Impact** | Users with large workflows cannot export them through the standard API or UI. CLI export via `renctl workflow export` is also affected since it uses the same endpoint. |
| **Workaround** | Export in batches by splitting the workflow into sub-workflows, or increase the API gateway timeout to 120s. A direct database export is available for emergency cases. |
| **Affected Versions** | 1.0.0 |
| **Status** | Open — targeted fix in 1.0.1 |
| **Category** | Core |

---

## KI-003: Windows Agent Auto-Update Restart Failure

| Field | Value |
|-------|-------|
| **ID** | KI-003 |
| **Title** | Windows agent auto-update fails to restart as a Windows Service |
| **Description** | When the RenOS agent running as a Windows Service performs an auto-update, the post-update restart attempt fails because the service manager does not release the process handle in time. The agent binary is updated but the service remains in a stopped state and requires manual restart. |
| **Impact** | Windows agent auto-updates require manual intervention (restart the service from Services console or via `sc start renos-agent`). |
| **Workaround** | Disable auto-update on Windows agents and update manually. Or configure a scheduled task to restart the service daily. |
| **Affected Versions** | 1.0.0 |
| **Status** | Open — fix planned for 1.0.1 |
| **Category** | Agent |

---

## KI-004: Plugin Hot-Reload Not Supported

| Field | Value |
|-------|-------|
| **ID** | KI-004 |
| **Title** | Plugin hot-reload not supported at runtime |
| **Description** | Plugins cannot be reloaded without restarting the control-plane process. While the Plugin API supports versioned manifests, there is no runtime mechanism to swap a plugin binary or configuration without a full process restart. The documentation incorrectly implies hot-reload is available. |
| **Impact** | Plugin updates require a rolling restart of control-plane pods, causing brief downtime for workflow submissions. |
| **Workaround** | Use a rolling update strategy in Kubernetes (`kubectl rollout restart deployment/renos-control`). For single-node deployments, schedule plugin updates during maintenance windows. |
| **Affected Versions** | 1.0.0 |
| **Status** | Open — feature request under design |
| **Category** | Agent |

---

## KI-005: Redis Sentinel Mode Untested

| Field | Value |
|-------|-------|
| **ID** | KI-005 |
| **Title** | Redis Sentinel high-availability mode untested |
| **Description** | The RenOS configuration allows setting `RENOS_REDIS_SENTINEL_MASTER` and related options, but Sentinel failover behavior has not been tested in production-like scenarios. Certain edge cases (e.g., sentinel failover during a workflow state write) may cause data inconsistency. |
| **Impact** | Using Redis with Sentinel for HA is unsupported in production until testing is complete. Users may experience workflow state corruption during failover events. |
| **Workaround** | Use Redis Cluster mode instead of Sentinel for HA deployments. For single-instance Redis, configure persistence (AOF + RDB) and accept brief downtime during restarts. |
| **Affected Versions** | 1.0.0 |
| **Status** | Open — testing scheduled for 1.1.0 |
| **Category** | Core |

---

## KI-006: Multi-Region NATS Requires Manual Configuration

| Field | Value |
|-------|-------|
| **ID** | KI-006 |
| **Title** | Multi-region NATS clustering requires manual leaf-node configuration |
| **Description** | RenOS does not automate the setup of NATS leaf nodes for multi-region deployments. Users must manually configure NATS leaf-node connections across regions. The Helm chart does not include leaf-node configuration templates. |
| **Impact** | Multi-region deployments require significant manual effort to configure NATS leaf-node connections, and there is no automated health monitoring for leaf-node links within RenOS. |
| **Workaround** | Follow the NATS documentation to manually configure leaf nodes. Use the NATS `nsc` tool to manage accounts and exports/imports between regions. Set up external monitoring for leaf-node connections. |
| **Affected Versions** | 1.0.0 |
| **Status** | Open — automation planned for 1.2.0 |
| **Category** | Core |

---

## KI-007: PostgreSQL Major Version Upgrade Requires Manual Steps

| Field | Value |
|-------|-------|
| **ID** | KI-007 |
| **Title** | PostgreSQL major version upgrade requires manual dump/restore |
| **Description** | RenOS does not include automation for PostgreSQL major version upgrades (e.g., PG 15 → 16). The migration guide recommends manual `pg_dump`/`pg_restore`, but in-place upgrades using `pg_upgrade` are not documented or tested with the RenOS schema. |
| **Impact** | Users performing PostgreSQL major upgrades must budget for extended downtime for dump/restore. In-place `pg_upgrade` may work but is unsupported. |
| **Workaround** | Use PostgreSQL logical replication to migrate with minimal downtime, or perform `pg_dump`/`pg_restore` during a maintenance window. Test the upgrade on a staging environment first. |
| **Affected Versions** | 1.0.0 |
| **Status** | Open — `pg_upgrade` validation planned for 1.1.0 |
| **Category** | Installer |

---

## KI-008: Air-Gapped Image Transfer Lacks Documentation

| Field | Value |
|-------|-------|
| **ID** | KI-008 |
| **Title** | Air-gapped installation image transfer procedure undocumented |
| **Description** | The RenOS documentation does not provide a step-by-step procedure for transferring container images to air-gapped environments. While the images can be exported and imported using standard `docker save`/`docker load` or `skopeo` commands, the required image list, version pinning, and verification steps are not documented. |
| **Impact** | Security-conscious users deploying in air-gapped environments must reverse-engineer the image dependencies from the Helm chart, increasing the risk of missing images or version mismatches. |
| **Workaround** | Extract the image list from the Helm chart values: `helm template renos/renos | grep 'image:' | awk '{print $2}' | sort -u`. Use `docker pull`/`docker save` on an internet-connected host, then transfer and `docker load` on the air-gapped host. |
| **Affected Versions** | 1.0.0 |
| **Status** | Open — documentation task scheduled for 1.0.1 |
| **Category** | Documentation |

---

## Issue Summary by Category

| Category | Count | Issue IDs |
|----------|-------|-----------|
| Core | 3 | KI-002, KI-005, KI-006 |
| Hub | 1 | KI-001 |
| Agent | 2 | KI-003, KI-004 |
| Installer | 1 | KI-007 |
| Documentation | 1 | KI-008 |

---

## Tracking

All known issues are tracked in the RenOS GitHub issue tracker with the `known-issue` label:

https://github.com/renos/renos/issues?q=label%3Aknown-issue

For status updates or to report a new issue, please open a GitHub issue or contact support at support@renos.io.
