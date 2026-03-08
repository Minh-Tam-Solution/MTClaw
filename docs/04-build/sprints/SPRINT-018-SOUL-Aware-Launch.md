---
sprint: 18
title: SOUL-Aware Bridge Launch
status: PLANNED
date: 2026-03-07
version: "1.0.0"
author: "[@pm] + [@architect]"
framework: SDLC Enterprise Framework 6.1.1
tier: STANDARD
---

# Sprint 18 — SOUL-Aware Bridge Launch

**Sprint**: 18 of 23 (bridge intelligence track: 18 of 18-23)
**Phase**: 4 (Bridge Intelligence — ADR-011)
**Duration**: 5 days
**Owner**: [@coder] (implementation) + [@pm] (coordination)
**Points**: ~10 (7 implementation + 3 tests)
**Gate**: Sprint gate — `/cc launch myproject --as coder` injects SOUL, `/cc sessions` shows persona provenance
**ADR**: `docs/02-design/01-ADRs/SPEC-0011-ADR-011-SOUL-Aware-Bridge-Launch.md`
**Plan**: `/home/dttai/.claude/plans/glowing-gliding-quill.md`
**CTO Approval**: Sprint 18 APPROVED 9.3/10 (2026-03-07, Round 3)

---

## 1. Entry Criteria

| Criterion | Status | Owner |
|-----------|--------|-------|
| CTO Sprint 17 review score received | CLEARED | [@cto] |
| ADR-011 committed | Done (2026-03-07) | [@architect] |
| Sprint 13-17 bridge tests passing (150+) | Done | [@coder] |
| `bridge.enabled = true` in config.json | Done | [@architect] |
| `bridge status` shows 7/7 PASS | Done | [@coder] |
| B4-B7 blocker resolutions in ADR-011 | Done | [@architect] |

---

## 2. Sprint Goal

**Inject SOUL personas into Claude Code bridge sessions via Strategy A/B/C cascade with full audit provenance.**

### Key Outcomes

1. `soul_loader.go` loads SOUL files with frontmatter parsing + hash tracking
2. `LaunchOpts` struct replaces rigid `LaunchCommand(workdir, hookURL, secret)` signature
3. `CreateSession` resolves Strategy A/B/C and launches Claude Code with persona
4. `/cc launch myproject --as coder` works from Telegram
5. `mtclaw bridge install-agents <path>` generates `.claude/agents/*.md` files
6. `BridgeConfig.SoulsDir` wired from config.json to session manager
7. ~28 new tests covering soul loading, strategy resolution, install-agents

---

## 3. Architecture — [@architect]

### 3.1 New Files

```
internal/claudecode/
  soul_loader.go                    -- SoulContent, LoadSOUL, KnownRoles
  soul_loader_test.go               -- ~10 tests
  bridge_agent_templates.json       -- //go:embed category->tool/model mappings
```

### 3.2 Modified Files

```
internal/claudecode/
  types.go                          -- Add AgentRole, SoulTemplateHash, PersonaSourceHash, PersonaSource
  config.go                         -- Add SoulsDir to BridgeConfig
  provider.go                       -- LaunchOpts struct, update interface + all adapters
  session_manager.go                -- SOUL loading + strategy resolution in CreateSession

internal/channels/telegram/
  commands_cc.go                    -- Parse --as <role> in /cc launch

cmd/
  bridge.go                         -- Add install-agents subcommand
```

### 3.3 Key Design Decisions (from ADR-011)

- **D10**: Strategy A/B/C cascade with deterministic resolution
- **D11**: Bridge is control surface, not inference path — Bflow AI-Platform exempt
- **D12**: LaunchCommand builds cmd, SendKeys executes in tmux pane
- **D13**: Duplicate frontmatter parser from `skills/loader.go` (~20 LOC)
- **D14**: `SoulsDir` in BridgeConfig, default `docs/08-collaborate/souls`

### 3.4 Interface Change (ATOMIC COMMIT)

```go
// OLD (Sprint 13-17):
LaunchCommand(workdir, hookURL, secret string) *exec.Cmd

// NEW (Sprint 18):
LaunchCommand(opts LaunchOpts) *exec.Cmd
```

All 4 adapters (ClaudeCodeAdapter + 3 StubAdapters) must be updated in the same commit. All callers of `LaunchCommand` must be updated simultaneously.

---

## 4. Task Breakdown

### Day 1: T18-01 — Soul Loader + Config Wiring

