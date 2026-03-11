# System Architecture Document — MTClaw

**SDLC Stage**: 02-Design
**Version**: 2.0.0
**Date**: 2026-03-08
**Author**: [@architect]
**Reviewer**: [@cto]
**Framework**: SDLC 6.1.1 — Stage 02 Required Artifact (STANDARD tier)
**Implements**: US-014 (System Architecture Document)

**Changelog**:
- v2.0.0 (2026-03-08): Sprint 26 realignment — add Claude Code Bridge (ADR-010/011), Provider Fallback Chain (ADR-014), Multi-Provider Adapters (ADR-013), Agent Teams (ADR-012), MS Teams channel (ADR-007), MCP server bridge, delegation system, cost guardrails. Update observability to reflect actual state (PG traces, no Prometheus yet). Fix stale ports and counts.
- v1.0.0 (2026-03-02): Initial architecture covering Sprints 1-12.

---

## 1. Executive Summary

MTClaw is a governance-first company assistant platform built on the GoClaw runtime (Go 1.25). It provides 17 role-aware AI personas (SOULs) + 3 governance rails (Spec Factory, PR Gate, Knowledge & Answering) for MTS employees via Telegram, Zalo, and MS Teams, with multi-tenant isolation for NQH expansion.

Since v1.0.0, the system has expanded with:
- **Claude Code Terminal Bridge** (Sprints 13-17, ADR-010/011): tmux-based bridge enabling Claude Code sessions from Telegram with SOUL-aware persona injection, 3-axis capability model, and permission hooks.
- **Provider Fallback Chain** (Sprints 24-25, ADR-014): Automatic failover from Bflow AI-Platform to Claude CLI on retryable errors (429, 500-599).
- **Agent Teams Coordination** (Sprint 22, ADR-012): DB-backed `team_tasks` table for multi-agent task assignment (Option B — MTClaw-coordinated, not Claude's experimental Agent Teams API).
- **Multi-Provider Adapter Pattern** (Sprint 23, ADR-013): Provider abstraction enabling future adapters for Cursor, Codex CLI, Gemini CLI.

This document defines the system architecture across 7 views: Component, Data Flow, Bridge Architecture, Deployment, Security, Observability, and Integration Points.

---

## 2. Component Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          LAYER 1: USER CHANNELS                         │
│                                                                         │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────────────────┐   │
│  │ Telegram Bot  │   │   Zalo Bot   │   │  MS Teams Bot            │   │
│  │ (Primary)     │   │  (Phase 2)   │   │  (ADR-007, Sprint 10)    │   │
│  └──────┬───────┘   └──────┬───────┘   └────────────┬─────────────┘   │
│         │                   │                         │                 │
│         └───────────────────┼─────────────────────────┘                 │
│                             │                                           │
│              BaseChannel.HandleMessage() → bus.InboundMessage           │
│              (unified channel abstraction — canonical message type)     │
│                                                                         │
│  Removed in ADR-006: Discord, Feishu, WhatsApp                         │
└─────────────────────────────┬───────────────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────────────┐
│                       LAYER 2: MESSAGE BUS + GATEWAY                    │
│                                                                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │              MessageBus (bus/bus.go) — SINGLE SOURCE OF TRUTH    │   │
│  │  PublishInbound() → ConsumeInbound() → PublishOutbound()        │   │
│  │  Deduplication (20min TTL, 5000 max), debouncing, WebSocket     │   │
│  └──────────────────────────┬──────────────────────────────────────┘   │
│                              │                                          │
│  ┌──────────────────────────▼──────────────────────────────────────┐   │
│  │              HTTP + WebSocket Server (net/http)                  │   │
│  │  Port 18790 — REST endpoints + WebSocket RPC v3                 │   │
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
│  │              Consumer Loop (gateway_consumer.go)                 │   │
│  │  Single consumer: dedup → @mention SOUL routing → session key   │   │
│  │  → Scheduler (concurrency control) → Agent Loop                 │   │
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
│  │  → 17 SOULs from agents table (13 SDLC + 3 business + 1 ops)   │   │
│  └──────────────────────────┬──────────────────────────────────────┘   │
│                              │                                          │
│  ┌──────────────────────────▼──────────────────────────────────────┐   │
│  │  Context File Loader (store/pg/agents_context.go)               │   │
│  │  → SOUL.md, IDENTITY.md, AGENTS.md from agent_context_files     │   │
│  │  → USER.md, BOOTSTRAP.md from user_context_files                │   │
│  └──────────────────────────┬──────────────────────────────────────┘   │
│                              │                                          │
│  ┌──────────────────────────▼──────────────────────────────────────┐   │
│  │  System Prompt Builder (agent/systemprompt.go)                  │   │
│  │  → BuildSystemPrompt() — 15 sections including:                 │   │
│  │    [1] BasePrompt  [2] SOUL.md  [3] IDENTITY.md  [4] AGENTS.md │   │
│  │    [5] USER.md  [6] TOOLS.md  [7] ExtraPrompt (rail injection) │   │
│  │    [8] ContextFiles (RAG injection)  [9-15] Session state       │   │
│  └──────────────────────────┬──────────────────────────────────────┘   │
│                              │                                          │
│  ┌──────────────────────────▼──────────────────────────────────────┐   │
│  │  Think → Act → Observe Loop (agent/loop.go)                     │   │
│  │  → LLM call → tool_calls execution → loop until "stop"         │   │
│  │  → Fallback: if primary fails (429/500+), try fallback provider │   │
│  │    Guard: no fallback at iteration 1 with tools (CTO-R2-1)     │   │
│  │    Always strip tools on fallback (CTO-501)                     │   │
│  └──────────────────────────┬──────────────────────────────────────┘   │
│                              │                                          │
│  ┌──────────────────────────▼──────────────────────────────────────┐   │
│  │  Skills System                                                   │   │
│  │  → /spec (Rail #1 — Spec Factory)                               │   │
│  │  → /review (Rail #2 — PR Gate)                                  │   │
│  │  → RAG query (Rail #3 — Knowledge & Answering)                  │   │
│  └──────────────────────────┬──────────────────────────────────────┘   │
│                              │                                          │
│  ┌──────────────────────────▼──────────────────────────────────────┐   │
│  │  Delegation & Teams (agent_links + team_tasks)                  │   │
│  │  → SOUL-to-SOUL delegation (assistant → pm, pm → coder)         │   │
│  │  → DelegateManager (tools/delegate.go, 502 LOC)                │   │
│  │  → Team tasks: create, claim, complete, search (ADR-012 Opt B) │   │
│  └──────────────────────────┬──────────────────────────────────────┘   │
│                              │                                          │
│  ┌──────────────────────────▼──────────────────────────────────────┐   │
│  │  70+ Agent Tools                                                │   │
│  │  → Filesystem, exec (sandboxed), memory (vector), web, browser │   │
│  │  → TTS, subagent spawn, handoff, MCP bridge                    │   │
│  │  → Policy engine filters tools per agent                        │   │
│  └──────────────────────────┬──────────────────────────────────────┘   │
│                              │                                          │
└──────────────────────────────┼──────────────────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────────────────┐
│              LAYER 3.5: CLAUDE CODE BRIDGE (ADR-010/011)                │
│              (internal/claudecode/ — 9,096 LOC, 49 files)               │
│                                                                         │
│  ┌───────────────────┐  ┌──────────────────────────────────────────┐  │
│  │  SessionManager    │  │  HookServer (127.0.0.1:18792)            │  │
│  │                    │  │                                           │  │
│  │  - CreateSession() │  │  POST /hook/permission — permission req  │  │
│  │  - KillSession()   │  │  POST /hook/notification — event stream │  │
│  │  - ListSessions()  │  │  HMAC-SHA256 auth per session            │  │
│  │  - TransitionState │  │                                           │  │
│  │  - UpdateRiskMode  │  │  Async permission polling + fail-safe    │  │
│  │  - HealthMonitor   │  │  fallback (30s timeout)                  │  │
│  └───────┬───────────┘  └──────────────────────────────────────────┘  │
│          │                                                              │
│  ┌───────▼───────────┐  ┌──────────────────────────────────────────┐  │
│  │  TmuxBridge        │  │  SOUL-Aware Launch (ADR-011)              │  │
│  │                    │  │                                           │  │
│  │  - LaunchSession() │  │  Strategy A: .claude/agents/{role}.md    │  │
│  │  - KillSession()   │  │  Strategy B: --append-system-prompt-file │  │
│  │  - CaptureOutput() │  │  Strategy C: bare launch (fallback)      │  │
│  │  - tmux new-session│  │                                           │  │
│  │  - 3-axis model:   │  │  claudemd_generator.go: auto-gen         │  │
│  │    InputMode ×     │  │  CLAUDE.md with SOUL + skills +          │  │
│  │    ToolPolicy ×    │  │  constraints                              │  │
│  │    OutputPolicy    │  │                                           │  │
│  └────────────────────┘  └──────────────────────────────────────────┘  │
│                                                                         │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │  Bridge Audit (bridge_audit.go)                                  │  │
│  │  → Dual-write: JSONL file + bridge_audit_events PG table         │  │
│  │  → Events: session.created, session.killed, permission.decided  │  │
│  └──────────────────────────────────────────────────────────────────┘  │
│                                                                         │
│  Telegram /cc commands: launch, sessions, kill, capture, register,     │
│  switch, risk, send, info, context (currently Telegram-only)           │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────────────────┐
│              LAYER 4: AI PROVIDERS + FALLBACK CHAIN (ADR-014)           │
│                                                                         │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │  Provider Chain: MTCLAW_PROVIDER_CHAIN env var                   │  │
│  │  Default: "bflow-ai-platform,claude-cli"                         │  │
│  │                                                                   │  │
│  │  ┌─────────────────────────┐  ┌─────────────────────────────┐   │  │
│  │  │  PRIMARY: Bflow          │  │  FALLBACK: Claude CLI        │   │  │
│  │  │  AI-Platform              │  │  (providers/claude_cli.go)   │   │  │
│  │  │                           │  │                               │   │  │
│  │  │  POST /v1/chat/completions│  │  Subprocess: claude --print  │   │  │
│  │  │  X-API-Key + X-Tenant-ID │  │  --output-format json         │   │  │
│  │  │  qwen3:14b (default)     │  │  Strips ANTHROPIC_API_KEY    │   │  │
│  │  │  RAG: /api/v1/rag/query  │  │  OAuth token in .claude/     │   │  │
│  │  └─────────────────────────┘  └─────────────────────────────┘   │  │
│  │                                                                   │  │
│  │  Fallback triggers: HTTP 429, 500-599 (retryable errors)         │  │
│  │  Guard: no fallback at iteration=1 with tools (CTO-R2-1)        │  │
│  │  Always strip tools on fallback (CTO-501)                        │  │
│  │  2-span tracing: primary fail span + fallback success span       │  │
│  │                                                                   │  │
│  │  Multi-Provider Adapter Pattern (ADR-013):                       │  │
│  │  → Provider interface: Chat(), ChatStream(), Name()              │  │
│  │  → Implementations: Anthropic, OpenAI-compat, DashScope,         │  │
│  │    Bflow AI-Platform, Claude CLI                                 │  │
│  │  → Future: Cursor, Codex CLI, Gemini CLI adapters               │  │
│  └──────────────────────────────────────────────────────────────────┘  │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
                               │
┌──────────────────────────────▼──────────────────────────────────────────┐
│                    LAYER 5: DATABASE                                     │
│                                                                         │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │  PostgreSQL 15 + pgvector 0.5+                                   │  │
│  │  18+ migrations (000001–000018+)                                 │  │
│  │                                                                   │  │
│  │  Core tables (RLS enforced):                                     │  │
│  │  ┌─────────────┐  ┌───────────────┐  ┌──────────────────────┐  │  │
│  │  │ agents (17)  │  │ sessions      │  │ traces + spans       │  │  │
│  │  │ + context    │  │ + messages    │  │ (observability)      │  │  │
│  │  │   files      │  │               │  │                      │  │  │
│  │  └─────────────┘  └───────────────┘  └──────────────────────┘  │  │
│  │                                                                   │  │
│  │  ┌─────────────┐  ┌───────────────┐  ┌──────────────────────┐  │  │
│  │  │ memory_docs  │  │ skills        │  │ llm_providers        │  │  │
│  │  │ + chunks     │  │ + grants      │  │ + config_secrets     │  │  │
│  │  │ (pgvector)   │  │               │  │ (global, no RLS)     │  │  │
│  │  └─────────────┘  └───────────────┘  └──────────────────────┘  │  │
│  │                                                                   │  │
│  │  Bridge tables (migration 000018, RLS enforced):                 │  │
│  │  ┌──────────────────┐  ┌─────────────────┐  ┌──────────────┐  │  │
│  │  │ bridge_sessions   │  │ bridge_projects  │  │ bridge_audit │  │  │
│  │  │ (22 cols, JSONB)  │  │ (workspace reg) │  │ _events      │  │  │
│  │  └──────────────────┘  └─────────────────┘  └──────────────┘  │  │
│  │                                                                   │  │
│  │  Team coordination tables (ADR-012 Option B):                    │  │
│  │  ┌──────────────────┐  ┌─────────────────┐                     │  │
│  │  │ agent_teams       │  │ team_tasks       │                     │  │
│  │  │ (team membership)│  │ (CRUD + search) │                     │  │
│  │  └──────────────────┘  └─────────────────┘                     │  │
│  │                                                                   │  │
│  │  RLS ENFORCED: agents, agent_context_files, sessions,            │  │
│  │    memory_documents, memory_chunks, traces, spans,               │  │
│  │    user_context_files, bridge_sessions, bridge_projects,         │  │
│  │    bridge_audit_events, agent_teams, team_tasks                  │  │
│  │                                                                   │  │
│  │  Global tables (no RLS): llm_providers, config_secrets,          │  │
│  │    builtin_tools, mcp_servers, embedding_cache                   │  │
│  └──────────────────────────────────────────────────────────────────┘  │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### Component Summary

| Layer | Component | Technology | Purpose |
|-------|-----------|-----------|---------|
| 1 | Channel Handlers | Telegram Bot API, Zalo API, MS Teams Bot Framework | User-facing messaging interfaces |
| 2 | Message Bus + Gateway | GoClaw bus + HTTP/WebSocket (net/http) | Request routing, RLS middleware, unified consumer |
| 3 | Agent Loop | GoClaw internals | SOUL loading, prompt building, skills, delegation, tools |
| 3.5 | Claude Code Bridge | tmux + HookServer + SOUL-aware launch | Terminal bridge for Claude Code sessions |
| 4 | AI Providers | Bflow AI-Platform + Claude CLI + adapter pattern | Chat inference, RAG, fallback chain |
| 5 | Database | PostgreSQL 15 + pgvector, 18+ migrations | Persistence, RLS, vector search, bridge state |

---

## 3. Data Flow Diagram

### 3.1 Full Request Lifecycle

```
User sends Telegram message: "Tao spec cho tinh nang login"
  │
  ▼
[1] Channel Handler
  │  internal/channels/telegram/handler.go
  │  → Parse message, extract user_id, chat_id
  │  → Create InboundMessage{channel: "telegram", chatID, userID, text}
  │  → bus.PublishInbound(msg) — unified message bus
  │
  ▼
[2] Consumer Loop (SINGLE consumer — no per-channel duplication)
  │  cmd/gateway_consumer.go
  │  → Deduplication check (20min TTL, 5000 max)
  │  → @mention SOUL routing (e.g., @pm → PM agent)
  │  → Session key resolution: channel + chat_id → agent routing
  │  → Scheduler concurrency control
  │
  ▼
[3] ★ RLS MIDDLEWARE ★ (tenant isolation boundary)
  │  internal/middleware/tenant.go
  │  → Extract owner_id from agent → tenant_id
  │  → BEGIN transaction
  │  → SET LOCAL app.tenant_id = 'mts'     ← BEFORE any DB query
  │  → All subsequent queries in this transaction are tenant-scoped
  │
  ▼
[4] Agent Loading + Context Files
  │  internal/bootstrap/load_store.go → agent config from DB
  │  internal/store/pg/agents_context.go → SOUL.md, IDENTITY.md, AGENTS.md
  │  internal/agent/systemprompt.go → BuildSystemPrompt() (15 sections)
  │
  ▼
[5] Agent Loop (Think → Act → Observe)
  │  internal/agent/loop.go
  │  → LLM call to primary provider (Bflow AI-Platform)
  │  → If retryable error (429, 500+): fallback to Claude CLI
  │  → Execute tool_calls → loop until "stop"
  │
  ▼
[6] Response Delivery
  │  → bus.PublishOutbound(msg)
  │  → Channel handler formats for platform (Telegram markdown, etc.)
  │  → Send via channel API
  │
  ▼
[7] Trace Logging
  │  internal/tracing/collector.go
  │  → trace record: {trace_id, agent_id, tokens, cost, duration, status}
  │  → span records for each LLM call
  │  → Structured log (slog): {trace_id, tenant_id, soul_role, duration_ms}
  │
  ▼
[8] COMMIT — SET LOCAL automatically reset
```

### 3.2 Bridge Session Flow (Layer 3.5)

```
User sends /cc launch mtclaw @coder in Telegram
  │
  ▼
[1] Telegram command handler (commands_cc.go)
  │  → Parse: project="mtclaw", role="coder"
  │  → Admission check: max sessions, CPU, memory thresholds
  │
  ▼
[2] SessionManager.CreateSession()
  │  → Generate session ID, hook secret (HMAC-SHA256)
  │  → SOUL-aware launch strategy: A → B → C
  │  → TmuxBridge.LaunchSession() — tmux new-session with claude CLI
  │  → Write to in-memory map + PG bridge_sessions (dual-write)
  │  → AuditWriter: "session.created" event
  │
  ▼
[3] Claude Code runs in tmux, calls HookServer on tool use
  │  → POST /hook/permission (HMAC-SHA256 auth)
  │  → SessionManager evaluates risk mode (read/write/admin)
  │  → Telegram inline keyboard for user approval (write/admin)
  │  → Response: allow/deny
  │
  ▼
[4] User sends /cc capture or /cc kill
  │  → capture: TmuxBridge.CaptureOutput() → format → send to Telegram
  │  → kill: TmuxBridge.KillSession() → update status → audit event
```

### 3.3 /spec Command Flow (Rail #1)

```
User: /spec Create login feature for Bflow mobile app
  │
  ▼
[1] Command Detection → SOUL Delegation (assistant → pm)
[2] PM SOUL + /spec skill instructions → system prompt
[3] AI-Platform → structured JSON spec
[4] Evidence capture → governance_specs table + trace_id
[5] Formatted response → Telegram
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
│  │  │  mtclaw              │     │  postgres                     │   │   │
│  │  │  (Go binary ~25MB)   │────▶│  PostgreSQL 15 + pgvector     │   │   │
│  │  │  Port: 18790         │     │  Port: 5432                   │   │   │
│  │  │  RAM: <35MB          │     │  18+ migrations               │   │   │
│  │  │                      │     │                                │   │   │
│  │  │  Bridge (optional):  │     │  Roles:                        │   │   │
│  │  │  HookServer: 18792   │     │  - mtclaw_admin (bypass RLS)   │   │   │
│  │  │  tmux sessions       │     │  - mtclaw_app (enforced RLS)   │   │   │
│  │  │  ENABLE_BRIDGE=true  │     │                                │   │   │
│  │  │                      │     │                                │   │   │
│  │  │  Claude CLI (opt.):  │     │                                │   │   │
│  │  │  ENABLE_CLAUDE_CLI   │     │                                │   │   │
│  │  │  OAuth: .claude/     │     │                                │   │   │
│  │  └─────────────────────┘     └──────────────────────────────┘   │   │
│  │                                                                   │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                                                          │
└───────────────┬──────────────────────────────────┬───────────────────────┘
                │                                   │
     ┌──────────▼──────────┐            ┌──────────▼──────────────────┐
     │  Telegram Bot API    │            │  Bflow AI-Platform           │
     │  api.telegram.org    │            │  api.nhatquangholding.com     │
     └─────────────────────┘            │  qwen3:14b + RAG              │
                                         └──────────────────────────────┘
```

### 4.2 Docker Compose Configuration

```yaml
# docker-compose.yml (Phase 1 — actual)
services:
  mtclaw:
    build:
      context: .
      args:
        ENABLE_BRIDGE: "${ENABLE_BRIDGE:-false}"
        ENABLE_CLAUDE_CLI: "${ENABLE_CLAUDE_CLI:-false}"
    ports: ["18790:18790"]
    environment:
      - MTCLAW_POSTGRES_DSN=postgres://mtclaw_app:${DB_PASS}@postgres:5432/mtclaw
      - MTCLAW_ENCRYPTION_KEY=${ENCRYPTION_KEY}
      - MTCLAW_BFLOW_API_KEY=${BFLOW_AI_API_KEY}
      - MTCLAW_BFLOW_BASE_URL=http://ai-platform:8120/api/v1  # via ai-net Docker network
      - BFLOW_TENANT_ID=mts
      - MTCLAW_BRIDGE_ENABLED=${MTCLAW_BRIDGE_ENABLED:-false}
      - MTCLAW_BRIDGE_HOOK_PORT=${MTCLAW_BRIDGE_HOOK_PORT:-19080}
      - MTCLAW_PROVIDER_CHAIN=${MTCLAW_PROVIDER_CHAIN:-bflow-ai-platform}
    volumes:
      - claude-oauth:/app/.claude  # Claude CLI OAuth persistence
    depends_on: [postgres]
    restart: unless-stopped

  postgres:
    image: pgvector/pgvector:pg15
    environment:
      - POSTGRES_DB=mtclaw
      - POSTGRES_USER=mtclaw_admin
      - POSTGRES_PASSWORD=${DB_PASS}
    volumes: ["pgdata:/var/lib/postgresql/data"]
    restart: unless-stopped

volumes:
  pgdata:
  claude-oauth:
```

**Note on Prometheus/Grafana**: The v1.0.0 architecture included Prometheus and Grafana in the deployment diagram. As of Sprint 25, these are **not implemented** — observability uses PG-backed traces and structured logging (slog). A proper metrics pipeline (OTel metrics) is planned for Sprint 28. The §6.3 metrics table below documents the **target state**, not current implementation.

### 4.3 Cost Estimate

| Component | Monthly | Annual |
|-----------|---------|--------|
| VPS (4 vCPU, 8GB) | $70-140 | $840-1,680 |
| Bflow AI-Platform | $0 (internal) | $0 |
| Claude CLI (fallback) | ~$0-50 (usage-based) | ~$0-600 |
| Domain/SSL | ~$1 | ~$12 |
| **Total** | **$71-191** | **$852-2,292** |

---

## 5. Security Architecture

### 5.1 Defense-in-Depth Model

```
┌─────────────────────────────────────────────────────────────────┐
│  LAYER 1: Network                                                │
│  → HTTPS/TLS 1.3 for all external communication                 │
│  → VPS firewall: only 18790 (HTTP), 443 (HTTPS reverse proxy)   │
│  → PostgreSQL: bind to localhost only (Docker internal network)  │
│  → Bridge HookServer: 127.0.0.1 only (no external access)       │
└─────────────────────────────────────────────┬───────────────────┘
                                               │
┌──────────────────────────────────────────────▼──────────────────┐
│  LAYER 2: Authentication                                         │
│  → JWT tokens (issued by MTClaw, 15-min expiry)                  │
│  → Telegram user verification (chat_id + user_id)                │
│  → API key auth for Bflow AI-Platform (X-API-Key header)         │
│  → Bridge: HMAC-SHA256 per-session hook authentication           │
│  → Claude CLI: env sanitization (strips ANTHROPIC_API_KEY)       │
│  → No anonymous access — every request authenticated             │
└─────────────────────────────────────────────┬───────────────────┘
                                               │
┌──────────────────────────────────────────────▼──────────────────┐
│  LAYER 3: Application — RLS Middleware                            │
│  → Extract tenant_id from authenticated context                  │
│  → SET LOCAL app.tenant_id = '{tenant_id}'                       │
│  → EVERY database query auto-scoped to tenant                    │
│  → Even application bugs cannot leak cross-tenant data           │
│  → Bridge sessions: owner_id RLS on bridge_sessions table        │
└─────────────────────────────────────────────┬───────────────────┘
                                               │
┌──────────────────────────────────────────────▼──────────────────┐
│  LAYER 4: Database — PostgreSQL RLS                              │
│                                                                   │
│  12+ tables with RLS enforced (up from 8 in v1.0.0):            │
│  → agents, agent_context_files, sessions, memory_documents,      │
│    memory_chunks, traces, spans, user_context_files,             │
│    bridge_sessions, bridge_projects, bridge_audit_events,        │
│    agent_teams, team_tasks                                       │
│                                                                   │
│  Roles:                                                           │
│  → mtclaw_admin: BYPASSRLS — migrations, admin operations        │
│  → mtclaw_app:   NOBYPASSRLS — application runtime (RLS ON)      │
│                                                                   │
│  Global tables (no RLS): llm_providers, config_secrets,           │
│    builtin_tools, mcp_servers, embedding_cache                    │
└─────────────────────────────────────────────┬───────────────────┘
                                               │
┌──────────────────────────────────────────────▼──────────────────┐
│  LAYER 5: Encryption                                             │
│  → config_secrets: AES-256-GCM (Go crypto/aes + crypto/cipher)  │
│  → Encryption key from environment (MTCLAW_ENCRYPTION_KEY)       │
│  → At-rest encryption for API keys, tokens, sensitive config     │
│  → In-transit: TLS 1.3 for all external connections              │
└─────────────────────────────────────────────────────────────────┘
```

### 5.2 Threat Model

| Threat | Mitigation | Layer |
|--------|-----------|-------|
| Cross-tenant data leak | RLS policies + SET LOCAL middleware | 3+4 |
| SQL injection | Parameterized queries (lib/pq) | 3 |
| Unauthorized API access | JWT + Telegram verification | 2 |
| AI prompt injection | SOUL constraints + input sanitization | 3 |
| Secret exposure | AES-256-GCM encryption + env vars (not DB) | 5 |
| Token cost abuse | Per-tenant cost guardrails (daily limits) | 3 |
| Bridge session hijack | HMAC-SHA256 per-session hook secret | 2 |
| Claude CLI env leak | filterEnv strips API keys before subprocess | 2 |
| Bridge permission bypass | 3-axis capability model + async approval flow | 3 |

---

## 6. Observability Architecture

### 6.1 Current State (Sprint 25)

```
┌───────────────┐     ┌───────────────┐     ┌───────────────┐
│   LOGGING      │     │   TRACING      │     │   METRICS      │
│   (slog)       │     │   (PG-backed)  │     │   (PLANNED)    │
│                │     │                │     │                │
│  Structured    │     │  traces +      │     │  No /metrics   │
│  JSON to       │     │  spans tables  │     │  endpoint yet  │
│  stdout        │     │                │     │                │
│                │     │  Optional:     │     │  Sprint 27:    │
│  Fields:       │     │  OTLP export   │     │  PG trace      │
│  - trace_id    │     │  via otelexport│     │  queries for   │
│  - tenant_id   │     │  package       │     │  adoption      │
│  - soul_role   │     │                │     │                │
│  - level       │     │  Per-request:  │     │  Sprint 28:    │
│  - msg         │     │  - agent_id    │     │  OTel metrics  │
│  - duration_ms │     │  - tokens_in   │     │  pipeline      │
│                │     │  - tokens_out  │     │                │
│                │     │  - cost        │     │                │
│                │     │  - status      │     │                │
│                │     │  - fallback    │     │                │
│                │     │    metadata    │     │                │
└───────────────┘     └───────────────┘     └───────────────┘
```

### 6.2 Trace ID Format

```
{tenant_id}-{session_id}-{ulid}
Example: mts-abc123-01HQXYZ9876543210
```

### 6.3 Metrics Target State (Sprint 28+)

| Metric | Type | Labels | Status |
|--------|------|--------|--------|
| `mtclaw_messages_total` | Counter | tenant, soul, channel | PLANNED (Sprint 28) |
| `mtclaw_request_duration_seconds` | Histogram | soul | PLANNED (Sprint 28) |
| `mtclaw_token_usage_total` | Counter | tenant, soul | PLANNED (Sprint 28) |
| `mtclaw_sessions_active` | Gauge | tenant | PLANNED (Sprint 28) |
| `mtclaw_bridge_commands_total` | Counter | cmd | PLANNED (Sprint 28) |
| `mtclaw_fallback_total` | Counter | status | PLANNED (Sprint 28) |

### 6.4 Tenant Cost Guardrails

| Guardrail | Implementation | Status |
|-----------|---------------|--------|
| Daily request limit | `internal/cost/guardrails.go` `CheckDailyLimit()` | IMPLEMENTED (Sprint 6) |
| Env var | `MTCLAW_TENANT_DAILY_REQUEST_LIMIT` (default: 500) | IMPLEMENTED |
| Warn threshold | 80% of daily limit (structured WARN log) | PLANNED (Sprint 27) |
| Fallback cost tracking | 2-span tracing with `fallback=true` metadata | IMPLEMENTED (Sprint 25) |

---

## 7. Integration Points

### 7.1 Bflow AI-Platform (ADR-005)

| Endpoint | Method | Purpose | Auth |
|----------|--------|---------|------|
| `/v1/chat/completions` | POST | Chat inference (qwen3:14b) | X-API-Key + X-Tenant-ID |
| `/api/v1/rag/query` | POST | RAG search (hybrid vector + BM25) | X-API-Key + X-Tenant-ID |
| `/v1/translations` | POST | Vietnamese <> English | X-API-Key + X-Tenant-ID |

**Fallback strategy** (ADR-014): If Bflow returns retryable error (429, 500-599), fallback to Claude CLI. Guard: no fallback at iteration=1 with tools. Always strip tools on fallback.

### 7.2 Claude CLI Provider (ADR-014)

| Config | Env Var | Default |
|--------|---------|---------|
| CLI path | `MTCLAW_CLAUDE_PATH` | `claude` (from PATH) |
| Model | `MTCLAW_CLAUDE_MODEL` | `claude-sonnet-4-5-20250514` |
| Timeout | `MTCLAW_CLAUDE_TIMEOUT` | `120s` |
| Provider chain | `MTCLAW_PROVIDER_CHAIN` | `bflow-ai-platform` |

### 7.3 Channel Integrations

| Channel | Mode | Config | Status |
|---------|------|--------|--------|
| Telegram | Polling / Webhook | `MTCLAW_TELEGRAM_TOKEN` | Primary (Sprint 1+) |
| Zalo | OAuth2 + Polling | `MTCLAW_ZALO_*` vars | Phase 2 (Sprint 6+) |
| MS Teams | Bot Framework + Webhook | `MTCLAW_MSTEAMS_*` vars | ADR-007 (Sprint 10+) |
| Discord | — | — | Removed (ADR-006) |
| Feishu | — | — | Removed (ADR-006) |
| WhatsApp | — | — | Removed (ADR-006) |

### 7.4 MCP Server Bridge (Sprint 10+)

| Component | File | Purpose |
|-----------|------|---------|
| MCP Manager | `internal/mcp/manager.go` | Dynamic tool registration from MCP subprocess servers |
| MCP HTTP API | `internal/http/mcp.go` | CRUD for MCP server configs + grants |
| Skill Grants | `internal/http/skills_grants.go` | Per-agent MCP tool access control |

### 7.5 GoClaw Internal Components

| Component | Source File | Purpose |
|-----------|-----------|---------|
| Agent Loop | `internal/agent/loop.go` | Think→Act→Observe cycle with fallback |
| System Prompt | `internal/agent/systemprompt.go` | BuildSystemPrompt() — 15 sections |
| Bootstrap | `internal/bootstrap/load_store.go` | DB-based agent + context loading |
| Delegation | `internal/tools/delegate.go` | DelegateManager (502 LOC) |
| Team Tasks | `internal/store/pg/teams_tasks.go` | Team task CRUD + search (ADR-012 Opt B) |
| Bridge | `internal/claudecode/session_manager.go` | Session lifecycle management |
| Tracing | `internal/tracing/collector.go` | Trace/span collection + PG persistence |
| Cost | `internal/cost/guardrails.go` | Per-tenant daily request limits |
| Migrations | `migrations/000001-000018+` | Schema evolution (18+ migrations) |

---

## 8. Context Drift & Semantic Blindness Prevention

### 8.1 Problem Statement

| Problem | Definition | Impact on MTClaw |
|---------|-----------|-----------------|
| **Context Drift** | AI forgets session goals, prior decisions, or SOUL identity after extended conversations | SOUL breaks character, governance rails bypassed |
| **Semantic Blindness** | AI has no codebase/knowledge awareness beyond what's manually provided in context | /spec generates specs without knowledge of existing code |

### 8.2 Architecture: 3-Layer Prevention

| Layer | Purpose | Implementation | Status |
|-------|---------|---------------|--------|
| **A: Context Anchoring** | Prevent Context Drift | SOUL.md + IDENTITY.md always injected (sections [2-4]). Session goals via ExtraPrompt. Decision log re-injection. | IMPLEMENTED (Sprint 3+) |
| **B: Retrieval Intelligence** | Prevent Semantic Blindness | SOUL-aware RAG routing via Bflow /v1/rag/query with collection filters per role | IMPLEMENTED (Sprint 6+) |
| **C: Evidence & Explainability** | Audit trail | RetrievalEvidence in traces.metadata (JSONB) | IMPLEMENTED (Sprint 6+) |

### 8.3 SOUL-Aware RAG Routing (Layer B)

| SOUL Role | RAG Collection | Priority Content |
|-----------|---------------|-------------------|
| dev | engineering | Bflow API docs, coding standards |
| sales | sales | Pricing, proposals, case studies |
| cs | engineering + sales | SOPs, ticket templates |
| coder | engineering | Source code patterns, tests |
| architect | engineering | ADRs, system design docs |
| pm | engineering + sales | Requirements, user stories |

---

## 9. Key Design Decisions Summary

| # | Decision | ADR | Sprint | Status |
|---|----------|-----|--------|--------|
| 1 | GoClaw as runtime | ADR-001 | 1 | IMPLEMENTED |
| 2 | Zero runtime coupling (3 systems) | ADR-002 | 1 | IMPLEMENTED |
| 3 | Tenant-aware observability | ADR-003 | 3 | IMPLEMENTED |
| 4 | Git-sourced SOULs (YAML+MD) | ADR-004 | 1 | IMPLEMENTED |
| 5 | Bflow AI-Platform primary provider | ADR-005 | 3 | IMPLEMENTED |
| 6 | Channel rationalization (remove Discord/Feishu/WhatsApp) | ADR-006 | 9 | IMPLEMENTED |
| 7 | MS Teams extension | ADR-007 | 10 | IMPLEMENTED |
| 8 | PDF library (maroto v2) | ADR-008 | — | NOT STARTED |
| 9 | Evidence linking schema | ADR-009 | — | NOT STARTED |
| 10 | Claude Code terminal bridge | ADR-010 | 13-17 | IMPLEMENTED |
| 11 | SOUL-aware bridge launch | ADR-011 | 18 | IMPLEMENTED |
| 12 | Agent Teams: NO-GO on Claude API, use team_tasks (Opt B) | ADR-012 | 22 | IMPLEMENTED (Opt B) |
| 13 | Multi-provider adapter pattern | ADR-013 | 23 | IMPLEMENTED (interface) |
| 14 | Provider fallback (Bflow → Claude CLI) | ADR-014 | 24-25 | IMPLEMENTED |

---

## References

- [ADR-001: GoClaw Adoption](01-ADRs/SPEC-0001-ADR-001-GoClaw-Adoption.md)
- [ADR-002: Three-System Architecture](01-ADRs/SPEC-0002-ADR-002-Three-System-Architecture.md)
- [ADR-003: Observability Architecture](01-ADRs/SPEC-0003-ADR-003-Observability-Architecture.md)
- [ADR-004: SOUL Implementation](01-ADRs/SPEC-0004-ADR-004-SOUL-Implementation.md)
- [ADR-005: Bflow AI-Platform Integration](01-ADRs/SPEC-0005-ADR-005-Bflow-AI-Platform-Integration.md)
- [ADR-006: Channel Rationalization](01-ADRs/SPEC-0006-ADR-006-Channel-Rationalization.md)
- [ADR-007: MSTeams Extension](01-ADRs/SPEC-0007-ADR-007-MSTeams-Extension.md)
- [ADR-008: PDF Library](01-ADRs/SPEC-0008-ADR-008-PDF-Library.md)
- [ADR-009: Evidence Linking Schema](01-ADRs/SPEC-0009-ADR-009-Evidence-Linking-Schema.md)
- [ADR-010: Claude Code Bridge](01-ADRs/SPEC-0010-ADR-010-Claude-Code-Bridge.md)
- [ADR-011: SOUL-Aware Bridge Launch](01-ADRs/SPEC-0011-ADR-011-SOUL-Aware-Bridge-Launch.md)
- [ADR-012: Agent Teams Integration](01-ADRs/SPEC-0012-ADR-012-Agent-Teams-Integration.md)
- [ADR-013: Provider Persona Projection](01-ADRs/SPEC-0013-ADR-013-Provider-Persona-Projection.md)
- [ADR-014: Provider Fallback Claude CLI](01-ADRs/SPEC-0014-ADR-014-Provider-Fallback-Claude-CLI.md)
- [RLS Tenant Isolation Design](rls-tenant-isolation-design.md)
- [SOUL Loading Implementation Plan](soul-loading-implementation-plan.md)
- [Requirements](../01-planning/requirements.md)
- [Data Model](../01-planning/data-model.md)
- [Product Vision](../00-foundation/product-vision.md)
