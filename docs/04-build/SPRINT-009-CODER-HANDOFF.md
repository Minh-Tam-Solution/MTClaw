# Sprint 9 — @coder Handoff

**Sprint**: 9 — Channel Rationalization + SOUL Suite Complete
**Date**: 2026-03-17
**From**: [@pm] + [@architect]
**To**: [@coder]
**CTO Approval**: ✅ ADR-006 APPROVED (2026-03-17) — Sprint 9 is UNBLOCKED
**CTO Score (Sprint 9 plan)**: 8.5/10 APPROVED

---

## Context

Sprint 8 closed at 8.5/10 (CTO APPROVED). All post-review fixes applied (CTO-26/27/28). 290 tests passing.

Sprint 9 is a **cleanup + hardening sprint**. No new user-facing features. Goal: remove 4 dead channels, complete SOUL behavioral test suite, scaffold MS Teams extension.

**CTO-32 resolution ([@pm] decision, 2026-03-17)**:
- Total SOUL files: 18
- Sprint 8 tested (5): pm, reviewer, coder, dev, sales
- Sprint 9 T9-03 — 12 governance SOULs (not 11):
  - **Include**: architect, cpo, cto, ceo, cs, devops, fullstack, itadmin, pjm, researcher, tester, writer
  - **Exclude**: assistant — category=router, sdlc_gates=[], no governance role. Documented exclusion, not a gap.
- Total behavioral tests after T9-03: 25 (Sprint 8) + 60 (12 × 5) = **85 tests** (not 80)

**ADR-006** (channel rationalization) is approved. Channels to remove: Feishu, Discord, Slack, WhatsApp. Channels to keep: Telegram + Zalo (`extensions/zalouser`).

---

## CTO Pre-Execution Notes (MUST READ BEFORE STARTING)

### CTO-29 (P1) — Verify Slack in gateway.go before Phase 3

Slack has zero implementation files — config-struct only. Before removing imports in Phase 3, verify:

```bash
grep -n "slack\|Slack" /home/nqh/shared/MTClaw/cmd/gateway.go
```

Expected outcomes:
- If result is empty → no Phase 3 action needed for Slack
- If there's a commented-out stub or conditional → remove it in Phase 3

**Do not skip this grep.** Incomplete cleanup = lint failure on next sprint when someone adds a Slack-related check.

### CTO-30 (P2) — Test count baseline before and after T9-02

The `gateway_consumer_test.go` Discord test case must be removed in T9-02 Phase 7. Record the baseline before any removals, then verify after:

```bash
# BEFORE starting T9-01 — record this number:
(cd /home/nqh/shared/MTClaw && go test ./... -count=1 -v 2>&1 | grep -c "^--- PASS")

# AFTER T9-02 Phase 7 — verify:
(cd /home/nqh/shared/MTClaw && go test ./... -count=1 -v 2>&1 | grep -c "^--- PASS")
```

**Acceptable**: result drops by exactly the number of Discord-specific test cases removed.
**Not acceptable**: result drops by more than expected — means an unintended test was broken.

Current baseline: **290 tests**.

### CTO-32 (P1) — SOUL count: 12 governance SOULs in T9-03 (not 11)

[@pm] resolved 2026-03-17: total SOUL files = 18.

| SOUL | Decision | Reason |
|------|----------|--------|
| `ceo` | ✅ **Include** | category=advisor, sdlc_gates=[G0.1, G4] — governance role |
| `assistant` | ❌ **Exclude** | category=router, sdlc_gates=[] — utility dispatcher, not a governance role |

**T9-03 scope**: 12 SOULs (Sprint 9 adds CEO to the original 11). Tests = 12 × 5 = 60 new tests. Total behavioral tests = 85.

Add this comment at top of `internal/souls/behavioral_test.go` before the new Sprint 9 tests:

```go
// Sprint 9 adds 12 governance SOULs (CTO-32 resolved 2026-03-17).
// CEO included: SE4H advisor, sdlc_gates=[G0.1, G4].
// assistant excluded: category=router, sdlc_gates=[] — not a governance role.
// Total behavioral tests: 25 (Sprint 8) + 60 (Sprint 9) = 85.
```

### CTO-31 (P2) — ADR-007 ownership is @pm (T9-05)

ADR-007 (MS Teams Extension architecture) is **@pm scope**, merged into T9-05 alongside the G4 gate proposal. @coder scope in T9-04 is strictly:

1. Create the directory scaffold: `extensions/msteams/`
2. Write `extensions/msteams/README.md` (setup steps: Azure AD app registration, Bot Framework)
3. Write `extensions/msteams/msteams.go.TODO` — a reference doc (NOT a `.go` file) with the channel interface stub

