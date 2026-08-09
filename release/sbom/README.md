# RenOS Software Bill of Materials (SBOM)

This directory contains tooling to generate and serve SBOMs for the RenOS project.

## What is an SBOM?

A Software Bill of Materials (SBOM) is a formal, machine-readable inventory of all components used to build a software artifact. It lists dependencies, versions, licenses, and package relationships. SBOMs are essential for:

- **Security**: Quickly identify vulnerable components (e.g., via CVE matching).
- **License compliance**: Track open-source license obligations across the supply chain.
- **Supply chain transparency**: Know exactly what runs in your stack.

RenOS produces SBOMs in two industry-standard formats:

| Format      | Version | Spec                                                                 |
|-------------|---------|----------------------------------------------------------------------|
| CycloneDX   | 1.5     | https://cyclonedx.org/specification/1.5/                             |
| SPDX        | 2.3     | https://spdx.dev/specifications/spdx-2.3/                            |

## How to Generate

### Prerequisites

- Bash 4+
- Python 3.8+
- Go (optional — script falls back to parsing `go.mod` if `go` is not available)
- `git` (optional — for version detection)

### Generate

```bash
cd /home/ren/Projects/RenOS
./release/sbom/generate-sbom.sh
```

The script scans:

| Ecosystem | Source                                                   | Tooling                    |
|-----------|----------------------------------------------------------|----------------------------|
| Go        | `core/`, `renctl/`, `sdk/go/`, `installer/upgrade/`, `installer/wizard/`, `core/cmd/renos-plugin/` | `go list -m -json all` or `go.mod` parsing |
| Rust      | `agent/`                                                 | `Cargo.lock`               |
| NPM       | `hub/`                                                   | `package-lock.json`        |

### Outputs

| File                    | Format      | Description                       |
|-------------------------|-------------|-----------------------------------|
| `renos-cyclonedx.json`  | CycloneDX 1.5 | Machine-readable SBOM (JSON)    |
| `renos-spdx.json`       | SPDX 2.3      | Machine-readable SBOM (JSON)    |
| `renos-sbom.html`       | HTML       | Human-readable, searchable table  |

## Verifying SBOM Integrity

Each SBOM should be verified after generation:

### 1. Schema validation

**CycloneDX** — validate against the official schema:

```bash
pip install json-schema
# CycloneDX 1.5 schema
curl -sL https://raw.githubusercontent.com/CycloneDX/specification/1.5/schema/bom-1.5.schema.json \
  > /tmp/cyclonedx-1.5.schema.json
python3 -m jsonschema -i renos-cyclonedx.json /tmp/cyclonedx-1.5.schema.json
```

**SPDX** — validate with the SPDX tools:

```bash
pip install spdx-tools
spdx-tools json validate renos-spdx.json
```

### 2. Cryptographic hash verification

The SPDX output includes `SHA256` checksums for every package. To verify a specific package:

```bash
echo -n "package-name@1.2.3" | sha256sum
```

Compare the output to the `checksums.checksumValue` field in `renos-spdx.json`.

### 3. Component count sanity

The HTML report displays total component counts. Compare against a rough baseline:

```bash
# Count unique Go dependencies
cd core && go list -m all | wc -l

# Count unique Rust dependencies
cd agent && cargo tree --prefix none | wc -l

# Count unique NPM dependencies
cd hub && jq '.packages | keys | length' package-lock.json
```

## Integrating with CI/CD

Add to your CI pipeline to generate SBOMs on every release:

```yaml
# .github/workflows/release.yml (example)
- name: Generate SBOM
  run: ./release/sbom/generate-sbom.sh
- name: Upload SBOM artifacts
  uses: actions/upload-artifact@v4
  with:
    name: renos-sbom
    path: release/sbom/renos-*.json
```

## CI/CD: Verify SBOM on Every PR

Add a workflow step to validate that the `renos-sbom.html` file matches the current dependency tree. For example, you can check that the number of components doesn't unexpectedly change:

```yaml
- name: Check SBOM component count
  run: |
    BEFORE=$(wc -l < release/sbom/renos-sbom.html)
    ./release/sbom/generate-sbom.sh
    AFTER=$(wc -l < release/sbom/renos-sbom.html)
    echo "Components: $BEFORE -> $AFTER"
```

## Updating SBOM Data Format

To add a new Go module to the scan, add a new `scan_go_module` call in `generate-sbom.sh`. For a new language ecosystem, add a new `scan_*` function and invoke it from `main`.

The HTML template is a standalone file with a `__SBOM_DATA__` placeholder that the script replaces with JSON data. Edit `renos-sbom.html` to change the HTML presentation.
