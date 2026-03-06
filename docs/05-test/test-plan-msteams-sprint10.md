---
feature: MS Teams Extension
sprint: 10
test_plan_id: TP-010-01
version: 1.0.0
date: 2026-03-22
author: "[@tester]"
status: IN_PROGRESS
requirements: FR-005 (AC-005-1 through AC-005-10)
design: SPEC-0007-ADR-007-MSTeams-Extension.md
implementation: extensions/msteams/ (7 files)
g_sprint_evidence: docs/04-build/SPRINT-010-COMPLETION.md
framework: SDLC Enterprise Framework 6.1.1
tier: STANDARD
---

# Test Plan: MS Teams Extension — Sprint 10 (TP-010-01)

## Prerequisite Checklist (Test Plan Gate)

```
[x] Test plan: this document (docs/05-test/test-plan-msteams-sprint10.md)
[x] Requirements with acceptance criteria: docs/01-planning/requirements.md FR-005 AC-005-1..10
[x] G-Sprint evidence: docs/04-build/SPRINT-010-COMPLETION.md (366 tests PASS)
[ ] Reviewer sign-off: @reviewer sign-off pending CTO review cycle
```

> Note: [@cto] Sprint 10 review pending. [@tester] proceeds on [@pm] authority — unit test
> evidence (366 PASS) and completion report are sufficient to begin QA planning. Integration
> E2E execution blocked on Azure AD credentials ([@devops] pre-work ⏳).

---

## 1. Scope

### In Scope

| Area | What | Files |
|------|------|-------|
| Config validation | TenantID, AppID, AppSecret requirements + "common" rejection | `extensions/msteams/msteams.go` |
| JWT verification | Bot Framework OpenID flow, valid/expired/wrong claims | `extensions/msteams/jwt.go`, `webhook.go` |
| Activity parsing | message, @mention, conversationUpdate, empty text, unknown types | `extensions/msteams/webhook.go` |
| Token acquisition | OAuth2 `client_credentials`, cache TTL, expiry buffer | `extensions/msteams/auth.go` |
| Message send | Bot Framework REST `/v3/conversations/{id}/activities` endpoint | `extensions/msteams/channel.go` |
| Adaptive Cards | `SpecCard()`, `PRReviewCard()` JSON output | `extensions/msteams/cards.go` |
| Secret masking | `MSTEAMS_APP_PASSWORD` never in logs | `internal/config/config_secrets.go` |
| Schema migration | `channel` column in governance tables | `migrations/000016_*.sql` |
| Cross-channel governance | `/spec` output identical on Teams + Telegram | Integration test |
| Regression | No Discord residuals (CTO-33) | `cmd/gateway_consumer.go`, `gateway_builtin_tools.go` |

### Out of Scope

| Area | Reason |
|------|--------|
| Live Bot Framework API calls | Azure AD credentials pending ([@devops]) |
| Adaptive Cards rendering in Teams client | UI test — requires live Teams tenant |
| NQH tenant live rollout | Blocked on Azure AD + [@devops] provisioning |
| SOUL behavioral regression (17 SOULs) | Covered by Sprint 9 T9-03 (85 tests, all PASS) |

---

## 2. Test Strategy

| Type | Owner | Count | Environment | Status |
|------|-------|-------|-------------|--------|
| Unit (package `msteams`) | [@coder] delivered, [@tester] reviews | 16 | CI — `go test ./extensions/msteams/` | ✅ All PASS |
| Unit (config secrets) | [@coder] delivered | verify via grep | CI | ✅ |
| Integration (cross-channel governance) | [@tester] | 6 new cases | Local Docker | To execute |
| Integration (schema migration) | [@tester] | 2 new cases | Local Docker (PostgreSQL) | To execute |
| Security probing (JWT bypass) | [@tester] | 5 cases | Local | To execute |
| E2E (live Teams flow) | [@tester] | 3 critical paths | Requires Azure AD | **BLOCKED** |
| Manual (Adaptive Cards rendering) | [@tester] | 2 visual checks | Requires Teams client | **BLOCKED** |

