# API Usage Guide

## Base URL

- **Development:** `http://localhost:8443`
- **Production:** `https://your-domain.com` (via Nginx reverse proxy)

All endpoints are under `/api/v1/`.

## Authentication

### Login

```bash
curl -X POST http://localhost:8443/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{
    "email": "admin@renos.local",
    "password": "admin"
  }'
```

Response:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "id": "...",
    "email": "admin@renos.local",
    "display_name": "Admin",
    "role": "admin"
  }
}
```

### Using the Token

Include the token in the `Authorization` header for all subsequent requests:

```bash
TOKEN="eyJhbGciOiJIUzI1NiIs..."
curl -H "Authorization: Bearer $TOKEN" http://localhost:8443/api/v1/devices
```

## Health & Status

### Health Check
```bash
curl http://localhost:8443/api/v1/healthz
# Returns: {"status":"ok","version":"0.1.0"}
```

### Readiness Check
```bash
curl http://localhost:8443/api/v1/readyz
# Returns detailed status of all dependencies (DB, NATS, Redis)
```

### Version Info
```bash
curl http://localhost:8443/api/v1/version
# Returns: {"version":"0.1.0","commit":"abc1234","build_time":"2026-07-23T00:00:00Z"}
```

## Devices

### List All Devices
```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8443/api/v1/devices
```

### Get Device Details
```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8443/api/v1/devices/{device_id}
```

### Generate Enrollment Token
```bash
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  http://localhost:8443/api/v1/devices/enroll-token \
  -d '{"ttl_minutes": 15}'
```

### Run Command on Device
```bash
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  http://localhost:8443/api/v1/devices/{device_id}/commands \
  -d '{"command": "uname -a"}'
```

### List Command Results
```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8443/api/v1/devices/{device_id}/commands
```

## Docker Management

### Containers
```bash
# List containers
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8443/api/v1/devices/{device_id}/containers

# Start/stop/restart container
curl -X POST -H "Authorization: Bearer $TOKEN" \
  http://localhost:8443/api/v1/devices/{device_id}/containers/{container_id}/start
```

### Images, Volumes, Networks
```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8443/api/v1/devices/{device_id}/docker-images

curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8443/api/v1/devices/{device_id}/docker-volumes

curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8443/api/v1/devices/{device_id}/docker-networks
```

## Systemd Services

```bash
# List services
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8443/api/v1/devices/{device_id}/systemd-services

# Start/stop/restart service
curl -X POST -H "Authorization: Bearer $TOKEN" \
  http://localhost:8443/api/v1/devices/{device_id}/services/{unit_name}/restart
```

## Device Groups

```bash
# List groups
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8443/api/v1/groups

# Create group
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  http://localhost:8443/api/v1/groups \
  -d '{"name": "webservers", "description": "Web server fleet"}'

# Add device to group
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  http://localhost:8443/api/v1/groups/{group_id}/devices \
  -d '{"device_id": "device-uuid"}'

# Run command on all devices in group
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  http://localhost:8443/api/v1/groups/{group_id}/commands \
  -d '{"command": "uptime"}'
```

## Schedules

```bash
# List schedules
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8443/api/v1/schedules

# Create schedule (daily health check at 2am)
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  http://localhost:8443/api/v1/schedules \
  -d '{
    "name": "Daily Health Check",
    "type": "command",
    "payload": {"command": "uptime"},
    "target_type": "group",
    "target_ids": ["group-id"],
    "cron_expression": "0 2 * * *"
  }'
```

## WebSocket (Real-time Events)

```javascript
const ws = new WebSocket(
  `ws://localhost:8443/api/v1/ws?token=${jwtToken}`
);

ws.onmessage = (event) => {
  const msg = JSON.parse(event.data);
  // msg.type: "device.health" | "device.online" | "device.offline"
  // msg.device_id: "..."
  // msg.data: { ... }
};

ws.onclose = () => {
  // Reconnect with backoff
};
```

## Error Handling

All error responses follow this format:
```json
{
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Invalid or expired token"
  }
}
```

### HTTP Status Codes

| Code | Meaning |
|---|---|
| 200 | Success |
| 201 | Created |
| 400 | Bad Request (invalid input) |
| 401 | Unauthorized (no/invalid token) |
| 403 | Forbidden (insufficient permissions) |
| 404 | Not Found |
| 429 | Rate Limited |
| 500 | Internal Server Error |
| 503 | Service Unavailable |

## Rate Limits

| Endpoint | Limit | Burst |
|---|---|---|
| Login | 5 req/min | 10 |
| API | 50 req/s | 100 |
| WebSocket | 2 req/s | 5 |
