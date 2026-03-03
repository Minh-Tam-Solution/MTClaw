-- 000010 DOWN: Remove observability columns

DROP INDEX IF EXISTS idx_traces_tenant_soul;
DROP INDEX IF EXISTS idx_traces_tenant_time;

ALTER TABLE spans DROP COLUMN IF EXISTS agent_key;
ALTER TABLE spans DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE traces DROP COLUMN IF EXISTS agent_key;
ALTER TABLE traces DROP COLUMN IF EXISTS tenant_id;
