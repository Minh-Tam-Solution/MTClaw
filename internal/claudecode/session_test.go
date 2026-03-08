package claudecode

import (
	"testing"
	"time"
)

func newTestSession() *Session {
	return NewSession(BridgeSession{
		ID:           "br:test1234:abcd5678",
		Status:       SessionStateActive,
		RiskMode:     RiskModeRead,
		Capabilities: CapabilitiesForRisk(RiskModeRead),
		OwnerActorID: "actor-owner",
		ApproverACL:  []string{"actor-approver"},
		NotifyACL:    []string{"actor-owner", "actor-notify"},
		TenantID:     "tenant-1",
	})
}

func TestSession_ValidTransitions(t *testing.T) {
	tests := []struct {
		name string
		from SessionState
		to   SessionState
		ok   bool
	}{
		{"active->busy", SessionStateActive, SessionStateBusy, true},
		{"active->idle", SessionStateActive, SessionStateIdle, true},
		{"active->stopped", SessionStateActive, SessionStateStopped, true},
		{"active->error", SessionStateActive, SessionStateError, true},
		{"busy->idle", SessionStateBusy, SessionStateIdle, true},
		{"busy->stopped", SessionStateBusy, SessionStateStopped, true},
		{"busy->error", SessionStateBusy, SessionStateError, true},
		{"idle->busy", SessionStateIdle, SessionStateBusy, true},
		{"idle->stopped", SessionStateIdle, SessionStateStopped, true},
		{"error->stopped", SessionStateError, SessionStateStopped, true},
		// Sprint 17: sessions can resume from idle/busy/error to active
		{"idle->active", SessionStateIdle, SessionStateActive, true},
		{"error->active", SessionStateError, SessionStateActive, true},
		{"busy->active", SessionStateBusy, SessionStateActive, true},
		// Invalid: stopped is terminal
		{"stopped->active", SessionStateStopped, SessionStateActive, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSession(BridgeSession{Status: tt.from})
			err := s.TransitionTo(tt.to)
			if (err == nil) != tt.ok {
				t.Errorf("TransitionTo(%s->%s): got err=%v, wantOK=%v", tt.from, tt.to, err, tt.ok)
			}
		})
	}
}

func TestSession_SameStateNoop(t *testing.T) {
	s := newTestSession()
	if err := s.TransitionTo(SessionStateActive); err != nil {
		t.Errorf("same state transition should be no-op: %v", err)
	}
}

func TestSession_ForceStop(t *testing.T) {
	s := newTestSession()
	s.TransitionTo(SessionStateBusy)
	s.ForceStop()
	data := s.Data()
	if data.Status != SessionStateStopped {
		t.Errorf("ForceStop: got %s, want stopped", data.Status)
	}
}

func TestSession_UpdateRiskMode_OwnerEscalate(t *testing.T) {
	s := newTestSession()
	if err := s.UpdateRiskMode(RiskModePatch, "actor-owner"); err != nil {
		t.Errorf("owner escalate to patch: %v", err)
	}
	data := s.Data()
	if data.RiskMode != RiskModePatch {
		t.Errorf("risk mode: got %s, want patch", data.RiskMode)
	}
	if data.Capabilities.ToolPolicy != ToolPolicyPatchAllowed {
		t.Errorf("tool policy: got %s, want patch_allowed", data.Capabilities.ToolPolicy)
	}
}

func TestSession_UpdateRiskMode_NonOwnerBlocked(t *testing.T) {
	s := newTestSession()
	if err := s.UpdateRiskMode(RiskModePatch, "actor-other"); err == nil {
		t.Error("non-owner should not escalate to patch")
	}
}

func TestSession_UpdateRiskMode_DowngradeAlwaysAllowed(t *testing.T) {
	s := newTestSession()
	s.UpdateRiskMode(RiskModePatch, "actor-owner")
	// Any actor can downgrade to read
	if err := s.UpdateRiskMode(RiskModeRead, "actor-random"); err != nil {
		t.Errorf("downgrade to read should be allowed by anyone: %v", err)
	}
}

