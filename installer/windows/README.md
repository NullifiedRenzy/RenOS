# RenOS Agent Windows Installer

This directory contains the WiX Toolset v4 source and supporting scripts for building the RenOS Agent MSI installer for 64-bit Windows (Windows 10 / Windows Server 2016+).

## Prerequisites

| Requirement | Version | Notes |
|---|---|---|
| **WiX Toolset v4** | v4.x | Build toolchain. Auto-downloaded if missing. |
| **PowerShell** | 5.1+ | Included with Windows 10/Server 2016+ |
| **.NET Framework** | 4.7.2+ | Required by WiX and enrollment wizard |
| **Windows SDK** | 10.0+ | Optional — only needed for code signing (`SignTool`) |

## Files

| File | Purpose |
|---|---|
| `renos.wxs` | WiX XML product definition: service, files, registry, shortcuts |
| `build-msi.ps1` | Build script: compiles `.wxs` → `.wixobj` → `.msi`, signs |
| `renos-agent-service.ps1` | Service installer: `New-Service`, recovery, logs, firewall |
| `enrollment-wizard.ps1` | Windows Forms GUI for device enrollment |
| `config.json` | Agent configuration file (shipped inside MSI) |

## Quick Start

```powershell
# From this directory:
.\build-msi.ps1

# With custom version:
.\build-msi.ps1 -Version "1.2.3"

# From any directory:
.\build-msi.ps1 -SourceDir "C:\path\to\artifacts" -Version "2.0.0"
```

The output MSI is written to `dist/RenOS-Agent-x64.msi`.

## Build Process

1. **Compilation** — `candle.exe` compiles `renos.wxs` into `renos.wixobj` (passes `-dVersion` and `-dSourceDir`).
2. **Linking** — `light.exe` links the `.wixobj` into a signed MSI package.
3. **Signing** — If `CODE_SIGN_CERT` environment variable is set to a certificate thumbprint, the MSI is Authenticode-signed with SHA-256 and RFC 3161 timestamping.

### Required Source Files

The following files must exist in `$SourceDir` before building:

- `renos-agent.exe` — Agent binary
- `config.json` — Agent configuration
- `enrollment-wizard.ps1` — Enrollment GUI
- `renos-agent-service.ps1` — Service installer (deployed alongside the agent)

## Code Signing Setup

Set the `CODE_SIGN_CERT` environment variable to the SHA-1 thumbprint of your code signing certificate:

```powershell
# User scope (recommended for CI):
$env:CODE_SIGN_CERT = "A1B2C3D4E5F6..."

# Or set permanently:
[Environment]::SetEnvironmentVariable("CODE_SIGN_CERT", "A1B2C3D4E5F6...", "User")

# Or in CI (GitHub Actions):
# env:
#   CODE_SIGN_CERT: ${{ secrets.CODE_SIGN_THUMBPRINT }}
```

The certificate must be installed in the `CurrentUser\My` or `LocalMachine\My` certificate store.

### Installing a Code Signing Certificate

```powershell
# Import a .pfx into the CurrentUser store:
Import-PfxCertificate -FilePath "certificate.pfx" -CertStoreLocation Cert:\CurrentUser\My
```

## CI/CD Integration

### GitHub Actions

```yaml
- name: Build MSI
  shell: pwsh
  run: ./installer/windows/build-msi.ps1 -Version "${{ github.ref_name }}"
  env:
    CODE_SIGN_CERT: ${{ secrets.CODE_SIGN_THUMBPRINT }}
```

### Azure DevOps

```yaml
- task: PowerShell@2
  inputs:
    filePath: 'installer/windows/build-msi.ps1'
    arguments: '-Version "$(Build.BuildNumber)"'
  env:
    CODE_SIGN_CERT: $(CodeSignThumbprint)
```

### Artifacts

The build script outputs one artifact:

| Artifact | Path |
|---|---|
| MSI Installer | `dist/RenOS-Agent-x64.msi` |

Add `dist/` to `.gitignore` or configure your CI to publish it as a build artifact.

## What the MSI Does

| Action | Details |
|---|---|
| **Install** | Copies `renos-agent.exe`, `config.json`, `enrollment-wizard.ps1`, `renos-agent-service.ps1` to `%ProgramFiles%\RenOS\Agent` |
| **Service** | Creates the `RenOS-Agent` Windows service (auto-start, LocalSystem) |
| **Firewall** | Adds inbound rules for TCP ports 80, 443, 8443 |
| **Registry** | Writes install path, version, and config location to `HKLM\Software\RenOS` |
| **Scheduled Task** | Creates `RenOS-Agent-AutoUpdate` (daily at 3 AM) |
| **Shortcuts** | Adds Start Menu entries for the enrollment wizard and uninstaller |
| **Uninstall** | Stops/removes the service, deletes firewall rules, removes scheduled task, cleans up all files and registry keys |

## Manual Service Installation

If you prefer to install the service manually (e.g., after extracting the MSI):

```powershell
# Install (admin):
.\renos-agent-service.ps1

# Uninstall (admin):
.\renos-agent-service.ps1 -Uninstall
```

## Troubleshooting

| Issue | Check |
|---|---|
| Build fails: "candle not found" | WiX Toolset v4 not installed. The script auto-downloads it, or install from https://wixtoolset.org/ |
| Build fails: "light.exe error LGHT0103" | The `.wxs` references files not in `$SourceDir` — verify all source files exist |
| Code signing fails | Ensure the certificate thumbprint in `CODE_SIGN_CERT` matches an installed certificate |
| MSI install fails | Check `%TEMP%\MSI*.log` for detailed error messages |
| Service won't start | Run `.\renos-agent-service.ps1` manually and check the Event Log under `RenOS-Agent` |
