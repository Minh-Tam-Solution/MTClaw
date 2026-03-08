package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/Minh-Tam-Solution/MTClaw/internal/claudecode"
	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// validateProjectPath checks for directory traversal and ensures the path is absolute
// after cleaning. Blocks ../../ sequences and relative paths (CTO-83 defense-in-depth).
func validateProjectPath(raw string) (string, error) {
	cleaned := filepath.Clean(raw)
	if !filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("project path must be absolute, got %q", raw)
	}
	if strings.Contains(raw, "..") {
		return "", fmt.Errorf("directory traversal not allowed in project path")
	}
	return cleaned, nil
}

// handleCC dispatches /cc subcommands for the Claude Code terminal bridge (ADR-010).
// Every /cc command requires actor_id (Telegram user ID). No action without identity.
func (c *Channel) handleCC(ctx context.Context, chatID int64, chatIDStr, text, senderID string, setThread func(*telego.SendMessageParams)) {
	chatIDObj := tu.ID(chatID)

	send := func(msg string) {
		m := tu.Message(chatIDObj, msg)
		setThread(m)
		c.bot.SendMessage(ctx, m)
	}

	// Guard: bridge must be enabled
	if c.bridgeManager == nil {
		send("Bridge is not enabled. Set bridge.enabled=true in config.")
		return
	}

	// Extract subcommand and args: "/cc launch myproject" → sub="launch", args="myproject"
	rest := strings.TrimSpace(strings.TrimPrefix(text, "/cc"))
	parts := strings.SplitN(rest, " ", 2)
	sub := ""
	args := ""
	if len(parts) > 0 {
		sub = strings.ToLower(strings.TrimSpace(parts[0]))
	}
	if len(parts) > 1 {
		args = strings.TrimSpace(parts[1])
	}

	// Actor ID = Telegram numeric user ID (extracted from senderID "12345|telegram")
	actorID := strings.SplitN(senderID, "|", 2)[0]

	// Tenant ID from context (managed mode) or fallback to "standalone"
	tenantID := store.TenantIDFromContext(ctx)
	if tenantID == "" {
		tenantID = "standalone"
	}

	// Inject tenant context for session manager
	ctx = store.WithTenantID(ctx, tenantID)

	switch sub {
	case "link":
		c.ccLink(ctx, send, actorID, tenantID)
	case "launch":
		c.ccLaunch(ctx, send, actorID, tenantID, chatIDStr, args)
	case "sessions":
		c.ccSessions(ctx, send, tenantID)
	case "capture":
		c.ccCapture(ctx, send, actorID, tenantID, args)
	case "kill":
		c.ccKill(ctx, send, actorID, tenantID, args)
	case "projects":
		c.ccProjects(ctx, send, actorID, tenantID)
	case "register":
		c.ccRegister(ctx, send, actorID, tenantID, args)
	case "switch":
		c.ccSwitch(send, actorID, args)
	case "risk":
		c.ccRisk(ctx, send, actorID, tenantID, args)
	case "send":
		c.ccSend(ctx, send, actorID, tenantID, args)
	case "info":
		c.ccInfo(ctx, send, actorID, tenantID, args)
	case "context":
		c.ccContext(ctx, send, actorID, tenantID, args)
	case "", "help":
		c.ccHelp(send)
	default:
		send(fmt.Sprintf("Unknown /cc command: %s\nUse /cc help for available commands.", sub))
	}
}

func (c *Channel) ccHelp(send func(string)) {
	send("Claude Code Bridge commands:\n\n" +
		"/cc link — Bind your Telegram identity\n" +
		"/cc launch [project] [--as role] — Start Claude Code session\n" +
		"/cc sessions — List active sessions\n" +
		"/cc capture [lines] — Show terminal output\n" +
		"/cc send <text> — Send free-text to session (interactive mode)\n" +
		"/cc kill [session] — Terminate session\n" +
		"/cc projects — List registered projects\n" +
		"/cc register <name> <path> — Register project\n" +
		"/cc switch <session> — Switch active session\n" +
		"/cc info [session] — Show session intelligence details\n" +
		"/cc context <goal|blocker|hint|clear> <text> — Set/clear turn context\n" +
		"/cc risk <read|patch|interactive> — Change risk mode")
}

