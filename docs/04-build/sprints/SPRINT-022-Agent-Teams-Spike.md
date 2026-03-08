---
sprint: 22
title: Agent Teams Research Spike
status: PLANNED
date: 2026-03-07
version: "1.0.0"
author: "[@pm] + [@architect]"
framework: SDLC Enterprise Framework 6.1.1
tier: STANDARD
---

# Sprint 22 — Agent Teams Research Spike

**Sprint**: 22 of 23
**Phase**: 4 (Bridge Intelligence)
**Duration**: 2 days (spike) + 1 day (ADR)
**Owner**: [@coder] (spike) + [@architect] (ADR)
**Points**: ~3
**Depends on**: Sprint 18 (tmux bridge working)
**Gate**: ADR-026 with GO/NO-GO decision and evidence

---

## Sprint Goal

**Validate Claude Code's experimental agent teams API before building production features.**

### Why Research, Not Production

Agent teams API is experimental (`CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1`) with known limitations:
- No session resumption
- Task status lag
- Shutdown slow (orphaned processes possible)

Building production features on experimental APIs generates firefighting.

---

## Deliverables

### 2-Day Research Spike

1. Test `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1` stability
2. Test failure modes: lead crash, teammate hang, network drop
3. Test shutdown behavior: graceful shutdown timing, orphaned processes
4. Test tmux interaction: does in-process mode work inside MTClaw's tmux sessions?
5. Test task list coordination: file locking, race conditions

### ADR-026: Agent Teams Integration (Day 3)

GO criteria:
- API stable for 2+ weeks of testing
- Graceful shutdown < 30s
- No orphaned processes after kill

NO-GO fallback:
- Use MTClaw's existing multi-session management + `team_tasks` table
- Document limitations, defer until Claude Code stabilizes

### Output Files

```
docs/02-design/01-ADRs/ADR-026-Agent-Teams-Integration.md  -- CREATE (GO/NO-GO + evidence)
```

No production code changes in this sprint.

---

## If GO -> Schedule Sprint 24

Full `/cc team` implementation with:
- `/cc team create "Fix login module" --members coder,tester`
- Shared task list coordination
- Team status dashboard in `/cc sessions`

## If NO-GO -> Document and Defer

Wait for Claude Code to stabilize agent teams API. Re-evaluate quarterly.
