package claudecode

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// LaunchOpts holds all parameters for building a provider launch command (D10).
// Introduced in Sprint 18 to support Strategy A/B/C SOUL injection.
type LaunchOpts struct {
	Workdir      string   // project working directory
	HookURL      string   // hook server callback URL
	Secret       string   // HMAC-SHA256 session secret
	AgentRole    string   // SOUL role name (empty = bare launch, Strategy C)
	AgentFile    string   // Strategy A: path to .claude/agents/{role}.md (if exists)
	PromptFile   string   // Strategy B: path to temp SOUL file for --append-system-prompt-file
	AllowedTools []string // Sprint 21: UX convenience tool filter (NOT security gate — D2 is the gate)
}

// ProviderAdapter abstracts agent-specific process management (D9).
// Sprint A-D: only ClaudeCodeAdapter is implemented.
type ProviderAdapter interface {
	Name() AgentProviderType
	LaunchCommand(opts LaunchOpts) *exec.Cmd
	InstallHooks(hookURL, secret string) error
	UninstallHooks() error
	ParseStopEvent(payload []byte) (*StopEvent, error)
	CapabilityProfile() ProviderCapabilities
	TranscriptPath(sessionDir string) string
}

// ClaudeCodeAdapter implements ProviderAdapter for Claude Code CLI.
type ClaudeCodeAdapter struct{}

func (a *ClaudeCodeAdapter) Name() AgentProviderType {
	return AgentClaudeCode
}

func (a *ClaudeCodeAdapter) LaunchCommand(opts LaunchOpts) *exec.Cmd {
	args := []string{"--dangerously-skip-permissions"}

	// Strategy A: native agent file (preferred — D10)
	if opts.AgentFile != "" && fileExists(opts.AgentFile) {
		args = append(args, "--agent", opts.AgentRole)
	} else if opts.PromptFile != "" && fileExists(opts.PromptFile) {
		// Strategy B: append SOUL via temp file fallback (D10)
		args = append(args, "--append-system-prompt-file", opts.PromptFile)
	}
	// Strategy C: bare launch (no persona flags) — implicit default

	// Sprint 21: --allowedTools as UX noise-reduction (NOT security gate — D2 is the gate)
	if len(opts.AllowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(opts.AllowedTools, ", "))
	}

	cmd := exec.Command("claude", args...)
	cmd.Dir = opts.Workdir
	// CTO-66: Sanitize env — whitelist only safe vars to prevent leaking
	// gateway secrets (API keys, encryption keys) to the subprocess.
	cmd.Env = safeEnvForSubprocess(
		fmt.Sprintf("CLAUDE_HOOK_URL=%s", opts.HookURL),
		fmt.Sprintf("CLAUDE_HOOK_SECRET=%s", opts.Secret),
	)
	return cmd
}

// fileExists checks if a file exists and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// safeEnvForSubprocess builds a minimal environment for Claude Code subprocess.
// Only whitelisted vars are inherited — prevents leaking MTCLAW_BFLOW_API_KEY,
// MTCLAW_ENCRYPTION_KEY, and other gateway secrets.
func safeEnvForSubprocess(extra ...string) []string {
	allowedKeys := []string{
		"HOME", "USER", "PATH", "SHELL", "LANG", "LC_ALL",
		"TERM", "COLORTERM", "EDITOR", "VISUAL",
		"XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME",
		"SSH_AUTH_SOCK", "GPG_TTY",
		"NODE_PATH", "NVM_DIR",
	}

	env := make([]string, 0, len(allowedKeys)+len(extra))
	for _, key := range allowedKeys {
		if val, ok := os.LookupEnv(key); ok {
			env = append(env, key+"="+val)
		}
	}
	env = append(env, extra...)
	return env
}

func (a *ClaudeCodeAdapter) InstallHooks(hookURL, secret string) error {
	// Sprint B: write hook scripts to ~/.claude/hooks/
	return nil
}

func (a *ClaudeCodeAdapter) UninstallHooks() error {
	// Sprint D: remove hook scripts
	return nil
}

func (a *ClaudeCodeAdapter) ParseStopEvent(payload []byte) (*StopEvent, error) {
	var evt StopEvent
	if err := json.Unmarshal(payload, &evt); err != nil {
		return nil, fmt.Errorf("parse stop event: %w", err)
	}
	return &evt, nil
}

func (a *ClaudeCodeAdapter) CapabilityProfile() ProviderCapabilities {
	return ProviderCapabilities{
		PermissionHooks:   true,
		TranscriptParsing: true,
		HookFormatVersion: 1,
	}
}

func (a *ClaudeCodeAdapter) TranscriptPath(sessionDir string) string {
	return filepath.Join(sessionDir, "transcript.jsonl")
}

// StubAdapter is a placeholder for agents not yet supported (Sprint E+).
// PermissionHooks=false prevents escalation to interactive mode (D7 Layer 0).
type StubAdapter struct {
	AgentType AgentProviderType
}

func (s *StubAdapter) Name() AgentProviderType { return s.AgentType }

func (s *StubAdapter) LaunchCommand(_ LaunchOpts) *exec.Cmd {
	return nil
}

func (s *StubAdapter) InstallHooks(_, _ string) error {
	return fmt.Errorf("%s: not yet supported", s.AgentType)
}

func (s *StubAdapter) UninstallHooks() error {
	return fmt.Errorf("%s: not yet supported", s.AgentType)
}

func (s *StubAdapter) ParseStopEvent(_ []byte) (*StopEvent, error) {
	return nil, fmt.Errorf("%s: not yet supported", s.AgentType)
}

func (s *StubAdapter) CapabilityProfile() ProviderCapabilities {
	return ProviderCapabilities{
		PermissionHooks:   false,
		TranscriptParsing: false,
		HookFormatVersion: 0,
	}
}

func (s *StubAdapter) TranscriptPath(_ string) string { return "" }

// ProviderRegistry maps agent types to their adapters.
type ProviderRegistry struct {
	adapters map[AgentProviderType]ProviderAdapter
}

// NewProviderRegistry creates a registry with all known adapters.
func NewProviderRegistry() *ProviderRegistry {
	r := &ProviderRegistry{
		adapters: make(map[AgentProviderType]ProviderAdapter),
	}
	r.adapters[AgentClaudeCode] = &ClaudeCodeAdapter{}
	r.adapters[AgentCursor] = &CursorProjectionAdapter{}
	r.adapters[AgentCodexCLI] = &StubAdapter{AgentType: AgentCodexCLI}
	r.adapters[AgentGeminiCLI] = &StubAdapter{AgentType: AgentGeminiCLI}
	return r
}

// Get returns the adapter for the given agent type.
func (r *ProviderRegistry) Get(agentType AgentProviderType) (ProviderAdapter, error) {
	a, ok := r.adapters[agentType]
	if !ok {
		return nil, fmt.Errorf("unknown agent type: %s", agentType)
	}
	return a, nil
}
