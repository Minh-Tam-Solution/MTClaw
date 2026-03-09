# Bridge Integration Specification

**Version**: 1.0.0
**Sprint**: 27 (T27.5a)
**Status**: Current

---

## Overview

The Claude Code Bridge enables MTClaw to manage interactive AI coding sessions (Claude Code, Cursor, Codex CLI, Gemini CLI) via tmux subprocess control. Sessions are created from channel commands (`/cc start`), managed through the SessionManager, and communicate back via webhook hooks.

## Architecture

```
Channel (Telegram /cc) → SessionManager → TmuxBridge → Claude CLI
                                                            ↓
                              HookServer:18792 ← hook script (POST)
                                    ↓
                              PermissionStore → Notifier → Channel callback
```

## Session Lifecycle

### States

| State | Description |
|-------|-------------|
| `active` | Normal operation, accepting input |
| `busy` | Processing, input queued |
| `idle` | Waiting for input |
| `stopped` | Terminated (normal exit) |
| `error` | Fatal error occurred |
| `disconnected` | Recovered from PG after restart (tmux gone) |

### Transitions

```
create → active → busy ↔ active
                     ↓
              idle / stopped / error
restart recovery: PG → disconnected
```

## Risk Modes (Capability Model)

Three risk levels map to a 3-axis capability model:

| Risk Mode | Input Mode | Tool Policy | Capture Lines | Redact |
|-----------|-----------|-------------|---------------|--------|
| `read` (default) | structured_only | observe | 30 | true |
| `patch` | structured_only | patch_allowed | 50 | false |
| `interactive` | free_text | exec_with_approval | 100 | false |

- **structured_only**: Only `/cc` commands accepted
- **free_text**: Shell-like text input allowed
- **observe**: Read-only tools only
- **patch_allowed**: File edits permitted, no exec
- **exec_with_approval**: Full capabilities, requires permission hooks

## Hook Protocol

### Endpoint

`POST http://127.0.0.1:{hook_port}/hook`

Default port: 18792. Never exposed externally (localhost only).

### Authentication

| Header | Value | Purpose |
|--------|-------|---------|
| `X-Hook-Signature` | HMAC-SHA256(secret, body) | Request authenticity |
| `X-Hook-Timestamp` | Unix timestamp | Replay protection |
| `X-Hook-Session` | Session ID | Canonical session override |

Rate limit: 10 hooks/sec per session.

### Stop Event

```json
{
  "session_id": "br:12345678:abcd1234",
  "event": "stop",
  "exit_code": 0,
  "summary": "Completed task successfully",
  "git_diff": "+3 -1 files changed"
}
```

### Permission Request

```json
{
  "session_id": "br:12345678:abcd1234",
  "event": "permission",
  "tool": "exec",
  "tool_input": {"command": "npm install"}
}
```

Response: `202 Accepted` (async flow).

### Permission Poll

`GET /hook/permission/{permission_id}`

```json
{
  "id": "perm:12345678:abcd1234",
  "decision": "allow|deny|pending",
  "tool": "exec",
  "expires_at": "2026-03-08T12:34:56Z"
}
```

Default TTL: 5 minutes.

### Health Check

`GET /health`

```json
{"status": "ok", "sessions": 5}
```

## Audit Events

Dual-write: JSONL (mandatory) + PostgreSQL (best-effort).

### Event Format

```json
{
  "owner_id": "tenant-1",
  "session_id": "br:12345678:abcd1234",
  "actor_id": "user-1",
  "action": "session.created",
  "risk_mode": "read",
  "detail": {"agent_type": "claude-code", "project": "/app"},
  "created_at": "2026-03-08T10:00:00Z"
}
```

### Actions

| Action | Trigger | Detail Fields |
|--------|---------|---------------|
| `session.created` | CreateSession | agent_type, project, risk_mode, agent_role |
| `session.killed` | KillSession | killed_by |
| `session.risk_changed` | UpdateRiskMode | old_risk, new_risk, actor |
| `permission.created` | Hook permission event | tool, risk_level |
| `permission.approved` | Telegram callback | approved_by, reason |
| `permission.denied` | Telegram callback | denied_by, reason |

### JSONL Storage

- Path: `{audit_dir}/bridge-audit-YYYY-MM-DD.jsonl`
- Default dir: `~/.mtclaw/bridge-audit`
- Rotation: Daily (midnight)
- Permissions: 0600

### PostgreSQL Table

```sql
CREATE TABLE bridge_audit_events (
  id BIGSERIAL PRIMARY KEY,
  owner_id TEXT NOT NULL,
  session_id TEXT,
  actor_id TEXT NOT NULL,
  action TEXT NOT NULL,
  risk_mode TEXT,
  detail JSONB,
  created_at TIMESTAMPTZ DEFAULT NOW()
);
```

## Persona Injection (SOUL Integration)

SessionManager resolves persona via 3 strategies (CTO-D10):

1. **Strategy A** (native agent file): Check `.claude/agents/{role}.md` in project → use `--agent` flag
2. **Strategy B** (temp file): Write SOUL body to `~/.mtclaw/sessions/{dir}/soul.md` → use `--append-system-prompt-file`
3. **Strategy C** (bare): No AgentRole → launch without persona

Stale detection: `.soul-hash` sidecar file (warning only).

## Admission Control

Resource limits enforced at session creation:

| Parameter | Default | Env Override |
|-----------|---------|-------------|
| `max_sessions_per_agent` | 2 | config.json bridge.admission |
| `max_total_sessions` | 6 | config.json bridge.admission |
| `per_tenant_session_cap` | 4 | config.json bridge.admission |
| `per_project_singleton` | false | config.json bridge.admission |
| `max_cpu_percent` | 85% | config.json bridge.admission |
| `max_memory_percent` | 80% | config.json bridge.admission |

## Configuration

### config.json

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

### Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `MTCLAW_BRIDGE_ENABLED` | false | Enable bridge |
| `MTCLAW_BRIDGE_HOOK_PORT` | 18792 | Hook server port |
| `MTCLAW_BRIDGE_AUDIT_DIR` | ~/.mtclaw/bridge-audit | Audit JSONL directory |

## Session Persistence (Sprint 26)

- **Memory primary**: `SessionManager.sessions` map (low-latency)
- **PG secondary**: `BridgeSessionStore` (persistence across restarts)
- Best-effort PG writes — log errors, don't fail operations
- On restart: `LoadFromStore()` recovers non-stopped sessions as `disconnected`

## Multi-Tenant Isolation

- Session IDs include tenant hash: `br:{sha256(tenant)[:8]}:{random}`
- All SessionManager operations enforce `TenantIDFromContext(ctx)`
- HookSecret stripped from `ListSessions` results
- PG RLS policies via `owner_id` column

## Future: Channel-Agnostic Command Extraction

Currently `/cc` commands are Telegram-only (`internal/channels/telegram/commands_cc.go`). Future extraction to a channel-agnostic handler would enable bridge commands across all channels (MS Teams, Zalo, etc.). This is tracked as a Sprint 28+ candidate.
