# Business Case — MTClaw

**SDLC Stage**: 00-Foundation
**Version**: 1.0.0
**Date**: 2026-03-02
**Author**: [@pm]

---

## Executive Summary

MTClaw transforms MTS-OpenClaw from a chat tool into a governance backbone for AI-first company operations. Investment is minimal (1 VPS, existing team), ROI is immediate (~393M VND/year productivity savings), and strategic value is high (governance rails as foundation for OaaS offering).

## Cost Analysis

### Operating Cost

| Item | Monthly | Annual |
|------|---------|--------|
| VPS (4 vCPU, 8GB RAM, 100GB SSD) | $70-140 | $840-1,680 |
| Bflow AI-Platform | $0 (internal) | $0 |
| Domain/SSL | ~$1 | ~$12 |
| **Total** | **$71-141** | **$852-1,692** |

### Development Cost

| Resource | Allocation | Duration |
|----------|-----------|----------|
| 1 Go developer (AI-assisted) | 80% | 10 sprints (~20 weeks) |
| [@pm] oversight | 20% | Continuous |
| [@cto] review | 5% | Gate reviews |

Note: Go competency gap mitigated per ADR-001 (AI Codex strategy, CTO review gate, 90-day eval).

## ROI Analysis

### Productivity Savings (Conservative)

| Team | Headcount | Hours saved/week/person | Value/year |
|------|-----------|------------------------|------------|
| Engineering | 4 | 3h (code review, specs, docs) | 156M VND |
| Sales | 3 | 2h (proposals, pricing, CRM) | 78M VND |
| CS | 2 | 2h (SOP lookup, ticket handling) | 52M VND |
| Back Office | 1 | 3h (contracts, HR, reporting) | 39M VND |
| **Total (10 MTS)** | | | **~325M VND** |

### Governance Value (Hard to Quantify)

| Value | Impact |
|-------|--------|
| Feature waste reduction (60% → <30%) | ~68M VND/year avoided rework |
| PR Gate quality improvement | Fewer production incidents |
| Evidence audit trail | Compliance readiness |
| **Total estimated** | **~393M VND/year** |

### Payback Period

- Operating cost: ~35M VND/year ($1,680 max)
- Savings: ~393M VND/year
- **Payback: < 1 month**

## Strategic Value

### Phase 1: MTS Internal (Sprint 1-8)
- 10 employees with AI-governance assistants
- 3 Rails running (Spec Factory + PR Gate + Knowledge)
- Foundation for multi-tenant expansion

### Phase 2: NQH Expansion (Sprint 6+, conditional on CEO decision)
- 100-150 NQH employees across F&B operations
- NQH-SOPs RAG collection (805 docs already indexed)
- Zalo channel for non-tech staff

### Phase 3: OaaS Foundation (Sprint 9+)
- Multi-tenant self-service
- Governance-as-a-Service offering
- Revenue potential from external tenants

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Go competency gap | Medium | Medium | AI Codex + CTO review gate + 90-day eval (ADR-001) |
| Low adoption | Low | High | Interview-validated needs (8/10 interested) |
| Bflow AI quality | Low | Medium | Fallback to generic + human escalation |
| Scope creep into Bflow/NQH-Bot territory | Medium | Medium | Clear positioning doc + CTO guard |

## Decision Requested

**G0.1 Gate**: Approve MTClaw as SDLC 6.1.1 STANDARD tier project with:
- GoClaw runtime (ported from MIT-licensed upstream, internal use only)
- 16 SOULs from Day 1
- Governance-first roadmap (3 Rails in 8 sprints)
- Bflow AI-Platform as single AI source
- **Not OSS** — proprietary internal platform for MTS/NQH

---

**References**:
- [Problem Statement](problem-statement.md)
- [Baseline Metrics](user-research/baseline-metrics.md)
- [ADR-001: GoClaw Adoption](../02-design/01-ADRs/SPEC-0001-ADR-001-GoClaw-Adoption.md)
