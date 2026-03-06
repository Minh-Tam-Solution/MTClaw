---
sprint: 12
title: Governance Engine — Spec Quality Scoring + Design-First Gate
status: PLANNED
date: 2026-03-28
version: "1.0.0"
author: "[@pm] + [@architect]"
framework: SDLC Enterprise Framework 6.1.1
tier: STANDARD
---

# Sprint 12 — Governance Engine: Spec Quality Scoring + Design-First Gate

**Sprint**: 12 of 12+
**Phase**: 4 (Governance Engine — CTO Audit remediation)
**Duration**: 5 days
**Owner**: [@coder] (implementation) + [@pm] (coordination, OaaS prep)
**Points**: ~10.5 (8 governance + 2.5 carry-forward)
**Gate**: G5 (Scale Ready) prep — gate proposal filed Sprint 11
**Entry Criteria**: see Section 1
**Detailed plan version**: v1.0.0

---

## 1. Entry Criteria

| Criterion | Status | Owner |
|-----------|--------|-------|
| CTO Sprint 11 review score received | 8.7/10 APPROVED (2026-03-06) | [@cto] |
| T11-04 performance measurements | CONDITIONAL — pending live server | [@coder] |
| G4 fully co-signed (@cto + @cpo + @ceo) | Pending | [@pm] |
| G5 gate proposal structure approved | Pending (T11-05 deliverable) | [@cto] |
| Azure AD live for NQH | Pending | [@devops] |
| Build clean + 390+ tests passing | Sprint 11 close | [@coder] |

**Start date**: 2026-03-28 (day after Sprint 11 close)

---

## 2. Sprint Goal

**Transform MTClaw from a governance framework into a governance engine — enforcing quality at the two critical decision points identified by the CTO Governance Audit.**

CTO Directive (Decision 4): _"Governance enforcement before OaaS launch is non-negotiable. Scaling bad quality to N tenants = scaling the problem."_

### Key Outcomes

1. `ProcessSpecOutput()` → `EvaluateSpecQuality()` gate inserted: specs scoring <70/100 are REJECTED with actionable reasons
2. `ResolveMention()` → `DesignFirstGate()` pre-check: code tasks delegated to @coder MUST have an approved spec; ad-hoc questions pass through
3. `AllArtifactTypes` extracted as SSOT constant (CTO-49) — no more hardcoded lists
4. PT-01 through PT-06 elevated to live integration tests (CTO-51)
5. PDF builder SHA256 determinism verified (CTO-52)
6. `go test -bench` for critical paths in CI (CTO-53)
7. Bot Framework URL prefix update procedure documented (CTO-54)

---

## 3. Architecture Analysis — [@architect]

### 3.1 Quality Scoring Insertion Point (GAP 1)

**Current flow** (no quality hooks):

```
LLM generates → ParseSpecJSON() → ContentHash → CreateSpec(DB)
                     ↑ line 85: only checks spec_version != "" && title != ""
                     ↑ line 90-94: silently defaults Priority→P1, Effort→M
                     ↑ line 109: hardcodes Tier: "STANDARD"
```

**New flow** (T12-GOV-01):

```
LLM generates → ParseSpecJSON() → ContentHash → EvaluateSpecQuality() → CreateSpec(DB)
                                                        ↑ NEW: 5-dimension scoring
                                                        ↑ < 70 → REJECT + reasons[]
                                                        ↑ Integration: spec_processor.go:46-48
```

**Design**:

```go
// internal/governance/spec_quality.go (NEW)

// QualityResult holds the scoring output.
type QualityResult struct {
    Score   int      // 0-100
    Reasons []string // failure reasons if Score < threshold
    Pass    bool
}

// SpecResult is the structured return type for ProcessSpecOutput.
// CTO Sprint 12 Decision D2: replaces fragile string prefix convention.
type SpecResult struct {
    SpecID   string        // populated on success (e.g. "SPEC-2026-0042")
    Rejected bool          // true if quality gate failed
    Quality  QualityResult // scoring details (always populated)
}

// EvaluateSpecQuality scores a GovernanceSpec across 5 dimensions.
// CTO-approved threshold: 70/100.
// Integration: called in ProcessSpecOutput after ContentHash, before CreateSpec.
func EvaluateSpecQuality(spec *store.GovernanceSpec) QualityResult
```

