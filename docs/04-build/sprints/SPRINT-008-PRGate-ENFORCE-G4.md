# Sprint 8 — Rail #2 PR Gate ENFORCE + G4

**SDLC Stage**: 04-Build
**Version**: 1.0.0
**Date**: 2026-03-04
**Author**: [@pm] + [@architect]
**Sprint**: 8 of 10+
**Phase**: 2 (Governance)
**Framework**: SDLC 6.1.1 — STANDARD tier

---

## 1. Sprint Context

### Predecessor: Sprint 7 ✅ (CTO 8.0/10 APPROVED)

Sprint 7 delivered Rail #1 Spec Factory Full + Context Drift Layer C:

| Deliverable | Status | CTO Verdict |
|-------------|--------|-------------|
| Spec Factory v1.0 (migration + store + SKILL.md) | ✅ | GOOD, P0 bug fixed (CTO-19) |
| Evidence vault link (spec ↔ trace bidirectional) | ✅ | Fixed via CTO-20 (TraceID in RunResult) |
| Retrieval Evidence Layer C (ranking_reason + evidence) | ✅ | EXCELLENT |
| Spec Telegram commands (/spec_list, /spec_detail) | ✅ | GOOD |
| CTO-14 gateway_consumer refactoring (5 modules extracted) | ✅ | EXCELLENT |
| SOUL drift detection (ADR-004 checksum) | ✅ | GOOD |

**Sprint 7 CTO issues resolved before handoff**:
- CTO-19 (P0): OwnerID never set → FIXED (tenantID param added)
- CTO-20 (P0): traceID always nil → FIXED (TraceID added to RunResult)
- CTO-21 (P2): extractJSONBlock brace-counter limitation → NOTED in code comment
- CTO-22 (P2): RAG evidence in trace tags → DEFERRED to Sprint 8

### Entry Criteria

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Spec Factory v1.0 operational | ✅ | governance_specs table + CreateSpec/ListSpecs working |
| 3 RAG collections operational | ✅ | mts-engineering, mts-sales, mts-general (Sprint 6) |
| SOUL drift detection active | ✅ | ChecksumContent + CheckDrift (Sprint 7) |
| 3 governance rails defined | ✅ | Rail #1 (Spec Factory), Rail #2 (PR Gate), Rail #3 (Knowledge) |
| CTO-14 refactoring complete | ✅ | 5 extracted modules, gateway_consumer ~927 lines |
| All tests pass | ✅ | 249 tests across 18 packages |

---

## 2. Sprint Goal

**Upgrade PR Gate from WARNING → ENFORCE mode** and validate all 3 governance rails running together. Prepare G4 gate proposal (Validation Ready).

**Key outcomes**:
1. GitHub webhook receives PR events → reviewer SOUL evaluates → posts PR comment + sets status check
2. Merge blocked on policy violations (missing spec ref, no tests, security issues)
3. Full E2E test: spec created → PR reviewed → knowledge queried → evidence logged for all 3 rails
4. SOUL behavioral validation: 16 SOULs stay in character across 50+ turns
5. Evidence export for audit trail (JSON + CSV)

---

## 3. Task Overview

| # | Task | Priority | Points | Days | Owner |
|---|------|----------|--------|------|-------|
| 1 | PR Gate ENFORCE — GitHub webhook + status checks | P0 | 3 | 1-2 | [@coder] |
| 2 | pr_gate_evaluations table + evidence storage | P0 | 2 | 2 | [@coder] |
| 3 | Context Drift full E2E validation test | P0 | 2 | 3 | [@coder] |
| 4 | SOUL behavioral test suite (5 critical SOULs × 5 tests) | P0 | 1 | 3-4 | [@coder] |
| 5 | Evidence export API (JSON + CSV) + CTO-22 cleanup | P1 | 2 | 4 | [@coder] |
| 6 | G4 gate proposal (Validation Ready) | P0 | 2 | 5 | [@pm] |

**Total**: ~12 points, 5 days

---

## 4. Task Details

### Task 1: PR Gate ENFORCE — GitHub Webhook + Status Checks (P0, 3 pts)

**Goal**: Receive GitHub PR events via webhook, route to reviewer SOUL, post results as PR comment + set commit status check (pass/fail).

**Architecture** (from pr-gate-design.md Section 5):

