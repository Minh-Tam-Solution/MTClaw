-- Sprint 10: MS Teams Extension — add channel column to governance tables.
-- CTO-37: governance_specs and pr_gate_evaluations were created without a channel
-- column, so the msteams governance processor cannot tag which channel triggered
-- the spec/review. Added as a non-breaking ALTER (nullable VARCHAR, no default).

ALTER TABLE governance_specs
    ADD COLUMN IF NOT EXISTS channel VARCHAR(32);

ALTER TABLE pr_gate_evaluations
    ADD COLUMN IF NOT EXISTS channel VARCHAR(32);

-- Optional index for channel-based filtering (e.g., list specs from msteams).
CREATE INDEX IF NOT EXISTS idx_governance_specs_channel
    ON governance_specs (owner_id, channel)
    WHERE channel IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_pr_gate_channel
    ON pr_gate_evaluations (owner_id, channel)
    WHERE channel IS NOT NULL;
