-- Rollback Sprint 10 channel column additions.
DROP INDEX IF EXISTS idx_pr_gate_channel;
DROP INDEX IF EXISTS idx_governance_specs_channel;

ALTER TABLE pr_gate_evaluations DROP COLUMN IF EXISTS channel;
ALTER TABLE governance_specs    DROP COLUMN IF EXISTS channel;