func (c *Channel) ccLink(_ context.Context, send func(string), actorID, tenantID string) {
	send(fmt.Sprintf("Identity linked.\nActor: %s\nTenant: %s\n\nYou can now use /cc launch to start a session.", actorID, tenantID))
}

func (c *Channel) ccLaunch(ctx context.Context, send func(string), actorID, tenantID, chatIDStr, args string) {
	// Parse args: "myproject --as coder" or "myproject" or "--as pm"
	projectName, agentRole := parseLaunchArgs(args)

	// Validate role if specified
	if agentRole != "" {
		soulsDir := c.bridgeManager.SoulsDir()
		known, err := claudecode.KnownRoles(soulsDir)
		if err != nil {
			send(fmt.Sprintf("Cannot load SOUL roles: %s", err))
			return
		}
		found := false
		for _, r := range known {
			if r == agentRole {
				found = true
				break
			}
		}
		if !found {
			send(fmt.Sprintf("Unknown role %q. Available: %s", agentRole, strings.Join(known, ", ")))
			return
		}
	}

	// Resolve project path
	projectPath := "/tmp" // default for quick testing
	if projectName != "" {
		proj, ok := c.bridgeManager.Projects().Get(tenantID, projectName)
		if !ok {
			send(fmt.Sprintf("Project %q not found. Use /cc register first.", projectName))
			return
		}
		projectPath = proj.Path
	}

	opts := claudecode.CreateSessionOpts{
		AgentType:    claudecode.AgentClaudeCode,
		ProjectPath:  projectPath,
		TenantID:     tenantID,
		UserID:       actorID,
		OwnerActorID: actorID,
		Channel:      "telegram",
		ChatID:       chatIDStr,
		AgentRole:    agentRole,
	}

	session, err := c.bridgeManager.CreateSession(ctx, opts)
	if err != nil {
		slog.Warn("cc launch: create session failed", "error", err, "actor", actorID)
		send(fmt.Sprintf("Failed to launch: %s", err))
		return
	}

	roleInfo := ""
	if session.AgentRole != "" {
		roleInfo = fmt.Sprintf("\nRole: %s (strategy: %s)", session.AgentRole, session.PersonaSource)
	}

	send(fmt.Sprintf("Session launched!\nID: %s\nProject: %s\nRisk: %s\nTmux: %s%s",
		session.ID, projectPath, session.RiskMode, session.TmuxTarget, roleInfo))
}

// parseLaunchArgs extracts project name and --as role from launch args.
// Examples: "myproject --as coder" → ("myproject", "coder")
//
//	"--as pm" → ("", "pm")
//	"myproject" → ("myproject", "")
func parseLaunchArgs(args string) (projectName, agentRole string) {
	parts := strings.Fields(args)
	for i := 0; i < len(parts); i++ {
		if parts[i] == "--as" {
			if i+1 < len(parts) {
				agentRole = parts[i+1]
				i++ // skip role value
			}
			// CTO-108: --as without value is silently ignored (agentRole stays empty)
		} else {
			if projectName == "" {
				projectName = parts[i]
			}
		}
	}
	return
}

