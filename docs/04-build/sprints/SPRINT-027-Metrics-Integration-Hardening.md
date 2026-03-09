# SPRINT-027: Success Metrics + Integration Specs + Cost Hardening

**Status**: COMPLETE
**Duration**: 5 days (planned), implementation in progress
**Depends on**: Sprint 26 (Bridge Production Readiness — COMPLETE)
**Priority**: P0 (Observability foundation, SDLC realignment, cost controls)

---

## Objective

Instrument adoption/usage metrics from existing PG traces, create missing integration specification documents, extend cost guardrails with monthly token tracking and warning thresholds, fix gap analysis errata from Sprint 26, generate OpenAPI specification for REST+WebSocket APIs, and add provider chain E2E tests.

## Context

- No Prometheus in stack — only OTel tracing to PG via `tracing.Collector`
- `TracingStore` has `SpanData` with `InputTokens`, `OutputTokens`, `Model`, `Provider`, `SpanType` — basis for metrics
- `internal/cost/guardrails.go` only counts daily requests; monthly token limit (`TENANT_MONTHLY_TOKEN_LIMIT`) defined in `.env.example` but never enforced
- Missing integration specs for bridge protocol, fallback chain, and provider registry
- 70+ REST routes + 40+ WebSocket RPC methods have no OpenAPI specification
- Provider chain fallback path has no E2E test coverage
- Sprint 26 gap analysis has 3 errata items needing correction

## PJM-026 Carryforward

| ID | Item | Priority | Target |
|----|------|----------|--------|
| PJM-026-P1 | PG integration test with real DB | P2 | Deferred to Sprint 28 T28.3 |
| PJM-026-P2 | `/cc reconnect` or auto-cleanup for disconnected sessions | P3 | Sprint 27 backlog |
| PJM-026-P3 | Audit JSONL rotation policy | P3 | Sprint 27 backlog |

---

## Deliverables

### T27.1: Success Metrics Instrumentation (1.5d) — REALIGNMENT

**Rationale**: Product vision (Stage 00) defined success metrics (80% WAU, <30% waste, 3h/week savings, 95% context retention) but they were never instrumented. No Prometheus exists — use PG trace queries instead.

#### Step 1: Extend TracingStore Interface

**File**: `internal/store/tracing_store.go`

Add 4 adoption query methods:

```go
// Adoption metrics — Sprint 27
CountDistinctUsers(ctx context.Context, since time.Time) (int, error)           // WAU
CountByAgent(ctx context.Context, since time.Time) (map[string]int, error)       // per-SOUL usage
CountByChannel(ctx context.Context, since time.Time) (map[string]int, error)     // per-channel
SumTokensByProvider(ctx context.Context, since time.Time) (map[string][2]int, error) // [input, output] per provider
```

#### Step 2: PG Implementation

**File**: `internal/store/pg/tracing.go`

Implement 4 methods using existing `traces` + `spans` tables:

| Method | Query Pattern |
|--------|--------------|
| `CountDistinctUsers` | `SELECT COUNT(DISTINCT user_id) FROM traces WHERE created_at >= $1 AND user_id != ''` |
| `CountByAgent` | `SELECT a.agent_key, COUNT(*) FROM traces t JOIN agents a ON t.agent_id = a.id WHERE t.created_at >= $1 GROUP BY a.agent_key` |
| `CountByChannel` | `SELECT channel, COUNT(*) FROM traces WHERE created_at >= $1 AND channel != '' GROUP BY channel` |
| `SumTokensByProvider` | `SELECT provider, SUM(input_tokens), SUM(output_tokens) FROM spans WHERE created_at >= $1 AND span_type = 'llm_call' GROUP BY provider` |

#### Step 3: Wire into Tracing Collector

**File**: `internal/tracing/collector.go`

Add `AdoptionMetrics()` method that calls the 4 store methods and returns a structured `AdoptionReport`:

```go
type AdoptionReport struct {
    WAU            int
    ByAgent        map[string]int
    ByChannel      map[string]int
    TokensByProvider map[string][2]int // [input, output]
    Since          time.Time
}
```

#### Step 4: Add Doctor Section

**File**: `cmd/doctor.go`

Add "Adoption Metrics" section (managed mode only, requires DB connection + TracingStore):

```
  Adoption Metrics (last 7 days):
    WAU:            42
    By SOUL:        assistant: 120, coder: 85, pm: 45, reviewer: 30
    By Channel:     telegram: 200, msteams: 80
    Tokens:         anthropic: 1.2M in / 450K out, openrouter: 800K in / 300K out
```

**Implementation note**: Create `TracingStore` from `db` (already opened in doctor), call `CountDistinctUsers`, `CountByAgent`, `CountByChannel`, `SumTokensByProvider` with `since = 7 days ago`.

#### Step 5: Tests

**File**: `internal/store/pg/tracing_adoption_test.go`

