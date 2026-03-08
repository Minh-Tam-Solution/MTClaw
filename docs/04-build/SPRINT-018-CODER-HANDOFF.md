# Sprint 18 — @coder Handoff

**Sprint**: 18 — SOUL-Aware Bridge Launch
**Date**: 2026-03-07
**From**: [@pm] + [@architect]
**To**: [@coder]
**CTO Approval**: Sprint 18 APPROVED 9.3/10 (2026-03-07, Round 3 Final)
**Sprint Plan**: `docs/04-build/sprints/SPRINT-018-SOUL-Aware-Launch.md`
**ADR**: `docs/02-design/01-ADRs/SPEC-0011-ADR-011-SOUL-Aware-Bridge-Launch.md`
**Master Plan**: `/home/dttai/.claude/plans/glowing-gliding-quill.md`

---

## What's Already Done (Pre-Sprint Architect Work)

| Deliverable | Status |
|-------------|--------|
| ADR-011 committed (D10-D14, L5-L7, B4-B7 resolutions) | Done |
| Sprint 18 plan written with 5-day breakdown | Done |
| Sprint 19-23 directional plans written | Done |
| CTO B4-B7 blockers all resolved in ADR-011 | Done |
| `bridge.enabled = true` in config.json | Done |
| `bridge status` shows 7/7 PASS | Done |
| `config.Load(resolveConfigPath())` bug fix in bridge.go | Done |

---

## Sprint 18 Goal

**Inject SOUL personas into Claude Code bridge sessions via Strategy A/B/C cascade with full audit provenance.**

After this sprint: `/cc launch myproject --as coder` creates a Claude Code session with the coder SOUL persona injected, tracked with dual-hash integrity.

---

## MUST READ FIRST

1. **ADR-011**: `docs/02-design/01-ADRs/SPEC-0011-ADR-011-SOUL-Aware-Bridge-Launch.md`
   - D10: Strategy A/B/C cascade
   - D11: Bflow AI-Platform exemption (bridge = control surface)
   - D12: LaunchCommand wiring via SendKeys
   - D13: Frontmatter parser from `skills/loader.go` (NOT `bootstrap/files.go`)
   - D14: SoulsDir in BridgeConfig

2. **Sprint 18 Plan**: `docs/04-build/sprints/SPRINT-018-SOUL-Aware-Launch.md`
   - Day-by-day breakdown with file list

3. **Existing Code to Understand**:
   - `internal/claudecode/session_manager.go` — CreateSession flow (lines 54-135)
   - `internal/claudecode/provider.go` — LaunchCommand interface (lines 13-21, 30-39)
   - `internal/claudecode/types.go` — BridgeSession struct (lines 105-126)
   - `internal/skills/loader.go` — Frontmatter parser to duplicate (lines 298-353)
   - `docs/08-collaborate/souls/SOUL-coder.md` — SOUL frontmatter format

---

## Execution Order (5 Days)

### Day 1: T18-01 — Soul Loader + Config Wiring

**Goal**: SOUL files loadable. BridgeConfig has SoulsDir.

Files to create:
| File | Purpose |
|------|---------|
| `internal/claudecode/soul_loader.go` | SoulContent, LoadSOUL, KnownRoles, frontmatter parsing |

Files to modify:
| File | Change |
|------|--------|
| `internal/claudecode/config.go` | Add `SoulsDir string` field + default |

**Implementation details**:

```go
// SoulContent — loaded SOUL data with integrity hash
type SoulContent struct {
    Role        string // from frontmatter
    Category    string // from frontmatter: "executor", "advisor", "router", "business"
    Body        string // markdown body (frontmatter stripped)
    ContentHash string // SHA-256 of full file content
    SourcePath  string // absolute path
}

func KnownRoles(soulsDir string) ([]string, error)         // scan SOUL-*.md
func LoadSOUL(soulsDir, role string) (*SoulContent, error)  // load + parse + hash
```

**Frontmatter parser** — copy from `internal/skills/loader.go:298-353`:
```go
var frontmatterRe = regexp.MustCompile(`(?s)^---\n(.*?)\n---\n?`)
func extractFrontmatter(content string) string
func stripFrontmatter(content string) string
func parseSimpleYAML(content string) map[string]string
```

**Path traversal guard** (CTO-B3 — defense-in-depth):
1. `KnownRoles()` allowlist pre-validation
2. `filepath.Join(soulsDir, "SOUL-"+role+".md")`
3. `strings.HasPrefix(filepath.Clean(resolved), filepath.Clean(soulsDir))`

**KnownRoles cache**: Use `sync.Once` for performance on hot path.

**Config default**: `SoulsDir: "docs/08-collaborate/souls"`

