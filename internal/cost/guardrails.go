// Package cost extracts tenant cost guardrail logic from gateway_consumer.
// Sprint 7: CTO-14 refactoring — better testability + maintainability.
// Sprint 27: Added monthly token tracking + warning threshold.
package cost

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// DefaultDailyLimit is the default daily request limit per tenant.
const DefaultDailyLimit = 500

// DefaultMonthlyTokenLimit is the default monthly token limit per tenant.
const DefaultMonthlyTokenLimit = 10_000_000

// DefaultWarningThreshold is the default warning percentage (0.0-1.0).
const DefaultWarningThreshold = 0.8

// CheckDailyLimit checks if the tenant has exceeded the daily request limit.
// Returns (exceeded, count, limit, err).
// Fail-open: returns (false, 0, 0, err) on error so the request proceeds.
// CTO-11 FIX: Filters by today's date (was counting ALL traces).
func CheckDailyLimit(ctx context.Context, tracingStore store.TracingStore) (bool, int, int, error) {
	today := time.Now().Truncate(24 * time.Hour)
	count, err := tracingStore.CountTraces(ctx, store.TraceListOpts{
		Since: &today,
	})
	if err != nil {
		return false, 0, 0, err
	}

	limit := DefaultDailyLimit
	if envLimit := os.Getenv("MTCLAW_TENANT_DAILY_REQUEST_LIMIT"); envLimit != "" {
		if parsed, parseErr := strconv.Atoi(envLimit); parseErr == nil && parsed > 0 {
			limit = parsed
		}
	}

	return count >= limit, count, limit, nil
}

// CheckMonthlyTokenLimit checks if the tenant has exceeded the monthly token limit.
// Sums input+output tokens across all providers since the first day of the current month.
// Returns (exceeded, totalTokens, limit, err).
// Fail-open: returns (false, 0, 0, err) on error so the request proceeds.
func CheckMonthlyTokenLimit(ctx context.Context, tracingStore store.TracingStore) (bool, int, int, error) {
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	byProvider, err := tracingStore.SumTokensByProvider(ctx, monthStart)
	if err != nil {
		return false, 0, 0, err
	}

	var totalTokens int
	for _, usage := range byProvider {
		totalTokens += usage.InputTokens + usage.OutputTokens
	}

	limit := DefaultMonthlyTokenLimit
	if envLimit := os.Getenv("MTCLAW_TENANT_MONTHLY_TOKEN_LIMIT"); envLimit != "" {
		if parsed, parseErr := strconv.Atoi(envLimit); parseErr == nil && parsed > 0 {
			limit = parsed
		}
	}

	return totalTokens >= limit, totalTokens, limit, nil
}

// CheckWarningThreshold checks if daily request usage exceeds a configurable
// warning percentage and emits a structured WARN log if so.
// Returns (warning, usagePct, threshold).
// Fail-open: returns (false, 0, 0, err) on error.
func CheckWarningThreshold(ctx context.Context, tracingStore store.TracingStore) (bool, float64, float64, error) {
	_, count, limit, err := CheckDailyLimit(ctx, tracingStore)
	if err != nil {
		return false, 0, 0, err
	}

	threshold := DefaultWarningThreshold
	if envThreshold := os.Getenv("MTCLAW_COST_WARNING_THRESHOLD"); envThreshold != "" {
		if parsed, parseErr := strconv.ParseFloat(envThreshold, 64); parseErr == nil && parsed > 0 && parsed <= 1.0 {
			threshold = parsed
		}
	}

	usagePct := float64(0)
	if limit > 0 {
		usagePct = float64(count) / float64(limit)
	}

	warning := usagePct >= threshold
	if warning {
		slog.Warn("daily request usage approaching limit",
			"count", count, "limit", limit,
			"usage_pct", int(usagePct*100),
			"threshold_pct", int(threshold*100))
	}

	return warning, usagePct, threshold, nil
}
