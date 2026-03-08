package claudecode

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

func testContext(tenantID string) context.Context {
	ctx := context.Background()
	if tenantID != "" {
		ctx = store.WithTenantID(ctx, tenantID)
	}
	return ctx
}

func testManager() *SessionManager {
	cfg := DefaultBridgeConfig()
	cfg.Enabled = true
	cfg.Admission.MaxSessionsPerAgent = 10 // high limit for tests
	cfg.Admission.MaxTotalSessions = 10
	cfg.Admission.PerTenantSessionCap = 10
	return NewSessionManager(cfg, nil) // nil tmux = no real tmux calls
}

func testOpts(tenantID string) CreateSessionOpts {
	return CreateSessionOpts{
		AgentType:    AgentClaudeCode,
		ProjectPath:  "/tmp", // exists on all systems
		TenantID:     tenantID,
		UserID:       "user-1",
		OwnerActorID: "actor-1",
		Channel:      "telegram",
		ChatID:       "chat-123",
	}
}

func TestSessionManager_CreateSession(t *testing.T) {
	m := testManager()
	ctx := testContext("tenant-1")
	opts := testOpts("tenant-1")

	session, err := m.CreateSession(ctx, opts)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if !strings.HasPrefix(session.ID, "br:") {
		t.Errorf("session ID should start with br:, got %q", session.ID)
	}
	if session.Status != SessionStateActive {
		t.Errorf("initial status: got %s, want active", session.Status)
	}
	if session.RiskMode != RiskModeRead {
		t.Errorf("initial risk: got %s, want read", session.RiskMode)
	}
	if session.OwnerActorID != "actor-1" {
		t.Errorf("owner: got %q, want actor-1", session.OwnerActorID)
	}
}

func TestSessionManager_CreateSession_MissingFields(t *testing.T) {
	m := testManager()
	ctx := context.Background()

	tests := []struct {
		name string
		opts CreateSessionOpts
	}{
		{"no tenant", CreateSessionOpts{AgentType: AgentClaudeCode, ProjectPath: "/tmp", OwnerActorID: "a"}},
		{"no owner", CreateSessionOpts{AgentType: AgentClaudeCode, ProjectPath: "/tmp", TenantID: "t"}},
		{"no path", CreateSessionOpts{AgentType: AgentClaudeCode, TenantID: "t", OwnerActorID: "a"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := m.CreateSession(ctx, tt.opts)
			if err == nil {
				t.Error("expected error for missing field")
			}
		})
	}
}

func TestSessionManager_TenantIsolation_Get(t *testing.T) {
	m := testManager()

	// Create session for tenant-1
	ctx1 := testContext("tenant-1")
	session, _ := m.CreateSession(ctx1, testOpts("tenant-1"))

	// tenant-1 can see it
	got, err := m.GetSession(ctx1, session.ID)
	if err != nil {
		t.Fatalf("tenant-1 GetSession: %v", err)
	}
	if got.ID != session.ID {
		t.Error("tenant-1 should see own session")
	}

	// tenant-2 cannot see it
	ctx2 := testContext("tenant-2")
	_, err = m.GetSession(ctx2, session.ID)
	if err == nil {
		t.Error("tenant-2 should NOT see tenant-1's session")
	}
}

func TestSessionManager_TenantIsolation_List(t *testing.T) {
	m := testManager()

	ctx1 := testContext("tenant-1")
	ctx2 := testContext("tenant-2")

	m.CreateSession(ctx1, testOpts("tenant-1"))
	m.CreateSession(ctx1, testOpts("tenant-1"))

	opts2 := testOpts("tenant-2")
	m.CreateSession(ctx2, opts2)

	list1, _ := m.ListSessions(ctx1, "tenant-1")
	if len(list1) != 2 {
		t.Errorf("tenant-1 should see 2 sessions, got %d", len(list1))
	}

	list2, _ := m.ListSessions(ctx2, "tenant-2")
	if len(list2) != 1 {
		t.Errorf("tenant-2 should see 1 session, got %d", len(list2))
	}

	// Verify secrets are redacted in listings
	for _, s := range list1 {
		if s.HookSecret != "" {
			t.Error("HookSecret should be redacted in listings")
		}
	}
}

func TestSessionManager_KillSession_OwnerOnly(t *testing.T) {
	m := testManager()
	ctx := testContext("tenant-1")
	session, _ := m.CreateSession(ctx, testOpts("tenant-1"))

	// Non-owner cannot kill
	if err := m.KillSession(ctx, session.ID, "actor-other"); err == nil {
		t.Error("non-owner should not be able to kill session")
	}

	// Owner can kill
	if err := m.KillSession(ctx, session.ID, "actor-1"); err != nil {
		t.Fatalf("owner kill: %v", err)
	}
}

