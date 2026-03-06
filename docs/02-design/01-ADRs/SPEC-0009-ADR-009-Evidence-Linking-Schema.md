---
spec_id: SPEC-0009
adr_id: ADR-009
title: Evidence Linking Schema — Junction Table for Cross-Rail Traceability
status: APPROVED
date: 2026-03-22
author: "[@architect]"
reviewers: "[@cto], [@pm]"
approved_by: "@cto"
approval_date: 2026-03-06
sdlc_version: "6.1.1"
implements: "T11-01"
related_adrs: [ADR-002, ADR-004]
---

# ADR-009: Evidence Linking Schema — Junction Table for Cross-Rail Traceability

**SDLC Stage**: 02-Design
**Status**: APPROVED — [@cto] 2026-03-06
**Date**: 2026-03-22

---

## Context

MTClaw's 3 governance rails produce evidence artifacts in separate tables:

- **Rail #1 (`/spec`)**: `governance_specs` table — BDD specifications with risk scores
- **Rail #2 (`/review`)**: `pr_gate_evaluations` table — PR verdicts (BLOCK/WARN/PASS)
- **Rail #3 (RAG)**: knowledge queries (stateless, no persistent table)

Both `governance_specs` and `pr_gate_evaluations` reference `traces` via `trace_id` FK, but there is **no programmatic link between them**. A `/spec` call and a later `/review` on the same PR have no direct relationship in the database.

### Current Schema Gap

```
governance_specs      — has: id, spec_id, trace_id FK, owner_id
pr_gate_evaluations   — has: id, trace_id FK, owner_id, pr_url, verdict
traces                — has: id (both reference this)

Missing: governance_specs ←→ pr_gate_evaluations link
```

The CTO currently assembles evidence chains manually by cross-referencing both tables. This does not scale for OaaS (Sprint 12) where multiple tenants need automated compliance reporting.

---

## Problem Statement

> How should MTClaw link governance artifacts (specs, PR gates, future test runs, deploy events) into a queryable evidence chain that supports N:M relationships and future artifact types without schema changes?

---

## Options Evaluated

| # | Option | Schema Change | Relationship | Future Extensibility | Migration Impact |
|---|--------|--------------|-------------|---------------------|-----------------|
| A | Add `spec_id FK` on `pr_gate_evaluations` | ALTER TABLE | 1:N only | Poor — new FK per artifact type | 1 column add |
| **B** | **`evidence_links` junction table** | **CREATE TABLE** | **N:M** | **Excellent — new types = new rows, no schema change** | **1 new table** |
| C | Join via `traces` table (no migration) | None | Implicit | None — traces not guaranteed linked | 0 |

---

## Decision

**Option B: `evidence_links` junction table**

### Rationale

1. **N:M relationships**: A spec can link to multiple PR gate evaluations (e.g., spec revised after first PR review). A PR gate can link back to multiple specs (mono-repo with multiple specs per PR).

2. **Future-proof**: Sprint 12 OaaS needs spec -> PR -> test_run -> deploy as a 4-node chain. Option A would require 3 more ALTER TABLE migrations. Option B handles all link types with zero schema changes — just new rows with different `from_type`/`to_type` values.

3. **Audit-friendly**: Each link has `link_reason` (manual, auto_spec_review, auto_pr_merge) and `created_at` — full traceability of how evidence was connected.

4. **RLS-compatible**: `owner_id` column + RLS policy ensures cross-tenant isolation on links, consistent with existing table patterns (ADR-002).

### Trade-offs

- **Slightly more joins** than a direct FK — acceptable given indexed queries and small table size
- **No referential integrity** on `from_id`/`to_id` (polymorphic pattern) — mitigated by application-level validation before INSERT

---

## Implementation

### Migration 000017 (`evidence_links`)

