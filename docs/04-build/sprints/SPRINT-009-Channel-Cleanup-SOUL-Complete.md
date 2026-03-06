# Sprint 9 — Channel Rationalization + SOUL Suite Complete

**SDLC Stage**: 04-Build
**Version**: 1.0.0
**Date**: 2026-03-17
**Author**: [@pm] + [@architect]
**Sprint**: 9 of 10+
**Phase**: 2 (Governance) → 3 (Hardening)
**Framework**: SDLC 6.1.1 — STANDARD tier
**Status**: PROPOSED — Awaiting @cto approval (ADR-006)

---

## 1. Sprint Context

### Predecessor: Sprint 8 ✅ (CTO 8.5/10 APPROVED)

Sprint 8 delivered PR Gate ENFORCE + Context Drift validation:

| Deliverable | Status | CTO Verdict |
|-------------|--------|-------------|
| PR Gate ENFORCE (webhook + commit status) | ✅ | EXCELLENT |
| pr_gate_evaluations table + RLS | ✅ | GOOD |
| Evidence export API (JSON + CSV) | ✅ | GOOD |
| Context Drift E2E tests (5 tests, 16 subtests) | ✅ | EXCELLENT |
| SOUL behavioral tests (5 SOULs × 5 = 25 tests) | ✅ | EXCELLENT |
| CTO-22 RAG evidence → JSONB metadata | ✅ | FIXED |

**Sprint 8 post-review fixes applied (all PASS, 290 tests)**:
- CTO-26 (P1): `slog.Warn` on scan error in `pg/pr_gate.go`
- CTO-27 (P1): GitHub credentials masked in all 3 secret functions
- CTO-28 (P2): Pagination warning in `evidence_export.go`

### Entry Criteria

| Criterion | Status | Evidence |
|-----------|--------|----------|
| 3 governance rails operational | ✅ | Rail #1 (Spec) + Rail #2 (PR Gate) + Rail #3 (RAG) |
| 290 tests passing | ✅ | `go test ./...` all green |
| CTO-26/27/28 fixed | ✅ | Build clean, 290 tests |
| ADR-006 approved | ⏳ | Awaiting @cto sign-off |
| G4 gate proposal filed | ⏳ | Sprint 8 deliverable [@pm] |

---

## 2. Sprint Goal

**Clean codebase down to Telegram + Zalo. Complete the SOUL suite for all 16 roles. Prepare MS Teams extension scaffold.**

### Key outcomes

1. Feishu, Discord, Slack, WhatsApp removed from codebase entirely (~2,836 LOC deleted, ~354 references cleaned)
2. SOUL behavioral tests for 12 governance SOULs (deferred from Sprint 8 CPO Condition 3; CTO-32: CEO added, assistant excluded — see handoff)
3. Onboarding wizard simplified to Telegram + Zalo only
4. MS Teams extension scaffold created (ADR-007 drafted, no working implementation)
5. Sprint 9 closes with ≥290 tests passing (no regression)

---

## 3. Task Overview

| ID | Task | Priority | Points | Owner |
|----|------|----------|--------|-------|
| T9-01 | Channel removal — core (Phase 1-3: delete dirs + config + gateway wiring) | P0 | 3 | @coder |
| T9-02 | Channel removal — periphery (Phase 4-7: onboard + agent + tools + tests) | P0 | 2 | @coder |
| T9-03 | SOUL behavioral tests — 12 governance SOULs (5 tests each = 60 tests, CTO-32: +CEO) | P0 | 2 | @coder |
| T9-04 | MS Teams extension scaffold (@coder) + ADR-007 draft (@pm, merged into T9-05) | P1 | 1 | @coder + @pm |
| T9-05 | G4 gate proposal ([@pm] deliverable, Sprint 8 carryover) | P0 | 1 | @pm |

**Total**: 9 points, 4 days @coder + 1 day @pm

---

## 4. Task Specifications

---

### T9-01: Channel Removal — Core (P0, 3 pts)

