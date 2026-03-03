-- ============================================================
-- 000008: Row-Level Security (RLS) Tenant Isolation
-- Implements: FR-001, ADR-002, rls-tenant-isolation-design.md
-- Sprint: 3 (P0 — US-015)
--
-- Tenant identifier: agents.owner_id (VARCHAR 255)
-- Session variable: SET LOCAL app.tenant_id = '{owner_id}'
-- Pattern: BFlow proven (200K users, 3 years production)
-- ============================================================

-- ============================================================
-- 1. Database Roles
-- ============================================================

-- mtclaw_admin: bypasses RLS (migrations, admin dashboard, seeding)
-- mtclaw_app: enforced RLS (application queries)
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'mtclaw_admin') THEN
        CREATE ROLE mtclaw_admin WITH BYPASSRLS;
    END IF;
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'mtclaw_app') THEN
        CREATE ROLE mtclaw_app;
    END IF;
END $$;

-- Grant table access to both roles
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO mtclaw_app;
GRANT USAGE ON ALL SEQUENCES IN SCHEMA public TO mtclaw_app;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO mtclaw_admin;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO mtclaw_admin;

-- ============================================================
-- 2. Fix Unique Constraints for Multi-Tenant (CTO-ISSUE-1)
-- ============================================================

-- agent_key must be unique PER TENANT, not globally.
-- Without this fix, MTS tenant 'dev' blocks NQH tenant 'dev'.
ALTER TABLE agents DROP CONSTRAINT IF EXISTS agents_agent_key_key;
DROP INDEX IF EXISTS agents_agent_key_key;
CREATE UNIQUE INDEX agents_owner_agent_key ON agents(owner_id, agent_key);

-- ============================================================
-- 3. Performance Index (CTO-ISSUE-2)
-- ============================================================

-- RLS subqueries on traces filter by agent_id + time range.
-- This index supports cost guardrail queries:
--   SELECT SUM(tokens) FROM traces WHERE agent_id IN (...) AND created_at >= ...
CREATE INDEX IF NOT EXISTS idx_traces_agent_created
    ON traces(agent_id, created_at) WHERE agent_id IS NOT NULL;

-- ============================================================
-- 4. RLS on Direct owner_id Tables
-- ============================================================

-- 4.1 agents — primary tenant table
ALTER TABLE agents ENABLE ROW LEVEL SECURITY;
ALTER TABLE agents FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_agents ON agents
    FOR ALL
    USING (owner_id = current_setting('app.tenant_id', true))
    WITH CHECK (owner_id = current_setting('app.tenant_id', true));

-- ============================================================
-- 5. RLS on FK Tables (agent_id → agents.owner_id)
-- ============================================================
-- Pattern: USING (agent_id IN (SELECT id FROM agents WHERE owner_id = ...))
-- Safety: if app.tenant_id is not set, current_setting returns '' → no rows match → fail-safe

-- 5.1 agent_context_files (SOUL content — critical for SOUL isolation)
ALTER TABLE agent_context_files ENABLE ROW LEVEL SECURITY;
ALTER TABLE agent_context_files FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_context_files ON agent_context_files
    FOR ALL
    USING (agent_id IN (
        SELECT id FROM agents WHERE owner_id = current_setting('app.tenant_id', true)
    ));

-- 5.2 sessions (conversation state)
ALTER TABLE sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE sessions FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_sessions ON sessions
    FOR ALL
    USING (agent_id IN (
        SELECT id FROM agents WHERE owner_id = current_setting('app.tenant_id', true)
    ));

-- 5.3 memory_documents (RAG documents)
ALTER TABLE memory_documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE memory_documents FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_memory_docs ON memory_documents
    FOR ALL
    USING (agent_id IN (
        SELECT id FROM agents WHERE owner_id = current_setting('app.tenant_id', true)
    ));

