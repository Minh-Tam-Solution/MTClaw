# SPRINT-024 COMPLETION: Provider Fallback Chain + Claude CLI Provider

**Status**: COMPLETE
**Duration**: 5 days (actual)
**Sprint Plan**: [SPRINT-024-Provider-Fallback.md](SPRINT-024-Provider-Fallback.md)
**ADR**: SPEC-0014-ADR-014-Provider-Fallback-Claude-CLI (APPROVED WITH CONDITIONS)
**Commit**: acf317e (feat(sprint-13-23): bundled with rebrand commit)

---

## Deliverables

### Delivered

| Task | File | Status |
|------|------|--------|
| T1 | `internal/providers/claude_cli.go` (6,286 bytes) | COMPLETE |
| T1a | `Chat()`: subprocess spawn, JSON parse, env strip | COMPLETE |
| T1b | `ChatStream()`: delegates to `Chat()` + single chunk (CTO-500/502) | COMPLETE |
| T1c | `filterEnv()`: strips ANTHROPIC_API_KEY + CLAUDE_API_KEY | COMPLETE |
| T1d | CLI flags: `-p --output-format json --model sonnet --max-turns 1` | COMPLETE |
| T2 | `internal/providers/claude_cli_test.go` (5,521 bytes) — 11+ tests | COMPLETE |
| T3 | `internal/config/config.go` — `ClaudeCLI` + `ProviderChain` config | COMPLETE |
| T4 | `internal/config/config_load.go` — env var loading | COMPLETE |
| T5 | `cmd/gateway_providers.go` — register claude-cli provider | COMPLETE |
| T6 | `internal/agent/loop.go` — fallback on retryable error | COMPLETE |
| T6a | `iteration > 1` guard (CTO-R2-1) | COMPLETE |
| T6b | Always strip tools on fallback (CTO-501) | COMPLETE |
| T7 | `internal/agent/resolver.go` — wire fallback from config | COMPLETE |
| T8 | `cmd/doctor.go` — `which claude` + version check | COMPLETE |
| T9 | `.env.example` — documented new env vars | COMPLETE |

### CTO Review Conditions Met

| Condition | Implementation |
|-----------|---------------|
| CTO-R2-1: No fallback at iteration=1 with tools | Guard in `loop.go`: `iteration == 1 && len(chatReq.Tools) > 0` → skip fallback |
| CTO-R2-2: Env sanitization | `filterEnv()` strips ANTHROPIC_API_KEY, CLAUDE_API_KEY |
| CTO-500: Fallback on both Chat + ChatStream | Both paths trigger fallback on retryable error |
| CTO-501: Always strip tools on fallback | `fallbackReq.Tools = nil` — no type assertion |
| CTO-502: ChatStream delegates to Chat | `ChatStream()` calls `Chat()` + wraps as single chunk |
| CTO-R2-5: Doctor check | `which claude` + version output in doctor command |

### Test Coverage

- 11 unit tests in `claude_cli_test.go`
- 5 tests in `internal/agent/fallback_test.go`
- All existing tests pass (no regression)

## Deviations from Plan

| Deviation | Reason |
|-----------|--------|
| Committed as part of larger bundle (acf317e) | Sprint 13-23 rebrand bundled with Sprint 24 code |
| E2E tests deferred to Sprint 25 | Scope split: Sprint 24 = provider + logic, Sprint 25 = Docker + E2E |

## Acceptance Criteria

- [x] `claude-cli` provider registered and functional
- [x] `ChatStream` delegates to `Chat` + single chunk (CTO-500/502)
- [x] Subprocess env strips `ANTHROPIC_API_KEY` (CTO-R2-2)
- [x] Fallback triggers automatically on retryable errors (both Chat + ChatStream)
- [x] Fallback ONLY when `iteration > 1` with tools (CTO-R2-1)
- [x] Tools always stripped on fallback (CTO-501)
- [x] All existing tests pass (no regression)
- [x] 11+ new tests for claude-cli provider + fallback logic
- [x] `doctor` command checks claude CLI availability (CTO-R2-5)

---

## Coder Handoff Notes

### Key Files Modified

| File | LOC | What Changed |
|------|-----|-------------|
| `internal/providers/claude_cli.go` | 189 | New provider: subprocess spawn, JSON parse, env strip |
| `internal/providers/claude_cli_test.go` | 165 | 11 tests: parse response, env filter, error cases |
| `internal/agent/loop.go` | +45 | Fallback logic: retryable error → fallback provider, strip tools |
| `internal/agent/resolver.go` | +20 | Wire `FallbackProvider` from `ProviderChain` config |
| `internal/config/config.go` | +15 | `ClaudeCLI` + `ProviderChain` config structs |
| `internal/config/config_load.go` | +12 | Env var loading for `MTCLAW_CLAUDE_CLI_*` |
| `cmd/gateway_providers.go` | +10 | Register `claude-cli` in provider registry |
| `cmd/doctor.go` | +25 | Claude CLI binary + version + OAuth check |

### Architecture Decisions

- **Subprocess model**: Claude CLI runs as child process (`os/exec.Command`), not SDK. Avoids Go SDK dependency, uses official CLI binary.
- **Single-turn only**: `--max-turns 1` prevents CLI from using its own tools. MTClaw controls the tool loop.
- **Env sanitization**: `filterEnv()` strips API keys from child process environment to prevent credential leakage.
- **Fallback guard**: No fallback at iteration=1 with tools — prevents partial tool calls from confusing fallback provider.

### Integration Points for Sprint 25+

- `loop.go` fallback path emits `emitFallbackLLMSpan()` — tracing already captures primary error + fallback provider
- `resolver.go` reads `ProviderChain` from config — same chain used for all agents (per-agent chain override not implemented yet)
- Docker deployment needs `claude` binary + OAuth token (`~/.claude/`) — addressed in Sprint 25
