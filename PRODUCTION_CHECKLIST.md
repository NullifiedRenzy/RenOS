# RenOS Production Readiness Checklist

**Version:** 1.0.0-production
**Last updated:** 2026-08-03
**Audience:** operators installing RenOS for real workloads

Use this checklist before, during, and after a production install. Every
command below was verified against the RenOS 1.0.0 repository. If an item
does not apply to your environment, mark it N/A and note why.

> **Scope notes**
> - Commands shown as `./installer/...` run from a checkout of the repository.
> - Commands shown as `curl ... localhost` assume the default API port `8443`
>   (docker install) or the reverse-proxy port `80`/`443` (linux install).
> - `ADMIN_TOKEN` below is obtained from `renctl/renctl login` or the login API — see
>   [Verification Command Reference](#verification-command-reference).

---

## Score your readiness

Give yourself a score per area, then add them up and divide by 19.

| # | Area | Max | How to verify | Your score |
|---|------|-----|---------------|-----------|
| 0 | Install & first-run validation | 100 | `renos-validate.sh --post` exit 0, onboarding complete | |
| 1 | Accounts & access control | 100 | RBAC roles reviewed, no default credentials | |
| 2 | Secrets & keys | 100 | JWT/encryption keys >= 32 chars, not in git, vault used | |
| 3 | TLS & certificates | 100 | HTTPS reachable, cert expiry checked | |
| 4 | Firewall & network | 100 | Only required ports exposed | |
| 5 | Backups | 100 | Backup runs on schedule, encrypted, off-site | |
| 6 | Disaster recovery | 100 | Restore drill passes at least once | |
| 7 | Database | 100 | PostgreSQL 16, backups verified, no default creds | |
| 8 | Monitoring & health | 100 | `/api/v1/health/dashboard` reports healthy | |
| 9 | Alerting & notification | 100 | Alert rules exist, webhook tested | |
| 10 | Logging & audit | 100 | Audit log on, log retention set, crash dir writable | |
| 11 | Rate limiting & DoS | 100 | Login/API limits reviewed | |
| 12 | Resource limits & capacity | 100 | Memory/CPU limits, disk monitoring | |
| 13 | AI integration (optional) | 100 | AI provider key set or disabled | |
| 14 | Email / SMTP | 100 | N/A documented (feature not available) | |
| 15 | MFA / 2FA | 100 | N/A documented (feature not available) | |
| 16 | Updates & upgrades | 100 | Upgrade + rollback procedure rehearsed | |
| 17 | Demo mode & examples (optional) | 100 | Seed/import done in staging only | |
| 18 | Telemetry (optional) | 100 | Telemetry opted in or off, decision recorded | |
| 19 | Health dashboard score | 100 | Dashboard reports >= 80 | |

**Result:** add the 19 scores and divide by 19.

| Overall | Status |
|---|---|
| 90-100 | Production ready |
| 70-89 | Almost — close the gaps listed below |
| < 70 | Do not go live yet |

---

## 0. Install & first-run validation

- [ ] Install RenOS with a supported method:
  - Docker / one-command: `./install.sh` (or `./installer/docker/run.sh`)
  - Linux systemd: `sudo ./installer/linux/one-command-install.sh` or `sudo ./installer/linux/install.sh`
- [ ] Run the fresh-install validation on a blank VM before your real host:

  ```bash
  sudo ./installer/linux/fresh-install-test.sh
  ```

- [ ] Run pre-install validation on the target host:

  ```bash
  sudo ./installer/validate/renos-validate.sh --pre
  ```

- [ ] After install, run post-install validation and keep the report:

  ```bash
  sudo ./installer/validate/renos-validate.sh --post --json --report /tmp/renos-validation.json
  ```

  Exit codes: `0` = pass, `1` = warnings, `2` = failures. Do not proceed with
  failures.

- [ ] Confirm the API is alive:

  ```bash
  curl -s http://localhost:8443/api/v1/healthz   # {"status":"ok","version":...}
  curl -s http://localhost:8443/api/v1/readyz    # status "ok" when DB/Redis/NATS healthy
  ```

- [ ] Complete the first-run wizard (onboarding). The hub shows the wizard on
  first login; status is exposed via:

  ```bash
  curl -s -H "Authorization: Bearer $ADMIN_TOKEN" http://localhost:8443/api/v1/onboarding/status
  ```

## 1. Accounts & access control

- [ ] Change the default admin password immediately. The dev default is
      `admin@renos.local` / `admin`; production validation refuses weak
      passwords (see `RENOS_ADMIN_PASSWORD`).
- [ ] Create least-privilege users. Only a few admins; operators for daily
      work; viewers for read-only.
- [ ] Review role assignments:

  ```bash
  ./renctl/renctl login --server http://localhost:8443 --email <admin-email>
  ./renctl/renctl users list
  ./renctl/renctl roles list
  ./renctl/renctl roles assignments
  ```

- [ ] Confirm only admins can: seed demo data, create enrollment tokens,
      manage alert rules, access `/api/v1/debug/*`.

## 2. Secrets & keys

- [ ] `RENOS_JWT_SECRET` — random, at least 32 chars. `install.sh` generates
      one; do not reuse the dev default.
- [ ] `RENOS_ENCRYPTION_KEY` — random, at least 32 chars. Used for AES-256-GCM
      vault encryption.
- [ ] Both live in the env file / secret store only — never in git. The
      installer's pre-commit secret scanner and `renos-validate.sh --pre`
      check for leaked secrets.
- [ ] Store secrets in the encrypted vault, not in config or scripts:

  ```bash
  ./renctl/renctl vault list
  ./renctl/renctl vault keys
  ```

- [ ] Document a rotation schedule (recommended: quarterly). Rotating
      `RENOS_ENCRYPTION_KEY` requires re-encrypting vault entries — re-run
      the install or validate after rotation.

## 3. TLS & certificates

- [ ] `RENOS_EXTERNAL_URL` set to the public URL (required by
      `docker-compose.prod.yml`).
- [ ] HTTPS is terminated in front of the API. The bundled
      `deploy/nginx/nginx.conf` ships an HTTP listener and a commented HTTPS
      server block using `/etc/nginx/ssl/fullchain.pem` + `privkey.pem`.
      Uncomment it and drop your certificates there (auto-Provision via
      Let's Encrypt is documented in the installer guides).
- [ ] Certificate expiry is monitored. The health dashboard already includes
      a certificate check:

  ```bash
  curl -s -H "Authorization: Bearer $ADMIN_TOKEN" http://localhost:8443/api/v1/health/dashboard
  ```

- [ ] Internal agent mTLS: the CA lives in `RENOS_CA_CERT_DIR`. Agent
      certificates are issued with a 365-day TTL by default (hardcoded, not
      configurable via env); re-enroll agents before their certs expire.

## 4. Firewall & network

- [ ] Expose only what is needed:

  | Port | Service | Exposure |
  |------|---------|----------|
  | 80 / 443 | Nginx reverse proxy (web UI + API) | Public |
  | 8443 | Core API | Internal or behind proxy |
  | 8444 | Core gRPC | Internal |
  | 5432 | PostgreSQL | Internal only |
  | 6379 | Redis | Internal only |
  | 4222 | NATS | Internal only |

- [ ] `installer/linux/install.sh` opens 80, 443, 8443 via ufw / firewalld /
      iptables. Confirm the rule set:

  ```bash
  sudo ufw status verbose        # or: sudo firewall-cmd --list-all
  ```

- [ ] Redis and NATS are not reachable from the public interface. If you run
      the docker deployment, keep the containers on the internal network only
      (they already are by default).

## 5. Backups

- [ ] Take a full backup now (before going live):

  ```bash
  sudo ./installer/backup/renos-backup.sh --full
  ```

- [ ] Schedule backups. The legacy quick scripts also exist
      (`./scripts/backup.sh [dir]`); the supported production tool is
      `renos-backup.sh`. Typical cadence: daily `--db-only`, weekly `--full`.
- [ ] Enable encryption for off-site copies (uses age/gpg, see
      `renos-backup-config.sh`):

  ```bash
  sudo ./installer/backup/renos-backup.sh --full --encrypt --remote
  ```

- [ ] Set retention (default `RETENTION_COUNT=7` in
      `installer/backup/renos-backup-config.sh`) and prune:

  ```bash
  sudo ./installer/backup/renos-backup.sh --prune
  ```

- [ ] List backups to confirm they exist:

  ```bash
  sudo ./installer/backup/renos-backup.sh --list
  ```

## 6. Disaster recovery

- [ ] Do a restore **dry run** and keep the output:

  ```bash
  sudo ./installer/backup/renos-restore.sh --dry-run /var/lib/renos/backups/renos-backup-<ID>
  ```

- [ ] Test partial restores (database / config separately):

  ```bash
  sudo ./installer/backup/renos-restore.sh --partial database,config /var/lib/renos/backups/renos-backup-<ID>
  ```

- [ ] Rehearse a full restore in a scratch directory once per quarter:

  ```bash
  sudo ./installer/backup/renos-restore.sh --to-dir /tmp/restore-test /var/lib/renos/backups/renos-backup-<ID>
  ```

- [ ] Verify you can reconstruct the CA + device identities from a backup
      (agents re-enroll against a fresh CA).

## 7. Database

- [ ] PostgreSQL 16, on the internal network, with non-default credentials.
- [ ] Confirm connectivity and current DB:

  ```bash
  docker compose exec postgres psql -U renos -d renos -c "SELECT version();"
  # or, on a systemd install:
  systemctl status renos-postgres
  ```

- [ ] Migrations applied (production containers run `renos-core migrate`
      automatically):

  ```bash
  cd core && go run ./cmd/renos-core/ migrate
  ```

- [ ] DB is covered by `renos-backup.sh --db-only` on its own schedule.
- [ ] Query performance / index bloat tracked (the dashboard disk check
      includes data volumes).

## 8. Monitoring & health

- [ ] Health endpoints reachable and watched (the production compose
      healthchecks poll `/api/v1/healthz`):

  ```bash
  curl -s http://localhost:8443/api/v1/healthz
  curl -s http://localhost:8443/api/v1/readyz
  curl -s http://localhost:8443/api/v1/deephealth    # deep dependency check (default on)
  curl -s http://localhost:8443/api/v1/slos          # SLO summary (default on)
  ```

- [ ] Health dashboard is your first-stop panel. It scores health 0-100 and
      checks DB, Redis, NATS, certificates, backups, CPU, memory, and disk:

  ```bash
  curl -s -H "Authorization: Bearer $ADMIN_TOKEN" http://localhost:8443/api/v1/health/dashboard
  ```

- [ ] Prometheus scrape endpoint `/metrics` is reachable (exposed on the API
      port, default enabled) and the auth-required summary API
      `/api/v1/metrics` works:

  ```bash
  curl -s http://localhost:8443/metrics | head -20
  curl -s -H "Authorization: Bearer $ADMIN_TOKEN" http://localhost:8443/api/v1/metrics
  ```

- [ ] Enable Prometheus + Grafana if you operate at scale — the repo ships
      dashboards under `deploy/grafana/dashboards/` and alerts under
      `deploy/prometheus/alerts.yml`.

## 9. Alerting & notification

- [ ] Create alert rules for the metrics you care about:

  ```bash
  ./renctl/renctl alerts list
  ./renctl/renctl alerts summary
  ```

- [ ] Wire a webhook subscription to a chat/incident channel so rules have a
      delivery path:

  ```bash
  # Hub UI → Developer → Webhooks (or the API under /api/v1/webhooks)
  ```

- [ ] Test the whole path: trigger an alert, confirm acknowledge/resolve:

  ```bash
  ./renctl/renctl alerts active
  ./renctl/renctl alerts recent
  ```

- [ ] **Note:** email/SMTP notifications are **not yet available** (no SMTP
      config exists in Core). Alerts are delivered via webhooks only.

## 10. Logging & audit

- [ ] `RENOS_LOG_LEVEL` set (default `info`; `debug` in dev only).
- [ ] `RENOS_LOG_FILE` configured so logs survive container restarts.
- [ ] Audit log reviewed: the platform writes an audit trail (see the Audit
      Log viewer in the hub and `renctl/renctl alerts`-adjacent audit APIs).
- [ ] Crash reports are enabled by default (dir `RENOS_CRASH_DIR`, default
      `/var/lib/renos/crashes`). Confirm the directory is writable and
      inspect any dumps:

  ```bash
  ls -la /var/lib/renos/crashes
  ```

- [ ] Log retention: rotate Core/hub/nginx logs (logrotate or Docker's
      json-file rotation) and confirm your retention window.
- [ ] Decide on telemetry (opt-in, anonymous) — see section 18.

## 11. Rate limiting & DoS

- [ ] Review login limits — default 5/min per IP with burst 10
      (`RENOS_RATE_LIMIT_LOGIN`, `RENOS_RATE_LIMIT_LOGIN_BURST`).
- [ ] Review API limits — default 50/s burst 100
      (`RENOS_RATE_LIMIT_API`, `RENOS_RATE_LIMIT_API_BURST`).
- [ ] WebSocket connect limits — default 2/s burst 5
      (`RENOS_RATE_LIMIT_WS_CONNECT`, `RENOS_RATE_LIMIT_WS_CONNECT_BURST`).
- [ ] Nginx adds `limit_req` zones (api 30r/s, login 5r/m) — keep them tuned
      to your expected load.

## 12. Resource limits & capacity

- [ ] Docker deployment: `docker-compose.prod.yml` sets a 512M memory limit
      on Core. Review limits for all services.
- [ ] Disk: dashboard checks disk; also watch the data volume
      (`RENOS_DATA_DIR`, Docker named volumes `pgdata`, `redisdata`,
      `natsdata`, `coreca`, `agentdata`).
- [ ] Plan capacity for: DB size, Redis cache, NATS JetStream queues, and the
      backup directory.
- [ ] `RENOS_BACKUPS_DIR` (default `/var/lib/renos/backups`) has headroom for
      your retention window.

## 13. AI integration (optional)

- [ ] Choose a provider: Ollama (default, local) or OpenAI / Anthropic /
      Google / AIEndpoint:

  ```bash
  # Example: local Ollama
  RENOS_AI_PROVIDER=ollama
  RENOS_AI_OLLAMA_URL=http://localhost:11434
  RENOS_AI_OLLAMA_MODEL=llama3.2

  # Example: OpenAI
  RENOS_AI_PROVIDER=openai
  RENOS_AI_OPENAI_KEY=...
  RENOS_AI_OPENAI_MODEL=gpt-4o
  ```

- [ ] If AI is not used, keep the default provider and do not set any API
      keys. Confirm `/api/v1/ai/settings` requires a role.
- [ ] API keys are secrets — store them in the env file / secret manager, not
      in git.

## 14. Email / SMTP (not yet available)

- [ ] **Status: not yet implemented.** There is no SMTP configuration in
      RenOS Core or the hub, so no email notifications can be sent.
- [ ] Mitigation: use webhook subscriptions (section 9) for alert delivery,
      and rely on the audit log + health dashboard for passive visibility.

## 15. MFA / 2FA (not yet available)

- [ ] **Status: not yet implemented.** There is no TOTP/2FA/MFA support in
      the hub or Core.
- [ ] Mitigations while it is missing:
  - Strong, unique admin passwords (enforced at install).
  - Least-privilege RBAC (section 1).
  - Rate limiting on login (section 11).
  - Restrict access to the UI/API by network (section 4) — e.g., VPN or
    IP allow-list in front of Nginx.

## 16. Updates & upgrades

- [ ] Document your update policy (recommended: monthly patch cadence, test on
      a staging box first).
- [ ] Back up before every upgrade:

  ```bash
  sudo ./installer/backup/renos-backup.sh --full
  ```

- [ ] Use the upgrade manager (dry-run first):

  ```bash
  ./installer/upgrade/upgrade check
  ./installer/upgrade/upgrade status
  ./installer/upgrade/upgrade dry-run --component core --version <new-version>
  ./installer/upgrade/upgrade upgrade --component core --version <new-version>
  ```

- [ ] Know the rollback path:

  ```bash
  ./installer/upgrade/upgrade history
  ./installer/upgrade/upgrade rollback <upgrade-id>
  ```

- [ ] Docker installs can instead `docker compose -f docker-compose.prod.yml
      pull` then `up -d`; keep an image tag pinned, not `latest`, in
      production.

## 17. Optional: demo mode & example library

- [ ] Seed demo data **in staging only** (admin):

  ```bash
  curl -s -X POST -H "Authorization: Bearer $ADMIN_TOKEN" \
    http://localhost:8443/api/v1/demo/seed
  ```

- [ ] Import a starter pack from the example library (8 packs:
      docker-host, enterprise, gaming-server, home-lab, kubernetes-cluster,
      proxmox-cluster, raspberry-pi-fleet, small-business):

  ```bash
  ./examples/library/import.sh home-lab https://renos.local admin@renos.local
  ./examples/library/import.sh --dry-run --all   # preview without changes
  ```

## 18. Optional: telemetry

- [ ] Telemetry is **opt-in, anonymous, and off by default**. If you opt in,
      confirm what is sent (version, OS/arch, random instance id, counts of
      devices/workflows/plugins — no names, IPs, or secrets):

  ```bash
  RENOS_TELEMETRY_ENABLED=true
  RENOS_TELEMETRY_ENDPOINT=https://telemetry.renos.io/v1/ping
  ```

- [ ] If you stay opted out, record that decision and ensure
      `RENOS_TELEMETRY_ENABLED` is unset or `false`.

---

## Verification Command Reference

All commands below are verified against this repository.

### Authenticate with `renctl`

```bash
./renctl/renctl login --server http://localhost:8443 --email admin@renos.local
./renctl/renctl devices list
./renctl/renctl devices get <id>
./renctl/renctl jobs list
./renctl/renctl workflows list
./renctl/renctl plugins list
./renctl/renctl cluster dashboard
./renctl/renctl vault list
./renctl/renctl vault keys
./renctl/renctl users list
./renctl/renctl roles list
./renctl/renctl roles assignments
./renctl/renctl alerts list
./renctl/renctl alerts active
./renctl/renctl alerts recent
./renctl/renctl metrics
./renctl/renctl version
```

### Get a token for authenticated `curl`

```bash
TOKEN=$(curl -s -X POST http://localhost:8443/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@renos.local","password":"<your-password>"}' | jq -r '.token')
export ADMIN_TOKEN="$TOKEN"
```

### Health endpoints

```bash
curl -s http://localhost:8443/api/v1/healthz          # public
curl -s http://localhost:8443/api/v1/readyz           # public
curl -s http://localhost:8443/api/v1/deephealth       # public, default on
curl -s http://localhost:8443/api/v1/slos             # public, default on
curl -s -H "Authorization: Bearer $ADMIN_TOKEN" http://localhost:8443/api/v1/health/dashboard
curl -s -H "Authorization: Bearer $ADMIN_TOKEN" http://localhost:8443/api/v1/metrics
```

### Backups & restore

```bash
sudo ./installer/backup/renos-backup.sh --full
sudo ./installer/backup/renos-backup.sh --db-only --encrypt --remote
sudo ./installer/backup/renos-backup.sh --list
sudo ./installer/backup/renos-backup.sh --verify renos-backup-<ID>
sudo ./installer/backup/renos-backup.sh --prune
sudo ./installer/backup/renos-restore.sh --dry-run /var/lib/renos/backups/renos-backup-<ID>
sudo ./installer/backup/renos-restore.sh --partial database,config /var/lib/renos/backups/renos-backup-<ID>
sudo ./installer/backup/renos-restore.sh --to-dir /tmp/restore-test /var/lib/renos/backups/renos-backup-<ID>
```

### Validation & upgrade

```bash
sudo ./installer/validate/renos-validate.sh --pre
sudo ./installer/validate/renos-validate.sh --post --json --report /tmp/renos-validation.json
sudo ./installer/validate/renos-validate.sh --quick
./installer/upgrade/upgrade status
./installer/upgrade/upgrade dry-run --component core --version <new-version>
./installer/upgrade/upgrade rollback <upgrade-id>
```

### Misc

```bash
cd core && go run ./cmd/renos-core/ migrate       # apply migrations
cd core && go run ./cmd/renos-core/ version       # print core version
make build                                        # build core + hub + agent
```

---

## Key configuration reference

Verified against `core/internal/config/config.go`.

| Variable | Default | Description |
|---|---|---|
| `RENOS_JWT_SECRET` | dev default | JWT signing key (>= 32 chars; validated in prod) |
| `RENOS_ENCRYPTION_KEY` | dev default | Vault AES-256-GCM key (>= 32 chars; validated) |
| `RENOS_ADMIN_EMAIL` | `admin@renos.local` | Initial admin email |
| `RENOS_ADMIN_PASSWORD` | `admin` | Initial admin password (prod rejects weak) |
| `RENOS_EXTERNAL_URL` | `https://localhost:8443` | Public URL (required in prod compose) |
| `RENOS_PORT` | `8443` | HTTP API port |
| `RENOS_GRPC_PORT` | `8444` | gRPC port |
| `RENOS_DATABASE_URL` | local postgres | PostgreSQL connection string |
| `RENOS_NATS_URL` | `nats://localhost:4222` | NATS server |
| `RENOS_REDIS_URL` | `redis://localhost:6379/0` | Redis URL (empty = no Redis) |
| `RENOS_CA_CERT_DIR` | `/var/lib/renos/ca` | CA directory (cert TTL is fixed at 365 days) |
| `RENOS_RATE_LIMIT_LOGIN` / `_BURST` | `5` / `10` | Login rate limit per IP |
| `RENOS_RATE_LIMIT_API` / `_BURST` | `50` / `100` | API rate limit |
| `RENOS_RATE_LIMIT_WS_CONNECT` / `_BURST` | `2` / `5` | WebSocket connect limit |
| `RENOS_MAX_WS_CONNS_PER_USER` | `10` | Max WebSocket connections per user |
| `RENOS_MAX_BODY_SIZE` | `1048576` | Max request body size (bytes) |
| `RENOS_POOL_SIZE` | `5` | DB connection pool size |
| `RENOS_CRASH_DIR` | `/var/lib/renos/crashes` | Crash report directory |
| `RENOS_BACKUPS_DIR` | `/var/lib/renos/backups` | Backup output directory |
| `RENOS_LOG_LEVEL` | `info` | debug / info / warn / error |
| `RENOS_LOG_FILE` | — | Log file path |
| `RENOS_METRICS_ENABLED` / `RENOS_METRICS_PATH` | `true` / `/metrics` | Metrics endpoint |
| `RENOS_SLO_ENABLED` / `RENOS_SLO_PATH` | `true` / `/api/v1/slos` | SLO endpoint |
| `RENOS_DEEP_HEALTH_ENABLED` / `RENOS_DEEP_HEALTH_PATH` | `true` / `/api/v1/deephealth` | Deep health endpoint |
| `RENOS_TELEMETRY_ENABLED` / `_ENDPOINT` | `false` / `https://telemetry.renos.io/v1/ping` | Opt-in telemetry |
| `RENOS_TRACING_ENABLED` / `_EXPORTER` / `_ENDPOINT` / `_SAMPLE_RATE` | `false` / `otlp` / `http://localhost:4318` / `0.1` | OpenTelemetry tracing |
| `RENOS_AI_PROVIDER` | `ollama` | AI provider |
| `RENOS_AI_*` | various | Ollama / OpenAI / Anthropic / Google / AIEndpoint keys & models |