The `.go.TODO` extension is intentional: it is NOT a Go source file, so `go build ./...` must remain clean. @pm will draft ADR-007 content; @coder creates only the scaffold.

---

## Task Breakdown

### T9-01: Channel Removal — Core (P0, 3 pts) — Day 1

#### Phase 1 — Delete implementation directories

```bash
cd /home/nqh/shared/MTClaw

# Delete Feishu (12 files, ~2,060 LOC)
rm -rf internal/channels/feishu/

# Delete Discord (2 files, ~477 LOC)
rm -rf internal/channels/discord/

# Delete WhatsApp (2 files, ~299 LOC)
rm -rf internal/channels/whatsapp/

# Delete Feishu-specific onboarding file
rm cmd/onboard_feishu.go
```

Verify phase 1:
```bash
(cd /home/nqh/shared/MTClaw && go build ./...) 2>&1
# Expected: import errors for removed packages — Phase 2+3 will fix these
```

#### Phase 2 — Config cleanup

**`internal/config/config_channels.go`** — remove 4 struct definitions and 4 `ChannelsConfig` fields:

```go
// REMOVE entire structs:
type DiscordConfig struct { ... }
type SlackConfig struct { ... }
type WhatsAppConfig struct { ... }
type FeishuConfig struct { ... }

// REMOVE fields from ChannelsConfig:
Discord   DiscordConfig   `...`
Slack     SlackConfig     `...`
WhatsApp  WhatsAppConfig  `...`
Feishu    FeishuConfig    `...`
```

**`internal/config/config_load.go`** — remove env var loading blocks for all 4 channels (look for `DISCORD_`, `FEISHU_`, `SLACK_`, `WHATSAPP_` env vars).

**`internal/config/config_secrets.go`** — remove masking calls for:
- `Feishu.AppID`, `Feishu.AppSecret`, `Feishu.EncryptKey`, `Feishu.VerificationToken`
- `Discord.Token`
- `Slack.BotToken`, `Slack.AppToken`

Note: GitHub masking (CTO-27) and Zalo/Telegram masking must NOT be touched.

#### Phase 3 — Gateway wiring

**First, run CTO-29 check** (mandatory):
```bash
grep -n "slack\|Slack" /home/nqh/shared/MTClaw/cmd/gateway.go
```

**`cmd/gateway.go`** — remove:
```go
// REMOVE imports:
"github.com/nextlevelbuilder/goclaw/internal/channels/discord"
"github.com/nextlevelbuilder/goclaw/internal/channels/feishu"
"github.com/nextlevelbuilder/goclaw/internal/channels/whatsapp"
// Slack: only remove if CTO-29 grep shows a reference

// REMOVE factory registrations (managed mode):
instanceLoader.RegisterFactory("discord", discord.Factory)
instanceLoader.RegisterFactory("feishu", feishu.Factory)
instanceLoader.RegisterFactory("whatsapp", whatsapp.Factory)

// REMOVE config-based init blocks (standalone mode):
// if cfg.Channels.Discord.Enabled { ... }
// if cfg.Channels.WhatsApp.Enabled { ... }
// if cfg.Channels.Feishu.Enabled { ... }
```

**Verification after T9-01:**
```bash
(cd /home/nqh/shared/MTClaw && go build ./...) 2>&1
# Expected: clean build (0 errors)

(cd /home/nqh/shared/MTClaw && go test ./... -count=1) 2>&1 | tail -5
# Expected: all pass
```

---

### T9-02: Channel Removal — Periphery (P0, 2 pts) — Day 2

**Record test baseline before starting** (CTO-30):
```bash
(cd /home/nqh/shared/MTClaw && go test ./... -count=1 -v 2>&1 | grep -c "^--- PASS")
```

#### Phase 4 — Onboarding + CLI

| File | Action |
|------|--------|
| `cmd/onboard.go` | Remove Feishu wizard step (27 refs) |
| `cmd/onboard_auto.go` | Remove feishu, discord, slack, whatsapp branches |
| `cmd/onboard_managed.go` | Remove discord + whatsapp initialization |
| `cmd/onboard_helpers.go` | Remove channel-specific helpers for removed channels |
| `cmd/doctor.go` | Remove 4 channels from diagnostics |
| `cmd/channels_cmd.go` | Remove from CLI help text |

#### Phase 5 — Managed mode type registry

| File | Action |
|------|--------|
| `internal/store/channel_instance_store.go` | Remove feishu/discord/whatsapp from type list/switch |
| `internal/http/channel_instances.go` | Remove from allowed channel types validation |
| `internal/gateway/methods/channel_instances.go` | Remove from allowed channel types validation |