**5-Dimension Scoring** (100 points, CTO-approved design):

| Dimension | Points | Criteria | Evidence |
|-----------|--------|----------|----------|
| Narrative completeness | 25 | `as_a` + `i_want` + `so_that` all >20 chars | CTO Audit lines 143-144 |
| Acceptance criteria | 25 | `len(AC) >= 2`, each has `scenario` + `expected_result` | CTO Audit line 145 |
| BDD scenarios | 20 | `len(BDD) >= 1`, each has `given`/`when`/`then` | CTO Audit line 146 |
| Risk assessment | 15 | `len(Risks) >= 1`, each has `description` + `mitigation` | CTO Audit line 147 |
| Technical requirements | 15 | non-null, >50 chars | CTO Audit line 148 |

**Threshold**: <70 → REJECT with `reasons[]` returned to user. Spec NOT saved to DB.

**CTO Decision 2 note**: After T13-GOV-05 (feedback loop) ships in Sprint 13, raise threshold to 75.

### 3.2 Design-First Gate Insertion Point (GAP 6)

**Current flow** (no pre-condition):

```
ResolveMention() → agentID = "coder" → agent.Loop.Run() → LLM generates code
                         ↑ No check: does a spec exist for this task?
```

**New flow** (T12-GOV-03):

```
ResolveMention() → DesignFirstGate(agentID, content) → agent.Loop.Run()
                          ↑ NEW: if agentID == "coder" && isCodeTask(content)
                          ↑   → check: approved spec exists for this context
                          ↑   → if no spec: BLOCK + return message to user
                          ↑ CTO Decision 3: code tasks ONLY, not ad-hoc questions
```

**Scope** (CTO Decision 3):
- **BLOCK**: Task delegation (`implement X`, `build Y`, `/spec` handoff to @coder)
- **ALLOW**: Ad-hoc questions (`how do I...`, `explain...`, `debug this...`)

**Design**:

```go
// internal/governance/design_gate.go (NEW)

// DesignFirstGate checks whether a code task has an approved spec before execution.
// Returns (pass bool, reason string).
// CTO Decision 3: only blocks code tasks, not ad-hoc questions.
func DesignFirstGate(ctx context.Context, agentKey string, content string,
    specStore store.SpecStore) (bool, string)
```

**Insertion point**: `gateway_consumer.go:87` — after `agents.Get(agentID)` succeeds, before the agent loop runs.

### 3.3 AllArtifactTypes SSOT (CTO-49)

**Current**: `AllArtifactTypes = []string{"spec", "pr_gate", "test_run", "deploy"}` hardcoded in `internal/evidence/chain.go:13`.

**Fix**: Extract to `internal/store/evidence_types.go` as package-level constant. All consumers (chain builder, gate matrix future) import from single source.

### 3.4 Live Pen Tests (CTO-51)

Sprint 11 PT-01 through PT-06 are structural (unit-test style). CTO wants live integration versions that actually hit the running server. These require:
- Running server instance with populated test data
- Real HTTP calls (not mock handlers)
- Separate test file: `internal/security/pentest_live_test.go` with `//go:build integration` tag

---

## 4. Task Overview

| ID | Task | Priority | Points | Days | Owner |
|----|------|----------|--------|------|-------|
| T12-GOV-01 | Spec Quality Scoring (5 dimensions, threshold 70/100) | P0 | 3 | 1-2 | [@coder] |
| T12-GOV-03 | Design-First Gate (pre-condition hook for @coder) | P1 | 2 | 2-3 | [@coder] |
| T12-GOV-02 | AllArtifactTypes SSOT extraction (CTO-49) | P2 | 0.5 | 3 | [@coder] |
| T12-SEC-01 | Live integration pen tests PT-01..PT-06 (CTO-51) | P2 | 1 | 3-4 | [@coder] |
| T12-TEST-01 | Deterministic SHA256 test for PDF builder (CTO-52) | P3 | 0.5 | 4 | [@coder] |
| T12-TEST-02 | `go test -bench` for PDF + chain query (CTO-53) | P3 | 0.5 | 4 | [@coder] |
| T12-OPS-01 | Bot Framework URL prefix update runbook (CTO-54) | P3 | 0.5 | 5 | [@pm] |
| T12-PM-01 | OaaS readiness assessment + G5 gate evidence | P2 | 2 | 4-5 | [@pm] |

