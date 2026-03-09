package providers

import (
	"sync"
	"testing"
	"time"
)

func TestHealthTracker_InitialHealthy(t *testing.T) {
	ht := NewProviderHealthTracker()
	if !ht.IsHealthy("unknown-provider") {
		t.Error("unknown provider should be healthy (no data)")
	}
}

func TestHealthTracker_RecordSuccess(t *testing.T) {
	ht := NewProviderHealthTracker()
	ht.RecordSuccess("anthropic")
	ht.RecordSuccess("anthropic")
	ht.RecordSuccess("anthropic")

	score := ht.Score("anthropic")
	if score != 1.0 {
		t.Errorf("expected score 1.0 after 3 successes, got %.2f", score)
	}
	if !ht.IsHealthy("anthropic") {
		t.Error("provider with all successes should be healthy")
	}
}

func TestHealthTracker_RecordFailure(t *testing.T) {
	ht := NewProviderHealthTracker()
	ht.RecordSuccess("anthropic")
	ht.RecordFailure("anthropic")

	score := ht.Score("anthropic")
	if score != 0.5 {
		t.Errorf("expected score 0.5 after 1 success + 1 failure, got %.2f", score)
	}
}

func TestHealthTracker_CircuitBreaker_Trip(t *testing.T) {
	ht := NewProviderHealthTracker()

	// 3 consecutive failures should trip the circuit breaker
	ht.RecordFailure("anthropic")
	ht.RecordFailure("anthropic")

	// Not yet tripped (only 2 consecutive)
	if !ht.IsHealthy("anthropic") {
		t.Error("should still be healthy after 2 consecutive failures")
	}

	ht.RecordFailure("anthropic") // 3rd consecutive failure → trip

	if ht.IsHealthy("anthropic") {
		t.Error("should be unhealthy after 3 consecutive failures (circuit open)")
	}

	stats := ht.Stats()
	if stats["anthropic"].Circuit != "open" {
		t.Errorf("expected circuit 'open', got %q", stats["anthropic"].Circuit)
	}
}

func TestHealthTracker_CircuitBreaker_Cooldown(t *testing.T) {
	ht := NewProviderHealthTracker()
	// Use a very short cooldown for testing
	ht.cooldown = 10 * time.Millisecond

	// Trip the breaker
	ht.RecordFailure("anthropic")
	ht.RecordFailure("anthropic")
	ht.RecordFailure("anthropic")

	if ht.IsHealthy("anthropic") {
		t.Error("should be unhealthy immediately after trip")
	}

	// Wait for cooldown
	time.Sleep(20 * time.Millisecond)

	// Should transition to half-open
	if !ht.IsHealthy("anthropic") {
		t.Error("should be healthy (half-open) after cooldown")
	}

	stats := ht.Stats()
	if stats["anthropic"].Circuit != "half-open" {
		t.Errorf("expected circuit 'half-open', got %q", stats["anthropic"].Circuit)
	}
}

func TestHealthTracker_CircuitBreaker_Recovery(t *testing.T) {
	ht := NewProviderHealthTracker()
	ht.cooldown = 10 * time.Millisecond

	// Trip the breaker
	ht.RecordFailure("anthropic")
	ht.RecordFailure("anthropic")
	ht.RecordFailure("anthropic")

	// Wait for cooldown → half-open
	time.Sleep(20 * time.Millisecond)
	ht.IsHealthy("anthropic") // triggers half-open transition

	// Success in half-open → closed
	ht.RecordSuccess("anthropic")

	stats := ht.Stats()
	if stats["anthropic"].Circuit != "closed" {
		t.Errorf("expected circuit 'closed' after recovery, got %q", stats["anthropic"].Circuit)
	}
	if !ht.IsHealthy("anthropic") {
		t.Error("should be healthy after recovery")
	}
}

