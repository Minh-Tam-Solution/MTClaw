# Sprint 7 — Coder Handoff

**Sprint**: 7 — Rail #1 Spec Factory Full + Retrieval Evidence (Context Drift Layer C)
**From**: [@pm] (plan) + [@architect] (SAD Section 8, Layer C design)
**To**: [@coder]
**Date**: 2026-03-04
**Predecessor**: Sprint 6 ✅ (CTO 8.0/10 APPROVED, 3 fixes applied: CTO-11/12/13)
**Points**: ~13 (5 days)
**Framework**: SDLC 6.1.1 — STANDARD tier

---

## What's Already Done (Sprint 6 Deliverables)

All Sprint 6 code is committed and verified (`go vet` + `go build` + 33 tests PASS):

| Deliverable | Files | Status |
|-------------|-------|--------|
| SOUL-Aware RAG Integration (Layer B) | `internal/rag/client.go` + `gateway_consumer.go` | ✅ |
| Team mention routing + charters | `gateway_consumer.go` (agent-first, team-second) | ✅ |
| `/teams` command | `commands.go` | ✅ |
| Cost guardrails (daily limit) | `gateway_consumer.go` + `tracing_store.go` | ✅ (CTO-11 fixed) |
| RAG `Metadata.Score` cleanup | `internal/rag/client.go` | ✅ (CTO-12 fixed) |
| Team cache optimization | `gateway_consumer.go` | ✅ (CTO-13 fixed) |

**Sprint 4 deliverables** (Sprint 7 builds on these):
- `/spec` command handler → PM SOUL via skill routing (`commands.go:98-135`)
- spec-factory SKILL.md (`docs/08-collaborate/skills/spec-factory/SKILL.md`) — prototype schema v0.1.0
- Context Anchoring Layer A → `gateway_consumer.go` (ExtraPrompt injection)
- Evidence metadata → TraceName + TraceTags on traces table

**Current `/spec` flow**:
```
User: /spec Create login feature
  → Telegram commands.go: publishes InboundMessage{AgentID:"pm", Metadata:{command:"spec"}}
  → gateway_consumer.go: routes to PM SOUL (skill-based)
  → PM SOUL: reads SKILL.md → generates spec JSON v0.1.0
  → Output: title, narrative, acceptance_criteria (strings), priority, effort
  → Evidence: trace with name="spec-factory", tags=["rail:spec-factory"]
```

---

## Sprint 7 Tasks — Implementation Guide

### Overview

| # | Task | Priority | Points | Days |
|---|------|----------|--------|------|
| 1 | Spec Factory v1.0 — full schema + migration | P0 | 3 | 1-2 |
| 2 | Evidence vault link (spec → trace) | P0 | 2 | 2-3 |
| 3 | Retrieval Evidence logging (Layer C) | P0 | 2 | 3 |
| 4 | Spec query APIs | P1 | 2 | 3-4 |
| 5 | Gateway consumer refactoring (CTO-14) | P1 | 2 | 4-5 |
| 6 | SOUL drift detection (ADR-004) | P1 | 2 | 4-5 |

**Scope adjustment vs roadmap v2.2.0**: SOUL behavioral test suite (80+ tests) DEFERRED to Sprint 8. Rationale: CTO-14 refactoring (extract gateway_consumer.go modules) is prerequisite for testability — behavioral tests are more effective after extraction. Sprint 7 adds drift detection (checksum); Sprint 8 adds behavioral validation.

---

### Task 1: Spec Factory v1.0 — Full Schema + Migration (Days 1-2, 3 pts)

**What**: Upgrade `/spec` from prototype (v0.1.0 — flat strings) to production (v1.0.0 — structured BDD + risk scoring). Create `governance_specs` table migration.

#### 1A. Database Migration: `migrations/000013_governance_specs.up.sql`

