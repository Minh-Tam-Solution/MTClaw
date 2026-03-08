package claudecode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckInputAllowed_StructuredOnly(t *testing.T) {
	caps := CapabilitiesForRisk(RiskModeRead) // structured_only

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty", "", false},
		{"cc command", "/cc capture", false},
		{"slash command", "/help", false},
		{"free text blocked", "hello world", true},
		{"code blocked", "git push origin main", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckInputAllowed(caps, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckInputAllowed(%q): err=%v, wantErr=%v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestCheckInputAllowed_FreeText(t *testing.T) {
	caps := CapabilitiesForRisk(RiskModeInteractive) // free_text

	tests := []struct {
		input string
	}{
		{""},
		{"/cc capture"},
		{"hello world"},
		{"git push origin main"},
		{"rm -rf /"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if err := CheckInputAllowed(caps, tt.input); err != nil {
				t.Errorf("free_text mode should allow %q: %v", tt.input, err)
			}
		})
	}
}

func TestCheckCaptureAllowed(t *testing.T) {
	tests := []struct {
		name      string
		mode      RiskMode
		wantLines int
	}{
		{"read", RiskModeRead, 30},
		{"patch", RiskModePatch, 50},
		{"interactive", RiskModeInteractive, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			caps := CapabilitiesForRisk(tt.mode)
			lines, err := CheckCaptureAllowed(caps)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if lines != tt.wantLines {
				t.Errorf("lines: got %d, want %d", lines, tt.wantLines)
			}
		})
	}
}

func TestCheckCaptureAllowed_ZeroLines(t *testing.T) {
	caps := SessionCapabilities{CaptureLines: 0}
	_, err := CheckCaptureAllowed(caps)
	if err == nil {
		t.Error("expected error for zero capture lines")
	}
}

func TestCheckToolAllowed_Observe(t *testing.T) {
	caps := CapabilitiesForRisk(RiskModeRead) // observe

	if err := CheckToolAllowed(caps, "Read", false); err != nil {
		t.Errorf("read tool should be allowed: %v", err)
	}
	if err := CheckToolAllowed(caps, "Edit", true); err == nil {
		t.Error("write tool should be blocked in observe mode")
	}
}

func TestCheckToolAllowed_PatchAllowed(t *testing.T) {
	caps := CapabilitiesForRisk(RiskModePatch) // patch_allowed

	if err := CheckToolAllowed(caps, "Read", false); err != nil {
		t.Errorf("read should be allowed: %v", err)
	}
	if err := CheckToolAllowed(caps, "Edit", true); err != nil {
		t.Errorf("write should be allowed in patch mode: %v", err)
	}
}

func TestCheckToolAllowed_ExecWithApproval(t *testing.T) {
	caps := CapabilitiesForRisk(RiskModeInteractive)

	if err := CheckToolAllowed(caps, "Bash", true); err != nil {
		t.Errorf("exec should be allowed (approval checked elsewhere): %v", err)
	}
}

func TestCheckRiskEscalation(t *testing.T) {
	tests := []struct {
		name    string
		current RiskMode
		target  RiskMode
		actor   string
		owner   string
		wantErr bool
	}{
		{"same mode noop", RiskModeRead, RiskModeRead, "anyone", "owner", false},
		{"owner escalate", RiskModeRead, RiskModePatch, "owner", "owner", false},
		{"non-owner escalate blocked", RiskModeRead, RiskModePatch, "other", "owner", true},
		{"anyone can downgrade", RiskModePatch, RiskModeRead, "random", "owner", false},
		{"owner to interactive", RiskModeRead, RiskModeInteractive, "owner", "owner", false},
		{"non-owner to interactive blocked", RiskModePatch, RiskModeInteractive, "other", "owner", true},
		{"downgrade interactive to read", RiskModeInteractive, RiskModeRead, "random", "owner", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckRiskEscalation(tt.current, tt.target, tt.actor, tt.owner)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckRiskEscalation: err=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

// --- Sprint 21: Role-Aware Default Tests ---

// writeTempSOUL creates a SOUL file in dir for testing role defaults.
func writeTempSOUL(t *testing.T, dir, role, category string) {
	t.Helper()
	content := "---\nrole: " + role + "\ncategory: " + category + "\n---\nTest SOUL body for " + role
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SOUL-"+role+".md"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestRoleDefaultRiskMode_Executor(t *testing.T) {
	InvalidateRolesCache()
	dir := filepath.Join(t.TempDir(), "souls")
	writeTempSOUL(t, dir, "coder", "executor")

	mode := RoleDefaultRiskMode(dir, "coder")
	if mode != RiskModePatch {
		t.Errorf("executor category should default to patch, got %s", mode)
	}
}

func TestRoleDefaultRiskMode_Advisor(t *testing.T) {
	InvalidateRolesCache()
	dir := filepath.Join(t.TempDir(), "souls")
	writeTempSOUL(t, dir, "cto", "advisor")

	mode := RoleDefaultRiskMode(dir, "cto")
	if mode != RiskModeRead {
		t.Errorf("advisor category should default to read, got %s", mode)
	}
}

func TestRoleDefaultRiskMode_UnknownRole(t *testing.T) {
	InvalidateRolesCache()
	dir := filepath.Join(t.TempDir(), "souls")
	writeTempSOUL(t, dir, "coder", "executor")

	mode := RoleDefaultRiskMode(dir, "nonexistent")
	if mode != RiskModeRead {
		t.Errorf("unknown role should default to read, got %s", mode)
	}
}

func TestRoleDefaultRiskMode_MissingDir(t *testing.T) {
	InvalidateRolesCache()
	mode := RoleDefaultRiskMode("/nonexistent", "coder")
	if mode != RiskModeRead {
		t.Errorf("missing dir should default to read, got %s", mode)
	}
}

func TestAllowedToolsForRole_Executor(t *testing.T) {
	InvalidateRolesCache()
	dir := filepath.Join(t.TempDir(), "souls")
	writeTempSOUL(t, dir, "coder", "executor")

	tools := AllowedToolsForRole(dir, "coder")
	if len(tools) == 0 {
		t.Fatal("expected non-empty tool list for executor")
	}
	// Executor should have Read, Edit, Write, Bash, Grep, Glob
	joined := strings.Join(tools, ",")
	for _, want := range []string{"Read", "Edit", "Bash"} {
		if !strings.Contains(joined, want) {
			t.Errorf("executor tools should contain %s, got %v", want, tools)
		}
	}
}

func TestAllowedToolsForRole_Advisor(t *testing.T) {
	InvalidateRolesCache()
	dir := filepath.Join(t.TempDir(), "souls")
	writeTempSOUL(t, dir, "reviewer", "advisor")

	tools := AllowedToolsForRole(dir, "reviewer")
	if len(tools) == 0 {
		t.Fatal("expected non-empty tool list for advisor")
	}
	joined := strings.Join(tools, ",")
	if strings.Contains(joined, "Edit") {
		t.Error("advisor tools should NOT contain Edit")
	}
	if strings.Contains(joined, "Bash") {
		t.Error("advisor tools should NOT contain Bash")
	}
}

func TestFormatAllowedTools(t *testing.T) {
	result := FormatAllowedTools([]string{"Read", "Edit", "Bash"})
	if result != "Read, Edit, Bash" {
		t.Errorf("expected 'Read, Edit, Bash', got %q", result)
	}
}

func TestRoleDefaultOverridable(t *testing.T) {
	// Verify that role defaults don't prevent manual override via /cc risk
	// This is an architectural test: even if role says "patch", owner can downgrade to "read"
	InvalidateRolesCache()
	dir := filepath.Join(t.TempDir(), "souls")
	writeTempSOUL(t, dir, "coder", "executor")

	defaultRisk := RoleDefaultRiskMode(dir, "coder")
	if defaultRisk != RiskModePatch {
		t.Fatalf("precondition: expected patch default, got %s", defaultRisk)
	}

	// Simulate downgrade: any actor can downgrade (CheckRiskEscalation allows it)
	err := CheckRiskEscalation(RiskModePatch, RiskModeRead, "any-actor", "owner")
	if err != nil {
		t.Errorf("downgrade from role default should always be allowed: %v", err)
	}
}