**Goal**: SOUL files loadable from Go. BridgeConfig has SoulsDir.

| File | Task | Status |
|------|------|--------|
| `internal/claudecode/soul_loader.go` | CREATE — SoulContent, LoadSOUL, KnownRoles, frontmatter parser | Create |
| `internal/claudecode/config.go` | MODIFY — Add `SoulsDir string` to BridgeConfig + default | Modify |

**soul_loader.go implementation**:

```go
type SoulContent struct {
    Role        string // "pm", "coder", etc.
    Category    string // from YAML frontmatter: "executor", "advisor", "router", "business"
    Body        string // markdown body (after frontmatter stripped)
    ContentHash string // SHA-256 of full file content
    SourcePath  string // absolute path to SOUL file
}

func KnownRoles(soulsDir string) ([]string, error) // scan SOUL-*.md
func LoadSOUL(soulsDir, role string) (*SoulContent, error)
```

**Frontmatter parser** — duplicate from `internal/skills/loader.go:298-353`:
- `frontmatterRe` regex: `(?s)^---\n(.*?)\n---\n?`
- `extractFrontmatter(content string) string`
- `stripFrontmatter(content string) string`
- `parseSimpleYAML(content string) map[string]string`

**Path traversal guard** (CTO-B3):
1. Validate role in `KnownRoles()` allowlist FIRST
2. `filepath.Join(soulsDir, "SOUL-"+role+".md")`
3. `strings.HasPrefix(filepath.Clean(resolved), filepath.Clean(soulsDir))`

**KnownRoles cache** (CTO-L4): Use `sync.Once` for hot path. Invalidate on `install-agents` run.

**Config wiring** (CTO-B7):
```go
type BridgeConfig struct {
    // ... existing fields ...
    SoulsDir string `json:"souls_dir,omitempty"` // default: "docs/08-collaborate/souls"
}
```

### Day 2: T18-02 — Types + Provider Interface Change

**Goal**: `LaunchOpts` compiles. Interface change atomic. All adapters updated.

| File | Task | Status |
|------|------|--------|
| `internal/claudecode/types.go` | MODIFY — Add 4 fields to BridgeSession + 1 to CreateSessionOpts | Modify |
| `internal/claudecode/provider.go` | MODIFY — LaunchOpts struct + interface change + all adapters | Modify |

**BridgeSession additions**:
```go
AgentRole         string `json:"agent_role,omitempty"`
SoulTemplateHash  string `json:"soul_template_hash,omitempty"`
PersonaSourceHash string `json:"persona_source_hash,omitempty"`
PersonaSource     string `json:"persona_source,omitempty"` // "agent_file" | "append_prompt" | "bare"
```

**CreateSessionOpts addition** (note: this struct is in `session_manager.go:16`, NOT `types.go`):
```go
AgentRole string // optional SOUL role to inject
```

**LaunchOpts struct**:
```go
type LaunchOpts struct {
    Workdir    string
    HookURL    string
    Secret     string
    AgentRole  string // empty = bare launch
    AgentFile  string // Strategy A: path to .claude/agents/{role}.md
    PromptFile string // Strategy B: path to temp SOUL file
}
```

**ClaudeCodeAdapter.LaunchCommand(opts LaunchOpts)**:
```go
args := []string{"--dangerously-skip-permissions"}
if opts.AgentFile != "" {
    args = append(args, "--agent", opts.AgentRole)
} else if opts.PromptFile != "" {
    args = append(args, "--append-system-prompt-file", opts.PromptFile)
}
cmd := exec.Command("claude", args...)
```

**StubAdapter.LaunchCommand(opts LaunchOpts)** — accept LaunchOpts, still return nil.

### Day 3: T18-03 — Session Manager Integration

**Goal**: CreateSession resolves Strategy A/B/C. Launches Claude Code with persona.

| File | Task | Status |
|------|------|--------|
| `internal/claudecode/session_manager.go` | MODIFY — SOUL loading + strategy resolution + launch wiring | Modify |

**CreateSession new flow** (after admission control, after tmux session creation):

