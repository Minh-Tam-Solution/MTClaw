# Sprint 13 — @coder Handoff

**Sprint**: 13 — Claude Code Bridge: Pre-Sprint + Local Session Core (A1)
**Date**: 2026-04-07
**From**: [@pm] + [@architect]
**To**: [@coder]
**CTO Approval**: Sprint 12 APPROVED 8.4/10 (2026-03-07) — Sprint 13 UNBLOCKED
**Sprint Plan**: `docs/04-build/sprints/SPRINT-013-Claude-Code-Bridge-A1.md`
**ADR**: `docs/02-design/01-ADRs/SPEC-0010-ADR-010-Claude-Code-Bridge.md`

---

## What's Already Done (Pre-Sprint Architect Work)

| Deliverable | Status |
|-------------|--------|
| ADR-010 committed (D1-D9, L1-L4 lock items) | Done |
| `DefaultDenyPatterns()` exported in shell.go | Done |
| `internal/claudecode/types.go` created (all types, D2/D8/D9) | Done |
| `internal/claudecode/config.go` created (BridgeConfig + defaults) | Done |
| Sprint 13-17 plans written | Done |

---

## Sprint 13 Goal

**Stand up the local tmux bridge layer — no Telegram, no network, no hooks.**

Debug the process/tmux layer in isolation before adding security layers in Sprint 14.

---

## MUST READ FIRST

1. **ADR-010**: `docs/02-design/01-ADRs/SPEC-0010-ADR-010-Claude-Code-Bridge.md`
   - D2: 3-axis capability model (lines 34-60)
   - D8: Session ownership model (lines 96-108)
   - D9: ProviderAdapter interface (lines 110-128)

2. **Sprint 13 Plan**: `docs/04-build/sprints/SPRINT-013-Claude-Code-Bridge-A1.md`
   - Day-by-day breakdown with file list

3. **CTO-approved plan**: See ADR-010 + sprint plan (above) for all design decisions.
   Package structure, key types, and database schema are in ADR-010 sections "Database Schema" and "Admission Control".

---

## Execution Order (4 Days)

### Day 1: T13-PRE-01 — ccpoke Source Audit

**Goal**: Document what to port from ccpoke and what to skip.

| Deliverable | Output |
|-------------|--------|
| Audit document | `docs/02-design/bridge-ccpoke-audit.md` |

**Required sections**:
1. tmux edge cases (paste-buffer race conditions, session naming conflicts)
2. Telegram API quirks (rate limits, message size limits, edit timing)
3. Session state transition bugs discovered in ccpoke
4. Hook payload format gotchas (JSON encoding, escaping)
5. **"Do Not Port" section** — features that conflict with MTClaw governance model

**Time-box**: 4 hours. Document known gaps, don't block on completeness.

### Day 2: T13-A1-01 — Provider + Config Wiring + Types Tests

**Goal**: ProviderAdapter interface compiles. Config wired. Types tested.

Files to create:
| File | Purpose | Status |
|------|---------|--------|
| `internal/claudecode/provider.go` | ProviderAdapter interface + ClaudeCodeAdapter | Create |
| `internal/claudecode/types_test.go` | ~8 tests: ID generation, capability mapping, hook secret | Create |

Files to modify:
| File | Change | Status |
|------|--------|--------|
| `internal/config/config.go` | Add `Bridge BridgeConfig` field to Config struct | Modify |

**ProviderAdapter interface** (from ADR-010 D9):

```go
type ProviderAdapter interface {
    Name() AgentProviderType
    LaunchCommand(workdir, hookURL, secret string) *exec.Cmd
    InstallHooks(hookURL, secret string) error
    UninstallHooks() error
    ParseStopEvent(payload []byte) (*StopEvent, error)
    CapabilityProfile() ProviderCapabilities
    TranscriptPath(sessionDir string) string
}
```

