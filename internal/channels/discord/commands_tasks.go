package discord

import (
	"context"
	"log/slog"
	"strings"

	"github.com/Minh-Tam-Solution/MTClaw/internal/commands"
)

// handleTasksList handles /tasks — lists team tasks.
func (c *Channel) handleTasksList(ctx context.Context, chatID string) {
	if c.teamStore == nil {
		c.sendChunkedText(chatID, "Team features are not available.")
		return
	}
	if c.agentStore == nil {
		c.sendChunkedText(chatID, "Team features are not available (no agent).")
		return
	}

	result, err := commands.ListTasks(ctx, c.agentStore, c.teamStore, c.AgentID())
	if err != nil {
		slog.Warn("tasks command failed", "error", err)
		c.sendChunkedText(chatID, "Failed to list tasks. Please try again.")
		return
	}

	c.sendChunkedText(chatID, result)
}

// handleTaskDetail handles /task_detail <id> — shows task details.
func (c *Channel) handleTaskDetail(ctx context.Context, chatID, text string) {
	if c.teamStore == nil {
		c.sendChunkedText(chatID, "Team features are not available.")
		return
	}
	if c.agentStore == nil {
		c.sendChunkedText(chatID, "Team features are not available (no agent).")
		return
	}

	// Extract task ID argument: "/task_detail <id>"
	parts := strings.SplitN(text, " ", 2)
	if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
		c.sendChunkedText(chatID, "Usage: /task_detail <task_id>")
		return
	}
	taskIDArg := strings.TrimSpace(parts[1])

	result, err := commands.GetTaskDetail(ctx, c.agentStore, c.teamStore, c.AgentID(), taskIDArg)
	if err != nil {
		slog.Warn("task_detail command failed", "error", err)
		c.sendChunkedText(chatID, err.Error())
		return
	}

	c.sendChunkedText(chatID, result)
}
