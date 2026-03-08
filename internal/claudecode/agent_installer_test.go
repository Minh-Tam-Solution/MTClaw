package claudecode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAgentTemplates(t *testing.T) {
	templates, err := LoadAgentTemplates()
	if err != nil {
		t.Fatalf("LoadAgentTemplates: %v", err)
	}
	if templates.ClaudeCodeVersionMin == "" {
		t.Error("version_min should not be empty")
	}
	if len(templates.Categories) == 0 {
		t.Error("categories should not be empty")
	}
	// Verify executor category exists
	exec, ok := templates.Categories["executor"]
	if !ok {
		t.Fatal("executor category missing")
	}
	if len(exec.Tools) == 0 {
		t.Error("executor tools should not be empty")
	}
}

func TestInstallAgents_CreatesFiles(t *testing.T) {
	InvalidateRolesCache()

	projectDir := t.TempDir()

	result, err := InstallAgents(projectDir, testSoulsDir, []string{"coder"}, false)
	if err != nil {
		t.Fatalf("InstallAgents: %v", err)
	}
	if result.Installed != 1 {
		t.Errorf("installed: got %d, want 1", result.Installed)
	}

	// Verify file exists
	agentFile := filepath.Join(projectDir, ".claude", "agents", "coder.md")
	data, err := os.ReadFile(agentFile)
	if err != nil {
		t.Fatalf("agent file not created: %v", err)
	}

	content := string(data)
	if !strings.HasPrefix(content, generatedHeader) {
		t.Error("agent file should start with generated header")
	}
	if !strings.Contains(content, "name: coder") {
		t.Error("agent file should contain name: coder")
	}
	if !strings.Contains(content, "model: sonnet") {
		t.Error("coder (executor category) should have model: sonnet")
	}
}

func TestInstallAgents_Idempotent(t *testing.T) {
	InvalidateRolesCache()

	projectDir := t.TempDir()

	// First install
	r1, err := InstallAgents(projectDir, testSoulsDir, []string{"coder"}, false)
	if err != nil {
		t.Fatalf("first install: %v", err)
	}
	if r1.Installed != 1 {
		t.Errorf("first: installed=%d, want 1", r1.Installed)
	}

	// Second install (same content = skipped)
	r2, err := InstallAgents(projectDir, testSoulsDir, []string{"coder"}, false)
	if err != nil {
		t.Fatalf("second install: %v", err)
	}
	if r2.Skipped != 1 {
		t.Errorf("second: skipped=%d, want 1", r2.Skipped)
	}
	if r2.Installed != 0 {
		t.Errorf("second: installed=%d, want 0", r2.Installed)
	}
}

func TestInstallAgents_SkipsUserFiles(t *testing.T) {
	InvalidateRolesCache()

	projectDir := t.TempDir()
	agentsDir := filepath.Join(projectDir, ".claude", "agents")
	os.MkdirAll(agentsDir, 0755)

	// Create a user-owned agent file (no generated header)
	userFile := filepath.Join(agentsDir, "coder.md")
	os.WriteFile(userFile, []byte("# My custom coder agent\nCustom content."), 0644)

	result, err := InstallAgents(projectDir, testSoulsDir, []string{"coder"}, false)
	if err != nil {
		t.Fatalf("InstallAgents: %v", err)
	}
	if result.Skipped != 1 {
		t.Errorf("should skip user file: skipped=%d, want 1", result.Skipped)
	}

	// Verify user content was preserved
	data, _ := os.ReadFile(userFile)
	if !strings.Contains(string(data), "My custom coder agent") {
		t.Error("user file should not be overwritten")
	}
}

func TestInstallAgents_ForceOverwritesUserFiles(t *testing.T) {
	InvalidateRolesCache()

	projectDir := t.TempDir()
	agentsDir := filepath.Join(projectDir, ".claude", "agents")
	os.MkdirAll(agentsDir, 0755)

	// Create a user-owned agent file
	userFile := filepath.Join(agentsDir, "coder.md")
	os.WriteFile(userFile, []byte("# My custom coder agent"), 0644)

	result, err := InstallAgents(projectDir, testSoulsDir, []string{"coder"}, true)
	if err != nil {
		t.Fatalf("InstallAgents --force: %v", err)
	}
	// With force, user file counts as updated
	if result.Installed+result.Updated != 1 {
		t.Errorf("force should install/update: got installed=%d updated=%d", result.Installed, result.Updated)
	}

	// Verify file was overwritten with generated content
	data, _ := os.ReadFile(userFile)
	if !strings.HasPrefix(string(data), generatedHeader) {
		t.Error("force should overwrite with generated header")
	}
}

func TestInstallAgents_RoleOverrides(t *testing.T) {
	InvalidateRolesCache()

	projectDir := t.TempDir()

	// CTO should get opus model (role override)
	result, err := InstallAgents(projectDir, testSoulsDir, []string{"cto"}, false)
	if err != nil {
		t.Fatalf("InstallAgents(cto): %v", err)
	}
	if result.Installed != 1 {
		t.Fatalf("expected 1 installed, got %d", result.Installed)
	}

	data, _ := os.ReadFile(filepath.Join(projectDir, ".claude", "agents", "cto.md"))
	if !strings.Contains(string(data), "model: opus") {
		t.Error("CTO should have model: opus (role override)")
	}
}
