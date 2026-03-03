# RLS Tenant Isolation Design — MTClaw

**SDLC Stage**: 02-Design
**Version**: 1.0.0
**Date**: 2026-03-02
**Author**: [@pm] + [@researcher]
**Implements**: FR-001 (Multi-Tenant Architecture), ADR-002 (Three-System Architecture)

---

## 1. Current State Analysis

[@researcher]: GoClaw's schema uses `owner_id` (VARCHAR 255) as the de facto tenant identifier. Current state:

### 1.1 Tables With owner_id

| Table | Column | Current Usage |
|-------|--------|--------------|
| `agents` | `owner_id` | Agent ownership. Index: `idx_agents_owner` |
| `skills` | `owner_id` | Skill ownership. Index: `idx_skills_owner` |

### 1.2 Tables With Implicit Tenant Isolation (via FK to agents)

| Table | Isolation Path | Notes |
|-------|---------------|-------|
| `agent_context_files` | agent_id → agents.owner_id | SOUL content |
| `user_context_files` | agent_id → agents.owner_id | Per-user overrides |
| `agent_shares` | agent_id → agents.owner_id | Access control |
| `sessions` | agent_id → agents.owner_id | Conversation state |
| `memory_documents` | agent_id → agents.owner_id | RAG documents |
| `memory_chunks` | agent_id → agents.owner_id | RAG chunks (pgvector) |
| `traces` | agent_id → agents.owner_id | Usage/billing |
| `spans` | agent_id → agents.owner_id (via trace) | Detailed tracing |
| `agent_links` | source/target_agent_id → agents.owner_id | Delegation |
| `delegation_history` | source/target_agent_id → agents.owner_id | Audit |
| `cron_jobs` | agent_id → agents.owner_id | Scheduled jobs |
| `channel_instances` | agent_id → agents.owner_id | Channel config |
| `custom_tools` | agent_id → agents.owner_id | Custom tools |

### 1.3 Tables Without Tenant Isolation (Global)

| Table | Reason |
|-------|--------|
| `llm_providers` | Shared infrastructure (Bflow AI-Platform) |
| `config_secrets` | System-level secrets |
| `builtin_tools` | Built-in tool registry |
| `mcp_servers` | External tool providers |
| `embedding_cache` | Content-addressed (hash-based), tenant-agnostic |

### 1.4 Gap Analysis

| Gap | Impact | Sprint |
|-----|--------|--------|
| **No RLS policies** on any table | HIGH — app-level filtering only | Sprint 3 |
| No `tenant_id` column — uses `owner_id` string | LOW — owner_id serves same purpose | Sprint 3 (alias) |
| No middleware injecting tenant context | HIGH — requests not tenant-scoped | Sprint 3 |
| Cross-agent queries not tenant-filtered | MEDIUM — delegation could leak | Sprint 3 |
| `agent_shares` allows sharing across owners | BY DESIGN — but needs audit | Sprint 5 |

---

## 2. Design Decisions

### 2.1 Tenant Identifier Strategy

**Decision**: Use `agents.owner_id` as tenant identifier (no new column).

**Rationale**:
- GoClaw already uses `owner_id` consistently across agents and skills
- Adding a separate `tenant_id` column would require migration on all tables + code changes
- `owner_id = 'mts'` (Phase 1), `owner_id = 'nqh'` (Phase 2) — maps naturally to tenants
- Rename in code: `OwnerID` → alias as `TenantID` in MTClaw's tenant middleware

### 2.2 RLS Enforcement Level

**Decision**: PostgreSQL Row-Level Security (RLS) as mandatory enforcement layer.

**Rationale**:
- App-level filtering (WHERE owner_id = ?) is necessary but insufficient — a bug in any query leaks data
- RLS provides defense-in-depth: even if app code misses a filter, PostgreSQL blocks the row
- PostgreSQL 15+ RLS is mature, tested, and adds <5% query overhead
- BFlow pattern: single schema + tenant_id + RLS (proven at 200K users, 3 years production)

### 2.3 Multi-Tenant Unique Constraint (CTO-ISSUE-1)

**Decision**: Change `agents.agent_key` unique constraint from global to per-tenant.

**Problem**: GoClaw default has `UNIQUE(agent_key)`. With tenant-agnostic SOUL naming (`dev`, `sales`, `cs`), both MTS and NQH tenants will have agents with key `dev` → constraint violation.

**Fix** (Sprint 3 RLS migration):
```sql
-- Drop global unique constraint
DROP INDEX IF EXISTS agents_agent_key_key;

-- Create composite unique constraint (per-tenant uniqueness)
CREATE UNIQUE INDEX agents_owner_agent_key ON agents(owner_id, agent_key);
```

**Timing**: Fix in Sprint 3 (not deferred). MTS-only today, but if forgotten by Sprint 6 (NQH tenant) → production bug. Defense-in-depth: fix early.

**Impact on GoClaw code**: Agent lookup queries already filter by `owner_id` in RLS context. The unique index change only affects the constraint, not query behavior.

