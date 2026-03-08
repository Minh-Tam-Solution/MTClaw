// Package integration_test validates Sprint 8-12 Governance Engine end-to-end.
// Tests cover: evidence chain, spec quality gate, design-first gate, PR verdict,
// audit PDF, and channel cleanup — all without live DB (in-memory stores).
package integration_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Minh-Tam-Solution/MTClaw/internal/audit"
	"github.com/Minh-Tam-Solution/MTClaw/internal/evidence"
	"github.com/Minh-Tam-Solution/MTClaw/internal/governance"
	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// ============================================================
// E2E-005: PR Gate flow — webhook → verdict → evidence link
// ============================================================

func TestE2E_PRGateFlow_WebhookToEvidenceLink(t *testing.T) {
	ctx := context.Background()

	specID := uuid.New()
	prGateID := uuid.New()
	ownerID := "tenant-mts"
	sessionKey := "agent:reviewer+telegram+private+123"

	// Step 1: Create spec in mock store.
	specStore := &mockSpecStore{
		specs: map[string]*store.GovernanceSpec{
			"SPEC-2026-0001": {
				ID:      specID,
				OwnerID: ownerID,
				SpecID:  "SPEC-2026-0001",
				Status:  store.SpecStatusApproved,
				Title:   "User authentication feature",
				Narrative: toJSON(t, map[string]string{
					"as_a":    "a platform administrator with security responsibilities",
					"i_want":  "to enforce multi-factor authentication for all users",
					"so_that": "unauthorized access is prevented even with leaked passwords",
				}),
				CreatedAt: time.Now().Add(-1 * time.Hour),
			},
		},
	}

	// Step 2: Create PR Gate store.
	prGateStore := &mockPRGateStore{
		evals: map[uuid.UUID]*store.PRGateEvaluation{},
	}

	// Step 3: Simulate reviewer SOUL output with explicit pass verdict.
	reviewContent := `## PR Review: #42 — Add MFA support

- [x] Code follows naming conventions
- [x] Error handling is present
- [x] No hardcoded secrets
- [ ] Unit tests cover edge cases

**Verdict**: Pass

The implementation looks solid. MFA flow is correctly integrated.`

	// Step 4: Process PR review → persist evaluation.
	evalIDStr := governance.ProcessPRReview(
		ctx, prGateStore, reviewContent,
		"reviewer", ownerID,
		"https://github.com/org/repo/pull/42",
		"org/repo", "abc123def", "enforce", "telegram",
		42, nil,
	)
	if evalIDStr == "" {
		t.Fatal("ProcessPRReview returned empty eval ID")
	}

	// Verify verdict is "pass".
	evalID, err := uuid.Parse(evalIDStr)
	if err != nil {
		t.Fatalf("invalid eval ID: %v", err)
	}
	eval, err := prGateStore.GetEvaluation(ctx, evalID)
	if err != nil {
		t.Fatalf("GetEvaluation failed: %v", err)
	}
	if eval.Verdict != "pass" {
		t.Errorf("expected verdict 'pass', got %q", eval.Verdict)
	}
	if eval.Repo != "org/repo" {
		t.Errorf("expected repo 'org/repo', got %q", eval.Repo)
	}
	if eval.Channel != "telegram" {
		t.Errorf("expected channel 'telegram', got %q", eval.Channel)
	}

	// Verify rules extracted from checklist.
	var rules []string
	if err := json.Unmarshal(eval.RulesEvaluated, &rules); err != nil {
		t.Fatalf("failed to parse rules_evaluated: %v", err)
	}
	if len(rules) != 4 {
		t.Errorf("expected 4 rules, got %d: %v", len(rules), rules)
	}

	// Step 5: Evidence linking — auto-link spec to PR gate.
	evidenceStore := &mockEvidenceLinkStore{
		links:          []*store.EvidenceLink{},
		recentSpecBySession: map[string]*uuid.UUID{
			sessionKey: &specID,
		},
	}

	linker := evidence.NewLinker(evidenceStore)
	if err := linker.AutoLinkSpecToPR(ctx, ownerID, sessionKey, prGateID); err != nil {
		t.Fatalf("AutoLinkSpecToPR failed: %v", err)
	}

	if len(evidenceStore.links) != 1 {
		t.Fatalf("expected 1 evidence link, got %d", len(evidenceStore.links))
	}

	link := evidenceStore.links[0]
	if link.FromType != "spec" || link.FromID != specID {
		t.Errorf("link from: expected spec/%s, got %s/%s", specID, link.FromType, link.FromID)
	}
	if link.ToType != "pr_gate" || link.ToID != prGateID {
		t.Errorf("link to: expected pr_gate/%s, got %s/%s", prGateID, link.ToType, link.ToID)
	}
	if link.LinkReason != "auto_spec_review" {
		t.Errorf("expected link_reason 'auto_spec_review', got %q", link.LinkReason)
	}

	// Step 6: Build evidence chain.
	builder := evidence.NewChainBuilder(evidenceStore, specStore, prGateStore)
	chain, err := builder.BuildChain(ctx, specStore.specs["SPEC-2026-0001"])
	if err != nil {
		t.Fatalf("BuildChain failed: %v", err)
	}

	if len(chain.Chain) != 2 {
		t.Fatalf("expected 2 chain nodes (spec + pr_gate), got %d", len(chain.Chain))
	}
	if chain.Chain[0].Type != "spec" {
		t.Errorf("first node should be 'spec', got %q", chain.Chain[0].Type)
	}
	if chain.Chain[1].Type != "pr_gate" {
		t.Errorf("second node should be 'pr_gate', got %q", chain.Chain[1].Type)
	}
	if chain.ChainComplete {
		t.Error("chain should NOT be complete (missing test_run, deploy)")
	}
	if len(chain.Missing) != 2 {
		t.Errorf("expected 2 missing types, got %d: %v", len(chain.Missing), chain.Missing)
	}
}

