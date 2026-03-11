# SPEC-0015 / ADR-015: Memory Enhancement — ClawVault Feature Porting

**Status**: APPROVED
**Date**: 2026-03-11
**Authors**: PM + Architect
**Sprint**: 31+ (Phase 0 starts Sprint 31)
**References**: ClawVault v3.2.0 (`clawvault/`), `internal/agent/memoryflush.go`, `internal/tools/memory_interceptor.go`
**CTO Review**: APPROVED — 7 mandatory changes + 7 corrections applied

---

## 1. Problem Statement

MTClaw's memory system has three gaps compared to ClawVault's structured memory approach:

**Gap 1: Unstructured memory flush.** When approaching compaction, the agent writes free-form notes to `memory/YYYY-MM-DD.md`. There is no structured extraction — the LLM self-selects what to remember under time pressure. Result: important facts (project decisions, user preferences, entity relationships) are lost or buried in prose.

**Gap 2: No fact store.** When asked "what Go version are we using?" or "who is the project lead?", `memory_search` returns noisy text chunks. There is no entity-relation-value index for precise factual queries.

**Gap 3: No automatic context injection.** Agents must explicitly call `memory_search` tool each turn to recall past context. This burns 1-2 tool calls per turn and relies on agent discipline.

**Business impact:**
- Agents lose context across compaction boundaries ("context death")
- Users repeat information that was previously told to the agent
- Multi-session workflows degrade as conversation history is truncated

## 2. Decision

Port 3 ClawVault features to MTClaw in 4 phases, each gated on the previous phase's success. Start with a zero-risk prompt improvement (Phase 0) before writing any code.

### 2.1 Feature Assessment (10 ClawVault features evaluated)

| # | ClawVault Feature | Verdict | Rationale |
|---|-------------------|---------|-----------|
| 1 | Observation Pipeline | **PORT** (Phase 2) | Systematic extraction > LLM self-selection |
| 2 | Fact Store | **PORT** (Phase 1) | Precise factual queries, entity-relation-value |
| 3 | Auto Context Injection | **PORT** (Phase 3) | Eliminates tool-call overhead for memory recall |
| 4 | Knowledge Graph | **DEFER** | Flat facts sufficient for v1; graph adds complexity |
| 5 | Context Profiles | **SKIP** | SOUL system provides implicit specialization |
| 6 | Session Lifecycle | **SKIP** | Always-on gateway; compaction handles context |
| 7 | Hybrid Search (RRF) | **SKIP** | Already has FTS + pgvector hybrid |
| 8 | Workgraph Coordination | **SKIP** | Team system already exists |
| 9 | Obsidian/Canvas | **SKIP** | Server-side gateway, UI is chat channels |
| 10 | Tailscale/WebDAV Sync | **SKIP** | PostgreSQL is single source of truth |

### 2.2 Phase 0: Improved Memory Flush Prompt (Sprint 31, 1-2 days)

Improve `DefaultMemoryFlushPrompt` in `memoryflush.go` to explicitly request structured output with entity-relation-value keys. Zero code risk — prompt template change only.

**Go/No-Go Gate (Phase 0 → Phase 1):**
- Structured entries <40% after prompt change → proceed to Phase 1
- Structured entries ≥60% → skip Phase 1 (prompt is sufficient)
- 40-60% → iterate prompt once more, then decide

### 2.3 Phase 1: Fact Store (Sprint N, 5-7 days)

**What**: `memory_facts` table with entity-relation-value triples, conflict resolution (newer supersedes older), LLM-based extraction.

**Extraction trigger**: `MemoryInterceptor.WriteFile()` — async goroutine after successful `PutDocument`. Bounded worker pool limits concurrent extractions across tenants.

**New components**:
- `internal/facts/extractor.go` — `FactExtractor` interface + LLM-based default (via Bflow AI-Platform). Vietnamese = LLM-only. Rule-based is Phase 1.5 optimization.
- `internal/facts/store.go` — CRUD + conflict resolution. `query()` returns active facts only (`WHERE valid_until IS NULL`).
- Migration 000021: `memory_facts` table with explicit RLS policy

**Go/No-Go Gate (Phase 1 → Phase 2):**
- Fact query precision <70% (manual review of 20 queries) → proceed
- Fact query precision ≥70% → evaluate if observations add value

### 2.4 Phase 2: Observation Pipeline (Sprint N+1, 8-12 days)

**What**: Post-compaction pipeline that extracts observations (decisions/lessons/facts/preferences) from truncated messages.

**Integration point**: `loop_history.go:maybeSummarize()` → after `TruncateHistory()` succeeds, observer pipeline runs inside the existing compaction background goroutine (inherits per-session `TryLock` serialization).

**New components**:
- `internal/observer/pipeline.go`, `compressor.go`, `classifier.go`
- Migration 000022: `memory_observations` table with `vector(1536)` embedding + explicit RLS policy

**Go/No-Go Gate (Phase 2 → Phase 3):**
- Agents still burn >1 tool call/turn on memory search despite having facts → proceed
- Tool call reduction sufficient → Phase 3 unnecessary

### 2.5 Phase 3: Context Injection (Sprint N+2, 3-4 days)

**What**: Inject top-N relevant facts into system prompt automatically. Facts only in v1 — observations behind separate `AutoInjectObservations` feature flag.

**Architecture**: `BuildSystemPrompt()` stays a pure function. Agent loop queries `memory_facts` and populates `SystemPromptConfig.RelevantFacts []string`. This matches how `ContextFiles` already works.

