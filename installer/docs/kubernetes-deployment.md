# RenOS Kubernetes Deployment Guide

## Table of Contents

- [Prerequisites](#prerequisites)
- [Installing the Helm Chart](#installing-the-helm-chart)
- [Configuration Reference](#configuration-reference)
- [Storage Classes and PVCs](#storage-classes-and-pvcs)
- [Ingress Controllers](#ingress-controllers)
- [TLS Configuration](#tls-configuration)
- [Resource Sizing Guide](#resource-sizing-guide)
- [Autoscaling Configuration](#autoscaling-configuration)
- [Network Policies](#network-policies)
- [Prometheus/Grafana Integration](#prometheusgrafana-integration)
- [Upgrading via Helm](#upgrading-via-helm)
- [Production Hardening](#production-hardening)

---

## Prerequisites

### Kubernetes Cluster Requirements

| Resource | Minimum | Recommended |
|----------|---------|-------------|
| Kubernetes version | 1.24+ | 1.28+ |
| Nodes | 3 | 5 (spread across AZs) |
| CPU per node | 4 cores | 8 cores |
| RAM per node | 8 GB | 16 GB |
| Disk per node | 50 GB SSD | 100 GB NVMe |
| etcd storage | 2 GB | 8 GB |

### Required Kubernetes Features

- **StorageClass** supporting `ReadWriteOnce` PVCs
- **Ingress controller** (nginx, traefik, or haproxy)
- **DNS provider** for ingress hostnames
- **Metrics Server** (for HPA)
- **Prometheus Operator** (optional, for monitoring)

### Required CLI Tools

```bash
# Verify prerequisites
kubectl version --short
helm version --short
kubectl cluster-info

# Expected output:
# Client Version: v1.28.0
# Server Version: v1.28.0
# Helm Version: v3.14.0
```

### Helm Repository Setup

```bash
# Add the RenOS Helm repository
helm repo add renos https://helm.renos.ai
helm repo update

# Verify the repository
helm search repo renos/renos

# Expected output:
# NAME            CHART VERSION   APP VERSION     DESCRIPTION
# renos/renos     0.1.0           1.0.0-rc1       RenOS - Device Management Platform
```

### Namespace Setup

```bash
# Create the renos namespace
kubectl create namespace renos

# Label the namespace for network policies
kubectl label namespace renos pod-security.kubernetes.io/enforce=restricted
```

---

## Installing the Helm Chart

### Quick Install

```bash
helm install renos renos/renos \
  --namespace renos --create-namespace \
  --set core.env.RENOS_ADMIN_EMAIL=admin@example.com \
  --set core.env.RENOS_ADMIN_PASSWORD=changeme123 \
  --set global.env.RENOS_EXTERNAL_URL=https://renos.example.com
```

### Customized Install

**Step 1: Create a values file**

```yaml
# values-production.yaml
global:
  imageRegistry: ""
  imagePullSecrets: []
  imagePullPolicy: IfNotPresent
  env:
    RENOS_EXTERNAL_URL: https://renos.example.com
    RENOS_LOG_FORMAT: json
    RENOS_LOG_LEVEL: info

images:
  core:
    repository: renos/core
    tag: "1.0.0-rc1"
  hub:
    repository: renos/hub
    tag: "1.0.0-rc1"
  agent:
    repository: renos/agent
    tag: "1.0.0-rc1"
  postgres:
    repository: postgres
    tag: 16-alpine
  redis:
    repository: redis
    tag: 7-alpine
  nats:
    repository: nats
    tag: 2-alpine

core:
  replicaCount: 2
  logLevel: info
  resources:
    requests:
      cpu: 250m
      memory: 256Mi
    limits:
      cpu: 500m
      memory: 512Mi
  persistence:
    enabled: true
    size: 10Gi
    storageClass: "standard"
    accessMode: ReadWriteOnce
  service:
    type: ClusterIP
    port: 8080
    metricsPort: 9090

hub:
  replicaCount: 2
  logLevel: info
  resources:
    requests:
      cpu: 250m
      memory: 256Mi
    limits:
      cpu: 500m
      memory: 512Mi
  service:
    type: ClusterIP
    port: 3000
    metricsPort: 9091

agent:
  logLevel: info
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 250m
      memory: 256Mi

postgres:
  enabled: true
  storage: 20Gi
  storageClass: "standard"
  auth:
    database: renos
    username: renos
    password: ""  # Auto-generated if empty
  resources:
    requests:
      cpu: 250m
      memory: 512Mi
    limits:
      cpu: 1
      memory: 1Gi

redis:
  enabled: true
  storage: 5Gi
  storageClass: "standard"
  auth:
    enabled: true
    password: ""  # Auto-generated if empty
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 250m
      memory: 256Mi

nats:
  enabled: true
  cluster:
    enabled: true
    replicas: 3
  resources:
    requests:
      cpu: 250m
      memory: 256Mi
    limits:
      cpu: 500m
      memory: 512Mi
  persistence:
    enabled: true
    size: 10Gi
    storageClass: "standard"

ingress:
  enabled: true
  className: "nginx"
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
  hosts:
    - host: renos.example.com
      paths:
        - path: /api
          pathType: Prefix
          service: core
          port: 8080
        - path: /
          pathType: Prefix
          service: hub
          port: 3000
  tls:
    - hosts:
        - renos.example.com
      secretName: renos-tls

backup:
  enabled: true
  schedule: "0 2 * * *"
  storage:
    size: 20Gi
    storageClass: "standard"
  retention:
    days: 30
    copies: 14

monitoring:
  serviceMonitor:
    enabled: true
    interval: 30s
    scrapeTimeout: 10s
  prometheusRules:
    enabled: true

autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 80
  targetMemoryUtilizationPercentage: 80

networkPolicy:
  enabled: true

secrets:
  existingSecret: ""
  autoGenerate: true
  jwtSecret: ""
  encryptionKey: ""
```

**Step 2: Install with custom values**

```bash
helm install renos renos/renos \
  --namespace renos \
  --values values-production.yaml \
  --wait \
  --timeout 15m
```

**Step 3: Verify the installation**

```bash
# Check all pods are running
kubectl get pods -n renos -w

# Expected output:
# NAME                                READY   STATUS    RESTARTS   AGE
# renos-agent-xxxxx                   1/1     Running   0          5m
# renos-core-xxxxx                    1/1     Running   0          5m
# renos-hub-xxxxx                     1/1     Running   0          5m
# renos-nats-0                        1/1     Running   0          5m
# renos-nats-1                        1/1     Running   0          5m
# renos-nats-2                        1/1     Running   0          5m
# renos-postgres-0                    1/1     Running   0          5m
# renos-redis-0                       1/1     Running   0          5m

# Check services
kubectl get svc -n renos

# Check ingress
kubectl get ingress -n renos

# Check secrets were created
kubectl get secrets -n renos
```

---

## Configuration Reference

### Global Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `global.imageRegistry` | string | `""` | Global image registry override |
| `global.imagePullSecrets` | list | `[]` | Global image pull secrets |
| `global.imagePullPolicy` | string | `IfNotPresent` | Image pull policy |
| `global.env` | object | `{}` | Global environment variables for all RenOS components |

### Image Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `images.core.repository` | string | `renos/core` | Core image repository |
| `images.core.tag` | string | `""` | Core image tag (defaults to Chart appVersion) |
| `images.hub.repository` | string | `renos/hub` | Hub image repository |
| `images.hub.tag` | string | `""` | Hub image tag |
| `images.agent.repository` | string | `renos/agent` | Agent image repository |
| `images.agent.tag` | string | `""` | Agent image tag |
| `images.postgres.repository` | string | `postgres` | PostgreSQL image |
| `images.postgres.tag` | string | `16-alpine` | PostgreSQL image tag |
| `images.redis.repository` | string | `redis` | Redis image |
| `images.redis.tag` | string | `7-alpine` | Redis image tag |
| `images.nats.repository` | string | `nats` | NATS image |
| `images.nats.tag` | string | `2-alpine` | NATS image tag |

### Core Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `core.replicaCount` | int | `2` | Number of core replicas |
| `core.logLevel` | string | `info` | Log level (debug, info, warn, error) |
| `core.resources.requests.cpu` | string | `250m` | CPU request per core pod |
| `core.resources.requests.memory` | string | `256Mi` | Memory request per core pod |
| `core.resources.limits.cpu` | string | `500m` | CPU limit per core pod |
| `core.resources.limits.memory` | string | `512Mi` | Memory limit per core pod |
| `core.persistence.enabled` | bool | `true` | Enable persistent storage |
| `core.persistence.size` | string | `10Gi` | Storage size for core data |
| `core.persistence.storageClass` | string | `""` | Storage class for core |
| `core.persistence.accessMode` | string | `ReadWriteOnce` | PVC access mode |
| `core.service.type` | string | `ClusterIP` | Service type |
| `core.service.port` | int | `8080` | Core API port |
| `core.service.metricsPort` | int | `9090` | Metrics port |
| `core.ingress.enabled` | bool | `false` | Enable ingress for core |
| `core.ingress.path` | string | `/api(/\|$)(.*)` | Ingress path |
| `core.ingress.pathType` | string | `ImplementationSpecific` | Ingress path type |

### Hub Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `hub.replicaCount` | int | `2` | Number of hub replicas |
| `hub.logLevel` | string | `info` | Log level |
| `hub.resources.requests.cpu` | string | `250m` | CPU request per hub pod |
| `hub.resources.requests.memory` | string | `256Mi` | Memory request per hub pod |
| `hub.resources.limits.cpu` | string | `500m` | CPU limit per hub pod |
| `hub.resources.limits.memory` | string | `512Mi` | Memory limit per hub pod |
| `hub.service.type` | string | `ClusterIP` | Service type |
| `hub.service.port` | int | `3000` | Hub web UI port |
| `hub.service.metricsPort` | int | `9091` | Metrics port |
| `hub.ingress.enabled` | bool | `false` | Enable ingress for hub |
| `hub.ingress.path` | string | `/(.*)` | Ingress path |
| `hub.ingress.pathType` | string | `ImplementationSpecific` | Ingress path type |

### Agent Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `agent.logLevel` | string | `info` | Log level |
| `agent.resources.requests.cpu` | string | `100m` | CPU request per agent pod |
| `agent.resources.requests.memory` | string | `128Mi` | Memory request per agent pod |
| `agent.resources.limits.cpu` | string | `250m` | CPU limit per agent pod |
| `agent.resources.limits.memory` | string | `256Mi` | Memory limit per agent pod |
| `agent.tolerations` | list | `[{Exists, NoSchedule}, {Exists, NoExecute}]` | Pod tolerations for scheduling on all nodes |
| `agent.nodeSelector` | object | `{}` | Node selector for agent pods |
| `agent.dockerSocketPath` | string | `/var/run/docker.sock` | Path to Docker socket |

### PostgreSQL Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `postgres.enabled` | bool | `true` | Deploy PostgreSQL |
| `postgres.storage` | string | `10Gi` | PostgreSQL storage size |
| `postgres.storageClass` | string | `""` | Storage class for PostgreSQL |
| `postgres.accessMode` | string | `ReadWriteOnce` | PVC access mode |
| `postgres.resources.requests.cpu` | string | `250m` | CPU request |
| `postgres.resources.requests.memory` | string | `512Mi` | Memory request |
| `postgres.resources.limits.cpu` | string | `1` | CPU limit |
| `postgres.resources.limits.memory` | string | `1Gi` | Memory limit |
| `postgres.auth.database` | string | `renos` | Database name |
| `postgres.auth.username` | string | `renos` | Database user |
| `postgres.auth.password` | string | `""` | Database password (auto-generated if empty) |
| `postgres.auth.existingSecret` | string | `""` | Existing secret for password |
| `postgres.service.port` | int | `5432` | PostgreSQL service port |

### Redis Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `redis.enabled` | bool | `true` | Deploy Redis |
| `redis.storage` | string | `5Gi` | Redis storage size |
| `redis.storageClass` | string | `""` | Storage class |
| `redis.accessMode` | string | `ReadWriteOnce` | PVC access mode |
| `redis.resources.requests.cpu` | string | `100m` | CPU request |
| `redis.resources.requests.memory` | string | `128Mi` | Memory request |
| `redis.resources.limits.cpu` | string | `250m` | CPU limit |
| `redis.resources.limits.memory` | string | `256Mi` | Memory limit |
| `redis.auth.enabled` | bool | `true` | Enable Redis authentication |
| `redis.auth.password` | string | `""` | Redis password (auto-generated if empty) |
| `redis.auth.existingSecret` | string | `""` | Existing secret for password |
| `redis.service.port` | int | `6379` | Redis service port |

### NATS Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `nats.enabled` | bool | `true` | Deploy NATS |
| `nats.cluster.enabled` | bool | `true` | Enable NATS clustering |
| `nats.cluster.replicas` | int | `3` | Number of NATS cluster replicas |
| `nats.resources.requests.cpu` | string | `250m` | CPU request |
| `nats.resources.requests.memory` | string | `256Mi` | Memory request |
| `nats.resources.limits.cpu` | string | `500m` | CPU limit |
| `nats.resources.limits.memory` | string | `512Mi` | Memory limit |
| `nats.persistence.enabled` | bool | `true` | Enable persistent storage for JetStream |
| `nats.persistence.size` | string | `10Gi` | Storage size |
| `nats.service.clientPort` | int | `4222` | NATS client port |
| `nats.service.clusterPort` | int | `6222` | NATS cluster port |
| `nats.service.monitorPort` | int | `8222` | NATS monitoring port |

### Ingress Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `ingress.enabled` | bool | `false` | Enable ingress |
| `ingress.className` | string | `""` | Ingress class name |
| `ingress.annotations` | object | `{}` | Ingress annotations |
| `ingress.hosts` | list | `[]` | Ingress host rules |
| `ingress.tls` | list | `[]` | TLS configuration |

### Backup Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `backup.enabled` | bool | `false` | Enable automated backups |
| `backup.schedule` | string | `"0 2 * * *"` | Cron schedule for backups |
| `backup.storage.size` | string | `20Gi` | Backup PVC size |
| `backup.storage.storageClass` | string | `""` | Backup storage class |
| `backup.retention.days` | int | `30` | Retain backups for N days |
| `backup.retention.copies` | int | `14` | Retain at most N copies |
| `backup.resources` | object | `{}` | Backup job resources |

### Monitoring Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `monitoring.serviceMonitor.enabled` | bool | `false` | Enable Prometheus ServiceMonitor |
| `monitoring.serviceMonitor.interval` | string | `30s` | Scrape interval |
| `monitoring.serviceMonitor.scrapeTimeout` | string | `10s` | Scrape timeout |
| `monitoring.serviceMonitor.labels` | object | `{}` | Additional labels for ServiceMonitor |
| `monitoring.prometheusRules.enabled` | bool | `false` | Enable Prometheus alerting rules |
| `monitoring.prometheusRules.labels` | object | `{}` | Additional labels for PrometheusRule |

### Autoscaling Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `autoscaling.enabled` | bool | `false` | Enable HPA |
| `autoscaling.minReplicas` | int | `2` | Minimum replicas |
| `autoscaling.maxReplicas` | int | `10` | Maximum replicas |
| `autoscaling.targetCPUUtilizationPercentage` | int | `80` | Target CPU utilization |
| `autoscaling.targetMemoryUtilizationPercentage` | int | `80` | Target memory utilization |

### Secrets Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `secrets.existingSecret` | string | `""` | Existing Kubernetes secret name |
| `secrets.autoGenerate` | bool | `true` | Auto-generate secrets if not provided |
| `secrets.jwtSecret` | string | `""` | JWT signing secret (256-bit hex) |
| `secrets.encryptionKey` | string | `""` | Encryption key (256-bit hex) |

### Network Policy Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `networkPolicy.enabled` | bool | `false` | Enable network policies |

---

## Storage Classes and PVCs

### Recommended Storage Classes

| Storage Type | Provisioner | Volume Binding Mode | Use Case |
|-------------|-------------|-------------------|----------|
| **SSD (fast)** | ebs.csi.aws.com / pd.csi.storage.gke.io / disk.csi.azure.com | WaitForFirstConsumer | PostgreSQL, NATS JetStream |
| **Standard** | kubernetes.io/aws-ebs / kubernetes.io/gce-pd | Immediate | Redis, Backups |
| **Network** (NAS) | nfs.csi.k8s.com | Immediate | File storage, shared access |

### Creating StorageClass

```yaml
# storage-class.yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: renos-fast
provisioner: ebs.csi.aws.com
parameters:
  type: gp3
  iopsPerGB: "10"
  encrypted: "true"
volumeBindingMode: WaitForFirstConsumer
allowedTopologies:
  - matchLabelExpressions:
      - key: topology.kubernetes.io/zone
        values:
          - us-east-1a
          - us-east-1b
          - us-east-1c
```

### PersistentVolumeClaims

The Helm chart creates PVCs for:

| Component | PVC Name | Size | Access Mode | Storage Class |
|-----------|----------|------|-------------|---------------|
| PostgreSQL | `{release}-postgres-0` | 10-50 Gi | RWO | configurable |
| Redis | `{release}-redis-0` | 5-10 Gi | RWO | configurable |
| NATS | `{release}-nats-0` | 10-20 Gi | RWO | configurable |
| Core | `{release}-core-0` | 10 Gi | RWO | configurable |
| Backups | `{release}-backup` | 20 Gi | RWO | configurable |

### Custom PVC Configuration

```yaml
# Override storage per component
postgres:
  storage: 50Gi
  storageClass: "renos-fast"
  persistence:
    size: 50Gi
    storageClass: "renos-fast"

redis:
  storage: 10Gi
  storageClass: "renos-fast"

nats:
  persistence:
    size: 20Gi
    storageClass: "renos-fast"
```

### PVC Expansion

```bash
# For supported StorageClasses, expand PVCs
kubectl patch pvc -n renos data-renos-postgres-0 \
  --patch '{"spec":{"resources":{"requests":{"storage":"100Gi"}}}}'
```

---

## Ingress Controllers

### Nginx Ingress Controller

**Installation:**

```bash
# Using Helm
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm install ingress-nginx ingress-nginx/ingress-nginx \
  --namespace ingress-nginx --create-namespace
```

**RenOS Ingress Configuration:**

```yaml
ingress:
  enabled: true
  className: "nginx"
  annotations:
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/use-regex: "true"
    nginx.ingress.kubernetes.io/proxy-body-size: "64m"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
    nginx.ingress.kubernetes.io/configuration-snippet: |
      more_set_headers "X-Frame-Options: SAMEORIGIN";
      more_set_headers "X-Content-Type-Options: nosniff";
```

### Traefik Ingress Controller

**Installation:**

```bash
helm repo add traefik https://traefik.github.io/charts
helm install traefik traefik/traefik \
  --namespace traefik --create-namespace
```

**RenOS IngressRoute:**

```yaml
ingress:
  enabled: true
  className: "traefik"
  annotations:
    traefik.ingress.kubernetes.io/router.entrypoints: websecure
    traefik.ingress.kubernetes.io/router.tls: "true"
```

### HAProxy Ingress Controller

**Installation:**

```bash
helm repo add haproxy-ingress https://haproxy-ingress.github.io/charts
helm install haproxy-ingress haproxy-ingress/haproxy-ingress \
  --namespace haproxy --create-namespace
```

**RenOS Ingress Configuration:**

```yaml
ingress:
  enabled: true
  className: "haproxy"
  annotations:
    haproxy.org/ssl-redirect: "true"
    haproxy.org/timeout-server: "3600s"
    haproxy.org/timeout-tunnel: "3600s"
```

---

## TLS Configuration

### Using cert-manager

**Install cert-manager:**

```bash
helm repo add jetstack https://charts.jetstack.io
helm install cert-manager jetstack/cert-manager \
  --namespace cert-manager --create-namespace \
  --set installCRDs=true
```

**Create a ClusterIssuer:**

```yaml
# cluster-issuer.yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@example.com
    privateKeySecretRef:
      name: letsencrypt-prod-account-key
    solvers:
      - http01:
          ingress:
            class: nginx
```

```bash
kubectl apply -f cluster-issuer.yaml
```

**Configure Helm with cert-manager:**

```yaml
ingress:
  enabled: true
  className: "nginx"
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    acme.cert-manager.io/http01-edit-in-place: "true"
  tls:
    - hosts:
        - renos.example.com
      secretName: renos-tls
```

### Using Self-Signed Certificates

```bash
# Generate a self-signed certificate
openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
  -keyout tls.key -out tls.crt \
  -subj "/CN=renos.example.com" \
  -addext "subjectAltName=DNS:renos.example.com"

# Create Kubernetes secret
kubectl create secret tls renos-tls \
  --namespace renos \
  --cert=tls.crt \
  --key=tls.key
```

```yaml
# Helm values
ingress:
  enabled: true
  tls:
    - hosts:
        - renos.example.com
      secretName: renos-tls
```

### Using Existing Certificates

```bash
# Import existing certificates into Kubernetes
kubectl create secret tls renos-tls \
  --namespace renos \
  --cert=/path/to/fullchain.pem \
  --key=/path/to/privkey.pem

# Reference the secret in Helm values
secrets:
  existingSecret: "renos-tls"

ingress:
  enabled: true
  tls:
    - hosts:
        - renos.example.com
      secretName: renos-tls
```

---

## Resource Sizing Guide

### Environment-Based Sizing

| Environment | Components | CPU Total | Memory Total | Storage Total | Nodes |
|-------------|------------|-----------|-------------|---------------|-------|
| **Development** | core(1), hub(1), postgres, redis, nats(1) | 2 cores | 4 GB | 30 GB | 1 |
| **Testing** | core(2), hub(2), postgres, redis, nats(3) | 6 cores | 12 GB | 80 GB | 3 |
| **Small Production** | core(2), hub(2), postgres, redis, nats(3) | 8 cores | 16 GB | 120 GB | 3 |
| **Medium Production** | core(3), hub(3), postgres, redis, nats(3) | 12 cores | 24 GB | 200 GB | 3 |
| **Large Production** | core(5), hub(5), postgres, redis, nats(5) | 24 cores | 48 GB | 500 GB | 5 |

### Per-Pod Resource Recommendations

| Component | Small | Medium | Large | X-Large |
|-----------|-------|--------|-------|---------|
| **core** | 250m/256Mi | 500m/512Mi | 1/1Gi | 2/2Gi |
| **hub** | 250m/256Mi | 500m/512Mi | 1/1Gi | 2/2Gi |
| **agent** | 100m/128Mi | 250m/256Mi | 500m/512Mi | 1/1Gi |
| **postgres** | 500m/1Gi | 1/2Gi | 2/4Gi | 4/8Gi |
| **redis** | 100m/256Mi | 250m/512Mi | 500m/1Gi | 1/2Gi |
| **nats** | 250m/512Mi | 500m/1Gi | 1/2Gi | 2/4Gi |

### Device Scaling Estimates

| Number of Devices | Core Replicas | Hub Replicas | PostgreSQL Size | Redis Size | NATS Size |
|------------------|---------------|--------------|----------------|------------|-----------|
| < 100 | 1 | 1 | 10 GB | 5 GB | 10 GB |
| 100 - 1,000 | 2 | 2 | 20 GB | 5 GB | 10 GB |
| 1,000 - 10,000 | 3 | 3 | 50 GB | 10 GB | 20 GB |
| 10,000 - 100,000 | 5 | 5 | 100 GB | 20 GB | 50 GB |
| > 100,000 | Contact sales | Contact sales | Contact sales | Contact sales | Contact sales |

---

## Autoscaling Configuration

### Horizontal Pod Autoscaler

The Helm chart includes HPA configuration for both core and hub deployments:

```yaml
autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 80
  targetMemoryUtilizationPercentage: 80
```

This creates HPA resources:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: renos-core
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: renos-core
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 80
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
```

### Custom Metrics Autoscaling

```yaml
# Values with custom metrics
autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Pods
      pods:
        metric:
          name: http_requests_per_second
        target:
          type: AverageValue
          averageValue: 1000
```

### Vertical Pod Autoscaler

For VPA (optional):

```yaml
# Requires VPA installation
verticalPodAutoscaler:
  enabled: true
  updateMode: "Auto"
  controlledResources: ["cpu", "memory"]
```

---

## Network Policies

### Default Deny Policy

```yaml
networkPolicy:
  enabled: true
```

When enabled, the chart creates a default-deny network policy:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: renos-default-deny
spec:
  podSelector: {}
  policyTypes:
    - Ingress
    - Egress
```

### Component-Level Policies

The chart creates granular policies for each component:

| Policy | Source | Destination | Ports |
|--------|--------|-------------|-------|
| `allow-core` | hub, agent | core | 8080, 9090 |
| `allow-hub` | core | hub | 3000, 9091 |
| `allow-agent` | core | agent | 9000 |
| `allow-data-services` | core, hub | postgres | 5432 |
| `allow-redis` | core, hub | redis | 6379 |
| `allow-nats` | core, hub | nats | 4222, 6222, 8222 |

### Custom Network Policies

```yaml
# Additional custom policies can be added as template overrides
# Example: Allow monitoring namespace access
extraNetworkPolicies:
  - name: allow-prometheus
    spec:
      podSelector:
        matchLabels:
          app.kubernetes.io/component: core
      ingress:
        - from:
            - namespaceSelector:
                matchLabels:
                  kubernetes.io/metadata.name: monitoring
          ports:
            - port: 9090
```

---

## Prometheus/Grafana Integration

### ServiceMonitors

The chart creates ServiceMonitor resources for Prometheus Operator:

```yaml
monitoring:
  serviceMonitor:
    enabled: true
    interval: 30s
    scrapeTimeout: 10s
    labels:
      release: prometheus  # Match your Prometheus Operator release
```

This creates ServiceMonitors for:

- **Core**: Scrapes `/metrics` on port 9090
- **Hub**: Scrapes `/metrics` on port 9091

### Prometheus Rules

```yaml
monitoring:
  prometheusRules:
    enabled: true
    labels:
      release: prometheus
```

Available alerts:

| Alert | Severity | Condition | Duration |
|-------|----------|-----------|----------|
| `RenOSCoreDown` | critical | `up{component="core"} == 0` | 1m |
| `RenOSHubDown` | critical | `up{component="hub"} == 0` | 1m |
| `RenOSHighCPUUsage` | warning | CPU usage > 80% | 5m |
| `RenOSHighMemoryUsage` | warning | Memory usage > 90% | 5m |

### Grafana Dashboard

Import the RenOS dashboard into Grafana:

```json
{
  "dashboard": {
    "uid": "renos-overview",
    "title": "RenOS Platform Overview",
    "panels": [
      {
        "title": "Core API Request Rate",
        "type": "graph",
        "targets": [{
          "expr": "rate(http_requests_total{app_kubernetes_component=\"core\"}[5m])",
          "legendFormat": "{{ route }}"
        }]
      },
      {
        "title": "Core Response Latency",
        "type": "heatmap",
        "targets": [{
          "expr": "rate(http_request_duration_seconds_bucket{app_kubernetes_component=\"core\"}[5m])"
        }]
      },
      {
        "title": "Active Devices",
        "type": "stat",
        "targets": [{
          "expr": "renos_devices_connected"
        }]
      },
      {
        "title": "Database Connections",
        "type": "gauge",
        "targets": [{
          "expr": "pg_stat_activity_count{datname=\"renos\"}"
        }]
      },
      {
        "title": "NATS JetStream Storage",
        "type": "gauge",
        "targets": [{
          "expr": "nats_jetstream_store_bytes"
        }]
      },
      {
        "title": "Pod CPU Usage",
        "type": "graph",
        "targets": [{
          "expr": "sum(rate(container_cpu_usage_seconds_total{namespace=\"renos\"}[5m])) by (pod)"
        }]
      },
      {
        "title": "Pod Memory Usage",
        "type": "graph",
        "targets": [{
          "expr": "sum(container_memory_working_set_bytes{namespace=\"renos\"}) by (pod)"
        }]
      }
    ]
  }
}
```

---

## Upgrading via Helm

### Standard Upgrade

```bash
# Update repository
helm repo update renos

# Upgrade with existing values
helm upgrade renos renos/renos -n renos \
  --values values-production.yaml \
  --atomic \
  --timeout 15m \
  --cleanup-on-fail
```

### Preserving Current Values

```bash
# Option 1: Export current values, edit, then upgrade
helm get values renos -n renos -o yaml > current-values.yaml
# Edit current-values.yaml as needed
helm upgrade renos renos/renos -n renos \
  --values current-values.yaml \
  --atomic \
  --timeout 15m

# Option 2: Reuse current values (only chart changes applied)
helm upgrade renos renos/renos -n renos --reuse-values

# Option 3: Reset to chart defaults
helm upgrade renos renos/renos -n renos --reset-values
```

### Rollback on Failure

```bash
# The --atomic flag auto-rollbacks on failure
# Manual rollback:
helm rollback renos -n renos

# Rollback to specific revision
helm rollback renos -n renos <revision-number>

# Check revision history
helm history renos -n renos
```

### Zero-Downtime Upgrade

```yaml
# Ensure zero-downtime by configuring:
core:
  replicaCount: 2
  # Rolling update config (default)
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      maxSurge: 1

hub:
  replicaCount: 2
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      maxSurge: 1
```

```bash
# Perform zero-downtime upgrade
helm upgrade renos renos/renos -n renos \
  --values values-production.yaml \
  --atomic \
  --timeout 15m \
  --max-history 10
```

---

## Production Hardening

### Security Contexts

```yaml
# Values for hardened security
core:
  podSecurityContext:
    runAsUser: 1000
    runAsGroup: 1000
    fsGroup: 1000
    seccompProfile:
      type: RuntimeDefault
  containerSecurityContext:
    allowPrivilegeEscalation: false
    readOnlyRootFilesystem: true
    capabilities:
      drop: ["ALL"]
    runAsNonRoot: true

hub:
  podSecurityContext:
    runAsUser: 1000
    runAsGroup: 1000
    fsGroup: 1000
    seccompProfile:
      type: RuntimeDefault
  containerSecurityContext:
    allowPrivilegeEscalation: false
    capabilities:
      drop: ["ALL"]
    runAsNonRoot: true
```

### Pod Disruption Budgets

The chart includes PDBs to ensure high availability:

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: renos-core
spec:
  minAvailable: 1
  selector:
    matchLabels:
      app.kubernetes.io/component: core
```

### Resource Quotas

```yaml
# namespace-quota.yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: renos-quota
  namespace: renos
spec:
  hard:
    requests.cpu: "16"
    requests.memory: "32Gi"
    limits.cpu: "32"
    limits.memory: "64Gi"
    persistentvolumeclaims: "10"
    requests.storage: "500Gi"
```

### Pod Security Standards

```bash
# Apply Pod Security Standards
kubectl label namespace renos \
  pod-security.kubernetes.io/enforce=restricted \
  pod-security.kubernetes.io/warn=restricted
```

### Node Taints and Tolerations

```yaml
# For dedicated RenOS nodes
agent:
  tolerations:
    - key: "renos-workload"
      operator: "Equal"
      value: "true"
      effect: "NoSchedule"
  nodeSelector:
    renos-workload: "true"

core:
  tolerations:
    - key: "renos-workload"
      operator: "Equal"
      value: "true"
      effect: "NoSchedule"
  nodeSelector:
    renos-workload: "true"
```

### Backup and Disaster Recovery

```yaml
backup:
  enabled: true
  schedule: "0 2 * * *"
  storage:
    size: 20Gi
  retention:
    days: 30
    copies: 14
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 512Mi
```

### Resource Monitoring

```bash
# Install metrics server for HPA
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

# Install Prometheus Stack
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring --create-namespace
```

### Audit Logging

```yaml
# Enable Kubernetes audit logging for the RenOS namespace
# Add to kube-apiserver configuration:
# --audit-policy-file=/etc/kubernetes/audit-policy.yaml
# --audit-log-path=/var/log/kubernetes/audit.log
```

```yaml
# audit-policy.yaml
apiVersion: audit.k8s.io/v1
kind: Policy
rules:
  - level: RequestResponse
    resources:
      - group: ""
        resources: ["secrets", "configmaps"]
    namespaces: ["renos"]
  - level: Metadata
    resources:
      - group: ""
        resources: ["pods", "services", "deployments"]
    namespaces: ["renos"]
```