// ============================================================
// E2E-006: Spec quality gate — create → score → accept/reject
// ============================================================

func TestE2E_SpecQualityGate_FullSpecAccepted(t *testing.T) {
	spec := &store.GovernanceSpec{
		ID:     uuid.New(),
		SpecID: "SPEC-2026-0002",
		Status: store.SpecStatusDraft,
		Narrative: toJSON(t, map[string]string{
			"as_a":    "a project manager overseeing multiple teams and deliverables",
			"i_want":  "to have real-time visibility into sprint progress and blockers",
			"so_that": "I can intervene early when delivery risk increases beyond threshold",
		}),
		AcceptanceCriteria: toJSON(t, []map[string]string{
			{"scenario": "PM opens dashboard", "expected_result": "Sprint progress shown with burndown chart"},
			{"scenario": "Blocker reported by dev", "expected_result": "PM receives alert within 5 minutes"},
		}),
		BDDScenarios: toJSON(t, []map[string]string{
			{"given": "a sprint with 10 stories", "when": "3 stories are blocked", "then": "dashboard shows 30% blocked indicator"},
		}),
		Risks: toJSON(t, []map[string]string{
			{"description": "Data staleness if webhook fails", "mitigation": "Polling fallback every 5 minutes"},
		}),
		TechnicalRequirements: toJSON(t, "WebSocket for real-time updates, PostgreSQL for persistence, Redis for caching sprint metrics"),
	}

	result := governance.EvaluateSpecQuality(spec)

	if !result.Pass {
		t.Errorf("full spec should pass quality gate, scored %d/100. Reasons: %v", result.Score, result.Reasons)
	}
	if result.Score != 100 {
		t.Errorf("full spec should score 100, got %d", result.Score)
	}
	if len(result.Reasons) != 0 {
		t.Errorf("full spec should have no reasons, got: %v", result.Reasons)
	}
}

