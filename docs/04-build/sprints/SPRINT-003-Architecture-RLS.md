# Sprint 003 — Architecture + RLS Implementation

**Sprint**: 3 of 10
**Duration**: 5 days
**Phase**: Phase 1 — Foundation + First Rails
**Gate**: G2 (Architecture Ready)
**Predecessor**: Sprint 2 (G0.2 APPROVED — CTO 9/10, CPO 8.5/10)
**Status**: NOT STARTED
**Owner**: [@architect] (design) → [@coder] (implementation)
**Points**: ~13

---

## Sprint Goal

> Implement RLS tenant isolation, seed 16 SOULs into database,
> set up observability pipeline, connect Bflow AI-Platform provider,
> produce System Architecture Document, and submit G2 gate.

---

## Entry Criteria

- [x] G0.2 APPROVED (CTO 9/10, CPO 8.5/10)
- [x] RLS Tenant Isolation Design reviewed and approved
- [x] SOUL Loading Implementation Plan reviewed and approved
- [x] GoClaw schema analysis complete (30+ tables documented)
- [x] Bflow AI-Platform API key provisioned (aip_c786)

---

## User Stories

### US-014: System Architecture Document

**As a** [@cto]
**I want** a comprehensive System Architecture Document for MTClaw
**So that** the team has a single reference for all architectural decisions and data flows

**Acceptance Criteria**:
- [ ] Component diagram: Telegram → Gateway → Agent Loop → AI-Platform → Response
- [ ] Data flow diagram: request lifecycle with RLS + SOUL injection
- [ ] Deployment diagram: VPS + Docker Compose (PostgreSQL, MTClaw binary, Prometheus)
- [ ] Security architecture: RLS, JWT, AES-256-GCM, tenant isolation
- [ ] Integration points: Bflow AI-Platform, Telegram Bot API
- [ ] References all 4 ADRs + Sprint 2 design docs
- [ ] Stored at `docs/02-design/system-architecture-document.md`

**Points**: 3 | **Priority**: P0 | **Day**: 1

---

### US-015: RLS Migration + Tenant Middleware

**As a** [@cto]
**I want** Row-Level Security policies on all tenant-scoped tables
**So that** even application bugs cannot leak cross-tenant data

**Acceptance Criteria**:
- [ ] Migration `000008_rls_tenant_isolation.up.sql` creates:
  - RLS policies on 8 core tables (agents, agent_context_files, sessions, memory_documents, memory_chunks, traces, spans, user_context_files)
  - `mtclaw_admin` role (bypasses RLS for migrations)
  - `mtclaw_app` role (RLS enforced)
- [ ] Tenant middleware: `SET LOCAL app.tenant_id = $1` per transaction
- [ ] All 55 inherited API endpoints still work after RLS
- [ ] Cross-tenant isolation test: tenant A cannot see tenant B's agents
- [ ] Down migration safely removes RLS policies

**Points**: 3 | **Priority**: P0 | **Day**: 1-2

**Design Reference**: [rls-tenant-isolation-design.md](../../02-design/rls-tenant-isolation-design.md)

**Implementation Notes**:
```sql
-- Pattern for all policies:
ALTER TABLE {table} ENABLE ROW LEVEL SECURITY;
ALTER TABLE {table} FORCE ROW LEVEL SECURITY;

-- Direct owner_id tables:
CREATE POLICY tenant_{table} ON {table}
  FOR ALL USING (owner_id = current_setting('app.tenant_id', true));

-- FK-based tables (via agent_id → agents.owner_id):
CREATE POLICY tenant_{table} ON {table}
  FOR ALL USING (agent_id IN (
    SELECT id FROM agents WHERE owner_id = current_setting('app.tenant_id', true)
  ));
```

**CTO ISSUE-2 from G0.2**: Consider adding `idx_traces_agent_created` index on `(agent_id, created_at)` for cost query performance.

---

### US-016: SOUL Seeding Migration

**As a** developer
**I want** 16 SOULs seeded into the database as agents with context files
**So that** SOUL loading works via GoClaw's existing `LoadFromStore()` architecture

**Acceptance Criteria**:
- [ ] Migration `000009_seed_mtclaw_souls.up.sql` creates:
  - 16 agent records (all `predefined` type, `owner_id = 'mts'`)
  - 48 agent_context_files (3 per SOUL: SOUL.md, IDENTITY.md, AGENTS.md)
  - Agent links (delegation permissions between SOULs)
  - Agent teams ("MTS Engineering", "MTS Business")
