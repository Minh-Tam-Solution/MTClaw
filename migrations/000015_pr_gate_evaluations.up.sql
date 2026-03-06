-- Sprint 8: PR Gate ENFORCE — evaluation evidence storage.
CREATE TABLE pr_gate_evaluations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id        VARCHAR(64) NOT NULL,
    trace_id        UUID REFERENCES traces(id),
    pr_url          TEXT NOT NULL,
    pr_number       INTEGER NOT NULL,
    repo            VARCHAR(256) NOT NULL,
    head_sha        VARCHAR(64) NOT NULL,
    mode            VARCHAR(16) NOT NULL DEFAULT 'enforce',
    verdict         VARCHAR(16) NOT NULL,
    rules_evaluated JSONB NOT NULL DEFAULT '[]',
    review_comment  TEXT,
    soul_author     VARCHAR(64),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_pr_gate_owner ON pr_gate_evaluations (owner_id);
CREATE INDEX idx_pr_gate_repo ON pr_gate_evaluations (repo, pr_number);
CREATE INDEX idx_pr_gate_created ON pr_gate_evaluations (owner_id, created_at DESC);

-- RLS policy (same pattern as governance_specs — CTO-19 pattern)
ALTER TABLE pr_gate_evaluations ENABLE ROW LEVEL SECURITY;
CREATE POLICY pr_gate_evaluations_tenant ON pr_gate_evaluations
    USING (owner_id = current_setting('app.tenant_id', true));