```sql
-- Sprint 7: Governance Specs — Rail #1 Spec Factory Full
-- Reference: SDLC Orchestrator GovernanceSpecification + SpecFunctionalRequirement models

CREATE TABLE IF NOT EXISTS governance_specs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id        VARCHAR(64) NOT NULL,       -- tenant isolation (RLS)
    spec_id         VARCHAR(16) NOT NULL,       -- SPEC-YYYY-NNNN format
    spec_version    VARCHAR(10) NOT NULL DEFAULT '1.0.0',
    title           VARCHAR(255) NOT NULL,
    narrative       JSONB NOT NULL,             -- {as_a, i_want, so_that}
    acceptance_criteria JSONB NOT NULL,          -- [{scenario, given, when, then}]
    bdd_scenarios   JSONB,                      -- Gherkin-formatted text array
    risks           JSONB,                      -- [{description, probability, impact, mitigation}]
    technical_requirements JSONB,               -- string array
    dependencies    JSONB,                      -- ["SPEC-YYYY-NNNN"] references
    priority        VARCHAR(4) NOT NULL DEFAULT 'P1',  -- P0|P1|P2|P3
    estimated_effort VARCHAR(4) DEFAULT 'M',   -- S|M|L|XL
    status          VARCHAR(16) NOT NULL DEFAULT 'draft', -- draft|review|approved|deprecated
    tier            VARCHAR(16) NOT NULL DEFAULT 'STANDARD',
    soul_author     VARCHAR(32) NOT NULL,       -- agent_key who generated it
    trace_id        UUID,                       -- FK to traces (evidence vault link)
    content_hash    VARCHAR(64),                -- SHA256 for drift detection
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (owner_id, spec_id)
);

-- Spec ID sequence (per-tenant per-year)
-- Generates SPEC-YYYY-NNNN via application logic, not DB sequence
-- (tenant isolation requires app-level counter)

-- Indexes
CREATE INDEX idx_governance_specs_owner ON governance_specs (owner_id);
CREATE INDEX idx_governance_specs_specid ON governance_specs (spec_id);
CREATE INDEX idx_governance_specs_status ON governance_specs (owner_id, status);
CREATE INDEX idx_governance_specs_created ON governance_specs (owner_id, created_at DESC);
CREATE INDEX idx_governance_specs_trace ON governance_specs (trace_id);

-- RLS policy (same pattern as agents table)
ALTER TABLE governance_specs ENABLE ROW LEVEL SECURITY;
CREATE POLICY governance_specs_tenant_isolation ON governance_specs
    USING (owner_id = current_setting('app.tenant_id', true));
```

Down migration: `000013_governance_specs.down.sql`
```sql
DROP POLICY IF EXISTS governance_specs_tenant_isolation ON governance_specs;
DROP TABLE IF EXISTS governance_specs;
```

#### 1B. Store Interface: `internal/store/spec_store.go`

```go
// SpecStore manages governance specs (Rail #1: Spec Factory).
// Reference: SDLC Orchestrator SpecificationService pattern.
type SpecStore interface {
    CreateSpec(ctx context.Context, spec *GovernanceSpec) error
    GetSpec(ctx context.Context, specID string) (*GovernanceSpec, error)
    ListSpecs(ctx context.Context, opts SpecListOpts) ([]GovernanceSpec, error)
    CountSpecs(ctx context.Context, opts SpecListOpts) (int, error)
    UpdateSpecStatus(ctx context.Context, specID string, status string) error
    NextSpecID(ctx context.Context, year int) (string, error)  // SPEC-YYYY-NNNN
}

type GovernanceSpec struct {
    ID                    uuid.UUID       `json:"id"`
    OwnerID               string          `json:"owner_id"`
    SpecID                string          `json:"spec_id"`         // SPEC-2026-0001
    SpecVersion           string          `json:"spec_version"`
    Title                 string          `json:"title"`
    Narrative             json.RawMessage `json:"narrative"`       // {as_a, i_want, so_that}
    AcceptanceCriteria    json.RawMessage `json:"acceptance_criteria"` // [{scenario,given,when,then}]
    BDDScenarios          json.RawMessage `json:"bdd_scenarios,omitempty"`
    Risks                 json.RawMessage `json:"risks,omitempty"`
    TechnicalRequirements json.RawMessage `json:"technical_requirements,omitempty"`
    Dependencies          json.RawMessage `json:"dependencies,omitempty"`
    Priority              string          `json:"priority"`
    EstimatedEffort       string          `json:"estimated_effort"`
    Status                string          `json:"status"`
    Tier                  string          `json:"tier"`
    SoulAuthor            string          `json:"soul_author"`
    TraceID               *uuid.UUID      `json:"trace_id,omitempty"`
    ContentHash           string          `json:"content_hash,omitempty"`
    CreatedAt             time.Time       `json:"created_at"`
    UpdatedAt             time.Time       `json:"updated_at"`
}

type SpecListOpts struct {
    Status string
    Since  *time.Time
    Limit  int
    Offset int
}
```

