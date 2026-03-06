package store

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// EvidenceLink represents a link between two governance artifacts (ADR-009).
// Polymorphic junction table: from_type/from_id -> to_type/to_id.
type EvidenceLink struct {
	ID         uuid.UUID `json:"id"`
	OwnerID    string    `json:"owner_id"`
	FromType   string    `json:"from_type"`   // 'spec', 'pr_gate', 'test_run', 'deploy'
	FromID     uuid.UUID `json:"from_id"`
	ToType     string    `json:"to_type"`
	ToID       uuid.UUID `json:"to_id"`
	LinkReason string    `json:"link_reason"` // 'manual', 'auto_spec_review', 'auto_pr_merge'
	CreatedAt  time.Time `json:"created_at"`
}

// ChainNode represents a single node in an evidence chain.
type ChainNode struct {
	Type      string    `json:"type"`
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	// Additional fields populated from source table.
	Status  string `json:"status,omitempty"`  // spec status
	Verdict string `json:"verdict,omitempty"` // pr_gate verdict
	PRURL   string `json:"pr_url,omitempty"`  // pr_gate PR URL
}

// EvidenceChain is the response for the evidence chain API.
type EvidenceChain struct {
	SpecID        string      `json:"spec_id"`
	Chain         []ChainNode `json:"chain"`
	ChainComplete bool        `json:"chain_complete"`
	Missing       []string    `json:"missing,omitempty"` // missing artifact types
}

// EvidenceLinkStore manages evidence links (ADR-009).
type EvidenceLinkStore interface {
	// CreateLink inserts a new evidence link. Returns ErrDuplicateLink if the
	// (owner_id, from_type, from_id, to_type, to_id) tuple already exists.
	CreateLink(ctx context.Context, link *EvidenceLink) error

	// GetChain returns all linked artifacts starting from a given spec.
	GetChain(ctx context.Context, specID uuid.UUID) ([]EvidenceLink, error)

	// FindRecentSpec finds the most recent governance_spec in the same session
	// (via traces.session_key) within the last 48h. Used for auto-linking.
	FindRecentSpecBySession(ctx context.Context, sessionKey string) (*uuid.UUID, error)
}
