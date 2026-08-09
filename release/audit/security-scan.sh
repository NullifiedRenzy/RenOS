#!/usr/bin/env bash
# ============================================================================
# RenOS Security Scan
# ============================================================================
# Runs static analysis, secret scanning, and security linting across the
# entire RenOS codebase. Generates a comprehensive security report.
#
# Exit codes:
#   0 - All clean, no issues
#   1 - Warnings (minor findings, missing tools, etc.)
#   2 - Critical issues found (vulnerabilities, secrets leaked)
# ============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
REPORT_DIR="${PROJECT_DIR}/release/reports"
REPORT_FILE="${REPORT_DIR}/security-scan.md"

GO_MODULES=(
  "${PROJECT_DIR}/core"
  "${PROJECT_DIR}/renctl"
  "${PROJECT_DIR}/sdk/go"
  "${PROJECT_DIR}/installer/upgrade"
  "${PROJECT_DIR}/installer/wizard"
  "${PROJECT_DIR}/core/cmd/renos-plugin"
)

NPM_DIR="${PROJECT_DIR}/hub"
CARGO_DIR="${PROJECT_DIR}/agent"

mkdir -p "$REPORT_DIR"

# --------------------------------------------------------------------------
# Helpers
# --------------------------------------------------------------------------

log()  { printf '[SEC SCAN] %s\n' "$*"; }
warn() { printf '[SEC SCAN] WARNING: %s\n' "$*" >&2; }
err()  { printf '[SEC SCAN] ERROR: %s\n' "$*" >&2; }

EXIT_CODE=0
TOTAL_CRITICAL=0
TOTAL_WARNINGS=0

detect_tool() {
  command -v "$1" >/dev/null 2>&1
}

write_section() {
  local heading="$1"
  local content="$2"
  printf '\n## %s\n\n%s\n' "$heading" "$content" >> "$REPORT_FILE"
}

record_finding() {
  local severity="$1"   # CRITICAL or WARNING
  local tool="$2"
  local file="$3"
  local line="$4"
  local desc="$5"
  {
    printf '| %s | %s | %s | %s | %s |\n' "$severity" "$tool" "$file" "$line" "$desc"
  } >> "$REPORT_FILE"
  if [ "$severity" = "CRITICAL" ]; then
    TOTAL_CRITICAL=$((TOTAL_CRITICAL + 1))
    EXIT_CODE=2
  else
    TOTAL_WARNINGS=$((TOTAL_WARNINGS + 1))
    [ "$EXIT_CODE" -eq 0 ] && EXIT_CODE=1
  fi
}

# --------------------------------------------------------------------------
# Begin report
# --------------------------------------------------------------------------

TIMESTAMP="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
HOSTNAME="$(hostname 2>/dev/null || echo 'unknown')"

cat > "$REPORT_FILE" <<EOF
# RenOS Security Scan Report

**Generated:** $TIMESTAMP
**Host:** $HOSTNAME

EOF

# ============================================================================
# 1. Go Static Analysis (gosec)
# ============================================================================

printf '\n## Go Security Analysis\n\n' >> "$REPORT_FILE"

GO_SEC_ISSUES=0

if detect_tool gosec; then
  log "Running gosec on Go modules"
  {
    printf '| Severity | Tool | File | Line | Description |\n'
    printf '|---|---|---|---|---|\n'
  } >> "$REPORT_FILE"

  for mod_dir in "${GO_MODULES[@]}"; do
    if [ ! -f "$mod_dir/go.mod" ]; then
      continue
    fi
    module_name="$(head -1 "$mod_dir/go.mod" | awk '{print $2}' 2>/dev/null || echo "unknown")"
    log "  Scanning Go module: $module_name"
    gosec_out="$(cd "$mod_dir" && gosec -fmt=json ./... 2>/dev/null)" || true
    if [ -n "$gosec_out" ]; then
      # Parse gosec JSON output
      echo "$gosec_out" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    for issue in data.get('Issues', []):
        sev = issue.get('Severity', 'WARNING').upper()
        file = issue.get('File', '')
        line = str(issue.get('Line', 0))
        desc = issue.get('Details', issue.get('What', '')).replace('|', '/')
        cwe = issue.get('CWE', {})
        if isinstance(cwe, dict):
            cwe_id = cwe.get('ID', '')
        else:
            cwe_id = str(cwe)
        if cwe_id:
            desc = f'{desc} (CWE-{cwe_id})'
        print(f'{sev}|gosec|{file}|{line}|{desc}')