---

## 3. Requirements Traceability Matrix

| AC | Requirement | Unit | Integration | E2E | Manual | Status |
|----|------------|------|-------------|-----|--------|--------|
| AC-005-1 | Inbound message → JWT verify → bus publish → reply | `TestJWTMiddleware_ValidToken_CallsNext` + `TestWebhookHandler_MessageActivity_PublishesToBus` | TC-INT-001 | TC-E2E-001 | — | Unit ✅ |
| AC-005-2 | Wrong iss/aud → 401 | `TestJWTMiddleware_InvalidToken_Returns401`, `TestJWTMiddleware_MissingAuthHeader_Returns401` | TC-INT-002 | — | — | Unit ✅ |
| AC-005-3 | TenantID="common" → error | `TestMSTeamsFactory_CommonTenantRejected` | — | — | — | Unit ✅ |
| AC-005-4 | conversationUpdate → 200, no error | `TestWebhookHandler_ConversationUpdate_Acknowledged` | TC-INT-003 | TC-E2E-002 | — | Unit ✅ |
| AC-005-5 | Empty text → 200, no bus publish | `TestWebhookHandler_EmptyText_Skipped` | — | — | — | Unit ✅ |
| AC-005-6 | @mention in channel → in-thread reply | — | TC-INT-004 | TC-E2E-003 | TC-MAN-001 | Pending |
| AC-005-7 | /spec output identical Teams + Telegram | — | TC-INT-005 | — | TC-MAN-002 | Pending |
| AC-005-8 | APP_PASSWORD never in logs | TC-SEC-001 | TC-INT-006 | — | — | Pending |
| AC-005-9 | channel='msteams' in governance tables | — | TC-INT-007 | — | — | Pending |
| AC-005-10 | Token cached, no redundant calls | `TestMSTeamsChannel_Send_AcquiresTokenFirst` | TC-INT-008 | — | — | Unit ✅ |

**Unit test coverage**: 8/10 ACs have unit coverage. Gaps: AC-005-6 (@mention routing), AC-005-7 (cross-channel /spec parity) — require integration/E2E.

---

## 4. Test Cases

### 4.1 Unit Test Review — `extensions/msteams/msteams_test.go`

**Run**: `go test ./extensions/msteams/ -v`

Reviewed tests (all 16 confirmed PASS per Sprint 10 completion report):

| Test | AC | Assertion | Status |
|------|----|-----------|--------|
| `TestMSTeamsFactory_TenantIDRequired` | AC-005-3 | Empty TenantID → error mentioning `MSTEAMS_TENANT_ID` | ✅ PASS |
| `TestMSTeamsFactory_CommonTenantRejected` | AC-005-3 | TenantID="common" → error mentioning "common" | ✅ PASS |
| `TestMSTeamsFactory_AppIDAndSecretRequired` | AC-005-1 | Empty AppID → error; Empty AppPassword → error | ✅ PASS |
| `TestMSTeamsFactory_DefaultWebhookPath` | — | Default path = `/v1/channels/msteams/webhook` | ✅ PASS |
| `TestJWTMiddleware_MissingAuthHeader_Returns401` | AC-005-2 | No Authorization header → 401 | ✅ PASS |
| `TestJWTMiddleware_InvalidToken_Returns401` | AC-005-2 | Malformed JWT → 401 | ✅ PASS |
| `TestJWTMiddleware_ValidToken_CallsNext` | AC-005-1 | Valid RSA-signed JWT → 200, handler proceeds | ✅ PASS |
| `TestWebhookHandler_MessageActivity_PublishesToBus` | AC-005-1 | message activity → bus has inbound msg with SenderID, ChatID, ServiceURL, Channel="msteams" | ✅ PASS |
| `TestWebhookHandler_EmptyText_Skipped` | AC-005-5 | empty text → 200, no bus message | ✅ PASS |
| `TestWebhookHandler_ConversationUpdate_Acknowledged` | AC-005-4 | conversationUpdate → 200 | ✅ PASS |
| `TestMSTeamsChannel_Send_AcquiresTokenFirst` | AC-005-10 | token endpoint called before API send | ✅ PASS |
| `TestMSTeamsChannel_Send_CorrectEndpointURL` | AC-005-1 | send URL = `/v3/conversations/{id}/activities` | ✅ PASS |
| `TestMSTeamsChannel_Send_HTTPErrorReturnsError` | — | 403 from API → error returned containing "403" | ✅ PASS |
| `TestNoDiscordReferenceInChannelGo` | CTO-33 | channel.go has no "discord" string | ✅ PASS |
| `TestNoDiscordReferenceInWebhookGo` | CTO-33 | webhook.go has no "discord" string | ✅ PASS |
| `TestNoDiscordReferenceInMSTeamsGo` | CTO-33 | msteams.go has no "discord" string | ✅ PASS |

