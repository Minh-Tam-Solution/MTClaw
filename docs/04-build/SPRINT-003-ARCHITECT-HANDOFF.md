# Sprint 3 — Architect Handoff

**From**: [@pm] (Sprint 1-2 owner)
**To**: [@architect] (Sprint 3 design lead) → [@coder] (Sprint 3 implementation)
**Date**: 2026-03-02
**Sprint**: 3 — Architecture + RLS Implementation
**Gate**: G2 (Architecture Ready) — **APPROVED 9.2/10** ([@cto], 2026-03-02)

---

## Executive Summary

Sprint 1-2 ([@pm] + [@researcher]) produced:
- **Stage 00 complete**: Problem statement, business case, user research (8 interviews), 3 user personas, product vision
- **Stage 01 complete**: Requirements (7 FR + 8 NFR), user stories (US-001 to US-040), API spec (73 endpoints), data model (30+ tables), technology stack, test strategy, user journey map, SOUL quality rubric, roadmap
- **Stage 02 partial**: 5 ADRs, SOUL loading plan, RLS design, /spec design, schema analysis
- **Gates passed**: G0.1 (CTO 8.5/10, CPO 8/10), G0.2 (CTO 9/10, CPO 8.5/10)

**Sprint 3 mission**: Complete Stage 02 (System Architecture Document) + implement RLS, SOUL seeding, observability, and Bflow AI-Platform provider → submit G2 gate.

---

## What [@architect] Must Deliver

### Day 1: System Architecture Document (P0 — US-014)

**Location**: `docs/02-design/system-architecture-document.md`

**Required Sections** (SDLC 6.1.1 STANDARD tier):

1. **Component Diagram**
   ```
   User → Telegram Bot API → GoClaw Gateway → Agent Loop
     → SOUL System Prompt Builder (LoadFromStore + BuildSystemPrompt)
     → Bflow AI-Platform (qwen3:14b)
     → Response → Telegram
   ```
   - Show: all 5 layers (User, Channel, Agent, AI-Platform, Database)
   - Show: RLS middleware position in request lifecycle

2. **Data Flow Diagram**
   - Full request lifecycle: Telegram message → session resolution → agent loading → context files → system prompt building → AI-Platform call → response → trace logging
   - Highlight: where `SET LOCAL app.tenant_id` happens (before any DB query)
   - Highlight: 3 SOUL injection points (agent_context_files, ExtraPrompt, ContextFiles)

3. **Deployment Diagram**
   - Phase 1: Single VPS (4 vCPU, 8GB RAM)
   - Docker Compose: MTClaw binary + PostgreSQL 15 + pgvector + Prometheus (optional)
   - External: Telegram Bot API, Bflow AI-Platform

4. **Security Architecture**
   - RLS: 8 tables, SET LOCAL, mtclaw_admin/mtclaw_app roles
   - Encryption: AES-256-GCM for config_secrets
   - Auth: JWT + Telegram user verification
   - Tenant isolation: defense-in-depth (app-level + RLS + admin audit)

5. **Integration Points**
   - Bflow AI-Platform: OpenAI-compatible chat + RAG API (ADR-005)
   - Telegram Bot API: webhook or polling mode
   - GoClaw internal: agent loop, skills system, memory/RAG, tracing

6. **References**: All 5 ADRs + Sprint 2 design docs

---

## What [@architect] Hands to [@coder]

After completing the System Architecture Document, hand these implementation tasks to [@coder]:

### Task 1: RLS Migration (Day 1-2, P0 — US-015)

**Input docs**:
- [RLS Tenant Isolation Design](../02-design/rls-tenant-isolation-design.md) — full policy specifications
- [GoClaw Schema Analysis](../02-design/goclaw-schema-analysis.md) — table inventory

**Implementation**:
```
File: migrations/000008_rls_tenant_isolation.up.sql
File: migrations/000008_rls_tenant_isolation.down.sql
File: internal/middleware/tenant.go (new — SET LOCAL middleware)
```

**Key details**:
- 8 core tables: agents, agent_context_files, sessions, memory_documents, memory_chunks, traces, spans, user_context_files
- `ENABLE ROW LEVEL SECURITY` + `FORCE ROW LEVEL SECURITY` on each
- Pattern for direct owner_id: `USING (owner_id = current_setting('app.tenant_id', true))`
- Pattern for FK tables: `USING (agent_id IN (SELECT id FROM agents WHERE owner_id = ...))`
- Create `mtclaw_admin` role (owner, bypasses RLS)
- Create `mtclaw_app` role (enforced RLS)
- **CTO ISSUE-2**: Add index `idx_traces_agent_created` on `(agent_id, created_at)`
- **CTO-ISSUE-1 (v2.0.0)**: Change `agents.agent_key` unique constraint from `UNIQUE(agent_key)` to `UNIQUE(owner_id, agent_key)` — required for tenant-agnostic SOUL naming (same `dev` key across MTS + NQH tenants)

