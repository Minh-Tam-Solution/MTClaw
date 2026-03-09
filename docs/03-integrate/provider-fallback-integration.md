# Provider Fallback Chain Integration Specification

**Version**: 1.0.0
**Sprint**: 27 (T27.5b)
**ADR**: ADR-014 — Provider Fallback via Claude CLI
**Status**: Current

---

## Overview

MTClaw supports a provider fallback chain that automatically switches to a backup LLM provider when the primary fails with a retryable error. This ensures continuous service during provider outages.

## Fallback Chain Architecture

```
User message → Agent Loop (loop.go)
                    ↓
              Primary Provider.Chat()
                    ↓ (success)
              Return response
                    ↓ (retryable error)
              Fallback Provider.Chat()
                    ↓
              Return fallback response
```

## Provider Chain Configuration

### config.json

```json
{
  "provider_chain": {
    "chain": ["bflow-ai-platform", "openrouter", "claude-cli"]
  }
}
```

The chain is ordered: first entry is primary, subsequent entries are fallback candidates. The resolver (`internal/agent/resolver.go`) picks the first non-primary provider from the chain as the fallback.

### Per-Agent Override

In managed mode, each agent's `provider` field in the `agents` table determines its primary. The fallback is resolved from the global chain, skipping the primary.

## Retryable Error Classification

Fallback triggers only on **retryable** errors (provider down, rate limited):

| HTTP Status | Retryable | Reason |
|-------------|-----------|--------|
| 429 | Yes | Rate limited |
| 500 | Yes | Internal server error |
| 502 | Yes | Bad gateway |
| 503 | Yes | Service unavailable |
| 504 | Yes | Gateway timeout |
| 400 | No | Bad request (client error) |
| 401 | No | Unauthorized (bad API key) |
| 403 | No | Forbidden |
| 404 | No | Not found |

Connection errors and timeouts are also retryable.

Implementation: `providers.IsRetryableError()` in `internal/providers/retry.go`.

## CTO Guards

### CTO-R2-1: No Fallback at Iteration 1 with Tools

```go
if iteration == 1 && len(chatReq.Tools) > 0 {
    canFallback = false // Don't give text-only answer when tools needed
}
```

**Rationale**: At iteration 1, the agent hasn't run any tools yet. Falling back to a provider without tools would produce a text-only answer that ignores available tools — wrong behavior.

At iteration > 1, tools have already executed, so a text-only synthesis is acceptable as fallback.

### CTO-501: Always Strip Tools on Fallback

```go
fallbackReq.Tools = nil // Always strip tools on fallback
```

**Rationale**: Fallback providers (especially Claude CLI with `--max-turns 1`) don't support the same tool schemas. Stripping tools ensures the fallback produces a clean text response.

### CTO-500/502: Both Chat and ChatStream Paths

Fallback triggers on both `Chat()` and `ChatStream()` error paths. For Claude CLI, `ChatStream()` delegates to `Chat()` and wraps the response as a single chunk.

## Retry Policy (Primary Provider)

Before fallback, the primary provider retries with exponential backoff:

| Parameter | Default |
|-----------|---------|
| Max attempts | 3 |
| Initial delay | 300ms |
| Max delay | 30s |
| Jitter | ±10% |

Configuration: `providers.RetryConfig` in `internal/providers/retry.go`.

Retry applies per-attempt within the primary. Only after all retry attempts fail does the fallback chain activate.

## Tracing and Observability

### Primary Failure Span

When primary fails and fallback succeeds, two spans are emitted:

1. **Primary span** (status=error): Records the primary provider's error
2. **Fallback span** (status=completed): Records the fallback response

Fallback span metadata:
```json
{
  "fallback": "true",
  "primary_provider": "anthropic",
  "primary_error": "HTTP 502: bad gateway"
}
```

Span tags: `fallback=true` for easy querying.

### Cost Tracking

Fallback token usage is recorded in spans (`input_tokens`, `output_tokens`) and aggregated into trace totals via `BatchUpdateTraceAggregates`.

## Supported Providers

| Provider | Type | Fallback Role |
|----------|------|---------------|
| `bflow-ai-platform` | Primary | Main inference |
| `anthropic` | Primary/Fallback | Direct Claude API |
| `openrouter` | Primary/Fallback | Multi-model gateway |
| `openai` | Primary/Fallback | GPT models |
| `gemini` | Primary/Fallback | Google models |
| `groq` | Primary/Fallback | Fast inference |
| `deepseek` | Primary/Fallback | DeepSeek models |
| `mistral` | Primary/Fallback | Mistral models |
| `xai` | Primary/Fallback | Grok models |
| `dashscope` | Primary/Fallback | Alibaba QWen |
| `claude-cli` | Fallback only | Subprocess (last resort) |

### Claude CLI Provider Details

- Runs as subprocess (`os/exec.Command`)
- Flags: `-p --output-format json --model {model} --max-turns 1`
- Environment: Strips `ANTHROPIC_API_KEY` and `CLAUDE_API_KEY` (CTO-R2-2)
- Requires OAuth login: `claude login`
- Docker: `ENABLE_BRIDGE=true` build arg adds `claude` binary

## Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `MTCLAW_CLAUDE_CLI_ENABLED` | false | Enable Claude CLI provider |
| `MTCLAW_CLAUDE_CLI_PATH` | claude | Binary path |
| `MTCLAW_CLAUDE_CLI_MODEL` | sonnet | Default model |
| `MTCLAW_CLAUDE_CLI_TIMEOUT` | 120 | Timeout in seconds |

## Doctor Diagnostics

`mtclaw doctor` displays:

```
  Claude CLI (fallback):
    Binary:      /usr/local/bin/claude
    Version:     1.0.0
    Model:       sonnet
    Timeout:     120s
    OAuth:       /home/user/.claude (OK)

  Provider Chain: bflow-ai-platform → openrouter → claude-cli
```

## Error Handling

| Scenario | Behavior |
|----------|----------|
| Primary succeeds | Normal response, no fallback |
| Primary retryable, fallback succeeds | Return fallback response, log warning |
| Primary fatal (400/401/403) | Propagate error, no fallback |
| Both fail | Return error with both failure details |
| No fallback configured | Propagate primary error |

## Testing

- Unit tests: `internal/providers/claude_cli_test.go` (11 tests)
- Fallback logic: `internal/agent/fallback_test.go` (11 tests including 6 E2E scenarios)
- E2E: Manual verification with `MTCLAW_CLAUDE_CLI_ENABLED=true`
