# SPRINT-025: Fallback Deploy + Observability + E2E (Revised)

**Status**: PLANNED (CTO-revised)
**Duration**: 3-4 days
**Depends on**: Sprint 24 (Provider Fallback Chain — COMPLETE)
**Priority**: P0 (Deploy Sprint 24 code to production Docker)

---

## Objective

Deploy the Claude CLI fallback provider (Sprint 24) to Docker, add observability (2-span fallback tracing), validate with E2E tests, and ensure the bot responds within 30s even when Ollama is down.

## Context

- Sprint 24 code is merged: `ClaudeCLIProvider`, fallback chain in `loop.go`, resolver wiring, doctor check
- Docker image is Alpine 3.22 — host binary mount **will NOT work** (glibc vs musl)
- Container is `read_only: true` + `cap_drop: ALL` + `no-new-privileges: true`
- Claude CLI needs OAuth token persistence across restarts
- Tracing: primary fail span + fallback span (2-span pattern) fixed in S24 hotfix

## CTO Review Resolutions

| CTO Issue | Resolution |
|-----------|-----------|
| **B1**: Host volume mount won't work (glibc/musl) | **Option B only**: Install claude CLI via npm in Dockerfile |
| **B2**: OAuth token lost on container restart | Docker volume `claude-oauth` mounted at `/app/.claude` |
| **B3**: Primary fail invisible in traces | **FIXED in S24 hotfix**: 2-span emission (primary fail + fallback success) |

---

## Deliverables

### Day 1: Dockerfile + OAuth + Docker-Compose

| Task | File | Description |
|------|------|-------------|
| T1 | `Dockerfile` | Add `nodejs npm` to apk, `npm install -g @anthropic-ai/claude-code`, create `/app/.claude/` |
| T2 | `docker-compose.yml` | Add `MTCLAW_CLAUDE_*` + `MTCLAW_PROVIDER_CHAIN` env vars |
| T3 | `docker-compose.yml` | Add `claude-oauth:/app/.claude` volume for OAuth token persistence |
| T4 | `docker-compose.yml` | Change tmpfs `/tmp` to remove `noexec` (Claude CLI needs exec in tmp) |
| T5 | `.env` (staging) | Add `MTCLAW_CLAUDE_ENABLED=true`, `MTCLAW_PROVIDER_CHAIN=bflow-ai-platform,claude-cli` |
| T6 | — | Deploy: `docker compose up -d --build` |
| T7 | — | One-time: `docker exec -it mtclaw-mtclaw-1 claude login` (OAuth setup) |
| T8 | — | Verify: `docker exec mtclaw-mtclaw-1 /app/mtclaw doctor` shows Claude CLI |

#### Dockerfile Changes (T1)

```dockerfile
# Stage 2: Runtime — add after existing apk line
RUN set -eux; \
    apk add --no-cache ca-certificates wget nodejs npm; \
    npm install -g @anthropic-ai/claude-code; \
    npm cache clean --force; \
    if [ "$ENABLE_SANDBOX" = "true" ]; then \
        apk add --no-cache docker-cli; \
    fi

# Add after mkdir line
RUN mkdir -p /app/.claude && chown mtclaw:mtclaw /app/.claude
```

#### Docker-compose Changes (T2-T4)

```yaml
environment:
  # Claude CLI fallback provider
  - MTCLAW_CLAUDE_ENABLED=${MTCLAW_CLAUDE_ENABLED:-false}
  - MTCLAW_CLAUDE_PATH=${MTCLAW_CLAUDE_PATH:-/usr/local/bin/claude}
  - MTCLAW_CLAUDE_MODEL=${MTCLAW_CLAUDE_MODEL:-sonnet}
  - MTCLAW_CLAUDE_TIMEOUT=${MTCLAW_CLAUDE_TIMEOUT:-120}
  - MTCLAW_PROVIDER_CHAIN=${MTCLAW_PROVIDER_CHAIN:-}

volumes:
  - claude-oauth:/app/.claude  # OAuth token persistence

tmpfs:
  - /tmp:rw,nosuid,size=256m  # removed noexec — Claude CLI needs it
```

#### Day 1 Pre-flight Checklist

```bash
# On host (before deploy)
docker compose config | grep CLAUDE       # verify env vars resolved
docker compose build --no-cache mtclaw    # rebuild with npm + claude
docker compose up -d
docker exec mtclaw-mtclaw-1 claude --version  # verify claude binary works
docker exec -it mtclaw-mtclaw-1 claude login  # one-time OAuth setup
docker exec mtclaw-mtclaw-1 /app/mtclaw doctor  # verify all green
```

### Day 2: Observability — Fallback Tracing Tags

| Task | File | Description |
|------|------|-------------|
| T9 | `internal/store/types.go` | Add `Metadata map[string]string` to `SpanData` if not exists |
| T10 | `internal/agent/loop_tracing.go` | New `emitFallbackLLMSpan()` that tags spans with `fallback=true`, `primary_provider`, `primary_error` |
| T11 | `internal/agent/loop.go` | Replace raw `emitLLMSpan` calls in fallback block with `emitFallbackLLMSpan` |
| T12 | `internal/tracing/collector.go` | Ensure metadata propagated to OTEL export |

