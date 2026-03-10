---
sprint: 30
title: "SOUL 6.1.2 Upgrade + Discord Channel"
status: IN_PROGRESS
start_date: 2026-03-10
end_date: 2026-03-14
lead: "@pm (docs) → @coder (implementation)"
framework: SDLC Enterprise Framework 6.1.2
---

# Sprint 30 — SOUL 6.1.2 Upgrade + Discord Channel

## Sprint Goal

Align MTClaw SOULs with SDLC Enterprise Framework 6.1.2 (SASE artifacts) and re-add Discord channel for Vietnamese dev team accessibility.

---

## Part A: SOUL 6.1.2 Upgrade (PM — document work)

### A1: Version Bump + Frontmatter ✅

| Task | Files | Status |
|------|-------|--------|
| Add `sdlc_framework: "6.1.2"` to all 18 SOULs | `docs/08-collaborate/souls/SOUL-*.md` | ✅ Done |
| Add/update `framework: SDLC Enterprise Framework 6.1.2` | 18 SOULs | ✅ Done |
| Replace `6.1.1` → `6.1.2` in all SOUL content | 18 SOULs | ✅ Done |
| Mark `SOUL-enghelp.md` with `extension: mtclaw` | 1 file | ✅ Done |
| Update migration 000009 inline SOUL content | `migrations/000009_seed_mtclaw_souls.up.sql` | ✅ Done |
| Update CLAUDE.md (17→18 SOULs, 6.1.2 ref) | `CLAUDE.md` | ✅ Done |

### A2: SASE Workflow Awareness ✅

| Task | Target SOULs | Status |
|------|-------------|--------|
| CRP escalation sections | 9 SE4A executors (coder, fullstack, architect, pm, pjm, reviewer, tester, devops, researcher) | ✅ Done |
| MRP completion sections | 3 code-producing (coder, fullstack, devops) | ✅ Done |
| VCR review sections | 3 SE4H advisors (cto, cpo, ceo) | ✅ Done |
| AGENTS.md awareness | assistant (router) | ✅ Done |

### A3: CTO Mandatory Fixes ✅

| Task | Files | Status |
|------|-------|--------|
| Fix migration 000012 `6.1.1` → `6.1.2` | `migrations/000012_seed_itadmin_soul.up.sql` | ✅ Done |
| Create fullstack seed migration 000020 | `migrations/000020_seed_fullstack_soul.up.sql` + `.down.sql` | ✅ Done |
| Bump RequiredSchemaVersion 19 → 20 | `internal/upgrade/version.go` | ✅ Done |
| Update CLAUDE.md migration count 19 → 20 | `CLAUDE.md` | ✅ Done |

### A Commit

```
feat(souls): upgrade 18 SOUL templates to SDLC 6.1.2 alignment
feat(souls): add SASE workflow awareness (CRP/MRP/VCR) to SOULs
```

---

## Part B: Discord Channel (@coder — implementation)

### Pre-requisites (before coding starts)

- [x] **ADR-006-Amendment** written and awaiting CTO approval
- [ ] **Discord bot** created in Developer Portal with:
  - Message Content Intent **enabled** (privileged — required since Sept 2022)
  - Server Members Intent **enabled**
  - Bot token provided as `MTCLAW_DISCORD_TOKEN`

### B1: Config struct + env vars

**Files to create/edit:**

| File | Change |
|------|--------|
| `internal/config/config_channels.go` | Add `DiscordConfig` struct, add `Discord` field to `ChannelsConfig` |
| `internal/config/config.go` | Add `MTCLAW_DISCORD_TOKEN` env loading |
| `.env.example` | Add `MTCLAW_DISCORD_TOKEN=` |

**DiscordConfig struct:**
```go
type DiscordConfig struct {
    Enabled        bool                `json:"enabled"`
    Token          string              `json:"-"`                      // env only
    AllowFrom      FlexibleStringSlice `json:"allow_from,omitempty"`
    DMPolicy       string              `json:"dm_policy,omitempty"`    // pairing|allowlist|open|disabled
    GroupPolicy    string              `json:"group_policy,omitempty"` // open|allowlist|disabled
    RequireMention bool                `json:"require_mention,omitempty"`
    GuildIDs       FlexibleStringSlice `json:"guild_ids,omitempty"`
}
```

### B2: Channel implementation

**New file**: `internal/channels/discord/discord.go` (~350 lines)

Follow **Zalo pattern** (`internal/channels/zalo/zalo.go`):

```go
type Channel struct {
    *channels.BaseChannel
    token           string
    session         *discordgo.Session
    dmPolicy        string
    groupPolicy     string
    requireMention  bool
    guildIDs        map[string]bool
    pairingService  store.PairingStore
    pairingDebounce sync.Map
    stopCh          chan struct{}
    botUserID       string
}
```

**Methods:**
- `New(cfg, msgBus, pairingSvc)` — configure discordgo session + intents
- `Start(ctx)` — open Gateway WebSocket, register handlers, store `botUserID`
- `Stop(ctx)` — `session.Close()`
- `Send(ctx, msg)` — chunked text (2000 chars) + embed for media
- `handleMessageCreate(s, m)` — skip bots, check DM/guild, enforce policy
- `checkPolicy(peerKind, senderID)` — reuse Zalo pattern
- `sendPairingReply(senderID, channelID)` — pairing code flow
- `sendChunkedText(channelID, text)` — 2000 char chunking, newline break
- `isMentioned(m)` — check `botUserID` in `m.Mentions`

