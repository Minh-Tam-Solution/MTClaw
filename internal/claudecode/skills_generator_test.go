package claudecode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSDLCFrameworkSkill_UnderBudget(t *testing.T) {
	size := SkillContentSize()
	if size > 5000 {
		t.Errorf("SDLC Framework skill is %d chars, exceeds 5000 char budget", size)
	}
	if size < 100 {
		t.Errorf("SDLC Framework skill is %d chars, suspiciously small", size)
	}
}

func TestSDLCFrameworkSkill_Content(t *testing.T) {
	content := SDLCFrameworkSkill()
	required := []string{
		"Gate Definitions",
		"SOUL Delegation Rules",
		"Evidence Requirements",
		"Key Rules",
		"G-Sprint",
		"Zero Mock Policy",
	}
	for _, r := range required {
		if !strings.Contains(content, r) {
			t.Errorf("skill content missing required section: %q", r)
		}
	}
}

func TestInstallSkills_CreatesFile(t *testing.T) {
	projectDir := t.TempDir()

	result, err := InstallSkills(projectDir, false)
	if err != nil {
		t.Fatalf("InstallSkills: %v", err)
	}
	if result.Installed != 1 {
		t.Errorf("expected 1 installed, got %d", result.Installed)
	}

	skillFile := filepath.Join(projectDir, ".claude", "skills", "sdlc-framework", "SKILL.md")
	data, err := os.ReadFile(skillFile)
	if err != nil {
		t.Fatalf("skill file not created: %v", err)
	}

	content := string(data)
	if !strings.HasPrefix(content, skillsGeneratedHeader) {
		t.Error("skill file should start with generated header")
	}
	if !strings.Contains(content, "Gate Definitions") {
		t.Error("skill file should contain Gate Definitions")
	}
}

func TestInstallSkills_Idempotent(t *testing.T) {
	projectDir := t.TempDir()

	r1, err := InstallSkills(projectDir, false)
	if err != nil {
		t.Fatalf("first install: %v", err)
	}
	if r1.Installed != 1 {
		t.Errorf("first: installed=%d, want 1", r1.Installed)
	}

	r2, err := InstallSkills(projectDir, false)
	if err != nil {
		t.Fatalf("second install: %v", err)
	}
	if r2.Skipped != 1 {
		t.Errorf("second: skipped=%d, want 1", r2.Skipped)
	}
}

func TestInstallSkills_SkipsUserFiles(t *testing.T) {
	projectDir := t.TempDir()
	skillDir := filepath.Join(projectDir, ".claude", "skills", "sdlc-framework")
	os.MkdirAll(skillDir, 0755)

	userFile := filepath.Join(skillDir, "SKILL.md")
	os.WriteFile(userFile, []byte("# My custom SDLC skill\nCustom content."), 0644)

	result, err := InstallSkills(projectDir, false)
	if err != nil {
		t.Fatalf("InstallSkills: %v", err)
	}
	if result.Skipped != 1 {
		t.Errorf("should skip user file: skipped=%d, want 1", result.Skipped)
	}

	data, _ := os.ReadFile(userFile)
	if !strings.Contains(string(data), "My custom SDLC skill") {
		t.Error("user file should not be overwritten")
	}
}

func TestInstallSkills_ForceOverwrite(t *testing.T) {
	projectDir := t.TempDir()
	skillDir := filepath.Join(projectDir, ".claude", "skills", "sdlc-framework")
	os.MkdirAll(skillDir, 0755)

	userFile := filepath.Join(skillDir, "SKILL.md")
	os.WriteFile(userFile, []byte("# My custom skill"), 0644)

	result, err := InstallSkills(projectDir, true)
	if err != nil {
		t.Fatalf("InstallSkills force: %v", err)
	}
	if result.Installed+result.Updated != 1 {
		t.Errorf("force: installed=%d updated=%d, want 1 total", result.Installed, result.Updated)
	}

	data, _ := os.ReadFile(userFile)
	if !strings.HasPrefix(string(data), skillsGeneratedHeader) {
		t.Error("force should overwrite with generated header")
	}
}

func TestInstallAgents_IncludesSkills(t *testing.T) {
	InvalidateRolesCache()

	projectDir := t.TempDir()

	result, err := InstallAgents(projectDir, testSoulsDir, []string{"coder"}, false)
	if err != nil {
		t.Fatalf("InstallAgents: %v", err)
	}
	if result.Skills == nil {
		t.Fatal("Skills result should not be nil")
	}
	if result.Skills.Installed != 1 {
		t.Errorf("skills installed=%d, want 1", result.Skills.Installed)
	}

	// Verify skill file exists
	skillFile := filepath.Join(projectDir, ".claude", "skills", "sdlc-framework", "SKILL.md")
	if _, err := os.Stat(skillFile); os.IsNotExist(err) {
		t.Error("skill file should be created by install-agents")
	}
}

func TestInstallAgents_AgentFileHasSkills(t *testing.T) {
	InvalidateRolesCache()

	projectDir := t.TempDir()

	_, err := InstallAgents(projectDir, testSoulsDir, []string{"coder"}, false)
	if err != nil {
		t.Fatalf("InstallAgents: %v", err)
	}

	agentFile := filepath.Join(projectDir, ".claude", "agents", "coder.md")
	data, err := os.ReadFile(agentFile)
	if err != nil {
		t.Fatalf("read agent file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "skills:") {
		t.Error("coder agent file (executor category) should have skills: section")
	}
	if !strings.Contains(content, "- sdlc-framework") {
		t.Error("coder agent file should reference sdlc-framework skill")
	}
}
