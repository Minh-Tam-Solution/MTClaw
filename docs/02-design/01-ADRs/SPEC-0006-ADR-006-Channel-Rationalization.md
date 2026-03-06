---
id: ADR-006
title: Channel Rationalization — Telegram + Zalo Only (Sprint 9)
status: APPROVED
author: "@pm + @architect"
approved_by: "@cto"
approval_date: 2026-03-17
date: 2026-03-17
sprint: 9
awaiting_approval: "@cto"
supersedes: null
related_adrs: [ADR-004]
tags: [channels, architecture, cleanup, feishu, discord, slack, whatsapp, msteams]
framework: SDLC Enterprise Framework 6.1.1
---

# ADR-006 — Channel Rationalization: Telegram + Zalo Only

> **Status**: APPROVED — [@cto] 2026-03-17
> **Author**: [@pm] + [@architect]
> **Date**: 2026-03-17

---

## 1. Context

MTClaw was built with support for 6 messaging channels:

| Channel | Status | Users |
|---------|--------|-------|
| Telegram | ✅ Active | MTS Engineering team |
| Zalo | ✅ Active | NQH Phase 2 (planned) |
| Feishu/Lark | ❌ Not deployed | 0 users |
| Discord | ❌ Not deployed | 0 users |
| Slack | ❌ Stub only (no implementation) | 0 users |
| WhatsApp | ❌ Not deployed | 0 users |

4 channels (Feishu, Discord, Slack, WhatsApp) have **zero active users** and are not in the MTS or NQH deployment plan. MS Teams is planned for future sprints but not yet scheduled.

**Current footprint of unused channels:**

| Channel | LOC | Files | References |
|---------|-----|-------|-----------|
| Feishu | ~2,060 | 12 + 1 onboarding | 170 |
| Discord | ~477 | 2 | 95 |
| WhatsApp | ~299 | 2 | 72 |
| Slack | 0 (stub) | 0 | 17 |
| **Total** | **~2,836** | **17** | **354** |

This dead code:
- Inflates binary size and build time
- Creates maintenance surface (dependency updates, security patches) with no user benefit
- Makes onboarding flow confusing (wizard shows Feishu/Discord options MTS users never use)
- Increases cognitive load for new contributors

**MS Teams decision**: CEO/CPO have confirmed MS Teams will be integrated in a future sprint (Sprint 10+). It is explicitly **NOT** in scope for Sprint 9.

---

## 2. Problem Statement

> Maintaining 4 unused channel implementations (Feishu, Discord, Slack, WhatsApp) creates:
> 1. Dead code accumulation — ~2,836 LOC with zero users
> 2. Ongoing maintenance burden — security patches, dependency upgrades for unused code
> 3. Cognitive overhead — new developers must understand channels that will never run
> 4. Onboarding UX confusion — wizard presents irrelevant options
> 5. Binary bloat — Feishu Lark SDK and Discord libraries compiled into every build

---

## 3. Decision

**Remove Feishu, Discord, Slack, and WhatsApp from the codebase entirely.**

Keep only:
- ✅ **Telegram** — primary channel, MTS Engineering team
- ✅ **Zalo** (extensions/zalouser) — Phase 2 NQH deployment

**MS Teams** — deferred to Sprint 10+. When ready: implement as a proper extension in `extensions/msteams` following the existing pattern, submit new ADR.

---

## 4. Alternatives Considered

### Option A: Feature flags / compile tags
**Rejected**: Adds complexity (build matrix, conditional compilation) without reducing maintenance surface. Security patches still needed.

### Option B: Keep stubs, remove implementations
**Rejected**: Stubs still accumulate technical debt. Config structs remain. Onboarding flow still confusing.

### Option C (Chosen): Full removal
Remove all code, config structs, tests, onboarding references. Clean codebase = clear intention.

### Option D: Keep Discord (potential engineering team use)
**Rejected**: MTS Engineering team is already on Telegram. No adoption plan for Discord.

---

## 5. Consequences

### Positive
- **~2,836 LOC removed** — smaller, faster builds
- **Simpler onboarding** — wizard shows only Telegram + Zalo
- **Zero maintenance cost** for removed channels
- **Clearer contributor onboarding** — new devs don't need to understand dead channels
- **Security posture** — fewer dependencies = fewer CVEs to track

### Negative / Risks
- If a user requests Discord/Feishu/WhatsApp in future: must re-implement from scratch (or from git history)
- **Mitigation**: Git history preserves the code. Re-integration is straightforward with existing channel interface.
- MS Teams is explicitly deferred — users who need Teams integration must wait for Sprint 10+

