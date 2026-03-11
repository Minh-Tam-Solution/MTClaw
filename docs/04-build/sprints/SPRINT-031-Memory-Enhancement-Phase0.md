---
sprint: 31
title: "Memory Enhancement — ADR + Phase 0 (Structured Flush Prompt)"
status: PARTIAL
start_date: 2026-03-11
end_date: 2026-03-14
lead: "@pm (plan + ADR) → @coder (Phase 0 implementation)"
framework: SDLC Enterprise Framework 6.1.2
adr: ADR-015-Memory-Enhancement-ClawVault-Port
---

# Sprint 31 — Memory Enhancement: ADR + Phase 0

## Sprint Goal

Write ADR-015 (ClawVault porting assessment — CTO APPROVED) and implement Phase 0: improve memory flush prompt to produce structured entries. Measure structured output rate to determine if Phase 1 (Fact Store) is needed.

---

## Context

ClawVault (TypeScript npm, v3.2.0) solves "context death" for AI agents. CTO approved porting 3 features in 4 gated phases. This sprint covers ADR + Phase 0 only — zero code risk, prompt change only.

**Reference**: `/home/dttai/.claude/plans/glowing-gliding-quill.md` (full assessment with 2 rounds of CTO review, 14 corrections applied)

---

## Part A: ADR-015 (@pm — documentation)

### A1: ClawVault Assessment ✅

| Task | Status |
|------|--------|
| Evaluate 10 ClawVault features against MTClaw architecture | ✅ Done |
| Identify 3 PORT, 1 DEFER, 6 SKIP decisions | ✅ Done |
| Self-critique with @architect (Vietnamese language, LLM cost, latency, alternatives) | ✅ Done |
| Map new components to 5-layer architecture diagram | ✅ Done |

### A2: CTO Review (2 Rounds) ✅

| Round | Changes | Status |
|-------|---------|--------|
| Round 1 | 7 mandatory (LLM-first extraction, explicit gates, facts-only injection, active-only query, flush-path trigger, confidence 0.5, bounded worker pool) | ✅ Applied |
| Round 2 | 7 corrections (embedding 1536, trigger scoping, worker pool scoping, Phase 3 Option A, explicit RLS, quantitative Phase 0 criteria, latency budget) | ✅ Applied |

### A3: ADR Written ✅

| File | Content | Status |
|------|---------|--------|
| `docs/02-design/01-ADRs/SPEC-0015-ADR-015-Memory-Enhancement-ClawVault-Port.md` | 7 sections: Problem, Decision (4 phases), Schema (2 tables), Architecture Fit, Self-Critique, Consequences, CTO Review Summary | ✅ Done |

---

## Part B: Phase 0 Implementation (@coder — prompt change)

### B1: Structured Memory Flush Prompt ✅

| File | Change | Status |
|------|--------|--------|
| `internal/agent/memoryflush.go` | Replace `DefaultMemoryFlushPrompt` with structured format requesting `[category]` tags + `entity/relation/value/context` fields | ✅ Done |
| `internal/agent/memoryflush.go` | Replace `DefaultMemoryFlushSystemPrompt` with `STRUCTURED OUTPUT REQUIRED` directive | ✅ Done |

**Before** (old prompt):
```
Pre-compaction memory flush. Store durable memories now (use memory/YYYY-MM-DD.md; create memory/ if needed).
IMPORTANT: If the file already exists, APPEND new content only and do not overwrite existing entries.
If nothing to store, reply with NO_REPLY.
```

**After** (new prompt — structured):
```
Pre-compaction memory flush. Store durable memories now (use memory/YYYY-MM-DD.md; create memory/ if needed).
IMPORTANT: If the file already exists, APPEND new content only and do not overwrite existing entries.

Write each memory entry using this structured format:

## [category] Short title
- **entity**: the subject (person, project, tool, concept)
- **relation**: what is being recorded (uses, decided, prefers, is, owns, learned)
- **value**: the specific fact or detail
- **context**: brief origin (which conversation, when, why)

Categories: decision, fact, preference, lesson, entity
```

### B2: Unit Tests ✅

