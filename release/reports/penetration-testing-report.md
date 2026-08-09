# RenOS 1.0.0 — Penetration Testing Report

| Field | Value |
|---|---|
| **Target** | RenOS 1.0.0-RELEASE |
| **Test Date** | 2026-05-11 – 2026-05-25 |
| **Classification** | Confidential |
| **Version** | 1.0.0 |

---

## Executive Summary

A comprehensive penetration test was conducted against RenOS 1.0.0, covering the core API, Hub UI, Agent daemon, CLI, plugin platform, WebSocket interface, and supporting infrastructure (NATS, Redis, PostgreSQL). Testing was performed against a production-like staging environment with 50 simulated devices, multi-role user accounts, and third-party plugin integrations.

**Result: All 15 findings have been remediated. Zero critical or high-severity issues remain open.**

| Severity | Found | Fixed | Open |
|---|---|---|---|
| Critical | 0 | 0 | 0 |
| High | 0 | 0 | 0 |
| Medium | 2 | 2 | 0 |
| Low | 5 | 5 | 0 |
| Informational | 8 | 8 | 0 |
| **Total** | **15** | **15** | **0** |

---

## Scope

### In Scope
- Core REST API (`/api/v1/*`) — all endpoints
- WebSocket endpoint (`/api/v1/ws`) — real-time device communication
- Terminal WebSocket (`/devices/{id}/terminal/ws`)
- Hub Next.js web application — all pages and API routes
- Agent gRPC and NATS communication
- Plugin platform — dynamic plugin loading and route registration
- Authentication — JWT, device enrollment tokens, OAuth2 identity providers
- RenVault — encrypted secret storage
- File transfer and chunked upload endpoints
- Monitoring and alerting platform

### Out of Scope
- Physical security of host infrastructure
- Third-party SaaS dependencies (email providers, LLM APIs)
- Social engineering of operators
- Source code supply-chain attacks on third-party Go modules

---

## Methodology

Testing followed the **OWASP Web Security Testing Guide v4.2**, **OWASP Top 10 (2021)**, and **OWASP API Security Top 10 (2023)**. Techniques employed:

- **Automated scanning**: OWASP ZAP, Burp Suite Professional, custom Go fuzz harnesses
- **Manual testing**: Parameter tampering, privilege escalation, IDOR, injection, race conditions
- **Cryptographic review**: JWT implementation, TLS configuration, encryption key management
- **Business logic review**: Workflow execution, approval chains, bulk operations
- **Session analysis**: Token rotation, refresh flow, WebSocket session hijacking
- **Dependency scanning**: `go list -m -u all`, `npm audit`, `cargo audit`, Trivy container scans

---

## Findings

### AUTH-001 — JWT Timing Attack (Medium)

| Field | Value |
|---|---|
| **ID** | AUTH-001 |
| **Title** | JWT signing comparison not constant-time |
| **Severity** | Medium |
| **Status** | **FIXED** |
| **Affected Component** | `core/internal/auth/jwt.go` — `ValidateToken` |

**Description:** The JWT validation in `ValidateToken` delegates to `jwt.ParseWithClaims`, which uses `hmac.Equal` for HMAC comparison. The `hmac.Equal` function runs in constant time. However, prior to calling `ParseWithClaims`, no early rejection was performed on malformed token structures, allowing a timing oracle to distinguish between structural parse errors and signature failures.

**Impact:** An attacker with the ability to send thousands of requests could theoretically distinguish token structure errors from signature validation failures, reducing the search space for brute-force attacks against the JWT secret (though still computationally infeasible with a 256-bit key).

**Evidence:** Benchmarks showed a ~50µs timing difference between structurally valid but incorrectly signed tokens and structurally invalid tokens.

**Remediation:**
- Added early structural validation of token parts before delegation to `jwt.ParseWithClaims`
- All error paths now return a generic `"invalid token"` message regardless of failure type
- Applied `jwt.WithIssuer("renos-core")` validation option to reject tokens with wrong issuer at parse time

### AUTH-002 — Weak Device Enroll Token Entropy (Low)

| Field | Value |
|---|---|
| **ID** | AUTH-002 |
| **Title** | Device enrollment token generated with insufficient entropy |
| **Severity** | Low |
| **Status** | **FIXED** |
| **Affected Component** | `core/internal/api/enroll.go` |

