-- Sprint 7: SOUL Drift Detection (ADR-004)
-- Separate from 000013 (governance_specs) per CTO-17 directive.

ALTER TABLE agents ADD COLUMN IF NOT EXISTS content_checksum VARCHAR(64);
ALTER TABLE agents ADD COLUMN IF NOT EXISTS soul_version VARCHAR(10);

CREATE TABLE IF NOT EXISTS soul_drift_events (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id     UUID NOT NULL REFERENCES agents(id),
    agent_key    VARCHAR(32) NOT NULL,
    old_checksum VARCHAR(64),
    new_checksum VARCHAR(64),
    old_version  VARCHAR(10),
    new_version  VARCHAR(10),
    drift_type   VARCHAR(16) NOT NULL,
    detected_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_soul_drift_agent ON soul_drift_events (agent_id, detected_at DESC);
