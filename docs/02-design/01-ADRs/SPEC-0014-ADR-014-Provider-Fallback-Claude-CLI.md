# SPEC-0014 / ADR-014: Provider Fallback Chain + Claude CLI Provider

**Status**: APPROVED WITH CONDITIONS → CONDITIONS MET
**Date**: 2026-03-08
**Authors**: PM + Architect
**Sprint**: 24 — Provider Fallback
**References**: EndiorBot `claude-code-bridge.ts`, ADR-013 Provider Persona Projection
**CTO Review**: 8.5/10 — APPROVED, 7 issues resolved (CTO-500 through CTO-507)

---

## 1. Problem Statement

MTClaw gateway relies on a **single LLM provider path**: Bflow AI-Platform → Ollama (local).
When Ollama is slow (CPU-only, no GPU) or unavailable, the entire gateway becomes unresponsive — users on Telegram see "Provider busy, retrying..." or "Sorry, something went wrong."

**Current failure chain:**
```
User message → Agent Loop → bflow-ai-platform → Ollama (CPU) → TIMEOUT → error
                                                                  ↓
                                                          No fallback → user error
```

**Root causes observed (2026-03-08):**
- Ollama running on CPU (`size_vram: 0`) — qwen3:14b takes >60s per request
- Bflow AI-Platform only routes to Ollama, no cloud fallback
- Agent loop retries same provider 3x → 3× timeout = user waits 90s+ for error

**Business impact:**
- MTS team pays $200/month for Claude Max plan — completely unused
- Bot reliability <50% when GPU unavailable
- No cost optimization: expensive local inference on CPU vs free Claude Max quota

## 2. Decision

### 2.1 Provider Fallback Chain (Loop Level)

Implement **provider-level fallback** in the agent loop, inspired by EndiorBot's `ResourceRouter`:

```
Provider Chain (ordered by priority):
  1. bflow-ai-platform (Ollama local) — FREE, fast when GPU available
  2. claude-cli (Claude Max subscription) — FREE* ($200/month plan), reliable cloud

  * Falls within existing Claude Max $200 subscription, no per-token cost
```

**Fallback trigger**: When primary provider returns retryable error OR exceeds timeout threshold.

### 2.2 Claude CLI Provider

New provider `claude-cli` implementing `providers.Provider` interface by wrapping the `claude` CLI binary as a subprocess. Billing goes through the user's Claude Max $200 subscription (OAuth), NOT an Anthropic API key.

**Reference**: EndiorBot `src/agents/invoke/claude-code-bridge.ts` (808 lines)

### 2.3 Architecture

```
                    ┌─────────────────────────────┐
                    │      Agent Loop (loop.go)    │
                    │                              │
                    │  1. Try primary provider     │
                    │  2. If fail → try fallback   │
                    │  3. If all fail → error       │
                    └──────────┬──────────────────┘
                               │
              ┌────────────────┼────────────────┐
              ▼                ▼                 ▼
     ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
     │ bflow-ai     │  │ claude-cli   │  │ (future)     │
     │ (Ollama)     │  │ (subprocess) │  │ anthropic    │
     │              │  │              │  │ openrouter   │
     │ HTTP API     │  │ claude -p    │  │ etc.         │
     │ qwen3:14b    │  │ --model      │  │              │
     │              │  │ sonnet       │  │              │
     └──────────────┘  └──────────────┘  └──────────────┘
```

## 3. Detailed Design

### 3.1 Claude CLI Provider (`internal/providers/claude_cli.go`)

```go
// ClaudeCLIProvider wraps the `claude` CLI binary as an LLM provider.
// Uses Claude Max subscription (OAuth) — no API key required.
// Reference: EndiorBot claude-code-bridge.ts
type ClaudeCLIProvider struct {
    claudePath string        // Path to claude binary (default: "claude")
    model      string        // Model name (default: "sonnet")
    timeout    time.Duration // Per-request timeout (default: 120s)
}
```

**CLI invocation pattern** (from EndiorBot, corrected per CTO-503/504):
```bash
claude -p \
  --output-format json \
  --model sonnet \
  --max-turns 1 \
  "user prompt here"
```

**Key flags:**
- `-p` / `--print`: Non-interactive mode, output to stdout
- `--output-format json`: Structured JSON output with token usage
- `--model sonnet`: Claude Sonnet (cost-efficient)
- `--max-turns 1`: Single-turn only — no internal tool loops (CTO-504)

**Environment (CTO review — subprocess isolation required):**
```go
// MUST strip ANTHROPIC_API_KEY to force OAuth (Claude Max subscription).
// If key exists in parent env, CLI would use API billing instead of Max plan.
cmd.Env = filterEnv(os.Environ(), "ANTHROPIC_API_KEY", "CLAUDECODE")
```

