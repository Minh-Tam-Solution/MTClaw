---
title: Performance Baseline Report — Sprint 11
version: 1.0.0
date: 2026-03-06
sprint: 11
owner: "[@coder]"
status: TEMPLATE — awaiting live measurements
---

# Performance Baseline Report — Sprint 11

## Purpose

Establish performance baselines for Sprint 11 features (evidence linking, PR Gate, audit trail PDF). Not optimization — measurement only. Results guide Sprint 12 OaaS infrastructure decisions.

## Environment

| Component | Spec |
|-----------|------|
| Gateway | GoClaw (Go, managed mode) |
| Database | PostgreSQL 16, RLS enabled |
| Server | Production VPS (TBD: record CPU/RAM) |
| AI-Platform | `api.nhatquangholding.com` (Ollama qwen3:14b backend) |

## API Latency Benchmarks

### Run instructions

```bash
# Install hey (HTTP benchmarking)
go install github.com/rakyll/hey@latest

# 1. Spec listing (Rail #1)
hey -n 1000 -c 10 -H "Authorization: Bearer $GATEWAY_TOKEN" \
    http://localhost:18790/v1/evidence/export?rail=spec-factory&format=json

# 2. Evidence export (JSON)
hey -n 500 -c 10 -H "Authorization: Bearer $GATEWAY_TOKEN" \
    http://localhost:18790/v1/evidence/export?format=json

# 3. Evidence export (CSV)
hey -n 500 -c 10 -H "Authorization: Bearer $GATEWAY_TOKEN" \
    http://localhost:18790/v1/evidence/export?format=csv

# 4. Audit trail PDF generation
hey -n 100 -c 5 -H "Authorization: Bearer $GATEWAY_TOKEN" \
    http://localhost:18790/v1/evidence/audit-trail.pdf?specId=SPEC-2026-0001
```

### Results

| Endpoint | p50 | p95 | p99 | Target | Status |
|----------|-----|-----|-----|--------|--------|
| GET /v1/evidence/export (JSON, 20 specs) | _TBD_ | _TBD_ | _TBD_ | <200ms | PENDING |
| GET /v1/evidence/export (CSV) | _TBD_ | _TBD_ | _TBD_ | <200ms | PENDING |
| GET /v1/evidence/audit-trail.pdf | _TBD_ | _TBD_ | _TBD_ | <500ms | PENDING |
| WebSocket JSON-RPC evidence.chain | _TBD_ | _TBD_ | _TBD_ | <100ms | PENDING |
| WebSocket JSON-RPC evidence.link | _TBD_ | _TBD_ | _TBD_ | <100ms | PENDING |

---

## Database Query Benchmarks

### Run instructions

```sql
-- Connect with tenant context
SET LOCAL app.tenant_id = '<tenant_id>';

-- 1. governance_specs list (owner_id + status filter)
EXPLAIN ANALYZE
SELECT * FROM governance_specs
WHERE owner_id = current_setting('app.tenant_id', true)
  AND status = 'approved'
ORDER BY created_at DESC LIMIT 20;

-- 2. pr_gate_evaluations by repo + pr_number
EXPLAIN ANALYZE
SELECT * FROM pr_gate_evaluations
WHERE owner_id = current_setting('app.tenant_id', true)
  AND repo = 'org/repo'
ORDER BY created_at DESC LIMIT 10;

-- 3. evidence_links chain query (2-hop join)
EXPLAIN ANALYZE
SELECT el.*, gs.spec_id, gs.title, gs.status,
       pge.verdict, pge.pr_url
FROM evidence_links el
LEFT JOIN governance_specs gs
    ON el.to_type = 'spec' AND el.to_id = gs.id
LEFT JOIN pr_gate_evaluations pge
    ON el.to_type = 'pr_gate' AND el.to_id = pge.id
WHERE el.owner_id = current_setting('app.tenant_id', true)
  AND el.from_id = '<spec_uuid>';

-- 4. traces cost aggregation (daily limit check)
EXPLAIN ANALYZE
SELECT COUNT(*) FROM traces
WHERE owner_id = current_setting('app.tenant_id', true)
  AND created_at >= CURRENT_DATE;

-- 5. SOUL loading (agent_context_files)
EXPLAIN ANALYZE
SELECT acf.* FROM agent_context_files acf
JOIN agents a ON acf.agent_id = a.id
WHERE a.owner_id = current_setting('app.tenant_id', true)
  AND a.agent_key = 'cto';
```