func TestE2E_SpecQualityGate_IncompleteSpecRejected(t *testing.T) {
	spec := &store.GovernanceSpec{
		ID:     uuid.New(),
		SpecID: "SPEC-2026-0003",
		Status: store.SpecStatusDraft,
		// Only narrative with 1 field → 8 pts
		Narrative: toJSON(t, map[string]string{
			"as_a": "a developer who needs better tooling for daily work",
		}),
		// No acceptance criteria → 0 pts
		// No BDD → 0 pts
		// No risks → 0 pts
		// No tech requirements → 0 pts
	}

	result := governance.EvaluateSpecQuality(spec)

	if result.Pass {
		t.Errorf("incomplete spec should fail quality gate, scored %d/100", result.Score)
	}
	if result.Score >= governance.QualityThreshold {
		t.Errorf("score %d should be below threshold %d", result.Score, governance.QualityThreshold)
	}
	if len(result.Reasons) == 0 {
		t.Error("rejected spec should have failure reasons")
	}

	// Verify rejection message formatting.
	msg := governance.FormatRejectionMessage(result)
	if !strings.Contains(msg, "Quality Gate:") {
		t.Error("rejection message missing 'Quality Gate:' prefix")
	}
	if !strings.Contains(msg, "Acceptance criteria") {
		t.Error("rejection message should mention missing acceptance criteria")
	}
}

func TestE2E_SpecQualityGate_BoundaryScore70Passes(t *testing.T) {
	// Construct spec that scores exactly 70: narrative(25) + AC(25) + BDD(20) = 70
	spec := &store.GovernanceSpec{
		ID:     uuid.New(),
		SpecID: "SPEC-2026-0004",
		Narrative: toJSON(t, map[string]string{
			"as_a":    "a system administrator managing cloud infrastructure",
			"i_want":  "automated alerts when resource usage exceeds thresholds",
			"so_that": "I can prevent outages before they impact end users",
		}),
		AcceptanceCriteria: toJSON(t, []map[string]string{
			{"scenario": "CPU exceeds 90%", "expected_result": "Alert sent within 60 seconds"},
			{"scenario": "Memory exceeds 85%", "expected_result": "Alert with memory breakdown"},
		}),
		BDDScenarios: toJSON(t, []map[string]string{
			{"given": "CPU at 91%", "when": "monitor checks", "then": "PagerDuty alert fires"},
		}),
		// No risks (0) + no tech requirements (0) = total 70
	}

	result := governance.EvaluateSpecQuality(spec)
	if result.Score != 70 {
		t.Errorf("expected score 70, got %d (reasons: %v)", result.Score, result.Reasons)
	}
	if !result.Pass {
		t.Errorf("score 70 should pass (threshold = %d)", governance.QualityThreshold)
	}
}

func TestE2E_SpecQualityGate_BoundaryScore69Fails(t *testing.T) {
	// 25 (narrative) + 25 (AC) + 10 (BDD incomplete) + 0 + 0 = 60 → fail
	// Or: 25 + 15 (1 AC) + 20 + 0 + 0 = 60 → fail
	// Try: 15 (2/3 narrative) + 25 (AC) + 20 (BDD) + 0 + 0 = 60
	spec := &store.GovernanceSpec{
		ID:     uuid.New(),
		SpecID: "SPEC-2026-0005",
		Narrative: toJSON(t, map[string]string{
			"as_a":    "a system administrator managing cloud infrastructure",
			"i_want":  "automated alerts when resource usage exceeds thresholds",
			// Missing so_that → 2/3 = 15 pts
		}),
		AcceptanceCriteria: toJSON(t, []map[string]string{
			{"scenario": "CPU exceeds 90%", "expected_result": "Alert sent within 60 seconds"},
			{"scenario": "Memory exceeds 85%", "expected_result": "Alert with memory breakdown"},
		}),
		BDDScenarios: toJSON(t, []map[string]string{
			{"given": "CPU at 91%", "when": "monitor checks", "then": "PagerDuty alert fires"},
		}),
		// 15 + 25 + 20 + 0 + 0 = 60
	}

	result := governance.EvaluateSpecQuality(spec)
	if result.Score >= governance.QualityThreshold {
		t.Errorf("score %d should be below threshold %d", result.Score, governance.QualityThreshold)
	}
	if result.Pass {
		t.Error("score below threshold should not pass")
	}
}

