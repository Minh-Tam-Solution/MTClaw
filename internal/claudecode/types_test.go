package claudecode

import (
	"strings"
	"testing"
)

func TestGenerateSessionID(t *testing.T) {
	tests := []struct {
		name     string
		tenantID string
	}{
		{"normal tenant", "tenant-abc-123"},
		{"empty tenant", ""},
		{"long tenant", strings.Repeat("x", 1000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := GenerateSessionID(tt.tenantID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.HasPrefix(id, "br:") {
				t.Errorf("session ID should start with 'br:', got %q", id)
			}
			parts := strings.Split(id, ":")
			if len(parts) != 3 {
				t.Errorf("session ID should have 3 parts (br:tenant8:rand8), got %d parts: %q", len(parts), id)
			}
			if len(parts) == 3 {
				if len(parts[1]) != 8 {
					t.Errorf("tenant hash should be 8 hex chars, got %d: %q", len(parts[1]), parts[1])
				}
				if len(parts[2]) != 8 {
					t.Errorf("random part should be 8 hex chars, got %d: %q", len(parts[2]), parts[2])
				}
			}
		})
	}
}

func TestGenerateSessionID_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id, err := GenerateSessionID("test-tenant")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if seen[id] {
			t.Fatalf("duplicate session ID generated: %s", id)
		}
		seen[id] = true
	}
}

func TestGenerateSessionID_SameTenantPrefix(t *testing.T) {
	id1, _ := GenerateSessionID("my-tenant")
	id2, _ := GenerateSessionID("my-tenant")
	parts1 := strings.Split(id1, ":")
	parts2 := strings.Split(id2, ":")
	if parts1[1] != parts2[1] {
		t.Errorf("same tenant should produce same prefix, got %q vs %q", parts1[1], parts2[1])
	}
	if parts1[2] == parts2[2] {
		t.Error("random parts should differ")
	}
}

func TestGenerateHookSecret(t *testing.T) {
	secret, err := GenerateHookSecret()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(secret) != 64 {
		t.Errorf("hook secret should be 64 hex chars (32 bytes), got %d", len(secret))
	}
}

func TestGenerateHookSecret_Uniqueness(t *testing.T) {
	s1, _ := GenerateHookSecret()
	s2, _ := GenerateHookSecret()
	if s1 == s2 {
		t.Error("two secrets should not be identical")
	}
}

func TestCapabilitiesForRisk(t *testing.T) {
	tests := []struct {
		mode         RiskMode
		wantInput    InputMode
		wantTool     ToolPolicy
		wantLines    int
		wantRedact   bool
	}{
		{RiskModeRead, InputStructuredOnly, ToolPolicyObserve, 30, true},
		{RiskModePatch, InputStructuredOnly, ToolPolicyPatchAllowed, 50, false},
		{RiskModeInteractive, InputFreeText, ToolPolicyExecWithApproval, 100, false},
		{"unknown", InputStructuredOnly, ToolPolicyObserve, 30, true}, // default = read
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			caps := CapabilitiesForRisk(tt.mode)
			if caps.InputMode != tt.wantInput {
				t.Errorf("InputMode: got %q, want %q", caps.InputMode, tt.wantInput)
			}
			if caps.ToolPolicy != tt.wantTool {
				t.Errorf("ToolPolicy: got %q, want %q", caps.ToolPolicy, tt.wantTool)
			}
			if caps.CaptureLines != tt.wantLines {
				t.Errorf("CaptureLines: got %d, want %d", caps.CaptureLines, tt.wantLines)
			}
			if caps.RedactHeavy != tt.wantRedact {
				t.Errorf("RedactHeavy: got %v, want %v", caps.RedactHeavy, tt.wantRedact)
			}
		})
	}
}

func TestDefaultAdmissionCheck(t *testing.T) {
	ac := DefaultAdmissionCheck()
	if ac.MaxSessionsPerAgent != 2 {
		t.Errorf("MaxSessionsPerAgent: got %d, want 2", ac.MaxSessionsPerAgent)
	}
	if ac.MaxTotalSessions != 6 {
		t.Errorf("MaxTotalSessions: got %d, want 6", ac.MaxTotalSessions)
	}
	if ac.MaxCPUPercent != 85.0 {
		t.Errorf("MaxCPUPercent: got %f, want 85.0", ac.MaxCPUPercent)
	}
	if ac.MaxMemoryPercent != 80.0 {
		t.Errorf("MaxMemoryPercent: got %f, want 80.0", ac.MaxMemoryPercent)
	}
	if ac.PerTenantSessionCap != 4 {
		t.Errorf("PerTenantSessionCap: got %d, want 4", ac.PerTenantSessionCap)
	}
	if ac.PerProjectSingleton {
		t.Error("PerProjectSingleton should default to false")
	}
}