```
GitHub PR Event (opened/synchronize)
    │
    ▼
internal/http/webhook_github.go
    │
    ├─ Verify webhook signature (X-Hub-Signature-256)
    ├─ Parse PR payload (owner, repo, number, head_sha, diff_url)
    ├─ PublishInbound({
    │    AgentID:  "reviewer",
    │    Content:  diff (fetched via GitHub API),
    │    Metadata: {
    │      command:    "review",
    │      rail:       "pr-gate",
    │      pr_url:     full PR URL,
    │      pr_number:  "42",
    │      head_sha:   "abc123",
    │      repo:       "owner/repo",
    │      mode:       "enforce"
    │    }
    │  })
    │
    ▼
gateway_consumer.go → reviewer SOUL → review report
    │
    ▼
internal/tools/github_pr.go
    │
    ├─ POST /repos/{owner}/{repo}/issues/{number}/comments (review comment)
    └─ POST /repos/{owner}/{repo}/statuses/{sha} (commit status: success/failure)
```

**Sprint 5 WARNING → Sprint 8 ENFORCE delta**:

| Component | Sprint 5 (WARNING) | Sprint 8 (ENFORCE) |
|-----------|-------------------|-------------------|
| Trigger | Telegram `/review` command | GitHub webhook (PR opened/updated) |
| Output | Telegram reply only | GitHub PR comment + commit status |
| Blocking | No | Yes (status check = failure blocks merge) |
| Policy rules | 5 soft warnings | 5 warnings + 3 hard blocks |

**New files**:
- `internal/http/webhook_github.go` — webhook handler with HMAC-SHA256 verification
- `internal/tools/github_pr.go` — GitHub REST API client (comments, status checks)
- `internal/http/webhook_github_test.go` — signature verification + payload parsing tests

**Config additions** (`config.yaml`):
```yaml
github:
  webhook_secret: "${GITHUB_WEBHOOK_SECRET}"
  app_token: "${GITHUB_APP_TOKEN}"  # PAT or GitHub App installation token
```

**PR Gate ENFORCE policy rules** (CTO tuned from WARNING data Sprint 5-7):

| Rule | Mode | Severity | Action |
|------|------|----------|--------|
| Missing spec reference (no `SPEC-` in PR title/body) | BLOCK | FAIL | Set status = failure |
| No test files (new `.go` without `_test.go`) | BLOCK | FAIL | Set status = failure |
| Security patterns (hardcoded secrets, SQL injection) | BLOCK | FAIL | Set status = failure |
| Low test coverage (<60%) | WARN | WARN | Comment only |
| Large diff (>500 lines) | WARN | WARN | Comment only |
| Missing docstrings | WARN | INFO | Comment only |
| TODO/FIXME added | WARN | INFO | Comment only |
| Missing CHANGELOG entry | WARN | INFO | Comment only |

---

### Task 2: pr_gate_evaluations Table + Evidence Storage (P0, 2 pts)

**Goal**: Persist every PR Gate evaluation with full evidence for audit trail.

**Migration 000015**:
```sql
CREATE TABLE pr_gate_evaluations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id        VARCHAR(64) NOT NULL,
    trace_id        UUID REFERENCES traces(id),
    pr_url          TEXT NOT NULL,
    pr_number       INTEGER NOT NULL,
    repo            VARCHAR(256) NOT NULL,
    head_sha        VARCHAR(64) NOT NULL,
    mode            VARCHAR(16) NOT NULL DEFAULT 'enforce',  -- 'warning' or 'enforce'
    verdict         VARCHAR(16) NOT NULL,                    -- 'pass', 'fail', 'warn'
    rules_evaluated JSONB NOT NULL DEFAULT '[]',             -- [{rule, severity, passed, detail}]
    review_comment  TEXT,                                    -- full review posted to GitHub
    soul_author     VARCHAR(64),                             -- reviewer SOUL agent key
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_pr_gate_owner ON pr_gate_evaluations (owner_id);
CREATE INDEX idx_pr_gate_repo ON pr_gate_evaluations (repo, pr_number);
CREATE INDEX idx_pr_gate_created ON pr_gate_evaluations (owner_id, created_at DESC);

-- RLS policy (same pattern as governance_specs)
ALTER TABLE pr_gate_evaluations ENABLE ROW LEVEL SECURITY;
CREATE POLICY pr_gate_evaluations_tenant ON pr_gate_evaluations
    USING (owner_id = current_setting('app.tenant_id', true));
```