**Objective**: Delete 4 unused channel implementations and clean config/gateway wiring.

#### Phase 1 — Delete implementation directories

```bash
# Full deletions
rm -rf internal/channels/feishu/    # 12 files, ~2,060 LOC
rm -rf internal/channels/discord/   # 2 files, ~477 LOC
rm -rf internal/channels/whatsapp/  # 2 files, ~299 LOC
rm cmd/onboard_feishu.go             # 1 file, Feishu-specific wizard
```

#### Phase 2 — Config cleanup

**`internal/config/config_channels.go`**: Remove 4 config structs + 4 fields from `ChannelsConfig`:
- Remove `DiscordConfig` struct (lines ~66-74)
- Remove `SlackConfig` struct (lines ~76-84)
- Remove `WhatsAppConfig` struct (lines ~86-92)
- Remove `FeishuConfig` struct (lines ~104-125)
- Remove `Discord`, `Slack`, `WhatsApp`, `Feishu` fields from `ChannelsConfig`

**`internal/config/config_load.go`**: Remove env var loading:
- `DISCORD_TOKEN`, `FEISHU_APP_ID`, `FEISHU_APP_SECRET`, `FEISHU_ENCRYPT_KEY`, `FEISHU_VERIFICATION_TOKEN`
- `SLACK_BOT_TOKEN`, `SLACK_APP_TOKEN`
- `WHATSAPP_BRIDGE_URL` (and any related vars)

**`internal/config/config_secrets.go`**: Remove masking for deleted fields:
- `Feishu.AppID`, `Feishu.AppSecret`, `Feishu.EncryptKey`, `Feishu.VerificationToken`
- `Discord.Token`
- `Slack.BotToken`, `Slack.AppToken`
(Zalo and Telegram masking must remain)

#### Phase 3 — Gateway wiring

**`cmd/gateway.go`**: Remove 3 channel imports + 3 factory registrations + 3 config-based init blocks:

```go
// REMOVE these imports:
"github.com/nextlevelbuilder/goclaw/internal/channels/discord"
"github.com/nextlevelbuilder/goclaw/internal/channels/feishu"
"github.com/nextlevelbuilder/goclaw/internal/channels/whatsapp"

// REMOVE RegisterFactory calls:
instanceLoader.RegisterFactory("discord", discord.Factory)
instanceLoader.RegisterFactory("feishu", feishu.Factory)
instanceLoader.RegisterFactory("whatsapp", whatsapp.Factory)

// REMOVE config-based init blocks:
// if cfg.Channels.Discord.Enabled && cfg.Channels.Discord.Token != "" { ... }
// if cfg.Channels.WhatsApp.Enabled && cfg.Channels.WhatsApp.BridgeURL != "" { ... }
// if cfg.Channels.Feishu.Enabled && cfg.Channels.Feishu.AppID != "" { ... }
```

**Verification after T9-01:**
```bash
go build ./...   # must compile clean
go test ./...    # must pass ≥290 tests
grep -r "feishu\|discord\|whatsapp" internal/ cmd/ --include="*.go" | grep -v "_test.go" | wc -l
# expect: 0
```

---

### T9-02: Channel Removal — Periphery (P0, 2 pts)

**Objective**: Clean all secondary references (onboarding, agent context, tools, managed mode, tests).

#### Phase 4 — Onboarding + CLI

| File | Action |
|------|--------|
| `cmd/onboard.go` | Remove Feishu wizard step (27 refs) |
| `cmd/onboard_auto.go` | Remove 4 channel branches (feishu, discord, slack, whatsapp) |
| `cmd/onboard_managed.go` | Remove discord + whatsapp initialization |
| `cmd/onboard_helpers.go` | Remove channel-specific helpers |
| `cmd/doctor.go` | Remove 4 channels from diagnostics |
| `cmd/channels_cmd.go` | Remove from CLI help text |

#### Phase 5 — Managed mode type registry