```go
// Step 1: Resolve SOUL (if --as specified)
var soul *SoulContent
var personaSource string
var soulHash, personaHash string

if opts.AgentRole != "" {
    soul, err = LoadSOUL(m.cfg.SoulsDir, opts.AgentRole)
    if err != nil {
        return nil, fmt.Errorf("load SOUL: %w", err)
    }
    soulHash = soul.ContentHash
}

// Step 2: Resolve Strategy A/B/C
var launchOpts LaunchOpts
launchOpts.Workdir = opts.ProjectPath
launchOpts.HookURL = hookURL
launchOpts.Secret = hookSecret

if soul != nil {
    agentFile := filepath.Join(opts.ProjectPath, ".claude", "agents", opts.AgentRole+".md")
    if fileExists(agentFile) {
        // Strategy A
        launchOpts.AgentFile = agentFile
        launchOpts.AgentRole = opts.AgentRole
        personaSource = "agent_file"
        personaHash = hashFile(agentFile)
    } else {
        // Strategy B
        tempFile := writeTempSOUL(sessionID, soul.Body)
        launchOpts.PromptFile = tempFile
        personaSource = "append_prompt"
        personaHash = soul.ContentHash
    }
} else {
    // Strategy C
    personaSource = "bare"
}

// Step 3: Build and launch command in tmux pane (D12)
cmd := adapter.LaunchCommand(launchOpts)
if cmd != nil && m.tmux != nil {
    cmdStr := strings.Join(cmd.Args, " ")
    m.tmux.SendKeys(ctx, session.TmuxTarget, cmdStr)
    m.tmux.SendEnter(ctx, session.TmuxTarget)
}

// Step 4: Store persona info in session
session.AgentRole = opts.AgentRole
session.SoulTemplateHash = soulHash
session.PersonaSourceHash = personaHash
session.PersonaSource = personaSource
```

**Temp file lifecycle** (Strategy B):
- Path: `~/.mtclaw/sessions/{sanitizedSessionID}/soul.md`
- `sanitizedSessionID = strings.ReplaceAll(sessionID, ":", "-")` (CTO-B2)
- Permissions: `0600`
- Cleanup: in `KillSession()` remove session directory
- Orphan sweep: existing cleanup goroutine (10min ticker)

**Stale detection** (CTO-M4):
- If Strategy A: compare `soulHash` with `personaHash`
- If mismatch: `slog.Warn("agent file may be stale", "soul_hash", soulHash, "agent_hash", personaHash)`
- Don't block launch

**Audit extension** (CTO-M3):
- `WriteSessionCreated()` detail includes: `agent_role`, `persona_source`, `soul_template_hash`, `persona_source_hash`, `strategy`, `inference_path: "external_claude_code"` (D11)

### Day 4: T18-04 — Telegram Command + Install-Agents

**Goal**: `/cc launch --as` works. `install-agents` generates agent files.

| File | Task | Status |
|------|------|--------|
| `internal/channels/telegram/commands_cc.go` | MODIFY — Parse `--as <role>` | Modify |
| `cmd/bridge.go` | MODIFY — Add `install-agents` subcommand | Modify |
| `internal/claudecode/bridge_agent_templates.json` | CREATE — Category->tool/model mappings | Create |

**`/cc launch` parsing**:
```
/cc launch myproject --as coder    -> AgentRole="coder"
/cc launch myproject --as pm       -> AgentRole="pm"
/cc launch myproject               -> AgentRole="" (bare)
```
Validation: role must be in `KnownRoles()`. Reject with error listing valid roles.

**install-agents** (`mtclaw bridge install-agents <project-path> [--souls-dir <path>] [--roles pm,coder,architect]`):

For each SOUL in soulsDir (filtered by `--roles` if specified):
1. Load SOUL via `LoadSOUL()`
2. Generate `.claude/agents/{role}.md`:
   - YAML frontmatter: `name`, `description`, `tools`, `model` from template config
   - Body: SOUL markdown content
   - Header: `# Generated by mtclaw bridge install-agents — do not edit manually`
   - Footer: `# Commit this file if the team shares SOUL-aware Claude sessions`
