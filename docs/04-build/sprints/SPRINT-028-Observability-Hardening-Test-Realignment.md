# SPRINT-028: Observability + Hardening + Test Realignment

**Status**: COMPLETE
**Duration**: 5 days
**Depends on**: Sprint 27 (Metrics + Integration + Hardening — COMPLETE)
**Priority**: P0 (Production readiness, observability, test automation)

---

## Objective

Add health-based provider routing with circuit breaker, instrument bridge session metrics, automate E2E fallback and bridge tests (converting MANUAL E2E-020/021 to AUTOMATED), create deployment runbooks for bridge and fallback operations, establish fallback performance benchmarks, and update the Master Test Plan to v6.0.0.

## Context

- Sprint 27 delivered adoption metrics, cost guardrails, integration specs, OpenAPI, and E2E fallback tests
- Provider fallback chain works but has no health awareness — a degraded provider gets retried every time until it fails
- Bridge session counts exist in memory (SessionManager) but are not observable outside `doctor` command
- E2E-020 (Fallback deploy) and E2E-021 (Fallback E2E via Telegram) are MANUAL — need automation
- No deployment runbooks exist for bridge or fallback operations
- No performance benchmarks for Claude CLI subprocess overhead
- Master Test Plan v5.0.0 covers Sprint 1-25; needs update for Sprint 26-28

## Sprint 27 Carryforward

| ID | Item | Priority | Target |
|----|------|----------|--------|
| PJM-026-P1 | PG integration test with real DB | P2 | T28.3 |
| OBS-027-6 | Loop-level fallback E2E (integrated with retry) | P1 | T28.3 |
| OBS-027-5 | 2x CountTraces pre-flight optimization | P3 | Backlog |
| PJM-026-P2 | `/cc reconnect` for disconnected sessions | P3 | Backlog |
| PJM-026-P3 | Audit JSONL rotation policy | P3 | Backlog |

---

## Deliverables

### T28.1: Health-Based Provider Routing (1.5d) — NEW

**Rationale**: Current fallback is blind — it tries the primary provider every time, waits for failure + retry exhaustion, then falls back. A health tracker with sliding window and circuit breaker can skip degraded providers proactively.

#### Step 1: Create ProviderHealthTracker

**File**: `internal/providers/health_tracker.go` (NEW)

```go
type ProviderHealthTracker struct {
    mu       sync.RWMutex
    records  map[string]*providerHealth // provider name → health
    window   int                         // sliding window size (default 100)
    windowDur time.Duration              // time window (default 10min)
}

type providerHealth struct {
    results    []healthResult // circular buffer
    idx        int
    totalOK    int
    totalFail  int
    lastFail   time.Time
    cbState    circuitState  // closed, open, half-open
    cbOpenAt   time.Time
    cbCooldown time.Duration // default 30s
}

type circuitState int
const (
    circuitClosed   circuitState = iota // normal
    circuitOpen                          // tripped — skip provider
    circuitHalfOpen                      // probe — allow 1 request
)
```

Key methods:
- `RecordSuccess(provider string)` — log success, transition open→half-open→closed
- `RecordFailure(provider string)` — log failure, trip breaker if threshold exceeded
- `IsHealthy(provider string) bool` — check if provider is usable
- `Score(provider string) float64` — success rate in current window (0.0-1.0)
- `Stats() map[string]HealthStats` — for doctor display

Circuit breaker logic:
- **Trip threshold**: 3 consecutive failures OR <50% success rate in window
- **Cooldown**: 30s default, configurable via `MTCLAW_PROVIDER_CB_COOLDOWN`
- **Half-open**: Allow 1 probe request after cooldown expires
- **Recovery**: Single success in half-open → closed (normal)
- **Fail-open on double circuit-open (OBS-028-1)**: If both primary AND fallback circuits are open, still attempt the fallback call rather than returning a health-check error. Rationale: a degraded provider may succeed on any individual request; blocking the user is worse than trying. Document this as explicit design choice in code comment.