func TestHealthTracker_SlidingWindow(t *testing.T) {
	ht := NewProviderHealthTracker()
	// Use a very short time window for testing
	ht.windowDur = 50 * time.Millisecond

	ht.RecordFailure("anthropic")
	ht.RecordFailure("anthropic")

	score := ht.Score("anthropic")
	if score != 0.0 {
		t.Errorf("expected score 0.0 after 2 failures, got %.2f", score)
	}

	// Wait for results to expire
	time.Sleep(60 * time.Millisecond)

	// Expired results should not count → score returns 1.0 (no active data)
	score = ht.Score("anthropic")
	if score != 1.0 {
		t.Errorf("expected score 1.0 after window expiry, got %.2f", score)
	}
}

func TestHealthTracker_Score_Empty(t *testing.T) {
	ht := NewProviderHealthTracker()
	score := ht.Score("nonexistent")
	if score != 1.0 {
		t.Errorf("expected score 1.0 for nonexistent provider, got %.2f", score)
	}
}

func TestHealthTracker_Stats(t *testing.T) {
	ht := NewProviderHealthTracker()
	ht.RecordSuccess("anthropic")
	ht.RecordSuccess("anthropic")
	ht.RecordFailure("anthropic")
	ht.RecordSuccess("openrouter")

	stats := ht.Stats()

	if len(stats) != 2 {
		t.Fatalf("expected 2 providers in stats, got %d", len(stats))
	}

	as := stats["anthropic"]
	if as.OKCount != 2 || as.FailCount != 1 {
		t.Errorf("anthropic: expected 2 OK + 1 fail, got %d OK + %d fail", as.OKCount, as.FailCount)
	}
	if as.Circuit != "closed" {
		t.Errorf("anthropic: expected circuit 'closed', got %q", as.Circuit)
	}

	os := stats["openrouter"]
	if os.OKCount != 1 || os.FailCount != 0 {
		t.Errorf("openrouter: expected 1 OK + 0 fail, got %d OK + %d fail", os.OKCount, os.FailCount)
	}
}

func TestHealthTracker_Concurrent(t *testing.T) {
	ht := NewProviderHealthTracker()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			ht.RecordSuccess("anthropic")
		}()
		go func() {
			defer wg.Done()
			ht.RecordFailure("openrouter")
		}()
		go func() {
			defer wg.Done()
			_ = ht.IsHealthy("anthropic")
			_ = ht.Score("openrouter")
			_ = ht.Stats()
		}()
	}
	wg.Wait()

	// If we got here without -race flag errors, concurrent access is safe
	if !ht.IsHealthy("anthropic") {
		t.Error("anthropic should be healthy (all successes)")
	}
}

func TestHealthTracker_FailOpenOnDoubleCircuitOpen(t *testing.T) {
	// OBS-028-1: When both primary and fallback circuits are open,
	// IsHealthy should still allow the call (fail-open behavior).
	// This test verifies that the caller can implement fail-open logic:
	// if IsHealthy returns false but no other option exists, still attempt.
	ht := NewProviderHealthTracker()

	// Trip both circuits
	for i := 0; i < 3; i++ {
		ht.RecordFailure("primary")
		ht.RecordFailure("fallback")
	}

	// Both are unhealthy
	if ht.IsHealthy("primary") {
		t.Error("primary should be unhealthy")
	}
	if ht.IsHealthy("fallback") {
		t.Error("fallback should be unhealthy")
	}

	// The caller should implement fail-open: if both are unhealthy
	// and there's no other option, still attempt the fallback call.
	// This test documents the expected pattern:
	//   if !tracker.IsHealthy(fallback) && noOtherOption { attempt anyway }
	// The health tracker itself doesn't enforce fail-open — it reports state.
	// The loop.go caller makes the final decision.
	stats := ht.Stats()
	if stats["primary"].Circuit != "open" {
		t.Errorf("expected primary circuit 'open', got %q", stats["primary"].Circuit)
	}
	if stats["fallback"].Circuit != "open" {
		t.Errorf("expected fallback circuit 'open', got %q", stats["fallback"].Circuit)
	}
}
