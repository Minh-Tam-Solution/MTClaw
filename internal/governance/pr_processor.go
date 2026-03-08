package governance

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// ParsePRVerdict extracts a verdict from the reviewer SOUL's output.
// Looks for explicit verdict markers in the response text.
// Returns "pass", "fail", or "pending". Default is "pending" (CTO-48: ambiguous reviews must not silently approve).
func ParsePRVerdict(content string) string {
	lower := strings.ToLower(content)

	// Check for explicit fail markers
	failMarkers := []string{
		"verdict: fail",
		"verdict: reject",
		"**verdict**: fail",
		"**verdict**: reject",
		"status: fail",
		"🔴 fail",
		"❌ fail",
		"policy violation found",
		"changes requested",
	}
	for _, marker := range failMarkers {
		if strings.Contains(lower, marker) {
			return "fail"
		}
	}

	// Check for explicit pass markers
	passMarkers := []string{
		"verdict: pass",
		"verdict: approve",
		"**verdict**: pass",
		"**verdict**: approve",
		"status: pass",
		"🟢 pass",
		"✅ pass",
		"all checks passed",
		"lgtm",
	}
	for _, marker := range passMarkers {
		if strings.Contains(lower, marker) {
			return "pass"
		}
	}

	// CTO-48: Default to "pending" — ambiguous reviews must NOT silently approve.
	// ENFORCE mode requires explicit verdict markers in SOUL prompt output.
	return "pending"
}

// ProcessPRReview persists a PR Gate evaluation record for audit trail.
// Called after the reviewer SOUL responds to a PR webhook event.
func ProcessPRReview(ctx context.Context, prGateStore store.PRGateStore, content string,
	agentKey, tenantID, prURL, repo, headSHA, mode, channel string, prNumber int, traceID *uuid.UUID,
) string {
	if prGateStore == nil {
		return ""
	}

	verdict := ParsePRVerdict(content)

	// Build rules_evaluated from content (simple extraction of section headers)
	rulesEvaluated := extractRulesFromContent(content)
	rulesJSON, _ := json.Marshal(rulesEvaluated)

	eval := &store.PRGateEvaluation{
		OwnerID:        tenantID,
		TraceID:        traceID,
		PRURL:          prURL,
		PRNumber:       prNumber,
		Repo:           repo,
		HeadSHA:        headSHA,
		Mode:           mode,
		Verdict:        verdict,
		RulesEvaluated: rulesJSON,
		ReviewComment:  content,
		SoulAuthor:     agentKey,
		Channel:        channel, // CTO-40: populate channel column from migration 000016
	}

	if err := prGateStore.CreateEvaluation(ctx, eval); err != nil {
		slog.Error("pr_gate: failed to persist evaluation",
			"repo", repo, "pr", prNumber, "error", err)
		return ""
	}

	slog.Info("pr_gate: evaluation persisted",
		"id", eval.ID, "repo", repo, "pr", prNumber,
		"verdict", verdict, "agent", agentKey)
	return eval.ID.String()
}

// extractRulesFromContent extracts rule names from reviewer output.
// Looks for markdown list items that appear to be rule checks.
func extractRulesFromContent(content string) []string {
	var rules []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		// Match markdown checklist items: "- [x] Rule name" or "- [ ] Rule name"
		if strings.HasPrefix(line, "- [x]") || strings.HasPrefix(line, "- [X]") {
			rule := strings.TrimSpace(line[5:])
			if rule != "" {
				rules = append(rules, rule)
			}
		} else if strings.HasPrefix(line, "- [ ]") {
			rule := strings.TrimSpace(line[5:])
			if rule != "" {
				rules = append(rules, rule)
			}
		}
	}
	return rules
}
