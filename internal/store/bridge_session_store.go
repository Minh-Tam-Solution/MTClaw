package store

import (
	"context"
	"encoding/json"
	"time"
)

// BridgeSessionRecord maps to the bridge_sessions PostgreSQL table (migration 000018).
type BridgeSessionRecord struct {
	ID                   string          `json:"id"`
	OwnerID              string          `json:"owner_id"`
	AgentType            string          `json:"agent_type"`
	TmuxTarget           string          `json:"tmux_target"`
	ProjectPath          string          `json:"project_path"`
	WorkspaceFingerprint string          `json:"workspace_fingerprint"`
	Status               string          `json:"status"`
	RiskMode             string          `json:"risk_mode"`
	Capabilities         json.RawMessage `json:"capabilities"`
	OwnerActorID         string          `json:"owner_actor_id"`
	ApproverACL          json.RawMessage `json:"approver_acl"`
	NotifyACL            json.RawMessage `json:"notify_acl"`
	UserID               string          `json:"user_id"`
	Channel              string          `json:"channel"`
	ChatID               string          `json:"chat_id"`
	InteractiveEligible  bool            `json:"interactive_eligible"`
	HookSecret           string          `json:"hook_secret"`
	CreatedAt            time.Time       `json:"created_at"`
	LastActivityAt       time.Time       `json:"last_activity_at"`
	StoppedAt            *time.Time      `json:"stopped_at,omitempty"`
}

// BridgeSessionStore persists bridge session state to PostgreSQL.
// nil in standalone mode.
type BridgeSessionStore interface {
	Upsert(ctx context.Context, rec *BridgeSessionRecord) error
	Get(ctx context.Context, id string) (*BridgeSessionRecord, error)
	ListByTenant(ctx context.Context, tenantID string) ([]*BridgeSessionRecord, error)
	ListActive(ctx context.Context) ([]*BridgeSessionRecord, error)
	UpdateStatus(ctx context.Context, id string, status string, stoppedAt *time.Time) error
	UpdateRiskMode(ctx context.Context, id string, riskMode string, capabilities json.RawMessage) error
	DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
}