#### 1C. PG Implementation: `internal/store/pg/spec_store.go`

Standard CRUD. `NextSpecID` implementation:
```go
func (s *PGSpecStore) NextSpecID(ctx context.Context, year int) (string, error) {
    var maxSeq int
    prefix := fmt.Sprintf("SPEC-%d-", year)
    err := s.db.QueryRowContext(ctx,
        `SELECT COALESCE(MAX(CAST(split_part(spec_id, '-', 3) AS INT)), 0)
         FROM governance_specs WHERE spec_id LIKE $1`, prefix+"%",
    ).Scan(&maxSeq)
    if err != nil {
        return "", err
    }
    return fmt.Sprintf("SPEC-%d-%04d", year, maxSeq+1), nil
}
```

**Note**: Simple counter via MAX+1. Concurrent collision risk is LOW for MTS volume (~20 specs/month). If needed later: use `SELECT ... FOR UPDATE` or `pg_advisory_lock`.

**CTO-18 NOTE**: `NextSpecID()` queries `governance_specs` which has RLS policy on `owner_id`. RLS ensures tenant isolation automatically — but only if `SET LOCAL app.tenant_id` was called earlier in the transaction. This is currently safe because `consumeInboundMessages` sets tenant at entry via the RLS middleware. **Do NOT call `NextSpecID()` outside of a tenant-scoped context.** Add a comment in the implementation:
```go
// NextSpecID relies on RLS to scope counter per tenant.
// Caller MUST ensure SET LOCAL app.tenant_id was called before invoking.
```

#### 1D. Update SKILL.md to v1.0.0

Replace `docs/08-collaborate/skills/spec-factory/SKILL.md` output format:

```json
{
  "spec_version": "1.0.0",
  "title": "Short descriptive title",
  "narrative": {
    "as_a": "role",
    "i_want": "feature/capability",
    "so_that": "business value"
  },
  "acceptance_criteria": [
    {
      "scenario": "Happy path",
      "given": "precondition",
      "when": "action",
      "then": "expected result"
    }
  ],
  "bdd_scenarios": [
    "Feature: Feature Name\n  Scenario: Scenario Name\n    Given precondition\n    When action\n    Then expected result"
  ],
  "risks": [
    {
      "description": "risk description",
      "probability": "low|medium|high",
      "impact": "low|medium|high",
      "mitigation": "mitigation plan"
    }
  ],
  "technical_requirements": ["requirement 1", "requirement 2"],
  "dependencies": [],
  "priority": "P0|P1|P2|P3",
  "estimated_effort": "S|M|L|XL"
}
```

**Key changes from v0.1.0**:
- `acceptance_criteria`: strings → structured `{scenario, given, when, then}` objects
- NEW: `bdd_scenarios` — Gherkin-formatted text (reference: SDLC 6.1.1 Specification Standard BDD format)
- NEW: `risks` — probability × impact matrix with mitigation
- NEW: `technical_requirements` — technical constraints
- NEW: `dependencies` — cross-spec references

**BDD format reference** (from SDLC Enterprise Framework `05-Templates-Tools/01-Specification-Standard/`):
```gherkin
GIVEN [initial context]
  AND [additional context if needed]
WHEN [action or trigger]
  AND [additional action if needed]
THEN [expected outcome]
  AND [additional outcome if needed]
```

#### 1E. Spec Processing in gateway_consumer.go

After PM SOUL generates response, detect spec JSON in output and store to `governance_specs`:

