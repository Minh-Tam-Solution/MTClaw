# SOUL Loading Implementation Plan — MTClaw

**SDLC Stage**: 02-Design
**Version**: 1.0.0
**Date**: 2026-03-02
**Author**: [@pm] + [@researcher]
**Source**: GoClaw source analysis (systemprompt.go, bootstrap/, store/, migrations/)

---

## Executive Summary

[@researcher]: Analysis of GoClaw's codebase reveals a **mature bootstrap/context file architecture** that directly supports SOUL loading. GoClaw already has:

1. **`agent_context_files` table** — stores SOUL.md, IDENTITY.md per agent in PostgreSQL
2. **`user_context_files` table** — per-user overrides (personalization layer)
3. **Bootstrap file system** — 8 standard files loaded into system prompt
4. **`BuildSystemPrompt()`** — 15-section system prompt builder with `ContextFiles` injection

**Conclusion**: MTClaw's 16 SOULs can be loaded using GoClaw's **existing architecture** with minimal code changes. The primary work is data seeding (populating `agent_context_files`) and governance extensions (ExtraPrompt for rail context).

---

## 1. GoClaw System Prompt Architecture

### 1.1 System Prompt Construction Flow

`BuildSystemPrompt()` in `internal/agent/systemprompt.go` constructs 15 sections:

```
Section  1: Identity        → "You are a personal assistant running inside GoClaw."
Section  2: Bootstrap        → First-run override (BOOTSTRAP.md detected)
Section  3: Tooling          → Available tools list with descriptions
Section  4: Safety           → Safety directives (no self-preservation, etc.)
Section  5: Skills           → Inline XML or skill_search mode (full only)
Section  6: Memory Recall    → memory_search + memory_get instructions (full only)
Section  7: Workspace        → Working directory path
Section  8: Sandbox          → Docker sandbox info (if enabled)
Section  9: User Identity    → Owner sender IDs (full only)
Section 10: Time             → Current timestamp
Section 11: Messaging        → Channel-specific messaging rules (full only)
Section 12: ExtraPrompt      → Additional context (wrapped in <extra_context> tags)
Section 13: Project Context  → Bootstrap files: SOUL.md, IDENTITY.md, AGENTS.md, etc.
Section 14: Silent Replies   → When to use NO_REPLY (full only)
Section 15: Runtime          → Agent ID, model, channel info
```

### 1.2 Bootstrap Files (Standard Set)

| File | Purpose | Load Mode |
|------|---------|-----------|
| `AGENTS.md` | Operating instructions (workspace rules) | Every session |
| `SOUL.md` | Persona, tone, boundaries, expertise | Every session |
| `IDENTITY.md` | Name, emoji, creature, vibe | Every session |
| `USER.md` | User profile (per-user) | Full only |
| `TOOLS.md` | Local tool notes | Full only |
| `HEARTBEAT.md` | Periodic check tasks | Full only |
| `BOOTSTRAP.md` | First-run ritual (one-time) | Full only |
| `MEMORY.md` | Long-term curated memory | Full only |

### 1.3 Two Loading Paths

```
Path A: Filesystem (standalone mode)
  LoadWorkspaceFiles(dir) → reads .md files from disk
  → FilterForSession(files, key) → full or minimal set
  → BuildContextFiles(files, cfg) → truncated ContextFile[]
  → SystemPromptConfig.ContextFiles

Path B: Database (managed/predefined mode) ★ MTClaw will use this
  LoadFromStore(ctx, agentStore, agentID) → reads agent_context_files from DB
  → returns ContextFile[] directly (already truncated)
  → SystemPromptConfig.ContextFiles
```

### 1.4 Agent Types

| Type | Context Source | SOUL Files | User Files |
|------|---------------|------------|------------|
| `open` | Per-user (user_context_files) | Seeded per-user from templates | Per-user |
| `predefined` | Shared (agent_context_files) + per-user overrides | Seeded at agent-level from templates | USER.md + BOOTSTRAP.md only |

**MTClaw SOULs = `predefined` type** — shared SOUL content, per-user USER.md.

---