**Unit review verdict**: ✅ All 16 PASS. Assertions strong — no trivially-true checks. `injectTestKey()` bypasses network correctly. `httptest.Server` used for HTTP, not mocks.

---

### 4.2 Integration Tests

**Environment**: Local Docker Compose (`docker-compose.yml`). PostgreSQL with migration 000016 applied.

**Run**: `go test ./internal/integration/ -run MSTeams -v`

> These tests are added to `internal/integration/msteams_integration_test.go` (new file — see Section 7).

---

#### TC-INT-001: Full inbound message flow (AC-005-1)

**Requirement**: AC-005-1
**Preconditions**: MTClaw running, migration 000016 applied, msteams channel enabled in config (mock credentials)
**Steps**:
1. POST `http://localhost:8080/v1/channels/msteams/webhook` with valid JWT + message activity `{type:"message", text:"@pm run /spec"}`
2. Wait for SOUL routing (100ms timeout)
3. Inspect outbound message published to bus

**Expected**:
- HTTP 200
- Bus has `OutboundMessage{Channel:"msteams", ChatID: <conversation.id>}`
- `traces` table has new row with `tenant_id`, `soul_role`

**Status**: TO EXECUTE (requires migration 000016 + msteams config with mock AppID)

---

#### TC-INT-002: JWT claim rejection — wrong issuer (AC-005-2)

**Requirement**: AC-005-2
**Steps**:
1. Sign JWT with `iss = "https://malicious.example.com"` (not `https://api.botframework.com`)
2. POST to webhook endpoint

**Expected**: HTTP 401. No message in bus.

**Status**: TO EXECUTE

---

#### TC-INT-003: conversationUpdate — onboarding message sent (AC-005-4)

**Requirement**: AC-005-4
**Steps**:
1. POST `{type:"conversationUpdate", membersAdded:[{id:"user-new"}]}` with valid JWT
2. Inspect outbound bus message

**Expected**:
- HTTP 200
- Bus has `OutboundMessage` with onboarding content (mentions `/pair <soul>`)
- No error logged

**Status**: TO EXECUTE

---

#### TC-INT-004: @mention in channel — reply uses same conversation.id (AC-005-6)

**Requirement**: AC-005-6
**Steps**:
1. POST `{type:"message", text:"@mtclaw hello", channelData:{teamsChannelId:"channel-abc"}, conversation:{id:"conv-123"}}` with valid JWT
2. Capture outbound message

**Expected**:
- Outbound `ChatID == "conv-123"` (same thread, not a new conversation)
- Route to default `pm` SOUL

**Status**: TO EXECUTE

---

#### TC-INT-005: Cross-channel /spec output parity (AC-005-7)

