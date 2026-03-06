# Sprint 8 — Coder Handoff

**Sprint**: 8 — Rail #2 PR Gate ENFORCE + G4 (Validation Ready)
**From**: [@pm] (plan) + [@architect] (webhook architecture, E2E validation design)
**To**: [@coder]
**Date**: 2026-03-04
**Predecessor**: Sprint 7 ✅ (CTO 8.0/10 APPROVED, CTO-19/20 P0 bugs fixed)
**Points**: ~12 (5 days) — CPO-adjusted: Task 4 reduced to 5 SOULs
**Framework**: SDLC 6.1.1 — STANDARD tier

---

## What's Already Done (Sprint 7 Deliverables)

All Sprint 7 code committed and verified (`go vet` + `go build` + 249 tests PASS):

| Deliverable | Files | Status |
|-------------|-------|--------|
| Spec Factory v1.0 (migration + store) | `migrations/000013_*`, `internal/store/spec_store.go`, `internal/store/pg/specs.go` | ✅ |
| Evidence vault link (spec ↔ trace) | `agent/loop.go` (TraceID in RunResult), `governance/spec_processor.go` | ✅ (CTO-20 fixed) |
| Retrieval Evidence Layer C | `internal/rag/evidence.go`, `internal/rag/injector.go` | ✅ |
| Spec Telegram commands | `internal/channels/telegram/commands_specs.go` | ✅ |
| CTO-14 gateway refactoring (5 modules) | `internal/routing/`, `internal/cost/`, `internal/governance/` | ✅ |
| SOUL drift detection | `internal/souls/drift.go`, `migrations/000014_*` | ✅ |

**Existing PR Gate infrastructure** (Sprint 5 WARNING mode):
- `/review` command handler in `commands.go`
- Reviewer SOUL seeded (migration 000009)
- `pr-gate/SKILL.md` in `docs/08-collaborate/skills/`
- TraceName="pr-gate", TraceTags=["rail:pr-gate","command:review"]
- `web_fetch` tool (fetches PR diff via URL)

---

## Sprint 8 Tasks — Implementation Guide

### Overview

| # | Task | Priority | Points | Days |
|---|------|----------|--------|------|
| 1 | PR Gate ENFORCE — GitHub webhook + status checks | P0 | 3 | 1-2 |
| 2 | pr_gate_evaluations table + evidence storage | P0 | 2 | 2 |
| 3 | Context Drift full E2E validation test | P0 | 2 | 3 |
| 4 | SOUL behavioral test suite (5 critical SOULs × 5 tests) | P0 | 1 | 3-4 |
| 5 | Evidence export API + CTO-22 cleanup | P1 | 2 | 4 |
| 6 | G4 gate proposal ([@pm] task, [@coder] skip) | P0 | 2 | 5 |

**[@coder] scope**: Tasks 1-5 (10 points, 4 days). Task 6 is [@pm] deliverable.

---

### Task 1: PR Gate ENFORCE — GitHub Webhook + Status Checks (P0, 3 pts, Days 1-2)

**Goal**: Receive GitHub webhook events for PRs, route to reviewer SOUL, post review comment + set commit status check on GitHub.

#### Subtask 1A: Config — GitHub settings

Add to `internal/config/config_channels.go`:

```go
type GitHubConfig struct {
    WebhookSecret string `json:"webhook_secret,omitempty"` // HMAC-SHA256 verification
    AppToken      string `json:"app_token,omitempty"`      // PAT or GitHub App installation token
    WebhookPath   string `json:"webhook_path,omitempty"`   // default "/github/webhook"
}
```

Add to `Config` struct (or `GatewayConfig`):
```go
GitHub GitHubConfig `json:"github,omitempty"`
```

Env var loading in `config_load.go`:
```go
if v := os.Getenv("GITHUB_WEBHOOK_SECRET"); v != "" {
    c.GitHub.WebhookSecret = v
}
if v := os.Getenv("GITHUB_APP_TOKEN"); v != "" {
    c.GitHub.AppToken = v
}
```

