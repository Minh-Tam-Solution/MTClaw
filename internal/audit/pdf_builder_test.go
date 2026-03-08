package audit

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

func TestAuditTrailPDF_NilSpec(t *testing.T) {
	_, err := AuditTrailPDF(nil, []store.ChainNode{{Type: "spec"}})
	if err == nil {
		t.Fatal("expected error for nil spec")
	}
	if err.Error() != "audit: spec is nil" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAuditTrailPDF_EmptyChain(t *testing.T) {
	spec := &store.GovernanceSpec{
		ID:     uuid.New(),
		SpecID: "SPEC-2026-0001",
		Title:  "Test Spec",
	}
	_, err := AuditTrailPDF(spec, nil)
	if err == nil {
		t.Fatal("expected error for empty chain")
	}
	if err.Error() != "audit: evidence chain is empty — cannot generate audit trail" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAuditTrailPDF_ValidChain(t *testing.T) {
	now := time.Now().UTC()
	specID := uuid.New()
	prID := uuid.New()

	bddScenarios, _ := json.Marshal([]map[string]string{
		{"scenario": "Given a spec, when reviewed, then pass"},
		{"scenario": "Given a PR, when checked, then verdict is set"},
	})

	spec := &store.GovernanceSpec{
		ID:              specID,
		OwnerID:         "tenant-001",
		SpecID:          "SPEC-2026-0042",
		SpecVersion:     "1.0.0",
		Title:           "User login with MFA",
		Status:          "approved",
		Priority:        "high",
		EstimatedEffort: "3 days",
		Tier:            "STANDARD",
		SoulAuthor:      "@architect",
		Channel:         "telegram",
		BDDScenarios:    bddScenarios,
		CreatedAt:       now.Add(-48 * time.Hour),
		UpdatedAt:       now,
	}

	chain := []store.ChainNode{
		{
			Type:      "spec",
			ID:        specID,
			CreatedAt: now.Add(-48 * time.Hour),
			Status:    "approved",
		},
		{
			Type:      "pr_gate",
			ID:        prID,
			CreatedAt: now.Add(-24 * time.Hour),
			Verdict:   "pass",
			PRURL:     "https://github.com/org/repo/pull/42",
		},
		{
			Type:      "test_run",
			ID:        uuid.New(),
			CreatedAt: now.Add(-12 * time.Hour),
			Status:    "passed",
		},
	}

	pdfBytes, err := AuditTrailPDF(spec, chain)
	if err != nil {
		t.Fatalf("AuditTrailPDF failed: %v", err)
	}

	if len(pdfBytes) == 0 {
		t.Fatal("expected non-empty PDF bytes")
	}

	// PDF files start with %PDF
	if len(pdfBytes) < 4 || string(pdfBytes[:4]) != "%PDF" {
		t.Errorf("expected PDF magic bytes, got %q", string(pdfBytes[:4]))
	}
}

func TestAuditTrailPDF_SpecOnlyChain(t *testing.T) {
	spec := &store.GovernanceSpec{
		ID:        uuid.New(),
		OwnerID:   "tenant-002",
		SpecID:    "SPEC-2026-0099",
		Title:     "Minimal spec",
		Status:    "draft",
		CreatedAt: time.Now().UTC(),
	}

	chain := []store.ChainNode{
		{
			Type:      "spec",
			ID:        spec.ID,
			CreatedAt: spec.CreatedAt,
			Status:    "draft",
		},
	}

	pdfBytes, err := AuditTrailPDF(spec, chain)
	if err != nil {
		t.Fatalf("AuditTrailPDF failed: %v", err)
	}

	if len(pdfBytes) < 4 || string(pdfBytes[:4]) != "%PDF" {
		t.Errorf("expected valid PDF output")
	}
}

func TestAuditTrailPDF_NoPRGateNodes(t *testing.T) {
	spec := &store.GovernanceSpec{
		ID:        uuid.New(),
		OwnerID:   "tenant-003",
		SpecID:    "SPEC-2026-0100",
		Title:     "Spec without PR reviews",
		Status:    "review",
		CreatedAt: time.Now().UTC(),
	}

	chain := []store.ChainNode{
		{
			Type:      "spec",
			ID:        spec.ID,
			CreatedAt: spec.CreatedAt,
			Status:    "review",
		},
		{
			Type:      "test_run",
			ID:        uuid.New(),
			CreatedAt: time.Now().UTC(),
			Status:    "passed",
		},
	}

	pdfBytes, err := AuditTrailPDF(spec, chain)
	if err != nil {
		t.Fatalf("AuditTrailPDF failed: %v", err)
	}

	// Should still produce valid PDF (with "No PR Gate evaluations linked." message)
	if len(pdfBytes) < 4 || string(pdfBytes[:4]) != "%PDF" {
		t.Errorf("expected valid PDF output")
	}
}
