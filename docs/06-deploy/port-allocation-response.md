---
title: MTClaw Port Allocation Response
version: 1.0.0
sdlc_stage: "06-deploy"
sdlc_version: "6.1.1"
status: approved
created: 2026-03-03
owner: "@itadmin"
---

# Port Allocation Response — MTClaw

**To:** MTClaw DevOps (@devops)
**From:** IT Admin (@itadmin)
**Date:** 2026-03-03
**Status:** ALL APPROVED

---

## Port Verification

| Port | Container | Conflict Check | Status |
|------|-----------|----------------|--------|
| **18790** | `mtclaw-mtclaw-1` | No conflicts — high range, well outside existing 2500-9xxx | **APPROVED** |
| **5470** | `mtclaw-postgres-1` | No conflicts — spacing OK from SOP (5460) and Bflow Auth (5454) | **APPROVED** |

Both ports verified running via `ss -tlnp` and `docker ps`.

## Network

- `ai-net` Docker network: Confirmed existing, shared with AI-Platform (8120)
- No public URL needed: Acknowledged — Telegram polling mode, no Cloudflare route required

## Documentation Updated

| Document | Version | Change |
|----------|---------|--------|
| `PORT_ALLOCATION_MANAGEMENT.md` | 3.8 → 3.9 | Added MTClaw section (2 ports), port ranges, allocation rules, statistics |
| Training copy (DevOps) | Synced | Same content |

## Next Steps

None required — ports are already active and documented.
