---
sprint: 33
title: "Unified Command Routing — Sprint A: Shared Commands Package"
status: PLANNING
start_date: 2026-03-17
end_date: 2026-03-28
lead: "@pm (plan) → @coder (implementation)"
framework: SDLC Enterprise Framework 6.1.2
adr: Architecture Review — Unified Command Routing
depends_on: Sprint 31 (Discord /workspace fix landed)
---

# Sprint 33 — Unified Command Routing: Sprint A

## Sprint Goal

Extract duplicated command logic from Telegram and Discord channels into a shared `internal/commands/` package. **Pure refactoring** — no behavior change, no new commands.

**Problem**: Sprint 31 Discord `/workspace` fix created ~300-400 lines of duplicated code across `telegram/commands_workspace.go` and `discord/commands_workspace.go`. Four functions are byte-for-byte identical or differ only in response delivery. Adding Zalo/MSTeams support would multiply duplication to 4×.

**Success criteria**: All existing tests pass (`make test`), zero behavior change, duplicated functions replaced by shared calls.

---

## CTO Corrections Applied

| # | Correction | How Applied |
|---|-----------|-------------|
| C1 | `Responder` interface for platform-agnostic response | `commands/responder.go` — each channel implements |
| C4 | `CommandMetadata` struct instead of flat `map[string]string` | `commands/metadata.go` — typed fields, optional platform-specific |
| C5 | Unit tests for `ResolveAgentUUID()` | `commands/resolver_test.go` — 3 test cases |
| F1 | `Rail` + `PRURL` fields in `CommandMetadata` for skill routing | `commands/metadata.go` — `Rail: "spec-factory"/"pr-gate"`, `PRURL` for `/review` |

---

## Deliverables

### A1: `commands/responder.go` — Responder Interface (Day 1)

| File | Change |
|------|--------|
| `internal/commands/responder.go` (NEW) | Define `Responder` interface |

```go
// Responder abstracts sending text back to the user.
// Each channel implements this — Telegram wraps telego, Discord wraps discordgo.
type Responder interface {
    Reply(ctx context.Context, chatID string, text string) error
}
```

**CTO C1**: Do NOT pass raw `*telego.SendMessageParams` or `*discordgo.Session` into shared code.

### A2: `commands/metadata.go` — CommandMetadata Struct (Day 1)

| File | Change |
|------|--------|
| `internal/commands/metadata.go` (NEW) | Define `CommandMetadata` struct with `ToMap()` |

```go
// CommandMetadata carries command-specific metadata for bus publishing.
// CTO C4: typed struct instead of flat map, platform-specific fields are optional.
type CommandMetadata struct {
    Command         string // "reset", "stop", "stopall", "spec", "review"
    Platform        string // "telegram", "discord", "zalo", "msteams"
    Rail            string // CTO F1: skill routing — "spec-factory", "pr-gate" (optional)
    PRURL           string // /review only: GitHub PR URL (optional)
    LocalKey        string // Telegram forum: "-1001234567890:topic:42" (optional)
    IsForum         string // Telegram: "true"/"false" (optional)
    MessageThreadID string // Telegram forum topic ID (optional)
}

// ToMap converts to bus metadata. PJM-033-1: skips empty fields to avoid
// Telegram-specific metadata leaking into Discord/Zalo bus messages.
func (m CommandMetadata) ToMap() map[string]string {
    result := map[string]string{"command": m.Command, "platform": m.Platform}
    if m.Rail != "" { result["rail"] = m.Rail }
    if m.PRURL != "" { result["pr_url"] = m.PRURL }
    if m.LocalKey != "" { result["local_key"] = m.LocalKey }
    if m.IsForum != "" { result["is_forum"] = m.IsForum }
    if m.MessageThreadID != "" { result["message_thread_id"] = m.MessageThreadID }
    return result
}

func (m CommandMetadata) ToMap() map[string]string
```

### A3: `commands/resolver.go` — ResolveAgentUUID (Day 1-2)

| File | Change |
|------|--------|
| `internal/commands/resolver.go` (NEW) | Extract `ResolveAgentUUID()` |
| `internal/commands/resolver_test.go` (NEW) | CTO C5: 3 unit tests |

Extracted from:
- `telegram/commands.go:17-34`
- `discord/commands.go:14-29`

**Tests** (CTO C5):
- `TestResolveAgentUUID_ParsesUUID` — direct UUID string input
- `TestResolveAgentUUID_FallbackToStore` — agent key → store lookup via mock
- `TestResolveAgentUUID_EmptyKey` — returns error

### A4: `commands/workspace.go` — WorkspaceCmd (Day 2-3)

| File | Change |
|------|--------|
| `internal/commands/workspace.go` (NEW) | `WorkspaceCmd` with `Get()`, `Set()`, `reloadProjectContext()` |
| `internal/commands/workspace_test.go` (NEW) | Unit tests for workspace logic |

```go
type WorkspaceCmd struct {
    AgentStore store.AgentStore
    Bus        *bus.MessageBus
}

func (w *WorkspaceCmd) Get(ctx context.Context, agentKey string) (string, error)
func (w *WorkspaceCmd) Set(ctx context.Context, agentKey, newPath string) (string, error)
// Private — single home for previously duplicated function
func (w *WorkspaceCmd) reloadProjectContext(ctx context.Context, agentID uuid.UUID, agentKey, workspace string)
```

Extracted from:
- `telegram/commands_workspace.go:23-166` (~144 lines)
- `discord/commands_workspace.go:21-139` (~119 lines)

### A5: `commands/projects.go` — ListProjects (Day 3)

| File | Change |
|------|--------|
| `internal/commands/projects.go` (NEW) | `ListProjects()` function |