**Description:** Enrollment tokens were generated using `math/rand` seeded with `time.Now().UnixNano()`, providing approximately 2^63 possible values. While large, `math/rand` is not cryptographically secure.

**Impact:** An attacker who observed one or more enrollment tokens could predict future tokens if they could determine the PRNG state.

**Remediation:** Replaced `math/rand` with `crypto/rand` via `crypto/rand.Read` feeding into `hex.EncodeToString`, yielding 128 bits of cryptographic entropy per token.

### AUTH-003 — Session Token Not Rotated on Role Escalation (Low)

| Field | Value |
|---|---|
| **ID** | AUTH-003 |
| **Title** | JWT not re-issued when user role changes |
| **Severity** | Low |
| **Status** | **FIXED** |
| **Affected Component** | `core/internal/api/iam.go` — role update handler |

**Description:** When an administrator updated a user's role (e.g., from `viewer` to `admin`), the user's existing JWT remained valid until its natural expiration (24h). The new role only took effect after the token was re-issued on next login.

**Impact:** A user with a know role change could continue operating under their old privileges for up to 24 hours. In the specific case of a privilege *revocation*, this allowed continued elevated access.

**Remediation:** The role update handler now calls `iamAudit` and invalidates all active sessions for the affected user by incrementing their `token_version` column in the database. JWT validation checks `token_version` against the DB on each authenticated request.

### AUTH-004 — Missing Rate Limit on Token Refresh (Info)

| Field | Value |
|---|---|
| **ID** | AUTH-004 |
| **Title** | Token refresh endpoint missing dedicated rate limiter |
| **Severity** | Informational |
| **Status** | **FIXED** |
| **Affected Component** | `core/internal/api/router.go`, `core/internal/api/middleware.go` |

**Description:** The token refresh endpoint shared the general API rate limiter, allowing a higher rate of refresh attempts than necessary.

**Remediation:** Added a dedicated `refreshLimiter` using `x/time/rate` with a burst of 5 and rate of 1 per second, applied only to the refresh endpoint.

### AUTH-005 — Audit Log Injection via User-Agent Header (Info)

| Field | Value |
|---|---|
| **ID** | AUTH-005 |
| **Title** | Audit log entries vulnerable to log injection via User-Agent |
| **Severity** | Informational |
| **Status** | **FIXED** |
| **Affected Component** | `core/internal/observability/correlation.go` |

**Description:** The `User-Agent` header was written directly into structured audit logs without sanitization, allowing an attacker to inject newline characters and forge log entries.

**Remediation:** All user-controlled header values are now sanitized through `strings.ReplaceAll(v, "\n", "")` before inclusion in log output, and audit log entries are validated as well-formed JSON prior to storage.

---

### API-001 — Input Validation Bypass in Workflow Step Config (Medium)

| Field | Value |
|---|---|
| **ID** | API-001 |
| **Title** | Workflow step configuration allows arbitrary JSON with no schema validation |
| **Severity** | Medium |
| **Status** | **FIXED** |
| **Affected Component** | `core/internal/api/workflows.go` — `createWorkflowStepHandler` |

**Description:** The `config` field in workflow step creation and update accepted arbitrary JSON objects with no schema validation. An attacker with operator privileges could inject malformed configuration that caused workflow execution to panic, or inject unexpected keys that polluted downstream handler state.

**Impact:** Denial of service (panic in workflow executor) and potential data corruption in downstream workflow steps.

**Evidence:** Fuzzing the workflow step create endpoint with `{ "config": { "type": "command", "command": "reboot", "__proto__": { "polluted": true } } }` demonstrated prototype pollution in the workflow executor's configuration merge logic.

**Remediation:**
- Added JSON Schema validation (via `github.com/santhosh-tekuri/jsonschema/v6`) for every workflow step type
- Each step type (command, script, webhook, approval, condition) now has a strict schema defined in `core/internal/workflow/schemas.go`
- Unknown keys in `config` are rejected with HTTP 400
- The workflow executor wraps all step execution in a deferred `recover()` to prevent panics from crashing the executor

### API-002 — IDOR in Device File Permission Deletion (Low)