#### Step 2: Wire into Agent Loop

**File**: `internal/agent/loop.go`

Modify `LoopConfig` to accept optional `*ProviderHealthTracker`:
```go
type LoopConfig struct {
    // ...existing fields...
    HealthTracker *providers.ProviderHealthTracker // optional, nil = no health tracking
}
```

In the LLM call section (~line 645-699):
- After primary success: `l.healthTracker.RecordSuccess(l.provider.Name())`
- After primary failure: `l.healthTracker.RecordFailure(l.provider.Name())`
- Before fallback call: check `l.healthTracker.IsHealthy(l.fallbackProvider.Name())` — but if unhealthy AND no other fallback option exists, still attempt (fail-open per OBS-028-1)
- After fallback success/failure: record accordingly

#### Step 3: Wire in Gateway

**File**: `cmd/gateway.go`

Create singleton `ProviderHealthTracker` at startup, inject into all `LoopConfig` instances.

#### Step 4: Doctor Display

**File**: `cmd/doctor.go`

Add "Provider Health" section (managed mode only):
```
Provider Health:
  bflow-ai-platform:  score=0.98  circuit=closed  (97/100 OK)
  claude-cli:         score=1.00  circuit=closed  (5/5 OK)
  openrouter:         score=0.85  circuit=closed  (85/100 OK)
```

#### Step 5: Tests

**File**: `internal/providers/health_tracker_test.go` (NEW)

| Test | Description |
|------|-------------|
| TestHealthTracker_InitialHealthy | New provider is healthy (no data) |
| TestHealthTracker_RecordSuccess | Success increases score |
| TestHealthTracker_RecordFailure | Failure decreases score |
| TestHealthTracker_CircuitBreaker_Trip | 3 consecutive failures trip breaker |
| TestHealthTracker_CircuitBreaker_Cooldown | After cooldown, transitions to half-open |
| TestHealthTracker_CircuitBreaker_Recovery | Single success in half-open → closed |
| TestHealthTracker_SlidingWindow | Old results expire from window |
| TestHealthTracker_Score_Empty | Empty provider returns 1.0 (healthy) |
| TestHealthTracker_Stats | Stats returns correct per-provider data |
| TestHealthTracker_Concurrent | Race-safe under concurrent access |

**Estimated tests**: 10

---

### T28.2: Bridge Session Metrics (0.75d) — NEW

**Rationale**: Bridge sessions are tracked in-memory by SessionManager but not observable from metrics queries. Add structured session counting for doctor and future OTel metrics.

#### Step 1: Add BridgeMetrics struct

**File**: `internal/claudecode/session_manager.go`

```go
type BridgeMetrics struct {
    ActiveSessions    int            `json:"active_sessions"`
    TotalCreated      int            `json:"total_created"`
    TotalKilled       int            `json:"total_killed"`
    ByRiskMode        map[string]int `json:"by_risk_mode"`
    ByAgentRole       map[string]int `json:"by_role"`
    ByChannel         map[string]int `json:"by_channel"` // OBS-028-5: channel (telegram/msteams) is more useful than provider (always claude-cli)
    AvgSessionDuration time.Duration `json:"avg_session_duration"`
}

func (sm *SessionManager) Metrics() BridgeMetrics { ... }
```

#### Step 2: Doctor Display

**File**: `cmd/doctor.go`

Add "Bridge Sessions" section (when bridge enabled):
```
Bridge Sessions:
  Active:     2
  Created:    15 (lifetime)
  Killed:     13 (lifetime)
  By Risk:    read=8  patch=4  interactive=1
  By Role:    coder=6  pm=3  (bare)=4
  Avg Duration: 12m30s
```

#### Step 3: Tests

**File**: `internal/claudecode/session_metrics_test.go` (NEW)

