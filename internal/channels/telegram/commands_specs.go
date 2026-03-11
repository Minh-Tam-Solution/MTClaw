package telegram

import (
	"context"
	"log/slog"
	"strings"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/Minh-Tam-Solution/MTClaw/internal/commands"
)

// handleSpecList lists recent governance specs for the current tenant.
func (c *Channel) handleSpecList(ctx context.Context, chatID int64, setThread func(*telego.SendMessageParams)) {
	chatIDObj := tu.ID(chatID)

	if c.specStore == nil {
		msg := tu.Message(chatIDObj, "Spec features are not available.")
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
		return
	}

	result, err := commands.ListSpecs(ctx, c.specStore)
	if err != nil {
		slog.Warn("spec-list: failed to list specs", "error", err)
		msg := tu.Message(chatIDObj, "Failed to list specs. Please try again.")
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
		return
	}

	msg := tu.Message(chatIDObj, result)
	setThread(msg)
	c.bot.SendMessage(ctx, msg)
}

// handleSpecDetail shows detailed information about a specific governance spec.
func (c *Channel) handleSpecDetail(ctx context.Context, chatID int64, text string, setThread func(*telego.SendMessageParams)) {
	chatIDObj := tu.ID(chatID)

	if c.specStore == nil {
		msg := tu.Message(chatIDObj, "Spec features are not available.")
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
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
		msg := tu.Message(chatIDObj, "Usage: /spec_detail <SPEC-ID>\n\nExample: /spec_detail SPEC-2026-0001")
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
		return
	}

	result, err := commands.GetSpecDetail(ctx, c.specStore, specID)
	if err != nil {
		slog.Warn("spec-detail: spec not found", "spec_id", specID, "error", err)
		msg := tu.Message(chatIDObj, err.Error())
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
		return
	}

	msg := tu.Message(chatIDObj, result)
	setThread(msg)
	c.bot.SendMessage(ctx, msg)
}