#### Subtask 1B: Webhook handler — `internal/http/webhook_github.go`

**Pattern**: Follow existing handler pattern (struct + RegisterRoutes + no auth middleware — webhook uses HMAC signature instead).

```go
// WebhookGitHubHandler handles GitHub webhook events for PR Gate ENFORCE.
type WebhookGitHubHandler struct {
    secret  string             // HMAC-SHA256 webhook secret
    msgBus  *bus.MessageBus    // to publish inbound messages
    channel string             // source channel identifier (e.g., "github")
}

func NewWebhookGitHubHandler(secret string, msgBus *bus.MessageBus) *WebhookGitHubHandler {
    return &WebhookGitHubHandler{secret: secret, msgBus: msgBus, channel: "github"}
}

func (h *WebhookGitHubHandler) RegisterRoutes(mux *http.ServeMux) {
    mux.HandleFunc("POST /github/webhook", h.handleWebhook)
}
```

**Webhook handler logic**:
1. Read body, verify `X-Hub-Signature-256` header against HMAC-SHA256(secret, body)
2. Parse `X-GitHub-Event` header — only process `pull_request` events
3. Parse PR payload: extract `action` (opened, synchronize, reopened), `pull_request.number`, `pull_request.html_url`, `pull_request.diff_url`, `pull_request.head.sha`, `repository.full_name`
4. Skip if action not in {opened, synchronize, reopened}
5. Publish InboundMessage to message bus:

```go
h.msgBus.PublishInbound(bus.InboundMessage{
    Channel: h.channel,
    ChatID:  fmt.Sprintf("%s#%d", repoFullName, prNumber),
    Content: prURL, // reviewer SOUL uses web_fetch to get diff
    Metadata: map[string]string{
        "command":   "review",
        "rail":      "pr-gate",
        "pr_url":    prURL,
        "pr_number": strconv.Itoa(prNumber),
        "head_sha":  headSHA,
        "repo":      repoFullName,
        "mode":      "enforce",
    },
})
```

**HMAC verification** (crypto/hmac, crypto/sha256):
```go
func verifySignature(secret string, body []byte, signature string) bool {
    // signature format: "sha256=<hex>"
    if !strings.HasPrefix(signature, "sha256=") {
        return false
    }
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(body)
    expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(expected), []byte(signature))
}
```

#### Subtask 1C: GitHub API client — `internal/tools/github_pr.go`

**Purpose**: Post PR review comments and set commit status checks.

```go
// GitHubClient posts PR comments and sets commit status checks.
type GitHubClient struct {
    token   string       // PAT or installation token
    baseURL string       // default "https://api.github.com"
    client  *http.Client
}

func NewGitHubClient(token string) *GitHubClient {
    return &GitHubClient{
        token:   token,
        baseURL: "https://api.github.com",
        client:  &http.Client{Timeout: 30 * time.Second},
    }
}

// PostComment posts a comment on a PR.
// GitHub API: POST /repos/{owner}/{repo}/issues/{number}/comments
func (c *GitHubClient) PostComment(ctx context.Context, repo string, prNumber int, body string) error

// SetCommitStatus sets a commit status check (success/failure/pending).
// GitHub API: POST /repos/{owner}/{repo}/statuses/{sha}
func (c *GitHubClient) SetCommitStatus(ctx context.Context, repo string, sha string, state string, description string) error
```

**Commit status states**: `"success"`, `"failure"`, `"pending"`
**Context label**: `"mtclaw/pr-gate"` (identifies the status check in GitHub UI)

#### Subtask 1D: Wire webhook + GitHubClient into gateway

In `internal/gateway/server.go`, add to `BuildMux()` (only if GitHub config present):

```go
if s.webhookGitHubHandler != nil {
    s.webhookGitHubHandler.RegisterRoutes(mux)
}
```

