# Technology Stack — MTClaw

**SDLC Stage**: 01-Planning
**Version**: 1.0.0
**Date**: 2026-03-02
**Author**: [@pm]
**Framework**: SDLC 6.1.1 — Stage 01 Required Artifact (STANDARD tier)

---

## 1. Stack Overview

| Layer | Technology | Version | License |
|-------|-----------|---------|---------|
| **Runtime** | GoClaw (Go) | Go 1.25 | MIT (upstream) |
| **Database** | PostgreSQL | 15+ | PostgreSQL License |
| **Vector Search** | pgvector | 0.5+ | PostgreSQL License |
| **AI Provider** | Bflow AI-Platform | — | Internal (NQH) |
| **AI Model** | qwen3:14b | — | Apache 2.0 |
| **Channel (P1)** | Telegram Bot API | — | — |
| **Channel (P2)** | Zalo API | — | — |
| **Observability** | slog + OTEL | Go stdlib | BSD |
| **Metrics** | Prometheus | 2.x | Apache 2.0 |
| **Encryption** | AES-256-GCM | Go crypto | BSD |
| **Container** | Docker + Compose | 24+ | Apache 2.0 |

---

## 2. Technology Decisions

### 2.1 Runtime: GoClaw (Go 1.25)

**Decision**: Adopt GoClaw as runtime (ADR-001)

| Criterion | Evaluation |
|-----------|-----------|
| Why Go | Single binary (~25MB), <35MB RAM, <1s startup, native concurrency |
| Why GoClaw | Proven multi-tenant agent platform, PostgreSQL-native, MIT license |
| Alternative rejected | MTS-OpenClaw (TypeScript) — no multi-tenant PostgreSQL, no RLS support |
| Risk | Go competency gap in team |
| Mitigation | AI Codex strategy, CTO review gate, 90-day eval (ADR-001) |

### 2.2 Database: PostgreSQL 15+ with pgvector

**Decision**: PostgreSQL as sole database

| Criterion | Evaluation |
|-----------|-----------|
| Why PostgreSQL | RLS (row-level security), JSONB, pgvector, GoClaw native support |
| Why pgvector | Hybrid RAG: 70% vector (cosine) + 30% BM25 (tsvector) — no external service |
| Why not Redis | GoClaw doesn't require cache layer for Phase 1 scale (10 users) |
| Why not separate vector DB | pgvector eliminates operational complexity of Pinecone/Qdrant |
| Hosting | Single VPS, Docker container, connection pooling if needed (PgBouncer) |

### 2.3 AI Provider: Bflow AI-Platform

**Decision**: Single source of AI infrastructure (ADR-002, ADR-005)

| Criterion | Evaluation |
|-----------|-----------|
| Why single provider | Centralized cost control, audit trail, model management |
| API format | OpenAI-compatible: `POST /v1/chat/completions` |
| RAG API | `POST /v1/rag/query` with collection filter |
| Auth | `X-API-Key` (prefix `aip_`) + `X-Tenant-ID` header |
| Public endpoint | `https://api.nhatquangholding.com` (Bflow AI-Platform) |
| Local endpoint | `http://ai-platform:8120` via Docker `ai-net` network (same server) |
| Cost | $0/query (internal infrastructure, RTX 5090 32GB) |
| Model | qwen3:14b (updated from qwen2.5:14b) |
| Fallback | Graceful degradation — user-friendly error, log, retry option |
| **Rule** | **No direct LLM calls bypassing AI-Platform** |

**Note**: `ai.nhatquangholding.com` is a separate NQH IT Admin temporary infrastructure (Ollama direct, experimental). MTClaw does NOT use it — all inference goes through `api.nhatquangholding.com` (Bflow AI-Platform).

### 2.4 Messaging Channels

**Decision**: Telegram first, Zalo second (ADR-002)

| Phase | Channel | Rationale |
|-------|---------|-----------|
| Phase 1 (MTS) | Telegram | Tech-savvy team, proven OpenClaw integration, bot API mature |
| Phase 2 (NQH) | Zalo | Most popular app in VN for F&B/hospitality staff, non-tech users |

GoClaw channel abstraction (`internal/channels/`) supports multiple channels via unified `InboundMessage` / `OutboundMessage` interface.

### 2.5 Observability: slog + OTEL + Prometheus

**Decision**: Go-native observability stack (ADR-003)