### Neutral
- Zalo extension (`extensions/zalouser`) remains untouched — it is a workspace package, not the removed channels
- Channel interface (`internal/channels/channel.go`) remains unchanged — only implementations removed

---

## 6. Implementation Plan (Sprint 9, @coder)

### Phase 1 — Delete implementation directories (day 1)
```
DELETE: internal/channels/feishu/      (12 files, ~2,060 LOC)
DELETE: internal/channels/discord/     (2 files, ~477 LOC)
DELETE: internal/channels/whatsapp/    (2 files, ~299 LOC)
DELETE: cmd/onboard_feishu.go          (1 file)
```

### Phase 2 — Config cleanup (day 1)
```
EDIT: internal/config/config_channels.go
  - Remove DiscordConfig struct
  - Remove SlackConfig struct
  - Remove WhatsAppConfig struct
  - Remove FeishuConfig struct
  - Remove Discord, Slack, WhatsApp, Feishu fields from ChannelsConfig

EDIT: internal/config/config.go
  - Remove ChannelsConfig fields that reference deleted structs

EDIT: internal/config/config_load.go
  - Remove env var loading: DISCORD_*, FEISHU_*, SLACK_*, WHATSAPP_*

EDIT: internal/config/config_secrets.go
  - Remove Feishu masking (4 fields)
  - Remove Discord masking (1 field)
  - Remove Slack masking (2 fields)
```

### Phase 3 — Gateway wiring (day 1)
```
EDIT: cmd/gateway.go
  - Remove 3 imports (discord, feishu, whatsapp packages)
  - Remove 3 RegisterFactory calls
  - Remove 3 config-based channel init blocks (~36 LOC)
```

### Phase 4 — Onboarding + CLI (day 2)
```
EDIT: cmd/onboard.go          — remove Feishu wizard step
EDIT: cmd/onboard_auto.go     — remove 4 channel branches
EDIT: cmd/onboard_managed.go  — remove discord + whatsapp branches
EDIT: cmd/onboard_helpers.go  — remove channel-specific helpers
EDIT: cmd/doctor.go           — remove 4 channels from diagnostics
EDIT: cmd/channels_cmd.go     — remove from CLI help
```

### Phase 5 — Managed mode cleanup (day 2)
```
EDIT: internal/store/channel_instance_store.go  — remove type cases
EDIT: internal/http/channel_instances.go        — update allowed types
EDIT: internal/gateway/methods/channel_instances.go — update allowed types
```

### Phase 6 — Agent + Tools + Bus (day 2)
```
EDIT: internal/agent/systemprompt.go           — remove Discord from channel list
EDIT: internal/agent/systemprompt_sections.go  — remove Discord reference
EDIT: internal/tools/message.go                — remove Discord
EDIT: internal/tools/policy.go                 — remove WhatsApp
EDIT: internal/tools/subagent.go               — remove WhatsApp
EDIT: internal/bus/types.go                    — remove Discord if hardcoded
```

### Phase 7 — Tests (day 2)
```
EDIT: cmd/gateway_consumer_test.go             — remove Discord test case
EDIT: internal/tools/context_keys_test.go      — remove Slack constants
```

### Verification
```bash
go build ./...     # must compile clean
go test ./...      # must maintain ≥290 tests passing
grep -r "feishu\|discord\|whatsapp" internal/ cmd/ --include="*.go"  # must return 0 results
```

---

## 7. MS Teams — Future Sprint (NOT in ADR-006 scope)

When MS Teams is prioritized (Sprint 10+), the approach will be:
1. Implement as `extensions/msteams` workspace package (existing extension pattern)
2. Submit new ADR (ADR-007) with Teams-specific decisions
3. Wire via `RegisterFactory` in gateway — no core code changes needed

---

## 8. CTO Approval

**Status: APPROVED — [@cto] 2026-03-17**

| Decision | CTO Decision |
|----------|-------------|
| Remove all 4 channels (Feishu, Discord, Slack, WhatsApp) | ✅ REMOVE ALL 4 |
| Include Discord removal | ✅ REMOVE — MTS Engineering on Telegram, no adoption plan |
| MS Teams deferred to Sprint 10+ | ✅ CONFIRMED — Sprint 9 scaffold only |
| Sprint 9 scope: channel cleanup + 11 remaining SOULs | ✅ APPROVED |

---

*[@pm] business justification: zero active users, non-trivial maintenance cost, clean slate for Sprint 10 MS Teams integration*
*[@architect] technical justification: clean interface boundary, git history preserves code, no shared state between removed and remaining channels*