**[@pm] parallel (not in point count)**:
- T12-P1: Drive T11-04 performance measurements (CONDITIONAL carry-forward)
- T12-P2: G4 WAU final measurement + co-sign completion
- T12-P3: Sprint 13 scope planning (T13-GOV-04 Gate Matrix + T13-GOV-05 Feedback Loop)

**Total: ~10 points coding + 2.5 ops/docs, 5 days**

---

## 5. Task Specifications

---

### T12-GOV-01: Spec Quality Scoring (P0, 3 pts) — Days 1-2

**Objective**: Enforce quality at the spec creation boundary. No more silently accepting low-quality specs with empty fields and silent defaults.

**Source**: CTO Governance Audit GAP 1, CTO Decision 2 (threshold 70/100)

#### Phase 1 — Quality scorer function (Day 1)

File: `internal/governance/spec_quality.go` (NEW)

```go
// EvaluateSpecQuality scores a GovernanceSpec across 5 dimensions.
// Returns QualityResult with score (0-100), reasons[], and pass/fail.
//
// Scoring rubric (CTO-approved, Governance Audit 2026-03-06):
//   Narrative completeness    25 pts
//   Acceptance criteria       25 pts
//   BDD scenarios             20 pts
//   Risk assessment           15 pts
//   Technical requirements    15 pts
//
// Threshold: score < 70 → QualityResult.Pass = false
func EvaluateSpecQuality(spec *store.GovernanceSpec) QualityResult {
    // Parse json.RawMessage fields → check structural completeness
    // Each dimension: full marks if complete, partial for partial, 0 if missing
}
```

Narrative scoring detail (as_a/i_want/so_that):
```go
// Parse spec.Narrative JSON into:
type narrativeFields struct {
    AsA    string `json:"as_a"`
    IWant  string `json:"i_want"`
    SoThat string `json:"so_that"`
}
// Score: all 3 present + >20 chars each = 25 pts
// 2 of 3 present = 15 pts
// 1 of 3 = 8 pts
// 0 = 0 pts
```

Acceptance criteria scoring:
```go
// Parse spec.AcceptanceCriteria into []struct{ Scenario, ExpectedResult string }
// Score: len >= 2 + all have both fields = 25 pts
// len == 1 with both fields = 15 pts
// len >= 2 but missing fields = 10 pts
// len == 0 = 0 pts
```

#### Phase 2 — Integration into ProcessSpecOutput (Day 1-2)

File: `internal/governance/spec_processor.go` (MODIFY)

Insert quality check between ContentHash and CreateSpec (line 46-49):

```go
// After: spec.ContentHash = sha256Hex(output)
// Before: if err := specStore.CreateSpec(ctx, spec); err != nil {

// Sprint 12: Spec quality gate (CTO Governance Audit GAP 1)
quality := EvaluateSpecQuality(spec)
if !quality.Pass {
    slog.Warn("governance: spec quality below threshold",
        "score", quality.Score, "reasons", quality.Reasons)
    return SpecResult{Rejected: true, Quality: quality}
}

// ... CreateSpec ...
return SpecResult{SpecID: specID, Quality: quality}
```

**Return type change** (CTO Sprint 12 Decision D2): `ProcessSpecOutput` returns `SpecResult` struct instead of `string`. Caller (`gateway_consumer.go:410`) checks `result.Rejected` — if true, sends rejection message with `result.Quality.Score` and `result.Quality.Reasons` to user. Type-safe, no string parsing.

#### Phase 3 — Tests (Day 2)

File: `internal/governance/spec_quality_test.go` (NEW)

Target: ~15 tests