## 2. Database Schema for SOUL Loading

### 2.1 Primary Tables

#### `agents` — Agent registry (SOUL identity)

| Column | Type | SOUL Purpose |
|--------|------|-------------|
| `id` | UUID v7 | Primary key |
| `agent_key` | VARCHAR(100) UNIQUE | SOUL identifier: `pm`, `mts-sales`, `cto`, etc. |
| `display_name` | VARCHAR(255) | Human-facing: "Product Manager", "MTS Sales Assistant" |
| `frontmatter` | TEXT | Short expertise summary for discovery/routing |
| `agent_type` | VARCHAR(20) | `predefined` for all 16 MTClaw SOULs |
| `owner_id` | VARCHAR(255) | Tenant isolation (→ `mts` for Phase 1) |
| `provider` | VARCHAR(50) | Default LLM provider (→ Bflow AI-Platform) |
| `model` | VARCHAR(200) | Default model (→ `qwen3:14b`) |
| `other_config` | JSONB | Description, thinking_level, etc. |
| `embedding` | vector(1536) | Semantic search for agent discovery/routing |
| `tsv` | tsvector | Full-text search (display_name + frontmatter) |

#### `agent_context_files` — SOUL content (shared)

| Column | Type | SOUL Purpose |
|--------|------|-------------|
| `agent_id` | UUID FK | Links to agents.id |
| `file_name` | VARCHAR(255) | `SOUL.md`, `IDENTITY.md`, `AGENTS.md` |
| `content` | TEXT | Full SOUL content (persona, expertise, boundaries) |
| UNIQUE | (agent_id, file_name) | One file per name per agent |

#### `user_context_files` — Per-user overrides

| Column | Type | SOUL Purpose |
|--------|------|-------------|
| `agent_id` | UUID FK | Links to agents.id |
| `user_id` | VARCHAR(255) | Telegram user ID, etc. |
| `file_name` | VARCHAR(255) | `USER.md`, `BOOTSTRAP.md` |
| `content` | TEXT | Per-user customization |
| UNIQUE | (agent_id, user_id, file_name) | One file per name per user per agent |

### 2.2 Supporting Tables

| Table | SOUL Relevance |
|-------|---------------|
| `agent_shares` | Access control: which users can interact with which SOULs |
| `user_agent_profiles` | Track first_seen_at, workspace per user per SOUL |
| `user_agent_overrides` | Per-user provider/model override (e.g., prefer Claude over Ollama) |
| `agent_links` | Delegation permissions: which SOULs can delegate to which |
| `handoff_routes` | Temporary routing override (escalation: `mts-cs` → `cto`) |
| `agent_teams` | SOUL team grouping (e.g., "MTS Engineering Team") |
| `delegation_history` | Audit trail of SOUL-to-SOUL delegations |

---

## 3. SOUL Loading Flow (Detailed)

### 3.1 Runtime Flow (Per Request)

```
User sends message via Telegram
  │
  ▼
1. Channel handler (internal/channels/telegram/handlers.go)
  │  Extract: sender_id, chat_id, message text
  │
  ▼
2. Session resolver (internal/gateway/)
  │  Lookup: sessions WHERE session_key = "telegram:{chat_id}"
  │  → Get agent_id from session (or default agent for user)
  │
  ▼
3. Agent loader (internal/store/pg/agents.go)
  │  SELECT * FROM agents WHERE id = {agent_id}
  │  → AgentData: agent_key, model, provider, agent_type, etc.
  │
  ▼
4. Context file loader (internal/bootstrap/load_store.go)
  │  a) LoadFromStore(ctx, store, agentID)
  │     → SELECT * FROM agent_context_files WHERE agent_id = {agent_id}
  │     → Returns: SOUL.md, IDENTITY.md, AGENTS.md content
  │
  │  b) Load user_context_files WHERE agent_id = ? AND user_id = ?
  │     → USER.md (per-user profile)
  │     → Merge: user files shadow agent files with same name
  │
  ▼
5. System prompt builder (internal/agent/systemprompt.go)
  │  BuildSystemPrompt(SystemPromptConfig{
  │    AgentID:      agent.ID,
  │    Model:        agent.Model,
  │    ContextFiles: contextFiles,    // ← SOUL content injected here
  │    ExtraPrompt:  governanceCtx,   // ← Rail context injected here
  │    ToolNames:    [...],
  │    ...
  │  })
  │
  ▼
6. LLM call (internal/providers/)
  │  POST /v1/chat/completions to Bflow AI-Platform
  │  Headers: X-API-Key, X-Tenant-ID: mts
  │  Body: { model: "qwen3:14b", messages: [{role: "system", content: systemPrompt}, ...] }
  │
  ▼
7. Response → Telegram
```