// ============================================================
// E2E-007: Design-first gate — @coder blocked without spec
// ============================================================

func TestE2E_DesignGate_CoderBlockedWithoutSpec(t *testing.T) {
	ctx := context.Background()
	emptySpecStore := &mockSpecStore{specs: map[string]*store.GovernanceSpec{}}

	pass, reason := governance.DesignFirstGate(ctx, "coder", "implement user authentication", emptySpecStore)
	if pass {
		t.Error("coder should be blocked without approved spec")
	}
	if !strings.Contains(reason, "Design-First Gate") {
		t.Errorf("reason should mention Design-First Gate, got: %q", reason)
	}
	if !strings.Contains(reason, "@pm /spec") {
		t.Errorf("reason should suggest using @pm /spec, got: %q", reason)
	}
}

func TestE2E_DesignGate_CoderAllowedWithApprovedSpec(t *testing.T) {
	ctx := context.Background()
	specStore := &mockSpecStore{
		specs: map[string]*store.GovernanceSpec{
			"SPEC-2026-0001": {
				ID:     uuid.New(),
				Status: store.SpecStatusApproved,
			},
		},
	}

	pass, reason := governance.DesignFirstGate(ctx, "coder", "implement user authentication", specStore)
	if !pass {
		t.Errorf("coder should pass with approved spec, reason: %q", reason)
	}
}

func TestE2E_DesignGate_PMBypassesGate(t *testing.T) {
	ctx := context.Background()
	emptySpecStore := &mockSpecStore{specs: map[string]*store.GovernanceSpec{}}

	for _, agent := range []string{"pm", "reviewer", "architect", "ceo", "devops"} {
		pass, _ := governance.DesignFirstGate(ctx, agent, "implement user authentication", emptySpecStore)
		if !pass {
			t.Errorf("agent %q should bypass design gate", agent)
		}
	}
}

func TestE2E_DesignGate_CoderAdHocQuestionAllowed(t *testing.T) {
	ctx := context.Background()
	emptySpecStore := &mockSpecStore{specs: map[string]*store.GovernanceSpec{}}

	questions := []string{
		"how do I fix this error?",
		"explain the authentication flow",
		"debug this null pointer exception",
		"what is the difference between channels?",
		"why is this test failing?",
		"can you review this approach?",
		"is this pattern correct?",
	}

	for _, q := range questions {
		pass, reason := governance.DesignFirstGate(ctx, "coder", q, emptySpecStore)
		if !pass {
			t.Errorf("ad-hoc question %q should pass design gate, reason: %q", q, reason)
		}
	}
}

// ============================================================
// E2E-008: Audit trail — spec + chain → PDF export
// ============================================================

func TestE2E_AuditTrail_SpecChainToPDF(t *testing.T) {
	spec := &store.GovernanceSpec{
		ID:          uuid.New(),
		SpecID:      "SPEC-2026-0010",
		Title:       "Audit Trail Export",
		Status:      store.SpecStatusApproved,
		ContentHash: "abc123def456",
		CreatedAt:   time.Now().Add(-24 * time.Hour),
	}

	chain := []store.ChainNode{
		{Type: "spec", ID: spec.ID, CreatedAt: spec.CreatedAt, Status: "approved"},
		{Type: "pr_gate", ID: uuid.New(), CreatedAt: time.Now().Add(-12 * time.Hour), Verdict: "pass", PRURL: "https://github.com/org/repo/pull/99"},
		{Type: "test_run", ID: uuid.New(), CreatedAt: time.Now().Add(-6 * time.Hour)},
	}

	pdf, err := audit.AuditTrailPDF(spec, chain)
	if err != nil {
		t.Fatalf("AuditTrailPDF failed: %v", err)
	}
	if len(pdf) == 0 {
		t.Fatal("PDF output is empty")
	}
	if string(pdf[:4]) != "%PDF" {
		t.Errorf("output should start with %%PDF magic bytes, got %q", string(pdf[:4]))
	}
	// Reasonable size check: PDF with 3 nodes should be > 1KB.
	if len(pdf) < 1024 {
		t.Errorf("PDF suspiciously small: %d bytes", len(pdf))
	}
}

