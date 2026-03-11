package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// ListProjects returns a formatted list of sibling directories in the workspace parent.
func ListProjects(ctx context.Context, agentStore store.AgentStore, agentKey string) (string, error) {
	agentUUID, err := ResolveAgentUUID(ctx, agentStore, agentKey)
	if err != nil {
		return "", fmt.Errorf("could not resolve agent: %w", err)
	}

	agent, err := agentStore.GetByID(ctx, agentUUID)
	if err != nil {
		return "", fmt.Errorf("could not load agent config: %w", err)
	}

	ws := agent.Workspace
	if ws == "" {
		return "", fmt.Errorf("no workspace configured. Use /workspace <path> to set one first")
	}

	parentDir := filepath.Dir(filepath.Clean(ws))
	entries, err := os.ReadDir(parentDir)
	if err != nil {
		return "", fmt.Errorf("cannot read directory: %s", parentDir)
	}

	var dirs []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			dirs = append(dirs, e.Name())
		}
	}

	if len(dirs) == 0 {
		return fmt.Sprintf("No projects found in %s", parentDir), nil
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

	return sb.String(), nil
}
