package providers

import (
	"log/slog"
	"os"
	"strconv"
	"sync"
	"time"
)

// circuitState represents the circuit breaker state for a provider.
type circuitState int

const (
	circuitClosed   circuitState = iota // normal — requests flow through
	circuitOpen                         // tripped — skip provider
	circuitHalfOpen                     // probe — allow 1 request to test recovery
)

// Default health tracker settings.
const (
	defaultWindowSize      = 100
	defaultWindowDur       = 10 * time.Minute
	defaultCBCooldown      = 30 * time.Second
	defaultTripThreshold   = 3   // consecutive failures to trip
	defaultMinScoreToTrip  = 0.5 // trip if success rate drops below 50%
)

// healthResult records one call outcome.
type healthResult struct {
	ok bool
	at time.Time
}

// providerHealth tracks per-provider health state.
type providerHealth struct {
	results    []healthResult // circular buffer
	idx        int
	cbState    circuitState
	cbOpenAt   time.Time     // when circuit opened
	cbCooldown time.Duration // how long to wait before half-open probe
	consecFail int           // consecutive failure counter
}

// HealthStats is the public view of a provider's health.
type HealthStats struct {
	Score      float64 `json:"score"`       // 0.0-1.0 success rate
	Circuit    string  `json:"circuit"`     // "closed", "open", "half-open"
	OKCount    int     `json:"ok_count"`
	FailCount  int     `json:"fail_count"`
	ConsecFail int     `json:"consec_fail"`
}

// ProviderHealthTracker monitors provider health with sliding window and circuit breaker.
type ProviderHealthTracker struct {
	mu        sync.RWMutex
	records   map[string]*providerHealth
	window    int
	windowDur time.Duration
	cooldown  time.Duration
}

// NewProviderHealthTracker creates a health tracker with default settings.
// Cooldown can be overridden via MTCLAW_PROVIDER_CB_COOLDOWN env var (seconds).
func NewProviderHealthTracker() *ProviderHealthTracker {
	cooldown := defaultCBCooldown
	if v := os.Getenv("MTCLAW_PROVIDER_CB_COOLDOWN"); v != "" {
		if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
			cooldown = time.Duration(sec) * time.Second
		}
	}
	return &ProviderHealthTracker{
		records:   make(map[string]*providerHealth),
		window:    defaultWindowSize,
		windowDur: defaultWindowDur,
		cooldown:  cooldown,
	}
}

// RecordSuccess records a successful call to the provider.
func (t *ProviderHealthTracker) RecordSuccess(provider string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	h := t.getOrCreate(provider)
	t.addResult(h, true)
	h.consecFail = 0

	// Recovery: half-open → closed on success
	if h.cbState == circuitHalfOpen {
		h.cbState = circuitClosed
		slog.Info("provider circuit breaker recovered",
			"provider", provider, "state", "closed")
	}
}

// RecordFailure records a failed call to the provider.
func (t *ProviderHealthTracker) RecordFailure(provider string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	h := t.getOrCreate(provider)
	t.addResult(h, false)
	h.consecFail++

	// Check trip conditions: consecutive failures OR low success rate (with minimum sample)
	if h.cbState == circuitClosed {
		ok, fail := t.countResults(h)
		total := ok + fail
		if h.consecFail >= defaultTripThreshold || (total >= defaultTripThreshold && t.scoreUnlocked(h) < defaultMinScoreToTrip) {
			h.cbState = circuitOpen
			h.cbOpenAt = time.Now()
			slog.Warn("provider circuit breaker tripped",
				"provider", provider, "consec_fail", h.consecFail,
				"score", t.scoreUnlocked(h))
		}
	}

	// Half-open failure → back to open
	if h.cbState == circuitHalfOpen {
		h.cbState = circuitOpen
		h.cbOpenAt = time.Now()
		slog.Warn("provider circuit breaker re-tripped from half-open",
			"provider", provider)
	}
}

// IsHealthy returns true if the provider should be attempted.
// Returns true for unknown providers (fail-open for new providers).
func (t *ProviderHealthTracker) IsHealthy(provider string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	h, ok := t.records[provider]
	if !ok {
		return true // unknown provider = healthy (no data)
	}

	switch h.cbState {
	case circuitClosed:
		return true
	case circuitHalfOpen:
		return true // allow probe request
	case circuitOpen:
		// Check if cooldown has elapsed → transition to half-open
		if time.Since(h.cbOpenAt) >= h.cbCooldown {
			h.cbState = circuitHalfOpen
			slog.Debug("provider circuit breaker half-open (cooldown elapsed)",
				"provider", provider)
			return true
		}
		return false
	}
	return true
}

// Score returns the success rate for a provider (0.0-1.0).
// Returns 1.0 for unknown providers.
func (t *ProviderHealthTracker) Score(provider string) float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	h, ok := t.records[provider]
	if !ok {
		return 1.0
	}
	return t.scoreUnlocked(h)
}

// Stats returns health statistics for all tracked providers.
func (t *ProviderHealthTracker) Stats() map[string]HealthStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	stats := make(map[string]HealthStats, len(t.records))
	for name, h := range t.records {
		ok, fail := t.countResults(h)
		stats[name] = HealthStats{
			Score:      t.scoreUnlocked(h),
			Circuit:    circuitStateString(h.cbState),
			OKCount:    ok,
			FailCount:  fail,
			ConsecFail: h.consecFail,
		}
	}
	return stats
}

// --- internal ---

func (t *ProviderHealthTracker) getOrCreate(provider string) *providerHealth {
	h, ok := t.records[provider]
	if !ok {
		h = &providerHealth{
			results:    make([]healthResult, 0, t.window),
			cbCooldown: t.cooldown,
		}
		t.records[provider] = h
	}
	return h
}

func (t *ProviderHealthTracker) addResult(h *providerHealth, ok bool) {
	now := time.Now()
	r := healthResult{ok: ok, at: now}

	if len(h.results) < t.window {
		h.results = append(h.results, r)
	} else {
		h.results[h.idx] = r
		h.idx = (h.idx + 1) % t.window
	}
}

// scoreUnlocked calculates success rate from non-expired results.
// Caller must hold at least RLock.
func (t *ProviderHealthTracker) scoreUnlocked(h *providerHealth) float64 {
	ok, fail := t.countResults(h)
	total := ok + fail
	if total == 0 {
		return 1.0 // no data = healthy
	}
	return float64(ok) / float64(total)
}

// countResults counts OK and fail results within the time window.
func (t *ProviderHealthTracker) countResults(h *providerHealth) (ok, fail int) {
	cutoff := time.Now().Add(-t.windowDur)
	for _, r := range h.results {
		if r.at.Before(cutoff) {
			continue // expired
		}
		if r.ok {
			ok++
		} else {
			fail++
		}
	}
	return
}

func circuitStateString(s circuitState) string {
	switch s {
	case circuitClosed:
		return "closed"
	case circuitOpen:
		return "open"
	case circuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}
