package commands

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/Minh-Tam-Solution/MTClaw/internal/bus"
	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
	"github.com/Minh-Tam-Solution/MTClaw/pkg/protocol"
)

// WorkspaceCmd handles /workspace get and set operations.
type WorkspaceCmd struct {
	AgentStore store.AgentStore
	Bus        *bus.MessageBus
}

// Get returns a human-readable string showing the current workspace for the agent.
func (w *WorkspaceCmd) Get(ctx context.Context, agentKey string) (string, error) {
	agentUUID, err := ResolveAgentUUID(ctx, w.AgentStore, agentKey)
	if err != nil {
		return "", fmt.Errorf("could not resolve agent: %w", err)
	}

	agent, err := w.AgentStore.GetByID(ctx, agentUUID)
	if err != nil {
		return "", fmt.Errorf("could not load agent config: %w", err)
	}

	ws := agent.Workspace
	if ws == "" {
		ws = "(not set — using container default)"
	}
	return fmt.Sprintf("Current workspace: %s\n\nUsage: /workspace <path>\nExample: /workspace /home/nqh/shared/MTClaw", ws), nil
}

// Set changes the workspace for the agent and returns a confirmation message.
func (w *WorkspaceCmd) Set(ctx context.Context, agentKey, newPath string) (string, error) {
	agentUUID, err := ResolveAgentUUID(ctx, w.AgentStore, agentKey)
	if err != nil {
		return "", fmt.Errorf("could not resolve agent: %w", err)
	}

	agent, err := w.AgentStore.GetByID(ctx, agentUUID)
	if err != nil {
		return "", fmt.Errorf("could not load agent config: %w", err)
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
			return "", fmt.Errorf("please provide an absolute path (e.g. /home/nqh/shared/MTClaw)")
		}
	}

	newPath = filepath.Clean(newPath)

	// Validate the path exists and is a directory.
	info, statErr := os.Stat(newPath)
	if statErr != nil {
		return "", fmt.Errorf("path not found: %s", newPath)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("not a directory: %s", newPath)
	}

	// Update workspace in DB.
	if err := w.AgentStore.Update(ctx, agentUUID, map[string]any{
		"workspace": newPath,
	}); err != nil {
		slog.Error("workspace: failed to update", "agent", agent.AgentKey, "path", newPath, "error", err)
		return "", fmt.Errorf("failed to update workspace: %w", err)
	}

	// Invalidate agent cache so the loop re-resolves with the new workspace.
	w.Bus.Broadcast(bus.Event{
		Name:    protocol.EventCacheInvalidate,
		Payload: bus.CacheInvalidatePayload{Kind: bus.CacheKindAgent, Key: agent.AgentKey},
	})

	// Auto-reload PROJECT.md from new workspace's CLAUDE.md or README.md.
	w.reloadProjectContext(ctx, agentUUID, agent.AgentKey, newPath)

	slog.Info("workspace: updated", "agent", agent.AgentKey, "old", agent.Workspace, "new", newPath)

	return fmt.Sprintf("Workspace changed to: %s\n\nAgent @%s will now operate in this directory.", newPath, agent.AgentKey), nil
}

// reloadProjectContext reads CLAUDE.md (or README.md) from the new workspace
// and upserts it as PROJECT.md in agent_context_files. This ensures the agent's
// system prompt reflects the current workspace, not the previous one.
func (w *WorkspaceCmd) reloadProjectContext(ctx context.Context, agentID uuid.UUID, agentKey, workspace string) {
	if w.AgentStore == nil {
		return
	}

	// Try CLAUDE.md first (richest context), then README.md.
	candidates := []string{"CLAUDE.md", "README.md"}
	for _, name := range candidates {
		path := filepath.Join(workspace, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		content := string(data)
		if len(content) == 0 {
			continue
		}

		// Truncate to 4000 chars to stay within context budget.
		if len(content) > 4000 {
			content = content[:4000] + "\n\n(truncated)"
		}

		header := fmt.Sprintf("# Project Context (auto-loaded from %s/%s)\n\n", workspace, name)
		if err := w.AgentStore.SetAgentContextFile(ctx, agentID, "PROJECT.md", header+content); err != nil {
			slog.Warn("workspace: failed to reload PROJECT.md", "agent", agentKey, "error", err)
			return
		}

		slog.Info("workspace: PROJECT.md reloaded", "agent", agentKey, "source", path, "len", len(content))
		return
	}

	// No context file found — clear stale PROJECT.md with a placeholder.
	placeholder := fmt.Sprintf("# Project Context\n\nWorkspace: %s\nNo CLAUDE.md or README.md found at workspace root.", workspace)
	if err := w.AgentStore.SetAgentContextFile(ctx, agentID, "PROJECT.md", placeholder); err != nil {
		slog.Warn("workspace: failed to set placeholder PROJECT.md", "agent", agentKey, "error", err)
	}
}