**Verification**:
```sql
-- Cross-tenant blocked:
SET LOCAL app.tenant_id = 'nqh';
SELECT * FROM agents WHERE owner_id = 'mts';  -- must return 0 rows

-- Same-tenant allowed:
SET LOCAL app.tenant_id = 'mts';
SELECT * FROM agents WHERE owner_id = 'mts';  -- must return 16 rows
```

### Task 2: SOUL Seeding Migration (Day 2-3, P0 — US-016)

**Input docs**:
- [SOUL Loading Implementation Plan](../02-design/soul-loading-implementation-plan.md) — Section 4
- 16 SOUL files in `docs/08-collaborate/souls/SOUL-*.md` (mts-general retired → merged into assistant)

**Implementation**:
```
File: migrations/000009_seed_mtclaw_souls.up.sql
File: migrations/000009_seed_mtclaw_souls.down.sql
```

**What to seed**:
1. **16 agent records** (all `predefined`, `owner_id='mts'`, `provider='bflow-ai-platform'`, `model='qwen3:14b'`)
   - See Sprint 3 plan US-016 for the full agent_key table
2. **48 agent_context_files** (3 per SOUL):
   - `SOUL.md` → content from `docs/08-collaborate/souls/SOUL-{key}.md` (body only, strip frontmatter)
   - `IDENTITY.md` → generated per SOUL (name, emoji, vibe)
   - `AGENTS.md` → shared governance workspace rules (same for all 16)
3. **Agent links** (delegation permissions):
   - `assistant → pm` (spec requests, /spec command)
   - `assistant → dev` (engineering tasks)
   - `assistant → sales` (sales tasks)
   - `assistant → cs` (CS tasks)
   - `assistant → coder` (implementation tasks)
   - `assistant → architect` (design tasks)
   - `assistant → researcher` (research tasks)
   - `pm → coder` (implementation)
   - `reviewer → coder` (fix after review)
   - All SOULs → `pm` (for `/spec` delegation)
   - **Note**: `assistant` is universal router (`is_default=true`), replaces retired `mts-general`
4. **Agent teams**:
   - "Engineering": pm, architect, coder, reviewer, researcher, writer, pjm, devops, tester
   - "Business": dev, sales, cs
   - "Advisors": cto, cpo, ceo
   - "Router": assistant (`is_default=true`, universal entry point)

**CTO ISSUE-3**: Add SOUL.md character count check — warn if body >2,000 chars in `make souls-validate`.

### Task 3: Observability (Day 3-4, P1 — US-017)

**Input docs**:
- [ADR-003 Observability](../02-design/01-ADRs/SPEC-0003-ADR-003-Observability-Architecture.md)

**Implementation**:
- Verify/enhance GoClaw's existing `slog` usage in `internal/` to include `trace_id`, `tenant_id`, `agent_key`
- Trace format: `{tenant_id}-{session_id}-{ulid}`
- OTEL metrics at `/metrics`:
  - `mtclaw_request_total{tenant, soul, channel}`
  - `mtclaw_request_duration_seconds{soul}`
  - `mtclaw_token_usage_total{tenant, soul}`
- Token cost: verify GoClaw already writes to `traces.total_input_tokens` / `total_output_tokens`

### Task 4: Bflow AI-Platform Provider (Day 4-5, P1 — US-018)

**Input docs**:
- [ADR-005 Bflow AI-Platform Integration](../02-design/01-ADRs/SPEC-0005-ADR-005-Bflow-AI-Platform-Integration.md)

**Implementation**:
- Register provider via `POST /v1/providers` or seed in migration
- Config from env: `BFLOW_AI_API_KEY`, `BFLOW_AI_BASE_URL`, `BFLOW_TENANT_ID`
- Verify: `POST /v1/providers/{id}/verify` returns OK
- Test: send message → receive AI response via `POST /v1/chat/completions`
- Fallback: graceful degradation (log error, user-friendly message)

---

## SDLC Artifacts Status (Post-Sprint 2)

### Stage 00 — Foundation ✅ COMPLETE

| Artifact | File | Status |
|----------|------|--------|
| Business Case | `docs/00-foundation/business-case.md` | ✅ |
| Problem Statement | `docs/00-foundation/problem-statement.md` | ✅ |
| User Research (8 interviews) | `docs/00-foundation/user-research/` | ✅ |
| User Personas (3 personas) | `docs/00-foundation/user-personas.md` | ✅ |
| Product Vision | `docs/00-foundation/product-vision.md` | ✅ |
| License Verification | `docs/00-foundation/goclaw-license-verification.md` | ✅ |
| G0.1 Gate Proposal | `docs/00-foundation/G0.1-GATE-PROPOSAL.md` | ✅ APPROVED |
| G0.2 Gate Proposal | `docs/00-foundation/G0.2-GATE-PROPOSAL.md` | ✅ APPROVED |

### Stage 01 — Planning ✅ COMPLETE