**ClaudeCodeAdapter** implementation notes:
- `LaunchCommand`: `claude --dangerously-skip-permissions` with hook env vars
- `InstallHooks`: Write Claude Code hooks to `~/.claude/hooks/`
- `TranscriptPath`: `~/.claude/projects/{workdir}/*.jsonl`
- `CapabilityProfile`: `PermissionHooks: true, TranscriptParsing: true, HookFormatVersion: 1`

**Config wiring** (`internal/config/config.go`):

```go
// Add to Config struct:
Bridge claudecode.BridgeConfig `json:"bridge,omitempty"`
```

**Import cycle verified by [@architect]**: `config` -> `claudecode` is safe. `config` already imports `cron` and `sandbox` with the same one-way pattern. `claudecode` does NOT import `config`. No cycle.

### Day 3: T13-A1-02 — Tmux Bridge + Project Registry + Tests

**Goal**: tmux operations work from Go. Tmux + project tests pass.

Files to create:
| File | Purpose | Status |
|------|---------|--------|
| `internal/claudecode/tmux.go` | TmuxBridge struct with all tmux operations | Create |
| `internal/claudecode/project.go` | Project registry + composite workspaceFingerprint | Create |
| `internal/claudecode/tmux_test.go` | ~10 tests: command construction (not live tmux) | Create |
| `internal/claudecode/project_test.go` | ~6 tests: fingerprint determinism, project CRUD | Create |

**TmuxBridge implementation** (port from ccpoke):

```go
type TmuxBridge struct {
    tmuxPath string // path to tmux binary
}

// Key methods:
func (t *TmuxBridge) CreateSession(name, workdir string) error
func (t *TmuxBridge) KillSession(target string) error
func (t *TmuxBridge) CapturePane(target string, lines int) (string, error)
func (t *TmuxBridge) SendKeys(target string, keys string) error
func (t *TmuxBridge) ListSessions() ([]TmuxSession, error)
func (t *TmuxBridge) SessionExists(target string) (bool, error)
```

**Critical tmux patterns**:
- `SendKeys`: Use `tmux load-buffer - <<< "text"` + `tmux paste-buffer -t {target}` (NOT `send-keys`)
- `CapturePane`: `tmux capture-pane -p -t {target} -S -{lines}`
- All commands: `exec.CommandContext(ctx, ...)` with 5s timeout
- Session naming: `cc-{tenant8}-{rand8}` (keep under tmux 256 char limit)

**WorkspaceFingerprint**:
```go
func ComputeWorkspaceFingerprint(projectPath, tenantID string) (string, error)
```
Composite: `sha256(canonical_path + ":" + device_inode + ":" + git_root + ":" + remote_url_normalized + ":" + tenant_salt)`

### Day 4: T13-A1-03 — Session + Manager + Tests

**Goal**: Full session lifecycle. Remaining tests pass. All green.

Files to create:
| File | Purpose | Status |
|------|---------|--------|
| `internal/claudecode/session.go` | Session state machine + message queue + ownership (D8) | Create |
| `internal/claudecode/session_manager.go` | In-memory lifecycle + admission control | Create |
| `internal/claudecode/session_test.go` | ~12 tests: state transitions, ownership rules | Create |
| `internal/claudecode/session_manager_test.go` | ~10 tests: admission control, tenant isolation | Create |

**Session state machine**:
```
active -> busy -> idle -> stopped
active -> error -> stopped
any -> stopped (via kill)
```

**SessionManager key methods**:
```go
type SessionManager struct { ... }

func NewSessionManager(cfg BridgeConfig) *SessionManager
func (m *SessionManager) CreateSession(ctx context.Context, opts CreateSessionOpts) (*BridgeSession, error)
func (m *SessionManager) GetSession(ctx context.Context, sessionID string) (*BridgeSession, error)
func (m *SessionManager) ListSessions(ctx context.Context, tenantID string) ([]*BridgeSession, error)
func (m *SessionManager) KillSession(ctx context.Context, sessionID, actorID string) error
func (m *SessionManager) UpdateRiskMode(ctx context.Context, sessionID string, mode RiskMode, actorID string) error
```

