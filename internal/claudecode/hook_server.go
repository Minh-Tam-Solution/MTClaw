package claudecode

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// HookServer listens on localhost for signed webhook callbacks from Claude Code (D4/D5).
// Binds to 127.0.0.1 only — never exposed to network.
type HookServer struct {
	server      *http.Server
	sessions    *SessionManager
	notifier    *Notifier
	permissions *PermissionStore
	rateLimits  sync.Map // sessionID -> *rateLimiter
}

// rateLimiter tracks request counts per session per second.
type rateLimiter struct {
	mu      sync.Mutex
	count   int
	resetAt time.Time
}

const maxHooksPerSecond = 10

func (rl *rateLimiter) allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	if now.After(rl.resetAt) {
		rl.count = 0
		rl.resetAt = now.Add(time.Second)
	}
	rl.count++
	return rl.count <= maxHooksPerSecond
}

// HookRequest is the JSON body sent by hook scripts.
type HookRequest struct {
	SessionID string          `json:"session_id"`
	Event     string          `json:"event"` // "stop", "permission"
	ExitCode  int             `json:"exit_code,omitempty"`
	Summary   string          `json:"summary,omitempty"`
	GitDiff   string          `json:"git_diff,omitempty"`
	Tool      string          `json:"tool,omitempty"`       // for permission events
	ToolInput json.RawMessage `json:"tool_input,omitempty"` // for permission events
}

// PermissionResponse is the JSON response for permission poll requests.
type PermissionResponse struct {
	ID       string             `json:"id"`
	Decision PermissionDecision `json:"decision"`
	Tool     string             `json:"tool"`
	ExpireAt string             `json:"expires_at"`
}

// NewHookServer creates a hook server. Binds to 127.0.0.1 by default.
// Pass a different bind address (e.g. "0.0.0.0") when running inside Docker
// so that host-side Claude Code can reach the hook endpoint.
func NewHookServer(port int, sessions *SessionManager, notifier *Notifier, opts ...HookServerOption) *HookServer {
	if port == 0 {
		port = 18792
	}

	bind := "127.0.0.1"
	for _, opt := range opts {
		opt(&bind)
	}

	hs := &HookServer{
		sessions:    sessions,
		notifier:    notifier,
		permissions: NewPermissionStore(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/hook", hs.handleHook)
	mux.HandleFunc("/hook/permission/", hs.handlePermissionPoll)
	mux.HandleFunc("/health", hs.handleHealth)

	hs.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", bind, port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	return hs
}

// HookServerOption configures the hook server.
type HookServerOption func(bind *string)

// WithHookBind overrides the default 127.0.0.1 bind address.
func WithHookBind(addr string) HookServerOption {
	return func(bind *string) {
		if addr != "" {
			*bind = addr
		}
	}
}

// Start begins listening. Blocks until context is cancelled or error occurs.
func (hs *HookServer) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", hs.server.Addr)
	if err != nil {
		return fmt.Errorf("hook server listen: %w", err)
	}
	slog.Info("hook server started", "addr", hs.server.Addr)

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		hs.server.Shutdown(shutCtx)
	}()

	if err := hs.server.Serve(ln); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("hook server serve: %w", err)
	}
	return nil
}

// handleHook processes incoming signed webhook requests.
func (hs *HookServer) handleHook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body (limit to 1MB)
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Extract auth headers
	signature := r.Header.Get("X-Hook-Signature")
	tsStr := r.Header.Get("X-Hook-Timestamp")
	sessionID := r.Header.Get("X-Hook-Session")

	if signature == "" || tsStr == "" || sessionID == "" {
		http.Error(w, "missing auth headers (X-Hook-Signature, X-Hook-Timestamp, X-Hook-Session)", http.StatusUnauthorized)
		return
	}

	timestamp, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid timestamp", http.StatusBadRequest)
		return
	}

	// Rate limit per session
	rlVal, _ := hs.rateLimits.LoadOrStore(sessionID, &rateLimiter{resetAt: time.Now().Add(time.Second)})
	rl := rlVal.(*rateLimiter)
	if !rl.allow() {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	// Look up session and verify HMAC
	session, err := hs.sessions.GetSession(r.Context(), sessionID)
	if err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	if err := VerifyHook(session.HookSecret, string(body), signature, timestamp); err != nil {
		slog.Warn("hook auth failed", "session", sessionID, "error", err)
		http.Error(w, "authentication failed", http.StatusForbidden)
		return
	}

	// Parse request
	var req HookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	req.SessionID = sessionID // override with header value (canonical)

	// Dispatch by event type
	switch req.Event {
	case "stop":
		hs.handleStopEvent(r.Context(), session, req)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	case "permission":
		hs.handlePermissionEvent(r.Context(), w, session, req)
	default:
		slog.Warn("unknown hook event", "event", req.Event, "session", sessionID)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}
}

