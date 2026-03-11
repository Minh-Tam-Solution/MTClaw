---
sprint: 32
title: "Memory Fact Store (Phase 1) + Discord Reactions"
status: PLANNING
start_date: 2026-03-17
end_date: 2026-03-28
lead: "@pm (plan) → @coder (implementation)"
framework: SDLC Enterprise Framework 6.1.2
adr: ADR-015-Memory-Enhancement-ClawVault-Port
depends_on: Sprint 31 Phase 0 Go/No-Go gate
---

# Sprint 32 — Memory Fact Store (Phase 1) + Discord Reactions

## Sprint Goal

**Conditional sprint** based on Sprint 31 Phase 0 measurement result:

| Phase 0 Result | Sprint 32 Scope |
|---------------|-----------------|
| <40% structured | **Track A**: Fact Store (Phase 1) — full sprint |
| 40-60% structured | **Track A-lite**: Iterate prompt once → re-measure → if still <60%, start Fact Store |
| ≥60% structured | **Track B**: Discord Reactions + ReactionChannel refactor — full sprint |

**Regardless of track**: Discord ReactionChannel `messageID` type fix ships in both tracks (1-day prerequisite, unblocks future Discord work).

---

## Gate Entry: Phase 0 Measurement (Day 1)

Before any implementation, run the Phase 0 measurement query:

```sql
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

Filesystem backup:
```bash
grep -rlc '## \[(decision\|fact\|preference\|lesson\|entity)\]' memory/
```

**Decision**: CTO reviews result → selects Track A or Track B → sprint begins.

---

## Track A: Fact Store — Phase 1 (8-10 days)

**Prerequisite**: Phase 0 gate = <40% structured (or 40-60% after one prompt iteration).

### A1: Migration 000021 — `memory_facts` table (Day 1-2)

| File | Change |
|------|--------|
| `migrations/000021_create_memory_facts.up.sql` | Create `memory_facts` table with RLS (per ADR-015 §3.1) |
| `migrations/000021_create_memory_facts.down.sql` | `DROP TABLE memory_facts` |
| `internal/upgrade/version.go` | Bump `RequiredSchemaVersion` 20 → 21 |

**Schema** (from ADR-015, CTO-approved):
```sql
CREATE TABLE memory_facts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    owner_id VARCHAR(255) NOT NULL,
    agent_id UUID NOT NULL REFERENCES agents(id),
    user_id TEXT,
    entity TEXT NOT NULL,
    entity_norm TEXT NOT NULL,
    relation TEXT NOT NULL,
    value TEXT NOT NULL,
    category TEXT NOT NULL,
    confidence REAL DEFAULT 0.5,
    source_document_id UUID,
    source_path TEXT,
    raw_text TEXT,
    last_accessed_at TIMESTAMPTZ,
    valid_from TIMESTAMPTZ DEFAULT now(),
    valid_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now()
);
CREATE INDEX idx_facts_entity ON memory_facts(agent_id, entity_norm) WHERE valid_until IS NULL;
CREATE INDEX idx_facts_agent_user ON memory_facts(agent_id, user_id) WHERE valid_until IS NULL;

ALTER TABLE memory_facts ENABLE ROW LEVEL SECURITY;
CREATE POLICY memory_facts_tenant_isolation ON memory_facts
    USING (owner_id = current_setting('app.tenant_id', true));
