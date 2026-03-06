# Sprint 12 — @coder Handoff

**Sprint**: 12 — Governance Engine: Spec Quality Scoring + Design-First Gate
**Date**: 2026-03-28
**From**: [@pm] + [@architect]
**To**: [@coder]
**CTO Approval**: Sprint 11 APPROVED 8.7/10 (2026-03-06)
**CTO Score (Sprint 11)**: 8.7/10
**Sprint Plan**: `docs/04-build/sprints/SPRINT-012-Governance-Engine.md`

---

## What's Already Done (Sprint 11 Close)

All Sprint 11 code committed and verified. Sprint 11 CTO score: 8.7/10 APPROVED.

| Deliverable | Status |
|-------------|--------|
| Evidence linking (ADR-009, migration 000017, recursive CTE) | Done (S11) |
| Security pen test (PT-01..PT-07, SSRF strongest) | Done (S11) |
| PDF audit trail (maroto v2, SOC2 5-section, SHA256 footer) | Done (S11) |
| Performance baseline template | Done (S11, CONDITIONAL — measurements TBD) |
| CTO-47 SSRF defense (ValidateServiceURL) | Done (S11) |
| CTO-48 PR Gate default→"pending" | Done (S11) |

**Open carry-forwards**:
- T11-04 performance measurements: CONDITIONAL — needs live server
- Azure AD live: pending [@devops]

---

## Sprint 12 Goal

**Transform MTClaw from governance framework → governance engine.**

Two critical insertion points identified by CTO Governance Audit:

1. **Quality gate on spec creation** (GAP 1) — reject low-quality specs before DB insert
2. **Design-first gate on @coder routing** (GAP 6) — require approved spec before code tasks

Plus 5 CTO carry-forward items (CTO-49 through CTO-54).

---

## MUST READ FIRST

1. **CTO Governance Audit**: `docs/09-govern/01-CTO-Reports/CTO-AUDIT-GOVERNANCE-ENGINE-SPRINT11.md`
   - Quality Scoring design: lines 138-152 (5 dimensions, threshold 70/100)
   - Root cause chain: lines 60-74 (where hooks go)

2. **Sprint 12 Plan**: `docs/04-build/sprints/SPRINT-012-Governance-Engine.md`
   - Architecture analysis: Section 3 (insertion points with line numbers)

---

## Execution Order (5 Days)

### Day 1: T12-GOV-01 Phase 1 — Quality Scorer

**Goal**: `internal/governance/spec_quality.go` — pure function, no DB dependency.

Files to create:
| File | Purpose | Status |
|------|---------|--------|
| `internal/governance/spec_quality.go` | `EvaluateSpecQuality()` — 5-dimension scorer | Create |
| `internal/governance/spec_quality_test.go` | ~15 tests (threshold edges, nil guards, partial specs) | Create |

**Key design decisions** (CTO-approved):

```
EvaluateSpecQuality(spec *store.GovernanceSpec) → QualityResult{Score, Reasons, Pass}

Scoring (100 points):
  Narrative (as_a + i_want + so_that)     25 pts — all >20 chars = full
  Acceptance criteria (len >= 2)           25 pts — each needs scenario + expected_result
  BDD scenarios (len >= 1)                20 pts — each needs given/when/then
  Risk assessment (len >= 1)              15 pts — each needs description + mitigation
  Technical requirements (>50 chars)      15 pts — non-null, non-empty

Threshold: < 70 → REJECT
```

**Spec struct fields** (from `internal/store/spec_store.go`):
- `Narrative json.RawMessage` — parse into `{as_a, i_want, so_that}`
- `AcceptanceCriteria json.RawMessage` — parse into `[]{ scenario, expected_result }`
- `BDDScenarios json.RawMessage` — parse into `[]{ given, when, then }`
- `Risks json.RawMessage` — parse into `[]{ description, mitigation }`
- `TechnicalRequirements json.RawMessage` — check non-nil, len > 50

**Critical**: All `json.RawMessage` fields can be `nil` — nil guard every parse. Parse error → 0 points for that dimension (not a panic).

### Day 2: T12-GOV-01 Phase 2 + T12-GOV-03 Phase 1

**Morning**: Integrate quality scorer into `spec_processor.go`.

File to modify:
| File | Change | Status |
|------|--------|--------|
| `internal/governance/spec_processor.go` | Insert `EvaluateSpecQuality()` at line 46-48 (after ContentHash, before CreateSpec) | Modify |

**Insertion point** (`spec_processor.go:46-48`):

