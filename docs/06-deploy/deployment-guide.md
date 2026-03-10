---
title: MTClaw Deployment Guide
version: 1.1.0
sdlc_stage: "06-deploy"
sdlc_version: "6.1.1"
status: active
created: 2026-03-03
updated: 2026-03-10
owner: "@devops"
---

# MTClaw Deployment Guide

## Overview

MTClaw runs as a Docker Compose stack with 3 overlay files layered on a base definition. The MTS deployment connects to Bflow AI-Platform via Docker's `ai-net` external network and exposes a Telegram bot (`@mtsclawbot`) as the primary channel. Discord is available as an additional channel (Sprint 30 — ADR-006-Amendment) for Vietnamese dev team accessibility.

**Architecture:**

```
┌──────────────────────────────────────────────────────────┐
│  Docker Host (192.168.2.2)                               │
│                                                          │
│  ┌────────────────┐    ┌──────────────────────┐          │
│  │  mtclaw-mtclaw  │───▶│  mtclaw-postgres     │          │
│  │  :18790 (GW)   │    │  :5470 → 5432 (PG)   │          │
│  │  :18792 (Hook) │    └──────────────────────┘          │
│  └──────┬─────────┘                                      │
│         │                                                │
│         │ ai-net (Docker external network)               │
│         ▼                                                │
│  ┌─────────────────────────────┐                         │
│  │  bflow-ai-gateway-staging   │  (Bflow AI-Platform)    │
│  │  ai-net alias: ai-platform  │                         │
│  │  :8120 (HTTP API)           │                         │
│  │  qwen3:14b (via Ollama)     │                         │
│  └─────────────────────────────┘                         │
│                                                          │
│         │ Internet                                       │
│         ▼                                                │
│  ┌────────────────────┐                                  │
│  │  Telegram API      │  (polling mode)                  │
│  │  @mtsclawbot       │                                  │
│  └────────────────────┘                                  │
└──────────────────────────────────────────────────────────┘
```

### ai-net Network

`ai-net` is a Docker external network shared between MTClaw and Bflow AI-Platform. It allows container-to-container communication using Docker hostnames.

```bash
# Create the network (one-time, if not already present)
docker network create ai-net 2>/dev/null || true
```

MTClaw resolves AI-Platform via hostname `ai-platform` (container name on `ai-net`). The actual container name is `bflow-ai-gateway-staging` — Docker aliases or the service name on `ai-net` provide the `ai-platform` hostname.

When running MTClaw on the **host** (not in Docker), use `localhost:8120` instead since `ai-platform` hostname is only resolvable inside Docker:

```bash
# Host mode override
export MTCLAW_BFLOW_BASE_URL=http://localhost:8120/api/v1
export MTCLAW_POSTGRES_DSN=postgres://mtclaw:PASSWORD@localhost:5470/mtclaw?sslmode=disable
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
| `docker-compose.mts.yml` | Bflow AI-Platform, Telegram, ai-net network, cost guardrails | MTS deployment |

**Compose command (MTS):**

```bash
COMPOSE="docker compose -f docker-compose.yml -f docker-compose.managed.yml -f docker-compose.mts.yml -f docker-compose.selfservice.yml"
```

The `docker-compose.selfservice.yml` overlay adds the Web Dashboard UI (React SPA on port `${MTCLAW_UI_PORT:-18791}`).

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
  -f docker-compose.selfservice.yml \
  up -d --build
```

### 4. Verify deployment

```bash
# Check containers are healthy
docker compose -f docker-compose.yml \
  -f docker-compose.managed.yml \
  -f docker-compose.mts.yml \
  -f docker-compose.selfservice.yml \
  ps

# Health check
curl -sf http://localhost:18790/health
# Expected: {"status":"ok","protocol":3}

# Check logs for successful auto-onboard
docker compose -f docker-compose.yml \
  -f docker-compose.managed.yml \
  -f docker-compose.mts.yml \
  -f docker-compose.selfservice.yml \
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
    → Detects MTCLAW_BFLOW_API_KEY in env
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
| Internal port | 18790 (gateway), 18792 (bridge hook) |
| Host port | `${MTCLAW_PORT:-18790}`, `${MTCLAW_BRIDGE_HOOK_PORT:-18792}` |
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

19 migration files in `/app/migrations/` (copied from `migrations/` at build):

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
| 000013 | Governance specs |
| 000014 | SOUL drift detection |
| 000015 | PR gate evaluations |
| 000016 | Add channel to governance tables |
| 000017 | Evidence links |
| 000018 | Claude Code bridge |
| 000019 | Rename dev to enghelp |

`RequiredSchemaVersion` in `internal/upgrade/version.go` must match the total migration count (currently 19).

## Port Allocation

| Port | Service | Status |
|------|---------|--------|
| 18790 | MTClaw Gateway (HTTP API + WebSocket) | Allocated |
| 18791 | MTClaw Web Dashboard (nginx → React SPA) | Allocated |
| 18792 | Claude Code Bridge Hook Server | Allocated |
| 5470 | PostgreSQL (host-side, internal 5432) | Allocated |

Reference: `/home/nqh/shared/models/core/docs/admin/PORT_ALLOCATION_MANAGEMENT.md`

## Claude Code Bridge

The bridge enables launching Claude Code CLI sessions from Telegram. See [bridge-deployment-runbook.md](bridge-deployment-runbook.md) for full setup.

Key config in `config.json`:

```json
{
  "bridge": {
    "enabled": true,
    "hook_port": 18792,
    "hook_bind": "0.0.0.0",
    "projects": [
      {"name": "MTClaw", "path": "/home/nqh/shared/MTClaw"},
      {"name": "NQH-Bot", "path": "/home/nqh/shared/NQH-Bot-Platform"},
      {"name": "SDLC", "path": "/home/nqh/shared/SDLC-Orchestrator"}
    ]
  }
}
```

- **`hook_bind`**: Set `0.0.0.0` when running inside Docker (so host-side Claude Code can reach the hook endpoint). Default `127.0.0.1` for host-mode.
- **`projects`**: Pre-registered at gateway startup as "global" owner (visible to all tenants). Users can also register per-tenant projects via `/cc register`.

## Host-Mode Deployment (Dev/Test)

For development, MTClaw can run directly on host instead of Docker. PostgreSQL and AI-Platform still run in Docker.

```bash
# Build
make build

# Override Docker-internal hostnames to localhost
export MTCLAW_POSTGRES_DSN=postgres://mtclaw:PASSWORD@localhost:5470/mtclaw?sslmode=disable
export MTCLAW_BFLOW_BASE_URL=http://localhost:8120/api/v1

# Load remaining env vars and run
set -a && source .env && set +a
./mtclaw
```

AI-Platform is reachable at `localhost:8120` because `bflow-ai-gateway-staging` exposes port 8120 on the host. PostgreSQL is at `localhost:5470` per port allocation.

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
