// Sprint 12: Design-First Gate (T12-GOV-03).
// CTO Governance Audit GAP 6: pre-condition hook for @coder routing.
// CTO Decision 3: blocks code tasks, allows ad-hoc questions.
package governance

import (
	"context"
	"log/slog"
	"strings"

	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// adHocPrefixes identifies question patterns that bypass the design-first gate.
// CTO Decision D3: expanded with "can", "should", "is", "does", "could"
// to reduce false-positive blocking (R23 mitigation).
var adHocPrefixes = []string{
	"how ", "explain", "debug", "what ", "why ", "where ", "help ",
	"can ", "should ", "is ", "does ", "could ",
}

// DesignFirstGate checks if a code task has an approved spec before execution.
// Returns (pass, reason):
//
//	pass=true  → proceed to agent loop
//	pass=false → reason contains user-facing message
//
// Gate only applies to agentKey == "coder". All other agents pass through.
// Ad-hoc questions (how, explain, debug, etc.) pass through per CTO Decision 3.
func DesignFirstGate(ctx context.Context, agentKey string, content string, specStore store.SpecStore) (bool, string) {
	// Gate only applies to @coder.
	if agentKey != "coder" {
		return true, ""
	}

	// Ad-hoc questions always pass (CTO Decision 3).
	if isAdHocQuestion(content) {
		return true, ""
	}

	// Graceful degradation: nil store → pass.
	if specStore == nil {
		return true, ""
	}

	// Empty content → pass (no blocking on empty messages).
	if strings.TrimSpace(content) == "" {
		return true, ""
	}

	// Check: does an approved spec exist for this tenant context?
	specs, err := specStore.ListSpecs(ctx, store.SpecListOpts{
		Status: store.SpecStatusApproved,
		Limit:  1,
	})
	if err != nil {
		slog.Warn("governance: design-first gate query failed, allowing through",
			"error", err)
		return true, "" // fail-open on DB errors to avoid blocking all requests
	}

	if len(specs) == 0 {
		return false, "Design-First Gate: No approved spec found. " +
			"Please create a spec first using @pm /spec before delegating code tasks to @coder. " +
			"Ad-hoc questions (how, explain, debug, etc.) are allowed without a spec."
	}

	return true, ""
}

// isAdHocQuestion returns true if content matches question patterns.
// CTO Decision 3: "how do I...", "explain...", "debug this...", etc.
// Also matches any content ending with "?".
func isAdHocQuestion(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	if lower == "" {
		return false
	}

	for _, prefix := range adHocPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}

	return strings.HasSuffix(lower, "?")
}
