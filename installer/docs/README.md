# RenOS Installer Documentation

## Overview

RenOS is a comprehensive device management platform that enables organizations to deploy, monitor, and manage devices at scale. The installer platform provides multiple deployment methods to accommodate any infrastructure — from single-server evaluations to multi-region, air-gapped production clusters.

### What is RenOS?

RenOS provides:
- **Device management** — Enroll, configure, monitor, and update devices remotely
- **Workflow automation** — Create automated workflows for device operations
- **Real-time messaging** — Powered by NATS JetStream for reliable device communication
- **Secure enrollment** — Certificate-based device authentication and TLS encryption
- **Scalable architecture** — From single node to multi-region clusters
- **Extensible platform** — Plugin system and marketplace for custom functionality

### Installer Platform Components

| Component | Description | Location |
|-----------|-------------|----------|
| **Linux Installer** | Bash-based installer for 7 Linux distributions | `linux/install.sh` |
| **Docker Compose** | Production Docker Compose configuration | `docker/docker-compose.yml` |
| **Helm Chart** | Kubernetes Helm chart for cluster deployment | `kubernetes/` |
| **macOS PKG Builder** | Script to build signed macOS installer packages | `macos/` |
| **Windows Installer** | MSI installer for Windows Agent | `windows/` |
| **Backup System** | Comprehensive backup and restore scripts | `backup/` |
| **Upgrade Manager** | Go-based version management and upgrade orchestration | `upgrade/` |
| **CLI Wizard** | Interactive installation wizard | `wizard/` |
| **Validation Suite** | Installation validation and health checks | `validate/` |

### Architecture Overview

```
                    ┌─────────────────────────────────────┐
                    │         Load Balancer / Ingress     │
                    └────────────────┬────────────────────┘
                                     │
              ┌──────────────────────┼──────────────────────┐
              │                      │                      │
              ▼                      ▼                      ▼
     ┌────────────────┐   ┌────────────────┐   ┌────────────────┐
     │   RenOS Core   │   │   RenOS Hub    │   │  RenOS Agent   │
     │   (API/Orch)   │   │   (Web UI)     │   │  (Device Conn) │
     └───────┬────────┘   └────────────────┘   └────────────────┘
              │
     ┌────────┼────────┐
     │        │        │
     ▼        ▼        ▼
  ┌──────┐ ┌──────┐ ┌──────┐
  │PostgreSQL│ Redis │ NATS │
  │   DB  │ │Cache │ │Msg   │
  └──────┘ └──────┘ └──────┘
```

### Version Compatibility

| RenOS Version | Chart Version | App Version | Status |
|--------------|--------------|-------------|--------|
| 0.9.x | 0.0.9 | 0.9.0-beta | End of life |
| **1.0.0-rc1** | **0.1.0** | **1.0.0-rc1** | **Current** |
| 1.0.0 | TBD | TBD | Upcoming |
| 1.1.0 | TBD | TBD | Planned |
| 2.0.0 | TBD | TBD | Planned |

### Platform Support Matrix

| Deployment Type | Development | Staging | Production | Air-Gapped |
|----------------|-------------|---------|------------|------------|
| Linux native | ✓ | ✓ | ✓ | ✓ |
| Docker Compose | ✓ | ✓ | ✓ | ✓ |
| Kubernetes (Helm) | ✓ | ✓ | ✓ | ✓ |
| macOS (PKG) | ✓ | ✓ | ✓ | ✓ |
| Windows (MSI) | ✓ | ✓ | ✓ | ✓ |
| Multi-node cluster | — | ✓ | ✓ | ✓ |
| Multi-AZ cluster | — | — | ✓ | ✓ |

---

## Quick Links

### Installation Guides