3. Skip files without generated header (user's own agents)
4. Update generated files only when SOUL hash changes
5. Report: installed/updated/skipped counts
6. `claude --version` check: warn if mismatch with `claude_code_version_min` (CTO-M6)

**bridge_agent_templates.json** (CTO-B1 — `//go:embed`):
```json
{
  "_comment": "Tool/model mappings for claude-code >= 2.x",
  "claude_code_version_min": "2.0",
  "categories": {
    "executor": { "tools": ["Read","Edit","Write","Bash","Grep","Glob"], "model": "sonnet" },
    "advisor": { "tools": ["Read","Grep","Glob"], "model": "opus" },
    "router": { "tools": ["Read","Grep","Glob","Bash"], "model": "inherit" },
    "business": { "tools": ["Read","Grep","Glob"], "model": "sonnet" }
  },
  "role_overrides": {
    "architect": { "model": "opus" },
    "cto": { "model": "opus" },
    "cpo": { "model": "opus" },
    "ceo": { "model": "opus" }
  }
}
```

### Day 5: T18-05 — Tests + Integration Verification

**Goal**: All 28 tests pass. Build clean. No regression.

| File | Tests | Key Assertions |
|------|-------|---------------|
| `internal/claudecode/soul_loader_test.go` | ~10 | Frontmatter parsing, hash computation, all 18 roles, missing file, path traversal guard, KnownRoles cache |
| `internal/claudecode/provider_test.go` (extend) | ~5 | `--agent` flag when file exists, `--append-system-prompt-file` fallback, bare launch, LaunchOpts |
| `internal/claudecode/session_manager_test.go` (extend) | ~6 | AgentRole stored, PersonaSource resolved, SoulContentHash populated, stale agent file warning, Strategy B temp file cleanup on kill |
| `internal/channels/telegram/commands_cc_test.go` (extend) | ~4 | `--as` parsing, invalid role rejection, no `--as` = bare, valid roles listed |
| `cmd/bridge_test.go` (extend) | ~3 | install-agents creates files, idempotent, skips user files |

---

## 5. Acceptance Criteria

| # | Criterion | Verification |
|---|-----------|-------------|
| 1 | `LoadSOUL("coder")` returns SoulContent with role, category, body, hash | Unit test |
| 2 | `KnownRoles()` returns 18 roles from filesystem scan | Unit test |
| 3 | Path traversal `LoadSOUL("../etc/passwd")` rejected | Unit test |
| 4 | `LaunchCommand(opts)` includes `--agent` when AgentFile set | Unit test |
| 5 | `LaunchCommand(opts)` includes `--append-system-prompt-file` when PromptFile set | Unit test |
| 6 | `CreateSession` with `--as coder` stores AgentRole + hashes | Unit test |
| 7 | Strategy A/B/C deterministic: same inputs -> same strategy | Unit test |
| 8 | `/cc launch myproject --as coder` parses correctly | Unit test |
| 9 | `install-agents` creates `.claude/agents/coder.md` | Unit test |
| 10 | `install-agents` skips user-created agent files | Unit test |
| 11 | Stale agent file: warning logged, launch not blocked | Unit test |
| 12 | `BridgeConfig.SoulsDir` wired from config.json | Compile + config test |
| 13 | `make build && make test` passes (zero regression) | CI gate |

---

## 6. Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| `--agent` flag behavior changes in Claude Code update | Low | High | `claude_code_version_min` check in templates |
| SOUL files missing from workspace | Low | Medium | Strategy C (bare) fallback |
| Interface change breaks existing tests | Medium | Medium | Atomic commit, update all callers |
| `//go:embed` binary size increase | Low | Low | JSON file ~500 bytes |

---

## 7. NOT in Sprint 18

| Item | Reason | Sprint |
|------|--------|--------|
| SessionIntelligenceEnvelope | Sprint 19 scope | 19 |
| `/cc info` command | Sprint 19 scope | 19 |
| Turn-time context injection | Sprint 20B scope | 20B |
| `--model` per-role override at launch | Sprint 21 scope | 21 |
| Agent teams integration | Sprint 22 research spike | 22 |
| Role-aware capability defaults | Sprint 21 scope | 21 |
| SOUL DB loading for bridge | Bridge reads files, agent loop reads DB | N/A |
| Automatic agent file watching | install-agents is run once | N/A |

---

## 8. Verification Checklist

```bash
# 1. Build
make build

# 2. All tests
make test

# 3. claudecode package tests
go test ./internal/claudecode/... -v -race -count=1

# 4. Verify soul loader
go test ./internal/claudecode/ -run TestSoulLoader -v

# 5. Verify provider interface change
go vet ./internal/claudecode/...

# 6. Verify config wiring
grep -n "SoulsDir" internal/claudecode/config.go

# 7. Count new tests
go test ./internal/claudecode/... -v 2>&1 | grep -c "=== RUN"
# Expected: >= 178 (150 existing + 28 new)

# 8. Manual: install agents into test project
./mtclaw bridge install-agents /path/to/project
ls /path/to/project/.claude/agents/
```
