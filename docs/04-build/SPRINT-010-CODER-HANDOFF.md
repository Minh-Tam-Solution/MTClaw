# Sprint 10 — @coder Handoff

**Sprint**: 10 — MS Teams Extension + NQH Corporate Rollout
**Date**: 2026-03-17
**From**: [@pm] + [@architect]
**To**: [@coder]
**CTO Approval**: ✅ Sprint 10 Plan APPROVED 8.5/10 (2026-03-17) — UNBLOCKED (pending @devops Azure pre-work)
**CTO Score (Sprint 9)**: 9.0/10 — best sprint score so far

---

## What's Already Done (Sprint 9 Deliverables)

All Sprint 9 code committed and verified (`go build` clean, 350 tests PASS):

| Deliverable | Files | Status |
|-------------|-------|--------|
| Channel removal: Feishu/Discord/WhatsApp/Slack | ~17 files deleted, 7-phase cleanup | ✅ |
| SOUL behavioral tests (12 SOULs × 5 = 60 tests) | `internal/souls/behavioral_test.go` | ✅ |
| MS Teams scaffold | `extensions/msteams/README.md`, `extensions/msteams/msteams.go.TODO` | ✅ |
| G4 gate proposal | `docs/08-collaborate/G4-GATE-PROPOSAL-SPRINT8.md` | ✅ |
| ADR-007 | `docs/02-design/01-ADRs/SPEC-0007-ADR-007-MSTeams-Extension.md` | ✅ APPROVED |

---

## CTO Notes for Sprint 10 (MUST READ)

### CTO-35 (P1) — JWKS key lookup MUST be fully implemented in T10-01 (not a stub)

The `jwt.go` plan shows `fetchJWKSKey` returning an error stub. **This is not a deferred item** — it is the core of the inbound security model. Every webhook call will return 401 until it is implemented.

Implement as part of T10-01 Phase 2 (auth.go + jwt.go day):

1. HTTP GET `jwksURI` (from `botFrameworkOpenIDURL` OpenID metadata)
2. Parse `{"keys": [...]}` response — find key where `kid` matches + `use == "sig"` + `kty == "RSA"`
3. Extract `n` + `e` (base64url-encoded) → `rsa.PublicKey` using `x509.ParsePKIXPublicKey` or manual big.Int decode
4. Cache the JWKS response with 24h TTL alongside `tokenCache` — keys rotate infrequently; add `jwksCache` struct
5. On `kid` miss: force-refresh JWKS (key rollover event)

The T10-02 JWT middleware tests must use the **real implementation path** with an injected mock key resolver — not bypass the JWKS chain.

### CTO-36 (P1) — cards.go: add "strings" import (compile error)

`PRReviewCard()` in `cards.go` calls `strings.ToUpper(verdict)` but the plan's import block only lists `"encoding/json"`. This is a compile error that will block T10-03 entirely.

Fix: import block must be:
```go
import (
    "encoding/json"
    "strings"
)
```

### CTO-37 (P1) — Channel column migration 000016 required before T10-04C SQL queries

The T10-04C cross-channel governance SQL queries (`SELECT channel FROM traces/governance_specs/pr_gate_evaluations`) will fail with "column does not exist" if `channel` was not added in earlier migrations.

**Before T10-04 execution**:
```bash
grep -n "channel" /home/nqh/shared/MTClaw/migrations/000013_governance_specs.sql
grep -n "channel" /home/nqh/shared/MTClaw/migrations/000015_pr_gate_evaluations.sql
```

If absent: create `migrations/000016_add_channel_to_governance_tables.up.sql` and apply before DoD SQL checks:
```sql
ALTER TABLE governance_specs ADD COLUMN IF NOT EXISTS channel VARCHAR(32);
ALTER TABLE pr_gate_evaluations ADD COLUMN IF NOT EXISTS channel VARCHAR(32);
```

T10-04C is a DoD item — it MUST succeed, not be skipped.

### CTO-38 (P2) — MSTeams AppSecret masking: use maskNonEmpty() helper, not inline if

The plan shows:
```go
if out.Channels.MSTeams.AppSecret != "" {
    out.Channels.MSTeams.AppSecret = MaskedValue
}
```

The established pattern from CTO-27 (Sprint 8) is:
```go
maskNonEmpty(&cp.Channels.MSTeams.AppSecret)
```

Use `maskNonEmpty()` in all 3 functions (`MaskedCopy`, `StripSecrets`, `StripMaskedSecrets`). The helper exists to prevent exactly this pattern drift.

