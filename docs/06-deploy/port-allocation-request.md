---
title: MTClaw Port Allocation Request
version: 1.1.0
sdlc_stage: "06-deploy"
sdlc_version: "6.1.1"
status: approved
created: 2026-03-03
updated: 2026-03-10
owner: "@devops"
---

# Port Allocation Request — MTClaw

**To:** IT Admin (dvhiep@nqh.com.vn)
**From:** MTClaw DevOps (@devops)
**Date:** 2026-03-03
**Status:** ✅ APPROVED by IT Admin (March 10, 2026)

## Requested Ports

### MTClaw Platform (`/home/nqh/shared/MTClaw/`)

| Port | Service | Container | Purpose | Status |
|------|---------|-----------|---------|--------|
| **18790** | MTClaw Gateway | `mtclaw-mtclaw-1` | HTTP API + Telegram Bot Gateway | ✅ Approved |
| **18791** | Web Dashboard | `mtclaw-mtclaw-ui-1` | React SPA (nginx reverse proxy) | ✅ Approved |
| **18792** | Bridge Hook Server | `mtclaw-mtclaw-1` | Claude Code hook webhooks | ✅ Approved |
| **5470** | PostgreSQL | `mtclaw-postgres-1` | Database (pgvector/pg18) | ✅ Approved |

### Justification

- **18790**: High port range to avoid conflicts with existing NQH infrastructure (all platforms use 2500-9xxx range). MTClaw is a lightweight internal tool — no public-facing exposure needed (Telegram polling, no inbound webhook).
- **18791**: Web Dashboard UI (React SPA served by nginx). Port 3000 (default) conflicts with existing services on dev host. Using 18791 (adjacent to gateway 18790) for consistency.
- **18792**: Claude Code Bridge hook server — receives HMAC-signed webhooks from Claude Code CLI processes (permission requests, session stop events).
- **5470**: Follows the database port convention (5xxx range). Nearest allocated PostgreSQL ports: 5460 (SOP Generator), 5454 (Bflow Auth Dev). Port 5470 provides comfortable spacing.

### Network Requirements

- **ai-net** Docker network: MTClaw gateway needs access to `ai-platform:8120` (Bflow AI-Platform). This network already exists and is shared with other services.
- **No public URL needed**: MTClaw operates via Telegram polling — no inbound HTTP traffic required from public internet.

### Suggested PORT_ALLOCATION_MANAGEMENT.md Addition

```markdown
### MTClaw Platform (MTS Internal AI Assistant)

| Port | Service | Container | Purpose | Status |
|------|---------|-----------|---------|--------|
| **18790** | MTClaw Gateway | `mtclaw-mtclaw-1` | HTTP API + Telegram Bot | 🆕 Mar 2026 |
| **18791** | Web Dashboard | `mtclaw-mtclaw-ui-1` | React SPA (nginx) | 🆕 Mar 2026 |
| **18792** | Bridge Hook | `mtclaw-mtclaw-1` | Claude Code webhooks | 🆕 Mar 2026 |
| **5470** | PostgreSQL | `mtclaw-postgres-1` | Database (pgvector/pg18) | 🆕 Mar 2026 |
```

And add to Port Ranges by Platform:

```
MTClaw:                    5470, 18790-18792
```
