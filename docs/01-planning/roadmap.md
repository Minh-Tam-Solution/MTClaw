# Product Roadmap — MTClaw

**SDLC Stage**: 01-Planning
**Version**: 2.3.0
**Date**: 2026-03-04 (updated Sprint 6 COMPLETE, Sprint 7 planning)
**Author**: [@pm]
**Framework**: SDLC Enterprise Framework 6.1.1
**Tier**: STANDARD
**Duration**: 10 sprints (5 days each) ≈ 20 weeks

---

## Big Picture

```
                        MTClaw 10-Sprint Roadmap (v2.2)
                        ===============================

   Phase 1: Foundation + First Rails          Phase 2: Governance    Phase 3: Scale
   ─────────────────────────────────          ────────────────────    ──────────────
   Sprint 1  Sprint 2  Sprint 3  Sprint 4  Sprint 5 │ Sprint 6  Sprint 7  Sprint 8 │ Sprint 9  Sprint 10
   ────────  ────────  ────────  ────────  ──────── │ ────────  ────────  ──────── │ ────────  ──────────
   Init +    Reqs +    Arch +    Core +    MTS      │ NQH +     Spec     PR Gate  │ Full 3    OaaS
   GoClaw    Design    RLS       /spec     Pilot    │ Rail #3   Full     ENFORCE  │ Rails     Prep
   16 SOULs  API Spec  Tenant    Telegram  PR Gate  │ Zalo      BDD      G4       │ Audit     Multi-
   G0.1      G0.2      G2        (proto)   G3 warn  │ RAG       Risk     Valid    │ Comply    Tenant
   ────────  ────────  ────────  ────────  ──────── │ ────────  ────────  ──────── │ ────────  ──────────
   ✅ DONE   ✅ DONE   ✅ DONE   ✅ DONE   ✅ DONE   │ ✅ DONE   → NEXT              │
                       9.2/10   9.0/10               │ 8.0/10                       │
   ◄───────── MTS Internal (10 users) ──────────────►│◄── NQH Expansion (150) ────►│◄── Revenue ──►
```

---

## Phase 1: Foundation + First Rails (Sprint 1-5)

### Sprint 1 — Project Init + GoClaw Runs ✅ COMPLETE

**Gate**: G0.1 (Problem Definition)
**Status**: COMPLETE — CTO 8.5/10, CPO 8/10

| Deliverable | Status |
|-------------|--------|
| Repo with SDLC 6.1.1 structure | ✅ |
| GoClaw binary builds (~25MB) | ✅ |
| PostgreSQL → migrations → API works | ✅ |
| 16 SOUL files ported (12 SDLC + 4 MTS) | ✅ |
| 4 ADRs (GoClaw, 3-System, Observability, SOUL) | ✅ |
| Stage 00: Problem statement, business case, user research | ✅ |
| License verification (MIT confirmed) | ✅ |

---

### Sprint 2 — Requirements & Design ✅ COMPLETE

**Gate**: G0.2 (Requirements Ready)
**Status**: COMPLETE — CTO 9/10, CPO 8.5/10

| Deliverable | Status |
|-------------|--------|
| User Journey Map (3 personas × first interaction) | ✅ |
| SOUL Quality Rubric (5-dimension score card) | ✅ |
| API Specification (73 endpoints: 55 inherited + 18 governance) | ✅ |
| SOUL Loading Implementation Plan (3 injection points) | ✅ |
| RLS Tenant Isolation Design (8 tables + SET LOCAL middleware) | ✅ |
| /spec Command Design (skill-based approach) | ✅ |
| GoClaw Schema Analysis (30+ tables documented) | ✅ |

---

### Sprint 3 — Architecture + RLS Implementation ✅ COMPLETE

**Gate**: G2 (Architecture Ready) — **APPROVED 9.2/10** ([@cto], 2026-03-02)
**Duration**: 5 days
**Owner**: [@architect] → [@coder]
**Points**: ~13 (P0 heavy sprint)

