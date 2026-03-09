package claudecode

import (
	"context"
	"fmt"
	"testing"
)

func TestBridgeMetrics_Empty(t *testing.T) {
	sm := NewSessionManager(DefaultBridgeConfig(), nil)
	m := sm.Metrics()

	if m.ActiveSessions != 0 {
		t.Errorf("expected 0 active, got %d", m.ActiveSessions)
	}
	if m.TotalCreated != 0 {
		t.Errorf("expected 0 created, got %d", m.TotalCreated)
	}
	if m.TotalKilled != 0 {
		t.Errorf("expected 0 killed, got %d", m.TotalKilled)
	}
	if len(m.ByRiskMode) != 0 {
		t.Errorf("expected empty ByRiskMode, got %v", m.ByRiskMode)
	}
	if len(m.ByAgentRole) != 0 {
		t.Errorf("expected empty ByAgentRole, got %v", m.ByAgentRole)
	}
	if len(m.ByChannel) != 0 {
		t.Errorf("expected empty ByChannel, got %v", m.ByChannel)
	}
}

func TestBridgeMetrics_ActiveCount(t *testing.T) {
	cfg := DefaultBridgeConfig()
	cfg.Admission.MaxSessionsPerAgent = 5
	cfg.Admission.MaxTotalSessions = 10
	sm := NewSessionManager(cfg, nil)
	ctx := context.Background()

	// Create 3 sessions with different project paths (avoid per-project singleton)
	for i := 0; i < 3; i++ {
		_, err := sm.CreateSession(ctx, CreateSessionOpts{
			AgentType:    AgentClaudeCode,
			ProjectPath:  fmt.Sprintf("/tmp/test-%d", i),
			TenantID:     "t1",
			UserID:       "u1",
			OwnerActorID: "actor1",
			Channel:      "telegram",
			ChatID:       "chat1",
		})
		if err != nil {
			t.Fatalf("create session %d: %v", i, err)
		}
	}

	m := sm.Metrics()
	if m.ActiveSessions != 3 {
		t.Errorf("expected 3 active, got %d", m.ActiveSessions)
	}
	if m.TotalCreated != 3 {
		t.Errorf("expected 3 created, got %d", m.TotalCreated)
	}
}

func TestBridgeMetrics_ByRiskMode(t *testing.T) {
	sm := NewSessionManager(DefaultBridgeConfig(), nil)
	ctx := context.Background()

	// Create 2 sessions (default risk = read)
	for i := 0; i < 2; i++ {
		_, err := sm.CreateSession(ctx, CreateSessionOpts{
			AgentType:    AgentClaudeCode,
			ProjectPath:  "/tmp/test",
			TenantID:     "t1",
			UserID:       "u1",
			OwnerActorID: "actor1",
			Channel:      "telegram",
			ChatID:       "chat1",
		})
		if err != nil {
			t.Fatalf("create session: %v", err)
		}
	}

	m := sm.Metrics()
	if m.ByRiskMode["read"] != 2 {
		t.Errorf("expected 2 read-mode sessions, got %d", m.ByRiskMode["read"])
	}
}

func TestBridgeMetrics_ByRole(t *testing.T) {
	sm := NewSessionManager(DefaultBridgeConfig(), nil)
	ctx := context.Background()

	// Create 2 bare sessions (no role — avoids SOUL loading)
	for i := 0; i < 2; i++ {
		_, err := sm.CreateSession(ctx, CreateSessionOpts{
			AgentType:    AgentClaudeCode,
			ProjectPath:  fmt.Sprintf("/tmp/test-%d", i),
			TenantID:     "t1",
			UserID:       "u1",
			OwnerActorID: "actor1",
			Channel:      "telegram",
			ChatID:       "chat1",
		})
		if err != nil {
			t.Fatalf("create session: %v", err)
		}
	}

	m := sm.Metrics()
	if m.ByAgentRole["(bare)"] != 2 {
		t.Errorf("expected 2 bare sessions, got %d", m.ByAgentRole["(bare)"])
	}
	if m.ByChannel["telegram"] != 2 {
		t.Errorf("expected 2 telegram sessions, got %d", m.ByChannel["telegram"])
	}
}

func TestBridgeMetrics_Lifetime(t *testing.T) {
	sm := NewSessionManager(DefaultBridgeConfig(), nil)
	ctx := context.Background()

	// Create session
	bs, err := sm.CreateSession(ctx, CreateSessionOpts{
		AgentType:    AgentClaudeCode,
		ProjectPath:  "/tmp/test",
		TenantID:     "t1",
		UserID:       "u1",
		OwnerActorID: "actor1",
		Channel:      "telegram",
		ChatID:       "chat1",
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	// Kill it
	err = sm.KillSession(ctx, bs.ID, "actor1")
	if err != nil {
		t.Fatalf("kill session: %v", err)
	}

	m := sm.Metrics()
	if m.TotalCreated != 1 {
		t.Errorf("expected 1 created, got %d", m.TotalCreated)
	}
	if m.TotalKilled != 1 {
		t.Errorf("expected 1 killed, got %d", m.TotalKilled)
	}
	// Session is still in map (stopped) — active count should be 0
	if m.ActiveSessions != 0 {
		t.Errorf("expected 0 active after kill, got %d", m.ActiveSessions)
	}
}
