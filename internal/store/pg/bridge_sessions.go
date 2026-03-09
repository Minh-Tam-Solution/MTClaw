package pg

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// PGBridgeSessionStore implements BridgeSessionStore with PostgreSQL.
type PGBridgeSessionStore struct {
	db *sql.DB
}

// NewPGBridgeSessionStore creates a new PG-backed bridge session store.
func NewPGBridgeSessionStore(db *sql.DB) *PGBridgeSessionStore {
	return &PGBridgeSessionStore{db: db}
}

const bridgeSessionCols = `id, owner_id, agent_type, tmux_target, project_path,
	workspace_fingerprint, status, risk_mode, capabilities, owner_actor_id,
	approver_acl, notify_acl, user_id, channel, chat_id, interactive_eligible,
	hook_secret, created_at, last_activity_at, stopped_at`

func (s *PGBridgeSessionStore) Upsert(ctx context.Context, rec *store.BridgeSessionRecord) error {
	caps := jsonOrEmpty(rec.Capabilities)
	approverACL := jsonOrEmpty(rec.ApproverACL)
	notifyACL := jsonOrEmpty(rec.NotifyACL)

	q := `INSERT INTO bridge_sessions (` + bridgeSessionCols + `)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			risk_mode = EXCLUDED.risk_mode,
			capabilities = EXCLUDED.capabilities,
			last_activity_at = EXCLUDED.last_activity_at,
			stopped_at = EXCLUDED.stopped_at`

	_, err := s.db.ExecContext(ctx, q,
		rec.ID, rec.OwnerID, rec.AgentType, rec.TmuxTarget, rec.ProjectPath,
		rec.WorkspaceFingerprint, rec.Status, rec.RiskMode, caps, rec.OwnerActorID,
		approverACL, notifyACL, nilStr(rec.UserID), nilStr(rec.Channel), nilStr(rec.ChatID),
		rec.InteractiveEligible, nilStr(rec.HookSecret), rec.CreatedAt, rec.LastActivityAt,
		nilTime(rec.StoppedAt),
	)
	if err != nil {
		return fmt.Errorf("upsert bridge session %s: %w", rec.ID, err)
	}
	return nil
}

func (s *PGBridgeSessionStore) Get(ctx context.Context, id string) (*store.BridgeSessionRecord, error) {
	q := `SELECT ` + bridgeSessionCols + ` FROM bridge_sessions WHERE id = $1`
	row := s.db.QueryRowContext(ctx, q, id)
	rec, err := scanBridgeSession(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get bridge session %s: %w", id, err)
	}
	return rec, nil
}

func (s *PGBridgeSessionStore) ListByTenant(ctx context.Context, tenantID string) ([]*store.BridgeSessionRecord, error) {
	q := `SELECT ` + bridgeSessionCols + ` FROM bridge_sessions
		WHERE owner_id = $1 AND status != 'stopped'
		ORDER BY created_at DESC`

	rows, err := s.db.QueryContext(ctx, q, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list bridge sessions for tenant %s: %w", tenantID, err)
	}
	defer rows.Close()

	var recs []*store.BridgeSessionRecord
	for rows.Next() {
		rec, err := scanBridgeSessionFromRows(rows)
		if err != nil {
			return nil, fmt.Errorf("scan bridge session row: %w", err)
		}
		recs = append(recs, rec)
	}
	return recs, rows.Err()
}

func (s *PGBridgeSessionStore) ListActive(ctx context.Context) ([]*store.BridgeSessionRecord, error) {
	q := `SELECT ` + bridgeSessionCols + ` FROM bridge_sessions
		WHERE status != 'stopped'
		ORDER BY created_at DESC`

	rows, err := s.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list active bridge sessions: %w", err)
	}
	defer rows.Close()

	var recs []*store.BridgeSessionRecord
	for rows.Next() {
		rec, err := scanBridgeSessionFromRows(rows)
		if err != nil {
			return nil, fmt.Errorf("scan bridge session row: %w", err)
		}
		recs = append(recs, rec)
	}
	return recs, rows.Err()
}

