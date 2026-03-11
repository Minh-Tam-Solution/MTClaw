package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/Minh-Tam-Solution/MTClaw/internal/commands"
	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// --- Team tasks ---

// taskStatusIcon returns an emoji icon for each task status (Telegram-specific).
func taskStatusIcon(status string) string {
	switch status {
	case "completed":
		return "✅"
	case "in_progress":
		return "🔄"
	case "blocked":
		return "⛔"
	default: // pending
		return "⏳"
	}
}

// handleTasksList handles the /tasks command — lists team tasks with inline keyboard.
func (c *Channel) handleTasksList(ctx context.Context, chatID int64, setThread func(*telego.SendMessageParams)) {
	chatIDObj := tu.ID(chatID)

	send := func(text string) {
		msg := tu.Message(chatIDObj, text)
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
	}

	if c.teamStore == nil {
		send("Team features are not available.")
		return
	}

	agentID, err := c.resolveAgentUUID(ctx)
	if err != nil {
		slog.Debug("tasks command: agent resolve failed", "error", err)
		send("Team features are not available (no agent).")
		return
	}

	team, err := c.teamStore.GetTeamForAgent(ctx, agentID)
	if err != nil {
		slog.Warn("tasks command: GetTeamForAgent failed", "error", err)
		send("Failed to look up team. Please try again.")
		return
	}
	if team == nil {
		send("This agent is not part of any team.")
		return
	}

	tasks, err := c.teamStore.ListTasks(ctx, team.ID, "newest", store.TeamTaskFilterAll)
	if err != nil {
		slog.Warn("tasks command: ListTasks failed", "error", err)
		send("Failed to list tasks. Please try again.")
		return
	}

	if len(tasks) == 0 {
		send(fmt.Sprintf("No tasks for team %q.", team.Name))
		return
	}

	total := len(tasks)
	display := tasks
	if total > commands.MaxTasksInList {
		display = tasks[:commands.MaxTasksInList]
	}

	// Use Telegram-specific formatting with emoji icons
	var sb strings.Builder
	if total > commands.MaxTasksInList {
		sb.WriteString(fmt.Sprintf("Tasks for team %q (showing %d of %d):\n\n", team.Name, commands.MaxTasksInList, total))
	} else {
		sb.WriteString(fmt.Sprintf("Tasks for team %q (%d):\n\n", team.Name, total))
	}
	for i, t := range display {
		owner := ""
		if t.OwnerAgentKey != "" {
			owner = " — @" + t.OwnerAgentKey
		}
		sb.WriteString(fmt.Sprintf("%d. %s %s%s\n", i+1, taskStatusIcon(t.Status), t.Subject, owner))
	}
	sb.WriteString("\nTap a button below to view details.")

	// Build inline keyboard — one button per task (Telegram-specific UX).
	var rows [][]telego.InlineKeyboardButton
	for i, t := range display {
		label := fmt.Sprintf("%d. %s %s", i+1, taskStatusIcon(t.Status), commands.TruncateStr(t.Subject, 35))
		rows = append(rows, []telego.InlineKeyboardButton{
			{Text: label, CallbackData: "td:" + t.ID.String()},
		})
	}

	msg := tu.Message(chatIDObj, sb.String())
	setThread(msg)
	if len(rows) > 0 {
		msg.ReplyMarkup = &telego.InlineKeyboardMarkup{InlineKeyboard: rows}
	}
	c.bot.SendMessage(ctx, msg)
}

// handleTaskDetail handles the /task_detail command — shows detail for a task.
func (c *Channel) handleTaskDetail(ctx context.Context, chatID int64, text string, setThread func(*telego.SendMessageParams)) {
	chatIDObj := tu.ID(chatID)

	send := func(t string) {
		for _, chunk := range chunkPlainText(t, telegramMaxMessageLen) {
			msg := tu.Message(chatIDObj, chunk)
			setThread(msg)
			c.bot.SendMessage(ctx, msg)
		}
	}

	parts := strings.SplitN(text, " ", 2)
	if len(parts) < 2 || strings.TrimSpace(parts[1]) == "" {
		send("Usage: /task_detail <task_id>")
		return
	}
	taskIDArg := strings.TrimSpace(parts[1])

	if c.teamStore == nil || c.agentStore == nil {
		send("Team features are not available.")
		return
	}

	result, err := commands.GetTaskDetail(ctx, c.agentStore, c.teamStore, c.AgentID(), taskIDArg)
	if err != nil {
		slog.Warn("task_detail command failed", "error", err)
		send(err.Error())
		return
	}

	send(result)
}

// handleCallbackQuery handles inline keyboard button presses.
func (c *Channel) handleCallbackQuery(ctx context.Context, query *telego.CallbackQuery) {
	// Always answer to dismiss the loading indicator.
	c.bot.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
	})

	// Route callback by prefix
	switch {
	case strings.HasPrefix(query.Data, "cc_approve:"), strings.HasPrefix(query.Data, "cc_deny:"):
		c.handlePermissionCallback(ctx, query)
		return
	case strings.HasPrefix(query.Data, "td:"):
		// fall through to task detail handler below
	default:
		return
	}

	taskIDStr := strings.TrimPrefix(query.Data, "td:")

	// Resolve chat ID from the callback's message.
	chatID := query.Message.GetChat().ID
	chatIDObj := tu.ID(chatID)

	send := func(text string) {
		for _, chunk := range chunkPlainText(text, telegramMaxMessageLen) {
			msg := tu.Message(chatIDObj, chunk)
			c.bot.SendMessage(ctx, msg)
		}
	}

	if c.teamStore == nil {
		send("Team features are not available.")
		return
	}

	agentID, err := c.resolveAgentUUID(ctx)
	if err != nil {
		send("Team features are not available (no agent).")
		return
	}

	team, err := c.teamStore.GetTeamForAgent(ctx, agentID)
	if err != nil || team == nil {
		send("Could not resolve team.")
		return
	}

	tasks, err := c.teamStore.ListTasks(ctx, team.ID, "newest", store.TeamTaskFilterAll)
	if err != nil {
		send("Failed to list tasks.")
		return
	}

	for i := range tasks {
		if tasks[i].ID.String() == taskIDStr {
			send(commands.FormatTaskDetail(&tasks[i]))
			return
		}
	}
	send(fmt.Sprintf("Task %s not found.", taskIDStr[:8]))
}
