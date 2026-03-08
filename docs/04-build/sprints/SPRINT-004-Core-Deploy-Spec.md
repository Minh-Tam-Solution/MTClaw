# Sprint 004 — Core Deploy + /spec Prototype (Rail #1)

**Sprint**: 4 of 10
**Duration**: 5 days
**Phase**: Phase 1 — Foundation + First Rails
**Gate**: None (mid-phase sprint)
**Predecessor**: Sprint 3 (G2 APPROVED — CTO 9.2/10)
**Status**: NOT STARTED
**Owner**: [@coder] (implementation) + [@pm] (feedback session)
**Points**: ~12

---

## Sprint Goal

> Connect Telegram channel, implement /spec command handler (Rail #1 prototype),
> add Context Anchoring Layer A, enable SOUL routing, seed IT Admin SOUL (CEO directive),
> and validate with real MTS users in a 15-minute feedback session.

---

## Entry Criteria

- [x] G2 APPROVED (CTO 9.2/10, 2026-03-02)
- [x] Sprint 3 Tasks 1-5 implemented (RLS, SOUL seeding, observability, Bflow provider, MTS deploy config)
- [x] Sprint 3 CTO review: 8.5/10 APPROVED, P1 bug fixed (HasAnyProvider)
- [x] /spec Command Design reviewed and approved (`docs/02-design/spec-command-design.md`)
- [x] Bflow AI-Platform API key provisioned and verified (aip_c786)
- [x] 16 SOULs seeded in database (migration 000009)
- [x] Bflow AI-Platform provider registered (migration 000011 + config path)
- [x] IT Admin SOUL template drafted (`docs/08-collaborate/souls/SOUL-itadmin.md`) — CEO directive

---

## CTO/CPO Issues to Address (from G0.2 + G2 Reviews)

| Issue | Source | Action | Sprint 4 Task |
|-------|--------|--------|---------------|
| CTO-1: SystemPromptMode minimal strips SOUL context | G0.2 | Document behavior + test with qwen3:14b | US-022 SOUL routing |
| CTO-3: SOUL.md 2,000 char budget enforcement | G0.2 | Add `make souls-validate` check | US-020 Day 1 |
| CTO ISSUE-B: token_usage via traces fields | G2 | Verify traces.total_input_tokens populated | US-021 evidence |
| CPO CONCERN-1: Sprint 4 validation plan detail | G0.2 | SOUL feedback session protocol | US-023 |
| CPO CONCERN-2: Sales RAG needs minimal content | G0.2 | Prepare 5-10 sales docs (Sprint 6 prep) | Out of scope |
| CPO CONCERN-3: Manual smoke-test 16 SOULs | G0.2 | Test during feedback session | US-023 |

---

## User Stories

### US-020: Telegram Channel Setup

**As a** MTS employee
**I want** to interact with MTClaw via Telegram
**So that** I can access governance-aware AI from my daily messaging app

**Acceptance Criteria**:
- [ ] Telegram bot registered via BotFather, token stored in `GOCLAW_TELEGRAM_TOKEN`
- [ ] Bot responds to `/start` with personalized welcome message
- [ ] Default SOUL routing works (assistant SOUL, `is_default=true`)
- [ ] Bot handles DM and group conversations
- [ ] `/help` lists available commands including `/spec`
- [ ] `make souls-validate` checks SOUL.md char budget (CTO-3)

**Points**: 1 | **Priority**: P0 | **Day**: 1

**Implementation Notes**:
- GoClaw already has Telegram channel infrastructure in `internal/channels/telegram/`
- Config: `GOCLAW_TELEGRAM_TOKEN` env var → `config.Channels.Telegram.Token`
- Webhook vs polling: use polling for dev (`GOCLAW_TELEGRAM_POLLING=true`), webhook for production
- Bot commands to register with BotFather: `/start`, `/help`, `/spec`, `/reset`, `/status`

---

### US-021: /spec Command Handler (Rail #1 Prototype)

**As a** [@pm]
**I want** `/spec` command to produce structured JSON specifications
**So that** requirements are standardized and evidence is captured

**Acceptance Criteria**:
- [ ] `/spec {description}` → PM SOUL generates JSON spec (Sprint 4 schema v0.1.0)
- [ ] Output fields: title, narrative (As a/I want/So that), acceptance criteria (Given/When/Then), priority, effort, soul_author, created_at
- [ ] Vietnamese input → Vietnamese output (bidirectional)
- [ ] Acknowledgment: "Generating spec..." sent immediately after command
- [ ] Evidence: trace record created with `name='spec-factory'`, metadata contains spec_version + title
- [ ] User prompted: "Approve, modify, or discard?"
- [ ] Latency: <30s p95 (LLM call dominates)

**Points**: 3 | **Priority**: P0 | **Day**: 1-3

**Design Reference**: [spec-command-design.md](../../02-design/spec-command-design.md)

**Implementation Path**:

```
/spec flow — 3 components:

1. SKILL.MD (new file)
   docs/08-collaborate/skills/spec-factory/SKILL.md
   - Frontmatter: name: spec-factory
   - Body: spec generation instructions + JSON schema
   - Quality criteria from SOUL rubric
   - Auto-discovered by skills loader (5-tier hierarchy)

2. COMMAND HANDLER (modify existing)
   internal/channels/telegram/commands.go
   - Add case "/spec" to handleBotCommand() switch
   - Extract task text: strings.TrimPrefix(text, "/spec ")
   - Publish InboundMessage with metadata: {command: "spec", rail: "spec-factory"}
   - Send acknowledgment: "Generating spec..."
   - Return true (command consumed)

3. AGENT LOOP (no changes needed)
   - Skills loader discovers spec-factory SKILL.md automatically
   - System prompt includes <available_skills> section with spec-factory
   - PM SOUL reads skill instructions → generates JSON spec
   - Trace record auto-created by agent loop
   - Response sent back to Telegram via existing callback pipeline
```

**Sprint 4 JSON Schema (v0.1.0)**:
```json
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
```

---

### US-022: SOUL Routing

**As a** MTS employee
**I want** my questions automatically routed to the right SOUL
**So that** I get role-appropriate answers without manual SOUL selection

**Acceptance Criteria**:
- [ ] Default SOUL: `assistant` (is_default=true) routes to appropriate SOUL via delegation
- [ ] Explicit `@mention` switching: `@reviewer review this code` → reviewer SOUL
- [ ] `@pm` mention → PM SOUL handles directly
- [ ] `/spec` always routes to PM SOUL (regardless of current SOUL)
- [ ] Delegation via `spawn(agent="pm")` works within agent loop
- [ ] No server restart needed when switching SOULs within a session
- [ ] CTO-1 verified: test SystemPromptMode behavior with qwen3:14b

**Points**: 2 | **Priority**: P0 | **Day**: 3

**Implementation Notes**:
- GoClaw already has delegation/spawn system in `internal/pairing/`
- Agent links (migration 000009) define delegation permissions
- SOUL routing table from design doc:

| Current SOUL | `/spec` Action |
|-------------|----------------|
| `pm` | Handle directly (primary owner) |
| `assistant` | Delegate to `pm` via spawn() |
| `coder` | Delegate to `pm` |
| Any other | Delegate to `pm` |

- `@mention` routing: detect `@{agent_key}` prefix in message → set agentID override
- Implementation location: `cmd/gateway_consumers.go` (processNormalMessage)

---

### US-023: SOUL Feedback Session

**As a** [@cpo]
**I want** real MTS users to test their SOULs and provide feedback
**So that** we validate SOUL quality before wider pilot rollout (Sprint 5)

**Acceptance Criteria**:
- [ ] 3-4 MTS users recruited: min 1 Engineering + 1 non-Engineering (Sales or Back Office)
- [ ] 15-minute sessions with assigned tasks per persona:
  - Engineering: ask code review question, try `/spec`
  - Sales: ask about pricing, try general assistant
  - Back Office: ask HR policy question
- [ ] Measure per session:
  - Time to first useful answer (target: <60s)
  - User satisfaction (1-5 scale)
  - Would-use-again (Yes/Maybe/No)
  - SOUL accuracy: did the right SOUL handle the request?
- [ ] Findings documented in `docs/04-build/SPRINT-004-FEEDBACK-RESULTS.md`
- [ ] SOUL tuning recommendations for Sprint 5

**Points**: 1 | **Priority**: P1 | **Day**: 4-5

**CPO Exit Criteria** (from G0.2 OBS-1):
- If 0/4 testers complete `/spec` flow without confusion → **P0 fix required before Sprint 5**
- If 2+/4 testers complete successfully → proceed to Sprint 5 as planned
- If blocking UX issue found → Sprint 5 scope adjusts (pilot delays, fix prioritized)

**Session Protocol**:
```
Preparation (Day 4 morning):
  1. Deploy MTClaw to staging (Docker Compose + ai-net)
  2. Create Telegram bot link for testers
  3. Prepare task cards per persona
  4. Set up screen recording (optional)

Session Flow (Day 4-5, 15 min each):
  1. [2 min] Introduction: "This is MTClaw, our new AI assistant"
  2. [3 min] Free exploration: "Ask any work question"
  3. [5 min] Directed task: persona-specific task card
  4. [3 min] /spec test: "Create a spec for [relevant topic]"
  5. [2 min] Feedback: satisfaction score + would-use-again

Post-Session (Day 5):
  1. Aggregate scores
  2. Document findings + SOUL tuning recommendations
  3. Go/No-Go decision for Sprint 5 pilot
```

---

### US-024: Context Anchoring Layer A

**As a** developer
**I want** session-level context anchoring (goal + decision log)
**So that** SOUL identity and task focus are maintained across long conversations

**Acceptance Criteria**:
- [ ] Session goal injected into ExtraPrompt Section [7] of system prompt
- [ ] Decision log: key decisions in conversation tracked and re-injected
- [ ] Does NOT re-implement SOUL.md injection (already in sections [2-4] via agent_context_files)
- [ ] Layer A only adds NEW anchoring context, complementary to existing SOUL injection
- [ ] Verified: after 20+ messages, SOUL still responds in-character

**Points**: 2 | **Priority**: P0 | **Day**: 2

**Design Reference**: [System Architecture Document Section 8](../../02-design/system-architecture-document.md) (Context Drift Prevention)

**Implementation Notes**:
- ExtraPrompt is built in `cmd/gateway_consumers.go` (processNormalMessage)
- Current ExtraPrompt includes: group chat context note, topic system prompt, topic skills
- Layer A adds: session goal (extracted from first message or /spec command), decision log (auto-extracted from agent responses containing decisions/conclusions)
- Storage: session metadata in sessions table (GoClaw already has metadata field)
- Phasing: Layer A (Sprint 4) → Layer B SOUL-Aware RAG (Sprint 6) → Layer C Evidence (Sprint 7)

```
ExtraPrompt format (Layer A):

## Session Context
Goal: {extracted goal from first message or command}
Key Decisions:
- {decision 1 from earlier in conversation}
- {decision 2}

Note: You are {soul_display_name}. Stay in character.
Refer to your SOUL.md persona for tone and boundaries.
```

---

### US-025: Evidence Attachment via trace_id

**As a** [@pm]
**I want** every /spec invocation to produce an evidence record
**So that** governance actions are auditable and traceable

**Acceptance Criteria**:
- [ ] Trace record created for every /spec invocation
- [ ] Trace fields: `name='spec-factory'`, `input_preview` = user's spec request (truncated), `output_preview` = spec JSON (first 500 chars)
- [ ] Trace metadata: `{command: "spec", spec_version: "0.1.0", spec_title: "...", soul_author: "pm"}`
- [ ] Token usage populated: `total_input_tokens`, `total_output_tokens` (CTO ISSUE-B verification)
- [ ] Queryable: `SELECT * FROM traces WHERE name = 'spec-factory' AND tenant_id = 'mts'`

**Points**: 2 | **Priority**: P1 | **Day**: 3-4

**Implementation Notes**:
- GoClaw agent loop already creates traces automatically in `internal/tracing/`
- The `name` field needs to be set based on the active skill/command
- metadata enrichment: check if agent loop propagates command metadata to trace record
- If not automatic, may need to hook into trace creation to inject spec-specific metadata

---

### US-026: Seed IT Admin SOUL (CEO Directive)

**As a** IT Admin
**I want** an IT Admin SOUL available in MTClaw
**So that** infrastructure questions are handled by a specialized AI persona

**Acceptance Criteria**:
- [ ] Migration `000012_seed_itadmin_soul.up.sql` creates:
  - 1 agent record: `agent_key='itadmin'`, `display_name='IT Admin'`, `agent_type='predefined'`, `owner_id='mts'`, `provider='bflow-ai-platform'`, `model='qwen3:14b'`
  - 3 agent_context_files: SOUL.md (from `SOUL-itadmin.md`), IDENTITY.md, AGENTS.md (shared)
  - Agent links: itadmin ↔ devops (mutual delegation)
  - Add itadmin to "MTS Engineering" team
- [ ] Down migration removes itadmin agent and context files cleanly
- [ ] `GET /v1/agents` returns 17 agents (was 16)
- [ ] `@itadmin` mention routes to IT Admin SOUL
- [ ] SOUL.md content within 2,500 char budget for system prompt injection

**Points**: 1 | **Priority**: P1 | **Day**: 1

**Implementation Notes**:
- Follow exact pattern from `000009_seed_mtclaw_souls.up.sql`
- SOUL template already exists: `docs/08-collaborate/souls/SOUL-itadmin.md`
- Template is ~8,600 chars — need to extract core sections (Identity + Capabilities + Constraints) for agent_context_files, keeping within char budget
- Operations playbooks section can be referenced via RAG instead of system prompt injection

---

## Daily Plan

### Day 1: Telegram Setup + Skill Definition + IT Admin Seed

| Task | Owner | Output |
|------|-------|--------|
| Register Telegram bot (BotFather), configure token | [@coder] | Bot responding to `/start` |
| Configure `GOCLAW_TELEGRAM_TOKEN` in `.env` | [@coder] | Env var set |
| Create `spec-factory/SKILL.md` | [@coder] | Skill file with schema + instructions |
| Add `make souls-validate` target (CTO-3) | [@coder] | Char budget check |
| Verify skills loader discovers spec-factory | [@coder] | `mtclaw skills list` shows it |
| Create migration `000012_seed_itadmin_soul` (CEO) | [@coder] | IT Admin SOUL in DB |
| Start /spec command handler implementation | [@coder] | Case added to commands.go |

### Day 2: /spec Handler Complete + Context Anchoring

| Task | Owner | Output |
|------|-------|--------|
| Complete /spec command handler in `commands.go` | [@coder] | InboundMessage published |
| Implement Context Anchoring Layer A | [@coder] | ExtraPrompt enriched |
| PM SOUL routing: /spec always → PM | [@coder] | Routing verified |
| Manual test: `/spec Create login feature` → JSON | [@coder] | Spec generated |

### Day 3: SOUL Routing + Evidence

| Task | Owner | Output |
|------|-------|--------|
| @mention SOUL switching (`@reviewer`, `@pm`) | [@coder] | Delegation works |
| Evidence attachment: trace metadata enrichment | [@coder] | Trace records queryable |
| Verify token usage in traces (CTO ISSUE-B) | [@coder] | Token counts populated |
| CTO-1 test: SystemPromptMode with qwen3:14b | [@coder] | Behavior documented |

### Day 4: Integration Test + Feedback Prep

| Task | Owner | Output |
|------|-------|--------|
| Integration test: full `/spec` flow end-to-end | [@coder] | Telegram → JSON → trace |
| Deploy staging (Docker Compose + ai-net) | [@coder] | Accessible for testers |
| Prepare feedback session materials | [@pm] | Task cards, session protocol |
| Begin SOUL feedback sessions (1-2 testers) | [@pm] | Recorded results |

### Day 5: Feedback Sessions + Sprint Close

| Task | Owner | Output |
|------|-------|--------|
| Complete feedback sessions (remaining testers) | [@pm] | All 3-4 sessions done |
| Document findings | [@pm] | `SPRINT-004-FEEDBACK-RESULTS.md` |
| Go/No-Go decision for Sprint 5 pilot | [@pm] + [@cpo] | Decision documented |
| Sprint 4 completion report | [@pm] | Sprint summary |

---

## Verification Checklist (DoD)

### Code Deliverables
- [ ] Telegram bot responds to `/start`, `/help`, `/spec`
- [ ] `/spec Create login feature` → structured JSON spec with title, narrative, acceptance criteria
- [ ] Vietnamese input → Vietnamese output
- [ ] SOUL routing: @mention delegation + /spec → PM automatic
- [ ] Context Anchoring Layer A: session goal in ExtraPrompt
- [ ] Evidence: trace record with name='spec-factory' and enriched metadata
- [ ] Token usage populated in traces (CTO ISSUE-B)
- [ ] `make souls-validate` enforces 2,000 char budget (CTO-3)
- [ ] IT Admin SOUL seeded: `GET /v1/agents` returns 17 agents (CEO directive)

### Validation
- [ ] 3-4 MTS users complete feedback session
- [ ] 2+/4 testers complete /spec flow without confusion (CPO exit criteria)
- [ ] No blocking UX issues identified
- [ ] Findings documented with SOUL tuning recommendations

### Quality
- [ ] `go vet ./...` passes
- [ ] `go build -o /dev/null .` succeeds
- [ ] All existing API endpoints still work (regression)
- [ ] Spec generation latency <30s (p95)

---

## Risk Register (Sprint 4 Specific)

| # | Risk | Prob | Impact | Mitigation |
|---|------|------|--------|------------|
| R3 | Bflow AI-Platform latency >5s | Low | Med | Already tested <5s (Sprint 3 verify); graceful degradation |
| R4 | MTS adoption <30% (feedback session) | Med | High | Iterate on SOUL tuning; 2+/4 success = proceed |
| R9 | Context drift in /spec (PM SOUL loses role) | Med | Med | Layer A anchoring; session goal injection |
| R12 | qwen3:14b poor JSON generation | Med | Med | Test structured output; fallback to plain text + manual extract |
| R13 | Telegram webhook unreliable on VPS | Low | Med | Use polling for dev; webhook + retry for production |

---

## Sprint 5 Dependencies (What Sprint 4 Unblocks)

| Sprint 5 Task | Sprint 4 Prerequisite |
|---------------|----------------------|
| PR Gate skill | Telegram channel working + skill system verified |
| MTS pilot (10 users) | Feedback session validates UX; bot accessible |
| Token cost tracking | traces.total_input_tokens populated (US-025) |
| MTS staging deployment | Docker Compose tested in feedback session |

---

## Architecture Reference

```
Sprint 4 Component Additions (★ = new)

┌─────────────────────────────────────────────────────┐
│ Telegram Bot API                                     │
│   /start  /help  /spec★  /reset  @mention★           │
└────────────────────┬────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────┐
│ GoClaw Gateway                                       │
│                                                       │
│ ┌───────────────────────┐  ┌──────────────────────┐ │
│ │ Telegram Handler      │  │ Message Bus          │ │
│ │ commands.go (+/spec)★ │──│ InboundMessage       │ │
│ └───────────────────────┘  └──────────┬───────────┘ │
│                                       │              │
│ ┌─────────────────────────────────────▼────────────┐ │
│ │ Gateway Consumer                                  │ │
│ │ • SOUL routing (@mention → agentID override)★    │ │
│ │ • ExtraPrompt (+ Layer A anchoring)★             │ │
│ │ • Scheduler → Agent Loop                          │ │
│ └─────────────────────────────────────┬────────────┘ │
│                                       │              │
│ ┌─────────────────────────────────────▼────────────┐ │
│ │ Agent Loop (PM SOUL)                              │ │
│ │ • System prompt: SOUL.md + IDENTITY.md + AGENTS  │ │
│ │ • Skills: spec-factory SKILL.md★                  │ │
│ │ • ExtraPrompt: session goal + decision log★       │ │
│ │ • LLM call → JSON spec                           │ │
│ │ • Trace record with evidence metadata★            │ │
│ └─────────────────────────────────────┬────────────┘ │
│                                       │              │
│ ┌─────────────────────────────────────▼────────────┐ │
│ │ Bflow AI-Platform (qwen3:14b)                     │ │
│ │ POST /v1/chat/completions                         │ │
│ │ X-API-Key + X-Tenant-ID auth                      │ │
│ └──────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────┘
```

---

## References

- [Sprint 3 Plan](SPRINT-003-Architecture-RLS.md) — predecessor (G2 APPROVED 9.2/10)
- [G2 Gate Approval](../../00-foundation/G2-GATE-APPROVAL.md)
- [/spec Command Design](../../02-design/spec-command-design.md) — Rail #1 detailed design
- [System Architecture Document](../../02-design/system-architecture-document.md) — Section 8: Context Drift
- [SOUL Loading Implementation Plan](../../02-design/soul-loading-implementation-plan.md)
- [ADR-005: Bflow AI-Platform](../../02-design/01-ADRs/SPEC-0005-ADR-005-Bflow-AI-Platform-Integration.md)
- [User Stories](../../01-planning/user-stories.md) — US-020 to US-023
- [Roadmap](../../01-planning/roadmap.md) — Sprint 4 section
