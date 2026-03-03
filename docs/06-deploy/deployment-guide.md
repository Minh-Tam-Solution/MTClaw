---
title: MTClaw Deployment Guide
version: 1.0.0
sdlc_stage: "06-deploy"
sdlc_version: "6.1.1"
status: active
created: 2026-03-03
updated: 2026-03-03
owner: "@devops"
---

# MTClaw Deployment Guide

## Overview

MTClaw runs as a Docker Compose stack with 3 overlay files layered on a base definition. The MTS deployment connects to Bflow AI-Platform via Docker's `ai-net` network and exposes a Telegram bot (`@mtsclawbot`) as the primary channel.

**Architecture:**

```
┌──────────────────────────────────────────────────┐
│  Docker Host (192.168.2.2)                       │
│                                                  │
│  ┌────────────────┐    ┌──────────────────────┐  │
│  │  mtclaw-mtclaw  │───▶│  mtclaw-postgres     │  │
│  │  :18790 (GW)   │    │  :5470 → 5432 (PG)   │  │
│  └──────┬─────────┘    └──────────────────────┘  │
│         │                                        │
│         │ ai-net (Docker network)                │
│         ▼                                        │
│  ┌────────────────────┐                          │
│  │  ai-platform:8120  │  (Bflow AI-Platform)     │
│  │  qwen3:14b         │                          │
│  └────────────────────┘                          │
│                                                  │
│         │ Internet                               │
│         ▼                                        │
│  ┌────────────────────┐                          │
│  │  Telegram API      │  (polling mode)          │
│  │  @mtsclawbot       │                          │
│  └────────────────────┘                          │
└──────────────────────────────────────────────────┘
```

## Prerequisites

1. **Docker Engine** 24+ with Compose V2
2. **Docker network `ai-net`** must exist (shared with Bflow AI-Platform)
3. **Bflow AI-Platform** running on port 8120 (same host)
4. **Telegram bot token** from BotFather
5. **Bflow API key** with `aip_` prefix (provisioned by CTO)

Verify prerequisites:

```bash
docker network inspect ai-net > /dev/null 2>&1 && echo "ai-net OK" || echo "MISSING: docker network create ai-net"
curl -sf http://ai-platform:8120/health && echo "AI-Platform OK" || echo "MISSING: AI-Platform not reachable"
```

## Docker Compose Overlays

MTClaw uses a 3-overlay pattern:

| File | Purpose | Required |
|------|---------|----------|
| `docker-compose.yml` | Base: mtclaw service, ports, security, resources | Always |
| `docker-compose.managed.yml` | PostgreSQL (pgvector/pg18), managed mode, DSN | Always (MTS) |
| `docker-compose.mts.yml` | Bflow AI-Platform, Telegram, ai-net, cost guardrails | MTS deployment |

**Compose command (MTS):**

```bash
COMPOSE="docker compose -f docker-compose.yml -f docker-compose.managed.yml -f docker-compose.mts.yml"
```

The `docker-compose.selfservice.yml` overlay (Web Dashboard UI) is available but not used in MTS Sprint 5 deployment.

## Deployment Steps

### 1. Clone and configure

```bash
cd /home/nqh/shared/MTClaw

# Copy env template (see docs/06-deploy/env-template.md for details)
cp .env.example .env
# Edit .env with real credentials
```

### 2. Verify ai-net network

```bash
docker network create ai-net 2>/dev/null || true
```

### 3. Build and start

```bash
# Full build + start (detached)
docker compose -f docker-compose.yml \
  -f docker-compose.managed.yml \
  -f docker-compose.mts.yml \
  up -d --build
```

### 4. Verify deployment

```bash
# Check containers are healthy
docker compose -f docker-compose.yml \
  -f docker-compose.managed.yml \
  -f docker-compose.mts.yml \
  ps

# Health check
curl -sf http://localhost:18790/health
# Expected: {"status":"ok","protocol":3}

# Check logs for successful auto-onboard
docker compose -f docker-compose.yml \
  -f docker-compose.managed.yml \
  -f docker-compose.mts.yml \
  logs mtclaw | tail -30
# Look for: "Auto-onboard complete", "Telegram polling started"
```

### 5. Verify Telegram bot

Send `/start` to `@mtsclawbot` on Telegram. Expected: welcome message with bot info.

