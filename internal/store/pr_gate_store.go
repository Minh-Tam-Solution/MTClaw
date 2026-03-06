package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// PRGateEvaluation represents a PR Gate review evaluation (Rail #2: PR Gate ENFORCE).
type PRGateEvaluation struct {
	ID             uuid.UUID       `json:"id"`
	OwnerID        string          `json:"owner_id"`
	TraceID        *uuid.UUID      `json:"trace_id,omitempty"`
	PRURL          string          `json:"pr_url"`
	PRNumber       int             `json:"pr_number"`
	Repo           string          `json:"repo"`
	HeadSHA        string          `json:"head_sha"`
	Mode           string          `json:"mode"`
	Verdict        string          `json:"verdict"`
	RulesEvaluated json.RawMessage `json:"rules_evaluated"`
	ReviewComment  string          `json:"review_comment,omitempty"`
	SoulAuthor     string          `json:"soul_author,omitempty"`
	Channel        string          `json:"channel,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// PRGateFilter configures pr_gate_evaluations listing.
type PRGateFilter struct {
	Repo     string
	PRNumber *int
	Verdict  string
	Since    *time.Time
	Until    *time.Time
	Limit    int
	Offset   int
}

// PRGateStore manages PR Gate evaluations (Rail #2: PR Gate ENFORCE).
type PRGateStore interface {
	CreateEvaluation(ctx context.Context, eval *PRGateEvaluation) error
	GetEvaluation(ctx context.Context, id uuid.UUID) (*PRGateEvaluation, error)
	ListEvaluations(ctx context.Context, filter PRGateFilter) ([]PRGateEvaluation, error)
}
