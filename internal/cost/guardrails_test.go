package cost

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// mockTracingStore implements store.TracingStore for testing cost guardrails.
type mockTracingStore struct {
	traceCount int
	countErr   error
	tokenUsage map[string]store.TokenUsage
	tokenErr   error
}

func (m *mockTracingStore) CreateTrace(_ context.Context, _ *store.TraceData) error  { return nil }
func (m *mockTracingStore) UpdateTrace(_ context.Context, _ uuid.UUID, _ map[string]any) error {
	return nil
}
func (m *mockTracingStore) GetTrace(_ context.Context, _ uuid.UUID) (*store.TraceData, error) {
	return nil, nil
}
func (m *mockTracingStore) ListTraces(_ context.Context, _ store.TraceListOpts) ([]store.TraceData, error) {
	return nil, nil
}
func (m *mockTracingStore) CountTraces(_ context.Context, _ store.TraceListOpts) (int, error) {
	return m.traceCount, m.countErr
}
func (m *mockTracingStore) CreateSpan(_ context.Context, _ *store.SpanData) error { return nil }
func (m *mockTracingStore) UpdateSpan(_ context.Context, _ uuid.UUID, _ map[string]any) error {
	return nil
}
func (m *mockTracingStore) GetTraceSpans(_ context.Context, _ uuid.UUID) ([]store.SpanData, error) {
	return nil, nil
}
func (m *mockTracingStore) BatchCreateSpans(_ context.Context, _ []store.SpanData) error { return nil }
func (m *mockTracingStore) BatchUpdateTraceAggregates(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (m *mockTracingStore) CountDistinctUsers(_ context.Context, _ time.Time) (int, error) {
	return 0, nil
}
func (m *mockTracingStore) CountByAgent(_ context.Context, _ time.Time) (map[string]int, error) {
	return nil, nil
}
func (m *mockTracingStore) CountByChannel(_ context.Context, _ time.Time) (map[string]int, error) {
	return nil, nil
}
func (m *mockTracingStore) SumTokensByProvider(_ context.Context, _ time.Time) (map[string]store.TokenUsage, error) {
	return m.tokenUsage, m.tokenErr
}

func TestCheckDailyLimit_UnderLimit(t *testing.T) {
	ms := &mockTracingStore{traceCount: 100}
	exceeded, count, limit, err := CheckDailyLimit(context.Background(), ms)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exceeded {
		t.Error("should not be exceeded with 100/500")
	}
	if count != 100 {
		t.Errorf("count: got %d, want 100", count)
	}
	if limit != DefaultDailyLimit {
		t.Errorf("limit: got %d, want %d", limit, DefaultDailyLimit)
	}
}

func TestCheckDailyLimit_AtLimit(t *testing.T) {
	ms := &mockTracingStore{traceCount: 500}
	exceeded, _, _, err := CheckDailyLimit(context.Background(), ms)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exceeded {
		t.Error("should be exceeded at 500/500")
	}
}

func TestCheckDailyLimit_FailOpen(t *testing.T) {
	ms := &mockTracingStore{countErr: errors.New("db down")}
	exceeded, _, _, err := CheckDailyLimit(context.Background(), ms)
	if err == nil {
		t.Fatal("expected error")
	}
	if exceeded {
		t.Error("should fail-open (not exceeded) on error")
	}
}

func TestCheckMonthlyTokenLimit_UnderLimit(t *testing.T) {
	ms := &mockTracingStore{
		tokenUsage: map[string]store.TokenUsage{
			"anthropic":  {InputTokens: 1_000_000, OutputTokens: 500_000},
			"openrouter": {InputTokens: 500_000, OutputTokens: 200_000},
		},
	}
	exceeded, totalTokens, limit, err := CheckMonthlyTokenLimit(context.Background(), ms)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exceeded {
		t.Error("should not be exceeded with 2.2M/10M")
	}
	if totalTokens != 2_200_000 {
		t.Errorf("totalTokens: got %d, want 2200000", totalTokens)
	}
	if limit != DefaultMonthlyTokenLimit {
		t.Errorf("limit: got %d, want %d", limit, DefaultMonthlyTokenLimit)
	}
}

func TestCheckMonthlyTokenLimit_Exceeded(t *testing.T) {
	ms := &mockTracingStore{
		tokenUsage: map[string]store.TokenUsage{
			"anthropic": {InputTokens: 8_000_000, OutputTokens: 3_000_000},
		},
	}
	exceeded, totalTokens, _, err := CheckMonthlyTokenLimit(context.Background(), ms)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exceeded {
		t.Error("should be exceeded with 11M/10M")
	}
	if totalTokens != 11_000_000 {
		t.Errorf("totalTokens: got %d, want 11000000", totalTokens)
	}
}

func TestCheckMonthlyTokenLimit_FailOpen(t *testing.T) {
	ms := &mockTracingStore{tokenErr: errors.New("db down")}
	exceeded, _, _, err := CheckMonthlyTokenLimit(context.Background(), ms)
	if err == nil {
		t.Fatal("expected error")
	}
	if exceeded {
		t.Error("should fail-open (not exceeded) on error")
	}
}

func TestCheckWarningThreshold_BelowThreshold(t *testing.T) {
	ms := &mockTracingStore{traceCount: 100} // 100/500 = 20%
	warning, usagePct, threshold, err := CheckWarningThreshold(context.Background(), ms)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if warning {
		t.Error("should not warn at 20% usage")
	}
	if usagePct >= threshold {
		t.Errorf("usagePct %.2f should be below threshold %.2f", usagePct, threshold)
	}
}

func TestCheckWarningThreshold_AboveThreshold(t *testing.T) {
	ms := &mockTracingStore{traceCount: 450} // 450/500 = 90%
	warning, usagePct, _, err := CheckWarningThreshold(context.Background(), ms)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !warning {
		t.Error("should warn at 90% usage")
	}
	if usagePct < 0.8 {
		t.Errorf("usagePct %.2f should be >= 0.8", usagePct)
	}
}

func TestCheckWarningThreshold_FailOpen(t *testing.T) {
	ms := &mockTracingStore{countErr: errors.New("db down")}
	warning, _, _, err := CheckWarningThreshold(context.Background(), ms)
	if err == nil {
		t.Fatal("expected error")
	}
	if warning {
		t.Error("should fail-open (no warning) on error")
	}
}