### 2.4 Session Variable Approach

**Decision**: Use `SET LOCAL app.tenant_id` in transaction-scoped middleware.

```sql
-- Middleware sets at start of each request
SET LOCAL app.tenant_id = 'mts';

-- RLS policies reference this variable
CREATE POLICY ... USING (owner_id = current_setting('app.tenant_id'));
```

**Why `SET LOCAL`**:
- Scoped to current transaction (auto-resets on commit/rollback)
- No risk of tenant leakage between requests
- Compatible with connection pooling (PgBouncer transaction mode)

---

## 3. RLS Policy Design

### 3.1 Core Policies (Sprint 3)

#### agents table

```sql
ALTER TABLE agents ENABLE ROW LEVEL SECURITY;
ALTER TABLE agents FORCE ROW LEVEL SECURITY;

-- Tenant sees only their agents
CREATE POLICY tenant_agents_select ON agents
  FOR SELECT
  USING (owner_id = current_setting('app.tenant_id', true));

-- Tenant can only insert agents they own
CREATE POLICY tenant_agents_insert ON agents
  FOR INSERT
  WITH CHECK (owner_id = current_setting('app.tenant_id', true));

-- Tenant can only update their agents
CREATE POLICY tenant_agents_update ON agents
  FOR UPDATE
  USING (owner_id = current_setting('app.tenant_id', true));

-- Soft delete (only own agents)
CREATE POLICY tenant_agents_delete ON agents
  FOR UPDATE
  USING (owner_id = current_setting('app.tenant_id', true));
```

#### agent_context_files (SOUL content)

```sql
ALTER TABLE agent_context_files ENABLE ROW LEVEL SECURITY;
ALTER TABLE agent_context_files FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_context_files ON agent_context_files
  FOR ALL
  USING (agent_id IN (
    SELECT id FROM agents WHERE owner_id = current_setting('app.tenant_id', true)
  ));
```

#### sessions

```sql
ALTER TABLE sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE sessions FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_sessions ON sessions
  FOR ALL
  USING (agent_id IN (
    SELECT id FROM agents WHERE owner_id = current_setting('app.tenant_id', true)
  ));
```

#### memory_chunks (RAG — critical for data isolation)

```sql
ALTER TABLE memory_chunks ENABLE ROW LEVEL SECURITY;
ALTER TABLE memory_chunks FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_memory ON memory_chunks
  FOR ALL
  USING (agent_id IN (
    SELECT id FROM agents WHERE owner_id = current_setting('app.tenant_id', true)
  ));
```

#### memory_documents

```sql
ALTER TABLE memory_documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE memory_documents FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_memory_docs ON memory_documents
  FOR ALL
  USING (agent_id IN (
    SELECT id FROM agents WHERE owner_id = current_setting('app.tenant_id', true)
  ));
```

#### traces (usage/billing)

```sql
ALTER TABLE traces ENABLE ROW LEVEL SECURITY;
ALTER TABLE traces FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_traces ON traces
  FOR ALL
  USING (agent_id IN (
    SELECT id FROM agents WHERE owner_id = current_setting('app.tenant_id', true)
  ));
```

#### spans

```sql
ALTER TABLE spans ENABLE ROW LEVEL SECURITY;
ALTER TABLE spans FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_spans ON spans
  FOR ALL
  USING (agent_id IN (
    SELECT id FROM agents WHERE owner_id = current_setting('app.tenant_id', true)
  ));
```

### 3.2 Additional Policies (Sprint 3-4)

| Table | Policy | Sprint |
|-------|--------|--------|
| `user_context_files` | Via agent_id → agents.owner_id | Sprint 3 |
| `agent_shares` | Via agent_id → agents.owner_id | Sprint 3 |
| `user_agent_profiles` | Via agent_id → agents.owner_id | Sprint 3 |
| `agent_links` | source AND target must be same tenant | Sprint 3 |
| `delegation_history` | Via source/target_agent_id | Sprint 3 |
| `handoff_routes` | Via agent resolution | Sprint 4 |
| `cron_jobs` | Via agent_id | Sprint 4 |
| `custom_tools` | Via agent_id (NULL = global, exempt) | Sprint 4 |
| `skills` | Via owner_id directly | Sprint 4 |

### 3.3 Bypass Role (Admin/Migration)

```sql
-- Admin role bypasses RLS (for migrations, admin dashboard)
CREATE ROLE mtclaw_admin;
ALTER TABLE agents OWNER TO mtclaw_admin;
-- OWNER bypasses RLS by default

-- App role has RLS enforced
CREATE ROLE mtclaw_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO mtclaw_app;
-- RLS applies to mtclaw_app
```

---

## 4. Middleware Design

### 4.1 Tenant Extraction Flow

