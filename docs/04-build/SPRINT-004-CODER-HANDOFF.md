# Sprint 4 — Coder Handoff

**Sprint**: 4 — Core Deploy + /spec Prototype (Rail #1)
**From**: [@pm] (plan) + [@architect] (Sprint 3 foundation)
**To**: [@coder]
**Date**: 2026-03-06
**Predecessor**: Sprint 3 — CTO 8.5/10 APPROVED, P1 bug fixed

---

## What's Already Done (Sprint 3 Deliverables)

All Sprint 3 code is committed and verified (`go vet` + `go build` PASS):

| Deliverable | Files | Status |
|-------------|-------|--------|
| RLS tenant isolation | `migrations/000008_rls_tenant_isolation.up.sql` + `.down.sql`, `internal/middleware/tenant.go` | ✅ 8 tables enforced |
| SOUL seeding | `migrations/000009_seed_mtclaw_souls.up.sql` + `.down.sql` | ✅ 16 agents + 48 context files |
| Observability | `migrations/000010_observability_columns.up.sql` + `.down.sql`, `internal/middleware/logging.go`, `internal/store/context.go` | ✅ slog + trace columns |
| Bflow AI-Platform provider | `internal/providers/bflow_ai.go`, `internal/store/provider_store.go`, `internal/http/providers.go`, `internal/config/config_*.go`, `cmd/gateway_providers.go`, `migrations/000011_seed_bflow_provider.up.sql` + `.down.sql` | ✅ Dual path (config + DB) |
| MTS deploy config | `docker-compose.mts.yml`, `.env.example` | ✅ ai-net network |

**CTO Bug Fixes Applied**:
- `HasAnyProvider()` now includes `p.BflowAI.APIKey != ""` check (P1 fixed)
- `bflowTransport.RoundTrip()` clones request before mutating headers

---

## Sprint 4 Tasks — Implementation Guide

### Task 1: Telegram Bot Setup (Day 1, 1 point)

**What**: Register Telegram bot and configure GoClaw to connect.

**Steps**:
1. Create bot via BotFather (`/newbot`), save token
2. Set `GOCLAW_TELEGRAM_TOKEN={token}` in `.env`
3. Register commands with BotFather (`/setcommands`):
   ```
   start - Start MTClaw assistant
   help - Show available commands
   spec - Generate structured specification
   reset - Reset conversation
   status - Show current SOUL and session info
   ```
4. Set `GOCLAW_TELEGRAM_POLLING=true` for local dev
5. Start gateway: `./mtclaw gateway run --standalone`
6. Verify: send `/start` → bot responds with welcome

**Config Reference** (`internal/config/config_channels.go`):
```go
type TelegramConfig struct {
    Token  string `json:"token"`
    // ... existing fields
}
```

**Env vars needed**:
```bash
GOCLAW_TELEGRAM_TOKEN=     # BotFather token
GOCLAW_TELEGRAM_POLLING=true  # use polling (not webhook) for dev
```

**Validation**:
- [ ] Bot appears online in Telegram
- [ ] `/start` returns welcome message
- [ ] `/help` lists available commands

---

### Task 2: spec-factory SKILL.md (Day 1, included in US-021)

**What**: Create the skill definition file that the skills loader auto-discovers.

**File**: `docs/08-collaborate/skills/spec-factory/SKILL.md`

**Content**: Copy from design doc (`docs/02-design/spec-command-design.md` Section 3.2), adapted:

```markdown
---
name: spec-factory
description: Generate structured specifications from natural language requirements. Governance Rail #1.
---

# Spec Factory — Governance Rail #1

## When This Skill Activates

- User sends `/spec` command
- User asks to "create a spec", "write requirements", "generate a user story"
- PM SOUL receives a requirements-related request

## Output Format

Generate a JSON specification following this schema:

{
  "spec_version": "0.1.0",
  "title": "Short descriptive title",
  "narrative": {
    "as_a": "role",
    "i_want": "feature/capability",
    "so_that": "business value"
  },
  "acceptance_criteria": [
    "Given X, When Y, Then Z"
  ],
  "priority": "P0|P1|P2|P3",
  "estimated_effort": "S|M|L|XL",
  "soul_author": "pm",
  "created_at": "ISO 8601 timestamp"
}

## Process Steps

1. **Clarify** (if input is vague):
   - Ask 1-2 targeted questions maximum
   - Do NOT ask more than 2 questions — generate best-effort spec instead

2. **Generate**:
   - Create spec JSON following the schema above
   - Use Vietnamese for narrative if user wrote in Vietnamese
   - Use BDD format (Given/When/Then) for acceptance criteria

3. **Present**:
   - Show formatted summary to user (not raw JSON)
   - Format: Title, Narrative, Acceptance Criteria list
   - Ask: "Approve, modify, or discard?"

4. **Record**:
   - On approval: Save spec as evidence (write_file to workspace)
   - Link to trace_id for audit trail

## Boundaries

- This skill generates SPECS only — not code, not designs, not test plans
- If user asks for implementation → delegate to @coder
- If user asks for architecture → delegate to @architect
- If user asks for test cases → delegate to @tester

## Vietnamese Support

- Input in Vietnamese → output in Vietnamese
- Input in English → output in English
- Mixed → follow user's primary language
```

**Skills Loader Discovery**:
- GoClaw's `internal/skills/loader.go` scans multiple directories
- For MTClaw, the skill file should also be deployed to the GoClaw skills directory at runtime
- Check: `ListSkills()` returns "spec-factory" after file is placed

**Validation**:
- [ ] File created at correct path
- [ ] `mtclaw skills list` (or equivalent) shows spec-factory
- [ ] Frontmatter parsed correctly (name + description)

---

### Task 3: /spec Command Handler (Day 1-2, 3 points)

**What**: Add `/spec` case to Telegram command handler.

**File to modify**: `internal/channels/telegram/commands.go`

**Location**: Inside `handleBotCommand()` switch statement (currently has: `/help`, `/reset`, `/stop`, `/stopall`, `/status`, `/tasks`, `/task_detail`, `/addwriter`, `/removewriter`, `/writers`)

**Pattern** (follow existing command cases):
```go
case "/spec":
    taskText := strings.TrimPrefix(text, "/spec ")
    if strings.TrimSpace(taskText) == "" || taskText == "/spec" {
        // No description provided
        msg := tu.Message(chatIDObj, "Usage: /spec <requirement description>\n\nExample: /spec Create login feature for Bflow mobile app")
        setThread(msg)
        c.bot.SendMessage(ctx, msg)
        return true
    }

    // Send acknowledgment
    ackMsg := tu.Message(chatIDObj, "Generating spec...")
    setThread(ackMsg)
    c.bot.SendMessage(ctx, ackMsg)

    // Publish to agent loop — PM SOUL handles /spec
    c.publishInbound(ctx, InboundMessage{
        Channel:  "telegram",
        SenderID: senderID,
        ChatID:   chatIDStr,
        Content:  taskText,
        PeerKind: peerKind,
        AgentID:  "pm",  // Route to PM SOUL
        Metadata: map[string]string{
            "command":           "spec",
            "rail":              "spec-factory",
            "local_key":         localKey,
            "is_forum":          fmt.Sprintf("%t", isForum),
            "message_thread_id": fmt.Sprintf("%d", messageThreadID),
        },
    })
    return true
```

**IMPORTANT**: Study existing command handler patterns in `commands.go` before implementing. The exact method signatures and helper functions (`publishInbound`, `setThread`, etc.) may differ — adapt to match.

**Validation**:
- [ ] `/spec` with no args → usage message
- [ ] `/spec Create login feature` → "Generating spec..." then JSON spec response
- [ ] Trace record created in `traces` table

---

### Task 4: Context Anchoring Layer A (Day 2, 2 points)

**What**: Inject session goal and decision log into ExtraPrompt.

**File to modify**: `cmd/gateway_consumers.go` (processNormalMessage function)

**Design** (from SAD Section 8):
- Extract session goal from first message or command
- Track key decisions made during conversation
- Inject into ExtraPrompt string (Section [7] of system prompt)
- Store in session metadata for persistence

**ExtraPrompt Addition**:
```
## Session Context
Goal: {extracted from first user message}
SOUL: {current_soul_display_name} — stay in character per your SOUL.md
```

**Implementation Approach**:
1. Check if session already has a goal in metadata
2. If not, extract goal from current message (first message = session goal)
3. Append to existing ExtraPrompt string
4. Decision log: extract from agent responses that contain decisions/conclusions (can be simplified in Sprint 4 — full implementation Sprint 6)

**Sprint 4 Scope (Minimal Viable Layer A)**:
- Session goal extraction (from first message)
- SOUL identity reminder in ExtraPrompt
- Decision log: DEFERRED to Sprint 6 (too complex for 0.5 day)

**Validation**:
- [ ] ExtraPrompt contains session goal after first message
- [ ] After 10+ messages, SOUL still responds in-character
- [ ] System prompt logs show Layer A content in ExtraPrompt section

---

### Task 5: SOUL Routing — @mention (Day 3, 2 points)

**What**: Enable `@soul_key` mention to switch active SOUL.

**File to modify**: `cmd/gateway_consumers.go` (processNormalMessage)

**Logic**:
```
1. Parse message content for @mention pattern: @{agent_key}
2. If @mention found:
   a. Look up agent by key in agent registry
   b. Override agentID for this message
   c. Strip @mention from content before passing to agent loop
3. For /spec command: always use PM SOUL (already handled in Task 3)
```

**Agent Keys** (from migration 000009 + 000012):
```
pm, architect, coder, reviewer, researcher, writer, pjm, devops,
tester, cto, cpo, ceo, enghelp, sales, cs, assistant, itadmin
```

**Validation**:
- [ ] `@reviewer review this code` → reviewer SOUL responds
- [ ] `@pm create requirements` → PM SOUL responds
- [ ] `@enghelp help me debug` → enghelp SOUL responds
- [ ] No @mention → default SOUL (assistant) handles

---

### Task 6: Evidence Metadata Enrichment (Day 3-4, 2 points)

**What**: Ensure /spec trace records have enriched metadata.

**Investigation Needed**:
- Check how GoClaw's agent loop creates trace records (`internal/tracing/`)
- Determine if command metadata from InboundMessage propagates to trace
- If not automatic, hook into trace creation to inject spec-specific fields

**Required Trace Fields**:
```sql
-- Example query to verify:
SELECT trace_id, name, input_preview, output_preview,
       metadata, total_input_tokens, total_output_tokens,
       tenant_id, agent_key
FROM traces
WHERE name = 'spec-factory'
  AND tenant_id = 'mts'
ORDER BY created_at DESC;
```

**If metadata propagation is NOT automatic**:
- May need to set trace name based on active skill
- May need to copy command metadata to trace metadata
- Check: `internal/tracing/tracer.go` or similar for trace creation API

**Validation**:
- [ ] Trace record exists after /spec invocation
- [ ] `name` = 'spec-factory' (not generic 'chat' or similar)
- [ ] `metadata` contains command, spec_version, spec_title
- [ ] `total_input_tokens` and `total_output_tokens` > 0

---

### Task 7: make souls-validate (Day 1, included in US-020)

**What**: Add Makefile target to validate SOUL.md character budget.

**File**: `Makefile` (create or append)

**Target**:
```makefile
.PHONY: souls-validate
souls-validate:
	@echo "Validating SOUL files..."
	@for f in docs/08-collaborate/souls/SOUL-*.md; do \
		chars=$$(wc -c < "$$f"); \
		if [ "$$chars" -gt 2500 ]; then \
			echo "FAIL: $$f ($$chars chars, max 2500)"; \
			exit 1; \
		else \
			echo "OK: $$f ($$chars chars)"; \
		fi \
	done
	@echo "All SOUL files within budget."
```

**Note**: 2,500 chars (not 2,000) — includes frontmatter overhead. CTO-3 intent is body content ~2,000 chars.

**Validation**:
- [ ] `make souls-validate` runs without error
- [ ] All 17 SOUL files pass budget check

---

### Task 8: Seed IT Admin SOUL (Day 1, 1 point — CEO Directive)

**What**: Create migration to seed IT Admin SOUL into database.

**Files to create**:
- `migrations/000012_seed_itadmin_soul.up.sql`
- `migrations/000012_seed_itadmin_soul.down.sql`

**Pattern**: Follow exactly `migrations/000009_seed_mtclaw_souls.up.sql`

**Agent Record**:
```sql
INSERT INTO agents (id, agent_key, display_name, agent_type, owner_id, provider, model, ...)
VALUES (gen_random_uuid(), 'itadmin', 'IT Admin', 'predefined', 'mts', 'bflow-ai-platform', 'qwen3:14b', ...);
```

**Context Files** (3 files, same pattern as other SOULs):
1. **SOUL.md** — Extract from `docs/08-collaborate/souls/SOUL-itadmin.md`:
   - Include: Identity + Capabilities + Constraints sections
   - Exclude: Operations Playbooks, Server Specs, Service URLs (too long for system prompt; use RAG instead)
   - Target: ~2,000 chars body content
2. **IDENTITY.md** — Short identity card:
   ```
   Name: IT Admin
   Emoji: 🖥️
   Vibe: Reliable infrastructure guardian — calm, methodical, always has a rollback plan
   ```
3. **AGENTS.md** — Reuse shared governance AGENTS.md (same as other SOULs)

**Agent Links** (delegation):
```sql
-- itadmin ↔ devops (mutual infrastructure delegation)
INSERT INTO agent_links (from_agent_id, to_agent_id, ...)
  SELECT a1.id, a2.id, ... FROM agents a1, agents a2
  WHERE a1.agent_key = 'itadmin' AND a2.agent_key = 'devops' AND a1.owner_id = 'mts';

INSERT INTO agent_links (from_agent_id, to_agent_id, ...)
  SELECT a1.id, a2.id, ... FROM agents a1, agents a2
  WHERE a1.agent_key = 'devops' AND a2.agent_key = 'itadmin' AND a1.owner_id = 'mts';
```

**Team Assignment**:
```sql
-- Add itadmin to "MTS Engineering" team
INSERT INTO agent_team_members (team_id, agent_id)
  SELECT t.id, a.id FROM agent_teams t, agents a
  WHERE t.name = 'MTS Engineering' AND a.agent_key = 'itadmin' AND a.owner_id = 'mts';
```

**Down Migration**:
```sql
DELETE FROM agent_team_members WHERE agent_id IN (SELECT id FROM agents WHERE agent_key = 'itadmin' AND owner_id = 'mts');
DELETE FROM agent_links WHERE from_agent_id IN (SELECT id FROM agents WHERE agent_key = 'itadmin' AND owner_id = 'mts')
   OR to_agent_id IN (SELECT id FROM agents WHERE agent_key = 'itadmin' AND owner_id = 'mts');
DELETE FROM agent_context_files WHERE agent_id IN (SELECT id FROM agents WHERE agent_key = 'itadmin' AND owner_id = 'mts');
DELETE FROM agents WHERE agent_key = 'itadmin' AND owner_id = 'mts';
```

**Validation**:
- [ ] `./mtclaw migrate up` succeeds
- [ ] `GET /v1/agents` returns 17 agents
- [ ] `@itadmin` mention routes correctly
- [ ] SOUL.md content injected into system prompt

---

## Environment Setup for Sprint 4

```bash
# Required env vars (add to .env):
GOCLAW_TELEGRAM_TOKEN=          # From BotFather
GOCLAW_TELEGRAM_POLLING=true    # Dev mode
GOCLAW_BFLOW_API_KEY=aip_...    # Already provisioned
GOCLAW_BFLOW_BASE_URL=http://ai-platform:8120/api/v1  # Local Docker
BFLOW_TENANT_ID=mts             # MTS tenant
GOCLAW_PROVIDER=bflow-ai-platform
GOCLAW_MODEL=qwen3:14b

# Build + run:
export PATH=$HOME/.local/go/bin:$PATH
go build -o mtclaw .
./mtclaw migrate up
./mtclaw gateway run --standalone

# Verify:
curl http://localhost:8080/v1/agents  # 17 agents
# Send /start in Telegram → bot responds
# Send /spec Create login feature → spec generated
```

---

## Files to Create

| File | Purpose |
|------|---------|
| `docs/08-collaborate/skills/spec-factory/SKILL.md` | Spec factory skill definition |
| `migrations/000012_seed_itadmin_soul.up.sql` | IT Admin SOUL seeding (CEO directive) |
| `migrations/000012_seed_itadmin_soul.down.sql` | IT Admin SOUL rollback |
| `docs/04-build/SPRINT-004-FEEDBACK-RESULTS.md` | Feedback session findings (Day 5) |

## Files to Modify

| File | Changes |
|------|---------|
| `internal/channels/telegram/commands.go` | Add `/spec` case |
| `cmd/gateway_consumers.go` | @mention routing + Layer A ExtraPrompt |
| `Makefile` | Add `souls-validate` target |
| `.env.example` | Add `GOCLAW_TELEGRAM_TOKEN`, `GOCLAW_TELEGRAM_POLLING` |

## Files NOT to Modify

| File | Reason |
|------|--------|
| `internal/skills/loader.go` | Already loads arbitrary SKILL.md files |
| `internal/skills/search.go` | BM25 search works for skill discovery |
| `internal/agent/systemprompt.go` | Auto-injects skills section |
| `internal/agent/loop.go` | No command-specific logic needed |
| `internal/providers/bflow_ai.go` | Already complete from Sprint 3 |
| `internal/middleware/tenant.go` | Already complete from Sprint 3 |

---

## Key Code Paths to Study Before Implementing

1. **Command handling pattern**: `internal/channels/telegram/commands.go` — study existing `/help`, `/reset` cases for exact method signatures
2. **Message publishing**: `internal/channels/telegram/handlers.go` — how InboundMessage is published to bus
3. **Consumer processing**: `cmd/gateway_consumers.go` — how processNormalMessage routes to agent loop
4. **Skills loading**: `internal/skills/loader.go` — 5-tier discovery hierarchy
5. **System prompt building**: `internal/agent/systemprompt.go` — 13-section prompt structure
6. **Trace creation**: `internal/tracing/` — how traces are recorded

---

## Zero Mock Policy Exceptions

None for Sprint 4. All implementations must be production-ready:
- Real Telegram bot connection (not mock)
- Real Bflow AI-Platform calls (not mock)
- Real trace records in PostgreSQL (not mock)

---

## References

- [Sprint 4 Plan](sprints/SPRINT-004-Core-Deploy-Spec.md) — full sprint plan
- [/spec Command Design](../02-design/spec-command-design.md) — detailed design
- [System Architecture Document](../02-design/system-architecture-document.md) — Section 8 Context Drift
- [SOUL Loading Implementation Plan](../02-design/soul-loading-implementation-plan.md)
- [Sprint 3 Plan](sprints/SPRINT-003-Architecture-RLS.md) — predecessor