```

**Acceptance**: `make migrate-up` succeeds. `\d memory_facts` shows correct schema + RLS enabled.

### A2: Fact Store package (Day 2-4)

| File | Purpose |
|------|---------|
| `internal/facts/store.go` | `FactStore` struct: `Upsert()`, `Query()`, `Supersede()`. Query returns active facts only (`WHERE valid_until IS NULL`). Conflict resolution: newer fact supersedes older by `(agent_id, entity_norm, relation)`. |
| `internal/facts/store_test.go` | Integration tests with real PostgreSQL. Test: upsert, supersede, query active-only, concurrent writes. |
| `internal/facts/models.go` | `Fact` struct matching DB schema. |

**Key design**:
- `Query(ctx, agentID, userID, entityNorm)` → `[]Fact` (active only)
- `QueryAll(ctx, agentID, userID)` → `[]Fact` (for context injection, limit 10)
- `Upsert(ctx, fact)` → sets `valid_until = now()` on old fact with same entity+relation, inserts new

**Acceptance**: `go test ./internal/facts/ -v` passes. Supersede logic verified.

### A3: LLM-based Fact Extractor (Day 4-6)

| File | Purpose |
|------|---------|
| `internal/facts/extractor.go` | `FactExtractor` interface + `LLMExtractor` implementation. Calls Bflow AI-Platform with structured prompt. Returns `[]Fact`. |
| `internal/facts/extractor_test.go` | Test with mock LLM response (test the parsing, not the LLM). |
| `internal/facts/prompt.go` | Extraction prompt template (request JSON array of entity/relation/value/category/confidence). |

**Extraction prompt** (sent to LLM):
```
Extract structured facts from the following text. Return a JSON array of objects:
[{"entity": "...", "relation": "...", "value": "...", "category": "fact|decision|preference|lesson|entity", "confidence": 0.0-1.0}]

Rules:
- entity: lowercase normalized subject (person name, project, tool, concept)
- relation: what is being stated (uses, decided, prefers, is, owns, located_at, version)
- value: the specific detail
- category: one of fact/decision/preference/lesson/entity
- confidence: 0.0-1.0 (how certain is this fact?)
- Only extract facts with confidence >= 0.5
- Return [] if no facts found

