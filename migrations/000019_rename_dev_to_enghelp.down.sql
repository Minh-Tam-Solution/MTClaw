-- Revert enghelp back to dev
UPDATE agents SET agent_key = 'dev', display_name = 'Developer Assistant'
WHERE agent_key = 'enghelp';

UPDATE agent_links SET description = REPLACE(description, 'to Engineering Helper', 'to Dev')
WHERE description LIKE '%to Engineering Helper%';
