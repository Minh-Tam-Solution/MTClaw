# ADR-004: SOUL Implementation

**SPEC ID**: SPEC-0004
**Status**: ACCEPTED
**Date**: 2026-03-02
**Deciders**: [@cto], [@cpo], [@pm]

---

## Context

MTClaw has 16 SOULs (12 SDLC + 4 MTS business). Need to define storage, loading, caching, and drift control mechanisms.

## Decision

### Format

YAML frontmatter + Markdown body (EndiorBot pattern):

```yaml
---
soul: pm
version: "1.0.0"
category: SE4A
type: executor
description: "Product Manager — requirements, specs, user stories"
active_default: true
rails: ["spec-factory"]
---

# PM SOUL

You are a Product Manager...
[system prompt content]
```

### Storage: Git Files = Source of Truth

```
docs/08-collaborate/souls/
  SOUL-pm.md
  SOUL-architect.md
  SOUL-coder.md
  ... (16 files total)
```

**Why Git, not database**:
- Version controlled (diff, blame, history)
- Reviewed via PR (governance by default)
- Portable (copy between projects)
- No migration needed

### Data Flow

```
1.  Source of truth: Git files → docs/08-collaborate/souls/SOUL-*.md
1b. Sprint 3:      Git files seeded to PostgreSQL via migration (SeedToStore pattern)
                    → 16 agents + 48 agent_context_files (SOUL.md, IDENTITY.md, AGENTS.md)
2.  Build-time:    make souls-validate → checks YAML frontmatter
3.  Startup:       GoClaw LoadFromStore() loads predefined agents from DB → memory
4.  Runtime:       Request arrives → session → agent → BuildSystemPrompt() injects SOUL
5.  Reload:        Re-run migration or API update → DB updated → next request loads fresh
```

**Note**: Git remains source of truth for SOUL content authoring (reviewed via PR). The DB seeding migration copies Git content into PostgreSQL for GoClaw's `LoadFromStore()` loading path. This is GoClaw's native pattern for "predefined" agent types.

### Memory Cache Structure

```go
type Soul struct {
    Role        string            // "pm", "coder", etc.
    Version     string            // "1.0.0"
    Category    string            // "SE4A", "SE4H", "Router", "MTS"
    Type        string            // "executor", "advisor", "router", "business"
    Active      bool              // default active or on-demand
    Rails       []string          // associated governance rails
    Content     string            // full markdown body
    Checksum    string            // SHA-256 of file content
    LoadedAt    time.Time
}

// In-memory cache
var souls = make(map[string]*Soul) // key = role name
```

### GoClaw Schema Verification (Sprint 1 Task)

Check if GoClaw has relevant tables:
- `user_agent_profiles` — may map to SOUL assignment
- `agents` — may map to SOUL definition
- `agent_teams` — may map to SOUL grouping

Document findings in `docs/02-design/mtclaw-schema-analysis.md`.

### SOUL Drift Control

| Mechanism | Purpose | Implementation |
|-----------|---------|----------------|
| **Checksum** | Detect file↔cache mismatch | SHA-256 stored in cache, compared on reload |
| **Version field** | Track intentional changes | YAML frontmatter `version: "1.0.0"` |
| **Startup check** | Ensure cache = disk | Compare checksums on every startup |
| **SOUL test suite** | Validate behavior | Input → expected output pattern per SOUL |
| **make souls-validate** | Build-time guard | Check YAML frontmatter presence and required fields |

### SOUL Test Suite (Sprint 2+)

```yaml
# tests/souls/pm_test.yaml
soul: pm
tests:
  - input: "Create a user story for login"
    expect_contains: ["As a", "I want", "So that"]
    expect_not_contains: ["TODO", "placeholder"]
  - input: "What is your role?"
    expect_contains: ["Product Manager", "requirements"]
```

## Consequences

### Positive
- 16 SOULs available Day 1 (just copy markdown files)
- Git-native versioning and review
- Lightweight cache (~1MB for 16 files)
- Drift detection prevents stale personas
- Build-time validation catches broken frontmatter

### Negative
- No hot-reload without SIGHUP or file watch
- SOUL content in Git = visible to all repo contributors
- No per-tenant SOUL customization in Phase 1 (Git = global)

### Phase 2 Extension
- Per-tenant SOUL overrides stored in PostgreSQL
- SOUL A/B testing (which prompt performs better)
- SOUL analytics (which SOULs are most used per tenant)

---

## References
- [ROLE_TOOL_MATRIX](../../08-collaborate/01-SDLC-Compliance/ROLE_TOOL_MATRIX.md)
- [ADR-002: Three-System Architecture](SPEC-0002-ADR-002-Three-System-Architecture.md)
- EndiorBot SOUL format: `/home/nqh/shared/EndiorBot/docs/reference/templates/souls/`
- SOUL files: `docs/08-collaborate/souls/SOUL-*.md`
