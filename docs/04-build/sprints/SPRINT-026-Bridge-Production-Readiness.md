# SPRINT-026: Bridge Production Readiness + Architecture Realignment

**Status**: COMPLETE
**Duration**: 5 days (planned), 4.75 days (actual)
**Depends on**: Sprint 25 (Fallback Deploy + Observability — COMPLETE)
**Priority**: P0 (Bridge persistence, SDLC realignment)

---

## Objective

Make the Claude Code Bridge production-ready with PostgreSQL session persistence, audit dual-write, Docker enablement, and fix standalone per-agent SOUL injection. Simultaneously realign SDLC documentation (Stages 02-04) that drifted during Sprint 13-25 feature development.

## Context

- Bridge sessions are in-memory only — lost on container restart
- Migration 000018 (`bridge_sessions` table) exists but no Go store
- `AuditWriter` (JSONL + PG dual-write) exists in `bridge_audit.go` but was never wired to `SessionManager`
- System Architecture Document v1.0.0 doesn't reflect bridge, fallback, or multi-provider
- Sprint 24-25 have no completion reports
- Standalone mode shares one `contextFiles` across all agents (per-agent SOUL broken)

## SDLC Gap Analysis Summary

12 drift categories identified between Stage 02 design and Sprint 1-25 implementation:

| # | Drift | Impact |
|---|-------|--------|
| 1 | Claude Code Bridge (9K LOC, 49 files) not in design | Architecture doc outdated |
| 2 | Provider fallback contradicts "single provider" design | ADR-014 exists, arch doc not updated |
| 3 | Agent Teams `team_tasks` is ADR-012 Option B (APPROVED) | Previously misframed as dead code |
| 4 | Channel landscape changed (removed Discord/Feishu, added MS Teams) | Integration docs partially updated |
| 5 | Bridge sessions in-memory only | Migration 000018 exists, no PG store |

---

## Deliverables

### T26.1: Update System Architecture Document v2.0.0 (1d) — REALIGNMENT

**File:** `docs/02-design/system-architecture-document.md`

- Added Layer 3.5: Claude Code Bridge (SessionManager, HookServer, TmuxBridge, HealthMonitor)
- Added Layer 4: Provider Fallback Chain (primary → fallback with retryable-error classification)
- Updated channels: removed Discord/Feishu/WhatsApp, added MS Teams (ADR-007)
- Fixed stale values: port 8080→18790, 16→17 SOULs, 7→18+ migrations
- Added bridge data flow diagram
- Referenced all 14 ADRs (ADR-001 through ADR-014)
- Updated observability: PG traces + optional OTLP export (NOT Prometheus)

### T26.2: BridgeSessionStore + PG Persistence (1.5d) — NEW

| File | Change |
|------|--------|
| `internal/store/bridge_session_store.go` | NEW — `BridgeSessionStore` interface + `BridgeSessionRecord` struct (20 fields) |
| `internal/store/pg/bridge_sessions.go` | NEW — Full PG implementation: Upsert, Get, ListByTenant, ListActive, UpdateStatus, UpdateRiskMode, DeleteOlderThan |
| `internal/store/stores.go` | Added `BridgeSessions BridgeSessionStore` field |
| `internal/store/pg/factory.go` | Wire `NewPGBridgeSessionStore(db)` |
| `internal/claudecode/types.go` | Added `SessionStateDisconnected` |
| `internal/claudecode/session_manager.go` | Added `pgStore` field, `SetStore()`, `LoadFromStore()`, best-effort dual-write on Create/Kill/Transition/UpdateRiskMode/CleanupStopped |
| `cmd/gateway.go` | Wire store in managed mode + startup recovery via `LoadFromStore()` |

**Design decisions:**
- Memory primary (low-latency), PG secondary (persistence across restarts)
- Best-effort PG writes — log errors, don't fail operations
- Recovered sessions marked `disconnected` (tmux gone after restart)
- `INSERT ... ON CONFLICT (id) DO UPDATE` for idempotent upsert

### T26.3: Docker Managed Mode Bridge (0.5d) — NEW

| File | Change |
|------|--------|
| `Dockerfile` | Added `ARG ENABLE_BRIDGE=false`, conditional `apk add tmux` |
| `docker-compose.yml` | Build arg `ENABLE_BRIDGE`, env vars `MTCLAW_BRIDGE_ENABLED`, `MTCLAW_BRIDGE_HOOK_PORT`, `MTCLAW_BRIDGE_AUDIT_DIR` |
| `internal/config/config_load.go` | Parse `MTCLAW_BRIDGE_ENABLED`, `MTCLAW_BRIDGE_HOOK_PORT`, `MTCLAW_BRIDGE_AUDIT_DIR` |
| `.env.example` | Bridge env vars with defaults |

### T26.4: Audit Dual-Write Wiring (0.25d) — NEW

| File | Change |
|------|--------|
| `internal/claudecode/session_manager.go` | Added `audit` field, `SetAuditWriter()`, `emitAudit()` helper. Audit events on: `session.created`, `session.killed`, `session.risk_changed` |
| `cmd/gateway.go` | Create `AuditWriter` with JSONL dir + optional PG connection, wire to SessionManager, `defer Close()` |

**Key finding:** `AuditWriter` already existed in `bridge_audit.go` with full JSONL+PG dual-write. T26.4 was wiring-only — smaller than originally planned.

### T26.5: Sprint 25.5 Bug Fixes (0.25d) — BUGFIX

Already in working tree from Sprint 25.5:
- `cmd/gateway.go` — Admission config wiring fix
- `internal/channels/telegram/commands_cc.go` — Filter stopped sessions from `/cc sessions`

### T26.6: Sprint 24-25 Completion Reports (0.25d) — REALIGNMENT

| File | Description |
|------|-------------|
| `docs/04-build/sprints/SPRINT-024-COMPLETION.md` | Sprint 24 deliverables, test coverage (11+5 tests), deviations |
| `docs/04-build/sprints/SPRINT-025-COMPLETION.md` | Sprint 25 deliverables, CTO B-series resolutions, outstanding debt |

### T26.7: Standalone Per-Agent SOUL Fix (0.5d) — BUGFIX

**Root cause:** In standalone mode, `contextFiles` were loaded once from the default agent's workspace and shared to all agents. Non-default agents with their own workspace never got their own SOUL.md injected.

**Fix in `cmd/gateway.go`:** When creating non-default agents in standalone mode, check if their workspace differs from default. If so, load per-agent context files via `bootstrap.LoadWorkspaceFiles()` + `bootstrap.BuildContextFiles()` from the agent's own workspace.

---

## Test Coverage

| Test File | Tests | Status |
|-----------|-------|--------|
| `internal/store/pg/bridge_sessions_test.go` | 5 (serialization, helpers, column count) | PASS |
| `internal/claudecode/bridge_store_test.go` | 5 (dual-write create/kill/risk, load from store, no-store safety) | PASS |
| Full suite regression | All existing tests | PASS |

---

## Acceptance Criteria

- [x] System architecture doc v2.0.0 reflects bridge + fallback + multi-provider
- [x] Container restart recovers bridge sessions from PostgreSQL (marked disconnected)
- [x] Audit events dual-written (JSONL + `bridge_audit_events` table)
- [x] `docker compose build --build-arg ENABLE_BRIDGE=true` includes tmux
- [x] Each standalone agent loads its own workspace's SOUL.md
- [x] Sprint 24-25 completion reports exist
- [x] Bug fixes committed (admission wiring + sessions filter)
- [x] All tests pass — zero regressions
