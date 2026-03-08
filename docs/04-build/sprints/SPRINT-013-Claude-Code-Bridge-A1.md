---
sprint: 13
title: Claude Code Bridge — Pre-Sprint + Local Session Core (A1)
status: PLANNED
date: 2026-04-07
version: "1.0.0"
author: "[@pm] + [@architect]"
framework: SDLC Enterprise Framework 6.1.1
tier: STANDARD
---

# Sprint 13 — Claude Code Bridge: Pre-Sprint + Local Session Core (A1)

**Sprint**: 13 of 17 (bridge track: A1 of A1/A2/B/C/D)
**Phase**: 5 (Claude Code Bridge — ADR-010)
**Duration**: 4 days (1 pre-sprint + 3 A1)
**Owner**: [@coder] (implementation) + [@pm] (coordination)
**Points**: ~8 (1 audit + 7 core)
**Gate**: Sprint gate — `tmux ls` shows session, create/capture/kill works via Go unit test
**ADR**: `docs/02-design/01-ADRs/SPEC-0010-ADR-010-Claude-Code-Bridge.md`
**Plan**: `/home/dttai/.claude/plans/glowing-gliding-quill.md`

---

## 1. Entry Criteria

| Criterion | Status | Owner |
|-----------|--------|-------|
| CTO Sprint 12 review score received | **CLEARED** — 8.4/10 APPROVED (2026-03-07) | [@cto] |
| ADR-010 committed | Done (2026-03-06) | [@architect] |
| `DefaultDenyPatterns()` exported in shell.go | Done (2026-03-06) | [@architect] |
| Build clean + tests passing | Sprint 12 close | [@coder] |
| tmux available on dev machine | Verify | [@coder] |

### Sprint 12 Carry-Forwards (non-blocking, tracked in Sprint 13 backlog)

| ID | Task | Priority | Owner |
|----|------|----------|-------|
| CTO-58 | `pentest_live_test.go` (T12-SEC-01) | P2 | [@coder] |
| CTO-59 | SHA256 determinism test (T12-TEST-01) | P3 | [@coder] |
| CTO-60 | Benchmark tests (T12-TEST-02) | P3 | [@coder] |
| CTO-61 | Bot Framework URL runbook (T12-OPS-01) | P3 | [@pm] |
| CTO-62 | G5 OaaS readiness doc (T12-PM-01) | P2 | [@pm] |
| CTO-55 | Replace custom contains with strings.Contains | Low | [@coder] (fix on next governance touch) |
| CTO-57 | Delete AllArtifactTypes alias in chain.go | Low | [@coder] (fix on next evidence touch) |
| CTO-64 | Verify rejection message replaces (not appends) agent output | Low | [@coder] |

---

## 2. Sprint Goal

**Stand up the local tmux bridge layer for Claude Code sessions — no Telegram, no hooks, no network.**

This sprint isolates the process/tmux layer so it can be debugged independently before adding Telegram identity, audit, and security in Sprint 14 (A2).

### Key Outcomes

1. ccpoke source audit documented — "Port" vs "Do Not Port" decisions
2. `internal/claudecode/` package created with 7 core files
3. `BridgeConfig` wired into `internal/config/config.go`
4. tmux create/capture/kill works via unit tests
5. Session state machine with ownership model (D8)
6. Project registry with composite workspaceFingerprint
7. Admission control (CPU/memory/per-tenant caps)

---

## 3. Architecture — [@architect]

### 3.1 Package Structure

```
internal/claudecode/
  types.go              -- AgentProviderType, SessionState, RiskMode, Capabilities, BridgeSession
  config.go             -- BridgeConfig struct + defaults
  provider.go           -- ProviderAdapter interface + ClaudeCodeAdapter
  tmux.go               -- TmuxBridge: paste-buffer sendKeys, capturePane, create/kill
  session.go            -- Session state machine, message queue, ownership (D8)
  session_manager.go    -- Multi-tenant lifecycle + admission control
  project.go            -- Project registry + composite workspaceFingerprint
```

### 3.2 Key Design Decisions (from ADR-010)

- **D1**: Bridge = control surface, NOT terminal surrogate
- **D2**: 3-axis capability model (InputMode x ToolPolicy x OutputPolicy)
- **D8**: Immutable session owner, explicit ApproverACL/NotifyACL
- **D9**: ProviderAdapter interface — only ClaudeCodeAdapter in Sprint A

### 3.3 Integration Points

| File | Change | Risk |
|------|--------|------|
| `internal/config/config.go` | Add `Bridge BridgeConfig` field | Low — additive |
| `internal/tools/shell.go` | `DefaultDenyPatterns()` export (done) | Low — single function, returns copy |

### 3.4 WorkspaceFingerprint Composition

```
sha256(canonical_path + ":" + device_inode + ":" + git_root + ":" + remote_url_normalized + ":" + tenant_salt)
```

Prevents workspace confusion on symlinks, bind mounts, worktree clones, multi-tenant shared hosts.

---

## 4. Task Breakdown

### Day 1: T13-PRE-01 — ccpoke Source Audit

**Goal**: Document what to port and what to skip.

