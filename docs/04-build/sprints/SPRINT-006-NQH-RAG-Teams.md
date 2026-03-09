# Sprint 6 — NQH Tenant + Rail #3 Knowledge + Team Routing

**SDLC Stage**: 04-Build
**Version**: 1.0.0
**Date**: 2026-03-03
**Author**: [@pm]
**Framework**: SDLC Enterprise Framework 6.1.1
**Tier**: STANDARD

---

## Sprint Summary

| Field | Value |
|-------|-------|
| Sprint | 6 |
| Goal | Rail #3 Knowledge operational (MTS) + Team routing + NQH pilot (conditional) |
| Duration | 5 days |
| Owner | [@coder] (implementation) + [@devops] (Zalo) + [@pm] (RAG content curation) |
| Points | ~17 |
| Gate | None (mid-phase) |
| Predecessor | Sprint 5 — G3 APPROVED |

---

## Entry Criteria

- [ ] Sprint 5 complete — G3 Build Ready APPROVED
- [ ] MTS pilot running (≥3/10 WAU from Sprint 5)
- [ ] 2 Rails operational: /spec + /review
- [ ] Integration tests (5 scenarios) passing
- [ ] Token cost queryable per tenant per SOUL
- [ ] Bflow AI-Platform verified (RAG endpoint: `POST /api/v1/rag/query`)

---

## Sprint Goal

