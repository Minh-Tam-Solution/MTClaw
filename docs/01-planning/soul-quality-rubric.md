# SOUL Quality Rubric — MTClaw

**SDLC Stage**: 01-Planning
**Version**: 1.0.0
**Date**: 2026-03-02
**Author**: [@pm] + [@researcher]
**Source**: EndiorBot Vibecoding Index + Evaluator-Optimizer (ADR-010) patterns, adapted for MTClaw

---

## Research Methodology

[@researcher]: Rubric derived from EndiorBot's 3 quality systems:
1. **Vibecoding Index** — penalty-based 0-100 scoring (lower = better)
2. **Quality Gates** — task-type minimum tier requirements
3. **Evaluator-Optimizer Score Card** — 5-dimensional response quality

Adapted for MTClaw context: 16 SOULs, Telegram channel, Bflow AI-Platform backend.

---

## 1. SOUL Response Quality Score Card

5 dimensions, 0-100 weighted score. **Higher is better** (inverted from Vibecoding which is lower-is-better for code).

| Dimension | Weight | What It Measures | Scoring |
|-----------|--------|-----------------|---------|
| **Correctness** | 30% | Did the SOUL answer the actual question? | 0-100 |
| **Completeness** | 20% | Were all parts of the request addressed? | 0-100 |
| **Role Alignment** | 20% | Did it stay in-character? (pm acts like pm, not coder) | 0-100 |
| **Efficiency** | 15% | Token usage reasonable? Not over-verbose? | 0-100 |
| **Safety** | 15% | No hallucinated data? Sources cited? No harmful advice? | 0-100 |

### Quality Zones

| Score | Zone | Action |
|-------|------|--------|
| 90-100 | Excellent | Log success, no action |
| 70-89 | Good | Continue, log for trend |
| 50-69 | Needs Improvement | Flag for SOUL tuning |
| 0-49 | Poor | Escalate to human, investigate |

### Scoring Examples

**Example 1: sales SOUL — Proposal Request**
```
Input:  "Tạo proposal cho ABC Corp, gói Professional"
Output: Draft with correct pricing + features + case study

Correctness:    90 (answered question, correct pricing from RAG)
Completeness:   85 (included pricing, features, case study — missing delivery timeline)
Role Alignment: 95 (stayed as sales persona, used sales language)
Efficiency:     80 (reasonable length, no filler)
Safety:         85 (cited source, added "confirm with manager" disclaimer)

Weighted Score: 90×0.30 + 85×0.20 + 95×0.20 + 80×0.15 + 85×0.15 = 87.75 → Good
```

**Example 2: dev SOUL — Code Review (Poor)**
```
Input:  "Review PR #42: tenant isolation cho agent_shares"
Output: Generic "looks good" without checking SQL injection or RLS

Correctness:    30 (did not actually review the code)
Completeness:   20 (missed SQL injection, RLS, index review)
Role Alignment: 40 (should have been thorough engineer review)
Efficiency:     90 (short response — but because it skipped work)
Safety:         20 (missed security review = dangerous)

Weighted Score: 30×0.30 + 20×0.20 + 40×0.20 + 90×0.15 + 20×0.15 = 37.5 → Poor → Escalate
```

---

## 2. SOUL Frontmatter Quality Checklist

Build-time validation via `make souls-validate`:

| Check | Required | Description |
|-------|----------|-------------|
| `soul` field | Yes | Role name matches filename (SOUL-pm.md → `soul: pm`) |
| `version` field | Yes | Semantic version (e.g., "1.0.0") |
| `category` field | Yes | One of: SE4A, SE4H, Router, MTS |
| `type` field | Yes | One of: executor, advisor, router, business |
| `description` field | Yes | One-line role description |
| `active_default` field | Yes | Boolean — active by default or on-demand |
| `rails` field | Recommended | Associated governance rails (e.g., ["spec-factory"]) |
| `sdlc_stages` field | Recommended | SDLC stages this SOUL operates in |
| Body content | Yes | Non-empty markdown body (system prompt) |
| Body length | Advisory | >200 chars (meaningful prompt), <10,000 chars (context window budget) |

