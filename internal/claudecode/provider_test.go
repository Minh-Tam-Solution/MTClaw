package claudecode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClaudeCodeAdapter_LaunchCommand_StrategyC(t *testing.T) {
	a := &ClaudeCodeAdapter{}
	opts := LaunchOpts{
		Workdir: "/tmp",
		HookURL: "http://127.0.0.1:18792/hook",
		Secret:  "test-secret",
	}

	cmd := a.LaunchCommand(opts)
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	args := strings.Join(cmd.Args, " ")
	if !strings.Contains(args, "--dangerously-skip-permissions") {
		t.Error("missing --dangerously-skip-permissions")
	}
	if strings.Contains(args, "--agent") {
		t.Error("Strategy C: should NOT have --agent flag")
	}
	if strings.Contains(args, "--append-system-prompt-file") {
		t.Error("Strategy C: should NOT have --append-system-prompt-file flag")
	}
}

func TestClaudeCodeAdapter_LaunchCommand_StrategyA(t *testing.T) {
	// Create a temp agent file
	tmpDir := t.TempDir()
	agentFile := filepath.Join(tmpDir, "coder.md")
	if err := os.WriteFile(agentFile, []byte("# Agent file"), 0644); err != nil {
		t.Fatal(err)
	}

	a := &ClaudeCodeAdapter{}
	opts := LaunchOpts{
		Workdir:   "/tmp",
		HookURL:   "http://127.0.0.1:18792/hook",
		Secret:    "test-secret",
		AgentRole: "coder",
		AgentFile: agentFile,
	}

	cmd := a.LaunchCommand(opts)
	args := strings.Join(cmd.Args, " ")
	if !strings.Contains(args, "--agent coder") {
		t.Errorf("Strategy A: expected --agent coder, got: %s", args)
	}
}

func TestClaudeCodeAdapter_LaunchCommand_StrategyB(t *testing.T) {
	// Create a temp prompt file
	tmpDir := t.TempDir()
	promptFile := filepath.Join(tmpDir, "soul.md")
	if err := os.WriteFile(promptFile, []byte("# SOUL content"), 0644); err != nil {
		t.Fatal(err)
	}

	a := &ClaudeCodeAdapter{}
	opts := LaunchOpts{
		Workdir:    "/tmp",
		HookURL:    "http://127.0.0.1:18792/hook",
		Secret:     "test-secret",
		AgentRole:  "coder",
		PromptFile: promptFile,
	}

	cmd := a.LaunchCommand(opts)
	args := strings.Join(cmd.Args, " ")
	if !strings.Contains(args, "--append-system-prompt-file") {
		t.Errorf("Strategy B: expected --append-system-prompt-file, got: %s", args)
	}
	if strings.Contains(args, "--agent") {
		t.Error("Strategy B: should NOT have --agent flag (no agent file)")
	}
}

func TestClaudeCodeAdapter_LaunchCommand_StrategyA_Precedence(t *testing.T) {
	// Both agent file and prompt file exist — Strategy A takes precedence
	tmpDir := t.TempDir()
	agentFile := filepath.Join(tmpDir, "coder.md")
	promptFile := filepath.Join(tmpDir, "soul.md")
	os.WriteFile(agentFile, []byte("# Agent"), 0644)
	os.WriteFile(promptFile, []byte("# SOUL"), 0644)

	a := &ClaudeCodeAdapter{}
	opts := LaunchOpts{
		Workdir:    "/tmp",
		HookURL:    "http://127.0.0.1:18792/hook",
		Secret:     "test-secret",
		AgentRole:  "coder",
		AgentFile:  agentFile,
		PromptFile: promptFile,
	}

	cmd := a.LaunchCommand(opts)
	args := strings.Join(cmd.Args, " ")
	if !strings.Contains(args, "--agent coder") {
		t.Errorf("Strategy A should take precedence, got: %s", args)
	}
	if strings.Contains(args, "--append-system-prompt-file") {
		t.Error("Should not have Strategy B flag when Strategy A applies")
	}
}

func TestClaudeCodeAdapter_LaunchCommand_AllowedTools(t *testing.T) {
	a := &ClaudeCodeAdapter{}
	opts := LaunchOpts{
		Workdir:      "/tmp",
		HookURL:      "http://127.0.0.1:18792/hook",
		Secret:       "test-secret",
		AllowedTools: []string{"Read", "Edit", "Grep"},
	}

	cmd := a.LaunchCommand(opts)
	args := strings.Join(cmd.Args, " ")
	if !strings.Contains(args, "--allowedTools") {
		t.Errorf("expected --allowedTools flag, got: %s", args)
	}
	if !strings.Contains(args, "Read, Edit, Grep") {
		t.Errorf("expected tool list in --allowedTools, got: %s", args)
	}
}

func TestClaudeCodeAdapter_LaunchCommand_NoAllowedTools(t *testing.T) {
	a := &ClaudeCodeAdapter{}
	opts := LaunchOpts{
		Workdir: "/tmp",
		HookURL: "http://127.0.0.1:18792/hook",
		Secret:  "test-secret",
	}

	cmd := a.LaunchCommand(opts)
	args := strings.Join(cmd.Args, " ")
	if strings.Contains(args, "--allowedTools") {
		t.Errorf("bare launch should NOT have --allowedTools, got: %s", args)
	}
}

func TestStubAdapter_LaunchCommand(t *testing.T) {
	s := &StubAdapter{AgentType: AgentCodexCLI}
	cmd := s.LaunchCommand(LaunchOpts{})
	if cmd != nil {
		t.Error("StubAdapter.LaunchCommand should return nil")
	}
}

func TestClaudeCodeAdapter_EnvSanitization(t *testing.T) {
	a := &ClaudeCodeAdapter{}
	opts := LaunchOpts{
		Workdir: "/tmp",
		HookURL: "http://127.0.0.1:18792/hook",
		Secret:  "secret-123",
	}

	cmd := a.LaunchCommand(opts)
	for _, env := range cmd.Env {
		if strings.HasPrefix(env, "MTCLAW_") {
			t.Errorf("gateway secret leaked to subprocess: %s", env)
		}
	}

	// Verify hook env vars are set
	foundHookURL := false
	foundHookSecret := false
	for _, env := range cmd.Env {
		if strings.HasPrefix(env, "CLAUDE_HOOK_URL=") {
			foundHookURL = true
		}
		if strings.HasPrefix(env, "CLAUDE_HOOK_SECRET=") {
			foundHookSecret = true
		}
	}
	if !foundHookURL {
		t.Error("CLAUDE_HOOK_URL not set in env")
	}
	if !foundHookSecret {
		t.Error("CLAUDE_HOOK_SECRET not set in env")
	}
}
