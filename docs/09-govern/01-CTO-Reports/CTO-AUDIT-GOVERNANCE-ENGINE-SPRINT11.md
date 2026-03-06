---
spec_id: CTO-AUDIT-001
title: "CTO Audit: Governance Engine Quality — EndiorBot Sprint 80 Comparison"
status: APPROVED
date: 2026-03-06
author: "[@cto]"
reviewers: "[@pm], [@architect]"
sdlc_version: "6.1.1"
tier: STANDARD
stage: "09"
---

# CTO Audit: MTClaw Governance Engine — EndiorBot Sprint 80 Quality Gap Assessment

**SDLC Stage**: 09-Govern
**Date**: 2026-03-06
**Author**: [@cto]
**Reviewed by**: [@pm] + [@architect] (code-confirmed all 6 findings)

---

## Audit Question

> If a dev team uses MTClaw to build a project (e.g., open-pencil), would MTClaw's 3 governance rails prevent the quality problems EndiorBot Sprint 80 exposed?

## Verdict

**GOVERNANCE FRAMEWORK exists, but GOVERNANCE ENGINE missing.**

MTClaw has the right components (RAG, SOULs, evidence chains, gate verdicts) but lacks enforcement at critical decision points. 3 of 6 EndiorBot gaps remain HIGH risk.

---

## Audit Results: 6 EndiorBot Gaps vs MTClaw

| # | Gap | EndiorBot S80 | MTClaw Status | Risk | Root Cause |
|---|-----|--------------|---------------|------|------------|
| 1 | Generic doc generation | FAIL | **HIGH** — no quality validation | HIGH | `spec_processor.go` accepts any JSON, silent defaults mask missing data |
| 2 | Gate-artifact-tier matrix | FAIL | **HIGH** — tier unused by gates | HIGH | `Tier: "STANDARD"` hardcoded, gates don't consume it |
| 3 | Stub gate checkers | FAIL | **FIXED** — CTO-48 applied (2026-03-06) | ~~MEDIUM~~ RESOLVED | `pr_processor.go:56` now returns `"pending"` (T11-00C) |
| 4 | Quality validation loop | FAIL | **HIGH** — no feedback mechanism | HIGH | No `EvaluateSpecQuality()` exists anywhere |
| 5 | Header/frontmatter compliance | FAIL | **LOW** — SOUL tests pass | LOW | 25 behavioral tests validate SOUL YAML structure |
| 6 | Artifact path misalignment | FAIL | **MEDIUM** — DB-centric helps | MEDIUM | No file path enforcement for generated markdown |

### Code Evidence (PM+Architect Verified)

| Gap | File:Line | Evidence |
|-----|-----------|---------|
| GAP 1 | `spec_processor.go:85-95` | Only checks `spec_version != ""` and `title != ""`, defaults `Priority→P1`, `Effort→M` |
| GAP 2 | `spec_processor.go:109` | Hardcodes `Tier: "STANDARD"`, never consumed by gates |
| GAP 3 | `pr_processor.go:56-58` | **FIXED**: now `return "pending"` + 2 new tests (CTO-48, T11-00C, 2026-03-06) |
| GAP 4 | `internal/governance/` | No `EvaluateSpecQuality()` function exists |
| GAP 5 | `internal/souls/behavioral_test.go` | 25 tests validate SOUL YAML structure — mitigated |
| GAP 6 | `internal/routing/mention.go` | No pre-condition check before routing to @coder |

---

## Root Cause Analysis

```
Current enforcement chain (NO quality hooks):

  User message → ResolveMention() → agent Loop.Run() → LLM generates → ProcessSpecOutput() → DB insert
                      ↑                    ↑                              ↑
                No pre-conditions    No quality hooks              No quality gate

Required enforcement chain:

  User message → ResolveMention() → DesignFirstGate() → agent Loop.Run() → LLM generates → QualityScoring() → DB insert or REJECT
                                         ↑ NEW                                                   ↑ NEW
```

**Pattern**: MTClaw documents the right behavior in SOULs but doesn't technically enforce it. Guidance without enforcement creates false confidence.

---

## CTO Decisions

### Decision 1: T11-00C — PR Gate Default → "pending"

**APPROVED for Sprint 11.** 1-line fix, zero risk. The default-pass bug is a silent safety failure. Hardening sprint is the right placement.