| Component | Technology | Purpose |
|-----------|-----------|---------|
| Logging | Go `slog` | Structured JSON logs with trace_id, tenant_id |
| Tracing | GoClaw traces/spans tables | Request lifecycle tracking + cost |
| Metrics | OTEL → Prometheus | Counter/histogram export at `/metrics` |
| Dashboards | Grafana (optional) | Visualization (iframe only — AGPL containment) |

### 2.6 Security

| Component | Technology | Purpose |
|-----------|-----------|---------|
| Tenant isolation | PostgreSQL RLS | Row-level security, defense-in-depth |
| Session scoping | `SET LOCAL app.tenant_id` | Transaction-scoped, auto-reset on commit |
| Secret encryption | AES-256-GCM (Go crypto) | `config_secrets` table values |
| Auth | JWT + Telegram user verification | API access + channel identity |
| Admin bypass | `mtclaw_admin` PostgreSQL role | Migrations and admin operations |

---

## 3. Deployment Architecture

### Phase 1 (MTS — 10 users)

```
┌─────────────────────────────────────────────────┐
│  VPS (4 vCPU, 8GB RAM, 100GB SSD)               │
│                                                  │
│  ┌──────────────┐  ┌──────────────────────────┐ │
│  │ MTClaw binary │  │ PostgreSQL 15 + pgvector │ │
│  │ (~25MB, Go)   │  │ (Docker container)       │ │
│  │ Port 8080     │  │ Port 5432                │ │
│  └──────┬───────┘  └──────────────────────────┘ │
│         │                                        │
│  ┌──────┴───────┐                                │
│  │ Prometheus    │  (optional, Docker)            │
│  │ Port 9090     │                                │
│  └──────────────┘                                │
└──────────┬──────────────────────────────────────┘
           │
    ┌──────┴──────┐        ┌──────────────────────┐
    │  Telegram    │        │  Bflow AI-Platform    │
    │  Bot API     │        │  ai.nhatquangholding  │
    └─────────────┘        │  .com                 │
                           └──────────────────────┘
```

### Cost

| Component | Monthly | Annual |
|-----------|---------|--------|
| VPS | $70-140 | $840-1,680 |
| Bflow AI-Platform | $0 | $0 |
| Domain/SSL | ~$1 | ~$12 |
| **Total** | **$71-141** | **$852-1,692** |

---

## 4. Dependency Inventory

### Go Dependencies (from GoClaw go.mod)

| Category | Key Dependencies | License |
|----------|-----------------|---------|
| HTTP | net/http (stdlib) | BSD |
| Database | lib/pq (PostgreSQL driver) | MIT |
| Migrations | golang-migrate | MIT |
| JSON | encoding/json (stdlib) | BSD |
| Crypto | crypto/aes, crypto/cipher | BSD |
| Logging | log/slog (stdlib) | BSD |
| OTEL | go.opentelemetry.io/otel | Apache 2.0 |
| Telegram | go-telegram-bot-api | MIT |
| pgvector | pgvector-go | MIT |

**License compliance**: All dependencies are MIT/BSD/Apache 2.0. No AGPL/GPL contamination.
**Verified**: Sprint 1 license verification (`docs/00-foundation/mtclaw-license-verification.md`)

---

## 5. Technology Constraints

| Constraint | Impact | Mitigation |
|-----------|--------|------------|
| Go competency gap | Medium — team is TypeScript-native | AI Codex + CTO review gate + 90-day eval (ADR-001) |
| Single VPS | Low — 10 users Phase 1 | Monitor, scale to multi-VPS if needed Phase 2 |
| pgvector scale | Low — <5K chunks Phase 1 | IVFFlat index, monitor query performance |
| Bflow AI-Platform single point of failure | Medium | Graceful degradation, queue-and-retry pattern |
| qwen3:14b context window | Low — monitor prompt length | SOUL.md budget: max 2,000 chars |

---

## References

- [ADR-001: GoClaw Adoption](../02-design/01-ADRs/SPEC-0001-ADR-001-GoClaw-Adoption.md)
- [ADR-002: Three-System Architecture](../02-design/01-ADRs/SPEC-0002-ADR-002-Three-System-Architecture.md)
- [ADR-003: Observability Architecture](../02-design/01-ADRs/SPEC-0003-ADR-003-Observability-Architecture.md)
- [Business Case — Cost Analysis](../00-foundation/business-case.md)
- [GoClaw License Verification](../00-foundation/mtclaw-license-verification.md)