### Day 2: T18-02 — Types + Provider Interface Change (ATOMIC COMMIT)

**Goal**: LaunchOpts compiles. Interface change complete.

Files to modify:
| File | Change |
|------|--------|
| `internal/claudecode/types.go` | Add AgentRole, SoulTemplateHash, PersonaSourceHash, PersonaSource to BridgeSession |
| `internal/claudecode/session_manager.go` | Add AgentRole to CreateSessionOpts (line ~16) |
| `internal/claudecode/provider.go` | LaunchOpts struct + update interface + all 4 adapters |

**CRITICAL**: Interface change is breaking — update ALL callers in same commit:
- `ClaudeCodeAdapter.LaunchCommand(opts LaunchOpts)` — add `--agent`/`--append-system-prompt-file` logic
- `StubAdapter.LaunchCommand(opts LaunchOpts)` — accept opts, still return nil
- Any tests that call `LaunchCommand` — update signatures

**LaunchOpts**:
```go
type LaunchOpts struct {
    Workdir    string
    HookURL    string
    Secret     string
    AgentRole  string
    AgentFile  string // Strategy A
    PromptFile string // Strategy B
}
```

### Day 3: T18-03 — Session Manager SOUL Integration

**Goal**: CreateSession resolves Strategy A/B/C. Persona tracked.

Files to modify:
| File | Change |
|------|--------|
| `internal/claudecode/session_manager.go` | SOUL loading + strategy resolution + launch in tmux |

**New flow in CreateSession** (after admission control, after tmux session creation — line ~126):

1. If `opts.AgentRole != ""` -> `LoadSOUL(m.cfg.SoulsDir, opts.AgentRole)`
2. Check `.claude/agents/{role}.md` in project -> Strategy A or B
3. Build `LaunchOpts` with resolved strategy
4. `adapter.LaunchCommand(opts)` -> serialize command -> `tmux.SendKeys` (D12)
5. Store persona fields in session

**Temp file for Strategy B**:
- Path: `~/.mtclaw/sessions/{sanitizedSessionID}/soul.md`
- `sanitizedSessionID = strings.ReplaceAll(sessionID, ":", "-")` (CTO-B2)
- Permissions: `0600`
- Cleanup: `KillSession()` removes directory

**Stale detection**: Compare hashes, warn on mismatch (CTO-M4).

**Audit**: Include `agent_role`, `persona_source`, hashes, `inference_path: "external_claude_code"` (D11).

### Day 4: T18-04 — Telegram Command + Install-Agents

**Goal**: `/cc launch --as` works. Agent installer creates files.

Files to modify:
| File | Change |
|------|--------|
| `internal/channels/telegram/commands_cc.go` | Parse `--as <role>` in /cc launch |
| `cmd/bridge.go` | Add `install-agents` subcommand |

