---
title: MTClaw Port Allocation Request
version: 1.0.0
sdlc_stage: "06-deploy"
sdlc_version: "6.1.1"
status: pending-approval
created: 2026-03-03
updated: 2026-03-03
owner: "@devops"
---

# Port Allocation Request — MTClaw

**To:** IT Admin (dvhiep@nqh.com.vn)
**From:** MTClaw DevOps (@devops)
**Date:** 2026-03-03
**Status:** Pending IT Admin approval

## Requested Ports

### MTClaw Platform (`/home/nqh/shared/MTClaw/`)

| Port | Service | Container | Purpose | Status |
|------|---------|-----------|---------|--------|
| **18790** | MTClaw Gateway | `mtclaw-mtclaw-1` | HTTP API + Telegram Bot Gateway | Pending |
| **5470** | PostgreSQL | `mtclaw-postgres-1` | Database (pgvector/pg18) | Pending |

### Justification

- **18790**: High port range to avoid conflicts with existing NQH infrastructure (all platforms use 2500-9xxx range). MTClaw is a lightweight internal tool — no public-facing exposure needed (Telegram polling, no inbound webhook).
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
| **5470** | PostgreSQL | `mtclaw-postgres-1` | Database (pgvector/pg18) | 🆕 Mar 2026 |
```

And add to Port Ranges by Platform:

```
MTClaw:                    5470, 18790
```
