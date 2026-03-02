# User Stories — MTClaw Sprint 1

**SDLC Stage**: 01-Planning
**Version**: 1.0.0
**Date**: 2026-03-02
**Author**: [@pm]

---

## Sprint 1: Foundation

### US-001: Project Initialization
**As a** development team member
**I want** MTClaw repo with SDLC 6.1.1 structure, GoClaw runtime, and 16 SOULs
**So that** we have a working foundation for governance rails development

**Acceptance Criteria**:
- [ ] Repo at `https://github.com/Minh-Tam-Solution/MTClaw`
- [ ] GoClaw binary builds (`make build`)
- [ ] 16 SOUL files in `docs/08-collaborate/souls/`
- [ ] SDLC folder structure (00-09)
- [ ] AGENTS.md, README.md, .env.example

### US-002: PostgreSQL Connection
**As a** developer
**I want** GoClaw to connect to PostgreSQL and run migrations
**So that** we have a working database for multi-tenant data

**Acceptance Criteria**:
- [ ] `make migrate-up` succeeds
- [ ] `GET /api/agents` returns agent list
- [ ] Database schema documented in `goclaw-schema-analysis.md`

### US-003: 4 ADRs
**As a** [@cto]
**I want** architecture decisions documented in ADR format
**So that** future developers understand design rationale

**Acceptance Criteria**:
- [ ] ADR-001: GoClaw Adoption + Go competency plan
- [ ] ADR-002: Three-System Architecture + coupling rules + integration mechanisms
- [ ] ADR-003: Observability + tenant cost guardrails
- [ ] ADR-004: SOUL Implementation + drift control + data flow

### US-004: Stage 00 Foundation
**As a** [@pm]
**I want** problem statement, business case, and user research documented
**So that** G0.1 gate has evidence for approval

**Acceptance Criteria**:
- [ ] Problem statement (who, current/desired state, root cause)
- [ ] Business case (cost, ROI, strategic value)
- [ ] User research reused from Sprint 29 (8 interviews)
- [ ] Baseline metrics with confidence caveats

### US-005: G0.1 Gate Proposal
**As a** [@pm]
**I want** G0.1 gate proposal submitted
**So that** project has formal approval to proceed

**Acceptance Criteria**:
- [ ] All evidence linked (problem statement, business case, interviews, ADRs)
- [ ] Running artifact: GoClaw → PostgreSQL → API works
- [ ] 16 SOULs ported
- [ ] License verification documented

### US-006: SOUL Validation
**As a** developer
**I want** `make souls-validate` to check SOUL file integrity
**So that** broken SOUL files are caught at build time

**Acceptance Criteria**:
- [ ] Validates YAML frontmatter presence
- [ ] Reports count of SOUL files found
- [ ] Warns on missing frontmatter

---

## Sprint 2 Preview (User Stories)

- US-007: Complete requirements + user stories + API spec
- US-008: SOUL quality rubric (from EndiorBot Vibecoding)
- US-009: GoClaw schema deep dive → SOUL loading plan
- US-010: G0.2 + G1 gates
- US-011: RLS tenant isolation design
- US-012: /spec command design (for Sprint 4 prototype)

---

**References**: [Requirements](requirements.md), [Sprint Plan](../04-build/sprints/SPRINT-001-Foundation.md)
