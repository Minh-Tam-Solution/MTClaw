package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// handleSpecList lists recent governance specs for the current tenant.
// Sprint 7: Rail #1 Spec Factory — /spec_list command.
func (c *Channel) handleSpecList(ctx context.Context, chatID int64, setThread func(*telego.SendMessageParams)) {
	chatIDObj := tu.ID(chatID)

	if c.specStore == nil {
		msg := tu.Message(chatIDObj, "Spec features are not available.")
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
		return
	}

	specs, err := c.specStore.ListSpecs(ctx, store.SpecListOpts{Limit: 10})
	if err != nil {
		slog.Warn("spec-list: failed to list specs", "error", err)
		msg := tu.Message(chatIDObj, "Failed to list specs. Please try again.")
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
		return
	}

	if len(specs) == 0 {
		msg := tu.Message(chatIDObj, "No specs found. Use /spec <description> to create one.")
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
		return
	}

	var sb strings.Builder
	sb.WriteString("Recent Specifications:\n\n")
	for i, s := range specs {
		sb.WriteString(fmt.Sprintf("%d. %s — %s [%s]\n", i+1, s.SpecID, s.Title, s.Status))
	}
	sb.WriteString("\nUse /spec_detail <SPEC-ID> to view details.")

	msg := tu.Message(chatIDObj, sb.String())
	setThread(msg)
	c.bot.SendMessage(ctx, msg)
}

// handleSpecDetail shows detailed information about a specific governance spec.
// Sprint 7: Rail #1 Spec Factory — /spec_detail command.
func (c *Channel) handleSpecDetail(ctx context.Context, chatID int64, text string, setThread func(*telego.SendMessageParams)) {
	chatIDObj := tu.ID(chatID)

	if c.specStore == nil {
		msg := tu.Message(chatIDObj, "Spec features are not available.")
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
		return
	}

	// Extract spec ID from command text.
	// Supports both "/spec_detail SPEC-2026-0001" and "/spec-detail SPEC-2026-0001".
	var specID string
	if strings.HasPrefix(text, "/spec_detail") {
		specID = strings.TrimSpace(text[len("/spec_detail"):])
	} else if strings.HasPrefix(text, "/spec-detail") {
		specID = strings.TrimSpace(text[len("/spec-detail"):])
	}

	if specID == "" {
		msg := tu.Message(chatIDObj, "Usage: /spec_detail <SPEC-ID>\n\nExample: /spec_detail SPEC-2026-0001")
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
		return
	}

	spec, err := c.specStore.GetSpec(ctx, strings.ToUpper(specID))
	if err != nil {
		slog.Warn("spec-detail: spec not found", "spec_id", specID, "error", err)
		msg := tu.Message(chatIDObj, fmt.Sprintf("Spec %q not found.", specID))
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
		return
	}

	// Format spec detail.
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 %s — %s\n", spec.SpecID, spec.Title))
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
			sb.WriteString(fmt.Sprintf("  • %s\n    GIVEN %s\n    WHEN %s\n    THEN %s\n", ac.Scenario, ac.Given, ac.When, ac.Then))
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
			sb.WriteString(fmt.Sprintf("  ⚠️ %s (P: %s, I: %s)\n", r.Description, r.Probability, r.Impact))
		}
		sb.WriteString("\n")
	}

	// Evidence link
	if spec.TraceID != nil {
		sb.WriteString(fmt.Sprintf("Evidence: trace %s\n", spec.TraceID.String()[:8]))
	}

	msg := tu.Message(chatIDObj, sb.String())
	setThread(msg)
	c.bot.SendMessage(ctx, msg)
}