Files to create:
| File | Purpose |
|------|---------|
| `internal/claudecode/bridge_agent_templates.json` | Category->tool/model mappings (//go:embed) |

**`/cc launch` arg parsing**: Extract `--as <role>` from args, validate against `KnownRoles()`.

**install-agents command**: `mtclaw bridge install-agents <project-path> [--souls-dir <path>] [--roles pm,coder]`
- Load each SOUL -> generate `.claude/agents/{role}.md`
- Header comment marks generated files
- Skip user-created files (no header)
- `claude --version` check for compatibility warning (CTO-M6)

**bridge_agent_templates.json**: Embedded via `//go:embed`, maps categories to tools/model.

### Day 5: T18-05 — Tests + Verification

**Goal**: 28 new tests pass. Zero regression.

Files to create:
| File | Tests |
|------|-------|
| `internal/claudecode/soul_loader_test.go` | ~10: frontmatter parsing, hash, 18 roles, path traversal, cache |

Files to extend:
| File | Tests |
|------|-------|
| `internal/claudecode/provider_test.go` | +5: --agent flag, --append-system-prompt-file, bare, LaunchOpts |
| `internal/claudecode/session_manager_test.go` | +6: AgentRole stored, PersonaSource, stale warning, temp cleanup |
| `internal/channels/telegram/commands_cc_test.go` | +4: --as parsing, invalid role, bare default |
| `cmd/bridge_test.go` | +3: install-agents creates, idempotent, skips user files |

---

## Files Summary

### Create (3 files)

| File | Purpose | Day |
|------|---------|-----|
| `internal/claudecode/soul_loader.go` | Soul loading + frontmatter + hash + path guard | 1 |
| `internal/claudecode/soul_loader_test.go` | ~10 tests | 5 |
| `internal/claudecode/bridge_agent_templates.json` | //go:embed tool/model mappings | 4 |

### Modify (6 files)

| File | Change | Day |
|------|--------|-----|
| `internal/claudecode/config.go` | Add SoulsDir | 1 |
| `internal/claudecode/types.go` | Add 4 persona fields to BridgeSession | 2 |
| `internal/claudecode/provider.go` | LaunchOpts + interface change | 2 |
| `internal/claudecode/session_manager.go` | SOUL integration + strategy + launch wiring | 3 |
| `internal/channels/telegram/commands_cc.go` | --as parsing | 4 |
| `cmd/bridge.go` | install-agents subcommand | 4 |

### Extend (4 test files)

| File | New Tests | Day |
|------|-----------|-----|
| `internal/claudecode/provider_test.go` | +5 | 5 |
| `internal/claudecode/session_manager_test.go` | +6 | 5 |
| `internal/channels/telegram/commands_cc_test.go` | +4 | 5 |
| `cmd/bridge_test.go` | +3 | 5 |

---

## Key Code References

| What | File | Grep Anchor |
|------|------|-------------|
| Frontmatter parser (COPY from here) | `internal/skills/loader.go` | `grep -n "frontmatterRe" internal/skills/loader.go` |
| CreateSession flow | `internal/claudecode/session_manager.go` | `grep -n "func.*CreateSession" internal/claudecode/session_manager.go` |
| LaunchCommand interface | `internal/claudecode/provider.go` | `grep -n "LaunchCommand" internal/claudecode/provider.go` |
| BridgeSession struct | `internal/claudecode/types.go` | `grep -n "type BridgeSession" internal/claudecode/types.go` |
| CreateSessionOpts (in session_manager, NOT types) | `internal/claudecode/session_manager.go` | `grep -n "type CreateSessionOpts" internal/claudecode/session_manager.go` |
| SOUL frontmatter example | `docs/08-collaborate/souls/SOUL-coder.md` | `head -10 docs/08-collaborate/souls/SOUL-coder.md` |
| BridgeConfig struct | `internal/claudecode/config.go` | `grep -n "type BridgeConfig" internal/claudecode/config.go` |
| ccLaunch command handler | `internal/channels/telegram/commands_cc.go` | `grep -n "ccLaunch\|func.*launch" internal/channels/telegram/commands_cc.go` |
| bridge subcommands | `cmd/bridge.go` | `grep -n "func.*bridge\|cobra.Command" cmd/bridge.go` |
| safeEnvForSubprocess | `internal/claudecode/provider.go` | `grep -n "safeEnvForSubprocess" internal/claudecode/provider.go` |

---

## CTO Blocker Resolution Quick Reference

| ID | Issue | Resolution | Where |
|----|-------|------------|-------|
| B4 | Bflow bypass | Bridge = control surface, not inference path | ADR-011 D11 |
| B5 | LaunchCommand not wired | SendKeys launches cmd in tmux after session creation | ADR-011 D12 |
| B6 | Wrong frontmatter ref | Copy from `skills/loader.go`, not `bootstrap/files.go` | ADR-011 D13 |
| B7 | No SoulsDir in config | Add `SoulsDir string` to BridgeConfig | ADR-011 D14 |

---

## Verification Checklist

```bash
# 1. Build
make build

# 2. All tests (existing + new)
make test

# 3. claudecode package tests (race-clean)
go test ./internal/claudecode/... -v -race -count=1

# 4. Soul loader tests specifically
go test ./internal/claudecode/ -run TestSoulLoader -v
go test ./internal/claudecode/ -run TestKnownRoles -v
go test ./internal/claudecode/ -run TestPathTraversal -v

# 5. Provider interface change compiles
go vet ./internal/claudecode/...

# 6. Config wiring
grep -n "SoulsDir" internal/claudecode/config.go

# 7. Count total claudecode tests
go test ./internal/claudecode/... -v 2>&1 | grep -c "=== RUN"
# Expected: >= 178 (150 existing + 28 new)

# 8. Verify bridge status still passes
./mtclaw bridge status

# 9. Manual test: install agents
./mtclaw bridge install-agents /tmp/test-project --souls-dir docs/08-collaborate/souls --roles coder,pm
ls /tmp/test-project/.claude/agents/
# Expected: coder.md, pm.md
```

---

## Post-Sprint: What's Next

Sprint 19 (Intelligence Envelope) starts after Sprint 18 CTO review. Sprint 19 adds:
- `SessionIntelligenceEnvelope` type
- `/cc info <session>` command
- `TurnContext` struct definition (no injection yet)

Full roadmap: Sprint 18-23 plans in `docs/04-build/sprints/SPRINT-018..023-*.md`
