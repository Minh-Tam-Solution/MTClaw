package claudecode

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// HealthStatus represents the overall health of the bridge system.
type HealthStatus struct {
	Healthy       bool              `json:"healthy"`
	TmuxAvailable bool             `json:"tmux_available"`
	ActiveCount   int              `json:"active_sessions"`
	DeadSessions  []string         `json:"dead_sessions,omitempty"`
	Checks        map[string]string `json:"checks"`
	CheckedAt     time.Time        `json:"checked_at"`
}

// HealthMonitor periodically checks bridge health and cleans up dead sessions.
type HealthMonitor struct {
	sessions *SessionManager
	tmux     *TmuxBridge
	interval time.Duration
	mu       sync.RWMutex
	last     HealthStatus
}

// NewHealthMonitor creates a health monitor with the given check interval.
func NewHealthMonitor(sessions *SessionManager, tmux *TmuxBridge, interval time.Duration) *HealthMonitor {
	if interval == 0 {
		interval = 30 * time.Second
	}
	return &HealthMonitor{
		sessions: sessions,
		tmux:     tmux,
		interval: interval,
	}
}

// Start begins periodic health checks. Blocks until context is cancelled.
func (h *HealthMonitor) Start(ctx context.Context) {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	// Initial check
	h.check(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.check(ctx)
		}
	}
}

// LastStatus returns the most recent health check result.
func (h *HealthMonitor) LastStatus() HealthStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.last
}

// check performs a single health check cycle.
func (h *HealthMonitor) check(ctx context.Context) {
	status := HealthStatus{
		Healthy:   true,
		Checks:    make(map[string]string),
		CheckedAt: time.Now(),
	}

	// 1. tmux availability
	if h.tmux != nil {
		if _, err := h.tmux.ListSessions(ctx); err != nil {
			status.TmuxAvailable = false
			status.Checks["tmux"] = fmt.Sprintf("unavailable: %v", err)
		} else {
			status.TmuxAvailable = true
			status.Checks["tmux"] = "ok"
		}
	} else {
		status.Checks["tmux"] = "not configured"
	}

	// 2. Session liveness — detect dead tmux sessions
	status.ActiveCount = h.sessions.SessionCount()
	if h.tmux != nil {
		dead := h.detectDeadSessions(ctx)
		status.DeadSessions = dead
		if len(dead) > 0 {
			status.Checks["dead_sessions"] = fmt.Sprintf("%d dead sessions detected", len(dead))
			slog.Warn("health check: dead sessions found", "count", len(dead), "sessions", dead)
		} else {
			status.Checks["dead_sessions"] = "none"
		}
	}

	// Overall healthy if tmux is available and no dead sessions
	status.Healthy = status.TmuxAvailable && len(status.DeadSessions) == 0

	h.mu.Lock()
	h.last = status
	h.mu.Unlock()
}

// detectDeadSessions finds bridge sessions whose tmux targets no longer exist.
func (h *HealthMonitor) detectDeadSessions(ctx context.Context) []string {
	// List all known tmux sessions
	tmuxSessions, err := h.tmux.ListSessions(ctx)
	if err != nil {
		return nil // can't check
	}

	alive := make(map[string]bool, len(tmuxSessions))
	for _, ts := range tmuxSessions {
		alive[ts.Name] = true
	}

	// Check each bridge session against tmux
	var dead []string
	allSessions := h.sessions.allSessions()
	for _, s := range allSessions {
		if s.Status == SessionStateStopped {
			continue
		}
		if s.TmuxTarget != "" && !alive[s.TmuxTarget] {
			dead = append(dead, s.ID)
		}
	}
	return dead
}
