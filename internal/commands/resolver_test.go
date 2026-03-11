package commands

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// mockAgentStore implements the subset of store.AgentStore needed for resolver tests.
type mockAgentStore struct {
	store.AgentStore
	agents map[string]*store.AgentData
}

func (m *mockAgentStore) GetByKey(ctx context.Context, agentKey string) (*store.AgentData, error) {
	if agent, ok := m.agents[agentKey]; ok {
		return agent, nil
	}
	return nil, fmt.Errorf("agent %q not found", agentKey)
}

// CTO C5: 3 unit tests for ResolveAgentUUID.

func TestResolveAgentUUID_ParsesUUID(t *testing.T) {
	id := uuid.New()
	result, err := ResolveAgentUUID(context.Background(), nil, id.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != id {
		t.Errorf("expected %s, got %s", id, result)
	}
}

func TestResolveAgentUUID_FallbackToStore(t *testing.T) {
	expectedID := uuid.New()
	mockStore := &mockAgentStore{
		agents: map[string]*store.AgentData{
			"pm": {
				BaseModel: store.BaseModel{ID: expectedID},
				AgentKey:  "pm",
			},
		},
	}

	result, err := ResolveAgentUUID(context.Background(), mockStore, "pm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expectedID {
		t.Errorf("expected %s, got %s", expectedID, result)
	}
}

func TestResolveAgentUUID_EmptyKey(t *testing.T) {
	_, err := ResolveAgentUUID(context.Background(), nil, "")
	if err == nil {
		t.Fatal("expected error for empty key")
	}
}
