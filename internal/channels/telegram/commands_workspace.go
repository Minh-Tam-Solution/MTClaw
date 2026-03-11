package telegram

import (
	"context"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/Minh-Tam-Solution/MTClaw/internal/commands"
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
		msg := tu.Message(chatIDObj, err.Error())
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
		return
	}

	msg := tu.Message(chatIDObj, result)
	setThread(msg)
	c.bot.SendMessage(ctx, msg)
}

// handleProjects lists subdirectories in the current workspace as "projects".
func (c *Channel) handleProjects(ctx context.Context, chatID int64, setThread func(*telego.SendMessageParams)) {
	chatIDObj := tu.ID(chatID)

	if c.agentStore == nil {
		msg := tu.Message(chatIDObj, "Project commands require managed mode.")
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
		return
	}

	result, err := commands.ListProjects(ctx, c.agentStore, c.AgentID())
	if err != nil {
		msg := tu.Message(chatIDObj, err.Error())
		setThread(msg)
		c.bot.SendMessage(ctx, msg)
		return
	}

	msg := tu.Message(chatIDObj, result)
	setThread(msg)
	c.bot.SendMessage(ctx, msg)
}