```go
// In consumeInboundMessages, after receiving PM SOUL response:
if msg.Metadata["command"] == "spec" && specStore != nil {
    // Parse spec JSON from SOUL output
    spec, err := parseSpecFromOutput(output)
    if err == nil {
        specID, _ := specStore.NextSpecID(ctx, time.Now().Year())
        spec.SpecID = specID
        spec.OwnerID = tenantID
        spec.SoulAuthor = agentKey
        spec.TraceID = &traceID
        spec.ContentHash = sha256Hex(output)
        specStore.CreateSpec(ctx, spec)
    }
}
```

**Where**: Extract this logic into helper function `processSpecOutput()` — see Task 5 (CTO-14 refactoring).

---

### Task 2: Evidence Vault Link (Days 2-3, 2 pts)

**What**: Link spec to trace for audit trail. When user queries `/spec-detail SPEC-2026-0001`, show spec + linked trace metadata.

#### 2A. New Command: `/spec-list` and `/spec-detail`

Add to `internal/channels/telegram/commands.go`:

```go
case "/spec-list":
    // List recent specs for this tenant
    // Response: numbered list of spec_id + title + status

case "/spec-detail":
    // Detail view: full spec + linked trace info
    // Input: /spec-detail SPEC-2026-0001
    // Response: title, narrative, acceptance criteria, trace metadata (who, when, tokens)
```

#### 2B. Evidence Linking Pattern

```
governance_specs.trace_id  →  traces.id
                               ├── traces.agent_id  →  agents.agent_key (PM)
                               ├── traces.start_time  →  when spec was generated
                               ├── traces.total_input_tokens + total_output_tokens  →  cost
                               └── traces.metadata  →  {command:"spec", spec_id:"SPEC-2026-0001"}
```

**Trace metadata enrichment**: When creating spec, also update trace metadata:
```go
tracingStore.UpdateTrace(ctx, traceID, map[string]any{
    "metadata": json.RawMessage(`{"command":"spec","spec_id":"` + specID + `"}`),
})
```

This enables querying from either direction:
- Spec → Trace: `SELECT * FROM traces WHERE id = governance_specs.trace_id`
- Trace → Spec: `SELECT * FROM governance_specs WHERE trace_id = traces.id`

---

### Task 3: Retrieval Evidence Logging — Context Drift Layer C (Day 3, 2 pts)

**What**: Log detailed metadata for every RAG retrieval call. Completes the 3-layer Context Drift Prevention system.

**Architecture** (SAD Section 8):
```
Layer A (Sprint 4): Context Anchoring — session goal + SOUL identity in ExtraPrompt
Layer B (Sprint 6): Retrieval Intelligence — SOUL-Aware RAG collection routing
Layer C (Sprint 7): Evidence & Explainability — log ranking_reason per retrieval
```

**Reference**: EndiorBot ADR-015 (Retrieval Explainability), SDLC Orchestrator SpecValidationResult pattern.

#### 3A. RetrievalEvidence Structure

Add to `internal/rag/client.go`:

```go
// RetrievalEvidence captures metadata about a RAG retrieval for audit trail.
// Context Drift Layer C — Evidence & Explainability.
type RetrievalEvidence struct {
    Query         string  `json:"query"`
    Collection    string  `json:"collection"`
    ResultCount   int     `json:"result_count"`
    TopScore      float64 `json:"top_score"`
    RankingReason string  `json:"ranking_reason"` // exact_match|semantic_similar|soul_domain_boost|fallback
    SoulRole      string  `json:"soul_role"`
    TokenCount    int     `json:"token_count"`
    LatencyMS     int     `json:"latency_ms"`
}
```

**Ranking Reason Logic**:
```go
func classifyRankingReason(topScore float64, soulRole string, collection string) string {
    if topScore >= 0.95 {
        return "exact_match"
    }
    // Check if collection matches SOUL domain
    soulCollections := CollectionMap[soulRole]
    for _, c := range soulCollections {
        if c == collection {
            return "soul_domain_boost"
        }
    }
    if topScore >= 0.5 {
        return "semantic_similar"
    }
    return "fallback"
}
```