Extracted from:
- `telegram/commands_workspace.go:170-246` (~77 lines)
- `discord/commands_workspace.go:142-199` (~58 lines)

### A6: `commands/bus_commands.go` — Shared Bus Publish Helpers (Day 3-4)

| File | Change |
|------|--------|
| `internal/commands/bus_commands.go` (NEW) | `PublishReset()`, `PublishStop()`, `PublishStopAll()` |

Extracted from:
- `telegram/commands.go:203-273` (reset/stop/stopall publish blocks)
- `discord/commands.go:66-113` (same publish blocks)

Uses `CommandMetadata.ToMap()` for metadata — platform-specific fields (local_key, is_forum) populated by caller.

### A7: Channel Integration — Thin Wrappers (Day 4-5)

| File | Change |
|------|--------|
| `internal/channels/telegram/commands_workspace.go` | Replace `handleWorkspace`, `handleProjects`, `reloadProjectContext` with calls to `commands.*` |
| `internal/channels/telegram/commands.go` | Replace `/reset`, `/stop`, `/stopall` publish blocks with `commands.PublishReset/Stop/StopAll` |
| `internal/channels/telegram/responder.go` (NEW) | Implement `commands.Responder` for Telegram |
| `internal/channels/discord/commands_workspace.go` | Replace with thin wrapper calling `commands.*` |
| `internal/channels/discord/commands.go` | Replace `/reset`, `/stop`, `/stopall` with shared publish helpers |
| `internal/channels/discord/responder.go` (NEW) | Implement `commands.Responder` for Discord |

### A8: Cleanup (Day 5)

- Delete duplicated `resolveAgentUUID()` from both channels (now in `commands/resolver.go`)
- Delete duplicated `reloadProjectContext()` from both channels
- Verify `make test` passes
- Verify `make build` passes

---

## Duplication Elimination Summary

| Function | Before | After |
|----------|--------|-------|
| `resolveAgentUUID()` | 2 copies (telegram + discord) | 1 copy (`commands/resolver.go`) |
| `handleWorkspace()` | 2 copies (~144 + ~119 lines) | 1 copy (`commands/workspace.go`) + 2 thin wrappers |
| `reloadProjectContext()` | 2 copies (byte-for-byte identical) | 1 copy (private method on `WorkspaceCmd`) |
| `handleProjects()` | 2 copies (~77 + ~58 lines) | 1 copy (`commands/projects.go`) + 2 thin wrappers |
| `/reset` publish block | 2 copies | 1 copy (`commands/bus_commands.go`) |
| `/stop` publish block | 2 copies | 1 copy (`commands/bus_commands.go`) |
| `/stopall` publish block | 2 copies | 1 copy (`commands/bus_commands.go`) |

**Lines eliminated**: ~300-400 duplicated lines → ~100 lines shared code + ~40 lines thin wrappers

---

## Acceptance Criteria

| # | Criterion | Verification |
|---|-----------|-------------|
| AC-1 | `make test` passes with zero failures | CI green |
| AC-2 | `make build` produces binary | Build succeeds |
| AC-3 | All 7 extracted functions exist in `internal/commands/` | `ls internal/commands/*.go` |
| AC-4 | `Responder` interface defined and implemented by Telegram + Discord | Grep for `commands.Responder` |
| AC-5 | `CommandMetadata` struct with `ToMap()` method | Grep for `CommandMetadata` |
| AC-6 | `ResolveAgentUUID` has 3 unit tests (CTO C5) | `go test ./internal/commands/ -v -run TestResolveAgentUUID` |
| AC-7 | No duplicated `reloadProjectContext()` in channel code | Grep confirms single location |
| AC-8 | Telegram `/workspace`, `/projects`, `/reset`, `/stop`, `/stopall` still work | Manual test on Telegram |
| AC-9 | Discord `/workspace`, `/projects`, `/reset`, `/stop`, `/stopall` still work | Manual test on Discord |

---

## Risk Log

| ID | Risk | Impact | Prob | Mitigation |
|----|------|--------|------|------------|
| R-33-1 | Telegram command regressions from refactoring | HIGH | 10% | `make test` + manual verification of all 30+ commands |
| R-33-2 | Responder interface doesn't fit Telegram's `setThread` pattern | MEDIUM | 20% | Telegram Responder wraps both `SendMessage` + `setThread` internally |
| R-33-3 | Import cycle between `commands/` and `channels/` | LOW | 10% | `commands/` depends on `store` and `bus` only, NOT on `channels/` |

---

## Sprint Sequence

```
Sprint 33 (this) → Sprint 34 (Discord parity) → Sprint 35 (Zalo/MSTeams)
     ↓                    ↓
  Shared package      Uses shared package
  extraction          + factory expansion
```

**Blocker for Sprint 34**: Sprint 33 must land first. Sprint 34's scope depends on what shared infrastructure Sprint 33 delivers.

**Scheduling note** (PJM-033-6): Sprint 32 (Memory Phase 1 / Discord Reactions) is planned for the same window. If same engineer, Sprint 32 runs first → Sprint 33 starts after Sprint 32 completes → Sprint 34 shifts accordingly.

---

## PJM Review — 9.4/10

| # | Finding | Resolution |
|---|---------|-----------|
| PJM-033-1 | `ToMap()` should skip empty fields | ✅ Applied — implementation shown in A2 |
| PJM-033-2 | Single `Reply()` is correct for Responder | No change needed — noted for Sprint 35 |
| PJM-033-6 | Sprint 32/33 scheduling conflict if same engineer | ✅ Scheduling note added |

**Verdict**: APPROVED — 9.4/10.