except Exception as e:
    sys.stderr.write(f'Failed to parse gosec output: {e}')
" 2>/dev/null | while IFS='|' read -r sev tool file line desc; do
        record_finding "$sev" "$tool" "$file" "$line" "$desc"
        GO_SEC_ISSUES=$((GO_SEC_ISSUES + 1))
      done
    fi
  done

  if [ "$GO_SEC_ISSUES" -eq 0 ]; then
    printf '| - | - | - | - | No issues found |\n' >> "$REPORT_FILE"
  fi
else
  warn "gosec not installed — skipping Go security scan"
  write_section "Go Security Analysis" "_gosec is not installed._\n\nInstall with: \`go install github.com/securego/gosec/v2/cmd/gosec@latest\`\n\nWould scan all Go packages for:\n- SQL injection vectors\n- Hardcoded credentials\n- Unsafe file permissions\n- TLS/SSL misconfigurations\n- Command injection\n- Insecure random number usage\n"
fi

# ============================================================================
# 2. Secret Scanning (gitleaks / trufflehog equivalent)
# ============================================================================

printf '\n## Secret Scanning\n\n' >> "$REPORT_FILE"

SECRETS_FOUND=0

# Try gitleaks first, then trufflehog, then fallback to grep-based scan
if detect_tool gitleaks; then
  log "Running gitleaks on repository"
  {
    printf '| Severity | Tool | File | Line | Description |\n'
    printf '|---|---|---|---|---|\n'
  } >> "$REPORT_FILE"

  gitleaks_out="$(gitleaks detect --source "$PROJECT_DIR" --no-git -v 2>/dev/null)" || true
  if echo "$gitleaks_out" | grep -qi 'leak\|secret\|finding'; then
    echo "$gitleaks_out" | while IFS= read -r line; do
      if echo "$line" | grep -qi 'leak\|finding'; then
        file="$(echo "$line" | grep -oP '(?<=File: )\S+' || echo "?")"
        desc="$(echo "$line" | grep -oP '(?<=Description: ).*' || echo "Potential secret")"
        record_finding "CRITICAL" "gitleaks" "$file" "?" "$desc"
        SECRETS_FOUND=$((SECRETS_FOUND + 1))
      fi
    done
  fi

  if [ "$SECRETS_FOUND" -eq 0 ]; then
    printf '| - | - | - | - | No secrets detected |\n' >> "$REPORT_FILE"
  fi

elif detect_tool trufflehog; then
  log "Running trufflehog on repository"
  {
    printf '| Severity | Tool | File | Line | Description |\n'
    printf '|---|---|---|---|---|\n'
  } >> "$REPORT_FILE"

  truffle_out="$(trufflehog filesystem "$PROJECT_DIR" --no-update 2>/dev/null)" || true
  if [ -n "$truffle_out" ]; then
    echo "$truffle_out" | while IFS= read -r line; do
      file="$(echo "$line" | grep -oP '(?<=File: )\S+' || echo "?")"
      desc="$(echo "$line" | grep -oP '(?<=Reason: ).*' || echo "Potential secret")"
      record_finding "CRITICAL" "trufflehog" "$file" "?" "$desc"
      SECRETS_FOUND=$((SECRETS_FOUND + 1))
    done
  fi

  if [ "$SECRETS_FOUND" -eq 0 ]; then
    printf '| - | - | - | - | No secrets detected |\n' >> "$REPORT_FILE"
  fi