-- 5.4 memory_chunks (RAG chunks — critical: tenant RAG data isolation)
ALTER TABLE memory_chunks ENABLE ROW LEVEL SECURITY;
ALTER TABLE memory_chunks FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_memory_chunks ON memory_chunks
    FOR ALL
    USING (agent_id IN (
        SELECT id FROM agents WHERE owner_id = current_setting('app.tenant_id', true)
    ));

-- 5.5 user_context_files (per-user overrides)
ALTER TABLE user_context_files ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_context_files FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_user_context ON user_context_files
    FOR ALL
    USING (agent_id IN (
        SELECT id FROM agents WHERE owner_id = current_setting('app.tenant_id', true)
    ));

-- 5.6 agent_shares (access control)
ALTER TABLE agent_shares ENABLE ROW LEVEL SECURITY;
ALTER TABLE agent_shares FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_shares ON agent_shares
    FOR ALL
    USING (agent_id IN (
        SELECT id FROM agents WHERE owner_id = current_setting('app.tenant_id', true)
    ));

-- 5.7 user_agent_profiles (user-agent association)
ALTER TABLE user_agent_profiles ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_agent_profiles FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_profiles ON user_agent_profiles
    FOR ALL
    USING (agent_id IN (
        SELECT id FROM agents WHERE owner_id = current_setting('app.tenant_id', true)
    ));

-- ============================================================
-- 6. RLS on Tracing Tables (agent_id, no FK constraint)
-- ============================================================
-- traces.agent_id is nullable with no FK. NULL agent_id → hidden (safe).

-- 6.1 traces (usage/billing — critical for tenant cost guardrails)
ALTER TABLE traces ENABLE ROW LEVEL SECURITY;
ALTER TABLE traces FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_traces ON traces
    FOR ALL
    USING (agent_id IN (
        SELECT id FROM agents WHERE owner_id = current_setting('app.tenant_id', true)
    ));

-- 6.2 spans (detailed tracing)
ALTER TABLE spans ENABLE ROW LEVEL SECURITY;
ALTER TABLE spans FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_spans ON spans
    FOR ALL
    USING (agent_id IN (
        SELECT id FROM agents WHERE owner_id = current_setting('app.tenant_id', true)
    ));

-- ============================================================
-- 7. RLS on Cross-Agent Tables (source + target must be same tenant)
-- ============================================================

-- 7.1 agent_links (delegation permissions)
-- USING: filter by source (SELECT shows only own links)
-- WITH CHECK: both source AND target must belong to tenant (prevents cross-tenant links)
ALTER TABLE agent_links ENABLE ROW LEVEL SECURITY;
ALTER TABLE agent_links FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_links ON agent_links
    FOR ALL
    USING (source_agent_id IN (
        SELECT id FROM agents WHERE owner_id = current_setting('app.tenant_id', true)
    ))
    WITH CHECK (
        source_agent_id IN (
            SELECT id FROM agents WHERE owner_id = current_setting('app.tenant_id', true)
        )
        AND target_agent_id IN (
            SELECT id FROM agents WHERE owner_id = current_setting('app.tenant_id', true)
        )
    );

-- 7.2 delegation_history (audit trail)
ALTER TABLE delegation_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE delegation_history FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_delegation ON delegation_history
    FOR ALL
    USING (source_agent_id IN (
        SELECT id FROM agents WHERE owner_id = current_setting('app.tenant_id', true)
    ))
    WITH CHECK (
        source_agent_id IN (
            SELECT id FROM agents WHERE owner_id = current_setting('app.tenant_id', true)
        )
        AND target_agent_id IN (
            SELECT id FROM agents WHERE owner_id = current_setting('app.tenant_id', true)
        )
    );

-- ============================================================
-- 8. Default Grants for Future Tables
-- ============================================================

-- Ensure mtclaw_app gets access to tables created by future migrations
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO mtclaw_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT USAGE ON SEQUENCES TO mtclaw_app;