**Interface mapping:**

| `providers.Provider` method | Implementation |
|---|---|
| `Name()` | Returns `"claude-cli"` |
| `DefaultModel()` | Returns `"sonnet"` |
| `Chat(ctx, req)` | Spawn subprocess, collect stdout, parse JSON response |
| `ChatStream(ctx, req, onChunk)` | Delegate to `Chat()`, emit single chunk (CTO-500/502) |

**ChatStream implementation (CTO-500, CTO-502):**
Claude CLI does NOT support true streaming. `ChatStream` delegates to `Chat()` then emits a single chunk:
```go
func (p *ClaudeCLIProvider) ChatStream(ctx context.Context, req ChatRequest,
    onChunk func(StreamChunk)) (*ChatResponse, error) {
    resp, err := p.Chat(ctx, req)
    if err != nil {
        return nil, err
    }
    onChunk(StreamChunk{Content: resp.Content})
    return resp, nil
}
```
This is acceptable UX for fallback: user gets complete response after brief wait instead of streaming chunks. The Telegram placeholder ("Provider busy, retrying...") keeps UX informative.

**Tool call support:**
- Claude CLI with `--max-turns 1` returns text-only (no internal tool loops)
- Fallback strips tools from request (see section 3.3)
- For the fallback use case (Ollama down), **text-only response is sufficient** — tools were already attempted by primary provider in earlier iterations

**JSON output format:**
```json
{
  "type": "result",
  "result": "response text",
  "cost_usd": 0.003,
  "duration_ms": 1500,
  "num_turns": 1
}
```

### 3.2 Fallback Chain Configuration

**Environment variables:**
```bash
# Provider chain (comma-separated, ordered by priority)
MTCLAW_PROVIDER_CHAIN=bflow-ai-platform,claude-cli

# Claude CLI specific
MTCLAW_CLAUDE_PATH=/home/dttai/.local/bin/claude
MTCLAW_CLAUDE_MODEL=sonnet
MTCLAW_CLAUDE_TIMEOUT=120
```

**Config struct addition** (`internal/config/config.go`):
```go
type ProviderChainConfig struct {
    Chain        []string      `json:"chain"`          // ["bflow-ai-platform", "claude-cli"]
    FallbackTimeout time.Duration `json:"fallback_timeout"` // timeout before trying next (default: 30s)
}

type ClaudeCLIConfig struct {
    Path    string `json:"path"`    // claude binary path
    Model   string `json:"model"`   // default: "sonnet"
    Timeout int    `json:"timeout"` // seconds, default: 120
}
```

### 3.3 Fallback Logic in Agent Loop

**Location:** `internal/agent/loop.go`, LLM call site (both Chat and ChatStream paths)

**Current code (two call paths — CTO-500):**
```go
if req.Stream {
    resp, err = l.provider.ChatStream(ctx, chatReq, func(chunk ...) { ... })
} else {
    resp, err = l.provider.Chat(ctx, chatReq)
}
if err != nil {
    return nil, fmt.Errorf("LLM call failed (iteration %d): %w", iteration, err)
}
```

**Proposed change (handles BOTH paths):**
```go
// Primary provider call (Chat or ChatStream)
if req.Stream {
    resp, err = l.provider.ChatStream(ctx, chatReq, onChunkFn)
} else {
    resp, err = l.provider.Chat(ctx, chatReq)
}

// Fallback on retryable error (CTO-500: covers both Chat and ChatStream)
if err != nil && l.fallbackProvider != nil && providers.IsRetryableError(err) {
    // CTO review: Only fallback text-only when iteration > 1 (tools already ran).
    // If iteration == 1 and tools are needed, don't fallback — propagate error.
    if iteration == 1 && len(chatReq.Tools) > 0 {
        slog.Warn("primary failed at iteration 1 with tools — no fallback (tools needed)",
            "agent", l.id, "error", err.Error())
        // Fall through to error return below
    } else {
        slog.Warn("primary provider failed, trying fallback",
            "agent", l.id, "iteration", iteration,
            "primary", l.provider.Name(),
            "fallback", l.fallbackProvider.Name(),
            "error", err.Error(),
        )
        l.emit(AgentEvent{Type: protocol.AgentEventRunRetrying, ...})

        // CTO-501: Always strip tools on fallback (no type assertion needed).
        // Rationale: fallback is for text generation after tools already executed.
        fallbackReq := chatReq
        fallbackReq.Tools = nil

        // CTO review (context window mismatch): prune history for fallback provider
        // Fallback may have different context window than primary.
        // Use fallback Chat (not ChatStream) — CLI doesn't support true streaming.
        resp, err = l.fallbackProvider.Chat(ctx, fallbackReq)
    }
}
if err != nil {
    return nil, fmt.Errorf("LLM call failed (iteration %d): %w", iteration, err)
}
```

