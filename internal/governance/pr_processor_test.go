package governance

import (
	"testing"
)

func TestParsePRVerdict_ExplicitFail(t *testing.T) {
	cases := []struct {
		name    string
		content string
	}{
		{"verdict_fail", "## PR Review\n\n**Verdict**: Fail\n\nMissing tests."},
		{"verdict_reject", "Verdict: reject — code quality issues found."},
		{"red_circle", "🔴 Fail — security vulnerability detected."},
		{"policy_violation", "Policy violation found in auth module."},
		{"changes_requested", "Changes requested: please add error handling."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if v := ParsePRVerdict(tc.content); v != "fail" {
				t.Errorf("expected 'fail', got %q for content: %s", v, tc.content[:50])
			}
		})
	}
}

func TestParsePRVerdict_ExplicitPass(t *testing.T) {
	cases := []struct {
		name    string
		content string
	}{
		{"verdict_pass", "## PR Review\n\n**Verdict**: Pass\n\nLooks good."},
		{"verdict_approve", "Verdict: approve — code is clean."},
		{"green_circle", "🟢 Pass — all checks passed."},
		{"lgtm", "LGTM! Ship it."},
		{"all_checks", "All checks passed. No issues found."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if v := ParsePRVerdict(tc.content); v != "pass" {
				t.Errorf("expected 'pass', got %q for content: %s", v, tc.content[:50])
			}
		})
	}
}

func TestParsePRVerdict_DefaultPending(t *testing.T) {
	// CTO-48: No explicit markers → "pending" (not "pass").
	// Ambiguous reviews must NOT silently approve code.
	content := "This PR adds a new feature to the dashboard. It follows existing patterns."
	if v := ParsePRVerdict(content); v != "pending" {
		t.Errorf("expected default 'pending', got %q", v)
	}
}

func TestParsePRVerdict_NoMarkers_ReturnsPending(t *testing.T) {
	// CTO-48: ENFORCE mode with no verdict markers → "pending" for human escalation.
	cases := []string{
		"Here is my review of the code changes.",
		"The implementation looks reasonable but I have some concerns.",
		"",
	}
	for _, content := range cases {
		if v := ParsePRVerdict(content); v != "pending" {
			t.Errorf("expected 'pending' for content %q, got %q", content, v)
		}
	}
}

func TestExtractRulesFromContent(t *testing.T) {
	content := `## Review Checklist
- [x] Code follows style guide
- [X] Tests added
- [ ] Documentation updated
- Regular bullet point (not a rule)
`
	rules := extractRulesFromContent(content)
	if len(rules) != 3 {
		t.Errorf("expected 3 rules, got %d: %v", len(rules), rules)
	}
	if rules[0] != "Code follows style guide" {
		t.Errorf("expected 'Code follows style guide', got %q", rules[0])
	}
}
