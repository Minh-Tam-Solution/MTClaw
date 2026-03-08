# ADR-012: Claude Code Agent Teams Integration

**SPEC ID**: SPEC-0012
**Status**: RESEARCH (Sprint 22 Spike)
**Date**: 2026-03-07
**Deciders**: [@cto]
**Tag**: `adr-012-agent-teams`
**Depends on**: ADR-010 (Bridge), ADR-011 (SOUL-Aware Launch)

---

## Context

Claude Code's experimental Agent Teams API allows multiple Claude Code sessions to coordinate:
- One **lead** session coordinates multiple **teammate** sessions
- Shared task list with self-claiming
- Direct inter-agent messaging (unlike subagents which only report back)
- Requires `CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS=1`

MTClaw's bridge (Sprint 13-17) already manages multiple tmux-based Claude Code sessions with SOUL personas (Sprint 18-21). The question: should MTClaw orchestrate Agent Teams natively, or continue with its existing multi-session management?

### Why This Matters

Current MTClaw multi-session workflow:
```
/cc launch myproject --as pm       -> Session A (PM persona)
/cc launch myproject --as coder    -> Session B (Coder persona)
/cc launch myproject --as reviewer -> Session C (Reviewer persona)
```

Each session is independent. No shared task list, no inter-session messaging. Users manually copy/paste context between sessions via `/cc send`.

Agent Teams would enable coordinated workflows:
```
/cc team create myproject --lead pm --members coder,reviewer
  -> Lead PM decomposes task into subtasks
  -> Coder claims and executes coding subtask
  -> Reviewer claims and reviews output
  -> Lead PM synthesizes results
```

---

## Research Spike Findings

### API Stability Assessment

| Criteria | Finding | Rating |
|----------|---------|--------|
| API maturity | Experimental flag required | RED |
| Documentation completeness | Partial — team creation, task assignment documented; error handling sparse | YELLOW |
| Breaking changes | No stability guarantees while experimental | RED |
| Community adoption | Limited — few production reports | RED |

### Failure Mode Testing

| Scenario | Observed Behavior | Impact on MTClaw |
|----------|-------------------|-----------------|
| Lead session crash | Teammates continue but lose coordination | **HIGH** — orphaned sessions with no orchestrator |
| Teammate session hang | Lead cannot force-terminate teammate | **MEDIUM** — requires tmux-level kill |
| Network drop during messaging | Message delivery not guaranteed | **MEDIUM** — lost inter-session context |
| Graceful shutdown | 30-60s shutdown time observed | **LOW** — acceptable if properly managed |
| File locking on shared task list | Occasional race conditions on rapid claim/unclaim | **MEDIUM** — data integrity concern |

### Tmux Interaction Testing

| Test | Result |
|------|--------|
| Agent Teams inside tmux session | Works — no conflict with MTClaw's tmux bridge |
| Multiple teams in parallel | Works — each team gets separate task files |
| Team + `--dangerously-skip-permissions` | Works — permissions bypass applies to all members |
| Team + `--agent` per member | Works — each member loads its own agent file |
| sendKeys to team member pane | Works — MTClaw can still relay to individual sessions |
| capturePane from team member | Works — output capture unaffected |

### Architecture Comparison

**Option A: MTClaw orchestrates via Agent Teams API**

```
MTClaw Bridge
  |
  v
Claude Code Team (native)
  Lead (PM persona)
    |-- Teammate (Coder)
    |-- Teammate (Reviewer)
```

Pros:
- Native inter-agent messaging (lower latency than sendKeys relay)
- Shared task list (built-in coordination primitive)
- Claude Code handles teammate lifecycle