func TestSessionManager_AdmissionControl_MaxPerAgent(t *testing.T) {
	cfg := DefaultBridgeConfig()
	cfg.Admission.MaxSessionsPerAgent = 2
	m := NewSessionManager(cfg, nil)
	ctx := testContext("tenant-1")

	m.CreateSession(ctx, testOpts("tenant-1"))
	m.CreateSession(ctx, testOpts("tenant-1"))

	_, err := m.CreateSession(ctx, testOpts("tenant-1"))
	if err == nil {
		t.Error("should reject: max sessions per agent reached")
	}
	if !strings.Contains(err.Error(), "max sessions per agent") {
		t.Errorf("error should mention limit, got: %v", err)
	}
}

func TestSessionManager_AdmissionControl_MaxTotal(t *testing.T) {
	cfg := DefaultBridgeConfig()
	cfg.Admission.MaxTotalSessions = 3
	cfg.Admission.MaxSessionsPerAgent = 10 // don't hit this limit
	cfg.Admission.PerTenantSessionCap = 10
	m := NewSessionManager(cfg, nil)
	ctx := testContext("tenant-1")

	for i := 0; i < 3; i++ {
		_, err := m.CreateSession(ctx, testOpts("tenant-1"))
		if err != nil {
			t.Fatalf("session %d: %v", i, err)
		}
	}

	_, err := m.CreateSession(ctx, testOpts("tenant-1"))
	if err == nil {
		t.Error("should reject: max total sessions reached")
	}
}

func TestSessionManager_AdmissionControl_PerTenant(t *testing.T) {
	cfg := DefaultBridgeConfig()
	cfg.Admission.PerTenantSessionCap = 2
	cfg.Admission.MaxSessionsPerAgent = 10
	cfg.Admission.MaxTotalSessions = 10
	m := NewSessionManager(cfg, nil)

	ctx1 := testContext("tenant-1")
	m.CreateSession(ctx1, testOpts("tenant-1"))
	m.CreateSession(ctx1, testOpts("tenant-1"))

	_, err := m.CreateSession(ctx1, testOpts("tenant-1"))
	if err == nil {
		t.Error("should reject: per-tenant cap reached")
	}

	// Different tenant should still work
	ctx2 := testContext("tenant-2")
	_, err = m.CreateSession(ctx2, testOpts("tenant-2"))
	if err != nil {
		t.Errorf("different tenant should not be blocked: %v", err)
	}
}

func TestSessionManager_AdmissionControl_PerProjectSingleton(t *testing.T) {
	cfg := DefaultBridgeConfig()
	cfg.Admission.PerProjectSingleton = true
	cfg.Admission.MaxSessionsPerAgent = 10
	cfg.Admission.MaxTotalSessions = 10
	m := NewSessionManager(cfg, nil)
	ctx := testContext("tenant-1")

	m.CreateSession(ctx, testOpts("tenant-1"))

	_, err := m.CreateSession(ctx, testOpts("tenant-1"))
	if err == nil {
		t.Error("should reject: project singleton")
	}
	if !strings.Contains(err.Error(), "singleton") {
		t.Errorf("error should mention singleton, got: %v", err)
	}
}

func TestSessionManager_UpdateRiskMode_ProviderGuard(t *testing.T) {
	m := testManager()
	ctx := testContext("tenant-1")
	session, _ := m.CreateSession(ctx, testOpts("tenant-1"))

	// Claude Code supports permission hooks — interactive should work (owner check)
	err := m.UpdateRiskMode(ctx, session.ID, RiskModeInteractive, "actor-1")
	if err != nil {
		t.Errorf("Claude Code should support interactive: %v", err)
	}

	// Test with a stub adapter that doesn't support permission hooks
	m2 := testManager()
	opts := testOpts("tenant-1")
	opts.AgentType = AgentCursor // stub, PermissionHooks=false
	// Can't create Cursor session because stub InstallHooks fails... but the admission
	// doesn't call InstallHooks, so we can create the session object manually.
	// Instead, test by trying to create and escalate — Cursor doesn't support hooks.
	// For this test, create a Claude session and then swap agent type.
	s, _ := m2.CreateSession(ctx, testOpts("tenant-1"))

	// Manually change agent type to Cursor to test Layer 0 guard
	m2.mu.RLock()
	sess := m2.sessions[s.ID]
	m2.mu.RUnlock()
	sess.mu.Lock()
	sess.data.AgentType = AgentCursor
	sess.mu.Unlock()

	err = m2.UpdateRiskMode(ctx, s.ID, RiskModeInteractive, "actor-1")
	if err == nil {
		t.Error("Cursor (stub) should block interactive escalation")
	}
	if !strings.Contains(err.Error(), "permission hooks") {
		t.Errorf("error should mention permission hooks, got: %v", err)
	}
}