else
  warn "Neither gitleaks nor trufflehog installed — falling back to regex-based secret scan"
  write_section "Secret Scanning" "_gitleaks nor trufflehog are installed._\n\nInstall with:\n  \`brew install gitleaks\` or \`go install github.com/gitleaks/gitleaks/v8@latest\`\n  \`pip install trufflehog\`\n\nFalling back to regex-based scan for common secret patterns.\n"

  # Regex-based secret scan
  {
    printf '| Severity | Tool | File | Line | Description |\n'
    printf '|---|---|---|---|---|\n'
  } >> "$REPORT_FILE"

  # Patterns to scan for
  SECRET_PATTERNS=(
    '(?i)AKIA[0-9A-Z]{16}'                          # AWS Access Key
    '(?i)-----BEGIN\s?(RSA|DSA|EC|PGP|OPENSSH)\s?PRIVATE KEY-----'  # Private keys
    '(?i)(ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9_]{36,}'  # GitHub tokens
    '(?i)sk_live_[0-9a-z]{32}'                      # Stripe live keys
    '(?i)pk_live_[0-9a-z]{32}'                      # Stripe live publishable
    '(?i)xox[baprs]-[0-9a-z-]{24}'                  # Slack tokens
    '(?i)sqlite.*://.*:.*@'                          # SQL connection strings with creds
    '(?i)postgres.*://.*:.*@'                        # Postgres connection strings with creds
    '(?i)mongodb.*://.*:.*@'                         # MongoDB connection strings with creds
    '(?i)redis.*://.*:.*@'                           # Redis connection strings with creds
    '(?i)-----BEGIN CERTIFICATE-----'                # Certificates in code
  )

  found_any=false
  for pattern in "${SECRET_PATTERNS[@]}"; do
    while IFS=':' read -r file line content; do
      [ -z "$file" ] && continue
      # Skip binary/generated files and common false positives
      case "$file" in
        *.test.go|*_test.go|*_test.tsx|*_test.ts|*_test.js|*.test.tsx|*.snap|*package-lock.json|*Cargo.lock|*go.sum|*.sum|*.pdf|*.png|*.jpg|*.svg|*.ico) continue ;;
      esac
      # Skip lines in .git directory
      if echo "$file" | grep -q '/\.git/'; then
        continue
      fi
      # Skip test fixtures
      if echo "$file" | grep -qi 'test.*fixture\|mock\|stub'; then
        continue
      fi
      desc="Potential secret matched pattern: ${pattern}"
      record_finding "CRITICAL" "regex-scan" "$file" "$line" "$desc"
      SECRETS_FOUND=$((SECRETS_FOUND + 1))
      found_any=true
    done < <(rg -n --no-heading "$pattern" "$PROJECT_DIR" 2>/dev/null || true)
  done

  if ! $found_any; then
    printf '| - | - | - | - | No secrets detected via regex |\n' >> "$REPORT_FILE"
  fi
fi

# ============================================================================
# 3. JavaScript/TypeScript Security Linting (eslint)
# ============================================================================

printf '\n## JavaScript / TypeScript Security Analysis\n\n' >> "$REPORT_FILE"

JS_SEC_ISSUES=0

if [ -f "$NPM_DIR/package.json" ]; then
  if detect_tool npx; then
    log "Running eslint on hub/"
    # Check if eslint is configured
    if [ -f "$NPM_DIR/.eslintrc*" ] || [ -f "$NPM_DIR/.eslintrc.json" ] || [ -f "$NPM_DIR/.eslintrc.js" ] || [ -f "$NPM_DIR/eslint.config.js" ] || grep -q '"eslint"' "$NPM_DIR/package.json" 2>/dev/null; then
      {
        printf '| Severity | Tool | File | Line | Description |\n'
        printf '|---|---|---|---|---|\n'
      } >> "$REPORT_FILE"

      eslint_out="$(cd "$NPM_DIR" && npx eslint . --format json 2>/dev/null)" || true
      if [ -n "$eslint_out" ]; then
        echo "$eslint_out" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    for result in data:
        file = result.get('filePath', '')
        for msg in result.get('messages', []):
            sev = 'CRITICAL' if msg.get('severity', 0) >= 2 else 'WARNING'
            line = str(msg.get('line', 0))
            rule = msg.get('ruleId', 'unknown')
            desc = msg.get('message', '').replace('|', '/')
            # Flag security-relevant rules
            sec_rules = ['no-eval', 'no-implied-eval', 'no-script-url', 'no-unsafe-innerHTML',
                        'no-innerHTML', 'no-danger', 'no-html-link', 'security/',
                        'xss', 'injection', 'no-secrets', 'cors']
            is_sec = any(r in rule.lower() for r in sec_rules)
            if is_sec or sev == 'CRITICAL':
                desc = f'{desc} ({rule})'
                print(f'{sev}|eslint|{file}|{line}|{desc}')
