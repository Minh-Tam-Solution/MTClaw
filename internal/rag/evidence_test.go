package rag

import "testing"

func TestClassifyRankingReason_ExactMatch(t *testing.T) {
	reason := ClassifyRankingReason(0.98, "enghelp", "engineering")
	if reason != RankingExactMatch {
		t.Errorf("expected %q for score 0.98, got %q", RankingExactMatch, reason)
	}
}

func TestClassifyRankingReason_SoulDomainBoost(t *testing.T) {
	reason := ClassifyRankingReason(0.80, "enghelp", "engineering")
	if reason != RankingSoulDomainBoost {
		t.Errorf("expected %q for enghelp+engineering, got %q", RankingSoulDomainBoost, reason)
	}
}

func TestClassifyRankingReason_SemanticSimilar(t *testing.T) {
	reason := ClassifyRankingReason(0.70, "enghelp", "sales")
	if reason != RankingSemanticSimilar {
		t.Errorf("expected %q for enghelp querying sales, got %q", RankingSemanticSimilar, reason)
	}
}

func TestClassifyRankingReason_Fallback(t *testing.T) {
	reason := ClassifyRankingReason(0.30, "unknown_role", "unknown_collection")
	if reason != RankingFallback {
		t.Errorf("expected %q for low score, got %q", RankingFallback, reason)
	}
}

func TestClassifyRankingReason_BoundaryExactMatch(t *testing.T) {
	// Exactly 0.95 should be exact_match
	reason := ClassifyRankingReason(0.95, "enghelp", "engineering")
	if reason != RankingExactMatch {
		t.Errorf("expected %q for score 0.95, got %q", RankingExactMatch, reason)
	}
}

func TestClassifyRankingReason_BoundarySemantic(t *testing.T) {
	// Exactly 0.5, non-matching collection should be semantic_similar
	reason := ClassifyRankingReason(0.50, "enghelp", "sales")
	if reason != RankingSemanticSimilar {
		t.Errorf("expected %q for score 0.50, got %q", RankingSemanticSimilar, reason)
	}
}

func TestClassifyRankingReason_CSRole(t *testing.T) {
	// CS maps to both engineering + sales
	reason := ClassifyRankingReason(0.80, "cs", "sales")
	if reason != RankingSoulDomainBoost {
		t.Errorf("expected %q for cs+sales, got %q", RankingSoulDomainBoost, reason)
	}
	reason2 := ClassifyRankingReason(0.80, "cs", "engineering")
	if reason2 != RankingSoulDomainBoost {
		t.Errorf("expected %q for cs+engineering, got %q", RankingSoulDomainBoost, reason2)
	}
}