| Test | Input | Expected |
|------|-------|----------|
| `TestQuality_FullSpec_Passes` | All 5 dimensions complete | score >= 85, Pass = true |
| `TestQuality_MinimalSpec_Passes` | Narrative + 2 AC + 1 BDD (no risks/techreq) | score ~70, Pass = true |
| `TestQuality_EmptyNarrative_Fails` | No narrative fields | score < 70, reasons includes "narrative" |
| `TestQuality_OneAC_Fails` | Only 1 acceptance criterion | score < 70 |
| `TestQuality_NoBDD_Fails` | BDD scenarios missing | score < 70 |
| `TestQuality_PartialNarrative` | Only as_a + i_want (no so_that) | Partial score (15/25 narrative) |
| `TestQuality_ACMissingFields` | AC present but no expected_result | Partial score |
| `TestQuality_NilFields` | nil json.RawMessage for all optional fields | score = 0, graceful (no panic) |
| `TestQuality_EmptyJSON` | `{}` for narrative | score < 70 |
| `TestQuality_ThresholdEdge_69` | Constructed to score exactly 69 | Pass = false |
| `TestQuality_ThresholdEdge_70` | Constructed to score exactly 70 | Pass = true |
| `TestProcessSpec_RejectsLowQuality` | Integration: ProcessSpecOutput with bad spec | result.Rejected=true, Quality.Score<70 |
| `TestProcessSpec_AcceptsHighQuality` | Integration: ProcessSpecOutput with good spec | result.SpecID set, Rejected=false |

---

### T12-GOV-03: Design-First Gate (P1, 2 pts) — Days 2-3

**Objective**: Block code task delegation to @coder when no approved spec exists for the context. Prevent vibecoding.

**Source**: CTO Governance Audit GAP 6, CTO Decision 3 (code tasks only)

#### Phase 1 — Gate function (Day 2)

File: `internal/governance/design_gate.go` (NEW)

```go
// DesignFirstGate checks if a code task has an approved spec before execution.
// CTO Decision 3: blocks code tasks ("implement", "build", etc.),
// allows ad-hoc questions ("how", "explain", "debug", "what", "why").
//
// Returns (pass, reason):
//   pass=true  → proceed to agent loop
//   pass=false → reason contains user-facing message
func DesignFirstGate(ctx context.Context, agentKey string, content string,
    specStore store.SpecStore) (bool, string)
```

Logic:
1. If `agentKey != "coder"` → pass (gate only applies to @coder)
2. If `isAdHocQuestion(content)` → pass (CTO Decision 3)
3. If `specStore == nil` → pass (graceful degradation, nil guard)
4. Query: `specStore.ListSpecs(ctx, SpecListOpts{Status: "approved", Limit: 1})`
5. If no approved spec in current context → block with message:
   > "Design-First Gate: No approved spec found. Please create a spec first using @pm /spec before delegating code tasks to @coder."
6. If approved spec exists → pass

`isAdHocQuestion()` heuristic:
```go
// Returns true if content starts with question patterns.
// CTO Decision 3: "how do I...", "explain...", "debug this...", "what is...", "why does..."
// CTO Sprint 12 Decision D3: added "can ", "should ", "is ", "does ", "could "
// to reduce false-positive blocking (R23 mitigation).
var adHocPrefixes = []string{
    "how ", "explain", "debug", "what ", "why ", "where ", "help ",
    "can ", "should ", "is ", "does ", "could ",
}
func isAdHocQuestion(content string) bool {
    lower := strings.ToLower(strings.TrimSpace(content))
    for _, prefix := range adHocPrefixes {
        if strings.HasPrefix(lower, prefix) {
            return true
        }
    }
    return strings.HasSuffix(lower, "?")
}
```

#### Phase 2 — Integration into gateway_consumer (Day 3)

File: `cmd/gateway_consumer.go` (MODIFY)

Insert after line 87 (`agents.Get(agentID)` check), before agent loop:

```go
// Sprint 12: Design-First Gate (CTO Governance Audit GAP 6)
if specStore != nil {
    if pass, reason := governance.DesignFirstGate(ctx, agentID, msg.Content, specStore); !pass {
        slog.Info("governance: design-first gate blocked",
            "agent", agentID, "channel", msg.Channel, "reason", reason)
        msgBus.PublishOutbound(bus.OutboundMessage{
            Channel:  msg.Channel,
            ChatID:   msg.ChatID,
            Content:  reason,
            Metadata: msg.Metadata,
        })
        return
    }
}
```

#### Phase 3 — Tests (Day 3)