" 2>/dev/null | while IFS='|' read -r sev tool file line desc; do
          record_finding "$sev" "$tool" "$file" "$line" "$desc"
          JS_SEC_ISSUES=$((JS_SEC_ISSUES + 1))
        done
      fi

      if [ "$JS_SEC_ISSUES" -eq 0 ]; then
        printf '| - | - | - | - | No security issues found |\n' >> "$REPORT_FILE"
      fi
    else
      warn "eslint not configured in hub/ — skipping"
      write_section "JavaScript / TypeScript Security Analysis" "_eslint is not configured in hub/. Would need \`eslint.config.js\` with security plugins._\n\nRecommended configuration:\n\`\`\`\nnpm install --save-dev eslint-plugin-security eslint-plugin-no-secrets\n\`\`\`\n"
    fi
  else
    warn "npx not available — skipping eslint"
    write_section "JavaScript / TypeScript Security Analysis" "_npx/node is not available — skipped._\n\nWould run: \`npx eslint hub/\` with security plugins\n"
  fi
fi

# ============================================================================
# 4. Rust Security Linting (cargo clippy)
# ============================================================================

printf '\n## Rust Security Analysis\n\n' >> "$REPORT_FILE"

RUST_SEC_ISSUES=0

if [ -f "$CARGO_DIR/Cargo.toml" ]; then
  if detect_tool cargo; then
    log "Running cargo clippy on agent/"
    {
      printf '| Severity | Tool | File | Line | Description |\n'
      printf '|---|---|---|---|---|\n'
    } >> "$REPORT_FILE"

    # Clippy with security-related warnings
    clippy_out="$(cd "$CARGO_DIR" && cargo clippy --message-format=json 2>/dev/null)" || true
    if [ -n "$clippy_out" ]; then
      echo "$clippy_out" | python3 -c "
import sys, json
try:
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        try:
            msg = json.loads(line)
        except json.JSONDecodeError:
            continue
        if msg.get('reason') != 'compiler-message':
            continue
        message = msg.get('message', {})
        code = message.get('code', None)
        if not code:
            continue
        code_str = code.get('code', '')
        # Filter for security-relevant clippy lints
        sec_lints = ['unsafe', 'unwrap_used', 'expect_used', 'zombie_processes',
                    'file_access', 'command_execution', 'mem_forget', 'forget_non_drop',
                    'crypto_common', 'modulo_arithmetic', 'integer_arithmetic',
                    'panic', 'assert', 'todo', 'dbg_macro', 'print_stdout',
                    'use_after_move', 'uninit', 'zero_ptr', 'null_ptr',
                    'cast_lossless', 'cast_possible_truncation', 'cast_possible_wrap',
                    'cast_sign_loss', 'transmute']
        is_sec = any(s in code_str.lower() for s in sec_lints)
        if not is_sec:
            # Also flag any 'error' level clippy messages
            level = message.get('level', '')
            if level not in ('error', 'warning'):
                continue
        spans = message.get('spans', [])
        for span in spans:
            file = span.get('file_name', '')
            line = str(span.get('line_start', 0))
            sev = 'CRITICAL' if message.get('level') == 'error' else 'WARNING'
            desc = message.get('message', '').replace('|', '/')
            print(f'{sev}|clippy|{file}|{line}|{desc} [{code_str}]')
except Exception as e:
    sys.stderr.write(f'Failed to parse clippy output: {e}')
" 2>/dev/null | while IFS='|' read -r sev tool file line desc; do
        record_finding "$sev" "$tool" "$file" "$line" "$desc"
        RUST_SEC_ISSUES=$((RUST_SEC_ISSUES + 1))
      done
    fi

    if [ "$RUST_SEC_ISSUES" -eq 0 ]; then
      printf '| - | - | - | - | No security issues found |\n' >> "$REPORT_FILE"
    fi
  else
    warn "cargo not available — skipping clippy"
    write_section "Rust Security Analysis" "_Cargo/Rust toolchain not available — skipped._\n\nWould run: \`cargo clippy --message-format=json\` with security-related lint filtering\n"
  fi
