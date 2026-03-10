package claudecode

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// CreateSessionOpts holds parameters for creating a new bridge session.
type CreateSessionOpts struct {
	AgentType    AgentProviderType
	ProjectPath  string
	TenantID     string
	UserID       string
	OwnerActorID string
	Channel      string
	ChatID       string
	AgentRole    string // optional SOUL role to inject ("pm", "coder", etc.)
}

// SessionManager manages bridge session lifecycle with multi-tenant isolation.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session // keyed by session ID
	cfg      BridgeConfig
	tmux     *TmuxBridge
	registry *ProviderRegistry
	projects *ProjectRegistry
	pgStore  store.BridgeSessionStore // optional PG persistence (nil in standalone)
	audit    *AuditWriter              // optional audit writer (nil = no audit)

	// Lifetime counters for metrics (OBS-028-4: atomic counters for memory-only tracking)
	lifetimeCreated int64
	lifetimeKilled  int64
}

// BridgeMetrics provides an observable snapshot of bridge session state.
type BridgeMetrics struct {
	ActiveSessions int            `json:"active_sessions"`
	TotalCreated   int            `json:"total_created"`
	TotalKilled    int            `json:"total_killed"`
	ByRiskMode     map[string]int `json:"by_risk_mode"`
	ByAgentRole    map[string]int `json:"by_role"`
	ByChannel      map[string]int `json:"by_channel"` // OBS-028-5: channel is more useful than provider for bridge
}

// NewSessionManager creates a session manager.
// tmux may be nil if tmux is not available (tests).
func NewSessionManager(cfg BridgeConfig, tmux *TmuxBridge) *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
		cfg:      cfg,
		tmux:     tmux,
		registry: NewProviderRegistry(),
		projects: NewProjectRegistry(),
	}
}

// SetStore attaches a PG bridge session store for dual-write persistence.
// Must be called before CreateSession if PG persistence is desired.
func (m *SessionManager) SetStore(s store.BridgeSessionStore) {
	m.pgStore = s
}

// SetAuditWriter attaches an audit writer for lifecycle event logging.
func (m *SessionManager) SetAuditWriter(w *AuditWriter) {
	m.audit = w
}

// emitAudit records a lifecycle event via the audit writer (best-effort).
func (m *SessionManager) emitAudit(bs *BridgeSession, action string, detail map[string]interface{}) {
	if m.audit == nil {
		return
	}
	event := AuditEvent{
		OwnerID:   bs.TenantID,
		SessionID: bs.ID,
		ActorID:   bs.OwnerActorID,
		Action:    action,
		RiskMode:  string(bs.RiskMode),
		Detail:    detail,
	}
	if err := m.audit.Write(event); err != nil {
		slog.Warn("audit write failed", "action", action, "session", bs.ID, "error", err)
	}
}