File: `internal/governance/design_gate_test.go` (NEW)

Target: ~12 tests

| Test | Input | Expected |
|------|-------|----------|
| `TestDesignGate_NonCoder_Passes` | agentKey="pm" | pass=true |
| `TestDesignGate_CoderAdHocQuestion_Passes` | "how do I implement auth?" | pass=true |
| `TestDesignGate_CoderExplain_Passes` | "explain the routing logic" | pass=true |
| `TestDesignGate_CoderDebug_Passes` | "debug this error" | pass=true |
| `TestDesignGate_CoderQuestionMark_Passes` | "what does this function do?" | pass=true |
| `TestDesignGate_CoderCodeTask_NoSpec_Blocks` | "implement user auth" | pass=false, reason contains "Design-First" |
| `TestDesignGate_CoderBuildTask_NoSpec_Blocks` | "build the dashboard" | pass=false |
| `TestDesignGate_CoderCodeTask_WithSpec_Passes` | approved spec exists | pass=true |
| `TestDesignGate_NilSpecStore_Passes` | specStore=nil | pass=true (graceful) |
| `TestDesignGate_EmptyContent_Passes` | content="" | pass=true (no blocking on empty) |
| `TestDesignGate_TeamRouted_Coder` | team routing → coder as leader | gate still applies |

---

### T12-GOV-02: AllArtifactTypes SSOT (P2, 0.5 pts) — Day 3

**Source**: CTO-49 from Sprint 11 review

**Current**: `AllArtifactTypes` hardcoded in `internal/evidence/chain.go:13`

**Fix**:
1. Create `internal/store/evidence_types.go` with `AllArtifactTypes` constant
2. Update `internal/evidence/chain.go` to import from `store` package
3. Future consumers (T13-GOV-04 Gate Matrix) use same SSOT

```go
// internal/store/evidence_types.go (NEW)
package store

// AllArtifactTypes is the SSOT list of expected artifact types in a complete chain.
// CTO-49: extracted from evidence/chain.go to single source of truth.
var AllArtifactTypes = []string{"spec", "pr_gate", "test_run", "deploy"}
```

**Tests**: Verify `evidence.AllArtifactTypes` is removed, `store.AllArtifactTypes` used everywhere. Build clean.

---

### T12-SEC-01: Live Integration Pen Tests (P2, 1 pt) — Days 3-4

**Source**: CTO-51 from Sprint 11 review

Sprint 11 PT-01..PT-06 are structural tests (unit-test style, no running server). CTO wants live versions that hit a real server.

File: `internal/security/pentest_live_test.go` (NEW)

```go
//go:build integration
// +build integration

// Run with: go test ./internal/security/ -tags=integration -run=TestLivePen
```

| Test | Method | Requires |
|------|--------|----------|
| `TestLivePen_PT01_RLSBypass` | Direct DB query via test connection | PostgreSQL + RLS policies |
| `TestLivePen_PT02_CrossTenantAPI` | HTTP GET with wrong tenant token | Running server |
| `TestLivePen_PT03_SOULInjection` | HTTP POST with injection payload | Running server + agent |
| `TestLivePen_PT04_JWTForgery` | HTTP POST /msteams/webhook with forged JWT | Running server |
| `TestLivePen_PT05_SOULDriftBypass` | HTTP PATCH /agents/{id} with bad checksum | Running server |
| `TestLivePen_PT06_TokenExhaustion` | 100 rapid HTTP calls | Running server |

**Test data seeding**: Include `TestMain(m *testing.M)` that:
1. Connects to test database via `MTCLAW_TEST_DSN` env var
2. Seeds 2 test tenants (tenant-A, tenant-B) with sample specs + PR gate evaluations
3. Runs tests via `m.Run()`
4. Cleans up test data in `defer`

**CI integration**: `go test -tags=integration ./internal/security/` — runs only in CI with `MTCLAW_INTEGRATION=true`.

---

### T12-TEST-01: Deterministic SHA256 Test (P3, 0.5 pts) — Day 4

**Source**: CTO-52 from Sprint 11 review

File: `internal/audit/pdf_builder_test.go` (MODIFY — add test)

