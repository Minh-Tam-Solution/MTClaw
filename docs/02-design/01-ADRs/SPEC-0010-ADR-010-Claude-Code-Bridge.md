# ADR-010: Claude Code Terminal Bridge

**SPEC ID**: SPEC-0010
**Status**: ACCEPTED
**Date**: 2026-03-06
**Deciders**: [@cto], [@cpo], [@ceo]
**Tag**: `adr-010-ccbridge`

---

## Context

Team MTS uses Claude Code as primary coding tool. Currently, developers must sit at the terminal to interact with Claude Code sessions. The standalone tool `ccpoke` (Node.js, 186 commits) provides 2-way Telegram interaction but is single-tenant, standalone, and lacks governance integration.

**Goal**: Integrate notification bridge + 2-way interaction natively into MTClaw's Go binary with multi-tenant governance.

**Strategy**: PORT 70% commodity mechanics (tmux, hooks, sessions) + BUILD 30% unique value (capability-based permission, tenant isolation, governance reuse).

**Non-negotiable invariant**: Bridge = control surface, NOT terminal surrogate. No command exposes arbitrary shell pane access.

---

## Decisions

### D1. Bridge = "Notification + Input relay", NOT "remote terminal"

- Shell panes disabled (`shellPanesDisabled = true`)
- sendKeys uses positive allowlist by capability model (not just negative blocklist)
- Sprint A: structured commands only. Free-text deferred to Sprint D (after permission approval is proven)

### D2. Capability Model (3-Axis)

**UX layer**: `RiskMode` (read/patch/interactive) for simple user-facing control.

**Internal enforcement**: 3 orthogonal axes:

| Axis | Values | Governs |
|------|--------|---------|
| `InputMode` | `structured_only` / `free_text` | What user can type into sendKeys |
| `ToolPolicy` | `observe` / `patch_allowed` / `exec_with_approval` | What tool calls agent is expected to make |
| `OutputPolicy` | 30 lines heavy redaction / 50 lines standard / 100 lines standard | What capturePane returns |

**RiskMode -> Capability mapping**:

| RiskMode | InputMode | ToolPolicy | OutputPolicy | Who authorizes |
|----------|-----------|------------|--------------|----------------|
| `read` | structured_only | observe | 30 lines, heavy redaction | Default |
| `patch` | structured_only | patch_allowed | 50 lines, standard | Session owner (`/cc risk patch`) |
| `interactive` | free_text | exec_with_approval | 100 lines, standard | Tenant admin + provider must support permission hooks |

**Escalation flow**:
- `/cc risk read` — anyone can downgrade
- `/cc risk patch` — session owner self-escalate, audit logged
- `/cc risk interactive` — tenant admin only, provider capability checked (D7 Layer 0), audit logged with `{action: "risk_escalate", from, to, actor_id, session_id}`
- Per-session. New sessions always start at `read`.

**Key**: Primary defense = capability gating on tool layer (ToolPolicy). Input sanitizer is secondary defense.

### D3. Telegram identity binding = non-negotiable

- `/cc link` required before `/cc launch`
- Every request carries `actor_id` -> audit log
- Cross-tenant notification isolation: Actor A cannot receive notifications of sessions of tenant B

### D4. HookServer = separate from gateway

- `127.0.0.1:18792` (localhost only, configurable via `GOCLAW_BRIDGE_HOOK_PORT`)
- Different lifecycle and security model from gateway
- Gateway starts HookServer if bridge enabled; also standalone via `mtclaw bridge serve`
- Token-bucket rate limiting per session ID

### D5. HMAC-SHA256 hook authentication

- Per-session secret (32 bytes `crypto/rand` hex)
- `HMAC-SHA256(payload + nonce + timestamp, secret)`, 30s window
- Nonce prefixed with sessionId
- Secrets encrypted with AES-256-GCM (`GOCLAW_ENCRYPTION_KEY`, reusing `internal/crypto/aes.go`) before DB storage
- Startup: fail if `GOCLAW_ENCRYPTION_KEY` missing when bridge is enabled

### D6. Async permission polling + fail-safe fallback

- `POST /hook/permission` -> 202 -> poll `GET /hook/permission/{id}` every 1s
- No held HTTP connections (async polling model)
- **Fail-safe fallback (fail-closed for high-risk)**:
  - Low-risk read tools (Read, Glob, Grep): configurable allow on HookServer unreachable
  - High-risk tools (Bash, Edit, Write, Agent): **deny by default** (fail closed)
  - Native terminal fallback ONLY for unmanaged local sessions with verified interactive TTY (`session.LocalInteractive = true`)
  - If no TTY verification: reject + notify Telegram
  - HookServer unreachable must NOT create silent success — explicit test required