// LoadFromStore recovers non-stopped sessions from PG on startup.
// Recovered sessions are marked as "disconnected" since tmux is gone after restart.
func (m *SessionManager) LoadFromStore(ctx context.Context) (int, error) {
	if m.pgStore == nil {
		return 0, nil
	}

	recs, err := m.pgStore.ListActive(ctx)
	if err != nil {
		return 0, fmt.Errorf("load bridge sessions from store: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	loaded := 0
	for _, rec := range recs {
		if _, exists := m.sessions[rec.ID]; exists {
			continue // already in memory
		}

		bs := bridgeSessionFromRecord(rec, rec.OwnerID)
		bs.Status = SessionStateDisconnected
		s := NewSession(bs)
		m.sessions[rec.ID] = s
		loaded++
	}

	if loaded > 0 {
		slog.Info("recovered bridge sessions from PG", "count", loaded)
	}
	return loaded, nil
}

// persistSession does a best-effort dual-write of session state to PG.
func (m *SessionManager) persistSession(ctx context.Context, bs *BridgeSession) {
	if m.pgStore == nil {
		return
	}
	rec := bridgeSessionRecordFromSession(bs)
	if err := m.pgStore.Upsert(ctx, rec); err != nil {
		slog.Warn("persist bridge session to PG failed (best-effort)",
			"session", bs.ID, "error", err)
	}
}

// persistStatus does a best-effort status update in PG.
func (m *SessionManager) persistStatus(ctx context.Context, id string, status string, stoppedAt *time.Time) {
	if m.pgStore == nil {
		return
	}
	if err := m.pgStore.UpdateStatus(ctx, id, status, stoppedAt); err != nil {
		slog.Warn("persist bridge session status to PG failed (best-effort)",
			"session", id, "error", err)
	}
}

// persistRiskMode does a best-effort risk mode update in PG.
func (m *SessionManager) persistRiskMode(ctx context.Context, id string, mode string, caps json.RawMessage) {
	if m.pgStore == nil {
		return
	}
	if err := m.pgStore.UpdateRiskMode(ctx, id, mode, caps); err != nil {
		slog.Warn("persist bridge session risk mode to PG failed (best-effort)",
			"session", id, "error", err)
	}
}

// bridgeSessionRecordFromSession converts a BridgeSession to a store record.
func bridgeSessionRecordFromSession(bs *BridgeSession) *store.BridgeSessionRecord {
	capsJSON, _ := json.Marshal(bs.Capabilities)
	approverJSON, _ := json.Marshal(bs.ApproverACL)
	notifyJSON, _ := json.Marshal(bs.NotifyACL)

	return &store.BridgeSessionRecord{
		ID:                   bs.ID,
		OwnerID:              bs.TenantID,
		AgentType:            string(bs.AgentType),
		TmuxTarget:           bs.TmuxTarget,
		ProjectPath:          bs.ProjectPath,
		WorkspaceFingerprint: bs.WorkspaceFingerprint,
		Status:               string(bs.Status),
		RiskMode:             string(bs.RiskMode),
		Capabilities:         capsJSON,
		OwnerActorID:         bs.OwnerActorID,
		ApproverACL:          approverJSON,
		NotifyACL:            notifyJSON,
		UserID:               bs.UserID,
		Channel:              bs.Channel,
		ChatID:               bs.ChatID,
		InteractiveEligible:  bs.InteractiveEligible,
		HookSecret:           bs.HookSecret,
		CreatedAt:            bs.CreatedAt,
		LastActivityAt:       bs.LastActivityAt,
	}
}

// bridgeSessionFromRecord converts a store record back to a BridgeSession.
func bridgeSessionFromRecord(rec *store.BridgeSessionRecord, tenantID string) BridgeSession {
	var caps SessionCapabilities
	if len(rec.Capabilities) > 0 {
		_ = json.Unmarshal(rec.Capabilities, &caps)
	}
	var approverACL, notifyACL []string
	if len(rec.ApproverACL) > 0 {
		_ = json.Unmarshal(rec.ApproverACL, &approverACL)
	}
	if len(rec.NotifyACL) > 0 {
		_ = json.Unmarshal(rec.NotifyACL, &notifyACL)
	}

	return BridgeSession{
		ID:                   rec.ID,
		AgentType:            AgentProviderType(rec.AgentType),
		TmuxTarget:           rec.TmuxTarget,
		ProjectPath:          rec.ProjectPath,
		WorkspaceFingerprint: rec.WorkspaceFingerprint,
		Status:               SessionState(rec.Status),
		RiskMode:             RiskMode(rec.RiskMode),
		Capabilities:         caps,
		OwnerActorID:         rec.OwnerActorID,
		ApproverACL:          approverACL,
		NotifyACL:            notifyACL,
		TenantID:             tenantID,
		UserID:               rec.UserID,
		Channel:              rec.Channel,
		ChatID:               rec.ChatID,
		InteractiveEligible:  rec.InteractiveEligible,
		HookSecret:           rec.HookSecret,
		CreatedAt:            rec.CreatedAt,
		LastActivityAt:       rec.LastActivityAt,
	}
}

// Projects returns the project registry.
func (m *SessionManager) Projects() *ProjectRegistry {
	return m.projects
}

// SoulsDir returns the configured SOUL files directory.
func (m *SessionManager) SoulsDir() string {
	if m.cfg.SoulsDir != "" {
		return m.cfg.SoulsDir
	}
	return "docs/08-collaborate/souls"
}

// CreateSession creates a new bridge session after admission control.
func (m *SessionManager) CreateSession(ctx context.Context, opts CreateSessionOpts) (*BridgeSession, error) {
	if opts.TenantID == "" {
		return nil, fmt.Errorf("tenant ID is required")
	}
	if opts.OwnerActorID == "" {
		return nil, fmt.Errorf("owner actor ID is required")
	}
	if opts.ProjectPath == "" {
		return nil, fmt.Errorf("project path is required")
	}

	// Verify tenant context
	if tid := store.TenantIDFromContext(ctx); tid != "" && tid != opts.TenantID {
		return nil, fmt.Errorf("tenant ID mismatch: context=%s, opts=%s", tid, opts.TenantID)
	}

	// Check provider exists and supports this session type
	adapter, err := m.registry.Get(opts.AgentType)
	if err != nil {
		return nil, err
	}
	providerCaps := adapter.CapabilityProfile()

	// Admission control
	if err := m.checkAdmission(ctx, opts); err != nil {
		return nil, fmt.Errorf("admission denied: %w", err)
	}

	// Generate session ID and hook secret
	sessionID, err := GenerateSessionID(opts.TenantID)
	if err != nil {
		return nil, err
	}
	hookSecret, err := GenerateHookSecret()
	if err != nil {
		return nil, err
	}

	// Compute workspace fingerprint
	fingerprint, err := ComputeWorkspaceFingerprint(opts.ProjectPath, opts.TenantID)
	if err != nil {
		return nil, fmt.Errorf("compute workspace fingerprint: %w", err)
	}

	now := time.Now()
	session := &BridgeSession{
		ID:                   sessionID,
		AgentType:            opts.AgentType,
		TmuxTarget:           buildTmuxTarget(sessionID),
		ProjectPath:          opts.ProjectPath,
		WorkspaceFingerprint: fingerprint,
		Status:               SessionStateActive,
		RiskMode:             RiskModeRead,
		Capabilities:         CapabilitiesForRisk(RiskModeRead),
		OwnerActorID:         opts.OwnerActorID,
		ApproverACL:          []string{},
		NotifyACL:            []string{opts.OwnerActorID},
		TenantID:             opts.TenantID,
		UserID:               opts.UserID,
		Channel:              opts.Channel,
		ChatID:               opts.ChatID,
		HookSecret:           hookSecret,
		InteractiveEligible:  providerCaps.PermissionHooks,
		CreatedAt:            now,
		LastActivityAt:       now,
	}

	// Sprint 21: Compute role defaults once for both risk mode and tool allowlist (CTO-122).
	var roleDefaults RoleDefaultsResult
	// Resolve SOUL persona injection (Sprint 18 — Strategy A/B/C cascade, D10)
	if opts.AgentRole != "" {
		if err := m.resolvePersona(session, opts); err != nil {
			return nil, fmt.Errorf("resolve persona: %w", err)
		}

		// Apply role-aware defaults (UX convenience, NOT security gate).
		// D2 capability model remains the only security boundary. User can always
		// override via /cc risk.
		roleDefaults = RoleDefaults(m.SoulsDir(), opts.AgentRole)
		if roleDefaults.RiskMode != RiskModeRead {
			session.RiskMode = roleDefaults.RiskMode
			session.Capabilities = CapabilitiesForRisk(roleDefaults.RiskMode)
			slog.Info("role-aware default risk applied",
				"session", session.ID,
				"role", opts.AgentRole,
				"default_risk", roleDefaults.RiskMode,
			)
		}
	} else {
		session.PersonaSource = "bare"
	}

	// Build intelligence envelope (Sprint 19 — contract populated from persona resolution)
	session.Intelligence = BuildIntelligenceEnvelope(session)

	// Create tmux session if bridge is available
	if m.tmux != nil {
		if err := m.tmux.CreateSession(ctx, session.TmuxTarget, opts.ProjectPath); err != nil {
			return nil, fmt.Errorf("create tmux session: %w", err)
		}

		// Wire LaunchCommand → tmux SendKeys (D12: adapter builds cmd, tmux executes)
		adapter, _ := m.registry.Get(opts.AgentType)
		hookPort := m.cfg.HookPort
		if hookPort == 0 {
			hookPort = 18792
		}
		launchOpts := LaunchOpts{
			Workdir:   opts.ProjectPath,
			HookURL:   fmt.Sprintf("http://127.0.0.1:%d/hook", hookPort),
			Secret:    hookSecret,
			AgentRole: session.AgentRole,
		}
		// Sprint 21: Pass role-based tool allowlist as UX convenience (reuse roleDefaults)
		if session.AgentRole != "" && roleDefaults.AllowedTools != nil {
			launchOpts.AllowedTools = roleDefaults.AllowedTools
		}
		if session.PersonaSource == "agent_file" {
			launchOpts.AgentFile = filepath.Join(opts.ProjectPath, ".claude", "agents", opts.AgentRole+".md")
		} else if session.PersonaSource == "append_prompt" {
			dirName := strings.ReplaceAll(session.ID, ":", "-")
			launchOpts.PromptFile = filepath.Join(m.standaloneDir(), "sessions", dirName, "soul.md")
		}
		cmd := adapter.LaunchCommand(launchOpts)
		if cmd != nil {
			// Build env exports + command. Unset CLAUDECODE to prevent nested session detection
			// when gateway itself runs inside a Claude Code session.
			var envPrefix string
			envPrefix = "unset CLAUDECODE; "
			for _, e := range cmd.Env {
				envPrefix += fmt.Sprintf("export %s; ", e)
			}
			cmdStr := envPrefix + strings.Join(cmd.Args, " ")
			if err := m.tmux.SendKeys(ctx, session.TmuxTarget, cmdStr); err != nil {
				slog.Warn("launch command sendKeys failed", "session", session.ID, "error", err)
			} else if err := m.tmux.SendEnter(ctx, session.TmuxTarget); err != nil {
				slog.Warn("launch command sendEnter failed", "session", session.ID, "error", err)
			}
		}
	}

	s := NewSession(*session)

	m.mu.Lock()
	m.sessions[sessionID] = s
	m.mu.Unlock()
	atomic.AddInt64(&m.lifetimeCreated, 1)

	// Best-effort PG dual-write
	m.persistSession(ctx, session)
	m.emitAudit(session, "session.created", map[string]interface{}{
		"agent_type":  string(session.AgentType),
		"project":     session.ProjectPath,
		"risk_mode":   string(session.RiskMode),
		"agent_role":  session.AgentRole,
	})

	return session, nil
}

// GetSession returns a session by ID, enforcing tenant isolation.
func (m *SessionManager) GetSession(ctx context.Context, sessionID string) (*BridgeSession, error) {
	m.mu.RLock()
	s, ok := m.sessions[sessionID]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("session %q not found", sessionID)
	}

	data := s.Data()

	// Enforce tenant isolation
	if tid := store.TenantIDFromContext(ctx); tid != "" && tid != data.TenantID {
		return nil, fmt.Errorf("session %q not found", sessionID) // don't leak existence
	}

	return &data, nil
}

// ListSessions returns all sessions for a tenant.
func (m *SessionManager) ListSessions(ctx context.Context, tenantID string) ([]*BridgeSession, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*BridgeSession
	for _, s := range m.sessions {
		data := s.Data()
		if data.TenantID == tenantID {
			d := data
			d.HookSecret = "" // never expose secrets in listings
			result = append(result, &d)
		}
	}
	return result, nil
}

// KillSession terminates a session, enforcing ownership.
func (m *SessionManager) KillSession(ctx context.Context, sessionID, actorID string) error {
	m.mu.RLock()
	s, ok := m.sessions[sessionID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("session %q not found", sessionID)
	}

	data := s.Data()

	// Enforce tenant isolation
	if tid := store.TenantIDFromContext(ctx); tid != "" && tid != data.TenantID {
		return fmt.Errorf("session %q not found", sessionID)
	}

	// Only owner can kill
	if !s.IsOwner(actorID) {
		return fmt.Errorf("only session owner can kill session")
	}

	// Kill tmux session
	if m.tmux != nil {
		_ = m.tmux.KillSession(ctx, data.TmuxTarget) // best effort
	}

	// Clean up Strategy B temp files
	if data.PersonaSource == "append_prompt" {
		m.CleanupSessionDir(data.ID)
	}

	s.ForceStop()
	atomic.AddInt64(&m.lifetimeKilled, 1)

	// Best-effort PG dual-write
	now := time.Now()
	m.persistStatus(ctx, sessionID, string(SessionStateStopped), &now)
	m.emitAudit(&data, "session.killed", map[string]interface{}{
		"killed_by": actorID,
	})

	return nil
}

// UpdateRiskMode changes the risk mode for a session.
func (m *SessionManager) UpdateRiskMode(ctx context.Context, sessionID string, mode RiskMode, actorID string) error {
	m.mu.RLock()
	s, ok := m.sessions[sessionID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("session %q not found", sessionID)
	}

	data := s.Data()

	// Enforce tenant isolation
	if tid := store.TenantIDFromContext(ctx); tid != "" && tid != data.TenantID {
		return fmt.Errorf("session %q not found", sessionID)
	}

	// Interactive mode requires provider capability check (D7 Layer 0)
	if mode == RiskModeInteractive {
		adapter, err := m.registry.Get(data.AgentType)
		if err != nil {
			return err
		}
		caps := adapter.CapabilityProfile()
		if !caps.PermissionHooks {
			return fmt.Errorf("provider %s does not support permission hooks — cannot escalate to interactive (reason_code=provider_capability_missing)", data.AgentType)
		}
	}

	if err := s.UpdateRiskMode(mode, actorID); err != nil {
		return err
	}

	// Best-effort PG dual-write
	newCaps := CapabilitiesForRisk(mode)
	capsJSON, _ := json.Marshal(newCaps)
	m.persistRiskMode(ctx, sessionID, string(mode), capsJSON)
	m.emitAudit(&data, "session.risk_changed", map[string]interface{}{
		"old_risk": string(data.RiskMode),
		"new_risk": string(mode),
		"actor":    actorID,
	})

	return nil
}

// TransitionSession changes session state and drains the message queue
// when transitioning from busy to active/idle (CTO-94).
func (m *SessionManager) TransitionSession(ctx context.Context, sessionID string, newState SessionState) error {
	m.mu.RLock()
	s, ok := m.sessions[sessionID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("session %q not found", sessionID)
	}

	oldData := s.Data()
	wasBusy := oldData.Status == SessionStateBusy

	if err := s.TransitionTo(newState); err != nil {
		return err
	}

	// Drain queued messages when leaving busy state (CTO-94)
	if wasBusy && (newState == SessionStateActive || newState == SessionStateIdle) {
		m.drainQueue(ctx, s)
	}

	// Best-effort PG dual-write
	var stoppedAt *time.Time
	if newState == SessionStateStopped {
		now := time.Now()
		stoppedAt = &now
	}
	m.persistStatus(ctx, sessionID, string(newState), stoppedAt)

	return nil
}

// drainQueue sends all queued messages to the session's tmux pane.
// Best-effort: logs errors but doesn't fail the transition.
func (m *SessionManager) drainQueue(ctx context.Context, s *Session) {
	msgs := s.DrainQueue()
	if len(msgs) == 0 {
		return
	}

	data := s.Data()
	if m.tmux == nil {
		slog.Warn("drainQueue: tmux unavailable, discarding queued messages",
			"session", data.ID, "count", len(msgs))
		return
	}

	for _, msg := range msgs {
		if err := CheckInputSafe(msg); err != nil {
			slog.Warn("drainQueue: skipping unsafe message",
				"session", data.ID, "error", err)
			continue
		}
		if err := m.tmux.SendKeys(ctx, data.TmuxTarget, msg); err != nil {
			slog.Warn("drainQueue: sendKeys failed",
				"session", data.ID, "error", err)
			return // stop on first failure
		}
		if err := m.tmux.SendEnter(ctx, data.TmuxTarget); err != nil {
			slog.Warn("drainQueue: sendEnter failed",
				"session", data.ID, "error", err)
			return
		}
	}
	slog.Info("drainQueue: delivered queued messages",
		"session", data.ID, "count", len(msgs))
}

// SendText relays free-text input to a session's tmux pane.
// Enforces: InputMode=free_text, session not stopped/busy, input sanitization.
func (m *SessionManager) SendText(ctx context.Context, sessionID, text, actorID string) error {
	m.mu.RLock()
	s, ok := m.sessions[sessionID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("session %q not found", sessionID)
	}

	data := s.Data()

	// Tenant isolation
	if tid := store.TenantIDFromContext(ctx); tid != "" && tid != data.TenantID {
		return fmt.Errorf("session %q not found", sessionID)
	}

	// Capability gate: InputMode must be free_text (D2 axis 1)
	if err := CheckInputAllowed(data.Capabilities, text); err != nil {
		return err
	}

	// State gate: only active or idle sessions accept input
	if data.Status == SessionStateStopped {
		return fmt.Errorf("session %q is stopped (reason_code=session_stopped)", sessionID)
	}
	if data.Status == SessionStateBusy {
		// Queue the message instead of rejecting
		return s.EnqueueMessage(text)
	}
	if data.Status == SessionStateError {
		return fmt.Errorf("session %q is in error state (reason_code=session_error)", sessionID)
	}

	// Input sanitization
	if err := CheckInputSafe(text); err != nil {
		return fmt.Errorf("input blocked: %w", err)
	}

	// Send to tmux
	if m.tmux == nil {
		return fmt.Errorf("tmux not available (reason_code=tmux_unavailable)")
	}

	// Sprint 20B: Prepend turn context if set (consumed once)
	if tc := s.ConsumeTurnContext(); tc != nil {
		prefix := FormatTurnContextMarkdown(tc)
		if prefix != "" {
			text = prefix + "\n" + text
		}
	}

	if err := m.tmux.SendKeys(ctx, data.TmuxTarget, text); err != nil {
		return fmt.Errorf("send to tmux: %w", err)
	}

	// Send Enter to execute
	if err := m.tmux.SendEnter(ctx, data.TmuxTarget); err != nil {
		return fmt.Errorf("send enter: %w", err)
	}

	s.Touch()
	return nil
}

// Maximum lengths for turn context fields (CTO-120).
const (
	maxContextFieldLen = 500  // per-field cap (single goal/blocker/hint)
	maxContextTotalLen = 2000 // total accumulated context cap
)

// SetContext sets turn context on a session (Sprint 20B).
// The context is consumed once: prepended to the next sendKeys message.
func (m *SessionManager) SetContext(ctx context.Context, sessionID string, tc *TurnContext, actorID string) error {
	m.mu.RLock()
	s, ok := m.sessions[sessionID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("session %q not found", sessionID)
	}

	data := s.Data()

	// Tenant isolation
	if tid := store.TenantIDFromContext(ctx); tid != "" && tid != data.TenantID {
		return fmt.Errorf("session %q not found", sessionID)
	}

	// CTO-120: Per-field length validation
	for _, f := range tc.SprintGoals {
		if len(f) > maxContextFieldLen {
			return fmt.Errorf("context field too long (%d chars, max %d)", len(f), maxContextFieldLen)
		}
	}
	for _, f := range tc.Blockers {
		if len(f) > maxContextFieldLen {
			return fmt.Errorf("context field too long (%d chars, max %d)", len(f), maxContextFieldLen)
		}
	}
	for _, f := range tc.FixHints {
		if len(f) > maxContextFieldLen {
			return fmt.Errorf("context field too long (%d chars, max %d)", len(f), maxContextFieldLen)
		}
	}

	// CTO-118: Sanitize context content before storing (prevents sendKeys injection)
	var combined strings.Builder
	for _, s := range tc.SprintGoals {
		combined.WriteString(s)
		combined.WriteByte(' ')
	}
	for _, s := range tc.Blockers {
		combined.WriteString(s)
		combined.WriteByte(' ')
	}
	for _, s := range tc.FixHints {
		combined.WriteString(s)
		combined.WriteByte(' ')
	}
	if err := CheckInputSafe(combined.String()); err != nil {
		return fmt.Errorf("context blocked by sanitizer: %w", err)
	}

	// CTO-120: Estimate total size after merge by peeking at existing + new
	existing := s.PeekTurnContext()
	preview := &TurnContext{}
	if existing != nil {
		preview.SprintGoals = append(preview.SprintGoals, existing.SprintGoals...)
		preview.Blockers = append(preview.Blockers, existing.Blockers...)
		preview.FixHints = append(preview.FixHints, existing.FixHints...)
	}
	preview.SprintGoals = append(preview.SprintGoals, tc.SprintGoals...)
	preview.Blockers = append(preview.Blockers, tc.Blockers...)
	preview.FixHints = append(preview.FixHints, tc.FixHints...)
	if rendered := FormatTurnContextMarkdown(preview); len(rendered) > maxContextTotalLen {
		return fmt.Errorf("total context too large (%d chars, max %d) — use /cc context clear first", len(rendered), maxContextTotalLen)
	}

	s.SetTurnContext(tc)
	return nil
}

// ClearContext removes all pending turn context from a session.
func (m *SessionManager) ClearContext(ctx context.Context, sessionID, actorID string) error {
	m.mu.RLock()
	s, ok := m.sessions[sessionID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("session %q not found", sessionID)
	}

	data := s.Data()
	if tid := store.TenantIDFromContext(ctx); tid != "" && tid != data.TenantID {
		return fmt.Errorf("session %q not found", sessionID)
	}

	s.ClearTurnContext()
	return nil
}

// CaptureOutput captures the last N lines from a session's tmux pane.
func (m *SessionManager) CaptureOutput(ctx context.Context, sessionID, actorID string, lines int) (string, error) {
	m.mu.RLock()
	s, ok := m.sessions[sessionID]
	m.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("session %q not found", sessionID)
	}

	data := s.Data()

	// Tenant isolation
	if tid := store.TenantIDFromContext(ctx); tid != "" && tid != data.TenantID {
		return "", fmt.Errorf("session %q not found", sessionID)
	}

	// Capability gate
	capLines, err := CheckCaptureAllowed(data.Capabilities)
	if err != nil {
		return "", err
	}
	if lines <= 0 || lines > capLines {
		lines = capLines
	}

	if data.Status == SessionStateStopped {
		return "", fmt.Errorf("session %q is stopped", sessionID)
	}

	if m.tmux == nil {
		return "", fmt.Errorf("tmux not available")
	}

	output, err := m.tmux.CapturePane(ctx, data.TmuxTarget, lines)
	if err != nil {
		return "", err
	}

	// Redact secrets from output
	redactor := NewOutputRedactor()
	output = redactor.Redact(output, data.Capabilities.RedactHeavy)

	s.Touch()
	return output, nil
}

