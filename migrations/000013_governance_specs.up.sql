-- Sprint 7: Governance Specs — Rail #1 Spec Factory Full
-- Creates governance_specs table for structured specification storage.
-- Reference: SDLC Orchestrator GovernanceSpecification pattern (simplified for MTClaw scale).

CREATE TABLE IF NOT EXISTS governance_specs (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id              VARCHAR(64) NOT NULL,
    spec_id               VARCHAR(16) NOT NULL,
    spec_version          VARCHAR(10) NOT NULL DEFAULT '1.0.0',
    title                 VARCHAR(255) NOT NULL,
    narrative             JSONB NOT NULL,
    acceptance_criteria   JSONB NOT NULL,
    bdd_scenarios         JSONB,
    risks                 JSONB,
    technical_requirements JSONB,
    dependencies          JSONB,
    priority              VARCHAR(4) NOT NULL DEFAULT 'P1',
    estimated_effort      VARCHAR(4) DEFAULT 'M',
    status                VARCHAR(16) NOT NULL DEFAULT 'draft',
    tier                  VARCHAR(16) NOT NULL DEFAULT 'STANDARD',
    soul_author           VARCHAR(32) NOT NULL,
    trace_id              UUID,
    content_hash          VARCHAR(64),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (owner_id, spec_id)
);

CREATE INDEX idx_governance_specs_owner ON governance_specs (owner_id);
CREATE INDEX idx_governance_specs_specid ON governance_specs (spec_id);
CREATE INDEX idx_governance_specs_status ON governance_specs (owner_id, status);
CREATE INDEX idx_governance_specs_created ON governance_specs (owner_id, created_at DESC);
CREATE INDEX idx_governance_specs_trace ON governance_specs (trace_id);

-- RLS policy (same pattern as agents table — tenant isolation via owner_id).
ALTER TABLE governance_specs ENABLE ROW LEVEL SECURITY;
CREATE POLICY governance_specs_tenant_isolation ON governance_specs
    USING (owner_id = current_setting('app.tenant_id', true));
