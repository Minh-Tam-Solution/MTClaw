# ADR-003: Observability Architecture

**SPEC ID**: SPEC-0003
**Status**: ACCEPTED
**Date**: 2026-03-02
**Deciders**: [@cto], [@architect]

---

## Context

MTClaw is multi-tenant from Day 1. Every trace, metric, and log must include tenant context. Cost guardrails are critical for OaaS viability.

## Decision

### Trace ID Format

```
{tenant_id}-{session_id}-{ulid}
```

Propagated via Go `context.Context` through all layers:
```go
ctx = context.WithValue(ctx, "trace_id", traceID)
ctx = context.WithValue(ctx, "tenant_id", tenantID)
```

### Logging

- **Library**: Go `slog` (structured JSON)
- **Required fields**: `trace_id`, `tenant_id`, `soul_role`, `level`, `msg`, `timestamp`
- **Format**: JSON in production, text in development

```go
slog.InfoContext(ctx, "soul invoked",
    "tenant_id", tenantID,
    "soul", "pm",
    "action", "/spec",
    "trace_id", traceID,
)
```

### Metrics (OTEL → Prometheus)

| Metric | Type | Labels |
|--------|------|--------|
| `mtclaw_request_total` | Counter | tenant_id, soul, status |
| `mtclaw_request_duration_seconds` | Histogram | tenant_id, soul, endpoint |
| `mtclaw_token_usage_total` | Counter | tenant_id, soul, provider |
| `mtclaw_active_sessions` | Gauge | tenant_id |

### Token Cost Tracking

```sql
CREATE TABLE token_usage (
    id          BIGSERIAL PRIMARY KEY,
    tenant_id   UUID NOT NULL REFERENCES tenants(id),
    soul_role   TEXT NOT NULL,
    provider    TEXT NOT NULL,
    prompt_tokens   INT NOT NULL,
    completion_tokens INT NOT NULL,
    total_tokens    INT NOT NULL,
    cost_usd    DECIMAL(10,6),
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_token_usage_tenant_month ON token_usage (tenant_id, date_trunc('month', created_at));
```

### Tenant Cost Guardrails

| Guardrail | Default | Behavior |
|-----------|---------|----------|
| `tenant_monthly_token_limit` | 1,000,000 tokens | Warn at 80%, degrade at 100% |
| `tenant_daily_request_limit` | 5,000 requests | Warn at 80%, reject at 100% |

**Degradation behavior** at 100%:
- Non-critical requests: Return cached/fallback response
- Critical requests (governance rails): Allow with alert to tenant admin
- All requests: Log `cost_guardrail_triggered` metric

### Sprint 1 Implementation

- ADR documented (this file)
- `slog` structured logging in all new code
- `trace_id` in context propagation
- `token_usage` table in migrations

### Sprint 3 Full Implementation

- OTEL exporter → Prometheus
- Grafana dashboards (tenant cost, SOUL usage, latency)
- Alerting (cost threshold, error rate)

## Consequences

### Positive
- Full tenant cost visibility from Day 1
- Prevent runaway costs before OaaS launch
- Structured logs enable debugging across tenants
- OTEL standard = portable to any observability backend

### Negative
- Context propagation adds ~1ms overhead per request
- Token counting requires Bflow AI-Platform to return usage data
- Guardrail enforcement adds complexity to request pipeline

---

## References
- OTEL Go SDK: `go.opentelemetry.io/otel`
- GoClaw existing OTEL: `internal/tracing/`
- Bflow AI-Platform token response format: OpenAI-compatible `usage` field
