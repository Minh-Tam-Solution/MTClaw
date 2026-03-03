-- ============================================================
-- 000008 DOWN: Revert RLS Tenant Isolation
-- ============================================================

-- 1. Drop all RLS policies
DROP POLICY IF EXISTS tenant_isolation_agents ON agents;
DROP POLICY IF EXISTS tenant_isolation_context_files ON agent_context_files;
DROP POLICY IF EXISTS tenant_isolation_sessions ON sessions;
DROP POLICY IF EXISTS tenant_isolation_memory_docs ON memory_documents;
DROP POLICY IF EXISTS tenant_isolation_memory_chunks ON memory_chunks;
DROP POLICY IF EXISTS tenant_isolation_user_context ON user_context_files;
DROP POLICY IF EXISTS tenant_isolation_shares ON agent_shares;
DROP POLICY IF EXISTS tenant_isolation_profiles ON user_agent_profiles;
DROP POLICY IF EXISTS tenant_isolation_traces ON traces;
DROP POLICY IF EXISTS tenant_isolation_spans ON spans;
DROP POLICY IF EXISTS tenant_isolation_links ON agent_links;
DROP POLICY IF EXISTS tenant_isolation_delegation ON delegation_history;

-- 2. Disable RLS on all tables
ALTER TABLE agents DISABLE ROW LEVEL SECURITY;
ALTER TABLE agent_context_files DISABLE ROW LEVEL SECURITY;
ALTER TABLE sessions DISABLE ROW LEVEL SECURITY;
ALTER TABLE memory_documents DISABLE ROW LEVEL SECURITY;
ALTER TABLE memory_chunks DISABLE ROW LEVEL SECURITY;
ALTER TABLE user_context_files DISABLE ROW LEVEL SECURITY;
ALTER TABLE agent_shares DISABLE ROW LEVEL SECURITY;
ALTER TABLE user_agent_profiles DISABLE ROW LEVEL SECURITY;
ALTER TABLE traces DISABLE ROW LEVEL SECURITY;
ALTER TABLE spans DISABLE ROW LEVEL SECURITY;
ALTER TABLE agent_links DISABLE ROW LEVEL SECURITY;
ALTER TABLE delegation_history DISABLE ROW LEVEL SECURITY;

-- 3. Restore original unique constraint (global agent_key)
DROP INDEX IF EXISTS agents_owner_agent_key;
ALTER TABLE agents ADD CONSTRAINT agents_agent_key_key UNIQUE (agent_key);

-- 4. Drop performance index
DROP INDEX IF EXISTS idx_traces_agent_created;

-- 5. Revoke default privileges
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM mtclaw_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    REVOKE USAGE ON SEQUENCES FROM mtclaw_app;

-- 6. Revoke grants and drop roles
REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM mtclaw_app;
REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public FROM mtclaw_app;
REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM mtclaw_admin;
REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public FROM mtclaw_admin;

DO $$
BEGIN
    DROP ROLE IF EXISTS mtclaw_app;
    DROP ROLE IF EXISTS mtclaw_admin;
EXCEPTION WHEN OTHERS THEN
    -- Roles may own objects; ignore errors
    NULL;
END $$;
