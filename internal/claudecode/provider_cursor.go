package claudecode

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CursorProjectionAdapter implements ProviderAdapter for Cursor.
// Cursor has no headless CLI — this adapter supports file generation only.
// Session management operations return unsupported errors.
type CursorProjectionAdapter struct{}

func (a *CursorProjectionAdapter) Name() AgentProviderType {
	return AgentCursor
}

// LaunchCommand returns nil — Cursor has no headless CLI mode.
func (a *CursorProjectionAdapter) LaunchCommand(_ LaunchOpts) *exec.Cmd {
	return nil // unsupported: no_headless_cli
}

// InstallHooks returns an error — Cursor has no hook system.
func (a *CursorProjectionAdapter) InstallHooks(_, _ string) error {
	return fmt.Errorf("%s: no hook support (unsupported_reason=no_permission_hooks)", AgentCursor)
}

// UninstallHooks returns an error — no hooks to uninstall.
func (a *CursorProjectionAdapter) UninstallHooks() error {
	return fmt.Errorf("%s: no hook support (unsupported_reason=no_permission_hooks)", AgentCursor)
}

// ParseStopEvent returns an error — Cursor has no transcript/event output.
func (a *CursorProjectionAdapter) ParseStopEvent(_ []byte) (*StopEvent, error) {
	return nil, fmt.Errorf("%s: no transcript parsing (unsupported_reason=no_transcript_parsing)", AgentCursor)
}

// CapabilityProfile reports Cursor's limited capabilities honestly.
func (a *CursorProjectionAdapter) CapabilityProfile() ProviderCapabilities {
	return ProviderCapabilities{
		PermissionHooks:   false,
		TranscriptParsing: false,
		HookFormatVersion: 0,
	}
}

// TranscriptPath returns empty — Cursor has no transcript output.
func (a *CursorProjectionAdapter) TranscriptPath(_ string) string {
	return ""
}

// CursorRule represents a generated .cursor/rules/*.mdc file.
type CursorRule struct {
	Role        string // SOUL role name
	AlwaysApply bool   // load for every conversation
	Body        string // SOUL markdown body
}

// FormatMDC formats a CursorRule as Cursor's .mdc file content.
func (r *CursorRule) FormatMDC() string {
	var sb strings.Builder
	sb.WriteString(generatedHeader + "\n")
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("description: SOUL persona for %s role\n", r.Role))
	if r.AlwaysApply {
		sb.WriteString("alwaysApply: true\n")
	}
	sb.WriteString("---\n\n")
	sb.WriteString(r.Body)
	sb.WriteString("\n")
	return sb.String()
}

// GenerateCursorRulesResult reports what happened during Cursor rule generation.
type GenerateCursorRulesResult struct {
	Installed int
	Updated   int
	Skipped   int
}

// GenerateCursorRules converts SOUL files to Cursor .mdc rule files.
// Output: {projectPath}/.cursor/rules/{role}.mdc for each SOUL.
func GenerateCursorRules(projectPath, soulsDir string, roles []string, force bool) (*GenerateCursorRulesResult, error) {
	allRoles, err := KnownRoles(soulsDir)
	if err != nil {
		return nil, err
	}

	targetRoles := allRoles
	if len(roles) > 0 {
		targetRoles = filterRoles(allRoles, roles)
		if len(targetRoles) == 0 {
			return nil, fmt.Errorf("none of the specified roles found in SOUL files")
		}
	}

	rulesDir := filepath.Join(projectPath, ".cursor", "rules")
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		return nil, fmt.Errorf("create cursor rules dir: %w", err)
	}

	result := &GenerateCursorRulesResult{}

	for _, role := range targetRoles {
		soul, err := LoadSOUL(soulsDir, role)
		if err != nil {
			return nil, fmt.Errorf("load SOUL %s: %w", role, err)
		}

		rule := &CursorRule{
			Role:        soul.Role,
			AlwaysApply: true,
			Body:        soul.Body,
		}
		content := rule.FormatMDC()

		ruleFile := filepath.Join(rulesDir, role+".mdc")

		if existingData, readErr := os.ReadFile(ruleFile); readErr == nil {
			existingContent := string(existingData)
			if !strings.HasPrefix(existingContent, generatedHeader) && !force {
				result.Skipped++
				continue
			}
			if !force {
				existingHash := hashString(existingContent)
				newHash := hashString(content)
				if existingHash == newHash {
					result.Skipped++
					continue
				}
			}
			result.Updated++
		} else {
			result.Installed++
		}

		if err := os.WriteFile(ruleFile, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("write cursor rule %s: %w", ruleFile, err)
		}
	}

	return result, nil
}

// CursorProjectionInfo returns a summary of what Cursor supports for audit/info display.
func CursorProjectionInfo() map[string]interface{} {
	return map[string]interface{}{
		"provider":           string(AgentCursor),
		"session_management": false,
		"file_generation":    true,
		"persona_mechanism":  ".cursor/rules/*.mdc",
		"knowledge_file":     ".cursorrules or .cursor/rules/",
		"hooks":              false,
		"permission_model":   "IDE-managed",
		"unsupported_reason": "no_headless_cli",
	}
}

// ensure CursorProjectionAdapter implements ProviderAdapter at compile time.
var _ ProviderAdapter = (*CursorProjectionAdapter)(nil)
