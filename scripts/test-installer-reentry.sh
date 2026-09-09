#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
scratch="$(mktemp -d)"
trap 'rm -rf "$scratch"' EXIT
mkdir -p "$scratch/bin" "$scratch/project"
cp "$root/install.sh" "$scratch/project/install.sh"
cp "$root/docker-compose.prod.yml" "$scratch/project/docker-compose.prod.yml"
export INSTALLER_TEST_LOG="$scratch/docker.log"

cat > "$scratch/bin/docker" <<'SH'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$INSTALLER_TEST_LOG"
if [[ "$*" == *'config --quiet'* ]] && [[ "${FAIL_CONFIG:-0}" == 1 ]]; then
    exit 1
fi
if [[ "$*" == *'ps -q core'* ]]; then printf 'test-core\n'; fi
if [[ "$*" == inspect* ]]; then printf '%s\n' "${TEST_CORE_HEALTH:-healthy}"; fi
SH
chmod +x "$scratch/bin/docker"
printf 'POSTGRES_PASSWORD=existing-db-secret\nRENOS_ENCRYPTION_KEY=existing-key\n' > "$scratch/project/.env"
cp "$scratch/project/.env" "$scratch/before.env"
PATH="$scratch/bin:$PATH" bash "$scratch/project/install.sh" </dev/null > "$scratch/success.log"
cmp "$scratch/before.env" "$scratch/project/.env"
test ! -e "$scratch/project/.env.backup"
grep -q 'build' "$INSTALLER_TEST_LOG"
printf 'PASS: rerun preserves existing configuration without prompting\n'

: > "$INSTALLER_TEST_LOG"
if PATH="$scratch/bin:$PATH" FAIL_CONFIG=1 bash "$scratch/project/install.sh" </dev/null > "$scratch/failure.log" 2>&1; then
    printf 'FAIL: invalid configuration accepted\n' >&2
    exit 1
fi
! grep -q 'build\|up -d' "$INSTALLER_TEST_LOG"
cmp "$scratch/before.env" "$scratch/project/.env"
printf 'PASS: invalid configuration stops before build or deployment\n'

if PATH="$scratch/bin:$PATH" TEST_CORE_HEALTH=unhealthy RENOS_INSTALL_TIMEOUT=1 \
    bash "$scratch/project/install.sh" </dev/null > "$scratch/health.log" 2>&1; then
    printf 'FAIL: unhealthy deployment reported success\n' >&2
    exit 1
fi
! grep -q 'Installation Complete' "$scratch/health.log"
printf 'PASS: unhealthy deployment fails instead of reporting success\n'

# A fresh setup must protect dotenv literals, not expand password characters.
for password in "literal\$dollar#hash'quote\\slash" "password-trailing\\" "password\\'quote" "  password with spaces  " 'double"quote${VARIABLE}'; do
    rm "$scratch/project/.env"
    printf 'test@example.com\n%s\nhttp://localhost\n' "$password" |
        PATH="$scratch/bin:$PATH" bash "$scratch/project/install.sh" > "$scratch/fresh.log"
    test "$(stat -c '%a' "$scratch/project/.env")" = 600
    parsed="$(docker compose --env-file "$scratch/project/.env" -f "$root/docker-compose.prod.yml" config --environment |
        sed -n 's/^RENOS_ADMIN_PASSWORD=//p')"
    test "$parsed" = "$password"
done
printf 'PASS: fresh configuration is private and preserves password literals\n'