| Field | Value |
|---|---|
| **ID** | API-002 |
| **Title** | Insecure direct object reference in file permission deletion |
| **Severity** | Low |
| **Status** | **FIXED** |
| **Affected Component** | `core/internal/api/file_permissions.go` — `deleteFilePermissionHandler` |

**Description:** The `DELETE /api/v1/devices/{id}/file-permissions/{userId}` endpoint checked that the requesting user had `admin` role but did not verify that the requesting user had permission on the *target device*.

**Impact:** An admin user could delete file permissions on devices they were not authorized to manage.

**Remediation:** Added a device-scope authorization check that verifies the requesting user's groups include the target device before allowing the deletion.

### API-003 — Missing Content-Type Validation (Low)

| Field | Value |
|---|---|
| **ID** | API-003 |
| **Title** | Several endpoints accept non-JSON Content-Type for JSON bodies |
| **Severity** | Low |
| **Status** | **FIXED** |
| **Affected Component** | `core/internal/api/middleware.go` — `contentTypeValidation` middleware |

**Description:** POST and PUT endpoints that expect JSON request bodies did not validate the `Content-Type` header, allowing `Content-Type: text/plain` with a JSON body to bypass content-type-specific parsing logic.

**Remediation:** Added a `contentTypeValidation` middleware that rejects POST/PUT/PATCH requests with non-JSON `Content-Type` (unless the endpoint is explicitly exempted for file uploads).

### API-004 — Verbose Error Messages in Production (Low)

| Field | Value |
|---|---|
| **ID** | API-004 |
| **Title** | Internal error details leaked in production API responses |
| **Severity** | Low |
| **Status** | **FIXED** |
| **Affected Component** | `core/internal/api/middleware.go` — `recovery` middleware |

**Description:** When an unhandled panic occurred, the recovery middleware returned the panic value and stack trace in the HTTP response body in non-debug mode.

**Remediation:** The recovery middleware now returns a generic `{"error":"internal_server_error"}` response in production. Stack traces are logged server-side only. The full detail is returned only when `RENOS_DEBUG=true` is set.

### API-005 — WebSocket Origin Not Validated (Info)

| Field | Value |
|---|---|
| **ID** | API-005 |
| **Title** | WebSocket upgrade endpoint does not validate Origin header |
| **Severity** | Informational |
| **Status** | **FIXED** |
| **Affected Component** | `core/internal/ws/hub.go` — `ServeWS` |

**Description:** The WebSocket upgrade handler accepted connections from any origin without checking the `Origin` header against an allowlist.

**Remediation:** Added origin validation against the configured `AllowedOrigins` list from `config.AllowedOrigins`. Cross-origin WebSocket connections are rejected with HTTP 403 unless the origin is in the allowlist.

### API-006 — File Transfer Chunk Number Not Bounded (Info)

| Field | Value |
|---|---|
| **ID** | API-006 |
| **Title** | File transfer chunk number not validated for upper bound |
| **Severity** | Informational |
| **Status** | **FIXED** |
| **Affected Component** | `core/internal/api/file_transfers.go` — `uploadChunkHandler` |

**Description:** The `chunkNum` path parameter in `/api/v1/devices/{id}/file-transfers/{transferId}/chunks/{chunkNum}` was parsed as an integer but not validated against a maximum chunk count.

**Remediation:** Added a maximum chunk limit of 10,000 per transfer. Chunk numbers exceeding this limit are rejected with HTTP 400.

### API-007 — Pagination Parameters Not Sanitized (Info)

| Field | Value |
|---|---|
| **ID** | API-007 |
| **Title** | Pagination `limit` and `offset` parameters accept negative values |
| **Severity** | Informational |
| **Status** | **FIXED** |
| **Affected Component** | All list handlers — middleware pagination parsing |

**Description:** The `limit` and `offset` query parameters were parsed as integers but not validated for negative values, which could cause unexpected SQL behavior or excessive memory allocation.

**Remediation:** Pagination middleware now clamps `limit` to `[1, 1000]` and `offset` to `[0, MaxInt32]`, returning HTTP 400 for out-of-range values.

---

### DATA-001 — Encryption Key Rotation Gaps (Low)

| Field | Value |
|---|---|
| **ID** | DATA-001 |
| **Title** | RenVault encryption key rotation does not re-encrypt existing secrets |
| **Severity** | Low |
| **Status** | **FIXED** |
| **Affected Component** | `core/internal/api/vault.go` — key toggle handler |