#### 3B. Logging in gateway_consumer.go

After RAG query completes, build evidence and attach to trace metadata:

```go
// After RAG query in consumeInboundMessages:
evidence := rag.RetrievalEvidence{
    Query:         ragQuery,
    Collection:    collection,
    ResultCount:   len(ragResp.Results),
    TopScore:      ragResp.Results[0].Score,  // if len > 0
    RankingReason: rag.ClassifyRankingReason(topScore, agentKey, collection),
    SoulRole:      agentKey,
    TokenCount:    ragResp.TokensUsed,
    LatencyMS:     int(ragDuration.Milliseconds()),
}

// Store in trace metadata (JSONB merge)
evidenceJSON, _ := json.Marshal(evidence)
tracingStore.UpdateTrace(ctx, traceID, map[string]any{
    "metadata": json.RawMessage(`{"retrieval_evidence":` + string(evidenceJSON) + `}`),
})
```

**Storage**: Uses existing traces.metadata (JSONB) column — no schema migration needed for Layer C. This matches SDLC Orchestrator pattern where validation results are stored in the parent entity's metadata.

#### 3C. Logging Considerations

- **Performance**: Evidence logging is async (after RAG response used). Does NOT add latency to user response.
- **Volume**: ~1 evidence record per user message with RAG. At MTS scale (10 users, ~50 messages/day) = ~50 records/day. Negligible storage.
- **Query pattern**: `SELECT metadata->'retrieval_evidence' FROM traces WHERE ...`

---

### Task 4: Spec Query APIs (Days 3-4, 2 pts)

**What**: HTTP API endpoints for spec listing and detail. Referenced by Telegram commands and future web dashboard.

**Reference**: SDLC Orchestrator `governance_specs.py` router pattern (4 endpoints).

#### 4A. API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/v1/governance/specs` | GET | List specs (paginated, filtered by status) |
| `/api/v1/governance/specs/{spec_id}` | GET | Get spec detail + linked trace |
| `/api/v1/governance/specs/search` | GET | Search specs by title (PostgreSQL full-text) |

**Note**: MTClaw uses GoClaw's built-in API server. Check if HTTP router is available in `cmd/` or if specs should be surfaced only via Telegram commands initially.

**Minimal viable approach** (Sprint 7): Telegram commands only (`/spec-list`, `/spec-detail`). HTTP API endpoints added in Sprint 8 when web dashboard is planned.

#### 4B. Telegram Command Handlers

Add to `commands.go`:

```go
case "/spec-list":
    specs, err := specStore.ListSpecs(ctx, store.SpecListOpts{Limit: 10})
    // Format: numbered list
    // "1. SPEC-2026-0001 — Login Feature [draft]"
    // "2. SPEC-2026-0002 — Payment Integration [approved]"

case "/spec-detail":
    specID := strings.TrimSpace(text[len("/spec-detail"):])
    spec, err := specStore.GetSpec(ctx, specID)
    // Format: full spec summary
    // Title, Narrative, Acceptance Criteria, Risks, Evidence link
```

Update `/help` text to include new commands.

---

### Task 5: Gateway Consumer Refactoring — CTO-14 (Days 4-5, 2 pts)

**What**: Extract `gateway_consumer.go` (993 lines) into focused modules. CTO flagged this as P2 (Sprint 7 target).

**Goal**: Better testability + maintainability. Each extracted module can be unit-tested independently.

#### 5A. Extraction Plan

| Current Location | Extract To | Functions |
|------------------|-----------|-----------|
| Lines 54-100 (mention routing) | `internal/routing/mention.go` | `ResolveMentionRoute()`, `resolveAgentKey()`, `resolveTeamLead()` |
| RAG injection block | `internal/rag/injector.go` | `InjectRAGContext()`, `BuildRAGPrompt()` |
| Cost guardrail block | `internal/cost/guardrails.go` | `CheckDailyLimit()`, `FormatWarning()` |
| Spec processing | `internal/governance/spec_processor.go` | `ProcessSpecOutput()`, `ParseSpecJSON()` |
| Team context block | `internal/routing/team_context.go` | `BuildTeamContext()` |

