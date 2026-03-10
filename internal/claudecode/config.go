package claudecode

// BridgeConfig configures the Claude Code terminal bridge.
type BridgeConfig struct {
	Enabled       bool           `json:"enabled"`
	HookPort      int            `json:"hook_port,omitempty"`
	HookBind      string         `json:"hook_bind,omitempty"` // default "127.0.0.1"; set "0.0.0.0" when running in Docker
	SoulsDir      string         `json:"souls_dir,omitempty"`
	Admission     AdmissionCheck `json:"admission,omitempty"`
	AuditDir      string         `json:"audit_dir,omitempty"`
	StandaloneDir string         `json:"standalone_dir,omitempty"`
}

// DefaultBridgeConfig returns sensible defaults for the bridge.
func DefaultBridgeConfig() BridgeConfig {
	return BridgeConfig{
		Enabled:       false,
		HookPort:      18792,
		SoulsDir:      "docs/08-collaborate/souls", // relative to working directory
		Admission:     DefaultAdmissionCheck(),
		AuditDir:      "", // empty = ~/.mtclaw/bridge-audit/
		StandaloneDir: "", // empty = ~/.mtclaw/
	}
}
