-- Sprint 14: Claude Code Bridge tables (ADR-010).
-- 4 tables: sessions, projects, permissions, audit_events.
-- All tables use owner_id for tenant isolation with RLS.

-- 1. Bridge sessions — tracks active/completed Claude Code terminal sessions.
CREATE TABLE bridge_sessions (
    id                    VARCHAR(32) PRIMARY KEY,  -- br:{tenant8}:{rand8}
    owner_id              VARCHAR(64) NOT NULL,
    agent_type            VARCHAR(32) NOT NULL DEFAULT 'claude-code',
    tmux_target           VARCHAR(64) NOT NULL,
    project_path          TEXT NOT NULL,
    workspace_fingerprint VARCHAR(128) NOT NULL,
    status                VARCHAR(16) NOT NULL DEFAULT 'active',
    risk_mode             VARCHAR(16) NOT NULL DEFAULT 'read',
    capabilities          JSONB NOT NULL DEFAULT '{}',
    owner_actor_id        VARCHAR(64) NOT NULL,
    approver_acl          JSONB NOT NULL DEFAULT '[]',
    notify_acl            JSONB NOT NULL DEFAULT '[]',
    user_id               VARCHAR(64),
    channel               VARCHAR(32),
    chat_id               VARCHAR(64),
    interactive_eligible  BOOLEAN NOT NULL DEFAULT false,
    hook_secret           VARCHAR(128),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_activity_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    stopped_at            TIMESTAMPTZ
);

CREATE INDEX idx_bridge_sessions_owner ON bridge_sessions (owner_id);
CREATE INDEX idx_bridge_sessions_owner_status ON bridge_sessions (owner_id, status);
CREATE INDEX idx_bridge_sessions_actor ON bridge_sessions (owner_actor_id, status);

ALTER TABLE bridge_sessions ENABLE ROW LEVEL SECURITY;
CREATE POLICY bridge_sessions_tenant ON bridge_sessions
    FOR ALL
    USING (owner_id = current_setting('app.tenant_id', true))
    WITH CHECK (owner_id = current_setting('app.tenant_id', true));

-- 2. Bridge projects — registered project directories for sessions.
CREATE TABLE bridge_projects (
    id          VARCHAR(32) PRIMARY KEY,
    owner_id    VARCHAR(64) NOT NULL,
    name        VARCHAR(128) NOT NULL,
    path        TEXT NOT NULL,
    agent_type  VARCHAR(32) NOT NULL DEFAULT 'claude-code',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (owner_id, name)
);

CREATE INDEX idx_bridge_projects_owner ON bridge_projects (owner_id);

ALTER TABLE bridge_projects ENABLE ROW LEVEL SECURITY;
CREATE POLICY bridge_projects_tenant ON bridge_projects
    FOR ALL
    USING (owner_id = current_setting('app.tenant_id', true))
    WITH CHECK (owner_id = current_setting('app.tenant_id', true));

-- 3. Bridge permissions — tracks permission requests and approvals (Sprint 16/C).
-- Created now with empty data; populated when HookServer is wired.
CREATE TABLE bridge_permissions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id        VARCHAR(64) NOT NULL,
    session_id      VARCHAR(32) NOT NULL REFERENCES bridge_sessions(id),
    tool_name       VARCHAR(128) NOT NULL,
    tool_input      JSONB,
    decision        VARCHAR(16),  -- 'pending', 'approved', 'denied', 'timeout'
    decided_by      VARCHAR(64),
    decided_at      TIMESTAMPTZ,
    hmac_verified   BOOLEAN DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_bridge_permissions_session ON bridge_permissions (session_id, created_at DESC);
CREATE INDEX idx_bridge_permissions_owner ON bridge_permissions (owner_id);

ALTER TABLE bridge_permissions ENABLE ROW LEVEL SECURITY;
CREATE POLICY bridge_permissions_tenant ON bridge_permissions
    FOR ALL
    USING (owner_id = current_setting('app.tenant_id', true))
    WITH CHECK (owner_id = current_setting('app.tenant_id', true));

-- 4. Bridge audit events — immutable audit trail for all bridge actions (L3).
-- Primary write target is JSONL files; PG is secondary (best-effort).
CREATE TABLE bridge_audit_events (
    id          BIGSERIAL PRIMARY KEY,
    owner_id    VARCHAR(64) NOT NULL,
    session_id  VARCHAR(32),
    actor_id    VARCHAR(64) NOT NULL,
    action      VARCHAR(64) NOT NULL,
    risk_mode   VARCHAR(16),
    detail      JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_bridge_audit_owner ON bridge_audit_events (owner_id, created_at DESC);
CREATE INDEX idx_bridge_audit_session ON bridge_audit_events (session_id, created_at DESC);

ALTER TABLE bridge_audit_events ENABLE ROW LEVEL SECURITY;
CREATE POLICY bridge_audit_tenant ON bridge_audit_events
    FOR ALL
    USING (owner_id = current_setting('app.tenant_id', true))
    WITH CHECK (owner_id = current_setting('app.tenant_id', true));