func TestSession_MessageQueue(t *testing.T) {
	s := newTestSession()

	// Enqueue messages
	for i := 0; i < 5; i++ {
		if err := s.EnqueueMessage("msg"); err != nil {
			t.Fatalf("enqueue %d: %v", i, err)
		}
	}
	if s.QueueLen() != 5 {
		t.Errorf("queue len: got %d, want 5", s.QueueLen())
	}

	// Drain
	msgs := s.DrainQueue()
	if len(msgs) != 5 {
		t.Errorf("drained: got %d, want 5", len(msgs))
	}
	if s.QueueLen() != 0 {
		t.Error("queue should be empty after drain")
	}

	// Drain empty is safe
	empty := s.DrainQueue()
	if empty != nil {
		t.Errorf("drain empty: got %v, want nil", empty)
	}
}

func TestSession_MessageQueue_Full(t *testing.T) {
	s := newTestSession()
	for i := 0; i < defaultMaxQueueSize; i++ {
		s.EnqueueMessage("msg")
	}
	if err := s.EnqueueMessage("overflow"); err == nil {
		t.Error("should reject when queue is full")
	}
}

func TestSession_IsOwner(t *testing.T) {
	s := newTestSession()
	if !s.IsOwner("actor-owner") {
		t.Error("should be owner")
	}
	if s.IsOwner("actor-other") {
		t.Error("should not be owner")
	}
}

func TestSession_IsApprover(t *testing.T) {
	s := newTestSession()
	if !s.IsApprover("actor-owner") {
		t.Error("owner should always be approver")
	}
	if !s.IsApprover("actor-approver") {
		t.Error("ACL member should be approver")
	}
	if s.IsApprover("actor-random") {
		t.Error("non-ACL member should not be approver")
	}
}

func TestSession_CanReceiveNotification(t *testing.T) {
	s := newTestSession()
	if !s.CanReceiveNotification("actor-owner") {
		t.Error("owner in notify ACL should receive")
	}
	if !s.CanReceiveNotification("actor-notify") {
		t.Error("notify ACL member should receive")
	}
	if s.CanReceiveNotification("actor-random") {
		t.Error("non-ACL member should not receive")
	}
}

func TestSession_Touch(t *testing.T) {
	s := newTestSession()
	before := s.Data().LastActivityAt
	time.Sleep(time.Millisecond)
	s.Touch()
	after := s.Data().LastActivityAt
	if !after.After(before) {
		t.Error("Touch should update LastActivityAt")
	}
}

func TestSession_TurnContext_SetAndConsume(t *testing.T) {
	s := newTestSession()

	// No context initially
	if tc := s.ConsumeTurnContext(); tc != nil {
		t.Error("expected nil turn context initially")
	}

	// Set context
	tc := &TurnContext{
		SprintGoals: []string{"Fix login bug"},
		Blockers:    []string{"API rate limit"},
	}
	s.SetTurnContext(tc)

	// Consume returns it
	got := s.ConsumeTurnContext()
	if got == nil {
		t.Fatal("expected non-nil turn context")
	}
	if len(got.SprintGoals) != 1 || got.SprintGoals[0] != "Fix login bug" {
		t.Errorf("unexpected goals: %v", got.SprintGoals)
	}

	// Second consume returns nil (consumed once)
	if tc2 := s.ConsumeTurnContext(); tc2 != nil {
		t.Error("turn context should be consumed once")
	}
}

func TestSession_TurnContext_Accumulate(t *testing.T) {
	s := newTestSession()

	s.SetTurnContext(&TurnContext{SprintGoals: []string{"First goal"}})
	s.SetTurnContext(&TurnContext{SprintGoals: []string{"Second goal"}})
	s.SetTurnContext(&TurnContext{Blockers: []string{"A blocker"}})

	got := s.ConsumeTurnContext()
	if got == nil {
		t.Fatal("expected non-nil context")
	}
	if len(got.SprintGoals) != 2 {
		t.Errorf("expected 2 goals, got %d: %v", len(got.SprintGoals), got.SprintGoals)
	}
	if len(got.Blockers) != 1 || got.Blockers[0] != "A blocker" {
		t.Errorf("expected 1 blocker, got %v", got.Blockers)
	}
}

func TestSession_TurnContext_Clear(t *testing.T) {
	s := newTestSession()

	s.SetTurnContext(&TurnContext{SprintGoals: []string{"Goal"}})
	s.ClearTurnContext()

	if tc := s.ConsumeTurnContext(); tc != nil {
		t.Error("ClearTurnContext should remove all context")
	}
}
