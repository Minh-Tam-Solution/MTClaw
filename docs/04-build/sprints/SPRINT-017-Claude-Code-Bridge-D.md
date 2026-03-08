---
sprint: 17
title: Claude Code Bridge — Free-Text + Installer + PG Store + Polish (D)
status: PLANNED
date: 2026-05-12
version: "1.0.0"
author: "[@pm] + [@architect]"
framework: SDLC Enterprise Framework 6.1.1
tier: STANDARD
---

# Sprint 17 — Claude Code Bridge: Free-Text + Installer + PG Store + Polish (D)

**Sprint**: 17 of 17 (bridge track: D — final)
**Phase**: 5 (Claude Code Bridge — ADR-010)
**Duration**: 4 days
**Owner**: [@coder] (implementation) + [@pm] (documentation)
**Points**: ~10
**Gate**: Free-text works under capability gate, installer end-to-end, ccpoke hooks detected
**Entry**: Sprint 16 gate + external security review PASSED

---

## 1. Sprint Goal

**Free-text relay (after permission proven), CLI installer, PG store, production polish.**

### Key Outcomes

1. Free-text Telegram -> active tmux pane (capability-gated: InputMode=free_text only)
2. `/cc risk interactive` requires `ProviderAdapter.CapabilityProfile().PermissionHooks` (D7 Layer 0)
3. `mtclaw bridge setup|serve|uninstall` CLI commands
4. Hook generator (HMAC secret, settings update, ccpoke migration detection)
5. `internal/store/pg/bridge_store.go` — full PG store implementation
6. Session cleanup cron job
7. Dockerfile: `apk add tmux` in runtime stage (CTO I3)
8. Documentation: bridge user guide

---

## 2. New Files

| File | Purpose |
|------|---------|
| `internal/store/pg/bridge_store.go` | PG persistence for all bridge tables |
| `cmd/bridge.go` (extend) | `setup`, `serve`, `uninstall` subcommands |

### Modified Files

| File | Change |
|------|--------|
| `internal/claudecode/session_manager.go` | Free-text sendKeys gated by InputMode |
| `internal/claudecode/bridge_policy.go` | `/cc risk interactive` checks ProviderCapabilities |
| `cmd/gateway_managed.go` | Inject PG bridge store |
| `Dockerfile` | Add `apk add tmux` in runtime stage |
| `internal/cron/service.go` | Register bridge session cleanup job |

---

## 3. Acceptance Criteria

| # | Criterion | Verification |
|---|-----------|-------------|
| 1 | Free-text sendKeys only works when InputMode=free_text | Unit test |
| 2 | Free-text rejected if session state != IDLE | Unit test |
| 3 | `/cc risk interactive` rejected if provider lacks PermissionHooks | Unit test |
| 4 | `mtclaw bridge setup` generates hook scripts + HMAC secret | CLI test |
| 5 | `mtclaw bridge uninstall` removes hooks cleanly | CLI test |
| 6 | PG bridge store CRUD operations work | Integration test |
| 7 | Session cleanup cron removes stopped sessions >24h | Unit test |
| 8 | Dockerfile builds with tmux available | Docker build test |
| 9 | ccpoke hooks detected and migration path documented | CLI check |
| 10 | All 6 mandatory acceptance criteria from ADR-010 pass | Full suite |
| 11 | `make build && make test` passes | CI gate |

---

## 4. Bridge Feature Complete Checklist

After Sprint 17, verify all ADR-010 acceptance criteria:

1. Cross-tenant isolation
2. Provider downgrade
3. Workspace integrity
4. Fail-closed
5. Replay + duplicate callback
6. Long-running busy queue

---

## 5. Post-Bridge: Sprint E+ (Future)

| Sprint | Content |
|--------|---------|
| E | Cursor adapter stub -> real implementation |
| F | Codex CLI adapter |
| G | Gemini CLI adapter |
| H | Role-based ApproverACL expansion |
| I | File-to-PG migration tooling |
| J | JSONL-to-PG reconciliation |