> **SOUL-Aware RAG operational for MTS** (Rail #3 Knowledge) + **Team routing active** + **NQH pilot** (10-20 users on Zalo, conditional on CEO re-confirmation).

Sprint 6 adds the third and final rail — Knowledge & Answering. Three parallel tracks:
1. **Rail #3 Knowledge** — SOUL-Aware RAG routing (Context Drift Layer B)
2. **Team Routing** — `@engineering`, `@business`, `@advisory` team mentions
3. **NQH Expansion** — Tenant config + Zalo + NQH-SOPs RAG (CONDITIONAL)

---

## User Stories

### US-034: SOUL-Aware RAG Routing — Context Drift Layer B (P0, 3 pts)

**As a** MTS employee,
**I want** the bot to query relevant knowledge based on which SOUL is active,
**So that** answers are accurate and domain-specific (dev gets code docs, sales gets pricing).

**Acceptance Criteria**:
- Given `@enghelp` asks about Bflow API → RAG queries `engineering` collection → response includes code-relevant content
- Given `@sales` asks about pricing → RAG queries `sales` collection → response includes pricing/product content
- Given `@cs` asks about customer issue → RAG queries `engineering` + `sales` collections → response includes cross-domain content
- Given `@assistant` asks general question → RAG queries broad collections → general response
- Given any RAG query → token budget enforced (≤2,500 tokens per retrieval)
- Given any RAG query → RetrievalEvidence metadata stored in trace record (query, collection, hits, tokens_used)
- Given AI-Platform RAG unavailable → graceful degradation (respond without RAG context, log warning)

**Design**: SAD Section 8.4 (SOUL-Aware RAG Routing)

**Implementation Notes**:
- API: `POST /api/v1/rag/query` with `collection` filter (AI-Platform native)
- Collection mapping defined in SOUL config or hardcoded mapping table
- RAG results injected into `ContextFiles` section of system prompt
- Token budget: hard cap 2,500 tokens (truncate results if exceeded)

---

### US-035: MTS RAG Collections — Engineering + Sales + HR (P1, 3 pts)

**As a** PM,
**I want** 3 RAG collections curated and verified for MTS,
**So that** SOULs have domain knowledge to answer accurately.

**Acceptance Criteria**:
- Given `engineering` collection → contains Bflow API docs, coding standards, architecture decisions
- Given `sales` collection → contains pricing, proposals, case studies, competitor analysis
- Given `hr-policies` collection → contains HR handbook, leave policies, onboarding guides
- Given each collection → ingested via AI-Platform `/v1/rag/ingest/batch` endpoint
- Given ingestion → document count and status reported

**Tasks**:
1. Curate engineering docs (from MTS-OpenClaw, Bflow, NQH-Bot repositories)
2. Curate sales docs (from existing MTS sales materials)
3. Curate HR docs (from MTS HR handbook if available)
4. Ingest via AI-Platform API
5. Verify: RAG query returns relevant results for each collection

**Note**: [@pm] scope for content curation. [@coder] builds the integration code (US-034).

---

### US-036: Team Mention Routing (P1, 3 pts)

**As a** MTS employee,
**I want to** mention `@engineering`, `@business`, or `@advisory` in Telegram,
**So that** my request is routed to the team leader with full team context.

**Acceptance Criteria**:
- Given `@engineering design auth system` → PM (team lead) receives with engineering team context in ExtraPrompt
- Given `@business help with pricing` → assistant (team lead) receives with business team context
- Given `@advisory gate approval G2` → CTO (team lead) receives with advisory context
- Given `@pm` (agent key) → routes directly to PM agent (agent-first, NOT team)
- Given `/teams` command → lists available teams with members and leads
- Given unknown team `@nonexistent-team` → treated as regular message (no routing change)
- Given team routing → trace record includes `team:engineering` tag
- Resolution order: agent-first (`@pm` → PM directly), team-second (`@engineering` → PM as leader)

**Design**: Roadmap v2.2.0 Team Routing section + Team Charter files (`docs/08-collaborate/teams/TEAM-*.md`)

**Implementation Notes**:
- Parse `@mention` → first check agents.Get() (existing), then check teams
- Team charters provide lead + members + delegation rules
- Team context injected into ExtraPrompt (similar to Layer A anchoring)
- Reuses existing TeamStore infrastructure (23 methods, 4 seeded teams)

---

### US-037: /teams Command (P1, 1 pt)

**As a** MTS employee,
**I want** a `/teams` command to see available teams,
**So that** I know which team mentions are available.

**Acceptance Criteria**:
- Given `/teams` → lists teams with format: `@engineering — SDLC Engineering (lead: @pm)`
- Given `/teams` → includes member count per team
- Given `/teams` → added to `/help` output

**CPO CONDITION-1**: Discoverability requirement — users must be able to discover team mentions.

---

### US-038: NQH Tenant Configuration (P0, 2 pts) — CONDITIONAL

**Condition**: CEO re-confirms NQH pilot for Sprint 6 (Option A was MTS-only).

**As a** NQH HO manager,
**I want** MTClaw available for NQH staff via Zalo,
**So that** HO/management can access NQH-SOPs via AI-powered chat.

**Acceptance Criteria**:
- Given NQH tenant → `owner_id='nqh'` created with RLS verified
- Given RLS → MTS queries return only MTS data; NQH queries return only NQH data
- Given NQH SOULs → subset seeded (assistant, itadmin, + NQH-specific SOULs TBD)
- Given Zalo → NQH bot registered and connected via OpenClaw extensions/zalo

**Tasks**:
1. Create migration: NQH tenant + NQH SOULs
2. Configure Zalo channel (extensions/zalo + extensions/zalouser)
3. Connect NQH-SOPs RAG collection (already indexed on AI-Platform, 805 docs)
4. Verify: cross-tenant isolation (MTS + NQH concurrent queries)

**Note**: If CEO does NOT re-confirm, defer entire US-038 to Sprint 7+. MTS-only items (US-034–037, US-039–040) proceed regardless.

---

### US-039: Tenant Cost Guardrails (P1, 2 pts)

**As a** CTO,
**I want** per-tenant cost limits enforced,
**So that** no tenant can run up unbounded AI costs.

**Acceptance Criteria**:
- Given tenant config → `monthly_token_limit` and `daily_request_limit` configurable
- Given MTS tenant → default limits: 1M tokens/month, 500 requests/day
- Given limit exceeded → request rejected with user-friendly message ("Đã đạt giới hạn sử dụng hôm nay")
- Given approaching limit (80%) → warning in response ("Lưu ý: còn 20% quota hôm nay")
- Given limits → queryable via admin API: `GET /v1/tenants/{id}/usage`
- Given reset → daily limit resets at 00:00 UTC+7; monthly at 1st of month

**Implementation Notes**:
- Track usage in Redis (fast increment) with PostgreSQL backup (hourly sync)
- Check limit before AI-Platform call (gateway_consumer.go)
- Limits stored in tenants table (new columns) or separate config table

---

### US-040: Cross-Tenant Isolation Regression Test (P0, 1 pt)

**As a** CTO,
**I want** regression tests confirming MTS and NQH data are isolated,
**So that** we don't regress on RLS after adding NQH tenant.

**Acceptance Criteria**:
- Given MTS tenant session → query returns only MTS agents, traces, teams
- Given NQH tenant session → query returns only NQH agents, traces, teams
- Given concurrent requests → no data leakage between tenants
- Given test suite → runs as part of `make test-integration`

---

## Sprint Schedule

| Day | Track 1: Rail #3 Knowledge | Track 2: Team Routing | Track 3: NQH (conditional) |
|-----|---------------------------|----------------------|---------------------------|
| 1 | SOUL-Aware RAG routing code | Team mention parser (gateway_consumer.go) | NQH migration (if approved) |
| 2 | RAG collection mapping + token budget | /teams command (commands.go) | Zalo channel config |
| 3 | RAG integration tests | Team context ExtraPrompt injection | NQH-SOPs RAG connection |
| 4 | Cost guardrails | Cross-tenant regression tests | NQH pilot onboarding |
| 5 | E2E: RAG + team routing together | Documentation | Sprint review |

---

## Risk Register

| # | Risk | Prob | Impact | Mitigation |
|---|------|------|--------|------------|
| R8 | RAG accuracy below threshold (wrong pricing/technical info) | Med | High | Manual curation first, verify each collection before go-live |
| R14 | Team mention conflicts with agent key | Low | Low | Agent-first resolution: @pm → agent, @engineering → team |
| R15 | AI-Platform RAG latency >3s (large collections) | Med | Med | Token budget cap (2,500), cache frequent queries |
| R16 | NQH CEO decision unclear for Sprint 6 | Med | Med | MTS items proceed regardless; NQH track is fully conditional |
| R17 | Cost guardrail bypass via concurrent requests | Low | Med | Redis atomic increment; PostgreSQL backup for audit |

---

## Tech Debt (from Sprint 5 Review)

| Item | Severity | Sprint |
|------|----------|--------|
| CTO NOTE-1: Test helper duplication (extractTraceMetadata/parseMention in test file) | LOW | 6 (extract to testable functions) |
| CTO NOTE-2: PR URL validation too permissive (accepts non-GitHub URLs) | LOW | 8 (ENFORCE mode) |
| ISSUE-3: Migration 000012 SELECT INTO without error guard | LOW | Defer |

---

## Exit Criteria

- [ ] Rail #3 operational: RAG queries routed by SOUL role
- [ ] 3 MTS collections ingested and verified (engineering, sales, hr-policies)
- [ ] Team routing: @engineering, @business, @advisory working
- [ ] /teams command working
- [ ] Cost guardrails enforced per tenant
- [ ] Cross-tenant regression passing (MTS + NQH if approved)
- [ ] RetrievalEvidence metadata logged in traces
- [ ] Zero P0 bugs
- [ ] CTO sprint review

---

## References

- [Roadmap Sprint 6](../../01-planning/roadmap.md#sprint-6--nqh-tenant--rail-3-knowledge--soul-aware-rag)
- [SAD Section 8](../../02-design/system-architecture-document.md) — Context Drift Prevention
- [Requirements FR-003, FR-008](../../01-planning/requirements.md)
- [Team Charters](../../08-collaborate/teams/)
- [Sprint 5 Coder Handoff](../SPRINT-005-CODER-HANDOFF.md)
- [Bflow AI-Platform Integration Guide](../../../docs/03-integrate/bflow-ai-platform-sop-generator-guide.md)