- Timeout: 3min default. Low-risk auto-approve, high-risk auto-reject.
- Dedicated audit action: `fallback_allow_low_risk` for degrade-mode decisions
- **Sanitizer failure policy**: reject, not pass-through (`sanitizer_failure_policy = deny`)
- **Fail-closed reason**: machine-readable `reason_code=hook_unreachable_fail_closed` in audit + Telegram message

### D7. Three-Layer Approval Model

- **Layer 0**: Provider capability guard — if provider doesn't support permission hooks, session CANNOT be `interactive`. Escalation rejected with explicit `reason_code` field.
- **Layer 1**: Bridge permission — controls tool calls via async polling
- **Layer 2**: MTClaw governance (existing hooks/policy) — controls SDLC gates, destructive ops
- Bridge CANNOT bypass Layer 2

### D8. Session Ownership Model

- **Creator** = session owner (immutable after creation)
- **Approver set** = explicit, separate from owner. Default: owner only. Can be extended by tenant admin. Concrete actor IDs only (no role expansion in Sprint A-D).
- **Notification recipients** = ACL (max 5 recipients), not auto-derived from active chat. Default: owner + chat where session was created.
- **`/cc switch`** only affects routing for the calling actor, never changes ownership
- **Cross-tenant isolation**: Actor A cannot see/switch/capture/approve sessions of tenant B (enforced via `owner_id` filtering + application checks). ACL mismatch rejections logged as `permission_acl_mismatch` audit event.
- **Session transfer**: not supported in Sprint A-D. No `/cc transfer` command.

### D9. Provider Adapter Contract

```go
type ProviderAdapter interface {
    Name() AgentProviderType
    LaunchCommand(workdir string, hookURL string, secret string) *exec.Cmd
    InstallHooks(hookURL string, secret string) error
    UninstallHooks() error
    ParseStopEvent(payload []byte) (*StopEvent, error)
    CapabilityProfile() ProviderCapabilities
    TranscriptPath(sessionDir string) string
}

type ProviderCapabilities struct {
    PermissionHooks    bool
    TranscriptParsing  bool
    HookFormatVersion  int
}
```

Sprint A-D: only `ClaudeCodeAdapter` implemented. Others registered as stubs returning `PermissionHooks: false`.

---

## Pre-ADR Lock Items

### L1. "Verified interactive TTY" definition

- Signal: `os.Stdin` is a terminal (`term.IsTerminal(fd)`) AND tmux pane has foreground process with controlling TTY
- Stored: `BridgeSession.LocalInteractive bool`, set at session creation, NOT re-evaluated
- Expires: when session transitions to `stopped` or gateway restarts (conservative: re-verify on restart)

### L2. `request_hash` input fields

- Hash input: `sha256(session_id + tool_name + json_canonical(tool_input) + timestamp_minute_bucket)`
- Minute bucket ensures same tool+input within 1 minute is deduped, but allows retry after timeout
- All providers must use same canonical JSON serialization (Go `json.Marshal` with sorted keys)

### L3. Audit dual-write source of truth

