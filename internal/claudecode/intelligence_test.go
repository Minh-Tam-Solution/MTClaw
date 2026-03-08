package claudecode

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestStrategyFromPersonaSource(t *testing.T) {
	tests := []struct {
		source   string
		expected string
	}{
		{"agent_file", "A"},
		{"append_prompt", "B"},
		{"bare", "C"},
		{"", "C"},
		{"unknown", "C"},
	}
	for _, tt := range tests {
		got := StrategyFromPersonaSource(tt.source)
		if got != tt.expected {
			t.Errorf("StrategyFromPersonaSource(%q) = %q, want %q", tt.source, got, tt.expected)
		}
	}
}

func TestBuildPersonaEnvelope_Bare(t *testing.T) {
	session := &BridgeSession{
		PersonaSource: "bare",
	}
	env := BuildPersonaEnvelope(session)
	if env.PersonaSource != "bare" {
		t.Errorf("expected bare, got %q", env.PersonaSource)
	}
	if env.Strategy != "C" {
		t.Errorf("expected C, got %q", env.Strategy)
	}
	if env.AgentRole != "" {
		t.Errorf("expected empty role, got %q", env.AgentRole)
	}
}

// CTO-200: Explicit test for PersonaSource == "" path (both "" and "bare" → bare+C).
func TestBuildPersonaEnvelope_EmptySource(t *testing.T) {
	session := &BridgeSession{
		PersonaSource: "",
	}
	env := BuildPersonaEnvelope(session)
	if env.PersonaSource != "bare" {
		t.Errorf("empty PersonaSource should resolve to bare, got %q", env.PersonaSource)
	}
	if env.Strategy != "C" {
		t.Errorf("expected C, got %q", env.Strategy)
	}
}

// CTO-111: Empty PersonaSource + non-empty AgentRole should still return bare (CTO-107 fix).
func TestBuildPersonaEnvelope_EmptySourceWithRole(t *testing.T) {
	session := &BridgeSession{
		AgentRole:     "coder",
		PersonaSource: "",
	}
	env := BuildPersonaEnvelope(session)
	if env.PersonaSource != "bare" {
		t.Errorf("empty PersonaSource with role should resolve to bare, got %q", env.PersonaSource)
	}
	if env.Strategy != "C" {
		t.Errorf("expected C, got %q", env.Strategy)
	}
	if env.AgentRole != "" {
		t.Errorf("bare envelope should have empty role, got %q", env.AgentRole)
	}
}

func TestBuildPersonaEnvelope_StrategyA(t *testing.T) {
	session := &BridgeSession{
		AgentRole:         "coder",
		SoulTemplateHash:  "abc123",
		PersonaSourceHash: "def456",
		PersonaSource:     "agent_file",
	}
	env := BuildPersonaEnvelope(session)
	if env.AgentRole != "coder" {
		t.Errorf("expected coder, got %q", env.AgentRole)
	}
	if env.Strategy != "A" {
		t.Errorf("expected A, got %q", env.Strategy)
	}
	if env.SoulTemplateHash != "abc123" {
		t.Errorf("expected abc123, got %q", env.SoulTemplateHash)
	}
	if env.PersonaSourceHash != "def456" {
		t.Errorf("expected def456, got %q", env.PersonaSourceHash)
	}
}

func TestBuildPersonaEnvelope_StrategyB(t *testing.T) {
	session := &BridgeSession{
		AgentRole:         "pm",
		SoulTemplateHash:  "hash1",
		PersonaSourceHash: "hash2",
		PersonaSource:     "append_prompt",
	}
	env := BuildPersonaEnvelope(session)
	if env.Strategy != "B" {
		t.Errorf("expected B, got %q", env.Strategy)
	}
	if env.PersonaSource != "append_prompt" {
		t.Errorf("expected append_prompt, got %q", env.PersonaSource)
	}
}

func TestBuildIntelligenceEnvelope(t *testing.T) {
	session := &BridgeSession{
		AgentRole:         "coder",
		SoulTemplateHash:  "abc",
		PersonaSourceHash: "def",
		PersonaSource:     "agent_file",
	}
	env := BuildIntelligenceEnvelope(session)
	if env == nil {
		t.Fatal("expected non-nil envelope")
	}
	if env.Persona == nil {
		t.Fatal("expected non-nil persona")
	}
	if env.Persona.AgentRole != "coder" {
		t.Errorf("expected coder, got %q", env.Persona.AgentRole)
	}
	if env.Persona.Strategy != "A" {
		t.Errorf("expected A, got %q", env.Persona.Strategy)
	}
}

func TestIntelligenceEnvelope_JSON(t *testing.T) {
	env := &SessionIntelligenceEnvelope{
		Persona: &PersonaEnvelope{
			AgentRole:         "pm",
			SoulTemplateHash:  "hash1",
			PersonaSourceHash: "hash2",
			PersonaSource:     "agent_file",
			Strategy:          "A",
		},
	}
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded SessionIntelligenceEnvelope
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Persona.AgentRole != "pm" {
		t.Errorf("roundtrip: expected pm, got %q", decoded.Persona.AgentRole)
	}
	if decoded.Persona.Strategy != "A" {
		t.Errorf("roundtrip: expected A, got %q", decoded.Persona.Strategy)
	}
}

func TestIntelligenceEnvelope_OmitEmpty(t *testing.T) {
	env := &SessionIntelligenceEnvelope{}
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(data) != "{}" {
		t.Errorf("empty envelope should marshal to {}, got %s", data)
	}
}

func TestMarshalTurnContext(t *testing.T) {
	tc := &TurnContext{
		SprintGoals: []string{"Implement SOUL injection"},
		Blockers:    []string{"CTO review pending"},
		FixHints:    []string{"Check provider.go line 42"},
	}
	data, err := MarshalTurnContext(tc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(data), "sprint_goals") {
		t.Error("expected sprint_goals in JSON")
	}
	if !strings.Contains(string(data), "Implement SOUL injection") {
		t.Error("expected goal content in JSON")
	}
}

func TestFormatTurnContextMarkdown(t *testing.T) {
	tc := &TurnContext{
		SprintGoals: []string{"Goal 1", "Goal 2"},
		Blockers:    []string{"Blocker A"},
		FixHints:    []string{"Hint X"},
	}
	md := FormatTurnContextMarkdown(tc)
	if !strings.Contains(md, "# Turn Context") {
		t.Error("expected Turn Context heading")
	}
	if !strings.Contains(md, "## Sprint Goals") {
		t.Error("expected Sprint Goals section")
	}
	if !strings.Contains(md, "- Goal 1") {
		t.Error("expected Goal 1 bullet")
	}
	if !strings.Contains(md, "## Known Blockers") {
		t.Error("expected Known Blockers section")
	}
	if !strings.Contains(md, "## Fix Hints") {
		t.Error("expected Fix Hints section")
	}
}

func TestFormatTurnContextMarkdown_Nil(t *testing.T) {
	if md := FormatTurnContextMarkdown(nil); md != "" {
		t.Errorf("nil TurnContext should return empty, got %q", md)
	}
}

func TestFormatTurnContextMarkdown_Empty(t *testing.T) {
	tc := &TurnContext{}
	if md := FormatTurnContextMarkdown(tc); md != "" {
		t.Errorf("empty TurnContext should return empty, got %q", md)
	}
}

func TestTurnContext_OmitEmpty(t *testing.T) {
	tc := &TurnContext{}
	data, err := json.Marshal(tc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(data) != "{}" {
		t.Errorf("empty TurnContext should marshal to {}, got %s", data)
	}
}
