---
sprint: 34
title: "Unified Command Routing — Sprint B: Discord Command Parity"
status: PLANNING
start_date: 2026-03-31
end_date: 2026-04-11
lead: "@pm (plan) → @coder (implementation)"
framework: SDLC Enterprise Framework 6.1.2
adr: Architecture Review — Unified Command Routing
depends_on: Sprint 33 (shared commands package)
---

# Sprint 34 — Unified Command Routing: Sprint B

## Sprint Goal

Bring Discord to **first-class command parity** with Telegram. After this sprint, Discord users can use all core commands (governance rails, workspace, specs, tasks, writers) — not just `/help`, `/workspace`, `/reset`, `/stop`, `/stopall`.

**Effort estimate**: ~8-10 days (PJM-CMD-3 revised estimate). Sprint B is larger than a typical command sprint due to store wiring expansion + formatting extraction.

**Prerequisite**: Sprint 33 (shared `internal/commands/` package) must be complete.

---

## CTO Corrections Applied

| # | Correction | How Applied |
|---|-----------|-------------|
| C2 | Discord `FactoryWithStores` expanded to `(agentStore, teamStore, specStore)` | B1: factory + struct expansion |
| C3 | `/writers`, `/addwriter`, `/removewriter` included in parity scope | B6: writers commands |
| F1 | `Rail` field in `CommandMetadata` for skill routing (`spec-factory`, `pr-gate`) | B3: governance command metadata |
| F2 | `/addwriter` Discord mention parsing design with concrete regex + resolve logic | B6: mention parsing section |
| F3 | Replace fragile `gateway.go:820` line reference with code pattern | B1: `instanceLoader.RegisterFactory("discord", ...)` |

---

## Deliverables

### B1: Discord Store Wiring Expansion (Day 1-2)

| File | Change |
|------|--------|
| `internal/channels/discord/discord.go` | Add `teamStore store.TeamStore`, `specStore store.SpecStore` fields to `Channel` struct. Update `New()` signature. |
| `internal/channels/discord/factory.go` | Expand `FactoryWithStores(agentStore, teamStore, specStore)` to match Telegram |
| `cmd/gateway.go` (`instanceLoader.RegisterFactory("discord", ...)`) | Update registration: `discord.FactoryWithStores(managedStores.Agents, managedStores.Teams, managedStores.Specs)` |
| `internal/channels/discord/discord_test.go` | Update all `New()` calls with new signature |

### B2: Shared Bus Publish for `/spec` and `/review` (Day 1-2) — PJM-033-3

**Sequencing fix**: B2 must land before B3, because B3 (governance commands) calls `PublishSpec()`/`PublishReview()`.

| File | Change |
|------|--------|
| `internal/commands/bus_commands.go` | Add `PublishSpec()`, `PublishReview()` shared helpers |

### B3: Bus-Routed Governance Commands (Day 2-3)

Add `/spec` and `/review` to Discord — bus-routed, minimal code. **Depends on B2** (`PublishSpec`/`PublishReview` helpers).

| File | Change |
|------|--------|
| `internal/channels/discord/commands.go` | Add `/spec` and `/review` cases in `handleBotCommand` |

```go
case "/spec":
    taskText := strings.TrimSpace(text[len("/spec"):])
    if taskText == "" {
        responder.Reply(ctx, chatID, "Usage: /spec <requirement description>")
        return true
    }
    responder.Reply(ctx, chatID, "Generating spec...")
    commands.PublishSpec(c.Bus(), c.Name(), senderID, chatID, c.AgentID(), peerKind,
        taskText, commands.CommandMetadata{Command: "spec", Platform: "discord", Rail: "spec-factory"})
    return true

case "/review":
    prURL := strings.TrimSpace(text[len("/review"):])
    if prURL == "" || !strings.Contains(prURL, "/pull/") {
        responder.Reply(ctx, chatID, "Usage: /review <github_pr_url>")
        return true
    }
    responder.Reply(ctx, chatID, "Reviewing PR...")
    commands.PublishReview(c.Bus(), c.Name(), senderID, chatID, c.AgentID(), peerKind,
        prURL, commands.CommandMetadata{Command: "review", Platform: "discord", Rail: "pr-gate", PRURL: prURL})
    return true
```