**Store interface**: `internal/store/pr_gate_store.go`
- `CreateEvaluation(ctx, *PRGateEvaluation) error`
- `GetEvaluation(ctx, id uuid.UUID) (*PRGateEvaluation, error)`
- `ListEvaluations(ctx, PRGateFilter) ([]PRGateEvaluation, error)`

**Integration with gateway_consumer.go**: Similar to spec processing (Sprint 7 pattern) — after reviewer SOUL response, detect review JSON → persist to pr_gate_evaluations → post to GitHub.

---

### Task 3: Context Drift Full E2E Validation Test (P0, 2 pts)

**Goal**: Validate all 3 layers of Context Drift Prevention work together end-to-end.

**Test scenarios** (integration tests in `internal/integration/drift_e2e_test.go` — per CTO-25, avoid package proliferation):

| Test | Layer | Validation |
|------|-------|------------|
| SOUL identity after 50+ messages | A (Anchoring) | PM SOUL doesn't "become" a developer after many code questions |
| RAG returns domain-correct results | B (Retrieval) | `@sales` queries return sales SOPs, not engineering docs |
| Evidence logged for every retrieval | C (Evidence) | RetrievalEvidence struct populated for each RAG query |
| Cross-SOUL handoff preserves identity | A+B | Delegating from PM→Dev maintains correct SOUL personality |
| Spec output format stability | A+C | `/spec` produces valid SPEC-YYYY-NNNN after 20+ unrelated messages |

**Test approach**: These are **behavioral tests** that validate prompts and routing logic, not live LLM calls. Use deterministic mocks for AI-Platform responses, validate that:
- Context Anchoring prompt is always injected (Layer A)
- RAG routing selects correct collection per SOUL domain (Layer B)
- RetrievalEvidence is generated with ranking_reason (Layer C)

---

### Task 4: SOUL Behavioral Test Suite — 5 Critical SOULs (P0, 1 pt)

**Goal**: Validate 5 critical SOULs maintain character integrity. **Deferred from Sprint 7** (required CTO-14 refactoring as prerequisite for testability).

**Sprint 8 scope** (25 tests, `internal/souls/behavioral_test.go`) — CPO Condition 3: focus on governance-critical SOULs:

| SOUL | Tests | Validates |
|------|-------|-----------|
| pm | 5 | Spec generation format, BDD output, delegation to dev |
| reviewer | 5 | PR review format, policy rule application |
| coder | 5 | Code output format, Go conventions, error handling |
| dev-be | 5 | Backend-specific code patterns, Go conventions |
| sales | 5 | Vietnamese business language, proposal format |

**Deferred to Sprint 9** (55 tests, P1):

| SOUL | Tests | Validates |
|------|-------|-----------|
| architect | 5 | SAD references, ADR format, architecture principles |
| dev (fe, mobile, devops) | 15 | Domain-specific code patterns |
| sales-2 | 5 | Sales SOP, pricing format |
| cs (×2) | 10 | Customer support tone, escalation protocol |
| general | 5 | Vietnamese natural language, company knowledge |
| tester | 5 | Test plan format, coverage analysis |
| mentor | 5 | Teaching tone, step-by-step guidance |
| hr | 5 | Policy compliance, sensitive information handling |

**Test pattern**:
```go
func TestSOUL_PM_SpecFormat(t *testing.T) {
    soul := loadSOUL(t, "pm")
    // Verify SOUL.md contains required sections
    assert.Contains(t, soul.Content, "## Identity")
    assert.Contains(t, soul.Content, "## Delegation")
    // Verify checksum matches stored hash (drift detection)
    assert.Equal(t, souls.ChecksumContent(soul.Content), soul.ExpectedHash)
}
```

---

### Task 5: Evidence Export API + CTO-22 Cleanup (P1, 2 pts)

**Goal A**: Export governance evidence (specs + PR evaluations + traces) as JSON and CSV for audit.

**New endpoints**:
- `GET /api/v1/evidence/export?format=json&from=2026-03-01&to=2026-03-31` — full evidence export
- `GET /api/v1/evidence/export?format=csv&rail=spec-factory` — filtered CSV export

**Export schema** (JSON):
```json
{
  "export_date": "2026-03-04T10:00:00Z",
  "tenant_id": "mts",
  "period": { "from": "2026-03-01", "to": "2026-03-31" },
  "specs": [{ "spec_id": "SPEC-2026-0001", "title": "...", "trace_id": "..." }],
  "pr_evaluations": [{ "pr_url": "...", "verdict": "pass", "rules": [...] }],
  "traces": [{ "id": "...", "name": "spec-factory", "tags": [...] }]
}
```