| Deliverable | Output |
|-------------|--------|
| ccpoke audit document | `docs/02-design/bridge-ccpoke-audit.md` |

**Audit sections**:
1. tmux edge cases (paste-buffer race conditions, process lifecycle quirks)
2. Telegram API quirks (rate limits, message ordering, edit timing)
3. Session state transition bugs found in ccpoke
4. Hook payload format gotchas
5. **"Do Not Port" section** — features that don't fit MTClaw's governance model

### Day 2: T13-A1-01 — Types + Config + Provider

**Goal**: Core type system compiles. Config wired.

| File | Task | Status |
|------|------|--------|
| `internal/claudecode/types.go` | All types (D2, D8, D9) | Done (architect) |
| `internal/claudecode/config.go` | BridgeConfig + defaults | Done (architect) |
| `internal/claudecode/provider.go` | ProviderAdapter interface + ClaudeCodeAdapter | Create |
| `internal/config/config.go` | Add `Bridge BridgeConfig` field to Config struct | Modify |

**ProviderAdapter interface (D9)**:
```go
type ProviderAdapter interface {
    Name() AgentProviderType
    LaunchCommand(workdir, hookURL, secret string) *exec.Cmd
    InstallHooks(hookURL, secret string) error
    UninstallHooks() error
    ParseStopEvent(payload []byte) (*StopEvent, error)
    CapabilityProfile() ProviderCapabilities
    TranscriptPath(sessionDir string) string
}
```

`ClaudeCodeAdapter` returns `PermissionHooks: true`. Stub adapters for Cursor/Codex/Gemini return `PermissionHooks: false`.

### Day 3: T13-A1-02 — Tmux Bridge + Project Registry

**Goal**: tmux operations work from Go code.

| File | Task | Status |
|------|------|--------|
| `internal/claudecode/tmux.go` | TmuxBridge: create, capture, sendKeys (paste-buffer), kill | Create |
| `internal/claudecode/project.go` | Project CRUD + composite workspaceFingerprint | Create |

**tmux implementation notes** (from ccpoke audit):
- `sendKeys` via `tmux load-buffer` + `tmux paste-buffer` (not char-by-char) for reliability
- `capturePane` via `tmux capture-pane -p -t {target} -S -{lines}`
- All commands use `execCommandContext` with 5s timeout
- Session naming: `cc-{tenant8}-{rand8}` (under tmux 256 char limit)

### Day 4: T13-A1-03 — Session + Manager + Tests

**Goal**: Full session lifecycle works. All unit tests pass.

| File | Task | Status |
|------|------|--------|
| `internal/claudecode/session.go` | Session state machine + message queue + ownership (D8) | Create |
| `internal/claudecode/session_manager.go` | In-memory lifecycle + admission control | Create |
| `internal/claudecode/types_test.go` | ID generation, capability mapping tests | Create |
| `internal/claudecode/tmux_test.go` | tmux command construction tests | Create |
| `internal/claudecode/session_test.go` | State transitions, ownership rules | Create |
| `internal/claudecode/session_manager_test.go` | Admission control, tenant isolation | Create |
| `internal/claudecode/project_test.go` | Fingerprint, project CRUD | Create |

---

## 5. Acceptance Criteria

| # | Criterion | Verification |
|---|-----------|-------------|
| 1 | ccpoke audit doc exists with "Do Not Port" section | File check |
| 2 | `internal/claudecode/` has 7 .go files + tests | `ls internal/claudecode/` |
| 3 | `BridgeConfig` accessible from `config.Config` | Compile check |
| 4 | Session state machine: active -> busy -> idle -> stopped | Unit test |
| 5 | Admission control rejects when limits exceeded | Unit test |
| 6 | WorkspaceFingerprint is deterministic for same inputs | Unit test |
| 7 | Session ID format: `br:{tenant8}:{rand8}` | Unit test |
| 8 | `make build && make test` passes (zero regression) | CI gate |

---

## 6. Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| tmux not on dev machine | Low | High | Day 1 verify, `apk add tmux` in Docker |
| ccpoke audit takes >1 day | Medium | Medium | Time-box to 4h, document known gaps |
| Go competency gap on process mgmt | Medium | Medium | ccpoke patterns as reference, table-driven tests |

---

## 7. NOT in Sprint 13

| Item | Reason | Sprint |
|------|--------|--------|
| Telegram /cc commands | A2 scope | 14 |
| HookServer | Sprint B scope | 15 |
| Permission approval | Sprint C scope | 16 |
| Free-text relay | Sprint D scope | 17 |
| PostgreSQL bridge store | Sprint D scope | 17 |
| Migration 000018 | Sprint 14 (A2) | 14 |
| Audit logging (JSONL + PG) | Sprint 14 (A2) | 14 |
| Input sanitizer / output redactor | Sprint 14 (A2) | 14 |

---

## 8. Verification Checklist

```bash
# 1. Build
make build

# 2. All tests
make test

# 3. claudecode package tests
go test ./internal/claudecode/... -v -race -count=1

# 4. Verify types compile
go vet ./internal/claudecode/...

# 5. Verify config integration
grep -n "Bridge" internal/config/config.go

# 6. Verify shell.go export
grep "DefaultDenyPatterns" internal/tools/shell.go
```
