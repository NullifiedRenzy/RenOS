# RenOS Python SDK

Official Python SDK for the RenOS API.

## Installation

```bash
pip install renos-sdk
```

## Quick Start

```python
from renos import RenOSClient

# Login with email/password
client = RenOSClient.login("https://renos.example.com", "admin@example.com", "password")

# Or with existing token
client = RenOSClient(base_url="https://renos.example.com", token="your-jwt-token")

# Or with API key
client = RenOSClient(base_url="https://renos.example.com", api_key="your-api-key")

# List devices
devices = client.list_devices(page=1, per_page=50)
for device in devices.data:
    print(f"{device.name}: {device.status}")

# Execute command on device
cmd = client.execute_command("device-uuid", "uptime")
print(f"Command {cmd.id}: {cmd.status}")

# List jobs
jobs = client.list_jobs()
for job in jobs.data:
    print(f"Job {job.id}: {job.status} ({job.progress}%)")

# Stream real-time events
for event in client.stream_events("/api/v1/deploy/stream"):
    print(event)

# WebSocket connection
ws = client.websocket("/api/v1/ws")
ws.enable_reconnect()
for message in ws:
    print(message)
```

## Development

```bash
pip install -e .
```
