-- ============================================================
-- 000011 UP: Seed Bflow AI-Platform provider for MTS tenant
-- ============================================================
-- Registers the Bflow AI-Platform as an LLM provider so
-- GoClaw can route chat requests through it.
--
-- Auth: X-API-Key + X-Tenant-ID (not standard Bearer token)
-- Model: qwen3:14b (Bflow AI-Platform stable)
-- Local: http://ai-platform:8120/api/v1 (Docker ai-net network)
-- Public: https://api.nhatquangholding.com/api/v1
-- ============================================================

INSERT INTO llm_providers (name, display_name, provider_type, api_base, api_key, enabled)
VALUES (
    'bflow-ai-platform',
    'Bflow AI-Platform',
    'bflow_ai',
    'http://ai-platform:8120/api/v1',
    '',  -- API key injected via BFLOW_AI_API_KEY env var at runtime
    true
)
ON CONFLICT (name) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    provider_type = EXCLUDED.provider_type,
    api_base = EXCLUDED.api_base,
    enabled = EXCLUDED.enabled;
