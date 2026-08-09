#!/usr/bin/env bash
# ───────────────────────────────────────────────────────────
# RenOS SBOM Generator
# Produces CycloneDX 1.5, SPDX 2.3, and human-readable HTML
# ───────────────────────────────────────────────────────────
set -euo pipefail

RENOS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
SBOM_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_CYCLONEDX="${SBOM_DIR}/renos-cyclonedx.json"
OUTPUT_SPDX="${SBOM_DIR}/renos-spdx.json"
OUTPUT_HTML="${SBOM_DIR}/renos-sbom.html"
TIMESTAMP="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
RENOS_VERSION="$(cd "$RENOS_ROOT" && git describe --tags --always 2>/dev/null || echo "0.1.0")"

TEMP_COMPONENTS=$(mktemp)
TEMP_DEPS=$(mktemp)
trap 'rm -f "$TEMP_COMPONENTS" "$TEMP_DEPS"' EXIT

record()  { printf '%s\t%s\t%s\t%s\t%s\n' "$1" "$2" "$3" "$4" "$5" >> "$TEMP_COMPONENTS"; }
log()     { echo "  $*"; }

# ── Go ──────────────────────────────────────────────────

scan_go_module() {
  local dir="$1" mod_name="$2" mod_type="${3:-library}"
  local mod_file="$dir/go.mod"
  [[ -f "$mod_file" ]] || { log "skip (no go.mod): $mod_name"; return; }

  record "$mod_name" "$RENOS_VERSION" "$mod_type" "pkg:golang/$mod_name@$RENOS_VERSION" ""

  if command -v go &>/dev/null && (cd "$dir" && go list -m -json all &>/dev/null); then
    (cd "$dir" && go list -m -json all 2>/dev/null) | python3 -c "
import sys, json
MOD = '$mod_name'
for line in sys.stdin:
    line = line.strip()
    if not line: continue
    try:
        m = json.loads(line)
    except json.JSONDecodeError:
        continue
    path = m.get('Path', '')
    ver  = m.get('Version', '')
    if path and ver and path != MOD:
        purl = 'pkg:golang/' + path + '@' + ver
        print(path + '\t' + ver + '\tlibrary\t' + purl + '\t')
" >> "$TEMP_DEPS"
  else
    log "  (parse go.mod: $mod_file)"
    python3 -c "
import re
with open('$mod_file') as f:
    text = f.read()
for line in text.splitlines():
    m = re.match(r'^\s+(\S+)\s+v?([\w.+\-/]+)', line)
    if m:
        name, ver = m.group(1), m.group(2)
        if name and ver and not name.startswith('//'):
            purl = 'pkg:golang/' + name + '@' + ver
            print(name + '\t' + ver + '\tlibrary\t' + purl + '\t')
" >> "$TEMP_DEPS"
  fi
}

# ── Rust ────────────────────────────────────────────────