### CTO-39 (P2) — auth.go and jwt.go: use http.Client with timeout, not http.DefaultClient

`http.DefaultClient` has no timeout. Bot Framework token endpoint + JWKS endpoint are external services — a hung connection blocks the goroutine indefinitely. Same issue as CTO-23 (Sprint 8, GitHub client).

Fix: add a package-level client in `auth.go` (used by both auth.go and jwt.go):
```go
var httpClient = &http.Client{Timeout: 10 * time.Second}
```

Replace `http.DefaultClient.Do(req)` with `httpClient.Do(req)` in both `auth.go` (token acquisition) and `jwt.go` (OpenID metadata + JWKS fetch).

---

### CTO-33 (P3) — Fix residual Discord references during gateway.go touchpoint

3 non-functional Discord references survived Sprint 9 cleanup:

```
cmd/gateway_consumer.go:46   — comment: "channels (Telegram, Discord, etc.)"
cmd/gateway_consumer.go:138  — comment: session-key description mentioning Discord
cmd/gateway_builtin_tools.go:68 — string: "Send messages to connected channels (Telegram, Discord, etc.)"
```

**Fix during T10-01 Phase 6** (gateway.go is already a T10 touchpoint for RegisterFactory wiring).
- Update comments to use "Telegram, Zalo, MSTeams, etc." or make channel-neutral
- String literal at line 68 is user-visible — update to reflect current supported channels

### ADR-007 CTO Decisions (BINDING)

From [@cto] 2026-03-17:

| Decision | Implementation requirement |
|----------|---------------------------|
| MTS tenant only, never `common` | Factory MUST return error if `TenantID == ""` or `TenantID == "common"` |
| Respond in channel (same thread) | Send to `conversation.id` (not user DM) for all message types |
| App password for Sprint 10 | Use `client_credentials` flow, not Managed Identity |
| MSTEAMS_APP_PASSWORD masking | Add to all 3 functions in `config_secrets.go` (same CTO-27 pattern) |

### Pre-execution blocker check

Before writing code, verify @devops pre-work is done:

```bash
# These env vars must exist on the development server before T10-01 can run end-to-end:
echo $MSTEAMS_APP_ID        # must be non-empty
echo $MSTEAMS_APP_PASSWORD  # must be non-empty
echo $MSTEAMS_TENANT_ID     # must be non-empty, NOT "common"
```

If not provisioned: do T10-02 (tests) first — all tests can run with mock credentials.

---

## Sprint 10 Tasks — Implementation Guide

### Overview

| ID | Task | Priority | Points | Days |
|----|------|----------|--------|------|
| T10-01 | MS Teams core: config + auth + webhook + channel + gateway wiring + CTO-33 | P0 | 4 | 1-2 |
| T10-02 | MS Teams unit tests (~15 tests) | P0 | 2 | 3 |
| T10-03 | Adaptive Cards (spec + PR review output, Teams-native) | P1 | 2 | 3-4 |
| T10-04 | NQH corporate onboarding: README update + cross-channel governance verify | P0 | 2 | 4-5 |
| T10-05 | Roadmap update + WAU log ([@pm] task — @coder skip) | P1 | 1 | 5 |

**[@coder] scope**: T10-01 through T10-04 (10 points, 4 days).

---

### T10-01: MS Teams Core (P0, 4 pts) — Days 1-2

See sprint plan for full code. Summary of files to create/edit:

#### New files (in `extensions/msteams/`):

| File | Purpose |
|------|---------|
| `msteams.go` | Package entry: `Config` struct + `Factory` function (replaces `.go.TODO`) |
| `channel.go` | `MSTeamsChannel` struct: `Start`, `Stop`, `Send`, `RegisterRoutes`, `SetAgentID` |
| `auth.go` | Bot Framework token acquisition + `tokenCache` (5-min expiry buffer) |
| `webhook.go` | HTTP handler: parse `Activity`, route to `bus.PublishInbound` |
| `jwt.go` | Bot Framework JWT middleware: OpenID metadata → JWKS → `rsa.PublicKey` verification |

**Delete**: `extensions/msteams/msteams.go.TODO` (replaced by `msteams.go`)

#### Files to edit:

