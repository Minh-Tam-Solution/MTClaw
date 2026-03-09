---
title: MTClaw Operations Runbook
version: 1.0.0
sdlc_stage: "06-deploy"
sdlc_version: "6.1.1"
status: active
created: 2026-03-03
updated: 2026-03-03
owner: "@devops"
---

# MTClaw Operations Runbook

## Quick Reference

```bash
# Set compose alias (add to ~/.bashrc)
alias mtc='cd /home/nqh/shared/MTClaw && docker compose -f docker-compose.yml -f docker-compose.managed.yml -f docker-compose.mts.yml'

# Common operations
mtc ps                     # Status
mtc logs mtclaw -f --tail 50  # Live logs
mtc restart mtclaw         # Restart gateway
mtc up -d --build          # Rebuild + deploy
mtc down                   # Stop all
curl -sf localhost:18790/health  # Health check
```

## 1. Start / Stop / Restart

### Start (fresh or after stop)

```bash
cd /home/nqh/shared/MTClaw
docker compose -f docker-compose.yml \
  -f docker-compose.managed.yml \
  -f docker-compose.mts.yml \
  up -d --build
```

### Stop (preserves data volumes)

```bash
docker compose -f docker-compose.yml \
  -f docker-compose.managed.yml \
  -f docker-compose.mts.yml \
  down
```

### Stop + remove volumes (DESTRUCTIVE — loses database)

```bash
docker compose -f docker-compose.yml \
  -f docker-compose.managed.yml \
  -f docker-compose.mts.yml \
  down -v
```

### Restart gateway only (no rebuild)

```bash
docker compose -f docker-compose.yml \
  -f docker-compose.managed.yml \
  -f docker-compose.mts.yml \
  restart mtclaw
```

## 2. Logs

### Follow gateway logs

```bash
docker compose -f docker-compose.yml \
  -f docker-compose.managed.yml \
  -f docker-compose.mts.yml \
  logs mtclaw -f --tail 100
```

### Check for errors only

```bash
docker compose -f docker-compose.yml \
  -f docker-compose.managed.yml \
  -f docker-compose.mts.yml \
  logs mtclaw 2>&1 | grep -i "error\|panic\|fatal"
```

### PostgreSQL logs

```bash
docker compose -f docker-compose.yml \
  -f docker-compose.managed.yml \
  -f docker-compose.mts.yml \
  logs postgres --tail 50
```

## 3. Health Checks

### Gateway health

```bash
curl -sf http://localhost:18790/health
# Expected: {"status":"ok","protocol":3}
```

### PostgreSQL connectivity

```bash
docker exec mtclaw-postgres-1 pg_isready -U mtclaw
# Expected: /var/run/postgresql:5432 - accepting connections
```

### AI-Platform reachability (from gateway container)

```bash
docker exec mtclaw-mtclaw-1 wget -qO- http://ai-platform:8120/health
```

### Container resource usage

```bash
docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}" mtclaw-mtclaw-1 mtclaw-postgres-1
```

## 4. Update / Deploy New Version

### Pull latest code and rebuild

```bash
cd /home/nqh/shared/MTClaw
git pull --rebase origin main

docker compose -f docker-compose.yml \
  -f docker-compose.managed.yml \
  -f docker-compose.mts.yml \
  up -d --build

# Verify
docker compose -f docker-compose.yml \
  -f docker-compose.managed.yml \
  -f docker-compose.mts.yml \
  ps

curl -sf http://localhost:18790/health
```

### Schema migration (automatic)

Migrations run automatically on startup via `docker-entrypoint.sh`. After adding new migration files, just rebuild.

If `RequiredSchemaVersion` in `internal/upgrade/version.go` doesn't match the migration count, the startup will fail. Fix:

```bash
# Check migration count
ls migrations/*.up.sql | wc -l
# Update RequiredSchemaVersion to match, then rebuild
```

### Manual migration (dry-run)

```bash
docker compose -f docker-compose.yml \
  -f docker-compose.managed.yml \
  -f docker-compose.upgrade.yml \
  run --rm upgrade --dry-run
```

## 5. Database Operations

### Connect to PostgreSQL

```bash
docker exec -it mtclaw-postgres-1 psql -U mtclaw -d mtclaw
```

