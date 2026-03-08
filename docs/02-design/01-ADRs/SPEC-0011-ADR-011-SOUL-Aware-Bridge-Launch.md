# ADR-011: SOUL-Aware Bridge Launch

**SPEC ID**: SPEC-0011
**Status**: ACCEPTED
**Date**: 2026-03-07
**Deciders**: [@cto], [@pm], [@architect]
**Tag**: `adr-011-soul-bridge`
**Depends on**: ADR-010 (Claude Code Bridge)

---

## Context

MTClaw's Claude Code bridge (Sprint 13-17, ADR-010) provides 2-way interaction with multi-tenant governance, 3-axis capability model, and 150+ tests. However, bridge sessions launch Claude Code **bare** — `claude --dangerously-skip-permissions` with zero context about what role the session should play.

MTClaw has 18 SOULs in `docs/08-collaborate/souls/SOUL-*.md` that define SDLC roles (pm, coder, architect, etc.). The agent loop (`internal/agent/systemprompt.go`) injects SOULs into LLM system prompts, but the bridge bypasses this entirely.

**Goal**: Inject SOUL persona into bridge sessions using Claude Code's native mechanisms (`--agent`, `--append-system-prompt-file`), with integrity tracking and audit provenance.

**Non-negotiable invariant**: Bridge capability model (D2: InputMode x ToolPolicy x OutputPolicy) remains the ONLY security boundary. Agent file tool restrictions are UX convenience only.

---

## Decisions

### D10. SOUL-Aware Launch via Strategy A/B/C Cascade

Three strategies, auto-resolved at launch time:

| Strategy | Condition | CLI Command | Persona Source |
|----------|-----------|-------------|----------------|
| **A** (preferred) | `.claude/agents/{role}.md` exists in project | `claude --agent {role} --dangerously-skip-permissions` | `agent_file` |
| **B** (fallback) | No agent file, but SOUL template exists | `claude --append-system-prompt-file {tempfile} --dangerously-skip-permissions` | `append_prompt` |
| **C** (bare) | No `--as` flag specified | `claude --dangerously-skip-permissions` | `bare` |

**Resolution order**: A -> B -> C. Deterministic — same project state always produces same selection.

### D11. Bflow AI-Platform Exemption for Bridge Sessions

**Context**: MTClaw CLAUDE.md mandates "All inference MUST go through Bflow AI-Platform." Bridge sessions are an explicit exemption.

**Rationale**: The bridge is a **control surface**, not an **inference path**. Claude Code sessions use their own API key (Anthropic direct) managed by the Claude Code CLI. MTClaw's bridge relays notifications and input — it does NOT route LLM inference. The Bflow AI-Platform constraint governs MTClaw's own agent loop (`internal/agent/loop.go`), not external tool sessions.

**Boundary**:
- MTClaw agent loop -> Bflow AI-Platform (mandatory, ADR-005)
- Bridge tmux session -> Claude Code CLI -> Anthropic API (exempt, own API key)
- Bridge NEVER proxies LLM calls. Bridge NEVER touches the Anthropic API key.

**Audit**: Bridge sessions log `inference_path: "external_claude_code"` in session creation audit event.

### D12. LaunchCommand Wiring in CreateSession

**Current flow** (Sprint 13-17):
```
CreateSession() -> tmux.CreateSession(target, workdir)  // creates empty tmux session
```
Bridge does NOT call `adapter.LaunchCommand()` in CreateSession. The Claude Code process is started separately via tmux sendKeys after session creation.

**New flow** (Sprint 18):
```
CreateSession() -> tmux.CreateSession(target, workdir)       // Step 1: create tmux pane
                -> resolve Strategy A/B/C                     // Step 2: determine persona
                -> adapter.LaunchCommand(opts) -> exec.Cmd    // Step 3: build command
                -> tmux.SendKeys(target, cmd.String())        // Step 4: launch Claude Code in pane
```

`adapter.LaunchCommand(opts LaunchOpts)` returns `*exec.Cmd` (command to execute). The session manager serializes this command and sends it to the tmux pane via `SendKeys`. The adapter does NOT exec the command directly — tmux owns the process lifecycle.

**Call site**: `session_manager.go:CreateSession()`, after tmux session creation (line ~126), before storing session in map.

### D13. Frontmatter Parser Source

**Correct reference**: `internal/skills/loader.go` (NOT `internal/bootstrap/files.go`)

The skills loader has battle-tested frontmatter functions:
- `frontmatterRe` — `(?s)^---\n(.*?)\n---\n?` regex
- `extractFrontmatter(content string) string` — extracts YAML block
- `stripFrontmatter(content string) string` — removes YAML block
- `parseSimpleYAML(content string) map[string]string` — key: value parsing

