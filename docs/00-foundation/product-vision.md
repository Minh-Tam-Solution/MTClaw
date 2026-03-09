# Product Vision — MTClaw

**SDLC Stage**: 00-Foundation
**Version**: 1.0.0
**Date**: 2026-03-02
**Author**: [@pm]
**Approved**: [@ceo] (Priority C directive), [@cto] (G0.1 8.5/10), [@cpo] (G0.1 8/10)

---

## Vision Statement

> **MTClaw is the governance backbone for AI-first company transformation.**
>
> Every employee gets a role-aware AI assistant that doesn't just answer questions —
> it enforces quality rails, produces auditable evidence, and connects to the company's
> knowledge base. From spec writing to code review to SOP lookup, every AI action
> passes through governance rails that reduce feature waste and increase accountability.

---

## One-Liner

**MTClaw = 16 SOUL personas + 3 governance rails + multi-tenant isolation, powered by Bflow AI-Platform.**

---

## Why Now

1. **CEO Priority C**: Governance backbone for AI-first transformation (decided 2026-03-05)
2. **AI access gap**: 8/8 MTS employees say `chat.nhatquangholding.com` is generic — no Bflow context, no persistent memory, no role awareness
3. **Feature waste crisis**: Industry-standard ~60% feature waste rate; governance rails target <30%
4. **Infrastructure ready**: Bflow AI-Platform (qwen3:14b) operational, GoClaw runtime proven, 16 SOULs already ported

---

## Target Users

### Phase 1: MTS (10 employees) — Sprint 1-8

| Persona | Count | Primary Need | SOUL |
|---------|-------|-------------|------|
| Engineering | 4 | Code review, spec writing, Bflow API docs | `enghelp`, `coder`, `reviewer` |
| Sales | 3 | Proposal drafts, pricing lookup, case studies | `sales` |
| Customer Service | 2 | SOP lookup, ticket resolution templates | `cs` |
| Back Office (HR/Admin) | 1 | HR policy Q&A, meeting minutes, contracts | `assistant` (default) |

### Phase 2: NQH (100-150 employees) — Sprint 6+ (conditional)

| Persona | Count | Primary Need | Channel |
|---------|-------|-------------|---------|
| F&B Operations | ~80 | Daily SOPs, checklists, workflow triggers | Zalo |
| Hotel Management | ~40 | Guest service SOPs, inventory, reporting | Zalo |
| NQH HO | ~30 | HR, accounting, compliance | Zalo + Telegram |

---

## 3 Governance Rails

### Rail #1: Spec Factory (`/spec`)

**Problem**: Requirements are written ad-hoc, inconsistent, not auditable.
**Solution**: `/spec` command transforms natural language into structured JSON specifications with evidence attachment.

```
Input:  "Tạo tính năng đăng nhập cho Bflow mobile app"
Output: JSON spec with title, narrative (As a/I want/So that),
        acceptance criteria (Given/When/Then), priority, effort estimate
Evidence: trace_id links to full generation audit trail
```

**Delivery**: Sprint 4 (prototype) → Sprint 7 (full with spec_id, BDD, risk scoring)

### Rail #2: PR Gate

**Problem**: Code reviews are inconsistent, security issues slip through, no quality baseline.
**Solution**: AI-powered PR review with structured checklist (SQL injection, RLS compliance, test coverage).

```
Input:  PR URL + review scope
Output: Structured review with verdict (PASS/PASS_WITH_WARNINGS/FAIL),
        findings (severity, category, file, line), score (0-100)
Mode:   WARNING (Sprint 5) → ENFORCE (Sprint 8)
```

**Delivery**: Sprint 5 (WARNING mode) → Sprint 8 (ENFORCE mode)

### Rail #3: Knowledge & Answering (RAG)

**Problem**: Company knowledge is scattered across Notion, Google Drive, personal documents. Employees waste 25-30 min/day searching.
**Solution**: Domain-specific RAG collections with SOUL-aware answers.

```
Collections:
  - engineering: Bflow API docs, coding standards, architecture
  - sales: Pricing, proposals, case studies
  - hr-policies: HR policies, contracts, org info
  - nqh-sops: 805 SOP documents (Phase 2)
```

**Delivery**: Sprint 6 (MTS collections) → Sprint 7 (NQH collections if Phase 2)

---

## 16 SOULs

### Design Principle

Users think in **tasks**, not in SOULs. SOUL routing is:
- **Automatic** for context detection (HR question → `assistant` handles directly)
- **Explicit** only for power users (`@reviewer`, `@pm`)
- **Never requires restart** — switch mid-conversation

### SOUL Map

