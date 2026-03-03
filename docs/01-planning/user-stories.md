# User Stories — MTClaw

**SDLC Stage**: 01-Planning
**Version**: 2.0.0
**Date**: 2026-03-02
**Author**: [@pm]
**Framework**: SDLC 6.1.1 — Stage 01 Required Artifact (STANDARD tier)

---

## Sprint 1: Foundation ✅ COMPLETE (G0.1 APPROVED)

### US-001: Project Initialization ✅
**As a** development team member
**I want** MTClaw repo with SDLC 6.1.1 structure, GoClaw runtime, and 16 SOULs
**So that** we have a working foundation for governance rails development

### US-002: PostgreSQL Connection ✅
**As a** developer
**I want** GoClaw to connect to PostgreSQL and run migrations
**So that** we have a working database for multi-tenant data

### US-003: 4 ADRs ✅
**As a** [@cto]
**I want** architecture decisions documented in ADR format
**So that** future developers understand design rationale

### US-004: Stage 00 Foundation ✅
**As a** [@pm]
**I want** problem statement, business case, and user research documented
**So that** G0.1 gate has evidence for approval

### US-005: G0.1 Gate Proposal ✅
**As a** [@pm]
**I want** G0.1 gate proposal submitted
**So that** project has formal approval to proceed

### US-006: SOUL Validation ✅
**As a** developer
**I want** `make souls-validate` to check SOUL file integrity
**So that** broken SOUL files are caught at build time

---

## Sprint 2: Requirements & Design ✅ COMPLETE (G0.2 APPROVED)

### US-007: Requirements + API Specification ✅
**As a** [@pm]
**I want** complete functional/non-functional requirements and API specification
**So that** development has clear targets and interfaces

### US-008: SOUL Quality Rubric ✅
**As a** [@cpo]
**I want** a quality scoring system for SOUL responses
**So that** we can measure and improve SOUL performance over time

### US-009: SOUL Loading Implementation Plan ✅
**As a** [@architect]
**I want** a detailed plan for how SOULs load into the GoClaw runtime
**So that** Sprint 3 implementation is guided by code analysis, not guesswork

### US-010: G0.2 Gate Proposal ✅
**As a** [@pm]
**I want** G0.2 gate proposal with all Sprint 2 evidence
**So that** project has formal requirements approval

### US-011: RLS Tenant Isolation Design ✅
**As a** [@cto]
**I want** a detailed RLS design for all tenant-scoped tables
**So that** Sprint 3 implementation follows a proven, reviewed pattern

### US-012: /spec Command Design ✅
**As a** [@cpo]
**I want** a design for the `/spec` governance rail
**So that** Sprint 4 can build the prototype from a clear specification

### US-013: User Journey Map ✅
**As a** [@cpo]
**I want** user journey maps for 3 personas showing first interaction
**So that** design decisions are grounded in real user flows

---

## Sprint 3: Architecture + RLS Implementation → NEXT

### US-014: System Architecture Document
**As a** [@cto]
**I want** a comprehensive System Architecture Document
**So that** the team has a single reference for all architectural decisions and data flows

**Acceptance Criteria**:
- [ ] Component diagram (Telegram → Gateway → Agent Loop → AI-Platform)
- [ ] Data flow diagram (request lifecycle with RLS + SOUL injection)
- [ ] Deployment diagram (VPS + Docker Compose)
- [ ] Security architecture (RLS, JWT, AES-256-GCM)
- [ ] Integration points documented
- [ ] References all 5 ADRs + Sprint 2 design docs

**Points**: 3 | **Priority**: P0

### US-015: RLS Migration + Tenant Middleware
**As a** [@cto]
**I want** PostgreSQL RLS policies on all tenant-scoped tables with SET LOCAL middleware
**So that** even application bugs cannot leak cross-tenant data

**Acceptance Criteria**:
- [ ] Migration: RLS on 8 core tables
- [ ] `mtclaw_admin` role (bypasses RLS)
- [ ] `mtclaw_app` role (RLS enforced)
- [ ] Tenant middleware: `SET LOCAL app.tenant_id = $1`
- [ ] Cross-tenant isolation test passes
- [ ] All 55 inherited API endpoints still work

**Points**: 3 | **Priority**: P0

### US-016: SOUL Seeding Migration
**As a** developer
**I want** 16 SOULs seeded into the database as agents with context files
**So that** SOUL loading works via GoClaw's existing LoadFromStore() architecture

**Acceptance Criteria**:
- [ ] 16 agent records (predefined type, owner_id='mts')
- [ ] 48 agent_context_files (3 per SOUL: SOUL.md, IDENTITY.md, AGENTS.md)
- [ ] Agent links (delegation permissions)
- [ ] Agent teams ("MTS Engineering", "MTS Business")
- [ ] `GET /v1/agents` returns 16 agents

**Points**: 3 | **Priority**: P0

### US-017: Observability Implementation
**As a** [@devops]
**I want** structured logging with trace_id/tenant_id and metrics export
**So that** we can monitor per-tenant, per-SOUL behavior