| Day | Deliverable | Priority | Status |
|-----|------------|----------|--------|
| 1 | System Architecture Document (10 sections, incl. Context Drift) | P0 | ✅ Complete |
| 1-2 | RLS migration + tenant middleware (8 tables) | P0 | ✅ Implemented (migration 000008) |
| 2-3 | SOUL seeding migration (16 agents + 48 context files) | P0 | ✅ Implemented (migration 000009) |
| 3-4 | Observability implementation (slog + OTEL + traces) | P1 | ✅ Implemented (migration 000010) |
| 4-5 | Bflow AI-Platform provider setup + verify | P1 | ✅ Implemented (migration 000011 + provider code) |
| 5 | G2 gate proposal | P0 | ✅ APPROVED |

**CTO Architecture Review (G2)**: 9.2/10 — 3 issues, none blocking
- ISSUE-A (LOW): ADR-004 DB seeding path — **FIXED** (SeedToStore note added)
- ISSUE-B (LOW): token_usage table timing — DEFERRED (use traces fields until Sprint 5)
- ISSUE-C (MEDIUM): spans RLS FK chain — **RESOLVED** (spans has direct agent_id column)

**CTO Code Review**: 8.5/10 APPROVED — 1 P1 bug fixed
- P1 BUG: `HasAnyProvider()` missing BflowAI check — **FIXED** (`config_channels.go`)
- ISSUE-2: bflowTransport RoundTripper contract — **FIXED** (clone before mutate)

**New deliverable added**: Context Drift & Semantic Blindness Prevention architecture (SAD Section 8)
- 3-layer system: Context Anchoring + Retrieval Intelligence + Evidence & Explainability
- Adapted from EndiorBot battle-tested patterns (TS-007, ADR-009, ADR-015)
- Phased into Sprint 4 (Layer A) → Sprint 6 (Layer B) → Sprint 7 (Layer C)

**Entry Criteria**: G0.2 APPROVED ✅
**Exit Criteria**: G2 APPROVED ✅ (9.2/10, 2026-03-02)

---

### Sprint 4 — Core Deploy + /spec Prototype (Rail #1) ✅ COMPLETE

**Gate**: None (mid-phase)
**Status**: COMPLETE — CTO 9.0/10 APPROVED, Reviewer 8.5/10
**Duration**: 5 days
**Owner**: [@coder] + [@pm] (pilot)
**Points**: ~12

