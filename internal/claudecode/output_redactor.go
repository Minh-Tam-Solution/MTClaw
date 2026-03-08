package claudecode

import (
	"regexp"
	"strings"
)

// OutputRedactor scrubs secrets from capturePane output before sending to Telegram.
// Applies both pattern-based redaction and heavy redaction mode (D2 OutputPolicy).
type OutputRedactor struct {
	patterns []*redactPattern
}

type redactPattern struct {
	re          *regexp.Regexp
	replacement string
}

const redactedPlaceholder = "[REDACTED]"

// NewOutputRedactor creates a redactor with standard secret patterns.
func NewOutputRedactor() *OutputRedactor {
	return &OutputRedactor{
		patterns: defaultRedactPatterns(),
	}
}

// defaultRedactPatterns returns patterns that match common secrets in terminal output.
func defaultRedactPatterns() []*redactPattern {
	defs := []struct {
		pattern     string
		replacement string
	}{
		// API keys (generic)
		{`(?i)(api[_-]?key|apikey)\s*[=:]\s*\S+`, "${1}=" + redactedPlaceholder},
		{`(?i)(secret[_-]?key|secretkey)\s*[=:]\s*\S+`, "${1}=" + redactedPlaceholder},
		{`(?i)(access[_-]?key|accesskey)\s*[=:]\s*\S+`, "${1}=" + redactedPlaceholder},

		// Bearer tokens
		{`(?i)(bearer\s+)[A-Za-z0-9\-_\.]+`, "${1}" + redactedPlaceholder},
		{`(?i)(authorization\s*[=:]\s*)\S+`, "${1}" + redactedPlaceholder},

		// AWS keys
		{`AKIA[0-9A-Z]{16}`, redactedPlaceholder},

		// GitHub/GitLab tokens
		{`gh[ps]_[A-Za-z0-9_]{36,}`, redactedPlaceholder},
		{`glpat-[A-Za-z0-9\-_]{20,}`, redactedPlaceholder},

		// Slack tokens
		{`xox[baprs]-[A-Za-z0-9\-]+`, redactedPlaceholder},

		// PostgreSQL DSN
		{`(?i)postgres(ql)?://[^\s]+@[^\s]+`, "postgres://***@" + redactedPlaceholder},

		// MySQL DSN
		{`(?i)mysql://[^\s]+@[^\s]+`, "mysql://***@" + redactedPlaceholder},

		// Generic connection strings with passwords
		{`(?i)(password|passwd|pwd)\s*[=:]\s*\S+`, "${1}=" + redactedPlaceholder},

		// Hex-encoded secrets (32+ chars, likely keys/hashes)
		{`(?i)(encryption[_-]?key|hook[_-]?secret|hmac[_-]?secret)\s*[=:]\s*[0-9a-fA-F]{32,}`, "${1}=" + redactedPlaceholder},

		// GOCLAW-specific env vars that should never leak
		{`(?i)MTCLAW_[A-Z_]*KEY\s*=\s*\S+`, "MTCLAW_***KEY=" + redactedPlaceholder},
		{`(?i)MTCLAW_ENCRYPTION_KEY\s*=\s*\S+`, "MTCLAW_ENCRYPTION_KEY=" + redactedPlaceholder},
		{`(?i)MTCLAW_POSTGRES_DSN\s*=\s*\S+`, "MTCLAW_POSTGRES_DSN=" + redactedPlaceholder},

		// JWT tokens (3 base64 segments separated by dots)
		{`eyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`, redactedPlaceholder},

		// Private keys
		{`(?i)-----BEGIN\s+(RSA\s+)?PRIVATE KEY-----`, redactedPlaceholder},

		// Generic long hex strings that look like secrets (64+ hex chars on their own)
		{`(?m)^[0-9a-fA-F]{64,}$`, redactedPlaceholder},
	}

	patterns := make([]*redactPattern, 0, len(defs))
	for _, d := range defs {
		patterns = append(patterns, &redactPattern{
			re:          regexp.MustCompile(d.pattern),
			replacement: d.replacement,
		})
	}
	return patterns
}

// Redact applies secret redaction to terminal output.
// If heavyRedact is true (read mode), additional aggressive redaction is applied.
func (r *OutputRedactor) Redact(text string, heavyRedact bool) string {
	if text == "" {
		return text
	}

	result := text

	// Apply all secret patterns
	for _, p := range r.patterns {
		result = p.re.ReplaceAllString(result, p.replacement)
	}

	// Heavy redaction: also redact file paths that look like they contain secrets
	if heavyRedact {
		result = heavyRedactPaths(result)
	}

	return result
}

// TruncateOutput limits output to the given number of lines.
func TruncateOutput(text string, maxLines int) string {
	if maxLines <= 0 {
		return ""
	}
	lines := strings.Split(text, "\n")
	if len(lines) <= maxLines {
		return text
	}
	truncated := strings.Join(lines[:maxLines], "\n")
	return truncated + "\n... [truncated]"
}

// heavyRedactPaths redacts paths that might contain sensitive config files.
var sensitivePathPattern = regexp.MustCompile(`(?i)(/etc/(shadow|passwd|sudoers)|\.env|credentials|\.aws/|\.ssh/id_)`)

func heavyRedactPaths(text string) string {
	return sensitivePathPattern.ReplaceAllString(text, redactedPlaceholder)
}

// PatternCount returns the number of redaction patterns.
func (r *OutputRedactor) PatternCount() int {
	return len(r.patterns)
}