| File | Tests | Status |
|------|-------|--------|
| `internal/agent/memoryflush_test.go` | `TestResolveMemoryFlushSettings_Defaults` — nil config returns enabled defaults | ✅ Pass |
| | `TestResolveMemoryFlushSettings_Disabled` — explicit disabled returns nil | ✅ Pass |
| | `TestResolveMemoryFlushSettings_CustomPrompt` — custom prompt overrides default | ✅ Pass |
| | `TestDefaultMemoryFlushPrompt_StructuredFormat` — prompt contains all 5 category tags + entity/relation/value | ✅ Pass |
| | `TestDefaultMemoryFlushSystemPrompt_StructuredRequirement` — system prompt contains STRUCTURED OUTPUT REQUIRED | ✅ Pass |

### B3: Build Verification ✅

| Check | Result | Status |
|-------|--------|--------|
| `make build` | Compiles cleanly (CGO_ENABLED=0) | ✅ Pass |
| `go test ./internal/agent/ -v` | All existing + new tests pass | ✅ Pass |

---

## Part C: Phase 0 Measurement (post-deploy — @pm + @cto)

### C1: Measurement Plan

After deploying the new prompt, wait for 10+ agent sessions to trigger memory flush (compaction boundary). Then measure:

```sql
-- Count structured vs unstructured entries in memory_documents
SELECT
  COUNT(*) AS total_entries,
  COUNT(*) FILTER (WHERE content LIKE '%## [decision]%' OR content LIKE '%## [fact]%'
    OR content LIKE '%## [preference]%' OR content LIKE '%## [lesson]%'
    OR content LIKE '%## [entity]%') AS structured_entries,
  ROUND(
    100.0 * COUNT(*) FILTER (WHERE content LIKE '%## [decision]%' OR content LIKE '%## [fact]%'
      OR content LIKE '%## [preference]%' OR content LIKE '%## [lesson]%'
      OR content LIKE '%## [entity]%') / NULLIF(COUNT(*), 0), 1
  ) AS structured_pct
FROM memory_documents
WHERE created_at > '2026-03-11';
```

### C2: Go/No-Go Gate (Phase 0 → Phase 1)

| Outcome | Structured % | Action |
|---------|-------------|--------|
| Prompt sufficient | ≥60% | Skip Phase 1 — prompt alone solves the problem |
| Gray zone | 40-60% | Iterate prompt once more, then re-measure |
| Proceed to Phase 1 | <40% | Prompt insufficient — build Fact Store (Sprint 32) |

**Measurement deadline**: If <10 sessions after 7 calendar days (2026-03-18), extend to 14 days max (2026-03-25), then decide with available data. Prevents indefinite wait.

### C3: Backup Measurement (filesystem verification)

If agents write to disk (standalone mode) instead of DB, the SQL query returns zero. Backup:
```bash
grep -rlc '## \[(decision\|fact\|preference\|lesson\|entity)\]' memory/
```

---

## Deliverables Summary

| # | Deliverable | Owner | Status |
|---|-------------|-------|--------|
| 1 | ADR-015 (CTO APPROVED, 14 corrections applied) | @pm + @architect | ✅ Done |
| 2 | Structured flush prompt (Phase 0) | @coder | ✅ Done |
| 3 | Unit tests (5 new tests) | @coder | ✅ Done |
| 4 | Phase 0 measurement | @pm + @cto | ⏳ Pending (need 10+ sessions) |
| 5 | Go/No-Go decision (Phase 0 → Phase 1) | @cto | ⏳ Pending |

---

## Risk Register

| Risk | Impact | Mitigation |
|------|--------|------------|
| Phase 0 solves everything (30% probability) | Phases 1-3 unnecessary — cheapest solution wins | Good outcome |
| LLM ignores structured format in flush prompt | Measurement shows <40% structured → proceed to Phase 1 | Gate catches this |
| Prompt too long → flush turn exceeds token budget | Monitor flush prompt token count (current: ~300 tokens) | Acceptable for 4096 max_tokens budget |

---

## Sprint 32 Outlook (conditional)

If Phase 0 gate says **proceed** (<40% structured):
- Sprint 32: Phase 1 — Fact Store (`memory_facts` table, `internal/facts/` package, migration 000021)
- Estimated: 5-7 days

If Phase 0 gate says **sufficient** (≥60% structured):
- Sprint 32: Pick next priority from backlog (not memory-related)
