package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// FormatSpecList formats a list of governance specs for display.
func FormatSpecList(specs []store.GovernanceSpec) string {
	if len(specs) == 0 {
		return "No specs found. Use /spec <description> to create one."
	}

	var sb strings.Builder
	sb.WriteString("Recent Specifications:\n\n")
	for i, s := range specs {
		sb.WriteString(fmt.Sprintf("%d. %s — %s [%s]\n", i+1, s.SpecID, s.Title, s.Status))
	}
	sb.WriteString("\nUse /spec_detail <SPEC-ID> to view details.")
	return sb.String()
}

// FormatSpecDetail formats a single governance spec for detailed display.
func FormatSpecDetail(spec *store.GovernanceSpec) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s — %s\n", spec.SpecID, spec.Title))
	sb.WriteString(fmt.Sprintf("Status: %s | Priority: %s | Effort: %s\n", spec.Status, spec.Priority, spec.EstimatedEffort))
	sb.WriteString(fmt.Sprintf("Author: @%s | Version: %s\n", spec.SoulAuthor, spec.SpecVersion))
	sb.WriteString(fmt.Sprintf("Created: %s\n\n", spec.CreatedAt.Format("2006-01-02 15:04")))

	// Narrative
	var narrative struct {
		AsA    string `json:"as_a"`
		IWant  string `json:"i_want"`
		SoThat string `json:"so_that"`
	}
	if json.Unmarshal(spec.Narrative, &narrative) == nil {
		sb.WriteString(fmt.Sprintf("As a %s\nI want %s\nSo that %s\n\n", narrative.AsA, narrative.IWant, narrative.SoThat))
	}

	// Acceptance criteria
	var criteria []struct {
		Scenario string `json:"scenario"`
		Given    string `json:"given"`
		When     string `json:"when"`
		Then     string `json:"then"`
	}
	if json.Unmarshal(spec.AcceptanceCriteria, &criteria) == nil && len(criteria) > 0 {
		sb.WriteString("Acceptance Criteria:\n")
		for _, ac := range criteria {
			sb.WriteString(fmt.Sprintf("  - %s\n    GIVEN %s\n    WHEN %s\n    THEN %s\n", ac.Scenario, ac.Given, ac.When, ac.Then))
		}
		sb.WriteString("\n")
	}

	// Risks
	var risks []struct {
		Description string `json:"description"`
		Probability string `json:"probability"`
		Impact      string `json:"impact"`
	}
	if len(spec.Risks) > 0 && json.Unmarshal(spec.Risks, &risks) == nil && len(risks) > 0 {
		sb.WriteString("Risks:\n")
		for _, r := range risks {
			sb.WriteString(fmt.Sprintf("  ! %s (P: %s, I: %s)\n", r.Description, r.Probability, r.Impact))
		}
		sb.WriteString("\n")
	}

	// Evidence link
	if spec.TraceID != nil {
		sb.WriteString(fmt.Sprintf("Evidence: trace %s\n", spec.TraceID.String()[:8]))
	}

	return sb.String()
}

// ListSpecs retrieves and formats specs from the store.
func ListSpecs(ctx context.Context, specStore store.SpecStore) (string, error) {
	specs, err := specStore.ListSpecs(ctx, store.SpecListOpts{Limit: 10})
	if err != nil {
		return "", fmt.Errorf("failed to list specs: %w", err)
	}
	return FormatSpecList(specs), nil
}

// GetSpecDetail retrieves and formats a single spec from the store.
func GetSpecDetail(ctx context.Context, specStore store.SpecStore, specID string) (string, error) {
	spec, err := specStore.GetSpec(ctx, strings.ToUpper(specID))
	if err != nil {
		return "", fmt.Errorf("spec %q not found", specID)
	}
	return FormatSpecDetail(spec), nil
}
