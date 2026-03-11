---
sprint: 30
title: "SOUL 6.1.2 Upgrade + Discord Channel"
status: DONE
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

### Pre-requisites ✅

- [x] **ADR-006-Amendment** approved
- [x] **Discord bot** created in Developer Portal with intents enabled
- [x] Bot token set as `MTCLAW_DISCORD_TOKEN`

### B1: Config struct + env vars ✅

| File | Change | Status |
|------|--------|--------|
| `internal/config/config_channels.go` | Add `DiscordConfig` struct, add `Discord` field to `ChannelsConfig` | ✅ Done |
| `internal/config/config_load.go` | Add `MTCLAW_DISCORD_TOKEN` env loading + auto-enable | ✅ Done |
| `internal/config/config_secrets.go` | Add Discord token to mask/strip/stripMasked | ✅ Done |
| `.env.example` | Add `MTCLAW_DISCORD_TOKEN=` | ✅ Done |

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

### B2: Channel implementation ✅

**File**: `internal/channels/discord/discord.go` (~290 lines) — Follows Zalo pattern.

### B3: Factory for managed mode ✅

**File**: `internal/channels/discord/factory.go` (~70 lines) — Zalo factory pattern, channel type `"discord"`.

### B4: Gateway wiring ✅

**File**: `cmd/gateway.go` — Factory registration + env-based init (works in managed mode).

### B5: Test updates ✅

**File**: `internal/integration/governance_e2e_test.go` — Discord removed from `removedDirs`.

### B6: Discord unit tests ✅

**File**: `internal/channels/discord/discord_test.go` — 16 tests, all passing.

### B7: Docker Compose + docs ✅

| File | Change | Status |
|------|--------|--------|
| `docker-compose.mts.yml` | Add `MTCLAW_DISCORD_TOKEN` env var | ✅ Done |
| `docs/06-deploy/env-template.md` | Add Discord section | ✅ Done |
| `docs/06-deploy/deployment-guide.md` | Add Discord channel + intents setup | ✅ Done |
| `docs/06-deploy/runbook.md` | Add Discord health check | ✅ Done |

### B Commits

```
7858eaa feat(discord): add Discord channel with DM/group policy support
a8196a2 fix(discord): allow env-based init in managed mode
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

### Part A ✅
- [x] `make souls-validate` passes
- [x] `grep -rn '6\.1\.1' docs/08-collaborate/souls/` returns 0 results
- [x] All 18 SOULs have `sdlc_framework: "6.1.2"` in frontmatter

### Part B ✅
- [x] `make test` passes (27 packages, including updated governance E2E test)
- [x] `make build` compiles cleanly
- [x] Discord bot connects and shows online status
- [x] DM bot → pairing code sent → approve → chat works
- [x] Agent routes message to LLM → response received via Discord
- [ ] @mention bot in guild → processes message (not tested — no guild_ids configured)
- [ ] Response >2000 chars → chunked correctly (unit tested, not E2E tested)
- [x] New user DMs → pairing code → `mtclaw pairing approve <code>` → chat works

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