### Decision 2: Quality Threshold — 70/100

**APPROVED at 70/100** with Sprint 13 tuning clause:
- 70 = "has all sections, each non-trivial" — fair minimum
- After T13-GOV-05 (feedback loop) ships, raise to 75

### Decision 3: Design-First Gate Scope — Code Tasks Only

**CODE TASKS ONLY**, not ad-hoc questions:
- **BLOCK**: Task delegation (`implement X`, `build Y`, `/spec` handoff to @coder)
- **ALLOW**: Ad-hoc questions (`how do I...`, `explain...`, `debug this...`)

### Decision 4: Sprint 12 Capacity — Governance Before OaaS

**Option (b): Defer OaaS tasks, prioritize governance.** Governance enforcement before OaaS launch is non-negotiable. Scaling bad quality to N tenants = scaling the problem.

Sprint 12 priority:
1. T12-GOV-01: Spec Quality Scoring (P0, 3pts)
2. T12-GOV-03: Design-First Gate (P1, 2pts)
3. OaaS tasks that fit remaining capacity

---

## Remediation Plan

### Sprint 11 Amendment

| Task | Description | Effort | Risk |
|------|-------------|--------|------|
| **T11-00C** | PR Gate default: `return "pass"` → `return "pending"` in `pr_processor.go:57` | 15 min | Near-zero |

### Sprint 12 — Governance Engine (P0)

| Task | Description | Effort | Files |
|------|-------------|--------|-------|
| **T12-GOV-01** | Spec Quality Scoring — 5 dimensions, threshold 70/100 | 3 pts | `internal/governance/spec_quality.go` (new), `spec_processor.go` (modify) |
| **T12-GOV-03** | Design-First Gate — pre-condition hook for @coder | 2 pts | `internal/governance/design_gate.go` (new), routing or agent loop (modify) |

### Sprint 13 — Quality Loop + Vibecoding

| Task | Description | Effort | Files |
|------|-------------|--------|-------|
| **T13-GOV-04** | Gate-Artifact-Tier Matrix enforcement | 2 pts | `internal/evidence/gate_matrix.go` (new) |
| **T13-GOV-05** | Quality validation feedback loop (retry with reasons) | 3 pts | `spec_processor.go`, migration 000018 |
| **T13-GOV-06** | Vibecoding Index (3 of 5 signals) | 2 pts | `internal/governance/vibecoding.go` (new) |

### Sprint 14+ — Sprint Governance + Authority

| Task | Description |
|------|-------------|
| **T14-GOV-07** | Sprint Governance (10 Golden Rules subset) |
| **T14-GOV-08** | SOUL Authority Enforcement (SE4H approve, SE4A execute) |

---

## Quality Scoring Design (T12-GOV-01)

```
EvaluateSpecQuality(spec *store.GovernanceSpec) → (score int, reasons []string)

Scoring (100 points):
  Narrative completeness    25 pts  as_a + i_want + so_that all >20 chars
  Acceptance criteria       25 pts  len(AC) >= 2, each has scenario + expected_result
  BDD scenarios             20 pts  len(BDD) >= 1, each has given/when/then
  Risk assessment           15 pts  len(Risks) >= 1, each has description + mitigation
  Technical requirements    15 pts  non-null, >50 chars

Threshold: < 70 → REJECT with reasons[]
Integration: spec_processor.go:48, after ContentHash, before CreateSpec
```

---

## References

| Document | Location |
|----------|----------|
| EndiorBot Sprint 80 | EndiorBot ADR-023, Sprint 80 acceptance criteria |
| SDLC 6.1.1 Anti-Vibecoding | `SDLC-Enterprise-Framework/02-Core-Methodology/Governance-Compliance/anti-vibecoding.yaml` |
| SDLC 6.1.1 Gates | `SDLC-Enterprise-Framework/02-Core-Methodology/Governance-Compliance/gates.yaml` |
| SDLC 6.1.1 Tier-Stage Matrix | `SDLC-Enterprise-Framework/02-Core-Methodology/Documentation-Standards/SDLC-Tier-Stage-Requirements.md` |
| Sprint 11 Handoff | `docs/04-build/SPRINT-011-CODER-HANDOFF.md` |
| Sprint 10 Completion | `docs/04-build/SPRINT-010-COMPLETION.md` |
