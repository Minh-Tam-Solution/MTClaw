package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/Minh-Tam-Solution/MTClaw/internal/bus"
	"github.com/Minh-Tam-Solution/MTClaw/pkg/protocol"
)

// handleWorkspace shows or changes the agent workspace directory.
// /workspace         — show current workspace
// /workspace <path>  — change workspace to <path>
func (c *Channel) handleWorkspace(ctx context.Context, chatID int64, text string, setThread func(*telego.SendMessageParams)) {
	chatIDObj := tu.ID(chatID)

	if c.agentStore == nil {
		msg := tu.Message(chatIDObj, "Workspace commands require managed mode.")
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
		return
	}

	agentUUID, err := c.resolveAgentUUID(ctx)
	if err != nil {
		msg := tu.Message(chatIDObj, "Could not resolve agent: "+err.Error())
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
		return
	}

	agent, err := c.agentStore.GetByID(ctx, agentUUID)
	if err != nil {
		msg := tu.Message(chatIDObj, "Could not load agent config: "+err.Error())
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
		return
	}

	// Extract new path from command text.
	newPath := strings.TrimSpace(text)

	// No argument: show current workspace.
	if newPath == "" {
		ws := agent.Workspace
		if ws == "" {
			ws = "(not set — using container default)"
		}
		msg := tu.Message(chatIDObj, fmt.Sprintf("Current workspace: %s\n\nUsage: /workspace <path>\nExample: /workspace /home/nqh/shared/MTClaw", ws))
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
		return
	}

	// Resolve ~ and make absolute.
	if strings.HasPrefix(newPath, "~") {
		home, _ := os.UserHomeDir()
		newPath = filepath.Join(home, newPath[1:])
	}
	if !filepath.IsAbs(newPath) {
		if agent.Workspace != "" {
			newPath = filepath.Join(agent.Workspace, newPath)
		} else {
			msg := tu.Message(chatIDObj, "Please provide an absolute path (e.g. /home/nqh/shared/MTClaw).")
			setThread(msg)
			c.bot.SendMessage(ctx, msg)
			return
		}
	}

	// Clean path (remove trailing slash, double slashes, etc.)
	newPath = filepath.Clean(newPath)

	// Validate the path exists and is a directory.
	info, statErr := os.Stat(newPath)
	if statErr != nil {
		msg := tu.Message(chatIDObj, fmt.Sprintf("Path not found: %s", newPath))
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
		return
	}
	if !info.IsDir() {
		msg := tu.Message(chatIDObj, fmt.Sprintf("Not a directory: %s", newPath))
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
		return
	}

	// Update workspace in DB.
	if err := c.agentStore.Update(ctx, agentUUID, map[string]any{
		"workspace": newPath,
	}); err != nil {
		slog.Error("workspace: failed to update", "agent", agent.AgentKey, "path", newPath, "error", err)
		msg := tu.Message(chatIDObj, "Failed to update workspace: "+err.Error())
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
		return
	}

	// Invalidate agent cache so the loop re-resolves with the new workspace.
	c.Bus().Broadcast(bus.Event{
		Name:    protocol.EventCacheInvalidate,
		Payload: bus.CacheInvalidatePayload{Kind: bus.CacheKindAgent, Key: agent.AgentKey},
	})

	slog.Info("workspace: updated", "agent", agent.AgentKey, "old", agent.Workspace, "new", newPath)

	msg := tu.Message(chatIDObj, fmt.Sprintf("Workspace changed to: %s\n\nAgent @%s will now operate in this directory.", newPath, agent.AgentKey))
	setThread(msg)
	c.bot.SendMessage(ctx, msg)
}

// handleProjects lists subdirectories in the current workspace as "projects".
// /projects — list directories in the workspace root
func (c *Channel) handleProjects(ctx context.Context, chatID int64, setThread func(*telego.SendMessageParams)) {
	chatIDObj := tu.ID(chatID)

	if c.agentStore == nil {
		msg := tu.Message(chatIDObj, "Project commands require managed mode.")
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
		return
	}

	agentUUID, err := c.resolveAgentUUID(ctx)
	if err != nil {
		msg := tu.Message(chatIDObj, "Could not resolve agent: "+err.Error())
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
		return
	}

	agent, err := c.agentStore.GetByID(ctx, agentUUID)
	if err != nil {
		msg := tu.Message(chatIDObj, "Could not load agent config: "+err.Error())
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
		return
	}

	ws := agent.Workspace
	if ws == "" {
		msg := tu.Message(chatIDObj, "No workspace configured. Use /workspace <path> to set one first.")
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
		return
	}

	// Look one level up from workspace to list sibling projects.
	// filepath.Clean removes trailing slash so Dir works correctly:
	// "/home/nqh/shared/Bflow-Platform/" → "/home/nqh/shared/Bflow-Platform" → Dir = "/home/nqh/shared"
	parentDir := filepath.Dir(filepath.Clean(ws))
	entries, err := os.ReadDir(parentDir)
	if err != nil {
		msg := tu.Message(chatIDObj, fmt.Sprintf("Cannot read directory: %s", parentDir))
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
		return
	}

	var dirs []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			dirs = append(dirs, e.Name())
		}
	}

	if len(dirs) == 0 {
		msg := tu.Message(chatIDObj, fmt.Sprintf("No projects found in %s", parentDir))
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
		return
	}

	sort.Strings(dirs)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Projects in %s:\n\n", parentDir))
	for i, d := range dirs {
		marker := "  "
		if filepath.Join(parentDir, d) == filepath.Clean(ws) {
			marker = "> " // current workspace
		}
		sb.WriteString(fmt.Sprintf("%s%d. %s\n", marker, i+1, d))
	}
	sb.WriteString(fmt.Sprintf("\nCurrent: %s\nSwitch: /workspace %s/<name>", ws, parentDir))

	msg := tu.Message(chatIDObj, sb.String())
	setThread(msg)
	c.bot.SendMessage(ctx, msg)
}