- [ ] Each SOUL's `SOUL.md` content sourced from `docs/08-collaborate/souls/SOUL-{key}.md`
- [ ] `AGENTS.md` shared across all 16 SOULs (governance workspace rules)
- [ ] `IDENTITY.md` per-SOUL (name, emoji, vibe)
- [ ] `GET /v1/agents` returns 16 agents after migration
- [ ] Down migration removes all seeded data cleanly

**Points**: 3 | **Priority**: P0 | **Day**: 2-3

**Design Reference**: [soul-loading-implementation-plan.md](../../02-design/soul-loading-implementation-plan.md) Section 4

**SOUL Agent Records**:
| # | agent_key | display_name | agent_type | active_default |
|---|-----------|-------------|------------|----------------|
| 1 | pm | Product Manager | predefined | yes |
| 2 | architect | Software Architect | predefined | no |
| 3 | coder | Software Engineer | predefined | yes |
| 4 | reviewer | Code Reviewer | predefined | yes |
| 5 | researcher | User Researcher | predefined | no |
| 6 | writer | Technical Writer | predefined | no |
| 7 | pjm | Project Manager | predefined | no |
| 8 | devops | DevOps Engineer | predefined | no |
| 9 | tester | QA Engineer | predefined | no |
| 10 | cto | CTO Advisor | predefined | no |
| 11 | cpo | CPO Advisor | predefined | no |
| 12 | ceo | CEO Advisor | predefined | no |
| 13 | dev | Developer Assistant | predefined | yes |
| 14 | sales | Sales Assistant | predefined | yes |
| 15 | cs | Customer Success Assistant | predefined | yes |
| 16 | assistant | Universal Router | predefined | yes (`is_default=true`) |

All: `provider = 'bflow-ai-platform'`, `model = 'qwen3:14b'`

---

### US-017: Observability Implementation

**As a** [@devops]
**I want** structured logging with trace_id and tenant_id, plus metrics export
**So that** we can monitor per-tenant, per-SOUL behavior and debug issues

**Acceptance Criteria**:
- [ ] Structured JSON logging via Go `slog`:
  - Every log line includes: `trace_id`, `tenant_id`, `agent_key`, `level`, `msg`, `ts`
  - Request-scoped context propagation via `context.Context`
- [ ] Trace format: `{tenant_id}-{session_id}-{ulid}` (from ADR-003)
- [ ] OTEL metrics exported → Prometheus-compatible `/metrics` endpoint:
  - `mtclaw_request_total` (by tenant, soul, channel)
  - `mtclaw_request_duration_seconds` (histogram, by soul)
  - `mtclaw_token_usage_total` (by tenant, soul)
  - `mtclaw_active_sessions` (gauge)
- [ ] Token cost tracking: write token counts to traces table per request
- [ ] Existing GoClaw tracing (`internal/tracing/`) verified compatible

**Points**: 2 | **Priority**: P1 | **Day**: 3-4

**Design Reference**: [ADR-003 Observability](../../02-design/01-ADRs/SPEC-0003-ADR-003-Observability-Architecture.md)

---

### US-018: Bflow AI-Platform Provider Setup

**As a** developer
**I want** Bflow AI-Platform registered as the default LLM provider
**So that** all SOULs use the Bflow AI-Platform for inference

**Acceptance Criteria**:
- [ ] Provider registered via `POST /v1/providers`:
  - Name: `bflow-ai-platform`
  - Base URL: `https://api.nhatquangholding.com` (or env var)
  - Auth: `X-API-Key` from `BFLOW_AI_API_KEY` env
  - Default model: `qwen3:14b`
- [ ] Provider connectivity verified: `POST /v1/providers/{id}/verify` returns OK
- [ ] All 16 SOULs configured with provider = `bflow-ai-platform`
- [ ] Chat completions work: send test message → receive AI response
- [ ] Tenant header: `X-Tenant-ID: mts` injected in all AI-Platform requests
- [ ] Graceful degradation: if AI-Platform unavailable, log error, return user-friendly message

**Points**: 1 | **Priority**: P1 | **Day**: 4-5

**Config** (from MEMORY):
```
BFLOW_AI_API_KEY=aip_c786...
BFLOW_AI_BASE_URL=https://api.nhatquangholding.com
BFLOW_TENANT_ID=mts
```

---

### US-019: G2 Gate Proposal

**As a** [@pm]
**I want** G2 gate proposal submitted with all evidence
**So that** the project has formal architectural approval to proceed to implementation

**Acceptance Criteria**:
- [ ] G2 proposal at `docs/00-foundation/G2-GATE-PROPOSAL.md`
- [ ] Evidence:
  - System Architecture Document ✅
  - RLS migration verified ✅
  - 16 SOULs loadable from DB ✅
  - Observability pipeline running ✅
  - Bflow AI-Platform connected ✅
  - All Sprint 2 design docs (SOUL loading plan, RLS design, /spec design, API spec) ✅