// CleanupStopped removes sessions in stopped state older than maxAge.
func (m *SessionManager) CleanupStopped(maxAge time.Duration) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0
	for id, s := range m.sessions {
		data := s.Data()
		if data.Status == SessionStateStopped && data.LastActivityAt.Before(cutoff) {
			delete(m.sessions, id)
			removed++
		}
	}

	// Best-effort PG cleanup
	if m.pgStore != nil && removed > 0 {
		if n, err := m.pgStore.DeleteOlderThan(context.Background(), cutoff); err != nil {
			slog.Warn("PG bridge session cleanup failed (best-effort)", "error", err)
		} else if n > 0 {
			slog.Info("cleaned up old bridge sessions from PG", "count", n)
		}
	}

	return removed
}

// SessionCount returns the total number of sessions (for diagnostics).
func (m *SessionManager) SessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// allSessions returns a snapshot of all sessions (no tenant filtering).
// INTERNAL: Used by HealthMonitor only. Do not call from request handlers.
func (m *SessionManager) allSessions() []BridgeSession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]BridgeSession, 0, len(m.sessions))
	for _, s := range m.sessions {
		result = append(result, s.Data())
	}
	return result
}

// Metrics returns an observable snapshot of bridge session state.
func (m *SessionManager) Metrics() BridgeMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics := BridgeMetrics{
		ByRiskMode:  make(map[string]int),
		ByAgentRole: make(map[string]int),
		ByChannel:   make(map[string]int),
	}

	for _, s := range m.sessions {
		d := s.Data()
		if d.Status != SessionStateStopped {
			metrics.ActiveSessions++
		}
		risk := string(d.RiskMode)
		if risk == "" {
			risk = "read"
		}
		metrics.ByRiskMode[risk]++

		role := d.AgentRole
		if role == "" {
			role = "(bare)"
		}
		metrics.ByAgentRole[role]++

		ch := d.Channel
		if ch == "" {
			ch = "(unknown)"
		}
		metrics.ByChannel[ch]++
	}

	metrics.TotalCreated = int(atomic.LoadInt64(&m.lifetimeCreated))
	metrics.TotalKilled = int(atomic.LoadInt64(&m.lifetimeKilled))

	return metrics
}

