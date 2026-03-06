---
spec_id: SPEC-0007
adr_id: ADR-007
title: MS Teams Channel Extension — Sprint 10
status: APPROVED
date: 2026-03-17
author: "[@pm]"
reviewers: "[@cto], [@architect]"
approved_by: "@cto"
approval_date: 2026-03-17
sdlc_version: "6.1.1"
implements: "FR-005"
related_adrs: [ADR-002, ADR-006]
---

# ADR-007: MS Teams Channel Extension

**SDLC Stage**: 02-Design
**Status**: APPROVED — [@cto] 2026-03-17
**Date**: 2026-03-17

---

## Context

ADR-006 (APPROVED 2026-03-17) removed 4 unused channels (Feishu, Discord, Slack, WhatsApp) and retained Telegram + Zalo. ADR-006 explicitly deferred MS Teams to Sprint 10+ with the instruction: "implement as a proper extension in `extensions/msteams` following the existing extension pattern."

Sprint 9 created the extension scaffold (`extensions/msteams/README.md` + `extensions/msteams/msteams.go.TODO`) and this ADR documents the architecture decisions for Sprint 10 implementation.

**Target users**: MTS Engineering team + NQH Head Office (management users who prefer Teams over Telegram).

**Use case**: Enterprise messaging integration for organizations running Microsoft 365. Complements Telegram (consumer app) + Zalo (Vietnamese mobile) with a corporate channel option.

---

## Problem Statement

> MS Teams is the primary messaging platform for MTS corporate communication (meetings, shared channels). A subset of users (management, enterprise clients) prefer Teams over Telegram for AI assistant interaction. Without a Teams channel, these users must switch context to Telegram for MTClaw access.

**Scope**: Inbound messages from Teams personal chat and channel mentions → MTClaw SOUL routing → reply via Teams Bot Framework.

---

## Decision

Implement MS Teams as an **extension package** in `extensions/msteams/`, following the established channel factory pattern. No changes to core code (`internal/channels/`, `internal/gateway/`, `cmd/gateway.go`) beyond adding one `RegisterFactory` call.

**Integration pattern**:
```
Teams user message (Bot Framework Activity)
    │
    ▼
extensions/msteams/webhook.go  (POST /v1/channels/msteams/webhook)
    │
    ├─ Verify Bot Framework JWT (Service URL validation + tenant check)
    ├─ Parse Activity (from.id, text, conversation.id)
    └─ bus.PublishInbound(InboundMessage{
         AgentID: resolved SOUL (from pairing or default),
         From:    teams_user_id,
         Content: activity.Text,
         Channel: "msteams",
       })
    │
    ▼
gateway_consumer.go → SOUL routing → reply
    │
    ▼
extensions/msteams/send.go
    └─ POST /v3/conversations/{conversationId}/activities (Bot Framework REST API)
```

---

## Alternatives Considered

### Option A: Implement via Microsoft Graph API only
**Rejected**: Graph API requires application permissions and admin consent for org-wide access. Bot Framework is the standard approach for interactive bots — simpler auth model (Bot registration only), better event delivery guarantees.

### Option B: Use Azure Communication Services
**Rejected**: ACS is for custom apps, not Teams-native integration. Users would not see messages inside Teams.

### Option C (Chosen): Bot Framework v3 REST API
Standard Teams bot integration. Uses Bot Framework token authentication (service URL + HMAC validation). Well-documented, stable API surface. Same webhook pattern as GitHub webhook (Sprint 8).

### Option D: Teams Incoming Webhook (one-way only)
**Rejected**: Incoming webhooks are send-only. Cannot receive user messages. Not suitable for an interactive assistant.

---

## Architecture

### Authentication

**Inbound (Teams → MTClaw)**:
- Bot Framework sends signed JWT in `Authorization` header
- Verify via Bot Framework OpenID metadata: `https://login.botframework.com/v1/.well-known/openidconfiguration`
- Validate: `iss` = `https://api.botframework.com`, `aud` = `MSTEAMS_APP_ID`

**Outbound (MTClaw → Teams)**:
- Acquire token: `POST https://login.microsoftonline.com/botframework.com/oauth2/v2.0/token`
- Scope: `https://api.botframework.com/.default`
- Token cache with 5-minute expiry buffer

### Message Routing

| Teams event | Action |
|------------|--------|
| `message` (personal chat) | Route to paired SOUL or default pm SOUL |
| `message` (channel @mention) | Route to default pm SOUL (channel context) |
| `conversationUpdate` (member added) | Trigger onboarding message |
| Other event types | Acknowledge 200, no-op |

**Pairing**: Same pairing store used by Telegram (`store.PairingStore`). Key: `msteams:{teams_user_id}`. Allows users to bind Teams identity to a specific SOUL via `/pair <soul>` command.

### Extension Package Structure

```
extensions/msteams/
├── README.md                  — setup + env vars (Sprint 9, created)
├── msteams.go.TODO            — interface stub, not compiled (Sprint 9, created)
├── msteams.go                 — package entry, Config + Factory (Sprint 10)
├── channel.go                 — MSTeamsChannel, Start/Stop/Send/RegisterRoutes (Sprint 10)
├── webhook.go                 — HTTP handler, JWT verification, Activity parsing (Sprint 10)
├── auth.go                    — Bot Framework token acquisition + cache (Sprint 10)
└── msteams_test.go            — unit tests: JWT verify, activity parsing, send (Sprint 10)
```

