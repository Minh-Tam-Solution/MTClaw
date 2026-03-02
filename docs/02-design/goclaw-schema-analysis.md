# GoClaw Schema Analysis

**Date**: 2026-03-02
**Author**: [@pm], [@architect]
**Purpose**: Map GoClaw tables to MTClaw SOUL/governance requirements

---

## Schema Overview (7 migrations, 30+ tables)

### Tables Relevant to SOULs

| Table | GoClaw Purpose | MTClaw SOUL Mapping |
|-------|---------------|---------------------|
| `agents` | Agent definitions (model, tools, config) | **Direct mapping** — each SOUL maps to an agent row |
| `user_agent_profiles` | Per-user agent settings (workspace, timestamps) | SOUL assignment per user (which SOUL is active) |
| `user_agent_overrides` | Per-user model/provider overrides | User-specific SOUL config (e.g., preferred model) |
| `agent_context_files` | Shared context files per agent | SOUL markdown content could be stored here |
| `user_context_files` | Per-user context files | User-specific SOUL context |
| `agent_teams` | Multi-agent team coordination | SOUL groups (e.g., "SDLC team", "MTS business team") |
| `agent_team_members` | Team membership | Which SOULs are in which groups |

### Tables Relevant to Governance Rails

| Table | GoClaw Purpose | MTClaw Rail Mapping |
|-------|---------------|---------------------|
| `sessions` | Conversation state (messages, tokens) | Evidence trail — session logs for audit |
| `traces` | LLM tracing (input/output, cost, duration) | **Token cost tracking** — per-tenant cost guardrails |
| `spans` | Detailed trace spans (model, tokens, cost) | Granular cost breakdown per SOUL per request |
| `skills` | Skill definitions (frontmatter, content, embedding) | Could store /spec, /review skill definitions |
| `skill_agent_grants` | Which agent has which skills | Rail access per SOUL |
| `memory_documents` + `memory_chunks` | pgvector RAG (1536-dim embeddings) | **Rail #3 Knowledge** — built-in RAG with vector search |
| `delegation_history` | Agent-to-agent delegation log | Governance audit trail for SOUL delegation |

### Tables Relevant to Multi-Tenant

| Table | GoClaw Purpose | MTClaw Tenant Mapping |
|-------|---------------|----------------------|
| `agents.owner_id` | Agent ownership | Maps to `tenant_id` — needs RLS policy |
| `agent_shares` | Cross-user agent access | Cross-tenant SOUL sharing (Phase 2) |
| `config_secrets` | Encrypted key-value | Per-tenant secrets (Bflow API key) |
| `channel_instances` | Channel connections (Telegram, etc.) | Per-tenant channel config |

### Tables Relevant to Observability

| Table | GoClaw Purpose | MTClaw Mapping |
|-------|---------------|----------------|
| `traces` | Request-level tracing | Tenant-scoped tracing with cost |
| `spans` | Span-level detail | Model/provider usage per SOUL |
| `sessions.input_tokens`, `output_tokens` | Token counting | Existing token tracking — add tenant aggregation |
| `cron_run_logs` | Cron execution logs | Scheduled task audit |

## Key Findings

### 1. SOUL → Agent Mapping (Direct)
GoClaw's `agents` table is a natural fit for SOULs:
- `agent_key` → SOUL role name (e.g., "pm", "coder")
- `display_name` → SOUL display name
- `model` → Default model for this SOUL
- `tools_config` → Rail access (which tools this SOUL can use)
- `other_config` → SOUL-specific settings

### 2. RAG Already Built-In
`memory_chunks` table with `vector(1536)` embedding + `tsvector` full-text search = Rail #3 Knowledge has infrastructure ready. Need to:
- Create collections per domain (engineering docs, SOPs, sales playbooks)
- Connect to Bflow AI-Platform embedding endpoint
- Map collections to SOULs

### 3. Token Tracking Exists
`traces` + `spans` already track `total_cost`, `input_tokens`, `output_tokens`. Need to:
- Add tenant-level aggregation view
- Add cost guardrail check middleware
- Create `token_usage` summary table (or view over `spans`)

### 4. Multi-Tenant Gap
GoClaw uses `owner_id` (VARCHAR) for ownership but does NOT have:
- RLS policies (Sprint 3 requirement)
- Explicit `tenant_id` column
- Tenant-level isolation enforced at DB level

**Action**: Sprint 3 — add `tenant_id` UUID column + RLS policies to all relevant tables.

### 5. Team Infrastructure Ready
`agent_teams` + `agent_team_members` + `team_tasks` + `delegation_history` = foundation for SOUL team coordination and governance audit trail.

## Sprint 1 Actions

- [x] Document schema (this file)
- [ ] Verify `make migrate-up` works with local PostgreSQL
- [ ] Map 16 SOUL files to `agents` table rows (Sprint 2)
- [ ] Design RLS migration (Sprint 3)

## Sprint 2 Actions

- [ ] GoClaw schema deep dive → SOUL loading implementation plan
- [ ] Design tenant_id migration
- [ ] Map Rail #3 to memory_chunks infrastructure
- [ ] Design token_usage aggregation

---

**References**:
- Migrations: `migrations/000001_init_schema.up.sql` through `000007_team_metadata.up.sql`
- [ADR-003: Observability](01-ADRs/SPEC-0003-ADR-003-Observability-Architecture.md)
- [ADR-004: SOUL Implementation](01-ADRs/SPEC-0004-ADR-004-SOUL-Implementation.md)
