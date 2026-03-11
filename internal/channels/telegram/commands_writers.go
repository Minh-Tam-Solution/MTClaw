package telegram

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/Minh-Tam-Solution/MTClaw/internal/commands"
)

// handleWriterCommand handles /addwriter and /removewriter commands.
// The target user is identified by replying to one of their messages (Telegram UX).
func (c *Channel) handleWriterCommand(ctx context.Context, message *telego.Message, chatID int64, chatIDStr, senderID string, isGroup bool, setThread func(*telego.SendMessageParams), action string) {
	chatIDObj := tu.ID(chatID)

	send := func(text string) {
		msg := tu.Message(chatIDObj, text)
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
	}

	if !isGroup {
		send("This command only works in group chats.")
		return
	}

	if c.agentStore == nil {
		send("File writer management is not available.")
		return
	}

	// Extract target user from reply-to message (Telegram-specific)
	if message.ReplyToMessage == nil || message.ReplyToMessage.From == nil {
		verb := "add"
		if action == "remove" {
			verb = "remove"
		}
		send(fmt.Sprintf("To %s a writer: find a message from that person, swipe to reply it, then type /%swriter.", verb, verb))
		return
	}

	targetUser := message.ReplyToMessage.From
	targetID := fmt.Sprintf("%d", targetUser.ID)

	wsCmd := &commands.WritersCmd{
		AgentStore: c.agentStore,
		Bus:        c.Bus(),
	}

	var result string
	var err error
	switch action {
	case "add":
		result, err = wsCmd.AddWriter(ctx, c.AgentID(), c.Name(), chatIDStr, senderID, targetID, targetUser.FirstName, targetUser.Username)
	case "remove":
		result, err = wsCmd.RemoveWriter(ctx, c.AgentID(), c.Name(), chatIDStr, senderID, targetID, targetUser.FirstName, targetUser.Username)
	}

	if err != nil {
		slog.Warn("writer mutation failed", "action", action, "error", err)
		send(err.Error())
		return
	}

	send(result)
}

// handleListWriters handles the /writers command.
func (c *Channel) handleListWriters(ctx context.Context, chatID int64, chatIDStr string, isGroup bool, setThread func(*telego.SendMessageParams)) {
	chatIDObj := tu.ID(chatID)

	send := func(text string) {
		msg := tu.Message(chatIDObj, text)
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
	}

	if !isGroup {
		send("This command only works in group chats.")
		return
	}

	if c.agentStore == nil {
		send("File writer management is not available.")
		return
	}

	wsCmd := &commands.WritersCmd{
		AgentStore: c.agentStore,
		Bus:        c.Bus(),
	}

	result, err := wsCmd.ListWriters(ctx, c.AgentID(), c.Name(), chatIDStr)
	if err != nil {
		slog.Warn("list writers failed", "error", err)
		send("Failed to list writers. Please try again.")
		return
	}

	send(result)
}
