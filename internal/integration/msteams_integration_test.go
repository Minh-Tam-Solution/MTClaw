// Package integration_test — MS Teams extension integration tests.
// Sprint 10, TP-010-01: AC-005-1 through AC-005-10.
//
// Scope of this file (cross-package integration, public API only):
//   - TC-INT-006: Config secrets masking — MSTeams.AppPassword never exposed
//   - TC-INT-007: Migration 000016 SQL declares channel column in both governance tables
//   - TC-INT-008: Token cache — Send() calls token endpoint exactly once for N sends
//   - TC-SEC-004: SSRF via ServiceURL — Send() with IMDS-like URL
//   - TC-CARDS-001: SpecCard() returns valid JSON Adaptive Card with required fields
//   - TC-CARDS-002: PRReviewCard() returns valid JSON Adaptive Card with required fields
//   - TC-REG-001: RegisterRoutes registers webhook path on mux (no panic, path present)
//
// JWT-dependent tests (TC-INT-001..005, TC-SEC-001..003,005) are tested at unit level
// in extensions/msteams/msteams_test.go (package msteams — access to unexported jwksCache
// and injectTestKey helper). Those tests are NOT duplicated here.
//
// E2E tests (TC-E2E-001..003) require live Azure AD credentials — BLOCKED pending [@devops].
package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Minh-Tam-Solution/MTClaw/extensions/msteams"
	"github.com/Minh-Tam-Solution/MTClaw/internal/bus"
	"github.com/Minh-Tam-Solution/MTClaw/internal/config"
)

// ─── TC-INT-006: AC-005-8 — AppPassword never exposed ────────────────────────

// TestMSTeams_MaskedCopy_AppPasswordMasked verifies that MaskedCopy() replaces
// MSTEAMS_APP_PASSWORD with "***" and never returns the raw credential.
func TestMSTeams_MaskedCopy_AppPasswordMasked(t *testing.T) {
	const rawPassword = "super-secret-bot-password-sprint10"

	cfg := config.Default()
	cfg.Channels.MSTeams.AppPassword = rawPassword

	masked := cfg.MaskedCopy()

	if masked.Channels.MSTeams.AppPassword == rawPassword {
		t.Errorf("MaskedCopy(): AppPassword must not be returned in plaintext, got %q", masked.Channels.MSTeams.AppPassword)
	}
	if masked.Channels.MSTeams.AppPassword != "***" {
		t.Errorf("MaskedCopy(): expected AppPassword='***', got %q", masked.Channels.MSTeams.AppPassword)
	}
	// Original must be untouched
	if cfg.Channels.MSTeams.AppPassword != rawPassword {
		t.Error("MaskedCopy(): must not modify the original config — original AppPassword changed")
	}
}

// TestMSTeams_StripSecrets_AppPasswordZeroed verifies that StripSecrets() zeroes
// MSTEAMS_APP_PASSWORD (used before writing config to disk).
func TestMSTeams_StripSecrets_AppPasswordZeroed(t *testing.T) {
	cfg := config.Default()
	cfg.Channels.MSTeams.AppPassword = "some-password"
	cfg.StripSecrets()

	if cfg.Channels.MSTeams.AppPassword != "" {
		t.Errorf("StripSecrets(): AppPassword must be empty string, got %q", cfg.Channels.MSTeams.AppPassword)
	}
}

// TestMSTeams_StripMaskedSecrets_AppPasswordCleaned verifies that StripMaskedSecrets()
// removes the mask value (used in standalone config persistence).
func TestMSTeams_StripMaskedSecrets_AppPasswordCleaned(t *testing.T) {
	cfg := config.Default()
	cfg.Channels.MSTeams.AppPassword = "***" // already masked

	cfg.StripMaskedSecrets()

	if cfg.Channels.MSTeams.AppPassword == "***" {
		t.Error("StripMaskedSecrets(): '***' mask should be stripped, but AppPassword still contains it")
	}
}

// ─── TC-INT-007: AC-005-9 — Migration 000016 SQL validity ────────────────────

// TestMSTeams_Migration000016_GovernanceSpecsColumnDeclared verifies that migration 000016
// adds the `channel` column to the `governance_specs` table.
func TestMSTeams_Migration000016_GovernanceSpecsColumnDeclared(t *testing.T) {
	upSQL := readMigrationFile(t, "../../migrations/000016_add_channel_to_governance_tables.up.sql")

	assertSQLContains(t, upSQL, "governance_specs",
		"migration 000016 must ALTER TABLE governance_specs")
	assertSQLContains(t, upSQL, "channel",
		"migration 000016 must add 'channel' column")
	assertSQLContains(t, strings.ToUpper(upSQL), "IF NOT EXISTS",
		"migration 000016 must use ADD COLUMN IF NOT EXISTS for idempotency")
}