### Configuration

```yaml
# config.yaml additions (Sprint 10)
channels:
  msteams:
    enabled: false  # default off until configured
    app_id: "${MSTEAMS_APP_ID}"
    app_password: "${MSTEAMS_APP_PASSWORD}"  # AES-256-GCM encrypted at rest
    tenant_id: "${MSTEAMS_TENANT_ID}"       # "common" for multi-tenant
    webhook_path: "/v1/channels/msteams/webhook"
```

**New env vars** (`.env.example`):
```
MSTEAMS_APP_ID=
MSTEAMS_APP_PASSWORD=
MSTEAMS_TENANT_ID=common
```

### Gateway Wiring (Sprint 10 — single line addition)

```go
// cmd/gateway.go — the only core change needed
instanceLoader.RegisterFactory("msteams", msteams.Factory)
```

No other core files change. Extension pattern enforces clean separation.

---

## Consequences

### Positive
- **Zero core coupling**: Extension pattern means `internal/` unchanged beyond optional factory registration
- **Consistent auth**: Follows Sprint 8 GitHub webhook JWT/HMAC pattern — same security model
- **Reuses pairing store**: No new infrastructure — Teams user pairing via existing `store.PairingStore`
- **Incremental deployment**: Can be enabled per-tenant without affecting Telegram/Zalo users

### Negative / Risks
- **Azure AD setup required**: Bot Framework registration + Azure app require MTS IT admin access (est. 1-2 days)
- **Private tenant restriction**: `tenant_id` must be set for MTS-internal deployment (avoid `common` in production to prevent external-user messages)
- **App password rotation**: Bot Framework app passwords expire or require manual rotation — no automatic renewal via Azure SDK
  - **Mitigation**: Document rotation procedure in `extensions/msteams/README.md`. Alert on auth failure.

### Neutral
- MS Teams webhook uses HTTP POST (same as Telegram/GitHub) — existing HTTP mux handles it cleanly
- Bot Framework token acquisition adds one HTTP call per response cycle (~50ms) — acceptable latency

---

## Implementation Plan (Sprint 10)

| Subtask | File | Effort |
|---------|------|--------|
| Config struct + env loading | `msteams.go` | 0.5 day |
| Bot Framework JWT verification | `auth.go` + `webhook.go` | 1 day |
| Activity parsing + bus publish | `webhook.go` | 0.5 day |
| Send via Bot Framework REST | `channel.go` | 0.5 day |
| Unit tests | `msteams_test.go` | 0.5 day |
| Integration + gateway wiring | `cmd/gateway.go` (1 line) | 0.5 day |
| **Total** | | **~3.5 days** |

**Sprint 10 entry criteria**:
- Azure AD app registered with Bot Framework
- `MSTEAMS_APP_ID` + `MSTEAMS_APP_PASSWORD` provisioned to development environment
- ADR-007 approved by @cto

---

## Secrets Handling

`MSTEAMS_APP_PASSWORD` is a high-value credential (allows impersonating the bot to any Teams user).

Requirements:
1. Stored via `AES-256-GCM` encryption in DB (same as Telegram token — `config_secrets.go` masking)
2. Masked in `MaskedCopy()` and `StripSecrets()` (same pattern as `Telegram.Token`)
3. Never logged — `slog` must use masked config
4. Rotation procedure documented in README

---

## CTO Decisions ([@cto] 2026-03-17)

| Question | Decision |
|----------|----------|
| Tenant restriction | **MTS tenant only** (`tenant_id=mts-tenant-id`) for Phase 1. `common` would allow any Microsoft org user to reach the bot — unacceptable. Document this restriction in `extensions/msteams/README.md` as a production requirement. |
| Channel @mention behavior | **Respond in channel (same thread)**. Private reply to a channel @mention is unexpected UX and breaks conversation context for other viewers. |
| Bot auth: app password vs Managed Identity | **App password for Sprint 10**. Managed Identity requires Azure hosting (AKS/App Service) — MTClaw runs on NQH Docker infrastructure, not applicable. Revisit if Azure-hosted deployment is adopted. |

**Additional @cto note**: `MSTEAMS_APP_PASSWORD` masking must be added to `config_secrets.go` in Sprint 10 (same CTO-27 pattern as GitHub credentials). Already called out in Secrets Handling section — @coder must include in Sprint 10 implementation checklist.

---

## CTO Approval

**Status**: APPROVED — [@cto] 2026-03-17

Sprint 10 scope green-lit. Architecture sound. Bot Framework REST API is the correct choice. JWT verification design (OpenID metadata + iss/aud validation) is standard pattern.

---

*[@pm] business justification: MS Teams integration enables enterprise adoption by management users who prefer corporate messaging. Extension pattern ensures zero maintenance cost if unused.*
*[@architect] to review Bot Framework JWT verification design before Sprint 10 start.*
*[@cto] approved 2026-03-17 — architecture sound, open questions resolved above.*
