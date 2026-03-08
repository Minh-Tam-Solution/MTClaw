package claudecode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCursorProjectionAdapter_Name(t *testing.T) {
	a := &CursorProjectionAdapter{}
	if a.Name() != AgentCursor {
		t.Errorf("expected %s, got %s", AgentCursor, a.Name())
	}
}

func TestCursorProjectionAdapter_LaunchCommand_ReturnsNil(t *testing.T) {
	a := &CursorProjectionAdapter{}
	cmd := a.LaunchCommand(LaunchOpts{Workdir: "/tmp"})
	if cmd != nil {
		t.Error("Cursor has no headless CLI — LaunchCommand should return nil")
	}
}

func TestCursorProjectionAdapter_InstallHooks_Unsupported(t *testing.T) {
	a := &CursorProjectionAdapter{}
	err := a.InstallHooks("http://localhost:18792", "secret")
	if err == nil {
		t.Fatal("expected error for unsupported hooks")
	}
	if !strings.Contains(err.Error(), "no_permission_hooks") {
		t.Errorf("error should contain unsupported_reason, got: %v", err)
	}
}

func TestCursorProjectionAdapter_UninstallHooks_Unsupported(t *testing.T) {
	a := &CursorProjectionAdapter{}
	err := a.UninstallHooks()
	if err == nil {
		t.Fatal("expected error for unsupported hooks")
	}
}

func TestCursorProjectionAdapter_ParseStopEvent_Unsupported(t *testing.T) {
	a := &CursorProjectionAdapter{}
	_, err := a.ParseStopEvent([]byte(`{}`))
	if err == nil {
		t.Fatal("expected error for unsupported transcript parsing")
	}
	if !strings.Contains(err.Error(), "no_transcript_parsing") {
		t.Errorf("error should contain unsupported_reason, got: %v", err)
	}
}

func TestCursorProjectionAdapter_CapabilityProfile(t *testing.T) {
	a := &CursorProjectionAdapter{}
	caps := a.CapabilityProfile()
	if caps.PermissionHooks {
		t.Error("Cursor should not report permission hooks support")
	}
	if caps.TranscriptParsing {
		t.Error("Cursor should not report transcript parsing support")
	}
	if caps.HookFormatVersion != 0 {
		t.Errorf("expected hook format version 0, got %d", caps.HookFormatVersion)
	}
}

func TestCursorProjectionAdapter_TranscriptPath_Empty(t *testing.T) {
	a := &CursorProjectionAdapter{}
	if path := a.TranscriptPath("/some/dir"); path != "" {
		t.Errorf("expected empty transcript path, got %q", path)
	}
}

func TestCursorRule_FormatMDC(t *testing.T) {
	rule := &CursorRule{
		Role:        "coder",
		AlwaysApply: true,
		Body:        "You are a software engineer.",
	}

	content := rule.FormatMDC()

	if !strings.HasPrefix(content, generatedHeader) {
		t.Error("generated .mdc should start with generated header")
	}
	if !strings.Contains(content, "description: SOUL persona for coder role") {
		t.Error("missing description in frontmatter")
	}
	if !strings.Contains(content, "alwaysApply: true") {
		t.Error("missing alwaysApply in frontmatter")
	}
	if !strings.Contains(content, "You are a software engineer.") {
		t.Error("missing SOUL body")
	}
}

func TestCursorRule_FormatMDC_NoAlwaysApply(t *testing.T) {
	rule := &CursorRule{
		Role:        "reviewer",
		AlwaysApply: false,
		Body:        "You are a code reviewer.",
	}

	content := rule.FormatMDC()

	if strings.Contains(content, "alwaysApply") {
		t.Error("should not contain alwaysApply when false")
	}
}