### 3.2 SOUL.md Content Structure (Per SOUL)

Each SOUL's `SOUL.md` in `agent_context_files` will contain:

```markdown
# SOUL.md — {SOUL Role Name}

## Core Identity
You are the {role} for MTClaw. {one-line mission}.

## Expertise
- {domain area 1}
- {domain area 2}
- {domain area 3}

## Boundaries
- {what this SOUL does NOT do}
- {when to delegate to another SOUL}

## Response Style
- {tone: formal/informal/technical}
- {language: Vietnamese preferred, English acceptable}
- {format preferences}

## Delegation Rules
- {when to @mention another SOUL}
- {escalation criteria}
```

### 3.3 Identity Line Override

Current (generic): `"You are a personal assistant running inside GoClaw."` (line 75)

**MTClaw override strategy**: Instead of modifying the Go source, use `SOUL.md` content which is injected in Section 13 (Project Context) — the model reads SOUL.md early in context and adopts the persona. The generic identity line serves as fallback if no SOUL.md is present.

---

## 4. 16 SOULs → Database Seeding Plan

### 4.1 Agent Records

| # | agent_key | display_name | agent_type | frontmatter |
|---|-----------|-------------|------------|-------------|
| 1 | `pm` | Product Manager | predefined | Requirements, user stories, /spec factory, G0.1/G1 gates |
| 2 | `architect` | Software Architect | predefined | ADRs, system design, G2 gate, architecture review |
| 3 | `coder` | Software Engineer | predefined | Implementation, tests, code generation, bug fixes |
| 4 | `reviewer` | Code Reviewer | predefined | PR Gate, code review, quality scoring, security review |
| 5 | `researcher` | User Researcher | predefined | User research, data analysis, interview synthesis |
| 6 | `writer` | Technical Writer | predefined | Documentation, guides, runbooks, README |
| 7 | `pjm` | Project Manager | predefined | Sprint planning, task breakdown, velocity tracking |
| 8 | `devops` | DevOps Engineer | predefined | Infrastructure, deployment, CI/CD, monitoring |
| 9 | `tester` | QA Engineer | predefined | Test strategy, test cases, regression, automation |
| 10 | `cto` | CTO Advisor | predefined | Architecture guard, P0 blocking, technical decisions |
| 11 | `cpo` | CPO Advisor | predefined | Product guard, strategic decisions, user advocacy |
| 12 | `ceo` | CEO Advisor | predefined | Business direction, priority setting, resource allocation |
| 13 | `mts-dev` | MTS Developer Assistant | predefined | Bflow API docs, code review, MTS engineering daily tasks |
| 14 | `mts-sales` | MTS Sales Assistant | predefined | Proposal templates, MTS pricing, client communication |
| 15 | `mts-cs` | MTS Customer Service | predefined | SOP lookup, customer context, ticket resolution |
| 16 | `mts-general` | MTS General Assistant | predefined | HR policy, contracts, general office tasks |

All SOULs: `owner_id = 'mts'`, `provider = 'bflow-ai-platform'`, `model = 'qwen3:14b'`

### 4.2 Context Files Per SOUL

Each SOUL gets 3 agent_context_files:

| File | Content Source | Purpose |
|------|---------------|---------|
| `SOUL.md` | Generated from `docs/08-collaborate/souls/SOUL-{key}.md` | Persona, expertise, boundaries |
| `IDENTITY.md` | Generated per SOUL | Name, emoji, vibe |
| `AGENTS.md` | Shared MTClaw workspace instructions | Governance rules, safety, tools |