#### Tracing Design (2-span pattern)

When fallback fires, 2 spans are emitted:
1. **Primary fail span**: `bflow-ai-platform/qwen3:14b #3` — status=error, error=timeout
2. **Fallback success span**: `claude-cli/sonnet #3 [fallback]` — status=completed, metadata: `{"fallback":"true","primary_provider":"bflow-ai-platform","primary_error":"timeout after 30s"}`

When NO fallback: 1 span as before (no changes).

### Day 3: E2E Validation Tests

| Task | File | Description |
|------|------|-------------|
| T13 | `internal/integration/fallback_e2e_test.go` | E2E: primary timeout → fallback succeeds → response delivered |
| T14 | — | E2E: primary OK → no fallback (happy path, 1 span) |
| T15 | — | E2E: iteration=1 with tools → primary fails → error (no fallback) |
| T16 | — | E2E: both providers fail → error propagated (primary error, not fallback error) |
| T17 | — | Unit: verify 2-span emission on fallback path |
| T18 | — | Manual: Telegram test — stop Ollama → send message → verify response via Claude CLI |
| T18a | — | Measure: end-to-end Telegram response time with Ollama stopped. Record in sprint report. |

#### E2E Test Approach

Tests use stub providers (not real Claude CLI) to validate loop.go fallback logic.
Build tag: `//go:build integration` to keep `make test` fast.

```go
//go:build integration

package integration

// stubFailProvider simulates primary provider failure
type stubFailProvider struct{ err error }

// stubSuccessProvider simulates fallback success
type stubSuccessProvider struct{ content string }
```

### Day 4 (if needed): Hardening

| Task | Description |
|------|-------------|
| T19 | Doctor: verify OAuth token exists in `/app/.claude/` — warn if missing |
| T20 | Monitoring: Grafana alert on fallback span count > 10/hour |
| T21 | Documentation: Deployment runbook — Claude CLI setup, OAuth refresh, troubleshooting |

---

## Success Criteria

- [ ] `docker compose up -d --build` with Claude CLI installed via npm in Alpine image
- [ ] `claude --version` works inside container
- [ ] `claude login` completes OAuth setup, token persisted in `claude-oauth` volume
- [ ] `mtclaw doctor` inside container shows Claude CLI binary + version + model
- [ ] Primary fail → fallback success emits 2 tracing spans (primary error + fallback success)
- [ ] E2E: primary timeout → fallback response within 30s
- [ ] E2E: iteration=1 with tools → error (no fallback)
- [ ] E2E: happy path → 1 span, no fallback
- [ ] Manual Telegram test: Ollama stopped → bot responds via Claude CLI
- [ ] Response time measured and recorded (T18a)
- [ ] All existing tests pass (no regression)

## Dependencies

- Node.js + npm available in Alpine (via `apk add nodejs npm`)
- `@anthropic-ai/claude-code` npm package compatible with Alpine/musl
- Claude Max OAuth session completed via `claude login` inside container
- Docker host has access to both `ai-net` and `bflow-staging-network`

## Risks

| Risk | Severity | Mitigation |
|------|----------|-----------|
| npm package `@anthropic-ai/claude-code` has native deps that fail on Alpine musl | HIGH | Test Day 1 first thing. Fallback: use `node:alpine` multi-stage build |
| OAuth token expires, no auto-refresh | MEDIUM | Document token refresh in runbook. Doctor check warns if token missing |
| Image size increases ~150-200MB (node + npm + claude) | LOW | Acceptable for fallback reliability |
| `/tmp` without `noexec` slightly reduces container hardening | LOW | Acceptable: only Claude CLI subprocess uses tmp, container is still read-only otherwise |
| Claude CLI startup overhead (~2-3s per request) | LOW | Only fires on fallback; primary path unaffected |

---

## Sprint 25 vs Sprint 24 Boundary

| Sprint 24 (DONE) | Sprint 25 (THIS) |
|-------------------|-------------------|
| Provider code (`claude_cli.go`) | Dockerfile: install claude via npm |
| Fallback logic (`loop.go`) | Docker-compose: env vars + volumes |
| Config + env loading | OAuth setup + token persistence |
| Doctor check code | Doctor OAuth token validation |
| Unit tests (17) | E2E tests (5-6) + build tag |
| `.env.example` docs | Production runbook |
| 2-span tracing (S24 hotfix) | Fallback metadata tags (provider, error) |

## PJM Review Resolutions

| PJM Issue | Resolution |
|-----------|-----------|
| PJM-025-C1: Day 1 go/no-go gate | npm install in Dockerfile (not host mount) eliminates glibc risk. New risk: npm package Alpine compat — test immediately Day 1 |
| PJM-025-C2: T7 may be no-op | Removed. Provider name already in span via `l.provider.Name()` |
| PJM-025-C3: E2E test build tag | Added `//go:build integration` requirement |
| PJM-025-C4: Latency measurement | Added T18a: measure + record in sprint report |
