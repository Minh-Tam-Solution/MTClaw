package evidence

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// AllArtifactTypes references the SSOT in store package (CTO-49).
// Kept as package-level alias for backward compatibility within evidence package.
var AllArtifactTypes = store.AllArtifactTypes

// ChainBuilder constructs evidence chains from linked artifacts.
type ChainBuilder struct {
	evidenceStore store.EvidenceLinkStore
	specStore     store.SpecStore
	prGateStore   store.PRGateStore
}

// NewChainBuilder creates a new chain builder.
func NewChainBuilder(evidenceStore store.EvidenceLinkStore, specStore store.SpecStore, prGateStore store.PRGateStore) *ChainBuilder {
	return &ChainBuilder{
		evidenceStore: evidenceStore,
		specStore:     specStore,
		prGateStore:   prGateStore,
	}
}

// BuildChain constructs the full evidence chain for a governance spec.
func (b *ChainBuilder) BuildChain(ctx context.Context, spec *store.GovernanceSpec) (*store.EvidenceChain, error) {
	chain := &store.EvidenceChain{
		SpecID: spec.SpecID,
		Chain: []store.ChainNode{
			{
				Type:      "spec",
				ID:        spec.ID,
				CreatedAt: spec.CreatedAt,
				Status:    spec.Status,
			},
		},
	}

	// Get all linked artifacts starting from this spec.
	links, err := b.evidenceStore.GetChain(ctx, spec.ID)
	if err != nil {
		slog.Warn("evidence: failed to get chain links", "spec_id", spec.SpecID, "error", err)
		return chain, nil
	}

	// Track which artifact types are present.
	typeSeen := map[string]bool{"spec": true}

	for _, link := range links {
		node := store.ChainNode{
			Type:      link.ToType,
			ID:        link.ToID,
			CreatedAt: link.CreatedAt,
		}

		// Enrich node with data from source table.
		if link.ToType == "pr_gate" && b.prGateStore != nil {
			if eval, err := b.prGateStore.GetEvaluation(ctx, link.ToID); err == nil {
				node.Verdict = eval.Verdict
				node.PRURL = eval.PRURL
				node.CreatedAt = eval.CreatedAt
			}
		}

		chain.Chain = append(chain.Chain, node)
		typeSeen[link.ToType] = true
	}

	// Determine missing artifact types.
	for _, t := range AllArtifactTypes {
		if !typeSeen[t] {
			chain.Missing = append(chain.Missing, t)
		}
	}
	chain.ChainComplete = len(chain.Missing) == 0

	return chain, nil
}

// BuildChainBySpecID looks up a spec by its human-readable ID (SPEC-YYYY-NNNN)
// and builds the evidence chain.
func (b *ChainBuilder) BuildChainBySpecID(ctx context.Context, specID string) (*store.EvidenceChain, error) {
	spec, err := b.specStore.GetSpec(ctx, specID)
	if err != nil {
		return nil, err
	}
	return b.BuildChain(ctx, spec)
}

// BuildChainByUUID looks up a spec by its UUID and builds the evidence chain.
func (b *ChainBuilder) BuildChainByUUID(ctx context.Context, specUUID uuid.UUID) (*store.EvidenceChain, error) {
	// GetChain already uses the UUID directly, but we need the spec for the root node.
	// For now, we get links and build a minimal chain. This will be enhanced when
	// GetSpec supports UUID lookup.
	links, err := b.evidenceStore.GetChain(ctx, specUUID)
	if err != nil {
		return nil, err
	}

	chain := &store.EvidenceChain{
		Chain: []store.ChainNode{
			{
				Type: "spec",
				ID:   specUUID,
			},
		},
	}

	typeSeen := map[string]bool{"spec": true}
	for _, link := range links {
		node := store.ChainNode{
			Type:      link.ToType,
			ID:        link.ToID,
			CreatedAt: link.CreatedAt,
		}
		if link.ToType == "pr_gate" && b.prGateStore != nil {
			if eval, err := b.prGateStore.GetEvaluation(ctx, link.ToID); err == nil {
				node.Verdict = eval.Verdict
				node.PRURL = eval.PRURL
				node.CreatedAt = eval.CreatedAt
			}
		}
		chain.Chain = append(chain.Chain, node)
		typeSeen[link.ToType] = true
	}

	for _, t := range AllArtifactTypes {
		if !typeSeen[t] {
			chain.Missing = append(chain.Missing, t)
		}
	}
	chain.ChainComplete = len(chain.Missing) == 0

	return chain, nil
}
