# RenOS Installation Guide

## Table of Contents

- [System Requirements](#system-requirements)
- [Supported Platforms](#supported-platforms)
- [Quick Start](#quick-start)
- [Linux Installation](#linux-installation)
- [Windows Installation](#windows-installation)
- [macOS Installation](#macos-installation)
- [Docker Installation](#docker-installation)
- [Kubernetes Installation (Helm)](#kubernetes-installation-helm)
- [Air-Gapped Installation](#air-gapped-installation)
- [Post-Installation Steps](#post-installation-steps)
- [Troubleshooting Common Issues](#troubleshooting-common-issues)
- [Getting Help](#getting-help)

---

## System Requirements

### Minimum Requirements

| Component | Requirement |
|-----------|------------|
| **CPU** | 2 cores (x86_64 or arm64) |
| **RAM** | 4 GB |
| **Disk** | 20 GB free space (SSD recommended) |
| **OS** | Linux kernel 5.x+, Windows Server 2019+, macOS 12+ |
| **Network** | Static IP or DNS hostname, outbound internet for telemetry |

### Recommended Requirements

| Component | Requirement |
|-----------|------------|
| **CPU** | 4 cores |
| **RAM** | 8 GB |
| **Disk** | 50 GB SSD |
| **Network** | 1 Gbps, low-latency connection to managed devices |
| **Bandwidth** | 100 Mbps minimum |

### Production Requirements

| Component | Requirement |
|-----------|------------|
| **CPU** | 8 cores (or 4 cores + dedicated PostgreSQL host) |
| **RAM** | 16 GB |
| **Disk** | 100 GB+ NVMe SSD (for databases and NATS JetStream) |
| **Database** | PostgreSQL 16 (included) |
| **Cache** | Redis 7 (included) |
| **Messaging** | NATS 2.10 with JetStream (included) |
| **Network** | Redundant 10 Gbps, dedicated management VLAN |

### Port Requirements

| Port | Protocol | Service | Direction |
|------|----------|---------|-----------|
| 80 | TCP | HTTP (redirect to HTTPS) | Inbound |
| 443 | TCP | HTTPS (RenOS Hub UI) | Inbound |
| 8443 | TCP | RenOS Core API (gRPC) | Inbound |
| 8444 | TCP | RenOS Core gRPC internal | Internal |
| 4222 | TCP | NATS client connections | Internal |
| 6222 | TCP | NATS cluster routing | Internal |
| 8222 | TCP | NATS monitoring | Internal |
| 5432 | TCP | PostgreSQL | Internal |
| 6379 | TCP | Redis | Internal |
| 9090 | TCP | Core metrics (Prometheus) | Internal |
| 9091 | TCP | Hub metrics (Prometheus) | Internal |

> **Note:** For single-server deployments, only ports 80 and 443 need to be exposed to the network. All other ports communicate over internal Docker networks.

### Software Dependencies

| Software | Version | Required For |
|----------|---------|--------------|
| Docker | 24.x+ | All deployments |
| Docker Compose | v2.24+ | Docker-based install |
| Helm | 3.8+ | Kubernetes install |
| Kubernetes | 1.24+ | Kubernetes install |
| OpenSSL | 1.1.1+ | TLS certificate generation |
| PostgreSQL client | 16.x | Database operations |
| Redis CLI | 7.x | Cache operations |

---

## Supported Platforms

### Linux Distributions

| Distribution | Version | Architecture | Package Manager | Status |
|-------------|---------|--------------|-----------------|--------|
| Ubuntu | 20.04 LTS, 22.04 LTS, 24.04 LTS | amd64, arm64 | apt | Fully supported |
| Debian | 11 (Bullseye), 12 (Bookworm) | amd64, arm64 | apt | Fully supported |
| Fedora | 38, 39, 40 | amd64, arm64 | dnf | Fully supported |
| RHEL | 8.x, 9.x | amd64, arm64 | dnf/yum | Fully supported |
| Rocky Linux | 8.x, 9.x | amd64, arm64 | dnf | Fully supported |
| AlmaLinux | 8.x, 9.x | amd64, arm64 | dnf | Fully supported |
| Arch Linux | Rolling | amd64, arm64 | pacman | Community supported |

### Windows

| Version | Architecture | Installer Type | Status |
|---------|-------------|---------------|--------|
| Windows Server 2019 | x86_64 | MSI | Supported |
| Windows Server 2022 | x86_64 | MSI | Supported |
| Windows 10 (21H2+) | x86_64 | MSI | Supported |
| Windows 11 | x86_64, arm64 | MSI | Supported |

### macOS

| Version | Architecture | Installer Type | Status |
|---------|-------------|---------------|--------|
| macOS 12 Monterey | x86_64, arm64 | PKG | Supported |
| macOS 13 Ventura | x86_64, arm64 | PKG | Supported |
| macOS 14 Sonoma | x86_64, arm64 | PKG | Supported |
| macOS 15 Sequoia | x86_64, arm64 | PKG | Supported |

### Container Platforms

| Platform | Version | Deployment Method |
|----------|---------|-------------------|
| Docker | 24.x+ | docker compose |
| Kubernetes | 1.24+ | Helm chart |
| Amazon EKS | 1.24+ | Helm chart |
| Azure AKS | 1.24+ | Helm chart |
| Google GKE | 1.24+ | Helm chart |
| Rancher RKE2 | 1.24+ | Helm chart |
| OpenShift | 4.12+ | Helm chart (with SCC) |

---

## Quick Start

### Single-Command Install (Linux)

```bash
curl -fsSL https://install.renos.ai | sudo bash
```

### Docker Quick Start

```bash
# Clone the repository
git clone https://github.com/renos/renos.git
cd renos/installer/docker

# Configure environment
cp .env.example .env
# Edit .env with your values

# Start RenOS
docker compose up -d
```

### Helm Quick Start

```bash
helm repo add renos https://helm.renos.ai
helm repo update
helm install renos renos/renos \
  --namespace renos --create-namespace \
  --set global.env.RENOS_ADMIN_EMAIL=admin@example.com \
  --set global.env.RENOS_ADMIN_PASSWORD=changeme123
```

---

## Linux Installation

### Native Installation

The Linux installer script supports seven major distributions and handles all aspects of installation including Docker setup, database configuration, TLS certificates, systemd services, and firewall rules.

#### Prerequisites

- Root or sudo access
- Outbound internet connectivity (for package downloads)
- A hostname resolving to the server's IP address
- 20 GB+ free disk space

#### Installation Steps

**Step 1: Download the installer**

```bash
curl -fsSL https://install.renos.ai -o install.sh
# Or from the repository
curl -fsSL https://raw.githubusercontent.com/renos/renos/main/installer/linux/install.sh -o install.sh
```

**Step 2: Review the script**

```bash
less install.sh
sha256sum install.sh
```

**Step 3: Run the installer**

```bash
sudo bash install.sh
```

The installer will guide you through:

1. **Distribution detection** - Automatically identifies your OS and package manager
2. **Docker installation** - Installs Docker Engine and Docker Compose if not present
3. **Firewall configuration** - Opens ports 80, 443, and 8443
4. **System user creation** - Creates `renos` user and group
5. **Directory creation** - Sets up `/etc/renos`, `/var/lib/renos`, `/var/log/renos`
6. **Database selection** - Choose Docker-based or native PostgreSQL/Redis/NATS
7. **TLS setup** - Self-signed (testing), Let's Encrypt (production), or none
8. **Configuration generation** - Random secrets, JWT tokens, encryption keys
9. **Service creation** - systemd services for core, hub, agent, nginx
10. **Validation** - Checks all components are running

**Step 4: Access the dashboard**

```
https://your-server-hostname
```

#### Non-Interactive Installation

For automated deployments, all prompts can be bypassed via environment variables:

```bash
sudo RENOS_ADMIN_EMAIL=admin@example.com \
     RENOS_ADMIN_PASSWORD=SuperS3cret! \
     RENOS_HOSTNAME=renos.example.com \
     RENOS_TLS_MODE=letsencrypt \
     RENOS_USE_DOCKER_DB=true \
     bash install.sh
```

#### Dry Run Mode

Preview all changes without applying them:

```bash
sudo bash install.sh --dry-run
```

#### Installation Log

The installer writes a detailed log to `/var/log/renos/install.log`. Review this file if any step fails.

### Docker Installation

See the [Docker Installation](#docker-installation) section below for detailed Docker deployment instructions.

### Kubernetes Installation

See the [Kubernetes Installation](#kubernetes-installation-helm) section below for detailed Kubernetes/Helm deployment instructions.

### Uninstalling RenOS

```bash
sudo bash install.sh --uninstall
```

The uninstaller will:
1. Stop and disable all RenOS services
2. Remove Docker containers and volumes
3. Remove configuration files and directories
4. Remove the `renos` system user
5. Optionally remove Docker, PostgreSQL, and Redis

> **Warning:** The uninstaller will prompt for confirmation. Backups are created at `/var/backups/renos/pre-install/` before uninstallation.

---

## Windows Installation

### MSI Installer Walkthrough

The RenOS Agent for Windows is distributed as an MSI package. The Windows installation currently supports the Agent component only; for the full server stack, use Docker Desktop or WSL2 with the Linux installer.

#### Prerequisites

- Windows 10 21H2+ or Windows Server 2019+
- Administrative privileges
- 2 GB RAM minimum, 10 GB free disk space
- Docker Desktop (optional, for local testing)

#### Installation Steps

**Step 1: Download the MSI**

Download from the RenOS releases page:
```
https://github.com/renos/renos/releases/latest/download/RenOS-Agent-x.x.x.msi
```

**Step 2: Run the installer**

Double-click the MSI file or run from command line:

```powershell
msiexec /i RenOS-Agent-x.x.x.msi /qb
```

**Step 3: Welcome screen**

The installer welcome screen displays the RenOS Agent version and a brief description. Click **Next** to continue.

**Step 4: License agreement**

Read and accept the End User License Agreement (EULA). Click **I Accept** then **Next**.

**Step 5: Installation options**

- **Installation path**: Default is `C:\Program Files\RenOS\Agent\`
- **Server URL**: Enter your RenOS server URL (e.g., `https://renos.example.com`)
- **Enrollment token**: (Optional) Pre-configured enrollment token
- **Start on boot**: Enable to start the agent automatically with Windows

**Step 6: Ready to install**

Review your selections and click **Install**. The installation progress bar will show:

1. Copying files to installation directory
2. Registering the RenOS Agent Windows service
3. Configuring Windows Firewall rules
4. Running initial device enrollment

**Step 7: Completion**

Click **Finish** to exit the installer. The RenOS Agent service starts automatically.

#### Silent Installation

```powershell
msiexec /i RenOS-Agent-x.x.x.msi /qn /norestart ^
  RENOS_SERVER_URL=https://renos.example.com ^
  RENOS_ENROLLMENT_TOKEN=token-here ^
  INSTALLDIR=C:\RenOS
```

#### Verifying Installation

```powershell
# Check service status
Get-Service -Name RenOSAgent

# View logs
Get-Content "C:\ProgramData\RenOS\Agent\logs\agent.log" -Tail 50

# Test connection
& "C:\Program Files\RenOS\Agent\renos-agent.exe" --status
```

#### Uninstalling

```powershell
# Via Programs and Features, or:
msiexec /x RenOS-Agent-x.x.x.msi /qn
```

---

## macOS Installation

### PKG Installer Walkthrough

The RenOS Agent for macOS is distributed as a signed PKG package with native macOS integration.

#### Prerequisites

- macOS 12 Monterey or later
- Intel (x86_64) or Apple Silicon (arm64)
- 2 GB RAM, 10 GB free disk space
- Administrative privileges

#### Installation Walkthrough

**Step 1: Download the PKG**

```
https://github.com/renos/renos/releases/latest/download/RenOS-Agent-x.x.x.pkg
```

**Step 2: Open the package**

Double-click the PKG file in Finder. The installer will verify the package signature (if signed) and display the welcome screen.

**Step 3: Welcome**

Click **Continue** on the welcome screen, which displays the RenOS Agent version and a brief description of the agent's capabilities.

**Step 4: Read Me**

Review the important notes about the installation, including:
- The agent will be installed in `/Library/Application Support/RenOS/`
- A LaunchAgent will be installed for automatic startup
- The agent will attempt automatic enrollment if configured

**Step 5: License**

Review and accept the software license agreement.

**Step 6: Installation type**

- **Standard Install**: Installs the RenOS Agent with default configuration
- **Customize**: Choose individual components:
  - RenOS Agent core binary
  - LaunchAgent for automatic startup
  - Keychain integration for secure credential storage

**Step 7: Installation**

Enter your administrator username and password when prompted, then click **Install Software**. The installer will:

1. Copy binaries to `/Library/Application Support/RenOS/`
2. Install LaunchAgent at `/Library/LaunchAgents/com.renos.agent.plist`
3. Configure macOS Firewall to allow agent traffic
4. Store enrollment credentials in the System Keychain
5. Start the agent immediately

**Step 8: Complete**

Click **Close** to exit the installer.

#### Building the PKG from Source

```bash
# Prerequisites: Xcode Command Line Tools
xcode-select --install

# Build unsigned package
./installer/macos/build-pkg.sh

# Build signed package (requires Developer ID Installer certificate)
export PKG_SIGN_IDENTITY="Developer ID Installer: Your Name (TEAMID)"
export PKG_DEV_TEAM="TEAMID"
./installer/macos/build-pkg.sh
```

Output: `installer/macos/dist/RenOS-Agent-1.0.0.pkg`

#### Post-Install Behavior

The post-install script (`postinstall.sh`) performs:

1. Creates `/Library/Application Support/RenOS/` with subdirectories (certs, config, state)
2. Installs the LaunchAgent for automatic startup
3. Adds firewall rules via `socketfilterfw`
4. Registers enrollment token in the System Keychain
5. Writes default `config.json` with device metadata
6. Starts the LaunchAgent via `launchctl bootstrap`
7. Runs `renos-agent --enroll` for initial device enrollment

#### Verifying Installation

```bash
# Check if agent is running
launchctl print system/com.renos.agent

# View logs
tail -f /var/log/renos/agent.log

# Keychain credential check
./keychain-setup.sh get

# Agent status
sudo /Library/Application\ Support/RenOS/renos-agent --status
```

#### Uninstalling

```bash
# Stop and unload the agent
sudo launchctl bootout system /Library/LaunchAgents/com.renos.agent.plist

# Remove files
sudo rm -rf /Library/Application\ Support/RenOS/
sudo rm /Library/LaunchAgents/com.renos.agent.plist
sudo rm -rf /var/log/renos/

# Remove keychain entries
/Library/Application\ Support/RenOS/keychain-setup.sh delete
```

---

## Docker Installation

The Docker deployment uses Docker Compose to orchestrate the full RenOS stack including PostgreSQL, Redis, NATS, Core, Hub, Agent, and Nginx.

### Prerequisites

- Docker Engine 24.x+
- Docker Compose v2.24+ (built into Docker CLI)
- Git (to clone the repository)

### Directory Structure

```
installer/docker/
├── docker-compose.yml      # Production Docker Compose file
├── nginx.conf              # Nginx reverse proxy configuration
├── .env.example            # Environment variable template
└── tls/                    # TLS certificates (created during setup)
    ├── fullchain.pem
    └── privkey.pem
```

### Step-by-Step Deployment

**Step 1: Clone the repository**

```bash
git clone https://github.com/renos/renos.git
cd renos/installer/docker
```

**Step 2: Configure environment**

```bash
cp .env.example .env
```

Edit `.env` with your production values:

```bash
# Required settings
RENOS_EXTERNAL_URL=https://renos.example.com
POSTGRES_PASSWORD=your-strong-password-here
REDIS_PASSWORD=another-strong-password
RENOS_JWT_SECRET=your-256-bit-hex-secret
RENOS_ENCRYPTION_KEY=your-256-bit-hex-encryption-key
RENOS_ADMIN_EMAIL=admin@example.com
RENOS_ADMIN_PASSWORD=your-admin-password

# Optional settings
RENOS_LOG_LEVEL=info
RENOS_HTTP_PORT=80
RENOS_HTTPS_PORT=443
RENOS_VERSION=latest
```

**Step 3: Generate secure secrets**

```bash
# Generate random secrets
echo "POSTGRES_PASSWORD=$(openssl rand -base64 24)"
echo "REDIS_PASSWORD=$(openssl rand -base64 24)"
echo "RENOS_JWT_SECRET=$(openssl rand -hex 32)"
echo "RENOS_ENCRYPTION_KEY=$(openssl rand -hex 32)"
```

**Step 4: Set up TLS certificates**

For production, obtain valid TLS certificates:

```bash
# Option A: Let's Encrypt
sudo certbot certonly --standalone -d renos.example.com
cp /etc/letsencrypt/live/renos.example.com/fullchain.pem ./tls/
cp /etc/letsencrypt/live/renos.example.com/privkey.pem  ./tls/

# Option B: Self-signed (testing only)
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout ./tls/privkey.pem -out ./tls/fullchain.pem \
  -subj "/CN=renos.example.com"
```

**Step 5: Start RenOS**

```bash
docker compose up -d
```

**Step 6: Verify deployment**

```bash
# Check all containers are running
docker compose ps

# Check health endpoints
curl -k https://localhost/api/v1/healthz

# View logs
docker compose logs -f
```

### Docker Compose Services

| Service | Image | Purpose |
|---------|-------|---------|
| `postgres` | postgres:16-alpine | Primary database |
| `redis` | redis:7-alpine | Cache and session storage |
| `nats` | nats:2.10-alpine | Message broker with JetStream |
| `renos-core` | renos/core:latest | Core API and orchestration |
| `renos-hub` | renos/hub:latest | Web UI (Next.js) |
| `renos-agent` | renos/agent:latest | Device agent (privileged) |
| `nginx` | nginx:1.27-alpine | Reverse proxy and TLS termination |

### Docker Networks

| Network | Type | Purpose |
|---------|------|---------|
| `renos-backend` | bridge, internal | Service-to-service communication |
| `renos-frontend` | bridge | External-facing (nginx only) |

### Docker Volumes

| Volume | Mount Point | Purpose |
|--------|------------|---------|
| `pgdata` | /var/lib/postgresql/data | PostgreSQL data |
| `redis-data` | /data | Redis persistence |
| `nats-data` | /data | NATS JetStream data |
| `renos-data` | /var/lib/renos/data | Core application data |
| `renos-ca` | /var/lib/renos/ca | Certificate authority data |
| `renos-backups` | /backup | Backup storage |

### Resource Limits

Default resource limits per service (set in docker-compose.yml):

| Service | CPU Limit | Memory Limit | Memory Reservation |
|---------|-----------|-------------|-------------------|
| postgres | 1.0 | 512M | 256M |
| redis | 0.5 | 256M | 128M |
| nats | 0.5 | 256M | 128M |
| renos-core | 1.0 | 512M | 256M |
| renos-hub | 0.5 | 256M | 128M |
| renos-agent | 0.5 | 256M | 128M |
| nginx | 0.5 | 128M | 64M |

### Health Checks

Each service has a configured health check:

```bash
# Check overall system health
docker compose ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}"

# Manual health check for specific service
docker compose exec renos-core wget -q http://localhost:8443/api/v1/healthz -O /dev/null
docker compose exec postgres pg_isready -U renos
docker compose exec redis redis-cli ping
docker compose exec nats wget -q http://localhost:8222/healthz -O /dev/null
```

### Docker Compose Production Configuration

The Linux installer generates a production-grade `docker-compose.yml` at `/etc/renos/docker-compose.yml` with embedded defaults. The key differences from the development compose file are:

- All ports are moved to internal networks (except nginx)
- Resource limits are enforced
- Health checks are configured for all services
- Named volumes are used for data persistence
- The agent runs in privileged mode for device management

### Customizing the Docker Deployment

To customize resource allocations:

```bash
# Edit the docker-compose.yml deploy section
nano docker-compose.yml
# Modify resources under each service's deploy section

# Apply changes
docker compose up -d
```

To add environment variables:

```bash
# Option 1: Add to .env file
echo "MY_CUSTOM_VAR=value" >> .env

# Option 2: Edit docker-compose.yml directly
# Add to the environment section of the target service

# Restart the affected service
docker compose up -d <service-name>
```

### Stopping RenOS

```bash
# Graceful stop (preserves volumes)
docker compose down

# Full stop with volume removal (WARNING: destroys data)
docker compose down -v

# Stop specific service
docker compose stop renos-core
```

---

## Kubernetes Installation (Helm)

### Prerequisites

- Kubernetes cluster 1.24+
- Helm 3.8+
- kubectl configured with cluster access
- StorageClass supporting ReadWriteOnce PVCs
- Ingress controller (optional, for external access)
- cert-manager (optional, for automatic TLS)

### Quick Install

```bash
# Add the RenOS Helm repository
helm repo add renos https://helm.renos.ai
helm repo update

# Install RenOS
helm install renos renos/renos \
  --namespace renos --create-namespace \
  --set core.env.RENOS_ADMIN_EMAIL=admin@example.com \
  --set core.env.RENOS_ADMIN_PASSWORD=changeme123 \
  --set global.env.RENOS_EXTERNAL_URL=https://renos.example.com
```

### Detailed Installation Steps

**Step 1: Create namespace**

```bash
kubectl create namespace renos
```

**Step 2: Create secrets**

```bash
# Generate secrets
POSTGRES_PASSWORD=$(openssl rand -base64 24)
REDIS_PASSWORD=$(openssl rand -base64 24)
JWT_SECRET=$(openssl rand -hex 32)
ENCRYPTION_KEY=$(openssl rand -hex 32)

# Create Kubernetes secrets
kubectl create secret generic renos-secrets \
  --namespace renos \
  --from-literal=postgres-password=$POSTGRES_PASSWORD \
  --from-literal=redis-password=$REDIS_PASSWORD \
  --from-literal=jwt-secret=$JWT_SECRET \
  --from-literal=encryption-key=$ENCRYPTION_KEY
```

**Step 3: Create values file**

```yaml
# values-production.yaml
global:
  env:
    RENOS_EXTERNAL_URL: https://renos.example.com
    RENOS_ADMIN_EMAIL: admin@example.com
    RENOS_ADMIN_PASSWORD: changeme123
    RENOS_LOG_LEVEL: info

images:
  core:
    repository: renos/core
    tag: latest
  hub:
    repository: renos/hub
    tag: latest
  agent:
    repository: renos/agent
    tag: latest

core:
  replicaCount: 2
  resources:
    requests:
      cpu: 250m
      memory: 256Mi
    limits:
      cpu: 500m
      memory: 512Mi
  ingress:
    enabled: true
    className: nginx
    hosts:
      - host: renos.example.com
        paths:
          - path: /api
            pathType: Prefix
            service: core
            port: 8080

hub:
  replicaCount: 2
  ingress:
    enabled: true
    className: nginx
    hosts:
      - host: renos.example.com
        paths:
          - path: /
            pathType: Prefix
            service: hub
            port: 3000

postgres:
  storage: 20Gi
  auth:
    password: $POSTGRES_PASSWORD

redis:
  storage: 10Gi
  auth:
    password: $REDIS_PASSWORD

ingress:
  enabled: true
  tls:
    - hosts:
        - renos.example.com
      secretName: renos-tls

backup:
  enabled: true
  schedule: "0 2 * * *"
  storage:
    size: 20Gi

monitoring:
  serviceMonitor:
    enabled: true
```

**Step 4: Install with Helm**

```bash
helm install renos ./kubernetes \
  --namespace renos \
  --values values-production.yaml
```

**Step 5: Verify installation**

```bash
# Check pods
kubectl get pods -n renos

# Check services
kubectl get svc -n renos

# Check ingress
kubectl get ingress -n renos

# View logs
kubectl logs -n renos -l app.kubernetes.io/component=core
```

### Customizing Installation

See the [Kubernetes Deployment Guide](./kubernetes-deployment.md) for detailed configuration reference including all Helm values, storage classes, ingress controllers, TLS configuration, autoscaling, network policies, and Prometheus/Grafana integration.

---

## Air-Gapped Installation

For environments without internet access, see the [Air-Gapped Deployment Guide](./air-gapped-deployment.md) for detailed instructions on:

1. Downloading all artifacts on an internet-connected machine
2. Transferring via USB drive or private network
3. Loading Docker images into a private registry
4. Deploying Helm charts from local charts
5. Configuring offline package repositories

---

## Post-Installation Steps

### 1. Verify Health

```bash
# Check all services
docker compose ps

# API health check
curl https://your-server/api/v1/healthz

# Dashboard access
open https://your-server
```

### 2. Configure DNS

Create DNS A records pointing to your RenOS server:

```
renos.example.com  A  <server-ip>
*.renos.example.com  A  <server-ip>   (if using subdomain per device)
```

### 3. Set up Monitoring

```bash
# Enable Prometheus metrics (Docker)
# Edit docker-compose.yml to expose metrics ports
# Restart the stack

# For Kubernetes, enable the ServiceMonitor
helm upgrade renos ./kubernetes \
  --set monitoring.serviceMonitor.enabled=true
```

### 4. Configure Backups

```bash
# Automated backups are configured via cron
# Default: daily at 2 AM
# See backup/renos-backup.sh for details

# Manual backup
sudo /usr/local/bin/renos-backup.sh --full --remote
```

### 5. Set Up Log Rotation

The installer configures log rotation automatically at `/etc/logrotate.d/renos`. Default settings:

- Daily rotation
- 30-day retention
- 100 MB max file size
- Compression enabled

### 6. Configure TLS

If using self-signed certificates during installation, replace them for production:

```bash
# Replace with valid certificates
cp /path/to/fullchain.pem /etc/renos/tls/
cp /path/to/privkey.pem /etc/renos/tls/
chmod 600 /etc/renos/tls/privkey.pem
systemctl restart renos-nginx
```

### 7. Create Admin User

If not set during installation:

```bash
# Access the dashboard at https://your-server
# Navigate to Settings > Users to create additional admin users
```

### 8. Test Device Enrollment

```bash
# On a device to be managed:
curl -fsSL https://install.renos.ai/agent.sh | sudo bash -s -- \
  --server https://your-server \
  --token your-enrollment-token
```

---

## Troubleshooting Common Issues

### Installation Fails with "Unsupported Distribution"

The installer supports: Ubuntu, Debian, Fedora, RHEL, Rocky, AlmaLinux, and Arch.

**Solutions:**
- Check your distribution: `cat /etc/os-release`
- Use the Docker-based installation instead of native
- Manually install Docker and run the docker-compose.yml

### Docker Installation Fails

```bash
# Check Docker is running
systemctl status docker

# View Docker logs
journalctl -u docker -n 100

# Restart Docker
systemctl restart docker
```

### Port Already in Use

```bash
# Check what is using port 80 or 443
sudo lsof -i :80
sudo lsof -i :443

# Stop conflicting service or change RenOS ports in .env
```

### Database Connection Failed

```bash
# Check PostgreSQL is running
docker compose ps postgres

# View PostgreSQL logs
docker compose logs postgres

# Test connection manually
docker compose exec postgres pg_isready -U renos
```

### TLS Certificate Issues

```bash
# Check certificate expiry
openssl x509 -in /etc/renos/tls/fullchain.pem -text -noout | grep -A2 Validity

# Test TLS connection
openssl s_client -connect your-server:443 -servername your-server
```

### NATS Connection Issues

```bash
# Check NATS is healthy
docker compose exec nats wget -q http://localhost:8222/healthz -O /dev/null

# View NATS logs
docker compose logs nats

# Check NATS cluster status
docker compose exec nats nats server check connection
```

### Container Constantly Restarting

```bash
# Check container logs
docker compose logs <service-name>

# Check resource limits
docker stats <container-name>

# Increase resource limits in docker-compose.yml
```

### Kubernetes Pods Not Starting

```bash
# Describe the pod for detailed error information
kubectl describe pod -n renos <pod-name>

# Check events
kubectl get events -n renos --sort-by='.lastTimestamp'

# Check logs
kubectl logs -n renos <pod-name>

# Check PVC status
kubectl get pvc -n renos

# Check resource constraints
kubectl describe node
```

### Slow Performance

**Common causes:**
- Insufficient CPU/RAM allocation
- Slow disk I/O (use SSD)
- Network latency between services
- Large device count without proper scaling

**Solutions:**
- Increase resource limits
- Use SSD/NVMe storage
- Deploy with dedicated database host
- Scale horizontally in Kubernetes
- Enable Redis persistence tuning

### Firewall Blocking Connections

```bash
# Check firewall status
sudo ufw status  # Ubuntu/Debian
sudo firewall-cmd --list-all  # Fedora/RHEL

# Add rules if needed
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo firewall-cmd --permanent --add-port=443/tcp
sudo firewall-cmd --reload
```

---

## Getting Help

### Documentation

- **Installation Guide**: See this document
- **Upgrade Guide**: [Upgrade Guide](./upgrade-guide.md)
- **Recovery Guide**: [Recovery Guide](./recovery-guide.md)
- **Cluster Deployment**: [Cluster Deployment](./cluster-deployment.md)
- **Kubernetes Deployment**: [Kubernetes Deployment](./kubernetes-deployment.md)
- **Air-Gapped Deployment**: [Air-Gapped Deployment](./air-gapped-deployment.md)
- **Disaster Recovery**: [Disaster Recovery](./disaster-recovery.md)

### Support Channels

| Channel | Details |
|---------|---------|
| GitHub Issues | https://github.com/renos/renos/issues |
| Documentation | https://docs.renos.ai |
| Community Forum | https://community.renos.ai |
| Email Support | support@renos.ai |
| Slack | https://slack.renos.ai |

### Gathering Diagnostic Information

Before contacting support, gather the following information:

```bash
# System information
uname -a
cat /etc/os-release
docker version
docker compose version

# RenOS version
cat /etc/renos/renos.env | grep VERSION

# Service status
systemctl status renos-core renos-hub renos-agent renos-nginx

# Container status
docker compose -f /etc/renos/docker-compose.yml ps

# Logs (last 100 lines)
tail -100 /var/log/renos/install.log
docker compose -f /etc/renos/docker-compose.yml logs --tail=50

# Diagnostic bundle
renos-diag.sh  # If available, generates a diagnostics tarball
```

### Reporting Issues

When reporting issues on GitHub, please include:

1. RenOS version: `grep RENOS_INSTALL_VERSION /etc/renos/renos.env`
2. Installation method: native, Docker, Kubernetes
3. Operating system: `cat /etc/os-release`
4. Docker version: `docker version --format '{{.Server.Version}}'`
5. Kubernetes version (if applicable): `kubectl version --short`
6. Steps to reproduce
7. Expected behavior
8. Actual behavior
9. Relevant logs and error messages
10. Any configuration changes from defaults