// checkAdmission validates resource limits before creating a session.
func (m *SessionManager) checkAdmission(_ context.Context, opts CreateSessionOpts) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	admission := m.cfg.Admission

	// Count existing sessions
	totalActive := 0
	agentCount := 0
	tenantCount := 0
	for _, s := range m.sessions {
		data := s.Data()
		if data.Status == SessionStateStopped {
			continue
		}
		totalActive++
		if data.AgentType == opts.AgentType {
			agentCount++
		}
		if data.TenantID == opts.TenantID {
			tenantCount++
		}
		if admission.PerProjectSingleton && data.ProjectPath == opts.ProjectPath && data.TenantID == opts.TenantID {
			return fmt.Errorf("project singleton: session already exists for project %q", opts.ProjectPath)
		}
	}

	if agentCount >= admission.MaxSessionsPerAgent {
		return fmt.Errorf("max sessions per agent (%d) reached for %s", admission.MaxSessionsPerAgent, opts.AgentType)
	}
	if totalActive >= admission.MaxTotalSessions {
		return fmt.Errorf("max total sessions (%d) reached", admission.MaxTotalSessions)
	}
	if tenantCount >= admission.PerTenantSessionCap {
		return fmt.Errorf("per-tenant session cap (%d) reached for tenant %s", admission.PerTenantSessionCap, opts.TenantID)
	}

	// Host resource checks
	if admission.MaxCPUPercent > 0 {
		cpuPercent := estimateCPUUsage()
		if cpuPercent > admission.MaxCPUPercent {
			return fmt.Errorf("host CPU usage %.1f%% exceeds threshold %.1f%%", cpuPercent, admission.MaxCPUPercent)
		}
	}
	if admission.MaxMemoryPercent > 0 {
		memPercent := estimateMemoryUsage()
		if memPercent > admission.MaxMemoryPercent {
			return fmt.Errorf("host memory usage %.1f%% exceeds threshold %.1f%%", memPercent, admission.MaxMemoryPercent)
		}
	}

	return nil
}

