-- Sprint 11: Evidence Linking — ADR-009 junction table for cross-rail traceability.
-- Links governance artifacts (specs, PR gates, future test_runs, deploys) into
-- queryable evidence chains. N:M relationship, extensible without schema changes.

CREATE TABLE evidence_links (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id    VARCHAR(64) NOT NULL,
    from_type   VARCHAR(32) NOT NULL,  -- 'spec', 'pr_gate', 'test_run', 'deploy'
    from_id     UUID NOT NULL,
    to_type     VARCHAR(32) NOT NULL,
    to_id       UUID NOT NULL,
    link_reason VARCHAR(64),           -- 'manual', 'auto_spec_review', 'auto_pr_merge'
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (owner_id, from_type, from_id, to_type, to_id)
);

CREATE INDEX idx_evidence_links_owner ON evidence_links (owner_id);
CREATE INDEX idx_evidence_links_from ON evidence_links (owner_id, from_type, from_id);
CREATE INDEX idx_evidence_links_to ON evidence_links (owner_id, to_type, to_id);

ALTER TABLE evidence_links ENABLE ROW LEVEL SECURITY;
CREATE POLICY evidence_links_tenant ON evidence_links
    USING (owner_id = current_setting('app.tenant_id', true));
