# Sprint 5 — Coder Handoff

**Sprint**: 5 — MTS Pilot + PR Gate WARNING (Rail #2)
**From**: [@pm] (plan) + [@architect] (PR Gate design)
**To**: [@coder]
**Date**: 2026-03-03
**Predecessor**: Sprint 4 ✅ (CTO 9.0/10 APPROVED)

---

## What's Already Done (Sprint 4 Deliverables)

All Sprint 4 code is committed and verified (`go vet` + `go build` PASS):

| Deliverable | Files | Status |
|-------------|-------|--------|
| /spec command (Rail #1) | `commands.go` (case `/spec`) | ✅ PM SOUL routes |
| spec-factory SKILL.md | `docs/08-collaborate/skills/spec-factory/SKILL.md` | ✅ Auto-discovered |
| @mention routing | `gateway_consumer.go` (lines 54-72) | ✅ Validated via agents.Get() |
| Context Anchoring Layer A | `gateway_consumer.go` (lines 194-224) | ✅ Session goal + SOUL reminder |
| Evidence metadata | `gateway_consumer.go` (lines 241-255) | ✅ TraceName + TraceTags |
| IT Admin SOUL | `migrations/000012_seed_itadmin_soul.up.sql` | ✅ 17th SOUL |
| souls-validate | `Makefile` (souls-validate target) | ✅ Frontmatter FAIL, char WARN |
| Telegram polling | `.env.example` | ✅ GOCLAW_TELEGRAM_POLLING |

**Post-Review Fixes Applied**:
- ISSUE-1: `/spec` case-sensitive prefix → `text[len("/spec"):]` (safe byte slice)
- CTO minor: Migration 000012 header "2 delegation links" → "3 delegation links"

---

## Sprint 5 Tasks — Implementation Guide

### Task 1: PR Gate SKILL.md (Day 1, included in US-027)

**What**: Create skill definition file for PR Gate rail.

**File**: `docs/08-collaborate/skills/pr-gate/SKILL.md`

**Content**:

```markdown
---
name: pr-gate
description: Code review for pull requests. Governance Rail #2 (WARNING mode).
---

# PR Gate — Governance Rail #2

## When This Skill Activates

- User sends `/review <PR_URL>` command
- User asks to "review this PR", "check this code", "evaluate this pull request"
- Reviewer SOUL receives a code review request

## Review Process

1. **Fetch**: Use web_fetch to retrieve the PR diff from the URL
2. **Analyze**: Evaluate the diff against policy rules below
3. **Report**: Format findings as a structured review report

## Policy Rules (WARNING Mode)

Evaluate and report — do NOT block merge.

| Rule | Severity | Check |
|------|----------|-------|
| Missing tests | WARN | PR adds .go/.ts files but no corresponding test files |
| Large diff | WARN | >500 lines changed — suggest splitting |
| Security patterns | WARN | Hardcoded secrets, SQL injection, XSS patterns |
| Missing spec reference | WARN | No SPEC- or issue # in PR title/body |
| TODO/FIXME | INFO | New TODO/FIXME comments added |

## Report Format

Format your review as:

🔍 **PR Review — WARNING Mode**

**PR**: {title} (#{number})
**Files**: {count} files, +{additions}/-{deletions} lines

### Issues Found
- ⚠️ WARN: {description}
- ℹ️ INFO: {description}

### Suggestions
1. {actionable suggestion}

### Summary
| Category | Status |
|----------|--------|
| Tests | ⚠️/✅ |
| Size | ⚠️/✅ |
| Security | ⚠️/✅ |
| Spec ref | ⚠️/✅ |

**Mode**: WARNING (report only — merge not blocked)

## Boundaries

- This skill reviews code only — not architecture, not specs, not deployment
- If PR requires architecture review → delegate to @architect
- If PR lacks a spec → suggest creating one via @pm /spec
- WARNING mode: NEVER say "merge blocked" or "PR rejected"

## Vietnamese Support

- Review in the language of PR content
- If mixed → use English (code convention)
```

**Skills Loader**: Auto-discovered by GoClaw. Same pattern as spec-factory.

---

### Task 2: /review Command Handler (Day 1-2, 3 pts)

**What**: Add `/review` case to Telegram command handler.

**File to modify**: `internal/channels/telegram/commands.go`

**Pattern**: Identical to `/spec` case (Sprint 4 Task 3). Changes:
- Command: `/review`
- AgentID: `"reviewer"` (not `"pm"`)
- Metadata: `command: "review"`, `rail: "pr-gate"`, `pr_url: URL`
- Ack message: `"🔍 Reviewing PR..."`
- URL validation: check that input looks like a GitHub PR URL

**Implementation**:

```go
case "/review":
    // Rail #2: PR Gate — route to reviewer SOUL for code review.
    prURL := strings.TrimSpace(text[len("/review"):])
    if prURL == "" || !strings.Contains(prURL, "/pull/") {
        usageMsg := tu.Message(chatIDObj, "Usage: /review <github_pr_url>\n\nExample: /review https://github.com/org/repo/pull/123")
        setThread(usageMsg)
        c.bot.SendMessage(ctx, usageMsg)
        return true
    }

    ackMsg := tu.Message(chatIDObj, "🔍 Reviewing PR...")
    setThread(ackMsg)
    c.bot.SendMessage(ctx, ackMsg)

    peerKind := "direct"
    if isGroup {
        peerKind = "group"
    }
    c.Bus().PublishInbound(bus.InboundMessage{
        Channel:  c.Name(),
        SenderID: senderID,
        ChatID:   chatIDStr,
        Content:  prURL,
        PeerKind: peerKind,
        AgentID:  "reviewer", // Always route to Reviewer SOUL
        UserID:   strings.SplitN(senderID, "|", 2)[0],
        Metadata: map[string]string{
            "command":           "review",
            "rail":              "pr-gate",
            "pr_url":            prURL,
            "local_key":         localKey,
            "is_forum":          fmt.Sprintf("%t", isForum),
            "message_thread_id": fmt.Sprintf("%d", messageThreadID),
        },
    })
    return true
```

**Also update `/help` text**: Add `/review` entry.

**Validation**:
- [ ] `/review` with no URL → usage message
- [ ] `/review https://github.com/org/repo/pull/123` → "🔍 Reviewing PR..." then review report
- [ ] Trace record: name='pr-gate', tags=['rail:pr-gate', 'command:review']

---

### Task 3: Fix Sprint 4 Tech Debt — UTF-8 Rune Truncation (Day 1, 0.5 pts)

**What**: Fix ISSUE-2 from Sprint 4 review — `len(goal) > 200` counts bytes, not runes.

**File**: `cmd/gateway_consumer.go` (around line 207)

**Current**:
```go
if len(goal) > 200 {
    goal = goal[:200] + "..."
}
```

**Fix**:
```go
goalRunes := []rune(goal)
if len(goalRunes) > 200 {
    goal = string(goalRunes[:200]) + "..."
}
```

**Note**: The hooks engine already does this correctly at `engine.go:75-80` (the `truncate` function uses `[]rune`). Follow that pattern.

---

### Task 4: Integration Tests (Day 3-4, 3 pts)

**What**: Create integration test scenarios for critical paths.

**File to create**: `cmd/gateway_consumer_test.go` (or `internal/integration_test.go`)

**Test Scenarios** (from test strategy):

1. **Tenant Isolation**: Verify RLS prevents cross-tenant data access
   - Set `app.tenant_id = 'mts'` → query agents → returns MTS agents only
   - Set `app.tenant_id = 'other'` → query agents → returns empty (no agents for 'other')

2. **@mention Routing**: Verify agent resolution
   - `@reviewer` → agents.Get("reviewer") succeeds
   - `@pm` → agents.Get("pm") succeeds
   - `@nonexistent` → agents.Get("nonexistent") fails → mentionAgent stays empty

3. **Command Metadata Flow**: Verify TraceName/TraceTags populated
   - InboundMessage with `Metadata["rail"] = "spec-factory"` → TraceName = "spec-factory"
   - InboundMessage with `Metadata["command"] = "review"` → TraceTags contains "command:review"

4. **Command Routing Priority**: Verify /spec always goes to PM
   - `/spec` message → AgentID = "pm" (not overridden by @mention or handoff)
   - `@reviewer` message → AgentID = "reviewer" (mention takes priority over default)

5. **AI-Platform Graceful Degradation**:
   - If provider returns error → formatAgentError() produces user-friendly message
   - If context cancelled → "run cancelled" logged, no error to user

**Build tag**: Use `//go:build integration` if tests require real database.

**Target**: ≥5 scenarios passing. 70% unit coverage per test strategy.

---

### Task 5: Staging Deployment Verification (Day 2-3, operational)

**What**: Deploy to MTS VPS and verify all endpoints.

**Steps**:
1. SSH to MTS VPS
2. Set `.env` with real credentials:
   ```bash
   GOCLAW_TELEGRAM_TOKEN=<from BotFather>
   GOCLAW_TELEGRAM_POLLING=true
   GOCLAW_BFLOW_API_KEY=<aip_c786...>
   GOCLAW_BFLOW_BASE_URL=http://ai-platform:8120/api/v1
   BFLOW_TENANT_ID=mts
   GOCLAW_ENCRYPTION_KEY=<openssl rand -hex 32>
   GOCLAW_POSTGRES_DSN=postgres://mtclaw:...@localhost:5432/mtclaw?sslmode=disable
   ```
3. Deploy:
   ```bash
   docker compose -f docker-compose.yml \
                  -f docker-compose.managed.yml \
                  -f docker-compose.mts.yml \
                  up -d --build
   ```
4. Run migrations: `docker exec mtclaw ./mtclaw migrate up`
5. Verify:
   - [ ] `GET /v1/agents` returns 17 agents
   - [ ] Telegram: `/start` → welcome message
   - [ ] Telegram: `/help` → includes /spec AND /review
   - [ ] Telegram: `/spec Create login feature` → PM SOUL responds with spec
   - [ ] Telegram: `/review <pr_url>` → Reviewer SOUL responds with review
   - [ ] Telegram: `@itadmin check server status` → IT Admin SOUL responds

**Note**: This is primarily [@devops] + [@pm] scope. [@coder] provides the code, verifies endpoints.

---

### Task 6: Token Cost Tracking Query (Day 4, 1 pt)

**What**: Verify CTO ISSUE-B — token usage queryable from traces table.

**Steps**:
1. After staging is running, send 5+ commands (/spec, /review, @soul messages)
2. Query:
   ```sql
   SELECT agent_key, COUNT(*) as runs,
          SUM(total_input_tokens) as input_tokens,
          SUM(total_output_tokens) as output_tokens,
          SUM(total_input_tokens + total_output_tokens) as total_tokens
   FROM traces
   WHERE tenant_id = 'mts'
   GROUP BY agent_key
   ORDER BY total_tokens DESC;
   ```
3. Document: if tokens > 0 → traces table is sufficient. If tokens = 0 → investigate provider token counting.
4. Decision: defer `token_usage` table or create it in Sprint 6.

---

## Files to Create

| File | Purpose |
|------|---------|
| `docs/08-collaborate/skills/pr-gate/SKILL.md` | PR Gate skill definition (reviewer SOUL) |
| `cmd/gateway_consumer_test.go` | Integration tests (≥5 scenarios) |

## Files to Modify

| File | Changes |
|------|---------|
| `internal/channels/telegram/commands.go` | Add `/review` case + update `/help` text |
| `cmd/gateway_consumer.go` | Fix rune truncation (line ~207) |

## Files NOT to Modify

| File | Reason |
|------|--------|
| `internal/skills/loader.go` | Already loads arbitrary SKILL.md files |
| `internal/agent/loop.go` | TraceName/TraceTags already flow through |
| `internal/hooks/` | Not wired in Sprint 5 — PR Gate uses SOUL-based review, not hook-based |
| `internal/providers/bflow_ai.go` | Complete from Sprint 3 |

---

## Key Code Paths (Study Before Implementing)

1. **`/spec` command pattern**: `commands.go:95-132` — copy for `/review`
2. **Evidence metadata flow**: `gateway_consumer.go:241-275` — TraceName/TraceTags from rail metadata
3. **Skills loader**: `internal/skills/loader.go` — auto-discovers SKILL.md from docs/
4. **web_fetch tool**: `internal/tools/web_fetch.go` — reviewer SOUL uses this to fetch PR diff
5. **Hooks truncate function**: `internal/hooks/engine.go:75-80` — rune-safe truncation pattern

---

## Sprint 5 vs Sprint 8 Scope Boundary

| Feature | Sprint 5 | Sprint 8 |
|---------|----------|----------|
| Trigger | Telegram `/review` command | GitHub webhook (PR opened/updated) |
| Output | Telegram reply | GitHub PR comment + status check |
| Blocking | No (WARNING) | Yes (ENFORCE) |
| GitHub integration | None (web_fetch only) | GitHub App + webhook + API |
| SKILL.md | Created here | Reused |
| Reviewer SOUL | Same | Same |

[@coder] implements Sprint 5 scope ONLY. GitHub webhook = Sprint 8.

---

## References

- [Sprint 5 Plan](sprints/SPRINT-005-MTS-Pilot-PRGate.md) — full sprint plan
- [PR Gate Design](../02-design/pr-gate-design.md) — architecture + WARNING vs ENFORCE
- [Sprint 4 Coder Handoff](SPRINT-004-CODER-HANDOFF.md) — predecessor
- [Test Strategy](../01-planning/test-strategy.md) — 70% coverage target
- [Requirements FR-003](../01-planning/requirements.md) — 3 Rails governance