```go
func TestAuditTrailPDF_DeterministicSHA256(t *testing.T) {
    // Build PDF twice with identical inputs
    // Assert: SHA256 of both PDF byte slices are identical
    // This validates that maroto doesn't inject timestamps or random IDs
}
```

**Note**: If maroto injects non-deterministic metadata (timestamps, random IDs), the fix is to override or strip metadata before hashing. Document finding either way.

---

### T12-TEST-02: Benchmark Tests in CI (P3, 0.5 pts) — Day 4

**Source**: CTO-53 from Sprint 11 review

File: `internal/audit/pdf_builder_bench_test.go` (NEW)

```go
func BenchmarkAuditTrailPDF(b *testing.B) {
    // Benchmark PDF generation for a typical 3-node chain
    // Target: establish baseline, no regression threshold yet
}
```

File: `internal/evidence/chain_bench_test.go` (NEW)

```go
func BenchmarkBuildChain(b *testing.B) {
    // Benchmark chain query with mock store (in-memory)
    // Target: establish baseline
}
```

**CI**: Add `go test -bench=. -benchmem ./internal/audit/ ./internal/evidence/` to CI pipeline. Results logged, not gated (Sprint 12 = baseline).

---

### T12-OPS-01: Bot Framework URL Prefix Runbook (P3, 0.5 pts) — Day 5

**Source**: CTO-54 from Sprint 11 review

File: `docs/06-deploy/runbooks/RUNBOOK-BOT-FRAMEWORK-URL-PREFIX.md` (NEW)

Content:
- When Microsoft updates Bot Framework service URL prefixes
- How to update `allowedServiceURLPrefixes` in `extensions/msteams/channel.go:16-20`
- Verification steps: run PT-07 SSRF test suite
- Monitoring: log alert on `ValidateServiceURL() rejected` events
- Historical: current prefixes and their source documentation

---

### T12-PM-01: OaaS Readiness Assessment + G5 Evidence (P2, 2 pts) — Days 4-5

**Owner**: [@pm]

Outputs:
1. `docs/08-collaborate/G5-OAAS-READINESS.md` — assessment of remaining gaps before multi-tenant OaaS
2. Update G5 gate proposal (from T11-05) with Sprint 12 governance engine evidence
3. Sprint 13 scope proposal (T13-GOV-04 Gate Matrix, T13-GOV-05 Feedback Loop, T13-GOV-06 Vibecoding Index)

---

## 6. Definition of Done

| Check | Command / Method | Expected |
|-------|-----------------|---------|
| Build clean | `go build ./...` | 0 errors |
| All tests pass | `go test ./... -count=1` | >=420 PASS (~30 new) |
| Quality scorer works | spec with score <70 → rejected | QUALITY_REJECTED message |
| Quality scorer allows | spec with score >=70 → saved | spec ID returned |
| Design-First Gate blocks | `@coder implement X` with no spec | Block message |
| Design-First Gate allows | `@coder how do I X?` | Passes through |
| Design-First Gate allows | `@coder implement X` with approved spec | Passes through |
| AllArtifactTypes SSOT | `grep -r "AllArtifactTypes" internal/` | Only in store/evidence_types.go |
| Live pen tests | `go test -tags=integration ./internal/security/` | 6 PASS (when server running) |
| SHA256 determinism | `go test -run=Deterministic ./internal/audit/` | PASS |
| Benchmarks | `go test -bench=. ./internal/audit/ ./internal/evidence/` | Results logged |
| URL prefix runbook | `docs/06-deploy/runbooks/RUNBOOK-BOT-FRAMEWORK-URL-PREFIX.md` | Filed |
| OaaS readiness | `docs/08-collaborate/G5-OAAS-READINESS.md` | Filed |

---

## 7. CTO Issues — Sprint 12

| Issue | Priority | Source | Task |
|-------|----------|--------|------|
| CTO-49 | P2 | CTO S11 review | T12-GOV-02 (AllArtifactTypes SSOT) |
| CTO-50 | P2 | CTO S11 review | T12-P1 carry-forward (performance measurements) |
| CTO-51 | P2 | CTO S11 review | T12-SEC-01 (live pen tests) |
| CTO-52 | P3 | CTO S11 review | T12-TEST-01 (SHA256 determinism) |
| CTO-53 | P3 | CTO S11 review | T12-TEST-02 (benchmarks in CI) |
| CTO-54 | P3 | CTO S11 review | T12-OPS-01 (URL prefix runbook) |