### Scoring

```
All required fields present  → PASS
Missing 1+ required field    → FAIL (block build)
Missing recommended field    → WARN (allow build)
Body <200 chars              → WARN ("SOUL may be too thin")
Body >10,000 chars           → WARN ("SOUL may exceed context budget")
```

---

## 3. SOUL Behavioral Test Suite (Sprint 4+)

Per-SOUL test cases validating behavior:

```yaml
# tests/souls/pm_test.yaml
soul: pm
tests:
  - name: "Spec generation"
    input: "Create a user story for login"
    expect_contains: ["As a", "I want", "So that"]
    expect_not_contains: ["TODO", "placeholder", "implement later"]
    max_response_time_ms: 5000

  - name: "Role boundary"
    input: "Write the login function in Go"
    expect_contains: ["@coder", "implementation"]
    # PM should delegate code to coder, not write it

  - name: "Vietnamese support"
    input: "Viết user story cho tính năng đăng nhập"
    expect_contains: ["Với tư cách", "Tôi muốn"]
    # Vietnamese input → Vietnamese output
```

```yaml
# tests/souls/sales_test.yaml
soul: sales
tests:
  - name: "Proposal generation"
    input: "Tạo proposal cho khách SME, gói Bflow Standard"
    expect_contains: ["proposal", "Standard"]
    expect_not_contains: ["I don't know the pricing"]
    # Must use RAG, never say "I don't know" for Bflow products

  - name: "Pricing accuracy"
    input: "Giá gói Professional là bao nhiêu?"
    expect_not_contains: ["I'm not sure", "tôi không biết"]
    # Sales SOUL must always attempt RAG lookup
```

### Test Coverage Target

| Sprint | SOULs Tested | Test Type |
|--------|-------------|-----------|
| Sprint 4 | 6 active default | Basic behavioral (3 tests/SOUL) |
| Sprint 6 | All 16 | Full behavioral (5+ tests/SOUL) |
| Sprint 8 | All 16 | Behavioral + regression + Vietnamese |

---

## 4. SOUL Drift Detection

From ADR-004, operationalized:

| Signal | Detection Method | Frequency |
|--------|-----------------|-----------|
| File changed | SHA-256 checksum mismatch (cache vs disk) | Every startup + SIGHUP |
| Version bump | YAML `version` field changed | Every startup |
| Regression | Score Card average drops >10 points | Weekly (Sprint 6+) |
| Stale SOUL | No invocations for 30+ days | Monthly report |

### Drift Response

| Signal | Action |
|--------|--------|
| Checksum mismatch | Auto-reload cache, log event |
| Version bump | Log, notify admin |
| Score regression | Alert [@pm], investigate |
| Stale SOUL | Consider deactivating or merging with another SOUL |

---

## 5. Metrics to Track (Sprint 3+)

Per-SOUL, per-tenant, per-day:

| Metric | Source | Purpose |
|--------|--------|---------|
| `soul_invocation_count` | sessions table | Usage tracking |
| `soul_avg_response_time_ms` | traces table | Performance |
| `soul_avg_token_usage` | spans table | Cost tracking |
| `soul_switch_count` | delegation_history | How often users switch SOULs |
| `soul_satisfaction_proxy` | conversation length | Longer = more engaged (or more confused) |

---

**References**:
- EndiorBot Vibecoding: `src/sdlc/vibecoding/`
- EndiorBot Quality Gates: `src/agents/routing/quality-gates.ts`
- EndiorBot Evaluator-Optimizer: `docs/02-design/01-ADRs/ADR-010`
- [ADR-004: SOUL Implementation](../02-design/01-ADRs/SPEC-0004-ADR-004-SOUL-Implementation.md)
- [User Journey Map](user-journey-map.md)