| Test | Description |
|------|-------------|
| TestBridgeMetrics_Empty | No sessions → all zeros |
| TestBridgeMetrics_ActiveCount | Create 3, kill 1 → active=2 |
| TestBridgeMetrics_ByRiskMode | Sessions with different risk modes counted correctly |
| TestBridgeMetrics_ByRole | Sessions with different roles counted correctly |
| TestBridgeMetrics_Lifetime | Total created/killed tracks lifetime |

**Estimated tests**: 5

---

### T28.3: E2E Test Automation (1.0d) — REALIGNMENT

**Rationale**: Convert MANUAL E2E-020/021 to AUTOMATED. Also addresses PJM-026-P1 (PG integration test) and OBS-027-6 (loop-level fallback E2E).

#### Step 1: Fallback E2E — Loop Integration Test

**File**: `internal/agent/fallback_loop_test.go` (NEW)

Test the fallback block within the agent loop (PM-028-1: targeting the internal fallback path at loop.go lines 648-699, not full `Loop.Run()` which requires bus/session/tool infrastructure). Approach: create a thin test helper that invokes the LLM call + fallback logic directly, reusing `stubProvider`/`failingProvider` from `fallback_test.go`:
- Uses `stubProvider` for primary (returns retryable error) and fallback (returns success)
- Exercises the Chat → fallback → response path in loop.go lines 652-689
- Verifies CTO guards within the loop context
- If full `Run()` setup proves necessary, budget +0.25d (PM-028-2 cut candidate: health tracker integration tests)

| Test | Description |
|------|-------------|
| TestLoopRun_FallbackOnRetryableError | Primary 502 → fallback succeeds → user gets response |
| TestLoopRun_NoFallbackOnFatalError | Primary 400 → error propagated, no fallback |
| TestLoopRun_CTOGuard_Iter1WithTools | Primary fails at iter=1 with tools → error propagated |
| TestLoopRun_ToolsStrippedInFallbackReq | Fallback request has nil tools (CTO-501) |
| TestLoopRun_FallbackTracingSpans | Primary fail span + fallback success span emitted |
| TestLoopRun_BothFail_ErrorPropagated | Primary + fallback both fail → error returned |

**Estimated tests**: 6

#### Step 2: Bridge Session Persistence Test

**File**: `internal/integration/bridge_persistence_test.go` (NEW)

Test BridgeSessionStore round-trip (if PG store implemented in Sprint 26):
- Create session → Upsert to PG → Restart (LoadFromStore) → Verify status=disconnected
- ListByTenant filtering
- UpdateRiskMode + verify
- DeleteOlderThan cleanup

If PG store not yet available, create structural test that validates the store interface contract:

| Test | Description |
|------|-------------|
| TestBridgePersistence_UpsertAndGet | Create → Upsert → Get returns same data |
| TestBridgePersistence_ListByTenant | Two tenants → ListByTenant returns only matching |
| TestBridgePersistence_RecoveryAsDisconnected | Non-stopped sessions recovered as disconnected |
| TestBridgePersistence_CleanupOldSessions | Old stopped sessions cleaned, recent kept |

**Estimated tests**: 4

#### Step 3: Health Tracker Integration with Loop

**File**: `internal/agent/fallback_loop_test.go` (append)

| Test | Description |
|------|-------------|
| TestLoopRun_HealthTracker_RecordsSuccess | Primary success → tracker.RecordSuccess called |
| TestLoopRun_HealthTracker_RecordsFailure | Primary failure → tracker.RecordFailure called |
| TestLoopRun_HealthTracker_SkipsUnhealthyFallback | Fallback tripped → error propagated |

**Estimated tests**: 3

---

### T28.4: Deployment Runbooks (0.5d) — REALIGNMENT

**Rationale**: No operational documentation exists for bridge or fallback deployment. Operations team needs step-by-step guides.

#### Step 1: Bridge Deployment Runbook