// TestMSTeams_Migration000016_PRGateColumnDeclared verifies the channel column is also
// added to pr_gate_evaluations.
func TestMSTeams_Migration000016_PRGateColumnDeclared(t *testing.T) {
	upSQL := readMigrationFile(t, "../../migrations/000016_add_channel_to_governance_tables.up.sql")

	assertSQLContains(t, upSQL, "pr_gate_evaluations",
		"migration 000016 must ALTER TABLE pr_gate_evaluations")
}

// TestMSTeams_Migration000016_DownRevertsUp verifies the down migration drops the columns.
func TestMSTeams_Migration000016_DownRevertsUp(t *testing.T) {
	downSQL := readMigrationFile(t, "../../migrations/000016_add_channel_to_governance_tables.down.sql")

	assertSQLContains(t, strings.ToUpper(downSQL), "DROP COLUMN",
		"down migration must DROP COLUMN channel")
	assertSQLContains(t, downSQL, "governance_specs",
		"down migration must target governance_specs")
	assertSQLContains(t, downSQL, "pr_gate_evaluations",
		"down migration must target pr_gate_evaluations")
}

// ─── TC-CARDS-001: SpecCard JSON validity ─────────────────────────────────────

// TestMSTeams_SpecCard_ValidAdaptiveCard verifies that SpecCard() returns valid JSON
// with Adaptive Card schema fields and spec metadata.
func TestMSTeams_SpecCard_ValidAdaptiveCard(t *testing.T) {
	card := msteams.SpecCard(
		"SPEC-2026-0042",
		"User login page",
		"APPROVED",
		[]string{"Given a valid user, When login, Then redirect to dashboard"},
	)

	// Must be valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(card, &parsed); err != nil {
		t.Fatalf("SpecCard(): not valid JSON: %v", err)
	}

	// Adaptive Card schema fields
	if parsed["type"] != "AdaptiveCard" {
		t.Errorf("SpecCard(): expected type='AdaptiveCard', got %v", parsed["type"])
	}
	if parsed["$schema"] == nil {
		t.Error("SpecCard(): missing '$schema' field (required for Adaptive Card)")
	}

	// Spec ID must appear in card body
	raw := string(card)
	if !strings.Contains(raw, "SPEC-2026-0042") {
		t.Error("SpecCard(): spec ID not found in card JSON")
	}
	if !strings.Contains(raw, "User login page") {
		t.Error("SpecCard(): spec title not found in card JSON")
	}

	// Status color for APPROVED should be "good"
	if !strings.Contains(raw, "good") {
		t.Error("SpecCard(): APPROVED status should use 'good' color")
	}
}

// TestMSTeams_SpecCard_BlockedStatus_AttentionColor verifies that BLOCKED/FAIL status
// uses "attention" color (red in Teams).
func TestMSTeams_SpecCard_BlockedStatus_AttentionColor(t *testing.T) {
	card := msteams.SpecCard("SPEC-2026-0001", "test", "BLOCKED", nil)

	if !strings.Contains(string(card), "attention") {
		t.Error("SpecCard(): BLOCKED status should use 'attention' color")
	}
}

// ─── TC-CARDS-002: PRReviewCard JSON validity ─────────────────────────────────

// TestMSTeams_PRReviewCard_ValidAdaptiveCard verifies PRReviewCard() output.
func TestMSTeams_PRReviewCard_ValidAdaptiveCard(t *testing.T) {
	card := msteams.PRReviewCard(
		"https://github.com/org/repo/pull/42",
		"BLOCK",
		[]string{"missing spec reference", "no tests added"},
		[]string{},
	)

	var parsed map[string]interface{}
	if err := json.Unmarshal(card, &parsed); err != nil {
		t.Fatalf("PRReviewCard(): not valid JSON: %v", err)
	}

	if parsed["type"] != "AdaptiveCard" {
		t.Errorf("PRReviewCard(): expected type='AdaptiveCard', got %v", parsed["type"])
	}

	raw := string(card)
	if !strings.Contains(raw, "BLOCK") {
		t.Error("PRReviewCard(): BLOCK verdict not in card")
	}
	if !strings.Contains(raw, "https://github.com/org/repo/pull/42") {
		t.Error("PRReviewCard(): PR URL not in card — required for action button")
	}
}

// ─── TC-REG-001: Route registration ──────────────────────────────────────────