**CTO F1**: `Rail` field is critical — it triggers skill routing in the agent loop. Without `Rail: "spec-factory"` / `Rail: "pr-gate"`, governance rails won't activate on Discord.

### B4: Static Text Commands (Day 3)

Add `/teams` and `/status` to Discord.

| File | Change |
|------|--------|
| `internal/channels/discord/commands.go` | Add `/teams` and `/status` cases |

### B5: DB-Backed Read Commands (Day 3-5)

Add spec and task listing commands. Uses shared formatting from `internal/commands/`.

| File | Change |
|------|--------|
| `internal/commands/specs.go` (NEW) | Extract `FormatSpecList()`, `FormatSpecDetail()` from `telegram/commands_specs.go` |
| `internal/commands/tasks.go` (NEW) | Extract `FormatTaskList()`, `FormatTaskDetail()` from `telegram/commands_tasks.go` |
| `internal/channels/discord/commands_specs.go` (NEW) | `/spec_list`, `/spec_detail` using shared formatters |
| `internal/channels/discord/commands_tasks.go` (NEW) | `/tasks`, `/task_detail` using shared formatters |
| `internal/channels/telegram/commands_specs.go` | Refactor to use shared `commands.FormatSpecList/Detail` |
| `internal/channels/telegram/commands_tasks.go` | Refactor to use shared `commands.FormatTaskList/Detail` |

### B6: Writers Commands (Day 5-7) — CTO C3

Add group file writer management to Discord.

| File | Change |
|------|--------|
| `internal/commands/writers.go` (NEW) | Extract `ListWriters()`, `AddWriter()`, `RemoveWriter()` logic from `telegram/commands_writers.go` |
| `internal/channels/discord/commands_writers.go` (NEW) | `/writers`, `/addwriter`, `/removewriter` |

**Discord adaptation** (PJM-033-5 + CTO F2): Telegram uses `message.ReplyToMessage.From` for `/addwriter`. Discord equivalent: `/addwriter @user` (mention-based targeting).

**Mention parsing design** (~10 lines in `discord/commands_writers.go`):
```go
// parseMention extracts user ID from Discord mention format: <@123456789> or <@!123456789>
func parseMention(text string) (userID string, err error) {
    re := regexp.MustCompile(`<@!?(\d+)>`)
    matches := re.FindStringSubmatch(text)
    if len(matches) < 2 {
        return "", fmt.Errorf("no valid @mention found — use /addwriter @user")
    }
    return matches[1], nil
}
```
- **Resolve display name**: Use `discordgo.Session.GuildMember(guildID, userID)` to get `Member.Nick` (guild nickname) or fall back to `Member.User.Username`.
- **Fallback if not in cache**: `GuildMember()` makes a REST API call if member is not in `Session.State` cache — no additional fallback needed.
- This parsing lives in `discord/commands_writers.go`, not in the shared `commands/writers.go`.

### B7: Update Help Text + Tests (Day 8-9)

| File | Change |
|------|--------|
| `internal/channels/discord/commands.go` | Update `/help` text with all new commands |
| `internal/channels/discord/discord_test.go` | Add tests for new command dispatch |

---

## Command Parity Matrix (After Sprint 34)

| Command | Telegram | Discord | Routing |
|---------|----------|---------|---------|
| `/help` | ✅ | ✅ | Channel-local |
| `/status` | ✅ | ✅ | Channel-local |
| `/teams` | ✅ | ✅ | Channel-local |
| `/workspace` | ✅ | ✅ | Channel-local + DB |
| `/projects` | ✅ | ✅ | Channel-local |
| `/reset` | ✅ | ✅ | Bus-routed |
| `/stop` | ✅ | ✅ | Bus-routed |
| `/stopall` | ✅ | ✅ | Bus-routed |
| `/spec` | ✅ | ✅ | Bus-routed → PM SOUL |
| `/review` | ✅ | ✅ | Bus-routed → reviewer SOUL |
| `/spec_list` | ✅ | ✅ | Channel-local (DB read) |
| `/spec_detail` | ✅ | ✅ | Channel-local (DB read) |
| `/tasks` | ✅ | ✅ | Channel-local (DB read) |
| `/task_detail` | ✅ | ✅ | Channel-local (DB read) |
| `/writers` | ✅ | ✅ | Channel-local (DB read) |
| `/addwriter` | ✅ | ✅ | Channel-local + DB mutation |
| `/removewriter` | ✅ | ✅ | Channel-local + DB mutation |
| `/cc *` | ✅ | ❌ | Telegram-only (bridge mgr) |