- [ ] CTO approval signature

**Points**: 1 | **Priority**: P0 | **Day**: 5

---

## Daily Plan

### Day 1: Architecture Document + RLS Migration Start

| Task | Owner | Output |
|------|-------|--------|
| System Architecture Document | [@architect] | `docs/02-design/system-architecture-document.md` |
| RLS migration: `000008_rls_tenant_isolation.up.sql` | [@coder] | Migration file + test |
| Create `mtclaw_admin` / `mtclaw_app` roles | [@coder] | Part of migration |

### Day 2: RLS Complete + SOUL Seeding Start

| Task | Owner | Output |
|------|-------|--------|
| Tenant middleware: `SET LOCAL app.tenant_id` | [@coder] | `internal/middleware/tenant.go` |
| RLS verification: cross-tenant test | [@coder] | Test passing |
| SOUL seeding migration: 16 agents | [@coder] | `000009_seed_mtclaw_souls.up.sql` |
| SOUL content: generate SOUL.md from source files | [@coder] | 48 context file entries |

### Day 3: SOUL Seeding Complete + Observability Start

| Task | Owner | Output |
|------|-------|--------|
| Complete SOUL seeding + delegation links | [@coder] | All 16 SOULs in DB |
| Verify: `GET /v1/agents` returns 16 agents | [@coder] | API test |
| slog structured logging setup | [@coder] | JSON logs with trace_id |
| Trace format: `{tenant}-{session}-{ulid}` | [@coder] | Context propagation |

### Day 4: Observability Complete + Bflow Provider

| Task | Owner | Output |
|------|-------|--------|
| OTEL metrics: request count, duration, token usage | [@coder] | `/metrics` endpoint |
| Token cost tracking in traces table | [@coder] | Token fields populated |
| Register Bflow AI-Platform provider | [@coder] | Provider in DB |
| Verify: chat completions via AI-Platform | [@coder] | Test message → response |

### Day 5: Integration Test + G2 Gate

| Task | Owner | Output |
|------|-------|--------|
| Integration test: full request lifecycle | [@coder] | Telegram → SOUL → AI → response |
| Cross-tenant isolation verification | [@coder] | RLS blocking confirmed |
| All 55 API endpoints regression test | [@coder] | No RLS breakage |
| G2 gate proposal | [@pm] | `G2-GATE-PROPOSAL.md` |

---

## Verification Checklist (DoD)

- [ ] System Architecture Document complete + reviewed
- [ ] RLS policies on 8 tables, cross-tenant isolation verified
- [ ] `mtclaw_admin` bypasses RLS, `mtclaw_app` enforces RLS
- [ ] 16 SOULs in `agents` table with 48 `agent_context_files`
- [ ] `GET /v1/agents` returns 16 agents (owner_id='mts')
- [ ] Structured JSON logging with trace_id, tenant_id
- [ ] OTEL metrics exposed at `/metrics`
- [ ] Bflow AI-Platform provider registered + verified
- [ ] Chat completions work: message → AI response → traced
- [ ] All 55 inherited API endpoints pass regression
- [ ] G2 gate proposal submitted

---

## CTO/CPO Issues to Address (from G0.2 Review)

| Issue | Source | Action | Sprint 3 Task |
|-------|--------|--------|---------------|
| SystemPromptMode minimal strips SOUL context | CTO ISSUE-1 | Document as known limitation | System Architecture Doc |
| Cost query perf with RLS subqueries | CTO ISSUE-2 | Add `idx_traces_agent_created` index | US-015 migration |
| SOUL.md 2,000 char budget enforcement | CTO ISSUE-3 | Add to `make souls-validate` | US-016 validation |
| Sprint 4 validation plan detail | CPO CONCERN-1 | Flesh out protocol | Sprint 4 plan |
| Sales RAG needs minimal content for Sprint 4 | CPO CONCERN-2 | Prepare 5-10 sales docs | Sprint 4 prep |
| Manual smoke-test 16 SOULs | CPO CONCERN-3 | Manual QA after seeding | Day 5 |

---

## References

- [Sprint 2 Plan](SPRINT-002-Requirements-Design.md)
- [G0.2 Gate Proposal](../../00-foundation/G0.2-GATE-PROPOSAL.md)
- [RLS Tenant Isolation Design](../../02-design/rls-tenant-isolation-design.md)
- [SOUL Loading Implementation Plan](../../02-design/soul-loading-implementation-plan.md)
- [Observability ADR-003](../../02-design/01-ADRs/SPEC-0003-ADR-003-Observability-Architecture.md)
- [API Specification](../../01-planning/api-specification.md)
- [Roadmap](../../01-planning/roadmap.md)