| File | Change |
|------|--------|
| `internal/config/config_channels.go` | Add `MSTeamsConfig` struct + `MSTeams MSTeamsConfig` field in `ChannelsConfig` |
| `internal/config/config_load.go` | Add `MSTEAMS_APP_ID`, `MSTEAMS_APP_PASSWORD`, `MSTEAMS_TENANT_ID` env var blocks |
| `internal/config/config_secrets.go` | Mask `MSTeams.AppSecret` in all 3 functions (CTO-33 + ADR-007 requirement) |
| `.env.example` | Add MS Teams section (3 vars + production note) |
| `cmd/gateway.go` | Add `msteams` import + `RegisterFactory("msteams", msteams.Factory)` + standalone init block |
| `cmd/gateway_consumer.go` | CTO-33: update Discord comments at :46 and :138 |
| `cmd/gateway_builtin_tools.go` | CTO-33: update Discord string at :68 |

#### Dependency note:

`jwt.go` imports `github.com/golang-jwt/jwt/v5`. This is a workspace extension package — add to `extensions/msteams/go.mod` (or add to root `go.mod` if no separate module). Check existing `go.mod` before deciding:

```bash
cat /home/nqh/shared/MTClaw/go.mod | grep -i jwt
# If already present: reuse it. If not: go get github.com/golang-jwt/jwt/v5
```

#### `InboundMessage` struct check:

`webhook.go` uses `bus.InboundMessage` with a `ServiceURL` field and `Metadata` map. Verify the existing struct supports these:

```bash
grep -n "ServiceURL\|Metadata" /home/nqh/shared/MTClaw/internal/bus/types.go
```

