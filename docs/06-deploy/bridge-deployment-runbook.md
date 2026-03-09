# Bridge Deployment Runbook

**Version**: 1.0.0
**Sprint**: 28 (T28.4)
**Status**: Current

---

## Prerequisites

| Component | Required | Check Command |
|-----------|----------|---------------|
| tmux 3.x+ | Yes | `tmux -V` |
| Claude CLI 2.x | Yes (for Claude Code bridge) | `claude --version` |
| OAuth login | Yes (for Claude Code) | `ls ~/.claude/` |
| MTClaw binary | Yes | `./mtclaw version` |
| PostgreSQL | Managed mode only | `psql $MTCLAW_POSTGRES_DSN -c 'SELECT 1'` |

## Docker Deployment

### Build with Bridge Enabled

```bash
# Build with bridge support (includes tmux)
docker compose build --build-arg ENABLE_BRIDGE=true mtclaw

# Build with Claude CLI fallback support
docker compose build --build-arg ENABLE_CLAUDE_CLI=true mtclaw

# Build with both
docker compose build \
  --build-arg ENABLE_BRIDGE=true \
  --build-arg ENABLE_CLAUDE_CLI=true \
  mtclaw
```

### Docker Compose Configuration

```yaml
services:
  mtclaw:
    build:
      args:
        ENABLE_BRIDGE: "true"
    environment:
      MTCLAW_BRIDGE_ENABLED: "true"
      MTCLAW_BRIDGE_HOOK_PORT: "18792"
      MTCLAW_BRIDGE_AUDIT_DIR: "/var/log/mtclaw/bridge-audit"
    volumes:
      - claude-oauth:/app/.claude  # Persist OAuth tokens across restarts
    ports:
      - "18792:18792"  # Hook server (localhost only in production)

volumes:
  claude-oauth:
```

### Volume Mounts

| Volume | Purpose | Required |
|--------|---------|----------|
| `claude-oauth` | OAuth token persistence | Yes (for Claude CLI) |
| `/var/log/mtclaw/bridge-audit` | Audit JSONL logs | Recommended |

## Bridge Setup Checklist

1. **Enable bridge** in config.json:
   ```json
   {
     "bridge": {
       "enabled": true,
       "hook_port": 18792,
       "audit_dir": "/var/log/mtclaw/bridge-audit",
       "admission": {
         "max_sessions_per_agent": 2,
         "max_total_sessions": 6,
         "per_tenant_session_cap": 4
       }
     }
   }
   ```

2. **Run bridge setup** to generate hook scripts:
   ```bash
   ./mtclaw bridge setup
   ```

3. **Verify bridge health**:
   ```bash
   ./mtclaw bridge status
   ```

4. **Verify doctor output**:
   ```bash
   ./mtclaw doctor
   ```

## OAuth Token Management

### Initial Login

```bash
# Inside container
docker compose exec mtclaw claude login

# Verify
docker compose exec mtclaw ls /app/.claude/
```

### Token Refresh

OAuth tokens are automatically refreshed by the Claude CLI. The named Docker volume (`claude-oauth`) ensures tokens persist across container restarts.

### Troubleshooting OAuth

| Symptom | Cause | Fix |
|---------|-------|-----|
| `claude: not authenticated` | OAuth expired or volume lost | `claude login` inside container |
| `claude: permission denied` | Binary permissions | Verify `chmod +x` on claude binary |
| Token lost after restart | Volume not mounted | Add `claude-oauth` volume to compose |

## Troubleshooting

### Common Errors

| Error | Cause | Fix |
|-------|-------|-----|
| `tmux: command not found` | tmux not installed | Build with `ENABLE_BRIDGE=true` |
| `hook auth failed: invalid signature` | Hook secret mismatch | Re-run `mtclaw bridge setup` |
| `session stuck in busy state` | Process hang or crash | `mtclaw bridge kill --all` or wait for health monitor |
| `admission denied: max sessions` | Session limit reached | Kill idle sessions or increase limits |
| `audit dir not writable` | Permission issue | `chmod 755 /var/log/mtclaw/bridge-audit` |
| `hook server: port in use` | Another process on 18792 | Change `hook_port` in config.json |

### Health Monitor

The bridge health monitor runs every 30 seconds and detects:
- Dead tmux sessions (process exited but session not stopped)
- Stale sessions (no activity for extended period)
- Resource limit violations

### Session Recovery After Restart

When the gateway restarts:
1. Non-stopped sessions are loaded from PG as `status=disconnected`
2. tmux sessions are gone (ephemeral) — users see "disconnected" status
3. Users can kill disconnected sessions and create new ones
4. No automatic reconnection (by design — tmux state is lost)

## Rollback

### Disable Bridge Without Restart

Set `MTCLAW_BRIDGE_ENABLED=false` in environment and restart the gateway. Existing sessions will remain in memory until killed or cleaned up.

### Emergency Kill-All Sessions

```bash
# Kill all active bridge sessions
docker compose exec mtclaw mtclaw bridge kill --all

# Or via Telegram
/cc kill-all  # (admin-only command)
```

### Revert to Non-Bridge Build

```bash
docker compose build mtclaw  # Without ENABLE_BRIDGE arg
docker compose up -d mtclaw
```

---

**Created**: 2026-03-08
**Author**: [@coder]
