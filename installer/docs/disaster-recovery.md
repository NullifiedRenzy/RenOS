# RenOS Disaster Recovery Plan

## Table of Contents

- [RPO/RTO Targets](#rporto-targets)
- [Backup Frequency Recommendations](#backup-frequency-recommendations)
- [Backup Storage Strategies](#backup-storage-strategies)
- [Testing Backups](#testing-backups)
- [DR Drill Procedures](#dr-drill-procedures)
- [Failover Plan](#failover-plan)
- [Restoration Time Estimates](#restoration-time-estimates)
- [Communication Plan During Incidents](#communication-plan-during-incidents)
- [Post-Incident Review Checklist](#post-incident-review-checklist)

---

## RPO/RTO Targets

### Recovery Point Objective (RPO)

RPO defines the maximum acceptable age of data that must be restored from backup in the event of a failure.

| Data Type | Target RPO | Maximum Tolerable RPO | Notes |
|-----------|-----------|----------------------|-------|
| Device configurations | 1 hour | 4 hours | Changes tracked in database |
| User accounts | 1 hour | 4 hours | Low change frequency |
| Workflow definitions | 1 hour | 4 hours | Moderate change frequency |
| Workflow execution history | 24 hours | 72 hours | High volume, low criticality |
| Audit logs | 24 hours | 72 hours | Append-only |
| TLS certificates | 90 days (renewal cycle) | N/A | External source of truth |
| NATS JetStream messages | 1 hour | 4 hours | Transient data |
| Redis cache | 0 (in-memory) | 1 hour | Regeneratable from database |

### Recovery Time Objective (RTO)

RTO defines the maximum acceptable time to restore services after a failure.

| Scenario | Target RTO | Maximum Tolerable RTO | Priority |
|----------|-----------|----------------------|----------|
| Single service crash | 5 minutes | 15 minutes | P0 |
| Database corruption | 30 minutes | 2 hours | P0 |
| Failed upgrade | 15 minutes | 1 hour | P0 |
| Hardware failure (single node) | 2 hours | 8 hours | P1 |
| Hardware failure (cluster) | 5 minutes (failover) | 30 minutes | P0 |
| Full site failure | 4 hours | 24 hours | P0 |
| Accidental data deletion | 1 hour | 4 hours | P1 |
| Security breach | 2 hours | 8 hours | P0 |
| NATS cluster failure | 5 minutes | 30 minutes | P1 |
| TLS certificate expiry | 1 hour | 4 hours | P1 |

### RPO/RTO Matrix

```
                    RTO
            ┌──────┬──────┬──────┬──────┐
            │ <5m  │ <30m │ <2h  │ <8h  │
     ┌──────┼──────┼──────┼──────┼──────┤
     │ 0    │Cache │  --  │  --  │  --  │
     │      │      │      │      │      │
RPO  ├──────┼──────┼──────┼──────┼──────┤
     │ <1h  │Svc   │ DB   │ Acc. │  --  │
     │      │crash │corrupt│del   │      │
     ├──────┼──────┼──────┼──────┼──────┤
     │<24h  │  --  │  --  │Hard- │Sec.  │
     │      │      │      │ware  │breach│
     ├──────┼──────┼──────┼──────┼──────┤
     │<72h  │  --  │  --  │Site  │  --  │
     │      │      │      │fail  │      │
     └──────┴──────┴──────┴──────┴──────┘
```

### Cost vs. RPO/RTO Tradeoffs

| Protection Level | RPO | RTO | Relative Cost | Method |
|-----------------|-----|-----|---------------|--------|
| Bronze | 24 hours | 8 hours | 1x | Daily backups, local only |
| Silver | 4 hours | 2 hours | 2x | Daily backups + offsite |
| Gold | 1 hour | 30 minutes | 3x | Hourly backups + streaming replication |
| Platinum | 0 | 5 minutes | 5x | Synchronous replication + HA cluster |

---

## Backup Frequency Recommendations

### By Environment

| Environment | Database | Redis | Files | Config | TLS Certs |
|-------------|----------|-------|-------|--------|-----------|
| **Development** | Daily | Daily | Weekly | On change | Monthly |
| **Staging** | Every 6 hours | Every 6 hours | Daily | On change | Monthly |
| **Production** | Every 1 hour | Every 1 hour | Every 6 hours | On change | On renewal |
| **Critical Production** | Every 15 minutes (WAL archiving) | Every 15 minutes | Every hour | On change | On renewal |

### By Data Criticality

| Data | Frequency | Retention (Local) | Retention (Remote) |
|------|-----------|-------------------|-------------------|
| Database (full) | Every 6 hours | 14 days | 90 days |
| Database (WAL) | Continuous | 7 days | 30 days |
| Redis | Every hour | 3 days | 7 days |
| Device configs | Every 6 hours | 14 days | 90 days |
| Workflow history | Daily | 30 days | 90 days |
| Audit logs | Daily | 90 days | 365 days |
| Config files | On change | 30 days | 365 days |
| TLS certificates | On renewal | 365 days | 365 days |
| NATS JetStream | Every 6 hours | 7 days | 30 days |

### Backup Schedule Example

```
Production backup schedule:

00:00 - Full backup (all components) + remote upload
01:00 - Database WAL archive
02:00 - Database-only backup
03:00 - Database WAL archive
04:00 - Full backup (all components) + prune old backups
05:00 - Database WAL archive
06:00 - Database-only backup
07:00 - Database WAL archive
08:00 - Full backup (all components) + remote upload
...
(Repeating every 2 hours for database, every 6 hours for full)
```

### Cron Schedule Implementation

```bash
# /etc/cron.d/renos-backup-schedule

# Full backup every 6 hours
0 */6 * * * root /usr/local/bin/renos-backup.sh --full --encrypt --remote

# Database-only backup every 2 hours between full backups
0 2,4,8,10,14,16,20,22 * * * root /usr/local/bin/renos-backup.sh --db-only --encrypt --remote

# WAL archiving (continuous, handled by PostgreSQL)

# Configuration backup
0 3 * * * root /usr/local/bin/renos-backup.sh --config --remote

# Prune old backups
30 1 * * * root /usr/local/bin/renos-backup.sh --prune
```

---

## Backup Storage Strategies

### Strategy A: Local Disk Only

| Aspect | Detail |
|--------|--------|
| Storage | `/var/lib/renos/backups/` (local SSD) |
| Retention | 7 days |
| Capacity | 50 GB minimum |
| Encryption | Recommended (age/gpg) |
| Offsite | None |
| Recovery speed | Fast (minutes) |
| Risk | Backups lost if server fails |

```bash
# Configuration
BACKUP_ROOT="/var/lib/renos/backups"
RETENTION_COUNT=7
REMOTE_ENABLED=false
```

### Strategy B: Local + S3

| Aspect | Detail |
|--------|--------|
| Storage | Local (7 days) + S3 (90 days) |
| S3 Bucket | `s3://renos-backups/{environment}/` |
| S3 Region | Same as primary infrastructure |
| Encryption | Server-side (SSE-S3) + client-side (age) |
| Lifecycle | Transition to Glacier after 90 days |
| Recovery speed | Minutes (local) / Hours (S3) |

```bash
# Configuration
BACKUP_ROOT="/var/lib/renos/backups"
RETENTION_COUNT=7
REMOTE_ENABLED=true
REMOTE_TYPE="s3"
S3_BUCKET="renos-backups"
S3_REGION="us-east-1"
RCLONE_REMOTE="renos-s3"
```

**S3 Lifecycle Policy:**

```json
{
  "Rules": [
    {
      "Id": "backup-lifecycle",
      "Status": "Enabled",
      "Filter": {
        "Prefix": "renos-backup-"
      },
      "Transitions": [
        {
          "Days": 90,
          "StorageClass": "GLACIER"
        }
      ],
      "Expiration": {
        "Days": 365
      }
    }
  ]
}
```

**IAM Policy for S3 Backup Access:**

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:PutObject",
        "s3:GetObject",
        "s3:ListBucket",
        "s3:DeleteObject"
      ],
      "Resource": [
        "arn:aws:s3:::renos-backups",
        "arn:aws:s3:::renos-backups/*"
      ]
    }
  ]
}
```

### Strategy C: Network Storage (NFS/NAS)

| Aspect | Detail |
|--------|--------|
| Storage | NFS mount + local cache |
| NFS Export | `nas.example.com:/export/renos-backups` |
| Encryption | Client-side (age) |
| Redundancy | RAID 6 on NAS + offsite replication |
| Recovery speed | Medium (network transfer) |

```bash
# Configuration
BACKUP_ROOT="/mnt/nfs/renos-backups"
RETENTION_COUNT=30
REMOTE_ENABLED=false

# Mount NFS
mkdir -p /mnt/nfs
echo "nas.example.com:/export/renos-backups /mnt/nfs nfs defaults 0 0" >> /etc/fstab
mount -a
```

### Strategy D: Tape Backup

| Aspect | Detail |
|--------|--------|
| Storage | LTO-9 tape library |
| Capacity | 18 TB per tape (compressed) |
| Retention | 1 year (quarterly), 7 years (annual) |
| Software | Bacula / Bareos / Amanda |
| Recovery speed | Slow (hours to days) |
| Use case | Archival compliance |

```bash
# Example: Writing backup to tape via tar
tar -czf /dev/st0 /var/lib/renos/backups/
mt -f /dev/st0 rewind
mt -f /dev/st0 eject
```

### Strategy E: Hybrid (Recommended)

Combines all strategies for maximum protection:

```
                      ┌─────────────────────────────┐
                      │   RenOS Production Server    │
                      │   Local SSD (7 days, fast)   │
                      └──────────────┬──────────────┘
                                     │
                ┌────────────────────┼────────────────────┐
                │                    │                    │
                ▼                    ▼                    ▼
      ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
      │  S3 / Object    │  │  NFS / NAS      │  │  Tape Library   │
      │  Store          │  │  (30 days)      │  │  (1 year+)      │
      │  (90 days)      │  │                 │  │                 │
      │  → Glacier      │  │  → Nightly sync │  │  → Weekly       │
      └─────────────────┘  └─────────────────┘  └─────────────────┘
```

```bash
# Hybrid backup script
#!/bin/bash
# renos-hybrid-backup.sh

# Step 1: Full backup to local SSD
renos-backup.sh --full --encrypt

# Step 2: Sync to NAS
rsync -avz /var/lib/renos/backups/ /mnt/nfs/renos-backups/

# Step 3: Upload to S3 (async)
renos-backup.sh --full --encrypt --remote &

# Step 4: Write to tape (weekly)
if [ "$(date +%u)" -eq 0 ]; then
  tar -czf /dev/st0 /var/lib/renos/backups/
fi
```

---

## Testing Backups

### Automated Verification

```bash
# Verify the most recent backup
renos-backup.sh --verify latest

# Verify a specific backup
renos-backup.sh --verify renos-backup-20250101-020000
```

### What the Verification Checks

The verification process checks:

| Check | Description | Pass/Fail Criteria |
|-------|-------------|-------------------|
| Manifest existence | `backup-info.json` must exist | File present |
| Component completeness | All expected components present | No critical missing components |
| Checksum integrity | SHA256 of each component file | Matches recorded checksum |
| File size sanity | File size > 0 bytes | All files non-empty |
| Database dump integrity | `pg_restore -l` exit code | Exit code 0 |
| Archive integrity | `tar -tzf` exit code | Exit code 0 |

### Database Restore Testing

Periodically test that database backups can actually be restored:

```bash
#!/bin/bash
# test-db-restore.sh

set -euo pipefail

BACKUP_ID="${1:-latest}"
TEST_DB="renos_restore_test_$(date +%s)"

echo "=== Database Restore Test ==="

# Step 1: Locate backup
if [ "${BACKUP_ID}" = "latest" ]; then
  BACKUP_DIR=$(ls -td /var/lib/renos/backups/renos-backup-* | head -1)
else
  BACKUP_DIR="/var/lib/renos/backups/renos-backup-${BACKUP_ID}"
fi

echo "Testing backup: ${BACKUP_DIR}"

# Step 2: Identify dump file
DUMP_FILE="${BACKUP_DIR}/renos-db.dump"
if [ ! -f "${DUMP_FILE}" ]; then
  # Check for encrypted version
  DUMP_FILE="${BACKUP_DIR}/renos-db.dump.age"
  if [ -f "${DUMP_FILE}" ]; then
    age --decrypt -i ~/.config/renos/backup-key.txt -o /tmp/test-restore.dump "${DUMP_FILE}"
    DUMP_FILE="/tmp/test-restore.dump"
  fi
fi

# Step 3: Create test database
echo "Creating test database: ${TEST_DB}"
createdb -U renos "${TEST_DB}"

# Step 4: Restore to test database
echo "Restoring to test database..."
pg_restore --no-owner --no-acl -d "${TEST_DB}" "${DUMP_FILE}" 2>&1

if [ $? -eq 0 ]; then
  echo "✓ Restore successful"
  
  # Check row counts
  TABLES=$(psql -U renos -d "${TEST_DB}" -t -c "SELECT count(*) FROM information_schema.tables WHERE table_schema='public';")
  echo "  Tables restored: ${TABLES}"
  
  ROWS=$(psql -U renos -d "${TEST_DB}" -t -c "
    SELECT SUM(n_live_tup) FROM pg_stat_user_tables;
  " 2>/dev/null || echo "N/A")
  echo "  Total rows: ${ROWS}"
else
  echo "✗ Restore FAILED"
fi

# Step 5: Clean up
echo "Cleaning up..."
dropdb -U renos "${TEST_DB}" 2>/dev/null || true
rm -f /tmp/test-restore.dump

echo "=== Test Complete ==="
```

### Full Recovery Drill Script

```bash
#!/bin/bash
# dr-drill.sh

set -euo pipefail

DRILL_ID="drill-$(date +%Y%m%d-%H%M%S)"
DRILL_DIR="/tmp/${DRILL_ID}"
LOG_FILE="${DRILL_DIR}/drill.log"

mkdir -p "${DRILL_DIR}"

echo "=== Disaster Recovery Drill: ${DRILL_ID} ===" | tee -a "${LOG_FILE}"
echo "Started: $(date)" | tee -a "${LOG_FILE}"
echo ""

# Phase 1: Backup verification
echo "PHASE 1: Verify backup integrity" | tee -a "${LOG_FILE}"
renos-backup.sh --verify latest >> "${LOG_FILE}" 2>&1
if [ $? -eq 0 ]; then
  echo "  ✓ Backup verification passed" | tee -a "${LOG_FILE}"
else
  echo "  ✗ Backup verification FAILED" | tee -a "${LOG_FILE}"
  exit 1
fi

# Phase 2: Database restore test
echo ""
echo "PHASE 2: Test database restore" | tee -a "${LOG_FILE}"
./test-db-restore.sh >> "${LOG_FILE}" 2>&1
echo "  ✓ Database restore test completed" | tee -a "${LOG_FILE}"

# Phase 3: Service restart test
echo ""
echo "PHASE 3: Service restart" | tee -a "${LOG_FILE}"
systemctl restart renos-core renos-hub renos-agent renos-nginx
sleep 10
for svc in renos-core renos-hub renos-agent renos-nginx; do
  if systemctl is-active "${svc}" > /dev/null 2>&1; then
    echo "  ✓ ${svc} restarted successfully" | tee -a "${LOG_FILE}"
  else
    echo "  ✗ ${svc} failed to restart" | tee -a "${LOG_FILE}"
  fi
done

# Phase 4: API health check
echo ""
echo "PHASE 4: API health check" | tee -a "${LOG_FILE}"
HEALTH=$(curl -sk -o /dev/null -w "%{http_code}" https://localhost/api/v1/healthz 2>/dev/null || echo "000")
if [ "${HEALTH}" = "200" ]; then
  echo "  ✓ API health check passed (${HEALTH})" | tee -a "${LOG_FILE}"
else
  echo "  ✗ API health check FAILED (${HEALTH})" | tee -a "${LOG_FILE}"
fi

# Phase 5: Data integrity
echo ""
echo "PHASE 5: Data integrity check" | tee -a "${LOG_FILE}"
docker compose exec postgres psql -U renos -d renos \
  -c "VACUUM VERBOSE ANALYZE;" >> "${LOG_FILE}" 2>&1
echo "  ✓ Data integrity check completed" | tee -a "${LOG_FILE}"

# Phase 6: Backup schedule verification
echo ""
echo "PHASE 6: Backup schedule verification" | tee -a "${LOG_FILE}"
renos-backup.sh --list >> "${LOG_FILE}" 2>&1
echo "  ✓ Backup schedule verified" | tee -a "${LOG_FILE}"

echo ""
echo "=== Drill Complete: ${DRILL_ID} ===" | tee -a "${LOG_FILE}"
echo "Finished: $(date)" | tee -a "${LOG_FILE}"
echo "Log: ${LOG_FILE}"
```

---

## DR Drill Procedures

### Quarterly DR Drill Schedule

| Quarter | Drill Type | Scope | Duration |
|---------|------------|-------|----------|
| Q1 | Tabletop | Review procedures, update documentation | 2 hours |
| Q2 | Partial recovery | Restore to staging environment | 4 hours |
| Q3 | Full recovery | Full failover to secondary site | 8 hours |
| Q4 | Surprise drill | Unannounced partial failure simulation | 4 hours |

### DR Drill Scenarios

**Scenario A: Database Corruption**

1. Simulate database corruption by injecting bad data
2. Detect corruption via monitoring
3. Declare DR event
4. Restore database from most recent backup
5. Verify data integrity
6. Restart services
7. Declare DR event resolved
8. Document lessons learned

**Scenario B: Complete Node Failure**

1. Simulate node failure (power off primary node)
2. Detect failure via monitoring
3. Verify failover to secondary node
4. Rebuild primary node
5. Restore replication
6. Fail back to primary (if desired)
7. Document lessons learned

**Scenario C: Security Breach**

1. Simulate compromised credentials
2. Declare DR event
3. Isolate affected systems
4. Rotate all secrets and certificates
5. Restore from pre-compromise backup
6. Verify system integrity
7. Document lessons learned

### Drill Success Criteria

| Metric | Target | Acceptable |
|--------|--------|------------|
| RTO for database corruption | < 30 min | < 60 min |
| RTO for node failure | < 5 min | < 15 min |
| RTO for full site failure | < 4 hours | < 8 hours |
| Data loss (RPO) | < 1 hour | < 4 hours |
| Successful data restoration | 100% | > 95% |
| Service availability after recovery | 100% | > 99% |

### Drill Documentation Template

```markdown
# DR Drill Report

## Drill Information
- **Date**: __________
- **Drill ID**: __________
- **Scenario**: __________
- **Participants**: __________
- **Duration**: __________

## Pre-Drill State
- System version: __________
- Backup timestamp: __________
- Data volume: __________

## Execution Timeline
| Time | Action | Duration | Notes |
|------|--------|----------|-------|
| T+0 | Declare DR event | | |
| T+5 | Verify backup integrity | | |
| T+15 | Start restore process | | |
| T+45 | Database restored | | |
| T+60 | Services started | | |
| T+75 | Verification complete | | |
| T+90 | Declare resolved | | |

## Results
- RTO achieved: __________
- RPO achieved: __________
- Data integrity: Pass / Fail
- Service availability: __________%

## Issues Encountered
1. __________
2. __________
3. __________

## Lessons Learned
1. __________
2. __________
3. __________

## Action Items
| Priority | Action | Owner | Due Date |
|----------|--------|-------|----------|
| P0 | | | |
| P1 | | | |
| P2 | | | |
```

---

## Failover Plan

### Automated Failover (Cluster)

```
Normal Operation:
  ┌──────────┐     ┌──────────┐     ┌──────────┐
  │ Leader   │◄───►│ Follower │◄───►│ Follower │
  │ (Active) │     │ (Standby)│     │ (Standby)│
  └──────────┘     └──────────┘     └──────────┘

Failure Detected (Node 1):
  ┌──────────┐     ┌──────────┐     ┌──────────┐
  │ Node 1   │     │ Follower │◄───►│ Follower │
  │ (DOWN)   │     │→ Leader  │     │ (Standby)│
  └──────────┘     └──────────┘     └──────────┘

Recovery:
  ┌──────────┐     ┌──────────┐     ┌──────────┐
  │ Node 1   │◄───►│ Leader   │◄───►│ Follower │
  │ (Standby)│     │ (Active) │     │ (Standby)│
  └──────────┘     └──────────┘     └──────────┘
```

### Manual Failover Procedure

**Step 1: Detect and confirm failure**

```bash
# Check system status
renos-health-check.sh

# Verify with multiple sources
curl -sk https://node1/api/v1/healthz
curl -sk https://node2/api/v1/healthz
ping -c 3 node1

# Check cluster status
docker compose exec nats nats server check cluster
docker compose exec postgres psql -U renos -c "SELECT * FROM pg_stat_replication;"
```

**Step 2: Declare DR event**

```bash
# Record the event
echo "[$(date)] DR EVENT DECLARED: Node1 failure" >> /var/log/renos/dr-events.log

# Notify on-call team (if integrated)
renos-notify --channel on-call \
  --message "DR EVENT: Node1 failure detected. Initiating failover."
```

**Step 3: Promote follower to leader**

```bash
# If using Patroni:
patronictl -c /etc/patroni.yml failover renos-db --master node1 --candidate node2

# If manual PostgreSQL failover:
ssh node2 "docker compose exec postgres psql -U renos -c 'SELECT pg_promote();'"

# Verify new leader
ssh node2 "docker compose exec postgres psql -U renos -c 'SELECT pg_is_in_recovery();'"
# Expected: f (false = not in recovery = leader)
```

**Step 4: Update load balancer**

```bash
# Update HAProxy to remove failed node
ssh load-balancer "sed -i '/node1/s/server/\\#server/' /etc/haproxy/haproxy.cfg"
ssh load-balancer "systemctl reload haproxy"
```

**Step 5: Redirect NATS clients**

```bash
# NATS auto-discovers cluster changes, but verify:
ssh node2 "docker compose exec nats nats server check cluster"
```

**Step 6: Verify service health**

```bash
# Check all services on surviving nodes
ssh node2 "renos-health-check.sh"
ssh node3 "renos-health-check.sh"

# Verify API
curl -sk https://node2/api/v1/healthz
curl -sk https://load-balancer/api/v1/healthz
```

### Failback Procedure

After the failed node is repaired:

```bash
# Step 1: Rejoin node to cluster
ssh node1 "docker compose up -d"

# Step 2: Re-establish database replication
ssh node1 "docker compose exec postgres pg_basebackup \
  -h node2 -D /var/lib/postgresql/data -U replicator -P -v --wal-method=stream"

# Step 3: Restart PostgreSQL
ssh node1 "docker compose restart postgres"

# Step 4: Verify replication
ssh node2 "docker compose exec postgres psql -U renos -c \"SELECT * FROM pg_stat_replication;\""

# Step 5: Add node back to load balancer
ssh load-balancer "sed -i '/#server node1/s/\\#//' /etc/haproxy/haproxy.cfg"
ssh load-balancer "systemctl reload haproxy"

# Step 6: Optionally failback to original leader
patronictl -c /etc/patroni.yml switchover renos-db --master node2 --candidate node1
```

---

## Restoration Time Estimates

### Estimated Times by Scenario

| Scenario | Preparation | Data Restore | Verification | Total |
|----------|-------------|-------------|--------------|-------|
| **Service crash** (restart) | 0 min | 0 min | 2 min | **2 min** |
| **Container restart** | 0 min | 0 min | 3 min | **3 min** |
| **Database corruption** (local backup) | 2 min | 10 min | 5 min | **17 min** |
| **Database corruption** (S3 backup) | 5 min | 30 min | 5 min | **40 min** |
| **Failed upgrade** (rollback) | 2 min | 5 min | 5 min | **12 min** |
| **Single node rebuild** (local backup) | 30 min | 20 min | 10 min | **60 min** |
| **Single node rebuild** (S3 backup) | 30 min | 60 min | 10 min | **100 min** |
| **Full site recovery** (local backup) | 60 min | 60 min | 30 min | **150 min** |
| **Full site recovery** (S3 backup) | 60 min | 120 min | 30 min | **210 min** |
| **Tape restore** | 120 min | 240 min | 60 min | **420 min** |

### Factors Affecting Restoration Time

| Factor | Impact | Mitigation |
|--------|--------|------------|
| Dataset size | Linear | Use incremental backups for large datasets |
| Backup storage speed | 10x difference SSD vs HDD | Use SSD for local backups |
| Network bandwidth | 3x difference 1Gbps vs 10Gbps | Ensure high-bandwidth infrastructure |
| Number of nodes | Fixed overhead per node | Parallelize restoration across nodes |
| Encryption overhead | +20% time | Use faster encryption (age > gpg) |
| Database size | Linear | Use pg_dump custom format for faster restore |

### Restoration Time Calculator

```bash
#!/bin/bash
# estimate-restore-time.sh

DB_SIZE_GB="${1:-10}"    # Database size in GB
STORAGE_MBPS="${2:-200}"  # Storage throughput in MB/s
NETWORK_MBPS="${3:-125}"  # Network throughput in MB/s (1Gbps = ~125MB/s)

echo "=== Restore Time Estimate ==="
echo "Database size: ${DB_SIZE_GB} GB"
echo "Storage speed: ${STORAGE_MBPS} MB/s"
echo "Network speed: ${NETWORK_MBPS} MB/s"
echo ""

# Calculate times
DB_RESTORE=$(( DB_SIZE_GB * 1024 / STORAGE_MBPS / 60 ))
S3_DOWNLOAD=$(( DB_SIZE_GB * 1024 / NETWORK_MBPS / 60 ))
VERIFY_TIME=5
SERVICE_START=3

echo "Local restore estimate:"
echo "  Database restore:       ${DB_RESTORE} min"
echo "  Verification:           ${VERIFY_TIME} min"
echo "  Service start:          ${SERVICE_START} min"
echo "  TOTAL (local):          $(( DB_RESTORE + VERIFY_TIME + SERVICE_START )) min"
echo ""

echo "S3 restore estimate:"
echo "  Download from S3:       ${S3_DOWNLOAD} min"
echo "  Database restore:       ${DB_RESTORE} min"
echo "  Verification:           ${VERIFY_TIME} min"
echo "  Service start:          ${SERVICE_START} min"
echo "  TOTAL (S3):             $(( S3_DOWNLOAD + DB_RESTORE + VERIFY_TIME + SERVICE_START )) min"
```

---

## Communication Plan During Incidents

### Incident Severity Levels

| Level | Description | Examples | Response Time |
|-------|-------------|----------|--------------|
| **SEV1** | Complete service outage | Full site down, data loss | 15 minutes |
| **SEV2** | Major feature degradation | Devices unable to connect, dashboard errors | 30 minutes |
| **SEV3** | Minor feature degradation | Slow performance, non-critical errors | 2 hours |
| **SEV4** | Cosmetic / informational | UI glitches, typographical errors | Next business day |

### Notification Flow

```
DR Event Detected
      │
      ▼
┌─────────────────┐     ┌─────────────────┐
│  SEV1/SEV2?     │────►│  PagerDuty/Ops  │
│                 │ Yes │  Genie Alert    │
└────────┬────────┘     └────────┬────────┘
         │ No                    │
         ▼                       ▼
┌─────────────────┐     ┌─────────────────┐
│  Log incident   │     │  Notify On-Call │
│  in tracker     │     │  Engineer       │
└─────────────────┘     └─────────────────┘
                                │
                                ▼
                     ┌─────────────────────┐
                     │  Initial Assessment │
                     │  (< 15 minutes)     │
                     └──────────┬──────────┘
                                │
                     ┌──────────┴──────────┐
                     │                     │
                     ▼                     ▼
              ┌─────────────────┐  ┌─────────────────┐
              │  Service        │  │  Data Loss /    │
              │  Degradation    │  │  Corruption     │
              └─────────────────┘  └─────────────────┘
                     │                     │
                     ▼                     ▼
              ┌─────────────────┐  ┌─────────────────┐
              │  Status Page    │  │  Declare DR     │
              │  Update         │  │  Event          │
              └─────────────────┘  └────────┬────────┘
                                            │
                                            ▼
                                  ┌─────────────────┐
                                  │  Execute        │
                                  │  Recovery Plan  │
                                  └────────┬────────┘
                                           │
                                           ▼
                                  ┌─────────────────┐
                                  │  Status Updates │
                                  │  Every 30 min   │
                                  └─────────────────┘
```

### Communication Templates

**Initial DR Declaration:**

```
Subject: [SEV1] RenOS DR Event Declared - <SITE/TIME>

Alert Type: DR Event
Severity: SEV1
Status: INVESTIGATING
Timestamp: <ISO 8601>
Affected System: <Site/Cluster>
Impact: <Description of impact>

Current Actions:
1. On-call engineer notified
2. Initial assessment in progress
3. <Next action>

Next Update: <Time + 30 minutes>
Incident ID: <ID>
```

**Status Update:**

```
Subject: [SEV1] RenOS DR Event - Update <N>

Status: <INVESTIGATING / MITIGATING / RESOLVED>
Elapsed Time: <Duration>
Recovery Actions Completed:
1. <Action>
2. <Action>

Remaining Actions:
1. <Action>
2. <Action>

Estimated Resolution: <Time>
```

**Resolution:**

```
Subject: [SEV1] RenOS DR Event - RESOLVED <SITE/TIME>

Status: RESOLVED
Root Cause: <Brief description>
Resolution: <Brief description>
Recovery Time: <Duration>
Data Loss: <If any>

Post-Incident Review Scheduled: <Date/Time>
```

### Communication Channels

| Channel | Use | Audience |
|---------|-----|----------|
| PagerDuty/OpsGenie | Critical alerts | On-call engineers |
| Slack #renos-alerts | Real-time updates | Engineering team |
| Email distribution | Status reports | All stakeholders |
| Status page | Public status | End users / Customers |
| Incident management tool | Tracking | Incident response team |

---

## Post-Incident Review Checklist

### Immediate (Within 24 Hours)

- [ ] **Stabilize system** — Ensure all services are healthy
- [ ] **Verify backups** — Confirm backup system is operational
- [ ] **Alert monitoring** — Verify all alerts are functioning
- [ ] **Document timeline** — Record exact times of all events
- [ ] **Preserve evidence** — Save logs, metrics, and snapshots
- [ ] **Notify affected users** — Send post-incident communication

### Incident Analysis (Within 72 Hours)

- [ ] **Identify root cause** — Determine what caused the incident
- [ ] **Determine contributing factors** — What enabled the failure
- [ ] **Review monitoring effectiveness** — Did alerts fire correctly?
- [ ] **Review response effectiveness** — Was the response timely and correct?
- [ ] **Assess data loss** — Quantify any data loss against RPO
- [ ] **Assess downtime** — Quantify downtime against RTO

### Remediation Planning (Within 1 Week)

- [ ] **Prioritize action items** — Based on severity and impact
- [ ] **Assign owners** — Each action item has a responsible person
- [ ] **Set deadlines** — Realistic due dates for each action
- [ ] **Plan improvements** — Monitoring, automation, documentation
- [ ] **Update runbooks** — Incorporate lessons learned

### Post-Incident Review Meeting

**Agenda:**

1. **Incident overview** (5 min)
2. **Timeline review** (10 min)
3. **What went well** (10 min)
4. **What went wrong** (15 min)
5. **What can be improved** (15 min)
6. **Action items and owners** (10 min)
7. **Next steps** (5 min)

### Post-Incident Review Report Template

```markdown
# Post-Incident Review Report

## Incident Summary
- **Incident ID**: __________
- **Date**: __________
- **Duration**: __________
- **Severity**: SEV__
- **Impact**: __________

## Timeline
| Time (UTC) | Event |
|------------|-------|
| 00:00 | Incident detected |
| 00:05 | On-call notified |
| 00:15 | DR event declared |
| 00:30 | Recovery started |
| 01:00 | Services restored |
| 01:15 | Verification complete |
| 01:30 | Incident resolved |

## Root Cause Analysis
**Primary Cause**:
__________

**Contributing Factors**:
1. __________
2. __________
3. __________

## Impact Assessment
- **Affected users**: __________
- **Downtime**: __________ minutes
- **Data loss**: __________ (RPO: __ / Actual: __)
- **Services affected**: __________

## What Went Well
1. __________
2. __________
3. __________

## What Went Wrong
1. __________
2. __________
3. __________

## Improvements
| Priority | Action | Owner | Due Date | Status |
|----------|--------|-------|----------|--------|
| P0 | | | | Open |
| P1 | | | | Open |
| P2 | | | | Open |

## Metrics Review
| Metric | Target | Actual | Met? |
|--------|--------|--------|------|
| RTO | __ min | __ min | Yes/No |
| RPO | __ min | __ min | Yes/No |
| Alert time | __ min | __ min | Yes/No |
| Response time | __ min | __ min | Yes/No |

## Closing
**Review Date**: __________
**Reviewers**: __________
**Follow-up Review Date**: __________
```