**AGENTS.md** (shared across all 16 SOULs):
```markdown
# AGENTS.md — MTClaw Workspace

## Governance Rules
- All responses follow 3 Rails governance framework
- Evidence trail required for governance actions
- Bflow AI-Platform is the ONLY AI provider (no bypass)

## SOUL Switching
- Use delegation (@mention) for cross-role requests
- pm → coder: implementation tasks
- reviewer → coder: code fixes after review
- mts-sales → mts-cs: customer handoff

## Memory
- Read MEMORY.md for long-term context
- Update memory when decisions are made

## Safety
- Never share tenant data cross-tenant
- Sensitive fields are AES-256-GCM encrypted
- All actions produce audit trail
```

### 4.3 Seeding Script (Sprint 3)

```
Migration: 000008_seed_mtclaw_souls.up.sql

1. INSERT INTO agents (16 rows) — one per SOUL
2. INSERT INTO agent_context_files (48 rows) — 3 files × 16 SOULs
3. INSERT INTO agent_links — delegation permissions between SOULs
4. INSERT INTO agent_teams — "MTS Engineering", "MTS Business" teams
5. INSERT INTO agent_team_members — assign SOULs to teams
```

---

## 5. SOUL Routing Design

### 5.1 Default Agent Resolution

```
New user → /start on Telegram
  │
  ▼
Router logic (internal/gateway/):
  1. Check handoff_routes: any temporary override for this chat?
  2. Check user_agent_profiles: returning user? → last used SOUL
  3. Default: mts-general (general-purpose entry point)
```

### 5.2 SOUL Switching (Delegation)

GoClaw supports delegation via `agent_links` + `spawn` tool:

```
User to mts-general: "Review PR #42"
  │
  ▼
mts-general detects: code review request → outside my expertise
  │
  ▼
Delegation: spawn(agent="reviewer", task="Review PR #42")
  │ (agent_links: mts-general → reviewer = active)
  │
  ▼
reviewer SOUL processes request with reviewer persona
  │
  ▼
Response returned to user (via mts-general session)
```

### 5.3 Handoff (Persistent Route Change)

```
User to mts-cs: "Tôi cần nói chuyện với sales team"
  │
  ▼
mts-cs creates handoff:
  INSERT INTO handoff_routes (channel='telegram', chat_id=X, to_agent_key='mts-sales')
  │
  ▼
Next message → routed to mts-sales SOUL
(handoff_routes checked before default resolution)
```

---

## 6. Governance Extensions (ExtraPrompt)

### 6.1 Rail-Specific Context Injection

For governance rails, inject context via `SystemPromptConfig.ExtraPrompt`:

```go
// Sprint 4: /spec command context
if rail == "spec-factory" {
    cfg.ExtraPrompt = `
## Spec Factory Rail
When user invokes /spec:
1. Gather requirements in structured format
2. Output JSON spec with: spec_id, title, description, acceptance_criteria
3. Attach evidence link
4. Return spec for review
`
}

// Sprint 5: PR Gate context
if rail == "pr-gate" {
    cfg.ExtraPrompt = `
## PR Gate Rail
When reviewing a PR:
1. Check: SQL injection, RLS compliance, test coverage
2. Score: 0-100 on correctness, security, completeness
3. Mode: WARNING (Sprint 5) — report issues, don't block
4. Output: structured review with evidence attachment
`
}
```

### 6.2 RAG Context (Knowledge Rail)

For Rail #3 (Knowledge & Answering), leverage GoClaw's built-in `memory_chunks` table:

```
Sprint 6: Populate memory_documents + memory_chunks for each SOUL:
  - mts-dev: Bflow API docs, engineering standards
  - mts-sales: Pricing tables, proposal templates, case studies
  - mts-cs: SOPs, customer FAQs, escalation procedures
  - mts-general: HR policies, company info, office procedures
```

GoClaw already has hybrid search (70% vector + 30% BM25) — no custom RAG needed.

---

## 7. Multi-Tenant Design (Sprint 3)