**Acceptance Criteria**:
- [ ] slog JSON logging with trace_id, tenant_id, agent_key
- [ ] OTEL metrics at `/metrics` (request count, duration, token usage)
- [ ] Token cost written to traces table per request

**Points**: 2 | **Priority**: P1

### US-018: Bflow AI-Platform Provider Setup
**As a** developer
**I want** Bflow AI-Platform registered as the default LLM provider
**So that** all SOULs use the Bflow AI-Platform for inference

**Acceptance Criteria**:
- [ ] Provider registered with connectivity verified
- [ ] Chat completions work: test message → AI response
- [ ] Tenant header injected: `X-Tenant-ID: mts`
- [ ] Graceful degradation if AI-Platform unavailable

**Points**: 1 | **Priority**: P1

### US-019: G2 Gate Proposal
**As a** [@pm]
**I want** G2 gate proposal with architecture evidence
**So that** the project has formal architectural approval

**Acceptance Criteria**:
- [ ] System Architecture Document referenced
- [ ] RLS verified, 16 SOULs loadable, AI-Platform connected
- [ ] CTO approval

**Points**: 1 | **Priority**: P0

---

## Sprint 4: Core Deploy + /spec Prototype (Rail #1)

### US-020: Telegram Channel Setup
**As a** MTS employee
**I want** to interact with MTClaw via Telegram
**So that** I can access governance-aware AI from my daily messaging app

**Acceptance Criteria**:
- [ ] Telegram bot registered and responding to `/start`
- [ ] Welcome message personalized (name + role)
- [ ] Default SOUL routing works (assistant for new users, `is_default=true`)

**Points**: 1 | **Priority**: P0

### US-021: /spec Command Handler (Rail #1 Prototype)
**As a** [@pm]
**I want** `/spec` command to produce structured JSON specifications
**So that** requirements are standardized and evidence is captured

**Acceptance Criteria**:
- [ ] `/spec {description}` → PM SOUL generates JSON spec
- [ ] Output: title, narrative (As a/I want/So that), acceptance criteria (Given/When/Then)
- [ ] Vietnamese input → Vietnamese output
- [ ] Evidence: trace_id links to generation audit trail
- [ ] User can approve, modify, or discard

**Points**: 3 | **Priority**: P0

### US-022: SOUL Routing
**As a** MTS employee
**I want** my questions automatically routed to the right SOUL
**So that** I get role-appropriate answers without manual SOUL selection

**Acceptance Criteria**:
- [ ] Auto-detect context (HR → assistant handles directly, code → dev)
- [ ] Explicit `@mention` switching (e.g., `@reviewer`)
- [ ] Delegation via spawn() works
- [ ] No restart needed when switching SOULs

**Points**: 2 | **Priority**: P0

### US-023: SOUL Feedback Session
**As a** [@cpo]
**I want** real MTS users to test their SOULs and provide feedback
**So that** we validate SOUL quality before wider rollout

**Acceptance Criteria**:
- [ ] 3-4 MTS users (1 Engineering, 1 Sales, 1 Back Office)
- [ ] 15-minute sessions with assigned tasks per persona
- [ ] Measure: time to first useful answer, satisfaction (1-5), would-use-again
- [ ] Findings documented → SOUL tuning for Sprint 5

**Points**: 1 | **Priority**: P1

---

## Sprint 5: MTS Pilot + PR Gate WARNING (Rail #2)

### US-024: PR Gate Skill (Rail #2 WARNING)
**As a** MTS Engineering team
**I want** AI-powered PR review with structured quality checklist
**So that** code quality is consistently evaluated before merge

**Acceptance Criteria**:
- [ ] PR Gate skill: reviewer SOUL + GitHub PR diff retrieval
- [ ] WARNING mode: report issues, don't block merge
- [ ] Structured output: verdict, score (0-100), findings by severity
- [ ] Checks: SQL injection, RLS compliance, test coverage

**Points**: 3 | **Priority**: P0

### US-025: MTS Pilot Deployment
**As a** [@pm]
**I want** all 10 MTS employees onboarded to the Telegram bot
**So that** we validate real-world adoption

**Acceptance Criteria**:
- [ ] VPS staging deployment (Docker Compose)
- [ ] 10 employees have Telegram bot access
- [ ] Onboarding guide shared
- [ ] WAU tracking active

**Points**: 2 | **Priority**: P0

### US-026: G3 Gate Proposal
**As a** [@pm]
**I want** G3 (Build Ready) gate proposal
**So that** MTS pilot has formal approval

**Points**: 1 | **Priority**: P0

---

## Sprint 6: NQH Tenant + Rail #3 Knowledge

### US-027: NQH Tenant Configuration (Conditional)
**As a** NQH administrator
**I want** NQH tenant isolated from MTS data
**So that** NQH employees have their own SOUL configurations and RAG collections

**Acceptance Criteria**:
- [ ] NQH tenant (owner_id='nqh') with RLS verified
- [ ] NQH-specific SOULs configured
- [ ] Cross-tenant isolation confirmed