**Key design decisions:**
- **SenderID format**: `"discord_user_id|username"` (no discriminator — deprecated May 2023)
- **ChatID**: Discord channel ID (string)
- **PeerKind**: `"direct"` for DMs, `"group"` for guild channels
- **Metadata**: `{"platform": "discord", "message_id": "...", "guild_id": "..."}`
- **GuildIDs default**: empty = **no guilds allowed** (security: opt-in only)
- **Gateway intents**: `IntentsGuildMessages | IntentsDirectMessages | IntentsMessageContent`

**Streaming + Reactions**: DEFERRED to Sprint 31.
- `ReactionChannel` interface uses `messageID int` — Discord uses string snowflakes
- Needs interface refactor before Discord reactions can work

### B3: Factory for managed mode

**New file**: `internal/channels/discord/factory.go` (~60 lines)

Follow Zalo factory pattern exactly. Channel type for DB: `"discord"`.

### B4: Gateway wiring

**File**: `cmd/gateway.go`

Two insertion points:
1. **Line ~818** (after MSTeams factory): `instanceLoader.RegisterFactory("discord", discord.Factory)`
2. **Line ~846** (after MSTeams config init): config-based Discord channel init block

### B5: Test updates (CRITICAL)

**File**: `internal/integration/governance_e2e_test.go:427-431`

Remove `"../../internal/channels/discord"` from `removedDirs` in `TestE2E_ChannelCleanup_RemovedChannelsDontExist`.

**Note**: MSTeams guard tests (`msteams_test.go:432-461`) are NOT affected — they check MSTeams-specific files only.

### B6: Discord unit tests

**New file**: `internal/channels/discord/discord_test.go` (~200 lines)

13 tests covering: token validation, DM policy variants, group policy, chunking, mention detection, senderID format, factory decode.

### B7: Docker Compose + docs

| File | Change |
|------|--------|
| `docker-compose.mts.yml` | Add `MTCLAW_DISCORD_TOKEN` env var |
| `docs/06-deploy/env-template.md` | Add Discord section |
| `docs/06-deploy/deployment-guide.md` | Add Discord channel + intents setup |
| `docs/06-deploy/runbook.md` | Add Discord health check |

### B Commits

```
feat(discord): add Discord channel with DM/group policy support
docs: add Discord channel to deployment docs
```

---

## Reference Files

| Purpose | File |
|---------|------|
| Channel interface | `internal/channels/channel.go` |
| Zalo channel (**primary reference**) | `internal/channels/zalo/zalo.go` |
| Zalo factory | `internal/channels/zalo/factory.go` |
| Instance loader | `internal/channels/instance_loader.go` |
| Gateway wiring | `cmd/gateway.go:813-846` |
| Config structs | `internal/config/config_channels.go` |
| E2E test to update | `internal/integration/governance_e2e_test.go:426-438` |
| ADR-006 (original removal) | `docs/02-design/01-ADRs/SPEC-0006-ADR-006-Channel-Rationalization.md` |
| ADR-006-Amendment (re-add) | `docs/02-design/01-ADRs/SPEC-0006-ADR-006-Amendment-Discord-Readd.md` |
| MTS-OpenClaw Discord (TS ref) | `/home/nqh/shared/MTS-OpenClaw/src/discord/` |
| discordgo library | Already in go.mod: `github.com/bwmarrin/discordgo v0.29.0` |

---

## Verification Checklist

### Part A
- [ ] `make souls-validate` passes
- [ ] `grep -rn '6\.1\.1' docs/08-collaborate/souls/` returns 0 results
- [ ] All 18 SOULs have `sdlc_framework: "6.1.2"` in frontmatter

### Part B
- [ ] `make test` passes (including updated governance E2E test)
- [ ] `make build` compiles cleanly
- [ ] Discord bot connects and shows online status
- [ ] DM bot → receives response from agent
- [ ] @mention bot in guild → processes message
- [ ] Response >2000 chars → chunked correctly
- [ ] New user DMs → pairing code → approve → chat works
- [ ] Guild not in `guild_ids` → message ignored (security)

---

## Risk Register

| Risk | Impact | Mitigation | Owner |
|------|--------|------------|-------|
| Discord bot intents not enabled | Blocker | P1 pre-req: verify before coding | @devops |
| `messageID int` vs snowflake | None (this sprint) | Reactions deferred to Sprint 31 | @architect |
| governance_e2e_test fails | Blocker | B5 removes discord from removedDirs | @coder |
| discordgo v0.29.0 compat | Low | Supports Gateway v10 (verified) | @coder |
| SOUL char budget exceeded | Low | Docs SOULs are reference; migration content budget-constrained | @pm |

---

## Deferred to Sprint 31

| Item | Reason |
|------|--------|
| Discord streaming (message edit) | Ship basic first |
| Discord reactions | `ReactionChannel.messageID int` → string refactor needed |
| Discord slash commands (`/help`, `/spec`) | Enhancement after basic validates |
| TEAM charter upgrade (10 teams) | After SOUL upgrade validates pattern |
| `ReactionChannel` interface refactor | Prerequisite for Discord reactions |