In `cmd/gateway.go`:
1. Construct webhook handler when `config.GitHub.WebhookSecret` is set
2. **CTO-23**: Construct `GitHubClient` ONCE at consumer startup, pass into `consumeInboundMessages()`:

```go
// cmd/gateway.go — construct once, not per-message (CTO-23)
var ghClient *tools.GitHubClient
if cfg.GitHub.AppToken != "" {
    ghClient = tools.NewGitHubClient(cfg.GitHub.AppToken)
}

consumeInboundMessages(ctx, msgBus, agents, cfg, sched, channelMgr,
    teamStore, tracingStore, ragClient, specStore, ghClient)
```

Update `consumeInboundMessages` signature to accept `ghClient *tools.GitHubClient`.

#### Subtask 1E: Post-review processing in gateway_consumer.go

**CTO-24**: PR metadata lives in `msg.Metadata` (inbound), NOT `outMeta` (outbound reply routing). Capture PR fields before the goroutine — same pattern as `specTenantID` (Sprint 7 CTO-19):

```go
// Before goroutine: capture PR-specific metadata from inbound msg (CTO-24)
prMode := msg.Metadata["mode"]
prRepo := msg.Metadata["repo"]
prNum, _ := strconv.Atoi(msg.Metadata["pr_number"])
prSHA := msg.Metadata["head_sha"]

// Goroutine signature: add prMode, prRepo string, prNum int, prSHA string
go func(channel, chatID, session, rID, command, agentKey, tenantID,
    prMode, prRepo string, prNum int, prSHA string, meta map[string]string) {

    // ... existing error handling + silent reply check ...

    // Sprint 8: PR Gate ENFORCE — post review to GitHub after reviewer SOUL responds.
    // ghClient constructed once at consumer startup (CTO-23), nil-safe check.
    if command == "review" && ghClient != nil && prMode == "enforce" {
        // Post review comment
        ghClient.PostComment(ctx, prRepo, prNum, outcome.Result.Content)

        // Parse verdict from reviewer output and set status
        verdict := governance.ParsePRVerdict(outcome.Result.Content)
        if verdict == "fail" {
            ghClient.SetCommitStatus(ctx, prRepo, prSHA, "failure", "PR Gate: policy violation found")
        } else {
            ghClient.SetCommitStatus(ctx, prRepo, prSHA, "success", "PR Gate: all checks passed")
        }
    }

    // ... existing outMsg publish ...

}(msg.Channel, msg.ChatID, sessionKey, runID, msgCommand, agentID,
    specTenantID, prMode, prRepo, prNum, prSHA, outMeta)
```

#### Tests — `internal/http/webhook_github_test.go`

| Test | Validates |
|------|-----------|
| TestVerifySignature_Valid | HMAC-SHA256 matches |
| TestVerifySignature_Invalid | Wrong signature rejected |
| TestVerifySignature_MalformedPrefix | Missing "sha256=" prefix rejected |
| TestHandleWebhook_PROpened | Publishes InboundMessage with correct metadata |
| TestHandleWebhook_PRSynchronize | Same processing as opened |
| TestHandleWebhook_IgnoredAction | "closed" action → 200 OK, no publish |
| TestHandleWebhook_NonPREvent | "push" event → 200 OK, no publish |

---

### Task 2: pr_gate_evaluations Table + Evidence Storage (P0, 2 pts, Day 2)

**Goal**: Persist every PR Gate evaluation for audit trail. Link to traces table.

#### Subtask 2A: Migration 000015

**File**: `migrations/000015_pr_gate_evaluations.up.sql`

