package claudecode

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// PermissionDecision represents the outcome of a permission request.
type PermissionDecision string

const (
	PermissionPending  PermissionDecision = "pending"
	PermissionApproved PermissionDecision = "approved"
	PermissionDenied   PermissionDecision = "denied"
	PermissionExpired  PermissionDecision = "expired"
)

// PermissionRequest represents an async tool approval request (D6/Sprint C).
type PermissionRequest struct {
	ID           string             `json:"id"`
	OwnerID      string             `json:"owner_id"`       // tenant
	SessionID    string             `json:"session_id"`
	Tool         string             `json:"tool"`
	ToolInput    json.RawMessage    `json:"tool_input,omitempty"`
	RiskLevel    string             `json:"risk_level"` // "low" or "high"
	RequestHash  string             `json:"request_hash"`   // idempotency key (L2)
	ActorID      string             `json:"actor_id"`       // who requested (usually the bridge user)
	Decision     PermissionDecision `json:"decision"`
	DecidedBy    string             `json:"decided_by,omitempty"`
	ExpiresAt    time.Time          `json:"expires_at"`
	DecidedAt    time.Time          `json:"decided_at,omitempty"`
	CreatedAt    time.Time          `json:"created_at"`
}

const (
	// DefaultPermissionTTL is how long a permission request stays pending before timeout.
	DefaultPermissionTTL = 3 * time.Minute
)

// HighRiskTools are tools that MUST be denied on timeout (fail-closed, D6).
var HighRiskTools = map[string]bool{
	"Bash":   true,
	"Edit":   true,
	"Write":  true,
	"Agent":  true,
	"Delete": true,
}

// IsHighRisk returns true if the tool requires explicit approval (deny on timeout).
func IsHighRisk(tool string) bool {
	return HighRiskTools[tool]
}

// ComputeRequestHash returns the idempotency key for a permission request (L2).
// Hash input: sha256(session_id + tool_name + canonical_tool_input + timestamp_minute_bucket).
func ComputeRequestHash(sessionID, tool string, toolInput json.RawMessage, ts time.Time) string {
	minuteBucket := ts.Truncate(time.Minute).Unix()
	payload := fmt.Sprintf("%s:%s:%s:%d", sessionID, tool, string(toolInput), minuteBucket)
	h := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(h[:])
}

// PermissionStore manages async permission requests with TTL enforcement.
// Thread-safe, in-memory implementation. PG backing store is Sprint D.
type PermissionStore struct {
	mu       sync.RWMutex
	requests map[string]*PermissionRequest // keyed by request ID
	byHash   map[string]string             // request_hash -> request ID (dedup)
}

// NewPermissionStore creates an empty permission store.
func NewPermissionStore() *PermissionStore {
	return &PermissionStore{
		requests: make(map[string]*PermissionRequest),
		byHash:   make(map[string]string),
	}
}

// Create adds a new permission request. Returns existing request if request_hash matches (dedup).
func (ps *PermissionStore) Create(req *PermissionRequest) (*PermissionRequest, error) {
	if req.ID == "" || req.SessionID == "" || req.Tool == "" || req.RequestHash == "" {
		return nil, fmt.Errorf("permission request missing required fields (id, session_id, tool, request_hash)")
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Dedup: if request_hash already exists, return existing request (L2/acceptance #5)
	if existingID, ok := ps.byHash[req.RequestHash]; ok {
		if existing, exists := ps.requests[existingID]; exists {
			return existing, nil
		}
	}

	if req.CreatedAt.IsZero() {
		req.CreatedAt = time.Now()
	}
	if req.ExpiresAt.IsZero() {
		req.ExpiresAt = req.CreatedAt.Add(DefaultPermissionTTL)
	}
	if req.Decision == "" {
		req.Decision = PermissionPending
	}

	ps.requests[req.ID] = req
	ps.byHash[req.RequestHash] = req.ID
	return req, nil
}

// Get returns a permission request by ID. Returns nil if not found.
// Applies TTL enforcement: if expired and still pending, transitions to the fail-safe decision.
func (ps *PermissionStore) Get(id string) *PermissionRequest {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	req, ok := ps.requests[id]
	if !ok {
		return nil
	}

	// TTL enforcement: auto-decide expired pending requests
	if req.Decision == PermissionPending && time.Now().After(req.ExpiresAt) {
		if IsHighRisk(req.Tool) {
			req.Decision = PermissionDenied // fail-closed (D6)
		} else {
			req.Decision = PermissionApproved // low-risk auto-approve
		}
		req.DecidedAt = req.ExpiresAt
		req.DecidedBy = "system:timeout"
	}

	return req
}

// GetByHash looks up a permission request by its request_hash.
func (ps *PermissionStore) GetByHash(hash string) *PermissionRequest {
	ps.mu.RLock()
	id, ok := ps.byHash[hash]
	ps.mu.RUnlock()
	if !ok {
		return nil
	}
	return ps.Get(id)
}

// Decide resolves a pending permission request. Returns error if already decided (prevents double-apply).
func (ps *PermissionStore) Decide(id string, decision PermissionDecision, decidedBy string, approverACL []string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	req, ok := ps.requests[id]
	if !ok {
		return fmt.Errorf("permission request %q not found", id)
	}

	// Already decided — prevent double-apply (acceptance #5)
	if req.Decision != PermissionPending {
		return fmt.Errorf("permission request %q already decided: %s (reason_code=permission_already_decided)", id, req.Decision)
	}

	// TTL check — if expired, the timeout decision takes precedence
	if time.Now().After(req.ExpiresAt) {
		if IsHighRisk(req.Tool) {
			req.Decision = PermissionDenied
		} else {
			req.Decision = PermissionApproved
		}
		req.DecidedAt = req.ExpiresAt
		req.DecidedBy = "system:timeout"
		return fmt.Errorf("permission request %q expired before decision (reason_code=permission_expired)", id)
	}

	// ACL check — decidedBy must be in approverACL or be the session owner
	if len(approverACL) > 0 {
		allowed := false
		for _, a := range approverACL {
			if a == decidedBy {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("actor %q not in approver ACL (reason_code=permission_acl_mismatch)", decidedBy)
		}
	}

	req.Decision = decision
	req.DecidedBy = decidedBy
	req.DecidedAt = time.Now()
	return nil
}

// ListPending returns all pending permission requests for a session.
func (ps *PermissionStore) ListPending(sessionID string) []*PermissionRequest {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var result []*PermissionRequest
	for _, req := range ps.requests {
		if req.SessionID == sessionID && req.Decision == PermissionPending {
			result = append(result, req)
		}
	}
	return result
}

// Cleanup removes expired and decided requests older than maxAge.
func (ps *PermissionStore) Cleanup(maxAge time.Duration) int {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0
	for id, req := range ps.requests {
		if req.Decision != PermissionPending && req.CreatedAt.Before(cutoff) {
			delete(ps.byHash, req.RequestHash)
			delete(ps.requests, id)
			removed++
		}
	}
	return removed
}

// Count returns the total number of permission requests in the store.
func (ps *PermissionStore) Count() int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return len(ps.requests)
}
