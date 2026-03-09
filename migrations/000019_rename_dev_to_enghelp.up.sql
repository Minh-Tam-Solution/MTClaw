-- Rename dev agent to enghelp (Engineering Helper).
-- Idempotent: no-op if agent_key 'dev' does not exist (e.g., fresh deploy from updated 000009).

UPDATE agents SET agent_key = 'enghelp', display_name = 'Engineering Helper'
WHERE agent_key = 'dev';

-- Update agent_links descriptions referencing Dev
UPDATE agent_links SET description = REPLACE(description, 'to Dev', 'to Engineering Helper')
WHERE description LIKE '%to Dev%';