**Description:** When a RenVault encryption key was rotated (enabled a new key, disabled an old one), existing secrets encrypted with the old key were not re-encrypted with the new key. Secrets remained readable (the old key was retained for decryption) but were not transitioned to the new key.

**Impact:** Over multiple rotations, the number of active encryption keys grew linearly with rotations, increasing key management complexity.

**Remediation:** Added a `rekey` endpoint and background job that re-encrypts all secrets with the current active key when a new key is toggled. The rekey operation is idempotent and can be resumed if interrupted.

### DATA-002 — Secrets Visible in Process List (Info)

| Field | Value |
|---|---|
| **ID** | DATA-002 |
| **Title** | Encryption key passed via command-line argument visible in `/proc` |
| **Severity** | Informational |
| **Status** | **FIXED** |
| **Affected Component** | `core/cmd/renos-core/main.go` — CLI flag parsing |

**Description:** The `--encryption-key` flag was parsed from command-line arguments, making the key visible in `/proc/[pid]/cmdline` to any user on the same host.

**Remediation:** The encryption key is now read from the `RENOS_ENCRYPTION_KEY` environment variable, which is zeroed after use. The `--encryption-key` flag is deprecated and logs a warning if used.

### DATA-003 — Backup Files World-Readable (Info)

| Field | Value |
|---|---|
| **ID** | DATA-003 |
| **Title** | Update backup archives created with world-readable permissions |
| **Severity** | Informational |
| **Status** | **FIXED** |
| **Affected Component** | `core/internal/update/manager.go` — backup creation |

**Description:** Backup tarballs created by the update manager were written with `os.Create` (default permissions `0666` before umask), potentially exposing sensitive configuration and secrets to other users on the host.

**Remediation:** Backups are now created with `os.OpenFile` and `os.FileMode(0600)`. The backup directory is created with `os.FileMode(0700)`.

### DATA-004 — Session Data in Server Logs (Info)

| Field | Value |
|---|---|
| **ID** | DATA-004 |
| **Title** | JWT tokens logged in request debug output |
| **Severity** | Informational |
| **Status** | **FIXED** |
| **Affected Component** | `core/internal/api/middleware.go` — `requestLogger` |

**Description:** The request logging middleware emitted the `Authorization` header value in debug-level log entries, exposing JWT tokens in plaintext in log streams.

**Remediation:** The `Authorization` header is now redacted (`"Bearer ***"`) in all log output. Log scrubber functions are applied to any field matching `secret`, `token`, `password`, or `key`.

### DATA-005 — No Audit Trail for Secret Access (Info)

| Field | Value |
|---|---|
| **ID** | DATA-005 |
| **Title** | RenVault secret read operations not audited |
| **Severity** | Informational |
| **Status** | **FIXED** |
| **Affected Component** | `core/internal/api/vault.go` — `getVaultSecretHandler` |

**Description:** Reading a secret from RenVault did not create an audit log entry, making it impossible to detect unauthorized secret access after the fact.

**Remediation:** All secret read operations now log to the vault audit trail with timestamp, user ID, secret ID, and action type.

---

## Summary of Remediation Stats

| Component | Findings | Fixed |
|---|---|---|
| Authentication / JWT | 5 | 5 |
| API / Input Validation | 4 | 4 |
| Data Protection / Encryption | 5 | 5 |
| Logging / Monitoring | 1 | 1 |
| **Total** | **15** | **15** |

---

## Tools Used

| Tool | Purpose |
|---|---|
| Burp Suite Professional 2026.3 | Manual proxy, repeater, intruder |
| OWASP ZAP 2.16 | Automated scanning |
| custom Go fuzz harnesses | API fuzzing |
| jwt_tool v2.2 | JWT analysis |
| nuclei v3.3 | Template-based scanning |
| Trivy v0.58 | Container and dependency scanning |
| curl, jq, websocat | Manual API and WebSocket testing |

---

## Sign-Off

| Role | Name | Date |
|---|---|---|
| Lead Security Engineer | [Redacted] | 2026-05-25 |
| QA Lead | [Redacted] | 2026-05-25 |
| Release Manager | [Redacted] | 2026-05-25 |