**Decision**: Duplicate ~20 LOC in `soul_loader.go` rather than export from `skills` package. Reasons:
1. `claudecode` package must not import `skills` (different dependency chain)
2. Functions are trivial (regex + string split) — duplication < abstraction cost
3. Skills loader may evolve independently (JSON frontmatter support)

### D14. SoulsDir Config Wiring

Add `SoulsDir string` to `BridgeConfig`:

```go
type BridgeConfig struct {
    Enabled       bool           `json:"enabled"`
    HookPort      int            `json:"hook_port,omitempty"`
    SoulsDir      string         `json:"souls_dir,omitempty"`     // NEW: path to SOUL files
    Admission     AdmissionCheck `json:"admission,omitempty"`
    AuditDir      string         `json:"audit_dir,omitempty"`
    StandaloneDir string         `json:"standalone_dir,omitempty"`
}
```

**Default**: `docs/08-collaborate/souls` (relative to working directory). Resolved to absolute path at config load time.

**Wiring chain**: `config.json` -> `Config.Bridge.SoulsDir` -> `SessionManager.cfg.SoulsDir` -> `LoadSOUL(cfg.SoulsDir, role)`.

**SessionManager already has `cfg BridgeConfig`** — no new dependency needed, just access `m.cfg.SoulsDir` in `CreateSession()`.

---

## Pre-ADR Lock Items

### L5. Strategy A/B/C determinism guarantee

Same project state (agent file exists/not, SOUL file exists/not) always produces the same strategy. No randomness, no time-based selection, no cache effects.

### L6. Agent file does NOT override bridge security

Agent file `permissionMode: bypassPermissions` is ignored — `--dangerously-skip-permissions` is already set by the bridge. Agent file `tools:` list does NOT override bridge ToolPolicy (D2). Bridge Layer 1 remains the security boundary.

### L7. install-agents idempotency

Running `mtclaw bridge install-agents` N times produces the same result. Never overwrites user-created agent files (files without the generated header). Updates generated files only when source SOUL hash changes. Requires `--force` to overwrite user files.

---

## Acceptance Criteria (Sprint 18)

1. **Persona provenance in `/cc sessions`** — show role, source type, template hash, persona hash, strategy used
2. **Stale agent file = warning, not block** — `SoulTemplateHash != PersonaSourceHash` -> log warning, show in `/cc sessions`, don't block launch
3. **Installer idempotency** — `install-agents` run N times = same result. Never overwrites user files. Updates generated files only when source changes.
4. **No governance bypass via agent file** — Agent file `permissionMode` is overridden by `--dangerously-skip-permissions`. Agent file tool config doesn't override bridge ToolPolicy (D2).
5. **Strategy determinism** — Same project state always produces same A/B/C selection. No randomness, no race conditions.
6. **Provider abstraction honesty** — Adapter returns `unsupported_reason` if it can't verify injection worked.

---

## Consequences

**Positive**:
- Bridge sessions gain SDLC-aware personas — Claude Code sessions know their role
- Dual-hash tracking enables stale detection without blocking workflow
- Strategy cascade provides graceful degradation (A -> B -> C)
- Config-driven tool mappings (`bridge_agent_templates.json`) avoid hardcoding

**Negative**:
- `ProviderAdapter.LaunchCommand` interface changes — requires atomic commit updating all callers
- Temporary files (Strategy B) need lifecycle management
- `install-agents` adds a setup step for new projects

**Risks**:
- Claude Code CLI flag behavior may change between versions (mitigated: `claude_code_version_min` in templates)
- `--agent` + `--allowedTools` interaction unknown (mitigated: deferred to Sprint 21, verified before use)

---

## CTO Review Resolution Log

| ID | Severity | Issue | Resolution |
|----|----------|-------|------------|
| CTO-B1 | BLOCKING | Tool mapping hardcoded | `bridge_agent_templates.json` with `//go:embed` |
| CTO-B2 | BLOCKING | Colon in sessionID path | `strings.ReplaceAll(sessionID, ":", "-")` |
| CTO-B3 | BLOCKING | Path traversal guard | `KnownRoles()` pre-validation + prefix check |
| CTO-B4 | BLOCKING | Bflow AI-Platform bypass | D11: Bridge is control surface, not inference path |
| CTO-B5 | BLOCKING | LaunchCommand not wired | D12: SendKeys launches command in tmux pane |
| CTO-B6 | BLOCKING | Frontmatter parser ref wrong | D13: Duplicate from `skills/loader.go`, not `bootstrap/files.go` |
| CTO-B7 | BLOCKING | BridgeConfig missing SoulsDir | D14: Add `SoulsDir string` to BridgeConfig |

---

## References

- ADR-010: Claude Code Terminal Bridge (`SPEC-0010`)
- Plan: `/home/dttai/.claude/plans/glowing-gliding-quill.md`
- Claude Code docs: `--agent` flag, `--append-system-prompt-file` flag
- CTO Review: 9.1/10 APPROVED (2026-03-07, Round 3 Final)