```sql
-- 000017_evidence_links.up.sql
CREATE TABLE evidence_links (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id    VARCHAR(64) NOT NULL,
    from_type   VARCHAR(32) NOT NULL,  -- 'spec', 'pr_gate', 'test_run', 'deploy'
    from_id     UUID NOT NULL,
    to_type     VARCHAR(32) NOT NULL,
    to_id       UUID NOT NULL,
    link_reason VARCHAR(64),           -- 'manual', 'auto_spec_review', 'auto_pr_merge'
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (owner_id, from_type, from_id, to_type, to_id)
);

CREATE INDEX idx_evidence_links_owner ON evidence_links (owner_id);
CREATE INDEX idx_evidence_links_from ON evidence_links (owner_id, from_type, from_id);
CREATE INDEX idx_evidence_links_to ON evidence_links (owner_id, to_type, to_id);

ALTER TABLE evidence_links ENABLE ROW LEVEL SECURITY;
CREATE POLICY evidence_links_tenant ON evidence_links
    USING (owner_id = current_setting('app.tenant_id', true));
```

```sql
-- 000017_evidence_links.down.sql
DROP POLICY IF EXISTS evidence_links_tenant ON evidence_links;
DROP TABLE IF EXISTS evidence_links;
```

### Auto-Linking Logic

When `/review <pr_url>` is called in a session where `/spec` was used within the last 48h:

```go
// internal/evidence/linker.go
func (l *Linker) AutoLinkSpecToPR(ctx context.Context, ownerID string,
    sessionKey string, prGateID uuid.UUID) error {
    // 1. Query: most recent governance_spec in same session, last 48h:
    //    SELECT gs.id FROM governance_specs gs
    //    JOIN traces t ON gs.trace_id = t.id
    //    WHERE t.session_key = $sessionKey
    //      AND gs.owner_id = $ownerID
    //      AND gs.created_at > now() - interval '48h'
    //    ORDER BY gs.created_at DESC LIMIT 1
    // 2. If found: INSERT INTO evidence_links
    //    (from_type='spec', from_id=spec.id, to_type='pr_gate', to_id=prGateID,
    //     link_reason='auto_spec_review')
    // 3. Log: slog.Info("evidence_links: auto-linked", ...)
}
```

Note: `sessionKey` maps to `traces.session_key` column (not a session_id — governance_specs has no session column).

Called from: `internal/channels/*/commands.go` after successful `/review` command.

### Manual Linking API

```
POST /api/v1/spec/{spec_id}/link
Body: { "to_type": "pr_gate", "to_id": "<pr_gate_id>", "link_reason": "manual" }
Response: 201 Created
```

### Evidence Chain Query API

```
GET /api/v1/spec/{spec_id}/evidence-chain
Response:
{
  "spec_id": "SPEC-2026-0042",
  "chain": [
    { "type": "spec",    "id": "...", "created_at": "...", "status": "approved" },
    { "type": "pr_gate", "id": "...", "pr_url": "...", "verdict": "PASS", "created_at": "..." }
  ],
  "chain_complete": false,
  "missing": ["test_run", "deploy"]
}
```

### Code Locations

- `migrations/000017_evidence_links.{up,down}.sql` — schema
- `internal/evidence/linker.go` — auto-link logic
- `internal/evidence/chain.go` — chain query builder
- `internal/evidence/linker_test.go` — unit tests (~12 new)

---

## Consequences

### Positive

- Full traceability: spec -> PR -> test -> deploy queryable as single chain
- Compliance-ready: audit trail PDF (ADR-008) consumes evidence chain directly
- Zero future migrations needed for new artifact types (test_run, deploy)
- RLS-safe: tenant isolation on evidence links matches existing pattern

### Negative

- Polymorphic `from_id`/`to_id` lacks database-level referential integrity
- Additional JOIN in chain queries (mitigated by indexes)

### Future Extensions (No Schema Change Required)

| from_type | to_type | link_reason | Sprint |
|-----------|---------|-------------|--------|
| spec | pr_gate | auto_spec_review | 11 |
| pr_gate | test_run | auto_ci_result | 12 |
| test_run | deploy | auto_deploy_trigger | 12 |
| spec | spec | revision | 12+ |

---

## References

| Document | Location |
|----------|----------|
| Sprint 11 Plan | `docs/04-build/sprints/SPRINT-011-Hardening.md` |
| ADR-002 (Three-System Architecture) | `docs/02-design/01-ADRs/SPEC-0002-ADR-002-Three-System-Architecture.md` |
| ADR-008 (PDF Library) | `docs/02-design/01-ADRs/SPEC-0008-ADR-008-PDF-Library.md` |
| Migration 000015 (pr_gate_evaluations) | `migrations/000015_pr_gate_evaluations.up.sql` |