- Test `CountDistinctUsers` with zero/multiple users
- Test `CountByAgent` grouping
- Test `CountByChannel` grouping
- Test `SumTokensByProvider` aggregation
- Test empty results (no traces in range)

---

### T27.5: Integration Specification Documents (0.75d) — REALIGNMENT

**Rationale**: Stage 03 (Integrate) missing specs for bridge protocol, fallback chain, and provider registry.

#### T27.5a: Bridge Integration Spec

**File**: `docs/03-integrate/bridge-integration.md`

Contents:
- Hook protocol (HookServer ↔ Claude CLI interaction)
- Permission model (risk modes: auto, supervised, manual)
- Session lifecycle (created → running → stopped → disconnected)
- Audit events (JSONL + PG dual-write format)
- Configuration (env vars, config.json bridge section)
- Design note: future extraction of `/cc` commands to channel-agnostic handler

#### T27.5b: Provider Fallback Integration Spec

**File**: `docs/03-integrate/provider-fallback-integration.md`

Contents:
- Fallback chain behavior (primary → fallback with retryable-error classification)
- Retry policy (exponential backoff in `providers/retry.go`)
- Configuration (provider_chain in config.json, per-agent DB overrides)
- CTO guards (no fallback at iter=1 with tools — prevents partial tool calls)
- Monitoring (fallback spans tagged in traces)
- ADR-014 reference

#### T27.5c: Update API Reference

**File**: `docs/03-integrate/api-reference.md`

Add missing sections:
- Provider registry endpoints
- Custom tools endpoints
- Bridge session management endpoints
- Channel instance CRUD

---

### T27.6: Cost Guardrails Extension (0.5d) — REALIGNMENT

**Rationale**: Monthly token limit defined in `.env.example` (`TENANT_MONTHLY_TOKEN_LIMIT`) but never enforced. Daily limit only counts requests, not tokens.

#### Step 1: Monthly Token Tracking

**File**: `internal/cost/guardrails.go`

Add `CheckMonthlyTokenLimit()`:

```go
func CheckMonthlyTokenLimit(ctx context.Context, tracingStore store.TracingStore) (exceeded bool, totalTokens int, limit int, err error)
```

- Query `SumTokensByProvider()` with `since = first day of current month`
- Sum all input+output tokens across providers
- Compare against `MTCLAW_TENANT_MONTHLY_TOKEN_LIMIT` env var (default: 10,000,000)
- Fail-open on error (consistent with `CheckDailyLimit`)

#### Step 2: Warning Threshold

**File**: `internal/cost/guardrails.go`

Add `CheckWarningThreshold()`:

```go
func CheckWarningThreshold(ctx context.Context, tracingStore store.TracingStore) (warning bool, usage float64, threshold float64, err error)
```

- Emit structured WARN log when daily request count exceeds configurable percentage (default 80%)
- Env var: `MTCLAW_COST_WARNING_THRESHOLD` (0.0-1.0, default 0.8)

#### Step 3: Wire into Consumer

**File**: `cmd/gateway_consumer.go`

Add monthly token check alongside existing daily limit check in the consumer pre-flight:

```go
// Existing daily limit check
if exceeded, count, limit, err := cost.CheckDailyLimit(ctx, tracingStore); ...

// NEW: monthly token limit check
if exceeded, tokens, limit, err := cost.CheckMonthlyTokenLimit(ctx, tracingStore); ...
```

#### Step 4: Doctor Section

**File**: `cmd/doctor.go`

Add "Cost Status" section:

```
  Cost Status:
    Daily:   145 / 500 requests (29%)
    Monthly: 2.3M / 10M tokens (23%)
    Status:  OK
```

#### Step 5: Tests

**File**: `internal/cost/guardrails_test.go`

- Test `CheckMonthlyTokenLimit` with mock store (under/over limit)
- Test `CheckWarningThreshold` at various percentages
- Test fail-open behavior on store error
- Test env var parsing for limits

---

### T27.7: Gap Analysis Errata (0.25d) — REALIGNMENT

Fix 3 errata from Sprint 26 gap analysis:

| # | Errata | Fix |
|---|--------|-----|
| 1 | Agent Teams `team_tasks` was misframed as "dead code" — it's ADR-012 Option B (APPROVED) | Update gap analysis in Sprint 26 doc (already corrected) |
| 2 | Sprint 24 doc missing coder handoff format | Add coder handoff section to SPRINT-024-COMPLETION.md |
| 3 | System architecture doc v2.0.0 missing MCP server section | Add MCP bridge section to SAD |

**Files**:
- `docs/04-build/sprints/SPRINT-024-COMPLETION.md` — Add coder handoff format
- `docs/02-design/system-architecture-document.md` — Add MCP server bridge section

---

### T27.8: OpenAPI Specification (0.75d) — REALIGNMENT

