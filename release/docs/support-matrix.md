# RenOS 1.0.0 Support Matrix

This document lists supported versions and configurations for RenOS 1.0.0.

---

## Deployment Methods

| Method | Support Level | Notes |
|--------|--------------|-------|
| Docker (single node) | Full | `renos/control-plane:1.0.0` and `renos/agent:1.0.0` |
| Docker Compose | Full | Multi-service orchestration with `docker-compose.yml` |
| Kubernetes (Helm) | Full | `renos/renos` Helm chart v1.0.0 |
| Native Linux (deb/rpm/apk) | Full | Systemd integration |
| Native macOS (Homebrew/pkg) | Full | Launchd integration |
| Native Windows (MSI) | Full | Windows Service integration |

---

## Databases

### PostgreSQL

| Version | Support Level | Notes |
|---------|--------------|-------|
| 15 | Full | Recommended minimum |
| 16 | Full | Tested; use matching `pg_dump`/`pg_restore` version |
| 14 | Not supported | Schema migrations require PG 15+ features |
| 13 | Not supported | |

### Redis

| Version | Support Level | Notes |
|---------|--------------|-------|
| 6.x | Full | Single instance and Cluster mode |
| 7.x | Full | Single instance and Cluster mode |
| 6.x Sentinel | Untested | See [KI-005](./known-issues.md#ki-005-redis-sentinel-mode-untested) |
| 7.x Sentinel | Untested | See [KI-005](./known-issues.md#ki-005-redis-sentinel-mode-untested) |

### NATS

| Version | Support Level | Notes |
|---------|--------------|-------|
| 2.9 | Full | Single server and cluster |
| 2.10 | Full | Single server and cluster |
| Multi-region (leaf nodes) | Manual | See [KI-006](./known-issues.md#ki-006-multi-region-nats-requires-manual-configuration) |

---

## Browsers (Hub UI)

| Browser | Minimum Version | Notes |
|---------|----------------|-------|
| Chrome | 100+ | Fully tested |
| Firefox | 100+ | Fully tested |
| Safari | 15.4+ | Fully tested |
| Edge | 100+ | Chromium-based; tested |
| Opera | 80+ | Community-tested |

### Mobile Browsers

| Browser | Notes |
|---------|-------|
| Chrome for Android | View-only; dashboard responsive |
| Safari iOS | View-only; limited interactive testing |

---

## Agent Platforms

| Platform | Arch | Support Level | Notes |
|----------|------|--------------|-------|
| Linux (glibc 2.28+) | amd64 | Full | Primary target |
| Linux (glibc 2.28+) | arm64 | Full | |
| Linux (glibc 2.28+) | armv7 | Beta | Limited testing |
| Linux (musl) | amd64 | Full | Alpine-based images |
| macOS 12+ | amd64 | Full | Intel |
| macOS 14+ | arm64 | Full | Apple Silicon |
| Windows Server 2019+ | amd64 | Full | |
| Windows 10+ | amd64 | Full | |
| Windows 10+ | arm64 | Experimental | Not production-recommended |

---

## Hub / Core Platforms

| Platform | Support Level | Notes |
|----------|--------------|-------|
| Linux (glibc 2.28+) | Full | amd64, arm64 |
| Linux (musl) | Full | Alpine container images |
| macOS (dev only) | Dev | Not for production control-plane |
| Windows (dev only) | Dev | Not for production control-plane |

---

## Container Runtimes

| Runtime | Support Level | Notes |
|---------|--------------|-------|
| Docker 24+ | Full | |
| containerd 1.6+ | Full | Used by Kubernetes |
| CRI-O 1.25+ | Full | |
| Podman 4.0+ | Full | Rootless tested |

---

## Kubernetes

| Component | Supported Versions | Notes |
|-----------|-------------------|-------|
| Kubernetes | 1.25, 1.26, 1.27, 1.28, 1.29, 1.30 | |
| Helm | 3.10+ | |
| Ingress Controller (NGINX) | 1.8+ | |
| Ingress Controller (Traefik) | 2.9+ | Community-tested |
| Ingress Controller (HAProxy) | 2.6+ | Community-tested |
| cert-manager | 1.12+ | TLS certificate provisioning |
| Prometheus Operator | 0.62+ | Metrics collection |
| cert-manager CSI driver | 1.12+ | Agent mTLS certificates |

---

## Storage Classes

| Storage Type | Provisioner | Notes |
|-------------|-------------|-------|
| SSD (ext4/xfs) | Any CSI driver | Recommended for PostgreSQL |
| Network (NFS) | nfs-csi | Tested; slower performance expected |
| Cloud (EBS) | ebs.csi.aws.com | gp3 recommended |
| Cloud (Persistent Disk) | pd.csi.storage.gke.io | pd-ssd recommended |
| Cloud (Azure Disk) | disk.csi.azure.com | Premium SSD recommended |
| Local PV | local.csi.kubernetes.io | Best performance; node-pinned |

---

## TLS Options

| Option | Support Level | Notes |
|--------|--------------|-------|
| Auto (cert-manager + Let's Encrypt) | Full | Default in Helm chart |
| Bring your own certificate | Full | Via Helm values |
| Internal CA (self-signed) | Full | For air-gapped/internal networks |
| mTLS (control-plane ↔ agent) | Full | Default in v1.0.0 |
| TLS termination at ingress | Full | Recommended for production |

---

## AI Providers (Plugin Ecosystem)

| Provider | Support Level | Notes |
|----------|--------------|-------|
| OpenAI | Full | Plugin available in marketplace |
| Anthropic Claude | Full | Plugin available in marketplace |
| Google Gemini | Full | Plugin available in marketplace |
| Azure OpenAI | Full | Community plugin |
| AWS Bedrock | Full | Community plugin |
| Ollama (local) | Full | Open-source plugin |
| Hugging Face | Beta | Experimental plugin |
| Cohere | Beta | Community plugin |
| Custom (OpenAI-compatible) | Full | BYO endpoint |

---

## Version Support Policy

- **Full Support**: Tested in CI, documented, and covered by SLAs.
- **Beta**: Feature-complete but limited production testing; feedback requested.
- **Experimental**: Proof-of-concept; may have breaking changes without notice.
- **Manual**: Works with manual configuration; no automation provided.
- **Not supported**: Known to not work; do not use in production.

For questions about supported configurations, contact support@renos.io or open a GitHub discussion.
