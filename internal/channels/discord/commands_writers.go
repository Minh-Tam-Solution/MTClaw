package discord

import (
	"context"
	"log/slog"
	"regexp"

	"github.com/Minh-Tam-Solution/MTClaw/internal/commands"
)

// discordMentionRe matches Discord user mentions: <@123456> or <@!123456>
var discordMentionRe = regexp.MustCompile(`<@!?(\d+)>`)

// handleListWriters handles /writers — lists file writers for this group channel.
func (c *Channel) handleListWriters(ctx context.Context, chatID, peerKind string) {
	if peerKind != "group" {
		c.sendChunkedText(chatID, "This command only works in group chats.")
		return
	}
	if c.agentStore == nil {
		c.sendChunkedText(chatID, "File writer management is not available.")
		return
	}

	wsCmd := &commands.WritersCmd{
		AgentStore: c.agentStore,
		Bus:        c.Bus(),
	}

	result, err := wsCmd.ListWriters(ctx, c.AgentID(), c.Name(), chatID)
	if err != nil {
		slog.Warn("list writers failed", "error", err)
		c.sendChunkedText(chatID, "Failed to list writers. Please try again.")
		return
	}

	c.sendChunkedText(chatID, result)
}

// handleAddWriter handles /addwriter @user — adds a file writer via Discord mention.
func (c *Channel) handleAddWriter(ctx context.Context, chatID, text, senderID, peerKind string) {
	c.handleWriterMutation(ctx, chatID, text, senderID, peerKind, "add")
}

// handleRemoveWriter handles /removewriter @user — removes a file writer via Discord mention.
func (c *Channel) handleRemoveWriter(ctx context.Context, chatID, text, senderID, peerKind string) {
	c.handleWriterMutation(ctx, chatID, text, senderID, peerKind, "remove")
}

// handleWriterMutation handles add/remove writer operations via Discord mention parsing.
func (c *Channel) handleWriterMutation(ctx context.Context, chatID, text, senderID, peerKind, action string) {
	if peerKind != "group" {
		c.sendChunkedText(chatID, "This command only works in group chats.")
		return
	}
	if c.agentStore == nil {
		c.sendChunkedText(chatID, "File writer management is not available.")
		return
	}

	// Parse Discord mention from command text: /addwriter <@123456>
	matches := discordMentionRe.FindStringSubmatch(text)
	if len(matches) < 2 {
		verb := "add"
		if action == "remove" {
			verb = "remove"
		}
		c.sendChunkedText(chatID, "To "+verb+" a writer, mention them: /"+verb+"writer @user")
		return
	}

	targetID := matches[1]

	wsCmd := &commands.WritersCmd{
		AgentStore: c.agentStore,
		Bus:        c.Bus(),
	}

	var result string
	var err error
	switch action {
	case "add":
		result, err = wsCmd.AddWriter(ctx, c.AgentID(), c.Name(), chatID, senderID, targetID, "", "")
	case "remove":
		result, err = wsCmd.RemoveWriter(ctx, c.AgentID(), c.Name(), chatID, senderID, targetID, "", "")
	}

	if err != nil {
		slog.Warn("writer mutation failed", "action", action, "error", err)
		c.sendChunkedText(chatID, err.Error())
		return
	}

	c.sendChunkedText(chatID, result)
}
