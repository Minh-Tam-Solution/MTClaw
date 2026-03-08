# ADR-001: GoClaw Runtime Adoption

**SPEC ID**: SPEC-0001
**Status**: ACCEPTED
**Date**: 2026-03-02
**Deciders**: [@cto], [@pm]

---

## Context

MTClaw needs a runtime that supports multi-tenant PostgreSQL, agent orchestration, and production-grade performance. Options considered:

1. **Extend MTS-OpenClaw (TypeScript/Node.js)** — Current platform, but lacks multi-tenant DB, governance rails
2. **Build from scratch (Go)** — Full control, high effort
3. **Adopt GoClaw (Go fork)** — MIT-licensed Go rewrite of OpenClaw with native PostgreSQL multi-tenant

## Decision

**Adopt GoClaw** as the MTClaw runtime.

- Port GoClaw source (upstream MIT license verified — see `docs/00-foundation/mtclaw-license-verification.md`)
- Build as single Go binary (~25MB)
- Leverage native PostgreSQL multi-tenant, agent teams, 13+ LLM providers
- Customize for MTClaw governance rails
- **MTClaw is NOT OSS** — proprietary internal platform for MTS. GoClaw upstream is MIT, our fork is internal use only.

## Go Competency Mitigation Plan

**Current state**: Team is TypeScript-primary. Go is new competency.

### Strategy: AI Codex + CTO Gate + 90-Day Eval

| Phase | Sprint | Milestone | Verification |
|-------|--------|-----------|-------------|
| **Read** | 1-3 | Navigate GoClaw codebase, understand patterns | Can explain any Go file in repo |
| **Bug Fix** | 4 | Fix a real bug in GoClaw | PR merged with tests |
| **Feature** | 5 | Implement small feature (e.g., SOUL loading endpoint) | PR merged, CTO approved |
| **Eval** | 8 | 90-day competency assessment | CTO decides: continue Go or pivot |

### Mitigation Tactics

1. **AI Codex**: Use Claude Code / Cursor for Go generation — AI handles syntax, human handles logic
2. **CTO Review Gate**: Every Go PR requires CTO review through Sprint 5
3. **Pattern Library**: Document Go patterns discovered in GoClaw for team reference
4. **Fallback**: If 90-day eval fails → evaluate GoClaw as black-box service, write governance rails in TypeScript

## Consequences

### Positive
- Single binary deployment (no Node.js runtime)
- Native PostgreSQL multi-tenant with RLS
- Agent team orchestration built-in
- <35MB RAM baseline, <1s startup
- MIT license allows full modification

### Negative
- Go learning curve for TypeScript team
- Dependency on GoClaw upstream for major features
- Need to understand Go testing patterns

### Risks
- **Go competency gap** → mitigated by AI Codex + phased learning
- **GoClaw upstream abandonment** → MIT fork, we own the code
- **LICENSE file missing** → documented, MIT declared in multiple places

### go.mod Module Name (Intentionally Kept)

`go.mod` retains `module github.com/Minh-Tam-Solution/MTClaw` through Sprint 1-3.

**Rationale**: Renaming the Go module path requires updating every internal import across 300+ Go files. Doing this prematurely risks import path breakage during the Go competency ramp-up phase. The module name does not affect binary output or runtime behavior.

**Plan**:
- Sprint 1-3: Keep `Minh-Tam-Solution/mtclaw` (stability during learning phase)
- Sprint 4+: Evaluate rename to `github.com/Minh-Tam-Solution/MTClaw` when team is confident with Go tooling
- Rename is a single `sed` + `go mod tidy` operation — low risk when ready

---

## References
- [GoClaw License Verification](../../00-foundation/mtclaw-license-verification.md)
- [GoClaw go.mod](../../go.mod)
- GoClaw upstream: `github.com/Minh-Tam-Solution/MTClaw`
