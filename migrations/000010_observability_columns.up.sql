-- ============================================================
-- 000010: Observability Enhancement — tenant_id + agent_key on traces/spans
-- Implements: ADR-003 (Observability Architecture), US-017
-- Sprint: 3 (P1)
--
-- Adds tenant_id and agent_key columns to traces and spans tables
-- for structured logging, OTEL attributes, and tenant cost guardrails.
-- ============================================================

-- 1. traces: add tenant_id + agent_key for direct query without JOIN
ALTER TABLE traces ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(255);
ALTER TABLE traces ADD COLUMN IF NOT EXISTS agent_key VARCHAR(100);

-- 2. spans: add tenant_id + agent_key for OTEL attribute export
ALTER TABLE spans ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(255);
ALTER TABLE spans ADD COLUMN IF NOT EXISTS agent_key VARCHAR(100);

-- 3. Index for tenant cost guardrail queries:
--    SELECT SUM(tokens) FROM traces WHERE tenant_id = 'mts' AND created_at >= ...
CREATE INDEX IF NOT EXISTS idx_traces_tenant_time
    ON traces(tenant_id, created_at DESC) WHERE tenant_id IS NOT NULL;

-- 4. Index for per-SOUL usage reporting:
--    SELECT agent_key, COUNT(*) FROM traces WHERE tenant_id = 'mts' GROUP BY agent_key
CREATE INDEX IF NOT EXISTS idx_traces_tenant_soul
    ON traces(tenant_id, agent_key) WHERE tenant_id IS NOT NULL;