**Discord gap after Sprint 34**: Only `/cc *` (Claude Code bridge) — intentionally Telegram-only per CTO decision.

---

## Acceptance Criteria

| # | Criterion | Verification |
|---|-----------|-------------|
| AC-1 | `make test` passes with zero failures | CI green |
| AC-2 | Discord `/spec <desc>` routes to PM SOUL and generates spec | Manual test + `TestHandleBotCommand_Spec` unit test |
| AC-3 | Discord `/review <pr_url>` routes to reviewer SOUL | Manual test + `TestHandleBotCommand_Review` unit test |
| AC-4 | Discord `/spec_list` shows specs from DB | Manual test + `TestSpecList_FormatsOutput` unit test |
| AC-5 | Discord `/tasks` shows team tasks from DB | Manual test + `TestTaskList_FormatsOutput` unit test |
| AC-6 | Discord `/writers` lists group file writers | Manual test + `TestWriters_ListOutput` unit test |
| AC-7 | Discord `/addwriter @user` adds writer (mention parsed) | Manual test + `TestAddWriter_ParsesMention` unit test |
| AC-8 | Discord `/help` lists all new commands | Manual test |
| AC-9 | Telegram commands unchanged (no regressions) | `go test ./internal/channels/telegram/ -v` — all existing tests pass |
| AC-10 | Shared formatters in `internal/commands/` used by both channels | `grep -r 'commands\.Format' internal/channels/` confirms both channels use shared formatters |
| AC-11 | `PublishSpec`/`PublishReview` shared helpers tested | `go test ./internal/commands/ -v -run TestPublish` |

**PJM-033-4**: AC-2 through AC-7 now include both automated unit tests AND manual verification. Automated tests validate dispatch and formatting logic; manual tests validate end-to-end UX.

---

## Risk Log

| ID | Risk | Impact | Prob | Mitigation |
|----|------|--------|------|------------|
| R-34-1 | Sprint 33 not complete, blocking Sprint 34 | HIGH | 15% | Sprint 33 is pure refactoring, low risk of delay |
| R-34-2 | Discord factory signature change breaks existing deployments | MEDIUM | 10% | Managed mode only — factory wired at startup, not runtime |
| R-34-3 | `/addwriter` Discord UX confusing (mention vs reply-to) | LOW | 25% | Document clearly in `/help` text |
| R-34-4 | Formatting extraction from Telegram creates regressions | MEDIUM | 15% | Keep Telegram formatting unchanged; shared functions match exactly |

---

## PJM Review — 8.9/10 → 9.3/10 (Corrections Applied)

| # | Finding | Resolution |
|---|---------|-----------|
| PJM-033-3 | **HIGH**: B6 (`PublishSpec`/`PublishReview`) was scheduled Day 7, but B2 (governance commands) depends on it Day 2-3 | ✅ Fixed — B6 renumbered to B2 (Day 1-2), all subsequent deliverables renumbered |
| PJM-033-4 | **MEDIUM**: AC-2 through AC-9 were all "Manual test" — no automated verification | ✅ Fixed — AC-2 through AC-7 now include unit test names + AC-11 added for PublishSpec/Review |
| PJM-033-5 | **LOW**: `/addwriter` Discord mention parsing (~10 lines) unaccounted in effort estimate | ✅ Fixed — B6 (writers) now documents mention parsing location and scope |

**Verdict**: APPROVED — 9.3/10 (after corrections).

---

## Future: Sprint 35 (Sprint D — Zalo/MSTeams)

Once Sprint 33 + 34 land, adding commands to Zalo/MSTeams is straightforward:
1. Implement `commands.Responder` for each channel
2. Add `handleBotCommand()` with `switch` statement
3. Call shared `commands.*` functions
4. Estimated effort: ~0.5 sprint per channel
