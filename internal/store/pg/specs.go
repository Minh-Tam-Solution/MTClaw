package pg

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// PGSpecStore implements store.SpecStore backed by Postgres.
type PGSpecStore struct {
	db *sql.DB
}

func NewPGSpecStore(db *sql.DB) *PGSpecStore {
	return &PGSpecStore{db: db}
}

func (s *PGSpecStore) CreateSpec(ctx context.Context, spec *store.GovernanceSpec) error {
	if spec.ID == uuid.Nil {
		spec.ID = store.GenNewID()
	}
	now := nowUTC()
	if spec.CreatedAt.IsZero() {
		spec.CreatedAt = now
	}
	spec.UpdatedAt = now

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO governance_specs (id, owner_id, spec_id, spec_version, title,
			narrative, acceptance_criteria, bdd_scenarios, risks, technical_requirements,
			dependencies, priority, estimated_effort, status, tier, soul_author,
			channel, trace_id, content_hash, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)`,
		spec.ID, spec.OwnerID, spec.SpecID, spec.SpecVersion, spec.Title,
		jsonOrEmpty(spec.Narrative), jsonOrEmpty(spec.AcceptanceCriteria),
		jsonOrNull(spec.BDDScenarios), jsonOrNull(spec.Risks), jsonOrNull(spec.TechnicalRequirements),
		jsonOrNull(spec.Dependencies), spec.Priority, nilStr(spec.EstimatedEffort),
		spec.Status, spec.Tier, spec.SoulAuthor,
		nilStr(spec.Channel), nilUUID(spec.TraceID), nilStr(spec.ContentHash), spec.CreatedAt, spec.UpdatedAt,
	)
	return err
}

func (s *PGSpecStore) GetSpec(ctx context.Context, specID string) (*store.GovernanceSpec, error) {
	var d store.GovernanceSpec
	var traceID *uuid.UUID
	var estimatedEffort, contentHash, channel *string
	var bddScenarios, risks, techReqs, deps *[]byte

	err := s.db.QueryRowContext(ctx,
		`SELECT id, owner_id, spec_id, spec_version, title,
			narrative, acceptance_criteria, bdd_scenarios, risks, technical_requirements,
			dependencies, priority, estimated_effort, status, tier, soul_author,
			channel, trace_id, content_hash, created_at, updated_at
		 FROM governance_specs WHERE spec_id = $1`, specID,
	).Scan(&d.ID, &d.OwnerID, &d.SpecID, &d.SpecVersion, &d.Title,
		&d.Narrative, &d.AcceptanceCriteria, &bddScenarios, &risks, &techReqs,
		&deps, &d.Priority, &estimatedEffort, &d.Status, &d.Tier, &d.SoulAuthor,
		&channel, &traceID, &contentHash, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}

	d.TraceID = traceID
	d.EstimatedEffort = derefStr(estimatedEffort)
	d.ContentHash = derefStr(contentHash)
	d.Channel = derefStr(channel)
	d.BDDScenarios = derefBytes(bddScenarios)
	d.Risks = derefBytes(risks)
	d.TechnicalRequirements = derefBytes(techReqs)
	d.Dependencies = derefBytes(deps)
	return &d, nil
}

func buildSpecWhere(opts store.SpecListOpts) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if opts.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, opts.Status)
		argIdx++
	}
	if opts.Since != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *opts.Since)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}
	return where, args
}

func (s *PGSpecStore) ListSpecs(ctx context.Context, opts store.SpecListOpts) ([]store.GovernanceSpec, error) {
	where, args := buildSpecWhere(opts)

	q := `SELECT id, owner_id, spec_id, spec_version, title,
		narrative, acceptance_criteria, bdd_scenarios, risks, technical_requirements,
		dependencies, priority, estimated_effort, status, tier, soul_author,
		channel, trace_id, content_hash, created_at, updated_at
		FROM governance_specs` + where

	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}
	q += fmt.Sprintf(" ORDER BY created_at DESC OFFSET %d LIMIT %d", opts.Offset, limit)

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []store.GovernanceSpec
	for rows.Next() {
		var d store.GovernanceSpec
		var traceID *uuid.UUID
		var estimatedEffort, contentHash, channel *string
		var bddScenarios, risks, techReqs, deps *[]byte

		if err := rows.Scan(&d.ID, &d.OwnerID, &d.SpecID, &d.SpecVersion, &d.Title,
			&d.Narrative, &d.AcceptanceCriteria, &bddScenarios, &risks, &techReqs,
			&deps, &d.Priority, &estimatedEffort, &d.Status, &d.Tier, &d.SoulAuthor,
			&channel, &traceID, &contentHash, &d.CreatedAt, &d.UpdatedAt); err != nil {
			continue
		}

		d.TraceID = traceID
		d.EstimatedEffort = derefStr(estimatedEffort)
		d.ContentHash = derefStr(contentHash)
		d.Channel = derefStr(channel)
		d.BDDScenarios = derefBytes(bddScenarios)
		d.Risks = derefBytes(risks)
		d.TechnicalRequirements = derefBytes(techReqs)
		d.Dependencies = derefBytes(deps)
		result = append(result, d)
	}
	return result, nil
}

func (s *PGSpecStore) CountSpecs(ctx context.Context, opts store.SpecListOpts) (int, error) {
	where, args := buildSpecWhere(opts)
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM governance_specs"+where, args...).Scan(&count)
	return count, err
}

func (s *PGSpecStore) UpdateSpecStatus(ctx context.Context, specID string, status string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE governance_specs SET status = $1, updated_at = $2 WHERE spec_id = $3`,
		status, nowUTC(), specID)
	return err
}

// NextSpecID generates the next SPEC-YYYY-NNNN for the current tenant.
// Relies on RLS to scope counter per tenant.
// Caller MUST ensure SET LOCAL app.tenant_id was called before invoking.
func (s *PGSpecStore) NextSpecID(ctx context.Context, year int) (string, error) {
	var maxSeq int
	prefix := fmt.Sprintf("SPEC-%d-", year)
	err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(CAST(split_part(spec_id, '-', 3) AS INT)), 0)
		 FROM governance_specs WHERE spec_id LIKE $1`, prefix+"%",
	).Scan(&maxSeq)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("SPEC-%d-%04d", year, maxSeq+1), nil
}