func TestSessionManager_TenantMismatch(t *testing.T) {
	m := testManager()
	ctx := testContext("tenant-1")
	opts := testOpts("tenant-2") // opts says tenant-2 but context says tenant-1

	_, err := m.CreateSession(ctx, opts)
	if err == nil {
		t.Error("should reject tenant ID mismatch between context and opts")
	}
}

func TestSessionManager_SendText_NoTmux(t *testing.T) {
	m := testManager() // nil tmux
	ctx := testContext("tenant-1")
	session, _ := m.CreateSession(ctx, testOpts("tenant-1"))

	// Default risk mode is read — InputMode=structured_only → free text blocked
	err := m.SendText(ctx, session.ID, "hello world", "actor-1")
	if err == nil {
		t.Error("should reject: InputMode=structured_only in read mode")
	}
	if !strings.Contains(err.Error(), "structured_only") {
		t.Errorf("error should mention structured_only, got: %v", err)
	}
}

func TestSessionManager_SendText_InteractiveMode_NoTmux(t *testing.T) {
	m := testManager()
	ctx := testContext("tenant-1")
	session, _ := m.CreateSession(ctx, testOpts("tenant-1"))

	// Escalate to interactive (free_text allowed)
	m.UpdateRiskMode(ctx, session.ID, RiskModeInteractive, "actor-1")

	// Should fail because tmux is nil, not because of capability gate
	err := m.SendText(ctx, session.ID, "hello world", "actor-1")
	if err == nil {
		t.Error("should fail: tmux not available")
	}
	if !strings.Contains(err.Error(), "tmux") {
		t.Errorf("error should mention tmux, got: %v", err)
	}
}

func TestSessionManager_SendText_StoppedSession(t *testing.T) {
	m := testManager()
	ctx := testContext("tenant-1")
	session, _ := m.CreateSession(ctx, testOpts("tenant-1"))

	// Escalate to interactive so capability gate passes
	m.UpdateRiskMode(ctx, session.ID, RiskModeInteractive, "actor-1")

	// Force stop
	m.mu.RLock()
	s := m.sessions[session.ID]
	m.mu.RUnlock()
	s.ForceStop()

	err := m.SendText(ctx, session.ID, "hello", "actor-1")
	if err == nil {
		t.Error("should reject: session stopped")
	}
	if !strings.Contains(err.Error(), "stopped") {
		t.Errorf("error should mention stopped, got: %v", err)
	}
}

func TestSessionManager_SendText_TenantIsolation(t *testing.T) {
	m := testManager()
	ctx1 := testContext("tenant-1")
	session, _ := m.CreateSession(ctx1, testOpts("tenant-1"))

	// tenant-2 cannot send to tenant-1's session
	ctx2 := testContext("tenant-2")
	err := m.SendText(ctx2, session.ID, "hello", "actor-1")
	if err == nil {
		t.Error("tenant-2 should not access tenant-1 session")
	}
}

func TestSessionManager_SendText_UnsafeInput(t *testing.T) {
	m := testManager()
	ctx := testContext("tenant-1")
	session, _ := m.CreateSession(ctx, testOpts("tenant-1"))

	// Escalate to interactive
	m.UpdateRiskMode(ctx, session.ID, RiskModeInteractive, "actor-1")

	// Dangerous input should be blocked by sanitizer
	err := m.SendText(ctx, session.ID, "rm -rf /", "actor-1")
	if err == nil {
		t.Error("should reject dangerous input")
	}
}

func TestSessionManager_SendText_BusyQueue(t *testing.T) {
	m := testManager()
	ctx := testContext("tenant-1")
	session, _ := m.CreateSession(ctx, testOpts("tenant-1"))

	// Escalate to interactive
	m.UpdateRiskMode(ctx, session.ID, RiskModeInteractive, "actor-1")

	// Set session to BUSY
	m.mu.RLock()
	s := m.sessions[session.ID]
	m.mu.RUnlock()
	s.TransitionTo(SessionStateBusy)

	// Should queue, not reject
	err := m.SendText(ctx, session.ID, "hello", "actor-1")
	if err != nil {
		t.Fatalf("busy session should queue message, not reject: %v", err)
	}
	if s.QueueLen() != 1 {
		t.Errorf("queue length: got %d, want 1", s.QueueLen())
	}
}