// buildTmuxTarget extracts tenant hash and random parts from a session ID
// to build a tmux session name. Session ID format: "br:{tenant8}:{rand8}".
func buildTmuxTarget(sessionID string) string {
	parts := strings.SplitN(sessionID, ":", 3)
	if len(parts) == 3 {
		return BuildSessionName(parts[1], parts[2])
	}
	// Fallback: use full ID (should never happen with valid IDs)
	return BuildSessionName(sessionID, sessionID)
}

// resolvePersona loads a SOUL file and determines the injection strategy (D10).
// Strategy A: .claude/agents/{role}.md exists → set AgentFile (--agent flag)
// Strategy B: no agent file → write SOUL to temp file → set PromptFile (--append-system-prompt-file)
// Strategy C: implicit when AgentRole is empty (handled by caller).
func (m *SessionManager) resolvePersona(session *BridgeSession, opts CreateSessionOpts) error {
	soulsDir := m.cfg.SoulsDir
	if soulsDir == "" {
		soulsDir = "docs/08-collaborate/souls"
	}

	soul, err := LoadSOUL(soulsDir, opts.AgentRole)
	if err != nil {
		return err
	}

	session.AgentRole = opts.AgentRole
	session.SoulTemplateHash = soul.ContentHash

	// Strategy A: check for native agent file
	agentFile := filepath.Join(opts.ProjectPath, ".claude", "agents", opts.AgentRole+".md")
	if fileExists(agentFile) {
		agentHash, hashErr := HashFileContent(agentFile)
		if hashErr != nil {
			return fmt.Errorf("hash agent file: %w", hashErr)
		}
		session.PersonaSourceHash = agentHash
		session.PersonaSource = "agent_file"

		// CTO-M4 + CTO-105: Stale detection via .soul-hash sidecar (warning, not block).
		// Compare SOUL file hash against the hash recorded at install-agents time,
		// NOT against the agent file content (which has a different header/footer).
		soulHashFile := filepath.Join(opts.ProjectPath, ".claude", "agents", opts.AgentRole+".soul-hash")
		if installedHash, err := os.ReadFile(soulHashFile); err == nil {
			if string(installedHash) != soul.ContentHash {
				slog.Warn("agent file may be stale — run mtclaw bridge install-agents to update",
					"session", session.ID,
					"role", opts.AgentRole,
					"soul_hash", soul.ContentHash,
					"installed_hash", string(installedHash),
				)
			}
		}
		// If no .soul-hash file exists (user-created agent), skip stale check silently
		return nil
	}

	// Strategy B: write SOUL body to temp file
	// CTO-B2: Sanitize sessionID colons for macOS HFS+
	dirName := strings.ReplaceAll(session.ID, ":", "-")
	sessionDir := filepath.Join(m.standaloneDir(), "sessions", dirName)
	if err := os.MkdirAll(sessionDir, 0700); err != nil {
		return fmt.Errorf("create session dir: %w", err)
	}

	promptFile := filepath.Join(sessionDir, "soul.md")
	if err := os.WriteFile(promptFile, []byte(soul.Body), 0600); err != nil {
		return fmt.Errorf("write prompt file: %w", err)
	}

	promptHash, _ := HashFileContent(promptFile)
	session.PersonaSourceHash = promptHash
	session.PersonaSource = "append_prompt"

	slog.Info("persona resolved via Strategy B (append-system-prompt-file)",
		"session", session.ID,
		"role", opts.AgentRole,
		"prompt_file", promptFile,
	)
	return nil
}