func TestE2E_AuditTrail_NilSpecReturnsError(t *testing.T) {
	_, err := audit.AuditTrailPDF(nil, []store.ChainNode{{Type: "spec"}})
	if err == nil {
		t.Error("expected error for nil spec")
	}
}

func TestE2E_AuditTrail_EmptyChainReturnsError(t *testing.T) {
	spec := &store.GovernanceSpec{ID: uuid.New(), SpecID: "SPEC-2026-0099"}
	_, err := audit.AuditTrailPDF(spec, nil)
	if err == nil {
		t.Error("expected error for nil chain")
	}
}

// ============================================================
// E2E-009: Channel cleanup — Discord/Feishu/WhatsApp removed
// ============================================================

func TestE2E_ChannelCleanup_RemovedChannelsDontExist(t *testing.T) {
	removedDirs := []string{
		"../../internal/channels/discord",
		"../../internal/channels/feishu",
		"../../internal/channels/whatsapp",
	}

	for _, dir := range removedDirs {
		absDir, _ := filepath.Abs(dir)
		if _, err := os.Stat(absDir); !os.IsNotExist(err) {
			t.Errorf("removed channel directory still exists: %s", absDir)
		}
	}
}

func TestE2E_ChannelCleanup_TelegramStillExists(t *testing.T) {
	telegramDir, _ := filepath.Abs("../../internal/channels/telegram")
	info, err := os.Stat(telegramDir)
	if err != nil {
		t.Fatalf("telegram channel directory missing: %v", err)
	}
	if !info.IsDir() {
		t.Error("telegram should be a directory")
	}
}

// ============================================================
// E2E: Evidence chain completeness
// ============================================================

func TestE2E_EvidenceChain_CompleteChain(t *testing.T) {
	ctx := context.Background()

	specID := uuid.New()
	prGateID := uuid.New()
	testRunID := uuid.New()
	deployID := uuid.New()

	evidenceStore := &mockEvidenceLinkStore{
		links: []*store.EvidenceLink{
			{FromType: "spec", FromID: specID, ToType: "pr_gate", ToID: prGateID, CreatedAt: time.Now()},
			{FromType: "spec", FromID: specID, ToType: "test_run", ToID: testRunID, CreatedAt: time.Now()},
			{FromType: "spec", FromID: specID, ToType: "deploy", ToID: deployID, CreatedAt: time.Now()},
		},
	}

	prGateStore := &mockPRGateStore{
		evals: map[uuid.UUID]*store.PRGateEvaluation{
			prGateID: {
				ID:      prGateID,
				Verdict: "pass",
				PRURL:   "https://github.com/org/repo/pull/1",
			},
		},
	}

	spec := &store.GovernanceSpec{
		ID:        specID,
		SpecID:    "SPEC-2026-0100",
		Status:    store.SpecStatusApproved,
		CreatedAt: time.Now(),
	}

	builder := evidence.NewChainBuilder(evidenceStore, nil, prGateStore)
	chain, err := builder.BuildChain(ctx, spec)
	if err != nil {
		t.Fatalf("BuildChain failed: %v", err)
	}

	if !chain.ChainComplete {
		t.Errorf("chain should be complete, missing: %v", chain.Missing)
	}
	if len(chain.Chain) != 4 {
		t.Errorf("expected 4 nodes (spec + pr_gate + test_run + deploy), got %d", len(chain.Chain))
	}

	// Verify pr_gate node is enriched.
	for _, node := range chain.Chain {
		if node.Type == "pr_gate" {
			if node.Verdict != "pass" {
				t.Errorf("pr_gate node verdict should be 'pass', got %q", node.Verdict)
			}
			if node.PRURL != "https://github.com/org/repo/pull/1" {
				t.Errorf("pr_gate node PRURL wrong: %q", node.PRURL)
			}
		}
	}
}

