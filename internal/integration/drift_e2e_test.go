// Package integration_test validates Context Drift prevention end-to-end.
// Sprint 8 Task 3 — CTO-25: avoid package proliferation, tests go here.
// These tests verify prompt construction and routing logic, NOT live LLM calls.
package integration_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Minh-Tam-Solution/MTClaw/internal/rag"
	"github.com/Minh-Tam-Solution/MTClaw/internal/souls"
)

const soulDir = "../../docs/08-collaborate/souls"

// ---------- Test 1: Layer A — Context Anchoring ----------
// SOUL identity must be consistently anchored. A PM SOUL must not "drift"
// into developer behavior after many code-related questions.

func TestLayerA_ContextAnchoring_SOULIdentityStable(t *testing.T) {
	soulRoles := []string{"pm", "reviewer", "coder", "dev", "sales"}

	for _, role := range soulRoles {
		t.Run(role, func(t *testing.T) {
			content := loadSOULFile(t, role)

			// Identity section must exist (anchoring)
			if !containsSection(content, "## Identity") {
				t.Errorf("SOUL %q missing required '## Identity' section for context anchoring", role)
			}

			// Capabilities section must exist (scope boundary)
			if !containsSection(content, "## Capabilities") {
				t.Errorf("SOUL %q missing '## Capabilities' section", role)
			}

			// Constraints section must exist (prevents role creep)
			if !containsSection(content, "## Constraints") {
				t.Errorf("SOUL %q missing '## Constraints' section", role)
			}

			// Content must not be empty
			if len(content) < 100 {
				t.Errorf("SOUL %q content too short (%d bytes) — likely incomplete", role, len(content))
			}
		})
	}
}

// ---------- Test 2: Layer B — RAG Routing Per SOUL ----------
// @sales queries must route to sales collection, NOT engineering docs.

func TestLayerB_RAGRouting_SOULDomainCorrect(t *testing.T) {
	tests := []struct {
		name        string
		agentID     string
		wantCollect []string
		wantNot     []string
	}{
		{
			name:        "sales routes to sales collection only",
			agentID:     "sales",
			wantCollect: []string{"sales"},
			wantNot:     []string{"engineering"},
		},
		{
			name:        "dev routes to engineering collection",
			agentID:     "dev",
			wantCollect: []string{"engineering"},
			wantNot:     []string{"sales"},
		},
		{
			name:        "coder routes to engineering collection",
			agentID:     "coder",
			wantCollect: []string{"engineering"},
			wantNot:     []string{"sales"},
		},
		{
			name:        "reviewer routes to engineering collection",
			agentID:     "reviewer",
			wantCollect: []string{"engineering"},
			wantNot:     []string{"sales"},
		},
		{
			name:        "pm routes to both engineering and sales",
			agentID:     "pm",
			wantCollect: []string{"engineering", "sales"},
		},
		{
			name:        "cs routes to both engineering and sales",
			agentID:     "cs",
			wantCollect: []string{"engineering", "sales"},
		},
		{
			name:        "unmapped agent returns no collections",
			agentID:     "nonexistent-role",
			wantCollect: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collections := rag.CollectionMap[tt.agentID]

			if tt.wantCollect == nil {
				if len(collections) != 0 {
					t.Errorf("expected no collections for %q, got %v", tt.agentID, collections)
				}
				return
			}

			if len(collections) != len(tt.wantCollect) {
				t.Fatalf("expected %d collections for %q, got %d: %v",
					len(tt.wantCollect), tt.agentID, len(collections), collections)
			}

			for _, want := range tt.wantCollect {
				if !sliceContains(collections, want) {
					t.Errorf("expected collection %q for agent %q, got %v", want, tt.agentID, collections)
				}
			}

			for _, notWant := range tt.wantNot {
				if sliceContains(collections, notWant) {
					t.Errorf("agent %q must NOT route to %q collection, but found it in %v",
						tt.agentID, notWant, collections)
				}
			}
		})
	}
}

// ---------- Test 3: Layer C — Retrieval Evidence ----------
// RetrievalEvidence must be populated with ranking_reason for every RAG query.

