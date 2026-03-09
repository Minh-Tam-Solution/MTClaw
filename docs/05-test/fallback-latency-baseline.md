# Fallback Latency Baseline

**Version**: 1.0.0
**Sprint**: 28 (T28.5)
**Date**: 2026-03-08

---

## Expected Latency Ranges

| Provider Type | Expected p50 | Expected p95 | Notes |
|---------------|-------------|-------------|-------|
| HTTP (bflow-ai-platform) | 500ms | 2s | Network + inference |
| HTTP (openrouter) | 800ms | 3s | Network + inference (multi-hop) |
| HTTP (anthropic direct) | 600ms | 2.5s | Network + inference |
| Claude CLI (subprocess) | 3s | 10s | Process spawn + inference + JSON parse |
| Fallback overhead | +50ms | +200ms | Health check + request rebuild + logging |

## Non-Subprocess Benchmark Results

Measured on AMD Ryzen 9 9950X (representative server hardware):

| Operation | Latency (ns/op) | Allocs | Notes |
|-----------|-----------------|--------|-------|
| `BuildPrompt` (4 messages) | 247 | 5 | String concatenation |
| `ParseResponse` (JSON) | 3,080 | 20 | JSON unmarshal + content extraction |
| `FilterEnv` (25 vars) | 131 | 1 | Linear scan + 2 vars stripped |

**Total non-subprocess overhead**: ~3.5μs (negligible compared to subprocess + inference)

## Fallback Path Breakdown

```
Total fallback time = Primary retry time + Fallback call time + Overhead

Where:
  Primary retry time = 3 attempts × (inference time + backoff delay)
                     = varies (0.9s - 90s depending on backoff)
  Fallback call time = subprocess spawn + inference + parse
                     = 3-10s for Claude CLI
  Overhead           = health check + request rebuild + span emit + logging
                     = ~50-200ms
```

## Circuit Breaker Impact

When the circuit breaker trips on the primary provider:
- **Skip primary entirely**: Saves 3 retry attempts × (300ms-30s backoff)
- **Direct to fallback**: Only fallback latency applies
- **Net effect**: 3-90s saved per request during outage

## Retry Policy Defaults

| Parameter | Value | Impact |
|-----------|-------|--------|
| Max attempts | 3 | Up to 3 retries before fallback |
| Initial delay | 300ms | First retry after 300ms |
| Max delay | 30s | Cap on exponential backoff |
| Jitter | ±10% | Prevents thundering herd |

---

**Benchmark command**: `go test ./internal/providers/ -bench=BenchmarkClaudeCLI -benchmem`
