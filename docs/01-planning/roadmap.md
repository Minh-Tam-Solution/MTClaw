# Product Roadmap — MTClaw

**SDLC Stage**: 01-Planning
**Version**: 2.7.0
**Date**: 2026-03-22 (Sprint 10 COMPLETE pending CTO review; Sprint 11 plan filed — Hardening; Sprint 12 OaaS + Dogfooding added)
**Author**: [@pm] + [@architect]
**Framework**: SDLC Enterprise Framework 6.1.1
**Tier**: STANDARD
**Duration**: 12 sprints (5 days each) ≈ 24 weeks

---

## Big Picture

```
                        MTClaw 12-Sprint Roadmap (v2.7)
                        ================================

   Phase 1: Foundation + First Rails          Phase 2: Governance         Phase 3: Scale
   ─────────────────────────────────          ─────────────────────        ─────────────────────────────────────
   Sprint 1  Sprint 2  Sprint 3  Sprint 4  Sprint 5 │ Sprint 6  Sprint 7  Sprint 8 │ Sprint 9  Sprint 10  Sprint 11  Sprint 12
   ────────  ────────  ────────  ────────  ──────── │ ────────  ────────  ──────── │ ────────  ──────────  ─────────  ─────────
   Init +    Reqs +    Arch +    Core +    MTS      │ NQH +     Spec     PR Gate  │ Channel   MS Teams    Hardening  OaaS +
   GoClaw    Design    RLS       /spec     Pilot    │ Rail #3   Full     ENFORCE  │ Cleanup   Extension   Pen Test   Dogfood
   16 SOULs  API Spec  Tenant    Telegram  PR Gate  │ Zalo      BDD      G4       │ SOUL x17  NQH Corp    Audit PDF  Self-Dev
   G0.1      G0.2      G2        (proto)   G3 warn  │ RAG       Risk     Valid    │ ADR-006   ADR-007     ADR-008/9  G5
   ────────  ────────  ────────  ────────  ──────── │ ────────  ────────  ──────── │ ────────  ──────────  ─────────  ─────────
   ✅ DONE   ✅ DONE   ✅ DONE   ✅ DONE   ✅ DONE  │ ✅ DONE  ✅ DONE  ✅ DONE  │ ✅ DONE  ✅ DONE    → NEXT     Planned
                       9.2/10   9.0/10              │ 8.0/10   8.0/10   8.5/10   │ 9.0/10   pending
   ◄───────── MTS Internal (10 users) ─────────────►│◄── NQH Expansion (150) ─────►│◄────────── Revenue ───────────────────►
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
- ISSUE-A (LOW): ADR-004 DB seeding path — **FIXED**
- ISSUE-B (LOW): token_usage table timing — DEFERRED (use traces fields until Sprint 5)
- ISSUE-C (MEDIUM): spans RLS FK chain — **RESOLVED** (spans has direct agent_id column)

**CTO Code Review**: 8.5/10 APPROVED — 1 P1 bug fixed
- P1 BUG: `HasAnyProvider()` missing BflowAI check — **FIXED**
- ISSUE-2: bflowTransport RoundTripper contract — **FIXED** (clone before mutate)

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

---

### Sprint 5 — MTS Pilot + PR Gate WARNING (Rail #2) ✅ COMPLETE

**Gate**: G3 (Build Ready)
**Duration**: 5 days
**Owner**: [@coder] (implementation) + [@pm] (pilot ops) + [@devops] (deploy)
**Points**: ~13

| Day | Deliverable | US | Priority | Points |
|-----|------------|-----|----------|--------|
| 1-2 | PR Gate SKILL.md + `/review` command (Telegram-first) | US-027 | P0 | 4 |
| 2-3 | MTS staging deployment (VPS + Docker Compose + ai-net) | US-028 | P0 | 2 |
| 3-4 | Integration tests (tenant isolation, SOUL routing, AI fallback) | US-029 | P1 | 3 |
| 4 | Token cost tracking verify (CTO ISSUE-B resolution) | US-030 | P1 | 1 |
| 4-5 | MTS pilot: 10 employees onboard to Telegram bot | US-031 | P0 | 1 |
| 5 | G3 gate proposal | US-033 | P0 | 1 |

**Success Criteria**: 3/10 WAU, PR Gate processes first real PR, cost tracking operational

---

## Phase 2: Governance Hardening (Sprint 6-8)

### Sprint 6 — NQH Tenant + Rail #3 Knowledge + SOUL-Aware RAG ✅ COMPLETE

**Duration**: 5 days
**Status**: COMPLETE — CTO 8.0/10 APPROVED

| Day | Deliverable | Priority | Points |
|-----|------------|----------|--------|
| 1 | NQH tenant configuration (owner_id='nqh', RLS verified) | P0 | 2 |
| 1-2 | Zalo channel integration | P0 | 2 |
| 2-3 | **SOUL-Aware RAG Routing (Context Drift Layer B)** | P0 | 3 |
| 2-3 | **Team mention routing + charters** | P1 | 3 |
| 3-4 | RAG collections: engineering + sales + NQH-SOPs | P1 | 4 |
| 4-5 | Tenant cost guardrail + cross-tenant isolation regression | P1 | 3 |

---

### Sprint 7 — Rail #1 Spec Factory Full + Retrieval Evidence ✅ COMPLETE

**Duration**: 5 days
**Status**: COMPLETE — CTO 8.0/10 APPROVED

| Day | Deliverable | Priority | Points |
|-----|------------|----------|--------|
| 1-2 | Spec Factory v1.0: spec_id, BDD scenarios, risk scoring + migration | P0 | 3 |
| 2-3 | Evidence vault link (spec → trace → bidirectional query) | P0 | 2 |
| 3 | **Retrieval Evidence logging (Context Drift Layer C)** | P0 | 2 |
| 3-4 | Spec Telegram commands: /spec-list, /spec-detail | P1 | 2 |
| 4-5 | Gateway consumer refactoring (CTO-14: 5 modules) + SOUL drift detection | P1 | 4 |

---

### Sprint 8 — Rail #2 PR Gate ENFORCE + G4 ✅ COMPLETE

**Gate**: G4 (Validation Ready) — **[@cto] APPROVED 2026-03-17**
**Duration**: 5 days
**Status**: COMPLETE — **CTO 8.5/10 APPROVED** (2026-03-04)

| Deliverable | Status | CTO Verdict |
|-------------|--------|-------------|
| PR Gate ENFORCE (GitHub webhook + HMAC + commit status) | ✅ | EXCELLENT |
| pr_gate_evaluations table + RLS (migration 000015) | ✅ | GOOD |
| Context Drift E2E validation (5 tests, 16 subtests) | ✅ | EXCELLENT |
| SOUL behavioral suite — 5 critical SOULs × 5 tests | ✅ | EXCELLENT |
| Evidence export API (JSON + CSV, bearer auth) | ✅ | GOOD |

---

## Phase 3: Scale (Sprint 9-12)

### Sprint 9 — Channel Rationalization + SOUL Suite Complete ✅ COMPLETE

**Duration**: 5 days
**Status**: COMPLETE — CTO 9.0/10 APPROVED

| ID | Task | Priority | Points |
|----|------|----------|--------|
| T9-01 | Channel removal core: feishu/discord/whatsapp (~2,836 LOC) | P0 | 3 |
| T9-02 | Channel removal periphery: onboard/agent/tools/managed mode | P0 | 2 |
| T9-03 | SOUL behavioral tests: 12 governance SOULs × 5 = 60 tests (85 total) | P0 | 2 |
| T9-04 | MS Teams scaffold + ADR-007 draft | P1 | 1 |
| T9-05 | G4 gate proposal | P0 | 1 |

**Test count at close**: 350 PASS

---

### Sprint 10 — MS Teams Extension + NQH Corporate Rollout ✅ COMPLETE

**Gate**: None (mid-phase)
**Status**: COMPLETE — CTO review pending
**Owner**: [@coder] + [@devops] (Azure AD) + [@pm] (NQH comms)
**Points**: ~12
**ADR**: ADR-007 APPROVED [@cto] 2026-03-17
**Detailed plan**: [SPRINT-010 plan](../04-build/sprints/SPRINT-010-MSTeams-NQH-Corporate.md)
**Completion**: [SPRINT-010 completion](../04-build/SPRINT-010-COMPLETION.md)

| Deliverable | Status | Notes |
|------------|--------|-------|
| `extensions/msteams/` — 7 files | ✅ | Full JWKS impl (CTO-35) |
| Bot Framework JWT verification (OpenID → JWKS → `rsa.PublicKey`, 24h cache) | ✅ | `globalJWKSCache` + kid-miss refresh |
| Bot Framework token acquisition (`client_credentials` OAuth2, 10s timeout) | ✅ | CTO-39 |
| Adaptive Cards: `SpecCard()` + `PRReviewCard()` | ✅ | CTO-36 |
| Migration 000016 (`channel` column in governance tables) | ✅ | CTO-37 |
| `MSTEAMS_APP_PASSWORD` masking via `maskNonEmpty()` | ✅ | CTO-38 |
| CTO-33: Discord residuals removed | ✅ | Sprint 9 carryover |
| 16 unit tests — 366 total PASS | ✅ | |

**Pending**: Azure AD live credentials ([@devops]) — unit tests use mock credentials, PASS.

---

### Sprint 11 — Hardening: Evidence Chain + Pen Test + Audit Trail → NEXT

**Duration**: 5 days
**Gate**: None (hardening sprint) — G5 structure filed
**Status**: PLANNED — pending CTO Sprint 10 review
**Owner**: [@coder] + [@tester] (pen test) + [@pm] (G4 close-out)
**Points**: ~12
**Entry Criteria**: CTO Sprint 10 score; G4 WAU ≥7/10 (closes 2026-03-31); @cpo+@ceo G4 co-sign; Azure AD provisioned
**Detailed plan**: [SPRINT-011 plan](../04-build/sprints/SPRINT-011-Hardening.md)
**New ADRs**: ADR-008 (PDF Library: maroto), ADR-009 (Evidence Linking: junction table)

| ID | Task | Priority | Points | Owner |
|----|------|----------|--------|-------|
| T11-00 | Azure AD live E2E verification (carry from Sprint 10) | P0 | 0 | [@devops] |
| T11-01 | Cross-rail evidence linking: `evidence_links` table (ADR-009) + auto-link `/spec`→`/review` + chain API | P0 | 3 | [@coder] |
| T11-02 | Security pen test: 6 vectors (RLS bypass, cross-tenant, SOUL injection, JWT forge, drift bypass, token exhaustion) | P1 | 3 | [@tester]+[@coder] |
| T11-03 | Audit trail PDF export: `maroto` v2 (ADR-008), `GET /spec/{id}/audit-trail.pdf`, SOC2/ISO27001 format | P1 | 3 | [@coder] |
| T11-04 | Performance baseline: RAG p95, DB EXPLAIN, API latency documented | P2 | 1 | [@coder] |
| T11-05 | Post-mortem Sprint 1-11 + G5 gate proposal structure | P1 | 2 | [@pm] |

**[@pm] parallel**: G4 WAU final measurement (2026-03-31) + @cpo/@ceo co-sign drive (Day 2)

**Architecture decisions ([@architect])**:
- **Evidence linking**: `evidence_links` junction table (Option B) — N:M, extends to test_run + deploy in Sprint 12. NOT `spec_id FK` on pr_gate_evaluations (would lock to 1:N).
- **PDF**: `johnfercher/maroto` v2 (MIT, no CGO) — rejected puppeteer (external binary) and unipdf (commercial).
- **Pen test PT-03** (SOUL injection): manual inspection; automation via SOUL drift checksum in Sprint 12.

**Test count target**: 366 → ≥400 (+pen test ~15 + evidence ~12 + PDF ~7)

**Key outputs**:
- `docs/05-test/SECURITY-PENTEST-SPRINT11.md` — 6 vectors + CVSS scores
- `docs/05-test/PERFORMANCE-BASELINE-SPRINT11.md` — p95 baselines
- `docs/09-govern/01-CTO-Reports/POST-MORTEM-SPRINT-1-11.md`
- `docs/08-collaborate/G5-GATE-PROPOSAL-STRUCTURE.md`
- `docs/09-govern/01-CTO-Reports/G4-WAU-TRACKING.md` — Day 14 row closed

---

### Sprint 12 — OaaS Preparation + MTClaw Dogfooding (Planned)

**Duration**: 5 days
**Gate**: G5 (Scale Ready — OaaS)
**Status**: PLANNED
**Owner**: [@coder] + [@pm] (pricing/docs) + [@architect] (self-dev design)
**Points**: ~12
**Entry Criteria**: Sprint 11 COMPLETE (CTO score); G4 fully co-signed; G5 structure approved [@cto]; Azure AD live for NQH

| ID | Task | Priority | Points | Notes |
|----|------|----------|--------|-------|
| T12-01 | Multi-tenant self-service: registration API + admin panel (new tenant → working bot <30 min) | P0 | 3 | OaaS core |
| T12-02 | **MTClaw self-development tools: file system tools + code execution sandbox** | P1 | 3 | Dogfooding: MTClaw builds MTClaw via Telegram |
| T12-03 | Pricing model: token-based tiers (Starter/Growth/Enterprise) | P1 | 2 | Revenue enabler |
| T12-04 | SOUL marketplace design: F&B, retail, tech industry personas | P2 | 2 | OaaS differentiator |
| T12-05 | Tenant admin guide + API reference + G5 gate proposal | P1 | 2 | Gate + docs |

**T12-02 — MTClaw Dogfooding Architecture** ([@architect]):

```
Sprint 12 target state:
  Developer sends message in Telegram: "@coder implement auth.go"
        ↓
  MTClaw @coder SOUL receives task
        ↓
  file_read("/home/nqh/shared/MTClaw/cmd/gateway.go") → context
  file_write("/home/nqh/shared/MTClaw/extensions/auth/auth.go") → creates file
        ↓
  code_exec("go build ./...") → returns build result
  code_exec("go test ./extensions/auth/... -v") → returns test output
        ↓
  Response: "auth.go created. Build: ✅ 0 errors. Tests: 3/3 PASS."
