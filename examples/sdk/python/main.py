#!/usr/bin/env python3
"""RenOS Python SDK usage example."""

import os
import time

from renos import RenOSClient


def main():
    email = os.environ.get("RENOS_EMAIL")
    password = os.environ.get("RENOS_PASSWORD")
    server = os.environ.get("RENOS_URL", "http://localhost:8443")

    if not email or not password:
        print("RENOS_EMAIL and RENOS_PASSWORD must be set")
        return

    client = RenOSClient.login(server, email, password)
    print("connected to RenOS")

    # List devices
    devices = client.list_devices(page=1, per_page=10)
    print(f"found {devices.total} devices:")
    for device in devices.data:
        print(f"  - {device.name} ({device.hostname}) [{device.status}]")

    # Execute a command on the first device
    if devices.data:
        device = devices.data[0]
        print(f"\nrunning command on {device.hostname}...")
        cmd = client.execute_command(device.id, "uname -a")
        print(f"command {cmd.id}: {cmd.status}")

        time.sleep(2)

        result = client.get_command(device.id, cmd.id)
        print(f"output: {result.output}")

    # List workflows
    workflows = client.list_workflows(page=1, per_page=20)
    print(f"\nworkflows: {workflows.total}")
    for wf in workflows.data:
        print(f"  - {wf.name} (enabled: {wf.enabled})")

    # List jobs
    jobs = client.list_jobs(page=1, per_page=10)
    print(f"\njobs: {jobs.total}")
    for job in jobs.data:
        print(f"  - {job.type} [{job.status}] {job.completed}/{job.total}")

    # Active alerts
    alerts = client.list_alerts(page=1, per_page=10)
    print(f"\nalerts: {alerts.total}")
    for alert in alerts.data:
        print(f"  - {alert.name} ({alert.severity}) [{alert.status}]")

    # Users
    users = client.list_users(page=1, per_page=20)
    print(f"\nusers: {users.total}")
    for user in users.data:
        print(f"  - {user.email} ({user.display_name}) [{user.role}]")

    # Groups
    groups = client.list_groups(page=1, per_page=20)
    print(f"\ngroups: {groups.total}")
    for group in groups.data:
        print(f"  - {group.name} ({group.device_count} devices)")

    # Schedules
    schedules = client.list_schedules(page=1, per_page=20)
    print(f"\nschedules: {schedules.total}")
    for sched in schedules.data:
        print(f"  - {sched.name} ({sched.cron_expression})")

    # Vault secrets
    secrets = client.list_vault_secrets(page=1, per_page=20)
    print(f"\nvault secrets: {secrets.total}")
    for secret in secrets.data:
        print(f"  - {secret.name} (v{secret.version})")

    # Cluster nodes
    nodes = client.list_cluster_nodes(page=1, per_page=20)
    print(f"\ncluster nodes: {nodes.total}")
    for node in nodes.data:
        print(f"  - {node.name} ({node.role}) [{node.status}]")

    # Stream events via SSE
    print("\nstreaming events (ctrl+c to stop)...")
    try:
        stream = client.stream_events("/api/v1/deploy/stream")
        for event in stream:
            print(f"event: {event}")
    except KeyboardInterrupt:
        print("\nstream stopped")

    # WebSocket example
    print("\nconnecting to WebSocket...")
    ws = client.websocket("/api/v1/ws")
    ws.enable_reconnect()
    try:
        msg = ws.recv()
        print(f"ws message: {msg}")
    except Exception as e:
        print(f"ws error: {e}")
    finally:
        ws.close()

    print("\ndone")


if __name__ == "__main__":
    main()
