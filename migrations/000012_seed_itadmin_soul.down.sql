-- ============================================================
-- 000012 DOWN: Remove IT Admin SOUL (MTS Tenant)
-- ============================================================

-- 1. Team memberships
DELETE FROM agent_team_members
WHERE agent_id IN (SELECT id FROM agents WHERE agent_key = 'itadmin' AND owner_id = 'mts');

-- 2. Agent links (both directions)
DELETE FROM agent_links
WHERE source_agent_id IN (SELECT id FROM agents WHERE agent_key = 'itadmin' AND owner_id = 'mts')
   OR target_agent_id IN (SELECT id FROM agents WHERE agent_key = 'itadmin' AND owner_id = 'mts');

-- 3. Context files
DELETE FROM agent_context_files
WHERE agent_id IN (SELECT id FROM agents WHERE agent_key = 'itadmin' AND owner_id = 'mts');

-- 4. Agent
DELETE FROM agents WHERE agent_key = 'itadmin' AND owner_id = 'mts';