// TestMSTeams_RegisterRoutes_WebhookPathPresent verifies that RegisterRoutes() registers
// the webhook path on the mux without panic, and that the mux responds (even if 401 —
// meaning it reached the handler, not a 404).
func TestMSTeams_RegisterRoutes_WebhookPathPresent(t *testing.T) {
	ch, err := msteams.New(config.MSTeamsConfig{
		AppID:       "test-app-id",
		AppPassword: "test-app-password",
		TenantID:    "mts-tenant-id",
	}, bus.New())
	if err != nil {
		t.Fatalf("msteams.New: %v", err)
	}

	mux := http.NewServeMux()
	ch.RegisterRoutes(mux)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// POST to webhook without Authorization → expect 401 (handler reached, not 404)
	body := bytes.NewBufferString(`{"type":"message","text":"hello"}`)
	resp, err := http.Post(srv.URL+"/v1/channels/msteams/webhook", "application/json", body)
	if err != nil {
		t.Fatalf("POST webhook: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		t.Errorf("RegisterRoutes(): webhook path not registered — got 404 (expected 401 from JWT middleware)")
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("RegisterRoutes(): expected 401 from JWT middleware, got %d", resp.StatusCode)
	}
}

// ─── TC-INT-008: AC-005-10 — Token cache, no redundant calls ─────────────────

// TestMSTeams_Send_TokenCachedAcrossMultipleSends verifies that N consecutive Send()
// calls trigger exactly one token acquisition (cache works correctly).
func TestMSTeams_Send_TokenCachedAcrossMultipleSends(t *testing.T) {
	var tokenCallCount int

	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenCallCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "cached-test-token",
			"expires_in":   3600,
			"token_type":   "Bearer",
		})
	}))
	defer tokenSrv.Close()

	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer apiSrv.Close()

	ch, err := msteams.New(config.MSTeamsConfig{
		AppID:       "app-id",
		AppPassword: "app-secret",
		TenantID:    "mts-tenant",
	}, bus.New())
	if err != nil {
		t.Fatalf("msteams.New: %v", err)
	}

	// Point the token provider at the test server and allow HTTP for SSRF test bypass
	ch.SetTokenEndpoint(tokenSrv.URL)
	ch.SetHTTPClient(apiSrv.Client())
	ch.SetAllowedPrefixes([]string{apiSrv.URL})

	ctx := context.Background()
	msg := bus.OutboundMessage{
		Channel:    "msteams",
		ChatID:     "conv-1",
		Content:    "hello",
		ServiceURL: apiSrv.URL,
	}

	// Send 3 times
	for i := range 3 {
		if err := ch.Send(ctx, msg); err != nil {
			t.Fatalf("Send #%d: %v", i+1, err)
		}
	}

	if tokenCallCount != 1 {
		t.Errorf("token endpoint called %d times for 3 sends; expected exactly 1 (cache should reuse token)", tokenCallCount)
	}
}

// ─── TC-SEC-004: SSRF via ServiceURL ─────────────────────────────────────────

// TestMSTeams_Send_SSRFViaServiceURL probes whether Send() validates the ServiceURL
// before making an outbound HTTP request.
//
// Security concern: if ServiceURL is accepted without validation, an attacker who
// can inject a crafted Teams activity could redirect bot replies to internal services
// (AWS IMDS at 169.254.169.254, internal APIs, etc.).
//
// Expected: Send() should validate ServiceURL against an allowlist of Bot Framework
// service URL prefixes (e.g., *.trafficmanager.net, *.botframework.com).
//
// If this test finds that Send() does NOT validate — it is BUG-010-001 (P2 SSRF).
func TestMSTeams_Send_SSRFViaServiceURL(t *testing.T) {
	// Capture if our server receives a request
	var ssrfRequestReceived bool
	internalSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		ssrfRequestReceived = true
		w.WriteHeader(http.StatusOK)
	}))
	defer internalSrv.Close()

	ch, err := msteams.New(config.MSTeamsConfig{
		AppID:       "app-id",
		AppPassword: "app-secret",
		TenantID:    "mts-tenant",
	}, bus.New())
	if err != nil {
		t.Fatalf("msteams.New: %v", err)
	}

	// Pre-fill token cache so we don't trigger a real token fetch
	ch.SetTokenEndpoint(internalSrv.URL) // also SSRF target for token endpoint
	ch.SetHTTPClient(internalSrv.Client())

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err = ch.Send(ctx, bus.OutboundMessage{
		Channel:    "msteams",
		ChatID:     "conv-1",
		Content:    "test",
		ServiceURL: internalSrv.URL, // SSRF: arbitrary URL, not a Bot Framework service URL
	})

	if ssrfRequestReceived && err == nil {
		// BUG: ServiceURL was not validated — request went through
		t.Errorf("BUG-010-001 (P2 SSRF): Send() sent reply to arbitrary ServiceURL %q without validation. "+
			"ServiceURL must be validated against Bot Framework allowed prefixes (*.trafficmanager.net, *.botframework.com). "+
			"Report to [@coder] — fix required before G5 / external exposure.", internalSrv.URL)
	} else if ssrfRequestReceived && err != nil {
		// Request was made but returned error (e.g., 200 is not "Created") — still an SSRF
		t.Errorf("BUG-010-001 (P2 SSRF, partial): Send() attempted HTTP request to arbitrary ServiceURL %q. "+
			"Even a failed request leaks that this URL exists. ServiceURL validation required.", internalSrv.URL)
	}
	// Pass: err != nil AND ssrfRequestReceived == false → ServiceURL was rejected before request
	// This is the expected behavior IF validation is implemented.
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func readMigrationFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migration file %q: %v", path, err)
	}
	return string(data)
}

func assertSQLContains(t *testing.T, sql, substr, msg string) {
	t.Helper()
	if !strings.Contains(sql, substr) {
		t.Errorf("%s: substring %q not found in SQL:\n%s", msg, substr, sql)
	}
}