**Result**: `gateway_consumer.go` drops from ~993 lines to ~500-600 lines (orchestration only).

#### 5B. Refactoring Rules

1. **No behavior changes** — extract only, no new features in this task
2. **Function signatures preserve context** — all extracted functions receive `context.Context` + required stores
3. **Tests move with code** — if a test covers extracted logic, create corresponding test file
4. **Import cycle prevention** — extracted packages must NOT import `cmd/` package

#### 5C. Example: `internal/routing/mention.go`

```go
package routing

import (
    "context"
    "strings"

    "github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// ResolveMentionRoute resolves @mention to agent key or team lead.
// Returns (agentKey, teamName, teamID, isTeam).
// Resolution order: agent-first, team-second.
func ResolveMentionRoute(ctx context.Context, mention string, agentStore store.AgentStore, teamStore store.TeamStore) (agentKey string, teamName string, teamID *uuid.UUID, isTeam bool) {
    // ... extracted from gateway_consumer.go lines 54-100
}
```

#### 5D. Example: `internal/cost/guardrails.go`

```go
package cost

import (
    "context"
    "os"
    "strconv"
    "time"

    "github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// CheckDailyLimit checks if tenant has exceeded daily request limit.
// Returns (exceeded bool, count int, limit int, err error).
func CheckDailyLimit(ctx context.Context, tracingStore store.TracingStore) (bool, int, int, error) {
    today := time.Now().Truncate(24 * time.Hour)
    count, err := tracingStore.CountTraces(ctx, store.TraceListOpts{Since: &today})
    if err != nil {
        return false, 0, 0, err
    }
    limit := 500
    if envLimit := os.Getenv("MTCLAW_TENANT_DAILY_REQUEST_LIMIT"); envLimit != "" {
        if parsed, parseErr := strconv.Atoi(envLimit); parseErr == nil && parsed > 0 {
            limit = parsed
        }
    }
    return count >= limit, count, limit, nil
}
```

---

### Task 6: SOUL Drift Detection — ADR-004 (Days 4-5, 2 pts)

**What**: Detect when SOUL content in DB diverges from Git source files. Per ADR-004: checksum + version field.

**Reference**: EndiorBot `spec-snapshot-anchor.ts` (SHA256 hashing + drift status tracking).

#### 6A. Schema Addition — Migration `000014_soul_drift_detection.up.sql`

**Separate migration from 000013** (CTO-17: single-responsibility + safe rollback). 000013 = governance_specs (Task 1). 000014 = drift detection (Task 6). Independent DDL, independent rollback.

`migrations/000014_soul_drift_detection.up.sql`:
```sql
-- Sprint 7: SOUL Drift Detection (ADR-004)
-- Separate from 000013 (governance_specs) per CTO-17 directive.

ALTER TABLE agents ADD COLUMN IF NOT EXISTS content_checksum VARCHAR(64);
ALTER TABLE agents ADD COLUMN IF NOT EXISTS soul_version VARCHAR(10);

-- Drift event log
CREATE TABLE IF NOT EXISTS soul_drift_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id    UUID NOT NULL REFERENCES agents(id),
    agent_key   VARCHAR(32) NOT NULL,
    old_checksum VARCHAR(64),
    new_checksum VARCHAR(64),
    old_version  VARCHAR(10),
    new_version  VARCHAR(10),
    drift_type   VARCHAR(16) NOT NULL, -- content_changed | version_mismatch | missing
    detected_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_soul_drift_agent ON soul_drift_events (agent_id, detected_at DESC);
```

`migrations/000014_soul_drift_detection.down.sql`:
```sql
DROP INDEX IF EXISTS idx_soul_drift_agent;
DROP TABLE IF EXISTS soul_drift_events;
ALTER TABLE agents DROP COLUMN IF EXISTS content_checksum;
ALTER TABLE agents DROP COLUMN IF EXISTS soul_version;
```

#### 6B. Drift Detection Logic: `internal/souls/drift.go`

