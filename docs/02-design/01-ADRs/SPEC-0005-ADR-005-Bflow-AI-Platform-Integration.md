---
spec_id: SPEC-0005
adr_id: ADR-005
title: Bflow AI-Platform as Single AI Provider
status: APPROVED
date: 2026-03-02
author: "[@pm]"
reviewers: "[@cto], [@cpo]"
sdlc_version: "6.1.1"
implements: "FR-004"
---

# ADR-005: Bflow AI-Platform as Single AI Provider

**SDLC Stage**: 02-Design
**Status**: APPROVED
**Date**: 2026-03-02

---

## Context

MTClaw needs an AI backend for:
1. Chat completions (16 SOULs × multiple concurrent users)
2. RAG queries (knowledge collections per domain)
3. Translation (Vietnamese ↔ English for multi-language support)

Three options were evaluated:

| Option | Provider | Cost | Latency | Control |
|--------|----------|------|---------|---------|
| A | Direct Ollama (ai.nhatquangholding.com:11434) | $0 | <3s | Full (but raw API) |
| B | **Bflow AI-Platform (api.nhatquangholding.com)** | $0 | <5s | Full (enterprise features) |
| C | External API (Claude/GPT-4o) | $500-1,000/mo | <5s | Limited |

## Decision

**Option B: Bflow AI-Platform as the single source of AI infrastructure.**

No direct LLM calls bypass AI-Platform for any inference.

## Rationale

1. **Enterprise features**: AI-Platform provides centralized auth (X-API-Key), tenant isolation (X-Tenant-ID), rate limiting, audit logging — Ollama direct has none of these
2. **RAG built-in**: `POST /v1/rag/query` with collection filter eliminates need for custom RAG pipeline
3. **Cost control**: Single billing point per tenant, token usage tracked centrally
4. **Model management**: Model upgrades (qwen2.5 → qwen3:14b) happen once at platform level, all consumers benefit
5. **Proven pattern**: SOP Generator already uses AI-Platform successfully (`backend/services/sop_generation_service/app/services/rag_client.py`)
6. **Zero cost**: Internal infrastructure (RTX 5090 32GB), no per-query charges

## Integration Specification

### Chat Completions

```
POST https://api.nhatquangholding.com/v1/chat/completions
Headers:
  X-API-Key: aip_c786...
  X-Tenant-ID: mts
  Content-Type: application/json
Body:
  {
    "model": "qwen3:14b",
    "messages": [
      {"role": "system", "content": "{SOUL system prompt}"},
      {"role": "user", "content": "{user message}"}
    ]
  }
```

### RAG Query

```
POST https://api.nhatquangholding.com/api/v1/rag/query
Headers:
  X-API-Key: aip_c786...
Body:
  {
    "query": "Quy trình xử lý khiếu nại",
    "collection": "mts-hr-policies",
    "max_results": 5
  }
```

### Configuration

| Env Var | Value | Purpose |
|---------|-------|---------|
| `BFLOW_AI_API_KEY` | `aip_c786...` | API authentication |
| `BFLOW_AI_BASE_URL` | `https://api.nhatquangholding.com` | Platform endpoint (local: `http://ai-platform:8120` via ai-net) |
| `BFLOW_TENANT_ID` | `mts` | Tenant identification |

### Fallback Strategy

```
AI-Platform request
  │
  ├─ Success → return response
  │
  └─ Failure (timeout/5xx)
       │
       ├─ Retry once (with backoff)
       │
       └─ If still fails:
            → Log error with trace_id
            → Return user-friendly message:
              "AI-Platform tạm thời không khả dụng. Vui lòng thử lại sau."
            → DO NOT fallback to direct Ollama or external API
```

## Consequences

### Positive
- Single audit trail for all AI usage across MTClaw
- Centralized cost tracking and tenant isolation
- Model upgrades are transparent to MTClaw
- Consistent API format (OpenAI-compatible)

### Negative
- Single point of failure — if AI-Platform is down, MTClaw AI features are unavailable
- Latency may be slightly higher than direct Ollama (~1-2s overhead for platform layer)
- Limited to models available on AI-Platform (currently qwen3 family)

### Risks
- AI-Platform downtime → MTClaw degraded (mitigation: graceful degradation, retry)
- Model quality regression → SOUL response quality drops (mitigation: SOUL quality rubric monitoring)

---

## References

- [ADR-002: Three-System Architecture](SPEC-0002-ADR-002-Three-System-Architecture.md)
- [FR-004: Bflow AI-Platform Integration](../../01-planning/requirements.md)
- [SOP Generator RAG Client](ref: Bflow-Platform/Sub-Repo/SOP-Generator/backend/services/sop_generation_service/app/services/rag_client.py)
- [Bflow AI-Platform Integration Guide](../../03-integrate/bflow-ai-platform-sop-generator-guide.md)