```sql
-- Sprint 8: PR Gate ENFORCE — evaluation evidence storage.
CREATE TABLE pr_gate_evaluations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id        VARCHAR(64) NOT NULL,
    trace_id        UUID REFERENCES traces(id),
    pr_url          TEXT NOT NULL,
    pr_number       INTEGER NOT NULL,
    repo            VARCHAR(256) NOT NULL,
    head_sha        VARCHAR(64) NOT NULL,
    mode            VARCHAR(16) NOT NULL DEFAULT 'enforce',
    verdict         VARCHAR(16) NOT NULL,
    rules_evaluated JSONB NOT NULL DEFAULT '[]',
    review_comment  TEXT,
    soul_author     VARCHAR(64),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_pr_gate_owner ON pr_gate_evaluations (owner_id);
CREATE INDEX idx_pr_gate_repo ON pr_gate_evaluations (repo, pr_number);
CREATE INDEX idx_pr_gate_created ON pr_gate_evaluations (owner_id, created_at DESC);

-- RLS policy (same pattern as governance_specs — CTO-19 pattern)
ALTER TABLE pr_gate_evaluations ENABLE ROW LEVEL SECURITY;
CREATE POLICY pr_gate_evaluations_tenant ON pr_gate_evaluations
    USING (owner_id = current_setting('app.tenant_id', true));
```

**File**: `migrations/000015_pr_gate_evaluations.down.sql`

```sql
DROP POLICY IF EXISTS pr_gate_evaluations_tenant ON pr_gate_evaluations;
DROP INDEX IF EXISTS idx_pr_gate_created;
DROP INDEX IF EXISTS idx_pr_gate_repo;
DROP INDEX IF EXISTS idx_pr_gate_owner;
DROP TABLE IF EXISTS pr_gate_evaluations;
```

#### Subtask 2B: Store interface — `internal/store/pr_gate_store.go`

```go
type PRGateEvaluation struct {
    ID             uuid.UUID       `json:"id"`
    OwnerID        string          `json:"owner_id"`
    TraceID        *uuid.UUID      `json:"trace_id,omitempty"`
    PRURL          string          `json:"pr_url"`
    PRNumber       int             `json:"pr_number"`
    Repo           string          `json:"repo"`
    HeadSHA        string          `json:"head_sha"`
    Mode           string          `json:"mode"`
    Verdict        string          `json:"verdict"`
    RulesEvaluated json.RawMessage `json:"rules_evaluated"`
    ReviewComment  string          `json:"review_comment,omitempty"`
    SoulAuthor     string          `json:"soul_author,omitempty"`
    CreatedAt      time.Time       `json:"created_at"`
    UpdatedAt      time.Time       `json:"updated_at"`
}

type PRGateFilter struct {
    Repo     string
    PRNumber *int
    Verdict  string
    Limit    int
    Offset   int
}

type PRGateStore interface {
    CreateEvaluation(ctx context.Context, eval *PRGateEvaluation) error
    GetEvaluation(ctx context.Context, id uuid.UUID) (*PRGateEvaluation, error)
    ListEvaluations(ctx context.Context, filter PRGateFilter) ([]PRGateEvaluation, error)
}
```

#### Subtask 2C: PG implementation — `internal/store/pg/pr_gate.go`

Follow the same pattern as `pg/specs.go`:
- `NewPGPRGateStore(db *sql.DB) *PGPRGateStore`
- `CreateEvaluation`: INSERT with RETURNING
- `GetEvaluation`: SELECT by ID
- `ListEvaluations`: SELECT with filter + parameterized WHERE

#### Subtask 2D: Wire into Stores struct

Add to `internal/store/stores.go`:
```go
PRGate PRGateStore // nil in standalone mode — Sprint 8 Rail #2
```

Add to `internal/store/pg/factory.go`:
```go
PRGate: NewPGPRGateStore(db),
```

#### Subtask 2E: PR verdict processing — `internal/governance/pr_processor.go`

Similar to `spec_processor.go` (Sprint 7 pattern):

```go
// ProcessPRReview detects PR review verdict in reviewer SOUL output and persists to pr_gate_evaluations.
func ProcessPRReview(ctx context.Context, output string, prGateStore store.PRGateStore,
    agentKey string, tenantID string, traceID *uuid.UUID, meta map[string]string) (string, error)

// ParsePRVerdict extracts verdict (pass/fail/warn) from reviewer SOUL output.
// Looks for "## Summary" section with pass/fail indicators.
func ParsePRVerdict(output string) string
```