| File | Action |
|------|--------|
| `internal/store/channel_instance_store.go` | Remove feishu/discord/whatsapp from type list |
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
| `cmd/gateway_consumer_test.go` | Remove Discord test case |
| `internal/tools/context_keys_test.go` | Remove Slack constants (3 refs) |

**CTO-note**: Zalo extension at `extensions/zalouser/` is NOT touched — it's a workspace package, not part of this cleanup.

**Verification after T9-02:**
```bash
go build ./...
go test ./...
grep -r "feishu\|discord\|whatsapp\|FeishuConfig\|DiscordConfig\|SlackConfig\|WhatsAppConfig" . --include="*.go" | wc -l
# expect: 0
```

---

### T9-03: SOUL Behavioral Tests — 12 Governance SOULs (P0, 2 pts)

**Objective**: Complete Sprint 8 CPO Condition 3 — test all governance SOULs (5 done in Sprint 8, 12 remaining). CTO-32: CEO added (SE4H advisor, G0.1+G4 gates); assistant excluded (category=router, no governance gates).

**File**: `internal/souls/behavioral_test.go` (extend existing file)

**12 governance SOULs** (5 tests each = 60 new tests):

| SOUL | Role | Focus |
|------|------|-------|
| architect | SE4H advisor | Architecture decisions, ADR output format |
| ceo | SE4H advisor | Strategic decisions, G0.1/G4 gate approvals |
| cpo | SE4H advisor | Product strategy, gate approval format |
| cto | SE4H advisor | Technical standards, performance requirements |
| cs | SE4A executor | Customer support, Vietnamese language |
| devops | SE4A executor | CI/CD, infrastructure patterns |
| fullstack | SE4A executor | Full-stack patterns, TypeScript + Go |
| itadmin | SE4A executor | Infrastructure ops, security hardening |
| pjm | SE4A executor | Project management, timeline format |
| researcher | SE4A executor | Research methodology, evidence format |
| tester | SE4A executor | QA patterns, test coverage requirements |
| writer | SE4A executor | Documentation format, Vietnamese |

**Test structure per SOUL** (same pattern as Sprint 8):
1. `TestSOUL_{Role}_HasYAMLFrontmatter` — YAML frontmatter with role/version/category fields
2. `TestSOUL_{Role}_IdentitySection` — `## Identity` section exists, role keyword present
3. `TestSOUL_{Role}_CapabilitiesSection` — `## Capabilities` exists, role-specific keyword present
4. `TestSOUL_{Role}_ConstraintsSection` — `## Constraints` exists
5. `TestSOUL_{Role}_ChecksumDeterministic` — ChecksumContent returns consistent 64-char SHA-256 hex

**Completion target**: Total behavioral tests after T9-03 = 25 (Sprint 8) + 60 (Sprint 9) = **85 SOUL behavioral tests**.

**Combined with drift E2E**: Full SOUL governance coverage = 85 + 5 (drift E2E) = **90 tests**.

---

### T9-04: MS Teams Extension Scaffold + ADR-007 (P1, 1 pt)

**Objective**: Prepare the extension scaffold so Sprint 10 @coder can start MS Teams implementation immediately. No working implementation — scaffold only.

#### ADR-007 — [@pm] deliverable (merged into T9-05)

File: `docs/02-design/01-ADRs/SPEC-0007-ADR-007-MSTeams-Extension.md`

Sections:
- Problem: NQH management team uses MS Teams; Telegram not suitable for formal corporate comms
- Decision: Implement as `extensions/msteams` workspace package (not core channel)
- Bot framework: Microsoft Bot Framework SDK or incoming webhook + Adaptive Cards
- Auth: Azure AD app registration, OAuth2 token flow
- Deployment: `RegisterFactory("msteams", msteams.Factory)` — zero core code changes
- Dependencies: `@microsoft/botframework-connector` or direct Graph API
- Timeline: Sprint 10 implementation, Sprint 11 NQH pilot

> **CTO-31**: ADR-007 content is @pm scope (T9-05). @coder creates only the directory scaffold.

#### Extension scaffold (@coder only)