**Rationale**: 70+ REST routes + 40+ WebSocket RPC methods have no formal API specification. Stage 01 planning gap.

**File**: `docs/01-planning/openapi-spec.yaml`

Generate OpenAPI 3.0 spec covering:

**REST Endpoints** (from `internal/gateway/router.go` + `cmd/gateway_methods.go`):
- `POST /api/v1/chat` — Message inference
- `GET/POST /api/v1/agents` — Agent CRUD
- `GET /api/v1/sessions` — Session history
- `GET/PUT /api/v1/config` — Configuration
- `GET/POST /api/v1/skills` — Skill library
- `GET/POST /api/v1/cron` — Background tasks
- `GET /api/v1/usage` — Token/cost analytics
- `POST /api/v1/send` — Outbound routing
- Provider registry, custom tools, channel instances

**WebSocket RPC Methods** (from `cmd/gateway_methods.go`):
- Chat methods (send, stream, cancel)
- Agent methods (list, get, create, update, delete)
- Session methods (list, get, delete, rename)
- Config methods (get, set)
- Skill methods (list, get, create, update, delete)
- Cron methods (list, create, delete)
- Pairing methods (request, status, confirm)
- Usage methods (summary, by-agent, by-provider)
- Exec approval methods (list, approve, deny)
- Send methods (message, media)
- Delegation methods (list, get)

**Implementation approach**: Document the actual method signatures from `gateway_methods.go` registrations. Include request/response schemas derived from `pkg/protocol/` types.

---

### T27.9: Provider Chain E2E Tests (1.25d) — NEW

**Rationale**: Provider fallback chain (ADR-014) has unit tests but no E2E coverage. Critical path for production reliability.

#### Step 1: Test Helpers

**File**: `internal/providers/mock_provider_test.go`

Create test helpers:
- `mockProvider` implementing `providers.Provider` with configurable responses/errors
- `failingProvider` that always returns specific error types (retryable vs fatal)
- `slowProvider` with configurable latency

#### Step 2: Fallback Chain Tests

**File**: `internal/agent/loop_fallback_test.go`

Test scenarios:
- Primary succeeds → no fallback triggered
- Primary fails (retryable error) → fallback succeeds
- Primary fails (fatal error) → no fallback, error returned
- Primary timeout → fallback succeeds
- Both primary and fallback fail → error with both failure details
- Fallback at iter>1 with tools → allowed (CTO guard: no fallback at iter=1 with tools)

#### Step 3: Tracing Verification

**File**: `internal/agent/loop_fallback_test.go`

Verify fallback spans are correctly emitted:
- Primary failure span (status=error)
- Fallback success span (metadata: `fallback=true`, `primary_provider`, `primary_error`)
- Trace aggregates include fallback token counts

---

## Test Coverage

| Test File | Tests | Status |
|-----------|-------|--------|
| `internal/store/pg/tracing_adoption_test.go` | 5 (TokenUsage struct, SQL patterns, time range, empty map, aggregation) | PASS |
| `internal/cost/guardrails_test.go` | 9 (daily under/at/fail-open, monthly under/exceeded/fail-open, warning below/above/fail-open) | PASS |
| `internal/agent/fallback_test.go` | 11 (5 existing + 6 new E2E: primary ok, retryable fallback, fatal no-fallback, both fail, CTO guard, tools stripped) | PASS |
| Full suite regression | All existing tests | PASS |

---

## Acceptance Criteria

- [x] `./mtclaw doctor` shows per-SOUL adoption metrics from PG traces (7-day window)
- [x] `./mtclaw doctor` shows cost status (daily requests + monthly tokens)
- [x] Monthly token limit enforced in consumer pre-flight (configurable via env var)
- [x] Warning logs emitted at 80% daily usage threshold
- [x] Integration specs for bridge and fallback exist in `docs/03-integrate/`
- [x] OpenAPI spec covers all REST + WebSocket methods
- [x] Gap analysis errata corrected in Sprint 24 completion
- [x] Provider chain fallback E2E tests pass (6 scenarios)
- [x] All existing tests pass — zero regressions

---

## Risk Register

| Risk | Mitigation |
|------|------------|
| PG adoption queries slow on large traces table | Add `WHERE created_at >= $1` index hint; queries are doctor-only, not hot path |
| Monthly token limit too aggressive for active tenants | Default 10M tokens with env var override; fail-open on error |
| OpenAPI spec maintenance overhead | Generate from code patterns; document as living spec |

---

## Sprint 27 → Sprint 28 Handoff

Sprint 28 targets:
- **T28.1**: Health-Based Provider Routing (sliding window health tracker)
- **T28.2**: OTel Metrics Pipeline (proper counters, not trace queries)
- **T28.3**: E2E Test Automation (bridge persistence + fallback with real DB)
- **T28.4**: Deployment Runbooks + Fallback Benchmarks
- **T28.5**: Legacy Sprint File Cleanup
