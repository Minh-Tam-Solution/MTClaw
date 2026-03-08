# SPRINT-025 COMPLETION: Fallback Deploy + Observability + E2E

**Status**: COMPLETE
**Duration**: 4 days (actual)
**Sprint Plan**: [SPRINT-025-Fallback-Deploy-Observability.md](SPRINT-025-Fallback-Deploy-Observability.md)
**Depends on**: Sprint 24 (Provider Fallback Chain — COMPLETE)
**Commit**: acf317e (bundled), subsequent working tree changes

---

## Deliverables

### Delivered

| Task | File | Status |
|------|------|--------|
| T1 | `Dockerfile` — conditional `npm install @anthropic-ai/claude-code` (gated by `ENABLE_CLAUDE_CLI`) | COMPLETE |
| T2 | `docker-compose.yml` — `MTCLAW_CLAUDE_*` + `MTCLAW_PROVIDER_CHAIN` env vars | COMPLETE |
| T3 | `docker-compose.yml` — `claude-oauth:/app/.claude` volume | COMPLETE |
| T4 | `docker-compose.yml` — `/tmp` tmpfs without `noexec` | COMPLETE |
| T9-T12 | `internal/agent/loop_tracing.go` — 2-span fallback tracing pattern | COMPLETE |
| T10 | `emitFallbackLLMSpan()` tags: `fallback=true`, `primary_provider`, `primary_error` | COMPLETE |
| T12 | `internal/tracing/otelexport/exporter.go` — metadata propagated to OTEL export | COMPLETE |

### CTO B-series Resolutions

| Issue | Resolution | Status |
|-------|-----------|--------|
| B1: glibc/musl compat | Option B: npm install in Dockerfile (conditional on `ENABLE_CLAUDE_CLI` build arg) | RESOLVED |
| B2: OAuth token persistence | `claude-oauth` Docker volume at `/app/.claude` | RESOLVED |
| B3: Primary fail invisible in traces | 2-span emission: primary fail span + fallback success span with metadata | RESOLVED |

### Test Coverage

- Integration tests: INT-052 through INT-058 (7 tests) — ALL PASSING
- Security tests: SEC-021 (env isolation), SEC-022 (Docker hardening) — PASSING
- E2E tests: E2E-020, E2E-021 — MANUAL only (not automated)

## Deviations from Plan

| Deviation | Reason |
|-----------|--------|
| T5-T8: One-time OAuth setup | Done manually during Sprint 25.5 testing |
| T19: Doctor OAuth check | Implemented as part of existing doctor command |
| T20: Grafana fallback alert | NOT DELIVERED — Prometheus/Grafana not in production stack |
| T21: Deployment runbook | NOT DELIVERED — deferred to Sprint 28 |
| E2E tests not automated | E2E-020, E2E-021 remain MANUAL in MASTER-TEST-PLAN |

## Sprint 25.5 Bug Fixes (in working tree, not yet committed)

During manual testing after Sprint 25, two bugs were found and fixed:

| Bug | File | Fix |
|-----|------|-----|
| Admission config not wired | `cmd/gateway.go` | Use `DefaultBridgeConfig()` as base, merge `config.Bridge.Admission` map values |
| `/cc sessions` shows stopped sessions | `internal/channels/telegram/commands_cc.go` | Filter `SessionStateStopped` from active session list |

These fixes are in the working tree awaiting commit in Sprint 26 (T26.5).

## Acceptance Criteria

- [x] `docker compose up -d --build` with Claude CLI installed via npm in Alpine image
- [x] `claude --version` works inside container
- [x] OAuth token persisted in `claude-oauth` volume
- [x] `mtclaw doctor` shows Claude CLI binary + version + model
- [x] Primary fail → fallback success emits 2 tracing spans
- [x] All existing tests pass (no regression)
- [ ] ~~E2E automated tests~~ — Deferred: E2E-020, E2E-021 remain MANUAL (Sprint 28)
- [ ] ~~Grafana alert on fallback rate~~ — Deferred: no Prometheus in stack (Sprint 28)
- [ ] ~~Deployment runbook~~ — Deferred to Sprint 28

## Outstanding Debt

| Item | Planned Sprint | Status |
|------|---------------|--------|
| E2E test automation (E2E-020, E2E-021) | Sprint 28 (T28.3) | PLANNED |
| Deployment runbook (bridge + fallback) | Sprint 28 (T28.4) | PLANNED |
| Grafana/metrics alerting | Sprint 28 (T28.2) | PLANNED |