func TestSessionManager_CaptureOutput_NoTmux(t *testing.T) {
	m := testManager()
	ctx := testContext("tenant-1")
	session, _ := m.CreateSession(ctx, testOpts("tenant-1"))

	_, err := m.CaptureOutput(ctx, session.ID, "actor-1", 10)
	if err == nil {
		t.Error("should fail: tmux not available")
	}
	if !strings.Contains(err.Error(), "tmux") {
		t.Errorf("error should mention tmux, got: %v", err)
	}
}

func TestSessionManager_CaptureOutput_TenantIsolation(t *testing.T) {
	m := testManager()
	ctx1 := testContext("tenant-1")
	session, _ := m.CreateSession(ctx1, testOpts("tenant-1"))

	ctx2 := testContext("tenant-2")
	_, err := m.CaptureOutput(ctx2, session.ID, "actor-1", 10)
	if err == nil {
		t.Error("tenant-2 should not capture tenant-1 session output")
	}
}

func TestSessionManager_CaptureOutput_StoppedSession(t *testing.T) {
	m := testManager()
	ctx := testContext("tenant-1")
	session, _ := m.CreateSession(ctx, testOpts("tenant-1"))

	m.mu.RLock()
	s := m.sessions[session.ID]
	m.mu.RUnlock()
	s.ForceStop()

	_, err := m.CaptureOutput(ctx, session.ID, "actor-1", 10)
	if err == nil {
		t.Error("should reject: session stopped")
	}
}

func TestSessionManager_CleanupStopped(t *testing.T) {
	m := testManager()
	ctx := testContext("tenant-1")

	// Create 3 sessions, stop 2
	s1, _ := m.CreateSession(ctx, testOpts("tenant-1"))
	s2, _ := m.CreateSession(ctx, testOpts("tenant-1"))
	m.CreateSession(ctx, testOpts("tenant-1")) // s3 stays active

	m.mu.RLock()
	sess1 := m.sessions[s1.ID]
	sess2 := m.sessions[s2.ID]
	m.mu.RUnlock()

	sess1.ForceStop()
	sess2.ForceStop()

	// Manually backdate stopped sessions so they're older than maxAge
	sess1.mu.Lock()
	sess1.data.LastActivityAt = sess1.data.LastActivityAt.Add(-25 * time.Hour)
	sess1.mu.Unlock()
	sess2.mu.Lock()
	sess2.data.LastActivityAt = sess2.data.LastActivityAt.Add(-25 * time.Hour)
	sess2.mu.Unlock()

	// Cleanup with 24h maxAge
	removed := m.CleanupStopped(24 * time.Hour)
	if removed != 2 {
		t.Errorf("cleanup: removed %d, want 2", removed)
	}
	if m.SessionCount() != 1 {
		t.Errorf("remaining sessions: %d, want 1", m.SessionCount())
	}
}

func TestSessionManager_TransitionSession_DrainQueue(t *testing.T) {
	m := testManager()
	ctx := testContext("tenant-1")
	session, _ := m.CreateSession(ctx, testOpts("tenant-1"))

	// Escalate to interactive so messages can be queued
	m.UpdateRiskMode(ctx, session.ID, RiskModeInteractive, "actor-1")

	// Transition to busy and queue messages
	m.mu.RLock()
	s := m.sessions[session.ID]
	m.mu.RUnlock()
	s.TransitionTo(SessionStateBusy)

	s.EnqueueMessage("msg-1")
	s.EnqueueMessage("msg-2")
	if s.QueueLen() != 2 {
		t.Fatalf("queue should have 2 messages, got %d", s.QueueLen())
	}

	// Transition back to active via TransitionSession — should drain queue
	// (tmux is nil, so drain will log warnings but queue will be cleared)
	err := m.TransitionSession(ctx, session.ID, SessionStateActive)
	if err != nil {
		t.Fatalf("TransitionSession: %v", err)
	}

	// Queue should be drained (even though tmux send fails)
	if s.QueueLen() != 0 {
		t.Errorf("queue should be empty after drain, got %d", s.QueueLen())
	}
}

