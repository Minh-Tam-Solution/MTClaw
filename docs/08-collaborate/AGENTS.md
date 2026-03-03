# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run Commands

```bash
make build              # Build binary → ./goclaw (CGO_ENABLED=0)
make run                # Build + run gateway
make test               # go test ./... -v -count=1
make test-coverage      # Generate coverage.out + coverage.html
make migrate-up         # Build + ./goclaw migrate up
make migrate-down       # Build + ./goclaw migrate down
make souls-validate     # Check SOUL files for YAML frontmatter + char budget (2500)
make up                 # Docker Compose up (yml + managed + selfservice)
make down               # Docker Compose down
make logs               # Follow goclaw container logs
```

Run a single test: `go test ./internal/store/ -run TestSessionStore -v -count=1`

## Architecture Overview

MTClaw is a multi-tenant AI agent gateway. It receives messages from channels (Telegram, Discord, Zalo), routes them through a message bus to SOUL-specific agents, calls LLM providers for inference, executes tools, and sends responses back.

### Message Flow

```
Channel webhook → Channel handler (parse, policy check)
  → bus.PublishInbound → Consumer (dedup, @mention SOUL routing, session key)
  → Scheduler (concurrency control) → Agent Loop (Think→Act→Observe cycle)
  → Agent calls LLM provider → executes tool_calls → loops until "stop"
  → bus.PublishOutbound → Channel send (format, media, threading)
```

### Core Packages

| Package | Purpose |
|---------|---------|
| `cmd/` | Cobra CLI commands. `gateway.go` wires everything. `gateway_consumer.go` is the inbound message orchestrator. `gateway_providers.go` registers AI providers. |
| `internal/agent/` | Agent execution loop (`loop.go`: Think→Act→Observe), system prompt building (`systemprompt.go`), agent router with TTL cache (`router.go`), output sanitization. |
| `internal/providers/` | LLM provider interface + implementations: Anthropic, OpenAI-compatible (OpenRouter, Groq, DeepSeek, Gemini, etc.), DashScope, Bflow AI-Platform. Retry with exponential backoff. |
| `internal/store/` | Data persistence. Standalone mode = file-based JSON in `~/.goclaw/`. Managed mode = PostgreSQL with RLS. Context propagation via `store.WithTenantID(ctx, id)`, `store.WithUserID(ctx, id)`. |
| `internal/channels/` | Channel interface (Start/Stop/Send) + per-platform implementations. Telegram is primary (webhook handlers, /spec commands, streaming via message edit, emoji reactions). |
| `internal/bus/` | Message bus: inbound channel (channels→consumer), outbound channel (agent→channels), deduplication (20min TTL, 5000 max), debouncing rapid messages, WebSocket event broadcasting. |
| `internal/config/` | Config loading: `config.json` (settings) + `.env` (secrets). Agents, channels, providers, tools, sessions, telemetry, cron, tailscale. |
| `internal/tools/` | 70+ agent tools: filesystem (read/write/edit/list), exec (with sandbox + approval), memory (vector search), web (search/fetch), browser, TTS, subagent spawn, handoff. Policy engine filters tools per agent. |
| `internal/skills/` | SOUL skill system. YAML+markdown format loaded from disk or DB. Per-agent allowlists. File watching for hot reload. |
| `internal/sessions/` | Session key building. Scoped by agent+channel+peer_kind+chat_id, with per-topic and per-thread isolation. |
| `internal/mcp/` | Model Context Protocol server bridge. Dynamically registers tools from MCP subprocess servers. |
| `internal/gateway/` | HTTP + WebSocket server for dashboard, streaming events, reactions. |
| `internal/sandbox/` | Docker-based sandboxing for tool execution. |
| `internal/middleware/` | HTTP middleware: tenant isolation, request logging, rate limiting. |
| `internal/hooks/` | Extensibility: command/agent evaluation hooks, policy evaluation (deny/warn/allow). |
| `internal/bootstrap/` | SOUL template seeding, context file initialization. |
| `internal/cron/` | Background task scheduler with retry logic. |
| `internal/tracing/` | OpenTelemetry integration (traces tagged with agent, rail, command, tenant). |
| `pkg/protocol/` | WebSocket RPC protocol definitions. |
| `pkg/browser/` | Browser automation (Playwright/Rod). |

### Two Operating Modes

- **Standalone**: File-based stores (`~/.goclaw/`), single-tenant, no PostgreSQL required.
- **Managed**: PostgreSQL with RLS, multi-tenant, DB-based agents/providers/channels, used in production Docker deployment.

### SOULs (AI Personas)

17 SOULs defined in `docs/08-collaborate/souls/SOUL-*.md` and seeded to PostgreSQL via migration 000009. Each SOUL has YAML frontmatter + markdown content. The agent loop injects SOUL content into the LLM system prompt.

Categories: 13 SDLC roles (pm, architect, coder, reviewer, researcher, writer, pjm, devops, tester, cto, cpo, ceo, assistant) + 3 business (dev, sales, cs) + 1 operations (itadmin). `assistant` is the default router (`is_default=true`).

### Database

PostgreSQL with 12 migrations (000001–000012). Key tables: `agents`, `agent_context_files`, `sessions`, `messages`, `llm_providers`, `agent_teams`, `agent_links`, `traces`, `channel_instances`, `topic_config`. RLS via `owner_id` = tenant ID.

### Provider Registration

Providers configured in `config.json` or loaded from DB (managed mode). DB providers take precedence. Bflow AI-Platform uses `X-API-Key` + `X-Tenant-ID` headers (not Bearer auth). All inference MUST go through Bflow AI-Platform.

## Key Constraints

- **Bflow AI-Platform only**: No direct LLM calls bypassing the platform.
- **Tenant isolation**: Every DB query must propagate `tenant_id` via context. Use `store.WithTenantID(ctx, id)`.
- **No AGPL imports**: Access MinIO/Grafana via HTTP API only, never import their SDKs.
- **No TODO/placeholder code**: All implementations must be production-ready.
- **Secrets in `.env` only**: Never put tokens/keys in `config.json` or commit `.env`.
- **SOUL char budget**: Reference docs in `docs/` can exceed 2500 chars, but deployed system prompts (from `agent_context_files`) must stay within budget.

## Environment Setup

```bash
cp .env.example .env    # Then fill in: GOCLAW_POSTGRES_DSN, GOCLAW_BFLOW_API_KEY, GOCLAW_TELEGRAM_TOKEN, GOCLAW_ENCRYPTION_KEY
```

Key env vars: `GOCLAW_POSTGRES_DSN`, `GOCLAW_BFLOW_API_KEY`, `GOCLAW_BFLOW_BASE_URL`, `BFLOW_TENANT_ID`, `GOCLAW_TELEGRAM_TOKEN`, `GOCLAW_PROVIDER` (default: `bflow-ai-platform`), `GOCLAW_MODEL` (default: `qwen3:14b`).

## CLI Subcommands

`goclaw` (default = gateway), `onboard`, `agent` (list/add/delete/chat), `pairing`, `config`, `channels`, `cron`, `skills`, `sessions`, `migrate` (up/down), `upgrade`, `doctor`, `version`.