// handleStopEvent processes a session stop notification.
func (hs *HookServer) handleStopEvent(ctx context.Context, session *BridgeSession, req HookRequest) {
	event := StopEvent{
		SessionID:  session.ID,
		ExitCode:   req.ExitCode,
		Summary:    req.Summary,
		GitDiff:    req.GitDiff,
		FinishedAt: time.Now(),
	}

	if hs.notifier != nil {
		hs.notifier.NotifyStop(ctx, session, event)
	}

	slog.Info("hook stop event processed",
		"session", session.ID,
		"exit_code", req.ExitCode,
		"tenant", session.TenantID,
	)
}

// handlePermissionEvent processes a permission request from Claude Code's hook script.
// Returns 202 Accepted — the hook script must poll GET /hook/permission/{id} for the decision.
func (hs *HookServer) handlePermissionEvent(ctx context.Context, w http.ResponseWriter, session *BridgeSession, req HookRequest) {
	if req.Tool == "" {
		http.Error(w, `{"error":"tool field required for permission events"}`, http.StatusBadRequest)
		return
	}

	// Compute request hash for dedup (L2)
	requestHash := ComputeRequestHash(session.ID, req.Tool, req.ToolInput, time.Now())

	// Generate permission request ID
	permID, err := GenerateSessionID(session.TenantID) // reuse ID generator
	if err != nil {
		http.Error(w, `{"error":"internal error generating permission ID"}`, http.StatusInternalServerError)
		return
	}
	permID = "perm:" + permID[3:] // br: -> perm:

	riskLevel := "low"
	if IsHighRisk(req.Tool) {
		riskLevel = "high"
	}

	permReq := &PermissionRequest{
		ID:          permID,
		OwnerID:     session.TenantID,
		SessionID:   session.ID,
		Tool:        req.Tool,
		ToolInput:   req.ToolInput,
		RiskLevel:   riskLevel,
		RequestHash: requestHash,
		ActorID:     session.OwnerActorID,
	}

	created, err := hs.permissions.Create(permReq)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	// Notify approvers via Telegram (non-blocking)
	if hs.notifier != nil {
		hs.notifier.NotifyPermission(ctx, session, created)
	}

	slog.Info("permission request created",
		"permission_id", created.ID,
		"session", session.ID,
		"tool", req.Tool,
		"risk_level", riskLevel,
		"tenant", session.TenantID,
	)

	// 202 Accepted — poll for decision
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(PermissionResponse{
		ID:       created.ID,
		Decision: created.Decision,
		Tool:     created.Tool,
		ExpireAt: created.ExpiresAt.Format(time.RFC3339),
	})
}

// handlePermissionPoll handles GET /hook/permission/{id} — poll for decision.
// No HMAC required for polling — the permission ID itself is the capability token.
// The ID is cryptographically random and only known to the requesting hook script.
func (hs *HookServer) handlePermissionPoll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract permission ID from path: /hook/permission/{id}
	permID := strings.TrimPrefix(r.URL.Path, "/hook/permission/")
	if permID == "" || permID == r.URL.Path {
		http.Error(w, "permission ID required", http.StatusBadRequest)
		return
	}

	perm := hs.permissions.Get(permID) // Get() applies TTL enforcement
	if perm == nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(PermissionResponse{
		ID:       perm.ID,
		Decision: perm.Decision,
		Tool:     perm.Tool,
		ExpireAt: perm.ExpiresAt.Format(time.RFC3339),
	})
}

// Permissions returns the permission store (for Telegram callback handler).
func (hs *HookServer) Permissions() *PermissionStore {
	return hs.permissions
}

// handleHealth returns server health status.
func (hs *HookServer) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","sessions":%d}`, hs.sessions.SessionCount())
}