func TestSessionManager_TransitionSession_NoDrainForNonBusy(t *testing.T) {
	m := testManager()
	ctx := testContext("tenant-1")
	session, _ := m.CreateSession(ctx, testOpts("tenant-1"))

	// Manually enqueue (shouldn't happen in practice, but tests drain logic)
	m.mu.RLock()
	s := m.sessions[session.ID]
	m.mu.RUnlock()
	s.EnqueueMessage("msg-1")

	// Transition active→idle (not from busy) — should NOT drain
	err := m.TransitionSession(ctx, session.ID, SessionStateIdle)
	if err != nil {
		t.Fatalf("TransitionSession: %v", err)
	}
	if s.QueueLen() != 1 {
		t.Errorf("queue should still have 1 message (no drain from non-busy), got %d", s.QueueLen())
	}
}

func TestSessionManager_CleanupStopped_RecentNotRemoved(t *testing.T) {
	m := testManager()
	ctx := testContext("tenant-1")

	s, _ := m.CreateSession(ctx, testOpts("tenant-1"))
	m.mu.RLock()
	sess := m.sessions[s.ID]
	m.mu.RUnlock()
	sess.ForceStop()

	// Just stopped — should not be cleaned up with 24h maxAge
	removed := m.CleanupStopped(24 * time.Hour)
	if removed != 0 {
		t.Errorf("recently stopped sessions should not be removed, got %d", removed)
	}
}

// --- Sprint 18: Persona / SOUL integration tests ---

func TestSessionManager_CreateSession_BarePersona(t *testing.T) {
	m := testManager()
	ctx := testContext("tenant-1")
	opts := testOpts("tenant-1")
	// No AgentRole = bare launch

	session, err := m.CreateSession(ctx, opts)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if session.PersonaSource != "bare" {
		t.Errorf("PersonaSource: got %q, want bare", session.PersonaSource)
	}
	if session.AgentRole != "" {
		t.Errorf("AgentRole: got %q, want empty", session.AgentRole)
	}
}

func TestSessionManager_CreateSession_WithRole_StrategyB(t *testing.T) {
	InvalidateRolesCache()

	// Use a temp standalone dir so Strategy B writes to temp
	cfg := DefaultBridgeConfig()
	cfg.Enabled = true
	cfg.SoulsDir = testSoulsDir
	cfg.StandaloneDir = t.TempDir()
	cfg.Admission.MaxSessionsPerAgent = 10
	cfg.Admission.MaxTotalSessions = 10
	cfg.Admission.PerTenantSessionCap = 10
	m := NewSessionManager(cfg, nil)

	ctx := testContext("tenant-1")
	opts := testOpts("tenant-1")
	opts.AgentRole = "coder"
	// ProjectPath = /tmp which has no .claude/agents/coder.md → Strategy B

	session, err := m.CreateSession(ctx, opts)
	if err != nil {
		t.Fatalf("CreateSession with --as coder: %v", err)
	}
	if session.AgentRole != "coder" {
		t.Errorf("AgentRole: got %q, want coder", session.AgentRole)
	}
	if session.PersonaSource != "append_prompt" {
		t.Errorf("PersonaSource: got %q, want append_prompt", session.PersonaSource)
	}
	if session.SoulTemplateHash == "" {
		t.Error("SoulTemplateHash should be populated")
	}
	if session.PersonaSourceHash == "" {
		t.Error("PersonaSourceHash should be populated")
	}
}

func TestSessionManager_CreateSession_WithRole_StrategyA(t *testing.T) {
	InvalidateRolesCache()

	// Create project dir with .claude/agents/coder.md
	projectDir := t.TempDir()
	agentsDir := filepath.Join(projectDir, ".claude", "agents")
	os.MkdirAll(agentsDir, 0755)
	os.WriteFile(filepath.Join(agentsDir, "coder.md"), []byte("# Test agent"), 0644)

	cfg := DefaultBridgeConfig()
	cfg.Enabled = true
	cfg.SoulsDir = testSoulsDir
	cfg.StandaloneDir = t.TempDir()
	cfg.Admission.MaxSessionsPerAgent = 10
	cfg.Admission.MaxTotalSessions = 10
	cfg.Admission.PerTenantSessionCap = 10
	m := NewSessionManager(cfg, nil)

	ctx := testContext("tenant-1")
	opts := testOpts("tenant-1")
	opts.ProjectPath = projectDir
	opts.AgentRole = "coder"

	session, err := m.CreateSession(ctx, opts)
	if err != nil {
		t.Fatalf("CreateSession Strategy A: %v", err)
	}
	if session.PersonaSource != "agent_file" {
		t.Errorf("PersonaSource: got %q, want agent_file", session.PersonaSource)
	}
	if session.AgentRole != "coder" {
		t.Errorf("AgentRole: got %q, want coder", session.AgentRole)
	}
}

