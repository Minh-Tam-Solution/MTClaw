// Package governance extracts spec processing logic from gateway_consumer.
// Sprint 7: Rail #1 Spec Factory — structured spec output processing.
package governance

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// ProcessSpecOutput detects spec JSON in agent output and persists to governance_specs.
// Called after PM SOUL generates a /spec response.
// Sprint 12 (CTO Decision D2): returns SpecResult struct instead of string.
func ProcessSpecOutput(ctx context.Context, output string, specStore store.SpecStore, agentKey string, tenantID string, traceID *uuid.UUID, channel string) SpecResult {
	if specStore == nil || output == "" {
		return SpecResult{}
	}

	spec, err := ParseSpecJSON(output)
	if err != nil {
		slog.Debug("governance: no spec JSON detected in output", "error", err)
		return SpecResult{}
	}

	// Generate next spec ID (SPEC-YYYY-NNNN).
	// NextSpecID relies on RLS — tenant must be set in context (CTO-18).
	specID, err := specStore.NextSpecID(ctx, time.Now().Year())
	if err != nil {
		slog.Warn("governance: failed to generate spec ID", "error", err)
		return SpecResult{}
	}

	spec.SpecID = specID
	spec.OwnerID = tenantID // CTO-19: MUST set before CreateSpec (NOT NULL constraint)
	spec.SoulAuthor = agentKey
	spec.Channel = channel // CTO-40: populate channel column from migration 000016
	spec.TraceID = traceID
	spec.ContentHash = sha256Hex(output)
	spec.Status = store.SpecStatusDraft

	// Sprint 12: Spec quality gate (CTO Governance Audit GAP 1).
	// Evaluate quality AFTER all fields populated, BEFORE CreateSpec.
	quality := EvaluateSpecQuality(spec)
	if !quality.Pass {
		slog.Warn("governance: spec quality below threshold",
			"score", quality.Score, "reasons", quality.Reasons,
			"title", spec.Title, "author", agentKey)
		return SpecResult{Rejected: true, Quality: quality}
	}

	if err := specStore.CreateSpec(ctx, spec); err != nil {
		slog.Warn("governance: failed to create spec", "spec_id", specID, "error", err)
		return SpecResult{Quality: quality}
	}

	slog.Info("governance: spec created", "spec_id", specID, "title", spec.Title,
		"author", agentKey, "quality_score", quality.Score)
	return SpecResult{SpecID: specID, Quality: quality}
}

// ParseSpecJSON extracts spec JSON from agent output text.
// Looks for a JSON block containing "spec_version" field.
// Returns a GovernanceSpec with parsed fields, or error if not found.
func ParseSpecJSON(output string) (*store.GovernanceSpec, error) {
	// Find JSON block in output (may be wrapped in ```json ... ``` markers)
	jsonStr := extractJSONBlock(output)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON block found in output")
	}

	// Parse into intermediate struct matching SKILL.md v1.0.0 schema.
	var raw struct {
		SpecVersion           string          `json:"spec_version"`
		Title                 string          `json:"title"`
		Narrative             json.RawMessage `json:"narrative"`
		AcceptanceCriteria    json.RawMessage `json:"acceptance_criteria"`
		BDDScenarios          json.RawMessage `json:"bdd_scenarios"`
		Risks                 json.RawMessage `json:"risks"`
		TechnicalRequirements json.RawMessage `json:"technical_requirements"`
		Dependencies          json.RawMessage `json:"dependencies"`
		Priority              string          `json:"priority"`
		EstimatedEffort       string          `json:"estimated_effort"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	if raw.SpecVersion == "" || raw.Title == "" {
		return nil, fmt.Errorf("missing required fields: spec_version or title")
	}

	// Default priority and effort if not set.
	if raw.Priority == "" {
		raw.Priority = "P1"
	}
	if raw.EstimatedEffort == "" {
		raw.EstimatedEffort = "M"
	}

	return &store.GovernanceSpec{
		ID:                    uuid.New(),
		SpecVersion:           raw.SpecVersion,
		Title:                 raw.Title,
		Narrative:             raw.Narrative,
		AcceptanceCriteria:    raw.AcceptanceCriteria,
		BDDScenarios:          raw.BDDScenarios,
		Risks:                 raw.Risks,
		TechnicalRequirements: raw.TechnicalRequirements,
		Dependencies:          raw.Dependencies,
		Priority:              raw.Priority,
		EstimatedEffort:       raw.EstimatedEffort,
		Tier:                  "STANDARD",
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}, nil
}

// extractJSONBlock finds the first JSON object in text, handling ```json fences.
func extractJSONBlock(text string) string {
	// Try fenced code block first: ```json ... ```
	if idx := strings.Index(text, "```json"); idx >= 0 {
		start := idx + len("```json")
		end := strings.Index(text[start:], "```")
		if end > 0 {
			return strings.TrimSpace(text[start : start+end])
		}
	}

	// Try bare code block: ``` ... ```
	if idx := strings.Index(text, "```"); idx >= 0 {
		start := idx + len("```")
		// Skip optional language identifier on same line
		if nl := strings.IndexByte(text[start:], '\n'); nl >= 0 {
			start += nl + 1
		}
		end := strings.Index(text[start:], "```")
		if end > 0 {
			candidate := strings.TrimSpace(text[start : start+end])
			if strings.HasPrefix(candidate, "{") {
				return candidate
			}
		}
	}

	// Try finding raw JSON object: { ... }
	start := strings.IndexByte(text, '{')
	if start < 0 {
		return ""
	}
	// Find matching closing brace (simple nesting counter).
	// CTO-21: This does not respect string boundaries — e.g. {"desc": "use } brace"}
	// would truncate at the wrong }. Low risk for LLM-generated spec JSON which rarely
	// contains literal braces in string values. Fenced code blocks (above) are preferred.
	depth := 0
	for i := start; i < len(text); i++ {
		switch text[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return text[start : i+1]
			}
		}
	}
	return ""
}

// sha256Hex computes SHA256 hex digest of content.
func sha256Hex(content string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(content)))
}