**Key design decisions (from CTO review):**
1. **CTO-500**: Fallback handles both `Chat` and `ChatStream` paths. Fallback always uses `Chat` (CLI doesn't support true streaming; `ChatStream` delegates to `Chat` + single chunk).
2. **CTO-501**: Always strip tools on fallback — no fragile type assertion. If future providers also can't do tools, no code change needed.
3. **CTO review (iteration guard)**: Only fallback text-only when `iteration > 1`. At iteration 1, if tools are required, propagate error instead of giving wrong text-only answer.
4. **CTO review (context window)**: Fallback may have different context limits. History should be pruned before sending to fallback.

**Loop struct addition:**
```go
type Loop struct {
    // ... existing fields ...
    fallbackProvider providers.Provider // nil if no fallback configured
}
```

### 3.4 Provider Registration

**Location:** `cmd/gateway_providers.go`

```go
// Register Claude CLI provider if binary exists
if claudePath := cfg.ClaudeCLI.Path; claudePath != "" {
    if _, err := exec.LookPath(claudePath); err == nil {
        registry.Register(providers.NewClaudeCLIProvider(claudePath,
            cfg.ClaudeCLI.Model, cfg.ClaudeCLI.Timeout))
        slog.Info("registered provider", "name", "claude-cli")
    }
}
```

### 3.5 Fallback Wiring in Resolver

**Location:** `internal/agent/resolver.go`

After resolving primary provider, resolve fallback from chain:

```go
// Resolve fallback provider (if chain configured)
var fallbackProvider providers.Provider
if chain := deps.ProviderChain; len(chain) > 1 {
    for _, name := range chain[1:] {
        if fp, err := deps.ProviderReg.Get(name); err == nil {
            fallbackProvider = fp
            break
        }
    }
}
```

## 4. Comparison with EndiorBot

| Aspect | EndiorBot | MTClaw (Proposed) |
|---|---|---|
| Language | TypeScript | Go |
| CLI call | `child_process.spawn()` | `exec.CommandContext()` |
| Timeout | `setTimeout()` + `SIGTERM` | `context.WithTimeout()` |
| Output format | `--output-format text` | `--output-format json` |
| Tool support | No (CLI bridge, not provider) | No (fallback mode strips tools) |
| Provider integration | Standalone bridge class | Implements `providers.Provider` |
| Fallback trigger | Health-based routing | Retryable error in agent loop |
| Model | Hardcoded `sonnet` | Configurable, default `sonnet` |
| Auth | OAuth (unset API key) | OAuth (unset API key) |

**Key difference:** EndiorBot's `ClaudeCodeBridge` is a standalone utility (not in provider registry). MTClaw integrates Claude CLI as a **standard provider** in the registry, enabling seamless fallback through the existing retry/provider infrastructure.

## 5. Security Considerations

- **No API key exposure**: Claude CLI uses OAuth (Claude Max subscription), no key in env
- **Subprocess isolation**: `exec.CommandContext` with timeout, no shell expansion
- **Environment sanitization**: Unset `ANTHROPIC_API_KEY` and `CLAUDECODE` to force OAuth
- **Cost control**: Claude Max $200/month plan has built-in quota — no runaway costs
- **Read-only by default**: Fallback calls strip tools → no filesystem/exec access

## 6. Files to Create/Modify

| File | Action | Description |
|---|---|---|
| `internal/providers/claude_cli.go` | **CREATE** | Claude CLI provider implementation |
| `internal/providers/claude_cli_test.go` | **CREATE** | Unit tests |
| `internal/config/config.go` | MODIFY | Add `ClaudeCLI` + `ProviderChain` config |
| `internal/config/config_load.go` | MODIFY | Load `MTCLAW_CLAUDE_*` + `MTCLAW_PROVIDER_CHAIN` env vars |
| `internal/agent/loop.go` | MODIFY | Add fallback provider call on retryable error |
| `internal/agent/resolver.go` | MODIFY | Wire fallback provider from chain |
| `cmd/gateway_providers.go` | MODIFY | Register claude-cli provider |
| `.env.example` | MODIFY | Add new env vars |

## 7. Test Plan

| ID | Test | Expected |
|---|---|---|
| UNIT-001 | `ClaudeCLIProvider.Name()` returns `"claude-cli"` | PASS |
| UNIT-002 | `ClaudeCLIProvider.Chat()` with mock subprocess | Returns parsed response |
| UNIT-003 | `ClaudeCLIProvider.Chat()` timeout | Returns error after timeout |
| UNIT-004 | `ClaudeCLIProvider.Chat()` binary not found | Returns clear error |
| UNIT-005 | Fallback triggers on retryable error (iteration > 1) | Loop uses fallback provider |
| UNIT-006 | Fallback NOT triggered on non-retryable error | Error propagated directly |
| UNIT-007 | Fallback always strips tools | `ChatRequest.Tools = nil` |
| UNIT-008 | `ChatStream` delegates to `Chat` + single chunk | Chunk emitted, response returned |
| UNIT-009 | Subprocess env strips `ANTHROPIC_API_KEY` | Key not in child env |
| UNIT-010 | Iteration=1 with tools → NO fallback | Error propagated, not text-only |
| UNIT-011 | Iteration=1 without tools → fallback OK | Fallback used (simple chat) |
| INT-001 | Primary Ollama timeout → Claude CLI fallback → response | User gets response |
| INT-002 | Primary succeeds → no fallback | Direct response, no subprocess |
| E2E-001 | Telegram message when Ollama down | Bot responds via Claude CLI |
| E2E-002 | Telegram message when Ollama up | Bot responds via Ollama (fast) |

## 8. Rollout Plan

1. **Phase A** (This sprint): Create `ClaudeCLIProvider` + fallback logic
2. **Phase B** (Next sprint): Health-based routing (EndiorBot pattern — track success rate per provider)
3. **Phase C** (Future): Multi-provider consultation for complex tasks

## 9. Consequences

**Positive:**
- Zero-downtime when Ollama GPU unavailable — automatic Claude CLI fallback
- Utilizes existing Claude Max $200 subscription (currently wasted)
- Simple implementation (~200 lines Go) — no new infrastructure
- Extensible: same pattern works for future providers (OpenRouter, Gemini, etc.)

**Negative:**
- Claude CLI subprocess adds ~2-3s startup overhead per call
- No native tool call support in fallback mode (text-only response)
- Requires `claude` binary installed on gateway host (Docker image needs update)

**Neutral:**
- Fallback responses may differ in style/quality (Sonnet vs Qwen3)
- Token usage tracking approximate (CLI output parsing)

## 10. CTO Review Resolution

| ID | Severity | Issue | Resolution |
|---|---|---|---|
| CTO-500 | BLOCKING | Streaming path not covered | Fallback handles both Chat/ChatStream; fallback always uses Chat (section 3.3) |
| CTO-501 | BLOCKING | Type assertion for tool stripping | Always strip tools on fallback — no type assertion (section 3.3) |
| CTO-502 | MEDIUM | ChatStream impl needed | `ChatStream` delegates to `Chat`, emits single chunk (section 3.1) |
| CTO-503 | MEDIUM | `--no-input` flag doesn't exist | Removed from spec, use `-p` only (section 3.1) |
| CTO-504 | MEDIUM | Missing `--max-turns 1` | Added to CLI invocation (section 3.1) |
| CTO-505 | LOW | Env var prefix (GOCLAW_ vs MTCLAW_) | Codebase fully renamed to MTCLAW_ — consistent |
| CTO-506 | LOW | Effort estimate undercount | Updated to ~300-350 lines total |
| CTO-507 | LOW | int→Duration conversion | Explicit `time.Duration(cfg.Timeout) * time.Second` |
| CTO-R2-1 | BLOCKING | iteration=1 tool-strip gives wrong answer | Only fallback text-only when `iteration > 1` (section 3.3) |
| CTO-R2-2 | BLOCKING | ANTHROPIC_API_KEY subprocess leak | `cmd.Env = filterEnv(...)` strips key (section 3.1) |
| CTO-R2-3 | BLOCKING | ChatStream blocks without emitting | Mock ChatStream: Chat() + single chunk (section 3.1) |
| CTO-R2-4 | MEDIUM | Context window mismatch on fallback | Prune history for fallback context window (section 3.3) |
| CTO-R2-5 | LOW | `doctor` command check | Add `which claude` + version check to doctor.go |
| CTO-R2-6 | LOW | Tracing double-fire | Tag fallback span with `provider=claude-cli, fallback=true` |
| CTO-R2-7 | LOW | Config disable flag | Add `enabled: bool` to fallback config |

---

**Decision**: APPROVED — All CTO conditions met
**Estimated effort**: 1 sprint (3-5 days), ~300-350 lines Go (CTO-506 corrected)
**Risk**: LOW — additive change, no breaking modifications
