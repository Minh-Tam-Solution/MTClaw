package discord

import (
	"context"

	"github.com/Minh-Tam-Solution/MTClaw/internal/commands"
)

// handleWorkspace shows or changes the agent workspace directory.
// /workspace         — show current workspace
// /workspace <path>  — change workspace to <path>
func (c *Channel) handleWorkspace(ctx context.Context, chatID, text string) {
	if c.agentStore == nil {
		c.sendChunkedText(chatID, "Workspace commands require managed mode.")
		return
	}

	wsCmd := &commands.WorkspaceCmd{
		AgentStore: c.agentStore,
		Bus:        c.Bus(),
	}

	var result string
	var err error
	if text == "" {
		result, err = wsCmd.Get(ctx, c.AgentID())
	} else {
		result, err = wsCmd.Set(ctx, c.AgentID(), text)
	}

	if err != nil {
		c.sendChunkedText(chatID, err.Error())
		return
	}

	c.sendChunkedText(chatID, result)
}

// handleProjects lists subdirectories in the current workspace as "projects".
func (c *Channel) handleProjects(ctx context.Context, chatID string) {
	if c.agentStore == nil {
		c.sendChunkedText(chatID, "Project commands require managed mode.")
		return
	}

	result, err := commands.ListProjects(ctx, c.agentStore, c.AgentID())
	if err != nil {
		c.sendChunkedText(chatID, err.Error())
		return
	}

	c.sendChunkedText(chatID, result)
}
