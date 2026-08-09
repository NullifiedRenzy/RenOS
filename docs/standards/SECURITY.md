# Security

Authenticate, authorize, audit every action. Avoid arbitrary shell execution.

## Security Measures

### Authentication & Authorization
- JWT-based authentication with role-based access control (RBAC)
- Admin-only endpoints for deploy, metrics, AI settings
- Pre-commit hooks scan for secrets before commit
- Production validation blocks empty/default secrets

### Input Validation
- SQL injection prevention via parameterized queries
- Path traversal blocked in file transfers (`..` and null bytes rejected)
- Command policy blocks destructive operations (`rm -rf /`, `mkfs`, fork bombs)
- Pagination limits: max 200 items per page, max 10,000 offset

### Encryption
- Vault secrets encrypted at rest with AES-256-GCM
- Key derivation via SHA-256 hash
- TLS for all agent communication (mTLS)

### Web Security
- Content Security Policy: `script-src 'self'` (no unsafe-eval, no unsafe-inline)
- CSRF protection on all state-changing requests (POST/PUT/PATCH/DELETE)
- Method override restricted to safe methods (GET/HEAD/OPTIONS only)
- WebSocket origin restricted to `localhost:PORT`
- Security headers: HSTS, X-Frame-Options, X-Content-Type-Options

### Error Handling
- Internal errors sanitized — no paths, DB details, or connection info leaked
- Generic error messages for authentication/authorization failures

### API Key Handling
- Google API keys sent via header, not URL query parameters
- API tokens use `crypto/rand` for generation (not `math/rand`)
- Terminal session tokens hidden from JSON serialization

### Resource Limits
- Rate limiter bounded (max 10,000 clients) with LRU eviction
- Enrollment tokens bounded (max 1,000) with expired token cleanup
- File transfer cleanup runs every 5 minutes

## Behavior Changes (v1.0.0-rc2)

### CSRF Protection
All POST/PUT/PATCH/DELETE requests require one of:
- `Origin` header (for browser requests)
- `X-Requested-With: XMLHttpRequest` header
- `Authorization: Bearer <token>` header

Agent endpoints (device enrollment, heartbeats, command results) are exempt.

### CSP Hardening
`script-src` now only allows `'self'`. This means:
- Inline `<script>` tags are blocked
- `eval()` and `new Function()` are blocked
- External scripts must be loaded from the same origin

### Method Override
`X-HTTP-Method-Override` header only allows GET, HEAD, or OPTIONS.
Previously any method could be overridden, including DELETE and PUT.

### WebSocket Connections
Origin must be `localhost:PORT` format. Previously `*.renos.local` was allowed.
