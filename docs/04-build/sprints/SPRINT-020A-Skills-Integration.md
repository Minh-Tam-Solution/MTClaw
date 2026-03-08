---
sprint: "20A"
title: Skills Integration
status: PLANNED
date: 2026-03-07
version: "1.0.0"
author: "[@pm] + [@architect]"
framework: SDLC Enterprise Framework 6.1.1
tier: STANDARD
---

# Sprint 20A — Skills Integration

**Sprint**: 20A of 23
**Phase**: 4 (Bridge Intelligence)
**Duration**: 3 days
**Owner**: [@coder] + [@pm]
**Points**: ~5
**Depends on**: Sprint 19 (envelope exists to track injected skills)
**Gate**: `.claude/skills/sdlc-framework/SKILL.md` generated, agent templates reference skills

---

## Sprint Goal

**SDLC Framework knowledge available to bridge sessions via Claude Code's native skills system.**

### Key Outcomes

1. `.claude/skills/sdlc-framework/SKILL.md` generated — condenses SDLC 6.1.1 key rules
2. `install-agents` extended to install skills alongside agent files
3. Agent templates reference `sdlc-framework` skill for executor roles
4. Skill content under 5000 chars (Claude Code skill budget)
5. ~6 new tests

---

## Architecture

### New Files

```
internal/claudecode/
  skills_generator.go           -- Generate SDLC framework skill content
  skills_generator_test.go      -- ~4 tests
```

### Modified Files

```
cmd/bridge.go                   -- install-agents adds skills
internal/claudecode/
  bridge_agent_templates.json   -- Add skills field to categories
```

### Skill Content Design

SDLC 6.1.1 condensed into skill format:
- Gate definitions (G0.1 through G5)
- Evidence requirements per gate
- SOUL delegation rules (SE4A executor vs SE4H advisor)
- Quality standards (coverage targets, security checklist)

**Budget**: Under 5000 chars. Iterative content design — test with real Claude Code sessions.

---

## NOT in Sprint 20A

| Item | Reason | Sprint |
|------|--------|--------|
| CLAUDE.md generator | Sprint 20B | 20B |
| Turn-time context injection | Sprint 20B | 20B |
| `/cc context set` command | Sprint 20B | 20B |
