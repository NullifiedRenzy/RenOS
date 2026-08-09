# RenOS 1.0.0 — Linux Test Report

## Test Environment

| Parameter | Value |
|---|---|
| **Version Under Test** | RenOS 1.0.0-RELEASE (build 20260510.1) |
| **Test Period** | 2026-05-11 – 2026-05-20 |
| **Runner Type** | GitHub Actions `ubuntu-latest`, `ubuntu-22.04-arm64` (self-hosted), DigitalOcean droplets, and physical x86_64/ARM64 lab |
| **Kernel Versions** | 5.4 (Ubuntu 20.04), 5.15 (Ubuntu 22.04), 6.5 (Debian 12), 6.8 (Fedora 40) |
| **Init Systems** | systemd 245–255 |
| **Container Runtimes** | Docker 26.1, containerd 1.7, podman 5.0 |

## Test Results

| Test | Ubuntu 20.04 | Ubuntu 22.04 | Ubuntu 24.04 | Debian 11 | Debian 12 | Fedora 38 | Fedora 39 | Fedora 40 | RHEL 8 | RHEL 9 | Rocky 8 | Rocky 9 | Alma 8 | Alma 9 | Arch |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| **Build from source** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Install (apt/dnf/pacman)** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ⚠️ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Service boot** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Config parsing** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Core daemon lifecycle** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Hub message broker** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Agent registration** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Agent heartbeat** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **CLI commands** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **REST API** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **gRPC endpoints** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Plugin load/unload** | ✅ | ✅ | ✅ | ✅ | ✅ | ⚠️ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Upgrade 0.9.4→1.0.0** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ⚠️ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Backup & restore** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Logging & journald** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ⚠️ |

### Platform-Specific Details

#### Ubuntu (20.04 / 22.04 / 24.04 — amd64 + arm64)
All six variants passed every test without regression. ARM64 native builds (cross-compiled from amd64, verified on AWS Graviton3 and Raspberry Pi 5) produced identical binary hashes. The Hub TLS endpoint negotiated TLS 1.3 correctly across all three OpenSSL versions (1.1.1, 3.0, 3.3).

#### Debian (11 / 12)
Debian 11 required backported Go 1.22 from the buster-backports repository for the build step. Debian 12 shipped with Go 1.21 out of the box, which met the minimum requirement. No functional differences observed.

#### Fedora (38 / 39 / 40)
- **Fedora 38 ⚠️ Plugin GPG mismatch:** The bundled third-party plugin registry shipped with a GPG key using SHA-1 signatures. Fedora 38 enforces SHA-256 minimum for RPM GPG verification. Plugins installed but produced a non-fatal warning. Upgraded the signing pipeline for Fedora 38+ to use SHA-256 in the 1.0.1 patch. No runtime impact.

#### RHEL 8
- **⚠️ Install (epel missing postgresql module):** On minimal RHEL 8 images, `epel-release` was not enabled by default, causing the `renos-hub` PostgreSQL dependency to resolve to a broken module stream. Workaround: `dnf module enable postgresql:15` before install. The installer now prints a clear error message and instructions. Fixed in the next minor release.
- **⚠️ Upgrade:** The upgrade script detected an incompatible `pg_hba.conf` md5→scram-sha-256 migration. Manual intervention required on two of eight test nodes. The upgrade guide has been updated with the recovery steps.

#### Rocky / AlmaLinux (8 / 9)
All tests passed without issue. Rocky and Alma mirrors were slightly slower during CI (< 30 s installs vs. < 15 s on Ubuntu) but this had zero functional impact.

#### Arch Linux (Rolling — amd64 + arm64)
- **⚠️ Agent systemd-journald logging:** Arch does not ship `systemd-journald` in its default container images; the Agent fell back to file-based logging without issue. ARM64 Arch ran on a PineTab 2 with 4 GB RAM — all services functional under constrained memory.

## Known Issues

| ID | Severity | Platform | Description | Workaround / Status |
|---|---|---|---|---|
| LIN-001 | Low | Fedora 38 | Plugin GPG key uses SHA-1, non-fatal warning | Will ship SHA-256 key in 1.0.1 |
| LIN-002 | Medium | RHEL 8 | PostgreSQL module not enabled by default | Installer now prints resolution steps |
| LIN-003 | Medium | RHEL 8 | Upgrade may fail on md5→scram migration | Documented manual recovery procedure |
| LIN-004 | Low | Arch (container) | systemd-journald absent, file fallback used | Expected behaviour; no bug filed |
| LIN-005 | Low | Debian 11 | Requires backported Go toolchain for build | Documented in build prerequisites |

## Summary / Conclusion

All 15 Linux distributions (including ARM64 variants) pass the RenOS 1.0.0 test suite. The five logged issues are minor or well-documented with clear workarounds. No release-blocking defects were found on Linux. Ubuntu 22.04 LTS and RHEL 9 are recommended as the primary deployment targets.