---

### Task 3: Context Drift Full E2E Validation Test (P0, 2 pts, Day 3)

**Goal**: Validate 3-layer Context Drift Prevention works end-to-end.

#### File: `internal/integration/drift_e2e_test.go`

**CTO-25**: Tests placed in `internal/integration/` (not a new `drift` package) to avoid package proliferation. Integration tests that cross multiple packages belong here.

**Approach**: These tests validate prompt construction and routing logic — NOT live LLM calls. All tests are deterministic (no flakiness from LLM output).

```go
package integration_test

// Test 1: Layer A — Context Anchoring always present
func TestLayerA_ContextAnchoringInjected(t *testing.T) {
    // Create a processNormalMessage-like flow
    // Verify ExtraPrompt contains "## Session Context" and "Note: You are operating as the **pm** SOUL"
    // After 50+ simulated messages, verify anchoring is STILL injected
}

// Test 2: Layer B — RAG routing per SOUL domain
func TestLayerB_RAGRoutingPerSOUL(t *testing.T) {
    // @sales query → rag.InjectRAGContext selects mts-sales collection
    // @dev query → rag.InjectRAGContext selects mts-engineering collection
    // @general query → rag.InjectRAGContext selects mts-general collection
    // Verify collection parameter in RAG client call
}

// Test 3: Layer C — Retrieval evidence generated
func TestLayerC_RetrievalEvidenceGenerated(t *testing.T) {
    // After rag.InjectRAGContext call
    // Verify ragInjection.Evidence is non-empty
    // Verify each evidence has Collection, Query, ranking_reason set
}

// Test 4: Cross-SOUL — Delegation preserves identity
func TestCrossSOUL_DelegationPreservesIdentity(t *testing.T) {
    // PM delegates to Dev → verify Dev's ExtraPrompt has Dev's anchoring
    // Not PM's anchoring leaking through
}

// Test 5: Spec output stability after many messages
func TestSpecOutputStability(t *testing.T) {
    // Verify ProcessSpecOutput still parses valid SPEC-YYYY-NNNN format
    // After a session with 20+ unrelated messages before /spec
}
```

**Test count**: 5 tests

---

### Task 4: SOUL Behavioral Test Suite — 5 Critical SOULs (P0, 1 pt, Days 3-4)

**Goal**: Validate 5 governance-critical SOULs maintain character integrity. Deferred from Sprint 7 — now that CTO-14 refactoring provides testable modules. **CPO Condition 3**: scoped to 5 SOULs for Sprint 8; remaining 11 deferred to Sprint 9.

#### File: `internal/souls/behavioral_test.go`

**Approach**: Test SOUL file structure, content, and checksum — NOT LLM behavioral output. These are **structural validation tests**.

**Sprint 8 SOULs** (5 × 5 = 25 tests):
- `pm` — Spec generation format, BDD output, delegation to dev
- `reviewer` — PR review format, policy rule application, pr-gate skill ref
- `coder` — Code output format, Go conventions, error handling
- `dev-be` — Backend-specific code patterns, Go conventions
- `sales` — Vietnamese business language, proposal format

