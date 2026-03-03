# Requirements — MTClaw

**SDLC Stage**: 01-Planning
**Version**: 1.1.0
**Date**: 2026-03-02
**Author**: [@pm]

---

## Functional Requirements

### FR-001: Multi-Tenant Architecture
- PostgreSQL Row-Level Security (RLS) mandatory
- Middleware injects `tenant_id` from auth context
- Phase 1: MTS tenant only
- Phase 2: NQH tenant (conditional on CEO decision)
- Each tenant has isolated data, SOUL config, and cost limits

### FR-002: 16 SOULs (Role-Aware AI Personas)
- 13 SDLC SOULs: pm, architect, coder, reviewer, researcher, writer, pjm, devops, tester, cto, cpo, ceo, assistant (universal router)
- 3 Business SOULs (tenant-agnostic): dev, sales, cs
- Stored as Git markdown files with YAML frontmatter
- Loaded at startup, cached in memory, reloadable via SIGHUP
- 6 active by default, 10 on-demand (see ROLE_TOOL_MATRIX)

### FR-003: 3 Rails Governance
- **Rail #1 — Spec Factory**: `/spec` command → structured JSON → evidence attachment
  - Sprint 4: prototype (basic /spec → JSON)
  - Sprint 7: full (spec_id, BDD, risk, evidence link)
- **Rail #2 — PR Gate**: Policy evaluation on pull requests
  - Sprint 5: WARNING mode (report issues, don't block)
  - Sprint 8: ENFORCE mode (block merge on policy violation)
- **Rail #3 — Knowledge & Answering**: RAG per domain, SOUL per role
  - Sprint 6: MTS engineering docs + NQH-SOPs (if Phase 2 approved)

### FR-004: Bflow AI-Platform Integration
- Single source of AI infra truth — no bypass allowed
- OpenAI-compatible API: `POST /v1/chat/completions`
- RAG API: `POST /v1/rag/query` with collection filter
- Auth: `X-API-Key` + `X-Tenant-ID` headers
- Fallback: graceful degradation if AI-Platform unavailable

### FR-005: Messaging Channels
- Phase 1: Telegram (primary — tech-savvy MTS team)
- Phase 2: Zalo (primary for NQH — most popular app in VN for F&B staff)
- Channel abstraction layer for future channels

### FR-006: Evidence Audit Trail
- Every governance action produces evidence record
- Fields: action, actor, tenant_id, soul_role, input, output, timestamp, trace_id
- Immutable append-only storage
- Queryable for compliance reporting

### FR-007: Observability
- Structured JSON logging (slog) with trace_id, tenant_id
- OTEL metrics → Prometheus
- Token cost tracking per tenant per SOUL
- Tenant cost guardrails (monthly token limit, daily request limit)

### FR-008: Context Drift & Semantic Blindness Prevention
- **Context Anchoring**: SOUL identity (SOUL.md + IDENTITY.md) always injected in system prompt to prevent role confusion in long conversations
- **Session Goal Anchoring**: Per-session objectives re-injected on every turn to prevent objective drift after 50-100K tokens
- **SOUL-Aware Retrieval**: RAG queries routed to domain-specific collections based on active SOUL role (dev → engineering, sales → sales)
- **Retrieval Evidence**: Every RAG retrieval logged with ranking_reason enum for auditability
- **Token Budget**: Hard cap 2,500 tokens per retrieval to prevent context window overflow
- **Reference**: EndiorBot TS-007, ADR-009, ADR-015 (battle-tested patterns adapted for Go/GoClaw)

## Non-Functional Requirements

| NFR | Requirement | Target |
|-----|------------|--------|
| NFR-001 | Memory baseline | <35MB RAM |
| NFR-002 | Startup time | <1s |
| NFR-003 | API latency (p95) | <5s (AI response), <100ms (non-AI) |
| NFR-004 | Binary size | <30MB |
| NFR-005 | Encryption | AES-256-GCM for sensitive fields |
| NFR-006 | Auth | JWT + tenant context |
| NFR-007 | Availability | 99.5% (single VPS target) |
| NFR-008 | Concurrency | 100 concurrent users (Phase 1: 10 active) |

## Test Coverage Targets (Tiered)

| Sprint | Unit Coverage | Integration |
|--------|-------------|-------------|
| 1-3 | 60% | Scenario-based checklist |
| 4-5 | 70% | All critical paths |
| 8+ | 80% | Full regression suite |

---

**Gate**: G0.2 (Sprint 2) — Full requirements review
**References**: [Problem Statement](../00-foundation/problem-statement.md), [ADR-002](../02-design/01-ADRs/SPEC-0002-ADR-002-Three-System-Architecture.md)