scan_rust() {
  local lockfile="$RENOS_ROOT/agent/Cargo.lock"
  local cargo_toml="$RENOS_ROOT/agent/Cargo.toml"
  [[ -f "$lockfile" ]] || { log "skip (no Cargo.lock)"; return; }

  if [[ -f "$cargo_toml" ]]; then
    local agent_ver
    agent_ver=$(python3 -c "
import re
with open('$cargo_toml') as f:
    m = re.search(r'^version\s*=\s*\"(.+?)\"', f.read(), re.M)
    print(m.group(1) if m else '$RENOS_VERSION')
")
    record "renos-agent" "$agent_ver" "application" "pkg:cargo/renos-agent@${agent_ver}" "MIT"
  fi

  python3 -c "
import re
with open('$lockfile') as f:
    content = f.read()
for m in re.finditer(
    r'\[\[package\]\]\s*\nname\s*=\s*\"(.+?)\"\s*\nversion\s*=\s*\"(.+?)\"',
    content
):
    name, ver = m.group(1), m.group(2)
    purl = 'pkg:cargo/' + name + '@' + ver
    print(name + '\t' + ver + '\tlibrary\t' + purl + '\t')
" >> "$TEMP_DEPS"
}

# ── NPM ─────────────────────────────────────────────────

scan_npm() {
  local lockfile="$RENOS_ROOT/hub/package-lock.json"
  [[ -f "$lockfile" ]] || { log "skip (no package-lock.json)"; return; }

  record "renos-hub" "$RENOS_VERSION" "application" "pkg:npm/renos-hub@${RENOS_VERSION}" "MIT"

  python3 -c "
import json
with open('$lockfile') as f:
    data = json.load(f)
skip_prefixes = ('node_modules/.cache',)
platform_suffixes = (
    '-darwin', '-linux', '-win32', '-freebsd', '-openbsd', '-android',
    '-darwin-arm64', '-darwin-x64', '-linux-arm64', '-linux-x64',
    '-win32-arm64', '-win32-x64',
)
packages = data.get('packages', {})
for path, info in packages.items():
    if path == '' or path.startswith(skip_prefixes):
        continue
    name = path.replace('node_modules/', '')
    if any(name.endswith(suf) for suf in platform_suffixes):
        continue
    if any(p in name for p in ('-darwin-', '-linux-', '-win32-', '-freebsd-', '-openbsd-', '-android-')):
        continue
    ver = info.get('version', '')
    lic = info.get('license', '')
    if not name or not ver:
        continue
    purl = 'pkg:npm/' + name + '@' + ver
    print(name + '\t' + ver + '\tlibrary\t' + purl + '\t' + lic)
" >> "$TEMP_DEPS"
}

# ── Dedup ───────────────────────────────────────────────

assemble_components() {
  cat "$TEMP_COMPONENTS" "$TEMP_DEPS" | python3 -c "
import sys
seen = set()
for line in sys.stdin:
    line = line.strip()
    if not line or line in seen:
        continue
    seen.add(line)
    print(line)
" > "${TEMP_COMPONENTS}.dedup"
  mv "${TEMP_COMPONENTS}.dedup" "$TEMP_COMPONENTS"
}

# ── CycloneDX 1.5 ─────────────────────────────────────

generate_cyclonedx() {
  python3 << PYEOF
import json

SBOM_DATA = '$TEMP_COMPONENTS'
TIMESTAMP = '$TIMESTAMP'
RENOS_VERSION = '$RENOS_VERSION'

components = []
with open(SBOM_DATA) as f:
    for line in f:
        line = line.strip()
        if not line:
            continue
        parts = line.split('\t')
        if len(parts) < 2:
            continue
        name, version = parts[0], parts[1]
        ctype = parts[2] if len(parts) > 2 else 'library'
        purl  = parts[3] if len(parts) > 3 else ''
        lic   = parts[4] if len(parts) > 4 else ''

        comp = {
            'name': name,
            'version': version,
            'type': ctype,
            'purl': purl,
        }
        if lic:
            comp['licenses'] = [{'license': {'name': lic}}]

        refs = []
        if purl.startswith('pkg:golang/'):
            refs.append({'url': f'https://pkg.go.dev/{name}', 'type': 'documentation'})
            if name.startswith('github.com/'):
                refs.append({'url': f'https://{name}', 'type': 'vcs'})
        elif purl.startswith('pkg:cargo/'):
            pkg_name = name.split('/')[-1].split('@')[0]
            refs.append({'url': f'https://crates.io/crates/{pkg_name}', 'type': 'distribution'})
        elif purl.startswith('pkg:npm/'):
            npm_name = name.split('@')[0] if name.startswith('@') else name.split('@')[0]
            refs.append({'url': f'https://www.npmjs.com/package/{npm_name}', 'type': 'distribution'})
        if refs:
            comp['externalReferences'] = refs

        components.append(comp)

doc = {
    'bomFormat': 'CycloneDX',
    'specVersion': '1.5',
    'version': 1,
    'metadata': {
        'timestamp': TIMESTAMP,
        'tools': [{'name': 'renos-sbom-generator', 'version': RENOS_VERSION}],
        'authors': [{'name': 'RenOS Contributors'}],
        'component': {
            'name': 'RenOS',
            'version': RENOS_VERSION,
            'type': 'application',
            'purl': f'pkg:generic/renos@{RENOS_VERSION}'
        }
    },
    'components': components
}

with open('$OUTPUT_CYCLONEDX', 'w') as f:
    json.dump(doc, f, indent=2)

print(f'  CycloneDX: {len(components)} components -> $OUTPUT_CYCLONEDX')
PYEOF
}

# ── SPDX 2.3 ──────────────────────────────────────────

generate_spdx() {
  python3 << PYEOF
import json, hashlib, re

SBOM_DATA = '$TEMP_COMPONENTS'
TIMESTAMP = '$TIMESTAMP'
RENOS_VERSION = '$RENOS_VERSION'

def pkg_id(name):
    s = re.sub(r'[/@.+()#]', '-', name)
    s = re.sub(r'-+', '-', s).strip('-')
    return f'SPDXRef-{s}' if s else 'SPDXRef-pkg'

packages = []
pkg_ids = []

with open(SBOM_DATA) as f:
    for line in f:
        line = line.strip()
        if not line:
            continue
        parts = line.split('\t')
        if len(parts) < 2:
            continue
        name, version = parts[0], parts[1]
        ctype = parts[2] if len(parts) > 2 else 'library'
        purl  = parts[3] if len(parts) > 3 else ''
        lic   = parts[4] if len(parts) > 4 else ''

        pid = pkg_id(f'{name}-{version}')

        pkg = {
            'name': name,
            'versionInfo': version,
            'SPDXID': pid,
            'licenseConcluded': lic if lic else 'NOASSERTION',
            'licenseDeclared': lic if lic else 'NOASSERTION',
            'copyrightText': 'NOASSERTION',
            'supplier': 'Organization: RenOS Contributors',
            'filesAnalyzed': False,
        }

        if purl:
            pkg['externalRefs'] = [{
                'referenceCategory': 'PACKAGE-MANAGER',
                'referenceType': 'purl',
                'referenceLocator': purl
            }]

        digest = hashlib.sha256(f'{name}@{version}'.encode()).hexdigest()
        pkg['checksums'] = [{'algorithm': 'SHA256', 'checksumValue': digest}]

        packages.append(pkg)
        pkg_ids.append(pid)

relationships = []
for pid in pkg_ids:
    relationships.append({
        'spdxElementId': 'SPDXRef-DOCUMENT',
        'relatedSpdxElement': pid,
        'relationshipType': 'DESCRIBES'
    })

doc = {
    'spdxVersion': 'SPDX-2.3',
    'dataLicense': 'CC0-1.0',
    'SPDXID': 'SPDXRef-DOCUMENT',
    'name': 'RenOS SBOM',
    'documentNamespace': f'https://renos-project.github.io/sbom/renos/spdx/{TIMESTAMP.replace(":", "-")}',
    'creationInfo': {
        'created': TIMESTAMP,
        'creators': [
            'Organization: RenOS Contributors',
            f'Tool: renos-sbom-generator-{RENOS_VERSION}'
        ],
        'licenseListVersion': '3.21'
    },
    'packages': packages,
    'relationships': relationships,
    'documentDescribes': pkg_ids[:1] if pkg_ids else []
}

with open('$OUTPUT_SPDX', 'w') as f:
    json.dump(doc, f, indent=2)

print(f'  SPDX:       {len(packages)} packages -> $OUTPUT_SPDX')
PYEOF
}

# ── HTML ───────────────────────────────────────────────

generate_html() {
  local template="$SBOM_DIR/renos-sbom.html"

  python3 << PYEOF
import json

SBOM_DATA = '$TEMP_COMPONENTS'
TEMPLATE = '$template'
OUTPUT = '$OUTPUT_HTML'

components = []
with open(SBOM_DATA) as f:
    for line in f:
        line = line.strip()
        if not line:
            continue
        parts = line.split('\t')
        if len(parts) < 2:
            continue
        name, version = parts[0], parts[1]
        ctype = parts[2] if len(parts) > 2 else 'library'
        purl  = parts[3] if len(parts) > 3 else ''
        lic   = parts[4] if len(parts) > 4 else ''
        components.append({
            'name': name,
            'version': version,
            'type': ctype,
            'purl': purl,
            'license': lic if lic else 'Unknown'
        })

with open(TEMPLATE) as f:
    html = f.read()
html = html.replace('__SBOM_DATA__', json.dumps(components))
with open(OUTPUT, 'w') as f:
    f.write(html)

print(f'  HTML:       {len(components)} components -> {OUTPUT}')
PYEOF
}

# ── main ──────────────────────────────────────────────

echo "=== RenOS SBOM Generator ==="
echo "  Root:    $RENOS_ROOT"
echo "  Version: $RENOS_VERSION"
echo ""

echo "Scanning Go modules..."
scan_go_module "$RENOS_ROOT/core"                    "github.com/renos-project/renos/core"           "application"
scan_go_module "$RENOS_ROOT/renctl"                  "github.com/renos-project/renos/renctl"         "application"
scan_go_module "$RENOS_ROOT/sdk/go"                  "github.com/renos-project/renos/sdk/go"         "library"
scan_go_module "$RENOS_ROOT/installer/upgrade"       "github.com/renos-project/renos/installer/upgrade" "application"
scan_go_module "$RENOS_ROOT/installer/wizard"        "github.com/renos-project/renos/installer/wizard"  "application"
scan_go_module "$RENOS_ROOT/core/cmd/renos-plugin"   "github.com/renos-project/renos/core/cmd/renos-plugin" "plugin"

echo "Scanning Rust (agent)..."
scan_rust

echo "Scanning NPM (hub)..."
scan_npm

assemble_components

total=$(wc -l < "$TEMP_COMPONENTS")
echo ""
echo "Total unique components: $total"
echo ""

echo "Generating SBOM formats..."
generate_cyclonedx
generate_spdx
generate_html
echo ""
echo "=== SBOM generation complete ==="
echo "  CycloneDX: $OUTPUT_CYCLONEDX"
echo "  SPDX:      $OUTPUT_SPDX"
echo "  HTML:      $OUTPUT_HTML"