```
┌─────────────────────────────────────────────────────────┐
│                    16 SOULs                               │
├───────────────────────┬─────────────────────────────────┤
│  13 SDLC Governance   │  3 Business (tenant-agnostic)    │
│                       │                                  │
│  ★ assistant (Router)  │  enghelp (Engineering daily)     │
│  ★ pm (Spec Factory)  │  sales (Proposals, pricing)      │
│  ★ reviewer (PR Gate) │  cs (SOPs, tickets)              │
│  ★ coder              │                                  │
│    architect          │  ★ = active by default           │
│    researcher         │                                  │
│    writer             │                                  │
│    pjm                │                                  │
│    devops             │                                  │
│    tester             │                                  │
│    cto (advisor)      │                                  │
│    cpo (advisor)      │                                  │
│    ceo (advisor)      │                                  │
├───────────────────────┴─────────────────────────────────┤
│  Default entry point: assistant (is_default=true)        │
│  Universal router: handles daily tasks + SDLC delegation │
│  Routing: assistant → auto-detect → handle or delegate   │
└─────────────────────────────────────────────────────────┘
```

---

## Context Integrity (Anti-Drift Design)

MTClaw SOULs are AI agents operating in long conversations. Two failure modes must be prevented:

| Problem | Risk | MTClaw Mitigation |
|---------|------|-------------------|
| **Context Drift** | SOUL forgets role identity or session goals after 50-100K tokens | Identity anchoring (SOUL.md always injected) + session goal re-injection |
| **Semantic Blindness** | SOUL answers without domain knowledge | SOUL-aware RAG routing (each SOUL → its domain collection) |

**Design principle**: Every SOUL response must be grounded in (1) its role identity, (2) current session objectives, and (3) domain knowledge from RAG — not just raw LLM generation.

**Reference**: Adapted from EndiorBot Sprint 63-65 battle-tested patterns (TS-007, ADR-009, ADR-015).

---

## Success Metrics (North Star)

| Metric | Current | Sprint 4 | Sprint 6 | Sprint 8 | Sprint 10 |
|--------|---------|----------|----------|----------|-----------|
| SOUL adoption (MTS WAU) | 0% | 30% (3/10) | 60% (6/10) | 70% (7/10) | 80% (8/10) |
| Feature waste rate | ~60% | — | — | <30% | <30% |
| Governance rails running | 0 | 1 (Spec) | 2 (+Knowledge) | 3 (+PR Gate ENFORCE) | 3 |
| Evidence capture rate | 0% | — | 50% | 100% gated | 100% |
| Time saved per employee/week | 0h | 1h | 2h | 3h | 3h+ |
| Spec quality (auto, 0-100) | — | Baseline | — | 70+ avg | 80+ avg |
| Context retention (SOUL identity) | — | — | 85%+ | 90%+ | 95%+ |

---

## Non-Goals (Scope Guard)

MTClaw is **NOT**:

1. **Not a replacement for Bflow** — Bflow is the ERP platform for customers. MTClaw is the internal governance assistant.
2. **Not a replacement for NQH-Bot** — NQH-Bot is Workforce Management (WFM) AI for F&B operations. MTClaw is a complementary Telegram/Zalo layer.
3. **Not a general chatbot** — Every AI action passes through governance rails. No "free chat" without SOUL context.
4. **Not an OSS product** — Proprietary internal platform for MTS/NQH. Built on MIT-licensed GoClaw but not published externally.
5. **Not a CI/CD tool** — PR Gate evaluates quality, but doesn't replace GitHub Actions or CI pipelines.

---

## Strategic Trajectory

```
Phase 1 (Sprint 1-8): MTS Internal
  → Prove governance value with 10 employees
  → 3 Rails running, evidence trail complete
  → Foundation for multi-tenant

Phase 2 (Sprint 6-8): NQH Expansion (conditional)
  → 100-150 NQH employees via Zalo
  → NQH-SOPs RAG (805 docs already indexed)
  → Validate multi-tenant isolation

Phase 3 (Sprint 9+): OaaS Foundation
  → Multi-tenant self-service
  → Governance-as-a-Service offering
  → Revenue from external tenants
  → SOUL marketplace (industry-specific personas)
```

---

## Technology Stack

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Runtime | GoClaw (Go 1.25) | Multi-tenant PostgreSQL, single binary, proven at scale |
| Database | PostgreSQL 15 + pgvector | RLS, hybrid RAG (vector + BM25), JSONB config |
| AI Provider | Bflow AI-Platform (`api.nhatquangholding.com`) | Single source of truth, $0/query, qwen3:14b |
| Channel Phase 1 | Telegram | Tech-savvy MTS team, proven OpenClaw integration |
| Channel Phase 2 | Zalo | Most popular app in VN, F&B/hospitality staff |
| Observability | slog + OTEL + Prometheus | Structured JSON logging, tenant-scoped metrics |
| Security | RLS + AES-256-GCM + JWT | Row-level tenant isolation, encrypted secrets |

---

## References

- [Problem Statement](problem-statement.md)
- [Business Case](business-case.md)
- [Requirements](../01-planning/requirements.md)
- [User Journey Map](../01-planning/user-journey-map.md)
- [ADR-002: Three-System Architecture](../02-design/01-ADRs/SPEC-0002-ADR-002-Three-System-Architecture.md)
- [G0.1 Gate Proposal](G0.1-GATE-PROPOSAL.md)
- [G0.2 Gate Proposal](G0.2-GATE-PROPOSAL.md)
