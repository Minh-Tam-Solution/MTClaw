# System Architecture Document — MTClaw

**SDLC Stage**: 02-Design
**Version**: 1.0.0
**Date**: 2026-03-02
**Author**: [@architect]
**Reviewer**: [@cto]
**Framework**: SDLC 6.1.1 — Stage 02 Required Artifact (STANDARD tier)
**Implements**: US-014 (System Architecture Document)

---

## 1. Executive Summary

MTClaw is a governance-first company assistant platform built on the GoClaw runtime (Go 1.25). It provides 16 role-aware AI personas (SOULs) + 3 governance rails (Spec Factory, PR Gate, Knowledge & Answering) for MTS employees via Telegram, with multi-tenant isolation for future NQH expansion.

This document defines the system architecture across 5 views: Component, Data Flow, Deployment, Security, and Integration Points.

---

## 2. Component Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          LAYER 1: USER CHANNELS                         │
│                                                                         │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────────────────┐   │
│  │  Telegram Bot │   │   Zalo Bot   │   │  Future: Web UI / API    │   │
│  │  (Phase 1)    │   │   (Phase 2)  │   │  (Phase 3)               │   │
│  └──────┬───────┘   └──────┬───────┘   └────────────┬─────────────┘   │
│         │                   │                         │                 │
│         └───────────────────┼─────────────────────────┘                 │
│                             │                                           │
│                    InboundMessage / OutboundMessage                     │
│                    (unified channel abstraction)                        │
└─────────────────────────────┬───────────────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────────────┐
│                       LAYER 2: GOCLAW GATEWAY                           │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    HTTP Server (net/http)                        │   │
│  │  Port 8080 — 55 inherited endpoints + 18 governance endpoints   │   │
│  └──────────────────────────┬──────────────────────────────────────┘   │
│                              │                                          │
│  ┌──────────────────────────▼──────────────────────────────────────┐   │
│  │              RLS Middleware (tenant.go)                          │   │
│  │  1. Extract tenant_id from JWT / channel context                │   │
│  │  2. SET LOCAL app.tenant_id = '{tenant_id}'                     │   │
│  │  3. All downstream queries scoped to tenant                     │   │
│  └──────────────────────────┬──────────────────────────────────────┘   │
│                              │                                          │
│  ┌──────────────────────────▼──────────────────────────────────────┐   │
│  │              Session Resolver                                    │   │
│  │  channel + chat_id → session_key → agent routing                │   │
│  └──────────────────────────┬──────────────────────────────────────┘   │
│                              │                                          │
└──────────────────────────────┼──────────────────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────────────────┐
│                       LAYER 3: AGENT LOOP                               │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  Agent Loader (bootstrap/load_store.go)                         │   │
│  │  → LoadFromStore() loads agent config from PostgreSQL            │   │
│  │  → Predefined mode: all 16 SOULs from agents table              │   │
│  └──────────────────────────┬──────────────────────────────────────┘   │
│                              │                                          │
│  ┌──────────────────────────▼──────────────────────────────────────┐   │
│  │  Context File Loader (store/pg/agents_context.go)               │   │
│  │  → Loads SOUL.md, IDENTITY.md, AGENTS.md from                   │   │
│  │    agent_context_files table                                     │   │
│  │  → Loads USER.md, BOOTSTRAP.md from user_context_files          │   │
│  └──────────────────────────┬──────────────────────────────────────┘   │
│                              │                                          │
│  ┌──────────────────────────▼──────────────────────────────────────┐   │
│  │  System Prompt Builder (agent/systemprompt.go)                  │   │
│  │  → BuildSystemPrompt() — 15 sections:                           │   │
│  │    [1] BasePrompt                                                │   │
│  │    [2] SOUL.md (role identity + constraints)                     │   │
│  │    [3] IDENTITY.md (name, emoji, vibe)                          │   │
│  │    [4] AGENTS.md (workspace rules)                              │   │
│  │    [5] USER.md (per-user overrides)                             │   │
│  │    [6] TOOLS.md (available tools description)                   │   │
│  │    [7] ExtraPrompt ← governance rail injection point            │   │
│  │    [8] ContextFiles ← RAG context injection point               │   │
│  │    [9-15] Session state, history, metadata                      │   │
│  └──────────────────────────┬──────────────────────────────────────┘   │
│                              │                                          │
│  ┌──────────────────────────▼──────────────────────────────────────┐   │
│  │  Skills System                                                   │   │
│  │  → /spec (Rail #1 — Spec Factory, Sprint 4)                     │   │
│  │  → /review (Rail #2 — PR Gate, Sprint 5)                        │   │
│  │  → RAG query (Rail #3 — Knowledge, Sprint 6)                    │   │
│  └──────────────────────────┬──────────────────────────────────────┘   │
│                              │                                          │
│  ┌──────────────────────────▼──────────────────────────────────────┐   │
│  │  Delegation Router (agent_links)                                 │   │
│  │  → SOUL-to-SOUL delegation (assistant → pm, pm → coder)          │   │
│  │  → Team-based routing (Engineering, Business, Advisors)          │   │
│  └──────────────────────────┬──────────────────────────────────────┘   │
│                              │                                          │
└──────────────────────────────┼──────────────────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────────────────┐
│                    LAYER 4: AI-PLATFORM                                  │
│                                                                         │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │  Bflow AI-Platform (api.nhatquangholding.com)                     │  │
│  │                                                                   │  │
│  │  POST /v1/chat/completions                                        │  │
│  │    → Chat inference (qwen3:14b)                                   │  │
│  │    → Headers: X-API-Key, X-Tenant-ID                              │  │
│  │                                                                   │  │
│  │  POST /api/v1/rag/query                                           │  │
│  │    → RAG search (collection filter + hybrid vector/BM25)          │  │
│  │                                                                   │  │
│  │  POST /v1/translations                                            │  │
│  │    → Vietnamese ↔ English translation                             │  │
│  └──────────────────────────────────────────────────────────────────┘  │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────────────────┐
│                    LAYER 5: DATABASE                                     │
│                                                                         │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │  PostgreSQL 15 + pgvector 0.5+                                   │  │
│  │                                                                   │  │
│  │  ┌─────────────┐  ┌───────────────┐  ┌──────────────────────┐  │  │
│  │  │ agents (16)  │  │ sessions      │  │ traces + spans       │  │  │
│  │  │ + context    │  │ + messages    │  │ (observability)      │  │  │
│  │  │   files (48) │  │               │  │                      │  │  │
│  │  └─────────────┘  └───────────────┘  └──────────────────────┘  │  │
│  │                                                                   │  │
│  │  ┌─────────────┐  ┌───────────────┐  ┌──────────────────────┐  │  │
│  │  │ memory_docs  │  │ skills        │  │ llm_providers        │  │  │
│  │  │ + chunks     │  │ + grants      │  │ + config_secrets     │  │  │
│  │  │ (pgvector)   │  │               │  │ (global, no RLS)     │  │  │
│  │  └─────────────┘  └───────────────┘  └──────────────────────┘  │  │
│  │                                                                   │  │
│  │  RLS ENFORCED: agents, agent_context_files, sessions,             │  │
│  │    memory_documents, memory_chunks, traces, spans,                │  │
│  │    user_context_files                                             │  │
│  └──────────────────────────────────────────────────────────────────┘  │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### Component Summary

| Layer | Component | Technology | Purpose |
|-------|-----------|-----------|---------|
| 1 | Channel Handlers | Telegram Bot API, Zalo API | User-facing messaging interfaces |
| 2 | Gateway | GoClaw HTTP (net/http) | Request routing, RLS middleware, session management |
| 3 | Agent Loop | GoClaw internals | SOUL loading, prompt building, skills, delegation |
| 4 | AI-Platform | Bflow (qwen3:14b) | Chat inference, RAG, translation |
| 5 | Database | PostgreSQL 15 + pgvector | Persistence, RLS, vector search |

---

## 3. Data Flow Diagram

### 3.1 Full Request Lifecycle

```
User sends Telegram message: "Tạo spec cho tính năng login"
  │
  ▼
[1] Telegram Bot API receives update
  │  internal/channels/telegram/handler.go
  │  → Parse message, extract user_id, chat_id
  │  → Create InboundMessage{channel: "telegram", chatID, userID, text}
  │
  ▼
[2] Session Resolution
  │  internal/channels/telegram/session.go
  │  → Lookup: channel + chat_id → session_key
  │  → If new: create session, resolve default agent (assistant, is_default=true)
  │  → If existing: load session.agent_id
  │
  ▼
[3] ★ RLS MIDDLEWARE ★ (tenant isolation boundary)
  │  internal/middleware/tenant.go (NEW — Sprint 3)
  │  → Extract owner_id from agent → tenant_id
  │  → BEGIN transaction
  │  → SET LOCAL app.tenant_id = 'mts'     ← ★ BEFORE any DB query
  │  → All subsequent queries in this transaction are tenant-scoped
  │
  ▼
[4] Agent Loading
  │  internal/bootstrap/load_store.go
  │  → LoadFromStore(agent_id) → agent config (model, provider, tools)
  │  → Agent type: "predefined" → load from DB (not filesystem)
  │
  ▼
[5] Context File Loading (3 SOUL injection points)
  │  internal/store/pg/agents_context.go
  │
  │  ★ Injection Point 1: agent_context_files
  │  │  → SOUL.md (role identity, capabilities, constraints)
  │  │  → IDENTITY.md (name, emoji, personality)
  │  │  → AGENTS.md (workspace governance rules)
  │
  │  ★ Injection Point 2: user_context_files
  │  │  → USER.md (per-user overrides, preferences)
  │  │  → BOOTSTRAP.md (onboarding context)
  │
  ▼
[6] System Prompt Building
  │  internal/agent/systemprompt.go
  │  → BuildSystemPrompt() assembles 15 sections:
  │    [BasePrompt] + [SOUL.md] + [IDENTITY.md] + [AGENTS.md]
  │    + [USER.md] + [TOOLS.md] + [ExtraPrompt] + [ContextFiles]
  │    + [session state, history, metadata...]
  │
  │  ★ Injection Point 3: ExtraPrompt
  │  │  → Governance rail context injected here:
  │  │    - /spec skill instructions (Rail #1)
  │  │    - PR review checklist (Rail #2)
  │  │    - RAG search results (Rail #3)
  │
  ▼
[7] AI-Platform Call
  │  → POST https://api.nhatquangholding.com/v1/chat/completions
  │  → Headers: X-API-Key: aip_..., X-Tenant-ID: mts
  │  → Body: { model: "qwen3:14b", messages: [system_prompt, user_message] }
  │  → Response: AI-generated text
  │  → Fallback: retry once (1s backoff), then graceful degradation message
  │
  ▼
[8] Response Delivery
  │  → Format response for channel (Telegram markdown)
  │  → Send via Telegram Bot API
  │  → Update session (append to messages JSONB)
  │
  ▼
[9] Trace Logging
  │  internal/tracing/collector.go
  │  → Create trace record: {trace_id, agent_id, input_tokens, output_tokens,
  │     total_cost, duration_ms, status}
  │  → Create span records for each LLM call
  │  → Structured log (slog): {trace_id, tenant_id, soul_role, duration_ms}
  │
  ▼
[10] COMMIT — SET LOCAL automatically reset
```

### 3.2 /spec Command Flow (Rail #1 — Sprint 4)

```
User: /spec Create login feature for Bflow mobile app
  │
  ▼
[1] Command Detection
  │  → Telegram handler detects /spec prefix
  │  → Metadata: {command: "spec", delegateTo: "pm"}
  │
  ▼
[2] SOUL Delegation
  │  → Current SOUL delegates to PM SOUL (via agent_links)
  │  → PM SOUL loads /spec skill instructions (skills table)
  │
  ▼
[3] Skill Execution
  │  → PM SOUL + skill instructions + user input → system prompt
  │  → ExtraPrompt injected with skill template:
  │    "Generate a structured specification in JSON format..."
  │
  ▼
[4] AI-Platform Response
  │  → Returns structured JSON:
  │    { title, narrative: {as_a, i_want, so_that},
  │      acceptance_criteria: [{given, when, then}],
  │      priority, effort_estimate }
  │
  ▼
[5] Evidence Capture
  │  → Store in governance_specs table (Sprint 7)
  │  → Link to trace_id for full audit trail
  │
  ▼
[6] Response to User
  │  → Format spec as readable Telegram message
  │  → Include trace_id for reference
```

---

## 4. Deployment Diagram

### 4.1 Phase 1 — MTS (Single VPS)

```
┌──────────────────────────────────────────────────────────────────────────┐
│  VPS: 4 vCPU, 8GB RAM, 100GB SSD                                        │
│  OS: Ubuntu 22.04 LTS                                                    │
│                                                                          │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │                    Docker Compose Stack                           │   │
│  │                                                                   │   │
│  │  ┌─────────────────────┐     ┌──────────────────────────────┐   │   │
│  │  │  mtclaw             │     │  postgres                     │   │   │
│  │  │  (Go binary ~25MB)  │────▶│  PostgreSQL 15 + pgvector     │   │   │
│  │  │  Port: 8080         │     │  Port: 5432                   │   │   │
│  │  │  RAM: <35MB         │     │  Data: /var/lib/postgresql     │   │   │
│  │  │                     │     │  pgvector: 0.5+                │   │   │
│  │  │  Env:               │     │                                │   │   │
│  │  │  - POSTGRES_DSN     │     │  Roles:                        │   │   │
│  │  │  - ENCRYPTION_KEY   │     │  - mtclaw_admin (bypass RLS)   │   │   │
│  │  │  - BFLOW_AI_API_KEY │     │  - mtclaw_app (enforced RLS)   │   │   │
│  │  │  - BFLOW_TENANT_ID  │     │                                │   │   │
│  │  └─────────┬───────────┘     └──────────────────────────────┘   │   │
│  │            │                                                      │   │
│  │  ┌────────┴────────────┐     ┌──────────────────────────────┐   │   │
│  │  │  prometheus         │     │  grafana (optional)           │   │   │
│  │  │  Port: 9090         │     │  Port: 3000                   │   │   │
│  │  │  Scrapes /metrics   │     │  Dashboards: iframe only      │   │   │
│  │  │  from mtclaw:8080   │     │  (AGPL containment)           │   │   │
│  │  └─────────────────────┘     └──────────────────────────────┘   │   │
│  │                                                                   │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                                                          │
└───────────────┬──────────────────────────────────┬───────────────────────┘
                │                                   │
     ┌──────────▼──────────┐            ┌──────────▼──────────────────┐
     │  Telegram Bot API    │            │  Bflow AI-Platform           │
     │  api.telegram.org    │            │  api.nhatquangholding.com     │
     │  (webhook or poll)   │            │  Port: 443 (HTTPS)           │
     └─────────────────────┘            │                              │
                                         │  RTX 5090 32GB               │
                                         │  qwen3:14b                   │
                                         │  RAG + Translation           │
                                         └──────────────────────────────┘
```

### 4.2 Docker Compose Configuration

```yaml
# docker-compose.yml (Phase 1)
services:
  mtclaw:
    build: .
    ports: ["8080:8080"]
    environment:
      - GOCLAW_POSTGRES_DSN=postgres://mtclaw_app:${DB_PASS}@postgres:5432/mtclaw?sslmode=disable
      - GOCLAW_ENCRYPTION_KEY=${ENCRYPTION_KEY}
      - BFLOW_AI_API_KEY=${BFLOW_AI_API_KEY}
      - BFLOW_AI_BASE_URL=https://api.nhatquangholding.com
      - BFLOW_TENANT_ID=mts
    depends_on: [postgres]
    restart: unless-stopped
    mem_limit: 256m

  postgres:
    image: pgvector/pgvector:pg15
    environment:
      - POSTGRES_DB=mtclaw
      - POSTGRES_USER=mtclaw_admin
      - POSTGRES_PASSWORD=${DB_PASS}
    volumes: ["pgdata:/var/lib/postgresql/data"]
    ports: ["5432:5432"]
    restart: unless-stopped

  prometheus:
    image: prom/prometheus:v2.48.0
    volumes: ["./prometheus.yml:/etc/prometheus/prometheus.yml"]
    ports: ["9090:9090"]
    restart: unless-stopped

volumes:
  pgdata:
```

### 4.3 Cost Estimate

| Component | Monthly | Annual |
|-----------|---------|--------|
| VPS (4 vCPU, 8GB) | $70-140 | $840-1,680 |
| Bflow AI-Platform | $0 (internal) | $0 |
| Domain/SSL | ~$1 | ~$12 |
| **Total** | **$71-141** | **$852-1,692** |

---

## 5. Security Architecture

### 5.1 Defense-in-Depth Model

```
┌─────────────────────────────────────────────────────────────────┐
│  LAYER 1: Network                                                │
│  → HTTPS/TLS 1.3 for all external communication                 │
│  → VPS firewall: only 8080 (HTTP), 443 (HTTPS reverse proxy)    │
│  → PostgreSQL: bind to localhost only (Docker internal network)  │
└─────────────────────────────────────────────┬───────────────────┘
                                               │
┌──────────────────────────────────────────────▼──────────────────┐
│  LAYER 2: Authentication                                         │
│  → JWT tokens (issued by MTClaw, 15-min expiry)                  │
│  → Telegram user verification (chat_id + user_id)                │
│  → API key auth for Bflow AI-Platform (X-API-Key header)         │
│  → No anonymous access — every request authenticated             │
└─────────────────────────────────────────────┬───────────────────┘
                                               │
┌──────────────────────────────────────────────▼──────────────────┐
│  LAYER 3: Application — RLS Middleware                            │
│  → Extract tenant_id from authenticated context                  │
│  → SET LOCAL app.tenant_id = '{tenant_id}'                       │
│  → EVERY database query auto-scoped to tenant                    │
│  → Even application bugs cannot leak cross-tenant data           │
└─────────────────────────────────────────────┬───────────────────┘
                                               │
┌──────────────────────────────────────────────▼──────────────────┐
│  LAYER 4: Database — PostgreSQL RLS                              │
│                                                                   │
│  8 tables with RLS enforced:                                      │
│                                                                   │
│  Direct owner_id:                                                 │
│  ┌──────────────────────────────────────────────────────────┐    │
│  │ agents:     USING (owner_id = current_setting            │    │
│  │               ('app.tenant_id', true))                    │    │
│  └──────────────────────────────────────────────────────────┘    │
│                                                                   │
│  Via FK to agents:                                                │
│  ┌──────────────────────────────────────────────────────────┐    │
│  │ agent_context_files, sessions, memory_documents,          │    │
│  │ memory_chunks, traces, spans, user_context_files:         │    │
│  │                                                            │    │
│  │   USING (agent_id IN (SELECT id FROM agents               │    │
│  │     WHERE owner_id = current_setting('app.tenant_id',     │    │
│  │       true)))                                              │    │
│  └──────────────────────────────────────────────────────────┘    │
│                                                                   │
│  Roles:                                                           │
│  ┌──────────────────────────────────────────────────────────┐    │
│  │ mtclaw_admin: BYPASSRLS — migrations, admin operations    │    │
│  │ mtclaw_app:   NOBYPASSRLS — application runtime (RLS ON)  │    │
│  └──────────────────────────────────────────────────────────┘    │
│                                                                   │
│  Note: spans has DIRECT agent_id column (verified in schema)      │
│  — no double-subquery via traces needed (CTO ISSUE-C resolved)    │
│                                                                   │
│  Global tables (no RLS): llm_providers, config_secrets,           │
│    builtin_tools, mcp_servers, embedding_cache                    │
└─────────────────────────────────────────────┬───────────────────┘
                                               │
┌──────────────────────────────────────────────▼──────────────────┐
│  LAYER 5: Encryption                                             │
│  → config_secrets: AES-256-GCM (Go crypto/aes + crypto/cipher)  │
│  → Encryption key from environment (GOCLAW_ENCRYPTION_KEY)       │
│  → At-rest encryption for API keys, tokens, sensitive config     │
│  → In-transit: TLS 1.3 for all external connections              │
└─────────────────────────────────────────────────────────────────┘
```

### 5.2 RLS Middleware Position in Request Lifecycle

```
Telegram message arrives
  → [Auth] Verify user identity (Telegram user_id)
  → [Session] Resolve session → agent → owner_id
  → ★ [RLS] SET LOCAL app.tenant_id = owner_id  ← CRITICAL: before ANY DB read
  → [Agent] Load agent config (RLS-filtered)
  → [Context] Load context files (RLS-filtered)
  → [AI] Call Bflow AI-Platform
  → [Trace] Write trace record (RLS-filtered INSERT)
  → [COMMIT] Transaction ends → SET LOCAL auto-resets
```

**CTO ISSUE-1 note**: When `SystemPromptMode=minimal` is used in agent spawning (delegation), the parent SOUL.md may be stripped. The system must ensure delegated sub-agents always receive their own SOUL.md context, not inherit a stripped parent prompt. Implementation: always reload full SOUL context for the target agent in delegation.

### 5.3 Threat Model

| Threat | Mitigation | Layer |
|--------|-----------|-------|
| Cross-tenant data leak | RLS policies + SET LOCAL middleware | 3+4 |
| SQL injection | Parameterized queries (lib/pq) | 3 |
| Unauthorized API access | JWT + Telegram verification | 2 |
| AI prompt injection | SOUL constraints + input sanitization | 3 |
| Secret exposure | AES-256-GCM encryption + env vars (not DB) | 5 |
| DDoS | VPS firewall + Telegram rate limits | 1 |
| Token cost abuse | Per-tenant cost guardrails (monthly/daily limits) | 3 |

---

## 6. Observability Architecture

### 6.1 Three Pillars

```
┌───────────────┐     ┌───────────────┐     ┌───────────────┐
│   LOGGING      │     │   TRACING      │     │   METRICS      │
│   (slog)       │     │   (traces/     │     │   (OTEL →      │
│                │     │    spans)       │     │    Prometheus)  │
│  Structured    │     │                │     │                │
│  JSON to       │     │  trace_id:     │     │  Counters:     │
│  stdout        │     │  {tenant}-     │     │  request_total │
│                │     │  {session}-    │     │  token_usage   │
│  Fields:       │     │  {ulid}        │     │                │
│  - trace_id    │     │                │     │  Histograms:   │
│  - tenant_id   │     │  Per-request:  │     │  duration_sec  │
│  - soul_role   │     │  - agent_id    │     │                │
│  - level       │     │  - tokens_in   │     │  Gauges:       │
│  - msg         │     │  - tokens_out  │     │  active_sess   │
│  - duration_ms │     │  - cost        │     │                │
│  - timestamp   │     │  - status      │     │  Labels:       │
│                │     │  - duration_ms │     │  tenant, soul, │
│                │     │                │     │  channel       │
└───────────────┘     └───────────────┘     └───────────────┘
```

### 6.2 Trace ID Format

```
{tenant_id}-{session_id}-{ulid}

Example: mts-abc123-01HQXYZ9876543210
         ─── ─────── ──────────────────
          │     │              │
          │     │              └─ ULID (time-sortable, unique)
          │     └─ Session identifier (first 6 chars)
          └─ Tenant identifier
```

### 6.3 Metrics Endpoints

| Metric | Type | Labels | Exposed at |
|--------|------|--------|-----------|
| `mtclaw_request_total` | Counter | tenant, soul, channel | `/metrics` |
| `mtclaw_request_duration_seconds` | Histogram | soul | `/metrics` |
| `mtclaw_token_usage_total` | Counter | tenant, soul | `/metrics` |
| `mtclaw_active_sessions` | Gauge | tenant | `/metrics` |

### 6.4 Tenant Cost Guardrails

| Guardrail | Default | Warn | Hard Limit |
|-----------|---------|------|-----------|
| Monthly token limit | 1,000,000 tokens | 80% (800K) | 100% (reject) |
| Daily request limit | 5,000 requests | 80% (4K) | 100% (reject) |

---

## 7. Integration Points

### 7.1 Bflow AI-Platform (ADR-005)

| Endpoint | Method | Purpose | Auth |
|----------|--------|---------|------|
| `/v1/chat/completions` | POST | Chat inference (qwen3:14b) | X-API-Key + X-Tenant-ID |
| `/api/v1/rag/query` | POST | RAG search (hybrid vector + BM25) | X-API-Key + X-Tenant-ID |
| `/v1/translations` | POST | Vietnamese ↔ English | X-API-Key + X-Tenant-ID |

**Environment variables**:
```bash
BFLOW_AI_API_KEY=aip_...        # API key (prefix aip_)
BFLOW_AI_BASE_URL=https://api.nhatquangholding.com
BFLOW_TENANT_ID=mts             # Tenant identifier
```

**Fallback strategy**: Retry once with 1s backoff → return user-friendly degradation message. No fallback to direct Ollama or external API.

### 7.2 Telegram Bot API

| Mode | Configuration | Use Case |
|------|--------------|----------|
| Polling (Phase 1) | `TELEGRAM_BOT_TOKEN` | Dev/staging, no public endpoint needed |
| Webhook (Production) | `TELEGRAM_WEBHOOK_URL` | Production, lower latency |

**Channel abstraction**: `internal/channels/telegram/` implements unified `InboundMessage`/`OutboundMessage` interface. Future Zalo channel follows same pattern via `internal/channels/zalo/`.

### 7.3 GoClaw Internal Components

| Component | Source File | Integration |
|-----------|-----------|-------------|
| Agent Loop | `internal/agent/agent.go` | Core orchestration — session → agent → prompt → LLM → response |
| System Prompt | `internal/agent/systemprompt.go` | BuildSystemPrompt() — 15-section builder |
| Bootstrap | `internal/bootstrap/load_store.go` | LoadFromStore() — DB-based agent loading |
| Store (agents) | `internal/store/pg/agents.go` | Agent CRUD with owner_id filtering |
| Store (context) | `internal/store/pg/agents_context.go` | agent_context_files / user_context_files CRUD |
| Tracing | `internal/tracing/collector.go` | Trace/span collection and persistence |
| HTTP Routes | `internal/http/agents.go` | 55 inherited REST endpoints |
| Migrations | `migrations/000001-000007_*.sql` | Schema evolution (7 existing migrations) |

### 7.4 Governance Skills Integration (Sprint 4+)

| Skill | Rail | SOUL | Trigger | Output |
|-------|------|------|---------|--------|
| spec-factory | Rail #1 | PM | `/spec` command | Structured JSON spec |
| pr-review | Rail #2 | Reviewer | `/review` command | Verdict + findings |
| rag-query | Rail #3 | Per-SOUL | Automatic context | RAG search results |

Skills are stored in the `skills` table and linked to SOULs via `skill_agent_grants`. Skill content is injected into `ExtraPrompt` section during system prompt building.

---

## 8. Context Drift & Semantic Blindness Prevention

### 8.1 Problem Statement

Two critical failure modes affect AI agents in multi-SOUL governance platforms:

| Problem | Definition | Impact on MTClaw |
|---------|-----------|-----------------|
| **Context Drift** (Trôi ngữ cảnh) | AI forgets session goals, prior decisions, or SOUL identity after extended conversations (50-100K tokens) | SOUL breaks character, governance rails bypassed, inconsistent responses |
| **Semantic Blindness** (Mù ngữ nghĩa) | AI has no codebase/knowledge awareness beyond what's manually provided in context | `/spec` generates specs without knowledge of existing code, `/review` misses architectural patterns |

**Reference**: EndiorBot solved these in Sprints 63-65 with battle-tested patterns (TS-007, ADR-009, ADR-015). MTClaw adapts these for Go/GoClaw.

### 8.2 Architecture: 3-Layer Prevention

```
┌──────────────────────────────────────────────────────────────────┐
│  LAYER A: Context Anchoring (prevents Context Drift)             │
│                                                                   │
│  ┌─────────────────┐  ┌─────────────────┐  ┌────────────────┐  │
│  │ SOUL Identity    │  │ Session Goals    │  │ Decision Log   │  │
│  │ Anchor           │  │ Anchor           │  │ Anchor         │  │
│  │                   │  │                   │  │                │  │
│  │ Always inject     │  │ Per-session       │  │ Key decisions  │  │
│  │ SOUL.md +         │  │ objectives auto-  │  │ from earlier   │  │
│  │ IDENTITY.md at    │  │ injected in       │  │ in conversation│  │
│  │ prompt start      │  │ every turn        │  │ re-injected    │  │
│  └─────────────────┘  └─────────────────┘  └────────────────┘  │
│                                                                   │
│  Token budget: ~400-800 tokens per turn (configurable)            │
│  Strategy: compact (default) → minimal (when context window full) │
├──────────────────────────────────────────────────────────────────┤
│  LAYER B: Retrieval Intelligence (prevents Semantic Blindness)   │
│                                                                   │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │  SOUL-Aware Retrieval Pipeline                               │ │
│  │                                                               │ │
│  │  User query                                                   │ │
│  │    ↓                                                          │ │
│  │  [1] RAG search (Bflow AI-Platform /v1/rag/query)             │ │
│  │    → Collection filter by SOUL role:                          │ │
│  │      dev → "engineering"                                      │ │
│  │      sales → "sales"                                          │ │
│  │      cs → "engineering" + "sales"                             │ │
│  │    ↓                                                          │ │
│  │  [2] Role-aware ranking                                       │ │
│  │    → @coder: boost src/**/*.go, tests/**/*                    │ │
│  │    → @architect: boost docs/**/*.md, ADR-*                    │ │
│  │    → @pm: boost requirements, user stories                    │ │
│  │    ↓                                                          │ │
│  │  [3] Token budget enforcement                                 │ │
│  │    → Hard cap: 2,500 tokens per retrieval                     │ │
│  │    → Injected into ContextFiles section of system prompt      │ │
│  └─────────────────────────────────────────────────────────────┘ │
├──────────────────────────────────────────────────────────────────┤
│  LAYER C: Evidence & Explainability (audit trail)                │
│                                                                   │
│  Every retrieval produces:                                        │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │  RetrievalEvidence {                                         │ │
│  │    trace_id, query, provider, elapsed_ms,                    │ │
│  │    total_hits, top_k_returned, tokens_used,                  │ │
│  │    soul_role, tenant_id,                                     │ │
│  │    results: [{path, ranking_reason, excerpt}]                │ │
│  │  }                                                           │ │
│  │                                                               │ │
│  │  ranking_reason enum:                                         │ │
│  │    EXACT_MATCH | SEMANTIC_MATCH | ROLE_BOOST |               │ │
│  │    COLLECTION_MATCH | RECENCY_BOOST                          │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  Stored in: traces.metadata (JSONB) — queryable for diagnostics  │
└──────────────────────────────────────────────────────────────────┘
```

### 8.3 SOUL Identity Anchoring (Layer A)

MTClaw's `BuildSystemPrompt()` already injects SOUL.md as section [2] of the 15-section prompt. The anti-drift extension adds:

| Anchor | Injection Point | Token Budget | Purpose |
|--------|----------------|-------------|---------|
| SOUL Identity | Section [2-4]: SOUL.md + IDENTITY.md + AGENTS.md | ~200 tokens | Prevent role confusion |
| Session Goal | Section [7]: ExtraPrompt | ~100 tokens | Prevent objective drift |
| Decision Log | Section [7]: ExtraPrompt | ~200 tokens | Prevent contradictory decisions |

**GoClaw implementation**: These are already supported by the existing `agent_context_files` + `ExtraPrompt` architecture. No new infrastructure needed — only content strategy for what goes into each injection point.

### 8.4 SOUL-Aware RAG Routing (Layer B)

| SOUL Role | RAG Collection | Priority Content | Sprint |
|-----------|---------------|-------------------|--------|
| dev | engineering | Bflow API docs, coding standards, architecture | 6 |
| sales | sales | Pricing, proposals, case studies | 6 |
| cs | engineering + sales | SOPs, ticket templates, product specs | 6 |
| assistant | engineering + sales (broad) | HR Q&A, meeting templates, general tasks | 6 |
| coder | engineering | Source code patterns, test examples | 6 |
| architect | engineering | ADRs, system design docs | 6 |
| pm | engineering + sales | Requirements, user stories, market data | 7 |

**Phase 2 (NQH)**: NQH-SOPs collection (805 docs already indexed on AI-Platform) routes to NQH-specific SOULs.

### 8.5 Phased Delivery

| Sprint | Capability | Anti-Drift Mechanism |
|--------|-----------|---------------------|
| 3 | SOUL seeding (48 context files) | Identity anchoring via SOUL.md + IDENTITY.md |
| 4 | `/spec` + SOUL delegation | Session goal injection via ExtraPrompt |
| 6 | RAG collections (3 MTS domains) | SOUL-aware retrieval with collection routing |
| 7 | Decision log anchoring | Re-inject key decisions in long conversations |

---

## 9. Key Design Decisions Summary

| # | Decision | ADR | Rationale |
|---|----------|-----|-----------|
| 1 | GoClaw as runtime | ADR-001 | Single binary, PostgreSQL-native, multi-tenant, MIT license |
| 2 | Zero runtime coupling | ADR-002 | EndiorBot port logic (not CLI), SDLC-Orchestrator patterns only |
| 3 | Tenant-aware observability | ADR-003 | Every trace/log/metric tagged with tenant_id + soul_role |
| 4 | Git-sourced SOULs | ADR-004 | Version-controlled, drift detection, YAML+MD format |
| 5 | Bflow AI-Platform single provider | ADR-005 | Centralized cost, single audit, enterprise auth |
| 6 | Context Drift prevention | EndiorBot TS-007, ADR-015 | 3-layer anchoring: identity + retrieval + evidence |

---

## 10. Sprint 3 Implementation Sequence

| Day | Task | Layer Affected |
|-----|------|---------------|
| 1 | System Architecture Document (this doc) | Design |
| 1-2 | RLS migration (8 tables, 2 roles, middleware) | Layer 2 + Layer 5 |
| 2-3 | SOUL seeding migration (16 agents, 48 context files) | Layer 3 + Layer 5 |
| 3-4 | Observability (slog enhancement, OTEL metrics) | Layer 2 + Layer 3 |
| 4-5 | Bflow AI-Platform provider setup | Layer 4 |
| 5 | Integration test + manual smoke test | All layers |

---

## References

- [ADR-001: GoClaw Adoption](01-ADRs/SPEC-0001-ADR-001-GoClaw-Adoption.md)
- [ADR-002: Three-System Architecture](01-ADRs/SPEC-0002-ADR-002-Three-System-Architecture.md)
- [ADR-003: Observability Architecture](01-ADRs/SPEC-0003-ADR-003-Observability-Architecture.md)
- [ADR-004: SOUL Implementation](01-ADRs/SPEC-0004-ADR-004-SOUL-Implementation.md)
- [ADR-005: Bflow AI-Platform Integration](01-ADRs/SPEC-0005-ADR-005-Bflow-AI-Platform-Integration.md)
- [RLS Tenant Isolation Design](rls-tenant-isolation-design.md)
- [SOUL Loading Implementation Plan](soul-loading-implementation-plan.md)
- [GoClaw Schema Analysis](goclaw-schema-analysis.md)
- [/spec Command Design](spec-command-design.md)
- [Requirements](../01-planning/requirements.md)
- [Data Model](../01-planning/data-model.md)
- [Technology Stack](../01-planning/technology-stack.md)
- [Product Vision](../00-foundation/product-vision.md)