```

**Required capabilities** (not in Sprint 11):
- `file_read` / `file_write` tools (scoped to MTClaw repo path)
- `code_exec` tool (runs in Docker sandbox, isolated from host)
- `git_status` / `git_commit` tools (scoped to MTClaw repo)

**Security**: code_exec runs in ephemeral Docker container, no host filesystem access except repo bind mount.

**Gate G5 criteria** (full proposal filed T12-05):
- Multi-tenant: new tenant → working bot <30 min
- WAU ≥15/10 for MTS + NQH combined
- Pen test Sprint 11: all 6 vectors PASS
- Evidence chain: spec → PR → test → deploy linkable
- Pricing model defined + approved [@ceo]
- Legal: terms of service draft reviewed

---

## Sprint-by-Sprint Summary

| Sprint | Phase | Goal | Gate | Tests | Users | Channels |
|--------|-------|------|------|-------|-------|----------|
| 1 ✅ | Foundation | Init + GoClaw + 16 SOULs | G0.1 ✅ | — | 0 | — |
| 2 ✅ | Foundation | Requirements + Design | G0.2 ✅ | — | 0 | — |
| 3 ✅ | Foundation | Architecture + RLS | G2 ✅ (9.2/10) | — | 0 | — |
| 4 ✅ | First Rails | /spec + Context Anchoring + @mention | — (9.0/10) | — | ~3 | Telegram |
| 5 ✅ | First Rails | MTS Pilot + PR Gate WARNING | G3 ✅ | — | 10 | Telegram |
| 6 ✅ | Governance | NQH pilot + Knowledge/RAG + Team routing | — (8.0/10) | — | 30 | Telegram + Zalo |
| 7 ✅ | Governance | Spec Factory full + Retrieval Evidence | — (8.0/10) | — | ~80 | Telegram + Zalo |
| 8 ✅ | Governance | PR Gate ENFORCE + G4 | G4 ✅ [@cto] | 290 | ~160 | Telegram + Zalo |
| 9 ✅ | Scale | Channel cleanup + SOUL x17 + MS Teams scaffold | — (9.0/10) | 350 | ~160 | Telegram + Zalo |
| 10 ✅ | Scale | MS Teams + NQH corporate rollout | — (pending CTO) | 366 | ~200 | + Teams |
| **11 →** | **Scale** | **Hardening + pen test + audit trail PDF** | **—** | **≥400** | **~200** | **3 channels** |
| 12 📋 | Scale | OaaS prep + MTClaw self-development (dogfooding) | G5 | TBD | Expand | Multi-channel |

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
           │                                   ┌──────────┤
           │                                   ▼          ▼
           │                              Sprint 9 ──► Sprint 10 ──► Sprint 11 ──► Sprint 12
           │                              (cleanup)    (MSTeams)     (hardening)   (OaaS+dogfood)
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
| G3 | 5 | Build Ready (MTS Pilot) | [@cto], [@cpo] | ✅ APPROVED |
| **G4** | **8→9** | **Validation Ready (3 Rails)** | **[@cto], [@cpo], [@ceo]** | **[@cto] ✅ 2026-03-17 — [@cpo]+[@ceo] co-sign pending** |
| G5 | 12 | Scale Ready (OaaS) | [@cto], [@cpo], [@ceo] | ⏳ Structure filed Sprint 11 |

---

## Resource Plan

| Sprint | Primary Owner | Secondary | CTO Review | Status |
|--------|--------------|-----------|------------|--------|
| 1-2 | [@pm] | [@researcher] | Gate reviews | ✅ Complete |
| 3 | [@architect] | [@coder] | G2 gate | ✅ Complete (9.2/10) |
| 4 | [@coder] | [@pm] | Sprint review | ✅ Complete (9.0/10) |
| 5 | [@coder] | [@pm] (pilot) | PR Gate design | ✅ Complete |
| 6 | [@coder] | [@devops] + [@pm] | Sprint review | ✅ Complete (8.0/10) |
| 7 | [@coder] | [@pm] | Sprint review | ✅ Complete (8.0/10) |
| 8 | [@coder] | [@pm] (G4) | Sprint review | ✅ Complete (8.5/10) |
| 9 | [@coder] | [@pm] (G4 + ADR-007) | ADR-006 APPROVED | ✅ Complete (9.0/10) |
| 10 | [@coder] | [@devops] (Azure AD) + [@pm] | MS Teams review | ✅ Complete (pending score) |
| **11** | **[@coder]** | **[@tester] (pen test) + [@pm] (G4 close)** | **Hardening review** | **← NEXT** |
| 12 | [@coder] | [@architect] (dogfood design) + [@pm] (OaaS) | G5 gate | 📋 Planned |

---

## Risk Register (Cross-Sprint)

| # | Risk | Prob | Impact | Mitigation | Sprint | Status |
|---|------|------|--------|------------|--------|--------|
| R1 | RLS breaks existing GoClaw queries | Med | High | Test all 55 endpoints after migration | 3 | ✅ Resolved |
| R2 | Go competency gap delays implementation | Med | Med | AI Codex + CTO review gate (ADR-001) | 3-5 | ✅ Mitigated |
| R3 | Bflow AI-Platform latency >5s | Low | Med | Fallback design in ADR-005 | 4 | Active |
| R4 | MTS adoption <30% WAU by Sprint 5 | Med | High | SOUL feedback, iterate | 4-5 | Active |
| R5 | SOUL context exceeds LLM context window | Low | High | 2,000 char budget (CTO-3) | 3 | ✅ Mitigated |
| R6 | NQH CEO approval delayed | Med | Med | CEO Option A: MTS-only | 6 | ✅ Resolved |
| R7 | PR Gate false positives | Med | Med | WARNING mode first, tune rules | 5-8 | Active |
| R8 | RAG accuracy below threshold | Med | High | Manual curation Phase 1 | 6 | Active |
| R9 | Context drift in long conversations | Med | High | 3-layer prevention (FR-008) | 4-7 | ✅ Validated Sprint 8 |
| R10 | AI-Platform single point of failure | Low | High | Graceful degradation (ADR-005) | 4+ | Active |
| R14 | Team mention conflicts with agent key | Low | Low | Agent-first resolution (EndiorBot pattern) | 6 | ✅ Resolved |
| R15 | Channel removal breaks hidden dependencies | Med | Med | Phased removal, 290-test regression gate | 9 | ✅ Resolved |
| R16 | MS Teams Azure AD OAuth2 delays Sprint 10 | Med | Med | Scaffold Sprint 9, mocks Sprint 10 | 10 | ⚠️ Azure AD still pending [@devops] |
| R17 | G4 WAU <7/10 at 2026-03-31 | Med | High | Intervention plan in G4-WAU-TRACKING.md | 11 | Active — measure Day 5 |
| R18 | SOUL injection (PT-03) hard to automate | High | Med | Manual Sprint 11; automated checksum Sprint 12 | 11 | NEW |
| R19 | RAG p95 > 3s baseline | Med | Med | Document in Sprint 11; fix Sprint 12 | 11-12 | NEW |
| R20 | Dogfooding file tools scope creep | Med | Med | Scope to MTClaw repo only; sandbox code_exec | 12 | NEW |

---

## Cross-Sprint: Context Drift Prevention (FR-008)

| Layer | Sprint | What | Status |
|-------|--------|------|--------|
| Design | 3 ✅ | 3-layer architecture in SAD Section 8 | ✅ Done |
| A: Anchoring | 4 ✅ | Session goal + SOUL identity → ExtraPrompt | ✅ Done |
| B: Retrieval | 6 ✅ | SOUL-Aware RAG routing + role-aware ranking | ✅ Done |
| C: Evidence | 7 ✅ | RetrievalEvidence logging with ranking_reason | ✅ Done |
| Validate | 8 ✅ | Full E2E: 5 tests, 16 subtests | ✅ G4 validated |

---

## Changelog

| Version | Date | Changes |
|---------|------|---------|
| 2.7.0 | 2026-03-22 | Sprint 12 expanded: OaaS + MTClaw Dogfooding (file tools + code exec sandbox). Sprint 11 detailed: evidence_links ADR-009, maroto ADR-008, pen test 6 vectors, post-mortem. Roadmap extended to 12 sprints. |
| 2.6.0 | 2026-03-22 | Sprint 10 COMPLETE (366 tests, 6 CTO issues resolved). Sprint 11 placeholder added. |
| 2.5.0 | 2026-03-17 | Sprint 9 COMPLETE (9.0/10). Sprint 10 entry criteria set. ADR-007 APPROVED. G4 @cto approved. |
| 2.4.0 | 2026-03-04 | Sprint 8 COMPLETE (8.5/10). G4 proposal filed Sprint 9. |

---

## References

- [Product Vision](../00-foundation/product-vision.md)
- [Requirements](requirements.md) (v1.1.0 — includes FR-008)
- [API Specification](api-specification.md)
- [System Architecture Document](../02-design/system-architecture-document.md)
- [Sprint 11 Plan](../04-build/sprints/SPRINT-011-Hardening.md) ← NEW v2.7.0
- [G4 Gate Proposal](../08-collaborate/G4-GATE-PROPOSAL-SPRINT8.md)
- [G4 WAU Tracking](../09-govern/01-CTO-Reports/G4-WAU-TRACKING.md)
- [ADR-007 MS Teams](../02-design/01-ADRs/SPEC-0007-ADR-007-MSTeams-Extension.md)
- [ADR-008 PDF Library](../02-design/01-ADRs/SPEC-0008-ADR-008-PDF-Library.md) ← NEW Sprint 11
- [ADR-009 Evidence Linking](../02-design/01-ADRs/SPEC-0009-ADR-009-Evidence-Linking-Schema.md) ← NEW Sprint 11