### Check schema version

```sql
SELECT version, dirty FROM schema_migrations;
```

### List agents

```sql
SELECT id, name, model, provider FROM agents;
```

### Check active sessions

```sql
SELECT COUNT(*) as active_sessions FROM sessions WHERE updated_at > NOW() - INTERVAL '1 hour';
```

### Backup database

```bash
docker exec mtclaw-postgres-1 pg_dump -U mtclaw mtclaw > /tmp/mtclaw-backup-$(date +%Y%m%d).sql
```

### Restore database

```bash
docker exec -i mtclaw-postgres-1 psql -U mtclaw mtclaw < /tmp/mtclaw-backup-YYYYMMDD.sql
```

## 6. Rollback

### Rollback to previous image (if build failed)

```bash
# List available images
docker images mtclaw-mtclaw --format "table {{.Tag}}\t{{.CreatedAt}}\t{{.Size}}"

# Revert code
cd /home/nqh/shared/MTClaw
git log --oneline -5   # find the commit to revert to
git checkout <commit-hash>

# Rebuild
docker compose -f docker-compose.yml \
  -f docker-compose.managed.yml \
  -f docker-compose.mts.yml \
  up -d --build
```

### Rollback database migration

```bash
docker exec mtclaw-mtclaw-1 /app/mtclaw migrate down
```

## 7. Troubleshooting

### Container won't start

```bash
# Check logs
docker compose -f docker-compose.yml \
  -f docker-compose.managed.yml \
  -f docker-compose.mts.yml \
  logs mtclaw --tail 50

# Common issues:
# 1. "Database schema (vN) is newer than this binary"
#    → Update RequiredSchemaVersion in internal/upgrade/version.go
#
# 2. "No configuration found. Starting setup wizard..."
#    → Check MTCLAW_BFLOW_API_KEY is set in .env
#    → Verify bflow-ai-platform is in providerPriority (cmd/onboard_auto.go)
#
# 3. "connection refused" to postgres
#    → Check postgres container is healthy: docker compose ps
#    → Check MTCLAW_POSTGRES_DSN uses container hostname "postgres", not "localhost"
```

### Telegram bot not responding

```bash
# Check Telegram token is valid
docker compose -f docker-compose.yml \
  -f docker-compose.managed.yml \
  -f docker-compose.mts.yml \
  logs mtclaw 2>&1 | grep -i telegram

# Common issues:
# 1. "401 Unauthorized" → Invalid MTCLAW_TELEGRAM_TOKEN
# 2. "409 Conflict" → Another bot instance is polling with same token
# 3. No Telegram logs → MTCLAW_TELEGRAM_TOKEN not set or empty
```

### AI-Platform not reachable

```bash
# Test from gateway container
docker exec mtclaw-mtclaw-1 wget -qO- http://ai-platform:8120/health

# If fails, check:
# 1. AI-Platform container is running
docker ps --filter "name=ai-platform"
# 2. ai-net network exists and both containers are on it
docker network inspect ai-net | grep -A2 "mtclaw\|ai-platform"
# 3. API key is valid
docker exec mtclaw-mtclaw-1 env | grep BFLOW
```

### High memory usage

```bash
# Check container stats
docker stats --no-stream mtclaw-mtclaw-1

# If memory > 800MB (80% of 1G limit):
# 1. Check for stuck sessions
docker exec mtclaw-postgres-1 psql -U mtclaw -c "SELECT COUNT(*) FROM sessions WHERE updated_at < NOW() - INTERVAL '24 hours';"
# 2. Restart gateway
docker compose -f docker-compose.yml \
  -f docker-compose.managed.yml \
  -f docker-compose.mts.yml \
  restart mtclaw
```

## 8. Port Allocation

| Port | Service | Container | Status |
|------|---------|-----------|--------|
| 18790 | MTClaw Gateway | mtclaw-mtclaw-1 | Pending IT Admin |
| 5470 | PostgreSQL | mtclaw-postgres-1 | Pending IT Admin |

Contact: dvhiep@nqh.com.vn (IT Admin)
Reference: `/home/nqh/shared/models/core/docs/admin/PORT_ALLOCATION_MANAGEMENT.md`
