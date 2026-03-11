package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// MaxTasksInList is the maximum number of tasks to show in a list.
const MaxTasksInList = 30

// TaskStatusIcon returns a short icon for each task status.
func TaskStatusIcon(status string) string {
	switch status {
	case "completed":
		return "done"
	case "in_progress":
		return ">>"
	case "blocked":
		return "!!"
	default: // pending
		return ".."
	}
}

// TruncateStr truncates a string to maxLen runes, appending "..." if truncated.
func TruncateStr(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// FormatTaskList formats a list of team tasks for display.
func FormatTaskList(teamName string, tasks []store.TeamTaskData) string {
	if len(tasks) == 0 {
		return fmt.Sprintf("No tasks for team %q.", teamName)
	}

	total := len(tasks)
	display := tasks
	if total > MaxTasksInList {
		display = tasks[:MaxTasksInList]
	}

	var sb strings.Builder
	if total > MaxTasksInList {
		sb.WriteString(fmt.Sprintf("Tasks for team %q (showing %d of %d):\n\n", teamName, MaxTasksInList, total))
	} else {
		sb.WriteString(fmt.Sprintf("Tasks for team %q (%d):\n\n", teamName, total))
	}
	for i, t := range display {
		owner := ""
		if t.OwnerAgentKey != "" {
			owner = " — @" + t.OwnerAgentKey
		}
		sb.WriteString(fmt.Sprintf("%d. [%s] %s%s\n", i+1, TaskStatusIcon(t.Status), t.Subject, owner))
	}
	sb.WriteString("\nUse /task_detail <task_id> to view details.")
	return sb.String()
}

// FormatTaskDetail formats a single task for display.
func FormatTaskDetail(t *store.TeamTaskData) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Task: %s\n", t.Subject))
	sb.WriteString(fmt.Sprintf("ID: %s\n", t.ID.String()))
	sb.WriteString(fmt.Sprintf("Status: [%s] %s\n", TaskStatusIcon(t.Status), t.Status))
	if t.OwnerAgentKey != "" {
		sb.WriteString(fmt.Sprintf("Owner: @%s\n", t.OwnerAgentKey))
	}
	sb.WriteString(fmt.Sprintf("Priority: %d\n", t.Priority))
	if !t.CreatedAt.IsZero() {
		sb.WriteString(fmt.Sprintf("Created: %s\n", t.CreatedAt.Format("2006-01-02 15:04")))
	}
	if t.Description != "" {
		sb.WriteString(fmt.Sprintf("\nDescription:\n%s\n", t.Description))
	}
	if t.Result != nil && *t.Result != "" {
		sb.WriteString(fmt.Sprintf("\nResult:\n%s\n", *t.Result))
	}
	if len(t.BlockedBy) > 0 {
		ids := make([]string, len(t.BlockedBy))
		for j, bid := range t.BlockedBy {
			ids[j] = bid.String()[:8]
		}
		sb.WriteString(fmt.Sprintf("\nBlocked by: %s\n", strings.Join(ids, ", ")))
	}
	return sb.String()
}

// ListTasks retrieves tasks for the agent's team and formats them.
func ListTasks(ctx context.Context, agentStore store.AgentStore, teamStore store.TeamStore, agentKey string) (string, error) {
	agentUUID, err := ResolveAgentUUID(ctx, agentStore, agentKey)
	if err != nil {
		return "", fmt.Errorf("could not resolve agent: %w", err)
	}

	team, err := teamStore.GetTeamForAgent(ctx, agentUUID)
	if err != nil {
		return "", fmt.Errorf("failed to look up team: %w", err)
	}
	if team == nil {
		return "This agent is not part of any team.", nil
	}

	tasks, err := teamStore.ListTasks(ctx, team.ID, "newest", store.TeamTaskFilterAll)
	if err != nil {
		return "", fmt.Errorf("failed to list tasks: %w", err)
	}

	return FormatTaskList(team.Name, tasks), nil
}

// GetTaskDetail retrieves and formats a single task by ID prefix.
func GetTaskDetail(ctx context.Context, agentStore store.AgentStore, teamStore store.TeamStore, agentKey, taskIDArg string) (string, error) {
	agentUUID, err := ResolveAgentUUID(ctx, agentStore, agentKey)
	if err != nil {
		return "", fmt.Errorf("could not resolve agent: %w", err)
	}

	team, err := teamStore.GetTeamForAgent(ctx, agentUUID)
	if err != nil {
		return "", fmt.Errorf("failed to look up team: %w", err)
	}
	if team == nil {
		return "", fmt.Errorf("this agent is not part of any team")
	}

	tasks, err := teamStore.ListTasks(ctx, team.ID, "newest", store.TeamTaskFilterAll)
	if err != nil {
		return "", fmt.Errorf("failed to list tasks: %w", err)
	}

	taskIDLower := strings.ToLower(taskIDArg)
	for i := range tasks {
		tid := tasks[i].ID.String()
		if tid == taskIDLower || strings.HasPrefix(tid, taskIDLower) {
			return FormatTaskDetail(&tasks[i]), nil
		}
	}

	return "", fmt.Errorf("task %q not found", taskIDArg)
}
