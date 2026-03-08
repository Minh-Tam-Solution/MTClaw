# Sprint 002 — Requirements & Design

**Sprint**: 2 of 10
**Duration**: 5 days
**Phase**: Phase 1 — Foundation + First Rails
**Gate**: G0.2 (Requirements Ready)
**Predecessor**: Sprint 1 (G0.1 APPROVED — CTO 8.5/10, CPO 8/10)
**Status**: COMPLETE

---

## Sprint Goal

> Complete requirements, user journey map, SOUL quality rubric, RLS design,
> /spec command design, and GoClaw schema deep dive. Submit G0.2 gate.

## Deliverables

### Day 1: CPO Priority + Cleanup

| # | Deliverable | User Story | Status |
|---|------------|------------|--------|
| 1 | GoClaw docs reorganized → `docs/99-mtclaw-upstream/` | CTO ISSUE-2 | Done |
| 2 | User Journey Map — 3 personas x first interaction | US-013 | Done |

### Day 1-2: Requirements & Design

| # | Deliverable | User Story | Status |
|---|------------|------------|--------|
| 3 | SOUL quality rubric | US-008 | Done |
| 4 | API spec (governance endpoints — 73 endpoints mapped) | US-007 | Done |
| 5 | GoClaw schema deep dive → SOUL loading plan | US-009 | Done |

### Day 3-4: Architecture

| # | Deliverable | User Story | Status |
|---|------------|------------|--------|
| 6 | RLS tenant isolation design | US-011 | Done |
| 7 | /spec command design (skill-based approach) | US-012 | Done |

### Day 5: Gate

| # | Deliverable | User Story | Status |
|---|------------|------------|--------|
| 8 | G0.2 gate proposal | US-010 | Done |

## Verification Checklist (DoD)

- [x] GoClaw docs separated from SDLC docs
- [x] User Journey Map (3 personas × first interaction flow)
- [x] SOUL quality rubric (5-dimension score card + behavioral tests)
- [x] API spec (73 endpoints: 55 inherited + 18 governance)
- [x] SOUL loading implementation plan (DB-based, 3 injection points)
- [x] RLS design with migration sketch (8 tables + SET LOCAL middleware)
- [x] /spec command design (skill-based, Sprint 4 prototype → Sprint 7 full)
- [x] G0.2 proposal submitted

---

**References**: [Sprint 1](SPRINT-001-Foundation.md) | [Requirements](../../01-planning/requirements.md) | [User Stories](../../01-planning/user-stories.md)
