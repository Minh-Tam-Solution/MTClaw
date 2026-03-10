---
id: ADR-006-Amendment
title: "ADR-006 Amendment — Re-add Discord Channel (Sprint 30)"
status: APPROVED
author: "@pm"
date: 2026-03-10
sprint: 30
amends: ADR-006
related_adrs: [ADR-006, ADR-004, ADR-007]
tags: [channels, discord, vietnam, accessibility]
framework: SDLC Enterprise Framework 6.1.2
---

# ADR-006 Amendment — Re-add Discord Channel

> **Status**: APPROVED — [@cto] 2026-03-10
> **Author**: [@pm]
> **Date**: 2026-03-10
> **Amends**: ADR-006 (Channel Rationalization, 2026-03-17)

---

## 1. Context

ADR-006 (Sprint 9) removed Discord along with Feishu, Slack, and WhatsApp — all had zero active users. The rationale was sound: ~477 LOC of dead code with no adoption plan.

**What changed since ADR-006:**

| Factor | ADR-006 (Sprint 9) | Now (Sprint 30) |
|--------|-------------------|-----------------|
| Discord users | 0 | Dev team requesting access |
| Telegram accessibility in VN | Not discussed | **Blocking issue** — registration requires VPN/foreign phone in many cases |
| Channel count | 2 (Telegram + Zalo) | 3 (+ MS Teams added Sprint 12, ADR-007) |
| Channel interface maturity | Basic | Stable — factory pattern, InstanceLoader, managed mode |
| Codebase size | ~2,836 LOC removed | Discord re-add: ~410 lines Go (lean port) |

**Demand signal**: MTS Vietnamese dev team members report difficulty registering Telegram accounts. Vietnam telecom operators intermittently block Telegram SMS verification. Discord has no such restriction — accounts are email-based.

---

## 2. Problem Statement

> Vietnamese developers on the MTS team cannot reliably access MTClaw via Telegram due to account registration barriers. The web UI (port 18791) exists but lacks mobile push notifications and conversational UX. A familiar, accessible messaging channel is needed.

---

## 3. Decision

**Re-add Discord as a supported channel in MTClaw.**

Implementation scope (Sprint 30 — basic channel only):
- Gateway WebSocket connection via `discordgo` v0.29.0 (already in go.mod)
- DM + guild message handling with DM/Group policy (pairing, allowlist, open, disabled)
- Text sending with 2000-char chunking
- Factory for managed mode (DB channel_instances)

**Deferred to Sprint 31:**
- Streaming (message edit) — needs implementation
- Reactions — needs `ReactionChannel` interface refactor (`messageID int` → string for Discord snowflakes)
- Slash commands
- Guild admin actions

---

## 4. Why This Doesn't Contradict ADR-006

ADR-006 Section 5 (Consequences → Negative/Risks) explicitly anticipated this:

> *"If a user requests Discord/Feishu/WhatsApp in future: must re-implement from scratch (or from git history)"*
> *"Mitigation: Git history preserves the code. Re-integration is straightforward with existing channel interface."*

The decision to remove was based on **zero users at the time**. The decision to re-add is based on **real demand from the Vietnamese dev team**. ADR-006's own risk mitigation applies.

**Key difference from Sprint 9 code:**
- Sprint 9 Discord: ~477 lines Go, ported from MTS-OpenClaw TS (6,678 lines), full-featured
- Sprint 30 Discord: ~410 lines Go, lean implementation following Zalo pattern (simplest channel)

---

## 5. Alternatives Considered

### Option A: Telegram proxy / VPN
**Rejected**: Adds operational complexity. Requires each developer to maintain VPN access. Doesn't solve the root cause (Telegram registration blocked).

### Option B: Web UI only (port 18791)
**Rejected**: Already available but lacks push notifications, mobile app, and conversational UX that messaging platforms provide. Developers prefer familiar tools.

### Option C (Chosen): Re-add Discord
Lean Go implementation. `discordgo` already in go.mod. Channel interface and factory pattern proven with 3 existing channels. Minimal risk.

### Option D: Zalo for VN developers
**Rejected**: Zalo OA Bot API has strict business verification requirements. MTClaw is an internal dev tool, not a consumer-facing business.

---

## 6. Consequences

### Positive
- Vietnamese dev team gains reliable access to MTClaw agents
- Email-based account registration — no telecom dependency
- Discord mobile app provides push notifications
- Reuses proven channel patterns (factory, BaseChannel, policy engine)

### Negative / Risks
- +~410 LOC maintenance surface (mitigated: follows established Zalo pattern)
- Another dependency on external platform (mitigated: discordgo is well-maintained, 10K+ GitHub stars)
- Discord Gateway Privileged Intents require Developer Portal configuration (documented in deployment runbook)

---

## 7. Implementation Reference

See Sprint 30 plan: `docs/04-build/02-Sprint-Plans/SPRINT-030-SOUL-DISCORD.md`

Pattern: `internal/channels/discord/` following `internal/channels/zalo/` structure.
- `discord.go` — Channel struct, Start/Stop/Send, policy, chunking
- `factory.go` — DB instance factory for managed mode
- `discord_test.go` — 13 unit tests

---

## 8. Approval

**Status: APPROVED — [@cto] 2026-03-10**

| Decision | Recommendation |
|----------|---------------|
| Re-add Discord channel | ✅ RE-ADD — real demand from VN dev team |
| Lean implementation (basic only) | ✅ APPROVED — streaming/reactions deferred |
| ADR-006 removal rationale still valid for Feishu/Slack/WhatsApp | ✅ UNCHANGED — still zero users |

---

*[@pm] business justification: Vietnamese dev team accessibility — Telegram registration blocked by VN telecom operators*
*[@architect] technical justification: proven channel interface, factory pattern, `discordgo` already in go.mod, ~410 LOC lean implementation*
