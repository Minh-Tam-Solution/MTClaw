package claudecode

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// NotifyFunc is the callback signature for delivering messages to a channel (e.g. Telegram).
type NotifyFunc func(ctx context.Context, channel, chatID, message string) error

// Notifier delivers bridge events to channels with circuit breaker protection.
type Notifier struct {
	sendFn   NotifyFunc
	redactor *OutputRedactor
	breaker  *circuitBreaker
}

// circuitBreaker prevents cascading failures when notification delivery fails.
// States: closed (healthy) -> open (tripped) -> half-open (probe).
type circuitBreaker struct {
	mu           sync.Mutex
	failures     int
	state        breakerState
	lastFailure  time.Time
	threshold    int           // failures before tripping
	resetTimeout time.Duration // time before trying half-open
}

type breakerState int

const (
	breakerClosed   breakerState = iota // normal operation
	breakerOpen                         // tripped — all calls rejected
	breakerHalfOpen                     // probing — one call allowed
)

const (
	defaultBreakerThreshold = 3
	defaultBreakerReset     = 30 * time.Second
)

// NewNotifier creates a notifier with circuit breaker.
func NewNotifier(sendFn NotifyFunc) *Notifier {
	return &Notifier{
		sendFn:   sendFn,
		redactor: NewOutputRedactor(),
		breaker: &circuitBreaker{
			threshold:    defaultBreakerThreshold,
			resetTimeout: defaultBreakerReset,
		},
	}
}

// NotifyStop sends a formatted stop notification to the session's channel.
func (n *Notifier) NotifyStop(ctx context.Context, session *BridgeSession, event StopEvent) {
	if n.sendFn == nil {
		return
	}

	// Format the message
	msg := n.formatStopMessage(session, event)

	// Send to all notification recipients
	for _, actorID := range session.NotifyACL {
		if err := n.send(ctx, session.Channel, session.ChatID, msg); err != nil {
			slog.Warn("stop notification failed",
				"session", session.ID,
				"actor", actorID,
				"error", err,
			)
		}
		_ = actorID // notification routing per-actor is Sprint D
	}
}

// send delivers a message through the circuit breaker.
func (n *Notifier) send(ctx context.Context, channel, chatID, message string) error {
	if !n.breaker.allow() {
		return fmt.Errorf("circuit breaker open: notifications degraded (reason_code=breaker_open)")
	}

	err := n.sendFn(ctx, channel, chatID, message)
	n.breaker.record(err)
	return err
}

// formatStopMessage creates a human-readable stop notification.
func (n *Notifier) formatStopMessage(session *BridgeSession, event StopEvent) string {
	heavyRedact := session.Capabilities.RedactHeavy

	status := "completed"
	if event.ExitCode != 0 {
		status = fmt.Sprintf("failed (exit %d)", event.ExitCode)
	}

	msg := fmt.Sprintf("Session %s %s\nProject: %s\nRisk: %s\n",
		session.ID, status, session.ProjectPath, session.RiskMode)

	if event.Summary != "" {
		summary := n.redactor.Redact(event.Summary, heavyRedact)
		summary = TruncateOutput(summary, 20)
		msg += fmt.Sprintf("\nSummary:\n%s\n", summary)
	}

	if event.GitDiff != "" {
		diff := n.redactor.Redact(event.GitDiff, heavyRedact)
		diff = TruncateOutput(diff, 50)
		if len(diff) > 2000 {
			diff = diff[:2000] + "\n... [diff truncated at 2000 chars]"
		}
		msg += fmt.Sprintf("\nGit diff:\n```\n%s\n```\n", diff)
	}

	msg += fmt.Sprintf("\nFinished: %s", event.FinishedAt.Format("15:04:05"))
	return msg
}

// NotifyPermission sends a permission approval request to the session's channel.
// The message includes tool name, risk level, and permission ID for callback.
func (n *Notifier) NotifyPermission(ctx context.Context, session *BridgeSession, perm *PermissionRequest) {
	if n.sendFn == nil {
		return
	}

	msg := n.formatPermissionMessage(session, perm)

	// Send to owner (who will see the inline keyboard in Telegram)
	if err := n.send(ctx, session.Channel, session.ChatID, msg); err != nil {
		slog.Warn("permission notification failed",
			"session", session.ID,
			"permission_id", perm.ID,
			"error", err,
		)
	}
}

// formatPermissionMessage creates a human-readable permission request notification.
func (n *Notifier) formatPermissionMessage(session *BridgeSession, perm *PermissionRequest) string {
	riskEmoji := "🟢"
	if perm.RiskLevel == "high" {
		riskEmoji = "🔴"
	}

	msg := fmt.Sprintf("%s Permission Request\n\nSession: %s\nTool: %s\nRisk: %s\nExpires: %s\n\nPermission ID: %s",
		riskEmoji,
		session.ID,
		perm.Tool,
		perm.RiskLevel,
		perm.ExpiresAt.Format("15:04:05"),
		perm.ID,
	)

	if len(perm.ToolInput) > 0 && string(perm.ToolInput) != "null" {
		input := n.redactor.Redact(string(perm.ToolInput), session.Capabilities.RedactHeavy)
		if len(input) > 500 {
			input = input[:500] + "..."
		}
		msg += fmt.Sprintf("\n\nInput:\n```\n%s\n```", input)
	}

	return msg
}

// BreakerState returns the current circuit breaker state for diagnostics.
func (n *Notifier) BreakerState() string {
	n.breaker.mu.Lock()
	defer n.breaker.mu.Unlock()
	switch n.breaker.state {
	case breakerOpen:
		return "open"
	case breakerHalfOpen:
		return "half-open"
	default:
		return "closed"
	}
}

// allow checks if a request should be permitted through the breaker.
func (cb *circuitBreaker) allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case breakerClosed:
		return true
	case breakerOpen:
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			cb.state = breakerHalfOpen
			return true // allow one probe
		}
		return false
	case breakerHalfOpen:
		return true // allow probe
	}
	return true
}

// record updates breaker state based on call result.
func (cb *circuitBreaker) record(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err == nil {
		// Success: reset
		cb.failures = 0
		cb.state = breakerClosed
		return
	}

	// Failure
	cb.failures++
	cb.lastFailure = time.Now()

	if cb.state == breakerHalfOpen {
		// Probe failed — reopen
		cb.state = breakerOpen
		return
	}

	if cb.failures >= cb.threshold {
		cb.state = breakerOpen
		slog.Warn("circuit breaker tripped",
			"failures", cb.failures,
			"threshold", cb.threshold,
		)
	}
}
