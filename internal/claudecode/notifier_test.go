package claudecode

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestNotifier_StopMessage_Success(t *testing.T) {
	var received string
	n := NewNotifier(func(_ context.Context, _, _, msg string) error {
		received = msg
		return nil
	})

	session := &BridgeSession{
		ID:           "br:abc12345:def67890",
		ProjectPath:  "/home/user/project",
		RiskMode:     RiskModeRead,
		Capabilities: CapabilitiesForRisk(RiskModeRead),
		NotifyACL:    []string{"owner-123"},
		Channel:      "telegram",
		ChatID:       "12345",
	}

	event := StopEvent{
		SessionID:  session.ID,
		ExitCode:   0,
		Summary:    "Implemented feature X",
		GitDiff:    "+func hello() {\n+    fmt.Println(\"hello\")\n+}",
		FinishedAt: time.Now(),
	}

	n.NotifyStop(context.Background(), session, event)

	if received == "" {
		t.Fatal("notification should have been delivered")
	}
	if len(received) < 20 {
		t.Errorf("notification too short: %q", received)
	}
}

func TestNotifier_StopMessage_Redaction(t *testing.T) {
	var received string
	n := NewNotifier(func(_ context.Context, _, _, msg string) error {
		received = msg
		return nil
	})

	session := &BridgeSession{
		ID:           "br:abc12345:def67890",
		ProjectPath:  "/home/user/project",
		RiskMode:     RiskModeRead,
		Capabilities: CapabilitiesForRisk(RiskModeRead),
		NotifyACL:    []string{"owner"},
		Channel:      "telegram",
		ChatID:       "12345",
	}

	event := StopEvent{
		SessionID:  session.ID,
		Summary:    "Set API_KEY=sk-1234567890abcdef for testing",
		FinishedAt: time.Now(),
	}

	n.NotifyStop(context.Background(), session, event)

	if received == "" {
		t.Fatal("notification should have been delivered")
	}
	// The API key pattern should be redacted
	if contains := "sk-1234567890abcdef"; len(received) > 0 {
		for i := 0; i <= len(received)-len(contains); i++ {
			if received[i:i+len(contains)] == contains {
				t.Error("API key should be redacted in notification")
			}
		}
	}
}

func TestCircuitBreaker_TripsAfterThreshold(t *testing.T) {
	callCount := 0
	n := NewNotifier(func(_ context.Context, _, _, _ string) error {
		callCount++
		return fmt.Errorf("delivery failed")
	})

	session := &BridgeSession{
		ID:        "br:abc12345:def67890",
		NotifyACL: []string{"owner"},
		Channel:   "telegram",
		ChatID:    "12345",
	}
	event := StopEvent{FinishedAt: time.Now()}

	// Trigger 3 failures (threshold)
	for i := 0; i < 3; i++ {
		n.NotifyStop(context.Background(), session, event)
	}

	if n.BreakerState() != "open" {
		t.Errorf("breaker should be open after %d failures, got %s", defaultBreakerThreshold, n.BreakerState())
	}

	// 4th call should be blocked by breaker
	prevCount := callCount
	n.NotifyStop(context.Background(), session, event)
	if callCount != prevCount {
		t.Error("breaker should block calls when open")
	}
}

func TestCircuitBreaker_ResetsOnSuccess(t *testing.T) {
	callNum := 0
	n := NewNotifier(func(_ context.Context, _, _, _ string) error {
		callNum++
		if callNum <= 2 {
			return fmt.Errorf("fail")
		}
		return nil // 3rd call succeeds
	})

	session := &BridgeSession{
		ID:        "br:test:session",
		NotifyACL: []string{"owner"},
		Channel:   "telegram",
		ChatID:    "12345",
	}
	event := StopEvent{FinishedAt: time.Now()}

	// 2 failures (below threshold)
	n.NotifyStop(context.Background(), session, event)
	n.NotifyStop(context.Background(), session, event)

	if n.BreakerState() != "closed" {
		t.Errorf("breaker should still be closed after 2 failures (threshold=3), got %s", n.BreakerState())
	}

	// Success resets
	n.NotifyStop(context.Background(), session, event)
	if n.BreakerState() != "closed" {
		t.Errorf("breaker should be closed after success, got %s", n.BreakerState())
	}
}

func TestNotifier_NilSendFn(t *testing.T) {
	n := NewNotifier(nil)
	session := &BridgeSession{
		ID:        "br:test:session",
		NotifyACL: []string{"owner"},
	}
	// Should not panic
	n.NotifyStop(context.Background(), session, StopEvent{FinishedAt: time.Now()})
}
