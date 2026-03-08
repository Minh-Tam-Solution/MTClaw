// Package evidence implements cross-rail evidence linking (ADR-009).
// Sprint 11: links governance specs to PR gate evaluations (and future artifact types)
// into queryable evidence chains for compliance audit trails.
package evidence

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// Linker handles automatic and manual evidence linking.
type Linker struct {
	evidenceStore store.EvidenceLinkStore
}

// NewLinker creates a new evidence linker.
func NewLinker(evidenceStore store.EvidenceLinkStore) *Linker {
	return &Linker{evidenceStore: evidenceStore}
}

// AutoLinkSpecToPR auto-links the most recent governance_spec in the same
// session (within 48h) to a newly created pr_gate_evaluation.
// sessionKey maps to traces.session_key column (CTO-42).
func (l *Linker) AutoLinkSpecToPR(ctx context.Context, ownerID string,
	sessionKey string, prGateID uuid.UUID) error {

	if l.evidenceStore == nil || sessionKey == "" {
		return nil
	}

	specID, err := l.evidenceStore.FindRecentSpecBySession(ctx, sessionKey)
	if err != nil {
		slog.Warn("evidence: failed to find recent spec for auto-link",
			"session_key", sessionKey, "error", err)
		return err
	}
	if specID == nil {
		slog.Debug("evidence: no recent spec in session for auto-link",
			"session_key", sessionKey)
		return nil
	}

	link := &store.EvidenceLink{
		OwnerID:    ownerID,
		FromType:   "spec",
		FromID:     *specID,
		ToType:     "pr_gate",
		ToID:       prGateID,
		LinkReason: "auto_spec_review",
	}
	if err := l.evidenceStore.CreateLink(ctx, link); err != nil {
		slog.Warn("evidence: auto-link failed",
			"spec_id", specID, "pr_gate_id", prGateID, "error", err)
		return err
	}

	slog.Info("evidence: auto-linked spec to pr_gate",
		"spec_id", specID, "pr_gate_id", prGateID,
		"link_reason", "auto_spec_review", "session_key", sessionKey)
	return nil
}

// ManualLink creates a manual evidence link between two artifacts.
func (l *Linker) ManualLink(ctx context.Context, ownerID, fromType string, fromID uuid.UUID,
	toType string, toID uuid.UUID, reason string) error {

	if l.evidenceStore == nil {
		return nil
	}

	link := &store.EvidenceLink{
		OwnerID:    ownerID,
		FromType:   fromType,
		FromID:     fromID,
		ToType:     toType,
		ToID:       toID,
		LinkReason: reason,
	}
	return l.evidenceStore.CreateLink(ctx, link)
}