func TestGenerateCursorRules_CreatesFiles(t *testing.T) {
	InvalidateRolesCache()
	soulsDir := filepath.Join(t.TempDir(), "souls")
	writeTempSOUL(t, soulsDir, "coder", "executor")
	writeTempSOUL(t, soulsDir, "pm", "advisor")

	projectDir := t.TempDir()

	result, err := GenerateCursorRules(projectDir, soulsDir, nil, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Installed != 2 {
		t.Errorf("expected 2 installed, got %d", result.Installed)
	}

	// Verify files exist
	coderFile := filepath.Join(projectDir, ".cursor", "rules", "coder.mdc")
	data, err := os.ReadFile(coderFile)
	if err != nil {
		t.Fatalf("coder.mdc not created: %v", err)
	}
	if !strings.HasPrefix(string(data), generatedHeader) {
		t.Error("coder.mdc should start with generated header")
	}
	if !strings.Contains(string(data), "alwaysApply: true") {
		t.Error("coder.mdc should have alwaysApply: true")
	}

	pmFile := filepath.Join(projectDir, ".cursor", "rules", "pm.mdc")
	if _, err := os.ReadFile(pmFile); err != nil {
		t.Fatalf("pm.mdc not created: %v", err)
	}
}

func TestGenerateCursorRules_Idempotent(t *testing.T) {
	InvalidateRolesCache()
	soulsDir := filepath.Join(t.TempDir(), "souls")
	writeTempSOUL(t, soulsDir, "coder", "executor")

	projectDir := t.TempDir()

	// First run
	result1, err := GenerateCursorRules(projectDir, soulsDir, nil, false)
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	if result1.Installed != 1 {
		t.Fatalf("expected 1 installed, got %d", result1.Installed)
	}

	// Second run — same content, should skip
	InvalidateRolesCache()
	result2, err := GenerateCursorRules(projectDir, soulsDir, nil, false)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if result2.Skipped != 1 {
		t.Errorf("expected 1 skipped on idempotent run, got %d", result2.Skipped)
	}
	if result2.Installed != 0 || result2.Updated != 0 {
		t.Errorf("expected no installs/updates, got installed=%d updated=%d", result2.Installed, result2.Updated)
	}
}

func TestGenerateCursorRules_SkipsUserFiles(t *testing.T) {
	InvalidateRolesCache()
	soulsDir := filepath.Join(t.TempDir(), "souls")
	writeTempSOUL(t, soulsDir, "coder", "executor")

	projectDir := t.TempDir()
	rulesDir := filepath.Join(projectDir, ".cursor", "rules")
	os.MkdirAll(rulesDir, 0755)

	// Create a user-created .mdc file (no generated header)
	userContent := "---\ndescription: My custom coder rules\n---\nCustom content"
	os.WriteFile(filepath.Join(rulesDir, "coder.mdc"), []byte(userContent), 0644)

	result, err := GenerateCursorRules(projectDir, soulsDir, nil, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Skipped != 1 {
		t.Errorf("expected 1 skipped (user file), got %d", result.Skipped)
	}

	// Verify user content preserved
	data, _ := os.ReadFile(filepath.Join(rulesDir, "coder.mdc"))
	if string(data) != userContent {
		t.Error("user-created file should not be overwritten")
	}
}

func TestGenerateCursorRules_ForceOverwritesUserFiles(t *testing.T) {
	InvalidateRolesCache()
	soulsDir := filepath.Join(t.TempDir(), "souls")
	writeTempSOUL(t, soulsDir, "coder", "executor")

	projectDir := t.TempDir()
	rulesDir := filepath.Join(projectDir, ".cursor", "rules")
	os.MkdirAll(rulesDir, 0755)

	userContent := "---\ndescription: My custom coder rules\n---\nCustom content"
	os.WriteFile(filepath.Join(rulesDir, "coder.mdc"), []byte(userContent), 0644)

	result, err := GenerateCursorRules(projectDir, soulsDir, nil, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Updated != 1 {
		t.Errorf("expected 1 updated with force, got %d", result.Updated)
	}

	data, _ := os.ReadFile(filepath.Join(rulesDir, "coder.mdc"))
	if !strings.HasPrefix(string(data), generatedHeader) {
		t.Error("force should overwrite with generated content")
	}
}

func TestGenerateCursorRules_FilteredRoles(t *testing.T) {
	InvalidateRolesCache()
	soulsDir := filepath.Join(t.TempDir(), "souls")
	writeTempSOUL(t, soulsDir, "coder", "executor")
	writeTempSOUL(t, soulsDir, "pm", "advisor")
	writeTempSOUL(t, soulsDir, "reviewer", "advisor")

	projectDir := t.TempDir()

	result, err := GenerateCursorRules(projectDir, soulsDir, []string{"coder", "pm"}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Installed != 2 {
		t.Errorf("expected 2 installed (filtered), got %d", result.Installed)
	}

	// reviewer.mdc should NOT exist
	if _, err := os.Stat(filepath.Join(projectDir, ".cursor", "rules", "reviewer.mdc")); err == nil {
		t.Error("reviewer.mdc should not exist when filtered out")
	}
}

func TestGenerateCursorRules_InvalidRole(t *testing.T) {
	InvalidateRolesCache()
	soulsDir := filepath.Join(t.TempDir(), "souls")
	writeTempSOUL(t, soulsDir, "coder", "executor")

	projectDir := t.TempDir()

	_, err := GenerateCursorRules(projectDir, soulsDir, []string{"nonexistent"}, false)
	if err == nil {
		t.Fatal("expected error for nonexistent role")
	}
}

func TestCursorProjectionInfo(t *testing.T) {
	info := CursorProjectionInfo()

	if info["provider"] != string(AgentCursor) {
		t.Errorf("expected provider=%s, got %v", AgentCursor, info["provider"])
	}
	if info["session_management"] != false {
		t.Error("Cursor should report session_management=false")
	}
	if info["file_generation"] != true {
		t.Error("Cursor should report file_generation=true")
	}
	if info["unsupported_reason"] != "no_headless_cli" {
		t.Errorf("unexpected unsupported_reason: %v", info["unsupported_reason"])
	}
}

func TestProviderRegistry_CursorAdapter(t *testing.T) {
	reg := NewProviderRegistry()
	adapter, err := reg.Get(AgentCursor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if adapter.Name() != AgentCursor {
		t.Errorf("expected %s, got %s", AgentCursor, adapter.Name())
	}
	// Should be CursorProjectionAdapter, not StubAdapter
	if _, ok := adapter.(*CursorProjectionAdapter); !ok {
		t.Errorf("expected CursorProjectionAdapter, got %T", adapter)
	}
	// LaunchCommand should return nil (no headless CLI)
	if cmd := adapter.LaunchCommand(LaunchOpts{}); cmd != nil {
		t.Error("Cursor LaunchCommand should return nil")
	}
}