**Admission control** (checked on CreateSession):
```go
func (m *SessionManager) checkAdmission(ctx context.Context, opts CreateSessionOpts) error
```
Checks: MaxSessionsPerAgent, MaxTotalSessions, MaxCPUPercent, MaxMemoryPercent, PerTenantSessionCap, PerProjectSingleton. Returns clear error with which limit hit.

**Test total across Days 2-4**: ~46 tests (types: Day 2, tmux+project: Day 3, session+manager: Day 4)

---

## Files Summary

### Already Created (by [@architect])

| File | Purpose |
|------|---------|
| `internal/claudecode/types.go` | All types (AgentProviderType, SessionState, RiskMode, Capabilities, BridgeSession, etc.) |
| `internal/claudecode/config.go` | BridgeConfig struct + DefaultBridgeConfig() |
| `internal/tools/shell.go` | `DefaultDenyPatterns()` export added |
| `docs/02-design/01-ADRs/SPEC-0010-ADR-010-Claude-Code-Bridge.md` | ADR with D1-D9, L1-L4 |

### Create (6 files)

| File | Task | Day |
|------|------|-----|
| `docs/02-design/bridge-ccpoke-audit.md` | ccpoke source audit | 1 |
| `internal/claudecode/provider.go` | ProviderAdapter + ClaudeCodeAdapter | 2 |
| `internal/claudecode/tmux.go` | TmuxBridge operations | 3 |
| `internal/claudecode/project.go` | Project registry + fingerprint | 3 |
| `internal/claudecode/session.go` | Session state machine | 4 |
| `internal/claudecode/session_manager.go` | Multi-tenant lifecycle | 4 |

### Create (5 test files)

| File | Day |
|------|-----|
| `internal/claudecode/types_test.go` | 2 |
| `internal/claudecode/tmux_test.go` | 3 |
| `internal/claudecode/project_test.go` | 3 |
| `internal/claudecode/session_test.go` | 4 |
| `internal/claudecode/session_manager_test.go` | 4 |

### Modify (1 file)

| File | Change |
|------|--------|
| `internal/config/config.go` | Add `Bridge` field |

---

## Key Code References

Use `grep -n` to find exact locations (line numbers may shift between sprints):

| What | File | Grep anchor | Why You Need It |
|------|------|-------------|-----------------|
| `DefaultDenyPatterns()` | `internal/tools/shell.go` | `grep -n "func DefaultDenyPatterns"` | Reuse for input sanitizer (Sprint 14) |
| `ExecApprovalManager` | `internal/tools/exec_approval.go` | `grep -n "type ExecApprovalManager"` | Pattern for permission store (Sprint 16) |
| `AES-256-GCM Encrypt/Decrypt` | `internal/crypto/aes.go` | `grep -n "func Encrypt\|func Decrypt"` | Hook secret encryption |
| `Config struct` | `internal/config/config.go` | `grep -n "type Config struct"` | Add Bridge field here |
| `CronService file store` | `internal/cron/service.go` | `grep -n "type Service struct"` | Pattern for standalone bridge store |
| `Channel struct` | `internal/channels/telegram/channel.go` | `grep -n "type Channel struct"` | Add bridgeManager field (Sprint 14) |
| Session types | `internal/claudecode/types.go` | `grep -n "type BridgeSession"` | All types already defined |

---

## Verification Checklist

```bash
# 1. Build
make build

# 2. All tests (existing + new)
make test

# 3. claudecode package tests
go test ./internal/claudecode/... -v -race -count=1

# 4. Verify types compile
go vet ./internal/claudecode/...

# 5. Verify config integration
grep -n "Bridge" internal/config/config.go

# 6. Count tests
go test ./internal/claudecode/... -v 2>&1 | grep -c "=== RUN"
# Expected: >= 40

# 7. ccpoke audit exists
test -f docs/02-design/bridge-ccpoke-audit.md && echo "OK" || echo "MISSING"
```