#### Phase 6 — Agent context + Tools + Bus

| File | Action |
|------|--------|
| `internal/agent/systemprompt.go` | Remove Discord from available channels list (2 refs) |
| `internal/agent/systemprompt_sections.go` | Remove Discord reference (1 ref) |
| `internal/tools/message.go` | Remove Discord from allowed channel list |
| `internal/tools/policy.go` | Remove WhatsApp from policy context |
| `internal/tools/subagent.go` | Remove WhatsApp handling |
| `internal/bus/types.go` | Remove Discord if hardcoded in message type enum |

#### Phase 7 — Tests cleanup

| File | Action |
|------|--------|
| `cmd/gateway_consumer_test.go` | Remove Discord test case(s) |
| `internal/tools/context_keys_test.go` | Remove Slack constants (3 refs) |

**Verify after T9-02** (CTO-30 check):
```bash
# Check test count drop is as expected
(cd /home/nqh/shared/MTClaw && go test ./... -count=1 -v 2>&1 | grep -c "^--- PASS")

# Check zero remaining references
grep -r "feishu\|FeishuConfig\|discord\|DiscordConfig\|whatsapp\|WhatsAppConfig\|SlackConfig" \
  /home/nqh/shared/MTClaw/internal/ \
  /home/nqh/shared/MTClaw/cmd/ \
  --include="*.go" | grep -v "_test.go" | wc -l
# Expected: 0

# Full build
(cd /home/nqh/shared/MTClaw && go build ./...) 2>&1
```

---

### T9-03: SOUL Behavioral Tests — 11 Remaining SOULs (P0, 2 pts) — Day 4

**File**: `internal/souls/behavioral_test.go` (extend existing file — do NOT create a new file)

**12 SOULs** (5 tests each = 60 new tests, CTO-32 adds CEO):

| SOUL file | Role key | Category | Focus area |
|-----------|----------|----------|------------|
| SOUL-architect.md | architect | SE4H advisor | Architecture decisions, ADR format |
| SOUL-ceo.md | ceo | SE4H advisor | Strategic decisions, G0.1/G4 approvals |
| SOUL-cpo.md | cpo | SE4H advisor | Product strategy, gate approval |
| SOUL-cto.md | cto | SE4H advisor | Technical standards, performance |
| SOUL-cs.md | cs | SE4A executor | Customer support, Vietnamese |
| SOUL-devops.md | devops | SE4A executor | CI/CD, infrastructure |
| SOUL-fullstack.md | fullstack | SE4A executor | Full-stack patterns |
| SOUL-itadmin.md | itadmin | SE4A executor | Infrastructure ops |
| SOUL-pjm.md | pjm | SE4A executor | Project management |
| SOUL-researcher.md | researcher | SE4A executor | Research methodology |
| SOUL-tester.md | tester | SE4A executor | QA patterns, test coverage |
| SOUL-writer.md | writer | SE4A executor | Documentation, Vietnamese |

**Test pattern per SOUL** (copy the established Sprint 8 pattern):

```go
// 5 checks per SOUL:
func TestSOUL_{Role}_HasYAMLFrontmatter(t *testing.T) { ... }
func TestSOUL_{Role}_IdentitySection(t *testing.T) { ... }
func TestSOUL_{Role}_CapabilitiesSection(t *testing.T) { ... }
func TestSOUL_{Role}_ConstraintsSection(t *testing.T) { ... }
func TestSOUL_{Role}_ChecksumDeterministic(t *testing.T) { ... }
```

Role-specific keyword checks to add per SOUL:

| SOUL | Identity keyword | Capabilities keyword |
|------|-----------------|---------------------|
| architect | "architect" OR "architecture" | "design" OR "system" OR "adr" |
| ceo | "ceo" OR "executive" OR "chief" | "strategic" OR "decision" OR "approve" |
| cpo | "product" OR "cpo" | "strategy" OR "roadmap" OR "gate" |
| cto | "technical" OR "cto" | "standard" OR "review" OR "architecture" |
| cs | "customer" OR "support" | "support" OR "client" OR "ticket" |
| devops | "devops" OR "infrastructure" | "deploy" OR "ci" OR "docker" |
| fullstack | "fullstack" OR "full-stack" OR "full stack" | "frontend" OR "backend" OR "typescript" |
| itadmin | "it" OR "admin" OR "infrastructure" | "server" OR "security" OR "infra" |
| pjm | "project" OR "pjm" | "timeline" OR "sprint" OR "milestone" |
| researcher | "research" | "research" OR "interview" OR "data" |
| tester | "tester" OR "qa" OR "quality" | "test" OR "coverage" OR "bug" |
| writer | "writer" OR "documentation" OR "writing" | "document" OR "write" OR "content" |