Text:
{content}
```

**Acceptance**: Extractor parses LLM JSON response correctly. Handles malformed JSON gracefully (returns empty, logs warning).

### A4: Integration with Memory Write Path (Day 6-8)

| File | Change |
|------|--------|
| `internal/tools/memory_interceptor.go` | After successful `PutDocument()` + `IndexDocument()` (line ~123), submit async fact extraction to bounded worker pool. Hook AFTER indexing to ensure consistent state (facts exist only when both document and chunks are stored). Non-fatal: extraction failure logs + skips. |
| `internal/config/config.go` | Add `FactExtraction` sub-config to `MemoryConfig`: `Enabled bool`, `Model string`, `MaxWorkers int` (default: 3). |
| `internal/facts/worker_pool.go` | Bounded worker pool (`MaxWorkers` goroutines + buffered channel). Context propagation via `store.WithTenantID()`. |

**Worker pool design**:
```go
type ExtractionPool struct {
    workers  int
    queue    chan extractionJob
    store    *FactStore
    extract  FactExtractor
}
// Submit is non-blocking. If queue full, drop silently (log warning).
func (p *ExtractionPool) Submit(ctx context.Context, doc MemoryDocument) { ... }
```

**Acceptance**: Memory write triggers async extraction. Queue-full scenario handled gracefully. Tenant ID propagated correctly.

### A5: Fact Search Tool (Day 8-9)

| File | Change |
|------|--------|
| `internal/tools/fact_search.go` | New tool `fact_search`: query facts by entity or list all facts for current agent+user. Returns formatted fact list. |
| `internal/tools/registry.go` | Register `fact_search` tool. |

**Tool definition**:
```json
{
  "name": "fact_search",
  "description": "Search stored facts about entities. Returns structured entity-relation-value triples.",
  "parameters": {
    "entity": {"type": "string", "description": "Entity to search for (optional, omit to list recent facts)"},
    "limit": {"type": "integer", "description": "Max results (default 10)"}
  }
}
```

**Acceptance**: Agent can call `fact_search` and get precise results. Existing `memory_search` unchanged.

### A6: Tests + Build Verification (Day 9-10)

| Check | Target |
|-------|--------|
| `make test` | All existing + new tests pass |
| `make build` | Compiles cleanly |
| Integration test | Write memory → facts extracted → `fact_search` returns them |
| Latency benchmark | Fact query <5ms p95 (indexed) |
| RLS verification | Tenant A cannot see Tenant B's facts |

---

## Track B: Discord Reactions + Enhancements (8-10 days)

**Prerequisite**: Phase 0 gate = ≥60% structured (prompt is sufficient).

### B0: ReactionChannel Interface Fix (Day 1, both tracks)

| File | Change |
|------|--------|
| `internal/bus/reactions.go` | Change `ReactionChannel` interface: `messageID int` → `messageID string` on both `OnReactionEvent` and `ClearReaction` methods. |
| `internal/bus/manager.go` | Update `RunContext.MessageID` field type and all callers of `reactionCh.OnReactionEvent()`. |
| `internal/channels/telegram/telegram.go` | Update reaction calls to use `strconv.Itoa(msgID)` at boundary. |
| `internal/channels/telegram/reactions.go` | Update `StatusReactionController` struct: `messageID int` → convert at Telegram API boundary only. |
| `internal/channels/discord/discord.go` | Pass Discord snowflake ID directly (already string). |

**Blast radius**: All files referencing `messageID int` in the reaction path must be updated. Compile will catch any misses.

**Acceptance**: `make test` passes. No runtime type assertions needed.

### B1: Discord Reactions (Day 2-4)

| File | Change |
|------|--------|
| `internal/channels/discord/reactions.go` | Implement `AddReaction(channelID, messageID, emoji)` using Discord API. Map MTClaw reaction types (thinking, done, error) to Discord emoji. |
| `internal/channels/discord/discord.go` | Wire reactions into message flow: thinking emoji on receive, checkmark on complete, cross on error. |
| `internal/channels/discord/reactions_test.go` | Test emoji mapping, reaction API call formatting. |

**Emoji mapping**:
| MTClaw Event | Discord Emoji |
|-------------|--------------|
| Message received (thinking) | :thinking: (🤔) |
| Response complete | :white_check_mark: (✅) |
| Error | :x: (❌) |
| Tool execution | :gear: (⚙️) |

### B2: Discord Streaming via Message Edit (Day 4-6)

| File | Change |
|------|--------|
| `internal/channels/discord/streaming.go` | Implement streaming by editing the initial "thinking..." message with progressive content via `session.ChannelMessageEdit(channelID, messageID, newContent)`. Throttle edits to 1/second (Discord rate limit). |
| `internal/channels/discord/discord.go` | Integrate streaming into Send flow when response is streaming-capable. |

**Design**: Same pattern as Telegram streaming (message edit). Discord rate limit: 5 edits/5 seconds per channel. Throttle to 1 edit/second to stay safe. Reactions via `session.MessageReactionAdd(channelID, messageID, emoji)` — Unicode emoji passed directly (e.g., "🤔").

### B3: Discord Slash Commands (Day 6-8)

| File | Change |
|------|--------|
| `internal/channels/discord/commands.go` | Register `/help`, `/spec`, `/soul` slash commands via Discord API. |
| `internal/channels/discord/commands_test.go` | Test command registration, response formatting. |

**Design notes**:
- Use **guild-scoped** command registration (instant propagation, matches `guildIDs` allowlist pattern). Global commands have 1hr delay.
- Register commands on `Start()`, deregister on `Stop()`.
- Acknowledge interactions within 3s via `InteractionResponseDeferredChannelMessageWithSource`, then follow up with full response.

**Commands**:
| Command | Description |
|---------|------------|
| `/help` | Show available commands and SOUL list |
| `/spec` | Show current agent's SOUL spec |
| `/soul [name]` | Switch to specific SOUL (if multi-agent routing) |

### B4: Tests + Build Verification (Day 8-10)

| Check | Target |
|-------|--------|
| `make test` | All existing + new tests pass |
| `make build` | Compiles cleanly |
| E2E test | Send Discord message → reaction appears → response streams → checkmark |
| Rate limit test | 10 rapid messages → no Discord 429 errors |

---

## Shared Work (Both Tracks)

### S1: ReactionChannel Refactor (Day 1)

See B0 above — ships regardless of track. Unblocks Discord reactions for current or future sprint.

### S2: Sprint 31 Part C — Phase 0 Measurement (Day 1)

Run measurement query, record result, CTO Go/No-Go decision.

### S3: Cross-Sprint Debt (if time permits)

| Item | Origin | Priority | Estimate |
|------|--------|----------|----------|
| `make souls-validate` pre-commit hook | S30-A OBS-030-5 | P2 | 2h |
| Doctor health/bridge display sections | S28 OBS-028-11 | P3 | 4h |
| CountTraces pre-flight optimization | S28 OBS-027-5 | P3 | 2h |

---

## Risk Register

| Risk | Impact | Prob | Mitigation |
|------|--------|------|------------|
| Phase 0 measurement data insufficient (<10 sessions by Day 1) | Sprint start delayed | 20% | Measurement deadline 2026-03-25 max. Start with shared work (B0, S3) while waiting. |
| Track A: LLM extraction produces poor JSON | Fact store has low-quality data | 25% | Confidence threshold 0.5. Malformed JSON → empty result (non-fatal). Manual review of first 20 extractions. |
| Track A: Worker pool backpressure under load | Memory writes slow down | 10% | Queue-full → drop extraction (log warning). Memory write itself is never blocked. |
| Track B: Discord rate limits hit during streaming | Messages throttled/delayed | 15% | 1 edit/second throttle. Backoff on 429 response. |
| Gray zone (40-60%) delays decision | Sprint scope unclear for 1-2 days | 30% | Start with shared work. Iterate prompt. Re-measure after 48h. |

---

## Timeline

```
Track A / Track B:
  Day 1:    Phase 0 measurement + CTO Go/No-Go + B0 (ReactionChannel fix)
  Day 2-4:  Track A: A1+A2 (migration + store) | Track B: B1 (reactions)
  Day 4-6:  Track A: A3 (extractor)            | Track B: B2 (streaming)
  Day 6-8:  Track A: A4 (integration)          | Track B: B3 (slash commands)
  Day 8-9:  Track A: A5 (fact_search tool)     | Track B: B4 (tests)
  Day 9-10: Track A: A6 (tests + verification) | Track B: debt (S3)

