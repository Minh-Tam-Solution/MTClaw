---
gate: G4
metric: Weekly Active Users (WAU)
target: ≥7/10 MTS employees
window_start: 2026-03-17
window_end: 2026-03-31
status: IN_PROGRESS
author: "[@pm]"
framework: SDLC Enterprise Framework 6.1.1
---

# G4 WAU Tracking — MTS Internal Validation

**Gate**: G4 — Internal Validation
**Metric**: Weekly Active Users (WAU) ≥ 7 out of 10 MTS employees
**Observation Window**: 2026-03-17 → 2026-03-31 (2 weeks)
**G4 Approval Date**: 2026-03-17 ([@cto])
**Status**: IN PROGRESS — window ends 2026-03-31

---

## Context

G4 criteria #1 requires ≥7/10 MTS employees use MTClaw at least once per week. This metric
requires a live 2-week observation window starting from G4 approval date (2026-03-17).
At proposal time (Sprint 9 T9-05), 10/11 G4 criteria were met — WAU is the only outstanding item.

**Measurement source**:
```sql
-- WAU: unique active users per week (at least 1 session in the week)
SELECT
    date_trunc('week', created_at) AS week,
    COUNT(DISTINCT owner_id)       AS unique_active_users
FROM sessions
WHERE created_at >= '2026-03-17'
  AND tenant_id = 'mts'
GROUP BY 1
ORDER BY 1;
```

**Session count per user** (secondary metric — target ≥3/user/week):
```sql
SELECT
    owner_id,
    COUNT(*)                       AS sessions_this_week
FROM sessions
WHERE created_at >= '2026-03-17'
  AND tenant_id = 'mts'
  AND created_at < '2026-03-24'
GROUP BY owner_id
ORDER BY sessions_this_week DESC;
```

**Active SOULs in use** (at least 3 roles — target met by Sprint 9):
```sql
SELECT agent_id, COUNT(*) AS sessions
FROM sessions
WHERE tenant_id = 'mts'
  AND created_at >= '2026-03-17'
GROUP BY agent_id
ORDER BY sessions DESC;
```

---

## Weekly Log

### Week 1: 2026-03-17 → 2026-03-23

| Day | Active Users | New Users | Sessions Total | Notes |
|-----|-------------|-----------|----------------|-------|
| Mon 17 | — | — | — | G4 approved, observation starts |
| Tue 18 | — | — | — | Sprint 10 starts |
| Wed 19 | — | — | — | |
| Thu 20 | — | — | — | |
| Fri 21 | — | — | — | |
| Sat 22 | — | — | — | |
| Sun 23 | — | — | — | |
| **Week 1 total** | **?/10** | | | |

**Week 1 query** (run 2026-03-23):
```bash
# Run on MTClaw DB
psql $POSTGRES_DSN -c "
  SELECT COUNT(DISTINCT owner_id) AS wau,
         COUNT(*) AS total_sessions
  FROM sessions
  WHERE tenant_id = 'mts'
    AND created_at >= '2026-03-17'
    AND created_at < '2026-03-24';
"
```

---

### Week 2: 2026-03-24 → 2026-03-31

| Day | Active Users | New Users | Sessions Total | Notes |
|-----|-------------|-----------|----------------|-------|
| Mon 24 | — | — | — | |
| Tue 25 | — | — | — | |
| Wed 26 | — | — | — | |
| Thu 27 | — | — | — | |
| Fri 28 | — | — | — | |
| Sat 29 | — | — | — | |
| Sun 30 | — | — | — | |
| Mon 31 | — | — | — | Window end — final WAU report due |
| **Week 2 total** | **?/10** | | | |

---

## G4 WAU Verdict

| Measurement | Target | Actual | Status |
|-------------|--------|--------|--------|
| WAU Week 1 | ≥7/10 | — | Pending |
| WAU Week 2 | ≥7/10 | — | Pending |
| Sessions per user per week | ≥3 | — | Pending |
| Active SOULs (roles in use) | ≥3 | ✅ (pm, reviewer, coder confirmed Sprint 8) | Met |
| **G4 WAU PASS** | Both weeks ≥7/10 | — | **PENDING** |

**G4 PASS criteria**: WAU ≥7/10 in BOTH Week 1 AND Week 2.

---

## Adoption Intervention (if WAU < 7/10)

Per G4 Gate Proposal: if WAU < 7/10 after 2-week window → adoption intervention required before G5.

**Intervention options (if needed)**:

| Option | Action | Owner |
|--------|--------|-------|
| A | Direct Telegram onboarding session for non-adopters (30 min, hands-on) | [@pm] |
| B | SOUL demo: engineering team walkthrough of `/spec` + `/review` commands | [@coder] + [@pm] |
| C | Identify blockers per user (UX friction? SOUL mismatch? Telegram habit?) | [@pm] interviews |
| D | Add Telegram reminder: weekly digest from `pm` SOUL → sprint progress, pending specs | [@coder] |

**Note**: MTS Engineering team is the primary target (10 users). Sales + CS users are stretch targets — their WAU is tracked but not G4-blocking (G4 proposal scoped to Engineering WAU ≥7/10).

---

## References

| Document | Location |
|----------|----------|
| G4 Gate Proposal | `docs/08-collaborate/G4-GATE-PROPOSAL-SPRINT8.md` |
| Sessions table schema | `migrations/000003_create_sessions.up.sql` |
| MTS user list | Telegram analytics → `@cto` or `@devops` for user IDs |
| G4 evidence export | `GET /api/v1/evidence/export?format=json` |
