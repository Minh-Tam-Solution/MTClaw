---
sprint: 14
title: Claude Code Bridge — Telegram Commands + Identity + Audit (A2)
status: PLANNED
date: 2026-04-14
version: "1.0.0"
author: "[@pm] + [@architect]"
framework: SDLC Enterprise Framework 6.1.1
tier: STANDARD
---

# Sprint 14 — Claude Code Bridge: Telegram Commands + Identity + Audit (A2)

**Sprint**: 14 of 17 (bridge track: A2 of A1/A2/B/C/D)
**Phase**: 5 (Claude Code Bridge — ADR-010)
**Duration**: 4 days
**Owner**: [@coder] (implementation) + [@pm] (CEO validation coordination)
**Points**: ~12 (security-critical)
**Gate**: All /cc commands functional, redaction passes, audit populated, cross-tenant isolation verified
**ADR**: `docs/02-design/01-ADRs/SPEC-0010-ADR-010-Claude-Code-Bridge.md`
**Entry**: Sprint 13 gate passed (tmux bridge unit tests green)

---

## 1. Entry Criteria

| Criterion | Status | Owner |
|-----------|--------|-------|
| Sprint 13 gate passed | Pending | [@cto] |
| `internal/claudecode/` 7 core files + tests green | Pending | [@coder] |
| `BridgeConfig` wired into config.go | Pending | [@coder] |
| tmux create/capture/kill verified | Pending | [@coder] |

---

## 2. Sprint Goal

**Security-first Telegram commands. CEO can `/cc launch` + `/cc capture` daily.**

This sprint adds the human-facing layer: Telegram commands, identity binding, capability enforcement, input sanitization, output redaction, audit logging, and the standalone file-based store.

### Key Outcomes

1. 9 `/cc` commands working in Telegram
2. Input sanitizer reusing `DefaultDenyPatterns()` (87 patterns)
3. Output redactor scrubbing secrets from capturePane
4. JSONL audit trail (primary) + PG dual-write (secondary, best-effort)
5. Migration 000018: 4 bridge tables with RLS
6. Standalone file-based store (`~/.mtclaw/bridge_sessions.json`)
7. `bridgeManager` wired into Telegram Channel struct
8. Doctor-lite: tmux present, hook port config, active sessions, audit writable

---

## 3. Architecture — [@architect]

### 3.1 New Files (Sprint 14)

| File | Purpose |
|------|---------|
| `internal/claudecode/bridge_policy.go` | Capability model enforcement (D2) |
| `internal/claudecode/input_sanitizer.go` | Secondary defense wrapping shell.go patterns |
| `internal/claudecode/output_redactor.go` | Secret redaction for capturePane output |
| `internal/claudecode/bridge_audit.go` | JSONL (primary) + PG (secondary) dual-write (L3) |
| `internal/channels/telegram/commands_cc.go` | /cc command handlers |
| `migrations/000018_claudecode_bridge.up.sql` | 4 tables + RLS |
| `migrations/000018_claudecode_bridge.down.sql` | Rollback |

### 3.2 Modified Files

| File | Change | Risk |
|------|--------|------|
| `internal/channels/telegram/commands.go` | Add `/cc` dispatch case | Low |
| `internal/channels/telegram/channel.go` | Add `bridgeManager` field + setter | Low |
| `cmd/gateway.go` | Wire bridge manager, pass to Telegram channel | Medium |
| `cmd/bridge.go` | New CLI: `mtclaw bridge status` (doctor-lite) | Low |

### 3.3 /cc Commands (Sprint A2)

| Command | Purpose | RiskMode required |
|---------|---------|-------------------|
| `/cc link` | Bind Telegram identity to bridge | - |
| `/cc launch [project]` | Start Claude Code session | - |
| `/cc sessions` | List active sessions | - |
| `/cc capture [n]` | Show last N lines from tmux | read+ |
| `/cc kill [session]` | Terminate session | owner |
| `/cc projects` | List registered projects | - |
| `/cc register [name] [path]` | Register project | - |
| `/cc switch [session]` | Switch active session routing | - |
| `/cc risk [read\|patch]` | Change session risk mode | owner |

**Note**: `/cc risk interactive` deferred to Sprint 17 (D) — requires permission approval (Sprint 16/C) to be proven first.

### 3.4 Wiring Pattern (CTO I2)

```go
// cmd/gateway.go — after tool registry creation
if cfg.Bridge.Enabled {
    bridgeMgr := claudecode.NewSessionManager(cfg.Bridge, store, auditWriter)
    telegramCh.SetBridgeManager(bridgeMgr)
}
```

### 3.5 Standalone File Store

