// Package cost extracts tenant cost guardrail logic from gateway_consumer.
// Sprint 7: CTO-14 refactoring — better testability + maintainability.
package cost

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// DefaultDailyLimit is the default daily request limit per tenant.
const DefaultDailyLimit = 500

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