**Goal B (CTO-22)**: Migrate RAG evidence from trace tags → `traces.metadata` JSONB field.

Current (Sprint 7): `traceTags = append(traceTags, "rag_evidence:"+string(evidenceJSON))`
Target (Sprint 8): Store in `traces.metadata` JSONB column (existing column, currently unused for RAG).

---

### Task 6: G4 Gate Proposal (P0, 2 pts)

**Goal**: Prepare G4 (Validation Ready) gate proposal for [@cto], [@cpo], [@ceo] review.

**G4 success criteria** (from roadmap):

| Metric | Target | Measurement |
|--------|--------|-------------|
| MTS WAU (Weekly Active Users) | ≥7/10 employees | Telegram analytics |
| Evidence capture rate | 100% for gated actions | `SELECT COUNT(*) FROM traces WHERE name IN ('spec-factory','pr-gate')` |
| Unit test coverage | ≥80% | `go test -cover` |
| P0/P1 bugs | 0 open | GitHub issues |
| 3 Rails operational | All running | Integration test pass |
| Context Drift validated | E2E test pass | Sprint 8 Task 3 result |
| PR Gate ENFORCE | Active on ≥1 repo | GitHub status checks configured |
| SOUL stability | 16/16 checksum match | Drift detection report |

**Gate proposal document**: `docs/00-foundation/G4-GATE-PROPOSAL.md`
- Format follows G0.1/G0.2 gate proposal pattern
- Includes Sprint 5-8 evidence summary
- Links to all CTO issue resolutions (CTO-1 through CTO-22)

---

## 5. Risk Register

| # | Risk | Probability | Impact | Mitigation |
|---|------|-------------|--------|------------|
| R1 | GitHub App/PAT setup delays (org approval needed) | Medium | High | Fallback: use PAT (personal access token) for Sprint 8, migrate to GitHub App in Sprint 9 |
| R2 | Private repo access for PR diff fetching | Medium | Medium | GitHub App token required for private repos; Sprint 5 WARNING mode used public repos |
| R3 | SOUL behavioral tests flaky (prompt sensitivity) | Low | Medium | Test structure/format, not LLM output content. Deterministic assertions only |
| R4 | Evidence export performance for large tenants | Low | Low | Paginate exports (1000 records per page), async for CSV generation |
| R5 | G4 gate criteria not met (WAU < 7) | Medium | High | Focus on MTS Engineering adoption (power users); Telegram group reminders |

---

## 6. Dependencies

| Dependency | Status | Sprint |
|------------|--------|--------|
| Reviewer SOUL seeded | ✅ | 3 |
| /review command (WARNING mode) | ✅ | 5 |
| PR Gate SKILL.md | ✅ | 5 |
| web_fetch tool | ✅ | GoClaw |
| Spec Factory v1.0 (Rail #1) | ✅ | 7 |
| SOUL-Aware RAG (Rail #3) | ✅ | 6 |
| CTO-14 refactored modules | ✅ | 7 |
| GitHub App/PAT | ✅ PAT provisioned | 8 (Task 1) |
| traces.metadata column | ✅ (existing, unused) | — |

---

## 7. Sprint 9 Preview

Sprint 9 (Full Governance + Hardening) builds on Sprint 8:
- Remaining 11 SOULs behavioral tests (55 tests, deferred from Sprint 8 Task 4 — CPO Condition 3)
- Full audit trail export (compliance reporting — JSON + CSV + PDF)
- Cross-rail evidence linking (SPEC-2026-001 → PR #42 → 95% coverage → deployed)
- SOUL quality regression suite (automated weekly CI)
- Performance tuning (cost query optimization, RAG latency <3s p95)
- Security penetration test (RLS bypass attempts, SOUL impersonation)

---

## References

- [Roadmap v2.3.0](../../01-planning/roadmap.md) — Sprint 8 section
- [PR Gate Design](../../02-design/pr-gate-design.md) — WARNING → ENFORCE architecture
- [Sprint 7 Coder Handoff](../SPRINT-007-CODER-HANDOFF.md) — Predecessor deliverables
- [System Architecture Document](../../02-design/system-architecture-document.md) — 5-layer architecture
- [Requirements FR-003](../../01-planning/requirements.md) — 3 Rails governance
- [Test Strategy](../../01-planning/test-strategy.md) — Testing pyramid + coverage targets
