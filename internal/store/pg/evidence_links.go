package pg

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// PGEvidenceLinkStore implements store.EvidenceLinkStore backed by Postgres (ADR-009).
type PGEvidenceLinkStore struct {
	db *sql.DB
}

func NewPGEvidenceLinkStore(db *sql.DB) *PGEvidenceLinkStore {
	return &PGEvidenceLinkStore{db: db}
}

func (s *PGEvidenceLinkStore) CreateLink(ctx context.Context, link *store.EvidenceLink) error {
	if link.ID == uuid.Nil {
		link.ID = store.GenNewID()
	}
	if link.CreatedAt.IsZero() {
		link.CreatedAt = nowUTC()
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO evidence_links (id, owner_id, from_type, from_id, to_type, to_id, link_reason, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (owner_id, from_type, from_id, to_type, to_id) DO NOTHING`,
		link.ID, link.OwnerID, link.FromType, link.FromID,
		link.ToType, link.ToID, nilStr(link.LinkReason), link.CreatedAt,
	)
	return err
}

func (s *PGEvidenceLinkStore) GetChain(ctx context.Context, specID uuid.UUID) ([]store.EvidenceLink, error) {
	// Walk outward from spec: spec -> pr_gate -> test_run -> deploy (max depth 4).
	// Uses iterative BFS via recursive CTE limited to 4 hops.
	rows, err := s.db.QueryContext(ctx,
		`WITH RECURSIVE chain AS (
			SELECT id, owner_id, from_type, from_id, to_type, to_id, link_reason, created_at, 1 AS depth
			FROM evidence_links
			WHERE from_id = $1 AND from_type = 'spec'
			UNION ALL
			SELECT el.id, el.owner_id, el.from_type, el.from_id, el.to_type, el.to_id, el.link_reason, el.created_at, c.depth + 1
			FROM evidence_links el
			JOIN chain c ON el.from_id = c.to_id AND el.from_type = c.to_type
			WHERE c.depth < 4
		)
		SELECT id, owner_id, from_type, from_id, to_type, to_id, link_reason, created_at
		FROM chain
		ORDER BY created_at ASC`, specID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []store.EvidenceLink
	for rows.Next() {
		var l store.EvidenceLink
		var linkReason *string
		if err := rows.Scan(&l.ID, &l.OwnerID, &l.FromType, &l.FromID,
			&l.ToType, &l.ToID, &linkReason, &l.CreatedAt); err != nil {
			continue
		}
		l.LinkReason = derefStr(linkReason)
		result = append(result, l)
	}
	return result, nil
}

func (s *PGEvidenceLinkStore) FindRecentSpecBySession(ctx context.Context, sessionKey string) (*uuid.UUID, error) {
	// CTO-42: query governance_specs JOIN traces ON trace_id WHERE traces.session_key matches
	// and spec was created within last 48h.
	var specID uuid.UUID
	err := s.db.QueryRowContext(ctx,
		`SELECT gs.id FROM governance_specs gs
		 JOIN traces t ON gs.trace_id = t.id
		 WHERE t.session_key = $1
		   AND gs.created_at > now() - interval '48 hours'
		 ORDER BY gs.created_at DESC
		 LIMIT 1`, sessionKey,
	).Scan(&specID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &specID, nil
}
