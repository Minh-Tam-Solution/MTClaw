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
MTCLAW_POSTGRES_DSN=postgres://mtclaw:CHANGE_ME_STRONG_PASSWORD@postgres:5432/mtclaw?sslmode=disable

# ─── Encryption ───
# Generate: openssl rand -hex 32
MTCLAW_ENCRYPTION_KEY=GENERATE_WITH_openssl_rand_hex_32

# ─── Bflow AI-Platform ───
# API key provisioned by CTO (aip_ prefix)
MTCLAW_BFLOW_API_KEY=aip_REPLACE_WITH_REAL_KEY
MTCLAW_BFLOW_BASE_URL=http://ai-platform:8120/api/v1
BFLOW_TENANT_ID=mts

# ─── AI Provider ───
MTCLAW_PROVIDER=bflow-ai-platform
MTCLAW_MODEL=qwen3:14b

# ─── Telegram Bot ───
# Token from @BotFather
MTCLAW_TELEGRAM_TOKEN=REPLACE_WITH_BOTFATHER_TOKEN
MTCLAW_TELEGRAM_POLLING=true

# ─── Server ───
MTCLAW_PORT=18790
MTCLAW_LOG_LEVEL=info
MTCLAW_LOG_FORMAT=json

# ─── Cost Guardrails ───
TENANT_MONTHLY_TOKEN_LIMIT=1000000
TENANT_DAILY_REQUEST_LIMIT=5000

# ─── Optional: Owner IDs ───
# Comma-separated Telegram user IDs for admin access
# MTCLAW_OWNER_IDS=123456789

# ─── Optional: Gateway Token ───
# Auto-generated on first start if not set
# MTCLAW_GATEWAY_TOKEN=

# ─── Optional: Debug ───
# MTCLAW_TRACE_VERBOSE=1
# MTCLAW_LOG_LEVEL=debug
```

## Variable Reference

### Required Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `POSTGRES_PORT` | Host port for PostgreSQL | `5470` |
| `POSTGRES_USER` | Database username | `mtclaw` |
| `POSTGRES_PASSWORD` | Database password (strong!) | Generated |
| `POSTGRES_DB` | Database name | `mtclaw` |
| `MTCLAW_POSTGRES_DSN` | Full connection string (uses container hostname) | See template |
| `MTCLAW_ENCRYPTION_KEY` | AES-256 hex key (32 bytes = 64 hex chars) | `openssl rand -hex 32` |
| `MTCLAW_BFLOW_API_KEY` | Bflow AI-Platform key (`aip_` prefix) | CTO provisioned |
| `MTCLAW_TELEGRAM_TOKEN` | Telegram bot token from BotFather | `123456:ABC-DEF` |

### Provider Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `MTCLAW_PROVIDER` | AI provider name | `bflow-ai-platform` |
| `MTCLAW_MODEL` | Model name | `qwen3:14b` |
| `MTCLAW_BFLOW_BASE_URL` | AI-Platform API URL | `http://ai-platform:8120/api/v1` |
| `BFLOW_TENANT_ID` | AI-Platform tenant ID | `mts` |

### Server Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `MTCLAW_PORT` | Gateway HTTP port | `18790` |
| `MTCLAW_LOG_LEVEL` | Log level (`debug`, `info`, `warn`, `error`) | `info` |
| `MTCLAW_LOG_FORMAT` | Log format (`json`, `text`) | `json` |

### Cost Control

| Variable | Description | Default |
|----------|-------------|---------|
| `TENANT_MONTHLY_TOKEN_LIMIT` | Max tokens per month | `1000000` |
| `TENANT_DAILY_REQUEST_LIMIT` | Max requests per day | `5000` |

### Optional Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `MTCLAW_OWNER_IDS` | Comma-separated admin user IDs | Empty |
| `MTCLAW_GATEWAY_TOKEN` | API auth token (auto-generated if empty) | Auto |
| `MTCLAW_TRACE_VERBOSE` | Enable verbose tracing (`0`/`1`) | `0` |
| `MTCLAW_TELEGRAM_POLLING` | Use polling instead of webhook | `true` |

## Security Checklist

- [ ] `.env` is in `.gitignore` (never commit secrets)
- [ ] `POSTGRES_PASSWORD` is unique and strong (not the default `mtclaw`)
- [ ] `MTCLAW_ENCRYPTION_KEY` generated fresh (`openssl rand -hex 32`)
- [ ] `MTCLAW_BFLOW_API_KEY` is the correct key for this environment
- [ ] `MTCLAW_TELEGRAM_TOKEN` matches the intended bot