**Latency budget**:
| Operation | Target | Kill Switch |
|-----------|--------|-------------|
| Fact query (agent_id+user_id index) | <5ms p95 | `AutoInjectFacts: false` |
| System prompt build (with facts) | <15ms p95 total | Skip facts if >10ms |

## 3. Schema

### 3.1 memory_facts (Migration 000021)

```sql
CREATE TABLE memory_facts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    owner_id VARCHAR(255) NOT NULL,
    agent_id UUID NOT NULL REFERENCES agents(id),
    user_id TEXT,
    entity TEXT NOT NULL,
    entity_norm TEXT NOT NULL,
    relation TEXT NOT NULL,
    value TEXT NOT NULL,
    category TEXT NOT NULL,       -- preference/fact/decision/entity
    confidence REAL DEFAULT 0.5,
    source_document_id UUID,
    source_path TEXT,
    raw_text TEXT,
    last_accessed_at TIMESTAMPTZ,
    valid_from TIMESTAMPTZ DEFAULT now(),
    valid_until TIMESTAMPTZ,     -- NULL = active; set when superseded
    created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX idx_facts_entity ON memory_facts(agent_id, entity_norm) WHERE valid_until IS NULL;
CREATE INDEX idx_facts_agent_user ON memory_facts(agent_id, user_id) WHERE valid_until IS NULL;

ALTER TABLE memory_facts ENABLE ROW LEVEL SECURITY;
CREATE POLICY memory_facts_tenant_isolation ON memory_facts
    USING (owner_id = current_setting('app.tenant_id', true));
```

### 3.2 memory_observations (Migration 000022)

```sql
CREATE TABLE memory_observations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    owner_id VARCHAR(255) NOT NULL,
    agent_id UUID NOT NULL REFERENCES agents(id),
    user_id TEXT,
    session_key TEXT NOT NULL,
    category TEXT NOT NULL,       -- decision/lesson/fact/preference/event
    content TEXT NOT NULL,
    confidence REAL DEFAULT 0.5,
    importance REAL DEFAULT 0.5,
    embedding vector(1536),      -- matches memory_chunks + text-embedding-3-small
    source_compaction_count INT,
    created_at TIMESTAMPTZ DEFAULT now(),
    superseded_at TIMESTAMPTZ
);
CREATE INDEX idx_obs_agent_user ON memory_observations(agent_id, user_id) WHERE superseded_at IS NULL;
CREATE INDEX idx_obs_category ON memory_observations(category);

ALTER TABLE memory_observations ENABLE ROW LEVEL SECURITY;
CREATE POLICY memory_observations_tenant_isolation ON memory_observations
    USING (owner_id = current_setting('app.tenant_id', true));
```

## 4. Architecture Fit

```
LAYER 3: AGENT LOOP (Think → Act → Observe)
  ┌─────────────┐  ┌──────────────┐  ┌────────────────────┐
  │ System      │  │ Tool         │  │ Compaction +       │
  │ Prompt      │  │ Execution    │  │ Memory Flush       │
  │ Builder     │  │ (70+ tools)  │  │                    │
  │             │  │              │  │                    │
  │ ★ Context   │  │ ★ fact_search│  │ ★ Observation      │
  │ Injection   │  │ tool         │  │ Pipeline           │
  │ (Phase 3)   │  │ (Phase 1)    │  │ (Phase 2)          │
  └─────────────┘  └──────────────┘  └────────────────────┘

LAYER 5: DATABASE (PostgreSQL + pgvector + RLS)
  Existing:              ★ NEW:
  memory_documents       memory_facts (Phase 1)
  memory_chunks          memory_observations (Phase 2)
```

No new layers. No new external dependencies. No changes to Layers 1 (Channels), 2 (Bus/Gateway), or 4 (AI Providers). Observation pipeline reuses existing provider chain for LLM calls.

## 5. Self-Critique

| Concern | Assessment |
|---------|------------|
| **Vietnamese language**: Rule-based extraction fails for Vietnamese (no word boundaries). | Start LLM-only. Rule-based is Phase 1.5 for English patterns only. |
| **Phase 0 might solve everything** (30% probability) | Good outcome — cheapest solution wins. |
| **Observation pipeline quality**: GIGO risk if LLM produces poor summaries. | Confidence threshold >0.5. Manual `supersede` capability. Kill switch. |
| **LLM cost**: 10-50 extra calls/day from observation extraction. | Use cheapest model (`qwen3.5:9b`). ~200 tokens per call. |
| **Context window bloat** from injection. | 5 facts max, 500 token hard cap, confidence >0.5 filter. |

## 6. Consequences

**Positive:**
- Agents retain structured knowledge across compaction boundaries
- Precise factual queries without noisy chunk search
- Reduced tool-call overhead for memory recall (Phase 3)

**Negative:**
- Additional LLM calls for fact extraction (async, non-blocking)
- New tables increase schema complexity (2 tables, 2 migrations)
- Phase 2 adds coupling between compaction flow and observer pipeline

**Risks:**
- Vietnamese fact extraction quality unknown until Phase 1 validation
- Latency regression if injection queries exceed 15ms budget → kill switch mitigates

## 7. CTO Review Summary

| Round | Changes | Status |
|-------|---------|--------|
| Round 1 | 7 mandatory (LLM-first, explicit gates, facts-only injection, active-only query, flush-path trigger, confidence 0.5, bounded worker pool) | ✅ Applied |
| Round 2 | 7 corrections (embedding 1536, Phase 1/2 trigger scoping, worker pool scoping, Phase 3 Option A, explicit RLS, quantitative Phase 0 criteria, latency budget) | ✅ Applied |

**Verdict**: APPROVED. Start Phase 0 immediately.