Following `internal/cron/service.go` pattern:
- `~/.mtclaw/bridge_sessions.json` — single-process local mode only
- File header warning: "Single-process only. File-to-PG migration not supported in Sprint A-D."
- Used when `config.Database.Mode != "managed"`

---

## 4. Task Breakdown

### Day 1: T14-SEC-01 — Policy + Sanitizer + Redactor

| File | Task |
|------|------|
| `internal/claudecode/bridge_policy.go` | Capability enforcement: check InputMode, ToolPolicy, OutputPolicy per session |
| `internal/claudecode/input_sanitizer.go` | Wrap `tools.DefaultDenyPatterns()` + bridge-specific patterns (tmux escape sequences) |
| `internal/claudecode/output_redactor.go` | Redact: API keys, tokens, DSN strings, encryption keys from capturePane output |
| Tests for all 3 files | ~30 tests (87 deny patterns, redaction patterns, capability enforcement) |

**Sanitizer failure policy**: reject, not pass-through.

### Day 2: T14-CMD-01 — Telegram /cc Commands

| File | Task |
|------|------|
| `internal/channels/telegram/commands_cc.go` | Implement 9 /cc commands |
| `internal/channels/telegram/commands.go` | Add `case "cc":` dispatch |
| `internal/channels/telegram/channel.go` | Add bridgeManager field + SetBridgeManager() |

**Identity flow**: Every /cc command checks `actor_id` via Telegram user ID. No action without identity.

### Day 3: T14-DATA-01 — Migration + Audit + Standalone Store

| File | Task |
|------|------|
| `migrations/000018_claudecode_bridge.up.sql` | 4 tables (sessions, projects, permissions, audit_events) + RLS |
| `migrations/000018_claudecode_bridge.down.sql` | DROP tables |
| `internal/claudecode/bridge_audit.go` | JSONL primary + PG secondary dual-write |
| Standalone file store in session_manager.go | JSON file persistence for standalone mode |

### Day 4: T14-WIRE-01 — Gateway Wiring + Doctor + Integration Tests

| File | Task |
|------|------|
| `cmd/gateway.go` | Wire bridgeManager into Telegram channel |
| `cmd/bridge.go` | `mtclaw bridge status` — doctor-lite checks |
| Integration tests | Cross-tenant isolation, /cc command flow, redaction verification |

**Doctor-lite minimum checks**:
- tmux binary present
- Hook port config valid
- Active sessions count
- Store path writable
- Audit log writable

---

## 5. Acceptance Criteria

| # | Criterion | Verification |
|---|-----------|-------------|
| 1 | All 9 /cc commands respond in Telegram | Manual test |
| 2 | Input sanitizer rejects 87+ deny patterns | Unit test |
| 3 | Output redactor scrubs API keys, DSN, tokens | Unit test |
| 4 | JSONL audit file written for every /cc action | File check |
| 5 | Cross-tenant isolation: Actor A cannot see tenant B sessions | Unit test |
| 6 | Migration 000018 applies cleanly | `make migrate-up` |
| 7 | Standalone store persists sessions across restart | Unit test |
| 8 | `mtclaw bridge status` reports green | CLI test |
| 9 | `make build && make test` passes | CI gate |

---

## 6. CEO Validation Phase (2-3 days after Sprint 14)

After Sprint 14 gate passes, CEO uses bridge daily:

| Day | Activity | Feedback |
|-----|----------|----------|
| 1 | `/cc launch` on real project, `/cc capture` output | Friction log |
| 2 | `/cc sessions` + `/cc switch` between projects | Friction log |
| 3 | `/cc risk patch` escalation, review audit trail | Go/No-Go for Sprint 15 |

**Exit criteria**: CEO signs off on UX + security model before Sprint 15 (HookServer).

---

## 7. NOT in Sprint 14

| Item | Sprint |
|------|--------|
| HookServer (D4) | 15 (B) |
| HMAC-SHA256 hook auth (D5) | 15 (B) |
| Stop notification | 15 (B) |
| Permission approval (D6) | 16 (C) |
| Free-text relay | 17 (D) |
| `/cc risk interactive` | 17 (D) |
| PG-only bridge store | 17 (D) |

---

## 8. Verification Checklist

```bash
# 1. Build
make build

# 2. All tests
make test

# 3. Bridge package
go test ./internal/claudecode/... -v -race -count=1

# 4. Telegram /cc tests
go test ./internal/channels/telegram/ -run TestCC -v

# 5. Migration
make migrate-up

# 6. Doctor
./mtclaw bridge status

# 7. Sanitizer coverage
go test ./internal/claudecode/ -run TestSanitizer -v -count=1
```