func TestLayerC_RetrievalEvidence_RankingReasonPopulated(t *testing.T) {
	tests := []struct {
		name       string
		topScore   float64
		soulRole   string
		collection string
		wantReason string
	}{
		{
			name:       "exact match at high score",
			topScore:   0.98,
			soulRole:   "dev",
			collection: "engineering",
			wantReason: rag.RankingExactMatch,
		},
		{
			name:       "soul domain boost for matching collection",
			topScore:   0.80,
			soulRole:   "sales",
			collection: "sales",
			wantReason: rag.RankingSoulDomainBoost,
		},
		{
			name:       "semantic similar for cross-domain query",
			topScore:   0.70,
			soulRole:   "dev",
			collection: "sales",
			wantReason: rag.RankingSemanticSimilar,
		},
		{
			name:       "fallback for low score unmapped role",
			topScore:   0.30,
			soulRole:   "unknown",
			collection: "unknown",
			wantReason: rag.RankingFallback,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason := rag.ClassifyRankingReason(tt.topScore, tt.soulRole, tt.collection)
			if reason != tt.wantReason {
				t.Errorf("expected ranking_reason=%q, got %q", tt.wantReason, reason)
			}
			if reason == "" {
				t.Error("ranking_reason must never be empty")
			}
		})
	}
}

// ---------- Test 4: Cross-SOUL Delegation Identity ----------
// When delegating PM→Dev, each SOUL must have distinct identity anchoring.

func TestCrossSOUL_DelegationIdentityPreserved(t *testing.T) {
	pmContent := loadSOULFile(t, "pm")
	devContent := loadSOULFile(t, "dev")
	reviewerContent := loadSOULFile(t, "reviewer")

	// Each pair must have different Identity sections
	pmIdentity := extractSection(pmContent, "## Identity")
	devIdentity := extractSection(devContent, "## Identity")
	reviewerIdentity := extractSection(reviewerContent, "## Identity")

	if pmIdentity == devIdentity {
		t.Error("PM and Dev SOUL have identical Identity sections — delegation would not preserve role distinction")
	}
	if pmIdentity == reviewerIdentity {
		t.Error("PM and Reviewer SOUL have identical Identity sections")
	}
	if devIdentity == reviewerIdentity {
		t.Error("Dev and Reviewer SOUL have identical Identity sections")
	}

	// Each SOUL checksum must be unique (no copy-paste)
	pmHash := souls.ChecksumContent(pmContent)
	devHash := souls.ChecksumContent(devContent)
	reviewerHash := souls.ChecksumContent(reviewerContent)

	if pmHash == devHash {
		t.Error("PM and Dev SOUL have identical checksums — likely copy-paste without customization")
	}
	if pmHash == reviewerHash {
		t.Error("PM and Reviewer SOUL have identical checksums")
	}
}

// ---------- Test 5: Spec Output Format Stability ----------
// PM SOUL must contain spec-related guidance for /spec output format stability.

func TestSpecOutputFormatStability(t *testing.T) {
	pmContent := loadSOULFile(t, "pm")

	// PM SOUL must mention spec-related output format
	specKeywords := []string{"spec", "requirement", "user stor"}
	foundKeyword := false
	for _, kw := range specKeywords {
		if strings.Contains(strings.ToLower(pmContent), strings.ToLower(kw)) {
			foundKeyword = true
			break
		}
	}
	if !foundKeyword {
		t.Error("PM SOUL must reference specs/requirements — needed for /spec output format stability")
	}

	// PM SOUL must have structured output or delegation guidance
	hasOutputGuidance := containsSection(pmContent, "## Delegation") ||
		containsSection(pmContent, "## Output") ||
		containsSection(pmContent, "## Format") ||
		strings.Contains(strings.ToLower(pmContent), "format") ||
		strings.Contains(strings.ToLower(pmContent), "output")

	if !hasOutputGuidance {
		t.Error("PM SOUL should contain output format or delegation guidance for /spec stability")
	}

	// Verify checksum function is deterministic (drift detection baseline)
	hash1 := souls.ChecksumContent(pmContent)
	hash2 := souls.ChecksumContent(pmContent)
	if hash1 != hash2 {
		t.Fatal("ChecksumContent is not deterministic — cannot detect drift")
	}
	if len(hash1) != 64 { // SHA-256 hex = 64 chars
		t.Errorf("expected 64-char SHA-256 hex, got %d chars: %s", len(hash1), hash1)
	}
}

// ---------- Helpers ----------

func loadSOULFile(t *testing.T, role string) string {
	t.Helper()
	path := soulDir + "/SOUL-" + role + ".md"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read SOUL file %s: %v", path, err)
	}
	return string(data)
}

func containsSection(content, heading string) bool {
	return strings.Contains(strings.ToLower(content), strings.ToLower(heading))
}

func sliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func extractSection(content, heading string) string {
	idx := strings.Index(content, heading)
	if idx == -1 {
		return ""
	}
	rest := content[idx+len(heading):]
	nextIdx := strings.Index(rest, "\n## ")
	if nextIdx == -1 {
		return rest
	}
	return rest[:nextIdx]
}
