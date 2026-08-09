# RenOS Cluster Deployment Guide

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Prerequisites for Cluster](#prerequisites-for-cluster)
- [Multi-Node Docker Deployment](#multi-node-docker-deployment)
- [Kubernetes Cluster Deployment (Multi-AZ)](#kubernetes-cluster-deployment-multi-az)
- [NATS Cluster Configuration](#nats-cluster-configuration)
- [Database Replication](#database-replication)
- [Load Balancing](#load-balancing)
- [Monitoring Cluster Health](#monitoring-cluster-health)
- [Scaling Up/Down](#scaling-updown)
- [Cluster Backup Strategies](#cluster-backup-strategies)

---

## Architecture Overview

RenOS supports clustered deployment for high availability, load distribution, and disaster recovery. The cluster architecture consists of multiple nodes working together to provide a unified device management platform.

### Cluster Components

```
                         ┌─────────────────────────────────────────────┐
                         │         Load Balancer (HAProxy/Nginx)       │
                         │              (Active/Passive)               │
                         └──────────────────┬──────────────────────────┘
                                            │
            ┌───────────────────────────────┼───────────────────────────────┐
            │                               │                               │
            ▼                               ▼                               ▼
   ┌──────────────────┐          ┌──────────────────┐          ┌──────────────────┐
   │    Node 1        │          │    Node 2        │          │    Node 3        │
   │  (Leader/Master) │◄────────►│  (Follower)      │◄────────►│  (Follower)      │
   └────────┬─────────┘          └────────┬─────────┘          └────────┬─────────┘
            │                              │                             │
            ▼                              ▼                             ▼
   ┌──────────────────┐          ┌──────────────────┐          ┌──────────────────┐
   │   Replicated     │◄────────►│   Replicated     │◄────────►│   Replicated     │
   │   PostgreSQL     │          │   PostgreSQL     │          │   PostgreSQL     │
   └──────────────────┘          └──────────────────┘          └──────────────────┘
   
   ┌──────────────────┐          ┌──────────────────┐          ┌──────────────────┐
   │   NATS Cluster   │◄────────►│   NATS Cluster   │◄────────►│   NATS Cluster   │
   │   (Full Mesh)    │          │   (Full Mesh)    │          │   (Full Mesh)    │
   └──────────────────┘          └──────────────────┘          └──────────────────┘
   
   ┌──────────────────┐          ┌──────────────────┐          ┌──────────────────┐
   │   Redis Sentinel  │◄────────►│   Redis Sentinel  │◄────────►│   Redis Sentinel  │
   │   (Sentinel +     │          │   (Sentinel +     │          │   (Sentinel +     │
   │    Replication)   │          │    Replication)   │          │    Replication)   │
   └──────────────────┘          └──────────────────┘          └──────────────────┘
```

### Node Roles

| Role | Function | Requirements |
|------|----------|-------------|
| **Leader** | Primary API server, write database, NATS route coordinator | 4 CPU, 8 GB RAM, SSD |
| **Follower** | Read replicas, API load distribution, NATS route members | 4 CPU, 8 GB RAM, SSD |
| **Agent-only** | Device management only, no database | 2 CPU, 4 GB RAM |

### Communication Flow

```
  Device ──► Load Balancer ──► Hub (Node 1/2/3)
                                   │
                                   ▼
                              Core API ──► PostgreSQL (Leader)
                                   │            │
                                   ▼            ▼
                              NATS Cluster ──► PostgreSQL (Replicas)
                                   │
                                   ▼
                              Agent (Remote Device)
```

### NATS Cluster Topology

NATS forms a full-mesh cluster where every node connects to every other node:

```
     ┌──────────┐
     │ Node 1   │◄────► Node 2
     │ 4222     │        │ 4222
     └────┬─────┘        │
          │              │
          ▼              ▼
     Node 3 ◄────────────┘
     │ 4222
```

Key characteristics:
- **Full mesh**: Every node connects to every other node
- **JetStream enabled**: Persistent message storage across cluster
- **Automatic failover**: Clients reconnect to available nodes
- **Raft consensus**: Meta-leader election for JetStream

### Database Replication Topology

PostgreSQL streaming replication:

```
       Leader (Primary)
           │
     ┌─────┼─────┐
     │     │     │
     ▼     ▼     ▼
  Replica1 Replica2 Replica3
  (Sync)  (Async)  (Async)
```

---

## Prerequisites for Cluster

### Hardware Requirements

| Component | Per Node (Minimum) | Per Node (Recommended) |
|-----------|-------------------|----------------------|
| CPU | 4 cores | 8 cores |
| RAM | 8 GB | 16 GB |
| Disk | 50 GB SSD | 100 GB NVMe SSD |
| Network | 1 Gbps | 10 Gbps |
| Nodes | 3 | 3-7 |

### Network Requirements

| Connection | Port | Protocol | Purpose |
|------------|------|----------|---------|
| Node-to-Node | 4222 | TCP | NATS client connections |
| Node-to-Node | 6222 | TCP | NATS cluster routing |
| Node-to-Node | 8222 | TCP | NATS monitoring |
| Node-to-Node | 5432 | TCP | PostgreSQL replication |
| Node-to-Node | 6379 | TCP | Redis replication |
| Node-to-Node | 26379 | TCP | Redis Sentinel |
| Load Balancer | 80 | TCP | HTTP |
| Load Balancer | 443 | TCP | HTTPS |
| Load Balancer | 8443 | TCP | Core API |

### Software Requirements

| Software | Version | Notes |
|----------|---------|-------|
| Docker Engine | 24.x+ | All nodes |
| Docker Compose | 2.24+ | All nodes |
| Kubernetes | 1.24+ | For K8s deployment |
| Helm | 3.8+ | For K8s deployment |
| Keepalived | 2.x | For VIP failover (optional) |
| HAProxy | 2.x | For load balancing |

### Latency Requirements

| Connection | Max Latency | Recommended |
|------------|-------------|-------------|
| Node-to-Node (within AZ) | < 1 ms | < 0.5 ms |
| Node-to-Node (cross-AZ) | < 10 ms | < 5 ms |
| Nodes to Load Balancer | < 5 ms | < 1 ms |
| NATS message delivery | < 100 ms | < 10 ms |
| Database replication | < 50 ms | < 10 ms |

### DNS Configuration

```bash
# Each node needs a unique hostname and DNS record
renos-node1.example.com  A 10.0.1.10
renos-node2.example.com  A 10.0.1.11
renos-node3.example.com  A 10.0.1.12

# Cluster VIP (for load balancer)
renos-cluster.example.com  A 10.0.1.100
```

### Time Synchronization

All cluster nodes must have synchronized clocks:

```bash
# Install and configure NTP on all nodes
sudo apt-get install chrony  # Debian/Ubuntu
sudo dnf install chrony      # Fedora/RHEL

# Configure time sync
sudo systemctl enable --now chronyd

# Verify synchronization
chronyc tracking
```

---

## Multi-Node Docker Deployment

### Overview

This deployment method uses Docker Compose across multiple physical or virtual machines, with NATS for inter-node messaging and PostgreSQL streaming replication for database redundancy.

### Step-by-Step Deployment

**Step 1: Configure each node**

On each node, install RenOS:

```bash
curl -fsSL https://install.renos.ai -o install.sh
sudo bash install.sh
```

**Step 2: Assign node roles**

```bash
# On the primary node
echo "RENOS_NODE_ROLE=leader" >> /etc/renos/renos.env

# On follower nodes
echo "RENOS_NODE_ROLE=follower" >> /etc/renos/renos.env
```

**Step 3: Configure NATS cluster**

On each node, edit `/etc/renos/docker-compose.yml` to add NATS cluster configuration:

```yaml
nats:
  image: nats:2.10-alpine
  container_name: renos-nats
  restart: always
  ports:
    - "4222:4222"
    - "6222:6222"
    - "8222:8222"
  command:
    - --jetstream
    - --http_port=8222
    - --cluster_name=renos
    - --cluster=nats://0.0.0.0:6222
    - --routes=nats://renos-node1:6222,nats://renos-node2:6222,nats://renos-node3:6222
    - --store_dir=/data
  volumes:
    - nats-data:/data
```

**Step 4: Configure PostgreSQL replication**

On the leader node, enable replication in `docker-compose.yml`:

```yaml
postgres:
  image: postgres:16-alpine
  container_name: renos-postgres
  restart: always
  ports:
    - "5432:5432"
  environment:
    POSTGRES_USER: renos
    POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    POSTGRES_DB: renos
    POSTGRES_CONFIG_FILE: /etc/postgresql/postgresql.conf
  volumes:
    - pgdata:/var/lib/postgresql/data
    - ./postgresql-leader.conf:/etc/postgresql/postgresql.conf:ro
```

On follower nodes:

```yaml
postgres:
  image: postgres:16-alpine
  container_name: renos-postgres
  restart: always
  ports:
    - "5432:5432"
  environment:
    POSTGRES_USER: renos
    POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    POSTGRES_DB: renos
    POSTGRES_CONFIG_FILE: /etc/postgresql/postgresql.conf
  volumes:
    - pgdata:/var/lib/postgresql/data
    - ./postgresql-replica.conf:/etc/postgresql/postgresql.conf:ro
```

**Step 5: Create leader PostgreSQL configuration**

```bash
# postgresql-leader.conf
cat > /etc/renos/postgresql-leader.conf << 'EOF'
wal_level = replica
max_wal_senders = 5
wal_keep_size = 1024
hot_standby = on
listen_addresses = '*'
EOF
```

**Step 6: Create replica PostgreSQL configuration**

```bash
# postgresql-replica.conf
cat > /etc/renos/postgresql-replica.conf << 'EOF'
wal_level = replica
hot_standby = on
primary_conninfo = 'host=renos-node1 port=5432 user=replicator password=replicator_password'
listen_addresses = '*'
EOF
```

**Step 7: Configure Redis Sentinel**

```yaml
redis:
  image: redis:7-alpine
  container_name: renos-redis
  restart: always
  ports:
    - "6379:6379"
  command:
    - redis-server
    - --requirepass ${REDIS_PASSWORD}
    - --maxmemory 128mb
    - --maxmemory-policy allkeys-lru
    - --appendonly yes
    - --replicaof renos-node1 6379
  volumes:
    - redis-data:/data

redis-sentinel:
  image: redis:7-alpine
  container_name: renos-redis-sentinel
  restart: always
  ports:
    - "26379:26379"
  command:
    - redis-sentinel
    - /etc/redis/sentinel.conf
  volumes:
    - ./sentinel.conf:/etc/redis/sentinel.conf:ro
```

**Step 8: Configure load balancer**

```yaml
haproxy:
  image: haproxy:2.8-alpine
  container_name: renos-haproxy
  restart: always
  ports:
    - "80:80"
    - "443:443"
    - "8443:8443"
    - "8404:8404"
  volumes:
    - ./haproxy.cfg:/usr/local/etc/haproxy/haproxy.cfg:ro
    - ./tls:/etc/ssl/renos:ro
```

**Step 9: Start the cluster**

On each node:

```bash
docker compose up -d
```

---

## Kubernetes Cluster Deployment (Multi-AZ)

### Architecture for Multi-AZ

For Kubernetes deployments across multiple availability zones:

```
                    ┌──────────────────────────┐
                    │  Ingress Controller       │
                    │  (nginx/traefik/haproxy)  │
                    └──────────┬───────────────┘
                               │
          ┌────────────────────┼────────────────────┐
          │                    │                     │
          ▼                    ▼                     ▼
   ┌──────────────┐    ┌──────────────┐    ┌──────────────┐
   │   AZ 1       │    │   AZ 2       │    │   AZ 3       │
   │              │    │              │    │              │
   │ ┌──────────┐ │    │ ┌──────────┐ │    │ ┌──────────┐ │
   │ │ Core     │ │    │ │ Core     │ │    │ │ Core     │ │
   │ │ Hub      │ │    │ │ Hub      │ │    │ │ Hub      │ │
   │ │ Agent    │ │    │ │ Agent    │ │    │ │ Agent    │ │
   │ │ NATS     │ │    │ │ NATS     │ │    │ │ NATS     │ │
   │ │ Postgres │ │    │ │ Postgres │ │    │ │ Postgres │ │
   │ │ Redis    │ │    │ │ Redis    │ │    │ │ Redis    │ │
   │ └──────────┘ │    │ └──────────┘ │    │ └──────────┘ │
   └──────────────┘    └──────────────┘    └──────────────┘
```

### Helm Deployment for Multi-AZ

```bash
# Create a values file for multi-AZ deployment
cat > values-multi-az.yaml << 'EOF'
global:
  env:
    RENOS_EXTERNAL_URL: https://renos.example.com

core:
  replicaCount: 3
  podAntiAffinity: preferred
  topologySpreadConstraints:
    - maxSkew: 1
      topologyKey: topology.kubernetes.io/zone
      whenUnsatisfiable: ScheduleAnyway

hub:
  replicaCount: 3
  podAntiAffinity: preferred
  topologySpreadConstraints:
    - maxSkew: 1
      topologyKey: topology.kubernetes.io/zone
      whenUnsatisfiable: ScheduleAnyway

nats:
  cluster:
    enabled: true
    replicas: 3
  podAntiAffinity: required
  topologySpreadConstraints:
    - maxSkew: 1
      topologyKey: topology.kubernetes.io/zone
      whenUnsatisfiable: DoNotSchedule

postgres:
  mode: replication
  replicas: 3
  synchronousMode: true
  podAntiAffinity: required
  topologySpreadConstraints:
    - maxSkew: 1
      topologyKey: topology.kubernetes.io/zone
      whenUnsatisfiable: DoNotSchedule

redis:
  mode: cluster
  sentinel:
    enabled: true
    replicas: 3
  podAntiAffinity: required
EOF

# Deploy
helm upgrade --install renos renos/renos -n renos --create-namespace \
  --values values-multi-az.yaml
```

### Storage Considerations for Multi-AZ

Each AZ should have its own storage class:

```yaml
# values-storage.yaml
persistence:
  postgres:
    storageClass: "ebs-gp3-az1"  # or "topology-aware" provisioner
    size: 50Gi
  redis:
    storageClass: "ebs-gp3-az1"
    size: 10Gi
  nats:
    storageClass: "ebs-gp3-az1"
    size: 20Gi

# Use CSI topology-aware storage when possible
# For AWS EBS:
storageClasses:
  - name: ebs-topology-aware
    provisioner: ebs.csi.aws.com
    volumeBindingMode: WaitForFirstConsumer
    allowedTopologies:
      - topology.kubernetes.io/zone: us-east-1a
      - topology.kubernetes.io/zone: us-east-1b
      - topology.kubernetes.io/zone: us-east-1c
```

---

## NATS Cluster Configuration

### NATS Cluster Setup

NATS JetStream with Raft-based meta-group for fault tolerance:

```yaml
# Docker Compose NATS cluster configuration
nats:
  image: nats:2.10-alpine
  container_name: renos-nats
  restart: always
  ports:
    - "4222:4222"    # Client connections
    - "6222:6222"    # Cluster routing
    - "8222:8222"    # Monitoring / HTTP
  command:
    - --jetstream
    - --http_port=8222
    - --cluster_name=renos
    - --cluster=nats://0.0.0.0:6222
    - --routes=nats://renos-node1:6222,nats://renos-node2:6222,nats://renos-node3:6222
    - --store_dir=/data
    - --max_payload=8MB
    - --max_connections=10000
    - --server_name=renos-nats-1
  volumes:
    - nats-data:/data
  healthcheck:
    test: ["CMD-SHELL", "wget -q http://localhost:8222/healthz -O /dev/null || exit 1"]
    interval: 10s
    timeout: 5s
    retries: 5
    start_period: 10s
```

### NATS Cluster Verification

```bash
# Check cluster members
docker compose exec nats nats server check cluster

# Expected output:
# +------------------------+---------+--------+
# | Server Name            | Cluster | Status |
# +------------------------+---------+--------+
# | renos-nats-1           | renos   | OK     |
# | renos-nats-2           | renos   | OK     |
# | renos-nats-3           | renos   | OK     |
# +------------------------+---------+--------+

# Check routes between nodes
docker compose exec nats nats server report routes

# Check JetStream cluster info
docker compose exec nats nats stream list

# Check connection information
docker compose exec nats nats server report connections
```

### NATS Configuration Reference

| Parameter | Description | Default | Recommended |
|-----------|-------------|---------|-------------|
| `--jetstream` | Enable JetStream persistence | disabled | enabled |
| `--cluster_name` | Cluster name for routing | generated | renos |
| `--cluster` | Cluster listen address | none | 0.0.0.0:6222 |
| `--routes` | Seed routes for cluster join | none | All node URLs |
| `--max_payload` | Maximum message size | 1MB | 8MB |
| `--max_connections` | Maximum client connections | 65536 | 10000 |
| `--server_name` | Unique server name | hostname | Descriptive name |

### NATS Client Configuration

```bash
# Each core service connects to all NATS nodes
RENOS_NATS_URL="nats://renos-node1:4222,nats://renos-node2:4222,nats://renos-node3:4222"

# Client-side failover is automatic
# If a NATS server goes down, clients reconnect to another node
```

### NATS JetStream Configuration

```bash
# Configure JetStream storage limits
docker compose exec nats nats stream edit <stream-name> \
  --max-msgs=-1 \
  --max-bytes=1G \
  --max-age=7d \
  --storage file \
  --replicas=3
```

---

## Database Replication

### PostgreSQL Streaming Replication

**Leader node configuration:**

```bash
# postgresql-leader.conf
cat > /etc/renos/postgresql-leader.conf << 'EOF'
wal_level = replica
max_wal_senders = 10
wal_keep_size = 2048
hot_standby = on
listen_addresses = '*'
max_replication_slots = 10
synchronous_standby_names = 'FIRST 1 (*)'
EOF
```

**Replica node configuration:**

```bash
# postgresql-replica.conf
cat > /etc/renos/postgresql-replica.conf << 'EOF'
wal_level = replica
hot_standby = on
primary_conninfo = 'host=10.0.1.10 port=5432 user=replicator password=replicator_password application_name=replica1'
primary_slot_name = 'replica1'
listen_addresses = '*'
hot_standby_feedback = on
EOF
```

**Creating replication slot on leader:**

```bash
docker compose exec postgres psql -U renos -c \
  "SELECT * FROM pg_create_physical_replication_slot('replica1');"
```

**Setting up a replica:**

```bash
# On the replica node, perform a base backup from the leader
docker compose exec postgres bash -c \
  "pg_basebackup -h 10.0.1.10 -D /var/lib/postgresql/data -U replicator -P -v --wal-method=stream"

# Start the replica
docker compose start postgres
```

### Monitoring Replication

```bash
# On the leader - check replication status
docker compose exec postgres psql -U renos -x -c "SELECT * FROM pg_stat_replication;"

# Output:
# -[ RECORD 1 ]----+------------------------------
# pid              | 12345
# application_name  | replica1
# state            | streaming
# sync_state       | sync
# sync_priority    | 1
# write_lag        | 00:00:00.001234
# flush_lag        | 00:00:00.001567
# replay_lag       | 00:00:00.001890

# On the replica - check recovery status
docker compose exec postgres psql -U renos -c "SELECT pg_is_in_recovery();"
docker compose exec postgres psql -U renos -c "SELECT pg_last_wal_receive_lsn();"
docker compose exec postgres psql -U renos -c "SELECT pg_last_wal_replay_lsn();"
```

### Promoting a Replica to Leader

```bash
# Manual promotion
docker compose exec postgres psql -U renos -c "SELECT pg_promote();"

# Or via trigger file
docker compose exec postgres touch /tmp/promote
```

### Failover Configuration

For automated failover, use a tool like Patroni or repmgr:

```bash
# Example with Patroni (requires additional setup)
# Patroni handles:
# - Automatic leader election
# - Replica configuration
# - Health checking
# - Failover coordination

# Basic Patroni configuration
cat > /etc/renos/patroni.yml << 'EOF'
scope: renos
namespace: /service/
name: renos-db-1

restapi:
  listen: 0.0.0.0:8008
  connect_address: 10.0.1.10:8008

etcd:
  host: 10.0.1.10:2379

bootstrap:
  dcs:
    ttl: 30
    loop_wait: 10
    retry_timeout: 10
    maximum_lag_on_failover: 1048576
    postgresql:
      use_pg_rewind: true
      parameters:
        wal_level: replica
        hot_standby: "on"
        max_connections: 200

  initdb:
    - encoding: UTF8

  pg_hba:
    - host replication replicator 10.0.0.0/8 md5

postgresql:
  listen: 0.0.0.0:5432
  connect_address: 10.0.1.10:5432
  data_dir: /var/lib/postgresql/data
  bin_dir: /usr/lib/postgresql/16/bin
  authentication:
    replication:
      username: replicator
      password: replicator_password
    superuser:
      username: renos
      password: secure_password
EOF
```

---

## Load Balancing

### HAProxy Configuration

```bash
# /etc/renos/haproxy.cfg
cat > /etc/renos/haproxy.cfg << 'EOF'
global
    log /dev/log local0
    maxconn 4096
    user haproxy
    group haproxy
    daemon
    stats socket /var/run/haproxy/admin.sock mode 660 level admin
    ssl-default-bind-ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256
    ssl-default-bind-options no-sslv3 no-tlsv10 no-tlsv11

defaults
    log global
    mode http
    option httplog
    option dontlognull
    option forwardfor
    option http-server-close
    retries 3
    timeout http-request 10s
    timeout queue 1m
    timeout connect 10s
    timeout client 1m
    timeout server 1m
    timeout http-keep-alive 10s
    timeout check 10s
    maxconn 2048

# Frontend: HTTPS (Hub UI + API)
frontend renos-https
    bind *:443 ssl crt /etc/ssl/renos/fullchain.pem
    bind *:80
    redirect scheme https if !{ ssl_fc }
    
    # Path-based routing
    acl is_api path_beg /api
    acl is_ws  path_beg /api/v1/ws
    
    use_backend renos-core if is_api
    use_backend renos-hub if !is_api
    
    # WebSocket support
    option http-server-close
    timeout tunnel 1h

# Backend: Core API
backend renos-core
    balance roundrobin
    option httpchk GET /api/v1/healthz
    http-check expect status 200
    default-server inter 3s fall 3 rise 2
    
    server renos-node1 10.0.1.10:8443 check ssl verify none
    server renos-node2 10.0.1.11:8443 check ssl verify none
    server renos-node3 10.0.1.12:8443 check ssl verify none

# Backend: Hub UI
backend renos-hub
    balance roundrobin
    option httpchk GET /login
    http-check expect status 200
    default-server inter 3s fall 3 rise 2
    cookie RENOS_SERVER insert indirect nocache
    
    server renos-node1 10.0.1.10:3000 check cookie node1
    server renos-node2 10.0.1.11:3000 check cookie node2
    server renos-node3 10.0.1.12:3000 check cookie node3

# Stats page
frontend stats
    bind *:8404
    stats enable
    stats uri /stats
    stats refresh 10s
    stats auth admin:password
EOF
```

### Nginx Load Balancing

```nginx
# /etc/nginx/nginx.conf (alternative to HAProxy)
upstream renos-core {
    least_conn;
    server 10.0.1.10:8443 max_fails=3 fail_timeout=30s;
    server 10.0.1.11:8443 max_fails=3 fail_timeout=30s;
    server 10.0.1.12:8443 max_fails=3 fail_timeout=30s;
    keepalive 32;
}

upstream renos-hub {
    ip_hash;
    server 10.0.1.10:3000 max_fails=3 fail_timeout=30s;
    server 10.0.1.11:3000 max_fails=3 fail_timeout=30s;
    server 10.0.1.12:3000 max_fails=3 fail_timeout=30s;
    keepalive 32;
}

server {
    listen 443 ssl http2;
    server_name renos.example.com;
    
    ssl_certificate /etc/renos/tls/fullchain.pem;
    ssl_certificate_key /etc/renos/tls/privkey.pem;
    
    location /api/ {
        proxy_pass https://renos-core;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
    
    location / {
        proxy_pass http://renos-hub;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

### Kubernetes Ingress Load Balancing

```yaml
# Kubernetes Ingress with path-based routing
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: renos
  namespace: renos
  annotations:
    kubernetes.io/ingress.class: nginx
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/use-regex: "true"
    nginx.ingress.kubernetes.io/proxy-body-size: "64m"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
spec:
  tls:
    - hosts:
        - renos.example.com
      secretName: renos-tls
  rules:
    - host: renos.example.com
      http:
        paths:
          - path: /api/v1/ws
            pathType: Prefix
            backend:
              service:
                name: renos-core
                port:
                  number: 8080
          - path: /api
            pathType: Prefix
            backend:
              service:
                name: renos-core
                port:
                  number: 8080
          - path: /
            pathType: Prefix
            backend:
              service:
                name: renos-hub
                port:
                  number: 3000
```

---

## Monitoring Cluster Health

### Health Endpoints

Each node exposes health endpoints:

```bash
# Core API health
curl -k https://node1:8443/api/v1/healthz

# NATS monitoring
curl http://node1:8222/healthz
curl http://node1:8222/varz
curl http://node1:8222/routez

# PostgreSQL
docker compose exec postgres pg_isready -U renos
```

### Prometheus Metrics

Core and Hub expose Prometheus-compatible metrics:

```bash
# Core metrics
curl -k https://node1:8443/metrics

# Hub metrics
curl -k https://node1:3000/metrics

# NATS metrics
curl http://node1:8222/metrics
```

### Grafana Dashboards

Import the RenOS Grafana dashboards:

```json
{
  "dashboard": {
    "title": "RenOS Cluster Overview",
    "panels": [
      {
        "title": "Node Status",
        "type": "stat",
        "targets": [
          {
            "expr": "up{app_kubernetes_component=~\"core|hub\"}"
          }
        ]
      },
      {
        "title": "NATS Cluster Health",
        "type": "table",
        "targets": [
          {
            "expr": "nats_server_routes"
          }
        ]
      },
      {
        "title": "Database Replication Lag",
        "type": "gauge",
        "targets": [
          {
            "expr": "pg_replication_lag"
          }
        ]
      },
      {
        "title": "API Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total[5m])"
          }
        ]
      }
    ]
  }
}
```

### Cluster Health Script

```bash
#!/bin/bash
# renos-cluster-health.sh

NODES=("renos-node1" "renos-node2" "renos-node3")

echo "=== RenOS Cluster Health Check ==="
echo ""

for node in "${NODES[@]}"; do
    echo "--- Node: $node ---"
    
    # Check if node is reachable
    if ping -c 1 -W 2 "$node" &>/dev/null; then
        echo "  Reachable: YES"
        
        # Check API health
        api_status=$(curl -sk -o /dev/null -w "%{http_code}" "https://$node:8443/api/v1/healthz" 2>/dev/null || echo "000")
        echo "  API Status: $api_status"
        
        # Check NATS health
        nats_status=$(curl -s -o /dev/null -w "%{http_code}" "http://$node:8222/healthz" 2>/dev/null || echo "000")
        echo "  NATS Health: $nats_status"
        
        # Check Docker compose
        ssh "$node" "docker compose -f /etc/renos/docker-compose.yml ps --format 'table {{.Name}}\t{{.Status}}'"
    else
        echo "  Reachable: NO"
    fi
    echo ""
done

echo "=== Check Complete ==="
```

### Alerting Rules

```yaml
# Prometheus alerting rules
groups:
  - name: renos-cluster
    rules:
      - alert: RenosNodeDown
        expr: up{app="renos"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "RenOS node {{ $labels.instance }} is down"

      - alert: RenosReplicationLag
        expr: pg_replication_lag > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Database replication lag on {{ $labels.instance }}"

      - alert: RenosNATSFailure
        expr: nats_routes < 2
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "NATS cluster connectivity degraded"

      - alert: RenosHighMemoryUsage
        expr: node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes < 0.1
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Node {{ $labels.instance }} memory below 10%"
```

---

## Scaling Up/Down

### Horizontal Scaling (Docker)

**Adding a new node:**

```bash
# On the new node:
# 1. Install RenOS
# 2. Add to NATS routes
# 3. Join as PostgreSQL replica
# 4. Update load balancer configuration

# Update NATS routes on existing nodes to include the new node
# Edit docker-compose.yml on each existing node:
# --routes=nats://node1:6222,nats://node2:6222,nats://node3:6222,nats://node4:6222

# Restart NATS on all nodes
docker compose up -d nats
```

**Removing a node:**

```bash
# 1. Remove from load balancer
# 2. Promote any replicas if this is the leader
# 3. Remove from NATS routes
# 4. Stop services on the node
# 5. Update remaining nodes' configurations
```

### Horizontal Scaling (Kubernetes)

```bash
# Scale core and hub deployments
kubectl scale deployment renos-core -n renos --replicas=5
kubectl scale deployment renos-hub -n renos --replicas=5

# Enable autoscaling
helm upgrade renos renos/renos -n renos \
  --set autoscaling.enabled=true \
  --set autoscaling.minReplicas=3 \
  --set autoscaling.maxReplicas=10 \
  --set autoscaling.targetCPUUtilizationPercentage=80
```

### Vertical Scaling

```bash
# Increase resource limits in docker-compose.yml
renos-core:
  deploy:
    resources:
      limits:
        cpus: "2"
        memory: 1G
      reservations:
        memory: 512M

# Apply changes
docker compose up -d renos-core
```

---

## Cluster Backup Strategies

### Recommended Backup Strategy for Clusters

| Component | Strategy | Frequency | Retention |
|-----------|----------|-----------|-----------|
| PostgreSQL | pg_dump from follower | Every 6 hours | 14 days local, 90 days S3 |
| NATS JetStream | File-level backup | Daily | 7 days |
| Redis | RDB snapshot | Hourly | 3 days |
| Configuration | File archive | On change | 30 days |
| TLS certificates | File archive | On renewal | 1 year |

### Backup from Follower

Always back up the database from a follower to avoid load on the leader:

```bash
# Backup script for follower node
#!/bin/bash
# Save as /usr/local/bin/renos-cluster-backup.sh

FOLLOWER_HOST="renos-node2"
BACKUP_DIR="/var/lib/renos/backups"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)

# Backup from follower
PGPASSWORD=$POSTGRES_PASSWORD pg_dump \
  -h $FOLLOWER_HOST \
  -U renos \
  -d renos \
  -F c \
  -Z 9 \
  -f "${BACKUP_DIR}/renos-cluster-db-${TIMESTAMP}.dump"

# Verify backup
pg_restore -l "${BACKUP_DIR}/renos-cluster-db-${TIMESTAMP}.dump" > /dev/null \
  && echo "Backup verified: ${TIMESTAMP}"
```

### NATS JetStream Backup

```bash
# Backup NATS JetStream data
docker compose exec nats nats stream backup \
  --dir /tmp/nats-backup \
  --full

# Archive the backup
tar -czf /var/lib/renos/backups/nats-backup-$(date +%Y%m%d).tar.gz -C /tmp nats-backup
```

### Cluster Configuration Backup

```bash
# Snapshot all cluster configurations
tar -czf /var/lib/renos/backups/cluster-config-$(date +%Y%m%d).tar.gz \
  /etc/renos/ \
  /var/lib/renos/backups/
```

### Automated Backup Cron

```bash
# /etc/cron.d/renos-cluster-backup
# Cluster backup schedule

# Full backup from follower every 6 hours
0 */6 * * * root /usr/local/bin/renos-cluster-backup.sh --full --remote

# Database-only backup hourly
0 * * * * root /usr/local/bin/renos-cluster-backup.sh --db-only

# Configuration backup daily
0 0 * * * root /usr/local/bin/renos-cluster-backup.sh --config

# Prune old backups
0 3 * * * root renos-backup.sh --prune
```