```go
// TestAllSOULs_HaveRequiredSections validates the 5 critical SOULs have Identity, Delegation, Tools sections.
func TestAllSOULs_HaveRequiredSections(t *testing.T) {
    criticalSOULs := []string{"pm", "reviewer", "coder", "dev-be", "sales"}
    for _, name := range criticalSOULs {
        soul := loadSOULFile(t, "docs/08-collaborate/souls/", name)
        t.Run(soul.Name, func(t *testing.T) {
            assert.Contains(t, soul.Content, "## Identity")
            // Each SOUL must define its domain
        })
    }
}

// TestAllSOULs_ChecksumStable validates no untracked drift in SOUL files.
func TestAllSOULs_ChecksumStable(t *testing.T) {
    // Load SOULs from git, compute checksum, compare against stored
}

// TestSOUL_PM_SpecDelegation validates PM SOUL can delegate to dev for implementation.
func TestSOUL_PM_SpecDelegation(t *testing.T)

// TestSOUL_Reviewer_PRGateSkill validates reviewer SOUL has pr-gate skill reference.
func TestSOUL_Reviewer_PRGateSkill(t *testing.T)

// Per-SOUL domain validation (5 tests each for key SOULs)
func TestSOUL_Dev_GoConventions(t *testing.T)
func TestSOUL_Sales_VietnameseLanguage(t *testing.T)
```

**Test matrix**: 5 SOULs × 5 structural checks = 25 tests (Sprint 8)
**Deferred**: Remaining 11 SOULs × 5 checks = 55 tests (Sprint 9)
**Helper**: `loadSOULFile(t, dir, name)` — reads `SOUL-{name}.md` from directory

---

### Task 5: Evidence Export API + CTO-22 Cleanup (P1, 2 pts, Day 4)

#### Subtask 5A: Evidence export handler — `internal/http/evidence_export.go`

```go
type EvidenceExportHandler struct {
    specStore    store.SpecStore
    prGateStore  store.PRGateStore
    tracingStore store.TracingStore
    token        string
}

func (h *EvidenceExportHandler) RegisterRoutes(mux *http.ServeMux) {
    mux.HandleFunc("GET /v1/evidence/export", h.authMiddleware(h.handleExport))
}
```

**Query params**:
- `format`: `json` (default) or `csv`
- `rail`: `spec-factory`, `pr-gate`, or empty (all rails)
- `from`: ISO date (default: 30 days ago)
- `to`: ISO date (default: now)

**JSON response structure**:
```json
{
  "export_date": "2026-03-04T10:00:00Z",
  "period": {"from": "2026-03-01", "to": "2026-03-31"},
  "specs": [],
  "pr_evaluations": [],
  "traces": [],
  "stats": {
    "total_specs": 12,
    "total_pr_reviews": 5,
    "total_traces": 47,
    "pass_rate": 0.80
  }
}
```

**CSV**: Standard RFC 4180, one row per evidence item, flattened JSONB fields.
- Set `Content-Disposition: attachment; filename="evidence-export.csv"` header for browser downloads (CPO note)
- Set `Content-Type: text/csv; charset=utf-8` header

#### Subtask 5B: CTO-22 — Migrate RAG evidence to traces.metadata

**Current** (Sprint 7):
```go
traceTags = append(traceTags, "rag_evidence:"+string(evidenceJSON))
```

**Target** (Sprint 8):
```go
// Store evidence in trace metadata JSONB (CTO-22: not in tags)
if len(ragInjection.Evidence) > 0 {
    traceMetadata["rag_evidence"] = ragInjection.Evidence
}
```

**Changes**:
- `cmd/gateway_consumer.go`: Replace tag append with metadata map entry
- Add `traceMetadata map[string]interface{}` alongside existing `traceTags`
- Pass metadata to `RunRequest` (check if field exists, or add)

**Verify**: `traces.metadata` column already exists in schema (confirmed in `internal/store/tracing_store.go:62`).

---

## Cross-Cutting Concerns

### Environment Variables (New in Sprint 8)

| Variable | Required | Purpose |
|----------|----------|---------|
| `GITHUB_WEBHOOK_SECRET` | Yes (for webhook) | HMAC-SHA256 verification of GitHub webhook payloads |
| `GITHUB_APP_TOKEN` | Yes (for ENFORCE) | PAT or GitHub App installation token for PR comments + status |

### File Checklist

**New files** (10):

