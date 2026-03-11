package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// ResolveAgentUUID looks up the agent UUID from an agent key string.
// If the key is already a valid UUID, it is returned directly.
// Otherwise, it looks up the agent by key in the store.
func ResolveAgentUUID(ctx context.Context, agentStore store.AgentStore, agentKey string) (uuid.UUID, error) {
	if agentKey == "" {
		return uuid.Nil, fmt.Errorf("no agent key configured")
	}

	// Try direct UUID parse first (future-proofing).
	if id, err := uuid.Parse(agentKey); err == nil {
		return id, nil
	}

	// Look up by agent key.
	agent, err := agentStore.GetByKey(ctx, agentKey)
	if err != nil {
		return uuid.Nil, fmt.Errorf("agent %q not found: %w", agentKey, err)
	}
	return agent.ID, nil
}
