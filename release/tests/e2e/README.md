# RenOS End-to-End QA Suite

End-to-end verification of the RenOS user journey. The suite talks to the real
`core` API over HTTP; nothing is mocked in the run — a phase that cannot reach
its service records `SKIP`, never a fake `PASS`.

## Layout

```
release/tests/e2e/
├── run-e2e.sh          entrypoint: flags, plan, report, exit code
├── lib.sh              shared lib (api_req, login, assertions, wait_for_health, report)
├── phases/             one script per phase
│   ├── install.sh      provision/detect backend (compose, project renos-e2e)
│   ├── health.sh       /healthz /readyz /livez /startupz + fleet dashboard
│   ├── login.sh        POST /api/v1/auth/login -> bearer token
│   ├── enroll.sh       enroll-token + RSA keypair + enroll + connect + list
│   ├── command.sh      device command submit, agent result, jobs
│   ├── workflow.sh     create workflow + step + execute + list
│   ├── terminal.sh     terminal session listing endpoints
│   ├── plugin.sh       marketplace / plugin-platform + best-effort install
│   ├── ai.sh           conversations / settings / chat (graceful)
│   ├── backup.sh       real renos-backup.sh -> manifest validation
│   ├── restore.sh      manifest + archive integrity (root-gated dry-run)
│   ├── upgrade.sh      verify-upgrade.sh + update API endpoints
│   └── rollback.sh     verify-rollback.sh + rollback API endpoints
└── results/            step-by-step results files (created at runtime)
```

## Usage

```bash
./run-e2e.sh                       # full run
./run-e2e.sh --quick               # skip install/backup/restore + standalone suites
./run-e2e.sh --setup-only          # provision backend, verify health, then exit
./run-e2e.sh --skip-install        # assume a backend is already running
./run-e2e.sh --cleanup             # remove e2e temp artifacts
./run-e2e.sh --report e2e-report.txt
./run-e2e.sh --help
```

## Configuration

| Env var          | Default                | Purpose                        |
|------------------|------------------------|--------------------------------|
| `RENOS_URL`      | `http://localhost:8443`| API base URL                   |
| `ADMIN_EMAIL`    | `admin@renos.local`    | login email                    |
| `ADMIN_PASSWORD` | `admin`                | login password                 |
| `HEALTH_TIMEOUT` | `90`                   | seconds to wait for healthz    |
| `E2E_COMMAND`    | `apt update`           | command sent in the command phase (must pass default policy allowlist) |

## Guarantees

- Every endpoint exercised is a real route in `core/internal/api/router.go`.
- Scripts run with `set -uo pipefail` (never `set -e`), so a failing curl or a
  network hiccup never aborts the run silently.
- Each phase records exactly one outcome: `PASS`, `FAIL`, or `SKIP`. SKIP is
  non-fatal (service unavailable, non-root restore, empty marketplace).
- Exit code is `0` only if every step PASSed.
- The AI phase never asserts on LLM output: it asserts the endpoint responded,
  tolerating error codes when the provider is unconfigured.
- The command phase verifies both sides of the default policy: a non-allowlisted
  command must be rejected with `403 POLICY_DENIED` (policy seeded by migration
  `000027_policies`), and `E2E_COMMAND` (default `apt update`, which the default
  allowlist permits) must be accepted with `201`. Set `E2E_COMMAND` to a command
  your deployment's allowlist permits.
- The real `renos-restore.sh` requires root; as non-root, the restore phase
  validates the manifest and archive integrity instead, and dry-runs the
  restore script only when run as root.

## Notes

- Agent-facing POSTs (`/devices/{id}/connect`, `/devices/{id}/heartbeat`,
  `/devices/enroll`, `/commands/{id}/result`) are exempt from the CSRF
  middleware, so no token is needed for them; user-facing requests carry the
  bearer token once `login` succeeds.
- `install.sh` only provisions a backend when none is reachable; it never
  touches an existing stack.