**Note**: Some SOULs (cs, devops, fullstack, itadmin, pjm, researcher, tester, writer) may NOT have `rag_collections` in frontmatter — don't add that check for them unless the SOUL file actually has it. Only dev and sales were confirmed to have it.

**Target**: Total behavioral tests after T9-03 = 25 (Sprint 8) + 60 (12 SOULs × 5) = **85 tests** in `behavioral_test.go`.

**Excluded**: `assistant` SOUL (category=router, sdlc_gates=[], no governance gates — deliberate exclusion per [@pm] CTO-32 decision). Document this in a comment at top of behavioral_test.go.

---

### T9-04: MS Teams Extension Scaffold (P1, 1 pt) — Day 5

**@coder scope only** (ADR-007 content is @pm):

```bash
mkdir -p /home/nqh/shared/MTClaw/extensions/msteams
```

Create 3 files:

**`extensions/msteams/README.md`**:
```markdown
# MS Teams Extension — MTClaw

Status: SCAFFOLD — Implementation in Sprint 10
ADR: docs/02-design/01-ADRs/SPEC-0007-ADR-007-MSTeams-Extension.md

## Prerequisites (Sprint 10 setup)

1. Azure AD app registration
   - App type: Multi-tenant
   - API permissions: Chat.ReadWrite, ChannelMessage.Send

2. Bot Framework registration
   - Bot handle: mtclaw-bot
   - Messaging endpoint: https://<host>/v1/channels/msteams/webhook

3. Environment variables
   - MSTEAMS_APP_ID=<Azure app ID>
   - MSTEAMS_APP_PASSWORD=<Azure app secret>
   - MSTEAMS_TENANT_ID=<target tenant ID>

## Architecture

Factory pattern — same as other channels:
  RegisterFactory("msteams", msteams.Factory)

Zero core code changes required.
```

**`extensions/msteams/msteams.go.TODO`**:
```
// THIS FILE IS NOT COMPILED — reference only for Sprint 10 implementation.
// Rename to msteams.go when implementing.
//
// package msteams
//
// import "github.com/nextlevelbuilder/goclaw/internal/channels"
//
// type MSTeamsChannel struct { ... }
//
// func (c *MSTeamsChannel) Send(ctx context.Context, msg channels.OutboundMessage) error { ... }
// func (c *MSTeamsChannel) RegisterRoutes(mux *http.ServeMux) { ... }
//
// var Factory = func(cfg map[string]any, bus *bus.MessageBus) (channels.Channel, error) {
//     // parse cfg, init Bot Framework client, return MSTeamsChannel
// }
```

**Verify**: `go build ./...` must still be clean (`.go.TODO` is not a Go source file).

---

### T9-05: G4 Gate Proposal + ADR-007 Draft (P0, 1 pt) — Day 5 [@pm scope]

**[@coder]**: No action. This is @pm deliverable.

**[@pm]** will:
1. File `docs/08-collaborate/G4-GATE-PROPOSAL-SPRINT8.md`
2. Draft `docs/02-design/01-ADRs/SPEC-0007-ADR-007-MSTeams-Extension.md` (architecture for Sprint 10)

---

## Definition of Done

| Check | Command | Expected |
|-------|---------|----------|
| Build clean | `go build ./...` | 0 errors |
| All tests pass | `go test ./... -count=1` | ≥290 PASS |
| No dead channel refs | `grep -r "feishu\|discord\|whatsapp\|FeishuConfig\|DiscordConfig\|WhatsAppConfig\|SlackConfig" internal/ cmd/ --include="*.go" \| wc -l` | 0 |
| SOUL tests complete | `go test ./internal/souls/ -v -count=1 \| grep -c "^--- PASS"` | ≥85 |
| MS Teams scaffold | `ls extensions/msteams/` | README.md + msteams.go.TODO |
| CTO-29 verified | grep slack in gateway.go checked | documented in PR |
| CTO-30 verified | test count before/after T9-02 recorded | delta = Discord test cases only |

---

## File Reference

| File | Path |
|------|------|
| ADR-006 (APPROVED) | `docs/02-design/01-ADRs/SPEC-0006-ADR-006-Channel-Rationalization.md` |
| Sprint 9 plan | `docs/04-build/sprints/SPRINT-009-Channel-Cleanup-SOUL-Complete.md` |
| Sprint 8 behavioral tests (pattern reference) | `internal/souls/behavioral_test.go` |
| Sprint 8 drift E2E (pattern reference) | `internal/integration/drift_e2e_test.go` |
