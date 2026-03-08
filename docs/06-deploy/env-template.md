---
title: MTClaw Environment Variables Template
version: 1.0.0
sdlc_stage: "06-deploy"
sdlc_version: "6.1.1"
status: active
created: 2026-03-03
updated: 2026-03-03
owner: "@devops"
---

# Environment Variables Template

Copy this template to `.env` in the project root and fill in real values.

## .env.example

```bash
# ─── MTClaw Deployment Configuration ───
# Copy to .env and fill in real values.
# DO NOT commit .env to git (already in .gitignore).

# ─── PostgreSQL (managed overlay) ───
# Host port mapping (internal always 5432)
POSTGRES_PORT=5470
POSTGRES_USER=mtclaw
POSTGRES_PASSWORD=CHANGE_ME_STRONG_PASSWORD
POSTGRES_DB=mtclaw

# DSN uses container hostname "postgres" (internal port 5432)
GOCLAW_POSTGRES_DSN=postgres://mtclaw:CHANGE_ME_STRONG_PASSWORD@postgres:5432/mtclaw?sslmode=disable

# ─── Encryption ───
# Generate: openssl rand -hex 32
GOCLAW_ENCRYPTION_KEY=GENERATE_WITH_openssl_rand_hex_32

# ─── Bflow AI-Platform ───
# API key provisioned by CTO (aip_ prefix)
GOCLAW_BFLOW_API_KEY=aip_REPLACE_WITH_REAL_KEY
GOCLAW_BFLOW_BASE_URL=http://ai-platform:8120/api/v1
BFLOW_TENANT_ID=mts

# ─── AI Provider ───
GOCLAW_PROVIDER=bflow-ai-platform
GOCLAW_MODEL=qwen3:14b

# ─── Telegram Bot ───
# Token from @BotFather
GOCLAW_TELEGRAM_TOKEN=REPLACE_WITH_BOTFATHER_TOKEN
GOCLAW_TELEGRAM_POLLING=true

# ─── Server ───
GOCLAW_PORT=18790
GOCLAW_LOG_LEVEL=info
GOCLAW_LOG_FORMAT=json

# ─── Cost Guardrails ───
TENANT_MONTHLY_TOKEN_LIMIT=1000000
TENANT_DAILY_REQUEST_LIMIT=5000

# ─── Optional: Owner IDs ───
# Comma-separated Telegram user IDs for admin access
# GOCLAW_OWNER_IDS=123456789

# ─── Optional: Gateway Token ───
# Auto-generated on first start if not set
# GOCLAW_GATEWAY_TOKEN=

# ─── Optional: Debug ───
# GOCLAW_TRACE_VERBOSE=1
# GOCLAW_LOG_LEVEL=debug
```

## Variable Reference

### Required Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `POSTGRES_PORT` | Host port for PostgreSQL | `5470` |
| `POSTGRES_USER` | Database username | `mtclaw` |
| `POSTGRES_PASSWORD` | Database password (strong!) | Generated |
| `POSTGRES_DB` | Database name | `mtclaw` |
| `GOCLAW_POSTGRES_DSN` | Full connection string (uses container hostname) | See template |
| `GOCLAW_ENCRYPTION_KEY` | AES-256 hex key (32 bytes = 64 hex chars) | `openssl rand -hex 32` |
| `GOCLAW_BFLOW_API_KEY` | Bflow AI-Platform key (`aip_` prefix) | CTO provisioned |
| `GOCLAW_TELEGRAM_TOKEN` | Telegram bot token from BotFather | `123456:ABC-DEF` |

### Provider Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GOCLAW_PROVIDER` | AI provider name | `bflow-ai-platform` |
| `GOCLAW_MODEL` | Model name | `qwen3:14b` |
| `GOCLAW_BFLOW_BASE_URL` | AI-Platform API URL | `http://ai-platform:8120/api/v1` |
| `BFLOW_TENANT_ID` | AI-Platform tenant ID | `mts` |

### Server Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GOCLAW_PORT` | Gateway HTTP port | `18790` |
| `GOCLAW_LOG_LEVEL` | Log level (`debug`, `info`, `warn`, `error`) | `info` |
| `GOCLAW_LOG_FORMAT` | Log format (`json`, `text`) | `json` |

### Cost Control

| Variable | Description | Default |
|----------|-------------|---------|
| `TENANT_MONTHLY_TOKEN_LIMIT` | Max tokens per month | `1000000` |
| `TENANT_DAILY_REQUEST_LIMIT` | Max requests per day | `5000` |

### Optional Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GOCLAW_OWNER_IDS` | Comma-separated admin user IDs | Empty |
| `GOCLAW_GATEWAY_TOKEN` | API auth token (auto-generated if empty) | Auto |
| `GOCLAW_TRACE_VERBOSE` | Enable verbose tracing (`0`/`1`) | `0` |
| `GOCLAW_TELEGRAM_POLLING` | Use polling instead of webhook | `true` |

## Security Checklist

- [ ] `.env` is in `.gitignore` (never commit secrets)
- [ ] `POSTGRES_PASSWORD` is unique and strong (not the default `mtclaw`)
- [ ] `GOCLAW_ENCRYPTION_KEY` generated fresh (`openssl rand -hex 32`)
- [ ] `GOCLAW_BFLOW_API_KEY` is the correct key for this environment
- [ ] `GOCLAW_TELEGRAM_TOKEN` matches the intended bot