```
extensions/msteams/
├── package.json          # workspace package, msteams deps declared
├── README.md             # setup instructions (Azure app registration, etc.)
└── msteams.go.TODO       # scaffold file with interface stub (NOT compilable yet)
```

**Constraint**: The scaffold must NOT break `go build ./...`. The `.go.TODO` file is not a Go source file — it's a reference document with extension (like `.md`). The actual implementation starts in Sprint 10.

---

### T9-05: G4 Gate Proposal [@pm] (P0, 1 pt) — carryover from Sprint 8

**Objective**: Submit G4 (Internal Validation) gate proposal based on Sprint 8 evidence.

**File**: `docs/08-collaborate/G4-GATE-PROPOSAL-SPRINT8.md`

**G4 criteria** (from SDLC 6.1.1):

| Criterion | Evidence |
|-----------|----------|
| All 3 governance rails operational | Rail #1 (Spec), Rail #2 (PR Gate), Rail #3 (RAG) — Sprint 7+8 |
| PR Gate ENFORCE active | GitHub webhook + commit status checks — Sprint 8 |
| SOUL behavioral validation | 25/16 SOULs tested, 5 drift E2E — Sprint 8 |
| Evidence export for audit | JSON + CSV export — Sprint 8 |
| Security: GitHub credentials masked | CTO-27 fix applied — Sprint 8 post-review |
| Audit trail: scan errors logged | CTO-26 fix applied — Sprint 8 post-review |
| Test coverage | 290 tests, all packages — Sprint 8 |

**Submission**: File to `docs/08-collaborate/` and notify @cto via Sprint 9 handoff.

---

## 5. Sprint 9 Timeline

```
Day 1 (Mon): T9-01 Phase 1-2 (delete dirs + config)
Day 2 (Tue): T9-01 Phase 3 + T9-02 Phase 4-5 (gateway + onboard)
Day 3 (Wed): T9-02 Phase 6-7 + verification (agent/tools/tests cleanup)
Day 4 (Thu): T9-03 (55 SOUL behavioral tests)
Day 5 (Fri): T9-04 (MS Teams scaffold + ADR-007) + T9-05 G4 proposal [@pm]
```

---

## 6. Success Criteria

| Metric | Target |
|--------|--------|
| Channels remaining | Telegram + Zalo (+ extensions/msteams scaffold) |
| References to removed channels | 0 (verified by grep) |
| SOUL behavioral tests | 85 total (25 Sprint 8 + 60 Sprint 9) |
| Total test count | ≥290 (no regression) |
| Build clean | `go build ./...` 0 errors |
| G4 proposal filed | ✅ |
| ADR-006 implemented | ✅ (pending @cto approval) |
| ADR-007 drafted | ✅ scaffold + architecture decision |

---

## 7. Dependencies & Risks

| Item | Type | Mitigation |
|------|------|-----------|
| ADR-006 @cto approval | **Blocker** | Filed Sprint 8 post-review. T9-01/T9-02 cannot start without approval. |
| 11 SOUL files must have YAML frontmatter | Risk | If missing: update SOUL files first (not a @coder blocker — SOUL files are markdown) |
| Discord used in gateway_consumer_test.go | Risk | Remove test case — test must be adapted, not deleted wholesale |
| Slack has 0 implementation files | Info | Config struct removal only — minimal risk |
| WhatsApp BridgeURL pattern differs from token-based auth | Info | Different config struct but same cleanup approach |

---

## 8. Gate Dependency

```
Sprint 9 → G4 (Internal Validation, 30 days post-launch)
         → Sprint 10 (MS Teams implementation — pending ADR-007 approval)
```

G4 can proceed in parallel with Sprint 9. G4 evidence is Sprint 7+8 deliverables — Sprint 9 is additive, not a G4 prerequisite.

---

*Sprint 9 is a cleanup + hardening sprint. No new user-facing features. Outcome: leaner, more maintainable codebase ready for Sprint 10 MS Teams integration.*