| Artifact | File | Status |
|----------|------|--------|
| Requirements (FR + NFR) | `docs/01-planning/requirements.md` | ✅ |
| User Stories (US-001 to US-040) | `docs/01-planning/user-stories.md` | ✅ |
| API Specification (73 endpoints) | `docs/01-planning/api-specification.md` | ✅ |
| Data Model (30+ tables) | `docs/01-planning/data-model.md` | ✅ |
| Technology Stack | `docs/01-planning/technology-stack.md` | ✅ |
| Test Strategy (tiered) | `docs/01-planning/test-strategy.md` | ✅ |
| User Journey Map (3 personas) | `docs/01-planning/user-journey-map.md` | ✅ |
| SOUL Quality Rubric | `docs/01-planning/soul-quality-rubric.md` | ✅ |
| Roadmap (10 sprints, 3 phases) | `docs/01-planning/roadmap.md` | ✅ |

### Stage 02 — Design (Sprint 3 completes this)

| Artifact | File | Status |
|----------|------|--------|
| ADR-001: GoClaw Adoption | `docs/02-design/01-ADRs/SPEC-0001-...` | ✅ |
| ADR-002: Three-System Architecture | `docs/02-design/01-ADRs/SPEC-0002-...` | ✅ |
| ADR-003: Observability | `docs/02-design/01-ADRs/SPEC-0003-...` | ✅ |
| ADR-004: SOUL Implementation | `docs/02-design/01-ADRs/SPEC-0004-...` | ✅ |
| ADR-005: Bflow AI-Platform | `docs/02-design/01-ADRs/SPEC-0005-...` | ✅ |
| GoClaw Schema Analysis | `docs/02-design/goclaw-schema-analysis.md` | ✅ |
| SOUL Loading Implementation Plan | `docs/02-design/soul-loading-implementation-plan.md` | ✅ |
| RLS Tenant Isolation Design | `docs/02-design/rls-tenant-isolation-design.md` | ✅ |
| /spec Command Design | `docs/02-design/spec-command-design.md` | ✅ |
| **System Architecture Document** | `docs/02-design/system-architecture-document.md` | ✅ COMPLETE (Day 1) |

### Stage 04 — Build

| Artifact | File | Status |
|----------|------|--------|
| Sprint 1 Plan | `docs/04-build/sprints/SPRINT-001-Foundation.md` | ✅ COMPLETE |
| Sprint 2 Plan | `docs/04-build/sprints/SPRINT-002-Requirements-Design.md` | ✅ COMPLETE |
| Sprint 3 Plan | `docs/04-build/sprints/SPRINT-003-Architecture-RLS.md` | ✅ READY |

---

## CTO/CPO Issues Tracker

| # | Issue | Source | Owner | Sprint 3 Action |
|---|-------|--------|-------|-----------------|
| CTO-1 | SystemPromptMode minimal strips SOUL | G0.2 Review | [@architect] | Document in System Architecture Doc |
| CTO-2 | Cost query perf with RLS subqueries | G0.2 Review | [@coder] | Add idx_traces_agent_created index |
| CTO-3 | SOUL.md 2,000 char budget enforcement | G0.2 Review | [@coder] | Add check to `make souls-validate` |
| CPO-1 | Sprint 4 validation plan detail | G0.2 Review | [@pm] | Flesh out before Sprint 4 |
| CPO-2 | Sales RAG needs minimal content for Sprint 4 | G0.2 Review | [@pm] | Prepare 5-10 sales docs |
| CPO-3 | Manual smoke-test 16 SOULs | G0.2 Review | [@coder] | Day 5 manual QA |

---

## Sprint 3 Success Criteria

| Criterion | Verification |
|-----------|-------------|
| System Architecture Document complete | [@cto] review + approval |
| RLS policies on 8 tables | Cross-tenant isolation test passes |
| 16 SOULs in DB | `GET /v1/agents` returns 16 records |
| 48 context files in DB | SOUL.md content matches source files |
| Structured logging | JSON logs include trace_id, tenant_id |
| Metrics export | `/metrics` returns OTEL counters |
| Bflow AI-Platform connected | Chat completion returns AI response |
| All 55 endpoints pass regression | No RLS breakage |
| G2 gate submitted | CTO approval |

---

## Key Source Files (GoClaw Codebase)

| Area | File | Why It Matters |
|------|------|---------------|
| System prompt | `internal/agent/systemprompt.go` | 15-section builder, ContextFiles injection |
| Bootstrap loader | `internal/bootstrap/load_store.go` | `LoadFromStore()` — DB-based SOUL loading |
| Bootstrap seeder | `internal/bootstrap/seed_store.go` | `SeedToStore()` — template seeding pattern |
| Agent store | `internal/store/pg/agents.go` | Agent CRUD, owner_id queries |
| Context store | `internal/store/pg/agents_context.go` | agent_context_files CRUD |
| HTTP routes | `internal/http/agents.go` | 55 inherited endpoints |
| Migrations | `migrations/000001_*.sql` through `000007_*.sql` | Schema evolution |
| Tracing | `internal/tracing/collector.go` | Trace/span collection |
| Telegram | `internal/channels/telegram/` | Channel handler, commands |

---

**[@pm] note**: Sprint 3 is the critical path sprint — RLS blocks ALL feature sprints. If RLS migration breaks queries, fix immediately (P0). Everything else can slip to Sprint 4 if needed.

Good luck, [@architect] + [@coder]. The foundation is solid. Build on it.
