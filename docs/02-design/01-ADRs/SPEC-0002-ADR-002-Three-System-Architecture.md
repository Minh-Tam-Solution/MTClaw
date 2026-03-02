# ADR-002: Three-System Architecture

**SPEC ID**: SPEC-0002
**Status**: ACCEPTED
**Date**: 2026-03-02
**Deciders**: [@cto], [@cpo], [@pm]

---

## Context

MTClaw sits between two existing systems:
- **EndiorBot**: TypeScript CLI, 12 SOUL agents, ActionControlPlane, Vibecoding Index — CEO power tool
- **SDLC-Orchestrator**: Python FastAPI, 117 endpoints, OPA policy engine, Evidence Vault — governance platform

MTClaw must leverage patterns from both without creating runtime coupling.

## Decision

**3-System Architecture with zero runtime coupling.**

```
┌──────────────────────────────────────────────────────┐
│  EndiorBot (reference only — zero runtime dependency) │
│  ✅ Port SOUL templates (copy 12 .md files)           │
│  ✅ Port skill LOGIC → re-implement in Go             │
│  ❌ KHÔNG gọi EndiorBot CLI từ MTClaw runtime         │
├──────────────────────────────────────────────────────┤
│  ★ MTClaw (Governance-First Assistant Platform)  ★    │
│  GoClaw runtime + 16 SOULs + governance skills        │
│  Channels: Telegram (P1) → Zalo (P2)                  │
│  AI: Bflow AI-Platform ONLY (single source of truth)   │
├──────────────────────────────────────────────────────┤
│  SDLC-Orchestrator (pattern reference only)            │
│  ✅ Copy patterns, adapt for Go, independent lifecycle │
│  ❌ KHÔNG import code, KHÔNG API call runtime          │
│  Gate eval = lightweight Go logic, NOT full OPA engine  │
└──────────────────────────────────────────────────────┘
```

## Integration Mechanisms (CTO Required)

| Integration | Phase 1-2 | Phase 3 | Mechanism |
|-------------|-----------|---------|-----------|
| EndiorBot → MTClaw | No integration | **Go re-implementation** of skill logic (~2-3 sprints) | Port logic, NOT CLI call (latency + coupling concern) |
| SDLC-Orchestrator → MTClaw | Pattern reference only | Pattern reference only | Copy design patterns, never code/API dependency |
| Bflow AI-Platform → MTClaw | HTTP API | Same | OpenAI-compatible `/v1/chat/completions` + custom `/v1/rag/query` |
| MTClaw gate evaluation | Lightweight Go conditions | Same | Simple rule checks, NOT OPA engine |

### Bflow AI-Platform Integration

**Single source of AI infra truth. No bypass allowed.**

```
MTClaw → HTTP POST → api.nhatquangholding.com
Headers: X-API-Key, X-Tenant-ID
Endpoints:
  POST /v1/chat/completions  (OpenAI-compatible)
  POST /v1/rag/query          (RAG with collection filter)
  POST /v1/rag/ingest/batch   (batch document ingestion)
```

### Coupling Rules

| Rule | Enforcement |
|------|------------|
| No EndiorBot CLI calls | Code review + grep CI check |
| No SDLC-Orchestrator API calls | Code review + grep CI check |
| No direct LLM calls (bypass AI-Platform) | Code review + env var check (no OPENAI_API_KEY, no ANTHROPIC_API_KEY) |
| No OPA engine in MTClaw | ADR reference in PR template |

## 3 Rails Governance (Phased)

| Rail | Phase | Sprint |
|------|-------|--------|
| #1 Spec Factory (`/spec` → JSON → evidence) | Prototype | Sprint 4 |
| #1 Spec Factory (full: spec_id, BDD, risk) | Full | Sprint 7 |
| #2 PR Gate (WARNING mode) | Initial | Sprint 5 |
| #2 PR Gate (ENFORCE mode) | Full | Sprint 8 |
| #3 Knowledge (RAG per domain, SOUL per role) | Initial | Sprint 6 |

## 16 SOULs

- **12 SDLC SOULs** (from EndiorBot): pm, architect, coder, reviewer, researcher, writer, pjm, devops, tester, cto, cpo, ceo
- **4 MTS Business SOULs** (from MTS-OpenClaw): mts-dev, mts-sales, mts-cs, mts-general
- **All ported Day 1** as markdown files → loaded at startup → injected as system prompt prefix

## Consequences

### Positive
- Zero runtime coupling = independent deployment lifecycle
- Each system can evolve independently
- MTClaw inherits battle-tested patterns without inheriting tech debt
- Bflow AI-Platform centralizes cost control and model management

### Negative
- Pattern duplication (Go re-implementation of EndiorBot skills)
- No real-time sync between systems
- Manual pattern updates when source systems evolve

---

## References
- [ROLE_TOOL_MATRIX](../../08-collaborate/01-SDLC-Compliance/ROLE_TOOL_MATRIX.md)
- [ADR-004: SOUL Implementation](SPEC-0004-ADR-004-SOUL-Implementation.md)
- EndiorBot: `/home/nqh/shared/EndiorBot/`
- SDLC-Orchestrator: `/home/nqh/shared/SDLC-Orchestrator/`
