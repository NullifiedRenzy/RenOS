# Release Process — v1.0.0-rc1

## Release Checklist

### Pre-Release

- [x] All CI checks pass on `main` branch
- [x] `make check` runs locally without errors
- [x] `go mod tidy` has been run and `go.sum` is committed
- [ ] `agent/Cargo.lock` is up to date
- [x] `hub/package-lock.json` is up to date
- [x] CHANGELOG.md is updated with all changes since last release
- [ ] All Dockerfiles build successfully:
  ```bash
  docker build -t renos-core:test -f core/Dockerfile core
  docker build -t renos-agent:test -f agent/Dockerfile agent
  docker build -t renos-hub:test -f hub/Dockerfile hub
  ```
- [ ] Quick start guide in README.md is tested end-to-end
- [x] `.env.example` reflects all current configuration options
- [x] No hardcoded secrets, debug endpoints, or TODOs remain

### Release Steps

```bash
# 1. Ensure main branch is up to date
git checkout main
git pull origin main

# 2. Run full pre-release checks
make check

# 3. Commit any pending changes
git add -A
git commit -m "chore: prep v1.0.0-rc1"

# 4. Tag the release
git tag -a v1.0.0-rc1 -m "v1.0.0-rc1 — Release candidate 1"

# 5. Push tag to trigger CI release workflow
git push origin v1.0.0-rc1
```

### Post-Release

- [ ] Verify GitHub Release is created with auto-generated notes
- [ ] Verify Docker images are pushed to GHCR:
  - `ghcr.io/{owner}/renos-core:v1.0.0-rc1`
  - `ghcr.io/{owner}/renos-agent:v1.0.0-rc1`
  - `ghcr.io/{owner}/renos-hub:v1.0.0-rc1`
- [ ] Test `docker compose up` with released images
- [ ] Update `dev` branch to match `main`

---

## Tagging Instructions

```bash
# Create an annotated tag (preferred):
git tag -a v1.0.0-rc1 -m "v1.0.0-rc1 — Release candidate 1"

# Push tag to remote:
git push origin v1.0.0-rc1

# Delete a tag locally and remotely (if needed):
git tag -d v1.0.0-rc1
git push origin :refs/tags/v1.0.0-rc1
```

### Tag Naming Convention

- `v{major}.{minor}.{patch}-{codename}`
- Example: `v1.0.0-rc1`
- Hotfixes: `v1.0.1`

---

## M09 Risk Documentation

### 1. WebSocket Direct Connection (Dev Only)

In development, the Hub's `useWebSocket` hook connects directly to the Core WebSocket endpoint at `ws://localhost:8443/api/v1/ws?token=JWT`. This bypasses the Next.js rewrite proxy because Next.js rewrites do not proxy WebSocket connections.

**Impact:** Works in development. In production, a reverse proxy (nginx/Caddy) must be configured to serve both Hub static files and Core API+WS on the same origin.

**Production topology:**
```
Browser → Reverse Proxy (:443)
            ├── /api/v1/* → Core (:8443)       [HTTP + WS upgrade]
            ├── /_next/*  → Hub (:3000)
            └── /*        → Hub (:3000)
```

### 2. WebSocket Origin Check

The Core WebSocket upgrader in `core/internal/ws/handler.go` uses:
```go
CheckOrigin: func(r *http.Request) bool { return true },
```

This accepts WebSocket connections from any origin, which is permissive for development but **must be tightened before public deployment**.

**TODO:** Replace with origin whitelist check before v0.2.0:
```go
var allowedOrigins = map[string]bool{
    "https://renos.example.com": true,
    "http://localhost:3000":      true,
}
CheckOrigin: func(r *http.Request) bool {
    return allowedOrigins[r.Header.Get("Origin")]
}
```

### 3. No TLS in Development

Both Core HTTP and Hub dev server run on plain HTTP. WebSocket connections use `ws://` instead of `wss://`. A production deployment must:
- Terminate TLS at the reverse proxy
- Use `wss://` for WebSocket connections
- Set `HSTS` headers
- Restrict `CheckOrigin` to the production domain
