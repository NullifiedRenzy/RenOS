# RenOS Agent — macOS PKG Installer

Builds a signed, distribution-ready macOS installer package for the RenOS Agent.

## Prerequisites

- macOS 12+ (for build host)
- Xcode Command Line Tools (`xcode-select --install`)
- Developer ID Installer certificate (for signed builds; optional)

## Directory Layout

```
installer/macos/
├── build-pkg.sh              # Build script
├── Distribution.xml          # Product distribution definition
├── postinstall.sh            # Post-installation script
├── com.renos.agent.plist     # LaunchAgent property list
├── keychain-setup.sh         # Keychain integration helpers
├── Resources/                # Localized resources (created if missing)
│   └── en.lproj/
│       ├── Welcome.html
│       ├── ReadMe.html
│       ├── License.html
│       └── Conclusion.html
├── dist/                     # Output directory
│   ├── RenOS-Agent-1.0.0-component.pkg
│   └── RenOS-Agent-1.0.0.pkg
└── README.md
```

## Building

### Quick Build (unsigned)

```bash
./installer/macos/build-pkg.sh
```

Output: `installer/macos/dist/RenOS-Agent-1.0.0.pkg`

### Signed Build

```bash
export PKG_SIGN_IDENTITY="Developer ID Installer: Your Name (TEAMID)"
export PKG_DEV_TEAM="TEAMID"
./installer/macos/build-pkg.sh
```

### With Custom Build Artifacts

Place compiled binaries and assets in `build/` at the project root. The build
script stages the entire `build/` directory into `/Library/Application Support/RenOS/`.

## Build Process

1. **Stage artifacts** — copies built binaries, LaunchAgent plist, and scripts into
   a temporary package root.
2. **Code sign** — signs all Mach-O binaries with the provided identity (if set).
3. **Component package** — builds a component `.pkg` via `pkgbuild` with an
   analyzed component plist.
4. **Distribution** — wraps the component into a product archive via `productbuild`
   using `Distribution.xml`.
5. **Verify** — inspects the final package signature and size.

## Installation Requirements

| Requirement          | Minimum               |
|----------------------|-----------------------|
| macOS                | 12 Monterey           |
| RAM                  | 2 GB                  |
| Free Disk Space      | 10 GB                 |
| Architecture         | Intel (x86_64) / Apple Silicon (arm64) |

## Post-Install Behaviour

The `postinstall.sh` script (embedded in the PKG) performs:

1. Creates `/Library/Application Support/RenOS/` with subdirectories
2. Installs `/Library/LaunchAgents/com.renos.agent.plist`
3. Adds firewall rules via `socketfilterfw`
4. Registers enrollment token in the System Keychain
5. Writes default `config.json` with device metadata
6. Starts the LaunchAgent via `launchctl bootstrap`
7. Runs `renos-agent --enroll` for initial device enrollment

## Keychain Integration

`keychain-setup.sh` provides CLI and library usage:

```bash
# Source as library
source keychain-setup.sh
store_credential "com.renos.agent" "<token>" "<url>" "<device-id>"

# Run as CLI
keychain-setup.sh store  "<token>" "<url>" "<device-id>"
keychain-setup.sh get
keychain-setup.sh delete
keychain-setup.sh has
```

Credentials are stored in the System Keychain as generic password items
with the service name `com.renos.agent`.

## Environment Variables

| Variable              | Default                     | Description                    |
|-----------------------|-----------------------------|--------------------------------|
| `PKG_SIGN_IDENTITY`   | (unset)                     | Developer ID Installer cert CN |
| `PKG_DEV_TEAM`        | (unset)                     | Team ID for keychain lookup    |

## Troubleshooting

- **"pkgbuild: command not found"** — Install Xcode Command Line Tools:
  ```bash
  xcode-select --install
  ```
- **"productbuild: error"** — Check that `Distribution.xml` is valid XML and all
  referenced resources exist under `Resources/`.
- **Codesign failures** — Verify the certificate in Keychain Access and confirm
  the identity matches `PKG_SIGN_IDENTITY`.
- **LaunchAgent not starting** — Check logs in `/var/log/renos/` and verify
  the plist with:
  ```bash
  plutil -lint /Library/LaunchAgents/com.renos.agent.plist
  launchctl print system/com.renos.agent
  ```

## CI Integration

```yaml
# GitHub Actions example
- name: Build macOS PKG
  run: |
    export PKG_SIGN_IDENTITY="${{ secrets.MAC_SIGN_IDENTITY }}"
    ./installer/macos/build-pkg.sh
  if: runner.os == 'macOS'
```
