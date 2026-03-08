package claudecode

import (
	"fmt"
	"strings"
)

// PolicyCheck validates whether an action is allowed under the session's capability model (D2).

// CheckInputAllowed validates that the given input text is permitted under the session's InputMode.
// structured_only: only /cc commands (closed vocabulary). free_text: anything passes policy check.
func CheckInputAllowed(caps SessionCapabilities, text string) error {
	if caps.InputMode == InputFreeText {
		return nil
	}
	// structured_only: text must be empty (relay not allowed) or a /cc command
	if text == "" {
		return nil
	}
	if len(text) > 0 && text[0] == '/' {
		return nil // /cc commands are always structured
	}
	return fmt.Errorf("free-text input not allowed in %s mode (reason_code=input_mode_structured_only)", caps.InputMode)
}

// CheckCaptureAllowed validates that capturePane is allowed and returns the line limit.
func CheckCaptureAllowed(caps SessionCapabilities) (lines int, err error) {
	if caps.CaptureLines <= 0 {
		return 0, fmt.Errorf("capture not allowed (capture_lines=0)")
	}
	return caps.CaptureLines, nil
}

// CheckToolAllowed validates whether a tool call is permitted under the session's ToolPolicy.
// Returns nil if allowed, error describing why the tool is blocked otherwise.
func CheckToolAllowed(caps SessionCapabilities, toolName string, isWrite bool) error {
	switch caps.ToolPolicy {
	case ToolPolicyObserve:
		if isWrite {
			return fmt.Errorf("tool %q requires write access, blocked by observe policy (reason_code=tool_policy_observe)", toolName)
		}
		return nil
	case ToolPolicyPatchAllowed:
		return nil // read + write allowed
	case ToolPolicyExecWithApproval:
		return nil // allowed, but approval is checked elsewhere (Sprint C)
	default:
		return fmt.Errorf("unknown tool policy: %s", caps.ToolPolicy)
	}
}

// CheckRiskEscalation validates whether a risk mode change is permitted.
// Returns nil if the escalation is allowed for the given actor.
func CheckRiskEscalation(current, target RiskMode, actorID, ownerID string) error {
	// Same mode = no-op
	if current == target {
		return nil
	}

	targetLevel := riskLevel(target)
	currentLevel := riskLevel(current)

	// Downgrade: anyone can lower risk
	if targetLevel < currentLevel {
		return nil
	}

	// Escalation: only owner can escalate
	if actorID != ownerID {
		return fmt.Errorf("only session owner can escalate risk mode from %s to %s (reason_code=escalation_not_owner)", current, target)
	}

	return nil
}

// RoleDefaults returns the default RiskMode and tool allowlist for a SOUL role.
// Single load of templates + SOUL to avoid double I/O (CTO-122).
// This is a UX convenience — NOT a security boundary. The bridge capability
// model (D2) remains the only security gate.
type RoleDefaultsResult struct {
	RiskMode     RiskMode
	AllowedTools []string
}

func RoleDefaults(soulsDir, role string) RoleDefaultsResult {
	result := RoleDefaultsResult{RiskMode: RiskModeRead}

	templates, err := LoadAgentTemplates()
	if err != nil {
		return result
	}

	soul, err := LoadSOUL(soulsDir, role)
	if err != nil {
		return result
	}

	category := soul.Category
	if category == "" {
		category = "executor"
	}

	cat, ok := templates.Categories[category]
	if !ok {
		return result
	}

	result.AllowedTools = cat.Tools

	switch RiskMode(cat.DefaultRiskMode) {
	case RiskModePatch:
		result.RiskMode = RiskModePatch
	case RiskModeInteractive:
		result.RiskMode = RiskModeInteractive
	}

	return result
}

// RoleDefaultRiskMode returns the default RiskMode for a SOUL role.
// Delegates to RoleDefaults for single-load efficiency.
func RoleDefaultRiskMode(soulsDir, role string) RiskMode {
	return RoleDefaults(soulsDir, role).RiskMode
}

// AllowedToolsForRole returns the tool allowlist for a SOUL role's category.
// Delegates to RoleDefaults for single-load efficiency.
func AllowedToolsForRole(soulsDir, role string) []string {
	return RoleDefaults(soulsDir, role).AllowedTools
}

// FormatAllowedTools joins tool names for the --allowedTools CLI flag.
func FormatAllowedTools(tools []string) string {
	return strings.Join(tools, ", ")
}

// VerifyBridgeOverridesAgentFile documents and enforces the invariant that bridge
// governance (D2 capability model) takes precedence over agent file configuration.
//
// Specifically:
//   - Agent file `permissionMode` is irrelevant: --dangerously-skip-permissions is always set.
//     Bridge Layer 1 (async permission polling) replaces Claude Code's native permission system.
//   - Agent file `tools` is UX convenience only. Bridge ToolPolicy (observe/patch_allowed/
//     exec_with_approval) is the security gate checked by CheckToolAllowed.
//   - Agent file `model` is informational. Cost guardrails are enforced at the Bflow AI-Platform layer.
//
// This function is a no-op — it exists as documented code, not runtime enforcement.
// The actual enforcement is architectural: LaunchCommand always includes
// --dangerously-skip-permissions, and bridge checks happen before tmux relay.
func VerifyBridgeOverridesAgentFile() {
	// Intentionally empty. See docstring above.
	// If you're looking for the actual enforcement:
	//   - --dangerously-skip-permissions: provider.go:LaunchCommand (always first arg)
	//   - ToolPolicy check: bridge_policy.go:CheckToolAllowed (called before sendKeys)
	//   - InputMode check: bridge_policy.go:CheckInputAllowed (called before sendKeys)
}

// riskLevel returns a numeric level for ordering risk modes.
func riskLevel(mode RiskMode) int {
	switch mode {
	case RiskModeRead:
		return 0
	case RiskModePatch:
		return 1
	case RiskModeInteractive:
		return 2
	default:
		return -1
	}
}