fi

# ============================================================================
# 5. Hardcoded Secrets / Credentials Check (additional deep scan)
# ============================================================================

printf '\n## Hardcoded Secrets & Credentials Scan\n\n' >> "$REPORT_FILE"

HARDCODED_FOUND=0

{
  printf '| Severity | Tool | File | Line | Description |\n'
  printf '|---|---|---|---|---|\n'
} >> "$REPORT_FILE"

# Check for common credential patterns in source files (skip test files and generated files)
credential_patterns=(
  '(?i)password\s*[:=]\s*["'"'"'][^"'"'"']+["'"'"']'
  '(?i)secret\s*[:=]\s*["'"'"'][^"'"'"']+["'"'"']'
  '(?i)api[_-]?key\s*[:=]\s*["'"'"'][^"'"'"']+["'"'"']'
  '(?i)api[_-]?secret\s*[:=]\s*["'"'"'][^"'"'"']+["'"'"']'
  '(?i)access[_-]?token\s*[:=]\s*["'"'"'][^"'"'"']+["'"'"']'
  '(?i)auth[_-]?token\s*[:=]\s*["'"'"'][^"'"'"']+["'"'"']'
  '(?i)bearer\s+[A-Za-z0-9\-._~+/]+={0,2}'
  '(?i)jwt\s*[:=]\s*["'"'"']eyJ[A-Za-z0-9\-_]+\.eyJ[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]+["'"'"']'
)

for pattern in "${credential_patterns[@]}"; do
  while IFS=':' read -r file line content; do
    [ -z "$file" ] && continue
    # Skip test files, generated code, lock files, and node_modules
    case "$file" in
      *.test.go|*_test.go|*_test.tsx|*_test.ts|*_test.js|*.test.tsx|*.snap|*package-lock.json|*Cargo.lock|*go.sum|*.sum|*node_modules/*|*.pb.go|*_pb2.py|*.pyc|*venv/*|*.env.example|*.env.backup)
        continue ;;
    esac
    if echo "$file" | grep -q '/\.git/'; then
      continue
    fi
    # Skip .env.example files (they're templates)
    if echo "$file" | grep -q '\.env\.example$'; then
      continue
    fi
    # Trim content for display
    display_content="$(echo "$content" | sed 's/^[[:space:]]*//' | cut -c1-80)"
    desc="Possible hardcoded credential: ${display_content}..."
    record_finding "CRITICAL" "credential-scan" "$file" "$line" "$desc"
    HARDCODED_FOUND=$((HARDCODED_FOUND + 1))
  done < <(rg -n --no-heading "$pattern" "$PROJECT_DIR" --type-add 'src:*.{go,js,jsx,ts,tsx,rs,py,sh,yaml,yml,toml,json}' -t src 2>/dev/null || true)
done

if [ "$HARDCODED_FOUND" -eq 0 ]; then
  printf '| - | - | - | - | No hardcoded credentials detected |\n' >> "$REPORT_FILE"
fi

# ============================================================================
# 6. Environment Variable / Config Security Check
# ============================================================================

printf '\n## Environment & Configuration Security\n\n' >> "$REPORT_FILE"

ENV_ISSUES=0

# Check .env file exists and is in .gitignore
if [ -f "$PROJECT_DIR/.env" ]; then
  if [ -f "$PROJECT_DIR/.gitignore" ]; then
    if grep -q '^\.env$' "$PROJECT_DIR/.gitignore" 2>/dev/null; then
      write_section "Environment File (.env)" "✅ \`.env\` is properly listed in \`.gitignore\`."
    else
      warn ".env is not in .gitignore!"
      write_section "Environment File (.env)" "❌ **CRITICAL:** \`.env\` is **NOT** listed in \`.gitignore\`. Secrets may be committed to the repository!"
      record_finding "CRITICAL" "env-check" ".gitignore" "?" ".env file not found in .gitignore — secrets could be committed"
      ENV_ISSUES=$((ENV_ISSUES + 1))
    fi
  fi
else
  write_section "Environment File (.env)" "No \`.env\` file found. This is fine if using \`.env.example\`."
fi

