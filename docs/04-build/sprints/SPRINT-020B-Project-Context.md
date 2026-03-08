---
sprint: "20B"
title: Project Context
status: PLANNED
date: 2026-03-07
version: "1.0.0"
author: "[@pm] + [@architect]"
framework: SDLC Enterprise Framework 6.1.1
tier: STANDARD
---

# Sprint 20B — Project Context

**Sprint**: 20B of 23
**Phase**: 4 (Bridge Intelligence)
**Duration**: 3 days
**Owner**: [@coder] + [@pm]
**Points**: ~5
**Depends on**: Sprint 20A (skills exist), Sprint 19 (TurnContext type defined)
**Gate**: `mtclaw bridge init-project` generates CLAUDE.md, `/cc context set` stores turn context

---

## Sprint Goal

**Project-level context injection for bridge sessions — CLAUDE.md generator and turn-time context.**

### Key Outcomes

1. `mtclaw bridge init-project <path>` generates concise project CLAUDE.md (<100 lines)
2. `/cc context set "Sprint goal: Fix login bug"` stores TurnContext per session
3. Turn-time context injection via `--append-system-prompt-file` rotating file (NOT sendKeys prefix — CTO-M3)
4. ~6 new tests

---

## Architecture

### New Files

```
internal/claudecode/
  claudemd_generator.go         -- Generate project CLAUDE.md from project analysis
  claudemd_generator_test.go    -- ~3 tests
```

### Modified Files

```
cmd/bridge.go                   -- init-project subcommand
internal/claudecode/
  session_manager.go            -- Prepend TurnContext to rotating context file
internal/channels/telegram/
  commands_cc.go                -- /cc context set/get commands
```

### Key Design Decisions

**CLAUDE.md overwrite protection** (CTO Sprint 20 caveat):
- Detect existing CLAUDE.md — warn, do not overwrite
- Use `--force` to overwrite
- Or append `# MTClaw Bridge Context` section to existing file

**Turn-time context via rotating file** (CTO-M3 resolution):
- NOT via sendKeys prefix (would trigger input sanitizer)
- `--append-system-prompt-file` points to `~/.mtclaw/sessions/{id}/context.md`
- Updated by `/cc context set` — Claude Code re-reads on next turn
- Content: sprint goals, blockers, fix hints from `TurnContext` struct

---

## NOT in Sprint 20B

| Item | Reason | Sprint |
|------|--------|--------|
| Role-aware capability defaults | Sprint 21 | 21 |
| Agent teams | Sprint 22 spike | 22 |
| Multi-provider projection | Sprint 23 research | 23 |
