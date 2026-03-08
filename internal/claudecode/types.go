// Package claudecode implements the Claude Code terminal bridge for MTClaw.
// It provides 2-way interaction with Claude Code sessions via tmux,
// with multi-tenant governance, capability-based permissions, and audit logging.
package claudecode

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// AgentProviderType identifies the AI coding agent backend.
type AgentProviderType string

const (
	AgentClaudeCode AgentProviderType = "claude-code"
	AgentCursor     AgentProviderType = "cursor"     // Sprint E+
	AgentCodexCLI   AgentProviderType = "codex-cli"  // Sprint E+
	AgentGeminiCLI  AgentProviderType = "gemini-cli" // Sprint E+
)

// SessionState represents the lifecycle state of a bridge session.
type SessionState string

const (
	SessionStateActive  SessionState = "active"
	SessionStateBusy    SessionState = "busy"
	SessionStateIdle    SessionState = "idle"
	SessionStateStopped SessionState = "stopped"
	SessionStateError   SessionState = "error"
)

// RiskMode is the user-facing risk level for a session (UX shorthand).
// Internal enforcement uses the 3-axis capability model (D2).
type RiskMode string

const (
	RiskModeRead        RiskMode = "read"
	RiskModePatch       RiskMode = "patch"
	RiskModeInteractive RiskMode = "interactive"
)

// InputMode controls what the user can type into sendKeys (D2 axis 1).
type InputMode string

const (
	InputStructuredOnly InputMode = "structured_only"
	InputFreeText       InputMode = "free_text"
)

// ToolPolicy controls what tool calls the agent is expected to make (D2 axis 2).
type ToolPolicy string

const (
	ToolPolicyObserve          ToolPolicy = "observe"
	ToolPolicyPatchAllowed     ToolPolicy = "patch_allowed"
	ToolPolicyExecWithApproval ToolPolicy = "exec_with_approval"
)

// SessionCapabilities holds the resolved capability axes for a session (D2).
type SessionCapabilities struct {
	InputMode    InputMode  `json:"input_mode"`
	ToolPolicy   ToolPolicy `json:"tool_policy"`
	CaptureLines int        `json:"capture_lines"`
	RedactHeavy  bool       `json:"redact_heavy"`
}

// CapabilitiesForRisk maps a RiskMode to its resolved capabilities (D2).
func CapabilitiesForRisk(mode RiskMode) SessionCapabilities {
	switch mode {
	case RiskModePatch:
		return SessionCapabilities{
			InputMode:    InputStructuredOnly,
			ToolPolicy:   ToolPolicyPatchAllowed,
			CaptureLines: 50,
			RedactHeavy:  false,
		}
	case RiskModeInteractive:
		return SessionCapabilities{
			InputMode:    InputFreeText,
			ToolPolicy:   ToolPolicyExecWithApproval,
			CaptureLines: 100,
			RedactHeavy:  false,
		}
	default: // read
		return SessionCapabilities{
			InputMode:    InputStructuredOnly,
			ToolPolicy:   ToolPolicyObserve,
			CaptureLines: 30,
			RedactHeavy:  true,
		}
	}
}

// ProviderCapabilities describes what a provider adapter supports (D9).
type ProviderCapabilities struct {
	PermissionHooks   bool `json:"permission_hooks"`
	TranscriptParsing bool `json:"transcript_parsing"`
	HookFormatVersion int  `json:"hook_format_version"`
}

