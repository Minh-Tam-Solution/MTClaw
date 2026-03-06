---
spec_id: SPEC-0008
adr_id: ADR-008
title: PDF Library Selection for Audit Trail Export
status: APPROVED
date: 2026-03-22
author: "[@architect]"
reviewers: "[@cto], [@pm]"
approved_by: "@cto"
approval_date: 2026-03-06
sdlc_version: "6.1.1"
implements: "T11-03"
related_adrs: [ADR-009]
---

# ADR-008: PDF Library Selection for Audit Trail Export

**SDLC Stage**: 02-Design
**Status**: APPROVED — [@cto] 2026-03-06
**Date**: 2026-03-22

---

## Context

Sprint 11 introduces `GET /api/v1/spec/{spec_id}/audit-trail.pdf` — a compliance-ready PDF export of the full evidence chain for a governance spec. The PDF must include spec details, PR Gate evaluation results, and a chronological evidence timeline in a format compatible with SOC2/ISO27001 audits.

MTClaw is a pure Go project with zero CGO dependencies. The PDF library must:

1. Generate tabular reports (spec summary, PR gate results, timeline)
2. Be MIT or BSD licensed (no commercial or AGPL)
3. Require no CGO or external binaries (Docker constraint)
4. Be actively maintained (last release within 6 months)

---

## Problem Statement

> MTClaw needs to generate compliance-ready PDF audit trail reports from Go code. The library must produce clean tabular output without requiring CGO, external binaries, or commercial licenses.

---

## Options Evaluated

| # | Library | License | CGO | GitHub Stars | API Level | Last Release | Verdict |
|---|---------|---------|-----|-------------|-----------|-------------|---------|
| A | `johnfercher/maroto` v2 | MIT | No | 1.9K | High-level (grid/table) | 2024 Q4 | **RECOMMENDED** |
| B | `jung-kurt/gofpdf` | MIT | No | 4.1K | Low-level (coordinates) | 2023 Q2 | Mature but verbose |
| C | `unidoc/unipdf` | Commercial | No | — | High-level | Active | Paid license required |
| D | `nicholasgasior/gofpdf` | MIT | No | <100 | Low-level (fork) | 2023 | Stale fork |
| E | puppeteer / wkhtmltopdf | Various | Yes | — | HTML→PDF | Active | External binary, CGO |

---

## Decision

**Option A: `johnfercher/maroto` v2**

### Rationale

1. **High-level API**: Grid/table primitives map directly to audit report sections (header, spec summary, PR gate table, timeline). Estimated 60% less code than gofpdf for equivalent output.

2. **MIT license**: No licensing risk. Compatible with MTClaw's Apache-2.0-compatible stack.

3. **Zero CGO**: Pure Go. Works in Alpine/scratch Docker images without additional dependencies.

4. **Active maintenance**: v2 released 2024 with breaking API improvements (builder pattern, composable components). Active issue tracker.

5. **Table support**: Native `table.New()` with column widths, alignment, borders — exactly what compliance reports need.

### Trade-offs

- **Smaller community** than gofpdf (1.9K vs 4.1K stars) — acceptable given active maintenance
- **v2 API changes** between minor versions possible — mitigated by pinning exact version in `go.mod`
- **No complex layout** (multi-column flowing text) — not needed for tabular audit reports

---

## Implementation

### Dependency

```
go get github.com/johnfercher/maroto/v2@v2.x.x
```

Pin exact version in `go.mod` to prevent API drift (R18 risk mitigation).

### PDF Structure (SOC2/ISO27001 Compatible)

```
+--------------------------------------------------+
| MTClaw Audit Trail Report                         |
| Spec: SPEC-2026-XXXX  |  Tenant: mts             |
| Period: 2026-03-01 to 2026-03-22                  |
| Generated: 2026-03-22T14:30:00Z                   |
+--------------------------------------------------+
|                                                    |
| 1. SPECIFICATION                                   |
| ┌────────────┬──────────────────────────────┐     |
| │ Field      │ Value                        │     |
| ├────────────┼──────────────────────────────┤     |
| │ Spec ID    │ SPEC-2026-0042               │     |
| │ Title      │ User authentication flow     │     |
| │ Status     │ approved                     │     |
| │ Risk Score │ medium                       │     |
| │ Scenarios  │ 3 BDD (GIVEN/WHEN/THEN)      │     |
| └────────────┴──────────────────────────────┘     |
|                                                    |
| 2. PR GATE EVALUATIONS                             |
| ┌──────┬──────────┬────────┬──────┬──────────┐   |
| │ PR   │ Verdict  │ Rules  │ SHA  │ Date     │   |
| ├──────┼──────────┼────────┼──────┼──────────┤   |
| │ #42  │ PASS     │ 5/5    │ a1b2 │ 03-20    │   |
| └──────┴──────────┴────────┴──────┴──────────┘   |
|                                                    |
| 3. EVIDENCE TIMELINE                               |
| ┌──────────┬────────────┬──────┬──────────────┐  |
| │ Date     │ Event Type │ Actor│ Outcome      │  |
| ├──────────┼────────────┼──────┼──────────────┤  |
| │ 03-15    │ spec       │ @pm  │ created      │  |
| │ 03-18    │ pr_gate    │ @rev │ PASS         │  |
| └──────────┴────────────┴──────┴──────────────┘  |
|                                                    |
+--------------------------------------------------+
| SHA256: a1b2c3d4...  | MTClaw SDLC Gateway v0.11 |
+--------------------------------------------------+
```

### Code Location

- `internal/audit/pdf_builder.go` — `AuditTrailPDF(spec, chain) ([]byte, error)`
- `internal/audit/pdf_builder_test.go` — unit tests (PDF builds, SHA256 footer, empty chain error)
- HTTP handler in existing API router: `GET /api/v1/spec/{spec_id}/audit-trail.pdf`

---

## Consequences

### Positive

- Audit trail export enables SOC2/ISO27001 compliance demonstrations
- Pure Go: no Docker image bloat, no CGO build complexity
- High-level API reduces development time for T11-03 (estimated 1.5 days vs 3 days with gofpdf)

### Negative

- New dependency added to `go.mod` (maroto v2 + transitive deps)
- PDF customization limited to maroto's grid system (acceptable for tabular reports)

### Risks

| Risk | Probability | Mitigation |
|------|------------|------------|
| R18: maroto v2 API changes between minor versions | Low | Pin exact version in `go.mod` |
| PDF rendering edge cases (long text, Unicode Vietnamese) | Medium | Test with Vietnamese spec content in T11-03 |

---

## References

| Document | Location |
|----------|----------|
| Sprint 11 Plan | `docs/04-build/sprints/SPRINT-011-Hardening.md` |
| ADR-009 (Evidence Linking) | `docs/02-design/01-ADRs/SPEC-0009-ADR-009-Evidence-Linking-Schema.md` |
| maroto v2 docs | `https://github.com/johnfercher/maroto` |
