package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/nextlevelbuilder/goclaw/internal/bus"
)

// resolveAgentUUID looks up the agent UUID from the channel's agent key.
// Returns uuid.Nil if the agent key is empty or not found.
func (c *Channel) resolveAgentUUID(ctx context.Context) (uuid.UUID, error) {
	key := c.AgentID()
	if key == "" {
		return uuid.Nil, fmt.Errorf("no agent key configured")
	}

	// Try direct UUID parse first (future-proofing).
	if id, err := uuid.Parse(key); err == nil {
		return id, nil
	}

	// Look up by agent key.
	agent, err := c.agentStore.GetByKey(ctx, key)
	if err != nil {
		return uuid.Nil, fmt.Errorf("agent %q not found: %w", key, err)
	}
	return agent.ID, nil
}

// handleBotCommand checks if the message is a known bot command and handles it.
// Returns true if the message was handled as a command.
func (c *Channel) handleBotCommand(ctx context.Context, message *telego.Message, chatID int64, chatIDStr, localKey, text, senderID string, isGroup, isForum bool, messageThreadID int) bool {
	if len(text) == 0 || text[0] != '/' {
		return false
	}

	// Extract command (strip @botname suffix if present)
	cmd := strings.SplitN(text, " ", 2)[0]
	cmd = strings.ToLower(cmd)

	// In groups, ignore commands addressed to other bots (e.g. /help@other_bot)
	if isGroup {
		if parts := strings.SplitN(cmd, "@", 2); len(parts) == 2 {
			if !strings.EqualFold(parts[1], c.bot.Username()) {
				return false
			}
		}
	}

	cmd = strings.SplitN(cmd, "@", 2)[0]

	chatIDObj := tu.ID(chatID)

	// Helper: set MessageThreadID on outgoing messages for forum topics.
	// TS ref: buildTelegramThreadParams() — General topic (1) must be omitted.
	setThread := func(msg *telego.SendMessageParams) {
		sendThreadID := resolveThreadIDForSend(messageThreadID)
		if sendThreadID > 0 {
			msg.MessageThreadID = sendThreadID
		}
	}

	switch cmd {
	case "/start":
		// Don't intercept /start — let it pass through to agent loop.
		return false

	case "/help":
		helpText := "Available commands:\n" +
			"/start — Start chatting with the bot\n" +
			"/help — Show this help message\n" +
			"/spec <description> — Generate structured specification (Rail #1)\n" +
			"/spec_list — List recent specifications\n" +
			"/spec_detail <id> — View specification detail\n" +
			"/review <pr_url> — Review a GitHub pull request (Rail #2)\n" +
			"/teams — List available teams and how to mention them\n" +
			"/stop — Stop current running task\n" +
			"/stopall — Stop all running tasks\n" +
			"/reset — Reset conversation history\n" +
			"/status — Show bot status\n" +
			"/tasks — List team tasks\n" +
			"/task_detail <id> — View task detail\n" +
			"/writers — List file writers for this group\n" +
			"/addwriter — Add a file writer (reply to their message)\n" +
			"/removewriter — Remove a file writer (reply to their message)\n" +
			"\nUse @soul_name to route to a specific SOUL (e.g. @reviewer, @pm).\n" +
			"Use @team_name to route to a team lead (e.g. @engineering, @business).\n" +
			"Just send a message to chat with the AI."
		msg := tu.Message(chatIDObj, helpText)
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
		return true

	case "/spec":
		// Rail #1: Spec Factory — route to PM SOUL for structured spec generation.
		// Extract description text after "/spec ".
		taskText := strings.TrimSpace(text[len("/spec"):])
		if taskText == "" {
			usageMsg := tu.Message(chatIDObj, "Usage: /spec <requirement description>\n\nExample: /spec Create login feature for Bflow mobile app")
			setThread(usageMsg)
			c.bot.SendMessage(ctx, usageMsg)
			return true
		}

		// Send acknowledgment
		ackMsg := tu.Message(chatIDObj, "📋 Generating spec...")
		setThread(ackMsg)
		c.bot.SendMessage(ctx, ackMsg)

		// Publish to agent loop — PM SOUL handles /spec via spec-factory skill
		peerKind := "direct"
		if isGroup {
			peerKind = "group"
		}
		c.Bus().PublishInbound(bus.InboundMessage{
			Channel:  c.Name(),
			SenderID: senderID,
			ChatID:   chatIDStr,
			Content:  taskText,
			PeerKind: peerKind,
			AgentID:  "pm", // Always route to PM SOUL for /spec
			UserID:   strings.SplitN(senderID, "|", 2)[0],
			Metadata: map[string]string{
				"command":           "spec",
				"rail":              "spec-factory",
				"local_key":         localKey,
				"is_forum":          fmt.Sprintf("%t", isForum),
				"message_thread_id": fmt.Sprintf("%d", messageThreadID),
			},
		})
		return true

	case "/review":
		// Rail #2: PR Gate — route to reviewer SOUL for code review.
		prURL := strings.TrimSpace(text[len("/review"):])
		if prURL == "" || !strings.Contains(prURL, "/pull/") {
			usageMsg := tu.Message(chatIDObj, "Usage: /review <github_pr_url>\n\nExample: /review https://github.com/org/repo/pull/123")
			setThread(usageMsg)
			c.bot.SendMessage(ctx, usageMsg)
			return true
		}

		ackMsg := tu.Message(chatIDObj, "🔍 Reviewing PR...")
		setThread(ackMsg)
		c.bot.SendMessage(ctx, ackMsg)

		peerKind := "direct"
		if isGroup {
			peerKind = "group"
		}
		c.Bus().PublishInbound(bus.InboundMessage{
			Channel:  c.Name(),
			SenderID: senderID,
			ChatID:   chatIDStr,
			Content:  prURL,
			PeerKind: peerKind,
			AgentID:  "reviewer", // Always route to Reviewer SOUL
			UserID:   strings.SplitN(senderID, "|", 2)[0],
			Metadata: map[string]string{
				"command":           "review",
				"rail":              "pr-gate",
				"pr_url":            prURL,
				"local_key":         localKey,
				"is_forum":          fmt.Sprintf("%t", isForum),
				"message_thread_id": fmt.Sprintf("%d", messageThreadID),
			},
		})
		return true

	case "/teams":
		// Sprint 6: List available teams (US-037).
		// Hardcoded for Sprint 6 (3 static teams). Dynamic listing in Sprint 9+.
		teamsText := "Available Teams:\n\n" +
			"@engineering — SDLC Engineering (lead: @pm)\n" +
			"@business — Business Operations (lead: @assistant)\n" +
			"@advisory — Advisory Board (lead: @cto)\n" +
			"\nUse @team_name <message> to route to a team.\n" +
			"Use @agent_name <message> to route to a specific agent."
		teamsMsg := tu.Message(chatIDObj, teamsText)
		setThread(teamsMsg)
		c.bot.SendMessage(ctx, teamsMsg)
		return true

	case "/reset":
		// Fix: use correct PeerKind so the gateway consumer builds the right session key.
		peerKind := "direct"
		if isGroup {
			peerKind = "group"
		}
		c.Bus().PublishInbound(bus.InboundMessage{
			Channel:  c.Name(),
			SenderID: senderID,
			ChatID:   chatIDStr,
			Content:  "/reset",
			PeerKind: peerKind,
			AgentID:  c.AgentID(),
			UserID:   strings.SplitN(senderID, "|", 2)[0],
			Metadata: map[string]string{
				"command":           "reset",
				"local_key":         localKey,
				"is_forum":          fmt.Sprintf("%t", isForum),
				"message_thread_id": fmt.Sprintf("%d", messageThreadID),
			},
		})
		msg := tu.Message(chatIDObj, "Conversation history has been reset.")
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
		return true

	case "/stop":
		peerKind := "direct"
		if isGroup {
			peerKind = "group"
		}
		c.Bus().PublishInbound(bus.InboundMessage{
			Channel:  c.Name(),
			SenderID: senderID,
			ChatID:   chatIDStr,
			Content:  "/stop",
			PeerKind: peerKind,
			AgentID:  c.AgentID(),
			UserID:   strings.SplitN(senderID, "|", 2)[0],
			Metadata: map[string]string{
				"command":           "stop",
				"local_key":         localKey,
				"is_forum":          fmt.Sprintf("%t", isForum),
				"message_thread_id": fmt.Sprintf("%d", messageThreadID),
			},
		})
		// Feedback is sent by the consumer after cancel result is known.
		return true

	case "/stopall":
		peerKind := "direct"
		if isGroup {
			peerKind = "group"
		}
		c.Bus().PublishInbound(bus.InboundMessage{
			Channel:  c.Name(),
			SenderID: senderID,
			ChatID:   chatIDStr,
			Content:  "/stopall",
			PeerKind: peerKind,
			AgentID:  c.AgentID(),
			UserID:   strings.SplitN(senderID, "|", 2)[0],
			Metadata: map[string]string{
				"command":           "stopall",
				"local_key":         localKey,
				"is_forum":          fmt.Sprintf("%t", isForum),
				"message_thread_id": fmt.Sprintf("%d", messageThreadID),
			},
		})
		// Feedback is sent by the consumer after cancel result is known.
		return true

	case "/status":
		statusText := fmt.Sprintf("Bot status: Running\nChannel: Telegram\nBot: @%s", c.bot.Username())
		msg := tu.Message(chatIDObj, statusText)
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
		return true

	case "/spec_list", "/spec-list":
		c.handleSpecList(ctx, chatID, setThread)
		return true

	case "/spec_detail", "/spec-detail":
		c.handleSpecDetail(ctx, chatID, text, setThread)
		return true

	case "/tasks":
		c.handleTasksList(ctx, chatID, setThread)
		return true

	case "/task_detail":
		c.handleTaskDetail(ctx, chatID, text, setThread)
		return true

	case "/addwriter":
		c.handleWriterCommand(ctx, message, chatID, chatIDStr, senderID, isGroup, setThread, "add")
		return true

	case "/removewriter":
		c.handleWriterCommand(ctx, message, chatID, chatIDStr, senderID, isGroup, setThread, "remove")
		return true

	case "/writers":
		c.handleListWriters(ctx, chatID, chatIDStr, isGroup, setThread)
		return true
	}

	return false
}

