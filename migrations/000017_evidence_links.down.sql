-- Sprint 11: Rollback evidence_links table (ADR-009).
DROP POLICY IF EXISTS evidence_links_tenant ON evidence_links;
DROP TABLE IF EXISTS evidence_links;
