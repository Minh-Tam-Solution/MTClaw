package discord

import (
	"context"
	"log/slog"
	"strings"

	"github.com/Minh-Tam-Solution/MTClaw/internal/commands"
)

// handleSpecList handles /spec_list — lists recent governance specs.
func (c *Channel) handleSpecList(ctx context.Context, chatID string) {
	if c.specStore == nil {
		c.sendChunkedText(chatID, "Spec features are not available.")
		return
	}

	result, err := commands.ListSpecs(ctx, c.specStore)
	if err != nil {
		slog.Warn("spec-list: failed", "error", err)
		c.sendChunkedText(chatID, "Failed to list specs. Please try again.")
		return
	}

	c.sendChunkedText(chatID, result)
}

// handleSpecDetail handles /spec_detail <SPEC-ID> — shows spec details.
func (c *Channel) handleSpecDetail(ctx context.Context, chatID, text string) {
	if c.specStore == nil {
		c.sendChunkedText(chatID, "Spec features are not available.")
		return
	}

	// Extract spec ID from command text.
	var specID string
	if strings.HasPrefix(text, "/spec_detail") {
		specID = strings.TrimSpace(text[len("/spec_detail"):])
	} else if strings.HasPrefix(text, "/spec-detail") {
		specID = strings.TrimSpace(text[len("/spec-detail"):])
	}

	if specID == "" {
		c.sendChunkedText(chatID, "Usage: /spec_detail <SPEC-ID>\n\nExample: /spec_detail SPEC-2026-0001")
		return
	}

	result, err := commands.GetSpecDetail(ctx, c.specStore, specID)
	if err != nil {
		slog.Warn("spec-detail: spec not found", "spec_id", specID, "error", err)
		c.sendChunkedText(chatID, err.Error())
		return
	}

	c.sendChunkedText(chatID, result)
}