// BridgeSession represents an active or completed bridge session (D8).
type BridgeSession struct {
	ID                   string              `json:"id"`
	AgentType            AgentProviderType   `json:"agent_type"`
	TmuxTarget           string              `json:"tmux_target"`
	ProjectPath          string              `json:"project_path"`
	WorkspaceFingerprint string              `json:"workspace_fingerprint"`
	Status               SessionState        `json:"status"`
	RiskMode             RiskMode            `json:"risk_mode"`
	Capabilities         SessionCapabilities `json:"capabilities"`
	OwnerActorID         string              `json:"owner_actor_id"`
	ApproverACL          []string            `json:"approver_acl"`
	NotifyACL            []string            `json:"notify_acl"`
	TenantID             string              `json:"tenant_id"`
	UserID               string              `json:"user_id"`
	Channel              string              `json:"channel"`
	ChatID               string              `json:"chat_id"`
	HookSecret           string              `json:"hook_secret,omitempty"`
	LocalInteractive     bool                `json:"local_interactive"`
	InteractiveEligible  bool                `json:"interactive_eligible"` // true if provider supports permission hooks (D7 Layer 0)
	AgentRole            string              `json:"agent_role,omitempty"`         // SOUL role injected ("pm", "coder", etc.) — empty = bare launch
	SoulTemplateHash     string              `json:"soul_template_hash,omitempty"` // SHA-256 of source SOUL-{role}.md file
	PersonaSourceHash    string              `json:"persona_source_hash,omitempty"` // SHA-256 of actual injected content (agent file or temp file)
	PersonaSource        string              `json:"persona_source,omitempty"`     // "agent_file" | "append_prompt" | "bare"
	Intelligence         *SessionIntelligenceEnvelope `json:"intelligence,omitempty"` // Derived from flat persona fields above. Rebuilt via BuildIntelligenceEnvelope in CreateSession.
	CreatedAt            time.Time           `json:"created_at"`
	LastActivityAt       time.Time           `json:"last_activity_at"`
}

// StopEvent is emitted when a Claude Code session finishes.
type StopEvent struct {
	SessionID  string `json:"session_id"`
	ExitCode   int    `json:"exit_code"`
	Summary    string `json:"summary,omitempty"`
	GitDiff    string `json:"git_diff,omitempty"`
	FinishedAt time.Time `json:"finished_at"`
}

// BridgeProject is a registered project directory for bridge sessions.
type BridgeProject struct {
	ID        string            `json:"id"`
	OwnerID   string            `json:"owner_id"`
	Name      string            `json:"name"`
	Path      string            `json:"path"`
	AgentType AgentProviderType `json:"agent_type"`
	CreatedAt time.Time         `json:"created_at"`
}

// AuditEvent records a bridge action for compliance (L3).
type AuditEvent struct {
	ID        int64                  `json:"id,omitempty"`
	OwnerID   string                 `json:"owner_id"`
	SessionID string                 `json:"session_id,omitempty"`
	ActorID   string                 `json:"actor_id"`
	Action    string                 `json:"action"`
	RiskMode  string                 `json:"risk_mode,omitempty"`
	Detail    map[string]interface{} `json:"detail,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

// AdmissionCheck defines resource limits for session creation.
type AdmissionCheck struct {
	MaxSessionsPerAgent int     `json:"max_sessions_per_agent"`
	MaxTotalSessions    int     `json:"max_total_sessions"`
	MaxCPUPercent       float64 `json:"max_cpu_percent"`
	MaxMemoryPercent    float64 `json:"max_memory_percent"`
	PerTenantSessionCap int     `json:"per_tenant_session_cap"`
	PerProjectSingleton bool    `json:"per_project_singleton"`
}

// DefaultAdmissionCheck returns conservative defaults.
func DefaultAdmissionCheck() AdmissionCheck {
	return AdmissionCheck{
		MaxSessionsPerAgent: 2,
		MaxTotalSessions:    6,
		MaxCPUPercent:       85.0,
		MaxMemoryPercent:    80.0,
		PerTenantSessionCap: 4,
		PerProjectSingleton: false,
	}
}

// MaxNotifyACL is the maximum number of notification recipients per session.
const MaxNotifyACL = 5

// GenerateSessionID creates a session ID: "br:{tenant8}:{rand8}".
func GenerateSessionID(tenantID string) (string, error) {
	tenantHash := sha256.Sum256([]byte(tenantID))
	tenantPrefix := hex.EncodeToString(tenantHash[:4]) // 8 hex chars

	randBytes := make([]byte, 4)
	if _, err := rand.Read(randBytes); err != nil {
		return "", fmt.Errorf("generate session ID: %w", err)
	}
	return fmt.Sprintf("br:%s:%s", tenantPrefix, hex.EncodeToString(randBytes)), nil
}

// GenerateHookSecret creates a 32-byte hex-encoded secret for HMAC signing.
func GenerateHookSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate hook secret: %w", err)
	}
	return hex.EncodeToString(b), nil
}