## Startup Flow (Auto-Onboard)

On first start with managed mode, the entrypoint runs:

```
docker-entrypoint.sh serve
  → mtclaw upgrade (schema migrations: 12 files)
  → mtclaw (main binary)
    → Detects GOCLAW_BFLOW_API_KEY in env
    → canAutoOnboard() = true (bflow-ai-platform in providerPriority)
    → Creates /app/data/config.json
    → Seeds database (agents, SOULs, teams)
    → Starts gateway on :18790
    → Starts Telegram polling
```

The auto-generated `config.json` contains:

```json
{
  "agents": {
    "defaults": {
      "model": "qwen3:14b",
      "provider": "bflow-ai-platform",
      "workspace": "/app/workspace"
    }
  },
  "database": { "mode": "managed" },
  "gateway": { "host": "0.0.0.0", "port": 18790 }
}
```

## Services

### mtclaw (Gateway)

| Property | Value |
|----------|-------|
| Image | Built from `Dockerfile` (multi-stage: Go 1.25 → Alpine 3.22) |
| Internal port | 18790 |
| Host port | `${GOCLAW_PORT:-18790}` |
| User | `mtclaw` (UID 1000, non-root) |
| Memory limit | 1 GB |
| CPU limit | 2.0 cores |
| PID limit | 200 |
| Security | `no-new-privileges`, `cap_drop: ALL`, `read_only: true` |
| Healthcheck | `wget -qO- http://localhost:18790/health` (30s interval) |
| Restart | `unless-stopped` |

### postgres (Database)

| Property | Value |
|----------|-------|
| Image | `pgvector/pgvector:pg18` |
| Internal port | 5432 |
| Host port | `${POSTGRES_PORT:-5432}` → **5470** (MTS allocation) |
| Data volume | `postgres-data:/var/lib/postgresql` |
| Healthcheck | `pg_isready` (5s interval, 10 retries) |
| Restart | `unless-stopped` |

## Volumes

| Volume | Mount | Purpose |
|--------|-------|---------|
| `mtclaw-data` | `/app/data` | config.json, runtime state |
| `mtclaw-workspace` | `/app/workspace` | Agent workspaces |
| `mtclaw-skills` | `/app/skills` | Custom skills |
| `postgres-data` | `/var/lib/postgresql` | Database files |

## Schema Migrations

12 migration files in `/app/migrations/` (copied from `migrations/` at build):

| Migration | Description |
|-----------|-------------|
| 000001 | Init schema (core tables) |
| 000002 | Agent links |
| 000003 | Agent teams |
| 000004 | Teams v2 |
| 000005 | Phase 4 |
| 000006 | Built-in tools |
| 000007 | Team metadata |
| 000008 | RLS tenant isolation |
| 000009 | Seed MTClaw SOULs |
| 000010 | Observability columns |
| 000011 | Seed Bflow provider |
| 000012 | Seed IT Admin SOUL |

`RequiredSchemaVersion` in `internal/upgrade/version.go` must match the total migration count (currently 12).

## Port Allocation

| Port | Service | Status |
|------|---------|--------|
| 18790 | MTClaw Gateway (HTTP API + WebSocket) | Pending IT Admin approval |
| 5470 | PostgreSQL (host-side, internal 5432) | Pending IT Admin approval |

Reference: `/home/nqh/shared/models/core/docs/admin/PORT_ALLOCATION_MANAGEMENT.md`

## Cost Guardrails

The MTS overlay sets default cost limits:

| Guardrail | Default | Env Var |
|-----------|---------|---------|
| Monthly token limit | 1,000,000 | `TENANT_MONTHLY_TOKEN_LIMIT` |
| Daily request limit | 5,000 | `TENANT_DAILY_REQUEST_LIMIT` |

## Security Notes

- Container runs as non-root user `mtclaw` (UID 1000)
- Filesystem is read-only (`read_only: true`) with `/tmp` as tmpfs
- All Linux capabilities dropped (`cap_drop: ALL`)
- No privilege escalation (`no-new-privileges`)
- Telegram uses polling mode (no inbound webhook exposure)
- AI-Platform accessed via internal Docker network only (`ai-net`)
- Encryption key: AES-256 generated via `openssl rand -hex 32`