```go
package souls

import (
    "crypto/sha256"
    "fmt"
)

// ChecksumContent computes SHA256 of SOUL content for drift detection.
func ChecksumContent(content string) string {
    return fmt.Sprintf("%x", sha256.Sum256([]byte(content)))
}

// DriftStatus represents the result of a drift check.
type DriftStatus struct {
    AgentKey    string
    InSync      bool
    OldChecksum string
    NewChecksum string
    DriftType   string // "content_changed", "version_mismatch", "missing"
}

// CheckDrift compares stored checksum vs current content.
func CheckDrift(stored, current string) DriftStatus {
    storedHash := ChecksumContent(stored)
    currentHash := ChecksumContent(current)
    if storedHash == currentHash {
        return DriftStatus{InSync: true}
    }
    return DriftStatus{
        InSync:      false,
        OldChecksum: storedHash,
        NewChecksum: currentHash,
        DriftType:   "content_changed",
    }
}
```

#### 6C. Detection Trigger

On server startup (or SIGHUP reload): compare DB-stored checksums vs file contents.

```go
// In server init or reload handler:
// NOTE: tenantID must be in scope (e.g. from config or loop over known tenants).
// SET LOCAL app.tenant_id must precede any RLS-protected queries.
agents, _ := agentStore.List(ctx, tenantID) // CTO-16 FIX: ownerID required for RLS
for _, agent := range agents {
    files, _ := agentStore.GetAgentContextFiles(ctx, agent.ID) // CTO-15 FIX: correct method name
    for _, f := range files {
        if f.FileName == "SOUL.md" {
            status := souls.CheckDrift(f.Content, currentFileContent)
            if !status.InSync {
                slog.Warn("soul drift detected",
                    "agent_key", agent.AgentKey,
                    "drift_type", status.DriftType,
                )
                // Log to soul_drift_events table
            }
        }
    }
}
```

#### 6D. What This Does NOT Do (Sprint 8)

- Does NOT auto-update DB from Git files (manual re-seed required)
- Does NOT run behavioral tests on drift (Sprint 8: SOUL behavioral test suite)
- Does NOT block requests on drift (warning only — `slog.Warn`)

---

## Cross-Cutting Concerns

### Testing Strategy (Sprint 7)

| Test Type | Target | New Tests |
|-----------|--------|-----------|
| Unit tests | 75%+ coverage | spec_store CRUD, NextSpecID, drift checksum, ranking_reason classifier |
| Integration tests | +3 scenarios | spec creation end-to-end, evidence linking, daily limit with date filter |
| Existing tests | 33 tests PASS | Must not regress |

**New test file**: `internal/store/pg/spec_store_test.go` (unit tests for spec CRUD)
**New test file**: `internal/souls/drift_test.go` (checksum + drift detection)
**New test file**: `internal/rag/evidence_test.go` (ranking reason classification)

### Zero Mock Exception

RAG client tests (Sprint 6 established pattern): mock HTTP responses for `POST /api/v1/rag/query` in CI. Documented exception per Sprint 6 handoff — real RAG endpoint not available in CI.

### File Checklist

**New files**:
- [ ] `migrations/000013_governance_specs.up.sql` + `.down.sql` (Task 1)
- [ ] `migrations/000014_soul_drift_detection.up.sql` + `.down.sql` (Task 6 — CTO-17: separate migration)
- [ ] `internal/store/spec_store.go` (interface + types)
- [ ] `internal/store/pg/spec_store.go` (PG implementation)
- [ ] `internal/souls/drift.go` (drift detection)
- [ ] `internal/routing/mention.go` (extracted from gateway_consumer)
- [ ] `internal/rag/injector.go` (extracted RAG injection)
- [ ] `internal/cost/guardrails.go` (extracted cost limits)
- [ ] `internal/governance/spec_processor.go` (spec output parsing)

**Modified files**:
- [ ] `docs/08-collaborate/skills/spec-factory/SKILL.md` → v1.0.0 schema
- [ ] `cmd/gateway_consumer.go` → reduced to ~500-600 lines (orchestration)
- [ ] `internal/channels/telegram/commands.go` → `/spec-list`, `/spec-detail`
- [ ] `internal/rag/client.go` → `RetrievalEvidence` struct + `ClassifyRankingReason()`