**Points**: 2 | **Priority**: P0 (conditional on CEO approval)

### US-028: Zalo Channel Integration (Conditional)
**As a** NQH employee
**I want** to use MTClaw via Zalo
**So that** I can access AI assistance from the messaging app I use daily

**Points**: 3 | **Priority**: P0 (conditional)

### US-029: RAG Collections (Rail #3)
**As a** MTS employee
**I want** domain-specific knowledge collections for my role
**So that** AI answers are enriched with real company data

**Acceptance Criteria**:
- [ ] engineering: Bflow API docs, coding standards
- [ ] sales: pricing, proposals, case studies
- [ ] hr-policies: HR policies, leave, benefits
- [ ] Source citation in all RAG answers

**Points**: 4 | **Priority**: P1

### US-030: Tenant Cost Guardrails
**As a** [@cto]
**I want** per-tenant token limits with soft-throttle at 80%/100%
**So that** AI costs are controlled and predictable

**Points**: 2 | **Priority**: P1

---

## Sprint 7: Rail #1 Spec Factory Full

### US-031: Spec Factory Full Version
**As a** [@pm]
**I want** `/spec` to produce complete specs with spec_id, BDD scenarios, risk scoring
**So that** specifications are traceable and quality-assured

**Acceptance Criteria**:
- [ ] spec_id: SPEC-YYYY-NNNN
- [ ] BDD scenarios (Given/When/Then structured)
- [ ] Risk scoring (probability × impact + mitigation)
- [ ] Dependencies (links to other specs)
- [ ] Evidence vault link (spec → trace → file)

**Points**: 3 | **Priority**: P0

### US-032: SOUL Behavioral Test Suite
**As a** [@cpo]
**I want** automated behavioral tests for all 16 SOULs
**So that** SOUL quality regression is detected automatically

**Acceptance Criteria**:
- [ ] 5+ test cases per SOUL (80+ total)
- [ ] Tests: correctness, role boundary, Vietnamese support
- [ ] Automated weekly run with score tracking

**Points**: 2 | **Priority**: P1

### US-033: SOUL Drift Detection
**As a** [@pm]
**I want** automatic detection when SOUL content changes or quality degrades
**So that** we catch issues before users notice

**Points**: 2 | **Priority**: P1

---

## Sprint 8: Rail #2 PR Gate ENFORCE + G4

### US-034: PR Gate ENFORCE Mode
**As a** [@cto]
**I want** PR Gate to block merge on policy violation
**So that** non-compliant code cannot reach production

**Points**: 3 | **Priority**: P0

### US-035: 3 Rails Integration
**As a** [@pm]
**I want** all 3 governance rails running together
**So that** the governance backbone is complete

**Points**: 2 | **Priority**: P0

### US-036: Evidence Export
**As a** compliance officer
**I want** to export evidence records for audit
**So that** governance actions are verifiable

**Points**: 2 | **Priority**: P1

### US-037: G4 Gate Proposal
**As a** [@pm]
**I want** G4 (Validation Ready) gate proposal
**So that** the governance platform has formal validation approval

**Points**: 2 | **Priority**: P0

---

## Sprint 9-10: Scale + OaaS

### US-038: Full Audit Trail
**As a** compliance officer
**I want** cross-rail evidence linking (spec → PR → test → deploy)
**So that** the full governance lifecycle is auditable

### US-039: Multi-Tenant Self-Service
**As a** new tenant admin
**I want** self-service tenant registration and SOUL configuration
**So that** onboarding doesn't require manual intervention

### US-040: OaaS Pricing Model
**As a** business owner
**I want** token-usage based billing for external tenants
**So that** MTClaw can generate revenue as Governance-as-a-Service

---

## Story Map Summary

| Sprint | Stories | Points | Gate |
|--------|---------|--------|------|
| 1 ✅ | US-001 to US-006 | ~10 | G0.1 |
| 2 ✅ | US-007 to US-013 | ~12 | G0.2 |
| **3** | **US-014 to US-019** | **~13** | **G2** |
| 4 | US-020 to US-023 | ~7 | — |
| 5 | US-024 to US-026 | ~6 | G3 |
| 6 | US-027 to US-030 | ~11 | — |
| 7 | US-031 to US-033 | ~7 | — |
| 8 | US-034 to US-037 | ~9 | G4 |
| 9-10 | US-038 to US-040 | ~10 | G5 |

**Cross-cutting concern**: Context Drift & Semantic Blindness prevention (FR-008) is addressed by:
- US-016 (SOUL seeding) — identity anchoring via SOUL.md + IDENTITY.md
- US-029 (RAG collections) — SOUL-aware retrieval with domain collection routing
- System Architecture Document Section 8 — full 3-layer prevention architecture

---

## References

- [Requirements](requirements.md)
- [Roadmap](roadmap.md)
- [Product Vision](../00-foundation/product-vision.md)
- [User Personas](../00-foundation/user-personas.md)
- [Sprint 3 Plan](../04-build/sprints/SPRINT-003-Architecture-RLS.md)
- [System Architecture Document](../02-design/system-architecture-document.md)