```
Telegram message → Channel handler
  │
  ▼
Extract sender_id from Telegram update
  │
  ▼
Resolve tenant from agent assignment:
  1. Lookup agent_id from session_key or default agent
  2. agent.owner_id = tenant_id
  │
  ▼
Set PostgreSQL session variable:
  SET LOCAL app.tenant_id = '{owner_id}';
  │
  ▼
All subsequent queries in this transaction
are automatically filtered by RLS
```

### 4.2 Go Middleware (Proposed)

```go
// internal/middleware/tenant.go
func TenantMiddleware(db *sql.DB) func(ctx context.Context, tenantID string) error {
    return func(ctx context.Context, tenantID string) error {
        tx := TxFromContext(ctx)
        _, err := tx.ExecContext(ctx,
            "SET LOCAL app.tenant_id = $1", tenantID)
        return err
    }
}
```

### 4.3 Transaction Scope

```
Request lifecycle:
  1. BEGIN transaction
  2. SET LOCAL app.tenant_id = 'mts'
  3. Execute queries (RLS auto-filters)
  4. COMMIT (app.tenant_id auto-resets)
```

---

## 5. Tenant Cost Guardrails (ADR-003)

### 5.1 Configuration (Per Tenant)

Stored in `config_secrets` or a new `tenant_config` table (Sprint 3 decision):

| Config Key | Type | Phase 1 (MTS) | Phase 2 (NQH) |
|------------|------|---------------|---------------|
| `monthly_token_limit` | INT | 10,000,000 | 5,000,000 |
| `daily_request_limit` | INT | 1,000 | 500 |
| `throttle_warn_pct` | INT | 80 | 80 |
| `throttle_degrade_pct` | INT | 100 | 100 |

### 5.2 Enforcement Flow

```
Request arrives
  │
  ▼
Check daily request count:
  SELECT COUNT(*) FROM traces
  WHERE agent_id IN (tenant agents)
  AND created_at >= DATE_TRUNC('day', NOW())
  │
  ├─ count < 80% limit → PROCEED
  ├─ count >= 80% limit → WARN (log, notify admin)
  └─ count >= 100% limit → DEGRADE (shorter responses, no RAG)
  │
  ▼
Check monthly token usage:
  SELECT SUM(total_input_tokens + total_output_tokens) FROM traces
  WHERE agent_id IN (tenant agents)
  AND created_at >= DATE_TRUNC('month', NOW())
  │
  ├─ tokens < 80% limit → PROCEED
  ├─ tokens >= 80% limit → WARN
  └─ tokens >= 100% limit → DEGRADE
```

---

## 6. Testing Strategy

### 6.1 Unit Tests

```
Test: RLS blocks cross-tenant agent access
  Setup: Insert agent with owner_id='mts'
  Action: SET app.tenant_id='nqh'; SELECT from agents
  Assert: 0 rows returned

Test: RLS allows same-tenant access
  Setup: Insert agent with owner_id='mts'
  Action: SET app.tenant_id='mts'; SELECT from agents
  Assert: 1 row returned

Test: RLS blocks cross-tenant memory access
  Setup: Insert memory_chunk for mts agent
  Action: SET app.tenant_id='nqh'; SELECT from memory_chunks
  Assert: 0 rows returned (critical — RAG isolation)
```

### 6.2 Integration Tests

```
Test: Cross-tenant delegation blocked
  Setup: Agent A (mts), Agent B (nqh)
  Action: A tries to delegate to B
  Assert: Delegation fails (agent_links check + RLS)

Test: Tenant cost guardrail triggers
  Setup: daily_request_limit = 10
  Action: Send 11 requests
  Assert: 11th request gets degraded response

Test: Session isolation
  Setup: User X has session with mts SOUL
  Action: Switch tenant to nqh
  Assert: Session not accessible
```

---

## 7. Implementation Timeline

| Sprint | Task | Effort | Priority |
|--------|------|--------|----------|
| **Sprint 3 Day 1** | Create RLS migration (agents, sessions, memory) | 1 day | P0 |
| **Sprint 3 Day 2** | Implement tenant middleware (SET LOCAL) | 1 day | P0 |
| **Sprint 3 Day 3** | RLS policies for remaining tables | 1 day | P0 |
| **Sprint 3 Day 4** | Cross-tenant isolation tests | 1 day | P0 |
| **Sprint 3 Day 5** | Tenant cost guardrail implementation | 1 day | P1 |
| **Sprint 4** | Admin bypass role + dashboard queries | 2 days | P1 |
| **Sprint 5** | Penetration testing for tenant isolation | 1 day | P1 |

---

## References

- FR-001: Multi-Tenant Architecture (`docs/01-planning/requirements.md`)
- ADR-002: Three-System Architecture (`docs/02-design/01-ADRs/SPEC-0002-ADR-002-Three-System-Architecture.md`)
- ADR-003: Observability + Tenant Cost Guardrails (`docs/02-design/01-ADRs/SPEC-0003-ADR-003-Observability-Architecture.md`)
- GoClaw Schema Analysis: `docs/02-design/goclaw-schema-analysis.md`
- BFlow RLS Pattern: Row-level security at 200K users, 3 years production
