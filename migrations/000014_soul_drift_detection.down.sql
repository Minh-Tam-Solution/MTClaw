DROP INDEX IF EXISTS idx_soul_drift_agent;
DROP TABLE IF EXISTS soul_drift_events;
ALTER TABLE agents DROP COLUMN IF EXISTS content_checksum;
ALTER TABLE agents DROP COLUMN IF EXISTS soul_version;