**File**: `docs/06-deploy/bridge-deployment-runbook.md` (NEW)

Sections:
1. **Prerequisites**: tmux, Claude CLI, OAuth login, hook scripts
2. **Docker Deployment**: Build args, compose config, volume mounts
3. **Bridge Setup Checklist**: enable flag, hook port, audit dir, admission limits
4. **OAuth Token Management**: Initial login, refresh procedure, volume persistence
5. **Troubleshooting**: Common errors (tmux not found, hook auth failed, session stuck)
6. **Rollback**: Disable bridge without restart, emergency kill-all sessions

#### Step 2: Fallback Operations Runbook

**File**: `docs/06-deploy/fallback-operations-runbook.md` (NEW)

Sections:
1. **Provider Chain Configuration**: config.json, env vars, precedence rules
2. **Health Monitoring**: Doctor output, health tracker scores, circuit breaker states
3. **Fallback Scenarios**: Manual trigger, monitoring alerts, trace analysis
4. **Troubleshooting**: Claude CLI errors, OAuth expiry, timeout tuning
5. **Rollback**: Disable fallback (set chain to single provider), emergency direct-provider override

---

### T28.5: Fallback Performance Benchmarks (0.5d) — REALIGNMENT

**Rationale**: No performance baseline exists for fallback path. Need to measure Claude CLI subprocess overhead vs HTTP provider latency.

#### Step 1: Benchmark Tests

**File**: `internal/providers/claude_cli_bench_test.go` (NEW)

```go
func BenchmarkClaudeCLI_BuildPrompt(b *testing.B) { ... }
func BenchmarkClaudeCLI_ParseResponse(b *testing.B) { ... }
func BenchmarkClaudeCLI_FilterEnv(b *testing.B) { ... }
```

These benchmark the non-subprocess parts (prompt building, JSON parsing, env filtering) since the actual subprocess requires the `claude` binary.

#### Step 2: Latency Comparison Documentation

**File**: `docs/05-test/fallback-latency-baseline.md` (NEW)

Document expected latency ranges:
| Provider Type | Expected p50 | Expected p95 | Notes |
|---------------|-------------|-------------|-------|
| HTTP (bflow) | 500ms | 2s | Network + inference |
| HTTP (openrouter) | 800ms | 3s | Network + inference |
| Claude CLI (subprocess) | 3s | 10s | Process spawn + inference + JSON parse |
| Fallback overhead | +50ms | +200ms | Health check + request rebuild + logging |

---

### T28.6: Master Test Plan Update v6.0.0 (0.25d) — REALIGNMENT

**Rationale**: Master Test Plan v5.0.0 covers Sprint 1-25. Sprint 26-28 added significant testing (25 Sprint 27 tests, Sprint 28 tests). Need to update all sections.

**File**: `docs/05-test/MASTER-TEST-PLAN.md`

Changes:
1. **Version**: 5.0.0 → 6.0.0, coverage Sprint 1-28
2. **Sprint Feature Map**: Add Sprint 26 (Bridge PG persistence, Docker bridge, audit wiring), Sprint 27 (adoption metrics, cost guardrails, OpenAPI, E2E fallback), Sprint 28 (health routing, bridge metrics, E2E automation)
3. **Unit Tests table (§2.1)**: Add rows for:
   - `agent/fallback_test.go` — Sprint 27 E2E scenarios (6 tests)
   - `cost/guardrails_test.go` — Monthly token + warning threshold (9 tests)
   - `store/pg/tracing_adoption_test.go` — Adoption metric queries (5 tests)
   - `providers/health_tracker_test.go` — Health tracking + circuit breaker (10 tests)
   - `agent/fallback_loop_test.go` — Loop-integrated fallback (9 tests)
   - `claudecode/session_metrics_test.go` — Bridge metrics (5 tests)