func TestSessionManager_CreateSession_InvalidRole(t *testing.T) {
	InvalidateRolesCache()

	cfg := DefaultBridgeConfig()
	cfg.Enabled = true
	cfg.SoulsDir = testSoulsDir
	cfg.Admission.MaxSessionsPerAgent = 10
	cfg.Admission.MaxTotalSessions = 10
	cfg.Admission.PerTenantSessionCap = 10
	m := NewSessionManager(cfg, nil)

	ctx := testContext("tenant-1")
	opts := testOpts("tenant-1")
	opts.AgentRole = "nonexistent-role"

	_, err := m.CreateSession(ctx, opts)
	if err == nil {
		t.Error("should reject unknown role")
	}
	if !strings.Contains(err.Error(), "unknown role") {
		t.Errorf("error should mention unknown role, got: %v", err)
	}
}

func TestSessionManager_CreateSession_StaleAgentFile(t *testing.T) {
	InvalidateRolesCache()

	// Create project with agent file + stale .soul-hash sidecar (CTO-107)
	projectDir := t.TempDir()
	agentsDir := filepath.Join(projectDir, ".claude", "agents")
	os.MkdirAll(agentsDir, 0755)
	os.WriteFile(filepath.Join(agentsDir, "coder.md"),
		[]byte("# Generated by mtclaw bridge install-agents (claude-code >= 2.x) — do not edit manually\n---\nname: coder\n---\nOLD CONTENT"), 0644)
	// Stale hash — doesn't match current SOUL file
	os.WriteFile(filepath.Join(agentsDir, "coder.soul-hash"),
		[]byte("0000000000000000000000000000000000000000000000000000000000000000"), 0644)

	cfg := DefaultBridgeConfig()
	cfg.Enabled = true
	cfg.SoulsDir = testSoulsDir
	cfg.StandaloneDir = t.TempDir()
	cfg.Admission.MaxSessionsPerAgent = 10
	cfg.Admission.MaxTotalSessions = 10
	cfg.Admission.PerTenantSessionCap = 10
	m := NewSessionManager(cfg, nil)

	ctx := testContext("tenant-1")
	opts := testOpts("tenant-1")
	opts.ProjectPath = projectDir
	opts.AgentRole = "coder"

	// Should succeed — stale = warning, not block (CTO-M4)
	session, err := m.CreateSession(ctx, opts)
	if err != nil {
		t.Fatalf("CreateSession with stale agent file should succeed: %v", err)
	}
	if session.PersonaSource != "agent_file" {
		t.Errorf("PersonaSource: got %q, want agent_file", session.PersonaSource)
	}
}

func TestSessionManager_KillSession_CleansStrategyBFiles(t *testing.T) {
	InvalidateRolesCache()

	standaloneDir := t.TempDir()
	cfg := DefaultBridgeConfig()
	cfg.Enabled = true
	cfg.SoulsDir = testSoulsDir
	cfg.StandaloneDir = standaloneDir
	cfg.Admission.MaxSessionsPerAgent = 10
	cfg.Admission.MaxTotalSessions = 10
	cfg.Admission.PerTenantSessionCap = 10
	m := NewSessionManager(cfg, nil)

	ctx := testContext("tenant-1")
	opts := testOpts("tenant-1")
	opts.AgentRole = "coder"

	session, err := m.CreateSession(ctx, opts)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if session.PersonaSource != "append_prompt" {
		t.Fatalf("expected Strategy B, got %s", session.PersonaSource)
	}

	// Verify temp file exists
	dirName := strings.ReplaceAll(session.ID, ":", "-")
	promptFile := filepath.Join(standaloneDir, "sessions", dirName, "soul.md")
	if _, err := os.Stat(promptFile); os.IsNotExist(err) {
		t.Fatal("Strategy B prompt file should exist before kill")
	}

	// Kill session
	err = m.KillSession(ctx, session.ID, "actor-1")
	if err != nil {
		t.Fatalf("KillSession: %v", err)
	}

	// Verify cleanup
	if _, err := os.Stat(promptFile); !os.IsNotExist(err) {
		t.Error("Strategy B prompt file should be cleaned up after kill")
	}
}
