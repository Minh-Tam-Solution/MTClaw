package rag

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// InjectionResult contains the RAG context string and metadata for trace enrichment.
type InjectionResult struct {
	Prompt   string              // RAG context to append to ExtraPrompt
	Tags     []string            // Tags for trace enrichment (rag:collections, rag_hits:N, rag_tokens:N)
	Evidence []RetrievalEvidence // Per-collection evidence for audit trail (Layer C)
}

// InjectRAGContext queries RAG collections for the given SOUL role and builds
// the knowledge base context prompt. Returns empty result if no matches.
// Sprint 6: SOUL-Aware RAG Routing (US-034, Context Drift Layer B).
// Sprint 7: Adds RetrievalEvidence per collection (Context Drift Layer C).
func InjectRAGContext(ctx context.Context, client *Client, query string, agentID string) InjectionResult {
	var result InjectionResult

	collections := CollectionMap[agentID]
	if len(collections) == 0 || query == "" {
		return result
	}

	ragCtx, ragCancel := context.WithTimeout(ctx, 5*time.Second)
	defer ragCancel()

	start := time.Now()
	ragResults, err := client.QueryMultiple(ragCtx, query, collections, 2500)
	latency := time.Since(start)

	if err != nil {
		return result
	}
	if ragResults == nil || len(ragResults.Results) == 0 {
		return result
	}

	// Build prompt
	var sb strings.Builder
	sb.WriteString("## Knowledge Base Context\n")
	sb.WriteString("The following information was retrieved from the knowledge base.\n")
	sb.WriteString("Cite sources when using this information.\n\n")
	for _, r := range ragResults.Results {
		sb.WriteString(fmt.Sprintf("### %s (score: %.2f)\n%s\n\n",
			r.Metadata.Source, r.Score, r.Content))
	}
	result.Prompt = sb.String()

	// Build trace tags
	result.Tags = append(result.Tags,
		"rag:"+strings.Join(collections, "+"),
		fmt.Sprintf("rag_hits:%d", ragResults.TotalHits),
		fmt.Sprintf("rag_tokens:%d", ragResults.TokensUsed),
	)

	// Sprint 7 Layer C: Build retrieval evidence per collection.
	// Uses top score from merged results as representative score.
	topScore := 0.0
	if len(ragResults.Results) > 0 {
		topScore = ragResults.Results[0].Score
	}
	for _, coll := range collections {
		evidence := RetrievalEvidence{
			Query:         query,
			Collection:    coll,
			ResultCount:   ragResults.TotalHits,
			TopScore:      topScore,
			RankingReason: ClassifyRankingReason(topScore, agentID, coll),
			SoulRole:      agentID,
			TokenCount:    ragResults.TokensUsed,
			LatencyMS:     int(latency.Milliseconds()),
		}
		result.Evidence = append(result.Evidence, evidence)
	}

	return result
}
