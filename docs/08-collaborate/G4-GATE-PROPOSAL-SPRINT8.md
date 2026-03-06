# G4 Gate Proposal — Internal Validation Ready

**Gate**: G4 — Internal Validation
**Project**: MTClaw — Governance-First Company Assistant Platform
**SDLC Tier**: STANDARD
**Framework**: SDLC Enterprise Framework 6.1.1
**Date**: 2026-03-17
**Author**: [@pm]
**Reviewers**: [@cto], [@cpo], [@ceo]
**Sprint Evidence**: Sprint 5 (PR Gate WARNING) through Sprint 9 (Channel Rationalization)

---

## Executive Summary

MTClaw has completed Phase 2 (Governance Hardening). All 3 governance rails are operational, PR Gate runs in ENFORCE mode with GitHub webhook integration, and 16 SOULs have complete behavioral test coverage (85 tests). The codebase has been rationalized to Telegram + Zalo only (Sprint 9). G4 requests approval to proceed to internal validation with MTS employees.

---

## G4 Success Criteria — Evidence

### 1. MTS Weekly Active Users

| Metric | Target | Current Status |
|--------|--------|---------------|
| WAU (MTS employees using MTClaw) | ≥7/10 | Pilot active — tracking via Telegram analytics |
| Active SOULs in use | ≥3 roles | pm, reviewer, coder confirmed active |
| Sessions per user per week | ≥3 | Tracked in `sessions` table |

**Evidence**: Telegram bot analytics + `sessions` table aggregation by owner_id. WAU measurement starts at G4 approval — requires 2-week observation window before G4 PASS.

---

### 2. 3 Governance Rails Operational

| Rail | Status | Sprint Delivered | Evidence |
|------|--------|-----------------|---------|
| Rail #1: Spec Factory | ✅ OPERATIONAL | Sprint 7 | `governance_specs` table + `/spec` command + SPEC-YYYY-NNNN format |
| Rail #2: PR Gate ENFORCE | ✅ OPERATIONAL | Sprint 8 | GitHub webhook + commit status checks + `pr_gate_evaluations` table |
| Rail #3: Knowledge & RAG | ✅ OPERATIONAL | Sprint 6 | 3 RAG collections (mts-engineering, mts-sales, mts-general) |

**Integration E2E**: Sprint 8 Task 3 — Context Drift E2E tests validate all 3 rails working together (5 tests, 16 subtests, all PASS).

---

### 3. Evidence Capture Rate

| Evidence Type | Target | Measurement |
|---------------|--------|-------------|
| Spec Factory traces | 100% | `SELECT COUNT(*) FROM traces WHERE name='spec-factory'` |
| PR Gate evaluations | 100% | `SELECT COUNT(*) FROM pr_gate_evaluations` |
| RAG retrieval evidence | 100% | `traces.metadata JSONB` — CTO-22 fixed Sprint 8 |

**Evidence export**: `GET /api/v1/evidence/export?format=json` operational (Sprint 8 Task 5). Audit trail covers full SPEC lifecycle + PR evaluation history.

---

### 4. Unit Test Coverage

| Metric | Target | Actual |
|--------|--------|--------|
| Unit test count | ≥290 (Sprint 8 baseline) | **350** (Sprint 9 — T9-03 adds 60 SOUL behavioral tests) |
| SOUL behavioral tests | 85 (all 17 governance SOULs) | **85** (25 Sprint 8 + 60 Sprint 9) |
| Build status | Clean | `go build ./...` 0 errors |
| Test pass rate | 100% | `go test ./...` 350/350 PASS |

**Coverage narrative**: Sprint 8 delivered 290 tests. Sprint 9 T9-03 added 60 SOUL behavioral tests (12 governance SOULs × 5 tests each). All tests deterministic — no flaky assertions, no LLM output dependencies.

---

### 5. P0/P1 Bug Status

| Issue | Status | Resolution |
|-------|--------|-----------|
| CTO-19 (P0): OwnerID never set | ✅ FIXED | Sprint 7 — tenantID param added to CreateSpec |
| CTO-20 (P0): traceID always nil | ✅ FIXED | Sprint 7 — TraceID added to RunResult |
| CTO-26 (P1): slog.Warn missing on scan error | ✅ FIXED | Sprint 8 post-review |
| CTO-27 (P1): GitHub creds not masked | ✅ FIXED | Sprint 8 post-review |

**Open P0/P1 bugs**: 0

**CTO issue log summary** (Sprint 1-9):

| Range | Count | Status |
|-------|-------|--------|
| CTO-01 to CTO-22 | 22 issues | All resolved Sprint 1-8 |
| CTO-23 to CTO-28 | 6 issues | All resolved Sprint 8 post-review |
| CTO-29 to CTO-32 | 4 notes | All verified/resolved Sprint 9 |
| **Total** | **32 CTO issues** | **0 open** |

---

### 6. Context Drift Validated

**Evidence**: Sprint 8 Task 3 — `internal/integration/drift_e2e_test.go`

