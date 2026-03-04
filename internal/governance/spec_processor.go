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

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// ProcessSpecOutput detects spec JSON in agent output and persists to governance_specs.
// Called after PM SOUL generates a /spec response.
// Returns the created spec ID or empty string if no spec detected.
func ProcessSpecOutput(ctx context.Context, output string, specStore store.SpecStore, agentKey string, traceID *uuid.UUID) string {
	if specStore == nil || output == "" {
		return ""
	}

	spec, err := ParseSpecJSON(output)
	if err != nil {
		slog.Debug("governance: no spec JSON detected in output", "error", err)
		return ""
	}

	// Generate next spec ID (SPEC-YYYY-NNNN).
	// NextSpecID relies on RLS — tenant must be set in context (CTO-18).
	specID, err := specStore.NextSpecID(ctx, time.Now().Year())
	if err != nil {
		slog.Warn("governance: failed to generate spec ID", "error", err)
		return ""
	}

	spec.SpecID = specID
	spec.SoulAuthor = agentKey
	spec.TraceID = traceID
	spec.ContentHash = sha256Hex(output)
	spec.Status = store.SpecStatusDraft

	if err := specStore.CreateSpec(ctx, spec); err != nil {
		slog.Warn("governance: failed to create spec", "spec_id", specID, "error", err)
		return ""
	}

	slog.Info("governance: spec created", "spec_id", specID, "title", spec.Title, "author", agentKey)
	return specID
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
	// Find matching closing brace (simple nesting counter)
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
