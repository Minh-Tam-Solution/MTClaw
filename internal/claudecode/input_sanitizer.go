package claudecode

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Minh-Tam-Solution/MTClaw/internal/tools"
)

// InputSanitizer is a secondary defense layer that checks user input for dangerous patterns.
// Primary defense is the capability model (bridge_policy.go). This is defense-in-depth.
// Failure policy: REJECT (never pass-through on error).
type InputSanitizer struct {
	shellPatterns  []*regexp.Regexp // from tools.DefaultDenyPatterns()
	bridgePatterns []*regexp.Regexp // bridge-specific patterns
}

// NewInputSanitizer creates a sanitizer with shell deny patterns + bridge-specific patterns.
func NewInputSanitizer() *InputSanitizer {
	return &InputSanitizer{
		shellPatterns:  tools.DefaultDenyPatterns(),
		bridgePatterns: bridgeDenyPatterns(),
	}
}

// bridgeDenyPatterns returns patterns specific to the bridge context (tmux escape sequences,
// control characters, and bridge-specific dangerous inputs).
func bridgeDenyPatterns() []*regexp.Regexp {
	patterns := []string{
		// tmux escape sequences that could manipulate the session
		`\x1b[\[\]PX^_]`,         // ANSI escape sequences (CSI, OSC, DCS, SOS, PM, APC)
		`\x1bk[^\x1b]*\x1b\\`,   // tmux window title escape
		`\x1bP[^\x1b]*\x1b\\`,   // Device Control String
		`\x07`,                    // BEL character
		`\x00`,                    // NULL byte
		`\x03`,                    // Ctrl+C (ETX)
		`\x04`,                    // Ctrl+D (EOT — can close session)
		`\x1a`,                    // Ctrl+Z (suspend)

		// tmux command injection via prefix key sequences
		`tmux\s+(send-keys|send-prefix|run-shell|if-shell|display-message.*-p)`,
		`tmux\s+(source|load-buffer|save-buffer|set-buffer)`,
		`tmux\s+(new-window|split-window|respawn-pane).*\s+["']`,

		// Environment variable exfiltration via tmux
		`tmux\s+show-environment`,
		`tmux\s+set-environment`,

		// Escape from tmux to host shell
		`tmux\s+detach`,
		`tmux\s+kill-server`,
	}

	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		compiled = append(compiled, regexp.MustCompile(p))
	}
	return compiled
}

// Check validates input text against all deny patterns.
// Returns nil if safe, error with the matched pattern if dangerous.
func (s *InputSanitizer) Check(text string) error {
	if text == "" {
		return nil
	}

	// Normalize: collapse whitespace, trim
	normalized := strings.TrimSpace(text)
	if normalized == "" {
		return nil
	}

	// Check shell patterns (from tools.DefaultDenyPatterns)
	for _, pat := range s.shellPatterns {
		if pat.MatchString(normalized) {
			return fmt.Errorf("input blocked by shell deny pattern: %s (reason_code=sanitizer_shell_deny)", pat.String())
		}
	}

	// Check bridge-specific patterns
	for _, pat := range s.bridgePatterns {
		if pat.MatchString(normalized) {
			return fmt.Errorf("input blocked by bridge deny pattern: %s (reason_code=sanitizer_bridge_deny)", pat.String())
		}
	}

	return nil
}

// PatternCount returns the total number of deny patterns loaded.
func (s *InputSanitizer) PatternCount() int {
	return len(s.shellPatterns) + len(s.bridgePatterns)
}

// CheckInputSafe is a convenience function for one-shot input validation.
func CheckInputSafe(text string) error {
	return NewInputSanitizer().Check(text)
}