func (s *PGBridgeSessionStore) UpdateStatus(ctx context.Context, id string, status string, stoppedAt *time.Time) error {
	q := `UPDATE bridge_sessions SET status = $1, stopped_at = $2, last_activity_at = $3 WHERE id = $4`
	_, err := s.db.ExecContext(ctx, q, status, nilTime(stoppedAt), nowUTC(), id)
	if err != nil {
		return fmt.Errorf("update bridge session status %s: %w", id, err)
	}
	return nil
}

func (s *PGBridgeSessionStore) UpdateRiskMode(ctx context.Context, id string, riskMode string, capabilities json.RawMessage) error {
	q := `UPDATE bridge_sessions SET risk_mode = $1, capabilities = $2, last_activity_at = $3 WHERE id = $4`
	_, err := s.db.ExecContext(ctx, q, riskMode, jsonOrEmpty(capabilities), nowUTC(), id)
	if err != nil {
		return fmt.Errorf("update bridge session risk mode %s: %w", id, err)
	}
	return nil
}

func (s *PGBridgeSessionStore) DeleteOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	q := `DELETE FROM bridge_sessions WHERE status = 'stopped' AND stopped_at < $1`
	result, err := s.db.ExecContext(ctx, q, cutoff)
	if err != nil {
		return 0, fmt.Errorf("delete old bridge sessions: %w", err)
	}
	return result.RowsAffected()
}

// scanBridgeSession scans a single row into a BridgeSessionRecord.
func scanBridgeSession(row *sql.Row) (*store.BridgeSessionRecord, error) {
	rec := &store.BridgeSessionRecord{}
	var (
		userID, channel, chatID, hookSecret *string
		caps, approverACL, notifyACL        []byte
		stoppedAt                           *time.Time
	)
	err := row.Scan(
		&rec.ID, &rec.OwnerID, &rec.AgentType, &rec.TmuxTarget, &rec.ProjectPath,
		&rec.WorkspaceFingerprint, &rec.Status, &rec.RiskMode, &caps, &rec.OwnerActorID,
		&approverACL, &notifyACL, &userID, &channel, &chatID, &rec.InteractiveEligible,
		&hookSecret, &rec.CreatedAt, &rec.LastActivityAt, &stoppedAt,
	)
	if err != nil {
		return nil, err
	}
	rec.UserID = derefStr(userID)
	rec.Channel = derefStr(channel)
	rec.ChatID = derefStr(chatID)
	rec.HookSecret = derefStr(hookSecret)
	rec.Capabilities = caps
	rec.ApproverACL = approverACL
	rec.NotifyACL = notifyACL
	rec.StoppedAt = stoppedAt
	return rec, nil
}

// scanBridgeSessionFromRows scans from sql.Rows (for list queries).
func scanBridgeSessionFromRows(rows *sql.Rows) (*store.BridgeSessionRecord, error) {
	rec := &store.BridgeSessionRecord{}
	var (
		userID, channel, chatID, hookSecret *string
		caps, approverACL, notifyACL        []byte
		stoppedAt                           *time.Time
	)
	err := rows.Scan(
		&rec.ID, &rec.OwnerID, &rec.AgentType, &rec.TmuxTarget, &rec.ProjectPath,
		&rec.WorkspaceFingerprint, &rec.Status, &rec.RiskMode, &caps, &rec.OwnerActorID,
		&approverACL, &notifyACL, &userID, &channel, &chatID, &rec.InteractiveEligible,
		&hookSecret, &rec.CreatedAt, &rec.LastActivityAt, &stoppedAt,
	)
	if err != nil {
		return nil, err
	}
	rec.UserID = derefStr(userID)
	rec.Channel = derefStr(channel)
	rec.ChatID = derefStr(chatID)
	rec.HookSecret = derefStr(hookSecret)
	rec.Capabilities = caps
	rec.ApproverACL = approverACL
	rec.NotifyACL = notifyACL
	rec.StoppedAt = stoppedAt
	return rec, nil
}
