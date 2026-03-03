# Data Model — MTClaw

**SDLC Stage**: 01-Planning
**Version**: 1.0.0
**Date**: 2026-03-02
**Author**: [@pm]
**Source**: GoClaw schema analysis (7 migrations, 30+ tables)
**Framework**: SDLC 6.1.1 — Stage 01 Required Artifact (STANDARD tier)

---

## 1. Overview

MTClaw inherits GoClaw's PostgreSQL schema (30+ tables across 7 migrations) and extends it with governance-specific data. The schema supports:

- **16 SOULs** as `agents` records with `agent_context_files`
- **Multi-tenant isolation** via `owner_id` + RLS policies
- **3 Governance Rails** using existing tables (traces, skills) + new governance tables
- **RAG/Knowledge** via `memory_documents` + `memory_chunks` (pgvector 1536-dim)
- **Observability** via `traces` + `spans` with token cost tracking

---

## 2. Entity Relationship Diagram (Logical)

```
┌─────────────────────────────────────────────────────────────────────┐
│                        CORE ENTITIES                                │
│                                                                     │
│  ┌──────────┐    1:N    ┌─────────────────────┐                    │
│  │  agents   │──────────│ agent_context_files  │  SOUL content     │
│  │ (16 SOULs)│          │ (SOUL.md, IDENTITY)  │                    │
│  └────┬─────┘          └─────────────────────┘                    │
│       │                                                             │
│       │ 1:N    ┌─────────────────────┐                              │
│       ├────────│ user_context_files   │  Per-user overrides         │
│       │        └─────────────────────┘                              │
│       │                                                             │
│       │ 1:N    ┌──────────┐    1:N    ┌────────┐                   │
│       ├────────│ sessions  │──────────│messages │  Conversation     │
│       │        └──────────┘          └────────┘                   │
│       │                                                             │
│       │ 1:N    ┌──────────┐    1:N    ┌────────┐                   │
│       ├────────│  traces   │──────────│ spans   │  Observability    │
│       │        └──────────┘          └────────┘                   │
│       │                                                             │
│       │ 1:N    ┌──────────────────┐   1:N  ┌──────────────┐       │
│       ├────────│ memory_documents  │───────│ memory_chunks │  RAG  │
│       │        └──────────────────┘       └──────────────┘       │
│       │                                                             │
│       │ M:N    ┌──────────┐                                        │
│       ├────────│  skills   │  Governance skills (/spec, /review)   │
│       │        └──────────┘                                        │
│       │                                                             │
│       │ M:N    ┌──────────────┐                                    │
│       └────────│ agent_links   │  Delegation permissions           │
│                └──────────────┘                                    │
│                                                                     │
├─────────────────────────────────────────────────────────────────────┤
│                     TENANT ISOLATION                                │
│                                                                     │
│  agents.owner_id = 'mts' ──── RLS policy ──── SET LOCAL tenant_id │
│                                                                     │
│  All child tables isolated via FK → agents.owner_id                 │
├─────────────────────────────────────────────────────────────────────┤
│                     GLOBAL (Shared)                                  │
│                                                                     │
│  llm_providers │ config_secrets │ builtin_tools │ mcp_servers      │
│  embedding_cache                                                    │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 3. Core Tables

### 3.1 Agent/SOUL Tables

| Table | Key Columns | Purpose | Tenant Scope |
|-------|-------------|---------|-------------|
| `agents` | id (UUID v7), agent_key, display_name, owner_id, agent_type, provider, model, embedding (vector 1536), tsv (tsvector) | SOUL registry — 16 records for MTClaw | Direct (owner_id) |
| `agent_context_files` | agent_id (FK), file_name, content | SOUL content: SOUL.md, IDENTITY.md, AGENTS.md | Via agent_id |
| `user_context_files` | agent_id (FK), user_id, file_name, content | Per-user overrides: USER.md, BOOTSTRAP.md | Via agent_id |
| `user_agent_profiles` | agent_id (FK), user_id, first_seen_at, workspace | User-SOUL interaction tracking | Via agent_id |
| `user_agent_overrides` | agent_id (FK), user_id, provider, model | Per-user model preferences | Via agent_id |
| `agent_shares` | agent_id (FK), user_id, granted_by | Access control for SOULs | Via agent_id |

### 3.2 Session & Messaging Tables

| Table | Key Columns | Purpose | Tenant Scope |
|-------|-------------|---------|-------------|
| `sessions` | session_key, agent_id (FK), input_tokens, output_tokens, messages (JSONB) | Conversation state | Via agent_id |

### 3.3 Delegation & Routing Tables

| Table | Key Columns | Purpose | Tenant Scope |
|-------|-------------|---------|-------------|
| `agent_links` | source_agent_id, target_agent_id, link_type | Delegation permissions between SOULs | Via agent_id |
| `agent_teams` | id, name, owner_id | SOUL team groups | Direct (owner_id) |
| `agent_team_members` | team_id, agent_id | Team membership | Via team_id |
| `delegation_history` | source_agent_id, target_agent_id, task, status | Audit log of SOUL-to-SOUL delegation | Via agent_id |
| `handoff_routes` | channel, chat_id, to_agent_key | Persistent route overrides | Via agent resolution |

### 3.4 Skills Tables

| Table | Key Columns | Purpose | Tenant Scope |
|-------|-------------|---------|-------------|
| `skills` | id, name, owner_id, frontmatter, content, embedding | Skill definitions (/spec, /review) | Direct (owner_id) |
| `skill_agent_grants` | skill_id, agent_id | Which SOULs have which skills | Via skill_id + agent_id |
| `skill_user_grants` | skill_id, user_id | User-level skill access | Via skill_id |

### 3.5 Memory/RAG Tables

| Table | Key Columns | Purpose | Tenant Scope |
|-------|-------------|---------|-------------|
| `memory_documents` | id, agent_id (FK), title, content, metadata | RAG source documents | Via agent_id |
| `memory_chunks` | id, agent_id (FK), document_id (FK), content, embedding (vector 1536), tsv (tsvector) | RAG chunks with pgvector + BM25 | Via agent_id |

### 3.6 Tracing/Observability Tables

| Table | Key Columns | Purpose | Tenant Scope |
|-------|-------------|---------|-------------|
| `traces` | trace_id (UUID), agent_id (FK), name, total_input_tokens, total_output_tokens, total_cost, duration_ms, status | Request-level tracing + cost | Via agent_id |
| `spans` | span_id, trace_id (FK), agent_id, model, input_tokens, output_tokens, cost | Span-level detail | Via agent_id |

### 3.7 Channel Tables

| Table | Key Columns | Purpose | Tenant Scope |
|-------|-------------|---------|-------------|
| `channel_instances` | id, agent_id (FK), channel_type, config (JSONB) | Channel connections (Telegram, Zalo) | Via agent_id |

### 3.8 Infrastructure Tables (Global — No RLS)

| Table | Key Columns | Purpose |
|-------|-------------|---------|
| `llm_providers` | id, name, base_url, config | LLM provider registry (Bflow AI-Platform) |
| `config_secrets` | key, value (encrypted), owner_id | System secrets (AES-256-GCM) |
| `builtin_tools` | name, description, config | Built-in tool registry |
| `mcp_servers` | id, name, url, config | External MCP tool providers |
| `embedding_cache` | content_hash, embedding | Content-addressed embedding cache |
| `cron_jobs` | id, agent_id, schedule, config | Scheduled tasks |
| `cron_run_logs` | id, cron_job_id, status, output | Cron execution audit |

---

## 4. MTClaw-Specific Extensions (Sprint 3+)

### 4.1 Governance Tables (New — Sprint 4-8)

These tables will be added via migrations as governance rails are built:

| Table | Sprint | Purpose | Key Columns |
|-------|--------|---------|-------------|
| `governance_specs` | 4 | Spec Factory output | spec_id, title, narrative (JSONB), acceptance_criteria (JSONB), soul_author, trace_id |
| `governance_pr_reviews` | 5 | PR Gate reviews | review_id, pr_url, verdict, score, findings (JSONB), soul_author |
| `governance_evidence` | 6 | Audit trail | evidence_id, action, actor, tenant_id, soul_role, input, output, trace_id |
| `tenant_config` | 3 | Per-tenant settings | tenant_id, monthly_token_limit, daily_request_limit, throttle_warn_pct |

### 4.2 RLS Policy Coverage

| Category | Tables | RLS Policy Pattern |
|----------|--------|-------------------|
| Direct owner_id | agents, skills | `owner_id = current_setting('app.tenant_id')` |
| Via agent_id FK | agent_context_files, sessions, traces, spans, memory_documents, memory_chunks, user_context_files | `agent_id IN (SELECT id FROM agents WHERE owner_id = ...)` |
| Global (no RLS) | llm_providers, config_secrets, builtin_tools, mcp_servers, embedding_cache | Shared infrastructure |

---

## 5. Key Indexes

| Table | Index | Columns | Purpose |
|-------|-------|---------|---------|
| agents | idx_agents_owner | owner_id | Tenant filtering |
| agents | idx_agents_key | agent_key | SOUL lookup |
| agents | idx_agents_embedding | embedding (ivfflat) | Semantic SOUL routing |
| memory_chunks | idx_memory_embedding | embedding (ivfflat) | RAG vector search |
| memory_chunks | idx_memory_tsv | tsv (GIN) | RAG full-text search (BM25) |
| traces | idx_traces_agent | agent_id | Per-SOUL trace lookup |
| traces | idx_traces_agent_created | agent_id, created_at | Cost query optimization (CTO ISSUE-2) |
| sessions | idx_sessions_key | session_key | Session lookup |

---

## 6. Data Volume Estimates (Phase 1 — MTS)

| Table | Initial Records | Growth Rate | 6-Month Estimate |
|-------|----------------|-------------|-----------------|
| agents | 16 | ~0/month (static SOULs) | 16 |
| agent_context_files | 48 (3 × 16) | ~0/month | 48 |
| sessions | 0 | ~100/month (10 users × 2/week × 5) | ~600 |
| traces | 0 | ~3,000/month (10 users × 10/day × 30) | ~18,000 |
| memory_chunks | 0 | +5,000 at RAG setup (Sprint 6) | ~5,000 |
| governance_specs | 0 | ~20/month (Sprint 4+) | ~80 |

---

## References

- [GoClaw Schema Analysis](../02-design/goclaw-schema-analysis.md) — detailed table-by-table analysis
- [RLS Tenant Isolation Design](../02-design/rls-tenant-isolation-design.md) — RLS policies + middleware
- [SOUL Loading Implementation Plan](../02-design/soul-loading-implementation-plan.md) — agent/context file usage
- [API Specification](api-specification.md) — 73 endpoints using this data model
