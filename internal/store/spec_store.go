package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Spec status constants.
const (
	SpecStatusDraft      = "draft"
	SpecStatusReview     = "review"
	SpecStatusApproved   = "approved"
	SpecStatusDeprecated = "deprecated"
)

// GovernanceSpec represents a structured specification (Rail #1: Spec Factory).
type GovernanceSpec struct {
	ID                    uuid.UUID       `json:"id"`
	OwnerID               string          `json:"owner_id"`
	SpecID                string          `json:"spec_id"`
	SpecVersion           string          `json:"spec_version"`
	Title                 string          `json:"title"`
	Narrative             json.RawMessage `json:"narrative"`
	AcceptanceCriteria    json.RawMessage `json:"acceptance_criteria"`
	BDDScenarios          json.RawMessage `json:"bdd_scenarios,omitempty"`
	Risks                 json.RawMessage `json:"risks,omitempty"`
	TechnicalRequirements json.RawMessage `json:"technical_requirements,omitempty"`
	Dependencies          json.RawMessage `json:"dependencies,omitempty"`
	Priority              string          `json:"priority"`
	EstimatedEffort       string          `json:"estimated_effort"`
	Status                string          `json:"status"`
	Tier                  string          `json:"tier"`
	SoulAuthor            string          `json:"soul_author"`
	Channel               string          `json:"channel,omitempty"`
	TraceID               *uuid.UUID      `json:"trace_id,omitempty"`
	ContentHash           string          `json:"content_hash,omitempty"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
}

// SpecListOpts configures spec listing.
type SpecListOpts struct {
	Status string
	Since  *time.Time
	Limit  int
	Offset int
}

// SpecStore manages governance specs (Rail #1: Spec Factory).
type SpecStore interface {
	CreateSpec(ctx context.Context, spec *GovernanceSpec) error
	GetSpec(ctx context.Context, specID string) (*GovernanceSpec, error)
	ListSpecs(ctx context.Context, opts SpecListOpts) ([]GovernanceSpec, error)
	CountSpecs(ctx context.Context, opts SpecListOpts) (int, error)
	UpdateSpecStatus(ctx context.Context, specID string, status string) error

	// NextSpecID generates the next SPEC-YYYY-NNNN for the current tenant.
	// Caller MUST ensure SET LOCAL app.tenant_id was called before invoking (CTO-18).
	NextSpecID(ctx context.Context, year int) (string, error)
}