### Environment Variables

No new env vars required. Existing:
- `MTCLAW_TENANT_DAILY_REQUEST_LIMIT` (Sprint 6, default 500)
- `MTCLAW_BFLOW_BASE_URL`, `MTCLAW_BFLOW_API_KEY`, `BFLOW_TENANT_ID` (Sprint 3)

---

## Reference Patterns

### From SDLC Orchestrator (adapted for Go)

| Pattern | Orchestrator (Python) | MTClaw (Go) |
|---------|----------------------|-------------|
| Spec model | `GovernanceSpecification` (7 related tables) | `GovernanceSpec` (1 table, JSONB fields) |
| Version tracking | `SpecVersion` table (immutable append) | `spec_version` field + `content_hash` (simpler) |
| Frontmatter validation | `SpecFrontmatterValidator` (JSON Schema) | Not needed Sprint 7 (AI generates, not human) |
| Risk scoring | `VibecodingIndexHistory` (5-signal, 0-100) | `risks` JSONB array (probability × impact matrix) |
| Evidence linking | `EvidenceVaultEntry` (S3 + SHA256 + 8-state) | `trace_id` FK to traces table (lighter weight) |
| Status workflow | `draft → review → approved → deprecated` | Same 4 states |

### From EndiorBot (adapted for Go)

| Pattern | EndiorBot (TypeScript) | MTClaw (Go) |
|---------|----------------------|-------------|
| Spec snapshot | `SpecSnapshotAnchor` (SHA256 + drift tracking) | `souls.CheckDrift()` (SHA256 checksum) |
| Drift policy | `threshold + action (warn/block/ignore)` | `slog.Warn` only (Sprint 7), block in Sprint 8 |
| BDD format | Gherkin in SOUL templates | Gherkin in SKILL.md + structured JSON |

### From SDLC Framework 6.1.1

| Reference | Location | Applied |
|-----------|----------|---------|
| Spec frontmatter schema | `05-Templates-Tools/01-Specification-Standard/spec-frontmatter-schema.json` | spec_id format: `SPEC-[0-9]{4}` |
| BDD requirement format | `SDLC-Specification-Standard.md` Section 3 | `GIVEN/WHEN/THEN` in acceptance_criteria |
| Tier classification | `spec-frontmatter-schema.json` | `tier` field on governance_specs |
| Status lifecycle | `spec-frontmatter-schema.json` | `DRAFT → REVIEW → APPROVED → DEPRECATED` |

---

## Success Criteria

| Metric | Target |
|--------|--------|
| `/spec` generates SPEC-2026-NNNN with BDD + risk | 100% of invocations |
| Spec ↔ trace evidence link | Bidirectional query works |
| Retrieval evidence logged | Every RAG call has ranking_reason |
| gateway_consumer.go | ≤650 lines (from 993) |
| SOUL drift detection | Detects checksum mismatch on startup |
| Tests | ≥38 total (33 existing + 5 new), all PASS |
| Build | `go vet` + `go build` clean |

---

## Risk Register

| Risk | Prob | Impact | Mitigation |
|------|------|--------|-----------|
| PM SOUL generates inconsistent BDD JSON | Medium | High | Clear examples in SKILL.md + regex post-processing |
| Spec ID collision on concurrent requests | Low | Medium | MTS volume ~20/month; add `FOR UPDATE` if needed |
| CTO-14 refactoring breaks existing tests | Medium | Medium | Run full test suite after each extraction step |
| SOUL drift false positives (whitespace changes) | Low | Low | Normalize whitespace before checksum |

---

## Sprint 8 Preview

Sprint 7 enables Sprint 8 (PR Gate ENFORCE + G4):
- **Full spec** enables PR Gate to reference spec_id in review
- **Evidence vault** enables G4 audit trail
- **Extracted modules** enable per-module testing + behavioral test suite
- **Drift detection** enables SOUL quality monitoring + behavioral validation
