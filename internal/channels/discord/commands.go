package discord

import (
	"context"
	"strings"

	"github.com/Minh-Tam-Solution/MTClaw/internal/commands"
)

// handleBotCommand checks if the message is a known bot command and handles it.
// Returns true if the message was handled as a command.
func (c *Channel) handleBotCommand(ctx context.Context, chatID, text, senderID, peerKind string) bool {
	if len(text) == 0 || text[0] != '/' {
		return false
	}

	cmd := strings.SplitN(text, " ", 2)[0]
	cmd = strings.ToLower(cmd)

	switch cmd {
	case "/workspace":
		wsArg := ""
		if len(text) > len("/workspace") {
			wsArg = strings.TrimSpace(text[len("/workspace"):])
		}
		c.handleWorkspace(ctx, chatID, wsArg)
		return true

	case "/projects":
		c.handleProjects(ctx, chatID)
		return true

	case "/help":
		helpText := "Available commands:\n" +
			"/help — Show this help message\n" +
			"/spec <description> — Generate structured specification\n" +
			"/spec_list — List recent specifications\n" +
			"/spec_detail <id> — View specification detail\n" +
			"/review <pr_url> — Review a GitHub pull request\n" +
			"/teams — List available teams\n" +
			"/workspace — Show or change current workspace/repo\n" +
			"/projects — List available projects in workspace parent\n" +
			"/tasks — List team tasks\n" +
			"/task_detail <id> — View task detail\n" +
			"/writers — List file writers for this group\n" +
			"/addwriter @user — Add a file writer\n" +
			"/removewriter @user — Remove a file writer\n" +
			"/stop — Stop current running task\n" +
			"/stopall — Stop all running tasks\n" +
			"/reset — Reset conversation history\n" +
			"/status — Show bot status\n" +
			"\nUse @soul_name to route to a specific SOUL (e.g. @reviewer, @pm).\n" +
			"Use @team_name to route to a team lead (e.g. @engineering, @business)."
		c.sendChunkedText(chatID, helpText)
		return true

	case "/reset":
		meta := commands.CommandMetadata{Platform: "discord"}
		commands.PublishReset(c.Bus(), c.Name(), senderID, chatID, c.AgentID(), peerKind, meta)
		c.sendChunkedText(chatID, "Conversation history has been reset.")
		return true

	case "/stop":
		meta := commands.CommandMetadata{Platform: "discord"}
		commands.PublishStop(c.Bus(), c.Name(), senderID, chatID, c.AgentID(), peerKind, meta)
		return true

	case "/stopall":
		meta := commands.CommandMetadata{Platform: "discord"}
		commands.PublishStopAll(c.Bus(), c.Name(), senderID, chatID, c.AgentID(), peerKind, meta)
		return true

	case "/spec":
		taskText := ""
		if len(text) > len("/spec") {
			taskText = strings.TrimSpace(text[len("/spec"):])
		}
		if taskText == "" {
			c.sendChunkedText(chatID, "Usage: /spec <requirement description>\n\nExample: /spec Create login feature for Bflow mobile app")
			return true
		}
		c.sendChunkedText(chatID, "Generating spec...")
		meta := commands.CommandMetadata{Platform: "discord"}
		commands.PublishSpec(c.Bus(), c.Name(), senderID, chatID, peerKind, taskText, meta)
		return true

	case "/review":
		prURL := ""
		if len(text) > len("/review") {
			prURL = strings.TrimSpace(text[len("/review"):])
		}
		if prURL == "" || !strings.Contains(prURL, "/pull/") {
			c.sendChunkedText(chatID, "Usage: /review <github_pr_url>\n\nExample: /review https://github.com/org/repo/pull/123")
			return true
		}
		c.sendChunkedText(chatID, "Reviewing PR...")
		meta := commands.CommandMetadata{Platform: "discord"}
		commands.PublishReview(c.Bus(), c.Name(), senderID, chatID, peerKind, prURL, meta)
		return true

	case "/teams":
		teamsText := "Available Teams:\n\n" +
			"@engineering — SDLC Engineering (lead: @pm)\n" +
			"@business — Business Operations (lead: @assistant)\n" +
			"@advisory — Advisory Board (lead: @cto)\n" +
			"\nUse @team_name <message> to route to a team.\n" +
			"Use @agent_name <message> to route to a specific agent."
		c.sendChunkedText(chatID, teamsText)
		return true

	case "/status":
		statusText := "Bot status: Running\nChannel: Discord"
		c.sendChunkedText(chatID, statusText)
		return true

	case "/spec_list", "/spec-list":
		c.handleSpecList(ctx, chatID)
		return true

	case "/spec_detail", "/spec-detail":
		c.handleSpecDetail(ctx, chatID, text)
		return true

	case "/tasks":
		c.handleTasksList(ctx, chatID)
		return true

	case "/task_detail":
		c.handleTaskDetail(ctx, chatID, text)
		return true

	case "/writers":
		c.handleListWriters(ctx, chatID, peerKind)
		return true

	case "/addwriter":
		c.handleAddWriter(ctx, chatID, text, senderID, peerKind)
		return true

	case "/removewriter":
		c.handleRemoveWriter(ctx, chatID, text, senderID, peerKind)
		return true
	}

	return false
}