If not present, add them to `bus.InboundMessage` (they're needed for the MS Teams `Send` to know the Bot Framework endpoint).

#### Verification after T10-01:

```bash
cd /home/nqh/shared/MTClaw

# Build clean
/home/dttai/.local/go/bin/go build ./...
# Expected: 0 errors

# Config masks app secret
grep -A5 "MSTeams" internal/config/config_secrets.go
# Expected: AppSecret masked in all 3 functions

# CTO-33 clean
grep -n "Discord\|discord" cmd/gateway_consumer.go cmd/gateway_builtin_tools.go | grep -v "_test.go"
# Expected: 0 results

# TenantID production guard in factory
grep -A3 "TenantID" extensions/msteams/msteams.go
# Expected: error return for empty TenantID
```

---

### T10-02: MS Teams Unit Tests (P0, 2 pts) — Day 3

**File**: `extensions/msteams/msteams_test.go`

**~15 tests** split across:

```go
// Config validation (3 tests)
TestMSTeamsFactory_TenantIDRequired
TestMSTeamsFactory_AppIDAndSecretRequired
TestMSTeamsFactory_DefaultWebhookPath

// JWT middleware (3 tests)
TestJWTMiddleware_MissingAuthHeader_Returns401
TestJWTMiddleware_InvalidToken_Returns401
TestJWTMiddleware_ValidToken_CallsNext  // use a test RSA key pair

// Activity parsing (3 tests)
TestWebhookHandler_MessageActivity_PublishesToBus
TestWebhookHandler_EmptyText_Skipped
TestWebhookHandler_ConversationUpdate_Acknowledged

// Send (3 tests)
TestMSTeamsChannel_Send_AcquiresTokenFirst
TestMSTeamsChannel_Send_CorrectEndpointURL  // serviceURL + conversationID
TestMSTeamsChannel_Send_HTTPErrorReturnsError

// CTO-33 regression (3 tests — compile-time content checks)
TestNoDiscordReferenceInChannelGo
TestNoDiscordReferenceInWebhookGo
TestNoDiscordReferenceInMSTeamsGo
```

> For JWT tests: generate an in-memory RSA key pair with `rsa.GenerateKey`. Sign a test JWT and pass it through the middleware. Bypass JWKS fetch by injecting a mock key resolver.

**Target test count after T10-02**:

```bash
/home/dttai/.local/go/bin/go test ./... -count=1 -v 2>&1 | grep -c "^--- PASS"
# Expected: ≥365 (350 Sprint 9 + ~15 msteams)
```

---

### T10-03: Adaptive Cards (P1, 2 pts) — Days 3-4

**File**: `extensions/msteams/cards.go`

Two card builders:
1. `SpecCard(specID, title, status string, scenarios []string) json.RawMessage`
2. `PRReviewCard(prURL, verdict string, blockRules, warnRules []string) json.RawMessage`

**Channel-aware send** in `channel.go` `Send()`:
- Check `msg.Format == "adaptive_card"` → wrap content as `application/vnd.microsoft.card.adaptive` attachment
- Otherwise: plain `{"type":"message","text":"..."}` payload

**How the governance processor sets format** (in `internal/governance/`):
```go
// In spec_processor.go or pr_gate_processor.go, after determining channel:
if channel == "msteams" {
    outbound.Format = "adaptive_card"
    outbound.Content = string(msteams.SpecCard(spec.ID, spec.Title, spec.Status, bddScenarioTitles))
}
```

This requires a small edit in the governance processor to detect channel context. Check `internal/governance/spec_processor.go` to see how `OutboundMessage` is constructed — add the format field there.

---

### T10-04: NQH Corporate Onboarding (P0, 2 pts) — Days 4-5

#### Subtask A — Update README.md (required by @cto decision)

Add to `extensions/msteams/README.md`:

```markdown
## Production Security Requirement — Tenant ID

MSTEAMS_TENANT_ID **MUST** be set to your organization's specific Azure tenant ID.

**NEVER use `common`** — this would allow any Microsoft 365 user worldwide to reach your bot.

To find your tenant ID:
1. Azure Portal → Azure Active Directory → Overview → Tenant ID
2. Or: `az account show --query tenantId -o tsv`

Example:
- MTS deployment: MSTEAMS_TENANT_ID=<mts-azure-tenant-id>
- NQH deployment: MSTEAMS_TENANT_ID=<nqh-azure-tenant-id>

Reference: ADR-007 Section "CTO Decisions" — MTS tenant only for Phase 1.
```

#### Subtask B — Cross-channel governance SQL check

After NQH management team starts using Teams, run verification queries:

```sql
-- Verify msteams traces exist
SELECT channel, COUNT(*) as count
FROM traces
WHERE created_at > now() - interval '7 days'
GROUP BY channel;
-- Expected: telegram N, msteams ≥1

-- Verify spec factory works via Teams
SELECT channel, spec_id, created_at
FROM governance_specs
WHERE channel = 'msteams'
ORDER BY created_at DESC LIMIT 5;

-- Verify PR Gate evaluations via Teams
SELECT channel, verdict, COUNT(*)
FROM pr_gate_evaluations
GROUP BY channel, verdict;
```

These SQL queries should return results after Day 4 rollout. If `channel` column doesn't exist on these tables, add it via a migration (000016):

```sql
-- Only if column doesn't exist:
ALTER TABLE governance_specs ADD COLUMN IF NOT EXISTS channel VARCHAR(32);
ALTER TABLE pr_gate_evaluations ADD COLUMN IF NOT EXISTS channel VARCHAR(32);
```

Check existing schema first:
```bash
grep -n "channel" /home/nqh/shared/MTClaw/migrations/000013_governance_specs.sql
grep -n "channel" /home/nqh/shared/MTClaw/migrations/000015_pr_gate_evaluations.sql
```

---

## Definition of Done

| Check | Command / Verification | Expected |
|-------|------------------------|----------|
| Build clean | `go build ./...` | 0 errors |
| All tests pass | `go test ./... -count=1` | ≥365 PASS |
| MS Teams tests | `go test ./extensions/msteams/... -v` | ~15 PASS |
| CTO-33 clean | `grep -n "Discord" cmd/gateway_consumer.go cmd/gateway_builtin_tools.go` | 0 results |
| AppSecret masked | `grep -A2 "MSTeams" internal/config/config_secrets.go` | masked in 3 functions |
| TenantID guard | Factory with empty TenantID → error returned | confirmed |
| Tenant restriction doc | README.md production section | present |
| Channels list | `./mtclaw channels list` (with MSTEAMS_APP_ID set) | msteams listed |
| Cross-channel governance | SQL queries in T10-04B | msteams rows present |
| `.go.TODO` removed | `ls extensions/msteams/` | no .go.TODO file |

---

## File Reference

| File | Path |
|------|------|
| ADR-007 (APPROVED) | `docs/02-design/01-ADRs/SPEC-0007-ADR-007-MSTeams-Extension.md` |
| Sprint 10 plan | `docs/04-build/sprints/SPRINT-010-MSTeams-NQH-Corporate.md` |
| MS Teams scaffold | `extensions/msteams/README.md` + `extensions/msteams/msteams.go.TODO` |
| Channel interface | `internal/channels/channel.go` |
| Config channels | `internal/config/config_channels.go` |
| Config secrets | `internal/config/config_secrets.go` |
| Bus types | `internal/bus/types.go` |
| Gateway entry | `cmd/gateway.go` |
| Gateway consumer (CTO-33) | `cmd/gateway_consumer.go` |
| Builtin tools (CTO-33) | `cmd/gateway_builtin_tools.go` |
| Governance spec processor | `internal/governance/spec_processor.go` |
| Governance PR gate processor | `internal/governance/pr_gate_processor.go` |