Gray zone (40-60%):
  Day 1-2:  Shared work (B0 + S3 debt)
  Day 3:    Iterate memory flush prompt + deploy
  Day 5:    Re-measure structured rate
  Day 5+:   Track A or Track B (effective sprint = 8 days)
```

---

## Success Criteria

### Track A (Fact Store)
- [ ] Migration 000021 applied, RLS verified
- [ ] `FactStore.Upsert/Query/Supersede` working with real PostgreSQL
- [ ] LLM extractor produces valid facts from memory documents
- [ ] Async extraction via bounded worker pool (non-blocking writes)
- [ ] `fact_search` tool registered and callable by agents
- [ ] Fact query latency <5ms p95

### Track B (Discord Reactions)
- [ ] ReactionChannel interface uses string messageID
- [ ] Discord reactions (thinking/done/error) on messages
- [ ] Discord streaming via message edit (1 edit/sec throttle)
- [ ] Discord slash commands (`/help`, `/spec`, `/soul`)
- [ ] No Discord 429 rate limit errors under normal load

### Both Tracks
- [ ] `make build` clean
- [ ] `make test` all pass
- [ ] No regressions in existing Telegram/Discord functionality

---

## Sprint 33 Outlook

| If Sprint 32 = Track A | If Sprint 32 = Track B |
|------------------------|------------------------|
| Sprint 33: Phase 1 validation (20 queries, CTO spot-check) + Phase 2 prep | Sprint 33: Memory Fact Store (Phase 1) — deferred from Track B |
| Sprint 34: Phase 2 (Observation Pipeline) if gate passes | Sprint 34: Phase 2 (Observation Pipeline) if Phase 1 gate passes |
