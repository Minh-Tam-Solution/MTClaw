# Sprint 6 — Coder Handoff

**Sprint**: 6 — NQH Tenant + Rail #3 Knowledge + Team Routing
**From**: [@pm] (plan) + [@architect] (SAD Section 8)
**To**: [@coder]
**Date**: 2026-03-03
**Predecessor**: Sprint 5 ✅ (CTO 9.0/10 APPROVED)

---

## What's Already Done (Sprint 5 Deliverables)

All Sprint 5 code is committed and verified (`go vet` + `go build` + 22 tests PASS):

| Deliverable | Files | Status |
|-------------|-------|--------|
| PR Gate SKILL.md (Rail #2) | `docs/08-collaborate/skills/pr-gate/SKILL.md` | ✅ |
| /review command | `commands.go` (case `/review`, lines 135-170) | ✅ |
| /help updated | `commands.go` (includes /review) | ✅ |
| Rune truncation fix (ISSUE-2) | `gateway_consumer.go` (`[]rune` pattern) | ✅ |
| Integration tests (5 scenarios, 22 tests) | `cmd/gateway_consumer_test.go` | ✅ |

**Sprint 4 deliverables** (still in production, Sprint 6 builds on these):
- Context Anchoring Layer A → `gateway_consumer.go:194-224` (ExtraPrompt injection)
- @mention SOUL routing → `gateway_consumer.go:54-72` (agent-first resolution)
- Evidence metadata → `gateway_consumer.go:241-257` (TraceName + TraceTags)

---

## Sprint 6 Tasks — Implementation Guide

### Task 1: SOUL-Aware RAG Integration (Day 1-2, 3 pts — US-034)

**What**: Query Bflow AI-Platform RAG API based on active SOUL role, inject results into agent context.

**Architecture**: SAD Section 8.4 — SOUL-Aware RAG Routing (Layer B)

**API**: `POST {BFLOW_BASE_URL}/api/v1/rag/query`

```json
// Request
{
  "query": "How does Bflow handle multi-tenant POS sync?",
  "collection": "engineering",
  "top_k": 5,
  "max_tokens": 2500
}

// Response
{
  "results": [
    {
      "content": "Bflow uses event-driven sync...",
      "metadata": { "source": "architecture/sync.md", "score": 0.92 }
    }
  ],
  "total_hits": 12,
  "tokens_used": 1850
}
```

**Auth**: Same as chat — `X-API-Key` + `X-Tenant-ID` headers.

**SOUL → Collection mapping** (hardcoded table first, configurable later):

```go
// ragCollectionMap maps SOUL agent_key to RAG collection(s).
var ragCollectionMap = map[string][]string{
    "enghelp":   {"engineering"},
    "coder":     {"engineering"},
    "architect": {"engineering"},
    "reviewer":  {"engineering"},
    "devops":    {"engineering"},
    "tester":    {"engineering"},
    "itadmin":   {"engineering"},
    "sales":     {"sales"},
    "cs":        {"engineering", "sales"},
    "assistant": {"engineering", "sales"},
    "pm":        {"engineering", "sales"},
    "writer":    {"engineering"},
    // NQH SOULs (Phase 2, conditional):
    // "nqh-*": {"nqh-sops"},
}
```

**Where to add RAG call**: `cmd/gateway_consumer.go`, AFTER agent resolution (line ~92) and BEFORE the agent loop call (line ~274). Insert between the existing ExtraPrompt construction (lines 177-224) and the `loop.Run()` call.

**Implementation pattern**:

```go
// --- Sprint 6: SOUL-Aware RAG Routing (US-034, Context Drift Layer B) ---
// Query RAG collections based on active SOUL role.
{
    collections := ragCollectionMap[agentID]
    if len(collections) > 0 && msg.Content != "" {
        ragCtx, ragCancel := context.WithTimeout(ctx, 5*time.Second)
        defer ragCancel()

        ragResults, err := queryRAG(ragCtx, cfg, msg.Content, collections, 2500)
        if err != nil {
            slog.Warn("rag: query failed, proceeding without RAG context",
                "agent", agentID, "error", err)
        } else if len(ragResults.Results) > 0 {
            // Inject RAG results into ExtraPrompt
            var ragSection strings.Builder
            ragSection.WriteString("## Knowledge Base Context\n")
            ragSection.WriteString("The following information was retrieved from the knowledge base.\n")
            ragSection.WriteString("Cite sources when using this information.\n\n")
            for _, r := range ragResults.Results {
                ragSection.WriteString(fmt.Sprintf("### %s (score: %.2f)\n%s\n\n",
                    r.Metadata.Source, r.Score, r.Content))
            }
            if extraPrompt != "" {
                extraPrompt += "\n\n"
            }
            extraPrompt += ragSection.String()

            // Evidence metadata (RetrievalEvidence)
            traceTags = append(traceTags,
                "rag:"+strings.Join(collections, "+"),
                fmt.Sprintf("rag_hits:%d", ragResults.TotalHits),
                fmt.Sprintf("rag_tokens:%d", ragResults.TokensUsed),
            )
        }
    }
}
```

**New function** (`queryRAG`): Create in `cmd/gateway_consumer.go` or extract to `internal/rag/client.go`:

```go
type RAGResult struct {
    Content  string `json:"content"`
    Metadata struct {
        Source string  `json:"source"`
        Score  float64 `json:"score"`
    } `json:"metadata"`
    Score float64 `json:"score"`
}

type RAGResponse struct {
    Results    []RAGResult `json:"results"`
    TotalHits  int         `json:"total_hits"`
    TokensUsed int         `json:"tokens_used"`
}

func queryRAG(ctx context.Context, cfg *config.Config, query string, collections []string, maxTokens int) (*RAGResponse, error) {
    // POST to AI-Platform RAG endpoint
    // Auth: X-API-Key, X-Tenant-ID (same as bflow_ai.go)
    // Endpoint: cfg.BflowAI.BaseURL + "/api/v1/rag/query"
    // Graceful degradation: return nil, err on failure
}
```

**Token budget**: Hard cap 2,500 tokens. If RAG response exceeds, truncate results (drop lowest-score results first).

**Graceful degradation**: If RAG fails (timeout, 500, network error), log warning and proceed WITHOUT RAG context. Never block the user message.

**Reference**: SOP Generator RAG client pattern at `/home/nqh/shared/Bflow-Platform/Sub-Repo/SOP-Generator/backend/services/sop_generation_service/app/services/rag_client.py`

---

### Task 2: Team Mention Routing (Day 1-2, 3 pts — US-036)

**What**: Extend @mention parsing to support team mentions (`@engineering`, `@business`, `@advisory`).

**File**: `cmd/gateway_consumer.go` (lines 54-92, the existing @mention block)

**Current behavior** (Sprint 4):
```
@pm → agents.Get("pm") → found → route to PM
@nonexistent → agents.Get("nonexistent") → not found → no routing change
```

**New behavior** (Sprint 6):
```
@pm → agents.Get("pm") → found → route to PM (agent-first, unchanged)
@engineering → agents.Get("engineering") → not found → check teams → found "engineering" team → route to team lead (pm)
@business → agents.Get("business") → not found → check teams → found "business" team → route to team lead (assistant)
@advisory → agents.Get("advisory") → not found → check teams → found "advisory" team → route to team lead (cto)
@nonexistent → agents.Get() fail → team lookup fail → no routing change
```

**CTO-8 FIX**: Team names in DB are full names ("SDLC Engineering"), but users type short mentions (`@engineering`). Use hardcoded mention-key map (Option B — no migration needed for 3 static teams). Add `mention_key` column in Sprint 9 when teams become configurable.

**Mention-key map** (add near ragCollectionMap):

```go
// teamMentionMap maps short mention keys to full team names in DB.
// CTO-8: DB names are "SDLC Engineering" but users type @engineering.
// Hardcoded for Sprint 6 (3 static teams). Sprint 9+: add mention_key column.
var teamMentionMap = map[string]string{
    "engineering": "SDLC Engineering",
    "business":    "Business Operations",
    "advisory":    "Advisory Board",
}
```

**Implementation**: Modify the existing @mention block (lines 54-72):

```go
// --- Sprint 4: @mention SOUL routing (US-022) ---
// Extended Sprint 6: team mention routing (US-036)
mentionAgent := ""
mentionTeam := "" // NEW: track if routed via team
if msg.Metadata["command"] == "" {
    if strings.HasPrefix(msg.Content, "@") {
        parts := strings.SplitN(msg.Content, " ", 2)
        candidate := strings.TrimPrefix(parts[0], "@")
        candidate = strings.ToLower(candidate)

        // Agent-first resolution (existing)
        if _, err := agents.Get(candidate); err == nil {
            mentionAgent = candidate
            if len(parts) > 1 {
                msg.Content = strings.TrimSpace(parts[1])
            }
            slog.Info("inbound: @mention agent route",
                "mention", candidate, "channel", msg.Channel)
        } else if teamStore != nil {
            // Team-second resolution (NEW Sprint 6)
            // CTO-8: Use teamMentionMap to resolve short keys to full DB names.
            if fullName, ok := teamMentionMap[candidate]; ok {
                teams, _ := teamStore.ListTeams(ctx)
                for _, t := range teams {
                    if t.Name == fullName {
                        mentionAgent = t.LeadAgentKey
                        mentionTeam = candidate // store the short mention key
                        if len(parts) > 1 {
                            msg.Content = strings.TrimSpace(parts[1])
                        }
                        slog.Info("inbound: @mention team route",
                            "team", t.Name, "mention", candidate,
                            "lead", t.LeadAgentKey, "channel", msg.Channel)
                        break
                    }
                }
            }
        }
    }
}
```

**Team context injection** (add to ExtraPrompt, after the existing anchor block):

```go
// --- Sprint 6: Team context injection ---
if mentionTeam != "" {
    // mentionTeam is the short key ("engineering"), resolve to full name for display
    fullName := teamMentionMap[mentionTeam]
    teamCtx := fmt.Sprintf("## Team Context\nYou are responding as the **lead** of the **%s** team.\n", fullName)
    teamCtx += "Team members available for delegation:\n"
    // List members from teamStore (team was already looked up above)
    teams, _ := teamStore.ListTeams(ctx)
    for _, t := range teams {
        if t.Name == fullName {
            members, _ := teamStore.ListMembers(ctx, t.ID)
            for _, m := range members {
                teamCtx += fmt.Sprintf("- @%s\n", m.AgentKey)
            }
            break
        }
    }
    if extraPrompt != "" {
        extraPrompt += "\n\n"
    }
    extraPrompt += teamCtx

    traceTags = append(traceTags, "team:"+mentionTeam)
}
```

**Performance note**: `teamStore.ListTeams()` is called per team-mention message. For Sprint 6 (3 teams, ~20 users), this is fine. If performance becomes an issue in Sprint 9+, cache the team list in memory with TTL.

**TeamMemberData**: Check if `AgentKey` is a joined field. If not, you may need to join via `agents` table. Look at `internal/store/pg/teams.go` for the ListMembers query.

---

### Task 3: /teams Command (Day 2, 1 pt — US-037)

**What**: Add `/teams` command to Telegram handler.

**File**: `internal/channels/telegram/commands.go`

**Pattern**: Simpler than /spec or /review — just list teams and reply.

```go
case "/teams":
    // Sprint 6: List available teams (US-037, CPO CONDITION-1 discoverability)
    teamsText := "📋 **Available Teams**\n\n"
    teamsText += "`@engineering` — SDLC Engineering (lead: @pm)\n"
    teamsText += "`@business` — Business Operations (lead: @assistant)\n"
    teamsText += "`@advisory` — Advisory Board (lead: @cto)\n"
    teamsText += "\nUse `@team_name <message>` to route to a team.\n"
    teamsText += "Use `@agent_name <message>` to route to a specific agent."
    msg := tu.Message(chatIDObj, teamsText)
    setThread(msg)
    c.bot.SendMessage(ctx, msg)
    return true
```

**Note**: For Sprint 6, hardcode the team list (3 teams are static). Dynamic listing from TeamStore can be added later if teams become configurable.

**Also update /help text**: Add `/teams` entry.

---

### Task 4: Tenant Cost Guardrails (Day 3-4, 2 pts — US-039)

**What**: Enforce per-tenant token and request limits.

**Where**: `cmd/gateway_consumer.go` — check BEFORE the AI-Platform call.

**Implementation approach — PostgreSQL-only** (CTO-9: no Redis in Sprint 6):

1. **Config**: Add to tenant config (or `.env`):
```
GOCLAW_TENANT_MONTHLY_TOKEN_LIMIT=1000000   # 1M tokens/month
GOCLAW_TENANT_DAILY_REQUEST_LIMIT=500       # 500 requests/day
```

2. **Tracking** (PostgreSQL — query traces table):
```go
// checkTenantLimits queries the traces table for current usage.
// CTO-9: PostgreSQL-only for Sprint 6. Redis deferred to Sprint 9+ if query performance is an issue.

func checkTenantLimits(ctx context.Context, db *sql.DB, tenantID string, cfg TenantLimits) error {
    // Daily request count
    var dailyCount int
    err := db.QueryRowContext(ctx,
        `SELECT COUNT(*) FROM traces
         WHERE tenant_id = $1 AND created_at >= CURRENT_DATE`,
        tenantID).Scan(&dailyCount)
    if err != nil {
        return nil // fail-open: don't block on query error
    }
    if dailyCount >= cfg.DailyRequestLimit {
        return fmt.Errorf("daily request limit exceeded (%d/%d)", dailyCount, cfg.DailyRequestLimit)
    }

    // Monthly token count
    var monthlyTokens int64
    err = db.QueryRowContext(ctx,
        `SELECT COALESCE(SUM(total_input_tokens + total_output_tokens), 0) FROM traces
         WHERE tenant_id = $1 AND created_at >= date_trunc('month', CURRENT_DATE)`,
        tenantID).Scan(&monthlyTokens)
    if err != nil {
        return nil // fail-open
    }
    if monthlyTokens >= int64(cfg.MonthlyTokenLimit) {
        return fmt.Errorf("monthly token limit exceeded (%d/%d)", monthlyTokens, cfg.MonthlyTokenLimit)
    }

    return nil
}
```

3. **Check in gateway_consumer.go** (before agent loop):
```go
// --- Sprint 6: Cost guardrails (US-039) ---
if err := checkTenantLimits(ctx, db, tenantID, tenantLimits); err != nil {
    slog.Warn("tenant limit exceeded", "tenant", tenantID, "error", err)
    // Reply to user: "Đã đạt giới hạn sử dụng hôm nay. Vui lòng thử lại sau."
    return
}
```

4. **Warning at 80%**: Check daily count after the AI call; if ≥80% of daily limit, append warning to response.

**Performance note**: Two COUNT/SUM queries per request is acceptable for Sprint 6 volume (~30 users, ~500 req/day). Add a composite index if needed: `CREATE INDEX idx_traces_tenant_date ON traces (tenant_id, created_at)`. Redis optimization deferred to Sprint 9+.

---

### Task 5: Cross-Tenant Regression Tests (Day 4, 1 pt — US-040)

**What**: Add tests confirming MTS + NQH data isolation.

**File**: `cmd/gateway_consumer_test.go` (extend existing test file)

**Test scenarios**:

```go
// Scenario 6: Cross-Tenant Isolation
func TestCrossTenantIsolation_AgentVisibility(t *testing.T) {
    // SET app.tenant_id = 'mts' → query agents → returns MTS agents
    // SET app.tenant_id = 'nqh' → query agents → returns NQH agents (or empty if not seeded)
    // Verify: no overlap
}

// Scenario 7: RAG Collection Isolation
func TestRAGCollectionMapping_BySOUL(t *testing.T) {
    // dev → engineering collection
    // sales → sales collection
    // cs → engineering + sales
    // Verify: correct collection(s) returned from ragCollectionMap
}

// Scenario 8: Team Mention Resolution
func TestTeamMentionRouting_AgentFirst(t *testing.T) {
    // @pm → routes to pm (agent-first)
    // @engineering → routes to pm (team lead)
    // @advisory → routes to cto (team lead)
    // @nonexistent → no routing change
}

// Scenario 9: Cost Guardrail Enforcement
func TestCostGuardrail_DailyLimit(t *testing.T) {
    // Given 500 requests today → next request returns limit error
    // Given 499 requests today → request succeeds with 80% warning
}
```

---

### Task 6: NQH Tenant Migration (CONDITIONAL — Day 1-3, 2 pts — US-038)

**Condition**: Only implement if CEO re-confirms NQH pilot for Sprint 6.

**What**: Create NQH tenant + seed NQH SOULs + connect Zalo.

**Migration file**: `migrations/000013_seed_nqh_tenant.up.sql`

```sql
-- NQH tenant
INSERT INTO tenants (id, name, owner_id, status, settings)
VALUES (gen_random_uuid(), 'NQH Holdings', 'nqh', 'active', '{"monthly_token_limit": 500000, "daily_request_limit": 200}');

-- NQH SOULs: assistant + itadmin (minimal set for pilot)
-- Pattern: same as migration 000009 (MTS SOULs) but with tenant_id = nqh
```

**Zalo channel**: Configure via OpenClaw `extensions/zalo` + `extensions/zalouser`. This is operational ([@devops] scope), not code.

**NQH-SOPs RAG**: Connect collection `nqh-sops` (already indexed on AI-Platform, 805 docs). Add to ragCollectionMap.

**If NOT approved**: Skip this task entirely. MTS-only items (Tasks 1-5) proceed.

---

## Files to Create

| File | Purpose |
|------|---------|
| `docs/08-collaborate/skills/knowledge/SKILL.md` | Rail #3 Knowledge skill definition (optional — RAG is implicit, not command-triggered) |
| Tests in `cmd/gateway_consumer_test.go` | Extend with Scenarios 6-9 |
| `migrations/000013_seed_nqh_tenant.up.sql` | NQH tenant + SOULs (CONDITIONAL) |
| `migrations/000013_seed_nqh_tenant.down.sql` | Rollback (CONDITIONAL) |

## Files to Modify

| File | Changes |
|------|---------|
| `cmd/gateway_consumer.go` | RAG query injection + team mention routing + cost guardrails |
| `internal/channels/telegram/commands.go` | Add `/teams` command + update `/help` text |

## Files NOT to Modify

| File | Reason |
|------|--------|
| `internal/providers/bflow_ai.go` | RAG uses HTTP client directly, not the chat provider interface |
| `internal/agent/loop.go` | ExtraPrompt already flows through — no changes needed |
| `internal/store/team_store.go` | TeamStore interface is sufficient (ListTeams, ListMembers) |
| `internal/store/pg/teams.go` | Existing PG implementation works — no schema changes |
| `internal/hooks/` | Not involved in RAG or team routing |

---

## Key Code Paths (Study Before Implementing)

1. **@mention routing**: `gateway_consumer.go:54-92` — extend for team-second resolution
2. **ExtraPrompt construction**: `gateway_consumer.go:177-224` — add RAG + team context sections
3. **Evidence metadata**: `gateway_consumer.go:241-257` — add RAG trace tags
4. **Bflow AI auth pattern**: `internal/providers/bflow_ai.go` — X-API-Key + X-Tenant-ID headers (reuse for RAG)
5. **TeamStore**: `internal/store/team_store.go` (interface), `internal/store/pg/teams.go` (implementation)
6. **SOP Generator RAG client**: `/home/nqh/shared/Bflow-Platform/Sub-Repo/SOP-Generator/backend/services/sop_generation_service/app/services/rag_client.py` — reference pattern

---

## Sprint 5 vs Sprint 6 Scope Boundary

| Feature | Sprint 5 (done) | Sprint 6 (this sprint) |
|---------|-----------------|----------------------|
| Rail #1 Spec Factory | /spec prototype ✅ | Unchanged |
| Rail #2 PR Gate | /review WARNING ✅ | Unchanged |
| Rail #3 Knowledge | Not started | SOUL-Aware RAG routing + 3 collections |
| Context Drift Layer A | Anchoring ✅ | Unchanged |
| Context Drift Layer B | Not started | RAG routing by SOUL role |
| Team routing | Not started | @engineering, @business, @advisory |
| Cost guardrails | Token tracking only | Enforce limits per tenant |
| NQH tenant | Not started | CONDITIONAL |

---

## CTO Sprint 5 Review Notes (carry into Sprint 6)

| # | Note | Action |
|---|------|--------|
| NOTE-1 | Test helper duplication (extractTraceMetadata/parseMention) | Extract to real functions when adding new tests |
| NOTE-2 | PR URL validation too permissive | Defer to Sprint 8 (ENFORCE mode) |
| INFO | 24 actual test functions vs 22 claimed | Update count when adding Sprint 6 tests |

---

## References

- [Sprint 6 Plan](sprints/SPRINT-006-NQH-RAG-Teams.md) — full sprint plan
- [SAD Section 8](../02-design/system-architecture-document.md) — Context Drift Prevention (Layer B)
- [Roadmap v2.2.0](../01-planning/roadmap.md) — Sprint 6 scope
- [Team Charters](../08-collaborate/teams/) — TEAM-engineering, TEAM-business, TEAM-advisory
- [Requirements FR-003, FR-008](../01-planning/requirements.md) — 3 Rails + Context Drift
- [Sprint 5 Coder Handoff](SPRINT-005-CODER-HANDOFF.md) — predecessor
- [Bflow AI-Platform Guide](../../docs/03-integrate/bflow-ai-platform-sop-generator-guide.md) — RAG API reference