> **L1 note**: Line numbers reference Sprint 11 close state. If T11 or other changes shifted lines, verify exact positions by searching for `sha256Hex(output)` and `specStore.CreateSpec` before inserting.

```go
// Current code (line 46-48):
spec.ContentHash = sha256Hex(output)
spec.Status = store.SpecStatusDraft

if err := specStore.CreateSpec(ctx, spec); err != nil {

// INSERT BETWEEN ContentHash and CreateSpec:
spec.ContentHash = sha256Hex(output)
spec.Status = store.SpecStatusDraft

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

**Return type change** (CTO Decision D2): `ProcessSpecOutput` now returns `SpecResult` struct (not string).

```go
// SpecResult — structured return type (CTO Decision D2, replaces string prefix)
type SpecResult struct {
    SpecID   string        // populated on success
    Rejected bool          // true if quality gate failed
    Quality  QualityResult // scoring details (always populated)
}
```

**Caller handling** (`cmd/gateway_consumer.go:410`): check `result.Rejected` — if true, format rejection message from `result.Quality.Score` and `result.Quality.Reasons`. Type-safe, no string parsing.

**Afternoon**: Start Design-First Gate function.

File to create:
| File | Purpose | Status |
|------|---------|--------|
| `internal/governance/design_gate.go` | `DesignFirstGate()` — pre-dispatch check | Create |

### Day 3: T12-GOV-03 Phase 2-3 + T12-GOV-02

**Morning**: Integrate Design-First Gate into gateway_consumer.go + write tests.

Files:
| File | Change | Status |
|------|--------|--------|
| `cmd/gateway_consumer.go` | Insert gate check at line ~87 (after agents.Get, before agent loop) | Modify |
| `internal/governance/design_gate_test.go` | ~12 tests | Create |

**Insertion point** (`gateway_consumer.go:87-90`):

```go
// Current code (line 87-90):
if _, err := agents.Get(agentID); err != nil {
    slog.Warn("inbound: agent not found", "agent", agentID, "channel", msg.Channel)
    return
}