func TestE2E_EvidenceChain_MissingArtifacts(t *testing.T) {
	ctx := context.Background()

	specID := uuid.New()

	// Only spec, no links → missing pr_gate, test_run, deploy.
	evidenceStore := &mockEvidenceLinkStore{links: []*store.EvidenceLink{}}

	spec := &store.GovernanceSpec{
		ID:        specID,
		SpecID:    "SPEC-2026-0101",
		Status:    store.SpecStatusDraft,
		CreatedAt: time.Now(),
	}

	builder := evidence.NewChainBuilder(evidenceStore, nil, nil)
	chain, err := builder.BuildChain(ctx, spec)
	if err != nil {
		t.Fatalf("BuildChain failed: %v", err)
	}

	if chain.ChainComplete {
		t.Error("chain should NOT be complete (spec only)")
	}
	if len(chain.Missing) != 3 {
		t.Errorf("expected 3 missing types (pr_gate, test_run, deploy), got %d: %v", len(chain.Missing), chain.Missing)
	}
	if len(chain.Chain) != 1 {
		t.Errorf("expected 1 chain node (spec only), got %d", len(chain.Chain))
	}
}

// ============================================================
// E2E: Evidence linker edge cases
// ============================================================

func TestE2E_EvidenceLinker_NoRecentSpec(t *testing.T) {
	ctx := context.Background()

	evidenceStore := &mockEvidenceLinkStore{
		links:               []*store.EvidenceLink{},
		recentSpecBySession: map[string]*uuid.UUID{}, // no recent spec
	}

	linker := evidence.NewLinker(evidenceStore)
	err := linker.AutoLinkSpecToPR(ctx, "tenant-mts", "some-session-key", uuid.New())
	if err != nil {
		t.Errorf("AutoLinkSpecToPR should succeed (no-op) when no recent spec, got: %v", err)
	}
	if len(evidenceStore.links) != 0 {
		t.Errorf("no link should be created when no recent spec")
	}
}

func TestE2E_EvidenceLinker_ManualLink(t *testing.T) {
	ctx := context.Background()

	evidenceStore := &mockEvidenceLinkStore{links: []*store.EvidenceLink{}}
	linker := evidence.NewLinker(evidenceStore)

	fromID := uuid.New()
	toID := uuid.New()

	err := linker.ManualLink(ctx, "tenant-mts", "spec", fromID, "test_run", toID, "manual")
	if err != nil {
		t.Fatalf("ManualLink failed: %v", err)
	}
	if len(evidenceStore.links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(evidenceStore.links))
	}
	if evidenceStore.links[0].LinkReason != "manual" {
		t.Errorf("link reason should be 'manual', got %q", evidenceStore.links[0].LinkReason)
	}
}

func TestE2E_EvidenceLinker_NilStore(t *testing.T) {
	linker := evidence.NewLinker(nil)
	err := linker.AutoLinkSpecToPR(context.Background(), "t", "s", uuid.New())
	if err != nil {
		t.Errorf("nil store should return nil, got: %v", err)
	}
}

// ============================================================
// E2E: PR verdict parsing edge cases
// ============================================================

func TestE2E_PRVerdict_DefaultPending(t *testing.T) {
	// CTO-48: ambiguous reviews MUST default to "pending", never silently approve.
	ambiguous := []string{
		"Looks okay, I have some minor comments.",
		"The code is well-structured overall.",
		"",
		"Need more context to evaluate this PR.",
	}

	for _, content := range ambiguous {
		verdict := governance.ParsePRVerdict(content)
		if verdict != "pending" {
			t.Errorf("ambiguous content %q should be 'pending', got %q", content, verdict)
		}
	}
}

