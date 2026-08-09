# Upgrade Guide Index

## Version Table

| From | To | Release Date | Impact | Breaking Changes |
|---|---|---|---|---|
| [v0.1.0](v0.1.0-to-v0.2.0.md) | v0.2.0 | 2026-04-15 | Medium | Plugin API v1, RenFlow engine, new auth system |
| [v0.2.0](v0.2.0-to-v0.3.0.md) | v0.3.0 | 2026-06-01 | High | Database schema changes, SDK v2, API namespace migration |
| [v0.3.0](v0.3.0-to-v1.0.0.md) | v1.0.0 | 2026-07-28 | High | Production hardening, security model, configuration format |

## Upgrade Path

For a multi-hop upgrade (e.g., v0.1.0 → v1.0.0), upgrade sequentially:

```
v0.1.0 → v0.2.0 → v0.3.0 → v1.0.0
```

Each step includes its own database migration, configuration migration,
and rollback procedure. Do not skip versions.

## Before Any Upgrade

1. **Backup your database:**
   ```bash
   ./scripts/backup.sh
   ```
2. **Backup configuration:**
   ```bash
   cp .env .env.backup
   cp -r /etc/renos /etc/renos.backup
   ```
3. **Check current version:**
   ```bash
   renos version
   # or
   curl http://localhost:8443/api/v1/version
   ```
4. **Review release notes** in [CHANGELOG.md](../../CHANGELOG.md)
5. **Test in staging** before production

## Health Check

After any upgrade, verify:

```bash
# All services healthy
curl http://localhost:8443/api/v1/healthz
curl http://localhost:8443/api/v1/readyz

# Version matches expected
curl http://localhost:8443/api/v1/version

# Core functionality works
curl http://localhost:8443/api/v1/devices
curl -X POST http://localhost:8443/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@renos.local","password":"admin"}'
```

See the specific upgrade guide for version-specific verification steps.