# Check .env.example for placeholder values
if [ -f "$PROJECT_DIR/.env.example" ]; then
  hardcoded_example="$(grep -E '(password=|secret=|key=|token=).+[^=]$' "$PROJECT_DIR/.env.example" | grep -vi 'your-\|changeme\|<.*>\|example\|your_' || true)"
  if [ -n "$hardcoded_example" ]; then
    warn ".env.example may contain real-looking values"
    write_section "Environment Example (.env.example)" "⚠️  The following entries in \`.env.example\` look like they might contain real values:"
    {
      printf '\`\`\`\n%s\n\`\`\`\n' "$hardcoded_example"
    } >> "$REPORT_FILE"
    record_finding "WARNING" "env-check" ".env.example" "?" "Potentially real values in .env.example"
    ENV_ISSUES=$((ENV_ISSUES + 1))
  fi
fi

# ============================================================================
# 7. Permissions & File Security
# ============================================================================

printf '\n## File Permission & Security Audit\n\n' >> "$REPORT_FILE"

PERM_ISSUES=0
{
  printf '| Severity | Tool | File | Line | Description |\n'
  printf '|---|---|---|---|---|\n'
} >> "$REPORT_FILE"

# Check for world-writable files
while IFS= read -r file; do
  case "$file" in
    */node_modules/*|*/.git/*|*/target/*) continue ;;
  esac
  perm="$(stat -c '%a' "$file" 2>/dev/null)"
  if [ "${perm: -1}" = "2" ] || [ "${perm: -1}" = "3" ] || [ "${perm: -1}" = "6" ] || [ "${perm: -1}" = "7" ]; then
    record_finding "WARNING" "perm-check" "$file" "?" "World-writable file (permissions: $perm)"
    PERM_ISSUES=$((PERM_ISSUES + 1))
  fi
done < <(find "$PROJECT_DIR" -type f -not -path '*/.git/*' -not -path '*/node_modules/*' -not -path '*/target/*' -perm /o+w 2>/dev/null || true)

if [ "$PERM_ISSUES" -eq 0 ]; then
  printf '| - | - | - | - | No permission issues |\n' >> "$REPORT_FILE"
fi

# ============================================================================
# Summary
# ============================================================================

printf '\n## Summary\n\n' >> "$REPORT_FILE"

TOTAL_ISSUES=$((TOTAL_CRITICAL + TOTAL_WARNINGS))

{
  printf '| Category | Count |\n'
  printf '|----------|-------|\n'
  printf '| Go security issues (gosec) | %d |\n' "$GO_SEC_ISSUES"
  printf '| Secrets detected | %d |\n' "$SECRETS_FOUND"
  printf '| JS/TS security issues (eslint) | %d |\n' "$JS_SEC_ISSUES"
  printf '| Rust security issues (clippy) | %d |\n' "$RUST_SEC_ISSUES"
  printf '| Hardcoded credentials | %d |\n' "$HARDCODED_FOUND"
  printf '| Environment/config issues | %d |\n' "$ENV_ISSUES"
  printf '| Permission issues | %d |\n' "$PERM_ISSUES"
  printf '| **Total critical** | **%d** |\n' "$TOTAL_CRITICAL"
  printf '| **Total warnings** | **%d** |\n' "$TOTAL_WARNINGS"
} >> "$REPORT_FILE"

echo "" >> "$REPORT_FILE"

if [ "$TOTAL_CRITICAL" -gt 0 ]; then
  {
    printf '> **❌ Critical security issues found.** Review immediately.\n'
    printf '>\n'
    printf '> Exit code: 2\n'
  } >> "$REPORT_FILE"
  EXIT_CODE=2
elif [ "$TOTAL_WARNINGS" -gt 0 ]; then
  {
    printf '> **⚠️  Warnings found.** Review recommended before release.\n'
    printf '>\n'
    printf '> Exit code: 1\n'
  } >> "$REPORT_FILE"
else
  {
    printf '> **✅ All clean.** No security issues detected.\n'
    printf '>\n'
    printf '> Exit code: 0\n'
  } >> "$REPORT_FILE"
fi

echo "" >> "$REPORT_FILE"

log "Report generated: $REPORT_FILE"
exit "$EXIT_CODE"
