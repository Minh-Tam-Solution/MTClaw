---
sprint: 23
title: Provider Persona Projection Research
status: PLANNED
date: 2026-03-07
version: "1.0.0"
author: "[@pm] + [@architect]"
framework: SDLC Enterprise Framework 6.1.1
tier: STANDARD
---

# Sprint 23 — Provider Persona Projection Research

**Sprint**: 23 of 23
**Phase**: 4 (Bridge Intelligence)
**Duration**: 3 days
**Owner**: [@coder] (POC) + [@architect] (capability matrix)
**Points**: ~4
**Depends on**: Sprint 18 (ProviderAdapter pattern)
**Gate**: ADR-027 with provider capability matrix, 1 POC adapter working

---

## Sprint Goal

**Map each provider's native persona mechanism to MTClaw's SOUL concept. Do NOT force a unified abstraction.**

### Key Insight (from Synthesis — Doc 16)

Provider parity is an illusion. Four providers = four different integration surfaces:

| Provider | Persona Mechanism | Knowledge Mechanism | Rules Mechanism |
|----------|------------------|-------------------|-----------------|
| Claude Code | `.claude/agents/*.md` | `.claude/skills/` | `CLAUDE.md` + hooks |
| Cursor | `.cursor/rules` | Rules system | Same as persona |
| Codex CLI | `AGENTS.md` | Config + approvals | Config-based |
| Gemini CLI | `GEMINI.md` | Extensions + commands | `GEMINI.md` |

A single `InjectSOUL(provider, content)` abstraction would either be too generic to be useful or too Claude-specific for others.

---

## Deliverables

1. **Provider capability matrix** — verified against current docs
2. **Projection contract per provider** — how SOUL maps to each native format
3. **One POC adapter** — likely Cursor (`CursorAdapter` with `.cursor/rules` generation)
4. **Honest assessment** — which providers support SOUL-equivalent injection, which don't

### Output Files

```
docs/02-design/01-ADRs/ADR-027-Provider-Persona-Projection.md  -- CREATE
internal/claudecode/provider_cursor.go                           -- CREATE (POC only)
internal/claudecode/provider_cursor_test.go                      -- CREATE (~4 tests)
```

### NOT Deliverables

- Full adapters for all 4 providers
- Claims of parity where none exists
- A single `InjectSOUL(provider, content)` abstraction
- `install-agents` multi-provider support

---

## Cumulative Sprint Output Summary (Sprint 18-23)

| Sprint | New Files | Modified Files | New Tests | New LOC |
|--------|-----------|---------------|-----------|---------|
| 18 | 3 | 6 | ~28 | ~350 |
| 19 | 2 | 3 | ~12 | ~200 |
| 20A | 2 | 2 | ~6 | ~150 |
| 20B | 2 | 3 | ~6 | ~150 |
| 21 | 0 | 4 | ~8 | ~120 |
| 22 | 1 | 0 | 0 | ~50 (ADR) |
| 23 | 3 | 0 | ~4 | ~200 |
| **Total** | **13** | **18** | **~64** | **~1220** |

Combined with Sprint 13-17: **~235 total bridge tests**, production-grade SOUL-aware multi-provider bridge.
