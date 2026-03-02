# AGENTS.md — MTClaw

## Quick Start
- **Repo**: `https://github.com/Minh-Tam-Solution/MTClaw`
- **Framework**: SDLC 6.1.1 STANDARD tier
- **Runtime**: GoClaw (Go 1.25) + PostgreSQL + Bflow AI-Platform
- **Build**: `make build` → `./goclaw`
- **Test**: `go test ./...`

## Architecture
- **3-System** (zero runtime coupling): MTClaw runtime | EndiorBot (reference) | SDLC-Orchestrator (pattern)
- **AI**: Bflow AI-Platform ONLY — no direct LLM calls bypassing AI-Platform
- **Tenant**: PostgreSQL RLS mandatory — `tenant_id` in every query context
- **16 SOULs**: 12 SDLC (pm, architect, coder, reviewer, researcher, writer, pjm, devops, tester, cto, cpo, ceo) + assistant + 4 MTS (mts-dev, mts-sales, mts-cs, mts-general)
- **Channels**: Telegram (P1) → Zalo (P2)

## Governance: 3 Rails
1. **Spec Factory** (`/spec`): Structured spec → JSON → evidence attachment
2. **PR Gate**: Policy evaluation (WARNING → ENFORCE)
3. **Knowledge**: RAG per domain, SOUL per role

## Security
- Bflow AI auth: `X-API-Key` + `X-Tenant-ID` headers
- Secrets: `.env` (never commit), rotate every 90 days
- Encryption: AES-256-GCM for sensitive fields
- Observability: OTEL traces, `slog` structured JSON

## DO NOT
- Import AGPL libraries directly (MinIO SDK, Grafana SDK)
- Call EndiorBot CLI from MTClaw runtime
- Call SDLC-Orchestrator API from MTClaw runtime
- Bypass Bflow AI-Platform for any inference
- Skip tenant_id in database operations
- Create TODO/placeholder implementations

## Key Paths
- SOULs: `docs/08-collaborate/souls/SOUL-*.md`
- ADRs: `docs/02-design/01-ADRs/`
- Migrations: `migrations/`
- Sprint plans: `docs/04-build/sprints/`

## SDLC Workflow
Document first → Gate approval → Implementation.
Follow: `SDLC-Enterprise-Framework/` for stage definitions.
