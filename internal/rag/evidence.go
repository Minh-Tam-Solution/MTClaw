package rag

// RetrievalEvidence captures metadata about a RAG retrieval for audit trail.
// Context Drift Layer C — Evidence & Explainability (Sprint 7).
// Reference: EndiorBot ADR-015 (Retrieval Explainability).
type RetrievalEvidence struct {
	Query         string  `json:"query"`
	Collection    string  `json:"collection"`
	ResultCount   int     `json:"result_count"`
	TopScore      float64 `json:"top_score"`
	RankingReason string  `json:"ranking_reason"`
	SoulRole      string  `json:"soul_role"`
	TokenCount    int     `json:"token_count"`
	LatencyMS     int     `json:"latency_ms"`
}

// Ranking reason constants.
const (
	RankingExactMatch       = "exact_match"
	RankingSoulDomainBoost  = "soul_domain_boost"
	RankingSemanticSimilar  = "semantic_similar"
	RankingFallback         = "fallback"
)

// ClassifyRankingReason determines the ranking reason for a RAG retrieval.
func ClassifyRankingReason(topScore float64, soulRole string, collection string) string {
	if topScore >= 0.95 {
		return RankingExactMatch
	}
	// Check if collection matches SOUL domain (soul_domain_boost)
	soulCollections, ok := CollectionMap[soulRole]
	if ok {
		for _, c := range soulCollections {
			if c == collection {
				return RankingSoulDomainBoost
			}
		}
	}
	if topScore >= 0.5 {
		return RankingSemanticSimilar
	}
	return RankingFallback
}
