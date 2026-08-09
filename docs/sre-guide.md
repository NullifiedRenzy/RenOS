# RenOS SRE Guide

## Overview

This document describes the Site Reliability Engineering (SRE) practices for RenOS.
It covers SLO definitions, runbooks for common incidents, and operational procedures.

## Table of Contents

1. [Service Level Objectives (SLOs)](#slos)
2. [Runbooks](#runbooks)
3. [On-Call Procedures](#on-call)
4. [Incident Response](#incident-response)
5. [Postmortem Process](#postmortem)

## SLOs

| SLO | Target | Window | Measurement |
|-----|--------|--------|-------------|
| API Availability | 99.5% | 30 days | /api/v1/healthz returning 200 |
| API P99 Latency | 500ms | 30 days | HTTP request duration |
| DB Query P99 Latency | 100ms | 30 days | Database query duration |
| Agent Heartbeat Reliability | 99% | 24 hours | Heartbeat receipt within interval |
| NATS Delivery Reliability | 99.9% | 30 days | NATS publish/subscribe success |
| AI Response P99 Latency | 30s | 24 hours | AI provider response time |

### Viewing SLOs

```bash
curl https://renos.example.com/api/v1/slos
```

### Key Metrics

```bash
# All SLO availability
curl https://renos.example.com/metrics | grep renos_slo

# Specific SLO
curl https://renos.example.com/metrics | grep 'renos_slo_availability{slo_name="api_availability"}'
```

## Runbooks

### Runbook: High HTTP Error Rate

**Symptoms:** PagerDuty alert "HighHTTPErrorRate"

**Check:**
```bash
# View recent error rates
curl https://renos.example.com/api/v1/stats/connections

# Check logs
journalctl -u renos-core --since "5 minutes ago" | grep "status=5"
```

**Steps:**
1. Check if this is a deployment issue (`/api/v1/version`)
2. Review recent configuration changes
3. Check database connectivity (`/api/v1/readyz`)
4. Check NATS connectivity
5. Rollback if recent deployment

### Runbook: Agents Mass Offline

**Symptoms:** ">50% agents offline" alert

**Check:**
```bash
curl https://renos.example.com/api/v1/stats | jq
```

**Steps:**
1. Check NATS connection — agents communicate via NATS
2. Verify NATS cluster health
3. Check if TLS certificates expired
4. Restart NATS if needed (agents will reconnect automatically via reconnect handlers)
5. Verify core is accepting heartbeats (`/api/v1/healthz`)

### Runbook: Database Latency Spike

**Symptoms:** "DB query P99 >1s" alert

**Check:**
```bash
curl https://renos.example.com/api/v1/deephealth
```

**Steps:**
1. Check connection pool saturation
2. Review slow queries in PostgreSQL logs
3. Check for missing indexes (run `ANALYZE`)
4. Consider increasing pool size or query optimization
5. If urgent, restart core to reset connection pool

### Runbook: NATS Disconnection

**Symptoms:** "NATS disconnection detected" alert

**Steps:**
1. NATS automatically reconnects (reconnect handlers configured)
2. Verify NATS server is running
3. Check network between core and NATS
4. Review core logs for NATS errors

### Runbook: SLO Error Budget Depleted

**Symptoms:** "SLO error budget <10%" alert

**Steps:**
1. Identify which SLO is impacted via `/api/v1/slos`
2. Review dashboards for the relevant subsystem
3. Determine if the budget burn is sustained or a spike
4. If sustained, consider pausing non-critical deployments
5. File a postmortem if budget is fully exhausted

## On-Call Procedures

### Rotation
- Primary on-call: 24/7 coverage
- Escalation: Primary -> Secondary -> Engineering Lead

### Handoff
1. Review active alerts and incidents
2. Check SLO dashboard
3. Note any ongoing deployments or maintenance
4. Transfer runbooks and context

### escalation
- If unresolved in 30 minutes: escalate to secondary
- If critical and unresolved in 1 hour: escalate to Engineering Lead
- If SLO violation projected: declare incident

## Incident Response

### Triage (0-5 min)
1. Acknowledge alert
2. Determine severity (SEV1/SEV2/SEV3)
3. Declare incident if SEV1

### Mitigation (5-30 min)
1. Apply known runbook steps
2. If no runbook: stabilize system (rollback, scale up, restart)
3. Communicate status in incident channel

### Resolution (30 min - ongoing)
1. Apply permanent fix
2. Verify monitoring shows recovery
3. Resolve alert
4. File postmortem within 48 hours

## Postmortem Process

1. Create postmortem document within 48 hours
2. Include: timeline, impact, root cause, detection, response
3. List action items with owners and deadlines
4. Review in postmortem meeting
5. Track action items to completion

### Postmortem Template

```markdown
# Postmortem: [Title]

Date: YYYY-MM-DD
Severity: SEV1/SEV2/SEV3
Duration: Xh Ym
SLO Impact: Yes/No

## Summary
One-paragraph description.

## Timeline
- HH:MM - Event started
- HH:MM - Alert fired
- HH:MM - Investigation began
- HH:MM - Mitigation applied
- HH:MM - System healthy

## Root Cause
Description of root cause.

## Detection
How was this detected? Alert? User report?

## Response
What went well? What could be improved?

## Action Items
- [ ] Item 1 (Owner, Due date)
- [ ] Item 2 (Owner, Due date)
```