4. **Integration Tests (§2.2)**: Add Sprint 26-28 entries
5. **E2E Tests (§2.3)**: Mark E2E-020 and E2E-021 as AUTOMATED (from MANUAL)
6. **Performance (§2.5)**: Add fallback latency targets
7. **Test Execution Matrix (§3.1)**: Add Sprint 26-28 cumulative rows
8. **Traceability (§3.2)**: Add Sprint 26-28 feature→test mapping
9. **Risk Register (§7)**: Add health tracker risks

---

## Execution Order

Recommended sequence (dependency-aware):

```
T28.1 (Health Tracker)  ──► T28.3 (E2E Tests, depends on T28.1)
T28.2 (Bridge Metrics)     independent
T28.4 (Runbooks)           independent
T28.5 (Benchmarks)         independent
T28.6 (Test Plan)          after T28.1-T28.5 complete
```

**Parallel groups**:
- **Group A** (1.5d): T28.1 (health tracker core + wiring)
- **Group B** (0.75d, parallel with A): T28.2 (bridge metrics)
- **Group C** (0.5d, parallel with A): T28.4 (runbooks) + T28.5 (benchmarks)
- **Group D** (1.0d, after A): T28.3 (E2E tests using health tracker)
- **Group E** (0.25d, after all): T28.6 (master test plan update)

**Critical path**: T28.1 → T28.3 → T28.6 (2.75d)

---

## Test Coverage Summary

| Task | New Tests | Type |
|------|-----------|------|
| T28.1 | 10 | Unit (health tracker + circuit breaker) |
| T28.2 | 5 | Unit (bridge metrics) |
| T28.3 | 13 | Integration (fallback loop + bridge persistence + health integration) |
| T28.5 | 3 | Benchmark (CLI prompt/parse/env) |
| **Total** | **31** | |

**Cumulative Sprint 28**: 635 (Sprint 25) + ~10 (Sprint 26) + 25 (Sprint 27) + 31 (Sprint 28) = **~701 tests** (OBS-028-7: Sprint 26 tests included)

---

## Acceptance Criteria

- [x] Health tracker skips unhealthy providers automatically (circuit breaker trips after 3 failures)
- [x] Health tracker recovers after cooldown (half-open probe succeeds → closed)
- [x] Bridge session metrics observable via `Metrics()` method (atomic counters, no PG needed)
- [x] `fallback_loop_test.go` exercises health tracker wiring + circuit breaker integration (6 tests)
- [ ] Bridge persistence test validates store contract — DEFERRED (depends on Sprint 26 T26.2 PG store)
- [x] E2E-020 and E2E-021 marked AUTOMATED in Master Test Plan
- [x] Deployment runbooks exist for bridge and fallback operations
- [x] Fallback benchmarks establish latency baseline (BuildPrompt 247ns, ParseResponse 3μs, FilterEnv 131ns)
- [x] Master Test Plan updated to v6.0.0 covering Sprint 1-28
- [x] All new tests pass: `make test` clean
- [x] No race conditions: `-race` flag clean

---

## Risk Log

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Health tracker false positives (trips on transient errors) | Medium | Medium | Require 3 consecutive failures, not single failure |
| Circuit breaker cooldown too long/short | Low | Low | Configurable via env var, 30s default validated empirically |
| Benchmark tests require `claude` binary | Low | Low | Only benchmark non-subprocess parts (prompt/parse/env) |
| Bridge persistence test blocked (no PG store yet) | Medium | Low | Fall back to interface contract test |
| Loop integration tests require careful stub setup | Low | Medium | Reuse existing `stubProvider`/`failingProvider` patterns from fallback_test.go |

---

**Created**: 2026-03-08
**Author**: [@coder], based on Sprint 28 research
**Approved by**: PJM (8.8/10) — 3 adjustments applied: PM-028-1 (entry point clarification), OBS-028-1 (fail-open spec), OBS-028-5 (ByChannel), OBS-028-7 (test count fix)