- JSONL file = primary (always written, never skipped)
- PostgreSQL = secondary (best-effort, log warning on failure, don't block operation)
- Reconciliation: not supported in Sprint A-D. PG may lag behind JSONL. Document as known limitation.
- For governance queries: PG is queryable but may miss entries. JSONL is complete but requires log parsing.

### L4. Cross-tenant notification isolation

- Actor A cannot see/switch/capture/approve/receive notifications of sessions of tenant B
- NotifyACL validated against tenant boundary on every notification dispatch

---

## Database Schema

```sql
CREATE TABLE bridge_sessions (
    id                    TEXT PRIMARY KEY,
    owner_id              TEXT NOT NULL,
    agent_type            TEXT NOT NULL DEFAULT 'claude-code',
    tmux_target           TEXT NOT NULL,
    project_path          TEXT NOT NULL,
    workspace_fingerprint TEXT NOT NULL,
    status                TEXT NOT NULL DEFAULT 'active',
    risk_mode             TEXT NOT NULL DEFAULT 'read',
    input_mode            TEXT NOT NULL DEFAULT 'structured_only',
    tool_policy           TEXT NOT NULL DEFAULT 'observe',
    owner_actor_id        TEXT NOT NULL,
    approver_acl          TEXT[] NOT NULL DEFAULT '{}',
    notify_acl            TEXT[] NOT NULL DEFAULT '{}',
    user_id               TEXT NOT NULL,
    channel               TEXT NOT NULL,
    chat_id               TEXT NOT NULL,
    hook_secret           TEXT NOT NULL,
    local_interactive     BOOLEAN NOT NULL DEFAULT false,
    created_at            TIMESTAMPTZ DEFAULT NOW(),
    updated_at            TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE bridge_projects (
    id         TEXT PRIMARY KEY,
    owner_id   TEXT NOT NULL,
    name       TEXT NOT NULL,
    path       TEXT NOT NULL,
    agent_type TEXT NOT NULL DEFAULT 'claude-code',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(owner_id, name)
);

CREATE TABLE bridge_permissions (
    id           TEXT PRIMARY KEY,
    owner_id     TEXT NOT NULL,
    session_id   TEXT NOT NULL REFERENCES bridge_sessions(id),
    tool         TEXT NOT NULL,
    risk_level   TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    actor_id     TEXT NOT NULL,
    decision     TEXT,
    expires_at   TIMESTAMPTZ NOT NULL,
    decided_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(request_hash)
);

CREATE TABLE bridge_audit_events (
    id         BIGSERIAL PRIMARY KEY,
    owner_id   TEXT NOT NULL,
    session_id TEXT,
    actor_id   TEXT NOT NULL,
    action     TEXT NOT NULL,
    risk_mode  TEXT,
    detail     JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_bridge_audit_owner_time ON bridge_audit_events(owner_id, created_at DESC);

ALTER TABLE bridge_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE bridge_projects ENABLE ROW LEVEL SECURITY;
ALTER TABLE bridge_permissions ENABLE ROW LEVEL SECURITY;
ALTER TABLE bridge_audit_events ENABLE ROW LEVEL SECURITY;
```

---

## Admission Control

Beyond static `maxSessions`, dynamic admission checks on `/cc launch` only (never auto-kill running sessions):

| Check | Default | Purpose |
|-------|---------|---------|
| MaxSessionsPerAgent | 2 | Per agent type limit |
| MaxTotalSessions | 6 | Global limit |
| MaxCPUPercent | 85% | Host CPU threshold |
| MaxMemoryPercent | 80% | Host memory threshold |
| PerTenantSessionCap | 4 | Multi-tenant fairness |
| PerProjectSingleton | false | One session per project path |

---

## Acceptance Criteria (Sprint A-D)

1. **Cross-tenant isolation**: Actor A cannot see/switch/capture/approve sessions of tenant B
2. **Provider downgrade**: If provider lacks permission hook support, session auto-stays at `read` (cannot escalate to `interactive`), escalation rejected with explicit reason
3. **Workspace integrity**: Rename repo, worktree clone, symlink mount don't cause session to map wrong workspace
4. **Fail-closed**: HookServer down + high-risk tool request => deny, audit logged, Telegram notified
5. **Replay + duplicate callback**: Approve 2x / callback race doesn't double-apply decision (request_hash UNIQUE constraint)
6. **Long-running busy queue**: Session BUSY 20+ min, queue full, late stop event, no out-of-order routing

---

## Consequences

**Positive**:
- Native Go integration eliminates standalone Node.js dependency (ccpoke)
- Multi-tenant governance from Day 1
- Capability model prevents privilege confusion
- Fail-closed security for high-risk operations
- Provider adapter enables future multi-agent support

**Negative**:
- Additional complexity in MTClaw binary (~3K LOC estimated)
- Requires tmux dependency in Docker runtime (`apk add tmux` ~2MB)
- JSONL-to-PG reconciliation not supported in Sprint A-D
- File-to-PG migration path out of scope for Sprint A-D

**Risks**:
- Go competency for security-critical code (mitigated: external security review after Sprint C)
- tmux edge cases (mitigated: ccpoke audit documents known issues)

---

## Reusable Patterns

| Pattern | Source | Reuse For |
|---------|--------|-----------|
| Shell deny patterns | `internal/tools/shell.go:21-127` | Input sanitizer (secondary defense) |
| Approval blocking on channel | `internal/tools/exec_approval.go` | Permission async store |
| Command dispatch | `internal/channels/telegram/commands.go` | /cc commands |
| Bus event broadcast | `internal/bus/bus.go:Broadcast` | Bridge events |
| RLS migration | `migrations/000008` | Bridge tables |
| AES-256-GCM encryption | `internal/crypto/aes.go` | Hook secret encryption |
| Cron pattern | `internal/cron/` | Session cleanup + health |

---

## References

- Plan: `/home/dttai/.claude/plans/glowing-gliding-quill.md`
- ccpoke: Standalone Node.js tool (186 commits, source audit pending)
- CTO Review: Approved with conditions (2026-03-06)
- CPO Review: Approved after 3-axis capability model revision
