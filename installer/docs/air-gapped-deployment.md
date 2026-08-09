# RenOS Air-Gapped Deployment Guide

## Table of Contents

- [Requirements for Air-Gapped Deployment](#requirements-for-air-gapped-deployment)
- [Preparing the Air-Gapped Environment](#preparing-the-air-gapped-environment)
- [Downloading Artifacts](#downloading-artifacts)
- [Transfer Mechanisms](#transfer-mechanisms)
- [Loading Docker Images Offline](#loading-docker-images-offline)
- [Private Registry Setup](#private-registry-setup)
- [Helm Chart Deployment from Local](#helm-chart-deployment-from-local)
- [Offline Package Installation](#offline-package-installation)
- [No-Internet Validation](#no-internet-validation)
- [Updates in Air-Gapped Environment](#updates-in-air-gapped-environment)

---

## Requirements for Air-Gapped Deployment

An air-gapped (offline/disconnected) environment has no direct internet access. All software artifacts must be downloaded on an internet-connected machine and transferred via approved media.

### Network Architecture

```
                    INTERNET
                        │
                        ▼
            ┌───────────────────────┐
            │  Internet-Connected   │
            │  Workstation          │
            │  (Downloads Artifacts)│
            └───────────┬───────────┘
                        │
            ┌───────────┴───────────┐
            │    Transfer Media     │
            │   (USB Drive / DVD /  │
            │   Private Network)    │
            └───────────┬───────────┘
                        │
                        ▼
            ┌───────────────────────┐
            │  Air-Gapped Network   │
            │  (No Internet Access) │
            │                       │
            │  ┌─────────────────┐  │
            │  │  Private        │  │
            │  │  Container      │  │
            │  │  Registry       │  │
            │  └─────────────────┘  │
            │                       │
            │  ┌─────────────────┐  │
            │  │  RenOS Servers  │  │
            │  └─────────────────┘  │
            └───────────────────────┘
```

### Hardware Requirements

| Component | Internet Workstation | Air-Gapped Server |
|-----------|-------------------|-------------------|
| CPU | 4 cores | 4+ cores |
| RAM | 8 GB | 16 GB |
| Disk | 100 GB free | 100 GB+ SSD |
| Network | Internet access | Isolated network |
| Storage | USB 3.0 / NVMe | Internal storage |

### Software Requirements

| Software | Internet Machine | Air-Gapped Machine |
|----------|-----------------|-------------------|
| Docker | 24.x+ | 24.x+ |
| Docker Compose | v2.24+ | v2.24+ |
| Helm | 3.8+ | 3.8+ |
| kubectl | 1.24+ | 1.24+ |
| skopeo | 1.x | 1.x (optional) |
| rsync | 3.x | 3.x |
| age/gpg | latest | latest (for encrypted transfer) |

### Artifact Storage Estimate

| Artifact Type | Size | Quantity | Total |
|--------------|------|----------|-------|
| Docker images (all services) | ~500 MB each | 7 images | 3.5 GB |
| Helm chart | ~50 KB | 1 | 50 KB |
| Linux packages (per distro) | ~2 GB | 7 distros | 14 GB |
| macOS PKG | ~50 MB | 1 | 50 MB |
| Windows MSI | ~50 MB | 1 | 50 MB |
| Documentation | ~5 MB | 1 | 5 MB |
| **Total** | | | **~18 GB** |

---

## Preparing the Air-Gapped Environment

### Step 1: Set Up Private Network

Ensure all air-gapped nodes have:

- Static IP addresses on the isolated network
- DNS resolution for internal hostnames
- NTP synchronization (internal NTP server)
- Shared storage or network file system

```bash
# Example network configuration
cat > /etc/netplan/01-netcfg.yaml << 'EOF'
network:
  version: 2
  ethernets:
    eth0:
      addresses:
        - 10.0.0.10/24
      gateway4: 10.0.0.1
      nameservers:
        addresses:
          - 10.0.0.2
          - 10.0.0.3
EOF

netplan apply
```

### Step 2: Configure Internal DNS

```bash
# /etc/hosts entries for all air-gapped nodes
echo "10.0.0.10 renos-registry.local" >> /etc/hosts
echo "10.0.0.11 renos-node1.local" >> /etc/hosts
echo "10.0.0.12 renos-node2.local" >> /etc/hosts
echo "10.0.0.13 renos-node3.local" >> /etc/hosts
```

### Step 3: Install Prerequisites Offline

Ensure all prerequisites are installed on the air-gapped machines before attempting RenOS deployment:

```bash
# Verify prerequisites
docker --version
docker compose version
openssl version
curl --version
```

### Step 4: Create Artifact Staging Directory

```bash
# On the internet-connected machine (for downloads)
mkdir -p ~/renos-airgap/
cd ~/renos-airgap/
mkdir -p {images,charts,packages,scripts,backups}
```

### Step 5: Verify Checksums (On Air-Gapped Machine)

Always verify checksums of transferred artifacts:

```bash
# After transfer, verify integrity
sha256sum --check SHA256SUMS

# Expected output:
# renos-core.tar: OK
# renos-hub.tar: OK
# renos-helm-chart.tgz: OK
# ...
```

---

## Downloading Artifacts

### Downloading Docker Images

**Method 1: Using docker pull + save**

```bash
# On internet-connected machine
# Set environment variables
export RENOS_VERSION="1.0.0-rc1"

# Pull all images
docker pull renos/core:${RENOS_VERSION}
docker pull renos/hub:${RENOS_VERSION}
docker pull renos/agent:${RENOS_VERSION}
docker pull postgres:16-alpine
docker pull redis:7-alpine
docker pull nats:2-alpine
docker pull nginx:1.27-alpine

# Save images to tarballs
docker save renos/core:${RENOS_VERSION} -o images/renos-core.tar
docker save renos/hub:${RENOS_VERSION} -o images/renos-hub.tar
docker save renos/agent:${RENOS_VERSION} -o images/renos-agent.tar
docker save postgres:16-alpine -o images/postgres.tar
docker save redis:7-alpine -o images/redis.tar
docker save nats:2-alpine -o images/nats.tar
docker save nginx:1.27-alpine -o images/nginx.tar

# Compress the images
gzip images/*.tar
```

**Method 2: Using skopeo (more efficient)**

```bash
# Pull images directly to a directory
skopeo copy \
  docker://renos/core:${RENOS_VERSION} \
  dir:images/renos-core

skopeo copy \
  docker://renos/hub:${RENOS_VERSION} \
  dir:images/renos-hub

# ... repeat for all images

# Archive the directory
tar -czf images/all-images.tar.gz images/
```

**Method 3: Using a download script**

```bash
#!/bin/bash
# download-all-images.sh

set -euo pipefail

RENOS_VERSION="${1:-1.0.0-rc1}"
OUTPUT_DIR="${2:-./images}"

IMAGES=(
  "renos/core:${RENOS_VERSION}"
  "renos/hub:${RENOS_VERSION}"
  "renos/agent:${RENOS_VERSION}"
  "postgres:16-alpine"
  "redis:7-alpine"
  "nats:2-alpine"
  "nginx:1.27-alpine"
)

mkdir -p "${OUTPUT_DIR}"

for image in "${IMAGES[@]}"; do
  echo "Pulling ${image}..."
  docker pull "${image}"
  
  filename=$(echo "${image}" | tr '/:' '--')
  echo "Saving ${image} to ${OUTPUT_DIR}/${filename}.tar..."
  docker save "${image}" -o "${OUTPUT_DIR}/${filename}.tar"
  gzip "${OUTPUT_DIR}/${filename}.tar"
  
  echo "Done: ${image}"
done

# Generate checksums
cd "${OUTPUT_DIR}"
sha256sum *.tar.gz > SHA256SUMS
echo "All images downloaded. Checksums saved to SHA256SUMS."
```

```bash
chmod +x download-all-images.sh
./download-all-images.sh 1.0.0-rc1
```

### Downloading the Helm Chart

```bash
# Add the repository (needs internet)
helm repo add renos https://helm.renos.ai
helm repo update

# Download the chart
helm pull renos/renos \
  --version 0.1.0 \
  --destination charts/

# Alternative: download from GitHub releases
curl -fsSL https://github.com/renos/renos/releases/download/v1.0.0-rc1/renos-helm-chart-0.1.0.tgz \
  -o charts/renos-helm-chart.tgz
```

### Downloading Linux Packages

```bash
# For Debian/Ubuntu systems
mkdir -p packages/debian

# Download RenOS packages
apt-get download renos-core renos-hub renos-agent \
  --download-only -o Dir::Cache=./packages/debian

# Download dependencies
apt-get build-dep renos-core --download-only -o Dir::Cache=./packages/debian

# For RHEL/Fedora systems
mkdir -p packages/rhel
dnf download --resolve --destdir packages/rhel renos-core renos-hub renos-agent

# For Arch Linux
mkdir -p packages/arch
pacman -Sw --cachedir packages/arch renos-core renos-hub renos-agent
```

### Downloading Documentation and Scripts

```bash
# Clone the repository (for scripts and documentation)
git clone https://github.com/renos/renos.git
tar -czf scripts/renos-installer.tar.gz \
  renos/installer/linux/ \
  renos/installer/docker/ \
  renos/installer/backup/
```

### Complete Download Manifest

Create a manifest file to track all downloaded artifacts:

```bash
cat > airgap-manifest.json << 'EOF'
{
  "version": "1.0.0-rc1",
  "created": "2025-01-01T00:00:00Z",
  "artifacts": {
    "images": [
      "renos/core:1.0.0-rc1",
      "renos/hub:1.0.0-rc1",
      "renos/agent:1.0.0-rc1",
      "postgres:16-alpine",
      "redis:7-alpine",
      "nats:2-alpine",
      "nginx:1.27-alpine"
    ],
    "charts": [
      "renos-helm-chart-0.1.0.tgz"
    ],
    "packages": {
      "debian": ["renos-core.deb", "renos-hub.deb"],
      "rhel": ["renos-core.rpm", "renos-hub.rpm"],
      "arch": ["renos-core.pkg.tar.zst"]
    },
    "checksums": {}
  }
}
EOF
```

---

## Transfer Mechanisms

### USB Drive Transfer

```bash
# On internet-connected machine
# Prepare USB drive (at least 32 GB)
sudo mkfs.ext4 /dev/sdX1 -L RENOS_AIRGAP
sudo mount /dev/sdX1 /mnt/usb

# Copy all artifacts
cp -r ~/renos-airgap/* /mnt/usb/

# Generate checksums
cd /mnt/usb
find . -type f -exec sha256sum {} \; > SHA256SUMS

# Safely remove
sync
sudo umount /mnt/usb

# On air-gapped machine
sudo mount /dev/sdX1 /mnt/usb

# Verify checksums
cd /mnt/usb
sha256sum --check SHA256SUMS

# Copy to local storage
cp -r /mnt/usb/* ~/renos-airgap/
```

### Private Network Transfer (rsync)

```bash
# On internet-connected machine, set up temporary HTTP server
cd ~/renos-airgap
python3 -m http.server 8080

# On air-gapped machine, download via rsync
# (if SSH connectivity exists)
rsync -avzP user@internet-host:~/renos-airgap/ ~/renos-airgap/

# Or via HTTP
wget -r -np -nH --cut-dirs=1 \
  http://internet-host:8080/ \
  -P ~/renos-airgap/
```

### Optical Media (DVD)

```bash
# For smaller deployments (< 4.7 GB per DVD)
# Create ISO image
mkisofs -J -R -V "RENOS_AIRGAP" \
  -o renos-airgap.iso \
  ~/renos-airgap/

# Split for multiple DVDs (if > 4.7 GB)
split -b 4700000000 renos-airgap.iso renos-airgap.iso.part-

# Burn to DVD
growisofs -dvd-compat -Z /dev/sr0=renos-airgap.iso
```

### Encrypted Transfer

For sensitive environments, encrypt artifacts during transfer:

```bash
# Encrypt with age (recommended)
# On internet machine
tar -czf - ~/renos-airgap/ | age -r age1... > renos-airgap.tar.age

# On air-gapped machine
age --decrypt -i ~/.keys/age-key.txt \
  -o renos-airgap.tar \
  renos-airgap.tar.age
tar -xzf renos-airgap.tar

# Encrypt with GPG
gpg --encrypt --recipient admin@example.com \
  --output renos-airgap.tar.gpg \
  renos-airgap.tar
```

---

## Loading Docker Images Offline

### Loading from Tarballs

```bash
# On the air-gapped machine
cd ~/renos-airgap/images

# Load all images
for image in *.tar.gz; do
  echo "Loading ${image}..."
  gunzip -c "${image}" | docker load
done

# Verify images are loaded
docker images | grep -E "renos/|postgres|redis|nats|nginx"
```

### Using a Load Script

```bash
#!/bin/bash
# load-images.sh

set -euo pipefail

IMAGE_DIR="${1:-./images}"

if [ ! -d "${IMAGE_DIR}" ]; then
  echo "Error: Image directory '${IMAGE_DIR}' not found"
  exit 1
fi

echo "Loading images from ${IMAGE_DIR}..."

loaded=0
failed=0

for archive in "${IMAGE_DIR}"/*.tar.gz; do
  if [ ! -f "${archive}" ]; then
    echo "No tar.gz files found in ${IMAGE_DIR}"
    exit 1
  fi
  
  echo "Loading: $(basename "${archive}")..."
  if gunzip -c "${archive}" | docker load > /dev/null 2>&1; then
    echo "  ✓ Loaded: $(basename "${archive}")"
    ((loaded++))
  else
    # Try without decompression (in case file is already tar)
    if docker load -i "${archive%.gz}" > /dev/null 2>&1; then
      echo "  ✓ Loaded: $(basename "${archive}")"
      ((loaded++))
    else
      echo "  ✗ Failed: $(basename "${archive}")"
      ((failed++))
    fi
  fi
done

echo ""
echo "Summary: ${loaded} images loaded, ${failed} failed"

# Verify all required images
echo ""
echo "Verifying required images..."
REQUIRED_IMAGES=(
  "renos/core"
  "renos/hub"
  "renos/agent"
  "postgres"
  "redis"
  "nats"
  "nginx"
)

for image in "${REQUIRED_IMAGES[@]}"; do
  if docker images --format '{{.Repository}}' | grep -q "^${image}$"; then
    echo "  ✓ ${image}"
  else
    echo "  ✗ ${image} NOT FOUND"
    ((failed++))
  fi
done

exit ${failed}
```

```bash
chmod +x load-images.sh
./load-images.sh ~/renos-airgap/images/
```

### Verifying Loaded Images

```bash
# Detailed image listing
docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}"

# Expected output:
# REPOSITORY    TAG           SIZE
# renos/core    1.0.0-rc1     450MB
# renos/hub     1.0.0-rc1     350MB
# renos/agent   1.0.0-rc1     200MB
# postgres      16-alpine     200MB
# redis         7-alpine      50MB
# nats          2-alpine      30MB
# nginx         1.27-alpine   50MB
```

---

## Private Registry Setup

### Option 1: Docker Registry (Simple)

```bash
# Deploy a local Docker registry on a dedicated node
docker run -d \
  --name renos-registry \
  --restart always \
  -p 5000:5000 \
  -v /var/lib/registry:/var/lib/registry \
  registry:2

# Verify registry is running
curl http://localhost:5000/v2/
```

### Option 2: Harbor Registry (Production)

Harbor provides vulnerability scanning, replication, RBAC, and audit logging:

```bash
# Download Harbor offline installer
# (Requires internet to download, but installs offline)
# Download from: https://github.com/goharbor/harbor/releases

wget https://github.com/goharbor/harbor/releases/download/v2.10.0/harbor-offline-installer-v2.10.0.tgz

# Transfer to air-gapped machine
# Extract and install
tar -xzf harbor-offline-installer-v2.10.0.tgz
cd harbor

# Configure harbor.yml
cp harbor.yml.tmpl harbor.yml
# Edit: hostname, certificate paths, password

# Install
sudo ./install.sh --with-trivy

# Access: https://renos-registry.local
# Default: admin / Harbor12345
```

### Tagging and Pushing Images to Private Registry

```bash
# Set the private registry URL
PRIVATE_REGISTRY="renos-registry.local:5000"

# Tag and push all images
docker tag renos/core:1.0.0-rc1 ${PRIVATE_REGISTRY}/renos/core:1.0.0-rc1
docker push ${PRIVATE_REGISTRY}/renos/core:1.0.0-rc1

docker tag renos/hub:1.0.0-rc1 ${PRIVATE_REGISTRY}/renos/hub:1.0.0-rc1
docker push ${PRIVATE_REGISTRY}/renos/hub:1.0.0-rc1

docker tag renos/agent:1.0.0-rc1 ${PRIVATE_REGISTRY}/renos/agent:1.0.0-rc1
docker push ${PRIVATE_REGISTRY}/renos/agent:1.0.0-rc1

docker tag postgres:16-alpine ${PRIVATE_REGISTRY}/postgres:16-alpine
docker push ${PRIVATE_REGISTRY}/postgres:16-alpine

docker tag redis:7-alpine ${PRIVATE_REGISTRY}/redis:7-alpine
docker push ${PRIVATE_REGISTRY}/redis:7-alpine

docker tag nats:2-alpine ${PRIVATE_REGISTRY}/nats:2-alpine
docker push ${PRIVATE_REGISTRY}/nats:2-alpine

docker tag nginx:1.27-alpine ${PRIVATE_REGISTRY}/nginx:1.27-alpine
docker push ${PRIVATE_REGISTRY}/nginx:1.27-alpine
```

### Push Script

```bash
#!/bin/bash
# push-to-private-registry.sh

set -euo pipefail

REGISTRY="${1:-renos-registry.local:5000}"

IMAGES=(
  "renos/core:1.0.0-rc1"
  "renos/hub:1.0.0-rc1"
  "renos/agent:1.0.0-rc1"
  "postgres:16-alpine"
  "redis:7-alpine"
  "nats:2-alpine"
  "nginx:1.27-alpine"
)

for image in "${IMAGES[@]}"; do
  echo "Pushing ${image} to ${REGISTRY}..."
  docker tag "${image}" "${REGISTRY}/${image}"
  docker push "${REGISTRY}/${image}"
  echo "  ✓ ${image} pushed"
done

echo ""
echo "All images pushed to ${REGISTRY}"
```

### Configuring RenOS to Use Private Registry

```bash
# For Docker Compose deployments
# Edit docker-compose.yml to reference private registry
sed -i "s|image: renos/|image: ${PRIVATE_REGISTRY}/renos/|g" docker-compose.yml
sed -i "s|image: postgres:|image: ${PRIVATE_REGISTRY}/postgres:|g" docker-compose.yml
# ... repeat for all images

# For Kubernetes/Helm deployments
# Set global.imageRegistry in values.yaml
global:
  imageRegistry: "renos-registry.local:5000"
```

---

## Helm Chart Deployment from Local

### Deploying from Downloaded Chart

```bash
# On air-gapped machine
cd ~/renos-airgap/charts

# Install from local .tgz
helm install renos renos-helm-chart-0.1.0.tgz \
  --namespace renos --create-namespace \
  --values ../values-airgap.yaml
```

### Deploying from Chart Directory

```bash
# Alternatively, extract the chart
tar -xzf renos-helm-chart-0.1.0.tgz

# Inspect the chart directory
ls -la renos/

# Install from directory
helm install renos ./renos \
  --namespace renos --create-namespace \
  --values ../values-airgap.yaml
```

### Air-Gapped Helm Values

```yaml
# values-airgap.yaml
global:
  imageRegistry: "renos-registry.local:5000"
  imagePullPolicy: IfNotPresent
  env:
    RENOS_EXTERNAL_URL: https://renos.local
    RENOS_ADMIN_EMAIL: admin@renos.local
    RENOS_ADMIN_PASSWORD: SuperS3cret!

images:
  core:
    repository: renos-registry.local:5000/renos/core
    tag: "1.0.0-rc1"
  hub:
    repository: renos-registry.local:5000/renos/hub
    tag: "1.0.0-rc1"
  agent:
    repository: renos-registry.local:5000/renos/agent
    tag: "1.0.0-rc1"
  postgres:
    repository: renos-registry.local:5000/postgres
    tag: 16-alpine
  redis:
    repository: renos-registry.local:5000/redis
    tag: 7-alpine
  nats:
    repository: renos-registry.local:5000/nats
    tag: 2-alpine

# Disable external connectivity features
core:
  env:
    RENOS_TELEMETRY_ENABLED: "false"
    RENOS_UPDATE_CHECK_ENABLED: "false"
    RENOS_EXTERNAL_INTEGRATIONS_ENABLED: "false"

hub:
  env:
    NEXT_PUBLIC_DISABLE_EXTERNAL_LINKS: "true"
```

---

## Offline Package Installation

### Debian/Ubuntu

```bash
# On the air-gapped machine
cd ~/renos-airgap/packages/debian

# Install all .deb packages
sudo dpkg -i *.deb

# If dependency issues, force install from local packages
sudo apt-get install -f -y --no-download

# Verify installation
dpkg -l | grep renos
```

### Creating a Local APT Repository

For larger deployments, create a local APT repository:

```bash
# On the air-gapped machine
apt-get install dpkg-dev

cd ~/renos-airgap/packages/debian

# Create repository metadata
dpkg-scanpackages . /dev/null | gzip -9c > Packages.gz

# Add to sources.list
echo "deb [trusted=yes] file:///home/renos/packages/debian ./" \
  >> /etc/apt/sources.list

# Update and install
apt-get update
apt-get install renos-core renos-hub renos-agent
```

### RHEL/Fedora/CentOS

```bash
# On the air-gapped machine
cd ~/renos-airgap/packages/rhel

# Install all .rpm packages
sudo dnf install -y *.rpm

# Or create a local repository
createrepo .

# Add repository
cat > /etc/yum.repos.d/renos-local.repo << 'EOF'
[renos-local]
name=RenOS Local Repository
baseurl=file:///home/renos/packages/rhel
enabled=1
gpgcheck=0
EOF

dnf install renos-core renos-hub renos-agent
```

### Arch Linux

```bash
# On the air-gapped machine
cd ~/renos-airgap/packages/arch

# Install packages
sudo pacman -U *.pkg.tar.zst
```

---

## No-Internet Validation

### Connectivity Test

```bash
#!/bin/bash
# validate-airgap.sh

set -euo pipefail

echo "=== RenOS Air-Gapped Validation ==="
echo ""

errors=0

echo "--- Network Validation ---"
# Test that we cannot reach external services
if curl -s --connect-timeout 5 https://google.com > /dev/null 2>&1; then
  echo "  ✗ WARNING: Internet access detected!"
  ((errors++))
else
  echo "  ✓ No internet access (expected for air-gapped)"
fi

# Test that we can reach local services
for host in renos-registry.local renos-node1.local; do
  if ping -c 1 -W 2 "${host}" > /dev/null 2>&1; then
    echo "  ✓ Can reach: ${host}"
  else
    echo "  ✗ Cannot reach: ${host}"
    ((errors++))
  fi
done

echo ""
echo "--- Docker Images ---"
REQUIRED_IMAGES=(
  "renos/core"
  "renos/hub"
  "renos/agent"
  "postgres"
  "redis"
  "nats"
  "nginx"
)
for image in "${REQUIRED_IMAGES[@]}"; do
  if docker images --format '{{.Repository}}' | grep -q "^${image}$"; then
    echo "  ✓ ${image}"
  else
    echo "  ✗ ${image} not found"
    ((errors++))
  fi
done

echo ""
echo "--- Private Registry ---"
if curl -s http://renos-registry.local:5000/v2/ > /dev/null 2>&1; then
  echo "  ✓ Private registry accessible"
else
  echo "  ✗ Private registry NOT accessible"
  ((errors++))
fi

echo ""
echo "--- Helm Chart ---"
if ls ~/renos-airgap/charts/renos*.tgz > /dev/null 2>&1; then
  echo "  ✓ Helm chart present"
else
  echo "  ✗ Helm chart not found"
  ((errors++))
fi

echo ""
echo "--- DNS Resolution ---"
for host in renos-registry.local renos-node1.local; do
  if host "${host}" > /dev/null 2>&1 || grep -q "${host}" /etc/hosts; then
    echo "  ✓ DNS: ${host}"
  else
    echo "  ✗ DNS: ${host} not resolvable"
    ((errors++))
  fi
done

echo ""
if [ ${errors} -eq 0 ]; then
  echo "✓ All validation checks passed!"
else
  echo "✗ ${errors} issue(s) found"
fi

exit ${errors}
```

```bash
chmod +x validate-airgap.sh
./validate-airgap.sh
```

### Deployment Dry Run

```bash
# Test that Docker Compose can resolve all images locally
cd ~/renos-airgap
docker compose -f docker/docker-compose.yml config

# Test Helm chart with dry run
helm install renos ./renos-helm-chart-0.1.0.tgz \
  --namespace renos \
  --values values-airgap.yaml \
  --dry-run --debug
```

---

## Updates in Air-Gapped Environment

### Update Process Overview

Updating an air-gapped deployment is similar to initial setup but requires only the changed artifacts:

1. Download new versions on internet-connected machine
2. Transfer differential artifacts (only changed files)
3. Load new Docker images
4. Push to private registry (retagging)
5. Apply configuration changes
6. Update Helm chart or Docker Compose
7. Verify deployment

### Downloading Updates

```bash
#!/bin/bash
# download-update.sh

set -euo pipefail

OLD_VERSION="${1}"
NEW_VERSION="${2}"

echo "Downloading update artifacts for ${OLD_VERSION} → ${NEW_VERSION}"

# Create update directory
UPDATE_DIR="update-${NEW_VERSION}"
mkdir -p "${UPDATE_DIR}"

# Pull new images
docker pull "renos/core:${NEW_VERSION}"
docker pull "renos/hub:${NEW_VERSION}"
docker pull "renos/agent:${NEW_VERSION}"

# Save to tarballs
docker save "renos/core:${NEW_VERSION}" | gzip > "${UPDATE_DIR}/renos-core-${NEW_VERSION}.tar.gz"
docker save "renos/hub:${NEW_VERSION}" | gzip > "${UPDATE_DIR}/renos-hub-${NEW_VERSION}.tar.gz"
docker save "renos/agent:${NEW_VERSION}" | gzip > "${UPDATE_DIR}/renos-agent-${NEW_VERSION}.tar.gz"

# Download updated Helm chart
helm repo update renos
helm pull renos/renos --destination "${UPDATE_DIR}"

# Download updated packages (if applicable)
# ...

# Generate checksums
cd "${UPDATE_DIR}"
sha256sum * > SHA256SUMS

echo "Update artifacts saved to ${UPDATE_DIR}"
```

### Applying Updates

```bash
#!/bin/bash
# apply-update.sh

set -euo pipefail

UPDATE_DIR="${1}"

if [ ! -d "${UPDATE_DIR}" ]; then
  echo "Error: Update directory '${UPDATE_DIR}' not found"
  exit 1
fi

echo "Applying update from ${UPDATE_DIR}"

# Verify checksums
cd "${UPDATE_DIR}"
sha256sum --check SHA256SUMS

# Load new Docker images
for archive in *.tar.gz; do
  echo "Loading ${archive}..."
  gunzip -c "${archive}" | docker load
done

# Tag and push to private registry
PRIVATE_REGISTRY="renos-registry.local:5000"
docker tag "renos/core:${NEW_VERSION}" "${PRIVATE_REGISTRY}/renos/core:${NEW_VERSION}"
docker push "${PRIVATE_REGISTRY}/renos/core:${NEW_VERSION}"
docker tag "renos/hub:${NEW_VERSION}" "${PRIVATE_REGISTRY}/renos/hub:${NEW_VERSION}"
docker push "${PRIVATE_REGISTRY}/renos/hub:${NEW_VERSION}"
docker tag "renos/agent:${NEW_VERSION}" "${PRIVATE_REGISTRY}/renos/agent:${NEW_VERSION}"
docker push "${PRIVATE_REGISTRY}/renos/agent:${NEW_VERSION}"

# Update Helm chart
helm upgrade renos "./${UPDATE_DIR}/renos-helm-chart-*.tgz" \
  --namespace renos \
  --values values-airgap.yaml \
  --set images.core.tag="${NEW_VERSION}" \
  --set images.hub.tag="${NEW_VERSION}" \
  --set images.agent.tag="${NEW_VERSION}"

echo "Update applied successfully"
```

### Rollback in Air-Gapped Environment

```bash
# Keep previous images available for rollback
# Previous versions remain in private registry

# Rollback to previous version
helm rollback renos -n renos

# Or upgrade to previous version
helm upgrade renos renos/renos -n renos \
  --values values-airgap.yaml \
  --set images.core.tag="${OLD_VERSION}" \
  --set images.hub.tag="${OLD_VERSION}" \
  --set images.agent.tag="${OLD_VERSION}"
```

### Update Frequency Recommendations

| Environment | Update Type | Frequency | Notes |
|-------------|-------------|-----------|-------|
| Air-gapped (critical) | Security patches | As needed | Priority 1 |
| Air-gapped (production) | Minor/patch | Quarterly | Test in staging first |
| Air-gapped (production) | Major | Annually | Requires migration planning |
| Air-gapped (development) | Any | As needed | Can afford more downtime |

### Maintaining an Update Registry

For air-gapped environments with multiple clusters, maintain a central update registry:

```
/airgap-updates/
├── 1.0.0/
│   ├── images/
│   ├── charts/
│   ├── packages/
│   └── SHA256SUMS
├── 1.1.0/
│   ├── images/
│   ├── charts/
│   ├── packages/
│   └── SHA256SUMS
├── 2.0.0/
│   ├── images/
│   ├── charts/
│   ├── packages/
│   └── SHA256SUMS
└── current -> 2.0.0/
```