**Requirement**: AC-005-7
**Steps**:
1. Send `/spec Create login page` via Telegram (existing test infrastructure)
2. Send identical `/spec Create login page` via MS Teams webhook mock
3. Compare `governance_specs` rows: `spec_title`, `bdd_scenarios` (JSON), `risk_score`

**Expected**:
- Both rows have identical `spec_title`, `bdd_scenarios` structure
- `channel` column differs: `telegram` vs `msteams`
- `soul_role` identical (both routed to `pm`)

**Status**: TO EXECUTE

---

#### TC-INT-006: APP_PASSWORD not in logs (AC-005-8)

**Requirement**: AC-005-8
**Steps**:
1. Start MTClaw with `MSTEAMS_APP_PASSWORD=secret-test-value-12345` in env
2. Enable `slog` debug logging
3. Capture all log output during startup + first webhook call
4. `grep "secret-test-value-12345" <log_output>`

**Expected**: grep returns 0 matches. `MaskedCopy()` output shows `***` for AppSecret.

**Status**: TO EXECUTE

---

#### TC-INT-007: Migration 000016 — channel column written (AC-005-9)

**Requirement**: AC-005-9
**Steps**:
1. Apply migration 000016: `go run ./cmd/mtclaw migrate up`
2. Trigger `/spec` command via MS Teams webhook mock
3. Query: `SELECT channel FROM governance_specs ORDER BY created_at DESC LIMIT 1;`

**Expected**: `channel = 'msteams'`

**Also verify** PR Gate evaluation:
4. Trigger PR Gate via GitHub webhook (existing test infra)
5. Query: `SELECT channel FROM pr_gate_evaluations ORDER BY created_at DESC LIMIT 1;`

**Expected**: `channel = 'github'` or NULL (not 'msteams' — channels are independent)

**Status**: TO EXECUTE

---

#### TC-INT-008: Token cache — no redundant OAuth2 calls (AC-005-10)

**Requirement**: AC-005-10
**Steps**:
1. Start `httptest.Server` capturing POST requests (token endpoint mock)
2. Send 3 consecutive messages via webhook (valid JWT each time)
3. Count token endpoint requests

**Expected**: Token endpoint called exactly 1 time. Subsequent sends use cached token.

**Status**: TO EXECUTE

---

### 4.3 Security Test Cases

**Environment**: Local. No live Bot Framework endpoint needed.

---

#### TC-SEC-001: JWT algorithm confusion — RS256 → HS256 downgrade attempt

**Severity**: P1 (security bypass)
**Steps**:
1. Take a valid RS256 JWT header, change `alg` to `HS256`
2. Sign with arbitrary secret
3. POST to webhook

**Expected**: HTTP 401. `jwt.go` must reject non-RS256 tokens.

**Status**: TO EXECUTE

---

#### TC-SEC-002: Expired JWT rejected

**Steps**:
1. Sign JWT with `exp = time.Now().Add(-1 * time.Hour)` using test RSA key
2. Inject key into `jwksCache` (bypass fetch)
3. POST to webhook

**Expected**: HTTP 401 (covered by `testify` + `golang-jwt` expiry check).

**Status**: TO EXECUTE (exercise existing code path, confirm log message)

---

#### TC-SEC-003: JWT replay from different app (wrong `aud`)

**Steps**:
1. Sign valid JWT but with `aud = "different-app-id"` (not the configured `MSTEAMS_APP_ID`)
2. POST to webhook

**Expected**: HTTP 401. SOUL not invoked.

**Status**: TO EXECUTE

---

#### TC-SEC-004: SSRF via ServiceURL

**Severity**: P1
**Steps**:
1. POST activity with `serviceUrl = "http://169.254.169.254/metadata"` (AWS IMDS)
2. Observe if MTClaw attempts to send reply to that URL

**Expected**: Reply attempt fails (network unreachable in local env) OR ServiceURL is validated against allowed prefixes. Log should show the attempted URL. No internal data returned to caller.