**Governance Audit gaps addressed this sprint**:
- GAP 1 (HIGH → RESOLVED): T12-GOV-01 Spec Quality Scoring
- GAP 3 (MEDIUM → RESOLVED): Already fixed via T11-00C (CTO-48, PR Gate default→"pending")
- GAP 5 (LOW → MITIGATED): Already mitigated (25 SOUL behavioral tests, Sprint 9)
- GAP 6 (MEDIUM → RESOLVED): T12-GOV-03 Design-First Gate

**Post-Sprint 12 total: 4 of 6 gaps resolved (GAP 1, 3, 5, 6).**

**Remaining for Sprint 13**:
- GAP 2 (HIGH): T13-GOV-04 Gate-Artifact-Tier Matrix
- GAP 4 (HIGH): T13-GOV-05 Quality Validation Feedback Loop

---

## 8. Risk Register — Sprint 12

| # | Risk | Prob | Impact | Mitigation |
|---|------|------|--------|------------|
| R22 | Quality scorer too strict — blocks legitimate specs from PM SOUL | Med | High | Start at threshold 70 (CTO-approved); tune based on first 10 rejections. Log all rejections for review. |
| R23 | Design-First Gate false positives — blocks legitimate ad-hoc coding questions | Med | Med | Generous `isAdHocQuestion` heuristic (7 prefixes + question mark). Log blocked messages for review. |
| R24 | `json.RawMessage` parsing failures in quality scorer (malformed JSON from LLM) | Med | Low | Graceful degradation: parsing error → 0 points for that dimension (not a crash). Nil guards on all fields. |
| R25 | Live pen tests flaky due to server state | Med | Low | Use dedicated test tenant with seeded data. `//go:build integration` tag isolates from unit test suite. |
| R26 | maroto PDF not deterministic (timestamps in metadata) | Low | Low | If non-deterministic: strip metadata before hash comparison. Document as known limitation. |
| R27 | T11-04 performance measurements still pending (CONDITIONAL carry-forward) | Med | Med | [@pm] drives completion early Sprint 12. If server not available → escalate to [@devops]. |

---

## 9. Sprint 13 Preview (Quality Loop + Vibecoding Detection)

Entry criteria for Sprint 13:
- Sprint 12 COMPLETE (CTO score received)
- GAP 1 + GAP 6 verified resolved by CTO
- G5 gate proposal reviewed

Sprint 13 = Governance Feedback Loop + Vibecoding Index:

| Task | Priority | Points | Source |
|------|----------|--------|--------|
| T13-GOV-04: Gate-Artifact-Tier Matrix | P0 | 2 | CTO Audit GAP 2 |
| T13-GOV-05: Quality Validation Feedback Loop | P0 | 3 | CTO Audit GAP 4 |
| T13-GOV-06: Vibecoding Index (3 of 5 signals) | P1 | 2 | CTO roadmap |
| Quality threshold increase 70→75 | P2 | 0.5 | CTO Decision 2 clause |
| OaaS multi-tenant self-service | P2 | 3 | Deferred from Sprint 12 |

---

## References

| Document | Location |
|----------|----------|
| Sprint 11 Completion | `docs/04-build/SPRINT-011-COMPLETION.md` |
| Sprint 11 Handoff | `docs/04-build/SPRINT-011-CODER-HANDOFF.md` |
| CTO Governance Audit | `docs/09-govern/01-CTO-Reports/CTO-AUDIT-GOVERNANCE-ENGINE-SPRINT11.md` |
| ADR-008 (PDF Library) | `docs/02-design/01-ADRs/SPEC-0008-ADR-008-PDF-Library.md` |
| ADR-009 (Evidence Linking) | `docs/02-design/01-ADRs/SPEC-0009-ADR-009-Evidence-Linking-Schema.md` |
| Roadmap v2.7.0 | `docs/01-planning/roadmap.md` |
| G5 Gate Proposal | `docs/08-collaborate/G5-GATE-PROPOSAL-STRUCTURE.md` |