### Results

| Query | Execution Time | Rows | Seq Scan? | Target | Status |
|-------|---------------|------|-----------|--------|--------|
| 1. governance_specs list | _TBD_ | _TBD_ | _TBD_ | <10ms | PENDING |
| 2. pr_gate_evaluations by repo | _TBD_ | _TBD_ | _TBD_ | <10ms | PENDING |
| 3. evidence_links chain (2-hop) | _TBD_ | _TBD_ | _TBD_ | <50ms | PENDING |
| 4. traces cost aggregation | _TBD_ | _TBD_ | _TBD_ | <20ms | PENDING |
| 5. SOUL loading | _TBD_ | _TBD_ | _TBD_ | <10ms | PENDING |

### Index coverage

```sql
-- Verify indexes exist for critical queries
SELECT tablename, indexname FROM pg_indexes
WHERE schemaname = 'public'
  AND tablename IN ('governance_specs', 'pr_gate_evaluations', 'evidence_links', 'traces', 'agent_context_files')
ORDER BY tablename, indexname;
```

---

## RAG Latency (AI-Platform)

### Run instructions

```bash
# 10 concurrent RAG queries against Bflow AI-Platform
for i in $(seq 1 10); do
  curl -w "%{time_total}\n" -o /dev/null -s \
    -X POST https://api.nhatquangholding.com/api/v1/rag/query \
    -H "X-API-Key: $BFLOW_AI_API_KEY" \
    -H "X-Tenant-ID: mts" \
    -H "Content-Type: application/json" \
    -d '{"query":"How to deploy Bflow POS module?","collection":"mts-engineering","top_k":5}' &
done
wait
```

### Results

| Metric | Value | Target | Status |
|--------|-------|--------|--------|
| RAG query p50 | _TBD_ | <1000ms | PENDING |
| RAG query p95 | _TBD_ | <3000ms | PENDING |
| RAG query p99 | _TBD_ | <5000ms | PENDING |

---

## PDF Generation Benchmark

### Run instructions

```bash
# Measure PDF generation for specs with varying chain lengths
# Small chain (1 node): spec only
# Medium chain (3 nodes): spec + pr_gate + test_run
# Large chain (10+ nodes): spec + multiple pr_gates + test_runs

# Unit test benchmark (if Go available):
go test ./internal/audit/ -bench=. -benchmem -count=3
```

### Results

| Chain Size | Generation Time | PDF Size | Target | Status |
|------------|----------------|----------|--------|--------|
| 1 node (spec only) | _TBD_ | _TBD_ | <100ms | PENDING |
| 3 nodes (typical) | _TBD_ | _TBD_ | <200ms | PENDING |
| 10+ nodes (large) | _TBD_ | _TBD_ | <500ms | PENDING |

---

## Findings & CTO Issues

| Finding | Severity | CTO Issue | Sprint |
|---------|----------|-----------|--------|
| _(none yet — populate after measurements)_ | | | |

**Rule**: Any metric exceeding its target gets filed as a CTO issue for Sprint 12.

---

## Run Checklist

```bash
# Pre-flight
[ ] Gateway running in managed mode (localhost:18790)
[ ] PostgreSQL 16 accessible with RLS enabled
[ ] At least 1 governance_spec + 1 pr_gate_evaluation in DB
[ ] hey installed (go install github.com/rakyll/hey@latest)
[ ] GATEWAY_TOKEN and BFLOW_AI_API_KEY set

# Execute
[ ] API latency benchmarks (hey)
[ ] DB EXPLAIN ANALYZE (5 queries)
[ ] RAG latency (10 concurrent)
[ ] PDF generation benchmark
[ ] Record results in tables above
[ ] File CTO issues for any exceeded targets
```