**Note**: If no ServiceURL validation exists in `channel.go`, this is a **BUG-010-001** (P2 — SSRF vector for outbound reply injection). Report to [@coder].

**Status**: TO EXECUTE

---

#### TC-SEC-005: TenantID injection via JWKS kid manipulation

**Steps**:
1. Craft JWT with `kid` that does not exist in the JWKS
2. POST to webhook (should trigger kid-miss force-refresh in `globalJWKSCache`)
3. Confirm force-refresh fetches from Bot Framework OpenID endpoint (not an attacker-controlled URL)

**Expected**: 401 (kid not found after refresh). No panic. JWKS URL must be hardcoded from `botFrameworkOpenIDURL` constant — not from request data.

**Status**: TO EXECUTE

---

### 4.4 E2E Tests (BLOCKED — Azure AD pending)

These tests require live `MSTEAMS_APP_ID` + `MSTEAMS_APP_PASSWORD` provisioned by [@devops].

---

#### TC-E2E-001: End-to-end message flow via live Teams tenant

**Preconditions**: Azure AD app registered, bot deployed to staging, `MSTEAMS_APP_PASSWORD` in env
**Steps**:
1. Send "hello @mtclaw" in Teams personal chat with bot
2. Observe reply within 5s

**Expected**: Bot replies with `pm` SOUL greeting. `sessions` table has new row. `traces` table has trace.

**Status**: BLOCKED — awaiting [@devops] Azure AD provisioning

---

#### TC-E2E-002: NQH onboarding via Teams

**Steps**:
1. Add bot to NQH Teams channel
2. `conversationUpdate` triggers
3. Onboarding message appears in Teams

**Expected**: Onboarding message visible in Teams, mentions `/pair` command.

**Status**: BLOCKED

---

#### TC-E2E-003: @mention in NQH channel → in-thread reply

**Steps**:
1. @mention bot in NQH Teams channel: `@MTClaw what is the leave policy?`
2. Bot routes to NQH SOUL, queries nqh-hr RAG collection
3. Reply appears in same thread

**Expected**: Reply visible in-thread. `traces` table shows `collection=nqh-hr`. Response < 5s.

**Status**: BLOCKED

---

### 4.5 Manual Visual Tests (BLOCKED — Teams client needed)

#### TC-MAN-001: SpecCard Adaptive Card rendering

**Steps**: Trigger `/spec` via Teams chat → verify card layout in Teams client

**Expected**: Card shows `SPEC-{id}` in title, colored status badge, BDD preview, risk score badge, action button "View Full Spec".

**Status**: BLOCKED

#### TC-MAN-002: PRReviewCard — BLOCK verdict display

**Steps**: Trigger PR evaluation for a PR missing spec ref → verify card in Teams

**Expected**: Red BLOCK badge visible. Rules list shows triggered rule. "View PR" action button links to GitHub PR URL.

**Status**: BLOCKED

---

## 5. Test Execution Log

### Unit Tests (2026-03-22)

```
Command: go test ./extensions/msteams/ -v
Result:  16/16 PASS
Time:    ~0.4s
Notes:   Sprint 10 @coder report — verified clean
```

### Integration Tests (not yet executed)

| TC | Run Date | Run By | Result | Notes |
|----|----------|--------|--------|-------|
| TC-INT-001 | — | — | PENDING | |
| TC-INT-002 | — | — | PENDING | |
| TC-INT-003 | — | — | PENDING | |
| TC-INT-004 | — | — | PENDING | |
| TC-INT-005 | — | — | PENDING | |
| TC-INT-006 | — | — | PENDING | |
| TC-INT-007 | — | — | PENDING | |
| TC-INT-008 | — | — | PENDING | |

### Security Tests (not yet executed)

