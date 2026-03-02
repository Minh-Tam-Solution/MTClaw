# Problem Statement — MTClaw

**SDLC Stage**: 00-Foundation
**Version**: 1.0.0
**Date**: 2026-03-02
**Author**: [@pm]

---

## Who

- **Primary users (Phase 1)**: 10 MTS employees (Engineering, Sales, CS, Back Office)
- **Secondary users (Phase 2)**: 100-150 NQH employees (F&B operations, hospitality staff)

## Current State

MTS-OpenClaw exists as a chat tool — it connects to AI providers and delivers responses via Telegram. However:

1. **No governance rails**: No structured spec output, no PR gate evaluation, no evidence audit trail
2. **Generic AI responses**: chat.nhatquangholding.com provides AI access but lacks Bflow business context, persistent memory, and role-aware personas
3. **No workflow integration**: AI answers exist in isolation — no connection to Bflow ERP, SOP library, or development toolchain
4. **Single-persona model**: One assistant for all roles — engineer and salesperson get the same treatment

## Desired State

MTClaw = **Governance backbone for AI-first transformation**:

1. **3 Rails Governance**:
   - **Spec Factory** (`/spec`): Structured requirement → JSON → evidence attachment
   - **PR Gate**: Policy evaluation on pull requests (WARNING → ENFORCE)
   - **Knowledge & Answering**: RAG per domain (engineering docs, SOPs, sales playbooks) with SOUL-aware responses
2. **16 Role-Aware SOULs**: Each employee interacts with an AI persona tuned for their role
3. **Bflow-Connected**: Queries enriched with Bflow business context via AI-Platform RAG
4. **Evidence Trail**: Every governance action produces auditable evidence

## Root Cause Analysis

**Why not just use chat.nhatquangholding.com?**

Interview evidence (n=8, Sprint 29):
- 8/8 respondents: "Generic AI, no Bflow context, no persistent memory"
- Engineering team: "Need code review context, not generic chat"
- Sales team: "Need proposal templates with MTS pricing, not ChatGPT"
- CS team: "Need SOP lookup with customer context"
- Back Office: "Need contract templates, not blank AI responses"

**Why not extend MTS-OpenClaw directly?**

- MTS-OpenClaw is a TypeScript chat platform (OpenClaw fork)
- MTClaw needs Go runtime (GoClaw) for multi-tenant PostgreSQL + production-grade performance
- Governance rails require structured backend (gate engine, evidence vault), not chat middleware

## Evidence

- 8 user interviews (Sprint 29): [interviews-engineering.md](user-research/interviews-engineering.md), [interviews-sales-cs.md](user-research/interviews-sales-cs.md), [interviews-back-office.md](user-research/interviews-back-office.md)
- Baseline metrics: [baseline-metrics.md](user-research/baseline-metrics.md)
- 3 expert reviews (Sprint 29 plan approval)
- CEO Priority C directive: Governance backbone for AI-first transformation

## Success Metrics

| Metric | Current | Target (Sprint 8) |
|--------|---------|-------------------|
| Feature waste rate | ~60% (industry) | <30% (with Spec Factory) |
| Time to first AI response | N/A | <5s (p95) |
| SOUL adoption (MTS) | 0% | 90% daily active |
| Governance compliance | 0 rails | 3 rails running |
| Evidence capture rate | 0% | 100% for gated actions |

---

**Gate**: G0.1 — Problem Definition
**Submission**: 2026-03-13