// standaloneDir returns the base directory for session data.
func (m *SessionManager) standaloneDir() string {
	if m.cfg.StandaloneDir != "" {
		return m.cfg.StandaloneDir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/mtclaw"
	}
	return filepath.Join(home, ".mtclaw")
}

// CleanupSessionDir removes the temp directory for a session (Strategy B cleanup).
func (m *SessionManager) CleanupSessionDir(sessionID string) {
	dirName := strings.ReplaceAll(sessionID, ":", "-")
	sessionDir := filepath.Join(m.standaloneDir(), "sessions", dirName)
	if err := os.RemoveAll(sessionDir); err != nil {
		slog.Warn("cleanup session dir failed", "dir", sessionDir, "error", err)
	}
}

// estimateCPUUsage returns a rough estimate of CPU usage.
// Uses Go runtime stats as a proxy — not precise, but lightweight.
func estimateCPUUsage() float64 {
	// NumGoroutine as a rough proxy for CPU pressure.
	// Real implementation should use /proc/stat or cgroups.
	goroutines := runtime.NumGoroutine()
	numCPU := runtime.NumCPU()
	if numCPU == 0 {
		numCPU = 1
	}
	// Rough estimate: if goroutines >> CPUs, we're under pressure
	ratio := float64(goroutines) / float64(numCPU*10)
	if ratio > 1.0 {
		ratio = 1.0
	}
	return ratio * 100
}

// estimateMemoryUsage returns a rough estimate of memory usage percentage.
func estimateMemoryUsage() float64 {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	// Use Sys (total memory obtained from OS) as a rough signal.
	// Real implementation should read /proc/meminfo or cgroups.
	sysMB := float64(stats.Sys) / (1024 * 1024)
	// Assume 8GB system as baseline; adjust when real /proc/meminfo is wired
	totalMB := float64(8 * 1024)
	return (sysMB / totalMB) * 100
}