| Test | Layer | Result |
|------|-------|--------|
| SOUL identity after 50+ messages | A (Anchoring) | PASS |
| RAG returns domain-correct results | B (Retrieval) | PASS |
| Evidence logged for every retrieval | C (Evidence) | PASS |
| Cross-SOUL handoff preserves identity | A+B | PASS |
| Spec output format stability | A+C | PASS |

All 5 scenarios PASS. Context Anchoring prompt injected deterministically. RAG routing selects correct collection per SOUL domain.

---

### 7. PR Gate ENFORCE Active

**Evidence**:
- `internal/http/webhook_github.go` — HMAC-SHA256 signature verification
- `internal/tools/github_pr.go` — PR comment + commit status API
- `pr_gate_evaluations` table with RLS (migration 000015)
- 3 BLOCK rules (missing spec ref, no tests, security patterns) → status = failure
- 5 WARN rules (low coverage, large diff, missing docstrings, TODOs, missing CHANGELOG) → comment only

**Configuration**: `github.webhook_secret` + `github.app_token` in config.yaml (secrets in env).

---

### 8. SOUL Stability — 17/17 Checksum Match

**Evidence**: `internal/souls/drift.go` + `ChecksumContent()` — ADR-004 drift detection.

| SOUL suite | Count | Behavioral tests | Drift detection |
|------------|-------|-----------------|----------------|
| Sprint 8 (governance-critical) | 5 SOULs | 25 tests | Checksum stored in `soul_checksums` table |
| Sprint 9 (12 governance SOULs) | 12 SOULs | 60 tests | Same checksum mechanism |
| **Total covered** | **17 SOULs** | **85 tests** | All PASS |

Note: `assistant` SOUL (category=router, sdlc_gates=[]) excluded from governance behavioral suite per [@pm] CTO-32 decision — utility dispatcher, not a governance role.

---

### 9. Codebase Rationalization (Sprint 9 Bonus)

| Metric | Value |
|--------|-------|
| Dead channel code removed | ~2,836 LOC (Feishu 2,060 + Discord 477 + WhatsApp 299) |
| Files deleted | 17 files (12 Feishu + 2 Discord + 2 WhatsApp + 1 onboard_feishu.go) |
| Dead references cleaned | ~354 references in internal/ + cmd/ |
| Active channels | Telegram + Zalo (ADR-006 APPROVED 2026-03-17) |
| MS Teams | Scaffold in `extensions/msteams/` — Sprint 10 implementation (ADR-007 APPROVED 2026-03-17) |

---

## Gate Checklist

```
[x] 3 Governance Rails operational (Spec Factory + PR Gate ENFORCE + RAG Knowledge)
[x] Evidence capture 100% (traces + pr_gate_evaluations + metadata JSONB)
[x] Unit tests: 350 PASS, 0 FAIL (go test ./... clean)
[x] SOUL behavioral tests: 85 tests covering 17 governance SOULs
[x] P0/P1 bugs: 0 open (CTO-01 through CTO-32, all resolved)
[x] Context Drift: 5/5 E2E scenarios PASS (3-layer validation)
[x] PR Gate ENFORCE: webhook + commit status + pr_gate_evaluations table
[x] SOUL drift detection: ChecksumContent() active, all checksums match
[x] Evidence export API: JSON + CSV operational
[x] Codebase clean: go build ./... 0 errors, dead channel code removed
[ ] MTS WAU ≥7/10: measurement starts at G4 approval (2-week observation window)
```

**10/11 criteria met at proposal time.** WAU criterion requires G4 approval to begin measurement window.

---

## G4 Approval Request

**Request**: [@cto] + [@cpo] + [@ceo] review and approve G4 to begin internal validation with MTS employees.

**Post-G4 milestones**:
- Week 1-2: WAU measurement with MTS Engineering team
- Week 2: NQH Phase 2 readiness review (Zalo channel, NQH SOUL variants)
- Sprint 10: MS Teams integration (ADR-007) + audit trail compliance export
- G5 (if applicable): External validation / OaaS preparation

**G4 BLOCKER if not met**: WAU < 7/10 after 2-week window → adoption intervention required before G5.

---

## References

| Document | Location |
|----------|----------|
| Problem Statement | `docs/00-foundation/problem-statement.md` |
| Business Case | `docs/00-foundation/business-case.md` |
| G0.1 Gate Proposal | `docs/00-foundation/G0.1-GATE-PROPOSAL.md` |
| G2 Gate Approval | `docs/00-foundation/G2-GATE-APPROVAL.md` |
| ADR-001 to ADR-006 | `docs/02-design/01-ADRs/` |
| PR Gate Design | `docs/02-design/pr-gate-design.md` |
| System Architecture | `docs/02-design/system-architecture-document.md` |
| Sprint 5 Plan (PR Gate WARNING) | `docs/04-build/sprints/SPRINT-005-MTS-Pilot-PRGate.md` |
| Sprint 8 Plan (PR Gate ENFORCE) | `docs/04-build/sprints/SPRINT-008-PRGate-ENFORCE-G4.md` |
| Sprint 9 Plan (Channel + SOUL) | `docs/04-build/sprints/SPRINT-009-Channel-Cleanup-SOUL-Complete.md` |
| Test Strategy | `docs/01-planning/test-strategy.md` |
