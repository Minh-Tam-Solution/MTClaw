package claudecode

import "encoding/json"

// SessionIntelligenceEnvelope defines the intelligence context attached to a bridge session.
// Sprint 19: only PersonaEnvelope populated. Future sprints extend with additional slots.
type SessionIntelligenceEnvelope struct {
	Persona *PersonaEnvelope `json:"persona,omitempty"`
	// Sprint 20A+: Skills *SkillsEnvelope
	// Sprint 20B+: Context *ContextEnvelope
}

// PersonaEnvelope captures SOUL injection metadata for audit and diagnostics.
type PersonaEnvelope struct {
	AgentRole         string `json:"agent_role"`
	SoulTemplateHash  string `json:"soul_template_hash"`
	PersonaSourceHash string `json:"persona_source_hash"`
	PersonaSource     string `json:"persona_source"` // "agent_file" | "append_prompt" | "bare"
	Strategy          string `json:"strategy"`        // "A" | "B" | "C"
}

// TurnContext holds per-turn intelligence injected into Claude Code sessions.
// Sprint 19: struct defined + serialization. Injection deferred to Sprint 20B.
type TurnContext struct {
	SprintGoals []string `json:"sprint_goals,omitempty"`
	Blockers    []string `json:"blockers,omitempty"`
	FixHints    []string `json:"fix_hints,omitempty"`
}

// StrategyFromPersonaSource maps PersonaSource to strategy letter (A/B/C).
func StrategyFromPersonaSource(source string) string {
	switch source {
	case "agent_file":
		return "A"
	case "append_prompt":
		return "B"
	case "bare":
		return "C"
	default:
		return "C"
	}
}

// BuildPersonaEnvelope creates a PersonaEnvelope from BridgeSession fields.
// If PersonaSource is empty or "bare", always returns a bare envelope (Strategy C)
// regardless of other fields — an empty PersonaSource means no SOUL was injected.
func BuildPersonaEnvelope(session *BridgeSession) *PersonaEnvelope {
	if session.PersonaSource == "" || session.PersonaSource == "bare" {
		return &PersonaEnvelope{
			PersonaSource: "bare",
			Strategy:      "C",
		}
	}
	return &PersonaEnvelope{
		AgentRole:         session.AgentRole,
		SoulTemplateHash:  session.SoulTemplateHash,
		PersonaSourceHash: session.PersonaSourceHash,
		PersonaSource:     session.PersonaSource,
		Strategy:          StrategyFromPersonaSource(session.PersonaSource),
	}
}

// BuildIntelligenceEnvelope creates the full envelope from a session.
func BuildIntelligenceEnvelope(session *BridgeSession) *SessionIntelligenceEnvelope {
	return &SessionIntelligenceEnvelope{
		Persona: BuildPersonaEnvelope(session),
	}
}

// MarshalTurnContext serializes TurnContext to JSON for file injection.
func MarshalTurnContext(tc *TurnContext) ([]byte, error) {
	return json.Marshal(tc)
}

// FormatTurnContextMarkdown renders TurnContext as markdown for injection.
func FormatTurnContextMarkdown(tc *TurnContext) string {
	if tc == nil {
		return ""
	}

	var parts []string

	if len(tc.SprintGoals) > 0 {
		s := "## Sprint Goals\n"
		for _, g := range tc.SprintGoals {
			s += "- " + g + "\n"
		}
		parts = append(parts, s)
	}

	if len(tc.Blockers) > 0 {
		s := "## Known Blockers\n"
		for _, b := range tc.Blockers {
			s += "- " + b + "\n"
		}
		parts = append(parts, s)
	}

	if len(tc.FixHints) > 0 {
		s := "## Fix Hints\n"
		for _, h := range tc.FixHints {
			s += "- " + h + "\n"
		}
		parts = append(parts, s)
	}

	if len(parts) == 0 {
		return ""
	}

	result := "# Turn Context\n\n"
	for _, p := range parts {
		result += p + "\n"
	}
	return result
}