| Guide | Description | Audience |
|-------|-------------|----------|
| [Installation Guide](./installation-guide.md) | Complete installation instructions for all platforms | All users |
| [Docker Deployment](./installation-guide.md#docker-installation) | Docker Compose deployment details | Docker users |
| [Kubernetes Deployment](./kubernetes-deployment.md) | Kubernetes/Helm deployment reference | Kubernetes users |
| [Cluster Deployment](./cluster-deployment.md) | Multi-node cluster architecture and setup | Cluster operators |
| [Air-Gapped Deployment](./air-gapped-deployment.md) | Offline/air-gapped deployment procedures | Air-gapped environments |

### Operations Guides

| Guide | Description | Audience |
|-------|-------------|----------|
| [Upgrade Guide](./upgrade-guide.md) | Version upgrade procedures and compatibility | All operators |
| [Recovery Guide](./recovery-guide.md) | Backup, restore, and disaster recovery | All operators |
| [Disaster Recovery Plan](./disaster-recovery.md) | Formal DR plan with RPO/RTO, drills, procedures | Cluster operators |

---

## Quick Start

### Single Command (Linux)

```bash
curl -fsSL https://install.renos.ai | sudo bash
```

### Docker Compose

```bash
git clone https://github.com/renos/renos.git
cd renos/installer/docker
cp .env.example .env
# Edit .env with your values
docker compose up -d
```

### Kubernetes (Helm)

```bash
helm repo add renos https://helm.renos.ai
helm repo update
helm install renos renos/renos \
  --namespace renos --create-namespace \
  --set core.env.RENOS_ADMIN_EMAIL=admin@example.com \
  --set core.env.RENOS_ADMIN_PASSWORD=changeme123
```

---

## Installer Components Summary

### Linux Installer (`linux/install.sh`)

A comprehensive bash script that automates RenOS installation on seven major Linux distributions.

**Supported Distributions:**

| Distribution | Versions | Package Manager | Architecture |
|-------------|----------|----------------|--------------|
| Ubuntu | 20.04 LTS, 22.04 LTS, 24.04 LTS | apt | amd64, arm64 |
| Debian | 11, 12 | apt | amd64, arm64 |
| Fedora | 38, 39, 40 | dnf | amd64, arm64 |
| RHEL | 8.x, 9.x | dnf/yum | amd64, arm64 |
| Rocky Linux | 8.x, 9.x | dnf | amd64, arm64 |
| AlmaLinux | 8.x, 9.x | dnf | amd64, arm64 |
| Arch Linux | Rolling | pacman | amd64, arm64 |

**Key Features:**
- Automatic distribution detection
- Docker Engine and Docker Compose installation
- Firewall configuration (UFW, firewalld, iptables)
- TLS setup (self-signed, Let's Encrypt)
- Systemd service integration
- Log rotation configuration
- Restore point creation before installation
- Uninstall capability with cleanup
- Dry-run mode for preview
- Environment variable overrides for non-interactive use

**Usage:**
```bash
# Interactive installation
sudo bash install.sh

# Non-interactive with environment variables
sudo RENOS_ADMIN_EMAIL=admin@example.com \
     RENOS_ADMIN_PASSWORD=secret \
     RENOS_HOSTNAME=renos.example.com \
     RENOS_TLS_MODE=letsencrypt \
     bash install.sh

# Preview mode
sudo bash install.sh --dry-run

# Uninstall
sudo bash install.sh --uninstall
```

### Docker Compose (`docker/docker-compose.yml`)

Production-grade Docker Compose configuration with full service stack.

**Services:**

| Service | Image | Function |
|---------|-------|----------|
| `postgres` | postgres:16-alpine | Primary database |
| `redis` | redis:7-alpine | Cache and session storage |
| `nats` | nats:2.10-alpine | Message broker with JetStream |
| `renos-core` | renos/core | Core API and orchestration |
| `renos-hub` | renos/hub | Web UI dashboard |
| `renos-agent` | renos/agent | Device agent |
| `nginx` | nginx:1.27-alpine | Reverse proxy and TLS termination |

**Features:**
- Health checks on all services
- Resource limits and reservations
- Named volumes for data persistence
- Internal and external network segmentation
- Privileged mode for device management
- Environment variable configuration

### Kubernetes Helm Chart (`kubernetes/`)

Full-featured Helm chart for Kubernetes deployment.

**Chart Details:**
- **Version**: 0.1.0
- **App Version**: 1.0.0-rc1
- **Min K8s Version**: 1.24+
- **Min Helm Version**: 3.8+

**Templates:**
| Template | Component | Description |
|----------|-----------|-------------|
| `namespace.yaml` | — | Kubernetes namespace |
| `serviceaccount.yaml` | — | RBAC service account |
| `secrets.yaml` | — | Secret management |
| `configmap.yaml` | — | Configuration |
| `postgres-statefulset.yaml` | PostgreSQL | Database with persistent storage |
| `redis-statefulset.yaml` | Redis | Cache with persistence |
| `nats-statefulset.yaml` | NATS | Message broker cluster |
| `core-deployment.yaml` | Core | API server deployment |
| `hub-deployment.yaml` | Hub | Web UI deployment |
| `agent-daemonset.yaml` | Agent | Device agent on all nodes |
| `ingress.yaml` | — | Ingress configuration |
| `backup-cronjob.yaml` | — | Automated database backups |
| `hpa.yaml` | — | Horizontal pod autoscaling |
| `pdb.yaml` | — | Pod disruption budgets |
| `networkpolicy.yaml` | — | Network security policies |
| `servicemonitor.yaml` | — | Prometheus metrics integration |

### macOS PKG Builder (`macos/`)

Tooling to build signed macOS installer packages for the RenOS Agent.

**Components:**
- `build-pkg.sh` — Build script for PKG creation
- `Distribution.xml` — Product distribution definition
- `postinstall.sh` — Post-installation configuration
- `com.renos.agent.plist` — LaunchAgent for automatic startup
- `keychain-setup.sh` — macOS Keychain integration

**Requirements:**
- macOS 12+
- Xcode Command Line Tools
- Developer ID Installer certificate (for signed builds)

### Backup System (`backup/`)

Comprehensive backup and restore solution.

**Components:**

| Script | Purpose |
|--------|---------|
| `renos-backup.sh` | Backup orchestrator |
| `renos-backup-config.sh` | Backup configuration |

**Backup Modes:**
| Mode | Components Backed Up |
|------|---------------------|
| `--full` | All components |
| `--db-only` | Database only |
| `--files-only` | File-based data only (skip DB/Redis) |

**Backup Components:**
- PostgreSQL database dump
- Redis RDB snapshot
- Vault secrets
- Plugin configurations
- Marketplace data
- User data (devices, workflows, schedules)
- Workflow history
- CA certificates and TLS certs
- Configuration files

**Storage Backends:**
- Local disk
- Amazon S3 (via rclone or aws CLI)
- SCP/SSH remote host
- Local network path

**Security:**
- age encryption support
- GPG encryption support
- SHA256 checksum verification
- Manifest tracking

### Upgrade Manager (`upgrade/`)

Go-based version management system.

**Key Files:**
| File | Purpose |
|------|---------|
| `state.go` | Upgrade state tracking (JSON-based) |
| `compat.go` | Version compatibility checking |

**Features:**
- Version parsing and comparison
- Component dependency tracking
- Breaking change detection
- Upgrade history with rollback support
- Backup ID tracking per upgrade
- Component-specific version tracking

### Installation Paths

| Path | Purpose | Created By |
|------|---------|------------|
| `/etc/renos/` | Configuration files | Installer |
| `/etc/renos/renos.env` | Environment variables | Installer |
| `/etc/renos/tls/` | TLS certificates | Installer |
| `/var/lib/renos/` | Application data | Installer |
| `/var/lib/renos/data/` | Core data | Installer |
| `/var/lib/renos/ca/` | Certificate authority | Installer |
| `/var/lib/renos/plugins/` | Plugin storage | Installer |
| `/var/lib/renos/backups/` | Backup storage | Backup system |
| `/var/log/renos/` | Log files | Installer |
| `/etc/systemd/system/renos-*.service` | Systemd services | Installer |
| `/etc/logrotate.d/renos` | Log rotation config | Installer |
| `/var/lib/renos/upgrade-state.json` | Upgrade state | Upgrade manager |

### Ports Used

| Port | Service | Protocol | Required |
|------|---------|----------|----------|
| 80 | HTTP (redirect) | TCP | External |
| 443 | HTTPS (UI + API) | TCP | External |
| 4222 | NATS client | TCP | Internal |
| 5432 | PostgreSQL | TCP | Internal |
| 6379 | Redis | TCP | Internal |
| 6222 | NATS cluster | TCP | Internal |
| 8222 | NATS monitoring | TCP | Internal |
| 3000 | Hub web UI | TCP | Internal |
| 8080 | Core API (K8s) | TCP | Internal |
| 8443 | Core API (Docker) | TCP | Internal |
| 8444 | Core gRPC | TCP | Internal |
| 9090 | Core metrics | TCP | Internal |
| 9091 | Hub metrics | TCP | Internal |

---

## Getting Started

### New Users

1. **Read the [Installation Guide](./installation-guide.md)** for your platform
2. Choose your deployment method:
   - **Single server**: Use the [Linux installer](./installation-guide.md#linux-installation)
   - **Docker**: Use [Docker Compose](./installation-guide.md#docker-installation)
   - **Kubernetes**: Use the [Helm chart](./kubernetes-deployment.md)
3. Configure [backups](./recovery-guide.md#creating-backups)
4. Plan your [upgrade strategy](./upgrade-guide.md)
5. Prepare for [disaster recovery](./disaster-recovery.md)

### Cluster Operators

1. Review the [Cluster Deployment Guide](./cluster-deployment.md) for architecture options
2. Set up [NATS clustering](./cluster-deployment.md#nats-cluster-configuration)
3. Configure [database replication](./cluster-deployment.md#database-replication)
4. Set up [load balancing](./cluster-deployment.md#load-balancing)
5. Configure [monitoring](./cluster-deployment.md#monitoring-cluster-health)
6. Establish [backup strategies](./cluster-deployment.md#cluster-backup-strategies)

### Air-Gapped Environments

1. Read the [Air-Gapped Deployment Guide](./air-gapped-deployment.md)
2. Prepare an internet-connected download workstation
3. Download all [artifacts](./air-gapped-deployment.md#downloading-artifacts)
4. Transfer via your approved [mechanism](./air-gapped-deployment.md#transfer-mechanisms)
5. Set up a [private registry](./air-gapped-deployment.md#private-registry-setup)
6. Deploy and [validate](./air-gapped-deployment.md#no-internet-validation)

---

## Support and Resources

| Resource | Link |
|----------|------|
| Documentation | https://docs.renos.ai |
| GitHub Repository | https://github.com/renos/renos |
| Issue Tracker | https://github.com/renos/renos/issues |
| Community Forum | https://community.renos.ai |
| Email Support | support@renos.ai |
| Status Page | https://status.renos.ai |
| Release Notes | https://github.com/renos/renos/releases |

### Reporting Issues

When reporting issues, please include:

1. RenOS version: `grep RENOS_INSTALL_VERSION /etc/renos/renos.env`
2. Installation method: native, Docker, or Kubernetes
3. Operating system and version
4. Deployment type: single node or cluster
5. Steps to reproduce the issue
6. Expected vs actual behavior
7. Relevant logs and error messages

### Security Disclosures

To report a security vulnerability, please email **security@renos.ai** rather than opening a public issue.

---

## Environment Variables Reference

### Core Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `POSTGRES_USER` | Yes | `renos` | PostgreSQL database user |
| `POSTGRES_PASSWORD` | Yes | (auto-generated) | PostgreSQL password |
| `POSTGRES_DB` | Yes | `renos` | PostgreSQL database name |
| `RENOS_DATABASE_URL` | Yes | (computed) | Full database connection URL |
| `RENOS_NATS_URL` | Yes | `nats://nats:4222` | NATS server URL |
| `RENOS_REDIS_URL` | Yes | (computed) | Redis connection URL |
| `RENOS_JWT_SECRET` | Yes | (auto-generated) | 256-bit hex JWT signing key |
| `RENOS_ENCRYPTION_KEY` | Yes | (auto-generated) | 256-bit hex encryption key |
| `RENOS_ADMIN_EMAIL` | Yes | `admin@renos.local` | Initial admin email |
| `RENOS_ADMIN_PASSWORD` | Yes | (prompted) | Initial admin password |
| `RENOS_EXTERNAL_URL` | Yes | `https://(hostname)` | Public-facing URL |
| `RENOS_PORT` | No | `8443` | Core API internal port |
| `RENOS_GRPC_PORT` | No | `8444` | Core gRPC internal port |
| `RENOS_LOG_LEVEL` | No | `info` | Log level (debug/info/warn/error) |
| `RENOS_CA_CERT_DIR` | No | `/var/lib/renos/ca` | CA certificate storage |
| `RENOS_DATA_DIR` | No | `/var/lib/renos/data` | Application data storage |
| `RENOS_HTTP_PORT` | No | `80` | HTTP redirect port |
| `RENOS_HTTPS_PORT` | No | `443` | HTTPS service port |
| `RENOS_TLS_MODE` | No | `self-signed` | TLS mode (self-signed/letsencrypt/none) |

### Docker-Specific Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `REDIS_PASSWORD` | Yes | (auto-generated) | Redis authentication password |
| `RENOS_USE_DOCKER_DB` | No | `true` | Use Docker-based databases |
| `RENOS_VERSION` | No | `latest` | RenOS image version tag |
| `RENOS_LOG_FORMAT` | No | `text` | Log format (text/json) |
| `NODE_ENV` | No | `production` | Node.js environment |
| `NEXT_TELEMETRY_DISABLED` | No | `1` | Disable Next.js telemetry |

## Troubleshooting Quick Reference

### Common Commands

```bash
# Check service status
systemctl status renos-core

# View logs
journalctl -u renos-core -f
docker compose logs -f renos-core

# Restart a service
systemctl restart renos-core
docker compose restart renos-core

# Database connection test
docker compose exec postgres pg_isready -U renos

# Redis connection test
docker compose exec redis redis-cli ping

# NATS health check
docker compose exec nats wget -q http://localhost:8222/healthz -O /dev/null

# API health check
curl -sk https://localhost/api/v1/healthz

# Create a backup
sudo renos-backup.sh --full

# List backups
sudo renos-backup.sh --list

# Check upgrade state
renos-upgrade status
```

### Quick Diagnostic

```bash
#!/bin/bash
# renos-quick-diag.sh

echo "=== RenOS Quick Diagnostic ==="
echo "Date: $(date)"
echo ""

echo "--- System ---"
uname -a
docker version --format '{{.Server.Version}}' 2>/dev/null || echo "Docker: not found"
docker compose version --short 2>/dev/null || echo "Compose: not found"

echo ""
echo "--- Services ---"
for svc in renos-core renos-hub renos-agent renos-nginx; do
  systemctl is-active "$svc" 2>/dev/null && echo "$svc: running" || echo "$svc: stopped"
done

echo ""
echo "--- Containers ---"
docker compose -f /etc/renos/docker-compose.yml ps 2>/dev/null || echo "No compose file found"

echo ""
echo "--- Disk ---"
df -h /var/lib/renos /var/lib/docker 2>/dev/null

echo ""
echo "--- Memory ---"
free -h

echo ""
echo "=== Diagnostic Complete ==="
```

## File Reference

### Configuration Files

| File | Purpose | Format |
|------|---------|--------|
| `/etc/renos/renos.env` | Main environment configuration | Key=Value |
| `/etc/renos/docker-compose.yml` | Docker Compose service definition | YAML |
| `/etc/renos/nginx.conf` | Nginx reverse proxy configuration | Nginx config |
| `/etc/renos/tls/fullchain.pem` | TLS certificate chain | PEM |
| `/etc/renos/tls/privkey.pem` | TLS private key | PEM |
| `/etc/renos/backup-config.sh` | Backup system configuration | Bash |
| `/var/lib/renos/upgrade-state.json` | Upgrade state tracking | JSON |
| `/var/log/renos/install.log` | Installation log | Text |

### Log Files

| Log File | Source | Rotation |
|----------|--------|----------|
| `/var/log/renos/install.log` | Installer | Manual |
| `/var/log/renos/core.log` | Core service | Daily, 30 days |
| `/var/log/renos/hub.log` | Hub service | Daily, 30 days |
| `/var/log/renos/agent.log` | Agent service | Daily, 30 days |
| `/var/log/nginx/access.log` | Nginx access | Daily, 30 days |
| `/var/log/nginx/error.log` | Nginx error | Daily, 30 days |
| `/var/log/renos/backup.log` | Backup system | Per-backup |

### systemd Service Files

| Service | Description | Dependencies |
|---------|-------------|--------------|
| `renos-core.service` | RenOS Core API | docker.service, network-online.target |
| `renos-hub.service` | RenOS Hub Web UI | renos-core.service |
| `renos-agent.service` | RenOS Device Agent | renos-core.service |
| `renos-nginx.service` | RenOS Nginx Proxy | renos-core.service, renos-hub.service |

## Glossary

| Term | Definition |
|------|------------|
| **Agent** | Software component running on managed devices for remote management |
| **Air-Gapped** | Network physically isolated from the internet |
| **Core** | Central RenOS API server handling orchestration, data, and device management |
| **DR** | Disaster Recovery — processes for restoring service after catastrophic failure |
| **Helm** | Kubernetes package manager used to deploy the RenOS chart |
| **Hub** | Web-based dashboard UI for managing RenOS |
| **JetStream** | NATS persistence layer for reliable message delivery |
| **Leader** | Primary node in a cluster handling write operations |
| **NATS** | Cloud-native messaging system used for device communication |
| **PDB** | PodDisruptionBudget — Kubernetes policy ensuring minimum available pods |
| **RPO** | Recovery Point Objective — maximum acceptable data loss |
| **RTO** | Recovery Time Objective — maximum acceptable downtime |
| **Sentinel** | Redis high-availability monitoring and automatic failover |
| **StatefulSet** | Kubernetes resource for stateful applications with persistent storage |
| **TLS** | Transport Layer Security — encryption for network communication |

## Documentation Version

| Metadata | Value |
|----------|-------|
| **Document Version** | 1.0.0-rc1 |
| **RenOS Version** | 1.0.0-rc1 |
| **Last Updated** | 2025-01-01 |
| **Author** | RenOS Documentation Team |
