# PR Gate Design — Governance Rail #2

**SDLC Stage**: 02-Design
**Version**: 1.0.0
**Date**: 2026-03-03
**Author**: [@architect]
**Sprint**: 5 (WARNING mode) → 8 (ENFORCE mode)

---

## 1. Overview

PR Gate = automated code review rail that evaluates pull requests for quality, security, and governance compliance. Phased:

| Phase | Sprint | Mode | Behavior |
|-------|--------|------|----------|
| Prototype | 5 | **WARNING** | Review via Telegram command, report only |
| Full | 8 | **ENFORCE** | GitHub webhook, block merge on violation |

---

## 2. Architecture — Sprint 5 (WARNING Mode)

### Flow Diagram

```
User in Telegram                    GoClaw Gateway                     Agent Loop
─────────────────                  ───────────────                    ──────────
/review <PR_URL>  ──────────────►  handleBotCommand()
                                   │
                                   ├─ Validate URL (github.com/*/pull/*)
                                   ├─ Send ack: "🔍 Reviewing PR..."
                                   ├─ PublishInbound({
                                   │    AgentID:  "reviewer",
                                   │    Content:  PR_URL,
                                   │    Metadata: {
                                   │      command: "review",
                                   │      rail:    "pr-gate",
                                   │      pr_url:  URL
                                   │    }
                                   │  })
                                   │
                                   ▼
                              processNormalMessage()
                                   │
                                   ├─ Route to reviewer SOUL
                                   ├─ TraceName: "pr-gate"
                                   ├─ TraceTags: ["rail:pr-gate", "command:review"]
                                   │
                                   ▼
                              Reviewer SOUL (agent loop)
                                   │
                                   ├─ Skills loader injects pr-gate/SKILL.md
                                   ├─ Tools: web_fetch (fetch PR diff)
                                   ├─ Evaluate PR against policy rules
                                   │
                                   ▼
                              Response → Telegram reply
                              "⚠️ PR Review (WARNING mode)
                               Issues found: ...
                               Suggestions: ..."
```

### Why Telegram-First (Not GitHub Webhook)

Sprint 5 = WARNING mode = "report only, don't block". Full GitHub webhook integration (receive PR events → post PR comments → block merge) requires:

1. GitHub App registration or PAT with repo write access
2. Webhook endpoint with signature verification
3. PR comment API integration
4. Merge blocking logic (status checks)

This is ~6 points of infrastructure work. For WARNING mode, **Telegram command is sufficient**:
- User pastes PR URL → reviewer SOUL fetches diff via `web_fetch` tool → reports findings
- Zero GitHub infrastructure needed
- Validates review logic before building webhook pipeline
- Same reviewer SOUL + SKILL.md reused in Sprint 8 (just changes trigger)

**Sprint 8**: Add GitHub webhook → same reviewer SOUL, different trigger (webhook vs Telegram command).

---

## 3. Components

### 3.1 pr-gate/SKILL.md (Reviewer SOUL Instruction)

```
docs/08-collaborate/skills/pr-gate/SKILL.md
```