| Day | Deliverable | Priority | Status |
|-----|------------|----------|--------|
| 1 | Telegram polling config (.env.example) | P0 | ✅ |
| 1 | spec-factory SKILL.md (auto-discovered by skills loader) | P0 | ✅ |
| 1-2 | /spec command handler → PM SOUL (Rail #1 prototype) | P0 | ✅ |
| 2 | **Context Anchoring Layer A** (session goal + SOUL identity → ExtraPrompt) | P0 | ✅ |
| 3 | SOUL routing (@mention → validated agent key) | P0 | ✅ |
| 3-4 | Evidence metadata enrichment (TraceName + TraceTags) | P1 | ✅ |
| 1 | `make souls-validate` (frontmatter FAIL, char budget WARN) | P1 | ✅ |
| 1 | IT Admin SOUL seed (migration 000012 — CEO directive) | P1 | ✅ |

**Key Deliverables**:
- `/spec Create login feature` → "📋 Generating spec..." → PM SOUL → structured JSON
- Context Anchoring Layer A: session goal + SOUL identity reminder in ExtraPrompt
- @mention routing: `@reviewer`, `@pm`, `@itadmin` → validated via agents.Get()
- Evidence: TraceName='spec-factory' + TraceTags=['rail:spec-factory', 'command:spec']
- IT Admin SOUL: 17th SOUL seeded (migration 000012), mutual delegation with devops

**CTO Review (9.0/10)**: No blockers. Minor: migration header fixed (2→3 links).
**Reviewer (8.5/10)**: ISSUE-1 fixed (/spec case-sensitive prefix). ISSUE-2 (UTF-8 rune) = tech debt.

**Remaining (not @coder scope)**:
- US-020: BotFather registration (operational — manual `/newbot`)
- US-023: SOUL feedback session ([@pm] scope — recruit testers, Day 4-5)

---

### Sprint 5 — MTS Pilot + PR Gate WARNING (Rail #2) ✅ COMPLETE

**Gate**: G3 (Build Ready)
**Duration**: 5 days
**Owner**: [@coder] (implementation) + [@pm] (pilot ops) + [@devops] (deploy)
**Points**: ~13
**Entry Criteria**: Sprint 4 complete (CTO 9.0/10), Telegram bot registered

| Day | Deliverable | US | Priority | Points |
|-----|------------|-----|----------|--------|
| 1-2 | PR Gate SKILL.md + `/review` command (Telegram-first) | US-027 | P0 | 4 |
| 2-3 | MTS staging deployment (VPS + Docker Compose + ai-net) | US-028 | P0 | 2 |
| 3-4 | Integration tests (tenant isolation, SOUL routing, AI fallback) | US-029 | P1 | 3 |
| 4 | Token cost tracking verify (CTO ISSUE-B resolution) | US-030 | P1 | 1 |
| 4-5 | MTS pilot: 10 employees onboard to Telegram bot | US-031 | P0 | 1 |
| 5 | Sprint 4 feedback incorporation (if blocking UX issues) | US-032 | P1 | 1 |
| 5 | G3 gate proposal | US-033 | P0 | 1 |

**Scope Adjustment** (from v2.0.0 roadmap):
- PR Gate Sprint 5 = **Telegram `/review` command** (user pastes PR URL → reviewer SOUL fetches diff → WARNING report in Telegram). GitHub webhook integration deferred to Sprint 8 (ENFORCE mode). Rationale: WARNING mode doesn't need webhook infrastructure — validates review logic before building pipeline. Same reviewer SOUL + SKILL.md reused in Sprint 8.
- See: [PR Gate Design](../02-design/pr-gate-design.md)

**Key Deliverables**:
- MTS employees use MTClaw daily; PR Gate reports (WARNING mode via Telegram)
- Token cost tracked per tenant per SOUL via traces table (ISSUE-B resolution)
- Deployment: Docker Compose with ai-net bridge to AI-Platform
- Integration tests: ≥5 scenarios, 70% unit coverage target

**Success Criteria**: 3/10 WAU, PR Gate processes first real PR, cost tracking operational
**Deferred from Sprint 3**: token_usage table (use traces fields until this sprint validates volume)

---

## Phase 2: Governance Hardening (Sprint 6-8)

### Sprint 6 — NQH Tenant + Rail #3 Knowledge + SOUL-Aware RAG ✅ COMPLETE

**Duration**: 5 days
**Owner**: [@coder] + [@devops] (Zalo) + [@pm] (RAG content curation)
**Points**: ~17 (delivered ~7 — MTS-focused scope per CEO Option A)
**Status**: COMPLETE — CTO 8.0/10 APPROVED, 3 fixes applied (CTO-11/12/13)
**Entry Criteria**: G3 APPROVED, MTS pilot running

| Day | Deliverable | Priority | Points |
|-----|------------|----------|--------|
| 1 | NQH tenant configuration (owner_id='nqh', RLS verified) | P0 | 2 |
| 1-2 | Zalo channel integration (OpenClaw extensions/zalo + extensions/zalouser) | P0 | 2 |
| 2-3 | **SOUL-Aware RAG Routing (Context Drift Layer B)** | P0 | 3 |
| 2-3 | **Team mention routing + charters (EndiorBot pattern adoption)** | P1 | 3 |
| 3 | RAG collection: engineering (MTS engineering docs) | P1 | 1 |
| 3-4 | RAG collection: sales + hr-policies | P1 | 2 |
| 4 | NQH-SOPs RAG (805 docs, already indexed → connect via AI-Platform) | P0 | 1 |
| 4-5 | Tenant cost guardrail implementation (monthly token + daily request limits) | P1 | 2 |
| 5 | Cross-tenant isolation regression test (MTS + NQH concurrent) | P0 | 1 |

**Context Drift Layer B — SOUL-Aware RAG Routing**:
```
SOUL Role     →  RAG Collection (AI-Platform)    →  Filter
─────────────────────────────────────────────────────────────
dev           →  engineering                       →  code, architecture
sales         →  sales                             →  pricing, competitors
cs            →  engineering + sales               →  procedures, escalation
assistant     →  engineering + sales (broad)       →  HR Q&A, general tasks
nqh-* SOULs   →  nqh-sops (805 docs)             →  department filter
```
- Token budget: 2,500 hard cap per retrieval (FR-008)
- Ranking: role-aware scoring (SOUL domain match boosts relevance)
- API: `POST /v1/rag/query` with `collection` filter (AI-Platform native)

**Team Routing** (adopted from EndiorBot Sprint 74):
- `@engineering` → PM as team leader + team context injection (ExtraPrompt)
- `@business` → assistant as team leader + business team context
- `@advisory` → CTO as team leader + advisory context
- `/teams` command for discoverability (CPO CONDITION-1)
- Team context enhances RAG routing: team membership → collection mapping
- Resolution order: agent-first (`@pm` → PM directly), team-second (`@engineering` → PM as leader)
- Reuses existing TeamStore infrastructure (23 methods, 4 seeded teams — no DB changes)
- See: `docs/08-collaborate/teams/TEAM-*.md` (charters)

**Conditional**: NQH expansion requires CEO approval (Option A = MTS-only; NQH collections deferred)
**Key Deliverable**: Rail #3 (Knowledge & Answering) operational for MTS; NQH **pilot** (10-20 users) if approved; Team routing active for all tenants
**CPO OBS-2**: Full NQH rollout (150 users) is Sprint 7-8, NOT Sprint 6. Sprint 6 = NQH pilot only (10-20 HO/management users on Zalo). Scale 15x in one sprint is too aggressive.

---

### Sprint 7 — Rail #1 Spec Factory Full + Retrieval Evidence ← NEXT

**Duration**: 5 days
**Owner**: [@coder] + [@pm] (spec validation)
**Points**: ~13
**Entry Criteria**: Rail #3 RAG operational ✅, SOUL-Aware routing working ✅

| Day | Deliverable | Priority | Points |
|-----|------------|----------|--------|
| 1-2 | Spec Factory v1.0: spec_id, BDD scenarios, risk scoring + migration | P0 | 3 |
| 2-3 | Evidence vault link (spec → trace → bidirectional query) | P0 | 2 |
| 3 | **Retrieval Evidence logging (Context Drift Layer C)** | P0 | 2 |
| 3-4 | Spec Telegram commands: /spec-list, /spec-detail | P1 | 2 |
| 4-5 | Gateway consumer refactoring (CTO-14: extract 5 modules) | P1 | 2 |
| 4-5 | SOUL drift detection (checksum monitoring, ADR-004) | P1 | 2 |

**Scope adjustment** (vs roadmap v2.1):
- SOUL behavioral test suite (80+ tests) DEFERRED to Sprint 8. CTO-14 refactoring is prerequisite for testability — behavioral tests more effective after module extraction.
- CTO-14 refactoring ADDED (P2 from Sprint 6 review): extract gateway_consumer.go from 993 → ~600 lines.

**Context Drift Layer C — Evidence & Explainability**:
- RetrievalEvidence logged per RAG call: `{query, collection, results, ranking_reason, soul_role, token_count}`
- Ranking reason enum: `exact_match | semantic_similar | soul_domain_boost | fallback`
- Enables audit trail for RAG quality + debugging retrieval drift
- Adapted from EndiorBot ADR-015 (Retrieval Explainability)
- Reference: SDLC Orchestrator `SpecValidationResult` pattern (metadata in parent entity)

**Spec Factory v1.0** (reference: SDLC Orchestrator `GovernanceSpecification` + SDLC Framework `spec-frontmatter-schema.json`):
- spec_id: `SPEC-YYYY-NNNN` format (Framework compliant)
- BDD: `{scenario, given, when, then}` structured objects (Framework `GIVEN/WHEN/THEN` standard)
- Risk: `{description, probability, impact, mitigation}` matrix
- Status lifecycle: `draft → review → approved → deprecated` (4 states, matches Orchestrator)
- Evidence: `trace_id` FK to traces table (lighter than Orchestrator's S3 vault — appropriate for MTClaw scale)

**Key Deliverable**: `/spec` produces `SPEC-2026-NNNN` with BDD, risk, evidence link
**SOUL Drift**: Checksum-based detection per ADR-004 + version field tracking
**CTO-14**: gateway_consumer.go refactored into 5 focused modules

**Handoff**: `docs/04-build/SPRINT-007-CODER-HANDOFF.md`

---

### Sprint 8 — Rail #2 PR Gate ENFORCE + G4

**Gate**: G4 (Validation Ready)
**Duration**: 5 days
**Owner**: [@coder] + [@cto] (PR Gate rules) + [@pm] (G4 proposal)
**Points**: ~13
**Entry Criteria**: Spec Factory full, 3 RAG collections operational, SOUL drift detection active

| Day | Deliverable | Priority | Points |
|-----|------------|----------|--------|
| 1-2 | PR Gate → ENFORCE mode (block merge on policy violation) | P0 | 3 |
| 2-3 | 3 Rails integration test (all 3 running together) | P0 | 2 |
| 3 | Context Drift full validation (Layer A+B+C end-to-end test) | P0 | 2 |
| 3-4 | **SOUL behavioral test suite (16 SOULs × 5+ tests)** (deferred from Sprint 7) | P1 | 2 |
| 4 | Evidence export for audit (JSON + CSV) | P1 | 2 |
| 5 | G4 gate proposal (Validation Ready) | P0 | 2 |

**Key Deliverable**: All 3 Rails running, PR Gate blocks non-compliant merges
**Context Drift Validation**: Full E2E — SOUL identity retained after 50+ turns, RAG returns domain-correct results, evidence logged for every retrieval
**Success Criteria**: 7/10 MTS WAU, evidence capture 100% for gated actions, 80% unit test coverage target met
**PR Gate Rules** (CTO tuned from WARNING data in Sprint 5-7):
- BLOCK: missing spec reference, no test coverage, security violations
- WARN: low coverage (<60%), missing docstrings, large diff (>500 lines)

---

## Phase 3: Scale (Sprint 9-10+)

### Sprint 9 — Full 3 Rails Governance + Hardening

**Duration**: 5 days
**Owner**: [@coder] + [@tester] (pen test) + [@pm] (audit reports)
**Points**: ~12
**Entry Criteria**: G4 APPROVED, all 3 Rails running, PR Gate in ENFORCE mode

| Day | Deliverable | Priority | Points |
|-----|------------|----------|--------|
| 1-2 | Full audit trail export (compliance reporting — JSON + CSV + PDF) | P0 | 3 |
| 2-3 | Cross-rail evidence linking (spec → PR → test → deploy traceability) | P1 | 2 |
| 3 | SOUL quality regression suite (automated weekly, 16 SOULs × 5+ tests) | P1 | 2 |
| 3-4 | Performance tuning (cost query optimization, RAG latency <3s p95) | P2 | 2 |
| 4-5 | Security penetration test (tenant isolation, RLS bypass attempts) | P1 | 2 |
| 5 | Post-mortem: Sprint 1-9 lessons + Phase 3 planning | P1 | 1 |

**Key Deliverables**:
- Complete governance trail: every spec, PR review, and knowledge query is auditable
- Cross-rail linking: `SPEC-2026-001 → PR #42 → 95% coverage → deployed v1.2`
- Pen test: attempt RLS bypass via SQL injection, cross-tenant API calls, SOUL impersonation

---

### Sprint 10+ — OaaS Preparation

**Duration**: 5 days
**Gate**: G5 (Scale Ready)
**Owner**: [@coder] + [@pm] (pricing) + [@ceo] (strategy)
**Points**: ~12
**Entry Criteria**: Sprint 9 audit trail complete, pen test passed

| Day | Deliverable | Priority | Points |
|-----|------------|----------|--------|
| 1-2 | Multi-tenant self-service (tenant registration API + admin panel) | P0 | 3 |
| 2-3 | Pricing model implementation (token usage billing per tenant) | P1 | 2 |
| 3-4 | External tenant onboarding flow (<30 min time to first value) | P1 | 3 |
| 4 | SOUL marketplace design (industry-specific personas: F&B, retail, tech) | P2 | 2 |
| 4-5 | Documentation: tenant admin guide, API reference, deployment guide | P1 | 2 |

**Key Deliverables**:
- Self-service: new tenant signs up → RLS auto-configured → 16 default SOULs cloned → Telegram/Zalo connected
- Pricing: token-based (per 1K tokens) + monthly cap per tier
- SOUL marketplace: tenants can browse/activate industry-specific SOULs beyond the 16 defaults
- **G5 proposal**: Scale readiness for OaaS commercialization

---

## Sprint-by-Sprint Summary

| Sprint | Phase | Goal | Gate | Rails | Users | Context Drift |
|--------|-------|------|------|-------|-------|---------------|
| 1 ✅ | Foundation | Init + GoClaw + 16 SOULs | G0.1 ✅ | 0 | 0 | — |
| 2 ✅ | Foundation | Requirements + Design | G0.2 ✅ | 0 | 0 | — |
| 3 ✅ | Foundation | Architecture + RLS | G2 ✅ (9.2/10) | 0 | 0 | Design |
| 4 ✅ | First Rails | /spec + Context Anchoring + @mention | — (9.0/10) | 1 | ~3 | Layer A |
| **5** | **First Rails** | **MTS Pilot + PR Gate WARNING** | **G3** | **2** | **10** | **—** |
| 6 | Governance | NQH pilot + Knowledge/RAG + Team routing | — | 3 | 10+20 | Layer B + Teams |
| 7 | Governance | Spec Factory full + NQH rollout | — | 3 | ~80 | Layer C |
| 8 | Governance | PR Gate ENFORCE + G4 | G4 | 3 | ~160 | Validate |
| 9 | Scale | Full governance + hardening | — | 3 | ~160 | — |
| 10+ | Scale | OaaS preparation | G5 | 3 | Expand | — |

---

## Dependency Map

```
Sprint 1 ─┐
           ├──► Sprint 2 ──► Sprint 3 ──► Sprint 4 ──► Sprint 5
           │    (reqs)       (arch+RLS)   (/spec)      (pilot)
           │                    │             │            │
           │                    ▼             ▼            ▼
           │                Sprint 6 ──► Sprint 7 ──► Sprint 8
           │                (NQH+RAG)    (spec full)  (PR Gate)
           │                                              │
           │                                              ▼
           │                                Sprint 9 ──► Sprint 10+
           │                                (audit)      (OaaS)
           │
           └──► Critical path: Sprint 3 (RLS) blocks ALL feature sprints
```

---

## Gate Schedule

| Gate | Sprint | Purpose | Reviewers | Status |
|------|--------|---------|-----------|--------|
| G0.1 | 1 | Problem Definition | [@cto], [@cpo] | ✅ APPROVED (CTO 8.5/10, CPO 8/10) |
| G0.2 | 2 | Requirements Ready | [@cto], [@cpo] | ✅ APPROVED (CTO 9/10, CPO 8.5/10) |
| G2 | 3 | Architecture Ready | [@cto] | ✅ APPROVED (9.2/10, 3 issues none blocking) |
| **G3** | **5** | **Build Ready (MTS Pilot)** | **[@cto], [@cpo]** | ⏳ Pending |
| G4 | 8 | Validation Ready (3 Rails) | [@cto], [@cpo], [@ceo] | ⏳ Pending |
| G5 | 10 | Scale Ready (OaaS) | [@ceo] | ⏳ Pending |

---

## Resource Plan

| Sprint | Primary Owner | Secondary | CTO Review | Status |
|--------|--------------|-----------|------------|--------|
| 1-2 | [@pm] | [@researcher] | Gate reviews | ✅ Complete |
| 3 | [@architect] | [@coder] | G2 gate | ✅ Complete (9.2/10) |
| 4 | [@coder] | [@pm] | Sprint review | ✅ Complete (9.0/10) |
| **5** | **[@coder]** | **[@pm] (pilot)** | **PR Gate design** | **← Next** |
| 6-8 | [@coder] | [@devops] (infra) + [@pm] (RAG content) | Gate reviews | Planned |
| 9-10 | [@coder] | [@pm] (OaaS) + [@tester] (pen test) | Final approval | Planned |

---

## Risk Register (Cross-Sprint)

| # | Risk | Prob | Impact | Mitigation | Sprint | Status |
|---|------|------|--------|------------|--------|--------|
| R1 | RLS breaks existing GoClaw queries | Med | High | Test all 55 endpoints after migration | 3 | Design verified (spans has direct agent_id) |
| R2 | Go competency gap delays implementation | Med | Med | AI Codex + CTO review gate (ADR-001) | 3-5 | Active |
| R3 | Bflow AI-Platform latency >5s | Low | Med | Fallback design in ADR-005, graceful degradation | 4 | Active |
| R4 | MTS adoption <30% WAU by Sprint 5 | Med | High | SOUL feedback session Sprint 4, iterate | 4-5 | Active |
| R5 | SOUL context exceeds LLM context window | Low | High | 2,000 char budget per SOUL.md (CTO-3) | 3 | Mitigated (make souls-validate) |
| R6 | NQH CEO approval delayed | Med | Med | Phase 2 deferred, Phase 1 continues | 6 | ✅ Resolved (CEO Option A: MTS-only) |
| R7 | PR Gate false positives (ENFORCE blocks valid PRs) | Med | Med | WARNING mode first (Sprint 5), tune rules | 5-8 | Active |
| R8 | RAG accuracy below threshold (sales pricing wrong) | Med | High | Manual curation Phase 1, auto-ingest Phase 2 | 6 | Active |
| R9 | Context drift in long conversations (SOUL role confusion) | Med | High | 3-layer prevention (FR-008): anchoring + RAG routing + evidence | 4-7 | **NEW** (designed Sprint 3) |
| R10 | AI-Platform single point of failure | Low | High | Graceful degradation (ADR-005), no direct Ollama fallback | 4+ | Active |
| R11 | token_usage table deferred → cost visibility gap | Low | Med | Use traces.total_input_tokens until Sprint 5 validates volume (CTO ISSUE-B) | 3-5 | Active |
| R14 | Team mention conflicts with agent key | Low | Low | Agent-first resolution: `@pm` → agent directly, `@engineering` → team leader (EndiorBot pattern) | 6 | NEW |

---

## Cross-Sprint: Context Drift Prevention (FR-008)

Context Drift & Semantic Blindness Prevention is a cross-cutting concern phased across Sprint 3-8:

| Layer | Sprint | What | Reference |
|-------|--------|------|-----------|
| Design | 3 ✅ | Architecture: 3-layer system designed in SAD Section 8 | SAD Section 8 |
| **A: Anchoring** | **4** | Session goal + decision log → ExtraPrompt Section [7] (SOUL.md already in [2-4]) | EndiorBot TS-007, ADR-009 |
| **B: Retrieval** | **6** | SOUL-Aware RAG routing with collection filter + role-aware ranking | EndiorBot ADR-015 |
| **C: Evidence** | **7** | RetrievalEvidence logging with ranking_reason enum | EndiorBot ADR-015 |
| Validate | 8 | Full E2E: identity retention + domain-correct RAG + audit trail | G4 gate |

---

## References

- [Product Vision](../00-foundation/product-vision.md)
- [Requirements](requirements.md) (v1.1.0 — includes FR-008)
- [API Specification](api-specification.md)
- [User Journey Map](user-journey-map.md)
- [System Architecture Document](../02-design/system-architecture-document.md)
- [G0.1 Gate Proposal](../00-foundation/G0.1-GATE-PROPOSAL.md)
- [G0.2 Gate Proposal](../00-foundation/G0.2-GATE-PROPOSAL.md)
- [G2 Gate Approval](../00-foundation/G2-GATE-APPROVAL.md) (9.2/10)
- [Sprint 3 Architect Handoff](../04-build/SPRINT-003-ARCHITECT-HANDOFF.md)