// INSERT AFTER agents.Get check:
// Sprint 12: Design-First Gate (CTO Governance Audit GAP 6)
if specStore != nil {
    if pass, reason := governance.DesignFirstGate(ctx, agentID, msg.Content, specStore); !pass {
        slog.Info("governance: design-first gate blocked",
            "agent", agentID, "channel", msg.Channel)
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

**CTO Decision 3 — scope rules**:
- BLOCK: `@coder implement X`, `@coder build Y`, task delegation
- ALLOW: `@coder how do I...`, `@coder explain...`, `@coder debug this...`
- ALLOW: `@coder can you explain...`, `@coder should we use...`, `@coder is it possible...`
- ALLOW: `@pm`, `@architect`, any non-coder agent (gate ONLY for @coder)

**adHocPrefixes** (CTO Decision D3 — expanded for R23 mitigation):
```go
var adHocPrefixes = []string{
    "how ", "explain", "debug", "what ", "why ", "where ", "help ",
    "can ", "should ", "is ", "does ", "could ",  // CTO D3: added to reduce false positives
}
```

**Afternoon**: CTO-49 AllArtifactTypes SSOT extraction.

Files:
| File | Change | Status |
|------|--------|--------|
| `internal/store/evidence_types.go` | New file — `AllArtifactTypes` constant | Create |
| `internal/evidence/chain.go` | Remove local `AllArtifactTypes`, import from `store` | Modify |

### Day 4: T12-SEC-01 + T12-TEST-01 + T12-TEST-02

**Live integration pen tests** (CTO-51):
| File | Purpose | Status |
|------|---------|--------|
| `internal/security/pentest_live_test.go` | PT-01..PT-06 live versions with `//go:build integration` | Create |

**Test data seeding** (CTO L2): Include `TestMain(m *testing.M)` that connects via `MTCLAW_TEST_DSN`, seeds 2 test tenants (tenant-A, tenant-B) with sample specs + PR gate evaluations, runs `m.Run()`, then cleans up in `defer`.

**SHA256 determinism test** (CTO-52):
| File | Change | Status |
|------|--------|--------|
| `internal/audit/pdf_builder_test.go` | Add `TestAuditTrailPDF_DeterministicSHA256` | Modify |

**Benchmark tests** (CTO-53):
| File | Purpose | Status |
|------|---------|--------|
| `internal/audit/pdf_builder_bench_test.go` | `BenchmarkAuditTrailPDF` | Create |
| `internal/evidence/chain_bench_test.go` | `BenchmarkBuildChain` | Create |

### Day 5: T12-OPS-01 + DoD verification

**[@pm] handles T12-OPS-01** (Bot Framework URL runbook) and **T12-PM-01** (OaaS readiness).

**[@coder] DoD checks**:
```bash
go build ./...                           # 0 errors
go test ./... -count=1                   # >=420 PASS
go test -bench=. ./internal/audit/ ./internal/evidence/  # Benchmarks logged
```

---

## Files Summary

### Create (8 files)

| File | Task | Purpose |
|------|------|---------|
| `internal/governance/spec_quality.go` | T12-GOV-01 | Quality scorer (5 dimensions, threshold 70) |
| `internal/governance/spec_quality_test.go` | T12-GOV-01 | ~15 tests |
| `internal/governance/design_gate.go` | T12-GOV-03 | Design-first gate function |
| `internal/governance/design_gate_test.go` | T12-GOV-03 | ~12 tests |
| `internal/store/evidence_types.go` | T12-GOV-02 | AllArtifactTypes SSOT (CTO-49) |
| `internal/security/pentest_live_test.go` | T12-SEC-01 | Live integration pen tests (CTO-51) |
| `internal/audit/pdf_builder_bench_test.go` | T12-TEST-02 | PDF benchmark (CTO-53) |
| `internal/evidence/chain_bench_test.go` | T12-TEST-02 | Chain benchmark (CTO-53) |

### Modify (4 files)

| File | Task | Change |
|------|------|--------|
| `internal/governance/spec_processor.go` | T12-GOV-01 | Insert quality gate at line 46-48 |
| `cmd/gateway_consumer.go` | T12-GOV-03 | Insert design-first gate at line ~87 |
| `internal/evidence/chain.go` | T12-GOV-02 | Import AllArtifactTypes from store |
| `internal/audit/pdf_builder_test.go` | T12-TEST-01 | Add deterministic SHA256 test (CTO-52) |

### NOT Modified

| File | Reason |
|------|--------|
| `internal/store/spec_store.go` | GovernanceSpec struct unchanged — quality scorer reads existing fields |
| `internal/routing/mention.go` | Design gate lives in gateway_consumer, not routing |
| `migrations/*` | No schema changes in Sprint 12 |
| `internal/audit/pdf_builder.go` | Only tests added, no code changes |

---

## Zero Mock Exception

Same as Sprint 11: unit tests for `spec_quality.go` and `design_gate.go` construct `store.GovernanceSpec` values directly (no mock stores needed — pure functions). `design_gate_test.go` uses a minimal `SpecStore` interface stub for the `ListSpecs` query — documented CI exception per Sprint 11 handoff pattern.

---

## Key Code References

| What | File:Line | Why You Need It |
|------|-----------|-----------------|
| `ProcessSpecOutput` entry | `spec_processor.go:22` | Main integration point for T12-GOV-01 |
| `ContentHash` assignment | `spec_processor.go:46` | Insert quality gate AFTER this line |
| `CreateSpec` call | `spec_processor.go:49` | Insert quality gate BEFORE this line |
| `ParseSpecJSON` + raw struct | `spec_processor.go:61-112` | Shows JSON fields available for scoring |
| `GovernanceSpec` struct | `store/spec_store.go:20-42` | Field types for quality scorer |
| `agents.Get(agentID)` check | `gateway_consumer.go:87` | Insert design gate AFTER this line |
| `ResolveMention` call | `gateway_consumer.go:66` | Upstream of design gate — mention already resolved |
| `AllArtifactTypes` current | `evidence/chain.go:13` | Move to store/evidence_types.go |
| `ProcessSpecOutput` caller | `gateway_consumer.go:410` | Handle QUALITY_REJECTED return value |

---

## Verification Checklist

```bash
# 1. Build
go build ./...

# 2. All tests (unit)
go test ./... -count=1

# 3. Quality scorer specific
go test -v -run=TestQuality ./internal/governance/

# 4. Design gate specific
go test -v -run=TestDesignGate ./internal/governance/

# 5. Benchmarks
go test -bench=. -benchmem ./internal/audit/ ./internal/evidence/

# 6. SHA256 determinism
go test -v -run=Deterministic ./internal/audit/

# 7. Live pen tests (requires running server)
go test -tags=integration -v ./internal/security/ -run=TestLivePen

# 8. AllArtifactTypes SSOT check
grep -r "AllArtifactTypes" internal/ | grep -v "_test.go"
# Expected: only internal/store/evidence_types.go and internal/evidence/chain.go (import)
```
