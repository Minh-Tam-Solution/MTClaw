-- Rollback Sprint 14: Claude Code Bridge tables (ADR-010).
-- Drop in reverse dependency order.

DROP POLICY IF EXISTS bridge_audit_tenant ON bridge_audit_events;
DROP TABLE IF EXISTS bridge_audit_events;

DROP POLICY IF EXISTS bridge_permissions_tenant ON bridge_permissions;
DROP TABLE IF EXISTS bridge_permissions;

DROP POLICY IF EXISTS bridge_projects_tenant ON bridge_projects;
DROP TABLE IF EXISTS bridge_projects;

DROP POLICY IF EXISTS bridge_sessions_tenant ON bridge_sessions;
DROP TABLE IF EXISTS bridge_sessions;