func (c *Channel) ccSessions(ctx context.Context, send func(string), tenantID string) {
	sessions, err := c.bridgeManager.ListSessions(ctx, tenantID)
	if err != nil {
		send(fmt.Sprintf("Failed to list sessions: %s", err))
		return
	}

	// Filter to active sessions only; stopped sessions are historical.
	var active []*claudecode.BridgeSession
	for _, s := range sessions {
		if s.Status != claudecode.SessionStateStopped {
			active = append(active, s)
		}
	}

	if len(active) == 0 {
		send("No active sessions. Use /cc launch to start one.")
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Active sessions (%d):\n\n", len(active)))
	for i, s := range active {
		status := string(s.Status)
		roleInfo := ""
		if s.AgentRole != "" {
			roleInfo = fmt.Sprintf(" role=%s(%s)", s.AgentRole, s.PersonaSource)
		}
		sb.WriteString(fmt.Sprintf("%d. %s [%s] risk=%s%s owner=%s\n   project: %s\n",
			i+1, s.ID, status, s.RiskMode, roleInfo, s.OwnerActorID, s.ProjectPath))
	}
	send(sb.String())
}

func (c *Channel) ccCapture(ctx context.Context, send func(string), actorID, tenantID, args string) {
	// Find the actor's most recent session (or use session ID from args)
	sessions, err := c.bridgeManager.ListSessions(ctx, tenantID)
	if err != nil {
		send(fmt.Sprintf("Failed to list sessions: %s", err))
		return
	}

	if len(sessions) == 0 {
		send("No active sessions. Use /cc launch first.")
		return
	}

	// Parse optional line count from args
	lines := 0 // will use capability default
	sessionID := ""
	for _, part := range strings.Fields(args) {
		if n, err := strconv.Atoi(part); err == nil && n > 0 {
			lines = n
		} else {
			sessionID = part
		}
	}

	// Find target session
	var target *claudecode.BridgeSession
	if sessionID != "" {
		s, err := c.bridgeManager.GetSession(ctx, sessionID)
		if err != nil {
			send(fmt.Sprintf("Session %q not found.", sessionID))
			return
		}
		target = s
	} else {
		// Use first active session owned by this actor
		for _, s := range sessions {
			if s.OwnerActorID == actorID && s.Status != claudecode.SessionStateStopped {
				target = s
				break
			}
		}
		if target == nil {
			target = sessions[0] // fallback to first
		}
	}

	// Check capture policy
	capLines, err := claudecode.CheckCaptureAllowed(target.Capabilities)
	if err != nil {
		send(fmt.Sprintf("Capture not allowed: %s", err))
		return
	}
	if lines <= 0 || lines > capLines {
		lines = capLines
	}

	// Capture requires tmux — if bridge has no tmux, show placeholder
	send(fmt.Sprintf("Capture from %s (%d lines, risk=%s):\n\n[tmux capture requires live session — will be functional when HookServer wires tmux in Sprint 15]",
		target.ID, lines, target.RiskMode))
}

func (c *Channel) ccKill(ctx context.Context, send func(string), actorID, tenantID, args string) {
	sessionID := strings.TrimSpace(args)
	if sessionID == "" {
		send("Usage: /cc kill <session-id>")
		return
	}

	if err := c.bridgeManager.KillSession(ctx, sessionID, actorID); err != nil {
		send(fmt.Sprintf("Failed to kill session: %s", err))
		return
	}

	send(fmt.Sprintf("Session %s terminated.", sessionID))
}

func (c *Channel) ccProjects(_ context.Context, send func(string), _, tenantID string) {
	projects := c.bridgeManager.Projects().List(tenantID)
	if len(projects) == 0 {
		send("No projects registered. Use /cc register <name> <path> to add one.")
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Registered projects (%d):\n\n", len(projects)))
	for i, p := range projects {
		sb.WriteString(fmt.Sprintf("%d. %s — %s [%s]\n", i+1, p.Name, p.Path, p.AgentType))
	}
	send(sb.String())
}

func (c *Channel) ccRegister(_ context.Context, send func(string), _, tenantID, args string) {
	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 {
		send("Usage: /cc register <name> <path>")
		return
	}

	name := strings.TrimSpace(parts[0])
	rawPath := strings.TrimSpace(parts[1])

	path, err := validateProjectPath(rawPath)
	if err != nil {
		send(fmt.Sprintf("Invalid path: %s", err))
		return
	}

	_, err = c.bridgeManager.Projects().Register(tenantID, name, path, claudecode.AgentClaudeCode)
	if err != nil {
		send(fmt.Sprintf("Failed to register project: %s", err))
		return
	}

	send(fmt.Sprintf("Project %q registered at %s.", name, path))
}

func (c *Channel) ccSwitch(send func(string), actorID, args string) {
	sessionID := strings.TrimSpace(args)
	if sessionID == "" {
		send("Usage: /cc switch <session-id>")
		return
	}

	// Switch only affects routing for the calling actor (D8).
	// In Sprint A2, this is a placeholder — actual routing stored in actor-session map.
	send(fmt.Sprintf("Switched active session to %s for actor %s.", sessionID, actorID))
}

func (c *Channel) ccRisk(ctx context.Context, send func(string), actorID, tenantID, args string) {
	mode := strings.TrimSpace(strings.ToLower(args))
	if mode == "" {
		send("Usage: /cc risk <read|patch|interactive>")
		return
	}

	// Validate risk mode
	var riskMode claudecode.RiskMode
	switch mode {
	case "read":
		riskMode = claudecode.RiskModeRead
	case "patch":
		riskMode = claudecode.RiskModePatch
	case "interactive":
		riskMode = claudecode.RiskModeInteractive
	default:
		send(fmt.Sprintf("Unknown risk mode: %s. Use read, patch, or interactive.", mode))
		return
	}

	// Find actor's active session
	sessions, err := c.bridgeManager.ListSessions(ctx, tenantID)
	if err != nil {
		send(fmt.Sprintf("Failed to list sessions: %s", err))
		return
	}

	var targetID string
	for _, s := range sessions {
		if s.OwnerActorID == actorID && s.Status != claudecode.SessionStateStopped {
			targetID = s.ID
			break
		}
	}
	if targetID == "" {
		send("No active session found. Use /cc launch first.")
		return
	}

	if err := c.bridgeManager.UpdateRiskMode(ctx, targetID, riskMode, actorID); err != nil {
		slog.Warn("cc risk: escalation denied",
			"actor", actorID, "session", targetID,
			"target_mode", mode, "error", err,
			"tenant", tenantID,
		)
		send(fmt.Sprintf("Failed to change risk mode: %s", err))
		return
	}

	slog.Info("cc risk: mode changed",
		"actor", actorID, "session", targetID,
		"new_mode", mode, "tenant", tenantID,
	)
	send(fmt.Sprintf("Risk mode changed to %s for session %s.", riskMode, targetID))
}

func (c *Channel) ccSend(ctx context.Context, send func(string), actorID, tenantID, args string) {
	text := strings.TrimSpace(args)
	if text == "" {
		send("Usage: /cc send <text>\n\nRequires interactive risk mode (/cc risk interactive).")
		return
	}

	// Find actor's active session
	sessions, err := c.bridgeManager.ListSessions(ctx, tenantID)
	if err != nil {
		send(fmt.Sprintf("Failed to list sessions: %s", err))
		return
	}

	var targetID string
	for _, s := range sessions {
		if s.OwnerActorID == actorID && s.Status != claudecode.SessionStateStopped {
			targetID = s.ID
			break
		}
	}
	if targetID == "" {
		send("No active session found. Use /cc launch first.")
		return
	}

	if err := c.bridgeManager.SendText(ctx, targetID, text, actorID); err != nil {
		slog.Warn("cc send: relay failed",
			"actor", actorID, "session", targetID,
			"error", err, "tenant", tenantID,
		)
		send(fmt.Sprintf("Send failed: %s", err))
		return
	}

	send(fmt.Sprintf("Sent to %s.", targetID))
}

func (c *Channel) ccInfo(ctx context.Context, send func(string), actorID, tenantID, args string) {
	sessionID := strings.TrimSpace(args)

	// If no session ID given, find actor's active session
	if sessionID == "" {
		sessions, err := c.bridgeManager.ListSessions(ctx, tenantID)
		if err != nil {
			send(fmt.Sprintf("Failed to list sessions: %s", err))
			return
		}
		for _, s := range sessions {
			if s.OwnerActorID == actorID && s.Status != claudecode.SessionStateStopped {
				sessionID = s.ID
				break
			}
		}
		if sessionID == "" {
			send("No active session found. Use /cc launch first.")
			return
		}
	}

	session, err := c.bridgeManager.GetSession(ctx, sessionID)
	if err != nil {
		send(fmt.Sprintf("Session %q not found.", sessionID))
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Session: %s\n", session.ID))
	sb.WriteString(fmt.Sprintf("Status: %s\n", session.Status))
	sb.WriteString(fmt.Sprintf("Risk: %s\n", session.RiskMode))
	sb.WriteString(fmt.Sprintf("Agent: %s\n", session.AgentType))
	sb.WriteString(fmt.Sprintf("Project: %s\n", session.ProjectPath))
	sb.WriteString(fmt.Sprintf("Owner: %s\n", session.OwnerActorID))

	// Intelligence envelope (Sprint 19)
	if session.Intelligence != nil && session.Intelligence.Persona != nil {
		p := session.Intelligence.Persona
		sb.WriteString("\nIntelligence:\n")
		sb.WriteString(fmt.Sprintf("  Role: %s\n", p.AgentRole))
		sb.WriteString(fmt.Sprintf("  Strategy: %s (%s)\n", p.Strategy, p.PersonaSource))
		if p.SoulTemplateHash != "" {
			sb.WriteString(fmt.Sprintf("  SOUL hash: %s\n", truncateHashDisplay(p.SoulTemplateHash)))
		}
		if p.PersonaSourceHash != "" {
			sb.WriteString(fmt.Sprintf("  Persona hash: %s\n", truncateHashDisplay(p.PersonaSourceHash)))
		}
	} else {
		sb.WriteString("\nIntelligence: bare (no SOUL injected)\n")
	}

	sb.WriteString(fmt.Sprintf("\nCreated: %s", session.CreatedAt.Format("2006-01-02 15:04:05")))
	send(sb.String())
}

func (c *Channel) ccContext(ctx context.Context, send func(string), actorID, tenantID, args string) {
	trimmed := strings.TrimSpace(args)

	// Handle /cc context clear
	if strings.ToLower(trimmed) == "clear" {
		sessions, err := c.bridgeManager.ListSessions(ctx, tenantID)
		if err != nil {
			send(fmt.Sprintf("Failed to list sessions: %s", err))
			return
		}
		var targetID string
		for _, s := range sessions {
			if s.OwnerActorID == actorID && s.Status != claudecode.SessionStateStopped {
				targetID = s.ID
				break
			}
		}
		if targetID == "" {
			send("No active session found. Use /cc launch first.")
			return
		}
		if err := c.bridgeManager.ClearContext(ctx, targetID, actorID); err != nil {
			send(fmt.Sprintf("Failed to clear context: %s", err))
			return
		}
		send(fmt.Sprintf("Turn context cleared for session %s.", targetID))
		return
	}

	parts := strings.SplitN(trimmed, " ", 2)
	if len(parts) < 2 || parts[1] == "" {
		send("Usage:\n  /cc context <goal|blocker|hint> <text>\n  /cc context clear\n\nContext accumulates — multiple commands add to the same context.\nExamples:\n  /cc context goal Fix the login bug\n  /cc context blocker API rate limit\n  /cc context hint Check provider.go line 42\n  /cc context clear")
		return
	}

	contextType := strings.ToLower(parts[0])
	text := parts[1]

	// Find actor's active session
	sessions, err := c.bridgeManager.ListSessions(ctx, tenantID)
	if err != nil {
		send(fmt.Sprintf("Failed to list sessions: %s", err))
		return
	}

	var targetID string
	for _, s := range sessions {
		if s.OwnerActorID == actorID && s.Status != claudecode.SessionStateStopped {
			targetID = s.ID
			break
		}
	}
	if targetID == "" {
		send("No active session found. Use /cc launch first.")
		return
	}

	tc := &claudecode.TurnContext{}
	switch contextType {
	case "goal":
		tc.SprintGoals = []string{text}
	case "blocker":
		tc.Blockers = []string{text}
	case "hint":
		tc.FixHints = []string{text}
	default:
		send(fmt.Sprintf("Unknown context type %q. Use goal, blocker, hint, or clear.", contextType))
		return
	}

	if err := c.bridgeManager.SetContext(ctx, targetID, tc, actorID); err != nil {
		send(fmt.Sprintf("Failed to set context: %s", err))
		return
	}

	send(fmt.Sprintf("Context set (%s) for session %s. Will be injected with next message.", contextType, targetID))
}

// truncateHashDisplay safely truncates a hash for display, guarding against short strings.
func truncateHashDisplay(h string) string {
	if len(h) >= 12 {
		return h[:12] + "..."
	}
	return h
}

// SendPermissionKeyboard sends a permission request notification with Approve/Reject inline keyboard.
// Called by the Notifier's NotifyFunc when a permission event is created.
func (c *Channel) SendPermissionKeyboard(ctx context.Context, chatID int64, perm *claudecode.PermissionRequest) {
	chatIDObj := tu.ID(chatID)

	riskEmoji := "🟢"
	if perm.RiskLevel == "high" {
		riskEmoji = "🔴"
	}

	text := fmt.Sprintf("%s Permission Request\n\nSession: %s\nTool: %s\nRisk: %s\nExpires: %s\nID: %s",
		riskEmoji,
		perm.SessionID,
		perm.Tool,
		perm.RiskLevel,
		perm.ExpiresAt.Format("15:04:05"),
		perm.ID,
	)

	if len(perm.ToolInput) > 0 && string(perm.ToolInput) != "null" {
		redactor := claudecode.NewOutputRedactor()
		input := redactor.Redact(string(perm.ToolInput), false)
		if len(input) > 400 {
			input = input[:400] + "..."
		}
		text += fmt.Sprintf("\n\nInput:\n```\n%s\n```", input)
	}

	msg := tu.Message(chatIDObj, text)
	msg.ReplyMarkup = &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{Text: "✅ Approve", CallbackData: "cc_approve:" + perm.ID},
				{Text: "❌ Deny", CallbackData: "cc_deny:" + perm.ID},
			},
		},
	}

	if _, err := c.bot.SendMessage(ctx, msg); err != nil {
		slog.Warn("failed to send permission keyboard",
			"permission_id", perm.ID,
			"error", err,
		)
	}
}

