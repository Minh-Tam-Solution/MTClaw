package claudecode

import (
	"context"
	"testing"
	"time"
)

func TestHealthMonitor_InitialCheck(t *testing.T) {
	mgr := testManager()
	// nil tmux — simulates tmux not configured
	mon := NewHealthMonitor(mgr, nil, 1*time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	go mon.Start(ctx)
	// Give initial check time to run
	time.Sleep(50 * time.Millisecond)
	cancel()

	status := mon.LastStatus()
	if status.CheckedAt.IsZero() {
		t.Error("expected initial check to run")
	}
	if status.Checks["tmux"] != "not configured" {
		t.Errorf("tmux check: got %q, want %q", status.Checks["tmux"], "not configured")
	}
}

func TestHealthMonitor_NoDeadSessions(t *testing.T) {
	mgr := testManager()
	mon := NewHealthMonitor(mgr, nil, 1*time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	go mon.Start(ctx)
	time.Sleep(50 * time.Millisecond)
	cancel()

	status := mon.LastStatus()
	if len(status.DeadSessions) != 0 {
		t.Errorf("expected no dead sessions, got %v", status.DeadSessions)
	}
	if status.ActiveCount != 0 {
		t.Errorf("expected 0 active, got %d", status.ActiveCount)
	}
}

func TestHealthMonitor_DetectsActiveSessions(t *testing.T) {
	mgr := testManager()
	ctx := testContext("tenant-1")
	_, err := mgr.CreateSession(ctx, testOpts("tenant-1"))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	mon := NewHealthMonitor(mgr, nil, 1*time.Hour)

	monCtx, cancel := context.WithCancel(context.Background())
	go mon.Start(monCtx)
	time.Sleep(50 * time.Millisecond)
	cancel()

	status := mon.LastStatus()
	if status.ActiveCount != 1 {
		t.Errorf("expected 1 active session, got %d", status.ActiveCount)
	}
}

func TestHealthMonitor_DefaultInterval(t *testing.T) {
	mon := NewHealthMonitor(testManager(), nil, 0)
	if mon.interval != 30*time.Second {
		t.Errorf("default interval: got %v, want 30s", mon.interval)
	}
}

func TestHealthMonitor_LastStatus_BeforeStart(t *testing.T) {
	mon := NewHealthMonitor(testManager(), nil, 1*time.Hour)
	status := mon.LastStatus()
	if !status.CheckedAt.IsZero() {
		t.Error("expected zero CheckedAt before Start()")
	}
}
