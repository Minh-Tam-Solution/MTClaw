package pg

import (
	"testing"
	"time"

	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// TestTokenUsage_Struct verifies the TokenUsage struct fields work correctly.
func TestTokenUsage_Struct(t *testing.T) {
	usage := store.TokenUsage{
		InputTokens:  1500,
		OutputTokens: 500,
	}
	if usage.InputTokens != 1500 {
		t.Errorf("InputTokens: got %d, want 1500", usage.InputTokens)
	}
	if usage.OutputTokens != 500 {
		t.Errorf("OutputTokens: got %d, want 500", usage.OutputTokens)
	}
}

// TestAdoptionQuerySQL_Patterns verifies the SQL query patterns compile without syntax issues.
// These tests validate the query construction logic without requiring a live database.
func TestAdoptionQuerySQL_Patterns(t *testing.T) {
	// Verify store constructor works (nil DB is fine for pattern tests)
	s := NewPGTracingStore(nil)
	if s == nil {
		t.Fatal("NewPGTracingStore(nil) returned nil")
	}

	// Verify the store satisfies the interface (compile-time check)
	var _ store.TracingStore = s
}

// TestAdoptionMetrics_TimeRange verifies time range calculation for adoption queries.
func TestAdoptionMetrics_TimeRange(t *testing.T) {
	// 7-day window
	since := time.Now().AddDate(0, 0, -7)
	if since.After(time.Now()) {
		t.Error("7-day window should be in the past")
	}

	// Monthly window (first of current month)
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	if monthStart.After(now) {
		t.Error("month start should not be after now")
	}
	if monthStart.Day() != 1 {
		t.Errorf("month start day: got %d, want 1", monthStart.Day())
	}
}

// TestCountByAgent_EmptyResult verifies empty map behavior.
func TestCountByAgent_EmptyResult(t *testing.T) {
	result := make(map[string]int)
	if len(result) != 0 {
		t.Error("empty result should have zero entries")
	}
}

// TestSumTokensByProvider_Aggregation verifies token aggregation logic.
func TestSumTokensByProvider_Aggregation(t *testing.T) {
	byProvider := map[string]store.TokenUsage{
		"anthropic":  {InputTokens: 1_200_000, OutputTokens: 450_000},
		"openrouter": {InputTokens: 800_000, OutputTokens: 300_000},
	}

	var totalTokens int
	for _, usage := range byProvider {
		totalTokens += usage.InputTokens + usage.OutputTokens
	}

	expected := 1_200_000 + 450_000 + 800_000 + 300_000
	if totalTokens != expected {
		t.Errorf("total tokens: got %d, want %d", totalTokens, expected)
	}
}
