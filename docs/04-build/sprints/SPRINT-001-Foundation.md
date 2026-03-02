# Sprint 001 — Foundation

**Sprint**: 1 of 10
**Duration**: 5 days
**Phase**: Phase 1 — Foundation + First Rails
**Gate**: G0.1 (Problem Definition)
**Status**: IN PROGRESS

---

## Sprint Goal

> MTClaw repo initialized with SDLC 6.1.1 structure, GoClaw runtime builds,
> 16 SOULs ported, 4 ADRs written, and G0.1 gate evidence complete.

## Deliverables

### Day 1: Repo Init + GoClaw Build + 16 SOULs

| # | Deliverable | Status |
|---|------------|--------|
| 1 | MTClaw repo at github.com/Minh-Tam-Solution/MTClaw | Done |
| 2 | SDLC 6.1.1 folder structure (00-09) | Done |
| 3 | GoClaw source copied + `make build` works | Done |
| 4 | 16 SOUL files in `docs/08-collaborate/souls/` | Done |
| 5 | LICENSE (proprietary, GoClaw upstream MIT documented) | Done |
| 6 | README.md, AGENTS.md, CLAUDE.md symlink | Done |
| 7 | .env.example, .gitignore, Makefile (with souls-validate) | Done |
| 8 | SDLC Framework symlink | Done |
| 9 | GitHub remote set | Done |

### Day 1-2: Foundation Docs

| # | Deliverable | Status |
|---|------------|--------|
| 10 | Problem statement | Done |
| 11 | Business case | Done |
| 12 | User research (reused from Sprint 29, 8 interviews) | Done |
| 13 | GoClaw license verification doc | Done |
| 14 | ROLE_TOOL_MATRIX (SASE 12-Role Model) | Done |

### Day 2-3: Architecture

| # | Deliverable | Status |
|---|------------|--------|
| 15 | ADR-001: GoClaw Adoption + Go competency plan | Done |
| 16 | ADR-002: Three-System Architecture + coupling rules | Done |
| 17 | ADR-003: Observability + tenant cost guardrails | Done |
| 18 | ADR-004: SOUL Implementation + drift control | Done |
| 19 | GoClaw schema analysis | Done |

### Day 3-4: Planning

| # | Deliverable | Status |
|---|------------|--------|
| 20 | Requirements (FR + NFR) | Done |
| 21 | Test strategy (tiered: 60% → 70% → 80%) | Done |
| 22 | User stories (Sprint 1) | Done |

### Day 4-5: Gate Submission

| # | Deliverable | Status |
|---|------------|--------|
| 23 | G0.1 gate proposal | Pending |
| 24 | Initial git commit + push | Pending |
| 25 | PostgreSQL connection verified | Pending (next sprint if no local PG) |

## Verification Checklist (DoD)

- [x] GoClaw binary builds (`make build`)
- [x] 16 SOUL files present
- [x] SDLC 6.1.1 structure
- [x] Problem statement + business case
- [x] 4 ADRs (GoClaw, 3-System, Observability, SOUL)
- [x] GoClaw schema analysis
- [x] Requirements + test strategy + user stories
- [x] AGENTS.md <60 lines
- [x] ROLE_TOOL_MATRIX
- [x] `make souls-validate` passes
- [ ] G0.1 proposal submitted
- [ ] Initial commit pushed to GitHub

## Sprint 2 Preview

- Complete requirements + user stories + API spec
- SOUL quality rubric
- GoClaw schema deep dive → SOUL loading plan
- G0.2 + G1 gates
- RLS tenant isolation design
- /spec command design

---

**References**: [Plan v7.0](../../../.claude/plans/) | [Requirements](../../01-planning/requirements.md)