The skill file instructs the reviewer SOUL on:
- How to fetch and parse PR diff
- What policy rules to evaluate
- How to format the review report
- WARNING mode behavior (report, don't block)

### 3.2 /review Command (Telegram Handler)

New case in `handleBotCommand()` switch:

```go
case "/review":
    // Rail #2: PR Gate — route to reviewer SOUL for code review.
    prURL := strings.TrimSpace(text[len("/review"):])
    if prURL == "" || !isGitHubPRURL(prURL) {
        // Usage message
        return true
    }
    // Ack + PublishInbound with AgentID: "reviewer"
```

Pattern: identical to `/spec` (Task 3, Sprint 4).

### 3.3 Policy Rules (WARNING Mode)

Sprint 5 rules — soft warnings only:

| Rule | Severity | Check | Tool |
|------|----------|-------|------|
| Missing test files | WARN | PR adds `.go` files but no `_test.go` | diff analysis |
| Large diff | WARN | >500 lines changed | diff size |
| Security patterns | WARN | Hardcoded secrets, SQL injection patterns | regex scan |
| Missing spec reference | WARN | No `SPEC-` or issue reference in PR title/body | PR metadata |
| TODO/FIXME | INFO | New TODO comments added | diff analysis |

Sprint 8 adds ENFORCE rules (BLOCK merge):
- Missing test coverage (<60%)
- Security violation (OWASP patterns)
- No spec reference for feature PRs

### 3.4 Review Report Format

```
🔍 PR Review — WARNING Mode

**PR**: <title> (#<number>)
**Author**: <author>
**Files**: <count> files, +<add>/-<del> lines

## Issues Found

⚠️ **WARN**: No test files added (3 new .go files, 0 test files)
⚠️ **WARN**: Large diff (847 lines) — consider splitting
ℹ️ **INFO**: 2 TODO comments added

## Suggestions

1. Add `*_test.go` for new files
2. Consider splitting into smaller PRs

## Summary

| Category | Status |
|----------|--------|
| Tests | ⚠️ Missing |
| Size | ⚠️ Large |
| Security | ✅ Clean |
| Spec ref | ✅ Found |

**Mode**: WARNING (report only — merge not blocked)
```

---

## 4. Data Flow (Evidence)

```
/review PR_URL
    │
    ├─► trace.name = "pr-gate"
    ├─► trace.tags = ["rail:pr-gate", "command:review"]
    ├─► trace.input_preview = PR_URL
    ├─► trace.output_preview = review summary (first 500 chars)
    ├─► trace.total_input_tokens = N (diff + system prompt)
    └─► trace.total_output_tokens = M (review report)

Query:
    SELECT * FROM traces
    WHERE name = 'pr-gate' AND tenant_id = 'mts'
    ORDER BY created_at DESC;
```

---

## 5. Sprint 8 Extension (ENFORCE Mode)

When upgrading from WARNING → ENFORCE:

| Component | Sprint 5 (WARNING) | Sprint 8 (ENFORCE) |
|-----------|-------------------|-------------------|
| Trigger | Telegram `/review` command | GitHub webhook (PR opened/updated) |
| Output | Telegram reply | GitHub PR comment + status check |
| Blocking | No | Yes (merge blocked on FAIL) |
| Policy rules | 5 soft warnings | 5 warnings + 3 hard blocks |
| SKILL.md | Same | Same (reused) |
| Reviewer SOUL | Same | Same (reused) |

New Sprint 8 components:
- `internal/http/webhook_github.go` — webhook receiver with signature verification
- `internal/tools/github_pr.go` — GitHub API client (post comments, set status checks)
- Migration for `pr_gate_evaluations` table

---

## 6. Dependencies

| Dependency | Status | Sprint |
|------------|--------|--------|
| Reviewer SOUL seeded | ✅ (migration 000009) | 3 |
| Skills loader auto-discovery | ✅ (internal/skills/loader.go) | — |
| web_fetch tool (fetch PR diff) | ✅ (internal/tools/web_fetch.go) | — |
| TraceName/TraceTags flow | ✅ (Sprint 4 US-025) | 4 |
| /spec command pattern | ✅ (Sprint 4 US-021) | 4 |
| Hooks engine | ✅ (internal/hooks/) | — |
| GitHub App/PAT | ❌ Not needed Sprint 5 | 8 |

---

## 7. Risks

| Risk | Mitigation |
|------|------------|
| web_fetch can't access private repos | Sprint 5: use public repos for testing. Sprint 8: GitHub App token |
| LLM hallucinating review issues | SKILL.md constrains output format + evidence-based analysis |
| Large PRs exceed context window | Truncate diff to 4,000 tokens (system prompt + diff must fit context) |
| Review quality varies | Collect WARNING mode data Sprint 5-7, tune SKILL.md before ENFORCE |

---

## References

- [Requirements FR-003](../01-planning/requirements.md) — 3 Rails governance
- [Test Strategy](../01-planning/test-strategy.md) — PR Gate WARNING scenario (Sprint 5)
- [System Architecture Document](system-architecture-document.md) — 5-layer architecture
- [/spec Command Design](spec-command-design.md) — Pattern reference (same command→SOUL flow)
- [Hooks System](../../internal/hooks/hooks.go) — Quality gate engine
