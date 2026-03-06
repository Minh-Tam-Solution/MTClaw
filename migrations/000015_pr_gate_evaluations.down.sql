DROP POLICY IF EXISTS pr_gate_evaluations_tenant ON pr_gate_evaluations;
DROP INDEX IF EXISTS idx_pr_gate_created;
DROP INDEX IF EXISTS idx_pr_gate_repo;
DROP INDEX IF EXISTS idx_pr_gate_owner;
DROP TABLE IF EXISTS pr_gate_evaluations;