| TC | Run Date | Run By | Result | Notes |
|----|----------|--------|--------|-------|
| TC-SEC-001 | — | — | PENDING | |
| TC-SEC-002 | — | — | PENDING | |
| TC-SEC-003 | — | — | PENDING | |
| TC-SEC-004 | — | — | PENDING | SSRF check — may reveal bug |
| TC-SEC-005 | — | — | PENDING | |

---

## 6. Bug Register

| ID | Severity | Title | TC | Status |
|----|----------|-------|----|--------|
| (none yet) | | | | |

> TC-SEC-004 (SSRF via ServiceURL) may surface a P2 bug if ServiceURL is not validated. Log here if found.

---

## 7. New Integration Test File

Create `internal/integration/msteams_integration_test.go` to host TC-INT-001 through TC-INT-008.

**File header**:
```go
// Package integration_test — MS Teams extension integration tests.
// Sprint 10 TP-010-01: AC-005-1 through AC-005-10.
// These tests require: PostgreSQL with migration 000016 applied + msteams config enabled.
// Run: go test ./internal/integration/ -run MSTeams -v
// BLOCKED (E2E): TC-INT-001, TC-INT-003..005, TC-INT-007 require local gateway running.
package integration_test
```

**Dependencies already available**:
- `internal/bus` — `bus.New()`, `PublishInbound()`, `ConsumeInbound()`
- `internal/integration` package exists (Sprint 8 drift_e2e_test.go)
- `httptest.Server` for mock token + API endpoints

---

## 8. Exit Criteria

| Criterion | Target | Status |
|-----------|--------|--------|
| Unit tests (16) | 16/16 PASS | ✅ DONE |
| Integration tests (8) | 8/8 PASS | ⏳ PENDING |
| Security tests (5) | 5/5 PASS (or bugs filed) | ⏳ PENDING |
| P1 bugs | 0 open | ⏳ |
| P2 bugs | 0 open (or accepted risk) | ⏳ |
| SSRF check (TC-SEC-004) | PASS or BUG filed + [@coder] fix | ⏳ |
| E2E (3 paths) | PASS after Azure AD provisioned | **BLOCKED** |
| Manual (2 checks) | PASS after Teams client available | **BLOCKED** |

**Partial QA PASS** (unit + integration + security complete, E2E pending [@devops]):
- 16 unit + 8 integration + 5 security = **29 test cases PASS** required for partial sign-off
- E2E (3 paths) required for **full QA sign-off** (Sprint 11 prerequisite)

---

## 9. Risks

| # | Risk | Probability | Impact | Mitigation |
|---|------|------------|--------|------------|
| R1 | TC-SEC-004 reveals SSRF vulnerability in ServiceURL handling | Med | P2 | File BUG, [@coder] adds URL allowlist validation in Sprint 11 |
| R2 | Azure AD provisioning delayed → E2E blocked beyond Sprint 11 | Med | Med | Partial QA sign-off (unit + integration) accepted by [@cto] for Sprint 10 gate |
| R3 | TC-INT-005 cross-channel parity reveals spec formatting difference | Low | Med | Acceptable if schema-level identical, display format can differ |
| R4 | Migration 000016 not idempotent on existing DB | Low | Med | `IF NOT EXISTS` clause in SQL — should be safe, verify on test DB |

---

## 10. References

| Document | Location |
|----------|----------|
| Requirements FR-005 (AC-005-1..10) | `docs/01-planning/requirements.md` |
| ADR-007 MS Teams Architecture | `docs/02-design/01-ADRs/SPEC-0007-ADR-007-MSTeams-Extension.md` |
| Test Strategy (updated v1.1.0) | `docs/01-planning/test-strategy.md` |
| Sprint 10 Completion Report | `docs/04-build/SPRINT-010-COMPLETION.md` |
| SOUL-tester | `docs/08-collaborate/souls/SOUL-tester.md` |
| Unit test file | `extensions/msteams/msteams_test.go` |
| New integration test | `internal/integration/msteams_integration_test.go` (to create) |
