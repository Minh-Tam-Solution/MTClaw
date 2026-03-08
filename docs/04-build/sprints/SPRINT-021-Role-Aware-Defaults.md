---
sprint: 21
title: Role-Aware Defaults
status: PLANNED
date: 2026-03-07
version: "1.0.0"
author: "[@pm] + [@architect]"
framework: SDLC Enterprise Framework 6.1.1
tier: STANDARD
---

# Sprint 21 — Role-Aware Defaults

**Sprint**: 21 of 23
**Phase**: 4 (Bridge Intelligence)
**Duration**: 3 days
**Owner**: [@coder] + [@pm]
**Points**: ~5
**Depends on**: Sprint 18 (agentRole), Sprint 20A (skills/tools)
**Gate**: `/cc launch --as coder` starts at `patch` mode, `/cc risk read` overrides

---

## Sprint Goal

**SOUL role influences default risk mode as UX convenience. NOT a security boundary.**

### Critical Constraint (from Synthesis Review)

Agent file tool restrictions are UX convenience, NOT security. Bridge capability model (D2: InputMode x ToolPolicy x OutputPolicy) remains the ONLY security boundary. Agent file `permissionMode` does NOT override bridge Layer 1.

### Key Outcomes

1. Role->RiskMode default mapping in `bridge_agent_templates.json`
2. `CreateSession` applies defaults based on SOUL category
3. `--allowedTools` passed to ClaudeCodeAdapter as noise-reduction (NOT security gate)
4. `/cc risk` command always overrides role defaults
5. Guard: agent file `permissionMode: bypassPermissions` ignored by bridge
6. ~8 new tests

---

## Role->Default Mapping

| SOUL Category | Default RiskMode | Default ToolPolicy | Nature |
|---------------|-----------------|-------------------|--------|
| `advisor` (cto, cpo, ceo) | `read` | `observe` | UX default |
| `executor` (coder, devops, fullstack) | `patch` | `patch_allowed` | UX default |
| `router` (assistant) | `read` | `observe` | UX default |
| `business` (dev, sales, cs) | `read` | `observe` | UX default |

### Pre-Sprint Verification Required (CTO-99)

Before implementing `--allowedTools` in this sprint, verify interaction with `--agent` flag:
- Does Claude Code intersect or override tool lists?
- If conflict, skip `--allowedTools` and rely on agent file `tools:` field only

---

## Modified Files

```
internal/claudecode/
  bridge_agent_templates.json   -- Add default_risk_mode per category
  bridge_policy.go              -- Role-aware defaults + explicit guard
  provider.go                   -- Pass --allowedTools in LaunchOpts (if verified safe)
  session_manager.go            -- Apply role defaults in CreateSession
```

---

## NOT in Sprint 21

| Item | Reason | Sprint |
|------|--------|--------|
| Agent teams | Sprint 22 spike | 22 |
| Multi-provider | Sprint 23 research | 23 |
| Security claims about agent file tools | NEVER — D2 is security boundary | N/A |