| File | Package | Purpose |
|------|---------|---------|
| `migrations/000015_pr_gate_evaluations.up.sql` | — | PR Gate evaluations table |
| `migrations/000015_pr_gate_evaluations.down.sql` | — | Rollback |
| `internal/store/pr_gate_store.go` | store | PRGateStore interface + PRGateEvaluation struct |
| `internal/store/pg/pr_gate.go` | pg | PGPRGateStore implementation |
| `internal/http/webhook_github.go` | http | GitHub webhook handler |
| `internal/http/webhook_github_test.go` | http | Webhook tests (7 tests) |
| `internal/tools/github_pr.go` | tools | GitHub API client (comments, status checks) |
| `internal/governance/pr_processor.go` | governance | PR verdict processing (like spec_processor.go) |
| `internal/http/evidence_export.go` | http | Evidence export API handler |
| `internal/integration/drift_e2e_test.go` | integration | Context Drift E2E tests (5 tests) — CTO-25 |

**Modified files** (7):

| File | Change |
|------|--------|
| `internal/config/config_channels.go` | Add GitHubConfig struct |
| `internal/config/config_load.go` | Load GITHUB_* env vars |
| `internal/store/stores.go` | Add `PRGate PRGateStore` field |
| `internal/store/pg/factory.go` | Add `PRGate: NewPGPRGateStore(db)` |
| `internal/gateway/server.go` | Register webhook + export handlers in BuildMux() |
| `cmd/gateway.go` | Construct GitHubClient once, pass to consumeInboundMessages (CTO-23) |
| `cmd/gateway_consumer.go` | Add ghClient param + PR metadata capture (CTO-24) + CTO-22 metadata migration |

**New test file** (behavioral):

| File | Tests |
|------|-------|
| `internal/souls/behavioral_test.go` | 25 tests (5 critical SOULs × 5 checks) — remaining 55 in Sprint 9 |

### Testing Strategy

| Category | Count | Source |
|----------|-------|--------|
| Webhook handler tests | 7 | `webhook_github_test.go` |
| Context Drift E2E tests | 5 | `integration/drift_e2e_test.go` |
| SOUL behavioral tests | 25 | `souls/behavioral_test.go` (5 SOULs — CPO Condition 3) |
| PR processor tests | ~5 | `governance/pr_processor_test.go` |
| **Sprint 8 new total** | **~42** | — |
| **Running total** | **~291** | 249 existing + 42 new |

### Zero Mock Exceptions

| Component | Why Mock | Documented |
|-----------|----------|------------|
| GitHub API calls in tests | Cannot POST to real GitHub in CI | Use `httptest.NewServer` for request/response validation |
| RAG client in E2E drift tests | Tests validate routing logic, not LLM output | Stub RAG response with deterministic content |

---

## Success Criteria

| Metric | Target | How to Measure |
|--------|--------|----------------|
| Webhook receives PR events | ✅ | curl test with signed payload |
| PR comment posted to GitHub | ✅ | Manual test on test repo |
| Commit status set (success/failure) | ✅ | GitHub UI shows mtclaw/pr-gate status |
| pr_gate_evaluations persisted | ✅ | `SELECT COUNT(*) FROM pr_gate_evaluations` |
| Context Drift E2E tests pass | ✅ | `go test ./internal/integration/ -run Drift` (CTO-25) |
| SOUL behavioral tests pass (25) | ✅ | `go test ./internal/souls/ -run Behavioral` (5 SOULs × 5) |
| Evidence export JSON works | ✅ | `curl /v1/evidence/export?format=json` |
| CTO-22 resolved | ✅ | RAG evidence in traces.metadata, not tags |
| All tests pass | ✅ | `go test ./...` (291+ tests, 0 failures) |
| go vet clean | ✅ | `go vet ./...` (no warnings) |

---

## Risk Register

