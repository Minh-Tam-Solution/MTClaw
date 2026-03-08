# SPRINT-024: Provider Fallback Chain + Claude CLI Provider

**Status**: PLANNED
**Duration**: 3-5 days
**ADR**: SPEC-0014-ADR-014-Provider-Fallback-Claude-CLI
**Priority**: P0 (Gateway reliability)

---

## Objective

When Ollama (primary) is slow/unavailable, automatically fallback to Claude CLI (Claude Max $200 subscription) so users always get a response.

## Context

- Ollama running on CPU (no GPU) → 60s+ per request → timeout
- MTS team pays $200/month Claude Max — currently unused
- EndiorBot has mature fallback patterns to reference

## Deliverables

### Day 1-2: Claude CLI Provider (CTO-adjusted)

| Task | File | Description |
|------|------|-------------|
| T1 | `internal/providers/claude_cli.go` | Provider implementation (~250 lines) |
| T1a | — | `Chat()`: subprocess spawn, JSON parse, env strip (CTO-R2-2) |
| T1b | — | `ChatStream()`: delegate to Chat + single chunk (CTO-500/502) |
| T1c | — | `filterEnv()`: strip ANTHROPIC_API_KEY from subprocess env |
| T1d | — | CLI flags: `-p --output-format json --model sonnet --max-turns 1` (CTO-503/504) |
| T2 | `internal/providers/claude_cli_test.go` | Unit tests (11+ tests) |
| T3 | `internal/config/config.go` | Add `ClaudeCLI` + `ProviderChain` config (with `Enabled` flag) |
| T4 | `internal/config/config_load.go` | Load `MTCLAW_CLAUDE_*` + `MTCLAW_PROVIDER_CHAIN` env vars |
| T5 | `cmd/gateway_providers.go` | Register claude-cli provider |

### Day 3: Fallback Chain (CTO-adjusted)

| Task | File | Description |
|------|------|-------------|
| T6 | `internal/agent/loop.go` | Fallback on retryable error — BOTH Chat + ChatStream paths (CTO-500) |
| T6a | — | `iteration > 1` guard: no text-only fallback at iteration 1 with tools (CTO-R2-1) |
| T6b | — | Always strip tools on fallback — no type assertion (CTO-501) |
| T7 | `internal/agent/resolver.go` | Wire fallback from provider chain |
| T8 | `cmd/doctor.go` | Add `which claude` + version check (CTO-R2-5) |
| T9 | `.env.example` | Document new env vars |

### Day 4-5: Integration + E2E

| Task | Description |
|------|-------------|
| T10 | docker-compose: add `MTCLAW_CLAUDE_*` env vars + volume mount `~/.claude/` |
| T11 | E2E: Telegram test with Ollama down → Claude fallback |
| T12 | E2E: Telegram test with Ollama up → direct response |
| T13 | E2E: iteration=1 fail with tools → error (no wrong text-only answer) |
| T14 | Tracing: fallback span tagged `provider=claude-cli, fallback=true` (CTO-R2-6) |

## Implementation Notes

### Claude CLI invocation (CTO-503/504 corrected)
```bash
claude -p --output-format json --model sonnet --max-turns 1 "prompt"
```

### Environment (force OAuth, no API key)
```go
env = removeEnv(env, "ANTHROPIC_API_KEY")
env = removeEnv(env, "CLAUDECODE")
```

### Fallback trigger (loop.go — CTO-adjusted)
```go
if err != nil && l.fallbackProvider != nil && providers.IsRetryableError(err) {
    // CTO: Only text-only fallback when iteration > 1 (tools already ran)
    if iteration == 1 && len(chatReq.Tools) > 0 {
        // Don't fallback — tools needed but not yet executed
    } else {
        fallbackReq := chatReq
        fallbackReq.Tools = nil  // CTO-501: always strip, no type assertion
        resp, err = l.fallbackProvider.Chat(ctx, fallbackReq)  // Always Chat, not ChatStream
    }
}
```

### Config (.env)
```bash
MTCLAW_PROVIDER_CHAIN=bflow-ai-platform,claude-cli
MTCLAW_CLAUDE_PATH=/home/dttai/.local/bin/claude
MTCLAW_CLAUDE_MODEL=sonnet
MTCLAW_CLAUDE_TIMEOUT=120
```

## Success Criteria

- [ ] `claude-cli` provider registered and functional
- [ ] `ChatStream` delegates to `Chat` + single chunk (CTO-500/502)
- [ ] Subprocess env strips `ANTHROPIC_API_KEY` (CTO-R2-2)
- [ ] Fallback triggers automatically on Ollama timeout/error (both Chat + ChatStream paths)
- [ ] Fallback ONLY when `iteration > 1` with tools (CTO-R2-1)
- [ ] Tools always stripped on fallback — no type assertion (CTO-501)
- [ ] Telegram users get response within 30s (even with Ollama down)
- [ ] All existing tests pass (no regression)
- [ ] 11+ new tests for claude-cli provider + fallback logic
- [ ] `doctor` command checks claude CLI availability (CTO-R2-5)

## Dependencies

- `claude` CLI v2.1.71+ installed on host
- Claude Max $200 subscription active (OAuth login)
- Docker image updated with claude binary (or host mount)

## Risks

| Risk | Mitigation |
|------|-----------|
| Claude CLI startup overhead (~2-3s) | Acceptable for fallback; primary path unaffected |
| No tool calls in fallback | Fallback is for text response; tools run on primary attempts |
| Claude Max quota exhaustion | $200/month plan has generous limits; monitor usage |
