---
sprint: 19
title: Session Intelligence Envelope
status: PLANNED
date: 2026-03-07
version: "1.0.0"
author: "[@pm] + [@architect]"
framework: SDLC Enterprise Framework 6.1.1
tier: STANDARD
---

# Sprint 19 — Session Intelligence Envelope

**Sprint**: 19 of 23
**Phase**: 4 (Bridge Intelligence — ADR-011)
**Duration**: 4 days
**Owner**: [@coder] + [@pm]
**Points**: ~7
**Depends on**: Sprint 18 (agentRole, soulContentHash fields exist)
**Gate**: `/cc info <session>` shows intelligence metadata (role, strategy, hashes — NOT full SOUL body)

---

## Sprint Goal

**Define the `SessionIntelligenceEnvelope` contract and expose intelligence state via `/cc info`.**

Contract only — no runtime intelligence injection beyond Sprint 18's persona.

### Key Outcomes

1. `SessionIntelligenceEnvelope` type with `PersonaEnvelope` (Sprint 19) + extensible slots
2. `TurnContext` struct defined (sprint goals, blockers, fix hints) — serialization only, no injection
3. `/cc info <session>` Telegram command shows intelligence metadata
4. Envelope attached to `BridgeSession` and populated in `CreateSession`
5. ~12 new tests

---

## Architecture

### New Files

```
internal/claudecode/
  intelligence.go               -- SessionIntelligenceEnvelope, PersonaEnvelope, TurnContext
  intelligence_test.go          -- ~6 tests
```

### Modified Files

```
internal/claudecode/
  types.go                      -- Add Intelligence field to BridgeSession
  session_manager.go            -- Populate envelope in CreateSession
internal/channels/telegram/
  commands_cc.go                -- /cc info command
```

### Key Types

```go
type SessionIntelligenceEnvelope struct {
    Persona *PersonaEnvelope `json:"persona,omitempty"`
    // Future (Sprint 20+): Skills, Context, Brain
}

type PersonaEnvelope struct {
    AgentRole         string `json:"agent_role"`
    SoulContentHash   string `json:"soul_content_hash"`
    PersonaSourceHash string `json:"persona_source_hash"`
    PersonaSource     string `json:"persona_source"` // "agent_file" | "append_prompt" | "bare"
    Strategy          string `json:"strategy"`        // "A" | "B" | "C"
}

type TurnContext struct {
    SprintGoals []string `json:"sprint_goals,omitempty"`
    Blockers    []string `json:"blockers,omitempty"`
    FixHints    []string `json:"fix_hints,omitempty"`
}
```

### CTO-M2 Compliance

NO commented-out types for Sprint 21+. Only `PersonaEnvelope` defined inline. Future envelope slots added when their sprint starts.

### `/cc info` Security (CTO Sprint 19 caveat)

`/cc info` shows metadata only — role, strategy, hashes, persona source. Does NOT dump `SoulContent` body into Telegram message. Full SOUL content available only via `install-agents` output or direct file read.

---

## NOT in Sprint 19

| Item | Reason | Sprint |
|------|--------|--------|
| TurnContext injection (sendKeys) | Sprint 20B | 20B |
| Skills preloading | Sprint 20A | 20A |
| Brain/Context envelope slots | YAGNI — add when Sprint 20+ starts | 20+ |
| `--model` per-role | Sprint 21 | 21 |
