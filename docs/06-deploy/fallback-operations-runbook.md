# Fallback Operations Runbook

**Version**: 1.0.0
**Sprint**: 28 (T28.4)
**Status**: Current

---

## Provider Chain Configuration

### config.json

```json
{
  "provider_chain": {
    "chain": ["bflow-ai-platform", "openrouter", "claude-cli"]
  }
}
```

The chain is ordered: first entry is primary, subsequent entries are fallback candidates. The resolver picks the first non-primary provider from the chain as the fallback.

### Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `MTCLAW_CLAUDE_CLI_ENABLED` | false | Enable Claude CLI provider |
| `MTCLAW_CLAUDE_CLI_PATH` | claude | Binary path |
| `MTCLAW_CLAUDE_CLI_MODEL` | sonnet | Default model |
| `MTCLAW_CLAUDE_CLI_TIMEOUT` | 120 | Timeout in seconds |
| `MTCLAW_PROVIDER_CB_COOLDOWN` | 30 | Circuit breaker cooldown (seconds) |

### Precedence Rules

1. Per-agent `provider` field in DB determines primary
2. Global `provider_chain` determines fallback order
3. Resolver skips the primary when selecting fallback
4. If no chain configured, no fallback (single provider mode)

## Health Monitoring

### Doctor Output

```bash
./mtclaw doctor
```

Shows:
- Claude CLI binary path, version, model, timeout
- OAuth token status
- Provider chain display

### Health Tracker (Runtime)

The health tracker monitors provider success/failure with:
- **Sliding window**: Last 100 calls or 10 minutes
- **Circuit breaker**: Trips after 3 consecutive failures or <50% success rate
- **Cooldown**: 30s (configurable via `MTCLAW_PROVIDER_CB_COOLDOWN`)
- **Recovery**: Half-open → single probe → closed on success

Circuit states:
| State | Behavior |
|-------|----------|
| `closed` | Normal — requests flow through |
| `open` | Tripped — skip provider, wait for cooldown |
| `half-open` | Probe — allow 1 request to test recovery |

### Trace Analysis

Fallback events are tagged in OTel traces:
- Primary fail span: `status=error`, `provider=<primary>`
- Fallback success span: `status=completed`, `fallback=true`, `primary_provider=<primary>`, `primary_error=<error>`

Query fallback events:
```sql
SELECT s.created_at, s.provider, s.status,
       s.metadata->>'fallback' as is_fallback,
       s.metadata->>'primary_provider' as primary,
       s.metadata->>'primary_error' as error
FROM spans s
WHERE s.metadata->>'fallback' = 'true'
ORDER BY s.created_at DESC
LIMIT 20;
```

## Fallback Scenarios

### Scenario 1: Primary Temporary Failure (429/502/503)

**Symptom**: Users still get responses but with higher latency.

**What happens**:
1. Primary returns retryable error (429, 500, 502, 503, 504)
2. Retry with exponential backoff (3 attempts, 300ms-30s)
3. After all retries fail, fallback provider is tried
4. Health tracker records failure on primary

**Action**: Monitor. If frequent, check primary provider status page.

### Scenario 2: Primary Extended Outage

**Symptom**: All requests hitting fallback, higher latency and cost.

**What happens**:
1. Circuit breaker trips on primary after 3 consecutive failures
2. All subsequent requests go directly to fallback (no primary attempt)
3. After cooldown, half-open probe tests primary recovery

**Action**: Check primary provider status. If outage confirmed, no action needed — fallback handles traffic automatically.

### Scenario 3: Both Providers Fail

**Symptom**: Users get error messages.

**What happens**:
1. Primary fails → fallback attempted
2. Fallback also fails → error returned to user
3. Both circuits may trip

**Action**: Check both provider status pages. If both are down, wait for recovery. Consider manual override to a third provider.

## Troubleshooting

### Claude CLI Errors

| Error | Cause | Fix |
|-------|-------|-----|
| `claude: not found` | Binary not installed | Build with `ENABLE_CLAUDE_CLI=true` |
| `claude: not authenticated` | OAuth expired | `claude login` inside container |
| `process exited with code 1` | CLI error (model issue, quota) | Check Claude CLI logs, verify model name |
| `context deadline exceeded` | Timeout (default 120s) | Increase `MTCLAW_CLAUDE_CLI_TIMEOUT` |

### OAuth Expiry

OAuth tokens for Claude CLI need periodic refresh:
1. Check: `docker compose exec mtclaw claude auth status`
2. Fix: `docker compose exec mtclaw claude login`
3. Prevent: Ensure `claude-oauth` Docker volume is mounted

### Timeout Tuning

Default timeouts:
- Primary HTTP providers: Determined by Go HTTP client (default 30s)
- Claude CLI subprocess: 120s (configurable)
- Retry backoff: 300ms initial, 30s max, ±10% jitter

To increase CLI timeout:
```bash
MTCLAW_CLAUDE_CLI_TIMEOUT=180  # 3 minutes
```

## Rollback

### Disable Fallback

Set provider chain to single provider:
```json
{
  "provider_chain": {
    "chain": ["bflow-ai-platform"]
  }
}
```

Or disable Claude CLI:
```bash
MTCLAW_CLAUDE_CLI_ENABLED=false
```

### Emergency Direct-Provider Override

Force a specific provider for all agents:
```bash
GOCLAW_PROVIDER=openrouter  # Override default provider
```

Note: This overrides config.json but not per-agent DB settings.

## CTO Guards

The fallback chain has two safety guards:

### CTO-R2-1: No Fallback at Iteration 1 with Tools

At iteration 1 (before any tools have run), falling back to a text-only provider would produce incorrect behavior — the agent should use its tools, not give a text-only answer. Fallback is only allowed at iteration > 1 (tools already ran, synthesis is acceptable).

### CTO-501: Always Strip Tools on Fallback

Fallback requests always have `Tools = nil`. Fallback providers (especially Claude CLI with `--max-turns 1`) don't support the same tool schemas. This ensures clean text responses.

---

**Created**: 2026-03-08
**Author**: [@coder]
