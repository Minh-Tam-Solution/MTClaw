-- ============================================================
-- 000011 DOWN: Remove Bflow AI-Platform provider
-- ============================================================

DELETE FROM llm_providers WHERE name = 'bflow-ai-platform';
