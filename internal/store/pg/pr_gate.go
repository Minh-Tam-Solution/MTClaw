package pg

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// PGPRGateStore implements store.PRGateStore backed by Postgres.
type PGPRGateStore struct {
	db *sql.DB
}

func NewPGPRGateStore(db *sql.DB) *PGPRGateStore {
	return &PGPRGateStore{db: db}
}

func (s *PGPRGateStore) CreateEvaluation(ctx context.Context, eval *store.PRGateEvaluation) error {
	if eval.ID == uuid.Nil {
		eval.ID = store.GenNewID()
	}
	now := nowUTC()
	if eval.CreatedAt.IsZero() {
		eval.CreatedAt = now
	}
	eval.UpdatedAt = now

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO pr_gate_evaluations (id, owner_id, trace_id, pr_url, pr_number,
			repo, head_sha, mode, verdict, rules_evaluated, review_comment, soul_author,
			channel, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		eval.ID, eval.OwnerID, nilUUID(eval.TraceID), eval.PRURL, eval.PRNumber,
		eval.Repo, eval.HeadSHA, eval.Mode, eval.Verdict,
		jsonOrEmpty(eval.RulesEvaluated), nilStr(eval.ReviewComment), nilStr(eval.SoulAuthor),
		nilStr(eval.Channel), eval.CreatedAt, eval.UpdatedAt,
	)
	return err
}

func (s *PGPRGateStore) GetEvaluation(ctx context.Context, id uuid.UUID) (*store.PRGateEvaluation, error) {
	var d store.PRGateEvaluation
	var traceID *uuid.UUID
	var reviewComment, soulAuthor, channel *string

	err := s.db.QueryRowContext(ctx,
		`SELECT id, owner_id, trace_id, pr_url, pr_number,
			repo, head_sha, mode, verdict, rules_evaluated,
			review_comment, soul_author, channel, created_at, updated_at
		 FROM pr_gate_evaluations WHERE id = $1`, id,
	).Scan(&d.ID, &d.OwnerID, &traceID, &d.PRURL, &d.PRNumber,
		&d.Repo, &d.HeadSHA, &d.Mode, &d.Verdict, &d.RulesEvaluated,
		&reviewComment, &soulAuthor, &channel, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}

	d.TraceID = traceID
	d.ReviewComment = derefStr(reviewComment)
	d.SoulAuthor = derefStr(soulAuthor)
	d.Channel = derefStr(channel)
	return &d, nil
}

func (s *PGPRGateStore) ListEvaluations(ctx context.Context, filter store.PRGateFilter) ([]store.PRGateEvaluation, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter.Repo != "" {
		conditions = append(conditions, fmt.Sprintf("repo = $%d", argIdx))
		args = append(args, filter.Repo)
		argIdx++
	}
	if filter.PRNumber != nil {
		conditions = append(conditions, fmt.Sprintf("pr_number = $%d", argIdx))
		args = append(args, *filter.PRNumber)
		argIdx++
	}
	if filter.Verdict != "" {
		conditions = append(conditions, fmt.Sprintf("verdict = $%d", argIdx))
		args = append(args, filter.Verdict)
		argIdx++
	}
	if filter.Since != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *filter.Since)
		argIdx++
	}
	if filter.Until != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *filter.Until)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	q := `SELECT id, owner_id, trace_id, pr_url, pr_number,
		repo, head_sha, mode, verdict, rules_evaluated,
		review_comment, soul_author, channel, created_at, updated_at
		FROM pr_gate_evaluations` + where

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	q += fmt.Sprintf(" ORDER BY created_at DESC OFFSET %d LIMIT %d", filter.Offset, limit)

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []store.PRGateEvaluation
	for rows.Next() {
		var d store.PRGateEvaluation
		var traceID *uuid.UUID
		var reviewComment, soulAuthor, channel *string

		if err := rows.Scan(&d.ID, &d.OwnerID, &traceID, &d.PRURL, &d.PRNumber,
			&d.Repo, &d.HeadSHA, &d.Mode, &d.Verdict, &d.RulesEvaluated,
			&reviewComment, &soulAuthor, &channel, &d.CreatedAt, &d.UpdatedAt); err != nil {
			slog.Warn("pr_gate: scan row", "error", err)
			continue
		}

		d.TraceID = traceID
		d.ReviewComment = derefStr(reviewComment)
		d.SoulAuthor = derefStr(soulAuthor)
		d.Channel = derefStr(channel)
		result = append(result, d)
	}
	return result, nil
}
