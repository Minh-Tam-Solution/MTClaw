package claudecode

import (
	"fmt"
	"sync"
	"time"
)

// Valid state transitions for a bridge session.
var validTransitions = map[SessionState][]SessionState{
	SessionStateActive: {SessionStateBusy, SessionStateIdle, SessionStateError, SessionStateStopped},
	SessionStateBusy:   {SessionStateActive, SessionStateIdle, SessionStateError, SessionStateStopped},
	SessionStateIdle:   {SessionStateActive, SessionStateBusy, SessionStateStopped},
	SessionStateError:  {SessionStateActive, SessionStateStopped},
	// SessionStateStopped is terminal — no transitions out
}

// Session wraps BridgeSession with state machine logic and message queue.
type Session struct {
	mu   sync.Mutex
	data BridgeSession

	// messageQueue holds pending messages when session is BUSY.
	// Max queue size prevents unbounded memory growth.
	messageQueue []string
	maxQueueSize int

	// turnContext holds per-session intelligence context (Sprint 20B).
	// Consumed once: prepended as markdown to the next sendKeys message.
	turnContext *TurnContext
}

const defaultMaxQueueSize = 20

// NewSession creates a Session from initial data.
func NewSession(data BridgeSession) *Session {
	return &Session{
		data:         data,
		maxQueueSize: defaultMaxQueueSize,
	}
}

// Data returns a copy of the session data (thread-safe).
func (s *Session) Data() BridgeSession {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data
}

// TransitionTo attempts a state transition. Returns error if invalid.
func (s *Session) TransitionTo(newState SessionState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data.Status == newState {
		return nil // no-op
	}

	allowed, ok := validTransitions[s.data.Status]
	if !ok {
		return fmt.Errorf("no transitions from terminal state %q", s.data.Status)
	}

	for _, a := range allowed {
		if a == newState {
			s.data.Status = newState
			s.data.LastActivityAt = time.Now()
			return nil
		}
	}

	return fmt.Errorf("invalid transition: %s -> %s", s.data.Status, newState)
}

// ForceStop transitions to stopped regardless of current state.
func (s *Session) ForceStop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Status = SessionStateStopped
	s.data.LastActivityAt = time.Now()
}

// UpdateRiskMode changes the risk mode and recalculates capabilities.
// Validates ownership rules:
//   - read: anyone can downgrade
//   - patch: session owner only
//   - interactive: requires admin (checked by caller)
func (s *Session) UpdateRiskMode(mode RiskMode, actorID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Downgrade to read is always allowed
	if mode == RiskModeRead {
		s.data.RiskMode = mode
		s.data.Capabilities = CapabilitiesForRisk(mode)
		return nil
	}

	// Escalation requires owner
	if mode == RiskModePatch && actorID != s.data.OwnerActorID {
		return fmt.Errorf("only session owner can escalate to %s", mode)
	}

	// Interactive requires admin check (done by caller) + owner check
	if mode == RiskModeInteractive && actorID != s.data.OwnerActorID {
		return fmt.Errorf("only session owner can escalate to %s", mode)
	}

	s.data.RiskMode = mode
	s.data.Capabilities = CapabilitiesForRisk(mode)
	return nil
}

// EnqueueMessage adds a message to the queue (when session is BUSY).
// Returns error if queue is full.
func (s *Session) EnqueueMessage(msg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.messageQueue) >= s.maxQueueSize {
		return fmt.Errorf("message queue full (max %d)", s.maxQueueSize)
	}
	s.messageQueue = append(s.messageQueue, msg)
	return nil
}

// DrainQueue returns all queued messages and clears the queue.
func (s *Session) DrainQueue() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.messageQueue) == 0 {
		return nil
	}
	msgs := s.messageQueue
	s.messageQueue = nil
	return msgs
}

// QueueLen returns the number of queued messages.
func (s *Session) QueueLen() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.messageQueue)
}

// IsOwner checks if the given actor is the session owner.
func (s *Session) IsOwner(actorID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data.OwnerActorID == actorID
}

// IsApprover checks if the given actor is in the approver ACL.
func (s *Session) IsApprover(actorID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Owner is always an approver
	if s.data.OwnerActorID == actorID {
		return true
	}
	for _, id := range s.data.ApproverACL {
		if id == actorID {
			return true
		}
	}
	return false
}

// CanReceiveNotification checks if the given actor is in the notify ACL.
func (s *Session) CanReceiveNotification(actorID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range s.data.NotifyACL {
		if id == actorID {
			return true
		}
	}
	return false
}

// Touch updates the last activity timestamp.
func (s *Session) Touch() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.LastActivityAt = time.Now()
}

// SetTurnContext merges into the per-session turn context (Sprint 20B).
// Non-empty fields in tc are appended to existing context.
// The accumulated context is consumed once on the next sendKeys call.
func (s *Session) SetTurnContext(tc *TurnContext) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.turnContext == nil {
		s.turnContext = &TurnContext{}
	}
	if len(tc.SprintGoals) > 0 {
		s.turnContext.SprintGoals = append(s.turnContext.SprintGoals, tc.SprintGoals...)
	}
	if len(tc.Blockers) > 0 {
		s.turnContext.Blockers = append(s.turnContext.Blockers, tc.Blockers...)
	}
	if len(tc.FixHints) > 0 {
		s.turnContext.FixHints = append(s.turnContext.FixHints, tc.FixHints...)
	}
}

// ClearTurnContext removes all pending turn context.
func (s *Session) ClearTurnContext() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.turnContext = nil
}

// PeekTurnContext returns a copy of the pending turn context without clearing it.
func (s *Session) PeekTurnContext() *TurnContext {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.turnContext == nil {
		return nil
	}
	cp := *s.turnContext
	cp.SprintGoals = append([]string(nil), s.turnContext.SprintGoals...)
	cp.Blockers = append([]string(nil), s.turnContext.Blockers...)
	cp.FixHints = append([]string(nil), s.turnContext.FixHints...)
	return &cp
}

// ConsumeTurnContext returns and clears the pending turn context.
// Returns nil if no context is set.
func (s *Session) ConsumeTurnContext() *TurnContext {
	s.mu.Lock()
	defer s.mu.Unlock()
	tc := s.turnContext
	s.turnContext = nil
	return tc
}