Cons:
- Experimental API — no stability guarantees
- Duplicate orchestration: MTClaw session manager + Claude Code team coordinator
- Lead crash = total coordination loss (no MTClaw-level recovery)
- Task list is file-based (not integrated with MTClaw's DB/audit)
- `maxTurns` per teammate but no cost guardrail integration

**Option B: MTClaw multi-session (current approach, enhanced)**

```
MTClaw Bridge
  Session A (PM) --sendKeys--> Session B (Coder)
  Session A (PM) --sendKeys--> Session C (Reviewer)
  MTClaw coordinates via DB task table
```

Pros:
- MTClaw controls all lifecycle (crash recovery, admission control, audit)
- Task coordination via `team_tasks` table (queryable, auditable)
- No dependency on experimental API
- Cost guardrails enforced per-session (existing admission control)
- Works with any provider (not Claude Code specific)

Cons:
- Higher latency inter-session messaging (sendKeys + capturePane cycle)
- No native Claude Code coordination primitives
- Manual task routing (user or MTClaw PM agent assigns)

**Option C: Hybrid — MTClaw coordinates, Agent Teams for execution**

```
MTClaw Bridge (coordinator)
  |
  v
Claude Code Team (execution only)
  Lead session managed by MTClaw
  Teammates spawned by MTClaw, coordinated by Claude Code within team
  MTClaw monitors via capturePane + stop hooks
```

Pros:
- MTClaw retains control (audit, admission, cost)
- Agent Teams provides low-latency coordination within execution
- Graceful degradation: if teams API breaks, fall back to Option B

Cons:
- Complex integration surface (two coordination layers)
- Unclear error boundaries (who handles teammate crash?)
- More code to maintain

---

## Go/No-Go Criteria

### GO Criteria (ALL must be met for production adoption)

1. **API stability**: Agent Teams API exits experimental status OR is stable for 4+ weeks of continuous testing
2. **Crash recovery**: Lead crash must not orphan teammate sessions (either API handles it or MTClaw can detect/recover)
3. **Shutdown time**: Graceful shutdown completes in <30s consistently
4. **No orphaned processes**: All teammate processes terminate when team is killed
5. **Task list integrity**: No data loss under concurrent claim/unclaim operations
6. **Audit integration**: MTClaw can observe all team events (task assignments, messages, completions) for audit logging

### NO-GO Fallback

If any GO criterion fails, implement enhanced Option B:
- Add `team_tasks` table to PostgreSQL schema
- Add `/cc team create` command that creates coordinated sessions (no Agent Teams API)
- PM session routes tasks to other sessions via sendKeys
- MTClaw monitors completion via capturePane polling

---

## Decision

**DECISION: NO-GO for production Agent Teams integration (2026-03-07)**

**Rationale**: The Agent Teams API remains experimental with no stability timeline from Anthropic. Key blockers:
1. Lead crash recovery is not handled — fails GO criterion #2
2. File-based task list has observed race conditions — fails GO criterion #5
3. No event hooks for audit integration — fails GO criterion #6

**Recommended path**: Enhanced Option B (MTClaw multi-session coordination)
- Sprint 24+: Add `team_tasks` table + `/cc team` commands
- Monitor Agent Teams API maturity quarterly
- Re-evaluate when API exits experimental status

**Fallback preserved**: If Agent Teams stabilizes before Sprint 30, Option C (hybrid) is architecturally compatible — MTClaw's session manager and agent installer already support per-member persona injection via `--agent` flag.

---

## Consequences

### Positive
- No dependency on experimental API for production features
- Full audit trail for all inter-session coordination
- Provider-agnostic team coordination (works with future Cursor/Codex adapters)
- Simpler error handling (MTClaw owns all lifecycle)

### Negative
- Higher latency inter-session messaging vs native Agent Teams
- No native task list coordination (must build in MTClaw)
- May need to retrofit if Agent Teams becomes the standard approach

### Neutral
- Existing Sprint 18-21 code (SOUL injection, role defaults, intelligence envelope) is fully compatible with both approaches
- The `--agent` per-member pattern works identically whether sessions are independent or in a team

---

## References

- ADR-010: Claude Code Terminal Bridge (foundation)
- ADR-011: SOUL-Aware Bridge Launch (persona injection)
- Claude Code docs: Agent Teams (experimental)
- MTClaw Sprint 13-17: Bridge implementation (150+ tests)
- MTClaw Sprint 18-21: Intelligence upgrade (SOUL injection, role defaults)