| # | Risk | Prob | Impact | Mitigation |
|---|------|------|--------|------------|
| R1 | ~~GitHub PAT not provisioned in time~~ | ~~Med~~ | ~~High~~ | **RESOLVED**: PAT provisioned + webhook secret generated (`.env` updated). No longer a risk. |
| R2 | Webhook signature verification edge cases | Low | Med | Test with real GitHub webhook payloads (use webhook.site for debugging) |
| R3 | ~~SOUL behavioral tests too many~~ | ~~Med~~ | ~~Low~~ | **RESOLVED by CPO Condition 3**: Scoped to 5 SOULs (25 tests). Remaining 55 → Sprint 9. |
| R4 | traces.metadata JSONB field not populated correctly | Low | Low | Verify column exists in schema, write migration test |

---

## Sprint 9 Preview

Sprint 9 (Full Governance + Hardening) builds on Sprint 8:
- Full audit trail export (PDF generation for compliance)
- Cross-rail evidence linking (SPEC-2026-001 → PR #42 → 95% coverage)
- SOUL quality regression suite (remaining 55 behavioral tests + CI weekly run)
- Performance tuning (cost query optimization, RAG latency <3s p95)
- Security penetration test (RLS bypass, SOUL impersonation, webhook signature forgery)

---

## CPO Review Conditions — Resolution Log

| Condition | Status | Resolution |
|-----------|--------|------------|
| **C1**: GitHub PAT blocker | ✅ RESOLVED | PAT provisioned + webhook secret generated (`.env` updated) |
| **C2**: Task 3 path inconsistency | ✅ RESOLVED | Sprint Plan + Handoff aligned to `internal/integration/drift_e2e_test.go` (CTO-25) |
| **C3**: Task 4 scope too aggressive (80 tests / 1 day) | ✅ RESOLVED | Scoped to 5 critical SOULs (25 tests, 1 pt). Remaining 55 deferred to Sprint 9 |

**CPO Minor Notes applied**:
- CTO-23/24/25 registered in handoff doc (already labeled inline) — add to G4 evidence trail
- File count: 10 new + 7 modified = 17 total (reconciled in File Checklist above)
- CSV export: `Content-Disposition` header added to Subtask 5A spec
- WAU adoption: [@pm] to announce PR Gate ENFORCE launch via Telegram group on Day 5 (alongside G4 proposal)

**CTO Issue Tracker** (for G4 evidence):

| ID | Sprint | Severity | Status | Description |
|----|--------|----------|--------|-------------|
| CTO-19 | 7 | P0 | ✅ Fixed | OwnerID never set in ProcessSpecOutput |
| CTO-20 | 7 | P0 | ✅ Fixed | traceID always nil — added TraceID to RunResult |
| CTO-21 | 7 | P2 | 📝 Noted | extractJSONBlock brace-counter limitation (code comment) |
| CTO-22 | 7→8 | P2 | 🔄 Sprint 8 | RAG evidence tags → traces.metadata JSONB (Task 5B) |
| CTO-23 | 8 | P1 | 📋 Handoff | GitHubClient constructed once at startup, not in goroutine |
| CTO-24 | 8 | P2 | 📋 Handoff | PR metadata captured from msg.Metadata before goroutine |
| CTO-25 | 8 | P2 | 📋 Handoff | Drift E2E tests in integration package (avoid proliferation) |

---

## References

- [PR Gate Design](../../02-design/pr-gate-design.md) — WARNING → ENFORCE architecture (Section 5)
- [Sprint 7 Coder Handoff](SPRINT-007-CODER-HANDOFF.md) — Predecessor deliverables + CTO-14 modules
- [Roadmap v2.3.0](../../01-planning/roadmap.md) — Sprint 8 scope (lines 268-291)
- [System Architecture Document](../../02-design/system-architecture-document.md) — 5-layer architecture
- [Test Strategy](../../01-planning/test-strategy.md) — Testing pyramid + coverage targets
- Sprint 7 CTO Review: CTO-19 (OwnerID), CTO-20 (TraceID), CTO-21 (brace-counter), CTO-22 (RAG evidence tags → metadata)
- CPO Sprint 8 Review: 3 conditions APPROVED, RESOLVED (2026-03-04)
