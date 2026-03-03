-- ============================================================
-- 000009 DOWN: Remove seeded MTClaw SOULs (MTS Tenant)
-- ============================================================

-- Remove team memberships, teams, links, context files, and agents
-- in reverse dependency order.

-- 1. Team memberships (FK to agent_teams + agents)
DELETE FROM agent_team_members
WHERE agent_id IN (SELECT id FROM agents WHERE owner_id = 'mts' AND agent_type = 'predefined');

-- 2. Teams (FK via lead_agent_id to agents)
DELETE FROM agent_teams
WHERE lead_agent_id IN (SELECT id FROM agents WHERE owner_id = 'mts' AND agent_type = 'predefined');

-- 3. Agent links (FK to agents)
DELETE FROM agent_links
WHERE created_by = 'seed'
  AND source_agent_id IN (SELECT id FROM agents WHERE owner_id = 'mts' AND agent_type = 'predefined');

-- 4. Context files (FK to agents)
DELETE FROM agent_context_files
WHERE agent_id IN (SELECT id FROM agents WHERE owner_id = 'mts' AND agent_type = 'predefined');

-- 5. Agents
DELETE FROM agents WHERE owner_id = 'mts' AND agent_type = 'predefined';