func TestE2E_PRVerdict_EmojiMarkers(t *testing.T) {
	tests := []struct {
		content  string
		expected string
	}{
		{"Review complete. 🟢 Pass — all checks satisfied.", "pass"},
		{"Security issue found. 🔴 Fail — SQL injection risk.", "fail"},
		{"All good! ✅ Pass", "pass"},
		{"Critical bug: ❌ Fail", "fail"},
	}

	for _, tt := range tests {
		verdict := governance.ParsePRVerdict(tt.content)
		if verdict != tt.expected {
			t.Errorf("content %q: expected %q, got %q", tt.content, tt.expected, verdict)
		}
	}
}

// ============================================================
// Mock implementations (in-memory, Zero Mock Policy compliant)
// ============================================================

type mockSpecStore struct {
	specs map[string]*store.GovernanceSpec
}

func (m *mockSpecStore) CreateSpec(_ context.Context, spec *store.GovernanceSpec) error {
	m.specs[spec.SpecID] = spec
	return nil
}

func (m *mockSpecStore) GetSpec(_ context.Context, specID string) (*store.GovernanceSpec, error) {
	if s, ok := m.specs[specID]; ok {
		return s, nil
	}
	return nil, os.ErrNotExist
}

func (m *mockSpecStore) ListSpecs(_ context.Context, opts store.SpecListOpts) ([]store.GovernanceSpec, error) {
	var result []store.GovernanceSpec
	for _, s := range m.specs {
		if opts.Status != "" && s.Status != opts.Status {
			continue
		}
		result = append(result, *s)
		if opts.Limit > 0 && len(result) >= opts.Limit {
			break
		}
	}
	return result, nil
}

func (m *mockSpecStore) CountSpecs(_ context.Context, opts store.SpecListOpts) (int, error) {
	count := 0
	for _, s := range m.specs {
		if opts.Status != "" && s.Status != opts.Status {
			continue
		}
		count++
	}
	return count, nil
}

func (m *mockSpecStore) UpdateSpecStatus(_ context.Context, specID string, status string) error {
	if s, ok := m.specs[specID]; ok {
		s.Status = status
		return nil
	}
	return os.ErrNotExist
}

func (m *mockSpecStore) NextSpecID(_ context.Context, year int) (string, error) {
	return "SPEC-2026-9999", nil
}

type mockPRGateStore struct {
	evals map[uuid.UUID]*store.PRGateEvaluation
}

func (m *mockPRGateStore) CreateEvaluation(_ context.Context, eval *store.PRGateEvaluation) error {
	eval.ID = uuid.New()
	eval.CreatedAt = time.Now()
	eval.UpdatedAt = time.Now()
	m.evals[eval.ID] = eval
	return nil
}

func (m *mockPRGateStore) GetEvaluation(_ context.Context, id uuid.UUID) (*store.PRGateEvaluation, error) {
	if e, ok := m.evals[id]; ok {
		return e, nil
	}
	return nil, os.ErrNotExist
}

func (m *mockPRGateStore) ListEvaluations(_ context.Context, _ store.PRGateFilter) ([]store.PRGateEvaluation, error) {
	var result []store.PRGateEvaluation
	for _, e := range m.evals {
		result = append(result, *e)
	}
	return result, nil
}

type mockEvidenceLinkStore struct {
	links               []*store.EvidenceLink
	recentSpecBySession map[string]*uuid.UUID
}

func (m *mockEvidenceLinkStore) CreateLink(_ context.Context, link *store.EvidenceLink) error {
	link.ID = uuid.New()
	link.CreatedAt = time.Now()
	m.links = append(m.links, link)
	return nil
}

func (m *mockEvidenceLinkStore) GetChain(_ context.Context, specID uuid.UUID) ([]store.EvidenceLink, error) {
	var result []store.EvidenceLink
	for _, l := range m.links {
		if l.FromID == specID {
			result = append(result, *l)
		}
	}
	return result, nil
}

func (m *mockEvidenceLinkStore) FindRecentSpecBySession(_ context.Context, sessionKey string) (*uuid.UUID, error) {
	if id, ok := m.recentSpecBySession[sessionKey]; ok {
		return id, nil
	}
	return nil, nil
}

func toJSON(t *testing.T, v interface{}) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("toJSON: %v", err)
	}
	return b
}