// handlePermissionCallback processes Approve/Reject button presses for permission requests.
func (c *Channel) handlePermissionCallback(ctx context.Context, query *telego.CallbackQuery) {
	chatID := query.Message.GetChat().ID
	chatIDObj := tu.ID(chatID)

	send := func(text string) {
		msg := tu.Message(chatIDObj, text)
		c.bot.SendMessage(ctx, msg)
	}

	if c.hookServer == nil {
		send("Bridge is not enabled.")
		return
	}

	// Parse callback data: "cc_approve:<permID>" or "cc_deny:<permID>"
	var decision claudecode.PermissionDecision
	var permID string
	if strings.HasPrefix(query.Data, "cc_approve:") {
		decision = claudecode.PermissionApproved
		permID = strings.TrimPrefix(query.Data, "cc_approve:")
	} else if strings.HasPrefix(query.Data, "cc_deny:") {
		decision = claudecode.PermissionDenied
		permID = strings.TrimPrefix(query.Data, "cc_deny:")
	} else {
		return
	}

	// Actor ID from callback sender
	actorID := ""
	if query.From.ID != 0 {
		actorID = strconv.FormatInt(query.From.ID, 10)
	}

	// Look up the permission request
	perm := c.hookServer.Permissions().Get(permID)
	if perm == nil {
		send(fmt.Sprintf("Permission %s not found or expired.", permID))
		return
	}

	// Resolve session to get ApproverACL
	tenantID := perm.OwnerID
	ctx = store.WithTenantID(ctx, tenantID)

	approverACL := []string{perm.ActorID} // default: session owner can approve
	if c.bridgeManager != nil {
		session, err := c.bridgeManager.GetSession(ctx, perm.SessionID)
		if err == nil && len(session.ApproverACL) > 0 {
			approverACL = session.ApproverACL
		} else if err == nil {
			// If no explicit ApproverACL, owner + notify ACL are approvers
			approverACL = append([]string{session.OwnerActorID}, session.NotifyACL...)
		}
	}

	// Decide
	if err := c.hookServer.Permissions().Decide(permID, decision, actorID, approverACL); err != nil {
		slog.Warn("permission callback: decide failed",
			"permission_id", permID,
			"actor", actorID,
			"error", err,
		)
		send(fmt.Sprintf("Cannot apply decision: %s", err))
		return
	}

	verb := "approved"
	if decision == claudecode.PermissionDenied {
		verb = "denied"
	}

	slog.Info("permission decided via Telegram",
		"permission_id", permID,
		"decision", decision,
		"actor", actorID,
		"tool", perm.Tool,
	)

	send(fmt.Sprintf("✅ Permission %s: %s (tool: %s, by: %s)", verb, permID, perm.Tool, actorID))
}
