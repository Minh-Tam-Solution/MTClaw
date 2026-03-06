# extensions/msteams — MS Teams Bot Framework Channel

**Status**: IMPLEMENTED — Sprint 10 (ADR-007 APPROVED 2026-03-17)

**Sprint**: Sprint 10 — MS Teams Extension + NQH Corporate Rollout
**ADR**: `docs/02-design/01-ADRs/SPEC-0007-ADR-007-MSTeams-Extension.md`

---

## Production Security Requirement — Tenant ID

`MSTEAMS_TENANT_ID` **MUST** be set to your organization's specific Azure tenant ID.

**NEVER use `common`** — this would allow any Microsoft 365 user worldwide to reach your bot.

To find your tenant ID:
1. Azure Portal → Azure Active Directory → Overview → Tenant ID
2. Or: `az account show --query tenantId -o tsv`

Example:
- MTS deployment: `MSTEAMS_TENANT_ID=<mts-azure-tenant-id>`
- NQH deployment: `MSTEAMS_TENANT_ID=<nqh-azure-tenant-id>`

Reference: ADR-007 Section "CTO Decisions" — MTS tenant only for Phase 1.

---

## Overview

MS Teams integration for MTClaw via Microsoft Bot Framework v3 REST API.

**Target users**: MTS Engineering team + NQH Head Office (management users who prefer Teams over Telegram).

Architecture:
- Zero core coupling: `RegisterFactory("msteams", msteams.Factory)` — one line in `cmd/gateway.go`
- Inbound: Bot Framework JWT verification → Activity parsing → bus.PublishInbound
- Outbound: `client_credentials` token acquisition → POST Bot Framework REST API
- Auth: Azure AD App password (Sprint 10). Managed Identity not applicable (NQH Docker infra).

## Files

| File | Purpose |
|------|---------|
| `msteams.go` | Package entry: Config + Factory (managed mode) |
| `channel.go` | MSTeamsChannel: Start, Stop, Send, RegisterRoutes |
| `auth.go` | Bot Framework token acquisition + 5-min expiry cache |
| `jwt.go` | Bot Framework JWT verification via OpenID → JWKS → RSA |
| `webhook.go` | HTTP handler: parse Activity, route to bus.PublishInbound |
| `cards.go` | Adaptive Card builders: SpecCard, PRReviewCard |
| `msteams_test.go` | 16 unit tests |

## Setup

### Prerequisites

1. **Azure AD App Registration** at [portal.azure.com](https://portal.azure.com)
   - Note: App (client) ID and a client secret

2. **Bot Channel registration** at [dev.botframework.com](https://dev.botframework.com)
   - Messaging endpoint: `https://<your-domain>/v1/channels/msteams/webhook`
   - Bot handle and Microsoft App ID

3. **Environment variables** (add to `.env.local`):
   ```bash
   export MSTEAMS_APP_ID=<azure-app-id>
   export MSTEAMS_APP_PASSWORD=<azure-app-secret>   # high-value credential — never log
   export MSTEAMS_TENANT_ID=<your-org-tenant-id>   # NEVER "common"
   ```

### App Password Rotation

Bot Framework app passwords expire or require manual rotation. Procedure:
1. Azure Portal → App Registrations → {your-app} → Certificates & secrets → New client secret
2. Update `MSTEAMS_APP_PASSWORD` in deployment environment
3. Restart the gateway process
4. Monitor auth failure logs: `grep "failed to acquire token" /var/log/mtclaw.log`

---

## Adaptive Cards (T10-03)

When a governance processor detects `channel == "msteams"`, set:
```go
outbound.Format = "adaptive_card"
outbound.Content = string(msteams.SpecCard(spec.ID, spec.Title, spec.Status, scenarios))
```

Available builders:
- `SpecCard(specID, title, status string, scenarios []string) json.RawMessage`
- `PRReviewCard(prURL, verdict string, blockRules, warnRules []string) json.RawMessage`

---

*Sprint 10: implemented. Sprint 11: NQH pilot deployment.*