### 7.1 Tenant Isolation via owner_id

GoClaw's `agents.owner_id` serves as tenant identifier:

```sql
-- Phase 1: MTS tenant
INSERT INTO agents (agent_key, owner_id, ...) VALUES ('pm', 'mts', ...);
INSERT INTO agents (agent_key, owner_id, ...) VALUES ('mts-sales', 'mts', ...);

-- Phase 2: NQH tenant (if approved)
INSERT INTO agents (agent_key, owner_id, ...) VALUES ('nqh-general', 'nqh', ...);
```

### 7.2 RLS Policies (Sprint 3 Implementation)

```sql
-- Enable RLS on key tables
ALTER TABLE agents ENABLE ROW LEVEL SECURITY;
ALTER TABLE agent_context_files ENABLE ROW LEVEL SECURITY;
ALTER TABLE sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE memory_chunks ENABLE ROW LEVEL SECURITY;

-- Policy: agents visible only to owner's tenant
CREATE POLICY tenant_agents ON agents
  USING (owner_id = current_setting('app.tenant_id'));

-- Policy: context files scoped to agent → tenant
CREATE POLICY tenant_context_files ON agent_context_files
  USING (agent_id IN (SELECT id FROM agents WHERE owner_id = current_setting('app.tenant_id')));

-- Middleware: SET app.tenant_id = 'mts' per request
```

---

## 8. Implementation Timeline

| Sprint | Deliverable | Effort |
|--------|------------|--------|
| **Sprint 3** | RLS policies + tenant isolation | 3 days |
| **Sprint 3** | SOUL seeding migration (16 agents + 48 context files) | 2 days |
| **Sprint 3** | AGENTS.md governance template | 1 day |
| **Sprint 4** | /spec ExtraPrompt injection + SOUL routing | 3 days |
| **Sprint 4** | SOUL quality rubric integration (scoring hook) | 2 days |
| **Sprint 5** | PR Gate ExtraPrompt + reviewer SOUL enhancement | 2 days |
| **Sprint 6** | RAG population for MTS business SOULs | 3 days |
| **Sprint 6** | SOUL drift detection (checksum monitoring) | 1 day |

### Key Milestones

| Milestone | Target | Success Criteria |
|-----------|--------|-----------------|
| SOUL loading works | Sprint 3 Day 3 | 16 SOULs loaded from DB, system prompt includes SOUL.md |
| Tenant isolation verified | Sprint 3 Day 5 | RLS policies block cross-tenant access |
| SOUL routing works | Sprint 4 Day 3 | User message → correct SOUL → role-appropriate response |
| First governance rail | Sprint 4 Day 5 | /spec produces structured JSON with SOUL context |

---

## 9. Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| System prompt too long (>context window) | Low | High | SOUL.md budget: max 2,000 chars. Monitor via `promptLen` slog field |
| SOUL persona leaks across sessions | Medium | Medium | Verify session isolation: agent_id + user_id scoped |
| RAG returns wrong tenant's data | Medium | High | RLS mandatory Sprint 3. Test cross-tenant queries |
| SOUL quality degrades silently | Medium | Medium | Quality rubric scoring (Sprint 4+), drift detection |
| Delegation loops (A→B→A) | Low | Medium | GoClaw has `spawn_depth` limit. Set max_depth=3 |

---

## References

- GoClaw system prompt: `internal/agent/systemprompt.go`
- Bootstrap package: `internal/bootstrap/` (files.go, load_store.go, seed_store.go, truncate.go)
- Embedded templates: `internal/bootstrap/templates/` (SOUL.md, IDENTITY.md, AGENTS.md, etc.)
- Agent store: `internal/store/pg/agents_context.go`
- Database schema: `migrations/000001_init_schema.up.sql` through `migrations/000007_*.up.sql`
- SOUL quality rubric: `docs/01-planning/soul-quality-rubric.md`
- GoClaw schema analysis: `docs/02-design/goclaw-schema-analysis.md`
- ADR-004: `docs/02-design/01-ADRs/SPEC-0004-ADR-004-SOUL-Implementation.md`
